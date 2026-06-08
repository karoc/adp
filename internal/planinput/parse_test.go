package planinput

import (
	"strings"
	"testing"
)

func TestParseValidYAML(t *testing.T) {
	doc, err := Parse([]byte(`
version: 1
phases:
  - id: " p14-parser "
    title: " Plan input parser "
    goal: " Parse deterministic local planning input "
tasks:
  - title: " Define schema "
    description: " Validate phase and task intake fields "
    priority: " high "
    phase: " existing-ledger-phase "
    status: " ready "
`))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if doc.Version != Version {
		t.Fatalf("version = %d, want %d", doc.Version, Version)
	}
	if len(doc.Phases) != 1 {
		t.Fatalf("phase count = %d, want 1", len(doc.Phases))
	}
	if got := doc.Phases[0].ID; got != "p14-parser" {
		t.Fatalf("phase id = %q, want trimmed value", got)
	}
	if got := doc.Phases[0].Title; got != "Plan input parser" {
		t.Fatalf("phase title = %q, want trimmed value", got)
	}
	if len(doc.Tasks) != 1 {
		t.Fatalf("task count = %d, want 1", len(doc.Tasks))
	}
	task := doc.Tasks[0]
	if task.Title != "Define schema" {
		t.Fatalf("task title = %q, want trimmed value", task.Title)
	}
	if task.Priority != "high" {
		t.Fatalf("task priority = %q, want trimmed value", task.Priority)
	}
	if task.Status != "ready" {
		t.Fatalf("task status = %q, want trimmed value", task.Status)
	}
	if task.Phase != "existing-ledger-phase" {
		t.Fatalf("task phase = %q, want unknown ledger phase accepted", task.Phase)
	}
}

func TestParseValidJSON(t *testing.T) {
	doc, err := Parse([]byte(`{
		"version": 1,
		"phases": [{"id": "p14-parser", "title": "Parser", "goal": "Parse input"}],
		"tasks": [{"title": "Write tests", "priority": "p0", "phase": "p14-parser", "status": "planned"}]
	}`))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if doc.Phases[0].ID != "p14-parser" {
		t.Fatalf("phase id = %q, want p14-parser", doc.Phases[0].ID)
	}
	if doc.Tasks[0].Priority != "p0" {
		t.Fatalf("task priority = %q, want p0", doc.Tasks[0].Priority)
	}
	if doc.Tasks[0].Status != "planned" {
		t.Fatalf("task status = %q, want planned", doc.Tasks[0].Status)
	}
}

func TestParseRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{
			name: "empty input",
			data: " \n\t ",
			want: "empty",
		},
		{
			name: "unsupported version",
			data: `
version: 2
phases:
  - id: p14
    title: Parser
`,
			want: "unsupported",
		},
		{
			name: "duplicate phase id",
			data: `
version: 1
phases:
  - id: p14
    title: Parser
  - id: " p14 "
    title: Duplicate
`,
			want: "duplicate phase id",
		},
		{
			name: "missing task title",
			data: `
version: 1
tasks:
  - priority: high
`,
			want: "task[0].title is required",
		},
		{
			name: "invalid status",
			data: `
version: 1
tasks:
  - title: Parse plan
    status: active
`,
			want: "status",
		},
		{
			name: "invalid priority",
			data: `
version: 1
tasks:
  - title: Parse plan
    priority: immediate
`,
			want: "priority",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.data))
			if err == nil {
				t.Fatal("Parse returned nil error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want substring %q", err.Error(), tt.want)
			}
		})
	}
}
