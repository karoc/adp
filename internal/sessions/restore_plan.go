package sessions

const (
	RestorePlanStatusReady   = "ready"
	RestorePlanStatusPartial = "partial"
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

	invocation, issues, ok := ExtractInvocationSnapshot(detail.Events)
	if !ok {
		plan.addMissing("fields.invocation", "session was recorded before invocation snapshots were available")
	} else {
		applyInvocation(&plan, &command, invocation, issues)
	}

	if hasAgent && hasWorkspace {
		plan.SuggestedCommand = command
	}
	return plan
}

func applyInvocation(plan *RestorePlan, command *[]string, invocation InvocationSnapshot, issues []InvocationIssue) {
	for _, issue := range issues {
		plan.addMissing(issue.Field, issue.Reason)
	}
	if invocation.KeepRuntime {
		*command = append(*command, "--keep-runtime")
	}
	if len(invocation.AgentArgs) > 0 {
		*command = append(*command, "--")
		*command = append(*command, invocation.AgentArgs...)
	}
}

func (p *RestorePlan) addMissing(field string, reason string) {
	p.Status = RestorePlanStatusPartial
	p.MissingFields = append(p.MissingFields, field)
	p.Reasons = append(p.Reasons, reason)
}
