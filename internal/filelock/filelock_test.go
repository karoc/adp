package filelock

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestWithRequiresPath(t *testing.T) {
	t.Parallel()
	if err := (Locker{}).With(context.Background(), func() error { return nil }); err == nil {
		t.Fatal("With without Path returned nil error")
	}
}

func TestWithReturnsContextErrorWhenAlreadyCanceled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	locker := Locker{Path: filepath.Join(t.TempDir(), ".lock")}
	ran := false
	if err := locker.With(ctx, func() error { ran = true; return nil }); !errors.Is(err, context.Canceled) {
		t.Fatalf("With on canceled ctx error = %v, want context.Canceled", err)
	}
	if ran {
		t.Fatal("fn ran despite canceled context")
	}
}

func TestWithRunsCriticalSectionAndReleasesLock(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), ".lock")
	locker := Locker{Path: path}
	ran := false
	if err := locker.With(context.Background(), func() error {
		ran = true
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("lock file missing inside critical section: %v", err)
		}
		return nil
	}); err != nil {
		t.Fatalf("With returned error: %v", err)
	}
	if !ran {
		t.Fatal("fn did not run")
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("lock file after With error = %v, want not exist", err)
	}
}

func TestWithPropagatesFnError(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), ".lock")
	sentinel := errors.New("boom")
	if err := (Locker{Path: path}).With(context.Background(), func() error { return sentinel }); !errors.Is(err, sentinel) {
		t.Fatalf("With error = %v, want sentinel", err)
	}
	// The lock must be released even when fn fails.
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("lock file after failing fn error = %v, want not exist", err)
	}
}

func TestWithWaitsForFreshLockUntilContextDeadline(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, ".lock")
	// Simulate a live holder: a fresh lock file that never gets released.
	if err := os.WriteFile(path, []byte("held\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	locker := Locker{Path: path, StaleAge: 30 * time.Minute, Retry: time.Millisecond}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	ran := false
	if err := locker.With(ctx, func() error { ran = true; return nil }); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("With against fresh lock error = %v, want context deadline exceeded", err)
	}
	if ran {
		t.Fatal("fn ran despite a fresh lock being held")
	}
}

func TestWithReclaimsStaleLockAndLeavesNoTrash(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, ".lock")
	if err := os.WriteFile(path, []byte("stale\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	staleAge := 30 * time.Minute
	staleTime := time.Now().Add(-staleAge - time.Minute)
	if err := os.Chtimes(path, staleTime, staleTime); err != nil {
		t.Fatal(err)
	}
	locker := Locker{Path: path, StaleAge: staleAge, Retry: time.Millisecond}
	ran := false
	if err := locker.With(context.Background(), func() error { ran = true; return nil }); err != nil {
		t.Fatalf("With against stale lock returned error: %v", err)
	}
	if !ran {
		t.Fatal("fn did not run after reclaiming stale lock")
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("lock file after With error = %v, want not exist", err)
	}
	if _, err := os.Stat(path + ".stale"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stale-break left trash file: %v", err)
	}
}

func TestWithStaleAgeZeroDoesNotReclaim(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, ".lock")
	if err := os.WriteFile(path, []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Even an ancient lock must block when stale reclamation is disabled.
	old := time.Now().Add(-24 * time.Hour)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	locker := Locker{Path: path, Retry: time.Millisecond}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	if err := locker.With(ctx, func() error { return nil }); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("With with StaleAge=0 error = %v, want context deadline exceeded", err)
	}
}

func TestWithSerializesConcurrentHolders(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), ".lock")
	locker := Locker{Path: path, StaleAge: 30 * time.Minute, Retry: time.Millisecond}

	const workers = 8
	var wg sync.WaitGroup
	var mu sync.Mutex
	inside := 0
	maxInside := 0
	total := 0
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = locker.With(context.Background(), func() error {
				mu.Lock()
				inside++
				if inside > maxInside {
					maxInside = inside
				}
				total++
				mu.Unlock()
				// Hold the critical section briefly so an unsynchronized peer
				// would overlap and push maxInside above 1.
				time.Sleep(time.Millisecond)
				mu.Lock()
				inside--
				mu.Unlock()
				return nil
			})
		}()
	}
	wg.Wait()

	if maxInside != 1 {
		t.Fatalf("max concurrent holders = %d, want 1", maxInside)
	}
	if total != workers {
		t.Fatalf("critical section entered %d times, want %d", total, workers)
	}
}
