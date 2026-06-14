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
	Git             *workspaceGitContextJSON  `json:"git,omitempty"`
	DiagnosticCount int                       `json:"diagnostic_count"`
	HasErrors       bool                      `json:"has_errors"`
	Diagnostics     []workspaceDiagnosticJSON `json:"diagnostics"`
}

type workspaceGitContextJSON struct {
	ProjectRoot        string `json:"project_root,omitempty"`
	GitRoot            string `json:"git_root,omitempty"`
	GitDir             string `json:"git_dir,omitempty"`
	MetadataPath       string `json:"metadata_path,omitempty"`
	MetadataKind       string `json:"metadata_kind,omitempty"`
	InsideWorkTree     bool   `json:"inside_work_tree"`
	ProjectBelowRoot   bool   `json:"project_below_root"`
	RelativeProjectDir string `json:"relative_project_dir,omitempty"`
	Branch             string `json:"branch,omitempty"`
	Upstream           string `json:"upstream,omitempty"`
	Ahead              int    `json:"ahead"`
	Behind             int    `json:"behind"`
	ChangeState        string `json:"change_state,omitempty"`
	ChangedEntries     int    `json:"changed_entries"`
	UntrackedEntries   int    `json:"untracked_entries"`
	InspectionError    string `json:"inspection_error,omitempty"`
	StatusError        string `json:"status_error,omitempty"`
	GitAvailable       bool   `json:"git_available"`
}

type workspaceDiagnosticJSON struct {
	Level      string                         `json:"level"`
	Code       string                         `json:"code"`
	Message    string                         `json:"message"`
	Path       string                         `json:"path,omitempty"`
	Suggestion *workspaceSuggestionJSON       `json:"suggestion,omitempty"`
}

type workspaceSuggestionJSON struct {
	Reason   string   `json:"reason,omitempty"`
	Commands []string `json:"commands,omitempty"`
	DocLink  string   `json:"doc_link,omitempty"`
	Notes    []string `json:"notes,omitempty"`
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
			Git:             workspaceGitContextOutput(report.Git),
			DiagnosticCount: len(report.Diagnostics),
			HasErrors:       report.HasErrors(),
			Diagnostics:     make([]workspaceDiagnosticJSON, 0, len(report.Diagnostics)),
		}
		if reportOut.HasErrors {
			out.HasErrors = true
		}
		for _, diagnostic := range report.Diagnostics {
			diagJSON := workspaceDiagnosticJSON{
				Level:   string(diagnostic.Level),
				Code:    diagnostic.Code,
				Message: diagnostic.Message,
				Path:    diagnostic.Path,
			}
			if diagnostic.Suggestion != nil {
				diagJSON.Suggestion = &workspaceSuggestionJSON{
					Reason:   diagnostic.Suggestion.Reason,
					Commands: diagnostic.Suggestion.Commands,
					DocLink:  diagnostic.Suggestion.DocLink,
					Notes:    diagnostic.Suggestion.Notes,
				}
			}
			reportOut.Diagnostics = append(reportOut.Diagnostics, diagJSON)
		}
		out.Reports = append(out.Reports, reportOut)
	}
	return out
}

func workspaceGitContextOutput(git *workspace.GitDiagnosticContext) *workspaceGitContextJSON {
	if git == nil {
		return nil
	}
	return &workspaceGitContextJSON{
		ProjectRoot:        git.ProjectRoot,
		GitRoot:            git.GitRoot,
		GitDir:             git.GitDir,
		MetadataPath:       git.MetadataPath,
		MetadataKind:       git.MetadataKind,
		InsideWorkTree:     git.InsideWorkTree,
		ProjectBelowRoot:   git.ProjectBelowRoot,
		RelativeProjectDir: git.RelativeProjectDir,
		Branch:             git.Branch,
		Upstream:           git.Upstream,
		Ahead:              git.Ahead,
		Behind:             git.Behind,
		ChangeState:        git.ChangeState,
		ChangedEntries:     git.ChangedEntries,
		UntrackedEntries:   git.UntrackedEntries,
		InspectionError:    git.InspectionError,
		StatusError:        git.StatusError,
		GitAvailable:       git.GitAvailable,
	}
}
