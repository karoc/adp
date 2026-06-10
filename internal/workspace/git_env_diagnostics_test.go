package workspace

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/karoc/adp/internal/gitenv"
)

func TestRegistryDiagnoseReportsInheritedGitRepositoryDirectives(t *testing.T) {
	clearGitDirectiveEnv(t)
	t.Setenv("GIT_DIR", filepath.Join(t.TempDir(), ".git"))
	t.Setenv("GIT_WORK_TREE", t.TempDir())
	t.Setenv("GIT_INDEX_FILE", filepath.Join(t.TempDir(), "index"))

	registry, _ := newTestRegistry(t)
	projectRoot := createProject(t)
	if _, err := registry.Add(context.Background(), "game-a", projectRoot); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	report, err := registry.Diagnose(context.Background(), "game-a")
	if err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}

	assertDiagnostic(t, report, DiagnosticCodeGitEnvRepositoryDirective, DiagnosticLevelWarning, "")
	assertDiagnosticMessageContains(t, report, DiagnosticCodeGitEnvRepositoryDirective, "GIT_DIR")
	assertDiagnosticMessageContains(t, report, DiagnosticCodeGitEnvRepositoryDirective, "GIT_WORK_TREE")
	assertDiagnosticMessageContains(t, report, DiagnosticCodeGitEnvRepositoryDirective, "GIT_INDEX_FILE")
	assertDiagnosticMessageContains(t, report, DiagnosticCodeGitEnvRepositoryDirective, "ADP runtime neutralizes these")
	assertDiagnosticMessageContains(t, report, DiagnosticCodeGitEnvRepositoryDirective, "Git commands should target ADP_PROJECT_ROOT")
	if report.HasErrors() {
		t.Fatalf("HasErrors() = true, want false: %+v", report.Diagnostics)
	}
}

func TestRegistryDiagnoseAllReportsInheritedGitRepositoryDirectives(t *testing.T) {
	clearGitDirectiveEnv(t)
	t.Setenv("GIT_COMMON_DIR", filepath.Join(t.TempDir(), "common"))

	registry, _ := newTestRegistry(t)
	projectRoot := createProject(t)
	if _, err := registry.Add(context.Background(), "game-a", projectRoot); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	reports, err := registry.DiagnoseAll(context.Background())
	if err != nil {
		t.Fatalf("DiagnoseAll() error = %v", err)
	}

	report := reportByWorkspace(t, reports, "game-a")
	assertDiagnostic(t, report, DiagnosticCodeGitEnvRepositoryDirective, DiagnosticLevelWarning, "")
	assertDiagnosticMessageContains(t, report, DiagnosticCodeGitEnvRepositoryDirective, "GIT_COMMON_DIR")
}

func clearGitDirectiveEnv(t *testing.T) {
	t.Helper()

	previous := make(map[string]string)
	present := make(map[string]bool)
	for _, name := range gitenv.RepositoryDirectiveNames() {
		if value, ok := os.LookupEnv(name); ok {
			previous[name] = value
			present[name] = true
		}
		if err := os.Unsetenv(name); err != nil {
			t.Fatalf("unset %s: %v", name, err)
		}
	}

	t.Cleanup(func() {
		for _, name := range gitenv.RepositoryDirectiveNames() {
			if !present[name] {
				_ = os.Unsetenv(name)
				continue
			}
			_ = os.Setenv(name, previous[name])
		}
	})
}

func assertNoGitEnvDiagnostic(t *testing.T, report DiagnosticReport) {
	t.Helper()

	for _, diagnostic := range report.Diagnostics {
		if diagnostic.Code == DiagnosticCodeGitEnvRepositoryDirective {
			t.Fatalf("unexpected Git environment diagnostic: %+v", report.Diagnostics)
		}
		if strings.Contains(diagnostic.Message, "repository-directing Git variables") {
			t.Fatalf("unexpected Git environment diagnostic message: %+v", report.Diagnostics)
		}
	}
}
