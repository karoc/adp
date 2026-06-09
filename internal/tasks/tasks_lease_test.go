package tasks

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreRenewsTaskLeaseAndListsStaleClaims(t *testing.T) {
	store := testStore(t)
	base := store.now()
	active, err := store.Add(context.Background(), AddRequest{Title: "Active lease", Priority: "high"})
	if err != nil {
		t.Fatal(err)
	}
	stale, err := store.Add(context.Background(), AddRequest{Title: "Expired lease", Priority: "critical"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Claim(context.Background(), ClaimRequest{TaskID: active.ID, Owner: "agent-a", Lease: 2 * time.Hour}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Claim(context.Background(), ClaimRequest{TaskID: stale.ID, Owner: "agent-b", Lease: time.Minute}); err != nil {
		t.Fatal(err)
	}

	renewed, err := store.Renew(context.Background(), RenewRequest{TaskID: active.ID, Owner: "agent-a", Lease: 3 * time.Hour})
	if err != nil {
		t.Fatalf("Renew returned error: %v", err)
	}
	if got, want := renewed.LeaseExpiresAt, base.Add(3*time.Hour); !got.Equal(want) {
		t.Fatalf("renewed lease expires at = %s, want %s", got, want)
	}

	store.Now = func() time.Time { return base.Add(2 * time.Minute) }
	tasks, err := store.Stale(context.Background())
	if err != nil {
		t.Fatalf("Stale returned error: %v", err)
	}
	if len(tasks) != 1 || tasks[0].ID != stale.ID {
		t.Fatalf("stale tasks = %+v, want %s only", tasks, stale.ID)
	}

	progressPath := filepath.Join(store.WorkspaceDir, "planning", "progress.jsonl")
	assertFileContains(t, progressPath, `"type":"task_lease_renewed"`)
	assertFileContains(t, progressPath, `"owner":"agent-a"`)
}

func TestStoreRenewRejectsInvalidOwnerLeaseAndTerminalTasks(t *testing.T) {
	store := testStore(t)
	task, err := store.Add(context.Background(), AddRequest{Title: "Renew guarded"})
	if err != nil {
		t.Fatal(err)
	}
	done, err := store.Add(context.Background(), AddRequest{Title: "Done task", Status: StatusDone})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Claim(context.Background(), ClaimRequest{TaskID: task.ID, Owner: "agent-a", Lease: time.Hour}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Renew(context.Background(), RenewRequest{TaskID: task.ID, Owner: "agent-b", Lease: time.Hour}); !errors.Is(err, ErrTaskOwnerMismatch) {
		t.Fatalf("owner mismatch error = %v, want ErrTaskOwnerMismatch", err)
	}
	if _, err := store.Renew(context.Background(), RenewRequest{TaskID: task.ID, Owner: "agent-a"}); err == nil {
		t.Fatal("Renew without positive lease returned nil error")
	}
	if _, err := store.Renew(context.Background(), RenewRequest{TaskID: task.ID, Owner: "agent-a", Lease: -time.Second}); err == nil {
		t.Fatal("Renew with negative lease returned nil error")
	}
	if _, err := store.Renew(context.Background(), RenewRequest{TaskID: done.ID, Owner: "agent-a", Lease: time.Hour}); !errors.Is(err, ErrTaskOwnerMismatch) {
		t.Fatalf("unclaimed terminal renew error = %v, want ErrTaskOwnerMismatch", err)
	}
}
