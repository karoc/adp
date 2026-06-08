package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/karoc/adp/internal/adapters"
	"github.com/karoc/adp/internal/schema"
	"github.com/karoc/adp/internal/workspace"
)

func TestCompletionValuesWorkspacesPrintsSortedNames(t *testing.T) {
	store := &fakeStore{records: []workspace.Record{
		{Name: "z-game"},
		{Name: "a-game"},
	}}
	var stdout bytes.Buffer

	code := NewApp(Dependencies{WorkspaceStore: store}, &stdout, &bytes.Buffer{}).Execute(
		context.Background(),
		[]string{"completion", "values", "workspaces"},
	)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if got, want := stdout.String(), "a-game\nz-game\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestCompletionValuesProfilesPrintsWorkspaceProfiles(t *testing.T) {
	workspaceDir := t.TempDir()
	profilesDir := filepath.Join(workspaceDir, "profiles")
	if err := os.Mkdir(profilesDir, 0o755); err != nil {
		t.Fatalf("create profiles dir: %v", err)
	}
	for _, name := range []string{"codex.yaml", "claude.md"} {
		if err := os.WriteFile(filepath.Join(profilesDir, name), []byte("profile\n"), 0o644); err != nil {
			t.Fatalf("write profile file %s: %v", name, err)
		}
	}

	cfg := testConfig()
	cfg.Agents = map[string]schema.AgentConfig{
		"codex":  {Profile: "senior"},
		"claude": {Profile: "architect"},
	}
	store := &fakeStore{cfg: cfg, workspaceDir: workspaceDir}
	var stdout bytes.Buffer

	code := NewApp(Dependencies{WorkspaceStore: store}, &stdout, &bytes.Buffer{}).Execute(
		context.Background(),
		[]string{"completion", "values", "profiles", "--workspace", "game-a"},
	)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	want := "architect\nclaude\ncodex\ndefault\nsenior\n"
	if got := stdout.String(); got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestCompletionValuesProfilesUsesWorkspaceEnvFallback(t *testing.T) {
	t.Setenv("ADP_WORKSPACE", "game-a")

	store := &fakeStore{cfg: testConfig(), workspaceDir: t.TempDir()}
	var stdout bytes.Buffer

	code := NewApp(Dependencies{WorkspaceStore: store}, &stdout, &bytes.Buffer{}).Execute(
		context.Background(),
		[]string{"completion", "values", "profiles"},
	)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if got, want := stdout.String(), "default\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestCompletionValuesAgentsUsesAdapterRegistry(t *testing.T) {
	registry := adapters.NewRegistry()
	for _, name := range []string{"future-agent", "codex"} {
		if err := registry.Register(&fakeAdapter{name: name}); err != nil {
			t.Fatalf("Register(%q) error = %v", name, err)
		}
	}
	var stdout bytes.Buffer

	code := NewApp(Dependencies{Adapters: registry}, &stdout, &bytes.Buffer{}).Execute(
		context.Background(),
		[]string{"completion", "values", "agents"},
	)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if got, want := stdout.String(), "codex\nfuture-agent\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}
