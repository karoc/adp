package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func field(root map[string]any, names ...string) any {
	for _, name := range names {
		if value, ok := root[name]; ok {
			return value
		}
	}
	return nil
}

func stringField(root map[string]any, names ...string) string {
	value, _ := field(root, names...).(string)
	return value
}

func numberField(root map[string]any, names ...string) int {
	number, ok := field(root, names...).(float64)
	if !ok {
		fail("missing numeric field %s", strings.Join(names, " or "))
	}
	return int(number)
}

func objectField(root map[string]any, names ...string) map[string]any {
	out, ok := field(root, names...).(map[string]any)
	if !ok {
		fail("missing object field %s", strings.Join(names, " or "))
	}
	return out
}

func arrayField(root map[string]any, names ...string) []any {
	out, ok := field(root, names...).([]any)
	if !ok {
		fail("missing array field %s", strings.Join(names, " or "))
	}
	return out
}

func text(value any) string {
	data, _ := json.Marshal(value)
	return string(data)
}

func contains(value any, want string) bool {
	return strings.Contains(text(value), want)
}

func requireContains(label string, value any, want string) {
	if !contains(value, want) {
		fail("%s missing %q", label, want)
	}
}

func itemIndex(items []any, want string) int {
	for i, item := range items {
		if contains(item, want) {
			return i
		}
	}
	return -1
}

func requireItem(label string, items []any, want string) any {
	index := itemIndex(items, want)
	if index < 0 {
		fail("%s missing item containing %q", label, want)
	}
	return items[index]
}

func requireObjectByID(label string, items []any, id string) map[string]any {
	for _, item := range items {
		object, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if stringField(object, "id") == id {
			return object
		}
	}
	fail("%s missing object with id %q", label, id)
	return nil
}

func requireStringField(label string, object map[string]any, field string, want string) {
	if got := stringField(object, field); got != want {
		fail("%s field %s = %q, want %q", label, field, got, want)
	}
}

func requireCountAtLeast(counts map[string]any, status string, want int) {
	number, ok := counts[status].(float64)
	if !ok || int(number) < want {
		fail("task count %s = %v, want at least %d", status, counts[status], want)
	}
}

func main() {
	if len(os.Args) != 7 {
		fail("usage: progress-report-json-assert <file> <done-task> <critical-task> <low-task> <session> <runtime-dir>")
	}
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fail("read JSON report: %v", err)
	}
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		fail("progress report is not parseable JSON: %v", err)
	}
	if stringField(root, "workspace") != "game-a" {
		fail("workspace = %q, want game-a", stringField(root, "workspace"))
	}
	if total := numberField(root, "total_tasks", "total"); total < 3 {
		fail("total task count = %d, want at least 3", total)
	}
	counts := objectField(root, "task_counts", "counts")
	requireCountAtLeast(counts, "done", 1)
	requireCountAtLeast(counts, "ready", 2)
	phases := arrayField(root, "phases")
	p3 := requireItem("phases", phases, "p3")
	requireContains("phase p3", p3, "pushed")
	tasks := arrayField(root, "tasks")
	doneTask := requireObjectByID("tasks", tasks, os.Args[2])
	criticalTask := requireObjectByID("tasks", tasks, os.Args[3])
	lowTask := requireObjectByID("tasks", tasks, os.Args[4])
	requireStringField("done task", doneTask, "claim_state", "leased")
	requireStringField("critical task", criticalTask, "claim_state", "unclaimed")
	requireStringField("low task", lowTask, "claim_state", "unclaimed")
	next := arrayField(root, "next_work", "next")
	criticalIndex := itemIndex(next, os.Args[3])
	lowIndex := itemIndex(next, os.Args[4])
	if criticalIndex < 0 || lowIndex < 0 {
		fail("next work missing critical or low priority task")
	}
	requireStringField("critical next work", requireObjectByID("next work", next, os.Args[3]), "claim_state", "unclaimed")
	requireStringField("low next work", requireObjectByID("next work", next, os.Args[4]), "claim_state", "unclaimed")
	if criticalIndex > lowIndex {
		fail("next work is not priority sorted: critical index %d, low index %d", criticalIndex, lowIndex)
	}
	if itemIndex(next, os.Args[2]) >= 0 {
		fail("next work included completed task %s", os.Args[2])
	}
	evidence := field(root, "phase_evidence", "phaseEvidence", "evidence")
	if evidence == nil {
		evidence = p3
	}
	for _, want := range []string{"scripts/task-manager-smoke.sh", "abc123", "origin", "main", "pushed"} {
		requireContains("phase evidence", evidence, want)
	}
	sessions := arrayField(root, "runtime_sessions", "runtimeSessions", "sessions")
	session := requireItem("runtime sessions", sessions, os.Args[5])
	for _, want := range []string{"codex", os.Args[2], os.Args[6]} {
		requireContains("runtime session", session, want)
	}
	if !contains(session, "run_finished") && !contains(session, "exit_code") && !contains(session, "0") {
		fail("runtime session missing finished or exit evidence")
	}
	lower := strings.ToLower(string(data))
	for _, bad := range []string{"dashboard_url", "issue_url", "tracker_url", "http://", "https://", "hosted"} {
		if strings.Contains(lower, bad) {
			fail("JSON report contains hosted tracker drift token %q", bad)
		}
	}
}
