package sessions

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/karoc/adp/internal/events"
	"github.com/karoc/adp/internal/paths"
)

func TestListGroupsFiltersLimitsAndOrdersSessions(t *testing.T) {
	t.Parallel()

	layout := paths.New(t.TempDir(), t.TempDir())
	logger := events.NewLogger(layout)
	exitCode := 0
	logEvents(t, logger, []events.Event{
		{
			Timestamp: time.Date(2026, 6, 8, 10, 0, 0, 0, time.FixedZone("CST", 8*60*60)),
			Type:      "ignored_without_session",
			Workspace: "game-a",
			Agent:     "codex",
		},
		{
			Timestamp:   time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
			Type:        "run_started",
			Workspace:   "game-a",
			Agent:       "codex",
			Profile:     "default",
			ProjectRoot: "/srv/game-a",
			RuntimePath: "/tmp/adp-runtime/s1",
			SessionID:   "s1",
			TaskID:      "task-1",
		},
		{
			Timestamp:      time.Date(2026, 6, 8, 10, 2, 0, 0, time.UTC),
			Type:           "run_finished",
			Workspace:      "game-a",
			Agent:          "codex",
			Profile:        "default",
			ProjectRoot:    "/srv/game-a",
			RuntimePath:    "/tmp/adp-runtime/s1",
			SessionID:      "s1",
			TaskID:         "task-1",
			ExitCode:       &exitCode,
			DurationMillis: 120000,
		},
		{
			Timestamp: time.Date(2026, 6, 8, 11, 0, 0, 0, time.UTC),
			Type:      "run_started",
			Workspace: "game-a",
			Agent:     "claude",
			SessionID: "s2",
		},
		{
			Timestamp: time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC),
			Type:      "run_started",
			Workspace: "game-a",
			Agent:     "codex",
			SessionID: "s3",
			TaskID:    "task-3",
		},
		{
			Timestamp: time.Date(2026, 6, 8, 13, 0, 0, 0, time.UTC),
			Type:      "run_started",
			Workspace: "game-b",
			Agent:     "codex",
			SessionID: "s4",
		},
	})

	summaries, err := List(context.Background(), layout, Query{
		Workspace: "game-a",
		Agent:     "codex",
	})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if got, want := summaryIDs(summaries), []string{"s1", "s3"}; !equalStrings(got, want) {
		t.Fatalf("session ids = %v, want %v", got, want)
	}

	first := summaries[0]
	if first.Workspace != "game-a" || first.Agent != "codex" || first.Profile != "default" {
		t.Fatalf("first summary identity fields are wrong: %#v", first)
	}
	if first.ProjectRoot != "/srv/game-a" || first.RuntimePath != "/tmp/adp-runtime/s1" {
		t.Fatalf("first summary paths are wrong: %#v", first)
	}
	if first.TaskID != "task-1" {
		t.Fatalf("first summary task id = %q, want task-1", first.TaskID)
	}
	if !first.StartedAt.Equal(time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC)) {
		t.Fatalf("started_at = %s, want 2026-06-08T10:00:00Z", first.StartedAt)
	}
	if !first.FinishedAt.Equal(time.Date(2026, 6, 8, 10, 2, 0, 0, time.UTC)) {
		t.Fatalf("finished_at = %s, want 2026-06-08T10:02:00Z", first.FinishedAt)
	}
	if first.ExitCode == nil || *first.ExitCode != 0 {
		t.Fatalf("exit_code = %v, want 0", first.ExitCode)
	}
	if first.DurationMillis == nil || *first.DurationMillis != 120000 {
		t.Fatalf("duration_ms = %v, want 120000", first.DurationMillis)
	}
	if first.EventCount != 2 {
		t.Fatalf("event_count = %d, want 2", first.EventCount)
	}

	limited, err := List(context.Background(), layout, Query{Agent: "codex", Limit: 2})
	if err != nil {
		t.Fatalf("limited List returned error: %v", err)
	}
	if got, want := summaryIDs(limited), []string{"s3", "s4"}; !equalStrings(got, want) {
		t.Fatalf("limited session ids = %v, want %v", got, want)
	}

	taskFiltered, err := List(context.Background(), layout, Query{TaskID: "task-3"})
	if err != nil {
		t.Fatalf("task-filtered List returned error: %v", err)
	}
	if got, want := summaryIDs(taskFiltered), []string{"s3"}; !equalStrings(got, want) {
		t.Fatalf("task-filtered session ids = %v, want %v", got, want)
	}
}

func TestGetReturnsSessionDetailWithOrderedEvents(t *testing.T) {
	t.Parallel()

	layout := paths.New(t.TempDir(), t.TempDir())
	logger := events.NewLogger(layout)
	exitCode := 7
	logEvents(t, logger, []events.Event{
		{
			Timestamp:   time.Date(2026, 6, 8, 9, 0, 0, 0, time.UTC),
			Type:        "run_started",
			Workspace:   "game-a",
			Agent:       "codex",
			Profile:     "safe",
			ProjectRoot: "/srv/game-a",
			RuntimePath: "/tmp/adp-runtime/s1",
			SessionID:   "s1",
			TaskID:      "task-1",
		},
		{
			Timestamp: time.Date(2026, 6, 8, 9, 1, 0, 0, time.UTC),
			Type:      "checkpoint",
			SessionID: "s1",
		},
		{
			Timestamp:      time.Date(2026, 6, 8, 9, 2, 0, 0, time.UTC),
			Type:           "run_finished",
			Workspace:      "game-a",
			Agent:          "codex",
			SessionID:      "s1",
			TaskID:         "task-1",
			ExitCode:       &exitCode,
			DurationMillis: 0,
		},
		{
			Timestamp: time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
			Type:      "run_started",
			SessionID: "s2",
		},
	})

	detail, err := Get(context.Background(), layout, "s1")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if detail.Summary.SessionID != "s1" || detail.Summary.EventCount != 3 {
		t.Fatalf("summary = %#v, want s1 with 3 events", detail.Summary)
	}
	if detail.Summary.TaskID != "task-1" {
		t.Fatalf("summary task id = %q, want task-1", detail.Summary.TaskID)
	}
	if detail.Summary.DurationMillis == nil || *detail.Summary.DurationMillis != 0 {
		t.Fatalf("duration_ms = %v, want present zero", detail.Summary.DurationMillis)
	}
	if got, want := eventTypes(detail.Events), []string{"run_started", "checkpoint", "run_finished"}; !equalStrings(got, want) {
		t.Fatalf("event types = %v, want %v", got, want)
	}
	for i, event := range detail.Events {
		if !event.Timestamp.IsZero() && event.Timestamp.Location() != time.UTC {
			t.Fatalf("event %d timestamp location = %s, want UTC", i, event.Timestamp.Location())
		}
	}
}

func TestGetReturnsErrNotFound(t *testing.T) {
	t.Parallel()

	layout := paths.New(t.TempDir(), t.TempDir())
	logger := events.NewLogger(layout)
	logEvents(t, logger, []events.Event{{
		Timestamp: time.Date(2026, 6, 8, 9, 0, 0, 0, time.UTC),
		Type:      "run_started",
		SessionID: "s1",
	}})

	_, err := Get(context.Background(), layout, "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}

	_, err = Get(context.Background(), layout, "")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("empty session error = %v, want ErrNotFound", err)
	}
}

func TestListRejectsNegativeLimit(t *testing.T) {
	t.Parallel()

	_, err := List(context.Background(), paths.New(t.TempDir(), t.TempDir()), Query{Limit: -1})
	if !errors.Is(err, ErrInvalidLimit) {
		t.Fatalf("error = %v, want ErrInvalidLimit", err)
	}
}

func TestListAndGetHonorCanceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	layout := paths.New(t.TempDir(), t.TempDir())

	_, err := List(ctx, layout, Query{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("List error = %v, want context.Canceled", err)
	}

	_, err = Get(ctx, layout, "s1")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Get error = %v, want context.Canceled", err)
	}
}

func TestListFallsBackToFirstTimestampWhenRunStartedIsMissing(t *testing.T) {
	t.Parallel()

	layout := paths.New(t.TempDir(), t.TempDir())
	logger := events.NewLogger(layout)
	logEvents(t, logger, []events.Event{{
		Timestamp: time.Date(2026, 6, 8, 9, 2, 0, 0, time.UTC),
		Type:      "run_finished",
		SessionID: "s1",
	}})

	summaries, err := List(context.Background(), layout, Query{})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("summary count = %d, want 1", len(summaries))
	}
	if !summaries[0].StartedAt.Equal(time.Date(2026, 6, 8, 9, 2, 0, 0, time.UTC)) {
		t.Fatalf("started_at = %s, want first event timestamp", summaries[0].StartedAt)
	}
}

func logEvents(t *testing.T, logger *events.Logger, entries []events.Event) {
	t.Helper()

	for _, event := range entries {
		if err := logger.Log(context.Background(), event); err != nil {
			t.Fatalf("log event %q/%q: %v", event.Type, event.SessionID, err)
		}
	}
}

func summaryIDs(summaries []Summary) []string {
	ids := make([]string, 0, len(summaries))
	for _, summary := range summaries {
		ids = append(ids, summary.SessionID)
	}
	return ids
}

func eventTypes(entries []events.Event) []string {
	types := make([]string, 0, len(entries))
	for _, event := range entries {
		types = append(types, event.Type)
	}
	return types
}

func equalStrings(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
