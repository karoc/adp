package resume

import (
	"strings"

	"github.com/karoc/adp/internal/redact"
)

const ReplayModeDryRun = "dry_run"

type ReplayDryRun struct {
	SchemaVersion          int        `json:"schema_version"`
	SourceSessionID        string     `json:"source_session_id,omitempty"`
	Mode                   string     `json:"mode"`
	Status                 string     `json:"status"`
	PlanStatus             string     `json:"plan_status,omitempty"`
	Summary                string     `json:"summary"`
	Source                 Source     `json:"source"`
	Target                 Target     `json:"target"`
	Task                   *TaskState `json:"task,omitempty"`
	TaskPreflight          string     `json:"task_preflight"`
	LaunchCommand          []string   `json:"launch_command,omitempty"`
	RequiredCommands       []Command  `json:"required_commands,omitempty"`
	Blockers               []string   `json:"blockers,omitempty"`
	ExecutedCommands       []Command  `json:"executed_commands"`
	NewSessionID           string     `json:"new_session_id,omitempty"`
	ReadOnly               bool       `json:"read_only"`
	WouldMutateTask        bool       `json:"would_mutate_task"`
	WouldCreateRuntime     bool       `json:"would_create_runtime"`
	ProviderNativeResume   bool       `json:"provider_native_resume"`
	GitSideEffects         bool       `json:"git_side_effects"`
	ProjectRootWritesByADP bool       `json:"project_root_writes_by_adp"`
	Guarantees             Guarantees `json:"guarantees"`
	Guidance               []string   `json:"guidance"`
}

func BuildReplayDryRun(plan Plan) ReplayDryRun {
	out := ReplayDryRun{
		SchemaVersion:    SchemaVersion,
		SourceSessionID:  plan.SessionID,
		Mode:             ReplayModeDryRun,
		Status:           StatusReady,
		PlanStatus:       plan.Status,
		Source:           plan.Source,
		Target:           plan.Target,
		Task:             plan.Task,
		TaskPreflight:    replayTaskPreflight(plan),
		ExecutedCommands: []Command{},
		ReadOnly:         true,
		Guarantees: Guarantees{
			ReadOnly: true,
		},
		Guidance: append([]string{
			"Replay dry-run is read-only and does not launch an agent.",
			"Execute mode is deferred to a later accepted phase.",
			"Provider-native conversation resume is not part of ADP local replay.",
		}, plan.Guidance...),
	}

	launch, hasLaunch := runtimeCreationCommand(plan)
	if hasLaunch {
		out.LaunchCommand = append([]string(nil), launch.Args...)
	}
	out.RequiredCommands = taskMutationCommands(plan)

	var blockers []string
	if plan.SessionID == "" {
		blockers = append(blockers, "source session ID is unavailable")
	}
	if plan.Status != StatusReady {
		blockers = append(blockers, "resume plan status is "+plan.Status+": "+plan.Summary)
	}
	if strings.TrimSpace(plan.Source.Workspace) == "" || strings.TrimSpace(plan.Target.Workspace) == "" {
		blockers = append(blockers, "source and target workspace are required for replay")
	}
	if strings.TrimSpace(plan.Source.Workspace) != "" &&
		strings.TrimSpace(plan.Target.Workspace) != "" &&
		strings.TrimSpace(plan.Source.Workspace) != strings.TrimSpace(plan.Target.Workspace) {
		blockers = append(blockers, "cross-workspace replay is deferred; use resume-plan guidance and an explicit adp run command")
	}
	if strings.TrimSpace(plan.Target.Agent) == "" {
		blockers = append(blockers, "target agent is required for replay")
	}
	if strings.TrimSpace(plan.Source.TaskID) == "" || plan.Task == nil {
		blockers = append(blockers, "local replay MVP requires a task-bound source session; workspace-only replay is deferred")
	}
	if plan.Invocation == nil || !plan.Invocation.Available {
		blockers = append(blockers, "invocation snapshot is unavailable; run an explicit adp run command instead")
	} else {
		if containsRedaction(plan.Invocation.AgentArgs) {
			blockers = append(blockers, "invocation data contains redacted agent arguments; run an explicit adp run command with replacement values")
		}
		for _, field := range plan.MissingFields {
			if strings.HasPrefix(field, "fields.invocation") {
				blockers = append(blockers, "invocation snapshot is incomplete: "+field)
			}
		}
	}
	if plan.Task != nil && plan.Task.ResumeAction != ActionRun {
		blockers = append(blockers, "task preflight requires explicit ADP ownership action before replay: "+plan.Task.ResumeAction)
	}
	if !hasLaunch {
		blockers = append(blockers, "resume plan has no runtime_creation launch command")
	}

	out.Blockers = blockers
	if len(blockers) == 0 {
		out.Status = StatusReady
		out.Summary = "Replay dry-run is ready; it would create a new ADP runtime only if a future execute mode is explicitly requested."
		out.WouldCreateRuntime = true
		return out
	}

	out.Status = replayBlockedStatus(plan.Status)
	out.Summary = "Replay dry-run is blocked; review blockers and required commands before launching work explicitly."
	return out
}

func replayBlockedStatus(planStatus string) string {
	switch planStatus {
	case StatusPartial, StatusBlocked, StatusClosed:
		return planStatus
	default:
		return StatusBlocked
	}
}

func replayTaskPreflight(plan Plan) string {
	if plan.Task == nil {
		return "workspace-only replay is deferred because no task binding is available"
	}
	switch plan.Task.ResumeAction {
	case ActionRun:
		if plan.Task.ClaimState == ClaimStateLeased {
			return "task is owned by " + valueOrUnknown(plan.Task.Owner) + " and lease is valid"
		}
		return "task is owned by " + valueOrUnknown(plan.Task.Owner) + " and can be launched without task mutation"
	case ActionClaim:
		return "task must be explicitly claimed before replay"
	case ActionRenew:
		return "task lease must be explicitly renewed before replay"
	case ActionWait:
		return "task is owned by another active worker; coordinate before replay"
	case ActionResolveBlocker:
		return "task blocker must be resolved before replay"
	case ActionCreateTask:
		return "task is closed; create or choose follow-up work before replay"
	default:
		return "task preflight is unavailable"
	}
}

func valueOrUnknown(value string) string {
	if strings.TrimSpace(value) == "" {
		return "<unknown>"
	}
	return strings.TrimSpace(value)
}

func runtimeCreationCommand(plan Plan) (Command, bool) {
	for _, command := range plan.SuggestedCommands {
		if command.SideEffect == CommandSideEffectRuntimeCreation {
			return command, true
		}
	}
	return Command{}, false
}

func taskMutationCommands(plan Plan) []Command {
	var commands []Command
	for _, command := range plan.SuggestedCommands {
		if command.SideEffect == CommandSideEffectTaskMutation {
			commands = append(commands, command)
		}
	}
	return commands
}

func containsRedaction(values []string) bool {
	for _, value := range values {
		if strings.Contains(value, redact.Placeholder) {
			return true
		}
	}
	return false
}
