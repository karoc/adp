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
	Level      DiagnosticLevel
	Code       string
	Message    string
	Path       string
	Suggestion *DiagnosticSuggestion
}

// DiagnosticSuggestion 包含诊断的建议信息
type DiagnosticSuggestion struct {
	Reason   string
	Commands []string
	DocLink  string
	Notes    []string
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
	// 构建建议生成的上下文
	ctx := SuggestionContext{
		Workspace:    r.Workspace,
		WorkspaceDir: r.WorkspaceDir,
		ConfigPath:   r.ConfigPath,
		Path:         path,
	}

	// 如果有 Git 上下文，添加项目根目录
	if r.Git != nil {
		ctx.ProjectRoot = r.Git.ProjectRoot
	}

	suggestion := generateSuggestion(code, ctx)

	r.Diagnostics = append(r.Diagnostics, Diagnostic{
		Level:      level,
		Code:       code,
		Message:    message,
		Path:       path,
		Suggestion: suggestion,
	})
}

// SuggestionContext 提供生成建议所需的上下文信息
type SuggestionContext struct {
	Workspace     string
	WorkspaceDir  string
	ConfigPath    string
	Path          string
	ProjectRoot   string
	AgentCommand  string
	ProfileName   string
	AgentName     string
	ExpectedValue string
	ActualValue   string
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

// generateSuggestion 根据诊断代码和上下文生成建议
func generateSuggestion(code string, ctx SuggestionContext) *DiagnosticSuggestion {
	switch code {

	// === 工作区配置类 ===
	case DiagnosticCodeWorkspaceNameInvalid:
		return &DiagnosticSuggestion{
			Reason: "工作区名称不符合规范",
			Commands: []string{
				fmt.Sprintf("adp workspace remove %s", ctx.Workspace),
				"使用有效名称重新添加工作区（只能包含字母、数字、连字符和下划线）",
			},
			DocLink: "operator-onboarding.md#工作区设置",
		}

	case DiagnosticCodeWorkspaceNameMismatch:
		return &DiagnosticSuggestion{
			Reason: "配置文件中的名称与目录名不匹配",
			Commands: []string{
				fmt.Sprintf("检查配置文件: cat %s", ctx.ConfigPath),
				"修改配置文件中的 workspace.name 字段，或重命名工作区目录",
			},
			DocLink: "operator-onboarding.md#工作区配置",
		}

	case DiagnosticCodeWorkspaceDirMissing:
		return &DiagnosticSuggestion{
			Commands: []string{
				fmt.Sprintf("检查目录是否存在: ls -ld %s", ctx.WorkspaceDir),
				fmt.Sprintf("重新创建工作区: adp workspace add %s /path/to/project", ctx.Workspace),
			},
		}

	case DiagnosticCodeWorkspaceDirSymlink:
		return &DiagnosticSuggestion{
			Reason: "工作区目录不能是符号链接",
			Commands: []string{
				fmt.Sprintf("检查链接目标: ls -l %s", ctx.WorkspaceDir),
				fmt.Sprintf("adp workspace remove %s", ctx.Workspace),
				"使用真实路径重新添加工作区",
			},
		}

	case DiagnosticCodeWorkspaceDirNotDirectory:
		return &DiagnosticSuggestion{
			Reason: "工作区路径必须是目录",
			Commands: []string{
				fmt.Sprintf("检查文件类型: file %s", ctx.WorkspaceDir),
				fmt.Sprintf("adp workspace remove %s", ctx.Workspace),
				"删除该文件后重新添加工作区",
			},
		}

	case DiagnosticCodeConfigMissing:
		return &DiagnosticSuggestion{
			Commands: []string{
				fmt.Sprintf("adp workspace add %s /path/to/project", ctx.Workspace),
			},
			DocLink: "operator-onboarding.md#添加工作区",
		}

	case DiagnosticCodeConfigInvalid:
		return &DiagnosticSuggestion{
			Commands: []string{
				fmt.Sprintf("检查配置语法: cat %s", ctx.ConfigPath),
				"修复 YAML 语法错误，或删除后重新生成",
			},
			DocLink: "operator-onboarding.md#工作区配置",
		}

	// === 项目根目录类 ===
	case DiagnosticCodeProjectRootMissing:
		return &DiagnosticSuggestion{
			Commands: []string{
				fmt.Sprintf("检查路径是否存在: ls -ld %s", ctx.Path),
				fmt.Sprintf("如果项目已移动: adp workspace remove %s && adp workspace add %s /new/path", ctx.Workspace, ctx.Workspace),
			},
			DocLink: "troubleshooting.zh-CN.md#project-root-does-not-exist",
		}

	case DiagnosticCodeProjectRootNotDirectory:
		return &DiagnosticSuggestion{
			Reason: "项目根目录必须是目录",
			Commands: []string{
				fmt.Sprintf("检查文件类型: file %s", ctx.Path),
				fmt.Sprintf("adp workspace remove %s && adp workspace add %s /correct/path", ctx.Workspace, ctx.Workspace),
			},
		}

	case DiagnosticCodeProjectRootReservedPath:
		return &DiagnosticSuggestion{
			Reason: "项目包含 ADP 保留路径，可能与运行时生成的文件冲突",
			Commands: []string{
				fmt.Sprintf("检查文件: ls -la %s", ctx.Path),
				fmt.Sprintf("移除或重命名: mv %s %s.bak", ctx.Path, ctx.Path),
			},
			Notes: []string{
				"保留路径包括: AGENTS.md, CLAUDE.md, tasks.yaml, phases.yaml 等",
			},
		}

	// === 运行时目录类 ===
	case DiagnosticCodeRuntimeParentMissing:
		return &DiagnosticSuggestion{
			Commands: []string{
				`export ADP_RUNTIME_DIR="/tmp/adp-runtime"`,
			},
			Notes: []string{
				"将此行添加到 ~/.bashrc 或 ~/.zshrc 使其持久化",
			},
			DocLink: "troubleshooting.zh-CN.md#failed-to-build-runtime",
		}

	case DiagnosticCodeRuntimeParentInsideProjectRoot:
		return &DiagnosticSuggestion{
			Reason: "运行时目录必须在项目外，以避免污染真实项目文件和 Git 状态",
			Commands: []string{
				`export ADP_RUNTIME_DIR="/tmp/adp-runtime"`,
			},
			Notes: []string{
				"推荐使用 /tmp 或其他临时目录",
			},
		}

	case DiagnosticCodeRuntimeParentContainsProject:
		return &DiagnosticSuggestion{
			Reason: "运行时父目录不能包含项目根目录",
			Commands: []string{
				`export ADP_RUNTIME_DIR="/tmp/adp-runtime"`,
			},
		}

	case DiagnosticCodeRuntimeParentProjectRoot:
		return &DiagnosticSuggestion{
			Reason: "运行时目录不能与项目根目录相同",
			Commands: []string{
				`export ADP_RUNTIME_DIR="/tmp/adp-runtime"`,
			},
		}

	case DiagnosticCodeRuntimeParentRoot:
		return &DiagnosticSuggestion{
			Reason: "运行时目录不能是文件系统根目录",
			Commands: []string{
				`export ADP_RUNTIME_DIR="/tmp/adp-runtime"`,
			},
		}

	case DiagnosticCodeRuntimeParentNotDirectory:
		return &DiagnosticSuggestion{
			Reason: "运行时父目录必须是目录",
			Commands: []string{
				fmt.Sprintf("检查文件类型: file %s", ctx.Path),
				"删除该文件后设置正确的运行时目录",
			},
		}

	case DiagnosticCodeRuntimeParentSymlink:
		return &DiagnosticSuggestion{
			Reason: "建议使用直接目录而非符号链接",
			Notes: []string{
				"符号链接可以使用，但直接目录路径更清晰",
			},
		}

	// === 文件引用类 ===
	case DiagnosticCodePromptMissing:
		return &DiagnosticSuggestion{
			Commands: []string{
				fmt.Sprintf("快速开始: adp quickstart %s", ctx.Workspace),
				"或手动创建: mkdir -p prompts && vim prompts/base.md",
			},
			DocLink: "operator-onboarding.md#提示文件",
		}

	case DiagnosticCodePromptOutsideWorkspace:
		return &DiagnosticSuggestion{
			Reason: "提示文件路径必须在工作区目录内",
			Commands: []string{
				fmt.Sprintf("检查配置: cat %s", ctx.ConfigPath),
				"修改配置文件中的 prompts.base 为相对路径",
			},
		}

	case DiagnosticCodePromptNotFile:
		return &DiagnosticSuggestion{
			Reason: "提示路径必须指向文件而非目录",
			Commands: []string{
				fmt.Sprintf("检查路径: ls -ld %s", ctx.Path),
				"修改配置文件中的 prompts.base 为文件路径",
			},
		}

	case DiagnosticCodeMemorySharedNotConfigured:
		return &DiagnosticSuggestion{
			Reason: "共享内存已启用但未配置路径",
			Commands: []string{
				"在 workspace.yaml 中配置 memory.shared: \"memory/shared.md\"",
				"或禁用: memory.enabled: false",
			},
		}

	case DiagnosticCodeMemorySharedMissing:
		return &DiagnosticSuggestion{
			Commands: []string{
				"创建文件: mkdir -p memory && vim memory/shared.md",
				"或禁用: 在 workspace.yaml 中设置 memory.enabled: false",
			},
		}

	case DiagnosticCodeMemorySharedOutside:
		return &DiagnosticSuggestion{
			Reason: "共享内存文件必须在工作区目录内",
			Commands: []string{
				fmt.Sprintf("检查配置: cat %s", ctx.ConfigPath),
				"修改配置文件中的 memory.shared 为相对路径",
			},
		}

	case DiagnosticCodeMemorySharedNotFile:
		return &DiagnosticSuggestion{
			Reason: "共享内存路径必须指向文件而非目录",
			Commands: []string{
				fmt.Sprintf("检查路径: ls -ld %s", ctx.Path),
				"修改配置文件中的 memory.shared 为文件路径",
			},
		}

	case DiagnosticCodeMCPConfigNotConfigured:
		return &DiagnosticSuggestion{
			Reason: "MCP 已启用但未配置路径",
			Commands: []string{
				"在 workspace.yaml 中配置 mcp.config: \"mcp-config.json\"",
				"或禁用: mcp.enabled: false",
			},
		}

	case DiagnosticCodeMCPConfigMissing:
		return &DiagnosticSuggestion{
			Commands: []string{
				"创建配置: vim mcp-config.json",
				"或禁用: 在 workspace.yaml 中设置 mcp.enabled: false",
			},
			DocLink: "operator-onboarding.md#mcp-配置",
		}

	case DiagnosticCodeMCPConfigOutside:
		return &DiagnosticSuggestion{
			Reason: "MCP 配置文件必须在工作区目录内",
			Commands: []string{
				fmt.Sprintf("检查配置: cat %s", ctx.ConfigPath),
				"修改配置文件中的 mcp.config 为相对路径",
			},
		}

	case DiagnosticCodeMCPConfigNotFile:
		return &DiagnosticSuggestion{
			Reason: "MCP 配置路径必须指向文件而非目录",
			Commands: []string{
				fmt.Sprintf("检查路径: ls -ld %s", ctx.Path),
				"修改配置文件中的 mcp.config 为文件路径",
			},
		}

	// === Agent 配置类 ===
	case DiagnosticCodeAgentUnknown:
		return &DiagnosticSuggestion{
			Reason: "配置中引用了未知的 agent",
			Commands: []string{
				fmt.Sprintf("检查配置: cat %s", ctx.ConfigPath),
				"移除未使用的 agent 配置，或安装对应的 agent",
			},
		}

	case DiagnosticCodeAgentCommandDefault:
		return &DiagnosticSuggestion{
			Notes: []string{
				"这是信息提示：agent 使用内置的默认命令",
			},
		}

	case DiagnosticCodeAgentCommandMissing:
		return &DiagnosticSuggestion{
			Commands: []string{
				fmt.Sprintf("检查命令: which %s", ctx.AgentCommand),
				"安装对应的 agent 或修改配置文件中的命令路径",
			},
		}

	case DiagnosticCodeAgentCommandNotExecutable:
		return &DiagnosticSuggestion{
			Reason: "agent 命令文件不可执行",
			Commands: []string{
				fmt.Sprintf("添加执行权限: chmod +x %s", ctx.Path),
				fmt.Sprintf("检查文件: ls -l %s", ctx.Path),
			},
		}

	case DiagnosticCodeAgentProfileInvalid:
		return &DiagnosticSuggestion{
			Reason: "profile 路径格式无效",
			Commands: []string{
				fmt.Sprintf("检查配置: cat %s", ctx.ConfigPath),
				"修改为有效的 profile 路径（如 \"profiles/expert.md\"）",
			},
		}

	case DiagnosticCodeAgentProfileOutside:
		return &DiagnosticSuggestion{
			Reason: "profile 文件必须在工作区目录内",
			Commands: []string{
				fmt.Sprintf("检查配置: cat %s", ctx.ConfigPath),
				"修改配置文件中的 profile 为相对路径",
			},
		}

	case DiagnosticCodeAgentProfileMissing:
		return &DiagnosticSuggestion{
			Commands: []string{
				fmt.Sprintf("创建 profile: vim profiles/%s.md", ctx.ProfileName),
				"或使用默认: 在 workspace.yaml 中移除 profile 配置",
			},
		}

	case DiagnosticCodeAgentProfileNotFile:
		return &DiagnosticSuggestion{
			Reason: "profile 路径必须指向文件而非目录",
			Commands: []string{
				fmt.Sprintf("检查路径: ls -ld %s", ctx.Path),
				"修改配置文件中的 profile 为文件路径",
			},
		}

	// === Git 状态类 ===
	case DiagnosticCodeGitEnvRepositoryDirective:
		return &DiagnosticSuggestion{
			Reason: "检测到 Git 环境变量 (GIT_DIR, GIT_WORK_TREE 等)",
			Notes: []string{
				"ADP 会自动中和这些变量，无需手动操作",
				"如果仍有问题，可以手动 unset GIT_DIR GIT_WORK_TREE",
			},
		}

	case DiagnosticCodeGitRootAbsent:
		return &DiagnosticSuggestion{
			Reason: "项目根目录不在 Git 仓库中",
			Notes: []string{
				"这是提示信息，不影响 ADP 运行",
				"但 phase evidence 和 agent handoff 在 Git 仓库中更易于审计",
			},
		}

	case DiagnosticCodeGitRootDetected:
		return &DiagnosticSuggestion{
			Notes: []string{
				"这是正常状态，ADP 将使用 Git 上下文",
			},
		}

	case DiagnosticCodeGitRootNested:
		return &DiagnosticSuggestion{
			Notes: []string{
				"项目在仓库子目录中，ADP 会正确处理",
			},
		}

	case DiagnosticCodeGitMetadataFile:
		return &DiagnosticSuggestion{
			Notes: []string{
				"检测到 .git 文件（worktree 或 submodule）",
				"ADP 会排除此元数据，Git 命令应显式指定 ADP_PROJECT_ROOT",
			},
		}

	case DiagnosticCodeGitMetadataOther:
		return &DiagnosticSuggestion{
			Reason: "Git 元数据路径存在但不是标准的 .git 目录或文件",
			Commands: []string{
				fmt.Sprintf("检查元数据: ls -la %s", ctx.Path),
				"在依赖 Git 状态前检查项目根目录",
			},
		}

	case DiagnosticCodeGitStatusDirty:
		return &DiagnosticSuggestion{
			Reason: "项目有未提交的更改",
			Commands: []string{
				fmt.Sprintf("查看状态: git -C %s status", ctx.ProjectRoot),
				fmt.Sprintf("查看差异: git -C %s diff", ctx.ProjectRoot),
			},
			Notes: []string{
				"这是提示信息，不影响 ADP 运行",
			},
		}

	case DiagnosticCodeGitStatusUnavailable:
		return &DiagnosticSuggestion{
			Reason: "检测到 Git 仓库但无法读取状态",
			Commands: []string{
				fmt.Sprintf("尝试手动检查: git -C %s status", ctx.ProjectRoot),
			},
		}

	default:
		return nil
	}
}
