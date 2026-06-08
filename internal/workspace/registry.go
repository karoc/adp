package workspace

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/karoc/adp/internal/paths"
	"github.com/karoc/adp/internal/schema"
)

var (
	ErrWorkspaceExists   = errors.New("workspace already exists")
	ErrWorkspaceNotFound = errors.New("workspace not found")
)

type Registry struct {
	Layout paths.Layout
}

type Record struct {
	Name         string
	ProjectRoot  string
	WorkspaceDir string
}

func NewRegistry(layout paths.Layout) *Registry {
	return &Registry{Layout: layout}
}

func (r *Registry) Init(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	for _, dir := range []string{r.Layout.Home, r.Layout.WorkspacesDir, r.Layout.LogsDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	return writeFileIfMissing(r.Layout.ConfigFile, []byte(defaultRegistryConfig))
}

func (r *Registry) Add(ctx context.Context, name string, projectRoot string) (*schema.Config, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := schema.ValidateWorkspaceName(name); err != nil {
		return nil, err
	}

	absRoot, err := absoluteProjectRoot(projectRoot)
	if err != nil {
		return nil, err
	}
	if err := ensureProjectDir(absRoot); err != nil {
		return nil, err
	}
	if err := r.Init(ctx); err != nil {
		return nil, err
	}

	workspaceDir := r.Layout.WorkspaceDir(name)
	if err := createNewWorkspaceDir(workspaceDir, name); err != nil {
		return nil, err
	}

	cfg := defaultWorkspaceConfig(name, absRoot)
	if err := writeWorkspaceDefaults(workspaceDir, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (r *Registry) Get(ctx context.Context, name string) (*schema.Config, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	if err := schema.ValidateWorkspaceName(name); err != nil {
		return nil, "", err
	}

	workspaceDir := r.Layout.WorkspaceDir(name)
	cfg, err := schema.LoadConfig(r.Layout.WorkspaceConfig(name))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, "", fmt.Errorf("%w: %s", ErrWorkspaceNotFound, name)
		}
		return nil, "", err
	}
	if cfg.Workspace.Name != name {
		return nil, "", fmt.Errorf("workspace config name mismatch: requested %s, found %s", name, cfg.Workspace.Name)
	}

	return cfg, workspaceDir, nil
}

func (r *Registry) List(ctx context.Context) ([]Record, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(r.Layout.WorkspacesDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read workspaces directory: %w", err)
	}

	records := []Record{}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !entry.IsDir() {
			continue
		}
		cfg, workspaceDir, err := r.Get(ctx, entry.Name())
		if err != nil {
			return nil, err
		}
		records = append(records, Record{
			Name:         cfg.Workspace.Name,
			ProjectRoot:  cfg.Project.Root,
			WorkspaceDir: workspaceDir,
		})
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].Name < records[j].Name
	})
	return records, nil
}

func (r *Registry) FindByProjectPath(ctx context.Context, path string) (*schema.Config, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, "", fmt.Errorf("resolve project path: %w", err)
	}

	records, err := r.List(ctx)
	if err != nil {
		return nil, "", err
	}

	var match Record
	matchLen := -1
	for _, record := range records {
		if isPathInside(absPath, record.ProjectRoot) && len(record.ProjectRoot) > matchLen {
			match = record
			matchLen = len(record.ProjectRoot)
		}
	}
	if matchLen < 0 {
		return nil, "", fmt.Errorf("%w for path: %s", ErrWorkspaceNotFound, absPath)
	}
	return r.Get(ctx, match.Name)
}

const defaultRegistryConfig = `version: 1
workspaces_dir: workspaces
logs_dir: logs
`

func absoluteProjectRoot(projectRoot string) (string, error) {
	if projectRoot == "" {
		return "", errors.New("project root is required")
	}
	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return "", fmt.Errorf("resolve project root %s: %w", projectRoot, err)
	}
	return absRoot, nil
}

func ensureProjectDir(projectRoot string) error {
	info, err := os.Stat(projectRoot)
	if err != nil {
		return fmt.Errorf("stat project root %s: %w", projectRoot, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("project root is not a directory: %s", projectRoot)
	}
	return nil
}

func createNewWorkspaceDir(workspaceDir string, name string) error {
	if err := os.Mkdir(workspaceDir, 0o755); err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("%w: %s", ErrWorkspaceExists, name)
		}
		return fmt.Errorf("create workspace directory %s: %w", workspaceDir, err)
	}
	return nil
}

func writeWorkspaceDefaults(workspaceDir string, cfg *schema.Config) error {
	for _, dir := range []string{"prompts", "memory", "mcp", "profiles"} {
		if err := os.MkdirAll(filepath.Join(workspaceDir, dir), 0o755); err != nil {
			return fmt.Errorf("create workspace subdirectory %s: %w", dir, err)
		}
	}

	files := map[string]string{
		"prompts/base.md":      "# Base Prompt\n\n",
		"memory/shared.md":     "# Shared Memory\n\n",
		"mcp/config.yaml":      "enabled: true\nservers: []\n",
		"profiles/codex.yaml":  "profile: default\ncommand: codex\n",
		"profiles/claude.yaml": "profile: default\ncommand: claude\n",
	}
	for relPath, content := range files {
		if err := os.WriteFile(filepath.Join(workspaceDir, relPath), []byte(content), 0o644); err != nil {
			return fmt.Errorf("write workspace file %s: %w", relPath, err)
		}
	}

	return schema.SaveConfig(filepath.Join(workspaceDir, "workspace.yaml"), cfg)
}

func writeFileIfMissing(path string, data []byte) error {
	info, err := os.Stat(path)
	if err == nil {
		if info.IsDir() {
			return fmt.Errorf("file path is a directory: %s", path)
		}
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat file %s: %w", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write file %s: %w", path, err)
	}
	return nil
}

func isPathInside(path string, root string) bool {
	cleanPath := filepath.Clean(path)
	cleanRoot := filepath.Clean(root)
	rel, err := filepath.Rel(cleanRoot, cleanPath)
	if err != nil {
		return false
	}
	return rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func defaultWorkspaceConfig(name string, projectRoot string) *schema.Config {
	return &schema.Config{
		Version:   schema.CurrentVersion,
		Workspace: schema.Workspace{Name: name},
		Project:   schema.Project{Root: projectRoot},
		Memory: schema.Memory{
			Enabled: true,
			Shared:  "memory/shared.md",
		},
		Prompts: schema.Prompts{Base: "prompts/base.md"},
		Rules:   map[string]string{"coding_style": "strict"},
		MCP: schema.MCP{
			Enabled: true,
			Config:  "mcp/config.yaml",
		},
		Agents: map[string]schema.AgentConfig{
			"codex": {
				Enabled: true,
				Profile: "default",
				Command: "codex",
			},
			"claude": {
				Enabled: true,
				Profile: "default",
				Command: "claude",
			},
		},
	}
}
