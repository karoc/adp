package resume

import (
	"strings"
	"testing"
	"time"

	taskstore "github.com/karoc/adp/internal/tasks"
)

// TestBuildPlanClassifyTaskMatrix exercises the decision branches of
// classifyTask that the existing tests do not reach: blocked tasks, canceled
// tasks, unowned tasks with and without a target owner, and cross-owner
// contention in both leased and stale states.
func TestBuildPlanClassifyTaskMatrix(t *testing.T) {
	now := time.Date(2026, 6, 10, 8, 0, 0, 0, time.UTC)

	cases := []struct {
		name        string
		mutateTask  func(*taskstore.Task)
		targetOwner string
		wantStatus  string
		wantAction  string
		wantSummary string // substring
	}{
		{
			name: "canceled_task_suggests_followup",
			mutateTask: func(task *taskstore.Task) {
				task.Status = taskstore.StatusCanceled
			},
			wantStatus:  StatusClosed,
			wantAction:  ActionCreateTask,
			wantSummary: "create or choose follow-up work",
		},
		{
			name: "blocked_task_with_reason",
			mutateTask: func(task *taskstore.Task) {
				task.Status = taskstore.StatusBlocked
				task.BlockedReason = "waiting on upstream API"
			},
			wantStatus:  StatusBlocked,
			wantAction:  ActionResolveBlocker,
			wantSummary: "waiting on upstream API",
		},
		{
			name: "blocked_task_without_reason",
			mutateTask: func(task *taskstore.Task) {
				task.Status = taskstore.StatusBlocked
			},
			wantStatus:  StatusBlocked,
			wantAction:  ActionResolveBlocker,
			wantSummary: "resolve the blocker",
		},
		{
			name: "unowned_task_no_target_owner_is_partial",
			mutateTask: func(task *taskstore.Task) {
				task.Status = taskstore.StatusInProgress
				task.Owner = ""
			},
			targetOwner: "",
			wantStatus:  StatusPartial,
			wantAction:  ActionClaim,
			wantSummary: "is not owned",
		},
		{
			name: "unowned_task_with_target_owner_is_ready",
			mutateTask: func(task *taskstore.Task) {
				task.Status = taskstore.StatusInProgress
				task.Owner = ""
			},
			targetOwner: "new-agent",
			wantStatus:  StatusReady,
			wantAction:  ActionClaim,
			wantSummary: "is not owned",
		},
		{
			name: "owned_by_other_no_target_owner_leased_waits",
			mutateTask: func(task *taskstore.Task) {
				task.Status = taskstore.StatusInProgress
				task.Owner = "other-agent"
				task.LeaseExpiresAt = now.Add(time.Hour)
			},
			targetOwner: "",
			wantStatus:  StatusBlocked,
			wantAction:  ActionWait,
			wantSummary: "already owned by other-agent",
		},
		{
			name: "owned_by_other_no_target_owner_stale_needs_owner",
			mutateTask: func(task *taskstore.Task) {
				task.Status = taskstore.StatusInProgress
				task.Owner = "other-agent"
				task.LeaseExpiresAt = now.Add(-time.Minute)
			},
			targetOwner: "",
			wantStatus:  StatusPartial,
			wantAction:  ActionClaim,
			wantSummary: "stale claim by other-agent",
		},
		{
			name: "owned_by_target_stale_renews",
			mutateTask: func(task *taskstore.Task) {
				task.Status = taskstore.StatusInProgress
				task.Owner = "codex-main"
				task.LeaseExpiresAt = now.Add(-time.Minute)
			},
			targetOwner: "codex-main",
			wantStatus:  StatusReady,
			wantAction:  ActionRenew,
			wantSummary: "lease is stale; renew",
		},
		{
			name: "owned_by_other_leased_target_present_waits",
			mutateTask: func(task *taskstore.Task) {
				task.Status = taskstore.StatusInProgress
				task.Owner = "other-agent"
				task.LeaseExpiresAt = now.Add(time.Hour)
			},
			targetOwner: "new-agent",
			wantStatus:  StatusBlocked,
			wantAction:  ActionWait,
			wantSummary: "still owned by other-agent",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			task := testTask("task-1", taskstore.StatusInProgress)
			tc.mutateTask(&task)

			plan := BuildPlan(Request{
				Detail:      testDetail("session-1", "task-1"),
				TargetAgent: "codex",
				Owner:       tc.targetOwner,
				Now:         now,
				Task:        &task,
			})

			if plan.Status != tc.wantStatus {
				t.Fatalf("status = %q, want %q", plan.Status, tc.wantStatus)
			}
			if plan.Task == nil {
				t.Fatalf("task state is nil")
			}
			if plan.Task.ResumeAction != tc.wantAction {
				t.Fatalf("resume action = %q, want %q", plan.Task.ResumeAction, tc.wantAction)
			}
			if !strings.Contains(plan.Summary, tc.wantSummary) {
				t.Fatalf("summary = %q, want it to contain %q", plan.Summary, tc.wantSummary)
			}
		})
	}
}

// TestBuildPlanRenewCommandForStaleSelfLease proves the renew action emits a
// renew-task-lease command (addRenewCommand, previously uncovered) using the
// target owner and lease.
func TestBuildPlanRenewCommandForStaleSelfLease(t *testing.T) {
	now := time.Date(2026, 6, 10, 8, 0, 0, 0, time.UTC)
	task := testTask("task-1", taskstore.StatusInProgress)
	task.Owner = "codex-main"
	task.LeaseExpiresAt = now.Add(-time.Minute)

	plan := BuildPlan(Request{
		Detail:      testDetail("session-1", "task-1"),
		TargetAgent: "codex",
		Owner:       "codex-main",
		Lease:       2 * time.Hour,
		Now:         now,
		Task:        &task,
	})

	if plan.Task.ResumeAction != ActionRenew {
		t.Fatalf("resume action = %q, want %q", plan.Task.ResumeAction, ActionRenew)
	}
	if !hasCommand(plan, "renew-task-lease", "tasks", "renew", "--owner", "codex-main", "--lease", "2h") {
		t.Fatalf("missing renew command: %+v", plan.SuggestedCommands)
	}
	// A stale self-lease should still be launchable after renewal.
	if !hasCommand(plan, "launch-resumed-worker", "--task", "task-1") {
		t.Fatalf("missing launch command: %+v", plan.SuggestedCommands)
	}
}

// TestBuildPlanBlockedTaskInspectsButDoesNotLaunch proves a blocked task
// suggests inspection and never a launch command.
func TestBuildPlanBlockedTaskInspectsButDoesNotLaunch(t *testing.T) {
	now := time.Date(2026, 6, 10, 8, 0, 0, 0, time.UTC)
	task := testTask("task-1", taskstore.StatusBlocked)
	task.BlockedReason = "waiting on review"

	plan := BuildPlan(Request{
		Detail:      testDetail("session-1", "task-1"),
		TargetAgent: "codex",
		Owner:       "codex-main",
		Now:         now,
		Task:        &task,
	})

	if !hasCommand(plan, "inspect-task", "tasks", "show") {
		t.Fatalf("missing inspect-task command: %+v", plan.SuggestedCommands)
	}
	if hasCommand(plan, "launch-resumed-worker") {
		t.Fatalf("blocked task should not suggest launch: %+v", plan.SuggestedCommands)
	}
}

// TestBuildPlanWorkspaceOverrideGuidance covers applyWorkspaceOverrideGuidance:
// when the target workspace differs from the source workspace, the plan must
// carry an explicit reason so the operator knows current task/phase state was
// read from a different workspace than the source session.
func TestBuildPlanWorkspaceOverrideGuidance(t *testing.T) {
	now := time.Date(2026, 6, 10, 8, 0, 0, 0, time.UTC)
	task := testTask("task-1", taskstore.StatusInProgress)
	task.Owner = "codex-main"
	task.LeaseExpiresAt = now.Add(time.Hour)

	// Source session recorded in game-a (from testDetail); override to game-b.
	plan := BuildPlan(Request{
		Detail:      testDetail("session-1", "task-1"),
		Workspace:   "game-b",
		TargetAgent: "codex",
		Owner:       "codex-main",
		Now:         now,
		Task:        &task,
	})

	if plan.Target.Workspace != "game-b" {
		t.Fatalf("target workspace = %q, want game-b", plan.Target.Workspace)
	}
	joined := strings.Join(plan.Guidance, "\n")
	if !strings.Contains(joined, "Workspace override is active") {
		t.Fatalf("guidance missing workspace-override reason: %#v", plan.Guidance)
	}
	if !strings.Contains(joined, "game-b") || !strings.Contains(joined, "game-a") {
		t.Fatalf("guidance should name both workspaces: %#v", plan.Guidance)
	}
	if !contains(plan.Reasons, "Workspace override is active; current task and phase state were read from game-b, while the source session was recorded in game-a.") {
		t.Fatalf("reasons missing workspace-override entry: %#v", plan.Reasons)
	}
}

// TestBuildPlanNilDetailIsPartial covers the earliest short-circuit in
// BuildPlan: a request with no session detail at all.
func TestBuildPlanNilDetailIsPartial(t *testing.T) {
	plan := BuildPlan(Request{Now: time.Date(2026, 6, 10, 8, 0, 0, 0, time.UTC)})

	if plan.Status != StatusPartial {
		t.Fatalf("status = %q, want %q", plan.Status, StatusPartial)
	}
	if !contains(plan.MissingFields, "session") {
		t.Fatalf("missing fields = %#v", plan.MissingFields)
	}
	if plan.Task != nil {
		t.Fatalf("task = %+v, want nil", plan.Task)
	}
}

// TestBuildPlanDefaultsNowWhenZero proves BuildPlan tolerates a zero Now by
// falling back to the current time (the now.IsZero() branch), still producing a
// coherent plan.
func TestBuildPlanDefaultsNowWhenZero(t *testing.T) {
	task := testTask("task-1", taskstore.StatusInProgress)
	task.Owner = "codex-main"
	task.LeaseExpiresAt = time.Now().Add(time.Hour)

	plan := BuildPlan(Request{
		Detail:      testDetail("session-1", "task-1"),
		TargetAgent: "codex",
		Owner:       "codex-main",
		// Now intentionally left zero.
		Task: &task,
	})

	if plan.Status != StatusReady || plan.Task.ResumeAction != ActionRun {
		t.Fatalf("plan = %+v task = %+v", plan, plan.Task)
	}
}

// TestBuildPlanPhaseGapWhenGateMissingOrErrored covers applyPhase's two
// degraded branches: a phase load error and a nil gate.
func TestBuildPlanPhaseGapWhenGateMissingOrErrored(t *testing.T) {
	now := time.Date(2026, 6, 10, 8, 0, 0, 0, time.UTC)
	task := testTask("task-1", taskstore.StatusInProgress)
	task.Owner = "codex-main"
	task.LeaseExpiresAt = now.Add(time.Hour)

	t.Run("phase_load_error", func(t *testing.T) {
		plan := BuildPlan(Request{
			Detail:         testDetail("session-1", "task-1"),
			TargetAgent:    "codex",
			Owner:          "codex-main",
			Now:            now,
			Task:           &task,
			PhaseLoadError: "phase ledger unreadable",
		})
		if !contains(plan.MissingFields, "phase.current") {
			t.Fatalf("missing fields = %#v", plan.MissingFields)
		}
		if !strings.Contains(strings.Join(plan.Guidance, "\n"), "phase ledger unreadable") {
			t.Fatalf("guidance = %#v", plan.Guidance)
		}
	})

	t.Run("nil_gate", func(t *testing.T) {
		plan := BuildPlan(Request{
			Detail:      testDetail("session-1", "task-1"),
			TargetAgent: "codex",
			Owner:       "codex-main",
			Now:         now,
			Task:        &task,
			// PhaseGate left nil.
		})
		if !contains(plan.MissingFields, "phase.current") {
			t.Fatalf("missing fields = %#v", plan.MissingFields)
		}
		if !strings.Contains(strings.Join(plan.Guidance, "\n"), "phase gate state is unavailable") {
			t.Fatalf("guidance = %#v", plan.Guidance)
		}
	})
}

// TestDurationString covers the duration formatting branches used to render the
// target lease string.
func TestDurationString(t *testing.T) {
	cases := []struct {
		in   time.Duration
		want string
	}{
		{0, ""},
		{4 * time.Hour, "4h"},
		{30 * time.Minute, "30m"},
		{45 * time.Second, "45s"},
		{90 * time.Minute, "1h30m"},
		{time.Hour + 30*time.Minute + 15*time.Second, "1h30m15s"},
	}
	for _, tc := range cases {
		if got := durationString(tc.in); got != tc.want {
			t.Fatalf("durationString(%s) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
