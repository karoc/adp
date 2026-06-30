package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	taskstore "github.com/karoc/adp/internal/tasks"
)

// TestTasksLookupNotFoundSuggestsList verifies that a missed task lookup
// prints an actionable "run: adp tasks list" hint so the operator can discover
// the identifiers that actually exist, instead of a bare "not found" message.
// This mirrors the actionable empty-list / did-you-mean philosophy already used
// elsewhere in the CLI.
func TestTasksLookupNotFoundSuggestsList(t *testing.T) {
	store := &fakeTaskStore{findByPrefixErr: taskstore.ErrTaskNotFound}
	deps := Dependencies{
		WorkspaceStore:   &fakeStore{cfg: testConfig()},
		TaskStoreFactory: func(string) TaskStore { return store },
	}
	var stderr bytes.Buffer

	code := NewApp(deps, &bytes.Buffer{}, &stderr).Execute(context.Background(), []string{
		"tasks", "show", "--workspace", "game-a", "nope",
	})

	if code == 0 {
		t.Fatalf("exit code = 0, want non-zero; stderr = %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "run: adp tasks list") {
		t.Fatalf("stderr missing list hint; stderr = %q", stderr.String())
	}
}

// TestPhaseLookupNotFoundSuggestsList verifies the same actionable hint for a
// missed phase lookup, pointing the operator at "adp phase list".
func TestPhaseLookupNotFoundSuggestsList(t *testing.T) {
	store := &fakeTaskStore{getPhaseErr: taskstore.ErrPhaseNotFound}
	deps := Dependencies{
		WorkspaceStore:   &fakeStore{cfg: testConfig()},
		TaskStoreFactory: func(string) TaskStore { return store },
	}
	var stderr bytes.Buffer

	code := NewApp(deps, &bytes.Buffer{}, &stderr).Execute(context.Background(), []string{
		"phase", "show", "--workspace", "game-a", "nope",
	})

	if code == 0 {
		t.Fatalf("exit code = 0, want non-zero; stderr = %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "run: adp phase list") {
		t.Fatalf("stderr missing list hint; stderr = %q", stderr.String())
	}
}

// TestAmbiguousTaskIDDoesNotSuggestList confirms the list hint is specific to a
// total miss: an ambiguous-prefix error already lists the matches, so it must
// not also print the generic "run: adp tasks list" line.
func TestAmbiguousTaskIDDoesNotSuggestList(t *testing.T) {
	store := &fakeTaskStore{tasks: []taskstore.Task{
		testTask("task-2026-1", "first", taskstore.StatusReady),
		testTask("task-2026-2", "second", taskstore.StatusReady),
	}}
	deps := Dependencies{
		WorkspaceStore:   &fakeStore{cfg: testConfig()},
		TaskStoreFactory: func(string) TaskStore { return store },
	}
	var stderr bytes.Buffer

	code := NewApp(deps, &bytes.Buffer{}, &stderr).Execute(context.Background(), []string{
		"tasks", "show", "--workspace", "game-a", "task-2026",
	})

	if code == 0 {
		t.Fatalf("exit code = 0, want non-zero; stderr = %q", stderr.String())
	}
	if strings.Contains(stderr.String(), "run: adp tasks list") {
		t.Fatalf("ambiguous error must not show generic list hint; stderr = %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "ambiguous task ID") {
		t.Fatalf("stderr missing ambiguous message; stderr = %q", stderr.String())
	}
}

// TestWorkspaceLookupNotFoundSuggestsList verifies the same actionable hint for
// a missed workspace lookup, pointing the operator at "adp workspace list".
// (Session not-found is covered by runtime acceptance because sessions.FindByPrefix
// reads the real event log rather than an injectable dep.)
func TestWorkspaceLookupNotFoundSuggestsList(t *testing.T) {
	deps := Dependencies{WorkspaceStore: &fakeStore{}}
	var stderr bytes.Buffer

	code := NewApp(deps, &bytes.Buffer{}, &stderr).Execute(context.Background(), []string{
		"workspace", "show", "nope",
	})

	if code == 0 {
		t.Fatalf("exit code = 0, want non-zero; stderr = %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "run: adp workspace list") {
		t.Fatalf("stderr missing list hint; stderr = %q", stderr.String())
	}
}
