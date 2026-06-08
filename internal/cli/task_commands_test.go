package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/karoc/adp/internal/paths"
	"github.com/karoc/adp/internal/sessions"
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
	assertJSONStringField(t, task, "phase", "phase-1.5")
	assertJSONStringField(t, task, "title", "Add task manager")

	detail := decodeJSONObject(t, showOut.Bytes())
	assertJSONStringField(t, detail, "id", "task-1")
	assertJSONStringField(t, detail, "status", "ready")
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

func TestProgressReportCommandPrintsMarkdown(t *testing.T) {
	task := testTask("task-1", "Add task manager", taskstore.StatusReady)
	task.Phase = "p6-progress-report"
	task.Owner = "codex-main"
	phase := testPhase("p6-progress-report", "Planning progress report output", taskstore.PhaseStatusPushed)
	phase.Goal = "local Markdown progress report"
	phase.Acceptance = taskstore.AcceptanceRecord{Commands: []string{"scripts/check-all.sh"}, Result: "passed", At: phase.UpdatedAt}
	phase.Commit = taskstore.CommitRecord{Hash: "abc123", Message: "Add progress report", At: phase.UpdatedAt}
	phase.Push = taskstore.PushRecord{Remote: "origin", Branch: "main", Result: "pushed", At: phase.UpdatedAt}
	store := &fakeTaskStore{
		tasks:  []taskstore.Task{task},
		phases: []taskstore.Phase{phase},
		progress: taskstore.Progress{
			Total: 1,
			Counts: map[taskstore.Status]int{
				taskstore.StatusReady: 1,
			},
			Next: []taskstore.Task{task},
		},
	}
	var english bytes.Buffer
	var markdown bytes.Buffer
	var chinese bytes.Buffer
	var jsonOut bytes.Buffer
	var jsonErr bytes.Buffer
	var invalidErr bytes.Buffer
	var invalidFormatErr bytes.Buffer
	exitCode := 0
	deps := Dependencies{
		WorkspaceStore:   &fakeStore{cfg: testConfig()},
		TaskStoreFactory: func(string) TaskStore { return store },
		ListSessions: func(_ context.Context, _ paths.Layout, query sessions.Query) ([]sessions.Summary, error) {
			if query.Workspace != "game-a" || query.Limit != 5 {
				t.Fatalf("session query = %+v", query)
			}
			return []sessions.Summary{{
				SessionID:      "session-1",
				Workspace:      "game-a",
				Agent:          "codex",
				TaskID:         "task-1",
				RuntimePath:    "/tmp/adp-runtime/session-1",
				StartedAt:      task.CreatedAt,
				FinishedAt:     task.UpdatedAt,
				ExitCode:       &exitCode,
				DurationMillis: ptrInt64(1234),
				EventCount:     2,
			}}, nil
		},
	}

	englishCode := NewApp(deps, &english, &bytes.Buffer{}).Execute(context.Background(), []string{"progress", "report", "--workspace", "game-a"})
	markdownCode := NewApp(deps, &markdown, &bytes.Buffer{}).Execute(context.Background(), []string{"progress", "report", "--workspace", "game-a", "--format", "markdown"})
	chineseCode := NewApp(deps, &chinese, &bytes.Buffer{}).Execute(context.Background(), []string{"progress", "report", "--workspace", "game-a", "--language", "zh-CN"})
	jsonCode := NewApp(deps, &jsonOut, &jsonErr).Execute(context.Background(), []string{"progress", "report", "--workspace", "game-a", "--format", "json"})
	invalidCode := NewApp(deps, &bytes.Buffer{}, &invalidErr).Execute(context.Background(), []string{"progress", "report", "--workspace", "game-a", "--language", "fr"})
	invalidFormatCode := NewApp(deps, &bytes.Buffer{}, &invalidFormatErr).Execute(context.Background(), []string{"progress", "report", "--workspace", "game-a", "--format", "xml"})

	if englishCode != 0 || markdownCode != 0 || chineseCode != 0 || jsonCode != 0 {
		t.Fatalf("report codes = (%d, %d, %d, %d), stderr = %q, want all 0", englishCode, markdownCode, chineseCode, jsonCode, jsonErr.String())
	}
	for _, want := range []string{"# ADP Progress Report", "Workspace: game-a", "Total Tasks: 1", "p6-progress-report", "task-1", "codex-main", "passed: scripts/check-all.sh", "abc123: Add progress report", "pushed: origin/main", "## Runtime Sessions", "session-1", "codex", "/tmp/adp-runtime/session-1"} {
		if !strings.Contains(english.String(), want) {
			t.Fatalf("English report missing %q: %q", want, english.String())
		}
	}
	for _, want := range []string{"# ADP 执行进度报告", "工作区：game-a", "任务总数：1", "p6-progress-report", "task-1", "## Runtime 会话", "session-1", "/tmp/adp-runtime/session-1"} {
		if !strings.Contains(chinese.String(), want) {
			t.Fatalf("Chinese report missing %q: %q", want, chinese.String())
		}
	}
	if !strings.Contains(markdown.String(), "# ADP Progress Report") {
		t.Fatalf("explicit Markdown report = %q", markdown.String())
	}
	payload := decodeJSONObject(t, jsonOut.Bytes())
	assertJSONStringField(t, payload, "workspace", "game-a")
	assertJSONNumberField(t, payload, "total", 1)
	counts := assertJSONObjectField(t, payload, "counts")
	assertJSONNumberField(t, counts, "ready", 1)
	phaseJSON := findJSONObject(t, assertJSONObjectListField(t, payload, "phases"), "id", "p6-progress-report")
	assertJSONStringField(t, phaseJSON, "status", "pushed")
	taskJSON := findJSONObject(t, assertJSONObjectListField(t, payload, "tasks"), "id", "task-1")
	assertJSONStringField(t, taskJSON, "owner", "codex-main")
	nextJSON := findJSONObject(t, assertJSONObjectListField(t, payload, "next"), "id", "task-1")
	assertJSONStringField(t, nextJSON, "status", "ready")
	evidenceJSON := findJSONObject(t, assertJSONObjectListField(t, payload, "phase_evidence"), "id", "p6-progress-report")
	acceptanceJSON := assertJSONObjectField(t, evidenceJSON, "acceptance")
	assertJSONStringField(t, acceptanceJSON, "result", "passed")
	commitJSON := assertJSONObjectField(t, evidenceJSON, "commit")
	assertJSONStringField(t, commitJSON, "hash", "abc123")
	pushJSON := assertJSONObjectField(t, evidenceJSON, "push")
	assertJSONStringField(t, pushJSON, "result", "pushed")
	sessionJSON := findJSONObject(t, assertJSONObjectListField(t, payload, "runtime_sessions"), "session_id", "session-1")
	assertJSONStringField(t, sessionJSON, "agent", "codex")
	assertJSONStringField(t, sessionJSON, "task_id", "task-1")
	assertJSONStringField(t, sessionJSON, "runtime_path", "/tmp/adp-runtime/session-1")
	assertJSONNumberField(t, sessionJSON, "event_count", 2)
	if invalidCode != 1 {
		t.Fatalf("invalid language exit code = %d, want 1", invalidCode)
	}
	if !strings.Contains(invalidErr.String(), `unknown progress report language "fr"`) {
		t.Fatalf("invalid language stderr = %q", invalidErr.String())
	}
	if invalidFormatCode != 1 {
		t.Fatalf("invalid format exit code = %d, want 1", invalidFormatCode)
	}
	if !strings.Contains(invalidFormatErr.String(), `unknown progress report format "xml"`) {
		t.Fatalf("invalid format stderr = %q", invalidFormatErr.String())
	}
}

func ptrInt64(value int64) *int64 {
	return &value
}

func TestPhaseCommandsRecordGateLifecycle(t *testing.T) {
	store := &fakeTaskStore{}
	deps := Dependencies{
		WorkspaceStore:   &fakeStore{cfg: testConfig()},
		TaskStoreFactory: func(string) TaskStore { return store },
	}
	var stdout bytes.Buffer

	commands := [][]string{
		{"phase", "add", "--workspace", "game-a", "--goal", "local phase gates", "p3", "Project", "planning"},
		{"phase", "start", "--workspace", "game-a", "p3"},
		{"phase", "accept", "--workspace", "game-a", "p3", "--command", "scripts/task-manager-smoke.sh", "--result", "passed", "--notes", "runtime smoke accepted"},
		{"phase", "commit", "--workspace", "game-a", "p3", "--hash", "abc123", "--message", "Add phase gates"},
		{"phase", "push", "--workspace", "game-a", "p3", "--remote", "origin", "--branch", "main", "--result", "pushed"},
		{"phase", "list", "--workspace", "game-a"},
		{"phase", "show", "--workspace", "game-a", "p3"},
	}
	for _, args := range commands {
		code := NewApp(deps, &stdout, &bytes.Buffer{}).Execute(context.Background(), args)
		if code != 0 {
			t.Fatalf("adp %v exit code = %d, want 0", args, code)
		}
	}

	output := stdout.String()
	for _, want := range []string{"phase p3 added", "status: active", "accepted: passed", "commit: abc123", "push: origin/main pushed", "Project planning", "commit_hash: abc123", "push_result: pushed"} {
		if !strings.Contains(output, want) {
			t.Fatalf("phase output missing %q: %q", want, output)
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
	phases        []taskstore.Phase
	updatedStatus taskstore.Status
	blockReason   string
	claimReq      taskstore.ClaimRequest
	releaseReq    taskstore.ReleaseRequest
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

func (s *fakeTaskStore) Claim(_ context.Context, req taskstore.ClaimRequest) (taskstore.Task, error) {
	s.claimReq = req
	task := testTask(req.TaskID, "Add task manager", taskstore.StatusInProgress)
	task.Owner = req.Owner
	if req.Lease > 0 {
		task.LeaseExpiresAt = task.UpdatedAt.Add(req.Lease)
	}
	return task, nil
}

func (s *fakeTaskStore) Release(_ context.Context, req taskstore.ReleaseRequest) (taskstore.Task, error) {
	s.releaseReq = req
	return testTask(req.TaskID, "Add task manager", taskstore.StatusReady), nil
}

func (s *fakeTaskStore) Progress(context.Context) (taskstore.Progress, error) {
	return s.progress, nil
}

func (s *fakeTaskStore) AddPhase(_ context.Context, req taskstore.PhaseAddRequest) (taskstore.Phase, error) {
	phase := testPhase(req.ID, req.Title, taskstore.PhaseStatusPlanned)
	phase.Goal = req.Goal
	s.phases = append(s.phases, phase)
	return phase, nil
}

func (s *fakeTaskStore) ListPhases(context.Context) ([]taskstore.Phase, error) {
	return s.phases, nil
}

func (s *fakeTaskStore) GetPhase(_ context.Context, id string) (taskstore.Phase, error) {
	for _, phase := range s.phases {
		if phase.ID == id {
			return phase, nil
		}
	}
	return testPhase(id, "Project planning", taskstore.PhaseStatusPushed), nil
}

func (s *fakeTaskStore) StartPhase(_ context.Context, id string) (taskstore.Phase, error) {
	phase := s.currentPhase(id)
	phase.Status = taskstore.PhaseStatusActive
	s.upsertPhase(phase)
	return phase, nil
}

func (s *fakeTaskStore) AcceptPhase(_ context.Context, req taskstore.PhaseAcceptRequest) (taskstore.Phase, error) {
	phase := s.currentPhase(req.ID)
	phase.Status = taskstore.PhaseStatusAccepted
	phase.Acceptance = taskstore.AcceptanceRecord{Commands: req.Commands, Result: req.Result, Notes: req.Notes, At: phase.UpdatedAt}
	s.upsertPhase(phase)
	return phase, nil
}

func (s *fakeTaskStore) RecordPhaseCommit(_ context.Context, req taskstore.PhaseCommitRequest) (taskstore.Phase, error) {
	phase := s.currentPhase(req.ID)
	phase.Status = taskstore.PhaseStatusCommitted
	phase.Commit = taskstore.CommitRecord{Hash: req.Hash, Message: req.Message, At: phase.UpdatedAt}
	s.upsertPhase(phase)
	return phase, nil
}

func (s *fakeTaskStore) RecordPhasePush(_ context.Context, req taskstore.PhasePushRequest) (taskstore.Phase, error) {
	phase := s.currentPhase(req.ID)
	phase.Status = taskstore.PhaseStatusPushed
	phase.Push = taskstore.PushRecord{Remote: req.Remote, Branch: req.Branch, Result: req.Result, At: phase.UpdatedAt}
	s.upsertPhase(phase)
	return phase, nil
}

func (s *fakeTaskStore) currentPhase(id string) taskstore.Phase {
	for _, phase := range s.phases {
		if phase.ID == id {
			return phase
		}
	}
	return testPhase(id, "Project planning", taskstore.PhaseStatusPlanned)
}

func (s *fakeTaskStore) upsertPhase(next taskstore.Phase) {
	for i := range s.phases {
		if s.phases[i].ID == next.ID {
			s.phases[i] = next
			return
		}
	}
	s.phases = append(s.phases, next)
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

func testPhase(id string, title string, status taskstore.PhaseStatus) taskstore.Phase {
	ts := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)
	return taskstore.Phase{
		ID:        id,
		Title:     title,
		Status:    status,
		CreatedAt: ts,
		UpdatedAt: ts,
	}
}
