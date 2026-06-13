package tasks

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

func (s *Store) Add(ctx context.Context, req AddRequest) (Task, error) {
	if err := ctx.Err(); err != nil {
		return Task{}, err
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		return Task{}, errors.New("task title is required")
	}
	if req.Status == "" {
		req.Status = StatusReady
	}
	if !isValidStatus(req.Status) {
		return Task{}, fmt.Errorf("unknown task status %q", req.Status)
	}
	if strings.TrimSpace(req.Priority) == "" {
		req.Priority = defaultPriority
	}
	if strings.TrimSpace(req.Phase) == "" {
		req.Phase = defaultTaskPhase
	}

	var task Task
	err := s.withPlanningLock(ctx, func() error {
		data, err := s.load(ctx)
		if err != nil {
			return err
		}
		phase := strings.TrimSpace(req.Phase)
		if err := s.validateTaskPhase(ctx, phase); err != nil {
			return err
		}
		now := s.now()
		task = Task{
			ID:          s.nextID(data.Tasks, now),
			Title:       req.Title,
			Status:      req.Status,
			Priority:    strings.TrimSpace(req.Priority),
			Phase:       phase,
			Description: strings.TrimSpace(req.Description),
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		data.Tasks = append(data.Tasks, task)
		if err := s.save(ctx, data); err != nil {
			return err
		}
		return s.appendEvent(ctx, progressEvent{Timestamp: now, Type: "task_created", TaskID: task.ID, Status: string(task.Status), Title: task.Title})
	})
	if err != nil {
		return Task{}, err
	}
	return task, nil
}

func (s *Store) List(ctx context.Context) ([]Task, error) {
	data, err := s.load(ctx)
	if err != nil {
		return nil, err
	}
	tasks := append([]Task(nil), data.Tasks...)
	sortTasks(tasks)
	return tasks, nil
}

func (s *Store) Get(ctx context.Context, id string) (Task, error) {
	data, err := s.load(ctx)
	if err != nil {
		return Task{}, err
	}
	for _, task := range data.Tasks {
		if task.ID == id {
			return task, nil
		}
	}
	return Task{}, fmt.Errorf("%w: %s", ErrTaskNotFound, id)
}

func (s *Store) FindByPrefix(ctx context.Context, prefix string) ([]Task, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return nil, errors.New("prefix cannot be empty")
	}

	data, err := s.load(ctx)
	if err != nil {
		return nil, err
	}

	// Check for exact match first
	for _, task := range data.Tasks {
		if task.ID == prefix {
			return []Task{task}, nil
		}
	}

	// Look for prefix matches
	var matches []Task
	for _, task := range data.Tasks {
		if strings.HasPrefix(task.ID, prefix) {
			matches = append(matches, task)
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("%w: no task with ID or prefix %q", ErrTaskNotFound, prefix)
	}

	if len(matches) > 1 {
		ids := make([]string, len(matches))
		for i, task := range matches {
			ids[i] = task.ID
		}
		return nil, fmt.Errorf("%w: prefix %q matches multiple tasks: %s", ErrAmbiguousTaskID, prefix, strings.Join(ids, ", "))
	}

	return matches, nil
}

func (s *Store) UpdateStatus(ctx context.Context, id string, status Status) (Task, error) {
	if !isValidStatus(status) {
		return Task{}, fmt.Errorf("unknown task status %q", status)
	}
	return s.update(ctx, id, func(task *Task, now time.Time) (progressEvent, error) {
		task.Status = status
		if status != StatusBlocked {
			task.BlockedReason = ""
		}
		return progressEvent{Timestamp: now, Type: "task_status_updated", TaskID: task.ID, Status: string(task.Status)}, nil
	})
}

func (s *Store) Block(ctx context.Context, id string, reason string) (Task, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return Task{}, errors.New("block reason is required")
	}
	return s.update(ctx, id, func(task *Task, now time.Time) (progressEvent, error) {
		task.Status = StatusBlocked
		task.BlockedReason = reason
		return progressEvent{Timestamp: now, Type: "task_blocked", TaskID: task.ID, Status: string(task.Status), Reason: reason}, nil
	})
}

func (s *Store) Claim(ctx context.Context, req ClaimRequest) (Task, error) {
	req.Owner = strings.TrimSpace(req.Owner)
	if req.Owner == "" {
		return Task{}, errors.New("owner is required")
	}
	if req.Lease < 0 {
		return Task{}, errors.New("lease must not be negative")
	}
	return s.update(ctx, req.TaskID, func(task *Task, now time.Time) (progressEvent, error) {
		if task.Owner != "" && task.Owner != req.Owner && !claimLeaseExpired(*task, now) {
			return progressEvent{}, fmt.Errorf("%w: task %s is claimed by %s", ErrTaskClaimed, task.ID, task.Owner)
		}
		if task.Status == StatusDone || task.Status == StatusCanceled {
			return progressEvent{}, fmt.Errorf("task %s with status %s cannot be claimed", task.ID, task.Status)
		}
		task.Owner = req.Owner
		task.ClaimedAt = now
		if req.Lease > 0 {
			task.LeaseExpiresAt = now.Add(req.Lease).UTC().Truncate(time.Second)
		} else {
			task.LeaseExpiresAt = time.Time{}
		}
		task.Status = StatusInProgress
		task.BlockedReason = ""
		return progressEvent{Timestamp: now, Type: "task_claimed", TaskID: task.ID, Status: string(task.Status), Owner: req.Owner, LeaseExpiresAt: task.LeaseExpiresAt}, nil
	})
}

func (s *Store) Take(ctx context.Context, req TakeRequest) (Task, error) {
	req.Owner = strings.TrimSpace(req.Owner)
	if req.Owner == "" {
		return Task{}, errors.New("owner is required")
	}
	if req.Lease < 0 {
		return Task{}, errors.New("lease must not be negative")
	}
	if err := ctx.Err(); err != nil {
		return Task{}, err
	}

	var task Task
	err := s.withPlanningLock(ctx, func() error {
		data, err := s.load(ctx)
		if err != nil {
			return err
		}
		now := s.now()
		candidates := claimableTasks(data.Tasks, now, 1)
		if len(candidates) == 0 {
			return ErrNoClaimableTask
		}
		targetID := candidates[0].ID
		for i := range data.Tasks {
			if data.Tasks[i].ID != targetID {
				continue
			}
			data.Tasks[i].Owner = req.Owner
			data.Tasks[i].ClaimedAt = now
			if req.Lease > 0 {
				data.Tasks[i].LeaseExpiresAt = now.Add(req.Lease).UTC().Truncate(time.Second)
			} else {
				data.Tasks[i].LeaseExpiresAt = time.Time{}
			}
			data.Tasks[i].Status = StatusInProgress
			data.Tasks[i].BlockedReason = ""
			data.Tasks[i].UpdatedAt = now
			task = data.Tasks[i]
			if err := s.save(ctx, data); err != nil {
				return err
			}
			return s.appendEvent(ctx, progressEvent{Timestamp: now, Type: "task_claimed", TaskID: task.ID, Status: string(task.Status), Owner: req.Owner, LeaseExpiresAt: task.LeaseExpiresAt})
		}
		return fmt.Errorf("%w: %s", ErrTaskNotFound, targetID)
	})
	if err != nil {
		return Task{}, err
	}
	return task, nil
}

func (s *Store) Renew(ctx context.Context, req RenewRequest) (Task, error) {
	req.Owner = strings.TrimSpace(req.Owner)
	if req.Owner == "" {
		return Task{}, errors.New("owner is required")
	}
	if req.Lease <= 0 {
		return Task{}, errors.New("lease must be positive")
	}
	return s.update(ctx, req.TaskID, func(task *Task, now time.Time) (progressEvent, error) {
		if task.Owner == "" || task.Owner != req.Owner {
			return progressEvent{}, fmt.Errorf("%w: task %s is claimed by %s", ErrTaskOwnerMismatch, task.ID, valueOrUnclaimed(task.Owner))
		}
		if task.Status == StatusDone || task.Status == StatusCanceled {
			return progressEvent{}, fmt.Errorf("task %s with status %s cannot renew a lease", task.ID, task.Status)
		}
		task.LeaseExpiresAt = now.Add(req.Lease).UTC().Truncate(time.Second)
		return progressEvent{Timestamp: now, Type: "task_lease_renewed", TaskID: task.ID, Status: string(task.Status), Owner: req.Owner, LeaseExpiresAt: task.LeaseExpiresAt}, nil
	})
}

func (s *Store) Stale(ctx context.Context) ([]Task, error) {
	data, err := s.load(ctx)
	if err != nil {
		return nil, err
	}
	return StaleTasks(data.Tasks, s.now()), nil
}

func (s *Store) Release(ctx context.Context, req ReleaseRequest) (Task, error) {
	req.Owner = strings.TrimSpace(req.Owner)
	return s.update(ctx, req.TaskID, func(task *Task, now time.Time) (progressEvent, error) {
		if req.Owner != "" && task.Owner != "" && task.Owner != req.Owner {
			return progressEvent{}, fmt.Errorf("%w: task %s is claimed by %s", ErrTaskOwnerMismatch, task.ID, task.Owner)
		}
		task.Owner = ""
		task.ClaimedAt = time.Time{}
		task.LeaseExpiresAt = time.Time{}
		if task.Status == StatusInProgress {
			task.Status = StatusReady
		}
		return progressEvent{Timestamp: now, Type: "task_released", TaskID: task.ID, Status: string(task.Status)}, nil
	})
}

func (s *Store) Progress(ctx context.Context) (Progress, error) {
	tasks, err := s.List(ctx)
	if err != nil {
		return Progress{}, err
	}
	progress := Progress{Total: len(tasks), Counts: map[Status]int{}}
	for _, status := range statusOrder {
		progress.Counts[status] = 0
	}
	for _, task := range tasks {
		progress.Counts[task.Status]++
	}
	progress.Next = NextTasks(tasks, defaultNextLimit)
	return progress, nil
}

func (s *Store) update(ctx context.Context, id string, mutate func(*Task, time.Time) (progressEvent, error)) (Task, error) {
	if err := ctx.Err(); err != nil {
		return Task{}, err
	}
	var task Task
	err := s.withPlanningLock(ctx, func() error {
		data, err := s.load(ctx)
		if err != nil {
			return err
		}
		now := s.now()
		for i := range data.Tasks {
			if data.Tasks[i].ID != id {
				continue
			}
			event, err := mutate(&data.Tasks[i], now)
			if err != nil {
				return err
			}
			data.Tasks[i].UpdatedAt = now
			task = data.Tasks[i]
			if err := s.save(ctx, data); err != nil {
				return err
			}
			return s.appendEvent(ctx, event)
		}
		return fmt.Errorf("%w: %s", ErrTaskNotFound, id)
	})
	if err != nil {
		return Task{}, err
	}
	return task, nil
}

func valueOrUnclaimed(owner string) string {
	if strings.TrimSpace(owner) == "" {
		return "unclaimed"
	}
	return owner
}
