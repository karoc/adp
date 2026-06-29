package tasks

import (
	"context"
	"os"
	"testing"
)

// TestStoreCreatesPlanningDirWithOwnerOnlyPermission verifies that the
// planning directory is created with 0o700 so other local users cannot
// read project paths, task content, or command history (CWE-732).
func TestStoreCreatesPlanningDirWithOwnerOnlyPermission(t *testing.T) {
	store := testStore(t)
	if _, err := store.Add(context.Background(), AddRequest{Title: "secret task"}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	info, err := os.Stat(store.planningPath())
	if err != nil {
		t.Fatalf("stat planning dir: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o700 {
		t.Fatalf("planning dir permission = %o, want 700", got)
	}
}

// TestStoreCreatesPlanningFilesWithOwnerOnlyPermission verifies the
// persisted task ledger is written with 0o600 (defense in depth) so the
// contents are not exposed to other local users even if the directory
// permission is later loosened.
func TestStoreCreatesPlanningFilesWithOwnerOnlyPermission(t *testing.T) {
	store := testStore(t)
	if _, err := store.Add(context.Background(), AddRequest{Title: "secret task"}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	info, err := os.Stat(store.tasksPath())
	if err != nil {
		t.Fatalf("stat tasks file: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("tasks file permission = %o, want 600", got)
	}
}
