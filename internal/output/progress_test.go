package output

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestSpinner_NonTTY(t *testing.T) {
	var buf bytes.Buffer
	spinner := NewSpinner(&buf, "Processing...")

	// In non-TTY mode, should print message immediately
	output := buf.String()
	if !strings.Contains(output, "Processing...") {
		t.Errorf("expected message in output, got: %q", output)
	}

	// Start should be no-op in non-TTY
	spinner.Start()
	time.Sleep(50 * time.Millisecond)
	spinner.Stop()

	// Should not have added animation characters
	output = buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 1 {
		t.Errorf("expected single line output in non-TTY, got %d lines", len(lines))
	}
}

func TestSpinner_Success(t *testing.T) {
	var buf bytes.Buffer
	spinner := NewSpinner(&buf, "Processing...")

	spinner.Success("Done successfully")

	output := buf.String()
	if !strings.Contains(output, "Done successfully") {
		t.Errorf("expected success message in output, got: %q", output)
	}
	if !strings.Contains(output, "✓") {
		t.Errorf("expected success checkmark in output, got: %q", output)
	}
}

func TestSpinner_Fail(t *testing.T) {
	var buf bytes.Buffer
	spinner := NewSpinner(&buf, "Processing...")

	spinner.Fail("Operation failed")

	output := buf.String()
	if !strings.Contains(output, "Operation failed") {
		t.Errorf("expected failure message in output, got: %q", output)
	}
	if !strings.Contains(output, "✗") {
		t.Errorf("expected failure mark in output, got: %q", output)
	}
}

func TestSpinner_StopIdempotent(t *testing.T) {
	var buf bytes.Buffer
	spinner := NewSpinner(&buf, "Processing...")

	// Multiple stops should not panic
	spinner.Stop()
	spinner.Stop()
	spinner.Stop()
}

func TestStepProgress_NonTTY(t *testing.T) {
	var buf bytes.Buffer
	steps := []string{"Step 1", "Step 2", "Step 3"}
	progress := NewStepProgress(&buf, steps)

	progress.StartStep(0)
	output := buf.String()
	if !strings.Contains(output, "Step 1") {
		t.Errorf("expected Step 1 in output, got: %q", output)
	}

	progress.CompleteStep(0)
	output = buf.String()
	if !strings.Contains(output, "✓") {
		t.Errorf("expected checkmark in output, got: %q", output)
	}

	progress.StartStep(1)
	progress.CompleteStep(1)

	progress.StartStep(2)
	progress.CompleteStep(2)

	progress.Finish()
}

func TestStepProgress_BoundsCheck(t *testing.T) {
	var buf bytes.Buffer
	steps := []string{"Step 1"}
	progress := NewStepProgress(&buf, steps)

	// Should not panic on invalid indices
	progress.StartStep(-1)
	progress.StartStep(999)
	progress.CompleteStep(-1)
	progress.CompleteStep(999)
	progress.FailStep(-1)
	progress.FailStep(999)
}

func TestStepProgress_AllStatuses(t *testing.T) {
	var buf bytes.Buffer
	steps := []string{"Step 1", "Step 2", "Step 3"}
	progress := NewStepProgress(&buf, steps)

	progress.StartStep(0)
	progress.CompleteStep(0)

	progress.StartStep(1)
	progress.FailStep(1)

	progress.StartStep(2)
	// Leave pending

	progress.Finish()

	// In non-TTY mode, should have output for started steps
	output := buf.String()
	if !strings.Contains(output, "Step 1") {
		t.Errorf("expected Step 1 in output, got: %q", output)
	}
	if !strings.Contains(output, "Step 2") {
		t.Errorf("expected Step 2 in output, got: %q", output)
	}
}

func TestNewSpinner_NilWriter(t *testing.T) {
	// Should not panic with nil writer
	spinner := NewSpinner(nil, "test")
	spinner.Start()
	time.Sleep(10 * time.Millisecond)
	spinner.Stop()
}

func TestNewStepProgress_NilWriter(t *testing.T) {
	// Should not panic with nil writer
	progress := NewStepProgress(nil, []string{"step"})
	progress.StartStep(0)
	progress.CompleteStep(0)
	progress.Finish()
}
