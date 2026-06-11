package runtime

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/karoc/adp/internal/adapters"
	"github.com/karoc/adp/internal/gitenv"
	"github.com/karoc/adp/internal/gitstate"
	"github.com/karoc/adp/internal/overlay"
	"github.com/karoc/adp/internal/paths"
	"github.com/karoc/adp/internal/schema"
	"gopkg.in/yaml.v3"
)

type Handle = adapters.RuntimeHandle

const (
	ManifestPath        = ".adp-runtime.yaml"
	ManifestVersion     = 1
	ManifestGeneratedBy = "adp"
)

var (
	ErrManifestPathReserved = errors.New(".adp-runtime.yaml is reserved for the ADP runtime manifest")
	ErrRuntimeParentUnsafe  = errors.New("unsafe runtime parent")
)

var sessionIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

type Manifest struct {
	Version            int       `yaml:"version"`
	SessionID          string    `yaml:"session_id"`
	Workspace          string    `yaml:"workspace"`
	TaskID             string    `yaml:"task_id,omitempty"`
	TaskTitle          string    `yaml:"task_title,omitempty"`
	TaskOwner          string    `yaml:"task_owner,omitempty"`
	TaskClaimedAt      time.Time `yaml:"task_claimed_at,omitempty"`
	TaskLeaseExpiresAt time.Time `yaml:"task_lease_expires_at,omitempty"`
	ProjectRoot        string    `yaml:"project_root"`
	GitRoot            string    `yaml:"git_root,omitempty"`
	GitMetadataSkipped bool      `yaml:"git_metadata_skipped,omitempty"`
	RuntimeRoot        string    `yaml:"runtime_root"`
	CreatedAt          time.Time `yaml:"created_at"`
	Keep               bool      `yaml:"keep"`
	GeneratedBy        string    `yaml:"generated_by"`
}

type BuildRequest struct {
	Layout       paths.Layout
	Config       schema.Config
	WorkspaceDir string
	Files        []adapters.GeneratedFile
	Env          map[string]string
	GitRoot      string
	Task         adapters.TaskContext
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

	runtimeParent, err := validateRuntimeParent(req.Layout.RuntimeParent, req.Config.Project.Root)
	if err != nil {
		return nil, err
	}
	runtimeRoot := filepath.Join(runtimeParent, req.Config.Workspace.Name+"-"+sessionID)
	gitRoot := strings.TrimSpace(req.GitRoot)
	if gitRoot == "" {
		gitRoot = gitstate.DiscoverRoot(ctx, req.Config.Project.Root)
	}
	if gitRoot != "" {
		if !filepath.IsAbs(gitRoot) {
			return nil, fmt.Errorf("git root must be absolute")
		}
		gitRoot = filepath.Clean(gitRoot)
	}
	files, err := appendRuntimeManifest(req.Files, Manifest{
		Version:            ManifestVersion,
		SessionID:          sessionID,
		Workspace:          req.Config.Workspace.Name,
		TaskID:             req.Task.ID,
		TaskTitle:          req.Task.Title,
		TaskOwner:          req.Task.Owner,
		TaskClaimedAt:      req.Task.ClaimedAt,
		TaskLeaseExpiresAt: req.Task.LeaseExpiresAt,
		ProjectRoot:        req.Config.Project.Root,
		GitRoot:            gitRoot,
		GitMetadataSkipped: true,
		RuntimeRoot:        runtimeRoot,
		CreatedAt:          time.Now().UTC(),
		Keep:               req.Keep,
		GeneratedBy:        ManifestGeneratedBy,
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

	env := runtimeEnv(req.Env, req.Layout, req.Config, runtimeRoot, sessionID, gitRoot, req.Task)
	return &Handle{
		SessionID:     sessionID,
		WorkspaceName: req.Config.Workspace.Name,
		TaskID:        req.Task.ID,
		ProjectRoot:   req.Config.Project.Root,
		GitRoot:       gitRoot,
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

func runtimeEnv(base map[string]string, layout paths.Layout, config schema.Config, runtimeRoot, sessionID, gitRoot string, task adapters.TaskContext) map[string]string {
	env := make(map[string]string, len(base)+10)
	for key, value := range base {
		if gitenv.IsRepositoryDirective(key) {
			continue
		}
		env[key] = value
	}
	env[paths.EnvHome] = layout.Home
	env["ADP_WORKSPACE"] = config.Workspace.Name
	env["ADP_PROJECT_ROOT"] = config.Project.Root
	if cliPath, err := os.Executable(); err == nil && strings.TrimSpace(cliPath) != "" {
		env["ADP_CLI"] = cliPath
	}
	if gitRoot != "" {
		env["ADP_GIT_ROOT"] = gitRoot
	}
	env["ADP_RUNTIME_ROOT"] = runtimeRoot
	env["ADP_SESSION_ID"] = sessionID
	env["GIT_CEILING_DIRECTORIES"] = mergePathList(env["GIT_CEILING_DIRECTORIES"], runtimeRoot)
	if !task.IsZero() {
		env["ADP_TASK_ID"] = task.ID
		env["ADP_TASK_TITLE"] = task.Title
		env["ADP_TASK_STATUS"] = task.Status
		env["ADP_TASK_PRIORITY"] = task.Priority
		env["ADP_TASK_PHASE"] = task.Phase
		if strings.TrimSpace(task.Owner) != "" {
			env["ADP_TASK_OWNER"] = task.Owner
		}
		if !task.ClaimedAt.IsZero() {
			env["ADP_TASK_CLAIMED_AT"] = task.ClaimedAt.UTC().Format(time.RFC3339)
		}
		if !task.LeaseExpiresAt.IsZero() {
			env["ADP_TASK_LEASE_EXPIRES_AT"] = task.LeaseExpiresAt.UTC().Format(time.RFC3339)
		}
	}
	return env
}

func mergePathList(existing string, addition string) string {
	addition = filepath.Clean(addition)
	if strings.TrimSpace(existing) == "" {
		return addition
	}
	for _, part := range filepath.SplitList(existing) {
		if part == addition {
			return existing
		}
	}
	return existing + string(os.PathListSeparator) + addition
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

func validateRuntimeParent(runtimeParent, projectRoot string) (string, error) {
	runtimeAbs, err := filepath.Abs(runtimeParent)
	if err != nil {
		return "", fmt.Errorf("resolve runtime parent: %w", err)
	}
	runtimeAbs = filepath.Clean(runtimeAbs)
	if filepath.Dir(runtimeAbs) == runtimeAbs {
		return "", fmt.Errorf("%w: runtime parent must not be the filesystem root: %s", ErrRuntimeParentUnsafe, runtimeAbs)
	}

	projectAbs, err := filepath.Abs(projectRoot)
	if err != nil {
		return "", fmt.Errorf("resolve project root: %w", err)
	}
	projectAbs = filepath.Clean(projectAbs)

	runtimeCandidates := appendResolvedPath(nil, runtimeAbs)
	projectCandidates := appendResolvedPath(nil, projectAbs)
	switch {
	case pathsOverlap(runtimeCandidates, projectCandidates, sameCleanPath):
		return "", fmt.Errorf("%w: runtime parent must not be the project root: %s", ErrRuntimeParentUnsafe, runtimeAbs)
	case pathsOverlap(projectCandidates, runtimeCandidates, pathInsideDir):
		return "", fmt.Errorf("%w: runtime parent must not be inside the project root: %s", ErrRuntimeParentUnsafe, runtimeAbs)
	case pathsOverlap(runtimeCandidates, projectCandidates, pathInsideDir):
		return "", fmt.Errorf("%w: runtime parent must not contain the project root: %s", ErrRuntimeParentUnsafe, runtimeAbs)
	default:
		return runtimeAbs, nil
	}
}

func sameCleanPath(left string, right string) bool {
	rel, err := filepath.Rel(filepath.Clean(left), filepath.Clean(right))
	return err == nil && rel == "."
}

func pathInsideDir(parent string, child string) bool {
	rel, err := filepath.Rel(filepath.Clean(parent), filepath.Clean(child))
	if err != nil || rel == "." || rel == ".." {
		return false
	}
	return !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func appendResolvedPath(paths []string, candidate string) []string {
	candidate = filepath.Clean(candidate)
	paths = appendUniqueCleanPath(paths, candidate)

	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return paths
	}
	resolvedAbs, err := filepath.Abs(resolved)
	if err != nil {
		return paths
	}
	return appendUniqueCleanPath(paths, resolvedAbs)
}

func appendUniqueCleanPath(paths []string, candidate string) []string {
	candidate = filepath.Clean(candidate)
	for _, existing := range paths {
		if sameCleanPath(existing, candidate) {
			return paths
		}
	}
	return append(paths, candidate)
}

func pathsOverlap(leftPaths []string, rightPaths []string, match func(string, string) bool) bool {
	for _, left := range leftPaths {
		for _, right := range rightPaths {
			if match(left, right) {
				return true
			}
		}
	}
	return false
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
