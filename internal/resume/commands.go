package resume

func (p *Plan) addInspectionCommands() {
	if p.SessionID != "" {
		p.SuggestedCommands = append(p.SuggestedCommands, Command{
			Label:      "inspect-session",
			SideEffect: CommandSideEffectInspect,
			Args:       []string{"adp", "sessions", "show", p.SessionID},
			Reason:     "Review the source ADP session evidence before resuming.",
		})
	}
	if p.Target.Workspace != "" {
		p.SuggestedCommands = append(p.SuggestedCommands, Command{
			Label:      "inspect-progress",
			SideEffect: CommandSideEffectInspect,
			Args:       []string{"adp", "progress", "report", "--workspace", p.Target.Workspace, "--format", "json"},
			Reason:     "Review current task, phase, progress, and session evidence.",
		})
	}
}

func (p *Plan) addTaskInspectCommand(taskID string) {
	if p.Target.Workspace == "" || taskID == "" {
		return
	}
	p.SuggestedCommands = append(p.SuggestedCommands, Command{
		Label:      "inspect-task",
		SideEffect: CommandSideEffectInspect,
		Args:       []string{"adp", "tasks", "show", "--workspace", p.Target.Workspace, taskID, "--format", "json"},
		Reason:     "Inspect the current ADP task state before changing ownership.",
	})
}

func (p *Plan) addStaleCommand() {
	if p.Target.Workspace == "" {
		return
	}
	p.SuggestedCommands = append(p.SuggestedCommands, Command{
		Label:      "inspect-stale-claims",
		SideEffect: CommandSideEffectInspect,
		Args:       []string{"adp", "tasks", "stale", "--workspace", p.Target.Workspace, "--format", "json"},
		Reason:     "Confirm interrupted or expired in-progress work before reclaiming it.",
	})
}

func (p *Plan) addClaimCommand(taskID string) {
	if p.Target.Workspace == "" || taskID == "" {
		return
	}
	owner := p.Target.Owner
	if owner == "" {
		owner = "<owner>"
	}
	args := []string{"adp", "tasks", "claim", "--workspace", p.Target.Workspace, taskID, "--owner", owner}
	if p.Target.Lease != "" {
		args = append(args, "--lease", p.Target.Lease)
	}
	p.SuggestedCommands = append(p.SuggestedCommands, Command{
		Label:      "claim-task",
		SideEffect: CommandSideEffectTaskMutation,
		Args:       args,
		Reason:     "Take durable ADP ownership before launching a resumed worker.",
	})
}

func (p *Plan) addRenewCommand(taskID string) {
	if p.Target.Workspace == "" || taskID == "" || p.Target.Owner == "" {
		return
	}
	lease := p.Target.Lease
	if lease == "" {
		lease = "4h"
	}
	p.SuggestedCommands = append(p.SuggestedCommands, Command{
		Label:      "renew-task-lease",
		SideEffect: CommandSideEffectTaskMutation,
		Args:       []string{"adp", "tasks", "renew", "--workspace", p.Target.Workspace, taskID, "--owner", p.Target.Owner, "--lease", lease},
		Reason:     "Keep the existing ADP task claim alive before continuing.",
	})
}

func (p *Plan) addRunCommand(taskID string) {
	if p.Target.Workspace == "" || p.Target.Agent == "" || taskID == "" {
		return
	}
	args := p.runArgs()
	args = append(args, "--task", taskID)
	args = p.appendRuntimeOptions(args)
	p.SuggestedCommands = append(p.SuggestedCommands, Command{
		Label:      "launch-resumed-worker",
		SideEffect: CommandSideEffectRuntimeCreation,
		Args:       args,
		Reason:     "Launch a new ADP runtime bound to the same task context.",
	})
}

func (p *Plan) addWorkspaceRunCommand() {
	if p.Target.Workspace == "" || p.Target.Agent == "" {
		return
	}
	args := p.appendRuntimeOptions(p.runArgs())
	p.SuggestedCommands = append(p.SuggestedCommands, Command{
		Label:      "launch-workspace-worker",
		SideEffect: CommandSideEffectRuntimeCreation,
		Args:       args,
		Reason:     "Launch only workspace context because no ADP task was bound to the source session.",
	})
}

func (p Plan) runArgs() []string {
	args := []string{"adp", "run", p.Target.Agent, "--workspace", p.Target.Workspace}
	if p.Target.Profile != "" {
		args = append(args, "--profile", p.Target.Profile)
	}
	return args
}

func (p Plan) appendRuntimeOptions(args []string) []string {
	if p.Invocation != nil && p.Invocation.KeepRuntime {
		args = append(args, "--keep-runtime")
	}
	if p.Invocation != nil && p.sameLaunchContext() && len(p.Invocation.AgentArgs) > 0 {
		args = append(args, "--")
		args = append(args, p.Invocation.AgentArgs...)
	}
	return args
}
