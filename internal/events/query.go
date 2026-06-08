package events

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/karoc/adp/internal/paths"
)

const eventLogFileName = "events.jsonl"

// ErrInvalidLimit reports a query that cannot be bounded predictably.
var ErrInvalidLimit = errors.New("event query limit must be non-negative")

// Query filters events read from the JSONL event log.
// Limit 0 returns every matching event; positive limits keep the latest matches.
type Query struct {
	Workspace string
	SessionID string
	TaskID    string
	Type      string
	Limit     int
}

// Read returns matching events from layout's event log.
func Read(ctx context.Context, layout paths.Layout, query Query) ([]Event, error) {
	if err := validateQuery(query); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	eventsFile := resolveEventsFile(layout)
	if eventsFile == "" {
		return nil, ErrEventsFileRequired
	}

	file, err := os.Open(eventsFile)
	if errors.Is(err, os.ErrNotExist) {
		return []Event{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return readEvents(ctx, file, query)
}

func resolveEventsFile(layout paths.Layout) string {
	if layout.EventsFile != "" {
		return layout.EventsFile
	}
	if layout.LogsDir == "" {
		return ""
	}
	return filepath.Join(layout.LogsDir, eventLogFileName)
}

func validateQuery(query Query) error {
	if query.Limit < 0 {
		return ErrInvalidLimit
	}
	return nil
}

func readEvents(ctx context.Context, reader io.Reader, query Query) ([]Event, error) {
	buffered := bufio.NewReader(reader)
	events := []Event{}
	lineNumber := 0

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		line, err := buffered.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		if len(line) > 0 {
			lineNumber++
			event, decodeErr := decodeEventLine(line, lineNumber)
			if decodeErr != nil {
				return nil, decodeErr
			}
			if matchesQuery(event, query) {
				events = appendLimited(events, event, query.Limit)
			}
		}

		if errors.Is(err, io.EOF) {
			return events, nil
		}
	}
}

func decodeEventLine(line string, lineNumber int) (Event, error) {
	var event Event
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		return Event{}, fmt.Errorf("decode event log line %d: %w", lineNumber, err)
	}
	if !event.Timestamp.IsZero() {
		event.Timestamp = event.Timestamp.UTC()
	}
	return event, nil
}

func matchesQuery(event Event, query Query) bool {
	if query.Workspace != "" && event.Workspace != query.Workspace {
		return false
	}
	if query.SessionID != "" && event.SessionID != query.SessionID {
		return false
	}
	if query.TaskID != "" && event.TaskID != query.TaskID {
		return false
	}
	if query.Type != "" && event.Type != query.Type {
		return false
	}
	return true
}

func appendLimited(events []Event, event Event, limit int) []Event {
	if limit == 0 || len(events) < limit {
		return append(events, event)
	}

	copy(events, events[1:])
	events[len(events)-1] = event
	return events
}
