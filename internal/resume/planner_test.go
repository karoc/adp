package resume

import (
	"strings"
	"testing"
	"time"

	"github.com/karoc/adp/internal/events"
	"github.com/karoc/adp/internal/sessions"
	taskstore "github.com/karoc/adp/internal/tasks"
)

func TestBuildPlanReadyForSameOwner(t *testing.T) {
	now := time.Date(2026, 6, 10, 8, 0, 0, 0, time.UTC)
	task := testTask("task-1", taskstore.StatusInProgress)
	task.Owner = "codex-main"
	task.LeaseExpiresAt = now.Add(time.Hour)
	gate := taskstore.PhaseGateStatus([]taskstore.Phase{testPhase("p50", taskstore.PhaseStatusActive)})

	plan := BuildPlan(Request{
		Detail:      testDetail("session-1", "task-1"),
		TargetAgent: "claude",
		Owner:       "codex-main",
		Now:         now,
		Task:        &task,
		PhaseGate:   &gate,
	})

	if plan.Status != StatusReady {
		t.Fatalf("status = %q, want %q", plan.Status, StatusReady)
	}
	if plan.Target.Agent != "claude" || plan.Task.ResumeAction != ActionRun || plan.Task.ClaimState != ClaimStateLeased {
		t.Fatalf("plan target/task = %+v / %+v", plan.Target, plan.Task)
	}
	if !hasCommand(plan, "launch-resumed-worker", "claude", "--task", "task-1") {
		t.Fatalf("missing launch command: %+v", plan.SuggestedCommands)
	}
	if plan.Phase == nil || plan.Phase.NextAction != taskstore.PhaseGateActionRecordAcceptance {
		t.Fatalf("phase = %+v", plan.Phase)
	}
}

func TestBuildPlanReusesInvocationForSameAgent(t *testing.T) {
	now := time.Date(2026, 6, 10, 8, 0, 0, 0, time.UTC)
	task := testTask("task-1", taskstore.StatusInProgress)
	task.Owner = "codex-main"
	task.LeaseExpiresAt = now.Add(time.Hour)

	detail := testDetail("session-1", "task-1")
	detail.Summary.Profile = "senior"
	detail.Events = []events.Event{testInvocationEvent(true, "--flag", "value with space")}

	plan := BuildPlan(Request{
		Detail: detail,
		Owner:  "codex-main",
		Now:    now,
		Task:   &task,
	})

	if plan.Status != StatusReady {
		t.Fatalf("status = %q, want %q", plan.Status, StatusReady)
	}
	if plan.Target.Profile != "senior" || plan.Invocation == nil || !contains(plan.Invocation.Reused, "agent_args") {
		t.Fatalf("target/invocation = %+v / %+v", plan.Target, plan.Invocation)
	}
	if !hasCommand(plan, "launch-resumed-worker", "--profile", "senior", "--keep-runtime", "--", "--flag", "value with space") {
		t.Fatalf("missing invocation-aware launch command: %+v", plan.SuggestedCommands)
	}
}

func TestBuildPlanOmitsProviderArgsForCrossTool(t *testing.T) {
	now := time.Date(2026, 6, 10, 8, 0, 0, 0, time.UTC)
	task := testTask("task-1", taskstore.StatusInProgress)
	task.Owner = "codex-main"
	task.LeaseExpiresAt = now.Add(time.Hour)

	detail := testDetail("session-1", "task-1")
	detail.Summary.Profile = "senior"
	detail.Events = []events.Event{testInvocationEvent(true, "--codex-only")}

	plan := BuildPlan(Request{
		Detail:      detail,
		TargetAgent: "claude",
		Owner:       "codex-main",
		Now:         now,
		Task:        &task,
	})

	if plan.Status != StatusReady {
		t.Fatalf("status = %q, want %q", plan.Status, StatusReady)
	}
	if plan.Target.Profile != "" || plan.Invocation == nil || !contains(plan.Invocation.Omitted, "profile") || !contains(plan.Invocation.Omitted, "agent_args") {
		t.Fatalf("target/invocation = %+v / %+v", plan.Target, plan.Invocation)
	}
	if hasCommand(plan, "launch-resumed-worker", "--profile") || hasCommand(plan, "launch-resumed-worker", "--codex-only") {
		t.Fatalf("cross-tool command should omit provider-specific args: %+v", plan.SuggestedCommands)
	}
	if !hasCommand(plan, "launch-resumed-worker", "--keep-runtime") {
		t.Fatalf("cross-tool command should keep ADP runtime option: %+v", plan.SuggestedCommands)
	}
}

func TestBuildPlanGuidesStaleReclaim(t *testing.T) {
	now := time.Date(2026, 6, 10, 8, 0, 0, 0, time.UTC)
	task := testTask("task-1", taskstore.StatusInProgress)
	task.Owner = "old-agent"
	task.LeaseExpiresAt = now.Add(-time.Minute)

	plan := BuildPlan(Request{
		Detail:      testDetail("session-1", "task-1"),
		TargetAgent: "codex",
		Owner:       "new-agent",
		Lease:       4 * time.Hour,
		Now:         now,
		Task:        &task,
	})

	if plan.Status != StatusReady || plan.Task.ResumeAction != ActionClaim || plan.Task.ClaimState != ClaimStateStale {
		t.Fatalf("plan = %+v task=%+v", plan, plan.Task)
	}
	if !hasCommand(plan, "inspect-stale-claims", "tasks", "stale") {
		t.Fatalf("missing stale command: %+v", plan.SuggestedCommands)
	}
	if !hasCommand(plan, "claim-task", "--owner", "new-agent", "--lease", "4h") {
		t.Fatalf("missing claim command: %+v", plan.SuggestedCommands)
	}
}

func TestBuildPlanClosedForDoneTask(t *testing.T) {
	task := testTask("task-1", taskstore.StatusDone)

	plan := BuildPlan(Request{
		Detail: testDetail("session-1", "task-1"),
		Task:   &task,
		Now:    time.Date(2026, 6, 10, 8, 0, 0, 0, time.UTC),
	})

	if plan.Status != StatusClosed || plan.Task.ResumeAction != ActionCreateTask {
		t.Fatalf("plan = %+v task=%+v", plan, plan.Task)
	}
	if hasCommand(plan, "launch-resumed-worker") {
		t.Fatalf("done task should not suggest launch: %+v", plan.SuggestedCommands)
	}
}

func TestBuildPlanPartialWithoutTaskBinding(t *testing.T) {
	plan := BuildPlan(Request{
		Detail: testDetail("session-1", ""),
		Now:    time.Date(2026, 6, 10, 8, 0, 0, 0, time.UTC),
	})

	if plan.Status != StatusPartial {
		t.Fatalf("status = %q", plan.Status)
	}
	if !contains(plan.MissingFields, "task") {
		t.Fatalf("missing fields = %#v", plan.MissingFields)
	}
	if plan.Task != nil {
		t.Fatalf("task = %+v, want nil", plan.Task)
	}
}

func TestBuildPlanPartialWhenCurrentTaskUnavailable(t *testing.T) {
	plan := BuildPlan(Request{
		Detail:        testDetail("session-1", "task-1"),
		TaskLoadError: "task not found: task-1",
		Now:           time.Date(2026, 6, 10, 8, 0, 0, 0, time.UTC),
	})

	if plan.Status != StatusPartial || !contains(plan.MissingFields, "task.current") {
		t.Fatalf("plan = %+v", plan)
	}
	if !strings.Contains(strings.Join(plan.Guidance, "\n"), "task not found") {
		t.Fatalf("guidance = %#v", plan.Guidance)
	}
}

func testDetail(sessionID string, taskID string) *sessions.Detail {
	return &sessions.Detail{
		Summary: sessions.Summary{
			SessionID:   sessionID,
			Workspace:   "game-a",
			Agent:       "codex",
			Profile:     "default",
			TaskID:      taskID,
			RuntimePath: "/tmp/runtime",
			EventCount:  2,
		},
		Events: []events.Event{testInvocationEvent(false)},
	}
}

func testInvocationEvent(keepRuntime bool, agentArgs ...string) events.Event {
	return events.Event{
		Type: "run_started",
		Fields: map[string]any{
			"invocation": map[string]any{
				"schema_version": 1,
				"keep_runtime":   keepRuntime,
				"agent_args":     agentArgs,
			},
		},
	}
}

func testTask(id string, status taskstore.Status) taskstore.Task {
	now := time.Date(2026, 6, 10, 7, 0, 0, 0, time.UTC)
	return taskstore.Task{
		ID:        id,
		Title:     "Resume work",
		Status:    status,
		Priority:  "high",
		Phase:     "p50",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func testPhase(id string, status taskstore.PhaseStatus) taskstore.Phase {
	now := time.Date(2026, 6, 10, 7, 0, 0, 0, time.UTC)
	return taskstore.Phase{ID: id, Title: "Resume phase", Status: status, CreatedAt: now, UpdatedAt: now}
}

func hasCommand(plan Plan, label string, parts ...string) bool {
	for _, command := range plan.SuggestedCommands {
		if command.Label != label {
			continue
		}
		joined := strings.Join(command.Args, " ")
		matched := true
		for _, part := range parts {
			if !strings.Contains(joined, part) {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
