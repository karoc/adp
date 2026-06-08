package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/karoc/adp/internal/adapters"
	"github.com/karoc/adp/internal/events"
	"github.com/karoc/adp/internal/paths"
	"github.com/karoc/adp/internal/runner"
	"github.com/karoc/adp/internal/runtime"
	"github.com/karoc/adp/internal/schema"
	"github.com/karoc/adp/internal/sessions"
	"github.com/karoc/adp/internal/shell"
	taskstore "github.com/karoc/adp/internal/tasks"
	"github.com/karoc/adp/internal/workspace"
)

const usage = `adp - Agent Development Platform

Usage:
  adp init
  adp doctor [workspace]
  adp version
  adp workspace add <name> <project-root>
  adp workspace list
  adp workspace show <name>
  adp workspace remove <name>
  adp workspace rename <old-name> <new-name>
  adp workspace doctor [name]
  adp enter <workspace> [--keep-runtime]
  adp env <workspace> [--cd]
  adp shell-hook [--shell <sh|bash|zsh>] [--name <function-name>]
  adp completion [--shell <bash|zsh>] [--command <name>]
  adp completion values <workspaces|profiles> [--workspace <name>]
  adp events list [--workspace <name>] [--session <session-id>] [--task <task-id>] [--type <event-type>] [--limit <n>]
	  adp sessions list [--workspace <name>] [--agent <agent>] [--task <task-id>] [--limit <n>]
	  adp sessions show <session-id>
	  adp runtime prune [--older-than <duration>] [--include-kept] [--dry-run]
	  adp tasks add [--workspace <name>] [--priority <value>] [--phase <value>] [--description <text>] <title>
	  adp tasks list [--workspace <name>]
	  adp tasks show [--workspace <name>] <task-id>
	  adp tasks update [--workspace <name>] <task-id> --status <status>
	  adp tasks claim [--workspace <name>] <task-id> --owner <owner> [--lease <duration>]
	  adp tasks release [--workspace <name>] <task-id> [--owner <owner>]
	  adp tasks done [--workspace <name>] <task-id>
	  adp tasks block [--workspace <name>] <task-id> --reason <reason>
	  adp phase add [--workspace <name>] [--goal <text>] <phase-id> <title>
	  adp phase list [--workspace <name>]
	  adp phase show [--workspace <name>] <phase-id>
	  adp phase start [--workspace <name>] <phase-id>
	  adp phase accept [--workspace <name>] <phase-id> [--command <cmd>] [--result <result>] [--notes <text>]
	  adp phase commit [--workspace <name>] <phase-id> --hash <commit-hash> [--message <text>]
	  adp phase push [--workspace <name>] <phase-id> --remote <remote> --branch <branch> [--result <result>]
	  adp progress [--workspace <name>]
	  adp run <agent> [--workspace <name>] [--profile <profile>] [--task <task-id>] [--keep-runtime] [-- <agent-args>...]
	`

var (
	Version   = "dev"
	Commit    = ""
	BuildDate = ""
)

type WorkspaceStore interface {
	Init(context.Context) error
	Add(context.Context, string, string) (*schema.Config, error)
	Get(context.Context, string) (*schema.Config, string, error)
	List(context.Context) ([]workspace.Record, error)
	Names(context.Context) ([]string, error)
	FindByProjectPath(context.Context, string) (*schema.Config, string, error)
	Remove(context.Context, string) error
	Rename(context.Context, string, string) (*schema.Config, error)
	Diagnose(context.Context, string) (workspace.DiagnosticReport, error)
	DiagnoseAll(context.Context) ([]workspace.DiagnosticReport, error)
}

type AdapterRegistry interface {
	Get(string) (adapters.Adapter, bool)
	Names() []string
}

type EventLogger interface {
	Log(context.Context, events.Event) error
}

type TaskStore interface {
	Add(context.Context, taskstore.AddRequest) (taskstore.Task, error)
	List(context.Context) ([]taskstore.Task, error)
	Get(context.Context, string) (taskstore.Task, error)
	UpdateStatus(context.Context, string, taskstore.Status) (taskstore.Task, error)
	Block(context.Context, string, string) (taskstore.Task, error)
	Claim(context.Context, taskstore.ClaimRequest) (taskstore.Task, error)
	Release(context.Context, taskstore.ReleaseRequest) (taskstore.Task, error)
	Progress(context.Context) (taskstore.Progress, error)
	AddPhase(context.Context, taskstore.PhaseAddRequest) (taskstore.Phase, error)
	ListPhases(context.Context) ([]taskstore.Phase, error)
	GetPhase(context.Context, string) (taskstore.Phase, error)
	StartPhase(context.Context, string) (taskstore.Phase, error)
	AcceptPhase(context.Context, taskstore.PhaseAcceptRequest) (taskstore.Phase, error)
	RecordPhaseCommit(context.Context, taskstore.PhaseCommitRequest) (taskstore.Phase, error)
	RecordPhasePush(context.Context, taskstore.PhasePushRequest) (taskstore.Phase, error)
}

type Dependencies struct {
	Layout           paths.Layout
	WorkspaceStore   WorkspaceStore
	Adapters         AdapterRegistry
	BuildRuntime     func(context.Context, runtime.BuildRequest) (*runtime.Handle, error)
	CleanupRuntime   func(context.Context, runtime.Handle) error
	RunProcess       func(context.Context, adapters.LaunchSpec, runner.Streams) (*runner.Result, error)
	EnterShell       func(context.Context, adapters.RuntimeHandle, shell.Streams) error
	EventLogger      EventLogger
	ReadEvents       func(context.Context, paths.Layout, events.Query) ([]events.Event, error)
	ListSessions     func(context.Context, paths.Layout, sessions.Query) ([]sessions.Summary, error)
	GetSession       func(context.Context, paths.Layout, string) (*sessions.Detail, error)
	PruneRuntimes    func(context.Context, runtime.PruneRequest) ([]runtime.PruneResult, error)
	RenderHook       func(shell.HookOptions) (string, error)
	RenderCompletion func(shell.CompletionOptions) (string, error)
	TaskStoreFactory func(string) TaskStore
	InitError        error
}

type App struct {
	deps   Dependencies
	stdout io.Writer
	stderr io.Writer
}

func Execute(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	app := NewApp(DefaultDependencies(), stdout, stderr)
	return app.Execute(ctx, args)
}

func NewApp(deps Dependencies, stdout io.Writer, stderr io.Writer) *App {
	return &App{deps: deps, stdout: stdout, stderr: stderr}
}

func DefaultDependencies() Dependencies {
	layout, err := paths.FromEnv()
	deps := Dependencies{Layout: layout, InitError: err}
	if err != nil {
		return deps
	}

	deps.WorkspaceStore = workspace.NewRegistry(layout)
	registry, registryErr := adapters.NewDefaultRegistry()
	if registryErr != nil {
		deps.InitError = registryErr
		return deps
	}
	deps.Adapters = registry
	deps.BuildRuntime = runtime.Build
	deps.CleanupRuntime = runtime.Cleanup
	deps.RunProcess = runner.Run
	deps.EnterShell = shell.Enter
	deps.EventLogger = events.NewLogger(layout)
	deps.ReadEvents = events.Read
	deps.ListSessions = sessions.List
	deps.GetSession = sessions.Get
	deps.PruneRuntimes = runtime.Prune
	deps.RenderHook = shell.RenderHook
	deps.RenderCompletion = shell.RenderCompletion
	deps.TaskStoreFactory = func(workspaceDir string) TaskStore {
		return taskstore.NewStore(workspaceDir)
	}
	return deps
}

func (a *App) Execute(ctx context.Context, args []string) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Fprint(a.stdout, usage)
		return 0
	}
	if args[0] == "--version" || args[0] == "-v" {
		fmt.Fprint(a.stdout, versionString())
		return 0
	}
	if a.deps.InitError != nil {
		return a.fail(a.deps.InitError)
	}

	var err error
	switch args[0] {
	case "init":
		err = a.init(ctx, args[1:])
	case "doctor":
		err = a.doctor(ctx, args[1:])
	case "version":
		err = a.version(ctx, args[1:])
	case "workspace":
		err = a.workspace(ctx, args[1:])
	case "enter":
		err = a.enter(ctx, args[1:])
	case "env":
		err = a.env(ctx, args[1:])
	case "shell-hook":
		err = a.shellHook(ctx, args[1:])
	case "completion":
		err = a.completion(ctx, args[1:])
	case "events":
		err = a.events(ctx, args[1:])
	case "sessions":
		err = a.sessions(ctx, args[1:])
	case "runtime":
		err = a.runtime(ctx, args[1:])
	case "tasks":
		err = a.tasks(ctx, args[1:])
	case "phase":
		err = a.phase(ctx, args[1:])
	case "progress":
		err = a.progress(ctx, args[1:])
	case "run":
		err = a.run(ctx, args[1:])
	default:
		err = fmt.Errorf("unknown command %q", args[0])
	}
	if err != nil {
		var processExit processExitError
		if errors.As(err, &processExit) {
			return processExit.code
		}
		var shellExit shell.ExitError
		if errors.As(err, &shellExit) {
			return shellExit.Code
		}
		return a.fail(err)
	}
	return 0
}

func (a *App) fail(err error) int {
	fmt.Fprintf(a.stderr, "adp: %v\n", err)
	return 1
}
