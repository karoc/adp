package paths

import (
	"os"
	"path/filepath"
	"testing"
)

// TestEnsurePrivateDirCreatesWithOwnerOnlyPermission verifies a freshly
// created directory tree is owner-only (0o700) so other local users cannot
// traverse or read ADP state (CWE-732).
func TestEnsurePrivateDirCreatesWithOwnerOnlyPermission(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "private")
	if err := EnsurePrivateDir(dir); err != nil {
		t.Fatalf("EnsurePrivateDir: %v", err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if got := info.Mode().Perm(); got != PrivateDirMode {
		t.Fatalf("permission = %o, want %o", got, PrivateDirMode)
	}
}

// TestEnsurePrivateDirTightensExistingPermission verifies a pre-existing
// directory created with loose permissions is tightened to 0o700.
func TestEnsurePrivateDirTightensExistingPermission(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "loose")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("seed dir: %v", err)
	}
	if err := EnsurePrivateDir(dir); err != nil {
		t.Fatalf("EnsurePrivateDir: %v", err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if got := info.Mode().Perm(); got != PrivateDirMode {
		t.Fatalf("permission = %o, want %o (not tightened)", got, PrivateDirMode)
	}
}

// TestEnsurePrivateDirRejectsEmptyPath guards against accidentally operating
// on the process working directory when a path is missing.
func TestEnsurePrivateDirRejectsEmptyPath(t *testing.T) {
	if err := EnsurePrivateDir(""); err == nil {
		t.Fatal("expected error for empty path, got nil")
	}
}
