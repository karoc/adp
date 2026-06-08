package events

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/karoc/adp/internal/paths"
)

func TestLoggerAppendsJSONLinesAndSanitizesEnvFields(t *testing.T) {
	t.Parallel()

	layout := paths.New(t.TempDir(), t.TempDir())
	logger := NewLogger(layout)
	ts := time.Date(2026, 6, 8, 12, 1, 2, 0, time.UTC)
	exitCode := 12

	err := logger.Log(context.Background(), Event{
		Timestamp:   ts,
		Type:        "run_finished",
		Workspace:   "game-a",
		Agent:       "codex",
		RuntimePath: "/tmp/adp-runtime/game-a-session",
		ProjectRoot: "/srv/game-a",
		SessionID:   "session-1",
		ExitCode:    &exitCode,
		Fields: map[string]any{
			"phase": "done",
			"env": map[string]string{
				"SECRET_TOKEN": "must-not-be-written",
			},
		},
	})
	if err != nil {
		t.Fatalf("Log returned error: %v", err)
	}

	err = logger.Log(context.Background(), Event{
		Timestamp: ts.Add(time.Second),
		Type:      "run_started",
		Workspace: "game-a",
	})
	if err != nil {
		t.Fatalf("second Log returned error: %v", err)
	}

	data, err := os.ReadFile(layout.EventsFile)
	if err != nil {
		t.Fatalf("read events file: %v", err)
	}

	if strings.Contains(string(data), "SECRET_TOKEN") {
		t.Fatalf("event log contains sanitized env data:\n%s", string(data))
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("line count = %d, want 2; data:\n%s", len(lines), string(data))
	}
	for _, line := range lines {
		if strings.TrimSpace(line) != line || strings.Contains(line, "\n") {
			t.Fatalf("event is not a single JSON line: %q", line)
		}
	}

	first := map[string]any{}
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("first line is not JSON: %v", err)
	}
	if first["ts"] != "2026-06-08T12:01:02Z" {
		t.Fatalf("ts = %v, want RFC3339 UTC timestamp", first["ts"])
	}
	if first["type"] != "run_finished" {
		t.Fatalf("type = %v, want run_finished", first["type"])
	}
	if first["exit_code"] != float64(exitCode) {
		t.Fatalf("exit_code = %v, want %d", first["exit_code"], exitCode)
	}

	fields, ok := first["fields"].(map[string]any)
	if !ok {
		t.Fatalf("fields missing or wrong type: %#v", first["fields"])
	}
	if fields["phase"] != "done" {
		t.Fatalf("fields.phase = %v, want done", fields["phase"])
	}
	if _, ok := fields["env"]; ok {
		t.Fatalf("fields.env should have been sanitized: %#v", fields)
	}

	if info, err := os.Stat(layout.LogsDir); err != nil || !info.IsDir() {
		t.Fatalf("logs dir was not created: info=%#v err=%v", info, err)
	}
}
