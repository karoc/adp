package sessions

import (
	"fmt"
	"strconv"

	"github.com/karoc/adp/internal/events"
)

const InvocationSchemaVersion = 1

type InvocationSnapshot struct {
	SchemaVersion int
	KeepRuntime   bool
	AgentArgs     []string
}

type InvocationIssue struct {
	Field  string
	Reason string
}

func ExtractInvocationSnapshot(read []events.Event) (InvocationSnapshot, []InvocationIssue, bool) {
	invocation, ok := findInvocation(read)
	if !ok {
		return InvocationSnapshot{}, nil, false
	}

	var snapshot InvocationSnapshot
	var issues []InvocationIssue
	version, ok := intField(invocation, "schema_version")
	if !ok {
		issues = append(issues, InvocationIssue{
			Field:  "fields.invocation.schema_version",
			Reason: "invocation snapshot has no schema version",
		})
	} else {
		snapshot.SchemaVersion = version
		if version != InvocationSchemaVersion {
			issues = append(issues, InvocationIssue{
				Field:  "fields.invocation.schema_version",
				Reason: fmt.Sprintf("unsupported invocation schema version %d", version),
			})
		}
	}

	keepRuntime, ok := boolField(invocation, "keep_runtime")
	if !ok {
		issues = append(issues, InvocationIssue{
			Field:  "fields.invocation.keep_runtime",
			Reason: "invocation snapshot does not record keep-runtime",
		})
	} else {
		snapshot.KeepRuntime = keepRuntime
	}

	agentArgs, ok := stringSliceField(invocation, "agent_args")
	if !ok {
		issues = append(issues, InvocationIssue{
			Field:  "fields.invocation.agent_args",
			Reason: "invocation snapshot does not record agent arguments",
		})
	} else {
		snapshot.AgentArgs = agentArgs
	}
	return snapshot, issues, true
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
