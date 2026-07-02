package events

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/karoc/adp/internal/filelock"
	"github.com/karoc/adp/internal/paths"
)

var ErrEventsFileRequired = errors.New("events file path is required")

const (
	// eventsLockFile serializes appends to the events log across processes.
	// Multiple `adp run` invocations can append concurrently; a single append
	// can exceed PIPE_BUF (typically 4096 bytes) once it carries recorded
	// command context, and O_APPEND only guarantees atomicity below that
	// threshold, so an unlocked concurrent write could interleave and corrupt a
	// JSONL line.
	eventsLockFile = ".events.lock"
	// eventsLockStaleAge lets a waiter reclaim a lock left by a crashed holder.
	// The append critical section is sub-millisecond, so a minute is orders of
	// magnitude longer than any legitimate hold — long enough never to break a
	// live holder, short enough to recover quickly from a crash.
	eventsLockStaleAge = time.Minute
)

type Event struct {
	Timestamp      time.Time      `json:"ts"`
	Type           string         `json:"type"`
	Workspace      string         `json:"workspace,omitempty"`
	Agent          string         `json:"agent,omitempty"`
	Profile        string         `json:"profile,omitempty"`
	RuntimePath    string         `json:"runtime_path,omitempty"`
	ProjectRoot    string         `json:"project_root,omitempty"`
	SessionID      string         `json:"session_id,omitempty"`
	TaskID         string         `json:"task_id,omitempty"`
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
	if err := paths.EnsurePrivateDir(logsDir); err != nil {
		return err
	}

	// Serialize the append across processes. Concurrent `adp run` invocations
	// each append to the same log, and a single event line can exceed the
	// PIPE_BUF atomicity threshold once it carries command context, so an
	// unlocked O_APPEND could interleave and corrupt a JSONL line.
	locker := filelock.Locker{
		Path:     filepath.Join(logsDir, eventsLockFile),
		StaleAge: eventsLockStaleAge,
	}
	return locker.With(ctx, func() error {
		return appendEvent(eventsFile, event)
	})
}

func appendEvent(eventsFile string, event Event) error {
	file, err := os.OpenFile(eventsFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()

	// OpenFile only applies the mode when it creates the file. Event logs may
	// carry recorded command context, so tighten the permission of a log that
	// predates this guard (or was created with a looser umask) to owner-only.
	if info, statErr := file.Stat(); statErr == nil && info.Mode().Perm() != 0o600 {
		_ = file.Chmod(0o600)
	}

	encoder := json.NewEncoder(file)
	return encoder.Encode(sanitizeEvent(event))
}

func (l *Logger) eventsFile() string {
	return resolveEventsFile(l.Layout)
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
