package resume

import (
	"fmt"
	"strings"
	"time"

	"github.com/karoc/adp/internal/sessions"
	taskstore "github.com/karoc/adp/internal/tasks"
)

func BuildPlan(req Request) Plan {
	now := req.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}

	plan := Plan{
		SchemaVersion: SchemaVersion,
		Status:        StatusReady,
		Guarantees: Guarantees{
			ReadOnly: true,
		},
		Guidance: []string{
			"ADP resumes portable work context; provider-native conversation resume is optional and tool-specific.",
			"ADP task, phase, progress, and session evidence remain the source of truth.",
			"Suggested commands are not executed by this plan.",
		},
	}

	if req.Detail == nil {
		plan.Status = StatusPartial
		plan.Summary = "Session detail is unavailable; inspect ADP session history before resuming work."
		plan.addMissing("session", "session detail is missing")
		return plan
	}

	summary := req.Detail.Summary
	plan.SessionID = summary.SessionID
	plan.Source = Source{
		Workspace:      summary.Workspace,
		Agent:          summary.Agent,
		Profile:        summary.Profile,
		TaskID:         summary.TaskID,
		ProjectRoot:    summary.ProjectRoot,
		RuntimePath:    summary.RuntimePath,
		StartedAt:      timeString(summary.StartedAt),
		FinishedAt:     timeString(summary.FinishedAt),
		ExitCode:       summary.ExitCode,
		DurationMillis: summary.DurationMillis,
		EventCount:     summary.EventCount,
	}
	plan.Target = Target{
		Workspace: firstNonEmpty(req.Workspace, summary.Workspace),
		Agent:     firstNonEmpty(req.TargetAgent, summary.Agent),
		Owner:     strings.TrimSpace(req.Owner),
		Lease:     durationString(req.Lease),
	}
	plan.applySummaryProfile()
	plan.applyWorkspaceOverrideGuidance()
	plan.applyInvocation(req.Detail)
	plan.addInspectionCommands()

	if plan.Source.Workspace == "" && plan.Target.Workspace == "" {
		plan.addMissing("workspace", "session has no workspace and no --workspace override was provided")
	}
	if plan.Source.Agent == "" && plan.Target.Agent == "" {
		plan.addMissing("agent", "session has no source agent and no --agent override was provided")
	}
	if summary.TaskID == "" {
		plan.Status = StatusPartial
		plan.Summary = "This session is resumable only at workspace level because no ADP task was bound."
		plan.addMissing("task", "session has no ADP task binding")
		plan.addWorkspaceRunCommand()
		plan.applyPhase(req.PhaseGate, req.PhaseLoadError)
		return plan
	}

	if req.TaskLoadError != "" {
		plan.Status = StatusPartial
		plan.Summary = "Task-bound context exists, but the current task ledger could not be read."
		plan.addMissing("task.current", req.TaskLoadError)
		plan.addTaskInspectCommand(summary.TaskID)
		plan.applyPhase(req.PhaseGate, req.PhaseLoadError)
		return plan
	}
	if req.Task == nil {
		plan.Status = StatusPartial
		plan.Summary = "Task-bound context exists, but current task state is unavailable."
		plan.addMissing("task.current", "current task state is unavailable")
		plan.addTaskInspectCommand(summary.TaskID)
		plan.applyPhase(req.PhaseGate, req.PhaseLoadError)
		return plan
	}

	plan.Task = taskState(*req.Task, now)
	plan.applyTaskGuidance(*req.Task, now)
	plan.applyPhase(req.PhaseGate, req.PhaseLoadError)
	return plan
}

func (p *Plan) applySummaryProfile() {
	if p.sameLaunchContext() && p.Source.Profile != "" && p.Source.Profile != "default" {
		p.Target.Profile = p.Source.Profile
	}
}

func (p *Plan) applyWorkspaceOverrideGuidance() {
	if p.Source.Workspace == "" || p.Target.Workspace == "" || p.Source.Workspace == p.Target.Workspace {
		return
	}
	reason := fmt.Sprintf("Workspace override is active; current task and phase state were read from %s, while the source session was recorded in %s.", p.Target.Workspace, p.Source.Workspace)
	p.Guidance = append(p.Guidance, reason)
	p.Reasons = append(p.Reasons, reason)
}

func (p *Plan) applyInvocation(detail *sessions.Detail) {
	if detail == nil {
		return
	}
	snapshot, issues, ok := sessions.ExtractInvocationSnapshot(detail.Events)
	if !ok {
		p.addContextGap("fields.invocation", "session was recorded before invocation snapshots were available")
		return
	}
	p.Invocation = &Invocation{
		Available:     true,
		SchemaVersion: snapshot.SchemaVersion,
		KeepRuntime:   snapshot.KeepRuntime,
		AgentArgs:     snapshot.AgentArgs,
	}
	for _, issue := range issues {
		p.addContextGap(issue.Field, issue.Reason)
	}
	if snapshot.KeepRuntime {
		p.Invocation.Reused = append(p.Invocation.Reused, "keep_runtime")
	}
	if p.sameLaunchContext() {
		if p.Target.Profile != "" {
			p.Invocation.Reused = append(p.Invocation.Reused, "profile")
		}
		if len(snapshot.AgentArgs) > 0 {
			p.Invocation.Reused = append(p.Invocation.Reused, "agent_args")
		}
		return
	}
	var omitted []string
	if p.Source.Profile != "" && p.Source.Profile != "default" {
		omitted = append(omitted, "profile")
	}
	if len(snapshot.AgentArgs) > 0 {
		omitted = append(omitted, "agent_args")
	}
	if len(omitted) == 0 {
		return
	}
	reason := "Target agent or workspace differs from the source session; provider-specific profile or agent arguments were not copied into the suggested launch command."
	p.Invocation.Omitted = omitted
	p.Invocation.OmissionReason = reason
	p.Guidance = append(p.Guidance, reason)
	p.Reasons = append(p.Reasons, reason)
}

func taskState(task taskstore.Task, now time.Time) *TaskState {
	return &TaskState{
		ID:             task.ID,
		Title:          task.Title,
		Status:         string(task.Status),
		Priority:       task.Priority,
		Phase:          task.Phase,
		Owner:          task.Owner,
		ClaimState:     ClaimState(task, now),
		ClaimedAt:      timeString(task.ClaimedAt),
		LeaseExpiresAt: timeString(task.LeaseExpiresAt),
		BlockedReason:  task.BlockedReason,
	}
}

func (p *Plan) applyTaskGuidance(task taskstore.Task, now time.Time) {
	state := ClaimState(task, now)
	action, status, reason := classifyTask(task, state, p.Target.Owner)
	p.Status = mergeStatus(p.Status, status)
	p.Task.ResumeAction = action
	p.Task.Reason = reason
	p.Summary = reason
	p.Guidance = append(p.Guidance, reason)

	switch action {
	case ActionClaim:
		if state == ClaimStateStale {
			p.addStaleCommand()
		}
		p.addClaimCommand(task.ID)
		p.addRunCommand(task.ID)
	case ActionRenew:
		p.addRenewCommand(task.ID)
		p.addRunCommand(task.ID)
	case ActionRun:
		p.addRunCommand(task.ID)
	case ActionWait:
		p.addTaskInspectCommand(task.ID)
		p.addStaleCommand()
	case ActionResolveBlocker, ActionCreateTask:
		p.addTaskInspectCommand(task.ID)
	}
}

func classifyTask(task taskstore.Task, claimState string, targetOwner string) (string, string, string) {
	switch task.Status {
	case taskstore.StatusDone, taskstore.StatusCanceled:
		return ActionCreateTask, StatusClosed, fmt.Sprintf("Task %s is %s; create or choose follow-up work instead of resuming it.", task.ID, task.Status)
	case taskstore.StatusBlocked:
		if task.BlockedReason != "" {
			return ActionResolveBlocker, StatusBlocked, fmt.Sprintf("Task %s is blocked: %s", task.ID, task.BlockedReason)
		}
		return ActionResolveBlocker, StatusBlocked, fmt.Sprintf("Task %s is blocked; resolve the blocker before continuing.", task.ID)
	}

	if task.Owner == "" {
		return ActionClaim, ownerStatus(targetOwner), fmt.Sprintf("Task %s is not owned; claim it before launching a resumed worker.", task.ID)
	}
	if targetOwner == "" {
		if claimState == ClaimStateStale {
			return ActionClaim, StatusPartial, fmt.Sprintf("Task %s has a stale claim by %s; choose an owner before reclaiming.", task.ID, task.Owner)
		}
		return ActionWait, StatusBlocked, fmt.Sprintf("Task %s is already owned by %s; pass --owner only if that owner is resuming.", task.ID, task.Owner)
	}
	if task.Owner == targetOwner {
		if claimState == ClaimStateStale {
			return ActionRenew, StatusReady, fmt.Sprintf("Task %s is owned by %s but the lease is stale; renew the lease before continuing.", task.ID, targetOwner)
		}
		return ActionRun, StatusReady, fmt.Sprintf("Task %s is owned by %s; continue with a new ADP runtime.", task.ID, targetOwner)
	}
	if claimState == ClaimStateStale {
		return ActionClaim, StatusReady, fmt.Sprintf("Task %s has a stale claim by %s; %s can reclaim it explicitly.", task.ID, task.Owner, targetOwner)
	}
	return ActionWait, StatusBlocked, fmt.Sprintf("Task %s is still owned by %s; wait, coordinate, or inspect stale work before reclaiming.", task.ID, task.Owner)
}

func (p *Plan) applyPhase(gate *taskstore.PhaseGate, errText string) {
	if errText != "" {
		p.addContextGap("phase.current", errText)
		return
	}
	if gate == nil {
		p.addContextGap("phase.current", "phase gate state is unavailable")
		return
	}
	state := PhaseState{
		PhaseCount:   gate.PhaseCount,
		CanStartNext: gate.CanStartNext,
		NextAction:   gate.NextAction,
		Reason:       gate.Reason,
	}
	if gate.OpenPhase != nil {
		state.OpenPhaseID = gate.OpenPhase.ID
		state.OpenPhaseStatus = string(gate.OpenPhase.Status)
	}
	if gate.NextPlannedPhase != nil {
		state.NextPlannedPhaseID = gate.NextPlannedPhase.ID
	}
	p.Phase = &state
	if gate.NextAction != "" {
		p.Guidance = append(p.Guidance, "Phase gate next action: "+gate.NextAction+". "+gate.Reason)
	}
}

func ClaimState(task taskstore.Task, now time.Time) string {
	if strings.TrimSpace(task.Owner) == "" {
		return ClaimStateUnclaimed
	}
	if task.LeaseExpiresAt.IsZero() {
		return ClaimStateClaimed
	}
	if task.LeaseExpiresAt.After(now.UTC()) {
		return ClaimStateLeased
	}
	return ClaimStateStale
}

func ownerStatus(owner string) string {
	if strings.TrimSpace(owner) == "" {
		return StatusPartial
	}
	return StatusReady
}

func mergeStatus(current string, next string) string {
	if current == StatusPartial && next == StatusReady {
		return current
	}
	return next
}

func (p *Plan) addMissing(field string, reason string) {
	p.Status = StatusPartial
	p.MissingFields = append(p.MissingFields, field)
	p.Reasons = append(p.Reasons, reason)
	p.Guidance = append(p.Guidance, reason)
}

func (p *Plan) addContextGap(field string, reason string) {
	p.MissingFields = append(p.MissingFields, field)
	p.Reasons = append(p.Reasons, reason)
	p.Guidance = append(p.Guidance, reason)
}

func timeString(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func durationString(value time.Duration) string {
	if value == 0 {
		return ""
	}
	if value%time.Hour == 0 {
		return fmt.Sprintf("%dh", int64(value/time.Hour))
	}
	if value%time.Minute == 0 && value < time.Hour {
		return fmt.Sprintf("%dm", int64(value/time.Minute))
	}
	if value%time.Second == 0 && value < time.Minute {
		return fmt.Sprintf("%ds", int64(value/time.Second))
	}
	return strings.ReplaceAll(strings.ReplaceAll(value.String(), "h0m", "h"), "m0s", "m")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (p Plan) sameLaunchContext() bool {
	return strings.TrimSpace(p.Source.Agent) != "" &&
		strings.EqualFold(strings.TrimSpace(p.Source.Agent), strings.TrimSpace(p.Target.Agent)) &&
		strings.TrimSpace(p.Source.Workspace) != "" &&
		strings.TrimSpace(p.Source.Workspace) == strings.TrimSpace(p.Target.Workspace)
}
