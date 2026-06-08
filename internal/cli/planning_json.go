package cli

import (
	"encoding/json"
	"io"
	"time"

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
