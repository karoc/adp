package cli

import "github.com/karoc/adp/internal/workspace"

type workspaceDoctorJSONOutput struct {
	ReportCount int                             `json:"report_count"`
	HasErrors   bool                            `json:"has_errors"`
	Reports     []workspaceDiagnosticReportJSON `json:"reports"`
}

type workspaceDiagnosticReportJSON struct {
	Workspace       string                    `json:"workspace"`
	WorkspaceDir    string                    `json:"workspace_dir,omitempty"`
	ConfigPath      string                    `json:"config_path,omitempty"`
	DiagnosticCount int                       `json:"diagnostic_count"`
	HasErrors       bool                      `json:"has_errors"`
	Diagnostics     []workspaceDiagnosticJSON `json:"diagnostics"`
}

type workspaceDiagnosticJSON struct {
	Level   string `json:"level"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Path    string `json:"path,omitempty"`
}

func workspaceDoctorOutput(reports []workspace.DiagnosticReport) workspaceDoctorJSONOutput {
	out := workspaceDoctorJSONOutput{
		ReportCount: len(reports),
		Reports:     make([]workspaceDiagnosticReportJSON, 0, len(reports)),
	}
	for _, report := range reports {
		reportOut := workspaceDiagnosticReportJSON{
			Workspace:       report.Workspace,
			WorkspaceDir:    report.WorkspaceDir,
			ConfigPath:      report.ConfigPath,
			DiagnosticCount: len(report.Diagnostics),
			HasErrors:       report.HasErrors(),
			Diagnostics:     make([]workspaceDiagnosticJSON, 0, len(report.Diagnostics)),
		}
		if reportOut.HasErrors {
			out.HasErrors = true
		}
		for _, diagnostic := range report.Diagnostics {
			reportOut.Diagnostics = append(reportOut.Diagnostics, workspaceDiagnosticJSON{
				Level:   string(diagnostic.Level),
				Code:    diagnostic.Code,
				Message: diagnostic.Message,
				Path:    diagnostic.Path,
			})
		}
		out.Reports = append(out.Reports, reportOut)
	}
	return out
}
