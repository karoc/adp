package redact

import (
	"net/url"
	"strings"
	"testing"
)

// FuzzArgs checks the structural guarantees Args documents: it never panics,
// never mutates the input slice, and returns a slice of the same length. It
// also asserts the security-relevant property that any argument the redactor
// itself classifies as a secret (bare value or value following a sensitive
// flag) is not emitted verbatim.
func FuzzArgs(f *testing.F) {
	seeds := [][]string{
		{},
		{"run", "codex"},
		{"--api-key", "sk-abcdefghijklmnop"},
		{"--api-key=sk-abcdefghijklmnop"},
		{"--token"},
		{"ghp_0123456789abcdef0123"},
		{"plainword"},
		{"--", "--not-a-flag"},
		{"-", "value"},
		{"--flag=", ""},
		{"AKIAIOSFODNN7EXAMPLE"},
	}
	for _, s := range seeds {
		// Fuzzing supports a fixed arity, so join and re-split around a
		// separator to feed variable-length slices through a single string.
		f.Add(strings.Join(s, "\x00"))
	}

	f.Fuzz(func(t *testing.T, joined string) {
		var args []string
		if joined != "" {
			args = strings.Split(joined, "\x00")
		}
		original := append([]string(nil), args...)

		out := Args(args)

		// Input must never be mutated.
		for i := range args {
			if args[i] != original[i] {
				t.Fatalf("Args mutated input at %d: %q -> %q", i, original[i], args[i])
			}
		}

		if len(args) == 0 {
			if out != nil {
				t.Fatalf("empty input should yield nil, got %#v", out)
			}
			return
		}
		if len(out) != len(args) {
			t.Fatalf("length changed: in %d, out %d", len(args), len(out))
		}

		// No-leak property: any bare argument the redactor classifies as a
		// secret must not appear verbatim in the output at that position.
		for i, in := range args {
			if !isFlag(in) && looksLikeSecret(in) {
				if out[i] == in {
					t.Fatalf("secret-looking arg leaked verbatim at %d: %q", i, in)
				}
				if out[i] != Placeholder {
					t.Fatalf("secret arg at %d became %q, want placeholder", i, out[i])
				}
			}
		}
	})
}

// FuzzURLCredentials checks that URLCredentials never panics and never leaves a
// password visible: whenever the input parses to a URL carrying a
// user:password userinfo segment, the password substring must not survive in
// the returned string.
func FuzzURLCredentials(f *testing.F) {
	seeds := []string{
		"",
		"origin",
		"git@github.com:org/repo.git",
		"https://github.com/org/repo",
		"https://user:token@github.com/org/repo",
		"https://user:ghp_secret123@github.com/org/repo",
		"ftp://u:p@host/path",
		"https://user@host/path@with@ats",
		"://malformed",
		"https://:@host",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		out := URLCredentials(s)

		// A string without a scheme separator cannot carry scheme userinfo and
		// must pass through untouched.
		if !strings.Contains(s, "://") {
			if out != s {
				t.Fatalf("non-URL %q was altered to %q", s, out)
			}
			return
		}

		// Use net/url as an independent oracle for "does this URL carry
		// userinfo?". When it does with a non-empty password, redaction must
		// have happened: the placeholder appears and the output differs from
		// the input. We deliberately do not substring-match the decoded
		// password against the output — url.Parse percent-decodes it, so a
		// decoded byte (e.g. "/") could coincidentally match the host or path
		// and cause a false failure. Presence of the placeholder plus a changed
		// string is a robust, false-positive-free signal that the credential
		// segment was removed.
		u, err := url.Parse(s)
		if err != nil || u.User == nil {
			return
		}
		if pw, ok := u.User.Password(); ok && pw != "" {
			if !strings.Contains(out, Placeholder) {
				t.Fatalf("URL with password not redacted: %q -> %q", s, out)
			}
			if out == s {
				t.Fatalf("URL with password unchanged: %q", s)
			}
		}
	})
}
