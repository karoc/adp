package sessions

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/karoc/adp/internal/events"
	"github.com/karoc/adp/internal/paths"
)

const (
	eventTypeRunStarted  = "run_started"
	eventTypeRunFinished = "run_finished"
)

var (
	ErrInvalidLimit = errors.New("session query limit must be non-negative")
	ErrNotFound     = errors.New("session not found")
)

type Query struct {
	Workspace string
	Agent     string
	TaskID    string
	Limit     int
}

type Summary struct {
	SessionID      string    `json:"session_id"`
	Workspace      string    `json:"workspace,omitempty"`
	Agent          string    `json:"agent,omitempty"`
	Profile        string    `json:"profile,omitempty"`
	ProjectRoot    string    `json:"project_root,omitempty"`
	RuntimePath    string    `json:"runtime_path,omitempty"`
	TaskID         string    `json:"task_id,omitempty"`
	StartedAt      time.Time `json:"started_at,omitempty"`
	FinishedAt     time.Time `json:"finished_at,omitempty"`
	ExitCode       *int      `json:"exit_code,omitempty"`
	DurationMillis *int64    `json:"duration_ms,omitempty"`
	EventCount     int       `json:"event_count"`
}

type Detail struct {
	Summary Summary        `json:"summary"`
	Events  []events.Event `json:"events"`
}

func List(ctx context.Context, layout paths.Layout, query Query) ([]Summary, error) {
	if err := validateQuery(query); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	read, err := events.Read(ctx, layout, events.Query{})
	if err != nil {
		return nil, err
	}
	return listFromEvents(ctx, read, query)
}

func Get(ctx context.Context, layout paths.Layout, sessionID string) (*Detail, error) {
	if sessionID == "" {
		return nil, ErrNotFound
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	read, err := events.Read(ctx, layout, events.Query{SessionID: sessionID})
	if err != nil {
		return nil, err
	}
	detail, ok, err := detailFromEvents(ctx, read, sessionID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	return detail, nil
}

func validateQuery(query Query) error {
	if query.Limit < 0 {
		return ErrInvalidLimit
	}
	return nil
}

func listFromEvents(ctx context.Context, read []events.Event, query Query) ([]Summary, error) {
	states, order, err := collectSessions(ctx, read, "")
	if err != nil {
		return nil, err
	}

	ordered := orderedStates(states, order)
	summaries := make([]Summary, 0, len(ordered))
	for _, state := range ordered {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !matchesQuery(state.summary, query) {
			continue
		}
		summaries = append(summaries, state.summary)
	}

	if query.Limit > 0 && len(summaries) > query.Limit {
		summaries = summaries[len(summaries)-query.Limit:]
	}
	return summaries, nil
}

func detailFromEvents(ctx context.Context, read []events.Event, sessionID string) (*Detail, bool, error) {
	states, order, err := collectSessions(ctx, read, sessionID)
	if err != nil {
		return nil, false, err
	}
	if len(order) == 0 {
		return nil, false, nil
	}

	state := states[order[0]]
	detail := &Detail{
		Summary: state.summary,
		Events:  append([]events.Event(nil), state.events...),
	}
	return detail, true, nil
}

func collectSessions(ctx context.Context, read []events.Event, onlySessionID string) (map[string]*sessionState, []string, error) {
	states := map[string]*sessionState{}
	order := []string{}

	for index, event := range read {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		if event.SessionID == "" || (onlySessionID != "" && event.SessionID != onlySessionID) {
			continue
		}

		event = normalizeEvent(event)
		state, ok := states[event.SessionID]
		if !ok {
			state = &sessionState{
				summary: Summary{SessionID: event.SessionID},
				first:   index,
			}
			states[event.SessionID] = state
			order = append(order, event.SessionID)
		}
		state.add(event)
	}
	return states, order, nil
}

type sessionState struct {
	summary        Summary
	events         []events.Event
	first          int
	startedAtSet   bool
	startedFromRun bool
}

func (s *sessionState) add(event events.Event) {
	s.events = append(s.events, event)
	s.summary.EventCount++

	fillString(&s.summary.Workspace, event.Workspace)
	fillString(&s.summary.Agent, event.Agent)
	fillString(&s.summary.Profile, event.Profile)
	fillString(&s.summary.ProjectRoot, event.ProjectRoot)
	fillString(&s.summary.RuntimePath, event.RuntimePath)
	fillString(&s.summary.TaskID, event.TaskID)
	s.addStartedAt(event)
	s.addFinishedFields(event)
}

func (s *sessionState) addStartedAt(event events.Event) {
	if event.Timestamp.IsZero() {
		return
	}
	if event.Type == eventTypeRunStarted && !s.startedFromRun {
		s.summary.StartedAt = event.Timestamp
		s.startedAtSet = true
		s.startedFromRun = true
		return
	}
	if !s.startedAtSet {
		s.summary.StartedAt = event.Timestamp
		s.startedAtSet = true
	}
}

func (s *sessionState) addFinishedFields(event events.Event) {
	if event.ExitCode != nil {
		code := *event.ExitCode
		s.summary.ExitCode = &code
	}
	if event.DurationMillis != 0 || event.Type == eventTypeRunFinished {
		duration := event.DurationMillis
		s.summary.DurationMillis = &duration
	}
	if event.Type == eventTypeRunFinished && !event.Timestamp.IsZero() {
		s.summary.FinishedAt = event.Timestamp
	}
}

func fillString(target *string, value string) {
	if *target == "" && value != "" {
		*target = value
	}
}

func normalizeEvent(event events.Event) events.Event {
	if !event.Timestamp.IsZero() {
		event.Timestamp = event.Timestamp.UTC()
	}
	return event
}

func orderedStates(states map[string]*sessionState, order []string) []*sessionState {
	ordered := make([]*sessionState, 0, len(order))
	for _, sessionID := range order {
		ordered = append(ordered, states[sessionID])
	}

	sort.SliceStable(ordered, func(i, j int) bool {
		left := ordered[i]
		right := ordered[j]
		if !left.summary.StartedAt.IsZero() && !right.summary.StartedAt.IsZero() {
			if !left.summary.StartedAt.Equal(right.summary.StartedAt) {
				return left.summary.StartedAt.Before(right.summary.StartedAt)
			}
		}
		return left.first < right.first
	})
	return ordered
}

func matchesQuery(summary Summary, query Query) bool {
	if query.Workspace != "" && summary.Workspace != query.Workspace {
		return false
	}
	if query.Agent != "" && summary.Agent != query.Agent {
		return false
	}
	if query.TaskID != "" && summary.TaskID != query.TaskID {
		return false
	}
	return true
}
