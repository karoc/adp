package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"
	"time"

	"github.com/karoc/adp/internal/events"
	"github.com/karoc/adp/internal/runtime"
	"github.com/karoc/adp/internal/sessions"
)

func (a *App) shellHook(ctx context.Context, args []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	opts, err := parseShellHookArgs(args)
	if err != nil {
		return err
	}
	if opts.Shell == "" {
		opts.Shell = os.Getenv("SHELL")
	}

	renderHook := a.deps.RenderHook
	if renderHook == nil {
		return errors.New("shell hook renderer is not configured")
	}
	output, err := renderHook(opts)
	if err != nil {
		return err
	}
	fmt.Fprint(a.stdout, output)
	return nil
}

func (a *App) completion(ctx context.Context, args []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	opts, err := parseCompletionArgs(args)
	if err != nil {
		return err
	}
	if opts.Shell == "" {
		opts.Shell = "bash"
	}
	if a.deps.RenderCompletion == nil {
		return errors.New("completion renderer is not configured")
	}
	output, err := a.deps.RenderCompletion(opts)
	if err != nil {
		return err
	}
	fmt.Fprint(a.stdout, output)
	return nil
}

func (a *App) events(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: adp events <list>")
	}

	switch args[0] {
	case "list":
		return a.eventsList(ctx, args[1:])
	default:
		return fmt.Errorf("unknown events command %q", args[0])
	}
}

func (a *App) eventsList(ctx context.Context, args []string) error {
	opts, err := parseEventsListArgs(args)
	if err != nil {
		return err
	}
	if a.deps.ReadEvents == nil {
		return errors.New("event reader is not configured")
	}

	read, err := a.deps.ReadEvents(ctx, a.deps.Layout, events.Query{
		Workspace: opts.workspace,
		SessionID: opts.sessionID,
		Type:      opts.eventType,
		Limit:     opts.limit,
	})
	if err != nil {
		return err
	}

	writer := tabwriter.NewWriter(a.stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "TIME\tTYPE\tWORKSPACE\tAGENT\tSESSION\tEXIT\tRUNTIME")
	for _, event := range read {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			formatEventTime(event.Timestamp),
			valueOrDash(event.Type),
			valueOrDash(event.Workspace),
			valueOrDash(event.Agent),
			valueOrDash(event.SessionID),
			formatExitCode(event.ExitCode),
			valueOrDash(event.RuntimePath),
		)
	}
	return writer.Flush()
}

func (a *App) sessions(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: adp sessions <list|show>")
	}

	switch args[0] {
	case "list":
		return a.sessionsList(ctx, args[1:])
	case "show":
		return a.sessionsShow(ctx, args[1:])
	default:
		return fmt.Errorf("unknown sessions command %q", args[0])
	}
}

func (a *App) sessionsList(ctx context.Context, args []string) error {
	opts, err := parseSessionsListArgs(args)
	if err != nil {
		return err
	}
	if a.deps.ListSessions == nil {
		return errors.New("session lister is not configured")
	}

	summaries, err := a.deps.ListSessions(ctx, a.deps.Layout, sessions.Query{
		Workspace: opts.workspace,
		Agent:     opts.agent,
		Limit:     opts.limit,
	})
	if err != nil {
		return err
	}

	writer := tabwriter.NewWriter(a.stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "SESSION\tWORKSPACE\tAGENT\tPROFILE\tSTARTED\tFINISHED\tEXIT\tDURATION\tEVENTS\tRUNTIME")
	for _, summary := range summaries {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%d\t%s\n",
			valueOrDash(summary.SessionID),
			valueOrDash(summary.Workspace),
			valueOrDash(summary.Agent),
			valueOrDash(summary.Profile),
			formatEventTime(summary.StartedAt),
			formatEventTime(summary.FinishedAt),
			formatExitCode(summary.ExitCode),
			formatDurationMillis(summary.DurationMillis),
			summary.EventCount,
			valueOrDash(summary.RuntimePath),
		)
	}
	return writer.Flush()
}

func (a *App) sessionsShow(ctx context.Context, args []string) error {
	sessionID, err := parseSessionsShowArgs(args)
	if err != nil {
		return err
	}
	if a.deps.GetSession == nil {
		return errors.New("session reader is not configured")
	}

	detail, err := a.deps.GetSession(ctx, a.deps.Layout, sessionID)
	if err != nil {
		return err
	}

	summary := detail.Summary
	fmt.Fprintf(a.stdout, "session_id: %s\n", valueOrDash(summary.SessionID))
	fmt.Fprintf(a.stdout, "workspace: %s\n", valueOrDash(summary.Workspace))
	fmt.Fprintf(a.stdout, "agent: %s\n", valueOrDash(summary.Agent))
	fmt.Fprintf(a.stdout, "profile: %s\n", valueOrDash(summary.Profile))
	fmt.Fprintf(a.stdout, "project_root: %s\n", valueOrDash(summary.ProjectRoot))
	fmt.Fprintf(a.stdout, "runtime_path: %s\n", valueOrDash(summary.RuntimePath))
	fmt.Fprintf(a.stdout, "started_at: %s\n", formatEventTime(summary.StartedAt))
	fmt.Fprintf(a.stdout, "finished_at: %s\n", formatEventTime(summary.FinishedAt))
	fmt.Fprintf(a.stdout, "exit_code: %s\n", formatExitCode(summary.ExitCode))
	fmt.Fprintf(a.stdout, "duration_ms: %s\n", formatDurationMillis(summary.DurationMillis))
	fmt.Fprintf(a.stdout, "event_count: %d\n", summary.EventCount)

	writer := tabwriter.NewWriter(a.stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "TIME\tTYPE\tWORKSPACE\tAGENT\tEXIT\tRUNTIME")
	for _, event := range detail.Events {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\n",
			formatEventTime(event.Timestamp),
			valueOrDash(event.Type),
			valueOrDash(event.Workspace),
			valueOrDash(event.Agent),
			formatExitCode(event.ExitCode),
			valueOrDash(event.RuntimePath),
		)
	}
	return writer.Flush()
}

func (a *App) runtime(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: adp runtime <prune>")
	}

	switch args[0] {
	case "prune":
		return a.runtimePrune(ctx, args[1:])
	default:
		return fmt.Errorf("unknown runtime command %q", args[0])
	}
}

func (a *App) runtimePrune(ctx context.Context, args []string) error {
	opts, err := parseRuntimePruneArgs(args)
	if err != nil {
		return err
	}
	if a.deps.PruneRuntimes == nil {
		return errors.New("runtime pruner is not configured")
	}

	results, err := a.deps.PruneRuntimes(ctx, runtime.PruneRequest{
		Layout:      a.deps.Layout,
		OlderThan:   opts.olderThan,
		IncludeKept: opts.includeKept,
		DryRun:      opts.dryRun,
	})
	if err != nil {
		return err
	}

	writer := tabwriter.NewWriter(a.stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "ACTION\tWORKSPACE\tSESSION\tCREATED AT\tKEEP\tRUNTIME ROOT")
	for _, result := range results {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%t\t%s\n",
			formatPruneAction(result),
			valueOrDash(result.Workspace),
			valueOrDash(result.SessionID),
			formatEventTime(result.CreatedAt),
			result.Keep,
			valueOrDash(result.Root),
		)
	}
	return writer.Flush()
}

func formatEventTime(ts time.Time) string {
	if ts.IsZero() {
		return "-"
	}
	return ts.UTC().Format(time.RFC3339)
}

func formatExitCode(code *int) string {
	if code == nil {
		return "-"
	}
	return strconv.Itoa(*code)
}

func formatDurationMillis(duration *int64) string {
	if duration == nil {
		return "-"
	}
	return strconv.FormatInt(*duration, 10)
}

func formatPruneAction(result runtime.PruneResult) string {
	if result.DryRun {
		return "would-remove"
	}
	if result.Removed {
		return "removed"
	}
	return "matched"
}

func valueOrDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
