package tasks

import (
	"context"
	"fmt"
	"time"

	"github.com/karoc/adp/internal/filelock"
)

const (
	planningLockFile     = ".lock"
	planningLockStaleAge = 30 * time.Minute
	planningLockRetry    = 10 * time.Millisecond
)

// withPlanningLock serializes mutating planning operations across processes.
// Each adp invocation is a separate OS process, so the planning ledger
// (tasks.yaml, phases.yaml, progress.jsonl) is protected by an on-disk lock
// rather than an in-process mutex. The cross-process locking primitive lives
// in the shared filelock package; this wrapper pins the planning lock path and
// the stale-reclaim policy.
func (s *Store) withPlanningLock(ctx context.Context, fn func() error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := s.ensurePlanningDir(); err != nil {
		return fmt.Errorf("create planning directory: %w", err)
	}
	locker := filelock.Locker{
		Path:     s.lockPath(),
		Now:      s.now,
		StaleAge: planningLockStaleAge,
		Retry:    planningLockRetry,
	}
	return locker.With(ctx, fn)
}
