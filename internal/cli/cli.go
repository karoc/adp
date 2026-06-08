package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

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
  adp enter <workspace> [--keep-runtime]
  adp run <agent> [--workspace <name>] [--profile <profile>] [--keep-runtime] [-- <agent-args>...]
`

type WorkspaceStore interface {
	Init(context.Context) error
	Add(context.Context, string, string) (*schema.Config, error)
	Get(context.Context, string) (*schema.Config, string, error)
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

func (a *App) init(ctx context.Context, args []string) error {
	if len(args) != 0 {
		return errors.New("usage: adp init")
	}
	if a.deps.WorkspaceStore == nil {
		return errors.New("workspace store is not configured")
	}
	if err := a.deps.WorkspaceStore.Init(ctx); err != nil {
		return err
	}
	fmt.Fprintln(a.stdout, "initialized ADP home")
	return nil
}

func (a *App) workspace(ctx context.Context, args []string) error {
	if len(args) != 3 || args[0] != "add" {
		return errors.New("usage: adp workspace add <name> <project-root>")
	}
	if a.deps.WorkspaceStore == nil {
		return errors.New("workspace store is not configured")
	}
	if _, err := a.deps.WorkspaceStore.Add(ctx, args[1], args[2]); err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "workspace %q added\n", args[1])
	return nil
}

func (a *App) enter(ctx context.Context, args []string) error {
	name, keep, err := parseEnterArgs(args)
	if err != nil {
		return err
	}

	cfg, workspaceDir, err := a.loadWorkspace(ctx, name)
	if err != nil {
		return err
	}
	if a.deps.BuildRuntime == nil {
		return errors.New("runtime builder is not configured")
	}
	handle, err := a.deps.BuildRuntime(ctx, runtime.BuildRequest{
		Layout:       a.deps.Layout,
		Config:       *cfg,
		WorkspaceDir: workspaceDir,
		Keep:         keep,
	})
	if err != nil {
		return err
	}
	defer a.cleanupRuntime(ctx, *handle)
	if a.deps.EnterShell == nil {
		return errors.New("shell runner is not configured")
	}

	return a.deps.EnterShell(ctx, *handle, shell.Streams{
		Stdin:  os.Stdin,
		Stdout: a.stdout,
		Stderr: a.stderr,
	})
}

func (a *App) run(ctx context.Context, args []string) error {
	opts, err := parseRunArgs(args)
	if err != nil {
		return err
	}

	cfg, workspaceDir, err := a.loadWorkspace(ctx, opts.workspace)
	if err != nil {
		return err
	}
	if a.deps.Adapters == nil {
		return errors.New("adapter registry is not configured")
	}
	adapter, ok := a.deps.Adapters.Get(opts.agent)
	if !ok {
		return fmt.Errorf("unknown adapter %q; available: %s", opts.agent, strings.Join(a.deps.Adapters.Names(), ", "))
	}

	agentCfg := cfg.Agents[opts.agent]
	profile := opts.profile
	if profile == "" {
		profile = agentCfg.Profile
	}
	adapterCtx := adapters.Context{
		Layout:       a.deps.Layout,
		WorkspaceDir: workspaceDir,
		Config:       *cfg,
		Agent:        agentCfg,
		Profile:      profile,
	}
	if err := adapter.Validate(ctx, adapterCtx); err != nil {
		return err
	}
	rendered, err := adapter.Render(ctx, adapterCtx)
	if err != nil {
		return err
	}

	started := time.Now()
	if a.deps.BuildRuntime == nil {
		return errors.New("runtime builder is not configured")
	}
	handle, err := a.deps.BuildRuntime(ctx, runtime.BuildRequest{
		Layout:       a.deps.Layout,
		Config:       *cfg,
		WorkspaceDir: workspaceDir,
		Files:        rendered.Files,
		Env:          rendered.Env,
		Keep:         opts.keep,
	})
	if err != nil {
		return err
	}
	defer a.cleanupRuntime(ctx, *handle)

	a.logEvent(ctx, events.Event{
		Timestamp:   started,
		Type:        "run_started",
		Workspace:   cfg.Workspace.Name,
		Agent:       opts.agent,
		Profile:     profile,
		RuntimePath: handle.Root,
		ProjectRoot: cfg.Project.Root,
		SessionID:   handle.SessionID,
	})

	spec, err := adapter.Launch(ctx, adapterCtx, *handle, opts.agentArgs)
	if err != nil {
		return err
	}
	if spec.Dir == "" {
		spec.Dir = handle.Root
	}
	spec.Env = mergedEnv(spec.Env, handle.Env)
	if a.deps.RunProcess == nil {
		return errors.New("process runner is not configured")
	}

	result, err := a.deps.RunProcess(ctx, *spec, runner.Streams{
		Stdin:  os.Stdin,
		Stdout: a.stdout,
		Stderr: a.stderr,
	})
	exitCode := 1
	if result != nil {
		exitCode = result.ExitCode
	}
	a.logEvent(ctx, events.Event{
		Timestamp:      time.Now(),
		Type:           "run_finished",
		Workspace:      cfg.Workspace.Name,
		Agent:          opts.agent,
		Profile:        profile,
		RuntimePath:    handle.Root,
		ProjectRoot:    cfg.Project.Root,
		SessionID:      handle.SessionID,
		ExitCode:       &exitCode,
		DurationMillis: time.Since(started).Milliseconds(),
	})
	if err != nil {
		return err
	}
	if exitCode != 0 {
		return processExitError{code: exitCode}
	}
	return nil
}

func (a *App) loadWorkspace(ctx context.Context, name string) (*schema.Config, string, error) {
	if name == "" {
		name = os.Getenv("ADP_WORKSPACE")
	}
	if name == "" {
		return nil, "", errors.New("workspace is required; pass --workspace or set ADP_WORKSPACE")
	}
	if a.deps.WorkspaceStore == nil {
		return nil, "", errors.New("workspace store is not configured")
	}
	return a.deps.WorkspaceStore.Get(ctx, name)
}

func (a *App) logEvent(ctx context.Context, event events.Event) {
	if a.deps.EventLogger == nil {
		return
	}
	if err := a.deps.EventLogger.Log(ctx, event); err != nil {
		fmt.Fprintf(a.stderr, "warning: failed to write event log: %v\n", err)
	}
}

func (a *App) cleanupRuntime(ctx context.Context, handle runtime.Handle) {
	if a.deps.CleanupRuntime == nil {
		return
	}
	if err := a.deps.CleanupRuntime(ctx, handle); err != nil {
		fmt.Fprintf(a.stderr, "warning: failed to clean runtime: %v\n", err)
	}
}

func (a *App) fail(err error) int {
	fmt.Fprintf(a.stderr, "adp: %v\n", err)
	return 1
}

type runOptions struct {
	agent     string
	workspace string
	profile   string
	keep      bool
	agentArgs []string
}

type processExitError struct {
	code int
}

func (e processExitError) Error() string {
	return fmt.Sprintf("process exited with code %d", e.code)
}

func parseRunArgs(args []string) (runOptions, error) {
	if len(args) == 0 {
		return runOptions{}, errors.New("usage: adp run <agent> [--workspace <name>] [--profile <profile>] [--keep-runtime] [-- <agent-args>...]")
	}
	opts := runOptions{agent: args[0]}
	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--":
			opts.agentArgs = append(opts.agentArgs, args[i+1:]...)
			return opts, nil
		case "--workspace", "-w":
			if i+1 >= len(args) {
				return runOptions{}, fmt.Errorf("%s requires a value", arg)
			}
			i++
			opts.workspace = args[i]
		case "--profile", "-p":
			if i+1 >= len(args) {
				return runOptions{}, fmt.Errorf("%s requires a value", arg)
			}
			i++
			opts.profile = args[i]
		case "--keep-runtime":
			opts.keep = true
		default:
			return runOptions{}, fmt.Errorf("unknown run option %q", arg)
		}
	}
	return opts, nil
}

func parseEnterArgs(args []string) (string, bool, error) {
	var name string
	var keep bool
	for _, arg := range args {
		switch arg {
		case "--keep-runtime":
			keep = true
		default:
			if strings.HasPrefix(arg, "-") {
				return "", false, fmt.Errorf("unknown enter option %q", arg)
			}
			if name != "" {
				return "", false, errors.New("usage: adp enter <workspace> [--keep-runtime]")
			}
			name = arg
		}
	}
	if name == "" {
		return "", false, errors.New("usage: adp enter <workspace> [--keep-runtime]")
	}
	return name, keep, nil
}

func mergedEnv(base map[string]string, overrides map[string]string) map[string]string {
	env := map[string]string{}
	for key, value := range base {
		env[key] = value
	}
	for key, value := range overrides {
		env[key] = value
	}
	return env
}
