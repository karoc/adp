package cli

import (
	"context"
	"errors"
	"io/fs"
	"sort"

	"github.com/karoc/adp/internal/adapters"
	"github.com/karoc/adp/internal/events"
	"github.com/karoc/adp/internal/schema"
	"github.com/karoc/adp/internal/workspace"
)

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
