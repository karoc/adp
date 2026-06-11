package workspace

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/karoc/adp/internal/gitenv"
	"github.com/karoc/adp/internal/gitstate"
	"github.com/karoc/adp/internal/schema"
)

type DiagnosticLevel string

const (
	DiagnosticLevelInfo    DiagnosticLevel = "info"
	DiagnosticLevelWarning DiagnosticLevel = "warning"
	DiagnosticLevelError   DiagnosticLevel = "error"
)

const (
	DiagnosticCodeWorkspaceNameInvalid      = "workspace.name.invalid"
	DiagnosticCodeWorkspaceNameMismatch     = "workspace.name.mismatch"
	DiagnosticCodeWorkspaceDirMissing       = "workspace.dir.missing"
	DiagnosticCodeWorkspaceDirStatFailed    = "workspace.dir.stat_failed"
	DiagnosticCodeWorkspaceDirSymlink       = "workspace.dir.symlink"
	DiagnosticCodeWorkspaceDirNotDirectory  = "workspace.dir.not_directory"
	DiagnosticCodeConfigMissing             = "workspace.config.missing"
	DiagnosticCodeConfigInvalid             = "workspace.config.invalid"
	DiagnosticCodeProjectRootMissing        = "workspace.project.root.missing"
	DiagnosticCodeProjectRootStatFailed     = "workspace.project.root.stat_failed"
	DiagnosticCodeProjectRootNotDirectory   = "workspace.project.root.not_directory"
	DiagnosticCodeProjectRootReservedPath   = "workspace.project.reserved_path.present"
	DiagnosticCodeProjectRootReservedStat   = "workspace.project.reserved_path.stat_failed"
	DiagnosticCodePromptOutsideWorkspace    = "workspace.prompt.outside_workspace"
	DiagnosticCodePromptMissing             = "workspace.prompt.missing"
	DiagnosticCodePromptStatFailed          = "workspace.prompt.stat_failed"
	DiagnosticCodePromptNotFile             = "workspace.prompt.not_file"
	DiagnosticCodeMemorySharedNotConfigured = "workspace.memory.shared.not_configured"
	DiagnosticCodeMemorySharedOutside       = "workspace.memory.shared.outside_workspace"
	DiagnosticCodeMemorySharedMissing       = "workspace.memory.shared.missing"
	DiagnosticCodeMemorySharedStatFailed    = "workspace.memory.shared.stat_failed"
	DiagnosticCodeMemorySharedNotFile       = "workspace.memory.shared.not_file"
	DiagnosticCodeMCPConfigNotConfigured    = "workspace.mcp.config.not_configured"
	DiagnosticCodeMCPConfigOutside          = "workspace.mcp.config.outside_workspace"
	DiagnosticCodeMCPConfigMissing          = "workspace.mcp.config.missing"
	DiagnosticCodeMCPConfigStatFailed       = "workspace.mcp.config.stat_failed"
	DiagnosticCodeMCPConfigNotFile          = "workspace.mcp.config.not_file"
	DiagnosticCodeAgentUnknown              = "workspace.agent.unknown"
	DiagnosticCodeAgentCommandDefault       = "workspace.agent.command.default"
	DiagnosticCodeAgentCommandArguments     = "workspace.agent.command.arguments"
	DiagnosticCodeAgentCommandMissing       = "workspace.agent.command.missing"
	DiagnosticCodeAgentCommandStatFailed    = "workspace.agent.command.stat_failed"
	DiagnosticCodeAgentCommandNotExecutable = "workspace.agent.command.not_executable"
	DiagnosticCodeAgentProfileInvalid       = "workspace.agent.profile.invalid"
	DiagnosticCodeAgentProfileOutside       = "workspace.agent.profile.outside_workspace"
	DiagnosticCodeAgentProfileMissing       = "workspace.agent.profile.missing"
	DiagnosticCodeAgentProfileStatFailed    = "workspace.agent.profile.stat_failed"
	DiagnosticCodeAgentProfileNotFile       = "workspace.agent.profile.not_file"
	DiagnosticCodeAgentProfileAmbiguous     = "workspace.agent.profile.ambiguous"
	DiagnosticCodeGitEnvRepositoryDirective = "workspace.git.env.repository_directive"
	DiagnosticCodeGitRootDetected           = "workspace.git.root.detected"
	DiagnosticCodeGitRootAbsent             = "workspace.git.root.absent"
	DiagnosticCodeGitRootNested             = "workspace.git.root.nested_project"
	DiagnosticCodeGitMetadataFile           = "workspace.git.metadata.file"
	DiagnosticCodeGitMetadataOther          = "workspace.git.metadata.other"
	DiagnosticCodeGitStatusDirty            = "workspace.git.status.dirty"
	DiagnosticCodeGitStatusUnavailable      = "workspace.git.status.unavailable"
)

type Diagnostic struct {
	Level   DiagnosticLevel
	Code    string
	Message string
	Path    string
}

type DiagnosticReport struct {
	Workspace    string
	WorkspaceDir string
	ConfigPath   string
	Git          *GitDiagnosticContext
	Diagnostics  []Diagnostic
}

type GitDiagnosticContext struct {
	ProjectRoot        string
	GitRoot            string
	GitDir             string
	MetadataPath       string
	MetadataKind       string
	InsideWorkTree     bool
	ProjectBelowRoot   bool
	RelativeProjectDir string
	Branch             string
	Upstream           string
	Ahead              int
	Behind             int
	ChangeState        string
	ChangedEntries     int
	UntrackedEntries   int
	InspectionError    string
	StatusError        string
	GitAvailable       bool
}

func (r DiagnosticReport) HasErrors() bool {
	for _, diagnostic := range r.Diagnostics {
		if diagnostic.Level == DiagnosticLevelError {
			return true
		}
	}
	return false
}

func (r *Registry) Diagnose(ctx context.Context, name string) (DiagnosticReport, error) {
	if err := ctx.Err(); err != nil {
		return DiagnosticReport{}, err
	}
	if err := schema.ValidateWorkspaceName(name); err != nil {
		return DiagnosticReport{}, err
	}

	workspaceDir, err := r.safeWorkspaceDir(name)
	if err != nil {
		return DiagnosticReport{}, err
	}
	return diagnoseWorkspaceDir(ctx, name, workspaceDir, r.Layout.RuntimeParent)
}

func (r *Registry) DiagnoseAll(ctx context.Context) ([]DiagnosticReport, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(r.Layout.WorkspacesDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read workspaces directory: %w", err)
	}

	reports := make([]DiagnosticReport, 0, len(entries))
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		name := entry.Name()
		workspaceDir := filepath.Join(r.Layout.WorkspacesDir, name)
		report, err := diagnoseWorkspaceDir(ctx, name, workspaceDir, r.Layout.RuntimeParent)
		if err != nil {
			return nil, err
		}
		if err := schema.ValidateWorkspaceName(name); err != nil {
			report.Diagnostics = append([]Diagnostic{{
				Level:   DiagnosticLevelError,
				Code:    DiagnosticCodeWorkspaceNameInvalid,
				Message: fmt.Sprintf("workspace directory name %q is invalid: %v", name, err),
				Path:    workspaceDir,
			}}, report.Diagnostics...)
		}
		reports = append(reports, report)
	}

	sort.Slice(reports, func(i, j int) bool {
		return reports[i].Workspace < reports[j].Workspace
	})
	return reports, nil
}

func diagnoseWorkspaceDir(ctx context.Context, name string, workspaceDir string, runtimeParent string) (DiagnosticReport, error) {
	report := DiagnosticReport{
		Workspace:    name,
		WorkspaceDir: workspaceDir,
		ConfigPath:   filepath.Join(workspaceDir, "workspace.yaml"),
	}
	if err := ctx.Err(); err != nil {
		return report, err
	}
	checkInheritedGitEnvironment(&report)

	info, err := os.Lstat(workspaceDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			report.add(DiagnosticLevelError, DiagnosticCodeWorkspaceDirMissing, "workspace directory is missing", workspaceDir)
			return report, nil
		}
		report.add(DiagnosticLevelError, DiagnosticCodeWorkspaceDirStatFailed, fmt.Sprintf("workspace directory could not be inspected: %v", err), workspaceDir)
		return report, nil
	}
	if info.Mode()&os.ModeSymlink != 0 {
		report.add(DiagnosticLevelError, DiagnosticCodeWorkspaceDirSymlink, "workspace directory must not be a symlink", workspaceDir)
		return report, nil
	}
	if !info.IsDir() {
		report.add(DiagnosticLevelError, DiagnosticCodeWorkspaceDirNotDirectory, "workspace path is not a directory", workspaceDir)
		return report, nil
	}

	cfg, err := schema.LoadConfig(report.ConfigPath)
	if err != nil {
		code := DiagnosticCodeConfigInvalid
		if errors.Is(err, os.ErrNotExist) {
			code = DiagnosticCodeConfigMissing
		}
		report.add(DiagnosticLevelError, code, fmt.Sprintf("workspace config could not be loaded and validated: %v", err), report.ConfigPath)
		return report, nil
	}
	if cfg.Workspace.Name != name {
		report.add(DiagnosticLevelError, DiagnosticCodeWorkspaceNameMismatch, fmt.Sprintf("workspace config name is %q, but directory name is %q", cfg.Workspace.Name, name), report.ConfigPath)
	}

	checkProjectRoot(&report, cfg.Project.Root)
	checkProjectGit(ctx, &report, cfg.Project.Root)
	checkProjectReservedPaths(&report, cfg.Project.Root, cfg.Agents)
	checkRuntimeParent(&report, runtimeParent, cfg.Project.Root)
	checkWorkspaceFile(&report, fileCheck{
		Label:        "base prompt",
		RelPath:      cfg.Prompts.Base,
		MissingCode:  DiagnosticCodePromptMissing,
		OutsideCode:  DiagnosticCodePromptOutsideWorkspace,
		StatCode:     DiagnosticCodePromptStatFailed,
		NotFileCode:  DiagnosticCodePromptNotFile,
		EmptyMessage: "",
	})

	if cfg.Memory.Enabled {
		checkWorkspaceFile(&report, fileCheck{
			Label:        "shared memory",
			RelPath:      cfg.Memory.Shared,
			MissingCode:  DiagnosticCodeMemorySharedMissing,
			OutsideCode:  DiagnosticCodeMemorySharedOutside,
			StatCode:     DiagnosticCodeMemorySharedStatFailed,
			NotFileCode:  DiagnosticCodeMemorySharedNotFile,
			EmptyCode:    DiagnosticCodeMemorySharedNotConfigured,
			EmptyMessage: "shared memory is enabled, but no shared memory path is configured",
		})
	}
	if cfg.MCP.Enabled {
		checkWorkspaceFile(&report, fileCheck{
			Label:        "MCP config",
			RelPath:      cfg.MCP.Config,
			MissingCode:  DiagnosticCodeMCPConfigMissing,
			OutsideCode:  DiagnosticCodeMCPConfigOutside,
			StatCode:     DiagnosticCodeMCPConfigStatFailed,
			NotFileCode:  DiagnosticCodeMCPConfigNotFile,
			EmptyCode:    DiagnosticCodeMCPConfigNotConfigured,
			EmptyMessage: "MCP is enabled, but no MCP config path is configured",
		})
	}
	if err := checkAgents(ctx, &report, cfg.Project.Root, cfg.Agents); err != nil {
		return report, err
	}
	return report, nil
}

func (r *DiagnosticReport) add(level DiagnosticLevel, code string, message string, path string) {
	r.Diagnostics = append(r.Diagnostics, Diagnostic{
		Level:   level,
		Code:    code,
		Message: message,
		Path:    path,
	})
}

func checkInheritedGitEnvironment(report *DiagnosticReport) {
	names := make([]string, 0)
	for _, name := range gitenv.RepositoryDirectiveNames() {
		if _, ok := os.LookupEnv(name); ok {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return
	}

	report.add(
		DiagnosticLevelWarning,
		DiagnosticCodeGitEnvRepositoryDirective,
		fmt.Sprintf("operator environment contains repository-directing Git variables (%s); ADP runtime neutralizes these for launched and shell-hook sessions, and Git commands should target ADP_PROJECT_ROOT explicitly", strings.Join(names, ", ")),
		"",
	)
}

func checkProjectRoot(report *DiagnosticReport, root string) {
	info, err := os.Stat(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			report.add(DiagnosticLevelError, DiagnosticCodeProjectRootMissing, "project root is missing", root)
			return
		}
		report.add(DiagnosticLevelError, DiagnosticCodeProjectRootStatFailed, fmt.Sprintf("project root could not be inspected: %v", err), root)
		return
	}
	if !info.IsDir() {
		report.add(DiagnosticLevelError, DiagnosticCodeProjectRootNotDirectory, "project root is not a directory", root)
	}
}

func checkProjectGit(ctx context.Context, report *DiagnosticReport, root string) {
	if hasProjectRootError(*report, root) {
		return
	}
	state := gitstate.Inspect(ctx, root)
	report.Git = gitDiagnosticContext(state)
	if !state.GitAvailable || state.InspectionError != "" {
		message := "project root is not inside a usable Git worktree; ADP can still run, but phase evidence and agent handoff are easier to audit when the real project root has Git status available"
		if state.InspectionError != "" {
			message += ": " + state.InspectionError
		}
		report.add(DiagnosticLevelWarning, DiagnosticCodeGitRootAbsent, message, root)
		return
	}

	report.add(DiagnosticLevelInfo, DiagnosticCodeGitRootDetected, fmt.Sprintf("Git worktree detected at %s; use git -C \"$ADP_PROJECT_ROOT\" status --short --branch for authoritative status, and treat ADP runtime roots as non-authoritative overlays", state.GitRoot), state.GitRoot)
	if state.ProjectBelowRoot {
		report.add(DiagnosticLevelInfo, DiagnosticCodeGitRootNested, fmt.Sprintf("project root is inside Git worktree %s at %s; ADP_GIT_ROOT will point at the repository root while ADP_PROJECT_ROOT remains the configured project root", state.GitRoot, state.RelativeProjectDir), root)
	}
	switch state.MetadataKind {
	case gitstate.MetadataFile:
		report.add(DiagnosticLevelInfo, DiagnosticCodeGitMetadataFile, "Git metadata is represented by a .git file, as in a worktree or submodule; ADP excludes this metadata from runtime overlays and Git commands should target ADP_PROJECT_ROOT explicitly", state.MetadataPath)
	case gitstate.MetadataOther:
		report.add(DiagnosticLevelWarning, DiagnosticCodeGitMetadataOther, "Git metadata path exists but is not a normal .git directory or gitfile; inspect the project root before relying on Git status", state.MetadataPath)
	}
	if state.StatusError != "" {
		report.add(DiagnosticLevelWarning, DiagnosticCodeGitStatusUnavailable, "Git worktree was detected, but read-only status inspection failed: "+state.StatusError, root)
		return
	}
	if state.ChangeState == gitstate.ChangeDirty {
		message := fmt.Sprintf("project has %d changed Git status entries", state.ChangedEntries)
		if state.UntrackedEntries > 0 {
			message += fmt.Sprintf(" including %d untracked", state.UntrackedEntries)
		}
		if state.Ahead > 0 || state.Behind > 0 {
			message += fmt.Sprintf("; upstream delta is +%d/-%d", state.Ahead, state.Behind)
		}
		message += "; inspect from the real project root, not the ADP runtime root"
		report.add(DiagnosticLevelWarning, DiagnosticCodeGitStatusDirty, message, root)
	}
}

func gitDiagnosticContext(state gitstate.State) *GitDiagnosticContext {
	return &GitDiagnosticContext{
		ProjectRoot:        state.ProjectRoot,
		GitRoot:            state.GitRoot,
		GitDir:             state.GitDir,
		MetadataPath:       state.MetadataPath,
		MetadataKind:       string(state.MetadataKind),
		InsideWorkTree:     state.InsideWorkTree,
		ProjectBelowRoot:   state.ProjectBelowRoot,
		RelativeProjectDir: state.RelativeProjectDir,
		Branch:             state.Branch,
		Upstream:           state.Upstream,
		Ahead:              state.Ahead,
		Behind:             state.Behind,
		ChangeState:        string(state.ChangeState),
		ChangedEntries:     state.ChangedEntries,
		UntrackedEntries:   state.UntrackedEntries,
		InspectionError:    state.InspectionError,
		StatusError:        state.StatusError,
		GitAvailable:       state.GitAvailable,
	}
}

func hasProjectRootError(report DiagnosticReport, root string) bool {
	for _, diagnostic := range report.Diagnostics {
		if diagnostic.Path != root || diagnostic.Level != DiagnosticLevelError {
			continue
		}
		switch diagnostic.Code {
		case DiagnosticCodeProjectRootMissing, DiagnosticCodeProjectRootStatFailed, DiagnosticCodeProjectRootNotDirectory:
			return true
		}
	}
	return false
}

type fileCheck struct {
	Label        string
	RelPath      string
	EmptyCode    string
	EmptyMessage string
	OutsideCode  string
	MissingCode  string
	StatCode     string
	NotFileCode  string
}

func checkWorkspaceFile(report *DiagnosticReport, check fileCheck) {
	rel := strings.TrimSpace(check.RelPath)
	if rel == "" {
		if check.EmptyCode != "" {
			report.add(DiagnosticLevelWarning, check.EmptyCode, check.EmptyMessage, report.ConfigPath)
		}
		return
	}

	inspection := inspectWorkspacePath(report.WorkspaceDir, rel)
	if inspection.Outside {
		report.add(DiagnosticLevelWarning, check.OutsideCode, fmt.Sprintf("configured %s path %q is outside the ADP workspace directory", check.Label, rel), inspection.Path)
		return
	}

	if inspection.Err != nil {
		if errors.Is(inspection.Err, os.ErrNotExist) {
			report.add(DiagnosticLevelWarning, check.MissingCode, fmt.Sprintf("configured %s file is missing", check.Label), inspection.Path)
			return
		}
		report.add(DiagnosticLevelWarning, check.StatCode, fmt.Sprintf("configured %s file could not be inspected: %v", check.Label, inspection.Err), inspection.Path)
		return
	}
	if inspection.Info.IsDir() {
		report.add(DiagnosticLevelWarning, check.NotFileCode, fmt.Sprintf("configured %s path is a directory, not a file", check.Label), inspection.Path)
	}
}

func workspaceFilePath(workspaceDir string, rel string) (string, bool) {
	if filepath.IsAbs(rel) {
		return filepath.Clean(rel), false
	}
	clean := filepath.Clean(rel)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return filepath.Join(workspaceDir, clean), false
	}
	return filepath.Join(workspaceDir, clean), true
}

type pathInspection struct {
	Path    string
	Outside bool
	Info    os.FileInfo
	Err     error
}

func inspectWorkspacePath(workspaceDir string, rel string) pathInspection {
	fullPath, ok := workspaceFilePath(workspaceDir, rel)
	if !ok {
		return pathInspection{Path: displayWorkspacePath(workspaceDir, rel), Outside: true}
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		return pathInspection{Path: fullPath, Err: err}
	}

	inside, err := resolvesInsideWorkspace(workspaceDir, fullPath)
	if err != nil {
		return pathInspection{Path: fullPath, Err: err}
	}
	if !inside {
		return pathInspection{Path: fullPath, Outside: true, Info: info}
	}
	return pathInspection{Path: fullPath, Info: info}
}

func resolvesInsideWorkspace(workspaceDir string, path string) (bool, error) {
	base, err := filepath.EvalSymlinks(workspaceDir)
	if err != nil {
		return false, err
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return false, err
	}
	return pathInsideDir(base, resolved), nil
}

func pathInsideDir(base string, target string) bool {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel))
}

func displayWorkspacePath(workspaceDir string, rel string) string {
	if filepath.IsAbs(rel) {
		return filepath.Clean(rel)
	}
	return filepath.Join(workspaceDir, filepath.Clean(rel))
}
