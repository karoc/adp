package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"text/tabwriter"

	taskstore "github.com/karoc/adp/internal/tasks"
)

func (a *App) tasks(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: adp tasks <add|list|show|update|claim|release|done|block>")
	}

	switch args[0] {
	case "add":
		return a.tasksAdd(ctx, args[1:])
	case "list":
		return a.tasksList(ctx, args[1:])
	case "show":
		return a.tasksShow(ctx, args[1:])
	case "update":
		return a.tasksUpdate(ctx, args[1:])
	case "claim":
		return a.tasksClaim(ctx, args[1:])
	case "release":
		return a.tasksRelease(ctx, args[1:])
	case "done":
		return a.tasksDone(ctx, args[1:])
	case "block":
		return a.tasksBlock(ctx, args[1:])
	default:
		return fmt.Errorf("unknown tasks command %q", args[0])
	}
}

func (a *App) tasksAdd(ctx context.Context, args []string) error {
	opts, err := parseTasksAddArgs(args)
	if err != nil {
		return err
	}
	store, _, err := a.loadTaskStore(ctx, opts.workspace)
	if err != nil {
		return err
	}
	task, err := store.Add(ctx, taskstore.AddRequest{
		Title:       opts.title,
		Description: opts.description,
		Priority:    opts.priority,
		Phase:       opts.phase,
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "task %s added\n", task.ID)
	return nil
}

func (a *App) tasksList(ctx context.Context, args []string) error {
	opts, err := parseWorkspaceOutputArgs(args, "adp tasks list [--workspace <name>] [--format <text|json>]")
	if err != nil {
		return err
	}
	store, workspaceName, err := a.loadTaskStore(ctx, opts.workspace)
	if err != nil {
		return err
	}
	tasks, err := store.List(ctx)
	if err != nil {
		return err
	}
	if opts.format == outputFormatJSON {
		return writePlanningJSON(a.stdout, taskListOutput(workspaceName, tasks))
	}

	writer := tabwriter.NewWriter(a.stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "ID\tSTATUS\tOWNER\tPRIORITY\tPHASE\tUPDATED\tTITLE")
	for _, task := range tasks {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			task.ID,
			task.Status,
			valueOrDash(task.Owner),
			valueOrDash(task.Priority),
			valueOrDash(task.Phase),
			formatEventTime(task.UpdatedAt),
			task.Title,
		)
	}
	return writer.Flush()
}

func (a *App) tasksShow(ctx context.Context, args []string) error {
	opts, err := parseTaskIDOutputArgs(args, "adp tasks show [--workspace <name>] <task-id> [--format <text|json>]")
	if err != nil {
		return err
	}
	store, _, err := a.loadTaskStore(ctx, opts.workspace)
	if err != nil {
		return err
	}
	task, err := store.Get(ctx, opts.taskID)
	if err != nil {
		return err
	}
	if opts.format == outputFormatJSON {
		return writePlanningJSON(a.stdout, taskOutput(task))
	}
	a.printTask(task)
	return nil
}

func (a *App) tasksUpdate(ctx context.Context, args []string) error {
	opts, err := parseTasksUpdateArgs(args)
	if err != nil {
		return err
	}
	status, err := taskstore.ParseStatus(opts.status)
	if err != nil {
		return err
	}
	store, _, err := a.loadTaskStore(ctx, opts.workspace)
	if err != nil {
		return err
	}
	task, err := store.UpdateStatus(ctx, opts.taskID, status)
	if err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "task %s status: %s\n", task.ID, task.Status)
	return nil
}

func (a *App) tasksClaim(ctx context.Context, args []string) error {
	opts, err := parseTasksClaimArgs(args)
	if err != nil {
		return err
	}
	store, _, err := a.loadTaskStore(ctx, opts.workspace)
	if err != nil {
		return err
	}
	task, err := store.Claim(ctx, taskstore.ClaimRequest{
		TaskID: opts.taskID,
		Owner:  opts.owner,
		Lease:  opts.lease,
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "task %s claimed by %s\n", task.ID, task.Owner)
	return nil
}

func (a *App) tasksRelease(ctx context.Context, args []string) error {
	opts, err := parseTasksReleaseArgs(args)
	if err != nil {
		return err
	}
	store, _, err := a.loadTaskStore(ctx, opts.workspace)
	if err != nil {
		return err
	}
	task, err := store.Release(ctx, taskstore.ReleaseRequest{TaskID: opts.taskID, Owner: opts.owner})
	if err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "task %s released\n", task.ID)
	return nil
}

func (a *App) tasksDone(ctx context.Context, args []string) error {
	workspace, taskID, err := parseTaskIDArgs(args, "adp tasks done [--workspace <name>] <task-id>")
	if err != nil {
		return err
	}
	store, _, err := a.loadTaskStore(ctx, workspace)
	if err != nil {
		return err
	}
	task, err := store.UpdateStatus(ctx, taskID, taskstore.StatusDone)
	if err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "task %s done\n", task.ID)
	return nil
}

func (a *App) tasksBlock(ctx context.Context, args []string) error {
	opts, err := parseTasksBlockArgs(args)
	if err != nil {
		return err
	}
	store, _, err := a.loadTaskStore(ctx, opts.workspace)
	if err != nil {
		return err
	}
	task, err := store.Block(ctx, opts.taskID, opts.reason)
	if err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "task %s blocked\n", task.ID)
	return nil
}

func (a *App) progress(ctx context.Context, args []string) error {
	opts, err := parseWorkspaceOutputArgs(args, "adp progress [--workspace <name>] [--format <text|json>]")
	if err != nil {
		return err
	}
	store, workspaceName, err := a.loadTaskStore(ctx, opts.workspace)
	if err != nil {
		return err
	}
	progress, err := store.Progress(ctx)
	if err != nil {
		return err
	}
	phases, err := store.ListPhases(ctx)
	if err != nil {
		return err
	}
	if opts.format == outputFormatJSON {
		return writePlanningJSON(a.stdout, progressOutput(workspaceName, progress, phases))
	}

	fmt.Fprintf(a.stdout, "workspace: %s\n", workspaceName)
	if len(phases) == 0 {
		fmt.Fprintln(a.stdout, "phases: -")
	} else {
		fmt.Fprintln(a.stdout, "phases:")
		for _, phase := range phases {
			fmt.Fprintf(a.stdout, "- %s [%s] %s\n", phase.ID, phase.Status, phase.Title)
		}
	}
	fmt.Fprintf(a.stdout, "total: %d\n", progress.Total)
	writer := tabwriter.NewWriter(a.stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "STATUS\tCOUNT")
	for _, status := range taskstore.Statuses() {
		fmt.Fprintf(writer, "%s\t%d\n", status, progress.Counts[status])
	}
	if err := writer.Flush(); err != nil {
		return err
	}
	if len(progress.Next) == 0 {
		fmt.Fprintln(a.stdout, "next: -")
		return nil
	}
	fmt.Fprintln(a.stdout, "next:")
	for _, task := range progress.Next {
		fmt.Fprintf(a.stdout, "- %s [%s] %s\n", task.ID, task.Status, task.Title)
	}
	return nil
}

func (a *App) loadTaskStore(ctx context.Context, workspace string) (TaskStore, string, error) {
	cfg, workspaceDir, err := a.loadWorkspace(ctx, workspace)
	if err != nil {
		return nil, "", err
	}
	if a.deps.TaskStoreFactory == nil {
		return nil, "", errors.New("task store is not configured")
	}
	return a.deps.TaskStoreFactory(workspaceDir), cfg.Workspace.Name, nil
}

func (a *App) printTask(task taskstore.Task) {
	fmt.Fprintf(a.stdout, "id: %s\n", task.ID)
	fmt.Fprintf(a.stdout, "title: %s\n", task.Title)
	fmt.Fprintf(a.stdout, "status: %s\n", task.Status)
	fmt.Fprintf(a.stdout, "priority: %s\n", valueOrDash(task.Priority))
	fmt.Fprintf(a.stdout, "phase: %s\n", valueOrDash(task.Phase))
	fmt.Fprintf(a.stdout, "owner: %s\n", valueOrDash(task.Owner))
	fmt.Fprintf(a.stdout, "claimed_at: %s\n", formatEventTime(task.ClaimedAt))
	fmt.Fprintf(a.stdout, "lease_expires_at: %s\n", formatEventTime(task.LeaseExpiresAt))
	fmt.Fprintf(a.stdout, "description: %s\n", valueOrDash(task.Description))
	fmt.Fprintf(a.stdout, "blocked_reason: %s\n", valueOrDash(task.BlockedReason))
	fmt.Fprintf(a.stdout, "created_at: %s\n", formatEventTime(task.CreatedAt))
	fmt.Fprintf(a.stdout, "updated_at: %s\n", formatEventTime(task.UpdatedAt))
}

func joinTitle(parts []string) string {
	return strings.TrimSpace(strings.Join(parts, " "))
}
