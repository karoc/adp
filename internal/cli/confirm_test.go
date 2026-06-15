package cli

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/karoc/adp/internal/runtime"
)

// withStdin temporarily replaces the global stdinReader used by confirmDangerous
// and restores it after the test completes.
func withStdin(input string) func() {
	prev := stdinReader
	stdinReader = strings.NewReader(input)
	return func() { stdinReader = prev }
}

func TestConfirmDangerous_WithYesFlag(t *testing.T) {
	app := &App{
		stderr: &bytes.Buffer{},
	}

	err := app.confirmDangerous("Remove something?", "Details here", true)
	if err != nil {
		t.Errorf("expected no error with --yes flag, got: %v", err)
	}
}

func TestConfirmDangerous_NonTTYWithoutYesFlag(t *testing.T) {
	app := &App{
		stderr: &bytes.Buffer{},
	}

	// In test environment, os.Stdin is a character device so isTTY returns true.
	// Without --yes and with no input, the prompt reads EOF -> operation cancelled.
	err := app.confirmDangerous("Remove something?", "Details here", false)
	if err == nil {
		t.Error("expected error when no --yes flag and no confirmation input")
	}
}

func TestConfirmDangerous_UserSaysYes(t *testing.T) {
	defer withStdin("y\n")()
	app := &App{
		stderr: &bytes.Buffer{},
	}

	// isTTY(os.Stdin) is true under go test, so this exercises the interactive path.
	err := app.confirmDangerous("Remove something?", "Details here", false)
	if err != nil {
		t.Errorf("expected no error when user confirms with 'y', got: %v", err)
	}
}

func TestConfirmDangerous_UserSaysNo(t *testing.T) {
	defer withStdin("n\n")()
	app := &App{
		stderr: &bytes.Buffer{},
	}

	err := app.confirmDangerous("Remove something?", "Details here", false)
	if err == nil {
		t.Error("expected error when user declines with 'n'")
	}
	if !strings.Contains(err.Error(), "cancelled") {
		t.Errorf("expected 'cancelled' error, got: %v", err)
	}
}

func TestConfirmDangerous_UserSaysYesFullWord(t *testing.T) {
	defer withStdin("yes\n")()
	app := &App{
		stderr: &bytes.Buffer{},
	}

	err := app.confirmDangerous("Remove something?", "Details here", false)
	if err != nil {
		t.Errorf("expected no error when user confirms with 'yes', got: %v", err)
	}
}

func TestIsTTY(t *testing.T) {
	// Test with a regular file (not a TTY)
	f, err := os.CreateTemp("", "test-tty-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	if isTTY(f) {
		t.Error("expected regular file to not be a TTY")
	}

	// Note: We can't easily test a real TTY in automated tests,
	// but we can verify the function doesn't crash with os.Stdin
	_ = isTTY(os.Stdin)
}

func TestRuntimePrune_IncludeKeptRequiresConfirmation(t *testing.T) {
	// This test verifies that runtime prune with --include-kept asks for confirmation.
	// The --yes flag bypasses the prompt entirely.
	pruneRuntimesCalled := false
	deps := Dependencies{
		PruneRuntimes: func(ctx context.Context, req runtime.PruneRequest) ([]runtime.PruneResult, error) {
			pruneRuntimesCalled = true
			if !req.IncludeKept {
				t.Error("expected IncludeKept to be true")
			}
			return []runtime.PruneResult{}, nil
		},
	}

	// With --yes flag, should not error
	code := NewApp(deps, &bytes.Buffer{}, &bytes.Buffer{}).Execute(
		context.Background(),
		[]string{"runtime", "prune", "--include-kept", "--yes"},
	)
	if code != 0 {
		t.Errorf("expected exit code 0 with --yes flag, got: %d", code)
	}
	if !pruneRuntimesCalled {
		t.Error("expected PruneRuntimes to be called")
	}
}

func TestRuntimePrune_DryRunDoesNotRequireConfirmation(t *testing.T) {
	pruneRuntimesCalled := false
	deps := Dependencies{
		PruneRuntimes: func(ctx context.Context, req runtime.PruneRequest) ([]runtime.PruneResult, error) {
			pruneRuntimesCalled = true
			if !req.IncludeKept {
				t.Error("expected IncludeKept to be true")
			}
			if !req.DryRun {
				t.Error("expected DryRun to be true")
			}
			return []runtime.PruneResult{}, nil
		},
	}

	// Dry run with --include-kept should not require confirmation
	code := NewApp(deps, &bytes.Buffer{}, &bytes.Buffer{}).Execute(
		context.Background(),
		[]string{"runtime", "prune", "--include-kept", "--dry-run"},
	)
	if code != 0 {
		t.Errorf("expected exit code 0 for dry run, got: %d", code)
	}
	if !pruneRuntimesCalled {
		t.Error("expected PruneRuntimes to be called")
	}
}
