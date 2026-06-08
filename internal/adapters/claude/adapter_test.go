package claude

import (
	"context"
	"encoding/json"
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
	writeFile(t, workspaceDir, "prompts/base.md", "Claude base prompt.\n")
	writeFile(t, workspaceDir, "memory/shared.md", "Claude shared memory.\n")
	writeFile(t, workspaceDir, "mcp/config.yaml", "servers:\n  github: {}\n")
	writeFile(t, workspaceDir, "profiles/architect.yaml", "mode: planning\n")

	adapter := New()
	result, err := adapter.Render(context.Background(), claudeContext(workspaceDir))
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	claude := generatedText(t, result, "CLAUDE.md")
	for _, want := range []string{
		"Claude base prompt.",
		"Claude shared memory.",
		"- coding_style: pragmatic",
		"- github",
		"- sqlite",
		"mode: planning",
	} {
		if !strings.Contains(claude, want) {
			t.Fatalf("CLAUDE.md missing %q:\n%s", want, claude)
		}
	}

	var settings map[string]map[string]any
	if err := json.Unmarshal([]byte(generatedText(t, result, ".claude/settings.json")), &settings); err != nil {
		t.Fatalf("settings.json is not valid JSON: %v", err)
	}
	if settings["adp"]["adapter"] != "claude" || settings["adp"]["workspace"] != "demo" || settings["adp"]["profile"] != "architect" {
		t.Fatalf("settings adp metadata = %v", settings["adp"])
	}
}

func TestLaunchUsesCommandOverrideAndExtraArgs(t *testing.T) {
	adapter := New()
	ctx := claudeContext(t.TempDir())
	extraArgs := []string{"--continue"}

	spec, err := adapter.Launch(context.Background(), ctx, api.RuntimeHandle{
		SessionID: "session-2",
		Root:      "/runtime/claude",
		Env:       map[string]string{"EXISTING": "1"},
	}, extraArgs)
	if err != nil {
		t.Fatalf("Launch() error = %v", err)
	}

	extraArgs[0] = "changed"
	if spec.Command != "claude-dev" {
		t.Fatalf("Command = %q, want claude-dev", spec.Command)
	}
	if spec.Dir != "/runtime/claude" {
		t.Fatalf("Dir = %q, want /runtime/claude", spec.Dir)
	}
	if !reflect.DeepEqual(spec.Args, []string{"--continue"}) {
		t.Fatalf("Args = %v", spec.Args)
	}
	if spec.Env["EXISTING"] != "1" || spec.Env["ADP_AGENT"] != "claude" || spec.Env["ADP_SESSION_ID"] != "session-2" {
		t.Fatalf("Env missing expected values: %v", spec.Env)
	}
}

func TestLaunchUsesDefaultCommand(t *testing.T) {
	adapter := New()
	ctx := claudeContext(t.TempDir())
	ctx.Agent.Command = ""

	spec, err := adapter.Launch(context.Background(), ctx, api.RuntimeHandle{Root: "/runtime/claude"}, nil)
	if err != nil {
		t.Fatalf("Launch() error = %v", err)
	}
	if spec.Command != "claude" {
		t.Fatalf("Command = %q, want claude", spec.Command)
	}
}

func claudeContext(workspaceDir string) api.Context {
	return api.Context{
		WorkspaceDir: workspaceDir,
		Config: schema.Config{
			Version:   schema.CurrentVersion,
			Workspace: schema.Workspace{Name: "demo"},
			Project:   schema.Project{Root: "/srv/demo"},
			Prompts:   schema.Prompts{Base: "prompts/base.md"},
			Memory:    schema.Memory{Enabled: true, Shared: "memory/shared.md"},
			Rules:     map[string]string{"coding_style": "pragmatic"},
			MCP: schema.MCP{
				Enabled: true,
				Config:  "mcp/config.yaml",
				Servers: []string{"sqlite", "github"},
			},
		},
		Agent: schema.AgentConfig{
			Enabled: true,
			Profile: "architect",
			Command: "claude-dev",
			Options: map[string]string{"mode": "plan"},
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
