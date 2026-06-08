package events

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/karoc/adp/internal/paths"
)

func TestReadReturnsEmptyWhenEventsFileDoesNotExist(t *testing.T) {
	t.Parallel()

	events, err := Read(context.Background(), paths.New(t.TempDir(), t.TempDir()), Query{})
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("event count = %d, want 0", len(events))
	}
}

func TestReadFiltersLimitsAndNormalizesTimestamps(t *testing.T) {
	t.Parallel()

	layout := paths.New(t.TempDir(), t.TempDir())
	writeEventLog(t, layout, strings.Join([]string{
		`{"ts":"2026-06-08T10:00:00+08:00","type":"run_finished","workspace":"game-a","session_id":"s1","task_id":"task-other","runtime_path":"first"}`,
		`{"ts":"2026-06-08T10:30:00+08:00","type":"run_finished","workspace":"game-b","session_id":"s1","runtime_path":"other-workspace"}`,
		`{"ts":"2026-06-08T11:00:00+08:00","type":"run_finished","workspace":"game-a","session_id":"s1","task_id":"task-1","runtime_path":"second"}`,
		`{"ts":"2026-06-08T11:30:00+08:00","type":"run_started","workspace":"game-a","session_id":"s1","runtime_path":"other-type"}`,
		`{"ts":"2026-06-08T12:00:00+08:00","type":"run_finished","workspace":"game-a","session_id":"s1","task_id":"task-1","runtime_path":"third"}`,
		`{"ts":"2026-06-08T13:00:00+08:00","type":"run_finished","workspace":"game-a","session_id":"s1","task_id":"task-1","runtime_path":"fourth"}`,
	}, "\n"))

	events, err := Read(context.Background(), layout, Query{
		Workspace: "game-a",
		SessionID: "s1",
		TaskID:    "task-1",
		Type:      "run_finished",
		Limit:     2,
	})
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("event count = %d, want 2", len(events))
	}
	if events[0].RuntimePath != "third" || events[1].RuntimePath != "fourth" {
		t.Fatalf("runtime paths = [%q %q], want [third fourth]", events[0].RuntimePath, events[1].RuntimePath)
	}

	wantTimes := []time.Time{
		time.Date(2026, 6, 8, 4, 0, 0, 0, time.UTC),
		time.Date(2026, 6, 8, 5, 0, 0, 0, time.UTC),
	}
	for i, event := range events {
		if !event.Timestamp.Equal(wantTimes[i]) {
			t.Fatalf("event %d timestamp = %s, want %s", i, event.Timestamp, wantTimes[i])
		}
		if event.Timestamp.Location() != time.UTC {
			t.Fatalf("event %d timestamp location = %s, want UTC", i, event.Timestamp.Location())
		}
	}
}

func TestReadReportsInvalidJSONLineNumber(t *testing.T) {
	t.Parallel()

	layout := paths.New(t.TempDir(), t.TempDir())
	writeEventLog(t, layout, strings.Join([]string{
		`{"type":"run_started","workspace":"game-a"}`,
		`not-json`,
	}, "\n"))

	_, err := Read(context.Background(), layout, Query{})
	if err == nil {
		t.Fatal("Read returned nil error, want invalid JSON error")
	}
	if !strings.Contains(err.Error(), "line 2") {
		t.Fatalf("error = %q, want line number", err.Error())
	}
}

func TestReadRejectsNegativeLimit(t *testing.T) {
	t.Parallel()

	_, err := Read(context.Background(), paths.New(t.TempDir(), t.TempDir()), Query{Limit: -1})
	if !errors.Is(err, ErrInvalidLimit) {
		t.Fatalf("error = %v, want ErrInvalidLimit", err)
	}
}

func TestReadHonorsCanceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := Read(ctx, paths.New(t.TempDir(), t.TempDir()), Query{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestReadEventsHonorsContextCanceledWhileReading(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	reader := &cancelAtEOFReader{cancel: cancel}

	_, err := readEvents(ctx, reader, Query{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

type cancelAtEOFReader struct {
	cancel context.CancelFunc
	reads  int
}

func (r *cancelAtEOFReader) Read(p []byte) (int, error) {
	if r.reads == 0 {
		r.reads++
		return copy(p, []byte(`{"type":"run_started"}`+"\n")), nil
	}

	r.cancel()
	return 0, io.EOF
}

func writeEventLog(t *testing.T, layout paths.Layout, content string) {
	t.Helper()

	if err := os.MkdirAll(layout.LogsDir, 0o755); err != nil {
		t.Fatalf("create logs dir: %v", err)
	}
	if err := os.WriteFile(layout.EventsFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write events file: %v", err)
	}
}
