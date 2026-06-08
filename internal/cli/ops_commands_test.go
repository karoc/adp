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
