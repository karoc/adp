package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	planningDir      = "planning"
	tasksFile        = "tasks.yaml"
	phasesFile       = "phases.yaml"
	progressFile     = "progress.jsonl"
	currentVersion   = 1
	defaultPriority  = "normal"
	defaultTaskPhase = "unassigned"
)

type Status string

const (
	StatusPlanned    Status = "planned"
	StatusReady      Status = "ready"
	StatusInProgress Status = "in_progress"
	StatusBlocked    Status = "blocked"
	StatusReview     Status = "review"
	StatusValidated  Status = "validated"
	StatusDone       Status = "done"
	StatusCanceled   Status = "canceled"
)

var (
	ErrTaskNotFound      = errors.New("task not found")
	ErrTaskClaimed       = errors.New("task already claimed")
	ErrTaskOwnerMismatch = errors.New("task owner mismatch")
	statusOrder          = []Status{StatusPlanned, StatusReady, StatusInProgress, StatusBlocked, StatusReview, StatusValidated, StatusDone, StatusCanceled}
)

type Task struct {
	ID             string    `yaml:"id"`
	Title          string    `yaml:"title"`
	Status         Status    `yaml:"status"`
	Priority       string    `yaml:"priority"`
	Phase          string    `yaml:"phase"`
	Owner          string    `yaml:"owner,omitempty"`
	ClaimedAt      time.Time `yaml:"claimed_at,omitempty"`
	LeaseExpiresAt time.Time `yaml:"lease_expires_at,omitempty"`
	Description    string    `yaml:"description,omitempty"`
	BlockedReason  string    `yaml:"blocked_reason,omitempty"`
	CreatedAt      time.Time `yaml:"created_at"`
	UpdatedAt      time.Time `yaml:"updated_at"`
}

type AddRequest struct {
	Title       string
	Description string
	Priority    string
	Phase       string
	Status      Status
}

type ClaimRequest struct {
	TaskID string
	Owner  string
	Lease  time.Duration
}

type ReleaseRequest struct {
	TaskID string
	Owner  string
}

type Progress struct {
	Total  int
	Counts map[Status]int
	Next   []Task
}

type Store struct {
	WorkspaceDir string
	Now          func() time.Time
}

type taskFile struct {
	Version int    `yaml:"version"`
	Tasks   []Task `yaml:"tasks"`
}

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

func NewStore(workspaceDir string) *Store {
	return &Store{WorkspaceDir: workspaceDir, Now: time.Now}
}

func Statuses() []Status {
	statuses := make([]Status, len(statusOrder))
	copy(statuses, statusOrder)
	return statuses
}

func ParseStatus(value string) (Status, error) {
	status := Status(strings.TrimSpace(value))
	if !isValidStatus(status) {
		return "", fmt.Errorf("unknown task status %q", value)
	}
	return status, nil
}

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
		if task.Status == StatusReady || task.Status == StatusInProgress || task.Status == StatusReview {
			progress.Next = append(progress.Next, task)
		}
	}
	if len(progress.Next) > 5 {
		progress.Next = progress.Next[:5]
	}
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

func (s *Store) load(ctx context.Context) (taskFile, error) {
	if err := ctx.Err(); err != nil {
		return taskFile{}, err
	}
	path := s.tasksPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return taskFile{Version: currentVersion}, nil
		}
		return taskFile{}, fmt.Errorf("read task file %s: %w", path, err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return taskFile{Version: currentVersion}, nil
	}

	var file taskFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return taskFile{}, fmt.Errorf("parse task file %s: %w", path, err)
	}
	if file.Version == 0 {
		file.Version = currentVersion
	}
	if file.Version != currentVersion {
		return taskFile{}, fmt.Errorf("unsupported task file version %d", file.Version)
	}
	for _, task := range file.Tasks {
		if !isValidStatus(task.Status) {
			return taskFile{}, fmt.Errorf("task %s has unknown status %q", task.ID, task.Status)
		}
	}
	return file, nil
}

func (s *Store) save(ctx context.Context, file taskFile) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	file.Version = currentVersion
	sortTasks(file.Tasks)

	if err := os.MkdirAll(s.planningPath(), 0o755); err != nil {
		return fmt.Errorf("create planning directory: %w", err)
	}
	data, err := yaml.Marshal(file)
	if err != nil {
		return fmt.Errorf("encode task file: %w", err)
	}
	path := s.tasksPath()
	tmpPath := path + ".tmp"
	defer os.Remove(tmpPath)

	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write temporary task file %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace task file %s: %w", path, err)
	}
	return nil
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

func (s *Store) nextID(tasks []Task, now time.Time) string {
	prefix := "task-" + now.UTC().Format("20060102") + "-"
	maxSeq := 0
	for _, task := range tasks {
		if !strings.HasPrefix(task.ID, prefix) {
			continue
		}
		seqText := strings.TrimPrefix(task.ID, prefix)
		var seq int
		if _, err := fmt.Sscanf(seqText, "%d", &seq); err == nil && seq > maxSeq {
			maxSeq = seq
		}
	}
	return fmt.Sprintf("%s%04d", prefix, maxSeq+1)
}

func (s *Store) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC().Truncate(time.Second)
	}
	return time.Now().UTC().Truncate(time.Second)
}

func (s *Store) planningPath() string {
	return filepath.Join(s.WorkspaceDir, planningDir)
}

func (s *Store) tasksPath() string {
	return filepath.Join(s.planningPath(), tasksFile)
}

func (s *Store) phasesPath() string {
	return filepath.Join(s.planningPath(), phasesFile)
}

func (s *Store) progressPath() string {
	return filepath.Join(s.planningPath(), progressFile)
}

func (s *Store) lockPath() string {
	return filepath.Join(s.planningPath(), planningLockFile)
}

func isValidStatus(status Status) bool {
	for _, known := range statusOrder {
		if status == known {
			return true
		}
	}
	return false
}

func sortTasks(tasks []Task) {
	sort.SliceStable(tasks, func(i, j int) bool {
		if tasks[i].CreatedAt.Equal(tasks[j].CreatedAt) {
			return tasks[i].ID < tasks[j].ID
		}
		return tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
	})
}

func claimLeaseExpired(task Task, now time.Time) bool {
	return !task.LeaseExpiresAt.IsZero() && !task.LeaseExpiresAt.After(now)
}
