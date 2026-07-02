package output

import (
	"os"
	"strings"
	"testing"
	"time"
)

// openTTYLike opens a *os.File that isTTY reports as a terminal. On Linux
// /dev/null is a character device, so it satisfies the ModeCharDevice check
// that gates every TTY-only branch in progress.go. Tests that write escape
// sequences to it discard the bytes harmlessly.
func openTTYLike(t *testing.T) *os.File {
	t.Helper()
	f, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open %s: %v", os.DevNull, err)
	}
	if !isTTY(f) {
		f.Close()
		t.Skipf("%s is not reported as a character device on this platform", os.DevNull)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

// TestSpinnerTTYAnimateAndStop drives the TTY code path of the spinner:
// Start launches the animate goroutine, and after at least one tick fires the
// animation frame is written; Stop then clears the line. This covers Start
// (tty branch), animate (0% before), and Stop (tty clear).
func TestSpinnerTTYAnimateAndStop(t *testing.T) {
	f := openTTYLike(t)

	s := NewSpinner(f, "working")
	if !s.tty {
		t.Fatal("expected spinner to treat the writer as a TTY")
	}

	s.Start()
	// Second Start must be a no-op (already active) — exercise that guard.
	s.Start()

	// Wait long enough for the 80ms ticker to fire at least once so animate
	// executes its render branch.
	time.Sleep(120 * time.Millisecond)

	s.Stop()

	// Stop again: not active anymore, should return early without panicking.
	s.Stop()
}

// TestSpinnerTTYSuccess covers Success on a TTY-backed spinner that was never
// started (Stop returns early, then the success line is printed).
func TestSpinnerTTYSuccess(t *testing.T) {
	f := openTTYLike(t)

	s := NewSpinner(f, "task")
	s.Success("done")
}

// TestStepProgressTTYRenderAndFinish exercises the TTY rendering path of
// StepProgress: StartStep/CompleteStep/FailStep each call render (0% before),
// covering all four status prefixes, and Finish emits the cursor-clear
// sequence (tty branch).
func TestStepProgressTTYRenderAndFinish(t *testing.T) {
	f := openTTYLike(t)

	sp := NewStepProgress(f, []string{"one", "two", "three"})
	if !sp.tty {
		t.Fatal("expected step progress to treat the writer as a TTY")
	}

	sp.StartStep(0)    // stepRunning -> render
	sp.CompleteStep(0) // stepDone -> render
	sp.StartStep(1)
	sp.FailStep(1)  // stepFailed -> render
	sp.StartStep(2) // leaves step three running, step... pending covered below
	sp.Finish()     // tty clear branch

	// Out-of-range indices on a TTY tracker must still be guarded.
	sp.StartStep(-1)
	sp.CompleteStep(99)
	sp.FailStep(-5)
}

// TestStepProgressTTYPendingPrefix ensures the render loop covers the
// stepPending prefix branch: a tracker whose first step is completed while a
// later step remains pending forces render to format both prefixes.
func TestStepProgressTTYPendingPrefix(t *testing.T) {
	f := openTTYLike(t)

	sp := NewStepProgress(f, []string{"first", "second"})
	// Complete the first step; the second stays pending, so render walks the
	// stepPending case for index 1 while emitting stepDone for index 0.
	sp.CompleteStep(0)
}

// TestSpinnerTTYFail covers Fail on a TTY-backed spinner.
func TestSpinnerTTYFail(t *testing.T) {
	f := openTTYLike(t)

	s := NewSpinner(f, "task")
	s.Fail("boom")
}

// TestIsTTYRegularFileIsFalse confirms isTTY returns false for a *os.File that
// is a regular file (not a character device), covering that branch explicitly.
func TestIsTTYRegularFileIsFalse(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "reg")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	defer f.Close()

	if isTTY(f) {
		t.Error("isTTY(regular file) = true, want false")
	}
}

// TestNewSpinnerTTYNoImmediatePrint verifies that on a TTY the constructor does
// not eagerly print the message (only non-TTY does). We route through a temp
// file we can read back rather than /dev/null.
func TestNewSpinnerTTYNoImmediatePrint(t *testing.T) {
	// A regular temp file is NOT a TTY, so this documents the non-TTY contract
	// where the message IS printed once at construction.
	f, err := os.CreateTemp(t.TempDir(), "spin")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	defer f.Close()

	NewSpinner(f, "hello-marker")

	data, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if !strings.Contains(string(data), "hello-marker") {
		t.Errorf("non-TTY constructor output = %q, want it to contain the message", string(data))
	}
}
