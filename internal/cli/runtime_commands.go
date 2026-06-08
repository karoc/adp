package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/karoc/adp/internal/adapters"
	"github.com/karoc/adp/internal/events"
	"github.com/karoc/adp/internal/runner"
	"github.com/karoc/adp/internal/runtime"
	"github.com/karoc/adp/internal/schema"
	"github.com/karoc/adp/internal/shell"
)

type processExitError struct {
	code int
}

func (e processExitError) Error() string {
	return fmt.Sprintf("process exited with code %d", e.code)
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

func (a *App) env(ctx context.Context, args []string) error {
	name, changeDir, err := parseEnvArgs(args)
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
		Keep:         true,
	})
	if err != nil {
		return err
	}

	output, err := shell.RenderExports(*handle, shell.ExportOptions{ChangeDir: changeDir})
	if err != nil {
		_ = a.cleanupRuntimeAfterError(ctx, *handle)
		return err
	}
	fmt.Fprint(a.stdout, output)
	return nil
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
	if a.deps.WorkspaceStore == nil {
		return nil, "", errors.New("workspace store is not configured")
	}
	if name == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, "", fmt.Errorf("resolve current directory: %w", err)
		}
		cfg, workspaceDir, err := a.deps.WorkspaceStore.FindByProjectPath(ctx, cwd)
		if err != nil {
			return nil, "", errors.New("workspace is required; pass --workspace, set ADP_WORKSPACE, or run from inside a registered project")
		}
		return cfg, workspaceDir, nil
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

func (a *App) cleanupRuntimeAfterError(ctx context.Context, handle runtime.Handle) error {
	if a.deps.CleanupRuntime == nil {
		return nil
	}
	return a.deps.CleanupRuntime(ctx, handle)
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
