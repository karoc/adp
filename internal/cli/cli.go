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
	"github.com/karoc/adp/internal/shell"
	"github.com/karoc/adp/internal/workspace"
)

const usage = `adp - Agent Development Platform

Usage:
  adp init
  adp workspace add <name> <project-root>
  adp workspace list
  adp workspace show <name>
  adp workspace remove <name>
  adp workspace rename <old-name> <new-name>
  adp enter <workspace> [--keep-runtime]
  adp env <workspace> [--cd]
  adp shell-hook [--shell <sh|bash|zsh>] [--name <function-name>]
  adp events list [--workspace <name>] [--session <session-id>] [--type <event-type>] [--limit <n>]
  adp runtime prune [--older-than <duration>] [--include-kept] [--dry-run]
  adp run <agent> [--workspace <name>] [--profile <profile>] [--keep-runtime] [-- <agent-args>...]
`

type WorkspaceStore interface {
	Init(context.Context) error
	Add(context.Context, string, string) (*schema.Config, error)
	Get(context.Context, string) (*schema.Config, string, error)
	List(context.Context) ([]workspace.Record, error)
	FindByProjectPath(context.Context, string) (*schema.Config, string, error)
	Remove(context.Context, string) error
	Rename(context.Context, string, string) (*schema.Config, error)
}

type AdapterRegistry interface {
	Get(string) (adapters.Adapter, bool)
	Names() []string
}

type EventLogger interface {
	Log(context.Context, events.Event) error
}

type Dependencies struct {
	Layout         paths.Layout
	WorkspaceStore WorkspaceStore
	Adapters       AdapterRegistry
	BuildRuntime   func(context.Context, runtime.BuildRequest) (*runtime.Handle, error)
	CleanupRuntime func(context.Context, runtime.Handle) error
	RunProcess     func(context.Context, adapters.LaunchSpec, runner.Streams) (*runner.Result, error)
	EnterShell     func(context.Context, adapters.RuntimeHandle, shell.Streams) error
	EventLogger    EventLogger
	ReadEvents     func(context.Context, paths.Layout, events.Query) ([]events.Event, error)
	PruneRuntimes  func(context.Context, runtime.PruneRequest) ([]runtime.PruneResult, error)
	RenderHook     func(shell.HookOptions) (string, error)
	InitError      error
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
	deps.PruneRuntimes = runtime.Prune
	deps.RenderHook = shell.RenderHook
	return deps
}

func (a *App) Execute(ctx context.Context, args []string) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Fprint(a.stdout, usage)
		return 0
	}
	if a.deps.InitError != nil {
		return a.fail(a.deps.InitError)
	}

	var err error
	switch args[0] {
	case "init":
		err = a.init(ctx, args[1:])
	case "workspace":
		err = a.workspace(ctx, args[1:])
	case "enter":
		err = a.enter(ctx, args[1:])
	case "env":
		err = a.env(ctx, args[1:])
	case "shell-hook":
		err = a.shellHook(ctx, args[1:])
	case "events":
		err = a.events(ctx, args[1:])
	case "runtime":
		err = a.runtime(ctx, args[1:])
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
