package tasks

import (
	"errors"
	"time"
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
	ErrPhaseNotFound          = errors.New("phase not found")
	ErrPhaseInvalidTransition = errors.New("invalid phase transition")
	phaseStatuses             = []PhaseStatus{PhaseStatusPlanned, PhaseStatusActive, PhaseStatusAccepted, PhaseStatusCommitted, PhaseStatusPushed}
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
