package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/karoc/adp/internal/workspace"
)

func TestDoctorCommandDelegatesToWorkspaceDiagnostics(t *testing.T) {
	store := &fakeStore{
		diagnoseReport: workspace.DiagnosticReport{
			Workspace:    "game-a",
			WorkspaceDir: "/tmp/adp-home/workspaces/game-a",
		},
	}
	var stdout bytes.Buffer

	code := NewApp(Dependencies{WorkspaceStore: store}, &stdout, &bytes.Buffer{}).Execute(
		context.Background(),
		[]string{"doctor", "game-a"},
	)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if store.diagnoseName != "game-a" {
		t.Fatalf("Diagnose called with %q", store.diagnoseName)
	}
	if output := stdout.String(); !strings.Contains(output, "game-a") || !strings.Contains(output, "ok") {
		t.Fatalf("doctor output missing healthy report: %q", output)
	}
}

func TestDoctorCommandReturnsTwoWhenRuntimeParentDiagnosticsFail(t *testing.T) {
	store := &fakeStore{
		diagnoseReport: workspace.DiagnosticReport{
			Workspace:    "game-a",
			WorkspaceDir: "/tmp/adp-home/workspaces/game-a",
			Diagnostics: []workspace.Diagnostic{{
				Level:   workspace.DiagnosticLevelError,
				Code:    workspace.DiagnosticCodeRuntimeParentProjectRoot,
				Message: "runtime parent must not be the project root",
				Path:    "/srv/game-a",
			}},
		},
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := NewApp(Dependencies{WorkspaceStore: store}, &stdout, &stderr).Execute(
		context.Background(),
		[]string{"doctor", "game-a"},
	)

	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	for _, want := range []string{"game-a", "error", workspace.DiagnosticCodeRuntimeParentProjectRoot, "/srv/game-a"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("doctor output missing %q: %q", want, stdout.String())
		}
	}
}

func TestDoctorCommandKeepsZeroForWarningDiagnostics(t *testing.T) {
	store := &fakeStore{
		diagnoseReport: workspace.DiagnosticReport{
			Workspace:    "game-a",
			WorkspaceDir: "/tmp/adp-home/workspaces/game-a",
			Diagnostics: []workspace.Diagnostic{{
				Level:   workspace.DiagnosticLevelWarning,
				Code:    workspace.DiagnosticCodeAgentProfileAmbiguous,
				Message: "profile matches multiple profile files",
				Path:    "/tmp/adp-home/workspaces/game-a/profiles/senior.{md,yaml,yml,json}",
			}},
		},
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := NewApp(Dependencies{WorkspaceStore: store}, &stdout, &stderr).Execute(
		context.Background(),
		[]string{"doctor", "game-a"},
	)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	for _, want := range []string{"game-a", "warning", workspace.DiagnosticCodeAgentProfileAmbiguous, "/tmp/adp-home/workspaces/game-a/profiles/senior.{md,yaml,yml,json}"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("doctor output missing %q: %q", want, stdout.String())
		}
	}
}

func TestDoctorCommandJSONIncludesCompleteDiagnosticsAndReturnsTwoForErrors(t *testing.T) {
	store := &fakeStore{
		diagnoseReport: workspace.DiagnosticReport{
			Workspace:    "game-a",
			WorkspaceDir: "/tmp/adp-home/workspaces/game-a",
			Diagnostics: []workspace.Diagnostic{
				{
					Level:   workspace.DiagnosticLevelInfo,
					Code:    workspace.DiagnosticCodeGitRootDetected,
					Message: "Git worktree detected",
					Path:    "/srv/game-a",
				},
				{
					Level:   workspace.DiagnosticLevelError,
					Code:    workspace.DiagnosticCodeRuntimeParentProjectRoot,
					Message: "runtime parent must not be the project root",
					Path:    "/srv/game-a",
				},
			},
		},
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := NewApp(Dependencies{WorkspaceStore: store}, &stdout, &stderr).Execute(
		context.Background(),
		[]string{"doctor", "--format", "json", "game-a"},
	)

	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	var got workspaceDoctorJSONOutput
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("doctor JSON did not parse: %v\n%s", err, stdout.String())
	}
	if !got.HasErrors || len(got.Reports) != 1 || !got.Reports[0].HasErrors {
		t.Fatalf("doctor JSON error summary = %+v, want errors", got)
	}
	if got.Reports[0].DiagnosticCount != 2 || len(got.Reports[0].Diagnostics) != 2 {
		t.Fatalf("doctor JSON diagnostics = %+v, want complete diagnostics", got.Reports[0])
	}
	for _, want := range []string{workspace.DiagnosticCodeGitRootDetected, workspace.DiagnosticCodeRuntimeParentProjectRoot} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("doctor JSON missing %q: %s", want, stdout.String())
		}
	}
}

func TestVersionCommandPrintsVersionMetadata(t *testing.T) {
	withVersion("1.2.3", "abc123", "2026-06-08T00:00:00Z", func() {
		var stdout bytes.Buffer

		code := NewApp(Dependencies{}, &stdout, &bytes.Buffer{}).Execute(context.Background(), []string{"version"})

		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		want := "adp 1.2.3 commit abc123 built 2026-06-08T00:00:00Z\n"
		if got := stdout.String(); got != want {
			t.Fatalf("stdout = %q, want %q", got, want)
		}
	})
}

func TestTopLevelVersionFlagPrintsVersionBeforeInit(t *testing.T) {
	withVersion("", "", "", func() {
		var stdout bytes.Buffer

		code := NewApp(Dependencies{InitError: errors.New("boom")}, &stdout, &bytes.Buffer{}).Execute(context.Background(), []string{"--version"})

		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		if got, want := stdout.String(), "adp dev\n"; got != want {
			t.Fatalf("stdout = %q, want %q", got, want)
		}
	})
}

func withVersion(version string, commit string, buildDate string, fn func()) {
	oldVersion, oldCommit, oldBuildDate := Version, Commit, BuildDate
	Version, Commit, BuildDate = version, commit, buildDate
	defer func() {
		Version, Commit, BuildDate = oldVersion, oldCommit, oldBuildDate
	}()
	fn()
}
