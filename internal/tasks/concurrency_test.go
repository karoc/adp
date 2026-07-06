package tasks

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestStoreConcurrentTaskAddsSerializeWithoutLostUpdates(t *testing.T) {
	store := testStore(t)
	if _, err := store.AddPhase(context.Background(), PhaseAddRequest{ID: "p70", Title: "P70"}); err != nil {
		t.Fatalf("AddPhase returned error: %v", err)
	}

	const workers = 25
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := store.Add(context.Background(), AddRequest{
				Title:    fmt.Sprintf("Concurrent task %02d", i),
				Priority: "high",
				Phase:    "p70",
			})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent Add returned error: %v", err)
		}
	}

	tasks, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(tasks) != workers {
		t.Fatalf("task count = %d, want %d; tasks = %+v", len(tasks), workers, tasks)
	}
	seen := map[string]struct{}{}
	for _, task := range tasks {
		if _, ok := seen[task.ID]; ok {
			t.Fatalf("duplicate task id %q in tasks %+v", task.ID, tasks)
		}
		seen[task.ID] = struct{}{}
		if task.Phase != "p70" {
			t.Fatalf("task %s phase = %q, want p70", task.ID, task.Phase)
		}
	}
	for i := 1; i <= workers; i++ {
		id := fmt.Sprintf("task-20260608-%04d", i)
		if _, ok := seen[id]; !ok {
			t.Fatalf("missing task id %s in %v", id, seen)
		}
	}
}

func TestStoreConcurrentPhaseAddsSerializeOrders(t *testing.T) {
	store := testStore(t)

	const workers = 12
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := store.AddPhase(context.Background(), PhaseAddRequest{
				ID:    fmt.Sprintf("p%02d", i),
				Title: fmt.Sprintf("Phase %02d", i),
			})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent AddPhase returned error: %v", err)
		}
	}

	phases, err := store.ListPhases(context.Background())
	if err != nil {
		t.Fatalf("ListPhases returned error: %v", err)
	}
	if len(phases) != workers {
		t.Fatalf("phase count = %d, want %d; phases = %+v", len(phases), workers, phases)
	}
	orders := map[int]string{}
	for _, phase := range phases {
		if phase.Order < 1 || phase.Order > workers {
			t.Fatalf("phase %s order = %d, want 1..%d", phase.ID, phase.Order, workers)
		}
		if prior, ok := orders[phase.Order]; ok {
			t.Fatalf("duplicate order %d for phases %s and %s", phase.Order, prior, phase.ID)
		}
		orders[phase.Order] = phase.ID
	}
}

func TestStoreConcurrentClaimsSingleWinner(t *testing.T) {
	store := testStore(t)
	task, err := store.Add(context.Background(), AddRequest{Title: "Claim once"})
	if err != nil {
		t.Fatalf("Add returned error: %v", err)
	}

	const workers = 8
	var wg sync.WaitGroup
	results := make(chan claimResult, workers)
	for i := 0; i < workers; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			owner := fmt.Sprintf("agent-%02d", i)
			claimed, err := store.Claim(context.Background(), ClaimRequest{TaskID: task.ID, Owner: owner, Lease: time.Hour})
			results <- claimResult{owner: owner, task: claimed, err: err}
		}()
	}
	wg.Wait()
	close(results)

	winners := []claimResult{}
	claimedErrors := 0
	for result := range results {
		if result.err == nil {
			winners = append(winners, result)
			continue
		}
		if errors.Is(result.err, ErrTaskClaimed) {
			claimedErrors++
			continue
		}
		t.Fatalf("unexpected Claim error for %s: %v", result.owner, result.err)
	}
	if len(winners) != 1 || claimedErrors != workers-1 {
		t.Fatalf("winners=%+v claimedErrors=%d, want one winner and %d ErrTaskClaimed", winners, claimedErrors, workers-1)
	}

	stored, err := store.Get(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if stored.Owner != winners[0].owner || stored.Status != StatusInProgress {
		t.Fatalf("stored task = %+v, winner = %+v", stored, winners[0])
	}
}

type claimResult struct {
	owner string
	task  Task
	err   error
}

func TestStoreConcurrentDuplicatePhaseAddHasSingleWinner(t *testing.T) {
	store := testStore(t)

	const workers = 8
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := store.AddPhase(context.Background(), PhaseAddRequest{ID: "p-duplicate", Title: "Duplicate"})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)

	winners := 0
	duplicates := 0
	for err := range errs {
		if err == nil {
			winners++
			continue
		}
		if strings.Contains(err.Error(), "phase already exists") {
			duplicates++
			continue
		}
		t.Fatalf("unexpected AddPhase error: %v", err)
	}
	if winners != 1 || duplicates != workers-1 {
		t.Fatalf("winners=%d duplicates=%d, want one winner and %d duplicates", winners, duplicates, workers-1)
	}

	phases, err := store.ListPhases(context.Background())
	if err != nil {
		t.Fatalf("ListPhases returned error: %v", err)
	}
	if len(phases) != 1 || phases[0].ID != "p-duplicate" {
		t.Fatalf("phases = %+v", phases)
	}
}

func TestStoreConcurrentPhaseStartsRespectGate(t *testing.T) {
	store := testStore(t)
	if _, err := store.AddPhase(context.Background(), PhaseAddRequest{ID: "p1", Title: "Phase one"}); err != nil {
		t.Fatalf("AddPhase p1 returned error: %v", err)
	}
	if _, err := store.AddPhase(context.Background(), PhaseAddRequest{ID: "p2", Title: "Phase two"}); err != nil {
		t.Fatalf("AddPhase p2 returned error: %v", err)
	}

	var wg sync.WaitGroup
	results := make(chan phaseStartResult, 2)
	for _, id := range []string{"p1", "p2"} {
		id := id
		wg.Add(1)
		go func() {
			defer wg.Done()
			phase, err := store.StartPhase(context.Background(), id)
			results <- phaseStartResult{id: id, phase: phase, err: err}
		}()
	}
	wg.Wait()
	close(results)

	for result := range results {
		switch result.id {
		case "p1":
			if result.err != nil {
				t.Fatalf("StartPhase p1 returned error: %v", result.err)
			}
			if result.phase.Status != PhaseStatusActive {
				t.Fatalf("p1 status = %s, want active", result.phase.Status)
			}
		case "p2":
			if !errors.Is(result.err, ErrPhaseInvalidTransition) {
				t.Fatalf("StartPhase p2 error = %v, want ErrPhaseInvalidTransition", result.err)
			}
		}
	}

	phases, err := store.ListPhases(context.Background())
	if err != nil {
		t.Fatalf("ListPhases returned error: %v", err)
	}
	if len(phases) != 2 || phases[0].ID != "p1" || phases[0].Status != PhaseStatusActive || phases[1].ID != "p2" || phases[1].Status != PhaseStatusPlanned {
		t.Fatalf("phases = %+v", phases)
	}
}

type phaseStartResult struct {
	id    string
	phase Phase
	err   error
}

func TestStoreConcurrentPlanImportsMergeWithoutLostUpdates(t *testing.T) {
	store := testStore(t)

	const workers = 8
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			phaseID := fmt.Sprintf("p-import-%02d", i)
			_, err := store.ApplyPlanImport(context.Background(), PlanImportRequest{
				Phases: []PlanImportPhase{{ID: phaseID, Title: fmt.Sprintf("Import phase %02d", i)}},
				Tasks: []PlanImportTask{{
					Title:    fmt.Sprintf("Import task %02d", i),
					Priority: "medium",
					Phase:    phaseID,
				}},
			})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent ApplyPlanImport returned error: %v", err)
		}
	}

	phases, err := store.ListPhases(context.Background())
	if err != nil {
		t.Fatalf("ListPhases returned error: %v", err)
	}
	if len(phases) != workers {
		t.Fatalf("phase count = %d, want %d; phases = %+v", len(phases), workers, phases)
	}

	tasks, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(tasks) != workers {
		t.Fatalf("task count = %d, want %d; tasks = %+v", len(tasks), workers, tasks)
	}
	knownPhases := map[string]struct{}{}
	for _, phase := range phases {
		knownPhases[phase.ID] = struct{}{}
	}
	seenTasks := map[string]struct{}{}
	for _, task := range tasks {
		if _, ok := seenTasks[task.ID]; ok {
			t.Fatalf("duplicate task id %q in tasks %+v", task.ID, tasks)
		}
		seenTasks[task.ID] = struct{}{}
		if _, ok := knownPhases[task.Phase]; !ok {
			t.Fatalf("task %s references missing phase %q", task.ID, task.Phase)
		}
	}
}

func TestStoreConcurrentDuplicatePlanImportHasSingleWinner(t *testing.T) {
	store := testStore(t)

	const workers = 8
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := store.ApplyPlanImport(context.Background(), PlanImportRequest{
				Phases: []PlanImportPhase{{ID: "p-import-duplicate", Title: "Duplicate import"}},
				Tasks: []PlanImportTask{{Title: "Duplicate import task", Phase: "p-import-duplicate"}},
			})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)

	winners := 0
	duplicates := 0
	for err := range errs {
		if err == nil {
			winners++
			continue
		}
		if strings.Contains(err.Error(), "phase already exists") {
			duplicates++
			continue
		}
		t.Fatalf("unexpected ApplyPlanImport error: %v", err)
	}
	if winners != 1 || duplicates != workers-1 {
		t.Fatalf("winners=%d duplicates=%d, want one winner and %d duplicates", winners, duplicates, workers-1)
	}

	phases, err := store.ListPhases(context.Background())
	if err != nil {
		t.Fatalf("ListPhases returned error: %v", err)
	}
	if len(phases) != 1 || phases[0].ID != "p-import-duplicate" {
		t.Fatalf("phases = %+v", phases)
	}
	tasks, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(tasks) != 1 || tasks[0].Phase != "p-import-duplicate" {
		t.Fatalf("tasks = %+v", tasks)
	}
}
