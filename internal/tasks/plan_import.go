package tasks

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

type PlanImportPhase struct {
	ID    string
	Title string
	Goal  string
}

type PlanImportTask struct {
	Title       string
	Description string
	Priority    string
	Phase       string
	Status      Status
}

type PlanImportRequest struct {
	Phases []PlanImportPhase
	Tasks  []PlanImportTask
}

type PlanImportResult struct {
	Phases []Phase
	Tasks  []Task
}

func (s *Store) PreviewPlanImport(ctx context.Context, req PlanImportRequest) (PlanImportResult, error) {
	if err := ctx.Err(); err != nil {
		return PlanImportResult{}, err
	}
	phaseData, err := s.loadPhases(ctx)
	if err != nil {
		return PlanImportResult{}, err
	}
	taskData, err := s.load(ctx)
	if err != nil {
		return PlanImportResult{}, err
	}
	return s.preparePlanImport(req, phaseData, taskData, s.now())
}

func (s *Store) ApplyPlanImport(ctx context.Context, req PlanImportRequest) (PlanImportResult, error) {
	if err := ctx.Err(); err != nil {
		return PlanImportResult{}, err
	}
	planningExisted, err := pathExists(s.planningPath())
	if err != nil {
		return PlanImportResult{}, err
	}
	var result PlanImportResult
	err = s.withPlanningLock(ctx, func() error {
		phaseData, err := s.loadPhases(ctx)
		if err != nil {
			return err
		}
		taskData, err := s.load(ctx)
		if err != nil {
			return err
		}
		result, err = s.preparePlanImport(req, phaseData, taskData, s.now())
		if err != nil {
			return err
		}
		if len(result.Phases) > 0 {
			phaseData.Phases = append(phaseData.Phases, result.Phases...)
		}
		if len(result.Tasks) > 0 {
			taskData.Tasks = append(taskData.Tasks, result.Tasks...)
		}
		return s.commitPlanImport(ctx, phaseData, taskData, result)
	})
	if err != nil {
		if !planningExisted {
			_ = os.Remove(s.planningPath())
		}
		return PlanImportResult{}, err
	}
	return result, nil
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, fmt.Errorf("inspect %s: %w", path, err)
}

func (s *Store) preparePlanImport(req PlanImportRequest, phaseData phaseFile, taskData taskFile, now time.Time) (PlanImportResult, error) {
	if len(req.Phases) == 0 && len(req.Tasks) == 0 {
		return PlanImportResult{}, errors.New("plan import requires at least one phase or task")
	}

	existingPhases := map[string]struct{}{}
	for _, phase := range phaseData.Phases {
		existingPhases[phase.ID] = struct{}{}
	}

	newPhases := make([]Phase, 0, len(req.Phases))
	newPhaseIDs := map[string]struct{}{}
	nextOrder := nextPhaseOrder(phaseData.Phases)
	for _, input := range req.Phases {
		id := strings.TrimSpace(input.ID)
		title := strings.TrimSpace(input.Title)
		if id == "" {
			return PlanImportResult{}, errors.New("phase id is required")
		}
		if title == "" {
			return PlanImportResult{}, errors.New("phase title is required")
		}
		if _, ok := existingPhases[id]; ok {
			return PlanImportResult{}, fmt.Errorf("phase already exists: %s", id)
		}
		if _, ok := newPhaseIDs[id]; ok {
			return PlanImportResult{}, fmt.Errorf("duplicate phase id: %s", id)
		}
		newPhaseIDs[id] = struct{}{}
		newPhases = append(newPhases, Phase{
			ID:        id,
			Title:     title,
			Status:    PhaseStatusPlanned,
			Order:     nextOrder,
			Goal:      strings.TrimSpace(input.Goal),
			CreatedAt: now,
			UpdatedAt: now,
		})
		nextOrder++
	}

	knownPhases := map[string]struct{}{}
	for id := range existingPhases {
		knownPhases[id] = struct{}{}
	}
	for id := range newPhaseIDs {
		knownPhases[id] = struct{}{}
	}

	nextTasks := append([]Task(nil), taskData.Tasks...)
	newTasks := make([]Task, 0, len(req.Tasks))
	for _, input := range req.Tasks {
		title := strings.TrimSpace(input.Title)
		if title == "" {
			return PlanImportResult{}, errors.New("task title is required")
		}
		status := input.Status
		if status == "" {
			status = StatusReady
		}
		if !isValidStatus(status) {
			return PlanImportResult{}, fmt.Errorf("unknown task status %q", status)
		}
		priority := strings.TrimSpace(input.Priority)
		if priority == "" {
			priority = defaultPriority
		}
		phase := strings.TrimSpace(input.Phase)
		if phase == "" {
			phase = defaultTaskPhase
		}
		if phase != defaultTaskPhase {
			if _, ok := knownPhases[phase]; !ok {
				return PlanImportResult{}, fmt.Errorf("%w: task phase %s", ErrPhaseNotFound, phase)
			}
		}
		task := Task{
			ID:          s.nextID(nextTasks, now),
			Title:       title,
			Status:      status,
			Priority:    priority,
			Phase:       phase,
			Description: strings.TrimSpace(input.Description),
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		nextTasks = append(nextTasks, task)
		newTasks = append(newTasks, task)
	}

	return PlanImportResult{Phases: newPhases, Tasks: newTasks}, nil
}
