package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/karoc/adp/internal/paths"
	"github.com/karoc/adp/internal/runtime"
)

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

func TestRuntimeCommandReportsUnknownSubcommand(t *testing.T) {
	var stderr bytes.Buffer

	code := NewApp(Dependencies{}, &bytes.Buffer{}, &stderr).Execute(context.Background(), []string{"runtime", "bogus"})

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), `adp: unknown runtime command "bogus"`) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRuntimePruneCommandRejectsBadDurations(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "invalid duration", args: []string{"runtime", "prune", "--older-than", "tomorrow"}, want: "adp: parse older-than duration:"},
		{name: "negative duration", args: []string{"runtime", "prune", "--older-than", "-1h"}, want: "adp: older-than must not be negative"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr bytes.Buffer
			pruneCalled := false
			deps := Dependencies{
				PruneRuntimes: func(context.Context, runtime.PruneRequest) ([]runtime.PruneResult, error) {
					pruneCalled = true
					return nil, nil
				},
			}

			code := NewApp(deps, &bytes.Buffer{}, &stderr).Execute(context.Background(), tt.args)

			if code != 1 {
				t.Fatalf("exit code = %d, want 1", code)
			}
			if pruneCalled {
				t.Fatal("PruneRuntimes should not be called")
			}
			if !strings.Contains(stderr.String(), tt.want) {
				t.Fatalf("stderr = %q, want to contain %q", stderr.String(), tt.want)
			}
		})
	}
}
