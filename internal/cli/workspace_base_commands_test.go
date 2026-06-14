package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/karoc/adp/internal/workspace"
)

func TestInitCommandCallsWorkspaceStore(t *testing.T) {
	store := &fakeStore{}
	var stdout bytes.Buffer

	code := NewApp(Dependencies{WorkspaceStore: store}, &stdout, &bytes.Buffer{}).Execute(context.Background(), []string{"init"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !store.initCalled {
		t.Fatal("Init was not called")
	}
}

func TestWorkspaceAddCommandCallsStore(t *testing.T) {
	store := &fakeStore{}

	code := NewApp(Dependencies{WorkspaceStore: store}, &bytes.Buffer{}, &bytes.Buffer{}).Execute(
		context.Background(),
		[]string{"workspace", "add", "game-a", "/srv/game-a"},
	)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if store.addName != "game-a" || store.addRoot != "/srv/game-a" {
		t.Fatalf("Add called with (%q, %q)", store.addName, store.addRoot)
	}
}

func TestWorkspaceListCommandPrintsRecords(t *testing.T) {
	store := &fakeStore{records: []workspace.Record{
		{Name: "game-a", ProjectRoot: "/srv/game-a", WorkspaceDir: "/tmp/adp/workspaces/game-a"},
	}}
	var stdout bytes.Buffer

	code := NewApp(Dependencies{WorkspaceStore: store}, &stdout, &bytes.Buffer{}).Execute(context.Background(), []string{"workspace", "list"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	output := stdout.String()
	if !strings.Contains(output, "game-a") || !strings.Contains(output, "/srv/game-a") {
		t.Fatalf("list output missing workspace: %q", output)
	}
}

func TestWorkspaceShowCommandPrintsDetails(t *testing.T) {
	store := &fakeStore{cfg: testConfig()}
	var stdout bytes.Buffer

	code := NewApp(Dependencies{WorkspaceStore: store}, &stdout, &bytes.Buffer{}).Execute(context.Background(), []string{"workspace", "show", "game-a"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	output := stdout.String()
	if !strings.Contains(output, "name: game-a") || !strings.Contains(output, "project_root: /srv/game-a") {
		t.Fatalf("show output missing details: %q", output)
	}
}

func TestWorkspaceRemoveCommandCallsStore(t *testing.T) {
	store := &fakeStore{cfg: testConfig()}
	var stdout bytes.Buffer

	code := NewApp(Dependencies{WorkspaceStore: store}, &stdout, &bytes.Buffer{}).Execute(context.Background(), []string{"workspace", "remove", "game-a", "--yes"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if store.removeName != "game-a" {
		t.Fatalf("Remove called with %q", store.removeName)
	}
	if !strings.Contains(stdout.String(), "removed") {
		t.Fatalf("remove output = %q", stdout.String())
	}
}

func TestWorkspaceRenameCommandCallsStore(t *testing.T) {
	store := &fakeStore{}
	var stdout bytes.Buffer

	code := NewApp(Dependencies{WorkspaceStore: store}, &stdout, &bytes.Buffer{}).Execute(context.Background(), []string{"workspace", "rename", "old", "new"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if store.renameOld != "old" || store.renameNew != "new" {
		t.Fatalf("Rename called with (%q, %q)", store.renameOld, store.renameNew)
	}
	if !strings.Contains(stdout.String(), `"old" renamed to "new"`) {
		t.Fatalf("rename output = %q", stdout.String())
	}
}

func TestWorkspaceDoctorCommandPrintsNamedReport(t *testing.T) {
	store := &fakeStore{
		diagnoseReport: workspace.DiagnosticReport{
			Workspace:    "game-a",
			WorkspaceDir: "/tmp/adp-home/workspaces/game-a",
		},
	}
	var stdout bytes.Buffer

	code := NewApp(Dependencies{WorkspaceStore: store}, &stdout, &bytes.Buffer{}).Execute(context.Background(), []string{"workspace", "doctor", "game-a"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if store.diagnoseName != "game-a" {
		t.Fatalf("Diagnose called with %q", store.diagnoseName)
	}
	output := stdout.String()
	if !strings.Contains(output, "game-a") || !strings.Contains(output, "ok") || !strings.Contains(output, "no issues") {
		t.Fatalf("doctor output missing healthy report: %q", output)
	}
}

func TestWorkspaceDoctorCommandHidesInfoDiagnosticsByDefault(t *testing.T) {
	store := &fakeStore{
		diagnoseReport: workspace.DiagnosticReport{
			Workspace:    "game-a",
			WorkspaceDir: "/tmp/adp-home/workspaces/game-a",
			Diagnostics: []workspace.Diagnostic{{
				Level:   workspace.DiagnosticLevelInfo,
				Code:    workspace.DiagnosticCodeGitRootDetected,
				Message: "Git worktree detected",
				Path:    "/srv/game-a",
			}},
		},
	}
	var stdout bytes.Buffer

	code := NewApp(Dependencies{WorkspaceStore: store}, &stdout, &bytes.Buffer{}).Execute(context.Background(), []string{"workspace", "doctor", "game-a"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	output := stdout.String()
	for _, want := range []string{"game-a", "ok", "no issues"} {
		if !strings.Contains(output, want) {
			t.Fatalf("doctor output missing %q: %q", want, output)
		}
	}
	for _, hidden := range []string{"info", workspace.DiagnosticCodeGitRootDetected} {
		if strings.Contains(output, hidden) {
			t.Fatalf("doctor output should hide %q by default: %q", hidden, output)
		}
	}
}

func TestWorkspaceDoctorCommandVerboseShowsInfoDiagnostics(t *testing.T) {
	store := &fakeStore{
		diagnoseReport: workspace.DiagnosticReport{
			Workspace:    "game-a",
			WorkspaceDir: "/tmp/adp-home/workspaces/game-a",
			Diagnostics: []workspace.Diagnostic{{
				Level:   workspace.DiagnosticLevelInfo,
				Code:    workspace.DiagnosticCodeGitRootDetected,
				Message: "Git worktree detected",
				Path:    "/srv/game-a",
			}},
		},
	}
	var stdout bytes.Buffer

	code := NewApp(Dependencies{WorkspaceStore: store}, &stdout, &bytes.Buffer{}).Execute(context.Background(), []string{"workspace", "doctor", "game-a", "--verbose"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	output := stdout.String()
	for _, want := range []string{"game-a", "info", workspace.DiagnosticCodeGitRootDetected, "/srv/game-a"} {
		if !strings.Contains(output, want) {
			t.Fatalf("verbose doctor output missing %q: %q", want, output)
		}
	}
}

func TestWorkspaceDoctorCommandJSONIncludesInfoDiagnostics(t *testing.T) {
	store := &fakeStore{
		diagnoseReport: workspace.DiagnosticReport{
			Workspace:    "game-a",
			WorkspaceDir: "/tmp/adp-home/workspaces/game-a",
			ConfigPath:   "/tmp/adp-home/workspaces/game-a/workspace.yaml",
			Git: &workspace.GitDiagnosticContext{
				ProjectRoot:        "/srv/repo/packages/game-a",
				GitRoot:            "/srv/repo",
				GitDir:             "/srv/repo/.git",
				MetadataPath:       "/srv/repo/.git",
				MetadataKind:       "directory",
				InsideWorkTree:     true,
				ProjectBelowRoot:   true,
				RelativeProjectDir: "packages/game-a",
				Branch:             "main",
				Upstream:           "origin/main",
				Ahead:              1,
				ChangeState:        "dirty",
				ChangedEntries:     2,
				UntrackedEntries:   1,
				GitAvailable:       true,
			},
			Diagnostics: []workspace.Diagnostic{{
				Level:   workspace.DiagnosticLevelInfo,
				Code:    workspace.DiagnosticCodeGitRootDetected,
				Message: "Git worktree detected",
				Path:    "/srv/game-a",
			}},
		},
	}
	var stdout bytes.Buffer

	code := NewApp(Dependencies{WorkspaceStore: store}, &stdout, &bytes.Buffer{}).Execute(context.Background(), []string{"workspace", "doctor", "game-a", "--format", "json"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	var got workspaceDoctorJSONOutput
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("doctor JSON did not parse: %v\n%s", err, stdout.String())
	}
	if got.ReportCount != 1 || got.HasErrors {
		t.Fatalf("doctor JSON summary = %+v, want one report without errors", got)
	}
	if len(got.Reports) != 1 || got.Reports[0].DiagnosticCount != 1 || len(got.Reports[0].Diagnostics) != 1 {
		t.Fatalf("doctor JSON diagnostics = %+v, want one diagnostic", got.Reports)
	}
	if got.Reports[0].Git == nil {
		t.Fatalf("doctor JSON missing Git context: %+v", got.Reports[0])
	}
	git := got.Reports[0].Git
	if git.ProjectRoot != "/srv/repo/packages/game-a" || git.GitRoot != "/srv/repo" || !git.ProjectBelowRoot {
		t.Fatalf("doctor JSON Git roots mismatch: %+v", git)
	}
	if git.Branch != "main" || git.Upstream != "origin/main" || git.Ahead != 1 || git.Behind != 0 {
		t.Fatalf("doctor JSON Git branch mismatch: %+v", git)
	}
	if git.ChangeState != "dirty" || git.ChangedEntries != 2 || git.UntrackedEntries != 1 {
		t.Fatalf("doctor JSON Git status mismatch: %+v", git)
	}
	diagnostic := got.Reports[0].Diagnostics[0]
	if diagnostic.Level != string(workspace.DiagnosticLevelInfo) || diagnostic.Code != workspace.DiagnosticCodeGitRootDetected || diagnostic.Path != "/srv/game-a" {
		t.Fatalf("doctor JSON diagnostic = %+v, want Git info diagnostic", diagnostic)
	}
}

func TestWorkspaceDoctorCommandReturnsTwoWhenDiagnosticsHaveErrors(t *testing.T) {
	store := &fakeStore{
		diagnoseAllReports: []workspace.DiagnosticReport{{
			Workspace: "game-a",
			Diagnostics: []workspace.Diagnostic{{
				Level:   workspace.DiagnosticLevelError,
				Code:    workspace.DiagnosticCodeProjectRootMissing,
				Message: "project root is missing",
				Path:    "/srv/game-a",
			}},
		}},
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := NewApp(Dependencies{WorkspaceStore: store}, &stdout, &stderr).Execute(context.Background(), []string{"workspace", "doctor"})

	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if !store.diagnoseAllCalled {
		t.Fatal("DiagnoseAll was not called")
	}
	output := stdout.String()
	for _, want := range []string{"game-a", "error", workspace.DiagnosticCodeProjectRootMissing, "/srv/game-a"} {
		if !strings.Contains(output, want) {
			t.Fatalf("doctor output missing %q: %q", want, output)
		}
	}
}

func TestWorkspaceDoctorCommandKeepsZeroForWarningDiagnostics(t *testing.T) {
	store := &fakeStore{
		diagnoseReport: workspace.DiagnosticReport{
			Workspace:    "game-a",
			WorkspaceDir: "/tmp/adp-home/workspaces/game-a",
			Diagnostics: []workspace.Diagnostic{{
				Level:   workspace.DiagnosticLevelWarning,
				Code:    workspace.DiagnosticCodeAgentCommandArguments,
				Message: "agent command looks like it contains arguments",
				Path:    "/tmp/adp-home/workspaces/game-a/workspace.yaml",
			}},
		},
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := NewApp(Dependencies{WorkspaceStore: store}, &stdout, &stderr).Execute(context.Background(), []string{"workspace", "doctor", "game-a"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{"game-a", "warning", workspace.DiagnosticCodeAgentCommandArguments, "/tmp/adp-home/workspaces/game-a/workspace.yaml"} {
		if !strings.Contains(output, want) {
			t.Fatalf("workspace doctor output missing %q: %q", want, output)
		}
	}
}

func TestWorkspaceDoctorCommandReturnsTwoWhenNamedRuntimeParentDiagnosticsFail(t *testing.T) {
	store := &fakeStore{
		diagnoseReport: workspace.DiagnosticReport{
			Workspace:    "game-a",
			WorkspaceDir: "/tmp/adp-home/workspaces/game-a",
			Diagnostics: []workspace.Diagnostic{{
				Level:   workspace.DiagnosticLevelError,
				Code:    workspace.DiagnosticCodeRuntimeParentInsideProjectRoot,
				Message: "runtime parent must not be inside the project root",
				Path:    "/srv/game-a/.adp-runtime-parent",
			}},
		},
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := NewApp(Dependencies{WorkspaceStore: store}, &stdout, &stderr).Execute(context.Background(), []string{"workspace", "doctor", "game-a"})

	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if store.diagnoseName != "game-a" {
		t.Fatalf("Diagnose called with %q", store.diagnoseName)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{"game-a", "error", workspace.DiagnosticCodeRuntimeParentInsideProjectRoot, "/srv/game-a/.adp-runtime-parent"} {
		if !strings.Contains(output, want) {
			t.Fatalf("workspace doctor output missing %q: %q", want, output)
		}
	}
}

func TestWorkspaceCommandReportsUnknownSubcommand(t *testing.T) {
	var stderr bytes.Buffer

	code := NewApp(Dependencies{WorkspaceStore: &fakeStore{}}, &bytes.Buffer{}, &stderr).Execute(context.Background(), []string{"workspace", "bogus"})

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), `adp: unknown workspace command "bogus"`) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
