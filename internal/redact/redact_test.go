package redact

import (
	"reflect"
	"testing"
)

func TestArgsRedactsSensitiveFlagValues(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "separate flag and value",
			in:   []string{"--api-key", "sk-abc123secretvalue", "--model", "gpt-4"},
			want: []string{"--api-key", Placeholder, "--model", "gpt-4"},
		},
		{
			name: "inline flag value",
			in:   []string{"--api-key=sk-abc123secretvalue"},
			want: []string{"--api-key=" + Placeholder},
		},
		{
			name: "token flag",
			in:   []string{"--auth-token", "plainbutsensitive"},
			want: []string{"--auth-token", Placeholder},
		},
		{
			name: "password flag short single dash",
			in:   []string{"-password", "hunter2"},
			want: []string{"-password", Placeholder},
		},
		{
			name: "credential flag",
			in:   []string{"--credential", "whatever-value-here"},
			want: []string{"--credential", Placeholder},
		},
		{
			name: "case insensitive flag name",
			in:   []string{"--API-KEY", "sk-abc123secretvalue"},
			want: []string{"--API-KEY", Placeholder},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Args(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Args(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestArgsRedactsBareSecretsByPrefix(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "openai key prefix",
			in:   []string{"sk-abcdefghij1234567890"},
			want: []string{Placeholder},
		},
		{
			name: "github pat prefix",
			in:   []string{"ghp_abcdefghij1234567890"},
			want: []string{Placeholder},
		},
		{
			name: "aws access key id",
			in:   []string{"AKIAIOSFODNN7EXAMPLE"},
			want: []string{Placeholder},
		},
		{
			name: "jwt prefix",
			in:   []string{"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"},
			want: []string{Placeholder},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Args(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Args(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestArgsRedactsBareHighEntropySecret(t *testing.T) {
	// A long, mixed-class, high-entropy bare token with no recognizable prefix.
	secret := "Xk7Qp2Lm9Rs4Tv6Wy8Zb1Nc3Df5Gh"
	got := Args([]string{secret})
	if len(got) != 1 || got[0] != Placeholder {
		t.Fatalf("Args(bare high-entropy) = %v, want [%s]", got, Placeholder)
	}
}

func TestArgsPreservesNonSecrets(t *testing.T) {
	cases := []struct {
		name string
		in   []string
	}{
		{name: "model name", in: []string{"--model", "gpt-4-turbo"}},
		{name: "plain word value", in: []string{"--name", "production"}},
		{name: "file path", in: []string{"--config", "/etc/adp/config.yaml"}},
		{name: "boolean flag", in: []string{"--verbose"}},
		{name: "double dash separator", in: []string{"--"}},
		{name: "stdin dash", in: []string{"-"}},
		{name: "short numeric", in: []string{"--retries", "5"}},
		{name: "lowercase word long", in: []string{"--message", "thequickbrownfoxjumpsover"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Args(tc.in)
			if !reflect.DeepEqual(got, tc.in) {
				t.Fatalf("Args(%v) = %v, want unchanged", tc.in, got)
			}
		})
	}
}

func TestArgsDoesNotMutateInput(t *testing.T) {
	in := []string{"--api-key", "sk-abc123secretvalue"}
	original := append([]string(nil), in...)
	_ = Args(in)
	if !reflect.DeepEqual(in, original) {
		t.Fatalf("Args mutated input: got %v, want %v", in, original)
	}
}

func TestArgsEmpty(t *testing.T) {
	if got := Args(nil); got != nil {
		t.Fatalf("Args(nil) = %v, want nil", got)
	}
	if got := Args([]string{}); got != nil {
		t.Fatalf("Args(empty) = %v, want nil", got)
	}
}

func TestArgsValueAfterSensitiveFlagAlwaysRedacted(t *testing.T) {
	// Even a short, low-entropy value is redacted when introduced by a
	// sensitive flag, because the flag name is the authoritative signal.
	got := Args([]string{"--token", "ab"})
	want := []string{"--token", Placeholder}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Args = %v, want %v", got, want)
	}
}

func TestArgsSensitiveFlagFollowedByFlagDoesNotOverRedact(t *testing.T) {
	// "--api-key" with no value (immediately followed by another flag) must
	// not swallow the next flag as a redacted value.
	got := Args([]string{"--api-key", "--verbose"})
	want := []string{"--api-key", "--verbose"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Args = %v, want %v", got, want)
	}
}

func TestArgsInlineHighEntropyValueWithoutSensitiveName(t *testing.T) {
	// An inline "--flag=value" whose name is not sensitive but whose value is a
	// high-entropy secret is still redacted on the value's own merits.
	got := Args([]string{"--data=Xk7Qp2Lm9Rs4Tv6Wy8Zb1Nc3Df5Gh"})
	want := []string{"--data=" + Placeholder}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Args = %v, want %v", got, want)
	}
}

func TestArgsInlineNonSecretPreserved(t *testing.T) {
	// An inline "--flag=value" with a benign name and benign value is untouched.
	got := Args([]string{"--model=gpt-4-turbo"})
	want := []string{"--model=gpt-4-turbo"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Args = %v, want %v", got, want)
	}
}

func TestArgsEmptyFlagNameNotSensitive(t *testing.T) {
	// A bare "--" is the argument separator and must never be treated as a
	// sensitive flag, even though TrimLeft would reduce it to an empty name.
	if isSensitiveFlagName("--") {
		t.Fatal("isSensitiveFlagName(--) = true, want false")
	}
	if isSensitiveFlagName("") {
		t.Fatal("isSensitiveFlagName(empty) = true, want false")
	}
}

func TestArgsKnownPrefixTooShortNotRedacted(t *testing.T) {
	// A value carrying a known prefix but shorter than minPrefixedSecretLen is
	// not redacted, avoiding false positives on tiny tokens like "sk-".
	got := Args([]string{"sk-short"})
	want := []string{"sk-short"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Args = %v, want %v", got, want)
	}
}

func TestLooksLikeSecretEmptyAndWhitespace(t *testing.T) {
	if looksLikeSecret("") {
		t.Fatal("looksLikeSecret(empty) = true, want false")
	}
	// A long value containing whitespace is treated as prose, not a secret.
	if looksLikeSecret("this is a long sentence with many words in it") {
		t.Fatal("looksLikeSecret(prose) = true, want false")
	}
}

func TestShannonEntropyEmpty(t *testing.T) {
	if got := shannonEntropy(""); got != 0 {
		t.Fatalf("shannonEntropy(empty) = %v, want 0", got)
	}
}
