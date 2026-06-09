package workspace

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/karoc/adp/internal/schema"
)

func TestRegistryDiagnoseHealthyWorkspace(t *testing.T) {
	registry, layout := newTestRegistry(t)
	projectRoot := createProject(t)

	if _, err := registry.Add(context.Background(), "game-a", projectRoot); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	report, err := registry.Diagnose(context.Background(), "game-a")
	if err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}
	if report.Workspace != "game-a" {
		t.Fatalf("Workspace = %q, want game-a", report.Workspace)
	}
	if report.WorkspaceDir != layout.WorkspaceDir("game-a") {
		t.Fatalf("WorkspaceDir = %q, want %q", report.WorkspaceDir, layout.WorkspaceDir("game-a"))
	}
	if report.ConfigPath != layout.WorkspaceConfig("game-a") {
		t.Fatalf("ConfigPath = %q, want %q", report.ConfigPath, layout.WorkspaceConfig("game-a"))
	}
	if len(report.Diagnostics) != 0 {
		t.Fatalf("Diagnostics = %+v, want none", report.Diagnostics)
	}
	if report.HasErrors() {
		t.Fatal("HasErrors() = true, want false")
	}
}

func TestRegistryDiagnoseReportsConfiguredResourceIssues(t *testing.T) {
	registry, layout := newTestRegistry(t)
	projectRoot := createProject(t)

	cfg, err := registry.Add(context.Background(), "game-a", projectRoot)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	workspaceDir := layout.WorkspaceDir("game-a")
	removeFile(t, filepath.Join(workspaceDir, "prompts", "base.md"))
	removeFile(t, filepath.Join(workspaceDir, "memory", "shared.md"))
	removeFile(t, filepath.Join(workspaceDir, "mcp", "config.yaml"))
	removeDir(t, projectRoot)

	codex := cfg.Agents["codex"]
	codex.Command = ""
	codex.Profile = "senior"
	cfg.Agents["codex"] = codex
	if err := schema.SaveConfig(layout.WorkspaceConfig("game-a"), cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	report, err := registry.Diagnose(context.Background(), "game-a")
	if err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}

	assertDiagnostic(t, report, DiagnosticCodeProjectRootMissing, DiagnosticLevelError, projectRoot)
	assertDiagnostic(t, report, DiagnosticCodePromptMissing, DiagnosticLevelWarning, filepath.Join(workspaceDir, "prompts", "base.md"))
	assertDiagnostic(t, report, DiagnosticCodeMemorySharedMissing, DiagnosticLevelWarning, filepath.Join(workspaceDir, "memory", "shared.md"))
	assertDiagnostic(t, report, DiagnosticCodeMCPConfigMissing, DiagnosticLevelWarning, filepath.Join(workspaceDir, "mcp", "config.yaml"))
	assertDiagnostic(t, report, DiagnosticCodeAgentCommandDefault, DiagnosticLevelInfo, layout.WorkspaceConfig("game-a"))
	assertDiagnostic(t, report, DiagnosticCodeAgentProfileMissing, DiagnosticLevelWarning, filepath.Join(workspaceDir, "profiles", "senior.{md,yaml,yml,json}"))
	if !report.HasErrors() {
		t.Fatal("HasErrors() = false, want true")
	}
}

func TestRegistryDiagnoseReportsAgentCommandReadinessIssues(t *testing.T) {
	registry, layout := newTestRegistry(t)
	projectRoot := createProject(t)

	cfg, err := registry.Add(context.Background(), "game-a", projectRoot)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	binDir := filepath.Join(projectRoot, "bin")
	if err := os.Mkdir(binDir, 0o755); err != nil {
		t.Fatalf("create bin dir: %v", err)
	}
	notExecutable := filepath.Join(binDir, "not-executable")
	writeFile(t, notExecutable, "#!/bin/sh\n")

	codex := cfg.Agents["codex"]
	codex.Command = "codex --model test"
	cfg.Agents["codex"] = codex
	claude := cfg.Agents["claude"]
	claude.Command = "bin/missing-claude"
	cfg.Agents["claude"] = claude
	cfg.Agents["local"] = schema.AgentConfig{
		Enabled: true,
		Profile: "default",
		Command: "bin/not-executable",
	}
	saveWorkspaceConfig(t, layout.WorkspaceConfig("game-a"), cfg)

	report, err := registry.Diagnose(context.Background(), "game-a")
	if err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}

	assertDiagnostic(t, report, DiagnosticCodeAgentCommandArguments, DiagnosticLevelWarning, layout.WorkspaceConfig("game-a"))
	assertDiagnostic(t, report, DiagnosticCodeAgentCommandMissing, DiagnosticLevelWarning, filepath.Join(projectRoot, "bin", "missing-claude"))
	assertDiagnostic(t, report, DiagnosticCodeAgentCommandNotExecutable, DiagnosticLevelWarning, notExecutable)
	assertDiagnostic(t, report, DiagnosticCodeAgentUnknown, DiagnosticLevelWarning, layout.WorkspaceConfig("game-a"))
}

func TestRegistryDiagnoseReportsProjectReservedPaths(t *testing.T) {
	registry, _ := newTestRegistry(t)
	projectRoot := createProject(t)
	writeFile(t, filepath.Join(projectRoot, "AGENTS.md"), "# project agents\n")
	writeFile(t, filepath.Join(projectRoot, "CLAUDE.md"), "# project claude\n")
	if err := os.Mkdir(filepath.Join(projectRoot, ".codex"), 0o755); err != nil {
		t.Fatalf("create .codex: %v", err)
	}
	writeFile(t, filepath.Join(projectRoot, ".codex", "config.toml"), "project codex\n")
	if err := os.Mkdir(filepath.Join(projectRoot, "planning"), 0o755); err != nil {
		t.Fatalf("create planning: %v", err)
	}

	cfg, err := registry.Add(context.Background(), "game-a", projectRoot)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	claude := cfg.Agents["claude"]
	claude.Enabled = false
	cfg.Agents["claude"] = claude
	saveWorkspaceConfig(t, registry.Layout.WorkspaceConfig("game-a"), cfg)

	report, err := registry.Diagnose(context.Background(), "game-a")
	if err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}

	assertDiagnostic(t, report, DiagnosticCodeProjectRootReservedPath, DiagnosticLevelWarning, filepath.Join(projectRoot, "AGENTS.md"))
	assertDiagnostic(t, report, DiagnosticCodeProjectRootReservedPath, DiagnosticLevelWarning, filepath.Join(projectRoot, ".codex", "config.toml"))
	assertDiagnostic(t, report, DiagnosticCodeProjectRootReservedPath, DiagnosticLevelWarning, filepath.Join(projectRoot, "planning"))
	assertNoDiagnosticPath(t, report, DiagnosticCodeProjectRootReservedPath, filepath.Join(projectRoot, "CLAUDE.md"))
}

func TestRegistryDiagnoseAllowsProviderLocalProjectFiles(t *testing.T) {
	registry, _ := newTestRegistry(t)
	projectRoot := createProject(t)
	if err := os.Mkdir(filepath.Join(projectRoot, ".codex"), 0o755); err != nil {
		t.Fatalf("create .codex: %v", err)
	}
	if err := os.Mkdir(filepath.Join(projectRoot, ".claude"), 0o755); err != nil {
		t.Fatalf("create .claude: %v", err)
	}
	writeFile(t, filepath.Join(projectRoot, ".codex", "local.toml"), "project local codex\n")
	writeFile(t, filepath.Join(projectRoot, ".claude", "settings.local.json"), "project local claude\n")

	if _, err := registry.Add(context.Background(), "game-a", projectRoot); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	report, err := registry.Diagnose(context.Background(), "game-a")
	if err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}

	assertNoDiagnosticPath(t, report, DiagnosticCodeProjectRootReservedPath, filepath.Join(projectRoot, ".codex"))
	assertNoDiagnosticPath(t, report, DiagnosticCodeProjectRootReservedPath, filepath.Join(projectRoot, ".codex", "local.toml"))
	assertNoDiagnosticPath(t, report, DiagnosticCodeProjectRootReservedPath, filepath.Join(projectRoot, ".claude"))
	assertNoDiagnosticPath(t, report, DiagnosticCodeProjectRootReservedPath, filepath.Join(projectRoot, ".claude", "settings.local.json"))
}

func TestRegistryDiagnoseReportsSymlinkResourceEscapes(t *testing.T) {
	registry, layout := newTestRegistry(t)
	projectRoot := createProject(t)

	cfg, err := registry.Add(context.Background(), "game-a", projectRoot)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	externalDir := t.TempDir()
	externalPrompt := filepath.Join(externalDir, "base.md")
	externalMemory := filepath.Join(externalDir, "shared.md")
	externalMCP := filepath.Join(externalDir, "config.yaml")
	externalProfile := filepath.Join(externalDir, "senior.yaml")
	writeFile(t, externalPrompt, "# external prompt\n")
	writeFile(t, externalMemory, "# external memory\n")
	writeFile(t, externalMCP, "enabled: true\nservers: []\n")
	writeFile(t, externalProfile, "profile: external\n")

	workspaceDir := layout.WorkspaceDir("game-a")
	promptPath := filepath.Join(workspaceDir, "prompts", "base.md")
	memoryPath := filepath.Join(workspaceDir, "memory", "shared.md")
	mcpPath := filepath.Join(workspaceDir, "mcp", "config.yaml")
	profilePath := filepath.Join(workspaceDir, "profiles", "senior.yaml")
	removeFile(t, promptPath)
	removeFile(t, memoryPath)
	removeFile(t, mcpPath)
	symlinkOrSkip(t, externalPrompt, promptPath)
	symlinkOrSkip(t, externalMemory, memoryPath)
	symlinkOrSkip(t, externalMCP, mcpPath)
	symlinkOrSkip(t, externalProfile, profilePath)

	codex := cfg.Agents["codex"]
	codex.Profile = "senior"
	cfg.Agents["codex"] = codex
	saveWorkspaceConfig(t, layout.WorkspaceConfig("game-a"), cfg)

	report, err := registry.Diagnose(context.Background(), "game-a")
	if err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}

	assertDiagnostic(t, report, DiagnosticCodePromptOutsideWorkspace, DiagnosticLevelWarning, promptPath)
	assertDiagnostic(t, report, DiagnosticCodeMemorySharedOutside, DiagnosticLevelWarning, memoryPath)
	assertDiagnostic(t, report, DiagnosticCodeMCPConfigOutside, DiagnosticLevelWarning, mcpPath)
	assertDiagnostic(t, report, DiagnosticCodeAgentProfileOutside, DiagnosticLevelWarning, profilePath)
}

func TestRegistryDiagnoseAllowsSymlinkResourcesInsideWorkspace(t *testing.T) {
	registry, layout := newTestRegistry(t)
	projectRoot := createProject(t)

	cfg, err := registry.Add(context.Background(), "game-a", projectRoot)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	workspaceDir := layout.WorkspaceDir("game-a")
	internalPrompt := filepath.Join(workspaceDir, "prompts", "internal.md")
	internalMemory := filepath.Join(workspaceDir, "memory", "internal.md")
	internalMCP := filepath.Join(workspaceDir, "mcp", "internal.yaml")
	internalProfile := filepath.Join(workspaceDir, "profiles", "senior-internal.yaml")
	writeFile(t, internalPrompt, "# internal prompt\n")
	writeFile(t, internalMemory, "# internal memory\n")
	writeFile(t, internalMCP, "enabled: true\nservers: []\n")
	writeFile(t, internalProfile, "profile: internal\n")

	promptPath := filepath.Join(workspaceDir, "prompts", "base.md")
	memoryPath := filepath.Join(workspaceDir, "memory", "shared.md")
	mcpPath := filepath.Join(workspaceDir, "mcp", "config.yaml")
	profilePath := filepath.Join(workspaceDir, "profiles", "senior.yaml")
	removeFile(t, promptPath)
	removeFile(t, memoryPath)
	removeFile(t, mcpPath)
	symlinkOrSkip(t, internalPrompt, promptPath)
	symlinkOrSkip(t, internalMemory, memoryPath)
	symlinkOrSkip(t, internalMCP, mcpPath)
	symlinkOrSkip(t, internalProfile, profilePath)

	codex := cfg.Agents["codex"]
	codex.Profile = "senior"
	cfg.Agents["codex"] = codex
	saveWorkspaceConfig(t, layout.WorkspaceConfig("game-a"), cfg)

	report, err := registry.Diagnose(context.Background(), "game-a")
	if err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}
	if len(report.Diagnostics) != 0 {
		t.Fatalf("Diagnostics = %+v, want none", report.Diagnostics)
	}
}

func TestRegistryDiagnoseReportsProfileDirectoryAsNotFile(t *testing.T) {
	registry, layout := newTestRegistry(t)
	projectRoot := createProject(t)

	cfg, err := registry.Add(context.Background(), "game-a", projectRoot)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	workspaceDir := layout.WorkspaceDir("game-a")
	profileDir := filepath.Join(workspaceDir, "profiles", "senior.yaml")
	if err := os.Mkdir(profileDir, 0o755); err != nil {
		t.Fatalf("create profile directory: %v", err)
	}
	codex := cfg.Agents["codex"]
	codex.Profile = "senior"
	cfg.Agents["codex"] = codex
	saveWorkspaceConfig(t, layout.WorkspaceConfig("game-a"), cfg)

	report, err := registry.Diagnose(context.Background(), "game-a")
	if err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}

	assertDiagnostic(t, report, DiagnosticCodeAgentProfileNotFile, DiagnosticLevelWarning, profileDir)
}

func TestRegistryDiagnoseReportsProfileConsistencyIssues(t *testing.T) {
	tests := []struct {
		name    string
		profile string
		setup   func(t *testing.T, workspaceDir string)
		code    string
		path    func(workspaceDir string) string
	}{
		{
			name:    "invalid-profile-name",
			profile: "../senior",
			code:    DiagnosticCodeAgentProfileInvalid,
			path:    func(workspaceDir string) string { return filepath.Join(workspaceDir, "workspace.yaml") },
		},
		{
			name:    "ambiguous-profile-files",
			profile: "senior",
			setup: func(t *testing.T, workspaceDir string) {
				writeFile(t, filepath.Join(workspaceDir, "profiles", "senior.md"), "# senior\n")
				writeFile(t, filepath.Join(workspaceDir, "profiles", "senior.yaml"), "profile: senior\n")
			},
			code: DiagnosticCodeAgentProfileAmbiguous,
			path: func(workspaceDir string) string {
				return filepath.Join(workspaceDir, "profiles", "senior.{md,yaml,yml,json}")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry, layout := newTestRegistry(t)
			projectRoot := createProject(t)

			cfg, err := registry.Add(context.Background(), "game-a", projectRoot)
			if err != nil {
				t.Fatalf("Add() error = %v", err)
			}
			workspaceDir := layout.WorkspaceDir("game-a")
			if tt.setup != nil {
				tt.setup(t, workspaceDir)
			}
			codex := cfg.Agents["codex"]
			codex.Profile = tt.profile
			cfg.Agents["codex"] = codex
			saveWorkspaceConfig(t, layout.WorkspaceConfig("game-a"), cfg)

			report, err := registry.Diagnose(context.Background(), "game-a")
			if err != nil {
				t.Fatalf("Diagnose() error = %v", err)
			}

			assertDiagnostic(t, report, tt.code, DiagnosticLevelWarning, tt.path(workspaceDir))
		})
	}
}

func TestRegistryDiagnoseReportsWorkspaceDirectorySymlink(t *testing.T) {
	registry, layout := newTestRegistry(t)

	if err := registry.Init(context.Background()); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	targetDir := filepath.Join(t.TempDir(), "game-a-target")
	if err := os.Mkdir(targetDir, 0o755); err != nil {
		t.Fatalf("create target workspace dir: %v", err)
	}
	workspaceDir := layout.WorkspaceDir("game-a")
	symlinkOrSkip(t, targetDir, workspaceDir)

	report, err := registry.Diagnose(context.Background(), "game-a")
	if err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}

	assertDiagnostic(t, report, DiagnosticCodeWorkspaceDirSymlink, DiagnosticLevelError, workspaceDir)
	if !report.HasErrors() {
		t.Fatal("HasErrors() = false, want true")
	}
}

func TestRegistryDiagnoseAllContinuesAcrossInvalidWorkspaceConfig(t *testing.T) {
	registry, layout := newTestRegistry(t)
	projectRoot := createProject(t)

	if _, err := registry.Add(context.Background(), "good", projectRoot); err != nil {
		t.Fatalf("Add() good error = %v", err)
	}
	badDir := layout.WorkspaceDir("bad")
	if err := os.MkdirAll(badDir, 0o755); err != nil {
		t.Fatalf("create bad workspace dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(badDir, "workspace.yaml"), []byte("version: 1\nworkspace: [bad\n"), 0o644); err != nil {
		t.Fatalf("write bad workspace config: %v", err)
	}

	reports, err := registry.DiagnoseAll(context.Background())
	if err != nil {
		t.Fatalf("DiagnoseAll() error = %v", err)
	}
	if len(reports) != 2 {
		t.Fatalf("DiagnoseAll() returned %d reports, want 2: %+v", len(reports), reports)
	}

	bad := reportByWorkspace(t, reports, "bad")
	assertDiagnostic(t, bad, DiagnosticCodeConfigInvalid, DiagnosticLevelError, filepath.Join(badDir, "workspace.yaml"))
	good := reportByWorkspace(t, reports, "good")
	if len(good.Diagnostics) != 0 {
		t.Fatalf("good diagnostics = %+v, want none", good.Diagnostics)
	}
}

func TestRegistryDiagnoseAllReportsInvalidWorkspaceDirectoryNames(t *testing.T) {
	registry, layout := newTestRegistry(t)

	if err := registry.Init(context.Background()); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	invalidDir := filepath.Join(layout.WorkspacesDir, "bad name")
	if err := os.MkdirAll(invalidDir, 0o755); err != nil {
		t.Fatalf("create invalid workspace dir: %v", err)
	}

	reports, err := registry.DiagnoseAll(context.Background())
	if err != nil {
		t.Fatalf("DiagnoseAll() error = %v", err)
	}
	report := reportByWorkspace(t, reports, "bad name")
	assertDiagnostic(t, report, DiagnosticCodeWorkspaceNameInvalid, DiagnosticLevelError, invalidDir)
	assertDiagnostic(t, report, DiagnosticCodeConfigMissing, DiagnosticLevelError, filepath.Join(invalidDir, "workspace.yaml"))
}

func TestRegistryDiagnoseAllReportsFileEntries(t *testing.T) {
	registry, layout := newTestRegistry(t)

	if err := registry.Init(context.Background()); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	filePath := filepath.Join(layout.WorkspacesDir, "not-a-workspace")
	writeFile(t, filePath, "plain file\n")

	reports, err := registry.DiagnoseAll(context.Background())
	if err != nil {
		t.Fatalf("DiagnoseAll() error = %v", err)
	}
	report := reportByWorkspace(t, reports, "not-a-workspace")
	assertDiagnostic(t, report, DiagnosticCodeWorkspaceDirNotDirectory, DiagnosticLevelError, filePath)
}

func TestRegistryDiagnoseRespectsContextCancellation(t *testing.T) {
	registry, _ := newTestRegistry(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := registry.Diagnose(ctx, "game-a")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Diagnose() error = %v, want context.Canceled", err)
	}

	_, err = registry.DiagnoseAll(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("DiagnoseAll() error = %v, want context.Canceled", err)
	}
}

func assertDiagnostic(t *testing.T, report DiagnosticReport, code string, level DiagnosticLevel, path string) {
	t.Helper()

	foundCode := false
	for _, diagnostic := range report.Diagnostics {
		if diagnostic.Code != code {
			continue
		}
		foundCode = true
		if diagnostic.Path != path {
			continue
		}
		if diagnostic.Level != level {
			t.Fatalf("diagnostic %s level = %q, want %q", code, diagnostic.Level, level)
		}
		if diagnostic.Message == "" {
			t.Fatalf("diagnostic %s message is empty", code)
		}
		return
	}
	if foundCode {
		t.Fatalf("diagnostic %s found, but not with path %q in %+v", code, path, report.Diagnostics)
	}
	t.Fatalf("diagnostic %s not found in %+v", code, report.Diagnostics)
}

func assertNoDiagnosticPath(t *testing.T, report DiagnosticReport, code string, path string) {
	t.Helper()

	for _, diagnostic := range report.Diagnostics {
		if diagnostic.Code == code && diagnostic.Path == path {
			t.Fatalf("unexpected diagnostic %s with path %q in %+v", code, path, report.Diagnostics)
		}
	}
}

func reportByWorkspace(t *testing.T, reports []DiagnosticReport, workspace string) DiagnosticReport {
	t.Helper()

	for _, report := range reports {
		if report.Workspace == workspace {
			return report
		}
	}
	t.Fatalf("workspace report %q not found in %+v", workspace, reports)
	return DiagnosticReport{}
}

func removeFile(t *testing.T, path string) {
	t.Helper()

	if err := os.Remove(path); err != nil {
		t.Fatalf("remove file %s: %v", path, err)
	}
}

func removeDir(t *testing.T, path string) {
	t.Helper()

	if err := os.Remove(path); err != nil {
		t.Fatalf("remove dir %s: %v", path, err)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}

func symlinkOrSkip(t *testing.T, oldname string, newname string) {
	t.Helper()

	if err := os.Symlink(oldname, newname); err != nil {
		t.Skipf("symlink not available: %v", err)
	}
}

func saveWorkspaceConfig(t *testing.T, path string, cfg *schema.Config) {
	t.Helper()

	if err := schema.SaveConfig(path, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
}
