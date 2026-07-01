package cli

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// safeText neutralizes terminal control characters in user-controlled text
// before it is written to a terminal (CWE-150). Planning data (task titles,
// descriptions, phase goals, owners, ...) can be edited directly in the shared
// workspace YAML files, so a malicious entry could embed ANSI escape sequences
// that move the cursor, clear the screen, or spoof output when another operator
// runs `adp tasks list`/`show`, `adp phase show`, or `adp progress report`.
//
// Legitimate whitespace (newline and tab) is preserved so multi-line
// descriptions and notes still render correctly; every other control rune —
// including ESC (0x1B), carriage return, backspace, and the C1 range — is
// replaced with the Unicode replacement character so it can no longer act as a
// control sequence. JSON output is not routed through this helper because the
// encoder already escapes control characters.
//
// The value must also be well-formed UTF-8 before it can take the passthrough
// fast path. A raw byte such as 0x9B is a single-byte C1 CSI (equivalent to
// "ESC [") yet, being invalid UTF-8, it decodes to utf8.RuneError during a
// range/IndexFunc scan, and RuneError is not a control rune — so a naive
// control-only check would wave it straight through to the terminal. Routing
// any non-UTF-8 value through strings.Map re-encodes those stray bytes as the
// replacement character, closing that bypass and guaranteeing valid UTF-8 out.
func safeText(value string) string {
	if value == "" {
		return value
	}
	if utf8.ValidString(value) && strings.IndexFunc(value, isUnsafeControl) == -1 {
		return value
	}
	return strings.Map(func(r rune) rune {
		if isUnsafeControl(r) {
			return '�'
		}
		return r
	}, value)
}

func isUnsafeControl(r rune) bool {
	if r == '\n' || r == '\t' {
		return false
	}
	return unicode.IsControl(r)
}
