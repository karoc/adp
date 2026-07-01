package cli

import (
	"strings"
	"testing"
	"time"
)

// TestLooksLikeFlag locks in the safety pivot of the flag-eats-flag fix: a
// leading '-' followed by a digit (negative numbers/durations) or a bare "-"
// must be treated as a value so the dedicated validators still surface their
// existing messages, while genuine flag tokens are rejected.
func TestLooksLikeFlag(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"", false},
		{"-", false},
		{"codex", false},
		{"game-a", false},
		{"-1", false},
		{"-5m", false},
		{"-0", false},
		{"-9223372036854775808", false},
		{"--workspace", true},
		{"--profile", true},
		{"-w", true},
		{"-p", true},
		{"--", true},
		{"-abc", true},
		{"-x", true},
	}
	for _, tt := range tests {
		if got := looksLikeFlag(tt.in); got != tt.want {
			t.Errorf("looksLikeFlag(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

// TestRequireValue verifies the central value-consuming helper rejects a
// missing value and a following flag token, while accepting ordinary values
// and dash-prefixed numeric values.
func TestRequireValue(t *testing.T) {
	t.Run("missing value at end", func(t *testing.T) {
		_, _, err := requireValue([]string{"--workspace"}, 0, "--workspace")
		if err == nil {
			t.Fatal("expected error for missing value, got nil")
		}
		if want := "--workspace requires a value"; err.Error() != want {
			t.Errorf("error = %q, want %q", err.Error(), want)
		}
	})
	t.Run("following flag rejected", func(t *testing.T) {
		_, idx, err := requireValue([]string{"--workspace", "--profile", "x"}, 0, "--workspace")
		if err == nil {
			t.Fatal("expected error when value looks like a flag, got nil")
		}
		if want := "--workspace requires a value"; err.Error() != want {
			t.Errorf("error = %q, want %q", err.Error(), want)
		}
		if idx != 0 {
			t.Errorf("index advanced to %d on error, want 0", idx)
		}
	})
	t.Run("ordinary value accepted", func(t *testing.T) {
		val, idx, err := requireValue([]string{"--workspace", "game-a"}, 0, "--workspace")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != "game-a" {
			t.Errorf("value = %q, want %q", val, "game-a")
		}
		if idx != 1 {
			t.Errorf("index = %d, want 1", idx)
		}
	})
	t.Run("dash-numeric value accepted", func(t *testing.T) {
		val, _, err := requireValue([]string{"--limit", "-1"}, 0, "--limit")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != "-1" {
			t.Errorf("value = %q, want %q", val, "-1")
		}
	})
}

func TestParseRunArgs(t *testing.T) {
	t.Run("agent only", func(t *testing.T) {
		opts, err := parseRunArgs([]string{"codex"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.agent != "codex" {
			t.Errorf("agent = %q, want %q", opts.agent, "codex")
		}
	})
	t.Run("empty requires agent", func(t *testing.T) {
		_, err := parseRunArgs(nil)
		if err == nil || err.Error() != runAgentRequiredMsg {
			t.Fatalf("error = %v, want %q", err, runAgentRequiredMsg)
		}
	})
	t.Run("leading flag requires agent", func(t *testing.T) {
		_, err := parseRunArgs([]string{"--workspace", "game-a", "codex"})
		if err == nil || err.Error() != runAgentRequiredMsg {
			t.Fatalf("error = %v, want %q", err, runAgentRequiredMsg)
		}
	})
	t.Run("full flags", func(t *testing.T) {
		opts, err := parseRunArgs([]string{
			"codex", "--workspace", "game-a", "--profile", "senior",
			"--take", "--owner", "worker", "--lease", "30m", "--keep-runtime",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.workspace != "game-a" || opts.profile != "senior" {
			t.Errorf("workspace/profile = %q/%q", opts.workspace, opts.profile)
		}
		if !opts.take || opts.owner != "worker" || opts.lease != 30*time.Minute || !opts.keep {
			t.Errorf("take=%v owner=%q lease=%v keep=%v", opts.take, opts.owner, opts.lease, opts.keep)
		}
	})
	t.Run("agent args after --", func(t *testing.T) {
		opts, err := parseRunArgs([]string{"codex", "--workspace", "game-a", "--", "--version", "-x"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []string{"--version", "-x"}
		if strings.Join(opts.agentArgs, ",") != strings.Join(want, ",") {
			t.Errorf("agentArgs = %v, want %v", opts.agentArgs, want)
		}
	})
	t.Run("flag-eats-flag rejected", func(t *testing.T) {
		_, err := parseRunArgs([]string{"codex", "--workspace", "--profile", "senior"})
		if err == nil || err.Error() != "--workspace requires a value" {
			t.Fatalf("error = %v, want %q", err, "--workspace requires a value")
		}
	})
	t.Run("negative lease message preserved", func(t *testing.T) {
		_, err := parseRunArgs([]string{"codex", "--take", "--owner", "w", "--lease", "-5m"})
		if err == nil || err.Error() != "lease must not be negative" {
			t.Fatalf("error = %v, want %q", err, "lease must not be negative")
		}
	})
	t.Run("take without owner", func(t *testing.T) {
		_, err := parseRunArgs([]string{"codex", "--take"})
		if err == nil || err.Error() != "--owner is required with --take" {
			t.Fatalf("error = %v, want %q", err, "--owner is required with --take")
		}
	})
	t.Run("take combined with task", func(t *testing.T) {
		_, err := parseRunArgs([]string{"codex", "--take", "--owner", "w", "--task", "task-1"})
		if err == nil || err.Error() != "--take cannot be combined with --task" {
			t.Fatalf("error = %v, want %q", err, "--take cannot be combined with --task")
		}
	})
	t.Run("owner without take", func(t *testing.T) {
		_, err := parseRunArgs([]string{"codex", "--owner", "w"})
		if err == nil || err.Error() != "--owner requires --take" {
			t.Fatalf("error = %v, want %q", err, "--owner requires --take")
		}
	})
	t.Run("lease without take", func(t *testing.T) {
		_, err := parseRunArgs([]string{"codex", "--lease", "30m"})
		if err == nil || err.Error() != "--lease requires --take" {
			t.Fatalf("error = %v, want %q", err, "--lease requires --take")
		}
	})
	t.Run("unknown option", func(t *testing.T) {
		_, err := parseRunArgs([]string{"codex", "--bogus"})
		if err == nil || err.Error() != `unknown run option "--bogus"` {
			t.Fatalf("error = %v, want %q", err, `unknown run option "--bogus"`)
		}
	})
}

func TestParseCompletionValuesArgs(t *testing.T) {
	t.Run("kind only", func(t *testing.T) {
		opts, err := parseCompletionValuesArgs([]string{"agents"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.kind != "agents" {
			t.Errorf("kind = %q, want %q", opts.kind, "agents")
		}
	})
	t.Run("kind with workspace", func(t *testing.T) {
		opts, err := parseCompletionValuesArgs([]string{"tasks", "--workspace", "game-a"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.kind != "tasks" || opts.workspace != "game-a" {
			t.Errorf("kind/workspace = %q/%q", opts.kind, opts.workspace)
		}
	})
	t.Run("empty shows usage", func(t *testing.T) {
		_, err := parseCompletionValuesArgs(nil)
		if err == nil || !strings.HasPrefix(err.Error(), "usage: adp completion values") {
			t.Fatalf("error = %v, want usage message", err)
		}
	})
	t.Run("leading flag shows usage not misleading option error", func(t *testing.T) {
		_, err := parseCompletionValuesArgs([]string{"--workspace", "game-a"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.HasPrefix(err.Error(), "usage: adp completion values") {
			t.Errorf("error = %q, want usage message", err.Error())
		}
		if strings.Contains(err.Error(), "unknown completion values option") {
			t.Errorf("error = %q, should not be the misleading option error", err.Error())
		}
	})
	t.Run("workspace eats flag rejected", func(t *testing.T) {
		_, err := parseCompletionValuesArgs([]string{"tasks", "--workspace", "-w"})
		if err == nil || err.Error() != "--workspace requires a value" {
			t.Fatalf("error = %v, want %q", err, "--workspace requires a value")
		}
	})
	t.Run("unknown option", func(t *testing.T) {
		_, err := parseCompletionValuesArgs([]string{"tasks", "--bogus"})
		if err == nil || err.Error() != `unknown completion values option "--bogus"` {
			t.Fatalf("error = %v, want %q", err, `unknown completion values option "--bogus"`)
		}
	})
}

func TestParseShellHookArgs(t *testing.T) {
	t.Run("shell and name", func(t *testing.T) {
		opts, err := parseShellHookArgs([]string{"--shell", "bash", "--name", "adp_hook"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.Shell != "bash" || opts.FunctionName != "adp_hook" {
			t.Errorf("shell/name = %q/%q", opts.Shell, opts.FunctionName)
		}
	})
	t.Run("shell eats flag rejected", func(t *testing.T) {
		_, err := parseShellHookArgs([]string{"--shell", "--name", "x"})
		if err == nil || err.Error() != "--shell requires a value" {
			t.Fatalf("error = %v, want %q", err, "--shell requires a value")
		}
	})
}

func TestParseCompletionArgs(t *testing.T) {
	t.Run("shell and command", func(t *testing.T) {
		opts, err := parseCompletionArgs([]string{"--shell", "zsh", "--command", "adp"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.Shell != "zsh" || opts.CommandName != "adp" {
			t.Errorf("shell/command = %q/%q", opts.Shell, opts.CommandName)
		}
	})
	t.Run("command eats flag rejected", func(t *testing.T) {
		_, err := parseCompletionArgs([]string{"--shell", "zsh", "--command", "--shell"})
		if err == nil || err.Error() != "--command requires a value" {
			t.Fatalf("error = %v, want %q", err, "--command requires a value")
		}
	})
}

func TestParseEventsListArgs(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		opts, err := parseEventsListArgs(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.limit != defaultEventLimit || opts.format != outputFormatText {
			t.Errorf("limit/format = %d/%v", opts.limit, opts.format)
		}
	})
	t.Run("negative limit message preserved", func(t *testing.T) {
		_, err := parseEventsListArgs([]string{"--limit", "-1"})
		if err == nil || err.Error() != "limit must not be negative" {
			t.Fatalf("error = %v, want %q", err, "limit must not be negative")
		}
	})
	t.Run("non-integer limit", func(t *testing.T) {
		_, err := parseEventsListArgs([]string{"--limit", "many"})
		if err == nil || !strings.HasPrefix(err.Error(), "parse limit:") {
			t.Fatalf("error = %v, want prefix %q", err, "parse limit:")
		}
	})
	t.Run("workspace eats flag rejected", func(t *testing.T) {
		_, err := parseEventsListArgs([]string{"--workspace", "--session"})
		if err == nil || err.Error() != "--workspace requires a value" {
			t.Fatalf("error = %v, want %q", err, "--workspace requires a value")
		}
	})
}

func TestParseSessionsListArgs(t *testing.T) {
	t.Run("negative limit message preserved", func(t *testing.T) {
		_, err := parseSessionsListArgs([]string{"--limit", "-1"})
		if err == nil || err.Error() != "limit must not be negative" {
			t.Fatalf("error = %v, want %q", err, "limit must not be negative")
		}
	})
	t.Run("agent eats flag rejected", func(t *testing.T) {
		_, err := parseSessionsListArgs([]string{"--agent", "--workspace"})
		if err == nil || err.Error() != "--agent requires a value" {
			t.Fatalf("error = %v, want %q", err, "--agent requires a value")
		}
	})
}

func TestParseRuntimePruneArgs(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		opts, err := parseRuntimePruneArgs(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.olderThan != 24*time.Hour {
			t.Errorf("olderThan = %v, want 24h", opts.olderThan)
		}
	})
	t.Run("negative older-than message preserved", func(t *testing.T) {
		_, err := parseRuntimePruneArgs([]string{"--older-than", "-5m"})
		if err == nil || err.Error() != "older-than must not be negative" {
			t.Fatalf("error = %v, want %q", err, "older-than must not be negative")
		}
	})
}
