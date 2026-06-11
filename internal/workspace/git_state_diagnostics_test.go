package workspace

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/karoc/adp/internal/gitstate"
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
	assertGitContext(t, report, func(git *GitDiagnosticContext) {
		if git.ProjectRoot != projectRoot || git.GitRoot != projectRoot {
			t.Fatalf("git roots = (%q, %q), want %q", git.ProjectRoot, git.GitRoot, projectRoot)
		}
		if !git.GitAvailable || !git.InsideWorkTree || git.ProjectBelowRoot {
			t.Fatalf("git topology mismatch: %+v", git)
		}
		if git.MetadataKind != string(gitstate.MetadataDirectory) || git.ChangeState != string(gitstate.ChangeClean) {
			t.Fatalf("git metadata/status mismatch: %+v", git)
		}
	})
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
	assertGitContext(t, report, func(git *GitDiagnosticContext) {
		if git.ProjectRoot != projectRoot || git.GitRoot != "" || git.GitDir != "" {
			t.Fatalf("non-git context roots mismatch: %+v", git)
		}
		if !git.GitAvailable || git.InsideWorkTree || git.ChangeState != string(gitstate.ChangeError) {
			t.Fatalf("non-git context mismatch: %+v", git)
		}
		if git.InspectionError == "" {
			t.Fatalf("non-git context missing inspection error: %+v", git)
		}
	})
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
	assertGitContext(t, report, func(git *GitDiagnosticContext) {
		if git.ProjectRoot != projectRoot || git.GitRoot != repoRoot {
			t.Fatalf("nested git roots = (%q, %q), want (%q, %q)", git.ProjectRoot, git.GitRoot, projectRoot, repoRoot)
		}
		if !git.ProjectBelowRoot || git.RelativeProjectDir != filepath.Join("packages", "app") {
			t.Fatalf("nested git context mismatch: %+v", git)
		}
	})
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
	assertGitContext(t, report, func(git *GitDiagnosticContext) {
		if git.ChangeState != string(gitstate.ChangeDirty) || git.ChangedEntries != 1 || git.UntrackedEntries != 1 {
			t.Fatalf("dirty git context mismatch: %+v", git)
		}
	})
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
	assertGitContext(t, report, func(git *GitDiagnosticContext) {
		if git.MetadataKind != string(gitstate.MetadataFile) || git.MetadataPath != filepath.Join(worktreeRoot, ".git") {
			t.Fatalf("gitfile context mismatch: %+v", git)
		}
	})
}

func assertGitContext(t *testing.T, report DiagnosticReport, check func(*GitDiagnosticContext)) {
	t.Helper()
	if report.Git == nil {
		t.Fatalf("report Git context is nil: %+v", report)
	}
	check(report.Git)
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
