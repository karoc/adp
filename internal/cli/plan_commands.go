package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/karoc/adp/internal/planinput"
	taskstore "github.com/karoc/adp/internal/tasks"
)

const planInputUsage = "adp plan <preview|apply|doctor>"

func (a *App) plan(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: " + planInputUsage)
	}
	switch args[0] {
	case "preview":
		return a.planPreview(ctx, args[1:])
	case "apply":
		return a.planApply(ctx, args[1:])
	case "doctor":
		return a.planDoctor(ctx, args[1:])
	default:
		return fmt.Errorf("unknown plan command %q", args[0])
	}
}

func (a *App) planPreview(ctx context.Context, args []string) error {
	opts, err := parsePlanInputArgs(args, "adp plan preview [--workspace <name>] --file <path|-> [--format <text|json>]")
	if err != nil {
		return err
	}
	req, err := readPlanImportRequest(opts.file)
	if err != nil {
		return err
	}
	store, workspaceName, err := a.loadTaskStore(ctx, opts.workspace)
	if err != nil {
		return err
	}
	result, err := store.PreviewPlanImport(ctx, req)
	if err != nil {
		return err
	}
	return a.printPlanImportResult(planImportPrintRequest{
		workspace: workspaceName,
		mode:      "preview",
		source:    opts.file,
		format:    opts.format,
		result:    result,
	})
}

func (a *App) planApply(ctx context.Context, args []string) error {
	opts, err := parsePlanInputArgs(args, "adp plan apply [--workspace <name>] --file <path|-> [--format <text|json>]")
	if err != nil {
		return err
	}
	req, err := readPlanImportRequest(opts.file)
	if err != nil {
		return err
	}
	store, workspaceName, err := a.loadTaskStore(ctx, opts.workspace)
	if err != nil {
		return err
	}
	result, err := store.ApplyPlanImport(ctx, req)
	if err != nil {
		return err
	}
	return a.printPlanImportResult(planImportPrintRequest{
		workspace: workspaceName,
		mode:      "apply",
		source:    opts.file,
		format:    opts.format,
		result:    result,
	})
}

func (a *App) planDoctor(ctx context.Context, args []string) error {
	opts, err := parseWorkspaceOutputArgs(args, "adp plan doctor [--workspace <name>] [--format <text|json>]")
	if err != nil {
		return err
	}
	store, workspaceName, err := a.loadTaskStore(ctx, opts.workspace)
	if err != nil {
		return err
	}
	report, err := store.DiagnosePlanning(ctx)
	if err != nil {
		return err
	}
	if opts.format == outputFormatJSON {
		if err := writePlanningJSON(a.stdout, planningDoctorOutput(workspaceName, report)); err != nil {
			return err
		}
	} else if err := a.printPlanningDoctorReport(workspaceName, report); err != nil {
		return err
	}
	if report.HasErrors() {
		return processExitError{code: 2}
	}
	return nil
}

type planImportPrintRequest struct {
	workspace string
	mode      string
	source    string
	format    outputFormat
	result    taskstore.PlanImportResult
}

func (a *App) printPlanImportResult(req planImportPrintRequest) error {
	if req.format == outputFormatJSON {
		return writePlanningJSON(a.stdout, planImportOutput(req.workspace, req.mode, req.source, req.result))
	}
	fmt.Fprintf(a.stdout, "workspace: %s\n", req.workspace)
	fmt.Fprintf(a.stdout, "mode: %s\n", req.mode)
	fmt.Fprintf(a.stdout, "source: %s\n", req.source)
	fmt.Fprintf(a.stdout, "phases: %d\n", len(req.result.Phases))
	if len(req.result.Phases) > 0 {
		writer := tabwriter.NewWriter(a.stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(writer, "PHASE\tSTATUS\tTITLE")
		for _, phase := range req.result.Phases {
			fmt.Fprintf(writer, "%s\t%s\t%s\n", phase.ID, phase.Status, phase.Title)
		}
		if err := writer.Flush(); err != nil {
			return err
		}
	}
	fmt.Fprintf(a.stdout, "tasks: %d\n", len(req.result.Tasks))
	if len(req.result.Tasks) == 0 {
		return nil
	}
	return a.printTaskTable(req.result.Tasks)
}

func (a *App) printPlanningDoctorReport(workspaceName string, report taskstore.PlanningDiagnosticReport) error {
	status := "ok"
	if report.HasErrors() {
		status = "error"
	}
	fmt.Fprintf(a.stdout, "workspace: %s\n", workspaceName)
	fmt.Fprintf(a.stdout, "planning_dir: %s\n", valueOrDash(report.PlanningDir))
	fmt.Fprintf(a.stdout, "status: %s\n", status)
	fmt.Fprintf(a.stdout, "task_count: %d\n", report.TaskCount)
	fmt.Fprintf(a.stdout, "phase_count: %d\n", report.PhaseCount)
	fmt.Fprintf(a.stdout, "progress_event_count: %d\n", report.ProgressEventCount)
	fmt.Fprintf(a.stdout, "error_count: %d\n", report.ErrorCount())
	fmt.Fprintf(a.stdout, "warning_count: %d\n", report.WarningCount())
	fmt.Fprintf(a.stdout, "phase_gate_next_action: %s\n", valueOrDash(report.PhaseGate.NextAction))
	fmt.Fprintf(a.stdout, "phase_gate_can_start_next: %t\n", report.PhaseGate.CanStartNext)
	if len(report.Diagnostics) == 0 {
		fmt.Fprintln(a.stdout, "diagnostics: -")
		return nil
	}
	fmt.Fprintln(a.stdout, "diagnostics:")
	writer := tabwriter.NewWriter(a.stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "LEVEL\tCODE\tLINE\tMESSAGE\tPATH")
	for _, diagnostic := range report.Diagnostics {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n",
			diagnostic.Level,
			valueOrDash(diagnostic.Code),
			diagnosticLine(diagnostic.Line),
			valueOrDash(diagnostic.Message),
			valueOrDash(diagnostic.Path),
		)
	}
	return writer.Flush()
}

func diagnosticLine(line int) string {
	if line <= 0 {
		return "-"
	}
	return fmt.Sprintf("%d", line)
}

func readPlanImportRequest(path string) (taskstore.PlanImportRequest, error) {
	var data []byte
	var err error
	if path == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return taskstore.PlanImportRequest{}, fmt.Errorf("read plan input %s: %w", path, err)
	}
	doc, err := planinput.Parse(data)
	if err != nil {
		return taskstore.PlanImportRequest{}, err
	}
	return planImportRequestFromDocument(doc)
}

func planImportRequestFromDocument(doc planinput.Document) (taskstore.PlanImportRequest, error) {
	req := taskstore.PlanImportRequest{
		Phases: make([]taskstore.PlanImportPhase, 0, len(doc.Phases)),
		Tasks:  make([]taskstore.PlanImportTask, 0, len(doc.Tasks)),
	}
	for _, phase := range doc.Phases {
		req.Phases = append(req.Phases, taskstore.PlanImportPhase{
			ID:    phase.ID,
			Title: phase.Title,
			Goal:  phase.Goal,
		})
	}
	for _, task := range doc.Tasks {
		var status taskstore.Status
		if task.Status != "" {
			parsed, err := taskstore.ParseStatus(task.Status)
			if err != nil {
				return taskstore.PlanImportRequest{}, err
			}
			status = parsed
		}
		req.Tasks = append(req.Tasks, taskstore.PlanImportTask{
			Title:       task.Title,
			Description: task.Description,
			Priority:    task.Priority,
			Phase:       task.Phase,
			Status:      status,
		})
	}
	return req, nil
}
