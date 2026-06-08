package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/karoc/adp/internal/events"
	"github.com/karoc/adp/internal/paths"
)

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
				TaskID:      "task-1",
				RuntimePath: "/tmp/runtime",
				ExitCode:    &exitCode,
			}}, nil
		},
	}

	code := NewApp(deps, &stdout, &bytes.Buffer{}).Execute(
		context.Background(),
		[]string{"events", "list", "--workspace", "game-a", "--session", "session-1", "--task", "task-1", "--type", "run_finished", "--limit", "2"},
	)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if gotLayout != layout {
		t.Fatalf("layout = %+v, want %+v", gotLayout, layout)
	}
	if gotQuery.Workspace != "game-a" || gotQuery.SessionID != "session-1" || gotQuery.TaskID != "task-1" || gotQuery.Type != "run_finished" || gotQuery.Limit != 2 {
		t.Fatalf("query = %+v", gotQuery)
	}
	output := stdout.String()
	for _, want := range []string{"run_finished", "game-a", "codex", "session-1", "task-1", "0", "/tmp/runtime"} {
		if !strings.Contains(output, want) {
			t.Fatalf("events output missing %q: %q", want, output)
		}
	}
}

func TestEventsCommandReportsUnknownSubcommand(t *testing.T) {
	var stderr bytes.Buffer

	code := NewApp(Dependencies{}, &bytes.Buffer{}, &stderr).Execute(context.Background(), []string{"events", "bogus"})

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), `adp: unknown events command "bogus"`) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestEventsListCommandRejectsBadLimits(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "not integer", args: []string{"events", "list", "--limit", "many"}, want: "adp: parse limit:"},
		{name: "negative", args: []string{"events", "list", "--limit", "-1"}, want: "adp: limit must not be negative"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr bytes.Buffer
			readCalled := false
			deps := Dependencies{
				ReadEvents: func(context.Context, paths.Layout, events.Query) ([]events.Event, error) {
					readCalled = true
					return nil, nil
				},
			}

			code := NewApp(deps, &bytes.Buffer{}, &stderr).Execute(context.Background(), tt.args)

			if code != 1 {
				t.Fatalf("exit code = %d, want 1", code)
			}
			if readCalled {
				t.Fatal("ReadEvents should not be called")
			}
			if !strings.Contains(stderr.String(), tt.want) {
				t.Fatalf("stderr = %q, want to contain %q", stderr.String(), tt.want)
			}
		})
	}
}
