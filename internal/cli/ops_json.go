package cli

import (
	"time"

	"github.com/karoc/adp/internal/events"
	"github.com/karoc/adp/internal/runtime"
	"github.com/karoc/adp/internal/sessions"
)

type eventsListJSON struct {
	Filters eventsListFiltersJSON `json:"filters"`
	Limit   int                   `json:"limit"`
	Count   int                   `json:"count"`
	Events  []eventJSON           `json:"events"`
}

type eventsListFiltersJSON struct {
	Workspace string `json:"workspace,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	TaskID    string `json:"task_id,omitempty"`
	Type      string `json:"type,omitempty"`
}

type eventJSON struct {
	Timestamp      *string        `json:"ts,omitempty"`
	Type           string         `json:"type"`
	Workspace      string         `json:"workspace,omitempty"`
	Agent          string         `json:"agent,omitempty"`
	Profile        string         `json:"profile,omitempty"`
	RuntimePath    string         `json:"runtime_path,omitempty"`
	ProjectRoot    string         `json:"project_root,omitempty"`
	SessionID      string         `json:"session_id,omitempty"`
	TaskID         string         `json:"task_id,omitempty"`
	PID            int            `json:"pid,omitempty"`
	ExitCode       *int           `json:"exit_code,omitempty"`
	DurationMillis int64          `json:"duration_ms,omitempty"`
	Fields         map[string]any `json:"fields,omitempty"`
}

type sessionsListJSON struct {
	Filters  sessionsListFiltersJSON `json:"filters"`
	Limit    int                     `json:"limit"`
	Count    int                     `json:"count"`
	Sessions []runtimeSessionJSON    `json:"sessions"`
}

type sessionsListFiltersJSON struct {
	Workspace string `json:"workspace,omitempty"`
	Agent     string `json:"agent,omitempty"`
	TaskID    string `json:"task_id,omitempty"`
}

type sessionDetailJSON struct {
	Summary runtimeSessionJSON `json:"summary"`
	Events  []eventJSON        `json:"events"`
}

type runtimePruneJSON struct {
	OlderThanSeconds int64                    `json:"older_than_seconds"`
	OlderThan        string                   `json:"older_than"`
	IncludeKept      bool                     `json:"include_kept"`
	DryRun           bool                     `json:"dry_run"`
	Count            int                      `json:"count"`
	Results          []runtimePruneResultJSON `json:"results"`
}

type runtimePruneResultJSON struct {
	Action    string  `json:"action"`
	Workspace string  `json:"workspace,omitempty"`
	SessionID string  `json:"session_id,omitempty"`
	CreatedAt *string `json:"created_at,omitempty"`
	Keep      bool    `json:"keep"`
	Removed   bool    `json:"removed"`
	DryRun    bool    `json:"dry_run"`
	Root      string  `json:"root,omitempty"`
}

func eventsListOutput(opts eventsListOptions, read []events.Event) eventsListJSON {
	out := eventsListJSON{
		Filters: eventsListFiltersJSON{
			Workspace: opts.workspace,
			SessionID: opts.sessionID,
			TaskID:    opts.taskID,
			Type:      opts.eventType,
		},
		Limit:  opts.limit,
		Count:  len(read),
		Events: make([]eventJSON, 0, len(read)),
	}
	for _, event := range read {
		out.Events = append(out.Events, eventOutput(event))
	}
	return out
}

func sessionsListOutput(opts sessionsListOptions, summaries []sessions.Summary) sessionsListJSON {
	out := sessionsListJSON{
		Filters: sessionsListFiltersJSON{
			Workspace: opts.workspace,
			Agent:     opts.agent,
			TaskID:    opts.taskID,
		},
		Limit:    opts.limit,
		Count:    len(summaries),
		Sessions: make([]runtimeSessionJSON, 0, len(summaries)),
	}
	for _, summary := range summaries {
		out.Sessions = append(out.Sessions, runtimeSessionOutput(summary))
	}
	return out
}

func sessionDetailOutput(detail *sessions.Detail) sessionDetailJSON {
	if detail == nil {
		return sessionDetailJSON{}
	}
	out := sessionDetailJSON{
		Summary: runtimeSessionOutput(detail.Summary),
		Events:  make([]eventJSON, 0, len(detail.Events)),
	}
	for _, event := range detail.Events {
		out.Events = append(out.Events, eventOutput(event))
	}
	return out
}

func runtimePruneOutput(opts runtimePruneOptions, results []runtime.PruneResult) runtimePruneJSON {
	out := runtimePruneJSON{
		OlderThanSeconds: int64(opts.olderThan / time.Second),
		OlderThan:        opts.olderThan.String(),
		IncludeKept:      opts.includeKept,
		DryRun:           opts.dryRun,
		Count:            len(results),
		Results:          make([]runtimePruneResultJSON, 0, len(results)),
	}
	for _, result := range results {
		out.Results = append(out.Results, runtimePruneResultOutput(result))
	}
	return out
}

func eventOutput(event events.Event) eventJSON {
	return eventJSON{
		Timestamp:      jsonTime(event.Timestamp),
		Type:           event.Type,
		Workspace:      event.Workspace,
		Agent:          event.Agent,
		Profile:        event.Profile,
		RuntimePath:    event.RuntimePath,
		ProjectRoot:    event.ProjectRoot,
		SessionID:      event.SessionID,
		TaskID:         event.TaskID,
		PID:            event.PID,
		ExitCode:       event.ExitCode,
		DurationMillis: event.DurationMillis,
		Fields:         event.Fields,
	}
}

func runtimePruneResultOutput(result runtime.PruneResult) runtimePruneResultJSON {
	return runtimePruneResultJSON{
		Action:    formatPruneAction(result),
		Workspace: result.Workspace,
		SessionID: result.SessionID,
		CreatedAt: jsonTime(result.CreatedAt),
		Keep:      result.Keep,
		Removed:   result.Removed,
		DryRun:    result.DryRun,
		Root:      result.Root,
	}
}
