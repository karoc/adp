package e2e

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Unique markers written into the workspace source files. Each marker must
// survive verbatim into the generated CLAUDE.md / AGENTS.md, proving that the
// real file-reading branches in shared.Instructions actually inject content at
// runtime (not just in unit tests with fake paths).
const (
	markerMemory  = "MEMORY-MARKER-9137-remember-the-invariants"
	markerBase    = "BASEPROMPT-MARKER-4471-obey-the-house-rules"
	markerMCP     = "MCPCONFIG-MARKER-5522-filesystem-server"
	markerProfile = "PROFILE-MARKER-7788-senior-reviewer"
	markerRule    = "- coding_style: strict" // default rule from workspace add
)

func TestRunInjectsWorkspaceContentIntoInstructions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fake agents are POSIX-only")
	}

	tmp := t.TempDir()
	projectRoot := filepath.Join(tmp, "project")
	binDir := filepath.Join(tmp, "bin")
	adpHome := filepath.Join(tmp, "adp-home")
	runtimeDir := filepath.Join(tmp, "runtime")
	adpBin := filepath.Join(tmp, "adp")
	mkdirAll(t, projectRoot, binDir, runtimeDir)
	buildADP(t, adpBin)
	writeFile(t, filepath.Join(projectRoot, "go.mod"), "module example.com/game\n")
	writeFile(t, filepath.Join(projectRoot, "main.go"), "package main\n")
	initGitProject(t, projectRoot)

	// Fake agents dump the generated instructions file to stdout so the test
	// can assert the injected markers round-tripped into the file the agent
	// actually receives.
	writeExecutable(t, filepath.Join(binDir, "codex"), dumpAgentScript("codex", "AGENTS.md"))
	writeExecutable(t, filepath.Join(binDir, "claude"), dumpAgentScript("claude", "CLAUDE.md"))

	env := append(os.Environ(),
		"ADP_HOME="+adpHome,
		"ADP_RUNTIME_DIR="+runtimeDir,
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
	)

	repoRoot := repositoryRoot(t)
	runADP(t, adpBin, repoRoot, env, "init")
	runADP(t, adpBin, repoRoot, env, "workspace", "add", "game-a", projectRoot)

	// Overwrite the workspace source files (created by `workspace add`) with
	// unique marker content. The workspace.yaml already enables and points at
	// each of these, so no config change is needed.
	workspaceDir := filepath.Join(adpHome, "workspaces", "game-a")
	writeFile(t, filepath.Join(workspaceDir, "memory", "shared.md"), "# Shared Memory\n\n"+markerMemory+"\n")
	writeFile(t, filepath.Join(workspaceDir, "prompts", "base.md"), "# Base Prompt\n\n"+markerBase+"\n")
	writeFile(t, filepath.Join(workspaceDir, "mcp", "config.yaml"), "servers:\n  fs: {} # "+markerMCP+"\n")
	writeFile(t, filepath.Join(workspaceDir, "profiles", "codex.yaml"), "profile: default\ncommand: codex\nnote: "+markerProfile+"\n")
	writeFile(t, filepath.Join(workspaceDir, "profiles", "claude.yaml"), "profile: default\ncommand: claude\nnote: "+markerProfile+"\n")

	codexOut := runADP(t, adpBin, repoRoot, env, "run", "codex", "--workspace", "game-a", "--", "--probe")
	claudeOut := runADP(t, adpBin, repoRoot, env, "run", "claude", "--workspace", "game-a", "--", "--probe")

	assertInjected(t, "codex/AGENTS.md", codexOut)
	assertInjected(t, "claude/CLAUDE.md", claudeOut)
}

func assertInjected(t *testing.T, label, out string) {
	t.Helper()
	for _, marker := range []string{markerMemory, markerBase, markerMCP, markerProfile, markerRule} {
		if !strings.Contains(out, marker) {
			t.Fatalf("%s: generated instructions missing injected content %q\n---\n%s", label, marker, out)
		}
	}
}

func dumpAgentScript(agent, instructions string) string {
	return `#!/usr/bin/env sh
set -eu
printf 'fake-` + agent + `-begin\n'
cat "` + instructions + `"
printf 'fake-` + agent + `-end\n'
`
}
