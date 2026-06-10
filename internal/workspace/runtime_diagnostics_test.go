package workspace

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRegistryDiagnoseReportsFilesystemRootRuntimeParent(t *testing.T) {
	registry, _ := newTestRegistry(t)
	projectRoot := createProject(t)
	registry.Layout.RuntimeParent = string(os.PathSeparator)

	if _, err := registry.Add(context.Background(), "game-a", projectRoot); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	report, err := registry.Diagnose(context.Background(), "game-a")
	if err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}

	assertDiagnostic(t, report, DiagnosticCodeRuntimeParentRoot, DiagnosticLevelError, string(os.PathSeparator))
	if !report.HasErrors() {
		t.Fatal("HasErrors() = false, want true")
	}
}

func TestRegistryDiagnoseReportsRuntimeParentProjectRootOverlap(t *testing.T) {
	tests := []struct {
		name          string
		runtimeParent func(string) string
		code          string
	}{
		{
			name:          "project-root",
			runtimeParent: func(projectRoot string) string { return projectRoot },
			code:          DiagnosticCodeRuntimeParentProjectRoot,
		},
		{
			name:          "inside-project-root",
			runtimeParent: func(projectRoot string) string { return filepath.Join(projectRoot, ".adp-runtime") },
			code:          DiagnosticCodeRuntimeParentInsideProjectRoot,
		},
		{
			name:          "contains-project-root",
			runtimeParent: func(projectRoot string) string { return filepath.Dir(projectRoot) },
			code:          DiagnosticCodeRuntimeParentContainsProject,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry, _ := newTestRegistry(t)
			projectRoot := createProject(t)
			registry.Layout.RuntimeParent = tt.runtimeParent(projectRoot)

			if _, err := registry.Add(context.Background(), "game-a", projectRoot); err != nil {
				t.Fatalf("Add() error = %v", err)
			}

			report, err := registry.Diagnose(context.Background(), "game-a")
			if err != nil {
				t.Fatalf("Diagnose() error = %v", err)
			}

			assertDiagnostic(t, report, tt.code, DiagnosticLevelError, registry.Layout.RuntimeParent)
			if !report.HasErrors() {
				t.Fatal("HasErrors() = false, want true")
			}
		})
	}
}

func TestRegistryDiagnoseReportsRuntimeParentSymlinkWarning(t *testing.T) {
	registry, _ := newTestRegistry(t)
	projectRoot := createProject(t)
	target := filepath.Join(t.TempDir(), "runtime-target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("create runtime target: %v", err)
	}
	runtimeParent := filepath.Join(t.TempDir(), "runtime-link")
	symlinkOrSkip(t, target, runtimeParent)
	registry.Layout.RuntimeParent = runtimeParent

	if _, err := registry.Add(context.Background(), "game-a", projectRoot); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	report, err := registry.Diagnose(context.Background(), "game-a")
	if err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}

	assertDiagnostic(t, report, DiagnosticCodeRuntimeParentSymlink, DiagnosticLevelWarning, runtimeParent)
	if report.HasErrors() {
		t.Fatalf("HasErrors() = true, want false: %+v", report.Diagnostics)
	}
}

func TestRegistryDiagnoseReportsRuntimeParentSymlinkTargetOverlap(t *testing.T) {
	tests := []struct {
		name   string
		target func(t *testing.T, projectRoot string) string
		code   string
	}{
		{
			name:   "target-project-root",
			target: func(t *testing.T, projectRoot string) string { return projectRoot },
			code:   DiagnosticCodeRuntimeParentProjectRoot,
		},
		{
			name: "target-inside-project-root",
			target: func(t *testing.T, projectRoot string) string {
				runtimeTarget := filepath.Join(projectRoot, ".adp-runtime-parent")
				if err := os.Mkdir(runtimeTarget, 0o755); err != nil {
					t.Fatalf("create runtime target: %v", err)
				}
				return runtimeTarget
			},
			code: DiagnosticCodeRuntimeParentInsideProjectRoot,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry, _ := newTestRegistry(t)
			projectRoot := createProject(t)
			runtimeParent := filepath.Join(t.TempDir(), "runtime-link")
			symlinkOrSkip(t, tt.target(t, projectRoot), runtimeParent)
			registry.Layout.RuntimeParent = runtimeParent

			if _, err := registry.Add(context.Background(), "game-a", projectRoot); err != nil {
				t.Fatalf("Add() error = %v", err)
			}

			report, err := registry.Diagnose(context.Background(), "game-a")
			if err != nil {
				t.Fatalf("Diagnose() error = %v", err)
			}

			assertDiagnostic(t, report, DiagnosticCodeRuntimeParentSymlink, DiagnosticLevelWarning, runtimeParent)
			assertDiagnostic(t, report, tt.code, DiagnosticLevelError, runtimeParent)
			if !report.HasErrors() {
				t.Fatal("HasErrors() = false, want true")
			}
		})
	}
}

func TestRegistryDiagnoseReportsRuntimeParentContainsResolvedProjectRoot(t *testing.T) {
	registry, _ := newTestRegistry(t)
	runtimeParent := filepath.Join(t.TempDir(), "runtime-parent")
	if err := os.Mkdir(runtimeParent, 0o755); err != nil {
		t.Fatalf("create runtime parent: %v", err)
	}
	realProjectRoot := filepath.Join(runtimeParent, "project")
	if err := os.Mkdir(realProjectRoot, 0o755); err != nil {
		t.Fatalf("create project root: %v", err)
	}
	projectRoot := filepath.Join(t.TempDir(), "project-link")
	symlinkOrSkip(t, realProjectRoot, projectRoot)
	registry.Layout.RuntimeParent = runtimeParent

	if _, err := registry.Add(context.Background(), "game-a", projectRoot); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	report, err := registry.Diagnose(context.Background(), "game-a")
	if err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}

	assertDiagnostic(t, report, DiagnosticCodeRuntimeParentContainsProject, DiagnosticLevelError, runtimeParent)
	if !report.HasErrors() {
		t.Fatal("HasErrors() = false, want true")
	}
}

func TestRegistryDiagnoseReportsRuntimeParentNotDirectory(t *testing.T) {
	registry, _ := newTestRegistry(t)
	projectRoot := createProject(t)
	runtimeParent := filepath.Join(t.TempDir(), "runtime-parent")
	writeFile(t, runtimeParent, "not a directory\n")
	registry.Layout.RuntimeParent = runtimeParent

	if _, err := registry.Add(context.Background(), "game-a", projectRoot); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	report, err := registry.Diagnose(context.Background(), "game-a")
	if err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}

	assertDiagnostic(t, report, DiagnosticCodeRuntimeParentNotDirectory, DiagnosticLevelError, runtimeParent)
	if !report.HasErrors() {
		t.Fatal("HasErrors() = false, want true")
	}
}

func TestRegistryDiagnoseAllowsMissingRuntimeParentOutsideProjectRoot(t *testing.T) {
	clearGitDirectiveEnv(t)

	registry, _ := newTestRegistry(t)
	projectRoot := createProject(t)
	initGitProject(t, projectRoot)
	registry.Layout.RuntimeParent = filepath.Join(t.TempDir(), "missing-runtime-parent")

	if _, err := registry.Add(context.Background(), "game-a", projectRoot); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	report, err := registry.Diagnose(context.Background(), "game-a")
	if err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}
	assertOnlyGitRootDetected(t, report, projectRoot)
}
