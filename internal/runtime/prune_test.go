package runtime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/karoc/adp/internal/paths"
	"gopkg.in/yaml.v3"
)

func TestPruneDryRunReportsOnlyStaleADPOwnedRuntimes(t *testing.T) {
	layout := paths.New(filepath.Join(t.TempDir(), "home"), filepath.Join(t.TempDir(), "runtime-parent"))
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)
	projectRoot := t.TempDir()

	oldRoot := writePruneRuntime(t, layout.RuntimeParent, "old", Manifest{
		SessionID:   "old-session",
		Workspace:   "game-a",
		ProjectRoot: projectRoot,
		CreatedAt:   now.Add(-2 * time.Hour),
		GeneratedBy: ManifestGeneratedBy,
	})
	writePruneRuntime(t, layout.RuntimeParent, "fresh", Manifest{
		SessionID:   "fresh-session",
		Workspace:   "game-a",
		ProjectRoot: projectRoot,
		CreatedAt:   now.Add(-10 * time.Minute),
		GeneratedBy: ManifestGeneratedBy,
	})
	writePruneRuntime(t, layout.RuntimeParent, "kept", Manifest{
		SessionID:   "kept-session",
		Workspace:   "game-a",
		ProjectRoot: projectRoot,
		CreatedAt:   now.Add(-2 * time.Hour),
		Keep:        true,
		GeneratedBy: ManifestGeneratedBy,
	})
	writePruneRuntime(t, layout.RuntimeParent, "foreign", Manifest{
		SessionID:   "foreign-session",
		Workspace:   "game-a",
		ProjectRoot: projectRoot,
		CreatedAt:   now.Add(-2 * time.Hour),
		GeneratedBy: "other",
	})
	writeFile(t, filepath.Join(layout.RuntimeParent, "invalid", ManifestPath), []byte("not: [valid\n"))
	if err := os.MkdirAll(filepath.Join(layout.RuntimeParent, "missing-manifest"), 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(layout.RuntimeParent, "plain-file"), []byte("not a runtime\n"))

	results, err := Prune(context.Background(), PruneRequest{
		Layout:    layout,
		OlderThan: time.Hour,
		Now:       now,
		DryRun:    true,
	})
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("result count mismatch: got %d want 1: %#v", len(results), results)
	}

	result := results[0]
	if result.Root != oldRoot {
		t.Fatalf("root mismatch: got %s want %s", result.Root, oldRoot)
	}
	if result.Workspace != "game-a" {
		t.Fatalf("workspace mismatch: %s", result.Workspace)
	}
	if result.SessionID != "old-session" {
		t.Fatalf("session id mismatch: %s", result.SessionID)
	}
	if !result.CreatedAt.Equal(now.Add(-2 * time.Hour)) {
		t.Fatalf("created_at mismatch: %s", result.CreatedAt.Format(time.RFC3339Nano))
	}
	if result.Keep {
		t.Fatalf("keep mismatch: expected false")
	}
	if !result.DryRun {
		t.Fatalf("dry run mismatch: expected true")
	}
	if result.Removed {
		t.Fatalf("removed mismatch: expected false")
	}
	assertDirExists(t, oldRoot)
}

func TestPruneRemovesCandidatesWithoutDeletingProjectRoot(t *testing.T) {
	layout := paths.New(filepath.Join(t.TempDir(), "home"), filepath.Join(t.TempDir(), "runtime-parent"))
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)
	projectRoot := t.TempDir()
	writeFile(t, filepath.Join(projectRoot, "sentinel.txt"), []byte("keep project\n"))

	runtimeRoot := writePruneRuntime(t, layout.RuntimeParent, "old", Manifest{
		SessionID:   "old-session",
		Workspace:   "game-a",
		ProjectRoot: projectRoot,
		CreatedAt:   now.Add(-24 * time.Hour),
		GeneratedBy: ManifestGeneratedBy,
	})

	results, err := Prune(context.Background(), PruneRequest{
		Layout:    layout,
		OlderThan: time.Hour,
		Now:       now,
	})
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("result count mismatch: got %d want 1", len(results))
	}
	if !results[0].Removed {
		t.Fatalf("removed mismatch: expected true")
	}
	if results[0].DryRun {
		t.Fatalf("dry run mismatch: expected false")
	}
	if _, err := os.Stat(runtimeRoot); !os.IsNotExist(err) {
		t.Fatalf("expected runtime root removal, stat err: %v", err)
	}
	assertContent(t, filepath.Join(projectRoot, "sentinel.txt"), "keep project\n")
}

func TestPruneIncludeKeptAllowsKeptRuntimeRemoval(t *testing.T) {
	layout := paths.New(filepath.Join(t.TempDir(), "home"), filepath.Join(t.TempDir(), "runtime-parent"))
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)
	root := writePruneRuntime(t, layout.RuntimeParent, "kept", Manifest{
		SessionID:   "kept-session",
		Workspace:   "game-a",
		ProjectRoot: t.TempDir(),
		CreatedAt:   now.Add(-2 * time.Hour),
		Keep:        true,
		GeneratedBy: ManifestGeneratedBy,
	})

	results, err := Prune(context.Background(), PruneRequest{
		Layout:      layout,
		OlderThan:   time.Hour,
		Now:         now,
		IncludeKept: true,
		DryRun:      true,
	})
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("result count mismatch: got %d want 1", len(results))
	}
	if results[0].Root != root {
		t.Fatalf("root mismatch: got %s want %s", results[0].Root, root)
	}
	if !results[0].Keep {
		t.Fatalf("keep mismatch: expected true")
	}
}

func TestPruneOlderThanRequiresCreatedAtBeforeCutoff(t *testing.T) {
	layout := paths.New(filepath.Join(t.TempDir(), "home"), filepath.Join(t.TempDir(), "runtime-parent"))
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)
	projectRoot := t.TempDir()
	writePruneRuntime(t, layout.RuntimeParent, "equal-cutoff", Manifest{
		SessionID:   "equal-session",
		Workspace:   "game-a",
		ProjectRoot: projectRoot,
		CreatedAt:   now.Add(-time.Hour),
		GeneratedBy: ManifestGeneratedBy,
	})
	oldRoot := writePruneRuntime(t, layout.RuntimeParent, "before-cutoff", Manifest{
		SessionID:   "old-session",
		Workspace:   "game-a",
		ProjectRoot: projectRoot,
		CreatedAt:   now.Add(-time.Hour - time.Nanosecond),
		GeneratedBy: ManifestGeneratedBy,
	})

	results, err := Prune(context.Background(), PruneRequest{
		Layout:    layout,
		OlderThan: time.Hour,
		Now:       now,
		DryRun:    true,
	})
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("result count mismatch: got %d want 1", len(results))
	}
	if results[0].Root != oldRoot {
		t.Fatalf("root mismatch: got %s want %s", results[0].Root, oldRoot)
	}
}

func TestPruneSkipsRuntimeCreatedAtCutoff(t *testing.T) {
	layout := paths.New(filepath.Join(t.TempDir(), "home"), filepath.Join(t.TempDir(), "runtime-parent"))
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)
	root := writePruneRuntime(t, layout.RuntimeParent, "cutoff", Manifest{
		SessionID:   "cutoff-session",
		Workspace:   "game-a",
		ProjectRoot: t.TempDir(),
		CreatedAt:   now.Add(-time.Hour),
		GeneratedBy: ManifestGeneratedBy,
	})

	results, err := Prune(context.Background(), PruneRequest{
		Layout:    layout,
		OlderThan: time.Hour,
		Now:       now,
	})
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("result count mismatch: got %d want 0", len(results))
	}
	assertDirExists(t, root)
}

func TestPruneSkipsRuntimeWhenManifestProjectRootMatchesRuntimeRoot(t *testing.T) {
	layout := paths.New(filepath.Join(t.TempDir(), "home"), filepath.Join(t.TempDir(), "runtime-parent"))
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)
	root := filepath.Join(layout.RuntimeParent, "project-root-match")
	writePruneRuntime(t, layout.RuntimeParent, "project-root-match", Manifest{
		SessionID:   "unsafe-session",
		Workspace:   "game-a",
		ProjectRoot: root,
		CreatedAt:   now.Add(-2 * time.Hour),
		GeneratedBy: ManifestGeneratedBy,
	})

	results, err := Prune(context.Background(), PruneRequest{
		Layout:    layout,
		OlderThan: time.Hour,
		Now:       now,
	})
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("result count mismatch: got %d want 0", len(results))
	}
	assertDirExists(t, root)
}

func TestPruneReturnsContextCancellation(t *testing.T) {
	layout := paths.New(filepath.Join(t.TempDir(), "home"), filepath.Join(t.TempDir(), "runtime-parent"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := Prune(ctx, PruneRequest{Layout: layout})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
}

func TestPruneReturnsEmptyForMissingRuntimeParent(t *testing.T) {
	layout := paths.New(filepath.Join(t.TempDir(), "home"), filepath.Join(t.TempDir(), "missing"))

	results, err := Prune(context.Background(), PruneRequest{Layout: layout})
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("result count mismatch: got %d want 0", len(results))
	}
}

func TestPruneRejectsNegativeOlderThan(t *testing.T) {
	layout := paths.New(filepath.Join(t.TempDir(), "home"), filepath.Join(t.TempDir(), "runtime-parent"))

	_, err := Prune(context.Background(), PruneRequest{
		Layout:    layout,
		OlderThan: -time.Second,
	})
	if err == nil {
		t.Fatalf("expected negative older than error")
	}
}

func writePruneRuntime(t *testing.T, parent, name string, manifest Manifest) string {
	t.Helper()
	root := filepath.Join(parent, name)
	manifest.RuntimeRoot = root
	data, err := yaml.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	writeFile(t, filepath.Join(root, ManifestPath), data)
	return root
}

func assertDirExists(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if !info.IsDir() {
		t.Fatalf("%s is not a directory", path)
	}
}
