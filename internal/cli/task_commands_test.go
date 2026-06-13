package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	taskstore "github.com/karoc/adp/internal/tasks"
)

func TestTasksAddCommandCreatesTask(t *testing.T) {
	store := &fakeTaskStore{}
	var gotWorkspaceDir string
	var stdout bytes.Buffer

	deps := Dependencies{
		WorkspaceStore: &fakeStore{cfg: testConfig()},
		TaskStoreFactory: func(workspaceDir string) TaskStore {
			gotWorkspaceDir = workspaceDir
			return store
		},
	}

	code := NewApp(deps, &stdout, &bytes.Buffer{}).Execute(context.Background(), []string{
		"tasks", "add", "--workspace", "game-a", "--priority", "high", "--phase", "phase-1.5", "--description", "local task state", "Add", "task", "manager",
	})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if gotWorkspaceDir != "/tmp/adp-home/workspaces/game-a" {
		t.Fatalf("workspace dir = %q", gotWorkspaceDir)
	}
	if store.addReq.Title != "Add task manager" || store.addReq.Priority != "high" || store.addReq.Phase != "phase-1.5" || store.addReq.Description != "local task state" {
		t.Fatalf("add request = %+v", store.addReq)
	}
	if !strings.Contains(stdout.String(), "task task-1 added") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestTasksListAndShowCommandsReadTasks(t *testing.T) {
	store := &fakeTaskStore{tasks: []taskstore.Task{testTask("task-1", "Add task manager", taskstore.StatusReady)}}
	var listOut bytes.Buffer
	var showOut bytes.Buffer
	deps := Dependencies{
		WorkspaceStore:   &fakeStore{cfg: testConfig()},
		TaskStoreFactory: func(string) TaskStore { return store },
	}

	listCode := NewApp(deps, &listOut, &bytes.Buffer{}).Execute(context.Background(), []string{"tasks", "list", "--workspace", "game-a"})
	showCode := NewApp(deps, &showOut, &bytes.Buffer{}).Execute(context.Background(), []string{"tasks", "show", "--workspace", "game-a", "task-1"})

	if listCode != 0 || showCode != 0 {
		t.Fatalf("codes = (%d, %d), want both 0", listCode, showCode)
	}
	for _, want := range []string{"task-1", "ready", "unclaimed", "Add task manager"} {
		if !strings.Contains(listOut.String(), want) {
			t.Fatalf("tasks list missing %q: %q", want, listOut.String())
		}
	}
	for _, want := range []string{"id: task-1", "title: Add task manager", "status: ready", "claim_state: unclaimed"} {
		if !strings.Contains(showOut.String(), want) {
			t.Fatalf("tasks show missing %q: %q", want, showOut.String())
		}
	}
}

func TestTasksListAndShowCommandsPrintJSON(t *testing.T) {
	store := &fakeTaskStore{tasks: []taskstore.Task{testTask("task-1", "Add task manager", taskstore.StatusReady)}}
	var listOut bytes.Buffer
	var listErr bytes.Buffer
	var showOut bytes.Buffer
	var showErr bytes.Buffer
	deps := Dependencies{
		WorkspaceStore:   &fakeStore{cfg: testConfig()},
		TaskStoreFactory: func(string) TaskStore { return store },
	}

	listCode := NewApp(deps, &listOut, &listErr).Execute(context.Background(), []string{"tasks", "list", "--workspace", "game-a", "--format", "json"})
	showCode := NewApp(deps, &showOut, &showErr).Execute(context.Background(), []string{"tasks", "show", "--workspace", "game-a", "task-1", "--format", "json"})

	if listCode != 0 {
		t.Fatalf("tasks list exit code = %d, stderr = %q", listCode, listErr.String())
	}
	if showCode != 0 {
		t.Fatalf("tasks show exit code = %d, stderr = %q", showCode, showErr.String())
	}

	task := findJSONObject(t, decodeJSONObjectList(t, listOut.Bytes(), "tasks"), "id", "task-1")
	assertJSONStringField(t, task, "status", "ready")
	assertJSONStringField(t, task, "claim_state", "unclaimed")
	assertJSONStringField(t, task, "phase", "phase-1.5")
	assertJSONStringField(t, task, "title", "Add task manager")

	detail := decodeJSONObject(t, showOut.Bytes())
	assertJSONStringField(t, detail, "id", "task-1")
	assertJSONStringField(t, detail, "status", "ready")
	assertJSONStringField(t, detail, "claim_state", "unclaimed")
	assertJSONStringField(t, detail, "priority", "high")
	assertJSONStringField(t, detail, "phase", "phase-1.5")
}

func TestTasksUpdateDoneAndBlockCommandsUpdateStatus(t *testing.T) {
	store := &fakeTaskStore{}
	deps := Dependencies{
		WorkspaceStore:   &fakeStore{cfg: testConfig()},
		TaskStoreFactory: func(string) TaskStore { return store },
	}

	updateCode := NewApp(deps, &bytes.Buffer{}, &bytes.Buffer{}).Execute(context.Background(), []string{"tasks", "update", "--workspace", "game-a", "task-1", "--status", "in_progress"})
	doneCode := NewApp(deps, &bytes.Buffer{}, &bytes.Buffer{}).Execute(context.Background(), []string{"tasks", "done", "--workspace", "game-a", "task-1"})
	blockCode := NewApp(deps, &bytes.Buffer{}, &bytes.Buffer{}).Execute(context.Background(), []string{"tasks", "block", "--workspace", "game-a", "task-1", "--reason", "needs review"})

	if updateCode != 0 || doneCode != 0 || blockCode != 0 {
		t.Fatalf("codes = (%d, %d, %d), want all 0", updateCode, doneCode, blockCode)
	}
	if store.updatedStatus != taskstore.StatusDone {
		t.Fatalf("updated status = %q, want done", store.updatedStatus)
	}
	if store.blockReason != "needs review" {
		t.Fatalf("block reason = %q", store.blockReason)
	}
}

func TestTasksClaimAndReleaseCommandsSetOwner(t *testing.T) {
	store := &fakeTaskStore{}
	deps := Dependencies{
		WorkspaceStore:   &fakeStore{cfg: testConfig()},
		TaskStoreFactory: func(string) TaskStore { return store },
	}
	var claimOut bytes.Buffer
	var releaseOut bytes.Buffer

	claimCode := NewApp(deps, &claimOut, &bytes.Buffer{}).Execute(context.Background(), []string{"tasks", "claim", "--workspace", "game-a", "task-1", "--owner", "codex-main", "--lease", "30m"})
	releaseCode := NewApp(deps, &releaseOut, &bytes.Buffer{}).Execute(context.Background(), []string{"tasks", "release", "--workspace", "game-a", "task-1", "--owner", "codex-main"})

	if claimCode != 0 || releaseCode != 0 {
		t.Fatalf("codes = (%d, %d), want both 0", claimCode, releaseCode)
	}
	if store.claimReq.Owner != "codex-main" || store.claimReq.Lease != 30*time.Minute || store.releaseReq.TaskID != "task-1" || store.releaseReq.Owner != "codex-main" {
		t.Fatalf("claim/release = (%+v, %+v)", store.claimReq, store.releaseReq)
	}
	if !strings.Contains(claimOut.String(), "claimed by codex-main") || !strings.Contains(releaseOut.String(), "released") {
		t.Fatalf("outputs = (%q, %q)", claimOut.String(), releaseOut.String())
	}
}

func TestTasksTakeCommandClaimsNextTask(t *testing.T) {
	store := &fakeTaskStore{}
	deps := Dependencies{
		WorkspaceStore:   &fakeStore{cfg: testConfig()},
		TaskStoreFactory: func(string) TaskStore { return store },
	}
	var stdout bytes.Buffer

	code := NewApp(deps, &stdout, &bytes.Buffer{}).Execute(context.Background(), []string{"tasks", "take", "--workspace", "game-a", "--owner", "agent-a", "--lease", "45m"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if store.takeReq.Owner != "agent-a" || store.takeReq.Lease != 45*time.Minute {
		t.Fatalf("take request = %+v", store.takeReq)
	}
	for _, want := range []string{"task task-take taken by agent-a", "id: task-take", "status: in_progress", "owner: agent-a", "claim_state: leased", "lease_expires_at:"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("take output missing %q: %q", want, stdout.String())
		}
	}
}

func TestTasksTakeCommandPrintsJSON(t *testing.T) {
	store := &fakeTaskStore{}
	deps := Dependencies{
		WorkspaceStore:   &fakeStore{cfg: testConfig()},
		TaskStoreFactory: func(string) TaskStore { return store },
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := NewApp(deps, &stdout, &stderr).Execute(context.Background(), []string{"tasks", "take", "--workspace", "game-a", "--owner", "agent-a", "--format", "json"})

	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr.String())
	}
	task := decodeJSONObject(t, stdout.Bytes())
	assertJSONStringField(t, task, "id", "task-take")
	assertJSONStringField(t, task, "status", "in_progress")
	assertJSONStringField(t, task, "owner", "agent-a")
	assertJSONStringField(t, task, "claim_state", "claimed")
}

func TestTasksRenewCommandExtendsLease(t *testing.T) {
	store := &fakeTaskStore{}
	deps := Dependencies{
		WorkspaceStore:   &fakeStore{cfg: testConfig()},
		TaskStoreFactory: func(string) TaskStore { return store },
	}
	var stdout bytes.Buffer

	code := NewApp(deps, &stdout, &bytes.Buffer{}).Execute(context.Background(), []string{"tasks", "renew", "--workspace", "game-a", "task-1", "--owner", "agent-a", "--lease", "50m"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if store.renewReq.TaskID != "task-1" || store.renewReq.Owner != "agent-a" || store.renewReq.Lease != 50*time.Minute {
		t.Fatalf("renew request = %+v", store.renewReq)
	}
	if !strings.Contains(stdout.String(), "task task-1 lease renewed until ") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestTasksStaleCommandPrintsTextAndJSON(t *testing.T) {
	stale := testTask("task-stale", "Expired task", taskstore.StatusInProgress)
	stale.Owner = "agent-old"
	stale.ClaimedAt = stale.UpdatedAt.Add(-time.Hour)
	stale.LeaseExpiresAt = stale.UpdatedAt.Add(-time.Minute)
	store := &fakeTaskStore{staleTasks: []taskstore.Task{stale}}
	deps := Dependencies{
		WorkspaceStore:   &fakeStore{cfg: testConfig()},
		TaskStoreFactory: func(string) TaskStore { return store },
	}
	var textOut bytes.Buffer
	var jsonOut bytes.Buffer
	var jsonErr bytes.Buffer

	textCode := NewApp(deps, &textOut, &bytes.Buffer{}).Execute(context.Background(), []string{"tasks", "stale", "--workspace", "game-a"})
	jsonCode := NewApp(deps, &jsonOut, &jsonErr).Execute(context.Background(), []string{"tasks", "stale", "--workspace", "game-a", "--format", "json"})

	if textCode != 0 {
		t.Fatalf("text exit code = %d, output = %q", textCode, textOut.String())
	}
	for _, want := range []string{"workspace: game-a", "stale_count: 1", "task-stale", "agent-old", "stale since", "Expired task"} {
		if !strings.Contains(textOut.String(), want) {
			t.Fatalf("tasks stale text missing %q: %q", want, textOut.String())
		}
	}
	if jsonCode != 0 {
		t.Fatalf("json exit code = %d, stderr = %q", jsonCode, jsonErr.String())
	}
	payload := decodeJSONObject(t, jsonOut.Bytes())
	assertJSONStringField(t, payload, "workspace", "game-a")
	assertJSONNumberField(t, payload, "stale_count", 1)
	task := findJSONObject(t, assertJSONObjectListField(t, payload, "tasks"), "id", "task-stale")
	assertJSONStringField(t, task, "owner", "agent-old")
	assertJSONStringField(t, task, "claim_state", "stale")
	assertJSONStringField(t, task, "status", "in_progress")
}

func TestTasksCommandReportsUnknownSubcommand(t *testing.T) {
	var stderr bytes.Buffer

	code := NewApp(Dependencies{}, &bytes.Buffer{}, &stderr).Execute(context.Background(), []string{"tasks", "bogus"})

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), `adp: unknown tasks command "bogus"`) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestTasksPrefixMatching(t *testing.T) {
	t.Run("exact match works", func(t *testing.T) {
		store := &fakeTaskStore{
			tasks: []taskstore.Task{
				testTask("task-20260611-0001", "First task", taskstore.StatusReady),
				testTask("task-20260611-0002", "Second task", taskstore.StatusReady),
			},
		}
		deps := Dependencies{
			WorkspaceStore:   &fakeStore{cfg: testConfig()},
			TaskStoreFactory: func(string) TaskStore { return store },
		}
		var stdout bytes.Buffer

		code := NewApp(deps, &stdout, &bytes.Buffer{}).Execute(context.Background(), []string{
			"tasks", "show", "--workspace", "game-a", "task-20260611-0001",
		})

		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		if !strings.Contains(stdout.String(), "id: task-20260611-0001") {
			t.Fatalf("stdout = %q", stdout.String())
		}
	})

	t.Run("unique prefix match works", func(t *testing.T) {
		store := &fakeTaskStore{
			tasks: []taskstore.Task{
				testTask("task-20260611-0001", "First task", taskstore.StatusReady),
				testTask("task-20260612-0001", "Second task", taskstore.StatusReady),
			},
		}
		deps := Dependencies{
			WorkspaceStore:   &fakeStore{cfg: testConfig()},
			TaskStoreFactory: func(string) TaskStore { return store },
		}
		var stdout bytes.Buffer

		code := NewApp(deps, &stdout, &bytes.Buffer{}).Execute(context.Background(), []string{
			"tasks", "show", "--workspace", "game-a", "task-20260611",
		})

		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		if !strings.Contains(stdout.String(), "id: task-20260611-0001") {
			t.Fatalf("stdout = %q", stdout.String())
		}
	})

	t.Run("ambiguous prefix reports error", func(t *testing.T) {
		store := &fakeTaskStore{
			tasks: []taskstore.Task{
				testTask("task-20260611-0001", "First task", taskstore.StatusReady),
				testTask("task-20260611-0002", "Second task", taskstore.StatusReady),
			},
		}
		deps := Dependencies{
			WorkspaceStore:   &fakeStore{cfg: testConfig()},
			TaskStoreFactory: func(string) TaskStore { return store },
		}
		var stderr bytes.Buffer

		code := NewApp(deps, &bytes.Buffer{}, &stderr).Execute(context.Background(), []string{
			"tasks", "show", "--workspace", "game-a", "task-20260611",
		})

		if code != 1 {
			t.Fatalf("exit code = %d, want 1", code)
		}
		output := stderr.String()
		if !strings.Contains(output, "ambiguous task ID") {
			t.Fatalf("stderr should mention ambiguous, got: %q", output)
		}
		if !strings.Contains(output, "task-20260611-0001") || !strings.Contains(output, "task-20260611-0002") {
			t.Fatalf("stderr should list matching tasks, got: %q", output)
		}
	})

	t.Run("prefix works with update command", func(t *testing.T) {
		store := &fakeTaskStore{
			tasks: []taskstore.Task{
				testTask("task-20260611-0001", "First task", taskstore.StatusReady),
			},
		}
		deps := Dependencies{
			WorkspaceStore:   &fakeStore{cfg: testConfig()},
			TaskStoreFactory: func(string) TaskStore { return store },
		}
		var stdout bytes.Buffer

		code := NewApp(deps, &stdout, &bytes.Buffer{}).Execute(context.Background(), []string{
			"tasks", "update", "--workspace", "game-a", "task-2026", "--status", "in_progress",
		})

		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		if store.updatedStatus != taskstore.StatusInProgress {
			t.Fatalf("status = %q, want in_progress", store.updatedStatus)
		}
	})

	t.Run("prefix works with claim command", func(t *testing.T) {
		store := &fakeTaskStore{
			tasks: []taskstore.Task{
				testTask("task-20260611-0001", "First task", taskstore.StatusReady),
			},
		}
		deps := Dependencies{
			WorkspaceStore:   &fakeStore{cfg: testConfig()},
			TaskStoreFactory: func(string) TaskStore { return store },
		}

		code := NewApp(deps, &bytes.Buffer{}, &bytes.Buffer{}).Execute(context.Background(), []string{
			"tasks", "claim", "--workspace", "game-a", "task-2026", "--owner", "agent-1",
		})

		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		if store.claimReq.TaskID != "task-20260611-0001" {
			t.Fatalf("claimed task = %q, want task-20260611-0001", store.claimReq.TaskID)
		}
	})

	t.Run("prefix works with done command", func(t *testing.T) {
		store := &fakeTaskStore{
			tasks: []taskstore.Task{
				testTask("task-20260611-0001", "First task", taskstore.StatusInProgress),
			},
		}
		deps := Dependencies{
			WorkspaceStore:   &fakeStore{cfg: testConfig()},
			TaskStoreFactory: func(string) TaskStore { return store },
		}

		code := NewApp(deps, &bytes.Buffer{}, &bytes.Buffer{}).Execute(context.Background(), []string{
			"tasks", "done", "--workspace", "game-a", "task-2026",
		})

		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		if store.updatedStatus != taskstore.StatusDone {
			t.Fatalf("status = %q, want done", store.updatedStatus)
		}
	})
}

