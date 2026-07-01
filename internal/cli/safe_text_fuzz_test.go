package cli

import (
	"testing"
	"unicode"
	"unicode/utf8"
)

// FuzzSafeText asserts the core safety property that motivates the helper:
// whatever text goes in, the output must contain no unsafe terminal control
// characters (CWE-150). It also checks that the transformation preserves
// legitimate content — newlines, tabs, and ordinary printable runes must
// survive unchanged — so the sanitizer cannot be accused of silently mangling
// well-formed input.
func FuzzSafeText(f *testing.F) {
	seeds := []string{
		"",
		"plain task title",
		"multi\nline\tdescription",
		"\x1b[2J\x1b[H", // clear-screen escape sequence
		"\x1b]0;spoofed title\a",
		"bell\aand\bbackspace",
		"carriage\rreturn",
		"c1next-linecsi",
		"emoji 🚀 and accents éàü",
		"\x00\x01\x02\x03",
		"mixed \x1b[31mred\x1b[0m and \n newline",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, in string) {
		out := safeText(in)

		// Core property: no unsafe control rune may survive in the output.
		for i, r := range out {
			if isUnsafeControl(r) {
				t.Fatalf("safeText output retains unsafe control rune %U at byte %d for input %q", r, i, in)
			}
		}

		// The output must remain valid UTF-8: strings.Map emits the replacement
		// character for stripped runes, never a broken encoding.
		if !utf8.ValidString(out) {
			t.Fatalf("safeText produced invalid UTF-8 for input %q", in)
		}

		// Preservation: newlines/tabs and other safe runes must be retained, so
		// the rune count is invariant — each unsafe rune maps to exactly one
		// replacement rune, and each stray non-UTF-8 byte likewise becomes one
		// replacement rune.
		outRunes := utf8.RuneCountInString(out)
		if outRunes != utf8.RuneCountInString(in) {
			// Length may only change if input actually held unsafe runes; each
			// unsafe rune maps to exactly one replacement rune, so the total
			// rune count must be preserved regardless.
			t.Fatalf("safeText changed rune count: in=%d out=%d for input %q",
				utf8.RuneCountInString(in), outRunes, in)
		}

		// When the input is well-formed UTF-8 with no unsafe control runes,
		// safeText must be an exact identity — it promises a fast-path
		// passthrough in that case. Invalid UTF-8 is intentionally excluded: a
		// stray byte such as 0x9B is a C1 CSI and must be rewritten, so it does
		// not qualify for passthrough even though it decodes to a non-control
		// RuneError.
		if utf8.ValidString(in) && !hasUnsafeControl(in) && out != in {
			t.Fatalf("safeText altered already-safe input %q into %q", in, out)
		}
	})
}

// hasUnsafeControl reports whether s contains any rune the sanitizer would
// strip. Kept local to the test so the property check does not depend on
// safeText's internal fast-path implementation.
func hasUnsafeControl(s string) bool {
	for _, r := range s {
		if r != '\n' && r != '\t' && unicode.IsControl(r) {
			return true
		}
	}
	return false
}
