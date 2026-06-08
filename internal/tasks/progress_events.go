package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

type progressEvent struct {
	Timestamp      time.Time `json:"ts"`
	Type           string    `json:"type"`
	TaskID         string    `json:"task_id,omitempty"`
	PhaseID        string    `json:"phase_id,omitempty"`
	Status         string    `json:"status,omitempty"`
	Owner          string    `json:"owner,omitempty"`
	LeaseExpiresAt time.Time `json:"lease_expires_at,omitempty"`
	Reason         string    `json:"reason,omitempty"`
	Title          string    `json:"title,omitempty"`
	Result         string    `json:"result,omitempty"`
	Commands       []string  `json:"commands,omitempty"`
	Commit         string    `json:"commit,omitempty"`
	Remote         string    `json:"remote,omitempty"`
	Branch         string    `json:"branch,omitempty"`
	Message        string    `json:"message,omitempty"`
	Notes          string    `json:"notes,omitempty"`
}

func (s *Store) appendEvent(ctx context.Context, event progressEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.MkdirAll(s.planningPath(), 0o755); err != nil {
		return fmt.Errorf("create planning directory: %w", err)
	}
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode progress event: %w", err)
	}
	data = append(data, '\n')

	file, err := os.OpenFile(s.progressPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open progress file: %w", err)
	}
	defer file.Close()
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("write progress event: %w", err)
	}
	return nil
}

func (s *Store) appendPhaseEvent(ctx context.Context, now time.Time, eventType string, phase Phase, extra progressEvent) error {
	extra.Timestamp = now
	extra.Type = eventType
	extra.PhaseID = phase.ID
	extra.Status = string(phase.Status)
	return s.appendEvent(ctx, extra)
}

func planImportEvents(result PlanImportResult) []progressEvent {
	events := make([]progressEvent, 0, len(result.Phases)+len(result.Tasks))
	for _, phase := range result.Phases {
		events = append(events, progressEvent{
			Timestamp: phase.CreatedAt,
			Type:      "phase_created",
			PhaseID:   phase.ID,
			Status:    string(phase.Status),
			Title:     phase.Title,
		})
	}
	for _, task := range result.Tasks {
		events = append(events, progressEvent{
			Timestamp: task.CreatedAt,
			Type:      "task_created",
			TaskID:    task.ID,
			Status:    string(task.Status),
			Title:     task.Title,
		})
	}
	return events
}

func (s *Store) planImportProgressData(ctx context.Context, events []progressEvent) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	base, err := os.ReadFile(s.progressPath())
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read progress file: %w", err)
	}
	if len(base) > 0 && base[len(base)-1] != '\n' {
		base = append(base, '\n')
	}
	for _, event := range events {
		data, err := json.Marshal(event)
		if err != nil {
			return nil, fmt.Errorf("encode progress event: %w", err)
		}
		base = append(base, data...)
		base = append(base, '\n')
	}
	return base, nil
}
