package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/karoc/adp/internal/workspace"
)

func (a *App) completionValues(ctx context.Context, args []string) error {
	opts, err := parseCompletionValuesArgs(args)
	if err != nil {
		return err
	}

	var values []string
	switch opts.kind {
	case "workspaces":
		values, err = a.completionWorkspaceValues(ctx)
	case "profiles":
		values, err = a.completionProfileValues(ctx, opts.workspace)
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
