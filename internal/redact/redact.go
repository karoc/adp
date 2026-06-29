// Package redact removes secret-looking values from data that ADP records to
// the local event log or prints back to operators. It is intentionally
// conservative: flag names are preserved so operators can still recognize the
// shape of a recorded command and refill a secret by hand, while only the
// value that looks like a credential is replaced.
package redact

import (
	"math"
	"strings"
)

// Placeholder replaces values that look like secrets in redacted output.
const Placeholder = "***REDACTED***"

const (
	// minStandaloneSecretLen is the minimum length for a bare argument (one
	// not introduced by a sensitive flag name) to be considered a secret on
	// entropy grounds alone.
	minStandaloneSecretLen = 24
	// minPrefixedSecretLen is the minimum length for a value carrying a known
	// secret prefix (for example "sk-" or "ghp_") to be redacted.
	minPrefixedSecretLen = 12
	// entropyThreshold is the Shannon entropy (bits per character) a bare
	// value must reach before it is treated as a secret.
	entropyThreshold = 4.0
)

// sensitiveNameParts are lowercase substrings that mark a flag name as carrying
// a credential value. Matching is substring-based so "--api-key" and "--apikey"
// are both covered.
var sensitiveNameParts = []string{
	"key", "secret", "token", "password", "passwd",
	"auth", "credential", "cred", "private", "apikey",
}

// knownSecretPrefixes are common credential prefixes used by popular providers.
// A value carrying one of these prefixes is redacted even when its entropy is
// modest, because the prefix itself is a strong signal.
var knownSecretPrefixes = []string{
	"sk-", "pk-", "rk_", "ghp_", "gho_", "ghu_", "ghs_", "ghr_",
	"github_pat_", "glpat-", "xoxb-", "xoxp-", "xoxa-", "xoxs-",
	"AKIA", "ASIA", "AIza", "ya29.", "eyJ",
}

// Args returns a copy of command arguments with values that look like secrets
// replaced by Placeholder. The input slice is never mutated. When there is
// nothing to redact the arguments are returned in a fresh slice so callers can
// safely retain the result.
func Args(args []string) []string {
	if len(args) == 0 {
		return nil
	}

	out := make([]string, len(args))
	expectSecretValue := false
	for i, arg := range args {
		switch {
		case isFlag(arg):
			expectSecretValue = false
			name, value, inline := splitInlineFlag(arg)
			if inline {
				if isSensitiveFlagName(name) || looksLikeSecret(value) {
					out[i] = name + "=" + Placeholder
				} else {
					out[i] = arg
				}
				continue
			}
			out[i] = arg
			if isSensitiveFlagName(arg) {
				expectSecretValue = true
			}
		case expectSecretValue:
			out[i] = Placeholder
			expectSecretValue = false
		case looksLikeSecret(arg):
			out[i] = Placeholder
		default:
			out[i] = arg
		}
	}
	return out
}

// isFlag reports whether arg looks like a command-line flag. A lone "-" (stdin
// marker) and "--" (argument separator) are not treated as flags.
func isFlag(arg string) bool {
	return len(arg) > 1 && arg[0] == '-' && arg != "--"
}

// splitInlineFlag splits "--name=value" into its parts. When no "=" is present
// the whole argument is returned as the name with inline=false.
func splitInlineFlag(arg string) (name string, value string, inline bool) {
	if idx := strings.IndexByte(arg, '='); idx >= 0 {
		return arg[:idx], arg[idx+1:], true
	}
	return arg, "", false
}

// isSensitiveFlagName reports whether a flag name signals that its value is a
// credential. Leading dashes are stripped and matching is case-insensitive.
func isSensitiveFlagName(flag string) bool {
	name := strings.ToLower(strings.TrimLeft(flag, "-"))
	if name == "" {
		return false
	}
	for _, part := range sensitiveNameParts {
		if strings.Contains(name, part) {
			return true
		}
	}
	return false
}

// looksLikeSecret reports whether a bare value resembles a credential, used for
// arguments that are not introduced by a sensitive flag name.
func looksLikeSecret(value string) bool {
	if value == "" {
		return false
	}
	for _, prefix := range knownSecretPrefixes {
		if len(value) >= minPrefixedSecretLen && strings.HasPrefix(value, prefix) {
			return true
		}
	}
	if len(value) < minStandaloneSecretLen {
		return false
	}
	if strings.ContainsAny(value, " \t\n\r") {
		return false
	}
	if !hasMixedClasses(value) {
		return false
	}
	return shannonEntropy(value) >= entropyThreshold
}

// hasMixedClasses reports whether a value mixes at least two character classes
// (lowercase, uppercase, digits, other), a common trait of encoded secrets and
// a cheap way to exclude plain words and simple paths.
func hasMixedClasses(value string) bool {
	var lower, upper, digit, other bool
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			lower = true
		case r >= 'A' && r <= 'Z':
			upper = true
		case r >= '0' && r <= '9':
			digit = true
		default:
			other = true
		}
	}
	classes := 0
	for _, present := range []bool{lower, upper, digit, other} {
		if present {
			classes++
		}
	}
	return classes >= 2
}

// shannonEntropy returns the Shannon entropy of a value in bits per character.
func shannonEntropy(value string) float64 {
	if value == "" {
		return 0
	}
	counts := make(map[rune]int)
	total := 0
	for _, r := range value {
		counts[r]++
		total++
	}
	if total == 0 {
		return 0
	}
	var entropy float64
	for _, count := range counts {
		p := float64(count) / float64(total)
		entropy -= p * math.Log2(p)
	}
	return entropy
}
