package shared

import (
	"reflect"
	"testing"

	"github.com/karoc/adp/internal/adapters/api"
	"github.com/karoc/adp/internal/schema"
)

func TestLaunchBuildsProviderNeutralSpec(t *testing.T) {
	ctx := sharedTestContext()
	extraArgs := []string{"--dry-run"}

	spec := Launch("future-agent", ctx, api.RuntimeHandle{
		SessionID: "session-1",
		TaskID:    "task-20260608-0099",
		Root:      "/tmp/adp-runtime/session-1",
		Env: map[string]string{
			"EXISTING":  "1",
			"ADP_AGENT": "runtime-value",
		},
	}, "future-cli", extraArgs)

	extraArgs[0] = "changed"
	if spec.Command != "future-override" {
		t.Fatalf("Command = %q, want future-override", spec.Command)
	}
	if !reflect.DeepEqual(spec.Args, []string{"--dry-run"}) {
		t.Fatalf("Args = %#v", spec.Args)
	}
	if spec.Dir != "/tmp/adp-runtime/session-1" {
		t.Fatalf("Dir = %q", spec.Dir)
	}
	for key, want := range map[string]string{
		"EXISTING":         "1",
		"ADP_AGENT":        "future-agent",
		"ADP_WORKSPACE":    "demo",
		"ADP_PROJECT_ROOT": "/srv/demo",
		"ADP_PROFILE":      "builder",
		"ADP_RUNTIME_ROOT": "/tmp/adp-runtime/session-1",
		"ADP_SESSION_ID":   "session-1",
		"ADP_TASK_ID":      "task-20260608-0099",
		"ADP_TASK_TITLE":   "Validate adapter boundary",
	} {
		if spec.Env[key] != want {
			t.Fatalf("Env[%s] = %q, want %q; env=%#v", key, spec.Env[key], want, spec.Env)
		}
	}
}

func TestLaunchUsesProviderNeutralDefaultCommand(t *testing.T) {
	ctx := sharedTestContext()
	ctx.Agent.Command = ""

	spec := Launch("future-agent", ctx, api.RuntimeHandle{Root: "/tmp/runtime"}, "future-cli", nil)

	if spec.Command != "future-cli" {
		t.Fatalf("Command = %q, want future-cli", spec.Command)
	}
}

func sharedTestContext() api.Context {
	return api.Context{
		WorkspaceDir: "/tmp/adp-home/workspaces/demo",
		Config: schema.Config{
			Version:   schema.CurrentVersion,
			Workspace: schema.Workspace{Name: "demo"},
			Project:   schema.Project{Root: "/srv/demo"},
		},
		Agent: schema.AgentConfig{
			Enabled: true,
			Profile: "builder",
			Command: "future-override",
		},
		Task: api.TaskContext{
			ID:       "task-20260608-0099",
			Title:    "Validate adapter boundary",
			Status:   "in_progress",
			Priority: "high",
			Phase:    "p28",
		},
	}
}
