package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/karoc/adp/internal/paths"
	"github.com/karoc/adp/internal/sessions"
	taskstore "github.com/karoc/adp/internal/tasks"
)

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
	var textFormatErr bytes.Buffer
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
	textFormatCode := NewApp(deps, &bytes.Buffer{}, &textFormatErr).Execute(context.Background(), []string{"progress", "report", "--workspace", "game-a", "--format", "text"})

	if englishCode != 0 || markdownCode != 0 || chineseCode != 0 || jsonCode != 0 {
		t.Fatalf("report codes = (%d, %d, %d, %d), stderr = %q, want all 0", englishCode, markdownCode, chineseCode, jsonCode, jsonErr.String())
	}
	for _, want := range []string{"# ADP Progress Report", "Workspace: game-a", "Total Tasks: 1", "p6-progress-report", "task-1", "codex-main", "Claim", "claim: claimed by codex-main", "passed: scripts/check-all.sh", "abc123: Add progress report", "pushed: origin/main", "## Runtime Sessions", "session-1", "codex", "/tmp/adp-runtime/session-1"} {
		if !strings.Contains(english.String(), want) {
			t.Fatalf("English report missing %q: %q", want, english.String())
		}
	}
	for _, want := range []string{"# ADP 执行进度报告", "工作区：game-a", "任务总数：1", "p6-progress-report", "task-1", "领取状态", "领取：codex-main 已领取", "## Runtime 会话", "session-1", "/tmp/adp-runtime/session-1"} {
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
	assertJSONStringField(t, taskJSON, "claim_state", "claimed")
	nextJSON := findJSONObject(t, assertJSONObjectListField(t, payload, "next"), "id", "task-1")
	assertJSONStringField(t, nextJSON, "status", "ready")
	assertJSONStringField(t, nextJSON, "claim_state", "claimed")
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
	if textFormatCode != 1 {
		t.Fatalf("text format exit code = %d, want 1", textFormatCode)
	}
	if !strings.Contains(textFormatErr.String(), `unknown progress report format "text"`) {
		t.Fatalf("text format stderr = %q", textFormatErr.String())
	}
}

func ptrInt64(value int64) *int64 {
	return &value
}
