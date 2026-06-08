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
	for _, want := range []string{"task-1", "ready", "Add task manager"} {
		if !strings.Contains(listOut.String(), want) {
			t.Fatalf("tasks list missing %q: %q", want, listOut.String())
		}
	}
	for _, want := range []string{"id: task-1", "title: Add task manager", "status: ready"} {
		if !strings.Contains(showOut.String(), want) {
			t.Fatalf("tasks show missing %q: %q", want, showOut.String())
		}
	}
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
	for _, want := range []string{"workspace: game-a", "total: 2", "ready", "in_progress", "task-1"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("progress output missing %q: %q", want, stdout.String())
		}
	}
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

type fakeTaskStore struct {
	addReq        taskstore.AddRequest
	tasks         []taskstore.Task
	updatedStatus taskstore.Status
	blockReason   string
	progress      taskstore.Progress
}

func (s *fakeTaskStore) Add(_ context.Context, req taskstore.AddRequest) (taskstore.Task, error) {
	s.addReq = req
	return testTask("task-1", req.Title, taskstore.StatusReady), nil
}

func (s *fakeTaskStore) List(context.Context) ([]taskstore.Task, error) {
	return s.tasks, nil
}

func (s *fakeTaskStore) Get(_ context.Context, id string) (taskstore.Task, error) {
	for _, task := range s.tasks {
		if task.ID == id {
			return task, nil
		}
	}
	return testTask(id, "Add task manager", taskstore.StatusReady), nil
}

func (s *fakeTaskStore) UpdateStatus(_ context.Context, id string, status taskstore.Status) (taskstore.Task, error) {
	s.updatedStatus = status
	return testTask(id, "Add task manager", status), nil
}

func (s *fakeTaskStore) Block(_ context.Context, id string, reason string) (taskstore.Task, error) {
	s.blockReason = reason
	task := testTask(id, "Add task manager", taskstore.StatusBlocked)
	task.BlockedReason = reason
	return task, nil
}

func (s *fakeTaskStore) Progress(context.Context) (taskstore.Progress, error) {
	return s.progress, nil
}

func testTask(id string, title string, status taskstore.Status) taskstore.Task {
	ts := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)
	return taskstore.Task{
		ID:        id,
		Title:     title,
		Status:    status,
		Priority:  "high",
		Phase:     "phase-1.5",
		CreatedAt: ts,
		UpdatedAt: ts,
	}
}
