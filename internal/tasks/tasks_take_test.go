package tasks

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestStoreTakeClaimsHighestPriorityTask(t *testing.T) {
	store := testStore(t)
	base := store.now()
	low, err := store.Add(context.Background(), AddRequest{Title: "Low", Priority: "low"})
	if err != nil {
		t.Fatal(err)
	}
	high, err := store.Add(context.Background(), AddRequest{Title: "High", Priority: "high"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Add(context.Background(), AddRequest{Title: "Review", Status: StatusReview, Priority: "critical"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Claim(context.Background(), ClaimRequest{TaskID: low.ID, Owner: "agent-old", Lease: time.Minute}); err != nil {
		t.Fatal(err)
	}
	store.Now = func() time.Time { return base.Add(2 * time.Minute) }

	taken, err := store.Take(context.Background(), TakeRequest{Owner: "agent-new", Lease: 30 * time.Minute})
	if err != nil {
		t.Fatalf("Take returned error: %v", err)
	}
	if taken.ID != high.ID || taken.Owner != "agent-new" || taken.Status != StatusInProgress {
		t.Fatalf("taken task = %+v, high task = %+v", taken, high)
	}
	if taken.LeaseExpiresAt.IsZero() || !taken.LeaseExpiresAt.Equal(store.now().Add(30*time.Minute)) {
		t.Fatalf("lease expires at = %s", taken.LeaseExpiresAt)
	}

	next, err := store.Take(context.Background(), TakeRequest{Owner: "agent-new"})
	if err != nil {
		t.Fatalf("second Take returned error: %v", err)
	}
	if next.ID != low.ID || next.Owner != "agent-new" || next.Status != StatusInProgress {
		t.Fatalf("second taken task = %+v, low task = %+v", next, low)
	}

	if _, err := store.Take(context.Background(), TakeRequest{Owner: "agent-new"}); !errors.Is(err, ErrNoClaimableTask) {
		t.Fatalf("empty take error = %v, want ErrNoClaimableTask", err)
	}

	progressPath := filepath.Join(store.WorkspaceDir, "planning", "progress.jsonl")
	assertFileContains(t, progressPath, `"type":"task_claimed"`)
	assertFileContains(t, progressPath, `"owner":"agent-new"`)
}

func TestStoreTakeRejectsInvalidRequest(t *testing.T) {
	store := testStore(t)

	if _, err := store.Take(context.Background(), TakeRequest{}); err == nil {
		t.Fatal("Take without owner returned nil error")
	}
	if _, err := store.Take(context.Background(), TakeRequest{Owner: "agent-a", Lease: -time.Second}); err == nil {
		t.Fatal("Take with negative lease returned nil error")
	}
}

func TestStoreTakeDoesNotDuplicateConcurrentClaims(t *testing.T) {
	store := testStore(t)
	for _, title := range []string{"First", "Second", "Third"} {
		if _, err := store.Add(context.Background(), AddRequest{Title: title, Priority: "high"}); err != nil {
			t.Fatal(err)
		}
	}

	const workers = 6
	var wg sync.WaitGroup
	var mu sync.Mutex
	taken := map[string]string{}
	noWork := 0
	errorsSeen := []error{}
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			task, err := store.Take(context.Background(), TakeRequest{Owner: "agent-" + string(rune('a'+i)), Lease: time.Hour})
			mu.Lock()
			defer mu.Unlock()
			if errors.Is(err, ErrNoClaimableTask) {
				noWork++
				return
			}
			if err != nil {
				errorsSeen = append(errorsSeen, err)
				return
			}
			if _, exists := taken[task.ID]; exists {
				errorsSeen = append(errorsSeen, errors.New("duplicate take for "+task.ID))
				return
			}
			taken[task.ID] = task.Owner
		}(i)
	}
	wg.Wait()

	if len(errorsSeen) > 0 {
		t.Fatalf("unexpected take errors: %v", errorsSeen)
	}
	if len(taken) != 3 || noWork != workers-3 {
		t.Fatalf("taken=%v noWork=%d", taken, noWork)
	}
}
