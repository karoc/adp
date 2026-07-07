package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/karoc/adp/internal/adapters"
	"github.com/karoc/adp/internal/events"
	"github.com/karoc/adp/internal/paths"
	"github.com/karoc/adp/internal/runner"
	"github.com/karoc/adp/internal/runtime"
	"github.com/karoc/adp/internal/sessions"
	taskstore "github.com/karoc/adp/internal/tasks"
)

func TestSessionsRestorePlanCommandPrintsReadOnlyPlan(t *testing.T) {
	layout := setupSessionTestLayout(t)
	createTestSession(t, layout, "session-1", "game-a")

	var stdout bytes.Buffer
	var gotSessionID string

	deps := Dependencies{
		Layout: layout,
		GetSession: func(_ context.Context, _ paths.Layout, sessionID string) (*sessions.Detail, error) {
			gotSessionID = sessionID
			return &sessions.Detail{
				Summary: sessions.Summary{
					SessionID: "session-1",
					Workspace: "game-a",
					Agent:     "codex",
					Profile:   "senior",
					TaskID:    "task-1",
				},
				Events: []events.Event{{
					Type: "run_started",
					Fields: map[string]any{
						"invocation": map[string]any{
							"schema_version": 1,
							"keep_runtime":   true,
							"agent_args":     []any{"--probe", "payload value", "it's-ok"},
						},
					},
				}},
			}, nil
		},
	}

	code := NewApp(deps, &stdout, &bytes.Buffer{}).Execute(context.Background(), []string{"sessions", "restore-plan", "session-1"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if gotSessionID != "session-1" {
		t.Fatalf("session id = %q", gotSessionID)
	}
	output := stdout.String()
	for _, want := range []string{
		"session_id: session-1",
		"status: ready",
		"suggested_command: adp run codex --workspace game-a --profile senior --task task-1 --keep-runtime -- --probe 'payload value' 'it'\"'\"'s-ok'",
		"missing_fields: -",
		"restore-plan is read-only",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("restore-plan output missing %q: %q", want, output)
		}
	}
}

func TestSessionsRestorePlanCommandPrintsJSON(t *testing.T) {
	layout := setupSessionTestLayout(t)
	createTestSession(t, layout, "session-1", "game-a")

	var stdout bytes.Buffer

	deps := Dependencies{
		Layout: layout,
		GetSession: func(_ context.Context, _ paths.Layout, sessionID string) (*sessions.Detail, error) {
			if sessionID != "session-1" {
				t.Fatalf("session id = %q", sessionID)
			}
			return &sessions.Detail{
				Summary: sessions.Summary{
					SessionID: "session-1",
					Workspace: "game-a",
					Agent:     "codex",
					Profile:   "senior",
					TaskID:    "task-1",
				},
				Events: []events.Event{{
					Type: "run_started",
					Fields: map[string]any{
						"invocation": map[string]any{
							"schema_version": 1,
							"keep_runtime":   true,
							"agent_args":     []any{"--probe"},
						},
					},
				}},
			}, nil
		},
	}

	code := NewApp(deps, &stdout, &bytes.Buffer{}).Execute(context.Background(), []string{"sessions", "restore-plan", "session-1", "--format", "json"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	payload := decodeJSONObject(t, stdout.Bytes())
	assertJSONStringField(t, payload, "session_id", "session-1")
	assertJSONStringField(t, payload, "status", "ready")
	assertJSONFieldAbsent(t, payload, "missing_fields")
	command, ok := payload["suggested_command"].([]any)
	if !ok {
		t.Fatalf("suggested_command = %T, want array", payload["suggested_command"])
	}
	wantCommand := []string{"adp", "run", "codex", "--workspace", "game-a", "--profile", "senior", "--task", "task-1", "--keep-runtime", "--", "--probe"}
	if len(command) != len(wantCommand) {
		t.Fatalf("suggested_command length = %d, want %d: %#v", len(command), len(wantCommand), command)
	}
	for i, want := range wantCommand {
		got, ok := command[i].(string)
		if !ok || got != want {
			t.Fatalf("suggested_command[%d] = %#v, want %q", i, command[i], want)
		}
	}
}

func TestSessionDetailCommandsRejectBadFormatsBeforeReading(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "show", args: []string{"sessions", "show", "session-1", "--format", "yaml"}},
		{name: "restore-plan", args: []string{"sessions", "restore-plan", "session-1", "--format", "yaml"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr bytes.Buffer
			getCalled := false
			deps := Dependencies{
				GetSession: func(context.Context, paths.Layout, string) (*sessions.Detail, error) {
					getCalled = true
					return nil, nil
				},
			}

			code := NewApp(deps, &bytes.Buffer{}, &stderr).Execute(context.Background(), tt.args)

			if code != 1 {
				t.Fatalf("exit code = %d, want 1", code)
			}
			if getCalled {
				t.Fatal("GetSession should not be called")
			}
			if !strings.Contains(stderr.String(), `adp: unknown output format "yaml"`) {
				t.Fatalf("stderr = %q", stderr.String())
			}
		})
	}
}

func TestSessionsRestorePlanCommandReportsMissingSession(t *testing.T) {
	layout := setupSessionTestLayout(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	deps := Dependencies{
		Layout: layout,
		GetSession: func(_ context.Context, _ paths.Layout, sessionID string) (*sessions.Detail, error) {
			return nil, sessions.ErrNotFound
		},
	}

	code := NewApp(deps, &stdout, &stderr).Execute(context.Background(), []string{"sessions", "restore-plan", "missing"})

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "not found") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestSessionsResumePlanCommandPrintsCrossToolGuidance(t *testing.T) {
	layout := setupSessionTestLayout(t)
	createTestSession(t, layout, "session-1", "game-a")

	now := time.Now().UTC().Truncate(time.Second)
	task := testTask("task-1", "Resume task", taskstore.StatusInProgress)
	task.Owner = "codex-main"
	task.ClaimedAt = now.Add(-time.Hour)
	task.LeaseExpiresAt = now.Add(time.Hour)
	store := &fakeTaskStore{
		tasks:  []taskstore.Task{task},
		phases: []taskstore.Phase{testPhase("phase-1.5", "Resume phase", taskstore.PhaseStatusActive)},
	}
	var stdout bytes.Buffer
	var gotSessionID string
	deps := Dependencies{
		Layout:           layout,
		WorkspaceStore:   &fakeStore{cfg: testConfig()},
		TaskStoreFactory: func(string) TaskStore { return store },
		GetSession: func(_ context.Context, _ paths.Layout, sessionID string) (*sessions.Detail, error) {
			gotSessionID = sessionID
			return &sessions.Detail{
				Summary: sessions.Summary{
					SessionID: "session-1",
					Workspace: "game-a",
					Agent:     "codex",
					Profile:   "senior",
					TaskID:    "task-1",
				},
				Events: []events.Event{{
					Type: "run_started",
					Fields: map[string]any{
						"invocation": map[string]any{
							"schema_version": 1,
							"keep_runtime":   true,
							"agent_args":     []any{"--codex-only"},
						},
					},
				}},
			}, nil
		},
	}

	code := NewApp(deps, &stdout, &bytes.Buffer{}).Execute(context.Background(), []string{
		"sessions", "resume-plan", "session-1", "--agent", "claude", "--owner", "codex-main", "--lease", "2h",
	})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if gotSessionID != "session-1" {
		t.Fatalf("session id = %q", gotSessionID)
	}
	output := stdout.String()
	for _, want := range []string{
		"session_id: session-1",
		"status: ready",
		"source_agent: codex",
		"source_profile: senior",
		"target_agent: claude",
		"target_profile: -",
		"target_owner: codex-main",
		"target_lease: 2h",
		"invocation_available: true",
		"invocation_keep_runtime: true",
		"invocation_omitted: profile; agent_args",
		"provider-specific profile or agent arguments were not copied",
		"task_claim_state: leased",
		"task_resume_action: run",
		"read_only: true",
		"ADP resumes portable work context",
		"launch-resumed-worker [runtime_creation]: adp run claude --workspace game-a --task task-1 --keep-runtime",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("resume-plan output missing %q: %q", want, output)
		}
	}
}

func TestSessionsResumePlanCommandPrintsJSONAndDoesNotMutate(t *testing.T) {
	layout := setupSessionTestLayout(t)
	createTestSession(t, layout, "session-1", "game-a")

	now := time.Now().UTC().Truncate(time.Second)
	task := testTask("task-1", "Resume task", taskstore.StatusInProgress)
	task.Owner = "old-agent"
	task.LeaseExpiresAt = now.Add(-time.Minute)
	store := &fakeTaskStore{
		tasks:  []taskstore.Task{task},
		phases: []taskstore.Phase{testPhase("phase-1.5", "Resume phase", taskstore.PhaseStatusActive)},
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	deps := Dependencies{
		Layout:           layout,
		WorkspaceStore:   &fakeStore{cfg: testConfig()},
		TaskStoreFactory: func(string) TaskStore { return store },
		GetSession: func(_ context.Context, _ paths.Layout, _ string) (*sessions.Detail, error) {
			return &sessions.Detail{
				Summary: sessions.Summary{
					SessionID: "session-1",
					Workspace: "game-a",
					Agent:     "codex",
					TaskID:    "task-1",
				},
				Events: []events.Event{{
					Type: "run_started",
					Fields: map[string]any{
						"invocation": map[string]any{
							"schema_version": 1,
							"keep_runtime":   false,
							"agent_args":     []any{"--probe"},
						},
					},
				}},
			}, nil
		},
	}

	code := NewApp(deps, &stdout, &stderr).Execute(context.Background(), []string{
		"sessions", "resume-plan", "session-1", "--owner", "new-agent", "--lease", "4h", "--format", "json",
	})

	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr.String())
	}
	payload := decodeJSONObject(t, stdout.Bytes())
	assertJSONStringField(t, payload, "session_id", "session-1")
	assertJSONStringField(t, payload, "status", "ready")
	taskJSON := assertJSONObjectField(t, payload, "task")
	assertJSONStringField(t, taskJSON, "claim_state", "stale")
	assertJSONStringField(t, taskJSON, "resume_action", "claim")
	target := assertJSONObjectField(t, payload, "target")
	assertJSONStringField(t, target, "owner", "new-agent")
	guarantees := assertJSONObjectField(t, payload, "guarantees")
	assertJSONBoolField(t, guarantees, "read_only", true)
	assertJSONBoolField(t, guarantees, "task_mutation", false)
	commands := assertJSONObjectListField(t, payload, "suggested_commands")
	claimCommand := findJSONObject(t, commands, "label", "claim-task")
	assertJSONStringField(t, claimCommand, "side_effect", "task_mutation")
	launchCommand := findJSONObject(t, commands, "label", "launch-resumed-worker")
	assertJSONStringField(t, launchCommand, "side_effect", "runtime_creation")
	if store.claimReq.TaskID != "" || store.renewReq.TaskID != "" || store.updatedStatus != "" {
		t.Fatalf("resume-plan mutated fake store: claim=%+v renew=%+v status=%q", store.claimReq, store.renewReq, store.updatedStatus)
	}
}

func TestSessionsResumePlanCommandPrintsStaleSameOwnerRenewalGuidance(t *testing.T) {
	layout := setupSessionTestLayout(t)
	createTestSession(t, layout, "session-1", "game-a")

	now := time.Now().UTC().Truncate(time.Second)
	task := testTask("task-1", "Resume task", taskstore.StatusInProgress)
	task.Owner = "same-agent"
	task.LeaseExpiresAt = now.Add(-time.Minute)
	store := &fakeTaskStore{
		tasks:  []taskstore.Task{task},
		phases: []taskstore.Phase{testPhase("phase-1.5", "Resume phase", taskstore.PhaseStatusActive)},
	}
	deps := Dependencies{
		Layout:           layout,
		WorkspaceStore:   &fakeStore{cfg: testConfig()},
		TaskStoreFactory: func(string) TaskStore { return store },
		GetSession: func(_ context.Context, _ paths.Layout, _ string) (*sessions.Detail, error) {
			return &sessions.Detail{
				Summary: sessions.Summary{
					SessionID: "session-1",
					Workspace: "game-a",
					Agent:     "codex",
					TaskID:    "task-1",
				},
				Events: []events.Event{{
					Type: "run_started",
					Fields: map[string]any{
						"invocation": map[string]any{
							"schema_version": 1,
							"agent_args":     []any{"--probe"},
						},
					},
				}},
			}, nil
		},
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := NewApp(deps, &stdout, &stderr).Execute(context.Background(), []string{
		"sessions", "resume-plan", "session-1", "--owner", "same-agent", "--lease", "45m",
	})

	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{
		"task_claim_state: stale",
		"task_resume_action: renew",
		"renew-task-lease [task_mutation]: adp tasks renew --workspace game-a task-1 --owner same-agent --lease 45m",
		"launch-resumed-worker [runtime_creation]: adp run codex --workspace game-a --task task-1 -- --probe",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("resume-plan output missing %q: %q", want, output)
		}
	}
	if store.claimReq.TaskID != "" || store.renewReq.TaskID != "" || store.updatedStatus != "" {
		t.Fatalf("resume-plan mutated fake store: claim=%+v renew=%+v status=%q", store.claimReq, store.renewReq, store.updatedStatus)
	}

	stdout.Reset()
	stderr.Reset()
	code = NewApp(deps, &stdout, &stderr).Execute(context.Background(), []string{
		"sessions", "resume-plan", "session-1", "--owner", "same-agent", "--lease", "45m", "--format", "json",
	})

	if code != 0 {
		t.Fatalf("json exit code = %d, stderr = %q", code, stderr.String())
	}
	payload := decodeJSONObject(t, stdout.Bytes())
	taskJSON := assertJSONObjectField(t, payload, "task")
	assertJSONStringField(t, taskJSON, "claim_state", "stale")
	assertJSONStringField(t, taskJSON, "resume_action", "renew")
	commands := assertJSONObjectListField(t, payload, "suggested_commands")
	renewCommand := findJSONObject(t, commands, "label", "renew-task-lease")
	assertJSONStringField(t, renewCommand, "side_effect", "task_mutation")
	guarantees := assertJSONObjectField(t, payload, "guarantees")
	assertJSONBoolField(t, guarantees, "read_only", true)
	assertJSONBoolField(t, guarantees, "task_mutation", false)
	if store.claimReq.TaskID != "" || store.renewReq.TaskID != "" || store.updatedStatus != "" {
		t.Fatalf("json resume-plan mutated fake store: claim=%+v renew=%+v status=%q", store.claimReq, store.renewReq, store.updatedStatus)
	}
}

func TestSessionsReplayDryRunPrintsReadyPlan(t *testing.T) {
	layout := setupSessionTestLayout(t)
	createTestSession(t, layout, "session-1", "game-a")

	now := time.Now().UTC().Truncate(time.Second)
	task := testTask("task-1", "Replay task", taskstore.StatusInProgress)
	task.Owner = "codex-main"
	task.LeaseExpiresAt = now.Add(time.Hour)
	store := &fakeTaskStore{
		tasks:  []taskstore.Task{task},
		phases: []taskstore.Phase{testPhase("phase-1.5", "Replay phase", taskstore.PhaseStatusActive)},
	}
	var stdout bytes.Buffer
	buildRuntimeCalled := false
	deps := Dependencies{
		Layout:           layout,
		WorkspaceStore:   &fakeStore{cfg: testConfig()},
		TaskStoreFactory: func(string) TaskStore { return store },
		BuildRuntime: func(context.Context, runtime.BuildRequest) (*runtime.Handle, error) {
			buildRuntimeCalled = true
			return nil, nil
		},
		RunProcess: func(context.Context, adapters.LaunchSpec, runner.Streams) (*runner.Result, error) {
			t.Fatal("dry-run should not launch a process")
			return nil, nil
		},
		EventLogger: eventLoggerFunc(func(context.Context, events.Event) error {
			t.Fatal("dry-run should not append events")
			return nil
		}),
		GetSession: func(_ context.Context, _ paths.Layout, sessionID string) (*sessions.Detail, error) {
			if sessionID != "session-1" {
				t.Fatalf("session id = %q", sessionID)
			}
			return &sessions.Detail{
				Summary: sessions.Summary{
					SessionID: "session-1",
					Workspace: "game-a",
					Agent:     "codex",
					TaskID:    "task-1",
				},
				Events: []events.Event{{
					Type: "run_started",
					Fields: map[string]any{
						"invocation": map[string]any{
							"schema_version": 1,
							"keep_runtime":   true,
							"agent_args":     []any{"--probe", "payload"},
						},
					},
				}},
			}, nil
		},
	}

	code := NewApp(deps, &stdout, &bytes.Buffer{}).Execute(context.Background(), []string{
		"sessions", "replay", "session-1", "--dry-run", "--owner", "codex-main", "--lease", "2h",
	})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if buildRuntimeCalled {
		t.Fatal("dry-run should not build a runtime")
	}
	if store.claimReq.TaskID != "" || store.renewReq.TaskID != "" || store.updatedStatus != "" {
		t.Fatalf("dry-run mutated fake store: claim=%+v renew=%+v status=%q", store.claimReq, store.renewReq, store.updatedStatus)
	}
	output := stdout.String()
	for _, want := range []string{
		"source_session: session-1",
		"status: ready",
		"mode: dry_run",
		"task_preflight: task is owned by codex-main and lease is valid",
		"runtime: would create a new ADP runtime",
		"provider_native_resume: false",
		"git_side_effects: false",
		"project_root_writes_by_adp: false",
		"read_only: true",
		"would_mutate_task: false",
		"would_create_runtime: true",
		"launch: adp run codex --workspace game-a --task task-1 --keep-runtime -- --probe payload",
		"required_commands: -",
		"blockers: -",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("replay dry-run output missing %q: %q", want, output)
		}
	}
}

func TestSessionsReplayDryRunPrintsJSONAndDoesNotMutate(t *testing.T) {
	layout := setupSessionTestLayout(t)
	createTestSession(t, layout, "session-1", "game-a")

	now := time.Now().UTC().Truncate(time.Second)
	task := testTask("task-1", "Replay task", taskstore.StatusInProgress)
	task.Owner = "same-agent"
	task.LeaseExpiresAt = now.Add(-time.Minute)
	store := &fakeTaskStore{
		tasks:  []taskstore.Task{task},
		phases: []taskstore.Phase{testPhase("phase-1.5", "Replay phase", taskstore.PhaseStatusActive)},
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	deps := Dependencies{
		Layout:           layout,
		WorkspaceStore:   &fakeStore{cfg: testConfig()},
		TaskStoreFactory: func(string) TaskStore { return store },
		GetSession: func(_ context.Context, _ paths.Layout, _ string) (*sessions.Detail, error) {
			return &sessions.Detail{
				Summary: sessions.Summary{
					SessionID: "session-1",
					Workspace: "game-a",
					Agent:     "codex",
					TaskID:    "task-1",
				},
				Events: []events.Event{{
					Type: "run_started",
					Fields: map[string]any{
						"invocation": map[string]any{
							"schema_version": 1,
							"keep_runtime":   false,
							"agent_args":     []any{"--probe"},
						},
					},
				}},
			}, nil
		},
	}

	code := NewApp(deps, &stdout, &stderr).Execute(context.Background(), []string{
		"sessions", "replay", "session-1", "--dry-run", "--owner", "same-agent", "--lease", "45m", "--format", "json",
	})

	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr.String())
	}
	payload := decodeJSONObject(t, stdout.Bytes())
	assertJSONStringField(t, payload, "source_session_id", "session-1")
	assertJSONStringField(t, payload, "mode", "dry_run")
	assertJSONStringField(t, payload, "status", "blocked")
	assertJSONBoolField(t, payload, "read_only", true)
	assertJSONBoolField(t, payload, "would_mutate_task", false)
	assertJSONBoolField(t, payload, "would_create_runtime", false)
	assertJSONObjectListField(t, payload, "executed_commands")
	commands := assertJSONObjectListField(t, payload, "required_commands")
	renewCommand := findJSONObject(t, commands, "label", "renew-task-lease")
	assertJSONStringField(t, renewCommand, "side_effect", "task_mutation")
	guarantees := assertJSONObjectField(t, payload, "guarantees")
	assertJSONBoolField(t, guarantees, "read_only", true)
	assertJSONBoolField(t, guarantees, "runtime_creation", false)
	if store.claimReq.TaskID != "" || store.renewReq.TaskID != "" || store.updatedStatus != "" {
		t.Fatalf("dry-run mutated fake store: claim=%+v renew=%+v status=%q", store.claimReq, store.renewReq, store.updatedStatus)
	}
}
