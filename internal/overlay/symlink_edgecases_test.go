package overlay

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/karoc/adp/internal/adapters"
)

// TestMaterializeRejectsInvalidRequests covers the input-validation guards in
// Materialize that reject malformed requests before any file is written.
func TestMaterializeRejectsInvalidRequests(t *testing.T) {
	projectRoot := t.TempDir()
	writeProjectFile(t, projectRoot, "go.mod", []byte("module example\n"))
	notADir := filepath.Join(projectRoot, "go.mod")

	cases := []struct {
		name    string
		req     Request
		wantMsg string
	}{
		{
			name: "empty_workspace_name",
			req: Request{
				WorkspaceName: "",
				ProjectRoot:   projectRoot,
				RuntimeRoot:   filepath.Join(t.TempDir(), "runtime"),
			},
			wantMsg: "workspace name is required",
		},
		{
			name: "relative_project_root",
			req: Request{
				WorkspaceName: "game-a",
				ProjectRoot:   "relative/project",
				RuntimeRoot:   filepath.Join(t.TempDir(), "runtime"),
			},
			wantMsg: "must be absolute",
		},
		{
			name: "empty_project_root",
			req: Request{
				WorkspaceName: "game-a",
				ProjectRoot:   "",
				RuntimeRoot:   filepath.Join(t.TempDir(), "runtime"),
			},
			wantMsg: "project root is required",
		},
		{
			name: "project_root_missing",
			req: Request{
				WorkspaceName: "game-a",
				ProjectRoot:   filepath.Join(projectRoot, "does-not-exist"),
				RuntimeRoot:   filepath.Join(t.TempDir(), "runtime"),
			},
			wantMsg: "project root",
		},
		{
			name: "project_root_not_a_directory",
			req: Request{
				WorkspaceName: "game-a",
				ProjectRoot:   notADir,
				RuntimeRoot:   filepath.Join(t.TempDir(), "runtime"),
			},
			wantMsg: "is not a directory",
		},
		{
			name: "relative_runtime_root",
			req: Request{
				WorkspaceName: "game-a",
				ProjectRoot:   projectRoot,
				RuntimeRoot:   "relative/runtime",
			},
			wantMsg: "must be absolute",
		},
		{
			name: "empty_runtime_root",
			req: Request{
				WorkspaceName: "game-a",
				ProjectRoot:   projectRoot,
				RuntimeRoot:   "",
			},
			wantMsg: "runtime root is required",
		},
		{
			name: "reserved_path_traversal",
			req: Request{
				WorkspaceName: "game-a",
				ProjectRoot:   projectRoot,
				RuntimeRoot:   filepath.Join(t.TempDir(), "runtime"),
				ReservedPaths: []string{"../escape"},
			},
			wantMsg: "reserved path",
		},
		{
			name: "generated_file_is_directory",
			req: Request{
				WorkspaceName: "game-a",
				ProjectRoot:   projectRoot,
				RuntimeRoot:   filepath.Join(t.TempDir(), "runtime"),
				Files: []adapters.GeneratedFile{
					{Path: "somedir", Mode: os.ModeDir | 0o755},
				},
			},
			wantMsg: "must not be a directory",
		},
		{
			name: "generated_file_duplicated",
			req: Request{
				WorkspaceName: "game-a",
				ProjectRoot:   projectRoot,
				RuntimeRoot:   filepath.Join(t.TempDir(), "runtime"),
				Files: []adapters.GeneratedFile{
					{Path: "AGENTS.md", Data: []byte("one")},
					{Path: "AGENTS.md", Data: []byte("two")},
				},
			},
			wantMsg: "is duplicated",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewSymlinkBackend().Materialize(context.Background(), tc.req)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantMsg)
			}
			if !strings.Contains(err.Error(), tc.wantMsg) {
				t.Fatalf("error = %q, want it to contain %q", err.Error(), tc.wantMsg)
			}
		})
	}
}

// TestMaterializeCanceledContext covers the ctx.Err() short-circuit at the top
// of Materialize.
func TestMaterializeCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := NewSymlinkBackend().Materialize(ctx, Request{
		WorkspaceName: "game-a",
		ProjectRoot:   t.TempDir(),
		RuntimeRoot:   filepath.Join(t.TempDir(), "runtime"),
	})
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}

// TestMaterializeRejectsSymlinkedGeneratedParent proves the symlink-parent
// guard: if a parent directory in the runtime tree is a symlink (e.g. an
// attacker pre-seeded one pointing outside the runtime root), the generated
// file must not be written through it. This is the core anti-escape defense
// for generated files.
func TestMaterializeRejectsSymlinkedGeneratedParent(t *testing.T) {
	requireSymlinks(t)
	projectRoot := t.TempDir()

	runtimeRoot := filepath.Join(t.TempDir(), "runtime")
	if err := os.MkdirAll(runtimeRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	// Pre-seed a symlinked "nested" directory inside the runtime root pointing
	// at an outside location.
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(runtimeRoot, "nested")); err != nil {
		t.Fatalf("seed symlink parent: %v", err)
	}

	_, err := NewSymlinkBackend().Materialize(context.Background(), Request{
		WorkspaceName: "game-a",
		ProjectRoot:   projectRoot,
		RuntimeRoot:   runtimeRoot,
		Files: []adapters.GeneratedFile{
			{Path: "nested/config.toml", Data: []byte("escape")},
		},
	})
	if err == nil {
		t.Fatal("expected error when a generated file's parent is a symlink")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("error = %q, want it to mention symlink", err.Error())
	}
	// The write must not have leaked through the symlink into the outside dir.
	if _, statErr := os.Stat(filepath.Join(outside, "config.toml")); !os.IsNotExist(statErr) {
		t.Fatalf("generated file leaked through symlink parent: %v", statErr)
	}
}

// TestMaterializeRuntimePathAlreadyExistsConflict covers the branch where a
// runtime path already exists as a non-directory (or a symlink) so the project
// child cannot be linked and is reported as a conflict rather than merged.
func TestMaterializeRuntimePathAlreadyExistsConflict(t *testing.T) {
	requireSymlinks(t)
	projectRoot := t.TempDir()
	// A project directory that will collide with a pre-existing runtime file.
	writeProjectFile(t, projectRoot, filepath.Join("data", "keep.txt"), []byte("keep\n"))

	runtimeRoot := filepath.Join(t.TempDir(), "runtime")
	if err := os.MkdirAll(runtimeRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	// Pre-create a plain FILE named "data" in the runtime root; the project has
	// a DIRECTORY named "data", so merge must fail and it becomes a conflict.
	if err := os.WriteFile(filepath.Join(runtimeRoot, "data"), []byte("runtime file\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := NewSymlinkBackend().Materialize(context.Background(), Request{
		WorkspaceName: "game-a",
		ProjectRoot:   projectRoot,
		RuntimeRoot:   runtimeRoot,
	})
	if err != nil {
		t.Fatalf("materialize: %v", err)
	}
	assertConflictPaths(t, result.Conflicts, "data")
	// The pre-existing runtime file must be preserved untouched.
	assertFileContent(t, filepath.Join(runtimeRoot, "data"), "runtime file\n")
}

// TestCleanupCanceledContext covers the ctx.Err() short-circuit in Cleanup.
func TestCleanupCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := NewSymlinkBackend().Cleanup(ctx, Handle{Root: t.TempDir()}); err == nil {
		t.Fatal("expected error for canceled context")
	}
}

// TestCleanupRejectsEmptyAndRelativeRoot covers the guards that refuse to run a
// destructive RemoveAll against a missing or non-absolute root.
func TestCleanupRejectsEmptyAndRelativeRoot(t *testing.T) {
	t.Run("empty_root", func(t *testing.T) {
		if err := NewSymlinkBackend().Cleanup(context.Background(), Handle{Root: ""}); err == nil {
			t.Fatal("expected error for empty runtime root")
		}
	})
	t.Run("relative_root", func(t *testing.T) {
		if err := NewSymlinkBackend().Cleanup(context.Background(), Handle{Root: "relative/runtime"}); err == nil {
			t.Fatal("expected error for relative runtime root")
		}
	})
}

// TestMaterializeGeneratedFileCollidesWithExistingRuntimeFile covers the
// writeNewFile O_EXCL branch: a generated file must never silently overwrite a
// file that already exists at its runtime target. This protects against
// clobbering content left in a reused runtime root.
func TestMaterializeGeneratedFileCollidesWithExistingRuntimeFile(t *testing.T) {
	projectRoot := t.TempDir()

	runtimeRoot := filepath.Join(t.TempDir(), "runtime")
	if err := os.MkdirAll(runtimeRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	// Pre-existing file at the exact runtime target the generated file wants.
	if err := os.WriteFile(filepath.Join(runtimeRoot, "AGENTS.md"), []byte("pre-existing\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := NewSymlinkBackend().Materialize(context.Background(), Request{
		WorkspaceName: "game-a",
		ProjectRoot:   projectRoot,
		RuntimeRoot:   runtimeRoot,
		Files: []adapters.GeneratedFile{
			{Path: "AGENTS.md", Data: []byte("generated\n")},
		},
	})
	if err == nil {
		t.Fatal("expected error when a generated file collides with an existing runtime file")
	}
	// The pre-existing content must be preserved (O_EXCL refused to overwrite).
	assertFileContent(t, filepath.Join(runtimeRoot, "AGENTS.md"), "pre-existing\n")
}
