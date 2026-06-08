package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	taskstore "github.com/karoc/adp/internal/tasks"
)

func TestTasksNextCommandPrintsPrioritizedWork(t *testing.T) {
	low := testTask("task-low", "Low priority", taskstore.StatusReady)
	low.Priority = "low"
	done := testTask("task-done", "Done critical", taskstore.StatusDone)
	done.Priority = "critical"
	high := testTask("task-high", "High priority", taskstore.StatusReady)
	high.Priority = "high"
	review := testTask("task-review", "Review medium", taskstore.StatusReview)
	review.Priority = "medium"
	store := &fakeTaskStore{tasks: []taskstore.Task{low, done, high, review}}
	var textOut bytes.Buffer
	var jsonOut bytes.Buffer
	var jsonErr bytes.Buffer
	deps := Dependencies{
		WorkspaceStore:   &fakeStore{cfg: testConfig()},
		TaskStoreFactory: func(string) TaskStore { return store },
	}

	textCode := NewApp(deps, &textOut, &bytes.Buffer{}).Execute(context.Background(), []string{"tasks", "next", "--workspace", "game-a", "--limit", "2"})
	jsonCode := NewApp(deps, &jsonOut, &jsonErr).Execute(context.Background(), []string{"tasks", "next", "--workspace", "game-a", "--limit", "0", "--format", "json"})

	if textCode != 0 {
		t.Fatalf("tasks next text exit code = %d, output = %q", textCode, textOut.String())
	}
	if jsonCode != 0 {
		t.Fatalf("tasks next json exit code = %d, stderr = %q", jsonCode, jsonErr.String())
	}
	for _, want := range []string{"workspace: game-a", "limit: 2", "task-high", "task-review"} {
		if !strings.Contains(textOut.String(), want) {
			t.Fatalf("tasks next text missing %q: %q", want, textOut.String())
		}
	}
	if strings.Contains(textOut.String(), "task-low") || strings.Contains(textOut.String(), "task-done") {
		t.Fatalf("tasks next text included out-of-limit or terminal task: %q", textOut.String())
	}

	payload := decodeJSONObject(t, jsonOut.Bytes())
	assertJSONStringField(t, payload, "workspace", "game-a")
	assertJSONStringField(t, payload, "planning_source", "/tmp/adp-home/workspaces/game-a/planning/tasks.yaml")
	if _, ok := payload["generated_at"].(string); !ok {
		t.Fatalf("generated_at = %#v, want string", payload["generated_at"])
	}
	assertJSONNumberField(t, payload, "total", 4)
	assertJSONNumberField(t, payload, "eligible_count", 3)
	assertJSONNumberField(t, payload, "limit", 0)
	counts := assertJSONObjectField(t, payload, "counts")
	assertJSONNumberField(t, counts, "ready", 2)
	assertJSONNumberField(t, counts, "review", 1)
	assertJSONNumberField(t, counts, "done", 1)
	candidates := assertJSONObjectListField(t, payload, "candidates")
	if len(candidates) != 3 {
		t.Fatalf("json candidates length = %d, want 3: %+v", len(candidates), candidates)
	}
	assertJSONStringField(t, candidates[0], "id", "task-high")
	assertJSONStringField(t, candidates[1], "id", "task-review")
	assertJSONStringField(t, candidates[2], "id", "task-low")
	next := assertJSONObjectField(t, payload, "next")
	assertJSONStringField(t, next, "id", "task-high")
}

func TestTasksNextCommandHandlesNoEligibleWorkAndInvalidArgs(t *testing.T) {
	done := testTask("task-done", "Done task", taskstore.StatusDone)
	blocked := testTask("task-blocked", "Blocked task", taskstore.StatusBlocked)
	store := &fakeTaskStore{tasks: []taskstore.Task{done, blocked}}
	var textOut bytes.Buffer
	var jsonOut bytes.Buffer
	var jsonErr bytes.Buffer
	var invalidLimitErr bytes.Buffer
	var positionalErr bytes.Buffer
	deps := Dependencies{
		WorkspaceStore:   &fakeStore{cfg: testConfig()},
		TaskStoreFactory: func(string) TaskStore { return store },
	}

	textCode := NewApp(deps, &textOut, &bytes.Buffer{}).Execute(context.Background(), []string{"tasks", "next", "--workspace", "game-a"})
	jsonCode := NewApp(deps, &jsonOut, &jsonErr).Execute(context.Background(), []string{"tasks", "next", "--workspace", "game-a", "--format", "json"})
	invalidLimitCode := NewApp(deps, &bytes.Buffer{}, &invalidLimitErr).Execute(context.Background(), []string{"tasks", "next", "--workspace", "game-a", "--limit", "-1"})
	positionalCode := NewApp(deps, &bytes.Buffer{}, &positionalErr).Execute(context.Background(), []string{"tasks", "next", "--workspace", "game-a", "task-1"})

	if textCode != 0 {
		t.Fatalf("tasks next text exit code = %d, output = %q", textCode, textOut.String())
	}
	if !strings.Contains(textOut.String(), "next: -") {
		t.Fatalf("tasks next text = %q", textOut.String())
	}
	if jsonCode != 0 {
		t.Fatalf("tasks next json exit code = %d, stderr = %q", jsonCode, jsonErr.String())
	}
	payload := decodeJSONObject(t, jsonOut.Bytes())
	assertJSONNumberField(t, payload, "total", 2)
	assertJSONNumberField(t, payload, "eligible_count", 0)
	if candidates := assertJSONObjectListField(t, payload, "candidates"); len(candidates) != 0 {
		t.Fatalf("candidates = %+v, want empty", candidates)
	}
	if _, ok := payload["next"]; ok {
		t.Fatalf("next field present for no eligible work: %+v", payload)
	}
	if invalidLimitCode != 1 || !strings.Contains(invalidLimitErr.String(), "limit must not be negative") {
		t.Fatalf("invalid limit code/stderr = %d/%q", invalidLimitCode, invalidLimitErr.String())
	}
	if positionalCode != 1 || !strings.Contains(positionalErr.String(), "usage: "+tasksNextUsage) {
		t.Fatalf("positional code/stderr = %d/%q", positionalCode, positionalErr.String())
	}
}
