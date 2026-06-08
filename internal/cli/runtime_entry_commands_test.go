package cli

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/karoc/adp/internal/adapters"
	"github.com/karoc/adp/internal/events"
	"github.com/karoc/adp/internal/paths"
	"github.com/karoc/adp/internal/runner"
	"github.com/karoc/adp/internal/runtime"
	"github.com/karoc/adp/internal/shell"
)

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

func TestRunCommandCanResolveWorkspaceFromCurrentDirectory(t *testing.T) {
	store := &fakeStore{cfg: testConfig(), findByProjectPath: true}
	registry := adapters.NewRegistry()
	adapter := &fakeAdapter{name: "codex"}
	if err := registry.Register(adapter); err != nil {
		t.Fatal(err)
	}
	deps := Dependencies{
		WorkspaceStore: store,
		Adapters:       registry,
		BuildRuntime: func(_ context.Context, _ runtime.BuildRequest) (*runtime.Handle, error) {
			return &runtime.Handle{Root: "/tmp/runtime", Env: map[string]string{}}, nil
		},
		CleanupRuntime: func(context.Context, runtime.Handle) error { return nil },
		RunProcess: func(context.Context, adapters.LaunchSpec, runner.Streams) (*runner.Result, error) {
			return &runner.Result{ExitCode: 0}, nil
		},
	}

	code := NewApp(deps, &bytes.Buffer{}, &bytes.Buffer{}).Execute(context.Background(), []string{"run", "codex"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !store.findCalled {
		t.Fatal("FindByProjectPath was not called")
	}
}

func TestEnvCommandBuildsKeptRuntimeAndPrintsExports(t *testing.T) {
	store := &fakeStore{cfg: testConfig()}
	var buildReq runtime.BuildRequest
	var stdout bytes.Buffer

	deps := Dependencies{
		Layout:         paths.New("/tmp/adp-home", "/tmp/adp-runtime"),
		WorkspaceStore: store,
		BuildRuntime: func(_ context.Context, req runtime.BuildRequest) (*runtime.Handle, error) {
			buildReq = req
			return &runtime.Handle{
				Root: "/tmp/runtime root",
				Env: map[string]string{
					"ADP_WORKSPACE":     "game-a",
					"ADP_RUNTIME_ROOT":  "/tmp/runtime root",
					"ADP_PROJECT_ROOT":  "/srv/game-a",
					"ADP_SESSION_ID":    "session-1",
					"NON_ADP_SHOULD_GO": "no",
				},
				Keep: true,
			}, nil
		},
		CleanupRuntime: func(context.Context, runtime.Handle) error {
			t.Fatal("env command should keep runtime on success")
			return nil
		},
	}

	code := NewApp(deps, &stdout, &bytes.Buffer{}).Execute(context.Background(), []string{"env", "game-a", "--cd"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !buildReq.Keep {
		t.Fatal("env command should build keep-runtime handle")
	}
	output := stdout.String()
	if !strings.Contains(output, "export ADP_WORKSPACE='game-a'") {
		t.Fatalf("env output missing ADP export: %q", output)
	}
	if !strings.Contains(output, "cd '/tmp/runtime root'") {
		t.Fatalf("env output missing cd: %q", output)
	}
	if strings.Contains(output, "NON_ADP") {
		t.Fatalf("env output leaked non ADP variable: %q", output)
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
