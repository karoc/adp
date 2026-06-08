package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	taskstore "github.com/karoc/adp/internal/tasks"
)

func TestProgressCommandPrintsSummary(t *testing.T) {
	store := &fakeTaskStore{
		progress: taskstore.Progress{
			Total: 2,
			Counts: map[taskstore.Status]int{
				taskstore.StatusReady:      1,
				taskstore.StatusInProgress: 1,
			},
			Next: []taskstore.Task{testTask("task-1", "Add task manager", taskstore.StatusReady)},
		},
		phases: []taskstore.Phase{testPhase("p3", "Project planning", taskstore.PhaseStatusActive)},
	}
	var stdout bytes.Buffer
	deps := Dependencies{
		WorkspaceStore:   &fakeStore{cfg: testConfig()},
		TaskStoreFactory: func(string) TaskStore { return store },
	}

	code := NewApp(deps, &stdout, &bytes.Buffer{}).Execute(context.Background(), []string{"progress", "--workspace", "game-a"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	for _, want := range []string{"workspace: game-a", "p3", "active", "total: 2", "ready", "in_progress", "task-1"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("progress output missing %q: %q", want, stdout.String())
		}
	}
}

func TestProgressCommandPrintsJSON(t *testing.T) {
	store := &fakeTaskStore{
		progress: taskstore.Progress{
			Total: 2,
			Counts: map[taskstore.Status]int{
				taskstore.StatusReady:      1,
				taskstore.StatusInProgress: 1,
			},
			Next: []taskstore.Task{testTask("task-1", "Add task manager", taskstore.StatusReady)},
		},
		phases: []taskstore.Phase{testPhase("p3", "Project planning", taskstore.PhaseStatusActive)},
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	deps := Dependencies{
		WorkspaceStore:   &fakeStore{cfg: testConfig()},
		TaskStoreFactory: func(string) TaskStore { return store },
	}

	code := NewApp(deps, &stdout, &stderr).Execute(context.Background(), []string{"progress", "--workspace", "game-a", "--format", "json"})

	if code != 0 {
		t.Fatalf("progress exit code = %d, stderr = %q", code, stderr.String())
	}
	payload := decodeJSONObject(t, stdout.Bytes())
	assertJSONStringField(t, payload, "workspace", "game-a")
	assertJSONNumberField(t, payload, "total", 2)

	counts := assertJSONObjectField(t, payload, "counts")
	assertJSONNumberField(t, counts, "ready", 1)
	assertJSONNumberField(t, counts, "in_progress", 1)

	phase := findJSONObject(t, assertJSONObjectListField(t, payload, "phases"), "id", "p3")
	assertJSONStringField(t, phase, "status", "active")

	next := findJSONObject(t, assertJSONObjectListField(t, payload, "next"), "id", "task-1")
	assertJSONStringField(t, next, "status", "ready")
}
