package cli

import (
	"context"
	"time"

	taskstore "github.com/karoc/adp/internal/tasks"
)

type fakeTaskStore struct {
	addReq            taskstore.AddRequest
	tasks             []taskstore.Task
	phases            []taskstore.Phase
	updatedStatus     taskstore.Status
	blockReason       string
	claimReq          taskstore.ClaimRequest
	takeReq           taskstore.TakeRequest
	releaseReq        taskstore.ReleaseRequest
	progress          taskstore.Progress
	planReq           taskstore.PlanImportRequest
	planPreviewResult taskstore.PlanImportResult
	planApplyResult   taskstore.PlanImportResult
	planningReport    taskstore.PlanningDiagnosticReport
	previewCalls      int
	applyCalls        int
	doctorCalls       int
}

func (s *fakeTaskStore) Add(_ context.Context, req taskstore.AddRequest) (taskstore.Task, error) {
	s.addReq = req
	return testTask("task-1", req.Title, taskstore.StatusReady), nil
}

func (s *fakeTaskStore) List(context.Context) ([]taskstore.Task, error) {
	return s.tasks, nil
}

func (s *fakeTaskStore) Get(_ context.Context, id string) (taskstore.Task, error) {
	for _, task := range s.tasks {
		if task.ID == id {
			return task, nil
		}
	}
	return testTask(id, "Add task manager", taskstore.StatusReady), nil
}

func (s *fakeTaskStore) UpdateStatus(_ context.Context, id string, status taskstore.Status) (taskstore.Task, error) {
	s.updatedStatus = status
	return testTask(id, "Add task manager", status), nil
}

func (s *fakeTaskStore) Block(_ context.Context, id string, reason string) (taskstore.Task, error) {
	s.blockReason = reason
	task := testTask(id, "Add task manager", taskstore.StatusBlocked)
	task.BlockedReason = reason
	return task, nil
}

func (s *fakeTaskStore) Claim(_ context.Context, req taskstore.ClaimRequest) (taskstore.Task, error) {
	s.claimReq = req
	task := testTask(req.TaskID, "Add task manager", taskstore.StatusInProgress)
	task.Owner = req.Owner
	if req.Lease > 0 {
		task.LeaseExpiresAt = task.UpdatedAt.Add(req.Lease)
	}
	return task, nil
}

func (s *fakeTaskStore) Take(_ context.Context, req taskstore.TakeRequest) (taskstore.Task, error) {
	s.takeReq = req
	task := testTask("task-take", "Take next task", taskstore.StatusInProgress)
	task.Owner = req.Owner
	task.ClaimedAt = task.UpdatedAt
	if req.Lease > 0 {
		task.LeaseExpiresAt = task.UpdatedAt.Add(req.Lease)
	}
	return task, nil
}

func (s *fakeTaskStore) Release(_ context.Context, req taskstore.ReleaseRequest) (taskstore.Task, error) {
	s.releaseReq = req
	return testTask(req.TaskID, "Add task manager", taskstore.StatusReady), nil
}

func (s *fakeTaskStore) Progress(context.Context) (taskstore.Progress, error) {
	return s.progress, nil
}

func (s *fakeTaskStore) PreviewPlanImport(_ context.Context, req taskstore.PlanImportRequest) (taskstore.PlanImportResult, error) {
	s.planReq = req
	s.previewCalls++
	return s.planPreviewResult, nil
}

func (s *fakeTaskStore) ApplyPlanImport(_ context.Context, req taskstore.PlanImportRequest) (taskstore.PlanImportResult, error) {
	s.planReq = req
	s.applyCalls++
	return s.planApplyResult, nil
}

func (s *fakeTaskStore) DiagnosePlanning(context.Context) (taskstore.PlanningDiagnosticReport, error) {
	s.doctorCalls++
	return s.planningReport, nil
}

func (s *fakeTaskStore) AddPhase(_ context.Context, req taskstore.PhaseAddRequest) (taskstore.Phase, error) {
	phase := testPhase(req.ID, req.Title, taskstore.PhaseStatusPlanned)
	phase.Goal = req.Goal
	s.phases = append(s.phases, phase)
	return phase, nil
}

func (s *fakeTaskStore) ListPhases(context.Context) ([]taskstore.Phase, error) {
	return s.phases, nil
}

func (s *fakeTaskStore) GetPhase(_ context.Context, id string) (taskstore.Phase, error) {
	for _, phase := range s.phases {
		if phase.ID == id {
			return phase, nil
		}
	}
	return testPhase(id, "Project planning", taskstore.PhaseStatusPushed), nil
}

func (s *fakeTaskStore) StartPhase(_ context.Context, id string) (taskstore.Phase, error) {
	phase := s.currentPhase(id)
	phase.Status = taskstore.PhaseStatusActive
	s.upsertPhase(phase)
	return phase, nil
}

func (s *fakeTaskStore) AcceptPhase(_ context.Context, req taskstore.PhaseAcceptRequest) (taskstore.Phase, error) {
	phase := s.currentPhase(req.ID)
	phase.Status = taskstore.PhaseStatusAccepted
	phase.Acceptance = taskstore.AcceptanceRecord{Commands: req.Commands, Result: req.Result, Notes: req.Notes, At: phase.UpdatedAt}
	s.upsertPhase(phase)
	return phase, nil
}

func (s *fakeTaskStore) RecordPhaseCommit(_ context.Context, req taskstore.PhaseCommitRequest) (taskstore.Phase, error) {
	phase := s.currentPhase(req.ID)
	phase.Status = taskstore.PhaseStatusCommitted
	phase.Commit = taskstore.CommitRecord{Hash: req.Hash, Message: req.Message, At: phase.UpdatedAt}
	s.upsertPhase(phase)
	return phase, nil
}

func (s *fakeTaskStore) RecordPhasePush(_ context.Context, req taskstore.PhasePushRequest) (taskstore.Phase, error) {
	phase := s.currentPhase(req.ID)
	phase.Status = taskstore.PhaseStatusPushed
	phase.Push = taskstore.PushRecord{Remote: req.Remote, Branch: req.Branch, Result: req.Result, At: phase.UpdatedAt}
	s.upsertPhase(phase)
	return phase, nil
}

func (s *fakeTaskStore) currentPhase(id string) taskstore.Phase {
	for _, phase := range s.phases {
		if phase.ID == id {
			return phase
		}
	}
	return testPhase(id, "Project planning", taskstore.PhaseStatusPlanned)
}

func (s *fakeTaskStore) upsertPhase(next taskstore.Phase) {
	for i := range s.phases {
		if s.phases[i].ID == next.ID {
			s.phases[i] = next
			return
		}
	}
	s.phases = append(s.phases, next)
}

func testTask(id string, title string, status taskstore.Status) taskstore.Task {
	ts := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)
	return taskstore.Task{
		ID:        id,
		Title:     title,
		Status:    status,
		Priority:  "high",
		Phase:     "phase-1.5",
		CreatedAt: ts,
		UpdatedAt: ts,
	}
}

func testPhase(id string, title string, status taskstore.PhaseStatus) taskstore.Phase {
	ts := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)
	return taskstore.Phase{
		ID:        id,
		Title:     title,
		Status:    status,
		CreatedAt: ts,
		UpdatedAt: ts,
	}
}
