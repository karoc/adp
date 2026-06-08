package tasks

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

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
	var phase Phase
	err := s.withPlanningLock(ctx, func() error {
		data, err := s.loadPhases(ctx)
		if err != nil {
			return err
		}
		if _, ok := findPhase(data.Phases, id); ok {
			return fmt.Errorf("phase already exists: %s", id)
		}
		now := s.now()
		phase = Phase{
			ID:        id,
			Title:     title,
			Status:    PhaseStatusPlanned,
			Goal:      strings.TrimSpace(req.Goal),
			CreatedAt: now,
			UpdatedAt: now,
		}
		data.Phases = append(data.Phases, phase)
		if err := s.savePhases(ctx, data); err != nil {
			return err
		}
		return s.appendPhaseEvent(ctx, now, "phase_created", phase, progressEvent{Title: phase.Title})
	})
	if err != nil {
		return Phase{}, err
	}
	return phase, nil
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
	if err := ctx.Err(); err != nil {
		return Phase{}, err
	}
	id = strings.TrimSpace(id)
	var phase Phase
	err := s.withPlanningLock(ctx, func() error {
		data, err := s.loadPhases(ctx)
		if err != nil {
			return err
		}
		now := s.now()
		for i := range data.Phases {
			if data.Phases[i].ID != id {
				continue
			}
			if data.Phases[i].Status != PhaseStatusPlanned && data.Phases[i].Status != PhaseStatusActive {
				return fmt.Errorf("%w: phase %s with status %s cannot be started", ErrPhaseInvalidTransition, id, data.Phases[i].Status)
			}
			if blocker, ok := openPhase(data.Phases, id); ok {
				return fmt.Errorf("%w: phase %s must be pushed before phase %s can start", ErrPhaseInvalidTransition, blocker.ID, id)
			}
			data.Phases[i].Status = PhaseStatusActive
			data.Phases[i].UpdatedAt = now
			phase = data.Phases[i]
			if err := s.savePhases(ctx, data); err != nil {
				return err
			}
			return s.appendPhaseEvent(ctx, now, "phase_started", phase, progressEvent{})
		}
		return fmt.Errorf("%w: %s", ErrPhaseNotFound, id)
	})
	if err != nil {
		return Phase{}, err
	}
	return phase, nil
}

func (s *Store) AcceptPhase(ctx context.Context, req PhaseAcceptRequest) (Phase, error) {
	result := strings.TrimSpace(req.Result)
	if result == "" {
		result = "passed"
	}
	if result != "passed" && result != "failed" {
		return Phase{}, errors.New("acceptance result must be passed or failed")
	}
	commands := cleanStrings(req.Commands)
	return s.updatePhase(ctx, req.ID, func(phase *Phase, now time.Time) (progressEvent, error) {
		if phase.Status == PhaseStatusCommitted || phase.Status == PhaseStatusPushed {
			return progressEvent{}, fmt.Errorf("%w: phase %s with status %s cannot record new acceptance evidence", ErrPhaseInvalidTransition, phase.ID, phase.Status)
		}
		if result == "passed" {
			phase.Status = PhaseStatusAccepted
		} else {
			phase.Status = PhaseStatusActive
		}
		phase.Acceptance = AcceptanceRecord{
			Commands: commands,
			Result:   result,
			Notes:    strings.TrimSpace(req.Notes),
			At:       now,
		}
		return progressEvent{Result: result, Commands: commands, Notes: phase.Acceptance.Notes}, nil
	}, "phase_accepted")
}

func (s *Store) RecordPhaseCommit(ctx context.Context, req PhaseCommitRequest) (Phase, error) {
	hash := strings.TrimSpace(req.Hash)
	if hash == "" {
		return Phase{}, errors.New("commit hash is required")
	}
	return s.updatePhase(ctx, req.ID, func(phase *Phase, now time.Time) (progressEvent, error) {
		if phase.Status != PhaseStatusAccepted && phase.Status != PhaseStatusCommitted {
			return progressEvent{}, fmt.Errorf("%w: phase %s must be accepted before commit evidence is recorded", ErrPhaseInvalidTransition, phase.ID)
		}
		if phase.Acceptance.Result != "passed" {
			return progressEvent{}, fmt.Errorf("%w: phase %s acceptance result is %q, want passed", ErrPhaseInvalidTransition, phase.ID, phase.Acceptance.Result)
		}
		phase.Status = PhaseStatusCommitted
		phase.Commit = CommitRecord{
			Hash:    hash,
			Message: strings.TrimSpace(req.Message),
			At:      now,
		}
		return progressEvent{Commit: hash, Message: phase.Commit.Message}, nil
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
	if result != "pushed" && result != "failed" {
		return Phase{}, errors.New("push result must be pushed or failed")
	}
	return s.updatePhase(ctx, req.ID, func(phase *Phase, now time.Time) (progressEvent, error) {
		if phase.Status != PhaseStatusCommitted && phase.Status != PhaseStatusPushed {
			return progressEvent{}, fmt.Errorf("%w: phase %s must have commit evidence before push evidence is recorded", ErrPhaseInvalidTransition, phase.ID)
		}
		if strings.TrimSpace(phase.Commit.Hash) == "" {
			return progressEvent{}, fmt.Errorf("%w: phase %s commit hash is required before push evidence", ErrPhaseInvalidTransition, phase.ID)
		}
		if pushSucceeded(result) {
			phase.Status = PhaseStatusPushed
		} else if phase.Status != PhaseStatusPushed {
			phase.Status = PhaseStatusCommitted
		}
		phase.Push = PushRecord{
			Remote: remote,
			Branch: branch,
			Result: result,
			At:     now,
		}
		return progressEvent{Remote: remote, Branch: branch, Result: result}, nil
	}, "phase_pushed")
}

func (s *Store) updatePhase(ctx context.Context, id string, mutate func(*Phase, time.Time) (progressEvent, error), eventType string) (Phase, error) {
	if err := ctx.Err(); err != nil {
		return Phase{}, err
	}
	var phase Phase
	err := s.withPlanningLock(ctx, func() error {
		data, err := s.loadPhases(ctx)
		if err != nil {
			return err
		}
		id = strings.TrimSpace(id)
		now := s.now()
		for i := range data.Phases {
			if data.Phases[i].ID != id {
				continue
			}
			event, err := mutate(&data.Phases[i], now)
			if err != nil {
				return err
			}
			data.Phases[i].UpdatedAt = now
			phase = data.Phases[i]
			if err := s.savePhases(ctx, data); err != nil {
				return err
			}
			return s.appendPhaseEvent(ctx, now, eventType, phase, event)
		}
		return fmt.Errorf("%w: %s", ErrPhaseNotFound, id)
	})
	if err != nil {
		return Phase{}, err
	}
	return phase, nil
}
