package tasks

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"
)

const (
	planningLockFile     = ".lock"
	planningLockStaleAge = 30 * time.Minute
	planningLockRetry    = 10 * time.Millisecond
)

func (s *Store) withPlanningLock(ctx context.Context, fn func() error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.MkdirAll(s.planningPath(), 0o755); err != nil {
		return fmt.Errorf("create planning directory: %w", err)
	}

	lockPath := s.lockPath()
	for {
		file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			_, writeErr := fmt.Fprintf(file, "%s\n", s.now().Format(time.RFC3339))
			closeErr := file.Close()
			if writeErr != nil {
				_ = os.Remove(lockPath)
				return fmt.Errorf("write planning lock: %w", writeErr)
			}
			if closeErr != nil {
				_ = os.Remove(lockPath)
				return fmt.Errorf("close planning lock: %w", closeErr)
			}
			defer os.Remove(lockPath)
			return fn()
		}
		if !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("create planning lock: %w", err)
		}

		stale, err := planningLockStale(lockPath, s.now())
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return err
		}
		if stale {
			_ = os.Remove(lockPath)
			continue
		}

		timer := time.NewTimer(planningLockRetry)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func planningLockStale(path string, now time.Time) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return now.Sub(info.ModTime()) > planningLockStaleAge, nil
}
