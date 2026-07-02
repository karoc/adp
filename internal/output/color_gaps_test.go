package output

import (
	"os"
	"testing"
)

// TestColorizefFormats covers Colorizef, which was previously untested (0%).
func TestColorizefFormats(t *testing.T) {
	origEnabled := colorEnabled
	colorEnabled = true
	defer func() { colorEnabled = origEnabled }()

	got := Colorizef(ColorMagenta, "value=%d", 42)
	want := "\033[35mvalue=42\033[0m"
	if got != want {
		t.Errorf("Colorizef(ColorMagenta, ...) = %q, want %q", got, want)
	}
}

// TestBoldfFormats covers Boldf, which was previously untested (0%).
func TestBoldfFormats(t *testing.T) {
	origEnabled := colorEnabled
	colorEnabled = true
	defer func() { colorEnabled = origEnabled }()

	got := Boldf("title: %s", "hello")
	want := "\033[1mtitle: hello\033[0m"
	if got != want {
		t.Errorf("Boldf(...) = %q, want %q", got, want)
	}
}

// TestColorizeUnknownColor covers the branch where the requested color has no
// ANSI code registered: Colorize must return the text unchanged.
func TestColorizeUnknownColor(t *testing.T) {
	origEnabled := colorEnabled
	colorEnabled = true
	defer func() { colorEnabled = origEnabled }()

	// Color(999) is not present in colorCodes.
	got := Colorize(Color(999), "plain")
	if got != "plain" {
		t.Errorf("Colorize(unknown, %q) = %q, want %q", "plain", got, "plain")
	}
}

// TestColorizefDisabled covers Colorizef returning unmodified text when color
// output is disabled.
func TestColorizefDisabled(t *testing.T) {
	origEnabled := colorEnabled
	colorEnabled = false
	defer func() { colorEnabled = origEnabled }()

	got := Colorizef(ColorRed, "n=%d", 7)
	if got != "n=7" {
		t.Errorf("Colorizef disabled = %q, want %q", got, "n=7")
	}
}

// TestSupportsColorNoColorSet exercises the NO_COLOR early-return branch of
// supportsColor deterministically (independent of whether stdout is a TTY).
func TestSupportsColorNoColorSet(t *testing.T) {
	orig, had := os.LookupEnv("NO_COLOR")
	if err := os.Setenv("NO_COLOR", "1"); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	defer func() {
		if had {
			os.Setenv("NO_COLOR", orig)
		} else {
			os.Unsetenv("NO_COLOR")
		}
	}()

	if supportsColor() {
		t.Error("supportsColor() = true with NO_COLOR set, want false")
	}
}

// TestSupportsColorNonTTY exercises the non-character-device branch of
// supportsColor by pointing at a regular file via a temporarily swapped
// stdout. It restores os.Stdout afterwards.
func TestSupportsColorNonTTY(t *testing.T) {
	// Ensure NO_COLOR does not short-circuit before the TTY check.
	orig, had := os.LookupEnv("NO_COLOR")
	os.Unsetenv("NO_COLOR")
	defer func() {
		if had {
			os.Setenv("NO_COLOR", orig)
		}
	}()

	f, err := os.CreateTemp(t.TempDir(), "stdout")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	defer f.Close()

	saved := os.Stdout
	os.Stdout = f
	defer func() { os.Stdout = saved }()

	if supportsColor() {
		t.Error("supportsColor() = true when stdout is a regular file, want false")
	}
}
