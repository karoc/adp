package cli

import (
	"context"
	"errors"
	"fmt"
	"text/tabwriter"
)

func (a *App) init(ctx context.Context, args []string) error {
	if len(args) != 0 {
		return errors.New("usage: adp init")
	}
	if a.deps.WorkspaceStore == nil {
		return errors.New("workspace store is not configured")
	}
	if err := a.deps.WorkspaceStore.Init(ctx); err != nil {
		return err
	}
	fmt.Fprintln(a.stdout, "initialized ADP home")
	return nil
}

func (a *App) workspace(ctx context.Context, args []string) error {
	if a.deps.WorkspaceStore == nil {
		return errors.New("workspace store is not configured")
	}
	if len(args) == 0 {
		return errors.New("usage: adp workspace <add|list|show|remove|rename>")
	}

	switch args[0] {
	case "add":
		if len(args) != 3 {
			return errors.New("usage: adp workspace add <name> <project-root>")
		}
		if _, err := a.deps.WorkspaceStore.Add(ctx, args[1], args[2]); err != nil {
			return err
		}
		fmt.Fprintf(a.stdout, "workspace %q added\n", args[1])
	case "list":
		if len(args) != 1 {
			return errors.New("usage: adp workspace list")
		}
		return a.workspaceList(ctx)
	case "show":
		if len(args) != 2 {
			return errors.New("usage: adp workspace show <name>")
		}
		return a.workspaceShow(ctx, args[1])
	case "remove":
		if len(args) != 2 {
			return errors.New("usage: adp workspace remove <name>")
		}
		if err := a.deps.WorkspaceStore.Remove(ctx, args[1]); err != nil {
			return err
		}
		fmt.Fprintf(a.stdout, "workspace %q removed\n", args[1])
	case "rename":
		if len(args) != 3 {
			return errors.New("usage: adp workspace rename <old-name> <new-name>")
		}
		if _, err := a.deps.WorkspaceStore.Rename(ctx, args[1], args[2]); err != nil {
			return err
		}
		fmt.Fprintf(a.stdout, "workspace %q renamed to %q\n", args[1], args[2])
	default:
		return fmt.Errorf("unknown workspace command %q", args[0])
	}
	return nil
}

func (a *App) workspaceList(ctx context.Context) error {
	records, err := a.deps.WorkspaceStore.List(ctx)
	if err != nil {
		return err
	}

	writer := tabwriter.NewWriter(a.stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "NAME\tPROJECT ROOT\tWORKSPACE DIR")
	for _, record := range records {
		fmt.Fprintf(writer, "%s\t%s\t%s\n", record.Name, record.ProjectRoot, record.WorkspaceDir)
	}
	return writer.Flush()
}

func (a *App) workspaceShow(ctx context.Context, name string) error {
	cfg, workspaceDir, err := a.deps.WorkspaceStore.Get(ctx, name)
	if err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "name: %s\n", cfg.Workspace.Name)
	fmt.Fprintf(a.stdout, "project_root: %s\n", cfg.Project.Root)
	fmt.Fprintf(a.stdout, "workspace_dir: %s\n", workspaceDir)
	fmt.Fprintf(a.stdout, "memory_enabled: %t\n", cfg.Memory.Enabled)
	fmt.Fprintf(a.stdout, "mcp_enabled: %t\n", cfg.MCP.Enabled)
	return nil
}
