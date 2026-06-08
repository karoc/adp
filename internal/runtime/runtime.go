package runtime

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/karoc/adp/internal/adapters"
	"github.com/karoc/adp/internal/overlay"
	"github.com/karoc/adp/internal/paths"
	"github.com/karoc/adp/internal/schema"
	"gopkg.in/yaml.v3"
)

type Handle = adapters.RuntimeHandle

const (
	ManifestPath        = ".adp-runtime.yaml"
	ManifestGeneratedBy = "adp"
)

var ErrManifestPathReserved = errors.New(".adp-runtime.yaml is reserved for the ADP runtime manifest")

var sessionIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

type Manifest struct {
	Version     int       `yaml:"version"`
	SessionID   string    `yaml:"session_id"`
	Workspace   string    `yaml:"workspace"`
	ProjectRoot string    `yaml:"project_root"`
	RuntimeRoot string    `yaml:"runtime_root"`
	CreatedAt   time.Time `yaml:"created_at"`
	Keep        bool      `yaml:"keep"`
	GeneratedBy string    `yaml:"generated_by"`
}

type BuildRequest struct {
	Layout       paths.Layout
	Config       schema.Config
	WorkspaceDir string
	Files        []adapters.GeneratedFile
	Env          map[string]string
	Backend      overlay.Backend
	Keep         bool
	SessionID    string
}

func Build(ctx context.Context, req BuildRequest) (*Handle, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := req.Config.Validate(); err != nil {
		return nil, err
	}
	if req.Layout.Home == "" {
		return nil, fmt.Errorf("ADP home is required")
	}
	if req.Layout.RuntimeParent == "" {
		return nil, fmt.Errorf("runtime parent is required")
	}
	if !filepath.IsAbs(req.Config.Project.Root) {
		return nil, fmt.Errorf("project root must be absolute")
	}

	sessionID := req.SessionID
	if sessionID == "" {
		generated, err := newSessionID()
		if err != nil {
			return nil, err
		}
		sessionID = generated
	}
	if !sessionIDPattern.MatchString(sessionID) {
		return nil, fmt.Errorf("invalid session id %q", sessionID)
	}

	runtimeParent, err := filepath.Abs(req.Layout.RuntimeParent)
	if err != nil {
		return nil, fmt.Errorf("resolve runtime parent: %w", err)
	}
	runtimeRoot := filepath.Join(runtimeParent, req.Config.Workspace.Name+"-"+sessionID)
	files, err := appendRuntimeManifest(req.Files, Manifest{
		Version:     schema.CurrentVersion,
		SessionID:   sessionID,
		Workspace:   req.Config.Workspace.Name,
		ProjectRoot: req.Config.Project.Root,
		RuntimeRoot: runtimeRoot,
		CreatedAt:   time.Now().UTC(),
		Keep:        req.Keep,
		GeneratedBy: ManifestGeneratedBy,
	})
	if err != nil {
		return nil, err
	}

	backend := req.Backend
	if backend == nil {
		backend = overlay.NewSymlinkBackend()
	}
	result, err := backend.Materialize(ctx, overlay.Request{
		WorkspaceName: req.Config.Workspace.Name,
		ProjectRoot:   req.Config.Project.Root,
		RuntimeRoot:   runtimeRoot,
		Files:         files,
		Keep:          req.Keep,
	})
	if err != nil {
		return nil, err
	}

	env := runtimeEnv(req.Env, req.Layout, req.Config, runtimeRoot, sessionID)
	return &Handle{
		SessionID:     sessionID,
		WorkspaceName: req.Config.Workspace.Name,
		ProjectRoot:   req.Config.Project.Root,
		Root:          runtimeRoot,
		Env:           env,
		Keep:          req.Keep,
		Warnings:      warningsFromConflicts(result.Conflicts),
	}, nil
}

func Cleanup(ctx context.Context, handle Handle) error {
	return overlay.NewSymlinkBackend().Cleanup(ctx, overlay.Handle{
		Root: handle.Root,
		Keep: handle.Keep,
	})
}

func runtimeEnv(base map[string]string, layout paths.Layout, config schema.Config, runtimeRoot, sessionID string) map[string]string {
	env := make(map[string]string, len(base)+5)
	for key, value := range base {
		env[key] = value
	}
	env[paths.EnvHome] = layout.Home
	env["ADP_WORKSPACE"] = config.Workspace.Name
	env["ADP_PROJECT_ROOT"] = config.Project.Root
	env["ADP_RUNTIME_ROOT"] = runtimeRoot
	env["ADP_SESSION_ID"] = sessionID
	return env
}

func appendRuntimeManifest(files []adapters.GeneratedFile, manifest Manifest) ([]adapters.GeneratedFile, error) {
	for _, file := range files {
		if isRuntimeManifestPath(file.Path) {
			return nil, fmt.Errorf("%w: adapter generated file %q", ErrManifestPathReserved, file.Path)
		}
	}

	data, err := yaml.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("marshal runtime manifest: %w", err)
	}

	withManifest := make([]adapters.GeneratedFile, 0, len(files)+1)
	withManifest = append(withManifest, files...)
	withManifest = append(withManifest, adapters.GeneratedFile{
		Path: ManifestPath,
		Mode: 0644,
		Data: data,
	})
	return withManifest, nil
}

func isRuntimeManifestPath(filePath string) bool {
	normalized := path.Clean(strings.ReplaceAll(filePath, "\\", "/"))
	return normalized == ManifestPath || strings.HasPrefix(normalized, ManifestPath+"/")
}

func warningsFromConflicts(conflicts []overlay.Conflict) []string {
	warnings := make([]string, 0, len(conflicts))
	for _, conflict := range conflicts {
		warnings = append(warnings, fmt.Sprintf("runtime conflict at %q: %s", conflict.Path, conflict.Reason))
	}
	return warnings
}

func newSessionID() (string, error) {
	random := make([]byte, 4)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}
	return time.Now().UTC().Format("20060102T150405") + "-" + hex.EncodeToString(random), nil
}
