package cli

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"reflect"
	"strings"
	"testing"

	"github.com/karoc/adp/internal/adapters"
	"github.com/karoc/adp/internal/events"
	"github.com/karoc/adp/internal/paths"
	"github.com/karoc/adp/internal/runner"
	"github.com/karoc/adp/internal/runtime"
	"github.com/karoc/adp/internal/schema"
	"github.com/karoc/adp/internal/shell"
)

func TestExecuteShowsHelp(t *testing.T) {
	var stdout bytes.Buffer

	code := NewApp(Dependencies{}, &stdout, &bytes.Buffer{}).Execute(context.Background(), nil)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "adp run <agent>") {
		t.Fatalf("help output missing run usage: %q", stdout.String())
	}
}

func TestInitCommandCallsWorkspaceStore(t *testing.T) {
	store := &fakeStore{}
	var stdout bytes.Buffer

	code := NewApp(Dependencies{WorkspaceStore: store}, &stdout, &bytes.Buffer{}).Execute(context.Background(), []string{"init"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !store.initCalled {
		t.Fatal("Init was not called")
	}
}

func TestWorkspaceAddCommandCallsStore(t *testing.T) {
	store := &fakeStore{}

	code := NewApp(Dependencies{WorkspaceStore: store}, &bytes.Buffer{}, &bytes.Buffer{}).Execute(
		context.Background(),
		[]string{"workspace", "add", "game-a", "/srv/game-a"},
	)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if store.addName != "game-a" || store.addRoot != "/srv/game-a" {
		t.Fatalf("Add called with (%q, %q)", store.addName, store.addRoot)
	}
}

func TestRunCommandWiresAdapterRuntimeAndRunner(t *testing.T) {
	store := &fakeStore{cfg: testConfig()}
	registry := adapters.NewRegistry()
	adapter := &fakeAdapter{name: "codex"}
	if err := registry.Register(adapter); err != nil {
		t.Fatal(err)
	}
	var buildReq runtime.BuildRequest
	var launchSpec adapters.LaunchSpec
	var logged []events.Event
	var cleaned runtime.Handle

	deps := Dependencies{
		Layout:         paths.New("/tmp/adp-home", "/tmp/adp-runtime"),
		WorkspaceStore: store,
		Adapters:       registry,
		BuildRuntime: func(_ context.Context, req runtime.BuildRequest) (*runtime.Handle, error) {
			buildReq = req
			env := map[string]string{"ADP_RUNTIME_ROOT": "/tmp/runtime"}
			for key, value := range req.Env {
				env[key] = value
			}
			return &runtime.Handle{
				SessionID:     "session-1",
				WorkspaceName: "game-a",
				ProjectRoot:   "/srv/game-a",
				Root:          "/tmp/runtime",
				Env:           env,
			}, nil
		},
		CleanupRuntime: func(_ context.Context, handle runtime.Handle) error {
			cleaned = handle
			return nil
		},
		RunProcess: func(_ context.Context, spec adapters.LaunchSpec, _ runner.Streams) (*runner.Result, error) {
			launchSpec = spec
			return &runner.Result{ExitCode: 7}, nil
		},
		EventLogger: eventLoggerFunc(func(_ context.Context, event events.Event) error {
			logged = append(logged, event)
			return nil
		}),
	}

	code := NewApp(deps, &bytes.Buffer{}, &bytes.Buffer{}).Execute(
		context.Background(),
		[]string{"run", "codex", "--workspace", "game-a", "--profile", "senior", "--keep-runtime", "--", "--version"},
	)

	if code != 7 {
		t.Fatalf("exit code = %d, want agent exit code 7", code)
	}
	if buildReq.WorkspaceDir != "/tmp/adp-home/workspaces/game-a" || !buildReq.Keep {
		t.Fatalf("unexpected runtime request: %+v", buildReq)
	}
	if len(buildReq.Files) != 1 || buildReq.Files[0].Path != "AGENTS.md" {
		t.Fatalf("unexpected generated files: %+v", buildReq.Files)
	}
	if buildReq.Env["ADAPTER_ENV"] != "1" {
		t.Fatalf("adapter env was not passed to runtime build: %#v", buildReq.Env)
	}
	if !reflect.DeepEqual(launchSpec.Args, []string{"--version"}) {
		t.Fatalf("launch args = %#v", launchSpec.Args)
	}
	if launchSpec.Env["ADAPTER_ENV"] != "1" || launchSpec.Env["ADP_RUNTIME_ROOT"] != "/tmp/runtime" {
		t.Fatalf("launch env = %#v", launchSpec.Env)
	}
	if len(logged) != 2 || logged[0].Type != "run_started" || logged[1].Type != "run_finished" {
		t.Fatalf("unexpected events: %+v", logged)
	}
	if cleaned.Root != "/tmp/runtime" {
		t.Fatalf("runtime was not cleaned: %+v", cleaned)
	}
}

func TestEnterCommandWiresRuntimeAndShell(t *testing.T) {
	store := &fakeStore{cfg: testConfig()}
	var entered adapters.RuntimeHandle
	var cleaned runtime.Handle

	deps := Dependencies{
		WorkspaceStore: store,
		BuildRuntime: func(_ context.Context, _ runtime.BuildRequest) (*runtime.Handle, error) {
			return &runtime.Handle{Root: "/tmp/runtime", Env: map[string]string{}}, nil
		},
		CleanupRuntime: func(_ context.Context, handle runtime.Handle) error {
			cleaned = handle
			return nil
		},
		EnterShell: func(_ context.Context, handle adapters.RuntimeHandle, _ shell.Streams) error {
			entered = handle
			return nil
		},
	}

	code := NewApp(deps, &bytes.Buffer{}, &bytes.Buffer{}).Execute(context.Background(), []string{"enter", "game-a"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if entered.Root != "/tmp/runtime" {
		t.Fatalf("entered runtime = %q", entered.Root)
	}
	if cleaned.Root != "/tmp/runtime" {
		t.Fatalf("runtime was not cleaned: %+v", cleaned)
	}
}

type fakeStore struct {
	initCalled bool
	addName    string
	addRoot    string
	cfg        schema.Config
}

func (s *fakeStore) Init(context.Context) error {
	s.initCalled = true
	return nil
}

func (s *fakeStore) Add(_ context.Context, name string, root string) (*schema.Config, error) {
	s.addName = name
	s.addRoot = root
	cfg := testConfig()
	return &cfg, nil
}

func (s *fakeStore) Get(_ context.Context, name string) (*schema.Config, string, error) {
	if name != "game-a" {
		return nil, "", errors.New("workspace not found")
	}
	cfg := s.cfg
	if cfg.Version == 0 {
		cfg = testConfig()
	}
	return &cfg, "/tmp/adp-home/workspaces/game-a", nil
}

type fakeAdapter struct {
	name string
}

func (a *fakeAdapter) Name() string {
	return a.name
}

func (a *fakeAdapter) Validate(context.Context, adapters.Context) error {
	return nil
}

func (a *fakeAdapter) Render(context.Context, adapters.Context) (*adapters.RenderResult, error) {
	return &adapters.RenderResult{
		Files: []adapters.GeneratedFile{{Path: "AGENTS.md", Mode: fs.FileMode(0o644), Data: []byte("prompt")}},
		Env:   map[string]string{"ADAPTER_ENV": "1"},
	}, nil
}

func (a *fakeAdapter) Launch(_ context.Context, _ adapters.Context, runtime adapters.RuntimeHandle, args []string) (*adapters.LaunchSpec, error) {
	return &adapters.LaunchSpec{
		Command: "fake-codex",
		Args:    args,
		Dir:     runtime.Root,
		Env:     map[string]string{"LAUNCH_ENV": "1"},
	}, nil
}

type eventLoggerFunc func(context.Context, events.Event) error

func (f eventLoggerFunc) Log(ctx context.Context, event events.Event) error {
	return f(ctx, event)
}

func testConfig() schema.Config {
	return schema.Config{
		Version:   schema.CurrentVersion,
		Workspace: schema.Workspace{Name: "game-a"},
		Project:   schema.Project{Root: "/srv/game-a"},
		Agents: map[string]schema.AgentConfig{
			"codex": {Enabled: true, Profile: "default", Command: "codex"},
		},
	}
}
