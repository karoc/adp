package tasks

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDiagnosePlanningMissingLedgerIsOKAndReadOnly(t *testing.T) {
	workspaceDir := filepath.Join(t.TempDir(), "workspace")
	store := NewStore(workspaceDir)

	report, err := store.DiagnosePlanning(context.Background())
	if err != nil {
		t.Fatalf("DiagnosePlanning returned error: %v", err)
	}
	if report.HasErrors() || len(report.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %+v, want none", report.Diagnostics)
	}
	if report.TaskCount != 0 || report.PhaseCount != 0 || report.ProgressEventCount != 0 {
		t.Fatalf("counts = tasks:%d phases:%d events:%d, want zero", report.TaskCount, report.PhaseCount, report.ProgressEventCount)
	}
	if _, err := os.Stat(filepath.Join(workspaceDir, planningDir)); !os.IsNotExist(err) {
		t.Fatalf("planning dir stat error = %v, want not exist", err)
	}
}

func TestDiagnosePlanningAllowsLegacyTaskPhaseWithoutPhaseLedger(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "workspace"))
	if err := os.MkdirAll(store.planningPath(), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, store.tasksPath(), []byte(`
version: 1
tasks:
  - id: task-1
    title: Legacy task
    status: ready
    priority: normal
    phase: legacy-free-form
    created_at: 2026-06-08T12:00:00Z
    updated_at: 2026-06-08T12:00:00Z
`))

	report, err := store.DiagnosePlanning(context.Background())
	if err != nil {
		t.Fatalf("DiagnosePlanning returned error: %v", err)
	}
	if hasPlanningDiagnostic(report, PlanningDiagnosticCodeTaskPhaseUnknown) {
		t.Fatalf("legacy free-form task phase was reported unknown: %+v", report.Diagnostics)
	}
	if report.HasErrors() {
		t.Fatalf("diagnostics = %+v, want no errors", report.Diagnostics)
	}
}

func TestDiagnosePlanningReportsLedgerInvariantsWithoutMutation(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "workspace"))
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)
	store.Now = func() time.Time { return now }
	if err := os.MkdirAll(store.planningPath(), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, store.tasksPath(), []byte(`
version: 1
tasks:
  - id: task-1
    title: Unknown phase task
    status: ready
    priority: critical
    phase: missing-phase
    created_at: 2026-06-08T12:00:00Z
    updated_at: 2026-06-08T12:00:00Z
  - id: task-1
    title: Duplicate task
    status: planned
    priority: high
    phase: p2
    created_at: 2026-06-08T12:00:00Z
    updated_at: 2026-06-08T12:00:00Z
`))
	writeFile(t, store.phasesPath(), []byte(`
version: 1
phases:
  - id: p1
    title: Earlier planned
    status: planned
    order: 1
    created_at: 2026-06-08T12:00:00Z
    updated_at: 2026-06-08T12:00:00Z
  - id: p2
    title: Later active
    status: active
    order: 2
    created_at: 2026-06-08T12:00:00Z
    updated_at: 2026-06-08T12:00:00Z
  - id: p3
    title: Duplicate order
    status: accepted
    order: 2
    acceptance:
      result: failed
    created_at: 2026-06-08T12:00:00Z
    updated_at: 2026-06-08T12:00:00Z
  - id: p4
    title: Missing commit hash
    status: committed
    order: 3
    acceptance:
      result: passed
    created_at: 2026-06-08T12:00:00Z
    updated_at: 2026-06-08T12:00:00Z
`))
	writeFile(t, store.progressPath(), []byte("{bad json\n"))
	writeFile(t, store.lockPath(), []byte("stale\n"))
	staleTime := now.Add(-planningLockStaleAge - time.Minute)
	if err := os.Chtimes(store.lockPath(), staleTime, staleTime); err != nil {
		t.Fatalf("make lock stale: %v", err)
	}
	before := planningContentSnapshot(t, store)

	report, err := store.DiagnosePlanning(context.Background())
	if err != nil {
		t.Fatalf("DiagnosePlanning returned error: %v", err)
	}

	for _, code := range []string{
		PlanningDiagnosticCodeTaskIDDuplicate,
		PlanningDiagnosticCodeTaskPhaseUnknown,
		PlanningDiagnosticCodePhaseOrderDuplicate,
		PlanningDiagnosticCodePhaseMultipleOpen,
		PlanningDiagnosticCodePhaseGateSkipped,
		PlanningDiagnosticCodePhaseEvidenceMissing,
		PlanningDiagnosticCodeProgressInvalidJSON,
		PlanningDiagnosticCodeLockStale,
	} {
		if !hasPlanningDiagnostic(report, code) {
			t.Fatalf("missing diagnostic %s in %+v", code, report.Diagnostics)
		}
	}
	if !report.HasErrors() {
		t.Fatalf("report.HasErrors() = false, want true: %+v", report.Diagnostics)
	}
	if report.TaskCount != 2 || report.PhaseCount != 4 || report.ProgressEventCount != 1 {
		t.Fatalf("counts = tasks:%d phases:%d events:%d", report.TaskCount, report.PhaseCount, report.ProgressEventCount)
	}
	after := planningContentSnapshot(t, store)
	if before != after {
		t.Fatalf("DiagnosePlanning mutated planning files\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestDiagnosePlanningReportsProgressReferences(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "workspace"))
	if err := os.MkdirAll(store.planningPath(), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, store.tasksPath(), []byte(`
version: 1
tasks:
  - id: task-1
    title: Known task
    status: ready
    priority: normal
    phase: unassigned
    created_at: 2026-06-08T12:00:00Z
    updated_at: 2026-06-08T12:00:00Z
`))
	writeFile(t, store.phasesPath(), []byte(`
version: 1
phases:
  - id: p1
    title: Known phase
    status: planned
    order: 1
    created_at: 2026-06-08T12:00:00Z
    updated_at: 2026-06-08T12:00:00Z
`))
	writeFile(t, store.progressPath(), []byte(`{"task_id":"missing-task"}
{"type":"phase_started","phase_id":"missing-phase"}
`))

	report, err := store.DiagnosePlanning(context.Background())
	if err != nil {
		t.Fatalf("DiagnosePlanning returned error: %v", err)
	}
	for _, code := range []string{
		PlanningDiagnosticCodeProgressTypeMissing,
		PlanningDiagnosticCodeProgressTaskUnknown,
		PlanningDiagnosticCodeProgressPhaseUnknown,
	} {
		if !hasPlanningDiagnostic(report, code) {
			t.Fatalf("missing diagnostic %s in %+v", code, report.Diagnostics)
		}
	}
	if report.HasErrors() {
		t.Fatalf("progress reference diagnostics should be warnings: %+v", report.Diagnostics)
	}
}

func hasPlanningDiagnostic(report PlanningDiagnosticReport, code string) bool {
	for _, diagnostic := range report.Diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func planningContentSnapshot(t *testing.T, store *Store) string {
	t.Helper()
	var snapshot string
	for _, path := range []string{store.tasksPath(), store.phasesPath(), store.progressPath(), store.lockPath()} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read snapshot file %s: %v", path, err)
		}
		snapshot += path + "\n" + string(data) + "\n"
	}
	return snapshot
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
