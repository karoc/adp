package cli

import (
	"strings"
	"testing"
)

// TestSafeTextPreservesPlainASCII verifies normal text passes through unchanged.
func TestSafeTextPreservesPlainASCII(t *testing.T) {
	got := safeText("deploy the service")
	if got != "deploy the service" {
		t.Fatalf("plain text changed: got %q", got)
	}
}

// TestSafeTextPreservesNewlineAndTab verifies legitimate multi-line whitespace
// survives so task descriptions and notes still render across lines.
func TestSafeTextPreservesNewlineAndTab(t *testing.T) {
	got := safeText("line one\n\tindented")
	if got != "line one\n\tindented" {
		t.Fatalf("newline/tab not preserved: got %q", got)
	}
}

// TestSafeTextNeutralizesAnsiEscapeSequence verifies an embedded CSI sequence
// (ESC [ 2J clears the screen) can no longer reach the terminal.
func TestSafeTextNeutralizesAnsiEscapeSequence(t *testing.T) {
	malicious := "ok\x1b[2Jspoofed"
	got := safeText(malicious)
	if strings.Contains(got, "\x1b") {
		t.Fatalf("ESC survived sanitization: got %q", got)
	}
	if !strings.Contains(got, "ok") || !strings.Contains(got, "spoofed") {
		t.Fatalf("visible text was dropped: got %q", got)
	}
}

// TestSafeTextNeutralizesCarriageReturn verifies CR (used to overwrite a line
// and spoof earlier output) is replaced.
func TestSafeTextNeutralizesCarriageReturn(t *testing.T) {
	got := safeText("real\x1b[2Kfake\rsafe")
	if strings.Contains(got, "\r") || strings.Contains(got, "\x1b") {
		t.Fatalf("CR/ESC survived sanitization: got %q", got)
	}
}

// TestSafeTextNeutralizesBackspaceAndBell verifies other C0 controls used in
// terminal attacks are replaced.
func TestSafeTextNeutralizesBackspaceAndBell(t *testing.T) {
	got := safeText("a\bb\x07c")
	for _, ch := range []string{"\b", "\x07"} {
		if strings.Contains(got, ch) {
			t.Fatalf("control char survived sanitization: got %q", got)
		}
	}
}

// TestSafeTextHandlesC1Range verifies a properly-encoded C1 control rune
// (U+009B, the 8-bit CSI an attacker could embed in a YAML string and yaml.v3
// would decode to this rune) is neutralized too. Raw invalid UTF-8 bytes are
// out of scope because modern UTF-8 terminals do not honor them as controls.
func TestSafeTextHandlesC1Range(t *testing.T) {
	got := safeText("xy")
	if strings.ContainsRune(got, '') {
		t.Fatalf("C1 control survived: got %q", got)
	}
}

// TestValueOrDashSanitizes verifies the shared helper routes through safeText,
// covering the majority of user-controlled field prints.
func TestValueOrDashSanitizes(t *testing.T) {
	got := valueOrDash("title\x1b[2Jevil")
	if strings.Contains(got, "\x1b") {
		t.Fatalf("valueOrDash left ESC: got %q", got)
	}
	if valueOrDash("") != "-" {
		t.Fatalf("empty valueOrDash = %q, want -", valueOrDash(""))
	}
}
