package events

import (
	"context"
	"errors"
	"time"

	"github.com/karoc/adp/internal/paths"
)

var ErrNotImplemented = errors.New("events scaffold is not implemented")

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

func (l *Logger) Log(_ context.Context, _ Event) error {
	return ErrNotImplemented
}
