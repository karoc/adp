package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	taskstore "github.com/karoc/adp/internal/tasks"
)

func TestPlanPreviewCommandReadsInputWithoutApplying(t *testing.T) {
	store := &fakeTaskStore{
		planPreviewResult: taskstore.PlanImportResult{
			Phases: []taskstore.Phase{testPhase("p14", "Plan intake", taskstore.PhaseStatusPlanned)},
			Tasks:  []taskstore.Task{testTask("task-1", "Define schema", taskstore.StatusReady)},
		},
	}
	input := writePlanInput(t, `
version: 1
phases:
  - id: p14
    title: Plan intake
tasks:
  - title: Define schema
    phase: p14
    priority: high
`)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	deps := Dependencies{
		WorkspaceStore:   &fakeStore{cfg: testConfig()},
		TaskStoreFactory: func(string) TaskStore { return store },
	}

	code := NewApp(deps, &stdout, &stderr).Execute(context.Background(), []string{"plan", "preview", "--workspace", "game-a", "--file", input})

	if code != 0 {
		t.Fatalf("plan preview exit code = %d, stderr = %q", code, stderr.String())
	}
	if store.previewCalls != 1 || store.applyCalls != 0 {
		t.Fatalf("calls preview=%d apply=%d", store.previewCalls, store.applyCalls)
	}
	if len(store.planReq.Phases) != 1 || store.planReq.Phases[0].ID != "p14" {
		t.Fatalf("plan phases = %+v", store.planReq.Phases)
	}
	if len(store.planReq.Tasks) != 1 || store.planReq.Tasks[0].Priority != "high" || store.planReq.Tasks[0].Status != "" {
		t.Fatalf("plan tasks = %+v", store.planReq.Tasks)
	}
	for _, want := range []string{"workspace: game-a", "mode: preview", "p14", "task-1"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("preview output missing %q: %q", want, stdout.String())
		}
	}
}

func TestPlanPreviewCommandReadsStdinWithoutApplying(t *testing.T) {
	store := &fakeTaskStore{
		planPreviewResult: taskstore.PlanImportResult{
			Phases: []taskstore.Phase{testPhase("p20", "Plan stdin", taskstore.PhaseStatusPlanned)},
			Tasks:  []taskstore.Task{testTask("task-stdin", "Preview stdin plan", taskstore.StatusReady)},
		},
	}
	withPlanStdin(t, `
version: 1
phases:
  - id: p20
    title: Plan stdin
tasks:
  - title: Preview stdin plan
    phase: p20
    priority: normal
`)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	deps := Dependencies{
		WorkspaceStore:   &fakeStore{cfg: testConfig()},
		TaskStoreFactory: func(string) TaskStore { return store },
	}

	code := NewApp(deps, &stdout, &stderr).Execute(context.Background(), []string{"plan", "preview", "--workspace", "game-a", "--file", "-"})

	if code != 0 {
		t.Fatalf("plan preview stdin exit code = %d, stderr = %q", code, stderr.String())
	}
	if store.previewCalls != 1 || store.applyCalls != 0 {
		t.Fatalf("calls preview=%d apply=%d", store.previewCalls, store.applyCalls)
	}
	if len(store.planReq.Phases) != 1 || store.planReq.Phases[0].ID != "p20" {
		t.Fatalf("plan phases = %+v", store.planReq.Phases)
	}
	if len(store.planReq.Tasks) != 1 || store.planReq.Tasks[0].Title != "Preview stdin plan" {
		t.Fatalf("plan tasks = %+v", store.planReq.Tasks)
	}
	for _, want := range []string{"source: -", "mode: preview", "p20", "task-stdin"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("preview stdin output missing %q: %q", want, stdout.String())
		}
	}
}

func TestPlanApplyCommandPrintsJSON(t *testing.T) {
	store := &fakeTaskStore{
		planApplyResult: taskstore.PlanImportResult{
			Phases: []taskstore.Phase{testPhase("p14", "Plan intake", taskstore.PhaseStatusPlanned)},
			Tasks:  []taskstore.Task{testTask("task-1", "Apply schema", taskstore.StatusReview)},
		},
	}
	input := writePlanInput(t, `{
  "version": 1,
  "phases": [{"id": "p14", "title": "Plan intake"}],
  "tasks": [{"title": "Apply schema", "phase": "p14", "status": "review"}]
}`)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	deps := Dependencies{
		WorkspaceStore:   &fakeStore{cfg: testConfig()},
		TaskStoreFactory: func(string) TaskStore { return store },
	}

	code := NewApp(deps, &stdout, &stderr).Execute(context.Background(), []string{"plan", "apply", "--workspace", "game-a", "--file", input, "--format", "json"})

	if code != 0 {
		t.Fatalf("plan apply exit code = %d, stderr = %q", code, stderr.String())
	}
	if store.previewCalls != 0 || store.applyCalls != 1 {
		t.Fatalf("calls preview=%d apply=%d", store.previewCalls, store.applyCalls)
	}
	if len(store.planReq.Tasks) != 1 || store.planReq.Tasks[0].Status != taskstore.StatusReview {
		t.Fatalf("plan request tasks = %+v", store.planReq.Tasks)
	}
	payload := decodeJSONObject(t, stdout.Bytes())
	assertJSONStringField(t, payload, "workspace", "game-a")
	assertJSONStringField(t, payload, "mode", "apply")

	phase := findJSONObject(t, assertJSONObjectListField(t, payload, "phases"), "id", "p14")
	assertJSONStringField(t, phase, "status", "planned")
	task := findJSONObject(t, assertJSONObjectListField(t, payload, "tasks"), "id", "task-1")
	assertJSONStringField(t, task, "status", "review")
}

func TestPlanApplyCommandReadsStdinAndPrintsJSONSource(t *testing.T) {
	store := &fakeTaskStore{
		planApplyResult: taskstore.PlanImportResult{
			Phases: []taskstore.Phase{testPhase("p20", "Plan stdin", taskstore.PhaseStatusPlanned)},
			Tasks:  []taskstore.Task{testTask("task-stdin", "Apply stdin plan", taskstore.StatusReview)},
		},
	}
	withPlanStdin(t, `{
  "version": 1,
  "phases": [{"id": "p20", "title": "Plan stdin"}],
  "tasks": [{"title": "Apply stdin plan", "phase": "p20", "status": "review"}]
}`)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	deps := Dependencies{
		WorkspaceStore:   &fakeStore{cfg: testConfig()},
		TaskStoreFactory: func(string) TaskStore { return store },
	}

	code := NewApp(deps, &stdout, &stderr).Execute(context.Background(), []string{"plan", "apply", "--workspace", "game-a", "--file", "-", "--format", "json"})

	if code != 0 {
		t.Fatalf("plan apply stdin exit code = %d, stderr = %q", code, stderr.String())
	}
	if store.previewCalls != 0 || store.applyCalls != 1 {
		t.Fatalf("calls preview=%d apply=%d", store.previewCalls, store.applyCalls)
	}
	if len(store.planReq.Tasks) != 1 || store.planReq.Tasks[0].Status != taskstore.StatusReview {
		t.Fatalf("plan request tasks = %+v", store.planReq.Tasks)
	}
	payload := decodeJSONObject(t, stdout.Bytes())
	assertJSONStringField(t, payload, "workspace", "game-a")
	assertJSONStringField(t, payload, "mode", "apply")
	assertJSONStringField(t, payload, "source", "-")
}

func TestPlanCommandRejectsInvalidArgsAndInput(t *testing.T) {
	deps := Dependencies{
		WorkspaceStore:   &fakeStore{cfg: testConfig()},
		TaskStoreFactory: func(string) TaskStore { return &fakeTaskStore{} },
	}
	cases := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "missing subcommand",
			args: []string{"plan"},
			want: "usage: adp plan",
		},
		{
			name: "unknown subcommand",
			args: []string{"plan", "bogus"},
			want: "unknown plan command",
		},
		{
			name: "missing file",
			args: []string{"plan", "preview", "--workspace", "game-a"},
			want: "usage: adp plan preview",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			var stderr bytes.Buffer
			code := NewApp(deps, &bytes.Buffer{}, &stderr).Execute(context.Background(), tt.args)
			if code == 0 {
				t.Fatal("plan command returned success")
			}
			if !strings.Contains(stderr.String(), tt.want) {
				t.Fatalf("stderr = %q, want substring %q", stderr.String(), tt.want)
			}
		})
	}

	input := writePlanInput(t, "version: 1\ntasks:\n  - priority: high\n")
	var stderr bytes.Buffer
	code := NewApp(deps, &bytes.Buffer{}, &stderr).Execute(context.Background(), []string{"plan", "preview", "--workspace", "game-a", "--file", input})
	if code == 0 {
		t.Fatal("invalid input preview returned success")
	}
	if !strings.Contains(stderr.String(), "task[0].title is required") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestPlanImportOutputJSONShape(t *testing.T) {
	payload := planImportOutput("game-a", "preview", "plan.yaml", taskstore.PlanImportResult{
		Phases: []taskstore.Phase{testPhase("p14", "Plan intake", taskstore.PhaseStatusPlanned)},
		Tasks:  []taskstore.Task{testTask("task-1", "Define schema", taskstore.StatusReady)},
	})
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal output: %v", err)
	}
	if !strings.Contains(string(data), `"mode":"preview"`) || !strings.Contains(string(data), `"source":"plan.yaml"`) {
		t.Fatalf("payload json = %s", data)
	}
}

func writePlanInput(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "plan.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write plan input: %v", err)
	}
	return path
}

func withPlanStdin(t *testing.T, content string) {
	t.Helper()
	oldStdin := os.Stdin
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdin pipe: %v", err)
	}
	if _, err := writer.WriteString(content); err != nil {
		t.Fatalf("write stdin pipe: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close stdin writer: %v", err)
	}
	os.Stdin = reader
	t.Cleanup(func() {
		os.Stdin = oldStdin
		_ = reader.Close()
	})
}
