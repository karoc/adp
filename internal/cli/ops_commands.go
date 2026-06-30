package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/karoc/adp/internal/events"
	"github.com/karoc/adp/internal/sessions"
	taskstore "github.com/karoc/adp/internal/tasks"
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
	if len(args) > 0 && args[0] == "values" {
		return a.completionValues(ctx, args[1:])
	}
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("unknown completion command %q", args[0])
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

	// Resolve task ID prefix if provided
	resolvedTaskID := opts.taskID
	if opts.taskID != "" && opts.workspace != "" {
		store, _, err := a.loadTaskStore(ctx, opts.workspace)
		if err == nil {
			tasks, err := store.FindByPrefix(ctx, opts.taskID)
			if err != nil {
				if errors.Is(err, taskstore.ErrAmbiguousTaskID) {
					ids := make([]string, len(tasks))
					for i, task := range tasks {
						ids[i] = task.ID
					}
					return fmt.Errorf("adp: ambiguous task ID %q, matches multiple tasks:\n  - %s\n\nPlease use a more specific prefix.", opts.taskID, strings.Join(ids, "\n  - "))
				}
				// If task not found, use the original ID as-is
				if !errors.Is(err, taskstore.ErrTaskNotFound) {
					return err
				}
			} else if len(tasks) == 1 {
				resolvedTaskID = tasks[0].ID
			}
		}
	}

	// Resolve session ID prefix if provided
	resolvedSessionID := opts.sessionID
	if opts.sessionID != "" && a.deps.Layout.Home != "" {
		summaries, err := sessions.FindByPrefix(ctx, a.deps.Layout, opts.sessionID)
		if err != nil {
			if errors.Is(err, sessions.ErrAmbiguousSessionID) {
				ids := extractSessionIDs(summaries)
				return fmt.Errorf("adp: ambiguous session ID %q, matches multiple sessions:\n%s\n\nPlease use a more specific prefix.", opts.sessionID, formatSessionIDList(ids))
			}
			// If session not found, use the original ID as-is
			if !errors.Is(err, sessions.ErrSessionNotFound) {
				return err
			}
		} else if len(summaries) == 1 {
			resolvedSessionID = summaries[0].SessionID
		}
	}

	read, err := a.deps.ReadEvents(ctx, a.deps.Layout, events.Query{
		Workspace: opts.workspace,
		SessionID: resolvedSessionID,
		TaskID:    resolvedTaskID,
		Type:      opts.eventType,
		Limit:     opts.limit,
	})
	if err != nil {
		return err
	}
	if opts.format == outputFormatJSON {
		return writePlanningJSON(a.stdout, eventsListOutput(opts, read))
	}

	writer := tabwriter.NewWriter(a.stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "TIME\tTYPE\tWORKSPACE\tAGENT\tSESSION\tTASK\tEXIT\tRUNTIME")
	for _, event := range read {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			formatEventTime(event.Timestamp),
			valueOrDash(event.Type),
			valueOrDash(event.Workspace),
			valueOrDash(event.Agent),
			valueOrDash(event.SessionID),
			valueOrDash(event.TaskID),
			formatExitCode(event.ExitCode),
			valueOrDash(event.RuntimePath),
		)
	}
	if err := writer.Flush(); err != nil {
		return err
	}
	if len(read) == 0 {
		fmt.Fprintln(a.stdout, "\nNo events recorded yet. Events are created when you run agents with 'adp run'")
	}
	return nil
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

func valueOrDash(value string) string {
	if value == "" {
		return "-"
	}
	return safeText(value)
}

func formatStringList(values []string) string {
	if len(values) == 0 {
		return "-"
	}
	sanitized := make([]string, len(values))
	for i, value := range values {
		sanitized[i] = safeText(value)
	}
	return strings.Join(sanitized, "; ")
}

func formatSuggestedCommand(args []string) string {
	if len(args) == 0 {
		return "-"
	}
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, shellQuoteArg(arg))
	}
	return strings.Join(quoted, " ")
}

func shellQuoteArg(arg string) string {
	if arg == "" {
		return "''"
	}
	if strings.IndexFunc(arg, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '_' || r == '-' || r == '.' || r == '/' || r == ':' || r == '=')
	}) == -1 {
		return arg
	}
	return "'" + strings.ReplaceAll(arg, "'", "'\"'\"'") + "'"
}
