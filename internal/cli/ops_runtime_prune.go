package cli

import (
	"context"
	"errors"
	"fmt"
	"text/tabwriter"

	"github.com/karoc/adp/internal/output"
	"github.com/karoc/adp/internal/runtime"
)

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

	// Confirm dangerous operation when including kept runtimes (unless dry-run)
	if opts.includeKept && !opts.dryRun {
		operation := "Remove kept runtime directories?"
		details := "This will delete runtime directories marked as 'keep', which may contain\nimportant agent session state. This operation cannot be undone."
		if err := a.confirmDangerous(operation, details, opts.yes); err != nil {
			return err
		}
	}

	// Show progress indicator for prune operation (skip for JSON output)
	var spinner *output.Spinner
	if opts.format != outputFormatJSON {
		message := "Scanning runtime directories..."
		if opts.dryRun {
			message = "Scanning runtime directories (dry run)..."
		}
		spinner = output.NewSpinner(a.stderr, message)
		spinner.Start()
	}

	results, err := a.deps.PruneRuntimes(ctx, runtime.PruneRequest{
		Layout:      a.deps.Layout,
		OlderThan:   opts.olderThan,
		IncludeKept: opts.includeKept,
		DryRun:      opts.dryRun,
	})

	if err != nil {
		if spinner != nil {
			spinner.Fail("")
		}
		return err
	}

	if spinner != nil {
		spinner.Stop()
	}

	if opts.format == outputFormatJSON {
		return writePlanningJSON(a.stdout, runtimePruneOutput(opts, results))
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

func formatPruneAction(result runtime.PruneResult) string {
	if result.DryRun {
		return "would-remove"
	}
	if result.Removed {
		return "removed"
	}
	return "matched"
}
