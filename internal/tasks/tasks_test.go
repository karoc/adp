package tasks

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStoreAddsListsAndGetsTasks(t *testing.T) {
	store := testStore(t)

	task, err := store.Add(context.Background(), AddRequest{
		Title:       "Add task manager",
		Description: "Persist task state outside the project root.",
		Priority:    "high",
		Phase:       "phase-1.5",
	})
	if err != nil {
		t.Fatalf("Add returned error: %v", err)
	}

	if task.ID != "task-20260608-0001" {
		t.Fatalf("task id = %q", task.ID)
	}
	if task.Status != StatusReady || task.Priority != "high" || task.Phase != "phase-1.5" {
		t.Fatalf("task fields = %+v", task)
	}

	tasks, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(tasks) != 1 || tasks[0].ID != task.ID {
		t.Fatalf("tasks = %+v", tasks)
	}

	got, err := store.Get(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got.Title != "Add task manager" {
		t.Fatalf("task title = %q", got.Title)
	}

	assertFileContains(t, filepath.Join(store.WorkspaceDir, "planning", "tasks.yaml"), "Add task manager")
	assertFileContains(t, filepath.Join(store.WorkspaceDir, "planning", "progress.jsonl"), `"type":"task_created"`)
}

func TestStoreUpdatesBlocksAndSummarizesProgress(t *testing.T) {
	store := testStore(t)

	first, err := store.Add(context.Background(), AddRequest{Title: "First"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Add(context.Background(), AddRequest{Title: "Second"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := store.UpdateStatus(context.Background(), first.ID, StatusInProgress); err != nil {
		t.Fatalf("UpdateStatus returned error: %v", err)
	}
	blocked, err := store.Block(context.Background(), second.ID, "waiting for real CLI")
	if err != nil {
		t.Fatalf("Block returned error: %v", err)
	}
	if blocked.Status != StatusBlocked || blocked.BlockedReason != "waiting for real CLI" {
		t.Fatalf("blocked task = %+v", blocked)
	}
	if _, err := store.UpdateStatus(context.Background(), second.ID, StatusDone); err != nil {
		t.Fatalf("UpdateStatus done returned error: %v", err)
	}

	progress, err := store.Progress(context.Background())
	if err != nil {
		t.Fatalf("Progress returned error: %v", err)
	}
	if progress.Total != 2 || progress.Counts[StatusInProgress] != 1 || progress.Counts[StatusDone] != 1 {
		t.Fatalf("progress = %+v", progress)
	}
	if len(progress.Next) != 1 || progress.Next[0].ID != first.ID {
		t.Fatalf("next tasks = %+v", progress.Next)
	}
	assertFileContains(t, filepath.Join(store.WorkspaceDir, "planning", "progress.jsonl"), `"type":"task_blocked"`)
}

func TestStoreClaimsAndReleasesTasks(t *testing.T) {
	store := testStore(t)
	task, err := store.Add(context.Background(), AddRequest{Title: "Phase gate MVP"})
	if err != nil {
		t.Fatal(err)
	}

	claimed, err := store.Claim(context.Background(), task.ID, "codex-main")
	if err != nil {
		t.Fatalf("Claim returned error: %v", err)
	}
	if claimed.Owner != "codex-main" || claimed.Status != StatusInProgress {
		t.Fatalf("claimed task = %+v", claimed)
	}

	released, err := store.Release(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("Release returned error: %v", err)
	}
	if released.Owner != "" || released.Status != StatusReady {
		t.Fatalf("released task = %+v", released)
	}

	progressPath := filepath.Join(store.WorkspaceDir, "planning", "progress.jsonl")
	assertFileContains(t, progressPath, `"type":"task_claimed"`)
	assertFileContains(t, progressPath, `"type":"task_released"`)
}

func TestStoreRecordsPhaseGateLifecycle(t *testing.T) {
	store := testStore(t)

	phase, err := store.AddPhase(context.Background(), PhaseAddRequest{
		ID:    "p3",
		Title: "Project planning",
		Goal:  "phase-aware task gates",
	})
	if err != nil {
		t.Fatalf("AddPhase returned error: %v", err)
	}
	if phase.Status != PhaseStatusPlanned || phase.Goal != "phase-aware task gates" {
		t.Fatalf("phase = %+v", phase)
	}

	phase, err = store.StartPhase(context.Background(), "p3")
	if err != nil {
		t.Fatalf("StartPhase returned error: %v", err)
	}
	if phase.Status != PhaseStatusActive {
		t.Fatalf("phase status = %q, want active", phase.Status)
	}

	phase, err = store.AcceptPhase(context.Background(), PhaseAcceptRequest{
		ID:       "p3",
		Commands: []string{"scripts/task-manager-smoke.sh", "scripts/check-all.sh"},
		Result:   "passed",
		Notes:    "runtime smoke accepted",
	})
	if err != nil {
		t.Fatalf("AcceptPhase returned error: %v", err)
	}
	if phase.Status != PhaseStatusAccepted || phase.Acceptance.Result != "passed" || len(phase.Acceptance.Commands) != 2 {
		t.Fatalf("accepted phase = %+v", phase)
	}

	phase, err = store.RecordPhaseCommit(context.Background(), PhaseCommitRequest{
		ID:      "p3",
		Hash:    "abc123",
		Message: "Add phase gates",
	})
	if err != nil {
		t.Fatalf("RecordPhaseCommit returned error: %v", err)
	}
	if phase.Status != PhaseStatusCommitted || phase.Commit.Hash != "abc123" {
		t.Fatalf("committed phase = %+v", phase)
	}

	phase, err = store.RecordPhasePush(context.Background(), PhasePushRequest{
		ID:     "p3",
		Remote: "origin",
		Branch: "main",
		Result: "pushed",
	})
	if err != nil {
		t.Fatalf("RecordPhasePush returned error: %v", err)
	}
	if phase.Status != PhaseStatusPushed || phase.Push.Remote != "origin" || phase.Push.Branch != "main" {
		t.Fatalf("pushed phase = %+v", phase)
	}

	got, err := store.GetPhase(context.Background(), "p3")
	if err != nil {
		t.Fatalf("GetPhase returned error: %v", err)
	}
	if got.Commit.Hash != "abc123" || got.Push.Result != "pushed" {
		t.Fatalf("stored phase evidence = %+v", got)
	}

	phases, err := store.ListPhases(context.Background())
	if err != nil {
		t.Fatalf("ListPhases returned error: %v", err)
	}
	if len(phases) != 1 || phases[0].ID != "p3" {
		t.Fatalf("phases = %+v", phases)
	}

	phasePath := filepath.Join(store.WorkspaceDir, "planning", "phases.yaml")
	progressPath := filepath.Join(store.WorkspaceDir, "planning", "progress.jsonl")
	assertFileContains(t, phasePath, "Project planning")
	assertFileContains(t, phasePath, "abc123")
	assertFileContains(t, progressPath, `"type":"phase_accepted"`)
	assertFileContains(t, progressPath, `"type":"phase_pushed"`)
}

func TestStoreReportsMissingAndInvalidTasks(t *testing.T) {
	store := testStore(t)

	if _, err := store.Add(context.Background(), AddRequest{}); err == nil {
		t.Fatal("Add without title returned nil error")
	}
	if _, err := store.UpdateStatus(context.Background(), "missing", StatusDone); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("UpdateStatus error = %v, want ErrTaskNotFound", err)
	}
	if _, err := store.Block(context.Background(), "missing", ""); err == nil {
		t.Fatal("Block without reason returned nil error")
	}
	if _, err := ParseStatus("bad"); err == nil {
		t.Fatal("ParseStatus returned nil error for bad status")
	}
	if _, err := store.GetPhase(context.Background(), "missing"); !errors.Is(err, ErrPhaseNotFound) {
		t.Fatalf("GetPhase error = %v, want ErrPhaseNotFound", err)
	}
}

func testStore(t *testing.T) *Store {
	t.Helper()
	store := NewStore(filepath.Join(t.TempDir(), "workspace"))
	store.Now = func() time.Time {
		return time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)
	}
	return store
}

func assertFileContains(t *testing.T, path string, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(data), want) {
		t.Fatalf("%s missing %q:\n%s", path, want, data)
	}
}
