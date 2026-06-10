package workspace

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/karoc/adp/internal/schema"
)

func TestRegistryDiagnoseReportsConfiguredResourcePathIssues(t *testing.T) {
	tests := []struct {
		name      string
		configure func(t *testing.T, cfg *schema.Config, workspaceDir string, externalPath string)
		code      string
		path      func(workspaceDir string, externalPath string) string
	}{
		{
			name: "prompt-relative-escape",
			configure: func(_ *testing.T, cfg *schema.Config, _ string, _ string) {
				cfg.Prompts.Base = "../base.md"
			},
			code: DiagnosticCodePromptOutsideWorkspace,
			path: func(workspaceDir string, _ string) string {
				return filepath.Join(workspaceDir, "..", "base.md")
			},
		},
		{
			name: "prompt-not-file",
			configure: func(t *testing.T, cfg *schema.Config, workspaceDir string, _ string) {
				promptPath := filepath.Join(workspaceDir, "prompts", "base.md")
				removeFile(t, promptPath)
				if err := os.Mkdir(promptPath, 0o755); err != nil {
					t.Fatalf("create prompt directory: %v", err)
				}
				cfg.Prompts.Base = "prompts/base.md"
			},
			code: DiagnosticCodePromptNotFile,
			path: func(workspaceDir string, _ string) string {
				return filepath.Join(workspaceDir, "prompts", "base.md")
			},
		},
		{
			name: "memory-not-configured",
			configure: func(_ *testing.T, cfg *schema.Config, _ string, _ string) {
				cfg.Memory.Enabled = true
				cfg.Memory.Shared = " "
			},
			code: DiagnosticCodeMemorySharedNotConfigured,
			path: func(workspaceDir string, _ string) string {
				return filepath.Join(workspaceDir, "workspace.yaml")
			},
		},
		{
			name: "memory-absolute-escape",
			configure: func(_ *testing.T, cfg *schema.Config, _ string, externalPath string) {
				cfg.Memory.Enabled = true
				cfg.Memory.Shared = externalPath
			},
			code: DiagnosticCodeMemorySharedOutside,
			path: func(_ string, externalPath string) string {
				return externalPath
			},
		},
		{
			name: "memory-not-file",
			configure: func(t *testing.T, cfg *schema.Config, workspaceDir string, _ string) {
				memoryPath := filepath.Join(workspaceDir, "memory", "shared.md")
				removeFile(t, memoryPath)
				if err := os.Mkdir(memoryPath, 0o755); err != nil {
					t.Fatalf("create memory directory: %v", err)
				}
				cfg.Memory.Enabled = true
				cfg.Memory.Shared = "memory/shared.md"
			},
			code: DiagnosticCodeMemorySharedNotFile,
			path: func(workspaceDir string, _ string) string {
				return filepath.Join(workspaceDir, "memory", "shared.md")
			},
		},
		{
			name: "mcp-not-configured",
			configure: func(_ *testing.T, cfg *schema.Config, _ string, _ string) {
				cfg.MCP.Enabled = true
				cfg.MCP.Config = ""
			},
			code: DiagnosticCodeMCPConfigNotConfigured,
			path: func(workspaceDir string, _ string) string {
				return filepath.Join(workspaceDir, "workspace.yaml")
			},
		},
		{
			name: "mcp-relative-escape",
			configure: func(_ *testing.T, cfg *schema.Config, _ string, _ string) {
				cfg.MCP.Enabled = true
				cfg.MCP.Config = "../mcp.yaml"
			},
			code: DiagnosticCodeMCPConfigOutside,
			path: func(workspaceDir string, _ string) string {
				return filepath.Join(workspaceDir, "..", "mcp.yaml")
			},
		},
		{
			name: "mcp-not-file",
			configure: func(t *testing.T, cfg *schema.Config, workspaceDir string, _ string) {
				mcpPath := filepath.Join(workspaceDir, "mcp", "config.yaml")
				removeFile(t, mcpPath)
				if err := os.Mkdir(mcpPath, 0o755); err != nil {
					t.Fatalf("create MCP directory: %v", err)
				}
				cfg.MCP.Enabled = true
				cfg.MCP.Config = "mcp/config.yaml"
			},
			code: DiagnosticCodeMCPConfigNotFile,
			path: func(workspaceDir string, _ string) string {
				return filepath.Join(workspaceDir, "mcp", "config.yaml")
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
			externalPath := filepath.Join(t.TempDir(), "external-resource.yaml")
			writeFile(t, externalPath, "external: true\n")
			tt.configure(t, cfg, workspaceDir, externalPath)
			saveWorkspaceConfig(t, layout.WorkspaceConfig("game-a"), cfg)

			report, err := registry.Diagnose(context.Background(), "game-a")
			if err != nil {
				t.Fatalf("Diagnose() error = %v", err)
			}

			assertDiagnostic(t, report, tt.code, DiagnosticLevelWarning, tt.path(workspaceDir, externalPath))
		})
	}
}

func TestRegistryDiagnoseReportsProfilePathIssueMatrix(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, workspaceDir string, externalPath string)
		code    string
		path    func(workspaceDir string) string
		profile string
	}{
		{
			name: "missing",
			code: DiagnosticCodeAgentProfileMissing,
			path: func(workspaceDir string) string {
				return filepath.Join(workspaceDir, "profiles", "senior.{md,yaml,yml,json}")
			},
			profile: "senior",
		},
		{
			name: "ambiguous",
			setup: func(t *testing.T, workspaceDir string, _ string) {
				writeFile(t, filepath.Join(workspaceDir, "profiles", "senior.md"), "# senior\n")
				writeFile(t, filepath.Join(workspaceDir, "profiles", "senior.yaml"), "profile: senior\n")
			},
			code: DiagnosticCodeAgentProfileAmbiguous,
			path: func(workspaceDir string) string {
				return filepath.Join(workspaceDir, "profiles", "senior.{md,yaml,yml,json}")
			},
			profile: "senior",
		},
		{
			name: "not-file",
			setup: func(t *testing.T, workspaceDir string, _ string) {
				profilePath := filepath.Join(workspaceDir, "profiles", "senior.yaml")
				if err := os.Mkdir(profilePath, 0o755); err != nil {
					t.Fatalf("create profile directory: %v", err)
				}
			},
			code: DiagnosticCodeAgentProfileNotFile,
			path: func(workspaceDir string) string {
				return filepath.Join(workspaceDir, "profiles", "senior.yaml")
			},
			profile: "senior",
		},
		{
			name: "symlink-escape",
			setup: func(t *testing.T, workspaceDir string, externalPath string) {
				profilePath := filepath.Join(workspaceDir, "profiles", "senior.yaml")
				symlinkOrSkip(t, externalPath, profilePath)
			},
			code: DiagnosticCodeAgentProfileOutside,
			path: func(workspaceDir string) string {
				return filepath.Join(workspaceDir, "profiles", "senior.yaml")
			},
			profile: "senior",
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
			externalPath := filepath.Join(t.TempDir(), "senior.yaml")
			writeFile(t, externalPath, "profile: external\n")
			if tt.setup != nil {
				tt.setup(t, workspaceDir, externalPath)
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

func TestRegistryDiagnoseReportsEveryReservedProjectRootPath(t *testing.T) {
	registry, _ := newTestRegistry(t)
	projectRoot := createProject(t)
	reservedPaths := []struct {
		rel string
		dir bool
	}{
		{rel: ".adp-runtime.yaml"},
		{rel: "planning", dir: true},
		{rel: "tasks.yaml"},
		{rel: "phases.yaml"},
		{rel: "progress.jsonl"},
		{rel: "AGENTS.md"},
		{rel: filepath.Join(".codex", "config.toml")},
		{rel: "CLAUDE.md"},
		{rel: filepath.Join(".claude", "settings.json")},
	}
	for _, reserved := range reservedPaths {
		path := filepath.Join(projectRoot, reserved.rel)
		if reserved.dir {
			if err := os.Mkdir(path, 0o755); err != nil {
				t.Fatalf("create reserved directory %s: %v", reserved.rel, err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create parent for reserved path %s: %v", reserved.rel, err)
		}
		writeFile(t, path, "reserved\n")
	}

	if _, err := registry.Add(context.Background(), "game-a", projectRoot); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	report, err := registry.Diagnose(context.Background(), "game-a")
	if err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}

	for _, reserved := range reservedPaths {
		assertDiagnostic(t, report, DiagnosticCodeProjectRootReservedPath, DiagnosticLevelWarning, filepath.Join(projectRoot, reserved.rel))
	}
}

func TestRegistryDiagnoseDoesNotCreateMissingResourcesOrRuntimeParent(t *testing.T) {
	registry, layout := newTestRegistry(t)
	projectRoot := createProject(t)
	missingRuntimeParent := filepath.Join(t.TempDir(), "missing-runtime-parent")
	registry.Layout.RuntimeParent = missingRuntimeParent
	cfg, err := registry.Add(context.Background(), "game-a", projectRoot)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	workspaceDir := layout.WorkspaceDir("game-a")
	missingResources := []string{
		filepath.Join(workspaceDir, "prompts", "base.md"),
		filepath.Join(workspaceDir, "memory", "shared.md"),
		filepath.Join(workspaceDir, "mcp", "config.yaml"),
	}
	for _, path := range missingResources {
		removeFile(t, path)
	}
	saveWorkspaceConfig(t, layout.WorkspaceConfig("game-a"), cfg)
	beforeConfig, err := os.ReadFile(layout.WorkspaceConfig("game-a"))
	if err != nil {
		t.Fatalf("read workspace config before Diagnose: %v", err)
	}

	report, err := registry.Diagnose(context.Background(), "game-a")
	if err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}

	assertDiagnostic(t, report, DiagnosticCodePromptMissing, DiagnosticLevelWarning, missingResources[0])
	assertDiagnostic(t, report, DiagnosticCodeMemorySharedMissing, DiagnosticLevelWarning, missingResources[1])
	assertDiagnostic(t, report, DiagnosticCodeMCPConfigMissing, DiagnosticLevelWarning, missingResources[2])
	for _, path := range missingResources {
		assertPathMissing(t, path)
	}
	assertPathMissing(t, missingRuntimeParent)
	afterConfig, err := os.ReadFile(layout.WorkspaceConfig("game-a"))
	if err != nil {
		t.Fatalf("read workspace config after Diagnose: %v", err)
	}
	if !bytes.Equal(afterConfig, beforeConfig) {
		t.Fatal("Diagnose() changed workspace config contents")
	}
}

func TestRegistryDiagnoseDoesNotExecuteConfiguredCommandWrapper(t *testing.T) {
	clearGitDirectiveEnv(t)

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
	sentinel := filepath.Join(projectRoot, "wrapper-ran")
	wrapperPath := filepath.Join(binDir, "provider wrapper")
	wrapper := "#!/bin/sh\n" + "touch " + sentinel + "\n"
	if err := os.WriteFile(wrapperPath, []byte(wrapper), 0o755); err != nil {
		t.Fatalf("write command wrapper: %v", err)
	}

	codex := cfg.Agents["codex"]
	codex.Command = filepath.Join("bin", "provider wrapper")
	cfg.Agents["codex"] = codex
	claude := cfg.Agents["claude"]
	claude.Enabled = false
	cfg.Agents["claude"] = claude
	saveWorkspaceConfig(t, layout.WorkspaceConfig("game-a"), cfg)

	report, err := registry.Diagnose(context.Background(), "game-a")
	if err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}

	assertPathMissing(t, sentinel)
	if len(report.Diagnostics) != 0 {
		t.Fatalf("Diagnostics = %+v, want none", report.Diagnostics)
	}
}

func assertPathMissing(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); err == nil {
		t.Fatalf("%s exists, want missing", path)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", path, err)
	}
}
