package shared

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

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
		"EXISTING":                  "1",
		"ADP_AGENT":                 "future-agent",
		"ADP_WORKSPACE":             "demo",
		"ADP_PROJECT_ROOT":          "/srv/demo",
		"ADP_PROFILE":               "builder",
		"ADP_RUNTIME_ROOT":          "/tmp/adp-runtime/session-1",
		"ADP_SESSION_ID":            "session-1",
		"ADP_TASK_ID":               "task-20260608-0099",
		"ADP_TASK_TITLE":            "Validate adapter boundary",
		"ADP_TASK_OWNER":            "codex-main",
		"ADP_TASK_LEASE_EXPIRES_AT": "2026-06-09T12:00:00Z",
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

func TestInstructionsIncludePlanningContractAndTaskboxBridge(t *testing.T) {
	ctx := sharedTestContext()

	instructions := string(Instructions("future-agent", ctx))
	for _, want := range []string{
		"## ADP Planning Contract",
		"ADP is the authoritative local planning and progress ledger",
		"Provider-native todo lists or task panels are scratch space only",
		"`ADP_CLI` environment variable",
		"$ADP_CLI tasks next --workspace \"demo\" --format json",
		"$ADP_CLI tasks take --workspace \"demo\" --owner <owner> --lease 4h --format json",
		"$ADP_CLI tasks stale --workspace \"demo\" --format json",
		"$ADP_CLI tasks claim --workspace \"demo\" <task-id> --owner <owner> --lease 4h",
		"$ADP_CLI tasks update --workspace \"$ADP_WORKSPACE\" \"$ADP_TASK_ID\" --status <status>",
		"$ADP_CLI tasks renew --workspace \"$ADP_WORKSPACE\" \"$ADP_TASK_ID\" --owner \"$ADP_TASK_OWNER\" --lease 4h",
		"## ADP Lease Handoff",
		"Keep this ADP task claim alive during long-running work",
		"Current lease expires at: 2026-06-09T12:00:00Z",
		"Inspect interrupted claims before taking over work",
		"## Git Boundary",
		"not the authoritative Git working tree",
		"Repository Git metadata is intentionally not exposed",
		"git -C \"$ADP_PROJECT_ROOT\" status --short --branch",
		"git -C \"$ADP_PROJECT_ROOT\" diff --cached",
		"Real project root: /srv/demo",
		"## Tool Taskbox Bridge",
		"mirror the active ADP task into this tool's native task or todo panel",
		"owner, lease expiration",
		"do not treat provider-native task state as authoritative",
	} {
		if !strings.Contains(instructions, want) {
			t.Fatalf("Instructions missing %q:\n%s", want, instructions)
		}
	}
}

func TestInstructionsIncludePlanModeBridge(t *testing.T) {
	ctx := sharedTestContext()

	instructions := string(Instructions("future-agent", ctx))
	for _, want := range []string{
		"## Tool Plan Mode Bridge",
		"native plan mode",
		"proposal view",
		"ADP remains the durable planning ledger",
		"do not edit project files, complete tasks, accept phases, commit, or push",
		"$ADP_CLI plan preview --workspace \"demo\" --file - --format json",
		"$ADP_CLI plan apply --workspace \"demo\" --file - --format json",
		"$ADP_CLI phase status --workspace \"demo\" --format json",
		"Provider-native plan approval is not ADP phase acceptance",
	} {
		if !strings.Contains(instructions, want) {
			t.Fatalf("Instructions missing %q:\n%s", want, instructions)
		}
	}
}

func TestMetadataIncludesTaskLeaseFields(t *testing.T) {
	ctx := sharedTestContext()

	toml := string(MetadataTOML("future-agent", ctx))
	for _, want := range []string{
		`task_owner = "codex-main"`,
		`task_claimed_at = "2026-06-09T08:00:00Z"`,
		`task_lease_expires_at = "2026-06-09T12:00:00Z"`,
	} {
		if !strings.Contains(toml, want) {
			t.Fatalf("MetadataTOML missing %q:\n%s", want, toml)
		}
	}

	data, err := MetadataJSON("future-agent", ctx)
	if err != nil {
		t.Fatalf("MetadataJSON() error = %v", err)
	}
	var metadata struct {
		ADP struct {
			Task map[string]string `json:"task"`
		} `json:"adp"`
	}
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatalf("MetadataJSON output is invalid JSON: %v\n%s", err, data)
	}
	for key, want := range map[string]string{
		"owner":          "codex-main",
		"claimedAt":      "2026-06-09T08:00:00Z",
		"leaseExpiresAt": "2026-06-09T12:00:00Z",
	} {
		if metadata.ADP.Task[key] != want {
			t.Fatalf("metadata task[%s] = %q, want %q; metadata=%s", key, metadata.ADP.Task[key], want, data)
		}
	}
}

func TestInstructionsWithoutTaskDirectAgentToClaimADPWork(t *testing.T) {
	ctx := sharedTestContext()
	ctx.Task = api.TaskContext{}

	instructions := string(Instructions("future-agent", ctx))
	for _, want := range []string{
		"## ADP Planning Contract",
		"$ADP_CLI tasks next --workspace \"demo\" --format json",
		"$ADP_CLI tasks update --workspace \"demo\" <task-id> --status <status>",
		"$ADP_CLI tasks renew --workspace \"demo\" <task-id> --owner <owner> --lease 4h",
		"## ADP Lease Handoff",
		"Inspect interrupted work",
		"Reclaim only through ADP ownership commands",
		"No ADP task is bound to this runtime session.",
		"claim the selected task through ADP",
		"$ADP_CLI plan preview --workspace \"demo\" --file - --format json",
	} {
		if !strings.Contains(instructions, want) {
			t.Fatalf("Instructions missing %q:\n%s", want, instructions)
		}
	}
	if strings.Contains(instructions, "\"$ADP_TASK_ID\"") {
		t.Fatalf("unbound task instructions should not reference ADP_TASK_ID as active task:\n%s", instructions)
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
			ID:             "task-20260608-0099",
			Title:          "Validate adapter boundary",
			Status:         "in_progress",
			Priority:       "high",
			Phase:          "p28",
			Owner:          "codex-main",
			ClaimedAt:      time.Date(2026, 6, 9, 8, 0, 0, 0, time.UTC),
			LeaseExpiresAt: time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC),
		},
	}
}
