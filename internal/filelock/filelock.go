// Package filelock provides a cross-process advisory lock built on the
// atomic O_CREATE|O_EXCL create of a lock file. Each adp invocation is a
// separate OS process, so in-process mutexes cannot serialize concurrent
// writers to the same on-disk state; a lock file is the portable primitive
// that does.
package filelock

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/karoc/adp/internal/paths"
)

// DefaultRetry is the poll interval used while waiting for a held lock to be
// released when a Locker does not set Retry.
const DefaultRetry = 10 * time.Millisecond

// Locker serializes a critical section across processes by creating Path
// exclusively. Only one process can win the create; others spin until the
// holder releases (removes) the file or, when StaleAge is set, until the lock
// is old enough to be reclaimed.
type Locker struct {
	// Path is the lock file. Its parent directory is created owner-only if
	// missing. Required.
	Path string
	// Now supplies the current time for stamping and staleness checks. Defaults
	// to time.Now when nil.
	Now func() time.Time
	// StaleAge, when positive, lets a waiter reclaim a lock whose file mtime is
	// older than StaleAge — covering a holder that crashed without releasing.
	// Zero disables stale reclamation (the lock is only freed by its holder or
	// context cancellation).
	StaleAge time.Duration
	// Retry is the poll interval while waiting for a fresh lock. Defaults to
	// DefaultRetry when non-positive.
	Retry time.Duration
}

func (l Locker) now() time.Time {
	if l.Now != nil {
		return l.Now()
	}
	return time.Now()
}

func (l Locker) retry() time.Duration {
	if l.Retry > 0 {
		return l.Retry
	}
	return DefaultRetry
}

// With acquires the lock, runs fn while holding it, and releases the lock
// before returning fn's error. It returns ctx.Err() if the context is done
// before the lock is acquired, and a wrapped filesystem error if the lock or
// its directory cannot be created.
func (l Locker) With(ctx context.Context, fn func() error) error {
	if l.Path == "" {
		return errors.New("lock path is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := paths.EnsurePrivateDir(filepath.Dir(l.Path)); err != nil {
		return fmt.Errorf("create lock directory: %w", err)
	}

	for {
		file, err := os.OpenFile(l.Path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, paths.PrivateFileMode)
		if err == nil {
			_, writeErr := fmt.Fprintf(file, "%s\n", l.now().Format(time.RFC3339))
			closeErr := file.Close()
			if writeErr != nil {
				_ = os.Remove(l.Path)
				return fmt.Errorf("write lock: %w", writeErr)
			}
			if closeErr != nil {
				_ = os.Remove(l.Path)
				return fmt.Errorf("close lock: %w", closeErr)
			}
			defer os.Remove(l.Path)
			return fn()
		}
		if !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("create lock: %w", err)
		}

		stale, err := l.stale()
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				// The holder released the lock between our create and stat;
				// retry the exclusive create immediately.
				continue
			}
			return err
		}
		if stale {
			// Atomically claim the stale lock by renaming it. Only one
			// stale-breaker can win the rename; a concurrent breaker that lost
			// the race gets os.ErrNotExist (the lock was already renamed) and
			// retries the exclusive create. This avoids the TOCTOU where two
			// breakers each os.Remove a lock the other just recreated, which
			// would let both enter the critical section (CWE-367).
			trash := l.Path + ".stale"
			if err := os.Rename(l.Path, trash); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					continue
				}
				return fmt.Errorf("claim stale lock: %w", err)
			}
			_ = os.Remove(trash)
			continue
		}

		timer := time.NewTimer(l.retry())
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (l Locker) stale() (bool, error) {
	if l.StaleAge <= 0 {
		return false, nil
	}
	info, err := os.Stat(l.Path)
	if err != nil {
		return false, err
	}
	return l.now().Sub(info.ModTime()) > l.StaleAge, nil
}
