package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/karoc/adp/internal/adapters"
	"github.com/karoc/adp/internal/paths"
	"github.com/karoc/adp/internal/schema"
	"github.com/karoc/adp/internal/sessions"
	taskstore "github.com/karoc/adp/internal/tasks"
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

func TestCompletionValuesPrintsPlanningCandidates(t *testing.T) {
	store := &fakeTaskStore{
		tasks: []taskstore.Task{
			{ID: "task-z", Owner: "marin"},
			{ID: "task-a", Owner: "karoc"},
			{ID: "task-b", Owner: "karoc"},
			{ID: "task-c"},
		},
		phases: []taskstore.Phase{
			{ID: "P20"},
			{ID: "P10"},
		},
	}
	deps := Dependencies{
		WorkspaceStore:   &fakeStore{cfg: testConfig(), workspaceDir: t.TempDir()},
		TaskStoreFactory: func(string) TaskStore { return store },
	}

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "tasks", args: []string{"completion", "values", "tasks", "--workspace", "game-a"}, want: "task-a\ntask-b\ntask-c\ntask-z\n"},
		{name: "phases", args: []string{"completion", "values", "phases", "--workspace", "game-a"}, want: "P10\nP20\n"},
		{name: "owners", args: []string{"completion", "values", "owners", "--workspace", "game-a"}, want: "karoc\nmarin\n"},
		{name: "statuses", args: []string{"completion", "values", "statuses"}, want: "planned\nready\nin_progress\nblocked\nreview\nvalidated\ndone\ncanceled\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer

			code := NewApp(deps, &stdout, &bytes.Buffer{}).Execute(context.Background(), tt.args)

			if code != 0 {
				t.Fatalf("exit code = %d, want 0", code)
			}
			if got := stdout.String(); got != tt.want {
				t.Fatalf("stdout = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCompletionValuesSessionsUsesSessionLister(t *testing.T) {
	var gotQuery sessions.Query
	var stdout bytes.Buffer
	deps := Dependencies{
		Layout: paths.New("/tmp/adp-home", "/tmp/adp-runtime"),
		ListSessions: func(_ context.Context, _ paths.Layout, query sessions.Query) ([]sessions.Summary, error) {
			gotQuery = query
			return []sessions.Summary{
				{SessionID: "session-z"},
				{SessionID: "session-a"},
				{SessionID: "session-z"},
				{SessionID: ""},
			}, nil
		},
	}

	code := NewApp(deps, &stdout, &bytes.Buffer{}).Execute(
		context.Background(),
		[]string{"completion", "values", "sessions", "--workspace", "game-a"},
	)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if gotQuery.Workspace != "game-a" {
		t.Fatalf("workspace query = %q, want game-a", gotQuery.Workspace)
	}
	if got, want := stdout.String(), "session-a\nsession-z\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}
