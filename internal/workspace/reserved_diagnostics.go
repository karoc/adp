package workspace

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/karoc/adp/internal/schema"
)

var globalProjectReservedPaths = []string{
	".adp-runtime.yaml",
	"planning",
	"tasks.yaml",
	"phases.yaml",
	"progress.jsonl",
}

var agentProjectReservedPaths = map[string][]string{
	"codex": {
		"AGENTS.md",
		".codex/config.toml",
	},
	"claude": {
		"CLAUDE.md",
		".claude/settings.json",
	},
}

func checkProjectReservedPaths(report *DiagnosticReport, projectRoot string, agents map[string]schema.AgentConfig) {
	info, err := os.Stat(projectRoot)
	if err != nil || !info.IsDir() {
		return
	}

	for _, rel := range reservedProjectPaths(agents) {
		checkProjectReservedPath(report, projectRoot, rel)
	}
}

func reservedProjectPaths(agents map[string]schema.AgentConfig) []string {
	seen := map[string]struct{}{}
	paths := make([]string, 0, len(globalProjectReservedPaths)+4)
	for _, rel := range globalProjectReservedPaths {
		paths = appendUniqueReservedPath(paths, seen, rel)
	}
	for name, agent := range agents {
		if !agent.Enabled {
			continue
		}
		for _, rel := range agentProjectReservedPaths[name] {
			paths = appendUniqueReservedPath(paths, seen, rel)
		}
	}
	return paths
}

func appendUniqueReservedPath(paths []string, seen map[string]struct{}, rel string) []string {
	if _, ok := seen[rel]; ok {
		return paths
	}
	seen[rel] = struct{}{}
	return append(paths, rel)
}

func checkProjectReservedPath(report *DiagnosticReport, projectRoot string, rel string) {
	path := filepath.Join(projectRoot, rel)
	if _, err := os.Lstat(path); err != nil {
		if os.IsNotExist(err) {
			return
		}
		report.add(DiagnosticLevelWarning, DiagnosticCodeProjectRootReservedStat, fmt.Sprintf("reserved ADP path %q could not be inspected: %v", rel, err), path)
		return
	}
	report.add(DiagnosticLevelWarning, DiagnosticCodeProjectRootReservedPath, fmt.Sprintf("project root contains reserved ADP path %q; keep ADP-generated runtime and planning files outside the real project root", rel), path)
}
