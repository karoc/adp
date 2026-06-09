package overlay

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/karoc/adp/internal/adapters"
)

func TestSymlinkBackendMaterializeWritesGeneratedFilesAndLinksProjectChildren(t *testing.T) {
	projectRoot := t.TempDir()
	writeProjectFile(t, projectRoot, "go.mod", []byte("module example\n"))
	if err := os.Mkdir(filepath.Join(projectRoot, "internal"), 0755); err != nil {
		t.Fatal(err)
	}

	runtimeRoot := filepath.Join(t.TempDir(), "runtime")
	result, err := NewSymlinkBackend().Materialize(context.Background(), Request{
		WorkspaceName: "game-a",
		ProjectRoot:   projectRoot,
		RuntimeRoot:   runtimeRoot,
		Files: []adapters.GeneratedFile{
			{Path: "AGENTS.md", Data: []byte("adp agents\n"), Mode: 0640},
			{Path: ".codex/config.toml", Data: []byte("model = \"test\"\n")},
		},
	})
	if err != nil {
		t.Fatalf("materialize: %v", err)
	}

	assertFileContent(t, filepath.Join(runtimeRoot, "AGENTS.md"), "adp agents\n")
	assertFileMode(t, filepath.Join(runtimeRoot, "AGENTS.md"), 0640)
	assertFileContent(t, filepath.Join(runtimeRoot, ".codex", "config.toml"), "model = \"test\"\n")
	assertSymlinkTarget(t, filepath.Join(runtimeRoot, "go.mod"), filepath.Join(projectRoot, "go.mod"))
	assertSymlinkTarget(t, filepath.Join(runtimeRoot, "internal"), filepath.Join(projectRoot, "internal"))

	if !slices.Contains(result.GeneratedPaths, "AGENTS.md") {
		t.Fatalf("generated paths missing AGENTS.md: %#v", result.GeneratedPaths)
	}
	if !slices.Contains(result.LinkedPaths, "go.mod") {
		t.Fatalf("linked paths missing go.mod: %#v", result.LinkedPaths)
	}
}

func TestSymlinkBackendGeneratedReservedPathsWinOverProjectConflicts(t *testing.T) {
	projectRoot := t.TempDir()
	writeProjectFile(t, projectRoot, "AGENTS.md", []byte("real agents\n"))
	writeProjectFile(t, projectRoot, filepath.Join(".codex", "config.toml"), []byte("real codex\n"))
	writeProjectFile(t, projectRoot, filepath.Join(".codex", "local.toml"), []byte("project local codex\n"))

	runtimeRoot := filepath.Join(t.TempDir(), "runtime")
	result, err := NewSymlinkBackend().Materialize(context.Background(), Request{
		WorkspaceName: "game-a",
		ProjectRoot:   projectRoot,
		RuntimeRoot:   runtimeRoot,
		Files: []adapters.GeneratedFile{
			{Path: "AGENTS.md", Data: []byte("adp agents\n")},
			{Path: ".codex/config.toml", Data: []byte("adp codex\n")},
		},
	})
	if err != nil {
		t.Fatalf("materialize: %v", err)
	}

	assertFileContent(t, filepath.Join(runtimeRoot, "AGENTS.md"), "adp agents\n")
	assertFileContent(t, filepath.Join(runtimeRoot, ".codex", "config.toml"), "adp codex\n")
	assertSymlinkTarget(t, filepath.Join(runtimeRoot, ".codex", "local.toml"), filepath.Join(projectRoot, ".codex", "local.toml"))
	assertFileContent(t, filepath.Join(projectRoot, "AGENTS.md"), "real agents\n")
	assertFileContent(t, filepath.Join(projectRoot, ".codex", "config.toml"), "real codex\n")
	assertNotSymlink(t, filepath.Join(runtimeRoot, ".codex"))
	assertConflictPaths(t, result.Conflicts, "AGENTS.md", filepath.Join(".codex", "config.toml"))
}

func TestSymlinkBackendMergesGeneratedDirectoriesWithProjectChildren(t *testing.T) {
	projectRoot := t.TempDir()
	writeProjectFile(t, projectRoot, filepath.Join(".claude", "settings.json"), []byte("project settings\n"))
	writeProjectFile(t, projectRoot, filepath.Join(".claude", "settings.local.json"), []byte("project local settings\n"))
	writeProjectFile(t, projectRoot, filepath.Join(".claude", "commands", "review.md"), []byte("review command\n"))

	runtimeRoot := filepath.Join(t.TempDir(), "runtime")
	result, err := NewSymlinkBackend().Materialize(context.Background(), Request{
		WorkspaceName: "game-a",
		ProjectRoot:   projectRoot,
		RuntimeRoot:   runtimeRoot,
		Files: []adapters.GeneratedFile{
			{Path: ".claude/settings.json", Data: []byte("adp settings\n")},
		},
	})
	if err != nil {
		t.Fatalf("materialize: %v", err)
	}

	assertFileContent(t, filepath.Join(runtimeRoot, ".claude", "settings.json"), "adp settings\n")
	assertSymlinkTarget(t, filepath.Join(runtimeRoot, ".claude", "settings.local.json"), filepath.Join(projectRoot, ".claude", "settings.local.json"))
	assertSymlinkTarget(t, filepath.Join(runtimeRoot, ".claude", "commands"), filepath.Join(projectRoot, ".claude", "commands"))
	assertFileContent(t, filepath.Join(projectRoot, ".claude", "settings.json"), "project settings\n")
	assertConflictPaths(t, result.Conflicts, filepath.Join(".claude", "settings.json"))
}

func TestSymlinkBackendRejectsUnsafeGeneratedFilePaths(t *testing.T) {
	projectRoot := t.TempDir()
	runtimeParent := t.TempDir()
	unsafePaths := []string{
		"",
		"/absolute",
		"../escape",
		"config/../../escape",
		"config/../safe",
	}

	for _, unsafePath := range unsafePaths {
		t.Run(unsafePath, func(t *testing.T) {
			_, err := NewSymlinkBackend().Materialize(context.Background(), Request{
				WorkspaceName: "game-a",
				ProjectRoot:   projectRoot,
				RuntimeRoot:   filepath.Join(runtimeParent, "runtime-"+safeTestName(t.Name())),
				Files: []adapters.GeneratedFile{
					{Path: unsafePath, Data: []byte("bad")},
				},
			})
			if err == nil {
				t.Fatalf("expected error for generated path %q", unsafePath)
			}
		})
	}
}

func TestSymlinkBackendCleanupRemovesRuntimeUnlessKept(t *testing.T) {
	backend := NewSymlinkBackend()

	removeRoot := filepath.Join(t.TempDir(), "remove")
	if err := os.MkdirAll(removeRoot, 0755); err != nil {
		t.Fatal(err)
	}
	if err := backend.Cleanup(context.Background(), Handle{Root: removeRoot}); err != nil {
		t.Fatalf("cleanup remove: %v", err)
	}
	if _, err := os.Stat(removeRoot); !os.IsNotExist(err) {
		t.Fatalf("expected runtime root to be removed, stat err: %v", err)
	}

	keepRoot := filepath.Join(t.TempDir(), "keep")
	if err := os.MkdirAll(keepRoot, 0755); err != nil {
		t.Fatal(err)
	}
	if err := backend.Cleanup(context.Background(), Handle{Root: keepRoot, Keep: true}); err != nil {
		t.Fatalf("cleanup keep: %v", err)
	}
	if _, err := os.Stat(keepRoot); err != nil {
		t.Fatalf("expected kept runtime root to remain: %v", err)
	}
}

func TestSymlinkBackendCleanupRefusesFilesystemRoot(t *testing.T) {
	root := filepath.Clean(filepath.VolumeName(os.TempDir()) + string(os.PathSeparator))

	err := NewSymlinkBackend().Cleanup(context.Background(), Handle{Root: root})

	if err == nil {
		t.Fatal("expected cleanup of filesystem root to fail")
	}
}

func writeProjectFile(t *testing.T, root, rel string, data []byte) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(got) != want {
		t.Fatalf("content mismatch for %s: got %q want %q", path, got, want)
	}
}

func assertFileMode(t *testing.T, path string, want fs.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("mode mismatch for %s: got %v want %v", path, got, want)
	}
}

func assertSymlinkTarget(t *testing.T, path, want string) {
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

func assertNotSymlink(t *testing.T, path string) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("lstat %s: %v", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("%s should not be a symlink", path)
	}
}

func assertConflictPaths(t *testing.T, conflicts []Conflict, want ...string) {
	t.Helper()
	got := map[string]bool{}
	for _, conflict := range conflicts {
		got[conflict.Path] = true
	}
	for _, path := range want {
		if !got[path] {
			t.Fatalf("missing conflict for %s in %#v", path, conflicts)
		}
	}
}

func safeTestName(name string) string {
	clean := ""
	for _, r := range name {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			clean += string(r)
		}
	}
	return clean
}
