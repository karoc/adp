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
	"github.com/karoc/adp/internal/gitenv"
	"github.com/karoc/adp/internal/gitstate"
	"github.com/karoc/adp/internal/output"
	"github.com/karoc/adp/internal/runner"
	"github.com/karoc/adp/internal/runtime"
	"github.com/karoc/adp/internal/schema"
	"github.com/karoc/adp/internal/shell"
	taskstore "github.com/karoc/adp/internal/tasks"
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
	taskCtx, err := a.loadRunTaskContext(ctx, workspaceDir, opts)
	if err != nil {
		return err
	}
	gitRoot := gitstate.DiscoverRoot(ctx, cfg.Project.Root)

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
		GitRoot:      gitRoot,
		Task:         taskCtx,
	}
	if err := adapter.Validate(ctx, adapterCtx); err != nil {
		return err
	}

	// Show progress for runtime setup
	spinner := output.NewSpinner(a.stderr, "Building runtime environment...")
	spinner.Start()

	rendered, err := adapter.Render(ctx, adapterCtx)
	if err != nil {
		spinner.Fail("")
		return err
	}

	started := time.Now()
	if a.deps.BuildRuntime == nil {
		spinner.Fail("")
		return errors.New("runtime builder is not configured")
	}
	handle, err := a.deps.BuildRuntime(ctx, runtime.BuildRequest{
		Layout:       a.deps.Layout,
		Config:       *cfg,
		WorkspaceDir: workspaceDir,
		Files:        rendered.Files,
		Env:          rendered.Env,
		GitRoot:      gitRoot,
		Task:         taskCtx,
		Keep:         opts.keep,
	})
	if err != nil {
		spinner.Fail("")
		return err
	}
	defer a.cleanupRuntime(ctx, *handle)

	spinner.Success("Runtime ready")

	a.logEvent(ctx, events.Event{
		Timestamp:   started,
		Type:        "run_started",
		Workspace:   cfg.Workspace.Name,
		Agent:       opts.agent,
		Profile:     profile,
		RuntimePath: handle.Root,
		ProjectRoot: cfg.Project.Root,
		SessionID:   handle.SessionID,
		TaskID:      taskCtx.ID,
		Fields:      runInvocationFields(opts, profile, taskCtx),
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

	// Handle command not found error with friendly message
	var cmdNotFoundErr *runner.CommandNotFoundError
	if errors.As(err, &cmdNotFoundErr) {
		return fmt.Errorf(`agent command not found: %s

The workspace is configured to use:
  command: %s
  path lookup: $PATH

Possible solutions:
  - Install %s CLI
  - Configure explicit command path in workspace settings
  - Check if the command is in your $PATH

try: adp workspace doctor %s`, cmdNotFoundErr.Command, cmdNotFoundErr.Command, opts.agent, cfg.Workspace.Name)
	}

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
		TaskID:         taskCtx.ID,
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

func (a *App) loadRunTaskContext(ctx context.Context, workspaceDir string, opts runOptions) (adapters.TaskContext, error) {
	if !opts.take && strings.TrimSpace(opts.taskID) == "" {
		return adapters.TaskContext{}, nil
	}
	if a.deps.TaskStoreFactory == nil {
		return adapters.TaskContext{}, errors.New("task store is not configured")
	}
	store := a.deps.TaskStoreFactory(workspaceDir)
	if opts.take {
		task, err := store.Take(ctx, taskstore.TakeRequest{
			Owner: opts.owner,
			Lease: opts.lease,
		})
		if err != nil {
			return adapters.TaskContext{}, fmt.Errorf("take task: %w", err)
		}
		return taskContext(task), nil
	}

	taskID := strings.TrimSpace(opts.taskID)

	// Resolve task ID prefix
	tasks, err := store.FindByPrefix(ctx, taskID)
	if err != nil {
		if errors.Is(err, taskstore.ErrAmbiguousTaskID) {
			// Extract task IDs for a friendly error message
			ids := make([]string, len(tasks))
			for i, task := range tasks {
				ids[i] = task.ID
			}
			return adapters.TaskContext{}, fmt.Errorf("ambiguous task ID %q, matches multiple tasks:\n  - %s\n\nPlease use a more specific prefix.", taskID, strings.Join(ids, "\n  - "))
		}
		return adapters.TaskContext{}, fmt.Errorf("load task %q: %w", taskID, err)
	}
	if len(tasks) != 1 {
		return adapters.TaskContext{}, fmt.Errorf("task %q not found", taskID)
	}

	return taskContext(tasks[0]), nil
}

func taskContext(task taskstore.Task) adapters.TaskContext {
	return adapters.TaskContext{
		ID:             task.ID,
		Title:          task.Title,
		Status:         string(task.Status),
		Priority:       task.Priority,
		Phase:          task.Phase,
		Owner:          task.Owner,
		ClaimedAt:      task.ClaimedAt,
		LeaseExpiresAt: task.LeaseExpiresAt,
		Description:    task.Description,
		BlockedReason:  task.BlockedReason,
	}
}

func runInvocationFields(opts runOptions, profile string, taskCtx adapters.TaskContext) map[string]any {
	invocation := map[string]any{
		"schema_version":       1,
		"agent_args":           append([]string(nil), opts.agentArgs...),
		"keep_runtime":         opts.keep,
		"workspace_resolution": workspaceResolutionSource(opts),
		"profile_source":       profileSource(opts, profile),
		"task_binding":         taskBindingSource(opts, taskCtx),
	}
	if opts.take {
		invocation["task_take"] = map[string]any{
			"owner":         strings.TrimSpace(opts.owner),
			"lease_seconds": int64(opts.lease.Seconds()),
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		invocation["original_cwd"] = cwd
	}
	if !taskCtx.IsZero() {
		taskSnapshot := map[string]any{
			"id":       taskCtx.ID,
			"title":    taskCtx.Title,
			"status":   taskCtx.Status,
			"priority": taskCtx.Priority,
			"phase":    taskCtx.Phase,
		}
		if strings.TrimSpace(taskCtx.Owner) != "" {
			taskSnapshot["owner"] = taskCtx.Owner
		}
		if !taskCtx.ClaimedAt.IsZero() {
			taskSnapshot["claimed_at"] = taskCtx.ClaimedAt.UTC().Format(time.RFC3339)
		}
		if !taskCtx.LeaseExpiresAt.IsZero() {
			taskSnapshot["lease_expires_at"] = taskCtx.LeaseExpiresAt.UTC().Format(time.RFC3339)
		}
		invocation["task_snapshot"] = taskSnapshot
	}
	return map[string]any{"invocation": invocation}
}

func taskBindingSource(opts runOptions, taskCtx adapters.TaskContext) string {
	switch {
	case opts.take:
		return "take"
	case !taskCtx.IsZero():
		return "explicit"
	default:
		return "none"
	}
}

func workspaceResolutionSource(opts runOptions) string {
	switch {
	case opts.workspace != "":
		return "--workspace"
	case os.Getenv("ADP_WORKSPACE") != "":
		return "ADP_WORKSPACE"
	default:
		return "cwd"
	}
}

func profileSource(opts runOptions, profile string) string {
	switch {
	case opts.profile != "":
		return "--profile"
	case profile != "":
		return "workspace"
	default:
		return "default"
	}
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
		if gitenv.IsRepositoryDirective(key) {
			continue
		}
		env[key] = value
	}
	for key, value := range overrides {
		if gitenv.IsRepositoryDirective(key) {
			continue
		}
		env[key] = value
	}
	return env
}
