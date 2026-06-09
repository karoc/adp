package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/karoc/adp/internal/adapters"
	"github.com/karoc/adp/internal/commandmeta"
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
	Take(context.Context, taskstore.TakeRequest) (taskstore.Task, error)
	Renew(context.Context, taskstore.RenewRequest) (taskstore.Task, error)
	Stale(context.Context) ([]taskstore.Task, error)
	Release(context.Context, taskstore.ReleaseRequest) (taskstore.Task, error)
	Progress(context.Context) (taskstore.Progress, error)
	PreviewPlanImport(context.Context, taskstore.PlanImportRequest) (taskstore.PlanImportResult, error)
	ApplyPlanImport(context.Context, taskstore.PlanImportRequest) (taskstore.PlanImportResult, error)
	DiagnosePlanning(context.Context) (taskstore.PlanningDiagnosticReport, error)
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

type commandHandler func(context.Context, []string) error

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
		fmt.Fprint(a.stdout, commandmeta.Usage())
		return 0
	}
	if args[0] == "--version" || args[0] == "-v" {
		fmt.Fprint(a.stdout, versionString())
		return 0
	}
	if strings.HasPrefix(args[0], "-") {
		return a.failWithHint(fmt.Errorf("unknown global option %q", args[0]), args)
	}
	if output, ok := commandHelp(args); ok {
		fmt.Fprint(a.stdout, output)
		return 0
	}
	if a.deps.InitError != nil {
		return a.fail(a.deps.InitError)
	}

	handler, ok := a.commandHandlers()[args[0]]
	if !ok {
		return a.failWithHint(fmt.Errorf("unknown command %q", args[0]), args)
	}
	err := handler(ctx, args[1:])
	if err != nil {
		var processExit processExitError
		if errors.As(err, &processExit) {
			return processExit.code
		}
		var shellExit shell.ExitError
		if errors.As(err, &shellExit) {
			return shellExit.Code
		}
		return a.failWithHint(err, args)
	}
	return 0
}

func (a *App) commandHandlers() map[string]commandHandler {
	return map[string]commandHandler{
		"init":       a.init,
		"doctor":     a.doctor,
		"version":    a.version,
		"workspace":  a.workspace,
		"enter":      a.enter,
		"env":        a.env,
		"shell-hook": a.shellHook,
		"completion": a.completion,
		"events":     a.events,
		"sessions":   a.sessions,
		"runtime":    a.runtime,
		"tasks":      a.tasks,
		"plan":       a.plan,
		"phase":      a.phase,
		"progress":   a.progress,
		"run":        a.run,
	}
}

func (a *App) fail(err error) int {
	fmt.Fprintf(a.stderr, "adp: %v\n", err)
	return 1
}

func (a *App) failWithHint(err error, args []string) int {
	fmt.Fprintf(a.stderr, "adp: %v\n", err)
	if shouldShowHelpHint(err) {
		hint := helpHint(args)
		fmt.Fprintf(a.stderr, "try: %s\n", hint)
	}
	return 1
}

func shouldShowHelpHint(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	switch {
	case strings.HasPrefix(message, "usage: "):
		return true
	case strings.HasPrefix(message, "unknown command "):
		return true
	case strings.HasPrefix(message, "unknown global option "):
		return true
	case strings.Contains(message, " command "):
		return strings.HasPrefix(message, "unknown ")
	case strings.Contains(message, " option "):
		return strings.HasPrefix(message, "unknown ")
	case strings.Contains(message, " requires a value"):
		return true
	case strings.HasPrefix(message, "unknown output format "):
		return true
	case strings.HasPrefix(message, "unknown completion values kind "):
		return true
	case strings.HasPrefix(message, "unknown progress report language "):
		return true
	case strings.HasPrefix(message, "unknown progress report format "):
		return true
	case strings.HasPrefix(message, "parse lease duration: "):
		return true
	case strings.HasPrefix(message, "parse older-than duration: "):
		return true
	case strings.HasPrefix(message, "parse limit: "):
		return true
	case message == "--take cannot be combined with --task":
		return true
	case message == "--owner is required with --take":
		return true
	case message == "--owner requires --take":
		return true
	case message == "--lease requires --take":
		return true
	case message == "lease must not be negative":
		return true
	case message == "older-than must not be negative":
		return true
	case message == "limit must not be negative":
		return true
	default:
		return false
	}
}

func helpHint(args []string) string {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return "adp --help"
	}
	command := args[0]
	if _, ok := commandmeta.Lookup(command); !ok {
		return "adp --help"
	}
	if len(args) >= 2 && !strings.HasPrefix(args[1], "-") && isKnownSubcommand(command, args[1]) {
		return fmt.Sprintf("adp %s %s --help", command, args[1])
	}
	return fmt.Sprintf("adp %s --help", command)
}

func commandHelp(args []string) (string, bool) {
	if len(args) == 2 && isHelpArg(args[1]) {
		return commandmeta.CommandHelp(args[0])
	}
	if len(args) == 3 && isHelpArg(args[2]) && isKnownSubcommand(args[0], args[1]) {
		return commandmeta.SubcommandHelp(args[0], args[1])
	}
	return "", false
}

func isHelpArg(arg string) bool {
	return arg == "--help" || arg == "-h"
}

func isKnownSubcommand(command, subcommand string) bool {
	for _, name := range commandmeta.SubcommandNames(command) {
		if name == subcommand {
			return true
		}
	}
	return false
}
