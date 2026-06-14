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
	"github.com/karoc/adp/internal/output"
	adpresume "github.com/karoc/adp/internal/resume"
	"github.com/karoc/adp/internal/runtime"
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

func (a *App) sessions(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: adp sessions <list|show|restore-plan|resume-plan>")
	}

	switch args[0] {
	case "list":
		return a.sessionsList(ctx, args[1:])
	case "show":
		return a.sessionsShow(ctx, args[1:])
	case "restore-plan":
		return a.sessionsRestorePlan(ctx, args[1:])
	case "resume-plan":
		return a.sessionsResumePlan(ctx, args[1:])
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
		TaskID:    opts.taskID,
		Limit:     opts.limit,
	})
	if err != nil {
		return err
	}
	if opts.format == outputFormatJSON {
		return writePlanningJSON(a.stdout, sessionsListOutput(opts, summaries))
	}

	writer := tabwriter.NewWriter(a.stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "SESSION\tWORKSPACE\tAGENT\tPROFILE\tTASK\tSTARTED\tFINISHED\tEXIT\tDURATION\tEVENTS\tRUNTIME")
	for _, summary := range summaries {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%d\t%s\n",
			valueOrDash(summary.SessionID),
			valueOrDash(summary.Workspace),
			valueOrDash(summary.Agent),
			valueOrDash(summary.Profile),
			valueOrDash(summary.TaskID),
			formatEventTime(summary.StartedAt),
			formatEventTime(summary.FinishedAt),
			formatExitCode(summary.ExitCode),
			formatDurationMillis(summary.DurationMillis),
			summary.EventCount,
			valueOrDash(summary.RuntimePath),
		)
	}
	if err := writer.Flush(); err != nil {
		return err
	}
	if len(summaries) == 0 {
		fmt.Fprintln(a.stdout, "\nNo sessions found. Start an agent with 'adp run <agent> --workspace <name>'")
	}
	return nil
}

func (a *App) sessionsShow(ctx context.Context, args []string) error {
	opts, err := parseSessionsShowArgs(args)
	if err != nil {
		return err
	}
	if a.deps.GetSession == nil {
		return errors.New("session reader is not configured")
	}

	// Resolve session ID prefix
	summaries, err := sessions.FindByPrefix(ctx, a.deps.Layout, opts.sessionID)
	if err != nil {
		if errors.Is(err, sessions.ErrAmbiguousSessionID) {
			ids := extractSessionIDs(summaries)
			return fmt.Errorf("adp: ambiguous session ID %q, matches multiple sessions:\n%s\n\nPlease use a more specific prefix.", opts.sessionID, formatSessionIDList(ids))
		}
		return err
	}
	if len(summaries) != 1 {
		return fmt.Errorf("session %q not found", opts.sessionID)
	}
	resolvedSessionID := summaries[0].SessionID

	detail, err := a.deps.GetSession(ctx, a.deps.Layout, resolvedSessionID)
	if err != nil {
		return err
	}
	if opts.format == outputFormatJSON {
		return writePlanningJSON(a.stdout, sessionDetailOutput(detail))
	}

	summary := detail.Summary
	fmt.Fprintf(a.stdout, "session_id: %s\n", valueOrDash(summary.SessionID))
	fmt.Fprintf(a.stdout, "workspace: %s\n", valueOrDash(summary.Workspace))
	fmt.Fprintf(a.stdout, "agent: %s\n", valueOrDash(summary.Agent))
	fmt.Fprintf(a.stdout, "profile: %s\n", valueOrDash(summary.Profile))
	fmt.Fprintf(a.stdout, "task_id: %s\n", valueOrDash(summary.TaskID))
	fmt.Fprintf(a.stdout, "project_root: %s\n", valueOrDash(summary.ProjectRoot))
	fmt.Fprintf(a.stdout, "runtime_path: %s\n", valueOrDash(summary.RuntimePath))
	fmt.Fprintf(a.stdout, "started_at: %s\n", formatEventTime(summary.StartedAt))
	fmt.Fprintf(a.stdout, "finished_at: %s\n", formatEventTime(summary.FinishedAt))
	fmt.Fprintf(a.stdout, "exit_code: %s\n", formatExitCode(summary.ExitCode))
	fmt.Fprintf(a.stdout, "duration_ms: %s\n", formatDurationMillis(summary.DurationMillis))
	fmt.Fprintf(a.stdout, "event_count: %d\n", summary.EventCount)

	writer := tabwriter.NewWriter(a.stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "TIME\tTYPE\tWORKSPACE\tAGENT\tTASK\tEXIT\tRUNTIME")
	for _, event := range detail.Events {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			formatEventTime(event.Timestamp),
			valueOrDash(event.Type),
			valueOrDash(event.Workspace),
			valueOrDash(event.Agent),
			valueOrDash(event.TaskID),
			formatExitCode(event.ExitCode),
			valueOrDash(event.RuntimePath),
		)
	}
	return writer.Flush()
}

func (a *App) sessionsRestorePlan(ctx context.Context, args []string) error {
	opts, err := parseSessionsRestorePlanArgs(args)
	if err != nil {
		return err
	}
	if a.deps.GetSession == nil {
		return errors.New("session reader is not configured")
	}

	// Resolve session ID prefix
	summaries, err := sessions.FindByPrefix(ctx, a.deps.Layout, opts.sessionID)
	if err != nil {
		if errors.Is(err, sessions.ErrAmbiguousSessionID) {
			ids := extractSessionIDs(summaries)
			return fmt.Errorf("adp: ambiguous session ID %q, matches multiple sessions:\n%s\n\nPlease use a more specific prefix.", opts.sessionID, formatSessionIDList(ids))
		}
		return err
	}
	if len(summaries) != 1 {
		return fmt.Errorf("session %q not found", opts.sessionID)
	}
	resolvedSessionID := summaries[0].SessionID

	detail, err := a.deps.GetSession(ctx, a.deps.Layout, resolvedSessionID)
	if err != nil {
		return err
	}

	plan := sessions.BuildRestorePlan(detail)
	if opts.format == outputFormatJSON {
		return writePlanningJSON(a.stdout, plan)
	}
	fmt.Fprintf(a.stdout, "session_id: %s\n", valueOrDash(plan.SessionID))
	fmt.Fprintf(a.stdout, "status: %s\n", valueOrDash(plan.Status))
	fmt.Fprintf(a.stdout, "suggested_command: %s\n", formatSuggestedCommand(plan.SuggestedCommand))
	fmt.Fprintf(a.stdout, "missing_fields: %s\n", formatStringList(plan.MissingFields))
	fmt.Fprintf(a.stdout, "reasons: %s\n", formatStringList(plan.Reasons))
	return nil
}

func (a *App) sessionsResumePlan(ctx context.Context, args []string) error {
	opts, err := parseSessionsResumePlanArgs(args)
	if err != nil {
		return err
	}
	if a.deps.GetSession == nil {
		return errors.New("session reader is not configured")
	}

	// Resolve session ID prefix
	summaries, err := sessions.FindByPrefix(ctx, a.deps.Layout, opts.sessionID)
	if err != nil {
		if errors.Is(err, sessions.ErrAmbiguousSessionID) {
			ids := extractSessionIDs(summaries)
			return fmt.Errorf("adp: ambiguous session ID %q, matches multiple sessions:\n%s\n\nPlease use a more specific prefix.", opts.sessionID, formatSessionIDList(ids))
		}
		return err
	}
	if len(summaries) != 1 {
		return fmt.Errorf("session %q not found", opts.sessionID)
	}
	resolvedSessionID := summaries[0].SessionID

	detail, err := a.deps.GetSession(ctx, a.deps.Layout, resolvedSessionID)
	if err != nil {
		return err
	}
	workspaceName := opts.workspace
	if workspaceName == "" {
		workspaceName = detail.Summary.Workspace
	}
	req := adpresume.Request{
		Detail:      detail,
		Workspace:   workspaceName,
		TargetAgent: opts.targetAgent,
		Owner:       opts.owner,
		Lease:       opts.lease,
		Now:         time.Now().UTC(),
	}
	a.fillResumePlanState(ctx, &req)
	plan := adpresume.BuildPlan(req)
	if opts.format == outputFormatJSON {
		return writePlanningJSON(a.stdout, plan)
	}
	a.printResumePlan(plan)
	return nil
}

func (a *App) fillResumePlanState(ctx context.Context, req *adpresume.Request) {
	if req == nil || req.Workspace == "" {
		return
	}
	store, _, err := a.loadTaskStore(ctx, req.Workspace)
	if err != nil {
		req.TaskLoadError = err.Error()
		req.PhaseLoadError = err.Error()
		return
	}
	if req.Detail != nil && req.Detail.Summary.TaskID != "" {
		task, err := store.Get(ctx, req.Detail.Summary.TaskID)
		if err != nil {
			req.TaskLoadError = err.Error()
		} else {
			req.Task = &task
		}
	}
	phases, err := store.ListPhases(ctx)
	if err != nil {
		req.PhaseLoadError = err.Error()
		return
	}
	gate := taskstore.PhaseGateStatus(phases)
	req.PhaseGate = &gate
}

func (a *App) printResumePlan(plan adpresume.Plan) {
	fmt.Fprintf(a.stdout, "session_id: %s\n", valueOrDash(plan.SessionID))
	fmt.Fprintf(a.stdout, "status: %s\n", valueOrDash(plan.Status))
	fmt.Fprintf(a.stdout, "summary: %s\n", valueOrDash(plan.Summary))
	fmt.Fprintf(a.stdout, "source_workspace: %s\n", valueOrDash(plan.Source.Workspace))
	fmt.Fprintf(a.stdout, "source_agent: %s\n", valueOrDash(plan.Source.Agent))
	fmt.Fprintf(a.stdout, "source_profile: %s\n", valueOrDash(plan.Source.Profile))
	fmt.Fprintf(a.stdout, "target_workspace: %s\n", valueOrDash(plan.Target.Workspace))
	fmt.Fprintf(a.stdout, "target_agent: %s\n", valueOrDash(plan.Target.Agent))
	fmt.Fprintf(a.stdout, "target_profile: %s\n", valueOrDash(plan.Target.Profile))
	fmt.Fprintf(a.stdout, "target_owner: %s\n", valueOrDash(plan.Target.Owner))
	fmt.Fprintf(a.stdout, "target_lease: %s\n", valueOrDash(plan.Target.Lease))
	if plan.Invocation != nil {
		fmt.Fprintf(a.stdout, "invocation_available: %t\n", plan.Invocation.Available)
		fmt.Fprintf(a.stdout, "invocation_keep_runtime: %t\n", plan.Invocation.KeepRuntime)
		fmt.Fprintf(a.stdout, "invocation_reused: %s\n", formatStringList(plan.Invocation.Reused))
		fmt.Fprintf(a.stdout, "invocation_omitted: %s\n", formatStringList(plan.Invocation.Omitted))
		if plan.Invocation.OmissionReason != "" {
			fmt.Fprintf(a.stdout, "invocation_omission_reason: %s\n", plan.Invocation.OmissionReason)
		}
	} else {
		fmt.Fprintln(a.stdout, "invocation_available: false")
	}
	if plan.Task != nil {
		fmt.Fprintf(a.stdout, "task_id: %s\n", valueOrDash(plan.Task.ID))
		fmt.Fprintf(a.stdout, "task_status: %s\n", valueOrDash(plan.Task.Status))
		fmt.Fprintf(a.stdout, "task_owner: %s\n", valueOrDash(plan.Task.Owner))
		fmt.Fprintf(a.stdout, "task_claim_state: %s\n", valueOrDash(plan.Task.ClaimState))
		fmt.Fprintf(a.stdout, "task_resume_action: %s\n", valueOrDash(plan.Task.ResumeAction))
		fmt.Fprintf(a.stdout, "task_reason: %s\n", valueOrDash(plan.Task.Reason))
	} else {
		fmt.Fprintln(a.stdout, "task_id: -")
	}
	if plan.Phase != nil {
		fmt.Fprintf(a.stdout, "phase_next_action: %s\n", valueOrDash(plan.Phase.NextAction))
		fmt.Fprintf(a.stdout, "phase_reason: %s\n", valueOrDash(plan.Phase.Reason))
	} else {
		fmt.Fprintln(a.stdout, "phase_next_action: -")
	}
	fmt.Fprintf(a.stdout, "missing_fields: %s\n", formatStringList(plan.MissingFields))
	fmt.Fprintf(a.stdout, "read_only: %t\n", plan.Guarantees.ReadOnly)
	fmt.Fprintln(a.stdout, "guidance:")
	for _, item := range plan.Guidance {
		fmt.Fprintf(a.stdout, "- %s\n", item)
	}
	if len(plan.SuggestedCommands) == 0 {
		fmt.Fprintln(a.stdout, "suggested_commands: -")
		return
	}
	fmt.Fprintln(a.stdout, "suggested_commands (not run by resume-plan):")
	for _, command := range plan.SuggestedCommands {
		fmt.Fprintf(a.stdout, "- %s [%s]: %s\n", command.Label, valueOrDash(command.SideEffect), formatSuggestedCommand(command.Args))
		if command.Reason != "" {
			fmt.Fprintf(a.stdout, "  reason: %s\n", command.Reason)
		}
	}
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

func formatStringList(values []string) string {
	if len(values) == 0 {
		return "-"
	}
	return strings.Join(values, "; ")
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

func extractSessionIDs(summaries []sessions.Summary) []string {
	ids := make([]string, 0, len(summaries))
	for _, summary := range summaries {
		ids = append(ids, summary.SessionID)
	}
	return ids
}

func formatSessionIDList(ids []string) string {
	var buf strings.Builder
	for _, id := range ids {
		buf.WriteString("  - ")
		buf.WriteString(id)
		buf.WriteString("\n")
	}
	return strings.TrimSuffix(buf.String(), "\n")
}
