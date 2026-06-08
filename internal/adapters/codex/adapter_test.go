package codex

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/karoc/adp/internal/adapters/api"
	"github.com/karoc/adp/internal/schema"
)

func TestRenderFilesIncludeWorkspaceSources(t *testing.T) {
	workspaceDir := t.TempDir()
	writeFile(t, workspaceDir, "prompts/base.md", "Base prompt body.\n")
	writeFile(t, workspaceDir, "memory/shared.md", "Shared memory body.\n")
	writeFile(t, workspaceDir, "mcp/config.yaml", "servers:\n  github: {}\n")
	writeFile(t, workspaceDir, "profiles/senior-engineer.yaml", "tone: direct\n")

	adapter := New()
	result, err := adapter.Render(context.Background(), codexContext(workspaceDir))
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	agents := generatedText(t, result, "AGENTS.md")
	for _, want := range []string{
		"Base prompt body.",
		"Shared memory body.",
		"- coding_style: strict",
		"- github",
		"- postgres",
		"tone: direct",
	} {
		if !strings.Contains(agents, want) {
			t.Fatalf("AGENTS.md missing %q:\n%s", want, agents)
		}
	}

	config := generatedText(t, result, ".codex/config.toml")
	for _, want := range []string{
		`adapter = "codex"`,
		`workspace = "demo"`,
		`profile = "senior-engineer"`,
		"mcp_enabled = true",
	} {
		if !strings.Contains(config, want) {
			t.Fatalf(".codex/config.toml missing %q:\n%s", want, config)
		}
	}
}

func TestLaunchUsesCommandOverrideAndExtraArgs(t *testing.T) {
	adapter := New()
	ctx := codexContext(t.TempDir())
	extraArgs := []string{"--ask-for-approval", "never"}

	spec, err := adapter.Launch(context.Background(), ctx, api.RuntimeHandle{
		SessionID: "session-1",
		Root:      "/runtime/root",
		Env:       map[string]string{"EXISTING": "1"},
	}, extraArgs)
	if err != nil {
		t.Fatalf("Launch() error = %v", err)
	}

	extraArgs[0] = "changed"
	if spec.Command != "codex-dev" {
		t.Fatalf("Command = %q, want codex-dev", spec.Command)
	}
	if spec.Dir != "/runtime/root" {
		t.Fatalf("Dir = %q, want /runtime/root", spec.Dir)
	}
	if !reflect.DeepEqual(spec.Args, []string{"--ask-for-approval", "never"}) {
		t.Fatalf("Args = %v", spec.Args)
	}
	if spec.Env["EXISTING"] != "1" || spec.Env["ADP_AGENT"] != "codex" || spec.Env["ADP_SESSION_ID"] != "session-1" {
		t.Fatalf("Env missing expected values: %v", spec.Env)
	}
}

func TestRenderIncludesTaskContext(t *testing.T) {
	adapter := New()
	ctx := codexContext(t.TempDir())
	ctx.Task = api.TaskContext{
		ID:          "task-20260608-0001",
		Title:       "Bind runtime session to task",
		Status:      "ready",
		Priority:    "high",
		Phase:       "p1",
		Description: "Runtime binding smoke.",
	}

	result, err := adapter.Render(context.Background(), ctx)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	agents := generatedText(t, result, "AGENTS.md")
	for _, want := range []string{"## Current Task", "task-20260608-0001", "Bind runtime session to task", "Runtime binding smoke."} {
		if !strings.Contains(agents, want) {
			t.Fatalf("AGENTS.md missing task text %q:\n%s", want, agents)
		}
	}

	config := generatedText(t, result, ".codex/config.toml")
	for _, want := range []string{`task_id = "task-20260608-0001"`, `task_title = "Bind runtime session to task"`} {
		if !strings.Contains(config, want) {
			t.Fatalf(".codex/config.toml missing task text %q:\n%s", want, config)
		}
	}
	if result.Env["ADP_TASK_ID"] != "task-20260608-0001" || result.Env["ADP_TASK_PHASE"] != "p1" {
		t.Fatalf("render env missing task values: %#v", result.Env)
	}
}

func TestLaunchUsesDefaultCommand(t *testing.T) {
	adapter := New()
	ctx := codexContext(t.TempDir())
	ctx.Agent.Command = ""

	spec, err := adapter.Launch(context.Background(), ctx, api.RuntimeHandle{Root: "/runtime/root"}, nil)
	if err != nil {
		t.Fatalf("Launch() error = %v", err)
	}
	if spec.Command != "codex" {
		t.Fatalf("Command = %q, want codex", spec.Command)
	}
}

func TestRenderMissingFilesUsesReadableDefaults(t *testing.T) {
	adapter := New()
	ctx := codexContext(t.TempDir())
	ctx.Config.Prompts.Base = "prompts/missing.md"
	ctx.Config.Memory.Shared = "memory/missing.md"
	ctx.Config.MCP.Config = "mcp/missing.yaml"
	ctx.Agent.Profile = "missing-profile"

	result, err := adapter.Render(context.Background(), ctx)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	agents := generatedText(t, result, "AGENTS.md")
	for _, want := range []string{
		`Configured file "prompts/missing.md" is missing; using default content.`,
		`Configured file "memory/missing.md" is missing; using default content.`,
		`Configured file "mcp/missing.yaml" is missing; using default content.`,
		`No profile file was found for profile "missing-profile"`,
	} {
		if !strings.Contains(agents, want) {
			t.Fatalf("AGENTS.md missing default text %q:\n%s", want, agents)
		}
	}
}

func codexContext(workspaceDir string) api.Context {
	return api.Context{
		WorkspaceDir: workspaceDir,
		Config: schema.Config{
			Version:   schema.CurrentVersion,
			Workspace: schema.Workspace{Name: "demo"},
			Project:   schema.Project{Root: "/srv/demo"},
			Prompts:   schema.Prompts{Base: "prompts/base.md"},
			Memory:    schema.Memory{Enabled: true, Shared: "memory/shared.md"},
			Rules:     map[string]string{"coding_style": "strict"},
			MCP: schema.MCP{
				Enabled: true,
				Config:  "mcp/config.yaml",
				Servers: []string{"postgres", "github"},
			},
		},
		Agent: schema.AgentConfig{
			Enabled: true,
			Profile: "senior-engineer",
			Command: "codex-dev",
			Options: map[string]string{"review_depth": "high"},
		},
	}
}

func writeFile(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func generatedText(t *testing.T, result *api.RenderResult, path string) string {
	t.Helper()
	for _, file := range result.Files {
		if file.Path == path {
			if file.Mode != 0o644 {
				t.Fatalf("%s mode = %v, want 0644", path, file.Mode)
			}
			return string(file.Data)
		}
	}
	t.Fatalf("generated file %q was not found", path)
	return ""
}
