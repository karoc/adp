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

func TestSessionsListCommandPrintsJSON(t *testing.T) {
	var stdout bytes.Buffer
	exitCode := 0
	duration := int64(120000)

	deps := Dependencies{
		ListSessions: func(_ context.Context, _ paths.Layout, query sessions.Query) ([]sessions.Summary, error) {
			if query.Workspace != "game-a" || query.Agent != "codex" || query.TaskID != "task-1" || query.Limit != 3 {
				t.Fatalf("query = %+v", query)
			}
			return []sessions.Summary{{
				SessionID:      "session-1",
				Workspace:      "game-a",
				Agent:          "codex",
				Profile:        "senior",
				TaskID:         "task-1",
				ProjectRoot:    "/srv/game-a",
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
		[]string{"sessions", "list", "--workspace", "game-a", "--agent", "codex", "--task", "task-1", "--limit", "3", "--format", "json"},
	)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	payload := decodeJSONObject(t, stdout.Bytes())
	assertJSONNumberField(t, payload, "limit", 3)
	assertJSONNumberField(t, payload, "count", 1)
	filters := assertJSONObjectField(t, payload, "filters")
	assertJSONStringField(t, filters, "workspace", "game-a")
	assertJSONStringField(t, filters, "agent", "codex")
	assertJSONStringField(t, filters, "task_id", "task-1")
	session := assertJSONObjectListField(t, payload, "sessions")[0]
	assertJSONStringField(t, session, "session_id", "session-1")
	assertJSONStringField(t, session, "workspace", "game-a")
	assertJSONStringField(t, session, "agent", "codex")
	assertJSONStringField(t, session, "profile", "senior")
	assertJSONStringField(t, session, "project_root", "/srv/game-a")
	assertJSONStringField(t, session, "runtime_path", "/tmp/runtime")
	assertJSONStringField(t, session, "task_id", "task-1")
	assertJSONStringField(t, session, "started_at", "2026-06-08T12:00:00Z")
	assertJSONStringField(t, session, "finished_at", "2026-06-08T12:02:00Z")
	assertJSONNumberField(t, session, "exit_code", 0)
	assertJSONNumberField(t, session, "duration_ms", 120000)
	assertJSONNumberField(t, session, "event_count", 2)
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
		{name: "unknown format", args: []string{"sessions", "list", "--format", "yaml"}, want: `adp: unknown output format "yaml"`},
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
	layout := setupSessionTestLayout(t)
	createTestSession(t, layout, "session-1", "game-a")

	var stdout bytes.Buffer
	var gotSessionID string
	exitCode := 7
	duration := int64(10)

	deps := Dependencies{
		Layout: layout,
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

func TestSessionsShowCommandPrintsJSON(t *testing.T) {
	layout := setupSessionTestLayout(t)
	createTestSession(t, layout, "session-1", "game-a")

	var stdout bytes.Buffer
	exitCode := 7
	duration := int64(10)

	deps := Dependencies{
		Layout: layout,
		GetSession: func(_ context.Context, _ paths.Layout, sessionID string) (*sessions.Detail, error) {
			if sessionID != "session-1" {
				t.Fatalf("session id = %q", sessionID)
			}
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
					Timestamp:      time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC),
					Type:           "run_started",
					Workspace:      "game-a",
					Agent:          "codex",
					SessionID:      "session-1",
					TaskID:         "task-1",
					RuntimePath:    "/tmp/runtime",
					DurationMillis: 10,
				}},
			}, nil
		},
	}

	code := NewApp(deps, &stdout, &bytes.Buffer{}).Execute(context.Background(), []string{"sessions", "show", "session-1", "--format", "json"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	payload := decodeJSONObject(t, stdout.Bytes())
	summary := assertJSONObjectField(t, payload, "summary")
	assertJSONStringField(t, summary, "session_id", "session-1")
	assertJSONStringField(t, summary, "project_root", "/srv/game-a")
	assertJSONStringField(t, summary, "runtime_path", "/tmp/runtime")
	assertJSONNumberField(t, summary, "exit_code", 7)
	assertJSONNumberField(t, summary, "duration_ms", 10)
	event := assertJSONObjectListField(t, payload, "events")[0]
	assertJSONStringField(t, event, "ts", "2026-06-08T12:00:00Z")
	assertJSONStringField(t, event, "type", "run_started")
	assertJSONStringField(t, event, "session_id", "session-1")
	assertJSONNumberField(t, event, "duration_ms", 10)
}

func TestSessionsShowCommandReportsMissingSession(t *testing.T) {
	layout := setupSessionTestLayout(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	deps := Dependencies{
		Layout: layout,
		GetSession: func(_ context.Context, _ paths.Layout, sessionID string) (*sessions.Detail, error) {
			return nil, sessions.ErrNotFound
		},
	}

	code := NewApp(deps, &stdout, &stderr).Execute(context.Background(), []string{"sessions", "show", "missing"})

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

func setupSessionTestLayout(t *testing.T) paths.Layout {
	t.Helper()
	tmpDir := t.TempDir()
	return paths.New(tmpDir, tmpDir)
}

func createTestSession(t *testing.T, layout paths.Layout, sessionID, workspace string) {
	t.Helper()
	ctx := context.Background()
	logger := events.NewLogger(layout)
	if err := logger.Log(ctx, events.Event{
		SessionID: sessionID,
		Workspace: workspace,
		Agent:     "test-agent",
		Type:      "run_started",
		Timestamp: time.Now(),
	}); err != nil {
		t.Fatalf("failed to log event: %v", err)
	}
}
