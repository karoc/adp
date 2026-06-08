package sessions

import (
	"fmt"
	"strconv"

	"github.com/karoc/adp/internal/events"
)

const (
	RestorePlanStatusReady   = "ready"
	RestorePlanStatusPartial = "partial"

	invocationSchemaVersion = 1
)

type RestorePlan struct {
	SessionID        string   `json:"session_id"`
	Status           string   `json:"status"`
	MissingFields    []string `json:"missing_fields,omitempty"`
	Reasons          []string `json:"reasons,omitempty"`
	SuggestedCommand []string `json:"suggested_command,omitempty"`
}

func BuildRestorePlan(detail *Detail) RestorePlan {
	if detail == nil {
		return RestorePlan{
			Status:        RestorePlanStatusPartial,
			MissingFields: []string{"session"},
			Reasons:       []string{"session detail is missing"},
		}
	}

	plan := RestorePlan{
		SessionID: detail.Summary.SessionID,
		Status:    RestorePlanStatusReady,
		Reasons: []string{
			"restore-plan is read-only and does not resume provider-native conversations or execute the suggested command",
		},
	}

	command := []string{"adp", "run"}
	hasAgent := false
	if detail.Summary.Agent == "" {
		plan.addMissing("agent", "session has no agent name")
	} else {
		hasAgent = true
		command = append(command, detail.Summary.Agent)
	}
	hasWorkspace := false
	if detail.Summary.Workspace == "" {
		plan.addMissing("workspace", "session has no workspace name")
	} else {
		hasWorkspace = true
		command = append(command, "--workspace", detail.Summary.Workspace)
	}
	if detail.Summary.Profile != "" && detail.Summary.Profile != "default" {
		command = append(command, "--profile", detail.Summary.Profile)
	}
	if detail.Summary.TaskID != "" {
		command = append(command, "--task", detail.Summary.TaskID)
	}

	invocation, ok := findInvocation(detail.Events)
	if !ok {
		plan.addMissing("fields.invocation", "session was recorded before invocation snapshots were available")
	} else {
		applyInvocation(&plan, &command, invocation)
	}

	if hasAgent && hasWorkspace {
		plan.SuggestedCommand = command
	}
	return plan
}

func applyInvocation(plan *RestorePlan, command *[]string, invocation map[string]any) {
	version, ok := intField(invocation, "schema_version")
	if !ok {
		plan.addMissing("fields.invocation.schema_version", "invocation snapshot has no schema version")
	} else if version != invocationSchemaVersion {
		plan.addMissing("fields.invocation.schema_version", fmt.Sprintf("unsupported invocation schema version %d", version))
	}

	keepRuntime, ok := boolField(invocation, "keep_runtime")
	if !ok {
		plan.addMissing("fields.invocation.keep_runtime", "invocation snapshot does not record keep-runtime")
	} else if keepRuntime {
		*command = append(*command, "--keep-runtime")
	}

	agentArgs, ok := stringSliceField(invocation, "agent_args")
	if !ok {
		plan.addMissing("fields.invocation.agent_args", "invocation snapshot does not record agent arguments")
		return
	}
	if len(agentArgs) > 0 {
		*command = append(*command, "--")
		*command = append(*command, agentArgs...)
	}
}

func (p *RestorePlan) addMissing(field string, reason string) {
	p.Status = RestorePlanStatusPartial
	p.MissingFields = append(p.MissingFields, field)
	p.Reasons = append(p.Reasons, reason)
}

func findInvocation(read []events.Event) (map[string]any, bool) {
	for _, event := range read {
		if event.Type != eventTypeRunStarted || len(event.Fields) == 0 {
			continue
		}
		value, ok := event.Fields["invocation"]
		if !ok {
			continue
		}
		invocation, ok := value.(map[string]any)
		return invocation, ok
	}
	return nil, false
}

func intField(fields map[string]any, key string) (int, bool) {
	value, ok := fields[key]
	if !ok {
		return 0, false
	}
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		if typed != float64(int(typed)) {
			return 0, false
		}
		return int(typed), true
	case jsonNumber:
		parsed, err := strconv.Atoi(typed.String())
		return parsed, err == nil
	default:
		return 0, false
	}
}

func boolField(fields map[string]any, key string) (bool, bool) {
	value, ok := fields[key]
	if !ok {
		return false, false
	}
	typed, ok := value.(bool)
	return typed, ok
}

func stringSliceField(fields map[string]any, key string) ([]string, bool) {
	value, ok := fields[key]
	if !ok {
		return nil, false
	}
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...), true
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if !ok {
				return nil, false
			}
			values = append(values, text)
		}
		return values, true
	default:
		return nil, false
	}
}

type jsonNumber interface {
	String() string
}
