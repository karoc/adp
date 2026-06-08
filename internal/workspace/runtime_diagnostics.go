package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	DiagnosticCodeRuntimeParentMissing           = "workspace.runtime.parent.missing"
	DiagnosticCodeRuntimeParentResolveFailed     = "workspace.runtime.parent.resolve_failed"
	DiagnosticCodeRuntimeParentRoot              = "workspace.runtime.parent.root"
	DiagnosticCodeRuntimeParentProjectRoot       = "workspace.runtime.parent.project_root"
	DiagnosticCodeRuntimeParentInsideProjectRoot = "workspace.runtime.parent.inside_project_root"
	DiagnosticCodeRuntimeParentContainsProject   = "workspace.runtime.parent.contains_project_root"
	DiagnosticCodeRuntimeParentSymlink           = "workspace.runtime.parent.symlink"
	DiagnosticCodeRuntimeParentNotDirectory      = "workspace.runtime.parent.not_directory"
	DiagnosticCodeRuntimeParentStatFailed        = "workspace.runtime.parent.stat_failed"
)

func checkRuntimeParent(report *DiagnosticReport, runtimeParent string, projectRoot string) {
	runtimeParent = strings.TrimSpace(runtimeParent)
	if runtimeParent == "" {
		report.add(DiagnosticLevelError, DiagnosticCodeRuntimeParentMissing, "runtime parent is not configured", report.ConfigPath)
		return
	}

	runtimeAbs, err := filepath.Abs(runtimeParent)
	if err != nil {
		report.add(DiagnosticLevelError, DiagnosticCodeRuntimeParentResolveFailed, fmt.Sprintf("runtime parent could not be resolved: %v", err), runtimeParent)
		return
	}
	runtimeAbs = filepath.Clean(runtimeAbs)
	if filepath.Dir(runtimeAbs) == runtimeAbs {
		report.add(DiagnosticLevelError, DiagnosticCodeRuntimeParentRoot, "runtime parent must not be the filesystem root", runtimeAbs)
		return
	}

	checkRuntimeProjectOverlap(report, runtimeAbs, projectRoot)
	checkRuntimeParentPath(report, runtimeAbs)
}

func checkRuntimeProjectOverlap(report *DiagnosticReport, runtimeAbs string, projectRoot string) {
	projectRoot = strings.TrimSpace(projectRoot)
	if projectRoot == "" {
		return
	}
	projectAbs, err := filepath.Abs(projectRoot)
	if err != nil {
		return
	}
	projectAbs = filepath.Clean(projectAbs)

	runtimeCandidates := appendResolvedPath(nil, runtimeAbs)
	projectCandidates := appendResolvedPath(nil, projectAbs)

	switch {
	case pathsOverlap(runtimeCandidates, projectCandidates, sameCleanPath):
		report.add(DiagnosticLevelError, DiagnosticCodeRuntimeParentProjectRoot, "runtime parent must not be the project root", runtimeAbs)
	case pathsOverlap(projectCandidates, runtimeCandidates, pathInsideDir):
		report.add(DiagnosticLevelError, DiagnosticCodeRuntimeParentInsideProjectRoot, "runtime parent must not be inside the project root", runtimeAbs)
	case pathsOverlap(runtimeCandidates, projectCandidates, pathInsideDir):
		report.add(DiagnosticLevelError, DiagnosticCodeRuntimeParentContainsProject, "runtime parent must not contain the project root", runtimeAbs)
	}
}

func checkRuntimeParentPath(report *DiagnosticReport, runtimeAbs string) {
	info, err := os.Lstat(runtimeAbs)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		report.add(DiagnosticLevelWarning, DiagnosticCodeRuntimeParentStatFailed, fmt.Sprintf("runtime parent could not be inspected: %v", err), runtimeAbs)
		return
	}
	if info.Mode()&os.ModeSymlink != 0 {
		report.add(DiagnosticLevelWarning, DiagnosticCodeRuntimeParentSymlink, "runtime parent is a symlink; use a direct local directory for clearer runtime cleanup boundaries", runtimeAbs)
		targetInfo, err := os.Stat(runtimeAbs)
		if err != nil {
			report.add(DiagnosticLevelWarning, DiagnosticCodeRuntimeParentStatFailed, fmt.Sprintf("runtime parent symlink target could not be inspected: %v", err), runtimeAbs)
			return
		}
		if !targetInfo.IsDir() {
			report.add(DiagnosticLevelError, DiagnosticCodeRuntimeParentNotDirectory, "runtime parent symlink target is not a directory", runtimeAbs)
		}
		return
	}
	if !info.IsDir() {
		report.add(DiagnosticLevelError, DiagnosticCodeRuntimeParentNotDirectory, "runtime parent exists but is not a directory", runtimeAbs)
	}
}

func sameCleanPath(left string, right string) bool {
	rel, err := filepath.Rel(filepath.Clean(left), filepath.Clean(right))
	return err == nil && rel == "."
}

func appendResolvedPath(paths []string, path string) []string {
	path = filepath.Clean(path)
	paths = appendUniqueCleanPath(paths, path)

	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return paths
	}
	resolvedAbs, err := filepath.Abs(resolved)
	if err != nil {
		return paths
	}
	return appendUniqueCleanPath(paths, resolvedAbs)
}

func appendUniqueCleanPath(paths []string, path string) []string {
	path = filepath.Clean(path)
	for _, existing := range paths {
		if sameCleanPath(existing, path) {
			return paths
		}
	}
	return append(paths, path)
}

func pathsOverlap(leftPaths []string, rightPaths []string, match func(string, string) bool) bool {
	for _, left := range leftPaths {
		for _, right := range rightPaths {
			if match(left, right) {
				return true
			}
		}
	}
	return false
}
