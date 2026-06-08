package sessions

import (
	"slices"
	"testing"

	"github.com/karoc/adp/internal/events"
)

func TestBuildRestorePlanReadyWithInvocationSnapshot(t *testing.T) {
	detail := &Detail{
		Summary: Summary{
			SessionID: "session-1",
			Workspace: "game-a",
			Agent:     "codex",
			Profile:   "senior",
			TaskID:    "task-1",
		},
		Events: []events.Event{{
			Type: eventTypeRunStarted,
			Fields: map[string]any{
				"invocation": map[string]any{
					"schema_version": 1,
					"keep_runtime":   true,
					"agent_args":     []string{"--probe", "payload value"},
				},
			},
		}},
	}

	plan := BuildRestorePlan(detail)

	if plan.Status != RestorePlanStatusReady {
		t.Fatalf("status = %q, want %q", plan.Status, RestorePlanStatusReady)
	}
	wantCommand := []string{"adp", "run", "codex", "--workspace", "game-a", "--profile", "senior", "--task", "task-1", "--keep-runtime", "--", "--probe", "payload value"}
	if !slices.Equal(plan.SuggestedCommand, wantCommand) {
		t.Fatalf("suggested command = %#v, want %#v", plan.SuggestedCommand, wantCommand)
	}
	if len(plan.MissingFields) != 0 {
		t.Fatalf("missing fields = %#v, want none", plan.MissingFields)
	}
	if len(plan.Reasons) == 0 {
		t.Fatal("expected read-only restore-plan reason")
	}
}

func TestBuildRestorePlanPartialForOldSession(t *testing.T) {
	detail := &Detail{
		Summary: Summary{
			SessionID: "session-1",
			Workspace: "game-a",
			Agent:     "codex",
		},
		Events: []events.Event{{Type: eventTypeRunStarted}},
	}

	plan := BuildRestorePlan(detail)

	if plan.Status != RestorePlanStatusPartial {
		t.Fatalf("status = %q, want %q", plan.Status, RestorePlanStatusPartial)
	}
	if !slices.Contains(plan.MissingFields, "fields.invocation") {
		t.Fatalf("missing fields = %#v, want fields.invocation", plan.MissingFields)
	}
	if !slices.Equal(plan.SuggestedCommand, []string{"adp", "run", "codex", "--workspace", "game-a"}) {
		t.Fatalf("suggested command = %#v", plan.SuggestedCommand)
	}
}

func TestBuildRestorePlanPartialForMissingAgentOrWorkspace(t *testing.T) {
	tests := []struct {
		name    string
		summary Summary
		missing []string
	}{
		{
			name:    "missing both",
			summary: Summary{SessionID: "session-1"},
			missing: []string{"agent", "workspace"},
		},
		{
			name:    "missing agent",
			summary: Summary{SessionID: "session-1", Workspace: "game-a"},
			missing: []string{"agent"},
		},
		{
			name:    "missing workspace",
			summary: Summary{SessionID: "session-1", Agent: "codex"},
			missing: []string{"workspace"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := BuildRestorePlan(&Detail{
				Summary: tt.summary,
				Events: []events.Event{{
					Type: eventTypeRunStarted,
					Fields: map[string]any{
						"invocation": map[string]any{
							"schema_version": 1,
							"keep_runtime":   false,
							"agent_args":     []any{},
						},
					},
				}},
			})

			if plan.Status != RestorePlanStatusPartial {
				t.Fatalf("status = %q, want %q", plan.Status, RestorePlanStatusPartial)
			}
			for _, want := range tt.missing {
				if !slices.Contains(plan.MissingFields, want) {
					t.Fatalf("missing fields = %#v, want %q", plan.MissingFields, want)
				}
			}
			if len(plan.SuggestedCommand) != 0 {
				t.Fatalf("suggested command = %#v, want none", plan.SuggestedCommand)
			}
		})
	}
}

func TestBuildRestorePlanPartialForUnsupportedSchema(t *testing.T) {
	plan := BuildRestorePlan(&Detail{
		Summary: Summary{
			SessionID: "session-1",
			Workspace: "game-a",
			Agent:     "codex",
		},
		Events: []events.Event{{
			Type: eventTypeRunStarted,
			Fields: map[string]any{
				"invocation": map[string]any{
					"schema_version": float64(99),
					"keep_runtime":   false,
					"agent_args":     []any{"--version"},
				},
			},
		}},
	})

	if plan.Status != RestorePlanStatusPartial {
		t.Fatalf("status = %q, want %q", plan.Status, RestorePlanStatusPartial)
	}
	if !slices.Contains(plan.MissingFields, "fields.invocation.schema_version") {
		t.Fatalf("missing fields = %#v, want schema version", plan.MissingFields)
	}
	if !slices.Equal(plan.SuggestedCommand, []string{"adp", "run", "codex", "--workspace", "game-a", "--", "--version"}) {
		t.Fatalf("suggested command = %#v", plan.SuggestedCommand)
	}
}
