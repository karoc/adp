package tasks

import (
	"errors"
	"fmt"
	"strings"
	"time"
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
	ErrNoClaimableTask   = errors.New("no claimable task")
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

type TakeRequest struct {
	Owner string
	Lease time.Duration
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

type taskFile struct {
	Version int    `yaml:"version"`
	Tasks   []Task `yaml:"tasks"`
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

func isValidStatus(status Status) bool {
	for _, known := range statusOrder {
		if status == known {
			return true
		}
	}
	return false
}

func claimLeaseExpired(task Task, now time.Time) bool {
	return !task.LeaseExpiresAt.IsZero() && !task.LeaseExpiresAt.After(now)
}
