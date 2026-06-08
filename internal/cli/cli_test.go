package cli

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/karoc/adp/internal/adapters"
	"github.com/karoc/adp/internal/events"
	"github.com/karoc/adp/internal/paths"
	"github.com/karoc/adp/internal/runner"
	"github.com/karoc/adp/internal/runtime"
	"github.com/karoc/adp/internal/schema"
	"github.com/karoc/adp/internal/shell"
	"github.com/karoc/adp/internal/workspace"
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

func TestWorkspaceListCommandPrintsRecords(t *testing.T) {
	store := &fakeStore{records: []workspace.Record{
		{Name: "game-a", ProjectRoot: "/srv/game-a", WorkspaceDir: "/tmp/adp/workspaces/game-a"},
	}}
	var stdout bytes.Buffer

	code := NewApp(Dependencies{WorkspaceStore: store}, &stdout, &bytes.Buffer{}).Execute(context.Background(), []string{"workspace", "list"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	output := stdout.String()
	if !strings.Contains(output, "game-a") || !strings.Contains(output, "/srv/game-a") {
		t.Fatalf("list output missing workspace: %q", output)
	}
}

func TestWorkspaceShowCommandPrintsDetails(t *testing.T) {
	store := &fakeStore{cfg: testConfig()}
	var stdout bytes.Buffer

	code := NewApp(Dependencies{WorkspaceStore: store}, &stdout, &bytes.Buffer{}).Execute(context.Background(), []string{"workspace", "show", "game-a"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	output := stdout.String()
	if !strings.Contains(output, "name: game-a") || !strings.Contains(output, "project_root: /srv/game-a") {
		t.Fatalf("show output missing details: %q", output)
	}
}

func TestWorkspaceRemoveCommandCallsStore(t *testing.T) {
	store := &fakeStore{}
	var stdout bytes.Buffer

	code := NewApp(Dependencies{WorkspaceStore: store}, &stdout, &bytes.Buffer{}).Execute(context.Background(), []string{"workspace", "remove", "game-a"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if store.removeName != "game-a" {
		t.Fatalf("Remove called with %q", store.removeName)
	}
	if !strings.Contains(stdout.String(), "removed") {
		t.Fatalf("remove output = %q", stdout.String())
	}
}

func TestWorkspaceRenameCommandCallsStore(t *testing.T) {
	store := &fakeStore{}
	var stdout bytes.Buffer

	code := NewApp(Dependencies{WorkspaceStore: store}, &stdout, &bytes.Buffer{}).Execute(context.Background(), []string{"workspace", "rename", "old", "new"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if store.renameOld != "old" || store.renameNew != "new" {
		t.Fatalf("Rename called with (%q, %q)", store.renameOld, store.renameNew)
	}
	if !strings.Contains(stdout.String(), `"old" renamed to "new"`) {
		t.Fatalf("rename output = %q", stdout.String())
	}
}

func TestWorkspaceDoctorCommandPrintsNamedReport(t *testing.T) {
	store := &fakeStore{
		diagnoseReport: workspace.DiagnosticReport{
			Workspace:    "game-a",
			WorkspaceDir: "/tmp/adp-home/workspaces/game-a",
		},
	}
	var stdout bytes.Buffer

	code := NewApp(Dependencies{WorkspaceStore: store}, &stdout, &bytes.Buffer{}).Execute(context.Background(), []string{"workspace", "doctor", "game-a"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if store.diagnoseName != "game-a" {
		t.Fatalf("Diagnose called with %q", store.diagnoseName)
	}
	output := stdout.String()
	if !strings.Contains(output, "game-a") || !strings.Contains(output, "ok") || !strings.Contains(output, "no issues") {
		t.Fatalf("doctor output missing healthy report: %q", output)
	}
}

func TestWorkspaceDoctorCommandReturnsTwoWhenDiagnosticsHaveErrors(t *testing.T) {
	store := &fakeStore{
		diagnoseAllReports: []workspace.DiagnosticReport{{
			Workspace: "game-a",
			Diagnostics: []workspace.Diagnostic{{
				Level:   workspace.DiagnosticLevelError,
				Code:    workspace.DiagnosticCodeProjectRootMissing,
				Message: "project root is missing",
				Path:    "/srv/game-a",
			}},
		}},
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := NewApp(Dependencies{WorkspaceStore: store}, &stdout, &stderr).Execute(context.Background(), []string{"workspace", "doctor"})

	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if !store.diagnoseAllCalled {
		t.Fatal("DiagnoseAll was not called")
	}
	output := stdout.String()
	for _, want := range []string{"game-a", "error", workspace.DiagnosticCodeProjectRootMissing, "/srv/game-a"} {
		if !strings.Contains(output, want) {
			t.Fatalf("doctor output missing %q: %q", want, output)
		}
	}
}

func TestWorkspaceDoctorCommandKeepsZeroForWarningDiagnostics(t *testing.T) {
	store := &fakeStore{
		diagnoseReport: workspace.DiagnosticReport{
			Workspace:    "game-a",
			WorkspaceDir: "/tmp/adp-home/workspaces/game-a",
			Diagnostics: []workspace.Diagnostic{{
				Level:   workspace.DiagnosticLevelWarning,
				Code:    workspace.DiagnosticCodeAgentCommandArguments,
				Message: "agent command looks like it contains arguments",
				Path:    "/tmp/adp-home/workspaces/game-a/workspace.yaml",
			}},
		},
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := NewApp(Dependencies{WorkspaceStore: store}, &stdout, &stderr).Execute(context.Background(), []string{"workspace", "doctor", "game-a"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{"game-a", "warning", workspace.DiagnosticCodeAgentCommandArguments, "/tmp/adp-home/workspaces/game-a/workspace.yaml"} {
		if !strings.Contains(output, want) {
			t.Fatalf("workspace doctor output missing %q: %q", want, output)
		}
	}
}

func TestWorkspaceDoctorCommandReturnsTwoWhenNamedRuntimeParentDiagnosticsFail(t *testing.T) {
	store := &fakeStore{
		diagnoseReport: workspace.DiagnosticReport{
			Workspace:    "game-a",
			WorkspaceDir: "/tmp/adp-home/workspaces/game-a",
			Diagnostics: []workspace.Diagnostic{{
				Level:   workspace.DiagnosticLevelError,
				Code:    workspace.DiagnosticCodeRuntimeParentInsideProjectRoot,
				Message: "runtime parent must not be inside the project root",
				Path:    "/srv/game-a/.adp-runtime-parent",
			}},
		},
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := NewApp(Dependencies{WorkspaceStore: store}, &stdout, &stderr).Execute(context.Background(), []string{"workspace", "doctor", "game-a"})

	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if store.diagnoseName != "game-a" {
		t.Fatalf("Diagnose called with %q", store.diagnoseName)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{"game-a", "error", workspace.DiagnosticCodeRuntimeParentInsideProjectRoot, "/srv/game-a/.adp-runtime-parent"} {
		if !strings.Contains(output, want) {
			t.Fatalf("workspace doctor output missing %q: %q", want, output)
		}
	}
}

func TestExecuteReportsUnknownCommand(t *testing.T) {
	var stderr bytes.Buffer

	code := NewApp(Dependencies{}, &bytes.Buffer{}, &stderr).Execute(context.Background(), []string{"bogus"})

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), `adp: unknown command "bogus"`) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestWorkspaceCommandReportsUnknownSubcommand(t *testing.T) {
	var stderr bytes.Buffer

	code := NewApp(Dependencies{WorkspaceStore: &fakeStore{}}, &bytes.Buffer{}, &stderr).Execute(context.Background(), []string{"workspace", "bogus"})

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), `adp: unknown workspace command "bogus"`) {
		t.Fatalf("stderr = %q", stderr.String())
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

type fakeStore struct {
	initCalled         bool
	addName            string
	addRoot            string
	cfg                schema.Config
	workspaceDir       string
	records            []workspace.Record
	findByProjectPath  bool
	findCalled         bool
	removeName         string
	renameOld          string
	renameNew          string
	diagnoseName       string
	diagnoseReport     workspace.DiagnosticReport
	diagnoseAllCalled  bool
	diagnoseAllReports []workspace.DiagnosticReport
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
	workspaceDir := s.workspaceDir
	if workspaceDir == "" {
		workspaceDir = "/tmp/adp-home/workspaces/game-a"
	}
	return &cfg, workspaceDir, nil
}

func (s *fakeStore) List(context.Context) ([]workspace.Record, error) {
	return s.records, nil
}

func (s *fakeStore) Names(context.Context) ([]string, error) {
	names := make([]string, 0, len(s.records))
	for _, record := range s.records {
		names = append(names, record.Name)
	}
	sort.Strings(names)
	return names, nil
}

func (s *fakeStore) FindByProjectPath(_ context.Context, _ string) (*schema.Config, string, error) {
	s.findCalled = true
	if !s.findByProjectPath {
		return nil, "", errors.New("workspace not found")
	}
	cfg := s.cfg
	if cfg.Version == 0 {
		cfg = testConfig()
	}
	workspaceDir := s.workspaceDir
	if workspaceDir == "" {
		workspaceDir = "/tmp/adp-home/workspaces/game-a"
	}
	return &cfg, workspaceDir, nil
}

func (s *fakeStore) Remove(_ context.Context, name string) error {
	s.removeName = name
	return nil
}

func (s *fakeStore) Rename(_ context.Context, oldName string, newName string) (*schema.Config, error) {
	s.renameOld = oldName
	s.renameNew = newName
	cfg := testConfig()
	cfg.Workspace.Name = newName
	return &cfg, nil
}

func (s *fakeStore) Diagnose(_ context.Context, name string) (workspace.DiagnosticReport, error) {
	s.diagnoseName = name
	if s.diagnoseReport.Workspace == "" {
		return workspace.DiagnosticReport{Workspace: name, WorkspaceDir: "/tmp/adp-home/workspaces/" + name}, nil
	}
	return s.diagnoseReport, nil
}

func (s *fakeStore) DiagnoseAll(context.Context) ([]workspace.DiagnosticReport, error) {
	s.diagnoseAllCalled = true
	return s.diagnoseAllReports, nil
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
