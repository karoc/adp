package cli

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"strings"
	"testing"

	"github.com/karoc/adp/internal/adapters"
	"github.com/karoc/adp/internal/events"
	"github.com/karoc/adp/internal/paths"
	"github.com/karoc/adp/internal/runner"
	"github.com/karoc/adp/internal/runtime"
	taskstore "github.com/karoc/adp/internal/tasks"
)

func TestRunCommandBindsTaskContextToRuntimeEventsAndEnv(t *testing.T) {
	task := taskstore.Task{
		ID:          "task-20260608-0001",
		Title:       "Bind runtime session to task",
		Status:      taskstore.StatusReady,
		Priority:    "high",
		Phase:       "p1",
		Description: "Runtime task binding.",
	}
	store := &runTaskStore{task: task}
	adapter := &runTaskAdapter{name: "codex"}
	registry := adapters.NewRegistry()
	if err := registry.Register(adapter); err != nil {
		t.Fatal(err)
	}

	var taskStoreWorkspaceDir string
	var buildReq runtime.BuildRequest
	var launchSpec adapters.LaunchSpec
	var logged []events.Event
	deps := Dependencies{
		Layout:         paths.New("/tmp/adp-home", "/tmp/adp-runtime"),
		WorkspaceStore: &fakeStore{cfg: testConfig()},
		Adapters:       registry,
		TaskStoreFactory: func(workspaceDir string) TaskStore {
			taskStoreWorkspaceDir = workspaceDir
			return store
		},
		BuildRuntime: func(_ context.Context, req runtime.BuildRequest) (*runtime.Handle, error) {
			buildReq = req
			return &runtime.Handle{
				SessionID:     "session-1",
				WorkspaceName: "game-a",
				TaskID:        req.Task.ID,
				ProjectRoot:   "/srv/game-a",
				Root:          "/tmp/runtime",
				Env: map[string]string{
					"ADP_RUNTIME_ROOT": "/tmp/runtime",
					"ADP_SESSION_ID":   "session-1",
					"ADP_TASK_ID":      req.Task.ID,
				},
			}, nil
		},
		CleanupRuntime: func(context.Context, runtime.Handle) error { return nil },
		RunProcess: func(_ context.Context, spec adapters.LaunchSpec, _ runner.Streams) (*runner.Result, error) {
			launchSpec = spec
			return &runner.Result{ExitCode: 0}, nil
		},
		EventLogger: eventLoggerFunc(func(_ context.Context, event events.Event) error {
			logged = append(logged, event)
			return nil
		}),
	}

	code := NewApp(deps, &bytes.Buffer{}, &bytes.Buffer{}).Execute(
		context.Background(),
		[]string{"run", "codex", "--workspace", "game-a", "--task", task.ID, "--", "--probe"},
	)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if taskStoreWorkspaceDir != "/tmp/adp-home/workspaces/game-a" || store.getID != task.ID {
		t.Fatalf("task lookup = (%q, %q)", taskStoreWorkspaceDir, store.getID)
	}
	if adapter.renderCtx.Task.ID != task.ID || adapter.renderCtx.Task.Description != "Runtime task binding." {
		t.Fatalf("adapter task context = %+v", adapter.renderCtx.Task)
	}
	if buildReq.Task.ID != task.ID || buildReq.Task.Phase != "p1" {
		t.Fatalf("runtime task context = %+v", buildReq.Task)
	}
	if launchSpec.Env["ADP_TASK_ID"] != task.ID {
		t.Fatalf("launch env missing task id: %#v", launchSpec.Env)
	}
	if len(logged) != 2 || logged[0].TaskID != task.ID || logged[1].TaskID != task.ID {
		t.Fatalf("logged task events = %+v", logged)
	}
}

func TestRunCommandRejectsMissingTaskBeforeRuntimeBuild(t *testing.T) {
	registry := adapters.NewRegistry()
	if err := registry.Register(&runTaskAdapter{name: "codex"}); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	buildCalled := false
	runCalled := false
	deps := Dependencies{
		WorkspaceStore: &fakeStore{cfg: testConfig()},
		Adapters:       registry,
		TaskStoreFactory: func(string) TaskStore {
			return &runTaskStore{err: taskstore.ErrTaskNotFound}
		},
		BuildRuntime: func(context.Context, runtime.BuildRequest) (*runtime.Handle, error) {
			buildCalled = true
			return nil, nil
		},
		RunProcess: func(context.Context, adapters.LaunchSpec, runner.Streams) (*runner.Result, error) {
			runCalled = true
			return nil, nil
		},
	}

	code := NewApp(deps, &bytes.Buffer{}, &stderr).Execute(
		context.Background(),
		[]string{"run", "codex", "--workspace", "game-a", "--task", "missing-task"},
	)

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if buildCalled || runCalled {
		t.Fatalf("runtime build/run should not be called: build=%t run=%t", buildCalled, runCalled)
	}
	if !strings.Contains(stderr.String(), `load task "missing-task"`) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

type runTaskStore struct {
	task  taskstore.Task
	getID string
	err   error
}

func (s *runTaskStore) Add(context.Context, taskstore.AddRequest) (taskstore.Task, error) {
	return taskstore.Task{}, errors.New("not implemented")
}

func (s *runTaskStore) List(context.Context) ([]taskstore.Task, error) {
	return nil, errors.New("not implemented")
}

func (s *runTaskStore) Get(_ context.Context, id string) (taskstore.Task, error) {
	s.getID = id
	if s.err != nil {
		return taskstore.Task{}, s.err
	}
	return s.task, nil
}

func (s *runTaskStore) UpdateStatus(context.Context, string, taskstore.Status) (taskstore.Task, error) {
	return taskstore.Task{}, errors.New("not implemented")
}

func (s *runTaskStore) Block(context.Context, string, string) (taskstore.Task, error) {
	return taskstore.Task{}, errors.New("not implemented")
}

func (s *runTaskStore) Progress(context.Context) (taskstore.Progress, error) {
	return taskstore.Progress{}, errors.New("not implemented")
}

type runTaskAdapter struct {
	name      string
	renderCtx adapters.Context
}

func (a *runTaskAdapter) Name() string {
	return a.name
}

func (a *runTaskAdapter) Validate(context.Context, adapters.Context) error {
	return nil
}

func (a *runTaskAdapter) Render(_ context.Context, ctx adapters.Context) (*adapters.RenderResult, error) {
	a.renderCtx = ctx
	return &adapters.RenderResult{
		Files: []adapters.GeneratedFile{{Path: "AGENTS.md", Mode: fs.FileMode(0o644), Data: []byte("prompt")}},
		Env:   map[string]string{"ADAPTER_ENV": "1"},
	}, nil
}

func (a *runTaskAdapter) Launch(_ context.Context, _ adapters.Context, runtime adapters.RuntimeHandle, args []string) (*adapters.LaunchSpec, error) {
	env := map[string]string{}
	for key, value := range runtime.Env {
		env[key] = value
	}
	return &adapters.LaunchSpec{
		Command: "fake-codex",
		Args:    append([]string(nil), args...),
		Dir:     runtime.Root,
		Env:     env,
	}, nil
}
