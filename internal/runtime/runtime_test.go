package runtime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/karoc/adp/internal/adapters"
	"github.com/karoc/adp/internal/paths"
	"github.com/karoc/adp/internal/schema"
	"gopkg.in/yaml.v3"
)

func TestBuildCreatesRuntimeHandleEnvAndOverlay(t *testing.T) {
	projectRoot := t.TempDir()
	writeFile(t, filepath.Join(projectRoot, "go.mod"), []byte("module example\n"))
	writeFile(t, filepath.Join(projectRoot, "AGENTS.md"), []byte("real agents\n"))

	layout := paths.New(filepath.Join(t.TempDir(), "adp-home"), filepath.Join(t.TempDir(), "runtime-parent"))
	handle, err := Build(context.Background(), BuildRequest{
		Layout: layout,
		Config: testConfig(projectRoot),
		Files: []adapters.GeneratedFile{
			{Path: "AGENTS.md", Data: []byte("adp agents\n")},
		},
		Env: map[string]string{
			"CUSTOM":      "1",
			paths.EnvHome: "adapter-should-not-win",
		},
		Task: adapters.TaskContext{
			ID:       "task-20260608-0001",
			Title:    "Bind runtime session to task",
			Status:   "ready",
			Priority: "high",
			Phase:    "p1",
		},
		SessionID: "session-1",
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	wantRoot := filepath.Join(layout.RuntimeParent, "game-a-session-1")
	if handle.Root != wantRoot {
		t.Fatalf("runtime root mismatch: got %s want %s", handle.Root, wantRoot)
	}
	if handle.SessionID != "session-1" {
		t.Fatalf("session id mismatch: %s", handle.SessionID)
	}
	if handle.TaskID != "task-20260608-0001" {
		t.Fatalf("task id mismatch: %s", handle.TaskID)
	}
	if handle.Env[paths.EnvHome] != layout.Home {
		t.Fatalf("ADP_HOME mismatch: %s", handle.Env[paths.EnvHome])
	}
	if handle.Env["ADP_WORKSPACE"] != "game-a" {
		t.Fatalf("ADP_WORKSPACE mismatch: %s", handle.Env["ADP_WORKSPACE"])
	}
	if handle.Env["ADP_PROJECT_ROOT"] != projectRoot {
		t.Fatalf("ADP_PROJECT_ROOT mismatch: %s", handle.Env["ADP_PROJECT_ROOT"])
	}
	if handle.Env["ADP_RUNTIME_ROOT"] != wantRoot {
		t.Fatalf("ADP_RUNTIME_ROOT mismatch: %s", handle.Env["ADP_RUNTIME_ROOT"])
	}
	if handle.Env["ADP_SESSION_ID"] != "session-1" {
		t.Fatalf("ADP_SESSION_ID mismatch: %s", handle.Env["ADP_SESSION_ID"])
	}
	if handle.Env["ADP_TASK_ID"] != "task-20260608-0001" || handle.Env["ADP_TASK_PHASE"] != "p1" {
		t.Fatalf("task env mismatch: %#v", handle.Env)
	}
	if handle.Env["CUSTOM"] != "1" {
		t.Fatalf("adapter env was not preserved: %#v", handle.Env)
	}

	assertContent(t, filepath.Join(handle.Root, "AGENTS.md"), "adp agents\n")
	assertSymlink(t, filepath.Join(handle.Root, "go.mod"), filepath.Join(projectRoot, "go.mod"))
	assertContent(t, filepath.Join(projectRoot, "AGENTS.md"), "real agents\n")
	if len(handle.Warnings) == 0 {
		t.Fatalf("expected conflict warning for project AGENTS.md")
	}
}

func TestBuildWritesRuntimeManifestWithoutPollutingProject(t *testing.T) {
	projectRoot := t.TempDir()
	layout := paths.New(filepath.Join(t.TempDir(), "adp-home"), filepath.Join(t.TempDir(), "runtime-parent"))

	handle, err := Build(context.Background(), BuildRequest{
		Layout:    layout,
		Config:    testConfig(projectRoot),
		Keep:      true,
		Task:      adapters.TaskContext{ID: "task-20260608-0002", Title: "Write task manifest"},
		SessionID: "manifest-session",
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	manifestPath := filepath.Join(handle.Root, ManifestPath)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read runtime manifest: %v", err)
	}

	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse runtime manifest: %v", err)
	}
	if manifest.Version != schema.CurrentVersion {
		t.Fatalf("manifest version mismatch: got %d want %d", manifest.Version, schema.CurrentVersion)
	}
	if manifest.SessionID != "manifest-session" {
		t.Fatalf("manifest session id mismatch: %s", manifest.SessionID)
	}
	if manifest.Workspace != "game-a" {
		t.Fatalf("manifest workspace mismatch: %s", manifest.Workspace)
	}
	if manifest.TaskID != "task-20260608-0002" || manifest.TaskTitle != "Write task manifest" {
		t.Fatalf("manifest task mismatch: %+v", manifest)
	}
	if manifest.ProjectRoot != projectRoot {
		t.Fatalf("manifest project root mismatch: %s", manifest.ProjectRoot)
	}
	if manifest.RuntimeRoot != handle.Root {
		t.Fatalf("manifest runtime root mismatch: %s", manifest.RuntimeRoot)
	}
	if !manifest.Keep {
		t.Fatalf("manifest keep mismatch: expected true")
	}
	if manifest.GeneratedBy != ManifestGeneratedBy {
		t.Fatalf("manifest generated_by mismatch: %s", manifest.GeneratedBy)
	}
	if manifest.CreatedAt.IsZero() {
		t.Fatalf("manifest created_at should be set")
	}
	if !manifest.CreatedAt.Equal(manifest.CreatedAt.UTC()) {
		t.Fatalf("manifest created_at should be UTC: %s", manifest.CreatedAt.Format(time.RFC3339Nano))
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ManifestPath)); !os.IsNotExist(err) {
		t.Fatalf("project root should not contain runtime manifest, stat err: %v", err)
	}
}

func TestBuildGeneratesSessionIDAndCleanupHonorsKeep(t *testing.T) {
	projectRoot := t.TempDir()
	layout := paths.New(filepath.Join(t.TempDir(), "adp-home"), filepath.Join(t.TempDir(), "runtime-parent"))

	handle, err := Build(context.Background(), BuildRequest{
		Layout: layout,
		Config: testConfig(projectRoot),
		Keep:   true,
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if handle.SessionID == "" {
		t.Fatalf("expected generated session id")
	}
	if _, err := os.Stat(handle.Root); err != nil {
		t.Fatalf("expected runtime root to exist: %v", err)
	}
	if err := Cleanup(context.Background(), *handle); err != nil {
		t.Fatalf("cleanup keep: %v", err)
	}
	if _, err := os.Stat(handle.Root); err != nil {
		t.Fatalf("expected kept runtime root to remain: %v", err)
	}

	handle.Keep = false
	if err := Cleanup(context.Background(), *handle); err != nil {
		t.Fatalf("cleanup remove: %v", err)
	}
	if _, err := os.Stat(handle.Root); !os.IsNotExist(err) {
		t.Fatalf("expected runtime root to be removed, stat err: %v", err)
	}
}

func TestBuildRejectsAdapterRuntimeManifest(t *testing.T) {
	projectRoot := t.TempDir()
	layout := paths.New(filepath.Join(t.TempDir(), "adp-home"), filepath.Join(t.TempDir(), "runtime-parent"))

	reservedPaths := []string{
		ManifestPath,
		"./" + ManifestPath,
		ManifestPath + "/child",
	}
	for _, reservedPath := range reservedPaths {
		t.Run(reservedPath, func(t *testing.T) {
			_, err := Build(context.Background(), BuildRequest{
				Layout:    layout,
				Config:    testConfig(projectRoot),
				SessionID: "session-1-" + safeSessionSuffix(t.Name()),
				Files: []adapters.GeneratedFile{
					{Path: reservedPath, Data: []byte("adapter manifest\n")},
				},
			})
			if !errors.Is(err, ErrManifestPathReserved) {
				t.Fatalf("expected manifest path reserved error, got %v", err)
			}

			runtimeRoot := filepath.Join(layout.RuntimeParent, "game-a-session-1-"+safeSessionSuffix(t.Name()))
			if _, err := os.Stat(runtimeRoot); !os.IsNotExist(err) {
				t.Fatalf("runtime root should not be created after manifest conflict, stat err: %v", err)
			}
		})
	}
}

func TestBuildRejectsInvalidProjectRootAndSessionID(t *testing.T) {
	layout := paths.New(filepath.Join(t.TempDir(), "adp-home"), filepath.Join(t.TempDir(), "runtime-parent"))

	_, err := Build(context.Background(), BuildRequest{
		Layout:    layout,
		Config:    testConfig("relative-project"),
		SessionID: "session-1",
	})
	if err == nil {
		t.Fatalf("expected relative project root to fail")
	}

	_, err = Build(context.Background(), BuildRequest{
		Layout:    layout,
		Config:    testConfig(t.TempDir()),
		SessionID: "../escape",
	})
	if err == nil {
		t.Fatalf("expected unsafe session id to fail")
	}
}

func testConfig(projectRoot string) schema.Config {
	return schema.Config{
		Version: schema.CurrentVersion,
		Workspace: schema.Workspace{
			Name: "game-a",
		},
		Project: schema.Project{
			Root: projectRoot,
		},
	}
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func assertContent(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(got) != want {
		t.Fatalf("content mismatch for %s: got %q want %q", path, got, want)
	}
}

func assertSymlink(t *testing.T, path, want string) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("lstat %s: %v", path, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%s is not a symlink", path)
	}
	got, err := os.Readlink(path)
	if err != nil {
		t.Fatalf("readlink %s: %v", path, err)
	}
	if got != want {
		t.Fatalf("symlink target mismatch for %s: got %s want %s", path, got, want)
	}
}

func safeSessionSuffix(name string) string {
	clean := ""
	for _, r := range name {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			clean += string(r)
		}
	}
	return clean
}
