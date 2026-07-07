package resume

import (
	"strings"
	"testing"
	"time"

	"github.com/karoc/adp/internal/events"
	taskstore "github.com/karoc/adp/internal/tasks"
)

func TestBuildReplayDryRunReady(t *testing.T) {
	now := time.Date(2026, 7, 7, 8, 0, 0, 0, time.UTC)
	task := testTask("task-1", taskstore.StatusInProgress)
	task.Owner = "codex-main"
	task.LeaseExpiresAt = now.Add(time.Hour)

	plan := BuildPlan(Request{
		Detail: testDetail("session-1", "task-1"),
		Owner:  "codex-main",
		Now:    now,
		Task:   &task,
	})

	dryRun := BuildReplayDryRun(plan)

	if dryRun.Status != StatusReady {
		t.Fatalf("status = %q, want %q: blockers=%#v", dryRun.Status, StatusReady, dryRun.Blockers)
	}
	if !dryRun.ReadOnly || !dryRun.Guarantees.ReadOnly {
		t.Fatalf("dry run must be read-only: %+v", dryRun)
	}
	if dryRun.WouldMutateTask {
		t.Fatalf("would mutate task = true")
	}
	if !dryRun.WouldCreateRuntime {
		t.Fatalf("would create runtime = false")
	}
	if len(dryRun.ExecutedCommands) != 0 {
		t.Fatalf("executed commands = %#v, want empty", dryRun.ExecutedCommands)
	}
	if len(dryRun.LaunchCommand) == 0 || !strings.Contains(strings.Join(dryRun.LaunchCommand, " "), "adp run codex") {
		t.Fatalf("launch command = %#v", dryRun.LaunchCommand)
	}
	if !strings.Contains(dryRun.TaskPreflight, "lease is valid") {
		t.Fatalf("task preflight = %q", dryRun.TaskPreflight)
	}
}

func TestBuildReplayDryRunBlocksTaskMutation(t *testing.T) {
	now := time.Date(2026, 7, 7, 8, 0, 0, 0, time.UTC)
	task := testTask("task-1", taskstore.StatusInProgress)
	task.Owner = "same-agent"
	task.LeaseExpiresAt = now.Add(-time.Minute)

	plan := BuildPlan(Request{
		Detail: testDetail("session-1", "task-1"),
		Owner:  "same-agent",
		Lease:  2 * time.Hour,
		Now:    now,
		Task:   &task,
	})

	dryRun := BuildReplayDryRun(plan)

	if dryRun.Status != StatusBlocked {
		t.Fatalf("status = %q, want %q", dryRun.Status, StatusBlocked)
	}
	if dryRun.WouldCreateRuntime {
		t.Fatalf("would create runtime = true for blocked replay")
	}
	if len(dryRun.RequiredCommands) != 1 || dryRun.RequiredCommands[0].Label != "renew-task-lease" {
		t.Fatalf("required commands = %#v", dryRun.RequiredCommands)
	}
	if !strings.Contains(strings.Join(dryRun.Blockers, "\n"), "explicit ADP ownership action") {
		t.Fatalf("blockers = %#v", dryRun.Blockers)
	}
}

func TestBuildReplayDryRunBlocksRedactedInvocation(t *testing.T) {
	now := time.Date(2026, 7, 7, 8, 0, 0, 0, time.UTC)
	task := testTask("task-1", taskstore.StatusInProgress)
	task.Owner = "codex-main"
	task.LeaseExpiresAt = now.Add(time.Hour)
	detail := testDetail("session-1", "task-1")
	detail.Events = []events.Event{testInvocationEvent(false, "--api-key", "sk-abc123secretvalue")}

	plan := BuildPlan(Request{
		Detail: detail,
		Owner:  "codex-main",
		Now:    now,
		Task:   &task,
	})

	dryRun := BuildReplayDryRun(plan)

	if dryRun.Status != StatusBlocked {
		t.Fatalf("status = %q, want %q", dryRun.Status, StatusBlocked)
	}
	if !strings.Contains(strings.Join(dryRun.Blockers, "\n"), "redacted agent arguments") {
		t.Fatalf("blockers = %#v", dryRun.Blockers)
	}
	if dryRun.WouldCreateRuntime {
		t.Fatalf("would create runtime = true")
	}
}

func TestBuildReplayDryRunDefersWorkspaceOnlyReplay(t *testing.T) {
	plan := BuildPlan(Request{
		Detail: testDetail("session-1", ""),
		Now:    time.Date(2026, 7, 7, 8, 0, 0, 0, time.UTC),
	})

	dryRun := BuildReplayDryRun(plan)

	if dryRun.Status != StatusPartial {
		t.Fatalf("status = %q, want %q", dryRun.Status, StatusPartial)
	}
	if !strings.Contains(strings.Join(dryRun.Blockers, "\n"), "task-bound source session") {
		t.Fatalf("blockers = %#v", dryRun.Blockers)
	}
	if !strings.Contains(dryRun.TaskPreflight, "workspace-only replay is deferred") {
		t.Fatalf("task preflight = %q", dryRun.TaskPreflight)
	}
}

func TestBuildReplayDryRunBlocksCrossWorkspaceReplay(t *testing.T) {
	now := time.Date(2026, 7, 7, 8, 0, 0, 0, time.UTC)
	task := testTask("task-1", taskstore.StatusInProgress)
	task.Owner = "codex-main"
	task.LeaseExpiresAt = now.Add(time.Hour)

	plan := BuildPlan(Request{
		Detail:    testDetail("session-1", "task-1"),
		Workspace: "game-b",
		Owner:     "codex-main",
		Now:       now,
		Task:      &task,
	})

	dryRun := BuildReplayDryRun(plan)

	if dryRun.Status != StatusBlocked {
		t.Fatalf("status = %q, want %q", dryRun.Status, StatusBlocked)
	}
	if !strings.Contains(strings.Join(dryRun.Blockers, "\n"), "cross-workspace replay is deferred") {
		t.Fatalf("blockers = %#v", dryRun.Blockers)
	}
}
