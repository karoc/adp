package events

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/karoc/adp/internal/paths"
)

var ErrEventsFileRequired = errors.New("events file path is required")

type Event struct {
	Timestamp      time.Time      `json:"ts"`
	Type           string         `json:"type"`
	Workspace      string         `json:"workspace,omitempty"`
	Agent          string         `json:"agent,omitempty"`
	Profile        string         `json:"profile,omitempty"`
	RuntimePath    string         `json:"runtime_path,omitempty"`
	ProjectRoot    string         `json:"project_root,omitempty"`
	SessionID      string         `json:"session_id,omitempty"`
	PID            int            `json:"pid,omitempty"`
	ExitCode       *int           `json:"exit_code,omitempty"`
	DurationMillis int64          `json:"duration_ms,omitempty"`
	Fields         map[string]any `json:"fields,omitempty"`
}

type Logger struct {
	Layout paths.Layout
}

func NewLogger(layout paths.Layout) *Logger {
	return &Logger{Layout: layout}
}

func (l *Logger) Log(ctx context.Context, event Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	eventsFile := l.eventsFile()
	if eventsFile == "" {
		return ErrEventsFileRequired
	}

	logsDir := l.logsDir(eventsFile)
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		return err
	}

	file, err := os.OpenFile(eventsFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	return encoder.Encode(sanitizeEvent(event))
}

func (l *Logger) eventsFile() string {
	if l.Layout.EventsFile != "" {
		return l.Layout.EventsFile
	}
	if l.Layout.LogsDir == "" {
		return ""
	}
	return filepath.Join(l.Layout.LogsDir, "events.jsonl")
}

func (l *Logger) logsDir(eventsFile string) string {
	if l.Layout.LogsDir != "" {
		return l.Layout.LogsDir
	}
	return filepath.Dir(eventsFile)
}

func sanitizeEvent(event Event) Event {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	} else {
		event.Timestamp = event.Timestamp.UTC()
	}
	event.Fields = sanitizeFields(event.Fields)
	return event
}

func sanitizeFields(fields map[string]any) map[string]any {
	if len(fields) == 0 {
		return nil
	}

	cleaned := make(map[string]any, len(fields))
	for key, value := range fields {
		if isEnvField(key) {
			continue
		}
		cleaned[key] = value
	}
	if len(cleaned) == 0 {
		return nil
	}
	return cleaned
}

func isEnvField(key string) bool {
	switch strings.ToLower(key) {
	case "env", "environ", "environment":
		return true
	default:
		return false
	}
}
