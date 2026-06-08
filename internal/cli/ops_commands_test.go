package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/karoc/adp/internal/events"
	"github.com/karoc/adp/internal/paths"
	"github.com/karoc/adp/internal/runtime"
	"github.com/karoc/adp/internal/sessions"
	"github.com/karoc/adp/internal/shell"
)

func TestShellHookCommandRendersHook(t *testing.T) {
	var stdout bytes.Buffer
	var gotOpts shell.HookOptions

	deps := Dependencies{
		RenderHook: func(opts shell.HookOptions) (string, error) {
			gotOpts = opts
			return "hook body\n", nil
		},
	}

	code := NewApp(deps, &stdout, &bytes.Buffer{}).Execute(
		context.Background(),
		[]string{"shell-hook", "--shell", "bash", "--name", "adp-enter"},
	)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if gotOpts.Shell != "bash" || gotOpts.FunctionName != "adp-enter" {
		t.Fatalf("hook options = %+v", gotOpts)
	}
	if stdout.String() != "hook body\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestCompletionCommandRendersCompletion(t *testing.T) {
	var stdout bytes.Buffer
	var gotOpts shell.CompletionOptions

	deps := Dependencies{
		RenderCompletion: func(opts shell.CompletionOptions) (string, error) {
			gotOpts = opts
			return "completion body\n", nil
		},
	}

	code := NewApp(deps, &stdout, &bytes.Buffer{}).Execute(
		context.Background(),
		[]string{"completion", "--shell", "zsh", "--command", "adp-dev"},
	)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if gotOpts.Shell != "zsh" || gotOpts.CommandName != "adp-dev" {
		t.Fatalf("completion options = %+v", gotOpts)
	}
	if stdout.String() != "completion body\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestEventsListCommandReadsAndPrintsEvents(t *testing.T) {
	var stdout bytes.Buffer
	var gotLayout paths.Layout
	var gotQuery events.Query
	exitCode := 0

	layout := paths.New("/tmp/adp-home", "/tmp/adp-runtime")
	deps := Dependencies{
		Layout: layout,
		ReadEvents: func(_ context.Context, layout paths.Layout, query events.Query) ([]events.Event, error) {
			gotLayout = layout
			gotQuery = query
			return []events.Event{{
				Timestamp:   time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC),
				Type:        "run_finished",
				Workspace:   "game-a",
				Agent:       "codex",
				SessionID:   "session-1",
				RuntimePath: "/tmp/runtime",
				ExitCode:    &exitCode,
			}}, nil
		},
	}

	code := NewApp(deps, &stdout, &bytes.Buffer{}).Execute(
		context.Background(),
		[]string{"events", "list", "--workspace", "game-a", "--session", "session-1", "--type", "run_finished", "--limit", "2"},
	)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if gotLayout != layout {
		t.Fatalf("layout = %+v, want %+v", gotLayout, layout)
	}
	if gotQuery.Workspace != "game-a" || gotQuery.SessionID != "session-1" || gotQuery.Type != "run_finished" || gotQuery.Limit != 2 {
		t.Fatalf("query = %+v", gotQuery)
	}
	output := stdout.String()
	for _, want := range []string{"run_finished", "game-a", "codex", "session-1", "0", "/tmp/runtime"} {
		if !strings.Contains(output, want) {
			t.Fatalf("events output missing %q: %q", want, output)
		}
	}
}

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
		[]string{"sessions", "list", "--workspace", "game-a", "--agent", "codex", "--limit", "3"},
	)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if gotLayout != layout {
		t.Fatalf("layout = %+v, want %+v", gotLayout, layout)
	}
	if gotQuery.Workspace != "game-a" || gotQuery.Agent != "codex" || gotQuery.Limit != 3 {
		t.Fatalf("query = %+v", gotQuery)
	}
	output := stdout.String()
	for _, want := range []string{"session-1", "game-a", "codex", "senior", "0", "120000", "2", "/tmp/runtime"} {
		if !strings.Contains(output, want) {
			t.Fatalf("sessions list output missing %q: %q", want, output)
		}
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
	for _, want := range []string{"session_id: session-1", "workspace: game-a", "project_root: /srv/game-a", "exit_code: 7", "duration_ms: 10", "run_started"} {
		if !strings.Contains(output, want) {
			t.Fatalf("sessions show output missing %q: %q", want, output)
		}
	}
}

func TestRuntimePruneCommandRunsPrunerAndPrintsResults(t *testing.T) {
	var stdout bytes.Buffer
	var gotReq runtime.PruneRequest

	layout := paths.New("/tmp/adp-home", "/tmp/adp-runtime")
	deps := Dependencies{
		Layout: layout,
		PruneRuntimes: func(_ context.Context, req runtime.PruneRequest) ([]runtime.PruneResult, error) {
			gotReq = req
			return []runtime.PruneResult{{
				Root:      "/tmp/adp-runtime/game-a-session",
				Workspace: "game-a",
				SessionID: "session-1",
				CreatedAt: time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
				Keep:      true,
				DryRun:    true,
			}}, nil
		},
	}

	code := NewApp(deps, &stdout, &bytes.Buffer{}).Execute(
		context.Background(),
		[]string{"runtime", "prune", "--older-than", "2h", "--include-kept", "--dry-run"},
	)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if gotReq.Layout != layout || gotReq.OlderThan != 2*time.Hour || !gotReq.IncludeKept || !gotReq.DryRun {
		t.Fatalf("prune request = %+v", gotReq)
	}
	output := stdout.String()
	for _, want := range []string{"would-remove", "game-a", "session-1", "true", "/tmp/adp-runtime/game-a-session"} {
		if !strings.Contains(output, want) {
			t.Fatalf("prune output missing %q: %q", want, output)
		}
	}
}
