package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/karoc/adp/internal/events"
	"github.com/karoc/adp/internal/paths"
	"github.com/karoc/adp/internal/sessions"
)

func TestSessionsListCommandReadsAndPrintsSummaries(t *testing.T) {
	var stdout bytes.Buffer
	var gotLayout paths.Layout
	var gotQuery sessions.Query
	exitCode := 0
	duration := int64(120000)

	layout := paths.New("/tmp/adp-home", "/tmp/adp-runtime")
	deps := Dependencies{
		Layout: layout,
		ListSessions: func(_ context.Context, layout paths.Layout, query sessions.Query) ([]sessions.Summary, error) {
			gotLayout = layout
			gotQuery = query
			return []sessions.Summary{{
				SessionID:      "session-1",
				Workspace:      "game-a",
				Agent:          "codex",
				Profile:        "senior",
				TaskID:         "task-1",
				RuntimePath:    "/tmp/runtime",
				StartedAt:      time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC),
				FinishedAt:     time.Date(2026, 6, 8, 12, 2, 0, 0, time.UTC),
				ExitCode:       &exitCode,
				DurationMillis: &duration,
				EventCount:     2,
			}}, nil
		},
	}

	code := NewApp(deps, &stdout, &bytes.Buffer{}).Execute(
		context.Background(),
		[]string{"sessions", "list", "--workspace", "game-a", "--agent", "codex", "--task", "task-1", "--limit", "3"},
	)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if gotLayout != layout {
		t.Fatalf("layout = %+v, want %+v", gotLayout, layout)
	}
	if gotQuery.Workspace != "game-a" || gotQuery.Agent != "codex" || gotQuery.TaskID != "task-1" || gotQuery.Limit != 3 {
		t.Fatalf("query = %+v", gotQuery)
	}
	output := stdout.String()
	for _, want := range []string{"session-1", "game-a", "codex", "senior", "task-1", "0", "120000", "2", "/tmp/runtime"} {
		if !strings.Contains(output, want) {
			t.Fatalf("sessions list output missing %q: %q", want, output)
		}
	}
}

func TestSessionsCommandReportsUnknownSubcommand(t *testing.T) {
	var stderr bytes.Buffer

	code := NewApp(Dependencies{}, &bytes.Buffer{}, &stderr).Execute(context.Background(), []string{"sessions", "bogus"})

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), `adp: unknown sessions command "bogus"`) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestSessionsListCommandRejectsBadLimits(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "not integer", args: []string{"sessions", "list", "--limit", "many"}, want: "adp: parse limit:"},
		{name: "negative", args: []string{"sessions", "list", "--limit", "-1"}, want: "adp: limit must not be negative"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr bytes.Buffer
			listCalled := false
			deps := Dependencies{
				ListSessions: func(context.Context, paths.Layout, sessions.Query) ([]sessions.Summary, error) {
					listCalled = true
					return nil, nil
				},
			}

			code := NewApp(deps, &bytes.Buffer{}, &stderr).Execute(context.Background(), tt.args)

			if code != 1 {
				t.Fatalf("exit code = %d, want 1", code)
			}
			if listCalled {
				t.Fatal("ListSessions should not be called")
			}
			if !strings.Contains(stderr.String(), tt.want) {
				t.Fatalf("stderr = %q, want to contain %q", stderr.String(), tt.want)
			}
		})
	}
}

func TestSessionsShowCommandReadsAndPrintsDetail(t *testing.T) {
	var stdout bytes.Buffer
	var gotSessionID string
	exitCode := 7
	duration := int64(10)

	deps := Dependencies{
		GetSession: func(_ context.Context, _ paths.Layout, sessionID string) (*sessions.Detail, error) {
			gotSessionID = sessionID
			return &sessions.Detail{
				Summary: sessions.Summary{
					SessionID:      "session-1",
					Workspace:      "game-a",
					Agent:          "codex",
					Profile:        "senior",
					TaskID:         "task-1",
					ProjectRoot:    "/srv/game-a",
					RuntimePath:    "/tmp/runtime",
					StartedAt:      time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC),
					FinishedAt:     time.Date(2026, 6, 8, 12, 0, 1, 0, time.UTC),
					ExitCode:       &exitCode,
					DurationMillis: &duration,
					EventCount:     2,
				},
				Events: []events.Event{{
					Timestamp: time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC),
					Type:      "run_started",
					Workspace: "game-a",
					Agent:     "codex",
					SessionID: "session-1",
					TaskID:    "task-1",
				}},
			}, nil
		},
	}

	code := NewApp(deps, &stdout, &bytes.Buffer{}).Execute(context.Background(), []string{"sessions", "show", "session-1"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if gotSessionID != "session-1" {
		t.Fatalf("session id = %q", gotSessionID)
	}
	output := stdout.String()
	for _, want := range []string{"session_id: session-1", "workspace: game-a", "task_id: task-1", "project_root: /srv/game-a", "exit_code: 7", "duration_ms: 10", "run_started"} {
		if !strings.Contains(output, want) {
			t.Fatalf("sessions show output missing %q: %q", want, output)
		}
	}
}

func TestSessionsShowCommandReportsMissingSession(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var gotSessionID string

	deps := Dependencies{
		GetSession: func(_ context.Context, _ paths.Layout, sessionID string) (*sessions.Detail, error) {
			gotSessionID = sessionID
			return nil, sessions.ErrNotFound
		},
	}

	code := NewApp(deps, &stdout, &stderr).Execute(context.Background(), []string{"sessions", "show", "missing"})

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if gotSessionID != "missing" {
		t.Fatalf("session id = %q", gotSessionID)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "adp: session not found") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestSessionsRestorePlanCommandPrintsReadOnlyPlan(t *testing.T) {
	var stdout bytes.Buffer
	var gotSessionID string

	deps := Dependencies{
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

func TestSessionsRestorePlanCommandReportsMissingSession(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var gotSessionID string

	deps := Dependencies{
		GetSession: func(_ context.Context, _ paths.Layout, sessionID string) (*sessions.Detail, error) {
			gotSessionID = sessionID
			return nil, sessions.ErrNotFound
		},
	}

	code := NewApp(deps, &stdout, &stderr).Execute(context.Background(), []string{"sessions", "restore-plan", "missing"})

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if gotSessionID != "missing" {
		t.Fatalf("session id = %q", gotSessionID)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "adp: session not found") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
