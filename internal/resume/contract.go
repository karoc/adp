package resume

import (
	"time"

	"github.com/karoc/adp/internal/sessions"
	taskstore "github.com/karoc/adp/internal/tasks"
)

const (
	SchemaVersion = 1

	StatusReady   = "ready"
	StatusPartial = "partial"
	StatusBlocked = "blocked"
	StatusClosed  = "closed"

	ClaimStateUnclaimed = "unclaimed"
	ClaimStateClaimed   = "claimed"
	ClaimStateLeased    = "leased"
	ClaimStateStale     = "stale"

	ActionInspectOnly    = "inspect_only"
	ActionClaim          = "claim"
	ActionRenew          = "renew"
	ActionRun            = "run"
	ActionWait           = "wait"
	ActionResolveBlocker = "resolve_blocker"
	ActionCreateTask     = "create_task"

	CommandSideEffectInspect         = "inspect"
	CommandSideEffectTaskMutation    = "task_mutation"
	CommandSideEffectRuntimeCreation = "runtime_creation"
)

type Request struct {
	Detail         *sessions.Detail
	Workspace      string
	TargetAgent    string
	Owner          string
	Lease          time.Duration
	Now            time.Time
	Task           *taskstore.Task
	TaskLoadError  string
	PhaseGate      *taskstore.PhaseGate
	PhaseLoadError string
}

type Plan struct {
	SchemaVersion     int         `json:"schema_version"`
	SessionID         string      `json:"session_id,omitempty"`
	Status            string      `json:"status"`
	Summary           string      `json:"summary"`
	Source            Source      `json:"source"`
	Target            Target      `json:"target"`
	Invocation        *Invocation `json:"invocation,omitempty"`
	Task              *TaskState  `json:"task,omitempty"`
	Phase             *PhaseState `json:"phase,omitempty"`
	MissingFields     []string    `json:"missing_fields,omitempty"`
	Guidance          []string    `json:"guidance"`
	SuggestedCommands []Command   `json:"suggested_commands"`
	Guarantees        Guarantees  `json:"guarantees"`
	Reasons           []string    `json:"reasons,omitempty"`
}

type Source struct {
	Workspace      string `json:"workspace,omitempty"`
	Agent          string `json:"agent,omitempty"`
	Profile        string `json:"profile,omitempty"`
	TaskID         string `json:"task_id,omitempty"`
	ProjectRoot    string `json:"project_root,omitempty"`
	RuntimePath    string `json:"runtime_path,omitempty"`
	StartedAt      string `json:"started_at,omitempty"`
	FinishedAt     string `json:"finished_at,omitempty"`
	ExitCode       *int   `json:"exit_code,omitempty"`
	DurationMillis *int64 `json:"duration_ms,omitempty"`
	EventCount     int    `json:"event_count"`
}

type Target struct {
	Workspace string `json:"workspace,omitempty"`
	Agent     string `json:"agent,omitempty"`
	Profile   string `json:"profile,omitempty"`
	Owner     string `json:"owner,omitempty"`
	Lease     string `json:"lease,omitempty"`
}

type Invocation struct {
	Available      bool     `json:"available"`
	SchemaVersion  int      `json:"schema_version,omitempty"`
	KeepRuntime    bool     `json:"keep_runtime"`
	AgentArgs      []string `json:"agent_args,omitempty"`
	Reused         []string `json:"reused,omitempty"`
	Omitted        []string `json:"omitted,omitempty"`
	OmissionReason string   `json:"omission_reason,omitempty"`
}

type TaskState struct {
	ID             string `json:"id"`
	Title          string `json:"title,omitempty"`
	Status         string `json:"status,omitempty"`
	Priority       string `json:"priority,omitempty"`
	Phase          string `json:"phase,omitempty"`
	Owner          string `json:"owner,omitempty"`
	ClaimState     string `json:"claim_state"`
	ClaimedAt      string `json:"claimed_at,omitempty"`
	LeaseExpiresAt string `json:"lease_expires_at,omitempty"`
	ResumeAction   string `json:"resume_action"`
	Reason         string `json:"reason"`
	BlockedReason  string `json:"blocked_reason,omitempty"`
}

type PhaseState struct {
	PhaseCount         int    `json:"phase_count"`
	OpenPhaseID        string `json:"open_phase_id,omitempty"`
	OpenPhaseStatus    string `json:"open_phase_status,omitempty"`
	NextPlannedPhaseID string `json:"next_planned_phase_id,omitempty"`
	CanStartNext       bool   `json:"can_start_next"`
	NextAction         string `json:"next_action,omitempty"`
	Reason             string `json:"reason,omitempty"`
}

type Command struct {
	Label      string   `json:"label"`
	SideEffect string   `json:"side_effect"`
	Args       []string `json:"args"`
	Reason     string   `json:"reason"`
}

type Guarantees struct {
	ReadOnly             bool `json:"read_only"`
	ProviderNativeResume bool `json:"provider_native_resume"`
	GitSideEffects       bool `json:"git_side_effects"`
	ProjectRootWrites    bool `json:"project_root_writes"`
	TaskMutation         bool `json:"task_mutation"`
	PhaseMutation        bool `json:"phase_mutation"`
	RuntimeCreation      bool `json:"runtime_creation"`
	EventLogWrites       bool `json:"event_log_writes"`
}
