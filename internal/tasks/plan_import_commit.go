package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type stagedPlanningFile struct {
	path string
	tmp  string
	data []byte
}

type planningFileSnapshot struct {
	path   string
	data   []byte
	exists bool
}

func (s *Store) commitPlanImport(ctx context.Context, phaseData phaseFile, taskData taskFile, result PlanImportResult) error {
	files := make([]stagedPlanningFile, 0, 3)
	if len(result.Phases) > 0 {
		data, err := encodePlanImportPhases(phaseData)
		if err != nil {
			return err
		}
		files = append(files, stagedPlanningFile{path: s.phasesPath(), data: data})
	}
	if len(result.Tasks) > 0 {
		data, err := encodePlanImportTasks(taskData)
		if err != nil {
			return err
		}
		files = append(files, stagedPlanningFile{path: s.tasksPath(), data: data})
	}
	events := planImportEvents(result)
	if len(events) > 0 {
		data, err := s.planImportProgressData(ctx, events)
		if err != nil {
			return err
		}
		files = append(files, stagedPlanningFile{path: s.progressPath(), data: data})
	}
	return replacePlanningFiles(ctx, files)
}

func encodePlanImportPhases(file phaseFile) ([]byte, error) {
	file.Version = currentVersion
	sortPhases(file.Phases)
	data, err := yaml.Marshal(file)
	if err != nil {
		return nil, fmt.Errorf("encode phase file: %w", err)
	}
	return data, nil
}

func encodePlanImportTasks(file taskFile) ([]byte, error) {
	file.Version = currentVersion
	sortTasks(file.Tasks)
	data, err := yaml.Marshal(file)
	if err != nil {
		return nil, fmt.Errorf("encode task file: %w", err)
	}
	return data, nil
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

func replacePlanningFiles(ctx context.Context, files []stagedPlanningFile) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	for i := range files {
		files[i].tmp = files[i].path + ".tmp"
	}
	defer cleanupStagedPlanningFiles(files)

	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := os.WriteFile(file.tmp, file.data, 0o644); err != nil {
			return fmt.Errorf("write temporary planning file %s: %w", file.tmp, err)
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	snapshots, err := snapshotPlanningFiles(files)
	if err != nil {
		return err
	}
	// Once replacement starts, finish the batch or roll back to avoid mixed ledger state.
	for _, file := range files {
		if err := os.Rename(file.tmp, file.path); err != nil {
			rollbackErr := restorePlanningFiles(snapshots)
			if rollbackErr != nil {
				return fmt.Errorf("replace planning file %s: %w; rollback failed: %v", file.path, err, rollbackErr)
			}
			return fmt.Errorf("replace planning file %s: %w", file.path, err)
		}
	}
	return nil
}

func cleanupStagedPlanningFiles(files []stagedPlanningFile) {
	for _, file := range files {
		_ = os.Remove(file.tmp)
	}
}

func snapshotPlanningFiles(files []stagedPlanningFile) ([]planningFileSnapshot, error) {
	snapshots := make([]planningFileSnapshot, 0, len(files))
	for _, file := range files {
		data, err := os.ReadFile(file.path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				snapshots = append(snapshots, planningFileSnapshot{path: file.path})
				continue
			}
			return nil, fmt.Errorf("snapshot planning file %s: %w", file.path, err)
		}
		snapshots = append(snapshots, planningFileSnapshot{path: file.path, data: data, exists: true})
	}
	return snapshots, nil
}

func restorePlanningFiles(snapshots []planningFileSnapshot) error {
	for _, snapshot := range snapshots {
		if snapshot.exists {
			if err := os.WriteFile(snapshot.path, snapshot.data, 0o644); err != nil {
				return fmt.Errorf("restore planning file %s: %w", snapshot.path, err)
			}
			continue
		}
		if err := os.Remove(snapshot.path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove planning file %s: %w", snapshot.path, err)
		}
	}
	return nil
}
