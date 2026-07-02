package shared

import (
	"strings"
	"testing"

	"github.com/karoc/adp/internal/adapters/api"
	"github.com/karoc/adp/internal/schema"
)

func TestMemoryTextBranches(t *testing.T) {
	t.Run("disabled", func(t *testing.T) {
		ctx := api.Context{Config: schema.Config{Memory: schema.Memory{Enabled: false}}}
		if got := memoryText(ctx); got != "Shared memory is disabled for this workspace." {
			t.Fatalf("memoryText disabled = %q", got)
		}
	})

	t.Run("enabled_without_configured_file", func(t *testing.T) {
		ctx := api.Context{
			WorkspaceDir: t.TempDir(),
			Config:       schema.Config{Memory: schema.Memory{Enabled: true, Shared: ""}},
		}
		got := memoryText(ctx)
		if !strings.Contains(got, "no shared memory file is configured") {
			t.Fatalf("memoryText enabled/no-file = %q", got)
		}
	})

	t.Run("enabled_with_content", func(t *testing.T) {
		dir := t.TempDir()
		writeWorkspaceFile(t, dir, "memory.md", "remember the invariants")
		ctx := api.Context{
			WorkspaceDir: dir,
			Config:       schema.Config{Memory: schema.Memory{Enabled: true, Shared: "memory.md"}},
		}
		if got := memoryText(ctx); got != "remember the invariants" {
			t.Fatalf("memoryText enabled/content = %q", got)
		}
	})
}

func TestRulesTextBranches(t *testing.T) {
	t.Run("no_rules", func(t *testing.T) {
		ctx := api.Context{Config: schema.Config{}}
		if got := rulesText(ctx); got != "No workspace rules are configured." {
			t.Fatalf("rulesText empty = %q", got)
		}
	})

	t.Run("rules_are_sorted", func(t *testing.T) {
		ctx := api.Context{Config: schema.Config{Rules: map[string]string{
			"zeta":  "last",
			"alpha": "first",
			"mid":   "middle",
		}}}
		got := rulesText(ctx)
		want := "- alpha: first\n- mid: middle\n- zeta: last\n"
		if got != want {
			t.Fatalf("rulesText sorted =\n%q\nwant\n%q", got, want)
		}
	})
}

func TestMCPTextBranches(t *testing.T) {
	t.Run("disabled", func(t *testing.T) {
		ctx := api.Context{Config: schema.Config{MCP: schema.MCP{Enabled: false}}}
		if got := mcpText(ctx); got != "MCP is disabled for this workspace." {
			t.Fatalf("mcpText disabled = %q", got)
		}
	})

	t.Run("enabled_no_servers_no_config", func(t *testing.T) {
		ctx := api.Context{
			WorkspaceDir: t.TempDir(),
			Config:       schema.Config{MCP: schema.MCP{Enabled: true}},
		}
		got := mcpText(ctx)
		if !strings.Contains(got, "Servers: none configured.") {
			t.Fatalf("mcpText no servers = %q", got)
		}
		if !strings.Contains(got, "No MCP config file is configured.") {
			t.Fatalf("mcpText no config = %q", got)
		}
	})

	t.Run("enabled_servers_sorted_and_config_injected", func(t *testing.T) {
		dir := t.TempDir()
		writeWorkspaceFile(t, dir, "mcp/config.yaml", "servers:\n  fs: {}\n")
		ctx := api.Context{
			WorkspaceDir: dir,
			Config: schema.Config{MCP: schema.MCP{
				Enabled: true,
				Servers: []string{"git", "fs"},
				Config:  "mcp/config.yaml",
			}},
		}
		got := mcpText(ctx)
		// Servers listed and sorted (fs before git).
		fsIdx := strings.Index(got, "- fs")
		gitIdx := strings.Index(got, "- git")
		if fsIdx < 0 || gitIdx < 0 || fsIdx > gitIdx {
			t.Fatalf("mcpText servers not sorted:\n%s", got)
		}
		if !strings.Contains(got, "servers:\n  fs: {}") {
			t.Fatalf("mcpText config not injected:\n%s", got)
		}
	})
}

func TestEffectiveProfileFallback(t *testing.T) {
	cases := []struct {
		name    string
		ctxProf string
		agtProf string
		want    string
	}{
		{"explicit_ctx_profile", "reviewer", "builder", "reviewer"},
		{"agent_profile_fallback", "  ", "builder", "builder"},
		{"default_fallback", "", "", "default"},
		{"trims_whitespace", "  senior  ", "builder", "senior"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := api.Context{
				Profile: tc.ctxProf,
				Agent:   schema.AgentConfig{Profile: tc.agtProf},
			}
			if got := EffectiveProfile(ctx); got != tc.want {
				t.Fatalf("EffectiveProfile = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestProfileTextBranches(t *testing.T) {
	t.Run("options_listed_sorted_and_no_profile_file", func(t *testing.T) {
		ctx := api.Context{
			WorkspaceDir: t.TempDir(),
			Profile:      "default",
			Agent: schema.AgentConfig{
				Enabled: true,
				Command: "codex",
				Options: map[string]string{"model": "opus", "approval": "auto"},
			},
		}
		got := profileText("codex", ctx)
		if !strings.Contains(got, "Name: default") {
			t.Fatalf("profileText missing name:\n%s", got)
		}
		if !strings.Contains(got, "Agent enabled: true") {
			t.Fatalf("profileText missing enabled:\n%s", got)
		}
		// Options sorted: approval before model.
		apIdx := strings.Index(got, "- approval: auto")
		mdIdx := strings.Index(got, "- model: opus")
		if apIdx < 0 || mdIdx < 0 || apIdx > mdIdx {
			t.Fatalf("profileText options not sorted:\n%s", got)
		}
		if !strings.Contains(got, "No profile file was found") {
			t.Fatalf("profileText missing fallback note:\n%s", got)
		}
	})

	t.Run("command_falls_back_to_adapter_name", func(t *testing.T) {
		ctx := api.Context{
			WorkspaceDir: t.TempDir(),
			Agent:        schema.AgentConfig{Command: ""},
		}
		got := profileText("claude", ctx)
		if !strings.Contains(got, "Agent command: claude") {
			t.Fatalf("profileText command fallback:\n%s", got)
		}
	})

	t.Run("profile_specific_file_wins_over_adapter", func(t *testing.T) {
		dir := t.TempDir()
		writeWorkspaceFile(t, dir, "profiles/senior.md", "senior profile body")
		writeWorkspaceFile(t, dir, "profiles/codex.md", "codex adapter body")
		ctx := api.Context{
			WorkspaceDir: dir,
			Profile:      "senior",
			Agent:        schema.AgentConfig{Command: "codex"},
		}
		got := profileText("codex", ctx)
		if !strings.Contains(got, "senior profile body") {
			t.Fatalf("profileText should prefer profile-specific file:\n%s", got)
		}
		if strings.Contains(got, "codex adapter body") {
			t.Fatalf("profileText should not fall through to adapter file:\n%s", got)
		}
	})
}

func TestReadProfileFileResolution(t *testing.T) {
	t.Run("adapter_file_when_profile_default", func(t *testing.T) {
		dir := t.TempDir()
		writeWorkspaceFile(t, dir, "profiles/codex.yaml", "adapter: codex")
		got := readProfileFile(dir, "codex", "default")
		if !strings.Contains(got, "adapter: codex") {
			t.Fatalf("readProfileFile default profile = %q", got)
		}
	})

	t.Run("profile_extension_priority_md_first", func(t *testing.T) {
		dir := t.TempDir()
		writeWorkspaceFile(t, dir, "profiles/senior.yaml", "yaml body")
		writeWorkspaceFile(t, dir, "profiles/senior.md", "md body")
		got := readProfileFile(dir, "codex", "senior")
		if !strings.Contains(got, "md body") {
			t.Fatalf("readProfileFile should prefer .md over .yaml: %q", got)
		}
	})

	t.Run("no_file_returns_summary_note", func(t *testing.T) {
		got := readProfileFile(t.TempDir(), "codex", "senior")
		if !strings.Contains(got, "No profile file was found") {
			t.Fatalf("readProfileFile no file = %q", got)
		}
	})
}

func TestShellQuoteAndDefaultText(t *testing.T) {
	if got := shellQuote("$ADP_WORKSPACE"); got != "$ADP_WORKSPACE" {
		t.Fatalf("shellQuote should pass through env refs: %q", got)
	}
	if got := shellQuote("demo"); got != `"demo"` {
		t.Fatalf("shellQuote should quote literals: %q", got)
	}
	if got := defaultText("  ", "fallback"); got != "fallback" {
		t.Fatalf("defaultText blank = %q", got)
	}
	if got := defaultText("  value  ", "fallback"); got != "value" {
		t.Fatalf("defaultText trims = %q", got)
	}
}
