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

func TestSessionsShowWithPrefix(t *testing.T) {
	ctx := context.Background()
	layout := setupTestLayout(t)
	logger := events.NewLogger(layout)

	// Create test sessions
	now := time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC)
	testSessions := []struct {
		id        string
		workspace string
	}{
		{"20260611T102030-abc123", "workspace1"},
		{"20260612T143045-def456", "workspace1"},
		{"20260613T090000-xyz789", "workspace2"},
	}

	for i, session := range testSessions {
		if err := logger.Log(ctx, events.Event{
			SessionID: session.id,
			Workspace: session.workspace,
			Agent:     "test-agent",
			Type:      "run_started",
			Timestamp: now.Add(time.Duration(i) * time.Minute),
		}); err != nil {
			t.Fatalf("failed to log event: %v", err)
		}
	}

	tests := []struct {
		name        string
		sessionID   string
		wantErr     string
		wantGetCall string
	}{
		{
			name:        "exact match",
			sessionID:   "20260611T102030-abc123",
			wantGetCall: "20260611T102030-abc123",
		},
		{
			name:        "unique prefix match",
			sessionID:   "20260611",
			wantGetCall: "20260611T102030-abc123",
		},
		{
			name:      "ambiguous prefix",
			sessionID: "202606",
			wantErr:   "ambiguous session ID",
		},
		{
			name:      "no match",
			sessionID: "notfound",
			wantErr:   "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			var gotSessionID string

			deps := Dependencies{
				Layout: layout,
				GetSession: func(ctx context.Context, layout paths.Layout, sessionID string) (*sessions.Detail, error) {
					gotSessionID = sessionID
					return sessions.Get(ctx, layout, sessionID)
				},
			}

			code := NewApp(deps, &stdout, &stderr).Execute(
				context.Background(),
				[]string{"sessions", "show", tt.sessionID},
			)

			if tt.wantErr != "" {
				if code == 0 {
					t.Fatalf("expected error containing %q, got success", tt.wantErr)
				}
				output := stderr.String()
				if !strings.Contains(output, tt.wantErr) {
					t.Fatalf("error output %q does not contain %q", output, tt.wantErr)
				}
				return
			}

			if code != 0 {
				t.Fatalf("exit code = %d, want 0, stderr: %s", code, stderr.String())
			}
			if gotSessionID != tt.wantGetCall {
				t.Fatalf("GetSession called with %q, want %q", gotSessionID, tt.wantGetCall)
			}
		})
	}
}

func TestSessionsRestorePlanWithPrefix(t *testing.T) {
	ctx := context.Background()
	layout := setupTestLayout(t)
	logger := events.NewLogger(layout)

	// Create a test session
	now := time.Date(2026, 6, 11, 10, 20, 30, 0, time.UTC)
	if err := logger.Log(ctx, events.Event{
		SessionID: "20260611T102030-abc123",
		Workspace: "workspace1",
		Agent:     "test-agent",
		Type:      "run_started",
		Timestamp: now,
	}); err != nil {
		t.Fatalf("failed to log event: %v", err)
	}

	var stdout, stderr bytes.Buffer
	var gotSessionID string

	deps := Dependencies{
		Layout: layout,
		GetSession: func(ctx context.Context, layout paths.Layout, sessionID string) (*sessions.Detail, error) {
			gotSessionID = sessionID
			return sessions.Get(ctx, layout, sessionID)
		},
	}

	code := NewApp(deps, &stdout, &stderr).Execute(
		context.Background(),
		[]string{"sessions", "restore-plan", "20260611"},
	)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0, stderr: %s", code, stderr.String())
	}
	if gotSessionID != "20260611T102030-abc123" {
		t.Fatalf("GetSession called with %q, want %q", gotSessionID, "20260611T102030-abc123")
	}
}

func TestSessionsResumePlanWithPrefix(t *testing.T) {
	ctx := context.Background()
	layout := setupTestLayout(t)
	logger := events.NewLogger(layout)

	// Create a test session
	now := time.Date(2026, 6, 11, 10, 20, 30, 0, time.UTC)
	if err := logger.Log(ctx, events.Event{
		SessionID: "20260611T102030-abc123",
		Workspace: "workspace1",
		Agent:     "test-agent",
		Type:      "run_started",
		Timestamp: now,
	}); err != nil {
		t.Fatalf("failed to log event: %v", err)
	}

	var stdout, stderr bytes.Buffer
	var gotSessionID string

	deps := Dependencies{
		Layout: layout,
		GetSession: func(ctx context.Context, layout paths.Layout, sessionID string) (*sessions.Detail, error) {
			gotSessionID = sessionID
			return sessions.Get(ctx, layout, sessionID)
		},
	}

	code := NewApp(deps, &stdout, &stderr).Execute(
		context.Background(),
		[]string{"sessions", "resume-plan", "20260611"},
	)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0, stderr: %s", code, stderr.String())
	}
	if gotSessionID != "20260611T102030-abc123" {
		t.Fatalf("GetSession called with %q, want %q", gotSessionID, "20260611T102030-abc123")
	}
}

func TestSessionsAmbiguousPrefix(t *testing.T) {
	ctx := context.Background()
	layout := setupTestLayout(t)
	logger := events.NewLogger(layout)

	// Create multiple sessions with same prefix
	now := time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC)
	testSessions := []string{
		"20260611T102030-abc123",
		"20260612T143045-def456",
	}

	for i, sessionID := range testSessions {
		if err := logger.Log(ctx, events.Event{
			SessionID: sessionID,
			Workspace: "workspace1",
			Agent:     "test-agent",
			Type:      "run_started",
			Timestamp: now.Add(time.Duration(i) * time.Minute),
		}); err != nil {
			t.Fatalf("failed to log event: %v", err)
		}
	}

	var stdout, stderr bytes.Buffer

	deps := Dependencies{
		Layout: layout,
		GetSession: func(ctx context.Context, layout paths.Layout, sessionID string) (*sessions.Detail, error) {
			t.Fatal("GetSession should not be called for ambiguous prefix")
			return nil, nil
		},
	}

	code := NewApp(deps, &stdout, &stderr).Execute(
		context.Background(),
		[]string{"sessions", "show", "202606"},
	)

	if code == 0 {
		t.Fatalf("expected error, got success")
	}
	output := stderr.String()
	if !strings.Contains(output, "ambiguous session ID") {
		t.Fatalf("output %q does not contain ambiguous error", output)
	}
	if !strings.Contains(output, "20260611T102030-abc123") || !strings.Contains(output, "20260612T143045-def456") {
		t.Fatalf("output %q does not list matching sessions", output)
	}
}

func setupTestLayout(t *testing.T) paths.Layout {
	t.Helper()
	tmpDir := t.TempDir()
	return paths.New(tmpDir, tmpDir)
}
