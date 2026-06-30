package cli

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	taskstore "github.com/karoc/adp/internal/tasks"
)

// escPayload is a malicious CSI "clear screen" sequence embedded in a field.
// safeText must replace the ESC (0x1b) byte with U+FFFD while keeping the
// surrounding visible text, so the raw 0x1b never reaches the terminal.
const escPayload = "\x1b[2J"

// assertNoEsc fails if the rendered output still contains a raw ESC byte,
// meaning a terminal control character survived sanitization (CWE-150).
func assertNoEsc(t *testing.T, label string, out []byte) {
	t.Helper()
	if bytes.Contains(out, []byte{0x1b}) {
		t.Fatalf("%s: raw ESC byte survived sanitization; output=%q", label, out)
	}
}

// TestPrintTaskSanitizesAllFields verifies the `adp tasks show` text renderer
// neutralizes control characters in every user-controlled field it prints
// (id, title, owner, description, blocked_reason). Task IDs and these fields
// originate in the hand-editable planning YAML, so they are untrusted.
func TestPrintTaskSanitizesAllFields(t *testing.T) {
	task := testTask("task-1"+escPayload, "T"+escPayload+"itle", taskstore.StatusReady)
	task.Owner = "o" + escPayload + "wner"
	task.Description = "d" + escPayload + "esc"
	task.BlockedReason = "b" + escPayload + "r"
	var buf bytes.Buffer
	NewApp(Dependencies{}, &buf, io.Discard).printTask(task)
	out := buf.Bytes()
	assertNoEsc(t, "printTask", out)
	for _, want := range []string{"task-1", "wner", "esc"} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("printTask dropped visible text %q; output=%q", want, out)
		}
	}
}

// TestPrintPhaseSanitizesAllFields verifies the `adp phase show` text renderer
// neutralizes control characters in id/title/goal and the lifecycle evidence
// fields (acceptance result, commit hash, push remote/branch/result) that are
// also stored in the hand-editable phases YAML.
func TestPrintPhaseSanitizesAllFields(t *testing.T) {
	phase := testPhase("p3"+escPayload, "G"+escPayload+"oal", taskstore.PhaseStatusActive)
	phase.Goal = "g" + escPayload
	phase.Acceptance.Result = "ar" + escPayload
	phase.Commit.Hash = "h" + escPayload
	phase.Push.Remote = "r" + escPayload
	phase.Push.Branch = "b" + escPayload
	phase.Push.Result = "pr" + escPayload
	var buf bytes.Buffer
	NewApp(Dependencies{}, &buf, io.Discard).printPhase(phase)
	out := buf.Bytes()
	assertNoEsc(t, "printPhase", out)
	if !strings.Contains(string(out), "p3") {
		t.Fatalf("printPhase dropped visible id; output=%q", out)
	}
	if !strings.Contains(string(out), "pr") {
		t.Fatalf("printPhase dropped visible push result; output=%q", out)
	}
}

// TestPhaseGatePhaseSummarySanitizesID verifies the `adp phase status` summary
// line neutralizes the phase id (the title was already covered by fix #4; the
// id is the gap this change closes).
func TestPhaseGatePhaseSummarySanitizesID(t *testing.T) {
	phase := testPhase("p"+escPayload, "t"+escPayload, taskstore.PhaseStatusPlanned)
	got := phaseGatePhaseSummary(&phase)
	assertNoEsc(t, "phaseGatePhaseSummary", []byte(got))
	if !strings.Contains(got, "planned") {
		t.Fatalf("status dropped from summary: %q", got)
	}
}

// TestFormatAmbiguousTaskIDListSanitizes verifies the "ambiguous task ID"
// error message joins sanitized ids, so a malicious id loaded from the
// planning YAML cannot inject control characters into the error output.
func TestFormatAmbiguousTaskIDListSanitizes(t *testing.T) {
	tasks := []taskstore.Task{
		testTask("a"+escPayload, "ta", taskstore.StatusReady),
		testTask("b"+escPayload, "tb", taskstore.StatusReady),
	}
	got := formatAmbiguousTaskIDList(tasks)
	assertNoEsc(t, "formatAmbiguousTaskIDList", []byte(got))
	// strings.Join places the separator only between elements (the caller's
	// "  - %s" format supplies the leading prefix), so the first id starts
	// the string and the second follows the separator.
	if !strings.HasPrefix(got, "a") || !strings.Contains(got, "\n  - b") {
		t.Fatalf("ambiguous list dropped visible ids: %q", got)
	}
}

// TestNextWorkReportSanitizesIDsAndOwner verifies the `adp progress report`
// "Next Work" bullet list sanitizes the task id (in backticks), title, and
// owner (rendered via the claim-handoff helper) — all of which fix #4 had left
// raw in this non-table section of the report.
func TestNextWorkReportSanitizesIDsAndOwner(t *testing.T) {
	task := testTask("task-1"+escPayload, "T"+escPayload+"itle", taskstore.StatusReady)
	task.Owner = "o" + escPayload + "wner"
	for label, fn := range map[string]func(io.Writer, []taskstore.Task){
		"english": writeNextWorkReportEnglish,
		"chinese": writeNextWorkReportChinese,
	} {
		var buf bytes.Buffer
		fn(&buf, []taskstore.Task{task})
		assertNoEsc(t, "nextWork/"+label, buf.Bytes())
	}
}

// TestPhaseAndTaskListTablesSanitizeIDs verifies the `adp phase list` and
// `adp tasks list` table renderers sanitize the id column (plus title/owner),
// exercising the sinks end-to-end through Execute.
func TestPhaseAndTaskListTablesSanitizeIDs(t *testing.T) {
	phase := testPhase("p3"+escPayload, "G"+escPayload+"oal", taskstore.PhaseStatusActive)
	task := testTask("task-1"+escPayload, "T"+escPayload+"itle", taskstore.StatusReady)
	task.Owner = "o" + escPayload + "wner"
	store := &fakeTaskStore{phases: []taskstore.Phase{phase}, tasks: []taskstore.Task{task}}
	deps := Dependencies{
		WorkspaceStore:   &fakeStore{cfg: testConfig()},
		TaskStoreFactory: func(string) TaskStore { return store },
	}

	cases := []struct {
		label string
		args  []string
	}{
		{"phase list", []string{"phase", "list", "--workspace", "game-a"}},
		{"tasks list", []string{"tasks", "list", "--workspace", "game-a"}},
	}
	for _, tc := range cases {
		var out bytes.Buffer
		var errBuf bytes.Buffer
		code := NewApp(deps, &out, &errBuf).Execute(context.Background(), tc.args)
		if code != 0 {
			t.Fatalf("%s: exit code %d, stderr=%q", tc.label, code, errBuf.String())
		}
		assertNoEsc(t, tc.label, out.Bytes())
	}
}

// TestProgressCommandSanitizesIDs verifies `adp progress` sanitizes the phase
// and task ids it prints in its `phases:` and `next:` bullet lists.
func TestProgressCommandSanitizesIDs(t *testing.T) {
	phase := testPhase("p3"+escPayload, "G"+escPayload+"oal", taskstore.PhaseStatusPlanned)
	task := testTask("task-1"+escPayload, "T"+escPayload+"itle", taskstore.StatusReady)
	store := &fakeTaskStore{
		phases: []taskstore.Phase{phase},
		tasks:  []taskstore.Task{task},
		progress: taskstore.Progress{
			Total:  1,
			Counts: map[taskstore.Status]int{taskstore.StatusReady: 1},
			Next:   []taskstore.Task{task},
		},
	}
	deps := Dependencies{
		WorkspaceStore:   &fakeStore{cfg: testConfig()},
		TaskStoreFactory: func(string) TaskStore { return store },
	}
	var out bytes.Buffer
	var errBuf bytes.Buffer
	code := NewApp(deps, &out, &errBuf).Execute(context.Background(), []string{"progress", "--workspace", "game-a"})
	if code != 0 {
		t.Fatalf("progress: exit code %d, stderr=%q", code, errBuf.String())
	}
	assertNoEsc(t, "progress", out.Bytes())
}
