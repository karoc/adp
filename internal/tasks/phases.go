package tasks

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type PhaseStatus string

const (
	PhaseStatusPlanned   PhaseStatus = "planned"
	PhaseStatusActive    PhaseStatus = "active"
	PhaseStatusAccepted  PhaseStatus = "accepted"
	PhaseStatusCommitted PhaseStatus = "committed"
	PhaseStatusPushed    PhaseStatus = "pushed"
)

var (
	ErrPhaseNotFound = errors.New("phase not found")
	phaseStatuses    = []PhaseStatus{PhaseStatusPlanned, PhaseStatusActive, PhaseStatusAccepted, PhaseStatusCommitted, PhaseStatusPushed}
)

type Phase struct {
	ID         string            `yaml:"id"`
	Title      string            `yaml:"title"`
	Status     PhaseStatus       `yaml:"status"`
	Goal       string            `yaml:"goal,omitempty"`
	Acceptance AcceptanceRecord  `yaml:"acceptance,omitempty"`
	Commit     CommitRecord      `yaml:"commit,omitempty"`
	Push       PushRecord        `yaml:"push,omitempty"`
	CreatedAt  time.Time         `yaml:"created_at"`
	UpdatedAt  time.Time         `yaml:"updated_at"`
	Metadata   map[string]string `yaml:"metadata,omitempty"`
}

type AcceptanceRecord struct {
	Commands []string  `yaml:"commands,omitempty"`
	Result   string    `yaml:"result,omitempty"`
	Notes    string    `yaml:"notes,omitempty"`
	At       time.Time `yaml:"at,omitempty"`
}

type CommitRecord struct {
	Hash    string    `yaml:"hash,omitempty"`
	Message string    `yaml:"message,omitempty"`
	At      time.Time `yaml:"at,omitempty"`
}

type PushRecord struct {
	Remote string    `yaml:"remote,omitempty"`
	Branch string    `yaml:"branch,omitempty"`
	Result string    `yaml:"result,omitempty"`
	At     time.Time `yaml:"at,omitempty"`
}

type PhaseAddRequest struct {
	ID    string
	Title string
	Goal  string
}

type PhaseAcceptRequest struct {
	ID       string
	Commands []string
	Result   string
	Notes    string
}

type PhaseCommitRequest struct {
	ID      string
	Hash    string
	Message string
}

type PhasePushRequest struct {
	ID     string
	Remote string
	Branch string
	Result string
}

type phaseFile struct {
	Version int     `yaml:"version"`
	Phases  []Phase `yaml:"phases"`
}

func PhaseStatuses() []PhaseStatus {
	statuses := make([]PhaseStatus, len(phaseStatuses))
	copy(statuses, phaseStatuses)
	return statuses
}

func (s *Store) AddPhase(ctx context.Context, req PhaseAddRequest) (Phase, error) {
	if err := ctx.Err(); err != nil {
		return Phase{}, err
	}
	id := strings.TrimSpace(req.ID)
	title := strings.TrimSpace(req.Title)
	if id == "" {
		return Phase{}, errors.New("phase id is required")
	}
	if title == "" {
		return Phase{}, errors.New("phase title is required")
	}
	data, err := s.loadPhases(ctx)
	if err != nil {
		return Phase{}, err
	}
	if _, ok := findPhase(data.Phases, id); ok {
		return Phase{}, fmt.Errorf("phase already exists: %s", id)
	}
	now := s.now()
	phase := Phase{
		ID:        id,
		Title:     title,
		Status:    PhaseStatusPlanned,
		Goal:      strings.TrimSpace(req.Goal),
		CreatedAt: now,
		UpdatedAt: now,
	}
	data.Phases = append(data.Phases, phase)
	if err := s.savePhases(ctx, data); err != nil {
		return Phase{}, err
	}
	return phase, s.appendPhaseEvent(ctx, now, "phase_created", phase, progressEvent{Title: phase.Title})
}

func (s *Store) ListPhases(ctx context.Context) ([]Phase, error) {
	data, err := s.loadPhases(ctx)
	if err != nil {
		return nil, err
	}
	phases := append([]Phase(nil), data.Phases...)
	sortPhases(phases)
	return phases, nil
}

func (s *Store) GetPhase(ctx context.Context, id string) (Phase, error) {
	data, err := s.loadPhases(ctx)
	if err != nil {
		return Phase{}, err
	}
	phase, ok := findPhase(data.Phases, strings.TrimSpace(id))
	if !ok {
		return Phase{}, fmt.Errorf("%w: %s", ErrPhaseNotFound, id)
	}
	return phase, nil
}

func (s *Store) StartPhase(ctx context.Context, id string) (Phase, error) {
	return s.updatePhase(ctx, id, func(phase *Phase, now time.Time) progressEvent {
		phase.Status = PhaseStatusActive
		return progressEvent{}
	}, "phase_started")
}

func (s *Store) AcceptPhase(ctx context.Context, req PhaseAcceptRequest) (Phase, error) {
	result := strings.TrimSpace(req.Result)
	if result == "" {
		result = "passed"
	}
	commands := cleanStrings(req.Commands)
	return s.updatePhase(ctx, req.ID, func(phase *Phase, now time.Time) progressEvent {
		phase.Status = PhaseStatusAccepted
		phase.Acceptance = AcceptanceRecord{
			Commands: commands,
			Result:   result,
			Notes:    strings.TrimSpace(req.Notes),
			At:       now,
		}
		return progressEvent{Result: result, Commands: commands, Notes: phase.Acceptance.Notes}
	}, "phase_accepted")
}

func (s *Store) RecordPhaseCommit(ctx context.Context, req PhaseCommitRequest) (Phase, error) {
	hash := strings.TrimSpace(req.Hash)
	if hash == "" {
		return Phase{}, errors.New("commit hash is required")
	}
	return s.updatePhase(ctx, req.ID, func(phase *Phase, now time.Time) progressEvent {
		phase.Status = PhaseStatusCommitted
		phase.Commit = CommitRecord{
			Hash:    hash,
			Message: strings.TrimSpace(req.Message),
			At:      now,
		}
		return progressEvent{Commit: hash, Message: phase.Commit.Message}
	}, "phase_committed")
}

func (s *Store) RecordPhasePush(ctx context.Context, req PhasePushRequest) (Phase, error) {
	remote := strings.TrimSpace(req.Remote)
	branch := strings.TrimSpace(req.Branch)
	if remote == "" {
		return Phase{}, errors.New("push remote is required")
	}
	if branch == "" {
		return Phase{}, errors.New("push branch is required")
	}
	result := strings.TrimSpace(req.Result)
	if result == "" {
		result = "pushed"
	}
	return s.updatePhase(ctx, req.ID, func(phase *Phase, now time.Time) progressEvent {
		phase.Status = PhaseStatusPushed
		phase.Push = PushRecord{
			Remote: remote,
			Branch: branch,
			Result: result,
			At:     now,
		}
		return progressEvent{Remote: remote, Branch: branch, Result: result}
	}, "phase_pushed")
}

func (s *Store) updatePhase(ctx context.Context, id string, mutate func(*Phase, time.Time) progressEvent, eventType string) (Phase, error) {
	if err := ctx.Err(); err != nil {
		return Phase{}, err
	}
	data, err := s.loadPhases(ctx)
	if err != nil {
		return Phase{}, err
	}
	id = strings.TrimSpace(id)
	now := s.now()
	for i := range data.Phases {
		if data.Phases[i].ID != id {
			continue
		}
		event := mutate(&data.Phases[i], now)
		data.Phases[i].UpdatedAt = now
		phase := data.Phases[i]
		if err := s.savePhases(ctx, data); err != nil {
			return Phase{}, err
		}
		return phase, s.appendPhaseEvent(ctx, now, eventType, phase, event)
	}
	return Phase{}, fmt.Errorf("%w: %s", ErrPhaseNotFound, id)
}

func (s *Store) loadPhases(ctx context.Context) (phaseFile, error) {
	if err := ctx.Err(); err != nil {
		return phaseFile{}, err
	}
	path := s.phasesPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return phaseFile{Version: currentVersion}, nil
		}
		return phaseFile{}, fmt.Errorf("read phase file %s: %w", path, err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return phaseFile{Version: currentVersion}, nil
	}
	var file phaseFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return phaseFile{}, fmt.Errorf("parse phase file %s: %w", path, err)
	}
	if file.Version == 0 {
		file.Version = currentVersion
	}
	if file.Version != currentVersion {
		return phaseFile{}, fmt.Errorf("unsupported phase file version %d", file.Version)
	}
	for _, phase := range file.Phases {
		if !isValidPhaseStatus(phase.Status) {
			return phaseFile{}, fmt.Errorf("phase %s has unknown status %q", phase.ID, phase.Status)
		}
	}
	return file, nil
}

func (s *Store) savePhases(ctx context.Context, file phaseFile) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	file.Version = currentVersion
	sortPhases(file.Phases)
	if err := os.MkdirAll(s.planningPath(), 0o755); err != nil {
		return fmt.Errorf("create planning directory: %w", err)
	}
	data, err := yaml.Marshal(file)
	if err != nil {
		return fmt.Errorf("encode phase file: %w", err)
	}
	path := s.phasesPath()
	tmpPath := path + ".tmp"
	defer os.Remove(tmpPath)
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write temporary phase file %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace phase file %s: %w", path, err)
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

func findPhase(phases []Phase, id string) (Phase, bool) {
	for _, phase := range phases {
		if phase.ID == id {
			return phase, true
		}
	}
	return Phase{}, false
}

func isValidPhaseStatus(status PhaseStatus) bool {
	for _, known := range phaseStatuses {
		if status == known {
			return true
		}
	}
	return false
}

func sortPhases(phases []Phase) {
	sort.SliceStable(phases, func(i, j int) bool {
		if phases[i].CreatedAt.Equal(phases[j].CreatedAt) {
			return phases[i].ID < phases[j].ID
		}
		return phases[i].CreatedAt.Before(phases[j].CreatedAt)
	})
}

func cleanStrings(values []string) []string {
	cleaned := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			cleaned = append(cleaned, value)
		}
	}
	return cleaned
}
