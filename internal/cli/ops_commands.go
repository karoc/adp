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
