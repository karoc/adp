package workspace

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/karoc/adp/internal/paths"
	"github.com/karoc/adp/internal/schema"
)

func TestRegistryInitIsIdempotent(t *testing.T) {
	registry, layout := newTestRegistry(t)
	ctx := context.Background()

	if err := registry.Init(ctx); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	requireDir(t, layout.Home)
	requireDir(t, layout.WorkspacesDir)
	requireDir(t, layout.LogsDir)
	requireFile(t, layout.ConfigFile)

	customConfig := []byte("custom: true\n")
	if err := os.WriteFile(layout.ConfigFile, customConfig, 0o644); err != nil {
		t.Fatalf("write custom config: %v", err)
	}

	if err := registry.Init(ctx); err != nil {
		t.Fatalf("Init() second call error = %v", err)
	}
	got, err := os.ReadFile(layout.ConfigFile)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if string(got) != string(customConfig) {
		t.Fatalf("Init() overwrote config.yaml: got %q, want %q", got, customConfig)
	}
}

func TestRegistryAddSuccess(t *testing.T) {
	registry, layout := newTestRegistry(t)
	projectRoot := createProject(t)
	parentDir := filepath.Dir(projectRoot)
	if err := os.Chdir(parentDir); err != nil {
		t.Fatalf("chdir project parent: %v", err)
	}

	cfg, err := registry.Add(context.Background(), "game-a", filepath.Base(projectRoot))
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if cfg.Workspace.Name != "game-a" {
		t.Fatalf("workspace name = %q, want game-a", cfg.Workspace.Name)
	}
	if cfg.Project.Root != projectRoot {
		t.Fatalf("project root = %q, want %q", cfg.Project.Root, projectRoot)
	}

	workspaceDir := layout.WorkspaceDir("game-a")
	requireFile(t, filepath.Join(workspaceDir, "workspace.yaml"))
	requireFile(t, filepath.Join(workspaceDir, "prompts", "base.md"))
	requireFile(t, filepath.Join(workspaceDir, "memory", "shared.md"))
	requireFile(t, filepath.Join(workspaceDir, "mcp", "config.yaml"))
	requireFile(t, filepath.Join(workspaceDir, "profiles", "codex.yaml"))
	requireFile(t, filepath.Join(workspaceDir, "profiles", "claude.yaml"))

	loaded, err := schema.LoadConfig(layout.WorkspaceConfig("game-a"))
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if loaded.Project.Root != projectRoot {
		t.Fatalf("loaded project root = %q, want %q", loaded.Project.Root, projectRoot)
	}
	if loaded.Prompts.Base != "prompts/base.md" {
		t.Fatalf("base prompt path = %q, want prompts/base.md", loaded.Prompts.Base)
	}
}

func TestRegistryAddDuplicateFails(t *testing.T) {
	registry, _ := newTestRegistry(t)
	projectRoot := createProject(t)

	if _, err := registry.Add(context.Background(), "game-a", projectRoot); err != nil {
		t.Fatalf("Add() first call error = %v", err)
	}
	_, err := registry.Add(context.Background(), "game-a", projectRoot)
	if !errors.Is(err, ErrWorkspaceExists) {
		t.Fatalf("Add() duplicate error = %v, want ErrWorkspaceExists", err)
	}
}

func TestRegistryAddRejectsInvalidName(t *testing.T) {
	registry, layout := newTestRegistry(t)
	projectRoot := createProject(t)

	_, err := registry.Add(context.Background(), "../bad", projectRoot)
	if err == nil {
		t.Fatal("Add() error = nil, want invalid name error")
	}
	if _, statErr := os.Stat(layout.Home); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("invalid Add() touched ADP home: stat error = %v", statErr)
	}
}

func TestRegistryGetSuccess(t *testing.T) {
	registry, layout := newTestRegistry(t)
	projectRoot := createProject(t)

	if _, err := registry.Add(context.Background(), "game-a", projectRoot); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	cfg, workspaceDir, err := registry.Get(context.Background(), "game-a")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if workspaceDir != layout.WorkspaceDir("game-a") {
		t.Fatalf("workspace dir = %q, want %q", workspaceDir, layout.WorkspaceDir("game-a"))
	}
	if cfg.Workspace.Name != "game-a" {
		t.Fatalf("workspace name = %q, want game-a", cfg.Workspace.Name)
	}
	if cfg.Project.Root != projectRoot {
		t.Fatalf("project root = %q, want %q", cfg.Project.Root, projectRoot)
	}
}

func TestRegistryListSortsWorkspaces(t *testing.T) {
	registry, _ := newTestRegistry(t)
	zRoot := createProject(t)
	aRoot := createProject(t)

	if _, err := registry.Add(context.Background(), "z-game", zRoot); err != nil {
		t.Fatalf("Add() z-game error = %v", err)
	}
	if _, err := registry.Add(context.Background(), "a-game", aRoot); err != nil {
		t.Fatalf("Add() a-game error = %v", err)
	}

	records, err := registry.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("List() returned %d records, want 2", len(records))
	}
	if records[0].Name != "a-game" || records[0].ProjectRoot != aRoot {
		t.Fatalf("first record = %+v, want a-game", records[0])
	}
	if records[1].Name != "z-game" || records[1].ProjectRoot != zRoot {
		t.Fatalf("second record = %+v, want z-game", records[1])
	}
}

func TestRegistryFindByProjectPathUsesLongestProjectRootMatch(t *testing.T) {
	registry, _ := newTestRegistry(t)
	parentRoot := createProject(t)
	childRoot := filepath.Join(parentRoot, "child")
	if err := os.Mkdir(childRoot, 0o755); err != nil {
		t.Fatalf("create child project: %v", err)
	}
	nestedDir := filepath.Join(childRoot, "internal", "pkg")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatalf("create nested dir: %v", err)
	}

	if _, err := registry.Add(context.Background(), "parent", parentRoot); err != nil {
		t.Fatalf("Add() parent error = %v", err)
	}
	if _, err := registry.Add(context.Background(), "child", childRoot); err != nil {
		t.Fatalf("Add() child error = %v", err)
	}

	cfg, workspaceDir, err := registry.FindByProjectPath(context.Background(), nestedDir)
	if err != nil {
		t.Fatalf("FindByProjectPath() error = %v", err)
	}
	if cfg.Workspace.Name != "child" {
		t.Fatalf("matched workspace = %q, want child", cfg.Workspace.Name)
	}
	if workspaceDir != registry.Layout.WorkspaceDir("child") {
		t.Fatalf("workspace dir = %q, want child dir", workspaceDir)
	}
}

func newTestRegistry(t *testing.T) (*Registry, paths.Layout) {
	t.Helper()

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})

	adpHome := filepath.Join(t.TempDir(), "adp-home")
	t.Setenv(paths.EnvHome, adpHome)

	layout, err := paths.FromEnv()
	if err != nil {
		t.Fatalf("FromEnv() error = %v", err)
	}
	if layout.Home != adpHome {
		t.Fatalf("layout home = %q, want %q", layout.Home, adpHome)
	}

	return NewRegistry(layout), layout
}

func createProject(t *testing.T) string {
	t.Helper()

	projectRoot := filepath.Join(t.TempDir(), "project")
	if err := os.Mkdir(projectRoot, 0o755); err != nil {
		t.Fatalf("create project root: %v", err)
	}
	return projectRoot
}

func requireDir(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if !info.IsDir() {
		t.Fatalf("%s is not a directory", path)
	}
}

func requireFile(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if info.IsDir() {
		t.Fatalf("%s is a directory, want file", path)
	}
}
