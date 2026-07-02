package shared

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These tests exercise the workspace file-reading and path-safety layer that
// decides what durable content ends up injected into agent instructions. The
// existing render_test.go suite builds Context with placeholder paths and never
// touches disk, so readWorkspaceFile / readExistingWorkspaceFile / workspacePath
// were almost entirely uncovered. They cover the real os.ReadFile branches and
// the path-containment guard that keeps configured paths inside the workspace.

func TestWorkspacePathContainment(t *testing.T) {
	const dir = "/tmp/adp-home/workspaces/demo"

	tests := []struct {
		name    string
		dir     string
		rel     string
		wantOK  bool
		wantEnd string // suffix the joined path must have when ok
	}{
		{name: "normal relative", dir: dir, rel: "rules.md", wantOK: true, wantEnd: "demo/rules.md"},
		{name: "nested relative", dir: dir, rel: "profiles/senior.md", wantOK: true, wantEnd: "demo/profiles/senior.md"},
		{name: "cleans redundant segments", dir: dir, rel: "profiles/./senior.md", wantOK: true, wantEnd: "demo/profiles/senior.md"},
		{name: "empty workspace dir", dir: "", rel: "rules.md", wantOK: false},
		{name: "blank workspace dir", dir: "   ", rel: "rules.md", wantOK: false},
		{name: "absolute path rejected", dir: dir, rel: "/etc/passwd", wantOK: false},
		{name: "current dir rejected", dir: dir, rel: ".", wantOK: false},
		{name: "parent dir rejected", dir: dir, rel: "..", wantOK: false},
		{name: "parent traversal rejected", dir: dir, rel: "../../etc/passwd", wantOK: false},
		{name: "traversal after clean rejected", dir: dir, rel: "profiles/../../escape", wantOK: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := workspacePath(tc.dir, tc.rel)
			if ok != tc.wantOK {
				t.Fatalf("workspacePath(%q, %q) ok = %v, want %v (path=%q)", tc.dir, tc.rel, ok, tc.wantOK, got)
			}
			if !tc.wantOK {
				if got != "" {
					t.Fatalf("rejected path should be empty, got %q", got)
				}
				return
			}
			if !strings.HasSuffix(filepath.ToSlash(got), tc.wantEnd) {
				t.Fatalf("workspacePath(%q, %q) = %q, want suffix %q", tc.dir, tc.rel, got, tc.wantEnd)
			}
		})
	}
}

func TestReadWorkspaceFileBranches(t *testing.T) {
	dir := t.TempDir()

	writeWorkspaceFile(t, dir, "rules.md", "be kind")
	writeWorkspaceFile(t, dir, "empty.md", "")

	const emptyMsg = "no content configured"

	tests := []struct {
		name string
		rel  string
		want string
	}{
		{name: "blank rel returns empty message", rel: "   ", want: emptyMsg},
		{name: "outside workspace uses default", rel: "../escape.md", want: `Configured path "../escape.md" is outside the ADP workspace directory`},
		{name: "absolute path outside workspace", rel: "/etc/passwd", want: `is outside the ADP workspace directory`},
		{name: "missing file reports missing", rel: "gone.md", want: `Configured file "gone.md" is missing`},
		{name: "empty file reports empty", rel: "empty.md", want: `Configured file "empty.md" is empty`},
		{name: "present file returns content", rel: "rules.md", want: "be kind"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := readWorkspaceFile(dir, tc.rel, emptyMsg)
			if !strings.Contains(got, tc.want) {
				t.Fatalf("readWorkspaceFile(%q) = %q, want to contain %q", tc.rel, got, tc.want)
			}
		})
	}
}

func TestReadWorkspaceFileReportsReadError(t *testing.T) {
	dir := t.TempDir()

	// A path whose parent is a regular file makes os.ReadFile fail with a
	// non-IsNotExist error (ENOTDIR), exercising the read-error branch that is
	// distinct from the missing-file branch.
	writeWorkspaceFile(t, dir, "notdir", "i am a file")

	got := readWorkspaceFile(dir, filepath.Join("notdir", "child.md"), "unused")
	if !strings.Contains(got, "could not be read") {
		t.Fatalf("readWorkspaceFile through a file parent = %q, want a read-error message", got)
	}
	if strings.Contains(got, "is missing") {
		t.Fatalf("read error must not be reported as a missing file: %q", got)
	}
}

func TestReadExistingWorkspaceFileBranches(t *testing.T) {
	dir := t.TempDir()
	writeWorkspaceFile(t, dir, filepath.Join("profiles", "senior.md"), "senior profile")
	writeWorkspaceFile(t, dir, filepath.Join("profiles", "blank.md"), "")

	if got := readExistingWorkspaceFile(dir, filepath.Join("profiles", "senior.md")); got != "senior profile" {
		t.Fatalf("existing file = %q, want %q", got, "senior profile")
	}
	if got := readExistingWorkspaceFile(dir, filepath.Join("profiles", "blank.md")); got != "" {
		t.Fatalf("empty file should return empty string, got %q", got)
	}
	if got := readExistingWorkspaceFile(dir, filepath.Join("profiles", "missing.md")); got != "" {
		t.Fatalf("missing file should return empty string, got %q", got)
	}
	if got := readExistingWorkspaceFile(dir, "../escape.md"); got != "" {
		t.Fatalf("out-of-workspace path should return empty string, got %q", got)
	}
	if got := readExistingWorkspaceFile("", "profiles/senior.md"); got != "" {
		t.Fatalf("empty workspace dir should return empty string, got %q", got)
	}
}

// writeWorkspaceFile writes content to <dir>/<rel>, creating parent dirs.
func writeWorkspaceFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	full := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o700); err != nil {
		t.Fatalf("mkdir for %s: %v", rel, err)
	}
	if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}
