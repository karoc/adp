package cli

import taskstore "github.com/karoc/adp/internal/tasks"

type planningDoctorJSON struct {
	Workspace          string                   `json:"workspace"`
	WorkspaceDir       string                   `json:"workspace_dir"`
	PlanningDir        string                   `json:"planning_dir"`
	TasksPath          string                   `json:"tasks_path"`
	PhasesPath         string                   `json:"phases_path"`
	ProgressPath       string                   `json:"progress_path"`
	Status             string                   `json:"status"`
	TaskCount          int                      `json:"task_count"`
	PhaseCount         int                      `json:"phase_count"`
	ProgressEventCount int                      `json:"progress_event_count"`
	DiagnosticCount    int                      `json:"diagnostic_count"`
	ErrorCount         int                      `json:"error_count"`
	WarningCount       int                      `json:"warning_count"`
	HasErrors          bool                     `json:"has_errors"`
	PhaseGate          phaseGateJSON            `json:"phase_gate"`
	Diagnostics        []planningDiagnosticJSON `json:"diagnostics"`
}

type planningDiagnosticJSON struct {
	Level   string `json:"level"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Path    string `json:"path,omitempty"`
	Line    int    `json:"line,omitempty"`
}

func planningDoctorOutput(workspace string, report taskstore.PlanningDiagnosticReport) planningDoctorJSON {
	status := "ok"
	hasErrors := report.HasErrors()
	if hasErrors {
		status = "error"
	}
	out := planningDoctorJSON{
		Workspace:          workspace,
		WorkspaceDir:       report.WorkspaceDir,
		PlanningDir:        report.PlanningDir,
		TasksPath:          report.TasksPath,
		PhasesPath:         report.PhasesPath,
		ProgressPath:       report.ProgressPath,
		Status:             status,
		TaskCount:          report.TaskCount,
		PhaseCount:         report.PhaseCount,
		ProgressEventCount: report.ProgressEventCount,
		DiagnosticCount:    len(report.Diagnostics),
		ErrorCount:         report.ErrorCount(),
		WarningCount:       report.WarningCount(),
		HasErrors:          hasErrors,
		PhaseGate:          phaseGateOutput("", report.PhaseGate),
		Diagnostics:        make([]planningDiagnosticJSON, 0, len(report.Diagnostics)),
	}
	for _, diagnostic := range report.Diagnostics {
		out.Diagnostics = append(out.Diagnostics, planningDiagnosticJSON{
			Level:   string(diagnostic.Level),
			Code:    diagnostic.Code,
			Message: diagnostic.Message,
			Path:    diagnostic.Path,
			Line:    diagnostic.Line,
		})
	}
	return out
}
