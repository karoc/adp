package workspace

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRegistryDiagnoseHealthyGitWorkspace(t *testing.T) {
	clearGitDirectiveEnv(t)

	registry, layout := newTestRegistry(t)
	projectRoot := createProject(t)
	initGitProject(t, projectRoot)

	if _, err := registry.Add(context.Background(), "game-a", projectRoot); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	report, err := registry.Diagnose(context.Background(), "game-a")
	if err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}
	if report.Workspace != "game-a" {
		t.Fatalf("Workspace = %q, want game-a", report.Workspace)
	}
	if report.WorkspaceDir != layout.WorkspaceDir("game-a") {
		t.Fatalf("WorkspaceDir = %q, want %q", report.WorkspaceDir, layout.WorkspaceDir("game-a"))
	}
	if report.ConfigPath != layout.WorkspaceConfig("game-a") {
		t.Fatalf("ConfigPath = %q, want %q", report.ConfigPath, layout.WorkspaceConfig("game-a"))
	}
	assertOnlyGitRootDetected(t, report, projectRoot)
	if report.HasErrors() {
		t.Fatal("HasErrors() = true, want false")
	}
}

func TestRegistryDiagnoseReportsNonGitProject(t *testing.T) {
	clearGitDirectiveEnv(t)

	registry, _ := newTestRegistry(t)
	projectRoot := createProject(t)
	if _, err := registry.Add(context.Background(), "game-a", projectRoot); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	report, err := registry.Diagnose(context.Background(), "game-a")
	if err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}

	assertDiagnostic(t, report, DiagnosticCodeGitRootAbsent, DiagnosticLevelWarning, projectRoot)
	assertDiagnosticMessageContains(t, report, DiagnosticCodeGitRootAbsent, "ADP can still run")
	if report.HasErrors() {
		t.Fatalf("HasErrors() = true, want false: %+v", report.Diagnostics)
	}
}

func TestRegistryDiagnoseReportsNestedGitProjectRoot(t *testing.T) {
	clearGitDirectiveEnv(t)

	registry, _ := newTestRegistry(t)
	repoRoot := createProject(t)
	initGitProject(t, repoRoot)
	projectRoot := filepath.Join(repoRoot, "packages", "app")
	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		t.Fatalf("create nested project root: %v", err)
	}
	writeFile(t, filepath.Join(projectRoot, "app.txt"), "nested\n")
	mustGit(t, repoRoot, "add", filepath.Join("packages", "app"))
	mustGit(t, repoRoot, "commit", "-q", "-m", "add nested project")

	if _, err := registry.Add(context.Background(), "game-a", projectRoot); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	report, err := registry.Diagnose(context.Background(), "game-a")
	if err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}

	assertDiagnostic(t, report, DiagnosticCodeGitRootDetected, DiagnosticLevelInfo, repoRoot)
	assertDiagnostic(t, report, DiagnosticCodeGitRootNested, DiagnosticLevelInfo, projectRoot)
	assertDiagnosticMessageContains(t, report, DiagnosticCodeGitRootNested, "ADP_GIT_ROOT")
}

func TestRegistryDiagnoseReportsDirtyGitStatus(t *testing.T) {
	clearGitDirectiveEnv(t)

	registry, _ := newTestRegistry(t)
	projectRoot := createProject(t)
	initGitProject(t, projectRoot)
	writeFile(t, filepath.Join(projectRoot, "untracked.txt"), "dirty\n")

	if _, err := registry.Add(context.Background(), "game-a", projectRoot); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	report, err := registry.Diagnose(context.Background(), "game-a")
	if err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}

	assertDiagnostic(t, report, DiagnosticCodeGitRootDetected, DiagnosticLevelInfo, projectRoot)
	assertDiagnostic(t, report, DiagnosticCodeGitStatusDirty, DiagnosticLevelWarning, projectRoot)
	assertDiagnosticMessageContains(t, report, DiagnosticCodeGitStatusDirty, "untracked")
}

func TestRegistryDiagnoseReportsGitfileMetadata(t *testing.T) {
	clearGitDirectiveEnv(t)

	registry, _ := newTestRegistry(t)
	repoRoot := createProject(t)
	initGitProject(t, repoRoot)
	worktreeRoot := filepath.Join(t.TempDir(), "linked-worktree")
	mustGit(t, repoRoot, "worktree", "add", "--detach", worktreeRoot, "HEAD")

	if _, err := registry.Add(context.Background(), "game-a", worktreeRoot); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	report, err := registry.Diagnose(context.Background(), "game-a")
	if err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}

	assertDiagnostic(t, report, DiagnosticCodeGitRootDetected, DiagnosticLevelInfo, worktreeRoot)
	assertDiagnostic(t, report, DiagnosticCodeGitMetadataFile, DiagnosticLevelInfo, filepath.Join(worktreeRoot, ".git"))
	assertDiagnosticMessageContains(t, report, DiagnosticCodeGitMetadataFile, "worktree or submodule")
}

func assertOnlyGitRootDetected(t *testing.T, report DiagnosticReport, projectRoot string) {
	t.Helper()

	if len(report.Diagnostics) != 1 {
		t.Fatalf("Diagnostics = %+v, want only Git root info", report.Diagnostics)
	}
	assertDiagnostic(t, report, DiagnosticCodeGitRootDetected, DiagnosticLevelInfo, projectRoot)
	if report.HasErrors() {
		t.Fatalf("HasErrors() = true, want false: %+v", report.Diagnostics)
	}
}
