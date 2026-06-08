package cli

import (
	"encoding/json"
	"io"
	"time"

	"github.com/karoc/adp/internal/sessions"
	taskstore "github.com/karoc/adp/internal/tasks"
)

type taskListJSON struct {
	Workspace string     `json:"workspace"`
	Tasks     []taskJSON `json:"tasks"`
}

type taskJSON struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	Status         string  `json:"status"`
	Priority       string  `json:"priority"`
	Phase          string  `json:"phase"`
	Owner          string  `json:"owner,omitempty"`
	ClaimedAt      *string `json:"claimed_at,omitempty"`
	LeaseExpiresAt *string `json:"lease_expires_at,omitempty"`
	Description    string  `json:"description,omitempty"`
	BlockedReason  string  `json:"blocked_reason,omitempty"`
	CreatedAt      *string `json:"created_at,omitempty"`
	UpdatedAt      *string `json:"updated_at,omitempty"`
}

type phaseListJSON struct {
	Workspace string      `json:"workspace"`
	Phases    []phaseJSON `json:"phases"`
}

type phaseJSON struct {
	ID         string                `json:"id"`
	Title      string                `json:"title"`
	Status     string                `json:"status"`
	Goal       string                `json:"goal,omitempty"`
	Acceptance *acceptanceRecordJSON `json:"acceptance,omitempty"`
	Commit     *commitRecordJSON     `json:"commit,omitempty"`
	Push       *pushRecordJSON       `json:"push,omitempty"`
	CreatedAt  *string               `json:"created_at,omitempty"`
	UpdatedAt  *string               `json:"updated_at,omitempty"`
}

type acceptanceRecordJSON struct {
	Commands []string `json:"commands,omitempty"`
	Result   string   `json:"result,omitempty"`
	Notes    string   `json:"notes,omitempty"`
	At       *string  `json:"at,omitempty"`
}

type commitRecordJSON struct {
	Hash    string  `json:"hash,omitempty"`
	Message string  `json:"message,omitempty"`
	At      *string `json:"at,omitempty"`
}

type pushRecordJSON struct {
	Remote string  `json:"remote,omitempty"`
	Branch string  `json:"branch,omitempty"`
	Result string  `json:"result,omitempty"`
	At     *string `json:"at,omitempty"`
}

type progressJSON struct {
	Workspace string         `json:"workspace"`
	Phases    []phaseJSON    `json:"phases"`
	Total     int            `json:"total"`
	Counts    map[string]int `json:"counts"`
	Next      []taskJSON     `json:"next"`
}

type progressReportJSON struct {
	Workspace       string               `json:"workspace"`
	Phases          []phaseJSON          `json:"phases"`
	Total           int                  `json:"total"`
	Counts          map[string]int       `json:"counts"`
	Tasks           []taskJSON           `json:"tasks"`
	Next            []taskJSON           `json:"next"`
	PhaseEvidence   []phaseEvidenceJSON  `json:"phase_evidence"`
	RuntimeSessions []runtimeSessionJSON `json:"runtime_sessions"`
}

type phaseEvidenceJSON struct {
	ID         string                `json:"id"`
	Acceptance *acceptanceRecordJSON `json:"acceptance,omitempty"`
	Commit     *commitRecordJSON     `json:"commit,omitempty"`
	Push       *pushRecordJSON       `json:"push,omitempty"`
}

type runtimeSessionJSON struct {
	SessionID      string  `json:"session_id"`
	Workspace      string  `json:"workspace,omitempty"`
	Agent          string  `json:"agent,omitempty"`
	Profile        string  `json:"profile,omitempty"`
	ProjectRoot    string  `json:"project_root,omitempty"`
	RuntimePath    string  `json:"runtime_path,omitempty"`
	TaskID         string  `json:"task_id,omitempty"`
	StartedAt      *string `json:"started_at,omitempty"`
	FinishedAt     *string `json:"finished_at,omitempty"`
	ExitCode       *int    `json:"exit_code,omitempty"`
	DurationMillis *int64  `json:"duration_ms,omitempty"`
	EventCount     int     `json:"event_count"`
}

func writePlanningJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func taskListOutput(workspace string, tasks []taskstore.Task) taskListJSON {
	out := taskListJSON{Workspace: workspace, Tasks: make([]taskJSON, 0, len(tasks))}
	for _, task := range tasks {
		out.Tasks = append(out.Tasks, taskOutput(task))
	}
	return out
}

func taskOutput(task taskstore.Task) taskJSON {
	return taskJSON{
		ID:             task.ID,
		Title:          task.Title,
		Status:         string(task.Status),
		Priority:       task.Priority,
		Phase:          task.Phase,
		Owner:          task.Owner,
		ClaimedAt:      jsonTime(task.ClaimedAt),
		LeaseExpiresAt: jsonTime(task.LeaseExpiresAt),
		Description:    task.Description,
		BlockedReason:  task.BlockedReason,
		CreatedAt:      jsonTime(task.CreatedAt),
		UpdatedAt:      jsonTime(task.UpdatedAt),
	}
}

func phaseListOutput(workspace string, phases []taskstore.Phase) phaseListJSON {
	out := phaseListJSON{Workspace: workspace, Phases: make([]phaseJSON, 0, len(phases))}
	for _, phase := range phases {
		out.Phases = append(out.Phases, phaseOutput(phase))
	}
	return out
}

func phaseOutput(phase taskstore.Phase) phaseJSON {
	return phaseJSON{
		ID:         phase.ID,
		Title:      phase.Title,
		Status:     string(phase.Status),
		Goal:       phase.Goal,
		CreatedAt:  jsonTime(phase.CreatedAt),
		UpdatedAt:  jsonTime(phase.UpdatedAt),
		Acceptance: acceptanceOutput(phase.Acceptance),
		Commit:     commitOutput(phase.Commit),
		Push:       pushOutput(phase.Push),
	}
}

func progressOutput(workspace string, progress taskstore.Progress, phases []taskstore.Phase) progressJSON {
	counts := map[string]int{}
	for _, status := range taskstore.Statuses() {
		counts[string(status)] = progress.Counts[status]
	}
	next := make([]taskJSON, 0, len(progress.Next))
	for _, task := range progress.Next {
		next = append(next, taskOutput(task))
	}
	return progressJSON{
		Workspace: workspace,
		Phases:    phaseListOutput(workspace, phases).Phases,
		Total:     progress.Total,
		Counts:    counts,
		Next:      next,
	}
}

func progressReportOutput(data progressReportData) progressReportJSON {
	counts := map[string]int{}
	for _, status := range taskstore.Statuses() {
		counts[string(status)] = data.Progress.Counts[status]
	}
	tasks := make([]taskJSON, 0, len(data.Tasks))
	for _, task := range data.Tasks {
		tasks = append(tasks, taskOutput(task))
	}
	next := make([]taskJSON, 0, len(data.Tasks))
	for _, task := range reportableTasks(data.Tasks) {
		next = append(next, taskOutput(task))
	}
	evidence := make([]phaseEvidenceJSON, 0, len(data.Phases))
	for _, phase := range data.Phases {
		evidence = append(evidence, phaseEvidenceOutput(phase))
	}
	runtimeSessions := make([]runtimeSessionJSON, 0, len(data.RuntimeSessions))
	for _, session := range data.RuntimeSessions {
		runtimeSessions = append(runtimeSessions, runtimeSessionOutput(session))
	}
	return progressReportJSON{
		Workspace:       data.Workspace,
		Phases:          phaseListOutput(data.Workspace, data.Phases).Phases,
		Total:           data.Progress.Total,
		Counts:          counts,
		Tasks:           tasks,
		Next:            next,
		PhaseEvidence:   evidence,
		RuntimeSessions: runtimeSessions,
	}
}

func phaseEvidenceOutput(phase taskstore.Phase) phaseEvidenceJSON {
	return phaseEvidenceJSON{
		ID:         phase.ID,
		Acceptance: acceptanceOutput(phase.Acceptance),
		Commit:     commitOutput(phase.Commit),
		Push:       pushOutput(phase.Push),
	}
}

func runtimeSessionOutput(summary sessions.Summary) runtimeSessionJSON {
	return runtimeSessionJSON{
		SessionID:      summary.SessionID,
		Workspace:      summary.Workspace,
		Agent:          summary.Agent,
		Profile:        summary.Profile,
		ProjectRoot:    summary.ProjectRoot,
		RuntimePath:    summary.RuntimePath,
		TaskID:         summary.TaskID,
		StartedAt:      jsonTime(summary.StartedAt),
		FinishedAt:     jsonTime(summary.FinishedAt),
		ExitCode:       summary.ExitCode,
		DurationMillis: summary.DurationMillis,
		EventCount:     summary.EventCount,
	}
}

func jsonTime(value time.Time) *string {
	if value.IsZero() {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339)
	return &formatted
}

func acceptanceOutput(record taskstore.AcceptanceRecord) *acceptanceRecordJSON {
	if len(record.Commands) == 0 && record.Result == "" && record.Notes == "" && record.At.IsZero() {
		return nil
	}
	return &acceptanceRecordJSON{
		Commands: append([]string(nil), record.Commands...),
		Result:   record.Result,
		Notes:    record.Notes,
		At:       jsonTime(record.At),
	}
}

func commitOutput(record taskstore.CommitRecord) *commitRecordJSON {
	if record.Hash == "" && record.Message == "" && record.At.IsZero() {
		return nil
	}
	return &commitRecordJSON{
		Hash:    record.Hash,
		Message: record.Message,
		At:      jsonTime(record.At),
	}
}

func pushOutput(record taskstore.PushRecord) *pushRecordJSON {
	if record.Remote == "" && record.Branch == "" && record.Result == "" && record.At.IsZero() {
		return nil
	}
	return &pushRecordJSON{
		Remote: record.Remote,
		Branch: record.Branch,
		Result: record.Result,
		At:     jsonTime(record.At),
	}
}
