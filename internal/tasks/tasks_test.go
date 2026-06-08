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

	claimed, err := store.Claim(context.Background(), ClaimRequest{TaskID: task.ID, Owner: "codex-main", Lease: 30 * time.Minute})
	if err != nil {
		t.Fatalf("Claim returned error: %v", err)
	}
	if claimed.Owner != "codex-main" || claimed.Status != StatusInProgress || claimed.ClaimedAt.IsZero() || claimed.LeaseExpiresAt.IsZero() {
		t.Fatalf("claimed task = %+v", claimed)
	}

	released, err := store.Release(context.Background(), ReleaseRequest{TaskID: task.ID, Owner: "codex-main"})
	if err != nil {
		t.Fatalf("Release returned error: %v", err)
	}
	if released.Owner != "" || released.Status != StatusReady || !released.ClaimedAt.IsZero() || !released.LeaseExpiresAt.IsZero() {
		t.Fatalf("released task = %+v", released)
	}

	progressPath := filepath.Join(store.WorkspaceDir, "planning", "progress.jsonl")
	assertFileContains(t, progressPath, `"type":"task_claimed"`)
	assertFileContains(t, progressPath, `"type":"task_released"`)
}

func TestStoreClaimConflictLeaseAndReleaseOwnerGuards(t *testing.T) {
	store := testStore(t)
	base := store.now()
	task, err := store.Add(context.Background(), AddRequest{Title: "Lease guarded task"})
	if err != nil {
		t.Fatal(err)
	}

	claimed, err := store.Claim(context.Background(), ClaimRequest{TaskID: task.ID, Owner: "agent-a", Lease: time.Hour})
	if err != nil {
		t.Fatalf("Claim returned error: %v", err)
	}
	if got, want := claimed.LeaseExpiresAt, base.Add(time.Hour); !got.Equal(want) {
		t.Fatalf("lease expires at = %s, want %s", got, want)
	}

	if _, err := store.Claim(context.Background(), ClaimRequest{TaskID: task.ID, Owner: "agent-b"}); !errors.Is(err, ErrTaskClaimed) {
		t.Fatalf("conflicting claim error = %v, want ErrTaskClaimed", err)
	}

	reclaimed, err := store.Claim(context.Background(), ClaimRequest{TaskID: task.ID, Owner: "agent-a", Lease: 2 * time.Hour})
	if err != nil {
		t.Fatalf("same-owner reclaim returned error: %v", err)
	}
	if got, want := reclaimed.LeaseExpiresAt, base.Add(2*time.Hour); !got.Equal(want) {
		t.Fatalf("renewed lease expires at = %s, want %s", got, want)
	}

	if _, err := store.Release(context.Background(), ReleaseRequest{TaskID: task.ID, Owner: "agent-b"}); !errors.Is(err, ErrTaskOwnerMismatch) {
		t.Fatalf("release owner mismatch error = %v, want ErrTaskOwnerMismatch", err)
	}

	store.Now = func() time.Time { return base.Add(3 * time.Hour) }
	claimed, err = store.Claim(context.Background(), ClaimRequest{TaskID: task.ID, Owner: "agent-b"})
	if err != nil {
		t.Fatalf("expired lease claim returned error: %v", err)
	}
	if claimed.Owner != "agent-b" || !claimed.LeaseExpiresAt.IsZero() {
		t.Fatalf("expired lease claim = %+v", claimed)
	}

	released, err := store.Release(context.Background(), ReleaseRequest{TaskID: task.ID, Owner: "agent-b"})
	if err != nil {
		t.Fatalf("matching owner release returned error: %v", err)
	}
	if released.Owner != "" || released.Status != StatusReady {
		t.Fatalf("released task = %+v", released)
	}
}

func TestStoreRejectsClaimingTerminalTasks(t *testing.T) {
	store := testStore(t)
	done, err := store.Add(context.Background(), AddRequest{Title: "Done task", Status: StatusDone})
	if err != nil {
		t.Fatal(err)
	}
	canceled, err := store.Add(context.Background(), AddRequest{Title: "Canceled task", Status: StatusCanceled})
	if err != nil {
		t.Fatal(err)
	}

	for _, task := range []Task{done, canceled} {
		if _, err := store.Claim(context.Background(), ClaimRequest{TaskID: task.ID, Owner: "agent-a"}); err == nil {
			t.Fatalf("Claim terminal task %s returned nil error", task.Status)
		}
	}
}

func TestStoreValidatesTaskPhaseWhenPhaseLedgerExists(t *testing.T) {
	store := testStore(t)

	if _, err := store.Add(context.Background(), AddRequest{Title: "Legacy free-form phase", Phase: "free-form"}); err != nil {
		t.Fatalf("Add before phase ledger returned error: %v", err)
	}
	if _, err := store.AddPhase(context.Background(), PhaseAddRequest{ID: "p3", Title: "Phase three"}); err != nil {
		t.Fatalf("AddPhase returned error: %v", err)
	}
	if _, err := store.Add(context.Background(), AddRequest{Title: "Known phase task", Phase: "p3"}); err != nil {
		t.Fatalf("Add known phase returned error: %v", err)
	}
	if _, err := store.Add(context.Background(), AddRequest{Title: "Unassigned task", Phase: defaultTaskPhase}); err != nil {
		t.Fatalf("Add unassigned phase returned error: %v", err)
	}
	if _, err := store.Add(context.Background(), AddRequest{Title: "Missing phase task", Phase: "p4"}); !errors.Is(err, ErrPhaseNotFound) {
		t.Fatalf("Add missing phase error = %v, want ErrPhaseNotFound", err)
	}
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

func TestStoreEnforcesPhaseGateTransitions(t *testing.T) {
	store := testStore(t)
	if _, err := store.AddPhase(context.Background(), PhaseAddRequest{ID: "p3", Title: "Phase three"}); err != nil {
		t.Fatalf("AddPhase p3 returned error: %v", err)
	}

	if _, err := store.RecordPhaseCommit(context.Background(), PhaseCommitRequest{ID: "p3", Hash: "abc123"}); !errors.Is(err, ErrPhaseInvalidTransition) {
		t.Fatalf("commit before accept error = %v, want ErrPhaseInvalidTransition", err)
	}

	phase, err := store.AcceptPhase(context.Background(), PhaseAcceptRequest{ID: "p3", Result: "failed", Notes: "smoke failed"})
	if err != nil {
		t.Fatalf("failed AcceptPhase returned error: %v", err)
	}
	if phase.Status == PhaseStatusAccepted || phase.Acceptance.Result != "failed" {
		t.Fatalf("failed acceptance phase = %+v", phase)
	}
	if _, err := store.RecordPhaseCommit(context.Background(), PhaseCommitRequest{ID: "p3", Hash: "abc123"}); !errors.Is(err, ErrPhaseInvalidTransition) {
		t.Fatalf("commit after failed accept error = %v, want ErrPhaseInvalidTransition", err)
	}

	phase, err = store.AcceptPhase(context.Background(), PhaseAcceptRequest{ID: "p3", Result: "passed"})
	if err != nil {
		t.Fatalf("passed AcceptPhase returned error: %v", err)
	}
	if phase.Status != PhaseStatusAccepted {
		t.Fatalf("passed acceptance status = %q, want accepted", phase.Status)
	}
	phase, err = store.AcceptPhase(context.Background(), PhaseAcceptRequest{ID: "p3", Result: "failed"})
	if err != nil {
		t.Fatalf("failed re-accept returned error: %v", err)
	}
	if phase.Status != PhaseStatusActive || phase.Acceptance.Result != "failed" {
		t.Fatalf("failed re-accept phase = %+v", phase)
	}
	if _, err := store.AddPhase(context.Background(), PhaseAddRequest{ID: "p4", Title: "Phase four"}); err != nil {
		t.Fatalf("AddPhase p4 returned error: %v", err)
	}
	if _, err := store.StartPhase(context.Background(), "p4"); !errors.Is(err, ErrPhaseInvalidTransition) {
		t.Fatalf("start p4 while p3 active error = %v, want ErrPhaseInvalidTransition", err)
	}
	phase, err = store.AcceptPhase(context.Background(), PhaseAcceptRequest{ID: "p3", Result: "passed"})
	if err != nil {
		t.Fatalf("second passed AcceptPhase returned error: %v", err)
	}
	if _, err := store.StartPhase(context.Background(), "p4"); !errors.Is(err, ErrPhaseInvalidTransition) {
		t.Fatalf("start p4 while p3 accepted error = %v, want ErrPhaseInvalidTransition", err)
	}

	if _, err := store.RecordPhasePush(context.Background(), PhasePushRequest{ID: "p3", Remote: "origin", Branch: "main"}); !errors.Is(err, ErrPhaseInvalidTransition) {
		t.Fatalf("push before commit error = %v, want ErrPhaseInvalidTransition", err)
	}
	phase, err = store.RecordPhaseCommit(context.Background(), PhaseCommitRequest{ID: "p3", Hash: "abc123"})
	if err != nil {
		t.Fatalf("RecordPhaseCommit returned error: %v", err)
	}
	phase, err = store.RecordPhasePush(context.Background(), PhasePushRequest{ID: "p3", Remote: "origin", Branch: "main", Result: "failed"})
	if err != nil {
		t.Fatalf("failed RecordPhasePush returned error: %v", err)
	}
	if phase.Status != PhaseStatusCommitted || phase.Push.Result != "failed" {
		t.Fatalf("failed push phase = %+v", phase)
	}
	phase, err = store.RecordPhasePush(context.Background(), PhasePushRequest{ID: "p3", Remote: "origin", Branch: "main", Result: "pushed"})
	if err != nil {
		t.Fatalf("pushed RecordPhasePush returned error: %v", err)
	}
	if phase.Status != PhaseStatusPushed || phase.Push.Result != "pushed" {
		t.Fatalf("pushed phase = %+v", phase)
	}
	phase, err = store.RecordPhasePush(context.Background(), PhasePushRequest{ID: "p3", Remote: "origin", Branch: "main", Result: "failed"})
	if err != nil {
		t.Fatalf("failed push after pushed returned error: %v", err)
	}
	if phase.Status != PhaseStatusPushed || phase.Push.Result != "failed" {
		t.Fatalf("failed push after pushed phase = %+v", phase)
	}
	phase, err = store.StartPhase(context.Background(), "p4")
	if err != nil {
		t.Fatalf("start p4 after p3 pushed returned error: %v", err)
	}
	if phase.Status != PhaseStatusActive {
		t.Fatalf("p4 status = %q, want active", phase.Status)
	}
}

func TestPlanningLockWaitsForFreshLockAndRemovesStaleLock(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "workspace"))
	if err := os.MkdirAll(store.planningPath(), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.lockPath(), []byte("fresh\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	if _, err := store.Add(ctx, AddRequest{Title: "Blocked by fresh lock"}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("fresh lock Add error = %v, want context deadline exceeded", err)
	}

	staleTime := time.Now().Add(-planningLockStaleAge - time.Minute)
	if err := os.Chtimes(store.lockPath(), staleTime, staleTime); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Add(context.Background(), AddRequest{Title: "Removes stale lock"}); err != nil {
		t.Fatalf("Add with stale lock returned error: %v", err)
	}
	if _, err := os.Stat(store.lockPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("lock file after Add error = %v, want not exist", err)
	}
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
