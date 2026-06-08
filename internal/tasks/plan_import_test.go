package tasks

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStorePreviewPlanImportDoesNotWrite(t *testing.T) {
	store := testStore(t)

	result, err := store.PreviewPlanImport(context.Background(), PlanImportRequest{
		Phases: []PlanImportPhase{{ID: "p14", Title: "Plan intake", Goal: "structured planning input"}},
		Tasks: []PlanImportTask{{
			Title:       "Define schema",
			Description: "versioned input",
			Priority:    "high",
			Phase:       "p14",
		}},
	})
	if err != nil {
		t.Fatalf("PreviewPlanImport returned error: %v", err)
	}
	if len(result.Phases) != 1 || result.Phases[0].ID != "p14" || result.Phases[0].Status != PhaseStatusPlanned {
		t.Fatalf("preview phases = %+v", result.Phases)
	}
	if len(result.Tasks) != 1 || result.Tasks[0].ID != "task-20260608-0001" || result.Tasks[0].Status != StatusReady {
		t.Fatalf("preview tasks = %+v", result.Tasks)
	}
	if _, err := os.Stat(store.planningPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("planning path after preview error = %v, want not exist", err)
	}
}

func TestStoreApplyPlanImportAddsPhasesAndTasks(t *testing.T) {
	store := testStore(t)

	result, err := store.ApplyPlanImport(context.Background(), PlanImportRequest{
		Phases: []PlanImportPhase{{ID: "p14", Title: "Plan intake", Goal: "structured planning input"}},
		Tasks: []PlanImportTask{
			{Title: "Define schema", Priority: "high", Phase: "p14"},
			{Title: "Review import", Status: StatusReview, Phase: "p14"},
		},
	})
	if err != nil {
		t.Fatalf("ApplyPlanImport returned error: %v", err)
	}
	if len(result.Phases) != 1 || len(result.Tasks) != 2 {
		t.Fatalf("import result = %+v", result)
	}
	if result.Tasks[0].ID != "task-20260608-0001" || result.Tasks[1].ID != "task-20260608-0002" {
		t.Fatalf("task ids = %s, %s", result.Tasks[0].ID, result.Tasks[1].ID)
	}

	phases, err := store.ListPhases(context.Background())
	if err != nil {
		t.Fatalf("ListPhases returned error: %v", err)
	}
	if len(phases) != 1 || phases[0].ID != "p14" {
		t.Fatalf("phases = %+v", phases)
	}
	tasks, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(tasks) != 2 || tasks[0].Title != "Define schema" || tasks[1].Status != StatusReview {
		t.Fatalf("tasks = %+v", tasks)
	}

	progressPath := filepath.Join(store.WorkspaceDir, "planning", "progress.jsonl")
	assertFileContains(t, progressPath, `"type":"phase_created"`)
	assertFileContains(t, progressPath, `"type":"task_created"`)
}

func TestStorePlanImportRejectsInvalidInputWithoutPartialWrite(t *testing.T) {
	store := testStore(t)
	if _, err := store.AddPhase(context.Background(), PhaseAddRequest{ID: "p-existing", Title: "Existing"}); err != nil {
		t.Fatalf("AddPhase returned error: %v", err)
	}

	_, err := store.ApplyPlanImport(context.Background(), PlanImportRequest{
		Phases: []PlanImportPhase{{ID: "p-new", Title: "New phase"}},
		Tasks:  []PlanImportTask{{Title: "Missing phase task", Phase: "p-missing"}},
	})
	if !errors.Is(err, ErrPhaseNotFound) {
		t.Fatalf("ApplyPlanImport error = %v, want ErrPhaseNotFound", err)
	}
	if _, err := store.GetPhase(context.Background(), "p-new"); !errors.Is(err, ErrPhaseNotFound) {
		t.Fatalf("GetPhase p-new error = %v, want ErrPhaseNotFound", err)
	}
	tasks, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(tasks) != 0 {
		t.Fatalf("tasks after failed import = %+v", tasks)
	}
	progressPath := filepath.Join(store.WorkspaceDir, "planning", "progress.jsonl")
	data, err := os.ReadFile(progressPath)
	if err != nil {
		t.Fatalf("read progress log: %v", err)
	}
	if strings.Contains(string(data), "p-new") || strings.Contains(string(data), "Missing phase task") {
		t.Fatalf("progress log contains failed import data:\n%s", data)
	}
}

func TestStorePlanImportRejectsFreshInvalidInputWithoutPlanningDir(t *testing.T) {
	store := testStore(t)

	_, err := store.ApplyPlanImport(context.Background(), PlanImportRequest{
		Tasks: []PlanImportTask{{Title: "Missing phase task", Phase: "p-missing"}},
	})
	if !errors.Is(err, ErrPhaseNotFound) {
		t.Fatalf("ApplyPlanImport error = %v, want ErrPhaseNotFound", err)
	}
	if _, err := os.Stat(store.planningPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("planning path after failed import error = %v, want not exist", err)
	}
}

func TestStorePlanImportStageFailureLeavesExistingState(t *testing.T) {
	store := testStore(t)
	if _, err := store.AddPhase(context.Background(), PhaseAddRequest{ID: "p-existing", Title: "Existing"}); err != nil {
		t.Fatalf("AddPhase returned error: %v", err)
	}
	if _, err := store.Add(context.Background(), AddRequest{Title: "Existing task", Phase: "p-existing"}); err != nil {
		t.Fatalf("Add returned error: %v", err)
	}
	phasePath := filepath.Join(store.WorkspaceDir, "planning", "phases.yaml")
	taskPath := filepath.Join(store.WorkspaceDir, "planning", "tasks.yaml")
	progressPath := filepath.Join(store.WorkspaceDir, "planning", "progress.jsonl")
	phasesBefore := readPlanImportFile(t, phasePath)
	tasksBefore := readPlanImportFile(t, taskPath)
	progressBefore := readPlanImportFile(t, progressPath)

	if err := os.Mkdir(filepath.Join(store.WorkspaceDir, "planning", "tasks.yaml.tmp"), 0o755); err != nil {
		t.Fatalf("create task temp directory: %v", err)
	}
	_, err := store.ApplyPlanImport(context.Background(), PlanImportRequest{
		Phases: []PlanImportPhase{{ID: "p-new", Title: "New phase"}},
		Tasks:  []PlanImportTask{{Title: "New task", Phase: "p-new"}},
	})
	if err == nil || !strings.Contains(err.Error(), "write temporary planning file") {
		t.Fatalf("ApplyPlanImport error = %v, want temporary planning write error", err)
	}
	if got := readPlanImportFile(t, phasePath); got != phasesBefore {
		t.Fatalf("phases changed after failed import:\n%s", got)
	}
	if got := readPlanImportFile(t, taskPath); got != tasksBefore {
		t.Fatalf("tasks changed after failed import:\n%s", got)
	}
	if got := readPlanImportFile(t, progressPath); got != progressBefore {
		t.Fatalf("progress changed after failed import:\n%s", got)
	}
	if _, err := store.GetPhase(context.Background(), "p-new"); !errors.Is(err, ErrPhaseNotFound) {
		t.Fatalf("GetPhase p-new error = %v, want ErrPhaseNotFound", err)
	}
}

func readPlanImportFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
