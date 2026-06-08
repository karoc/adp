package workspace

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
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

func TestRegistryRemoveSuccess(t *testing.T) {
	registry, layout := newTestRegistry(t)
	projectRoot := createProject(t)

	if _, err := registry.Add(context.Background(), "game-a", projectRoot); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if err := registry.Remove(context.Background(), "game-a"); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	requireNotExist(t, layout.WorkspaceDir("game-a"))
	requireDir(t, layout.WorkspacesDir)
	requireFile(t, layout.ConfigFile)

	_, _, err := registry.Get(context.Background(), "game-a")
	if !errors.Is(err, ErrWorkspaceNotFound) {
		t.Fatalf("Get() after Remove() error = %v, want ErrWorkspaceNotFound", err)
	}
}

func TestRegistryRemoveMissingFails(t *testing.T) {
	registry, _ := newTestRegistry(t)

	err := registry.Remove(context.Background(), "missing")
	if !errors.Is(err, ErrWorkspaceNotFound) {
		t.Fatalf("Remove() missing error = %v, want ErrWorkspaceNotFound", err)
	}
}

func TestRegistryRemoveRejectsInvalidName(t *testing.T) {
	registry, layout := newTestRegistry(t)

	err := registry.Remove(context.Background(), "../bad")
	if err == nil {
		t.Fatal("Remove() error = nil, want invalid name error")
	}
	if _, statErr := os.Stat(layout.Home); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("invalid Remove() touched ADP home: stat error = %v", statErr)
	}
}

func TestRegistryRenameSuccess(t *testing.T) {
	registry, layout := newTestRegistry(t)
	projectRoot := createProject(t)

	if _, err := registry.Add(context.Background(), "game-a", projectRoot); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	customPrompt := []byte("custom prompt\n")
	promptPath := filepath.Join(layout.WorkspaceDir("game-a"), "prompts", "base.md")
	if err := os.WriteFile(promptPath, customPrompt, 0o644); err != nil {
		t.Fatalf("write custom prompt: %v", err)
	}

	cfg, err := registry.Rename(context.Background(), "game-a", "game-b")
	if err != nil {
		t.Fatalf("Rename() error = %v", err)
	}
	if cfg.Workspace.Name != "game-b" {
		t.Fatalf("renamed config name = %q, want game-b", cfg.Workspace.Name)
	}
	if cfg.Project.Root != projectRoot {
		t.Fatalf("project root = %q, want %q", cfg.Project.Root, projectRoot)
	}
	requireNotExist(t, layout.WorkspaceDir("game-a"))
	requireDir(t, layout.WorkspaceDir("game-b"))

	loaded, err := schema.LoadConfig(layout.WorkspaceConfig("game-b"))
	if err != nil {
		t.Fatalf("LoadConfig() renamed config error = %v", err)
	}
	if loaded.Workspace.Name != "game-b" {
		t.Fatalf("loaded workspace name = %q, want game-b", loaded.Workspace.Name)
	}
	if loaded.Project.Root != projectRoot {
		t.Fatalf("loaded project root = %q, want %q", loaded.Project.Root, projectRoot)
	}
	if loaded.Prompts.Base != "prompts/base.md" {
		t.Fatalf("base prompt path = %q, want prompts/base.md", loaded.Prompts.Base)
	}
	gotPrompt, err := os.ReadFile(filepath.Join(layout.WorkspaceDir("game-b"), "prompts", "base.md"))
	if err != nil {
		t.Fatalf("read renamed prompt: %v", err)
	}
	if string(gotPrompt) != string(customPrompt) {
		t.Fatalf("renamed prompt content = %q, want %q", gotPrompt, customPrompt)
	}
	requireFile(t, filepath.Join(layout.WorkspaceDir("game-b"), "memory", "shared.md"))
	requireFile(t, filepath.Join(layout.WorkspaceDir("game-b"), "mcp", "config.yaml"))
	requireFile(t, filepath.Join(layout.WorkspaceDir("game-b"), "profiles", "codex.yaml"))
	requireFile(t, filepath.Join(layout.WorkspaceDir("game-b"), "profiles", "claude.yaml"))
}

func TestRegistryRenameToExistingFails(t *testing.T) {
	registry, layout := newTestRegistry(t)
	gameARoot := createProject(t)
	gameBRoot := createProject(t)

	if _, err := registry.Add(context.Background(), "game-a", gameARoot); err != nil {
		t.Fatalf("Add() game-a error = %v", err)
	}
	if _, err := registry.Add(context.Background(), "game-b", gameBRoot); err != nil {
		t.Fatalf("Add() game-b error = %v", err)
	}

	_, err := registry.Rename(context.Background(), "game-a", "game-b")
	if !errors.Is(err, ErrWorkspaceExists) {
		t.Fatalf("Rename() existing target error = %v, want ErrWorkspaceExists", err)
	}

	gameA, err := schema.LoadConfig(layout.WorkspaceConfig("game-a"))
	if err != nil {
		t.Fatalf("LoadConfig() game-a error = %v", err)
	}
	if gameA.Workspace.Name != "game-a" || gameA.Project.Root != gameARoot {
		t.Fatalf("game-a config changed after failed rename: %+v", gameA)
	}
	gameB, err := schema.LoadConfig(layout.WorkspaceConfig("game-b"))
	if err != nil {
		t.Fatalf("LoadConfig() game-b error = %v", err)
	}
	if gameB.Workspace.Name != "game-b" || gameB.Project.Root != gameBRoot {
		t.Fatalf("game-b config changed after failed rename: %+v", gameB)
	}
}

func TestRegistryRenameRejectsInvalidName(t *testing.T) {
	registry, layout := newTestRegistry(t)

	_, err := registry.Rename(context.Background(), "game-a", "../bad")
	if err == nil {
		t.Fatal("Rename() error = nil, want invalid name error")
	}
	if _, statErr := os.Stat(layout.Home); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("invalid Rename() touched ADP home: stat error = %v", statErr)
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

func TestRegistryNamesReadsWorkspaceDirsWithoutLoadingConfigs(t *testing.T) {
	registry, layout := newTestRegistry(t)
	if err := registry.Init(context.Background()); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	for _, name := range []string{"z-game", "a-game", "broken", ".hidden", "bad name"} {
		if err := os.MkdirAll(layout.WorkspaceDir(name), 0o755); err != nil {
			t.Fatalf("create workspace dir %s: %v", name, err)
		}
	}
	if err := os.WriteFile(filepath.Join(layout.WorkspaceDir("broken"), "workspace.yaml"), []byte("version: [broken\n"), 0o644); err != nil {
		t.Fatalf("write broken workspace config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(layout.WorkspacesDir, "file-workspace"), []byte("not a dir\n"), 0o644); err != nil {
		t.Fatalf("write non-dir workspace entry: %v", err)
	}

	names, err := registry.Names(context.Background())
	if err != nil {
		t.Fatalf("Names() error = %v", err)
	}

	want := []string{"a-game", "broken", "z-game"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("Names() = %#v, want %#v", names, want)
	}
}

func TestRegistryNamesReturnsEmptyWhenWorkspacesDirIsMissing(t *testing.T) {
	registry, _ := newTestRegistry(t)

	names, err := registry.Names(context.Background())
	if err != nil {
		t.Fatalf("Names() error = %v", err)
	}
	if len(names) != 0 {
		t.Fatalf("Names() = %#v, want empty", names)
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
	runtimeParent := filepath.Join(t.TempDir(), "runtime-parent")
	t.Setenv(paths.EnvHome, adpHome)
	t.Setenv(paths.EnvRuntimeDir, runtimeParent)

	layout, err := paths.FromEnv()
	if err != nil {
		t.Fatalf("FromEnv() error = %v", err)
	}
	if layout.Home != adpHome {
		t.Fatalf("layout home = %q, want %q", layout.Home, adpHome)
	}
	if layout.RuntimeParent != runtimeParent {
		t.Fatalf("runtime parent = %q, want %q", layout.RuntimeParent, runtimeParent)
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

func requireNotExist(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stat %s error = %v, want ErrNotExist", path, err)
	}
}
