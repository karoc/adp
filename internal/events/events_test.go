package events

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"
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
		TaskID:      "task-20260608-0001",
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
	if first["task_id"] != "task-20260608-0001" {
		t.Fatalf("task_id = %v, want task-20260608-0001", first["task_id"])
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

func TestLoggerCreatesEventsFileWithOwnerOnlyPermission(t *testing.T) {
	t.Parallel()

	layout := paths.New(t.TempDir(), t.TempDir())
	logger := NewLogger(layout)

	if err := logger.Log(context.Background(), Event{Type: "run_started", Workspace: "game-a"}); err != nil {
		t.Fatalf("Log returned error: %v", err)
	}

	info, err := os.Stat(layout.EventsFile)
	if err != nil {
		t.Fatalf("stat events file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("events file permission = %o, want 600", perm)
	}
}

func TestLoggerTightensPreexistingLoosePermission(t *testing.T) {
	t.Parallel()

	layout := paths.New(t.TempDir(), t.TempDir())
	if err := os.MkdirAll(layout.LogsDir, 0o755); err != nil {
		t.Fatalf("mkdir logs dir: %v", err)
	}
	// Simulate a log written by an older build with a world-readable mode.
	if err := os.WriteFile(layout.EventsFile, []byte("{\"type\":\"old\"}\n"), 0o644); err != nil {
		t.Fatalf("seed legacy log: %v", err)
	}

	logger := NewLogger(layout)
	if err := logger.Log(context.Background(), Event{Type: "run_started", Workspace: "game-a"}); err != nil {
		t.Fatalf("Log returned error: %v", err)
	}

	info, err := os.Stat(layout.EventsFile)
	if err != nil {
		t.Fatalf("stat events file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("legacy events file permission = %o, want 600 after tightening", perm)
	}
}

// TestLoggerConcurrentAppendsProduceIntactJSONLines drives many goroutines
// appending to one log at once. Run under `go test -race`, it guards the
// file-lock serialization in Log: without it, concurrent O_APPEND writes of a
// line that exceeds PIPE_BUF could interleave and leave a half-written record,
// so every line must still parse as JSON and every event must be present
// exactly once.
func TestLoggerConcurrentAppendsProduceIntactJSONLines(t *testing.T) {
	t.Parallel()

	layout := paths.New(t.TempDir(), t.TempDir())
	logger := NewLogger(layout)

	const writers = 16
	// Pad each event past the PIPE_BUF atomicity threshold (~4096 bytes) so an
	// unlocked append would be free to interleave mid-line.
	padding := strings.Repeat("x", 5000)

	var wg sync.WaitGroup
	errCh := make(chan error, writers)
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			err := logger.Log(context.Background(), Event{
				Type:      "run_started",
				Workspace: "game-a",
				SessionID: "session-" + string(rune('a'+i)),
				Fields:    map[string]any{"pad": padding, "seq": i},
			})
			if err != nil {
				errCh <- err
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatalf("concurrent Log returned error: %v", err)
	}

	data, err := os.ReadFile(layout.EventsFile)
	if err != nil {
		t.Fatalf("read events file: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != writers {
		t.Fatalf("line count = %d, want %d (interleaved appends corrupt the log)", len(lines), writers)
	}

	seen := map[float64]bool{}
	for _, line := range lines {
		var event struct {
			Type   string         `json:"type"`
			Fields map[string]any `json:"fields"`
		}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("event line is not intact JSON: %v\nline: %q", err, line)
		}
		seq, ok := event.Fields["seq"].(float64)
		if !ok {
			t.Fatalf("event line missing seq field: %q", line)
		}
		if seen[seq] {
			t.Fatalf("event seq %v written more than once", seq)
		}
		seen[seq] = true
	}
	if len(seen) != writers {
		t.Fatalf("distinct events = %d, want %d", len(seen), writers)
	}
}
