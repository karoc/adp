package cli

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/karoc/adp/internal/sessions"
	taskstore "github.com/karoc/adp/internal/tasks"
	"github.com/karoc/adp/internal/workspace"
)

func (a *App) completionValues(ctx context.Context, args []string) error {
	opts, err := parseCompletionValuesArgs(args)
	if err != nil {
		return err
	}

	var values []string
	switch opts.kind {
	case "agents":
		values, err = a.completionAgentValues()
	case "workspaces":
		values, err = a.completionWorkspaceValues(ctx)
	case "profiles":
		values, err = a.completionProfileValues(ctx, opts.workspace)
	case "tasks":
		values, err = a.completionTaskValues(ctx, opts.workspace)
	case "phases":
		values, err = a.completionPhaseValues(ctx, opts.workspace)
	case "sessions":
		values, err = a.completionSessionValues(ctx, opts.workspace)
	case "owners":
		values, err = a.completionOwnerValues(ctx, opts.workspace)
	case "statuses":
		values = completionStatusValues()
	default:
		return fmt.Errorf("unknown completion values kind %q", opts.kind)
	}
	if err != nil {
		return err
	}
	for _, value := range values {
		fmt.Fprintln(a.stdout, value)
	}
	return nil
}

func (a *App) completionAgentValues() ([]string, error) {
	if a.deps.Adapters == nil {
		return nil, errors.New("adapter registry is not configured")
	}
	return a.deps.Adapters.Names(), nil
}

func (a *App) completionWorkspaceValues(ctx context.Context) ([]string, error) {
	if a.deps.WorkspaceStore == nil {
		return nil, errors.New("workspace store is not configured")
	}
	return a.deps.WorkspaceStore.Names(ctx)
}

func (a *App) completionProfileValues(ctx context.Context, workspaceName string) ([]string, error) {
	cfg, workspaceDir, err := a.loadWorkspace(ctx, workspaceName)
	if err != nil {
		return nil, err
	}
	return workspace.ListProfiles(ctx, workspaceDir, *cfg)
}

func (a *App) completionTaskValues(ctx context.Context, workspaceName string) ([]string, error) {
	store, _, err := a.loadTaskStore(ctx, workspaceName)
	if err != nil {
		return nil, err
	}
	tasks, err := store.List(ctx)
	if err != nil {
		return nil, err
	}
	values := make([]string, 0, len(tasks))
	for _, task := range tasks {
		values = append(values, task.ID)
	}
	sort.Strings(values)
	return values, nil
}

func (a *App) completionPhaseValues(ctx context.Context, workspaceName string) ([]string, error) {
	store, _, err := a.loadTaskStore(ctx, workspaceName)
	if err != nil {
		return nil, err
	}
	phases, err := store.ListPhases(ctx)
	if err != nil {
		return nil, err
	}
	values := make([]string, 0, len(phases))
	for _, phase := range phases {
		values = append(values, phase.ID)
	}
	sort.Strings(values)
	return values, nil
}

func (a *App) completionSessionValues(ctx context.Context, workspaceName string) ([]string, error) {
	if a.deps.ListSessions == nil {
		return nil, errors.New("session lister is not configured")
	}
	summaries, err := a.deps.ListSessions(ctx, a.deps.Layout, sessions.Query{Workspace: workspaceName})
	if err != nil {
		return nil, err
	}
	values := make([]string, 0, len(summaries))
	for _, summary := range summaries {
		values = append(values, summary.SessionID)
	}
	return uniqueSorted(values), nil
}

func (a *App) completionOwnerValues(ctx context.Context, workspaceName string) ([]string, error) {
	store, _, err := a.loadTaskStore(ctx, workspaceName)
	if err != nil {
		return nil, err
	}
	tasks, err := store.List(ctx)
	if err != nil {
		return nil, err
	}
	values := make([]string, 0, len(tasks))
	for _, task := range tasks {
		values = append(values, task.Owner)
	}
	return uniqueSorted(values), nil
}

func completionStatusValues() []string {
	statuses := taskstore.Statuses()
	values := make([]string, 0, len(statuses))
	for _, status := range statuses {
		values = append(values, string(status))
	}
	return values
}

func uniqueSorted(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
