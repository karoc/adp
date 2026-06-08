package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRunCodexAndClaudeWithRuntimeOverlay(t *testing.T) {
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
	writeExecutable(t, filepath.Join(binDir, "codex"), fakeAgentScript("codex", "AGENTS.md", ".codex/config.toml", "go.mod"))
	writeExecutable(t, filepath.Join(binDir, "claude"), fakeAgentScript("claude", "CLAUDE.md", ".claude/settings.json", "main.go"))

	env := append(os.Environ(),
		"ADP_HOME="+adpHome,
		"ADP_RUNTIME_DIR="+runtimeDir,
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
	)

	repoRoot := repositoryRoot(t)
	runADP(t, adpBin, repoRoot, env, "init")
	runADP(t, adpBin, repoRoot, env, "workspace", "add", "game-a", projectRoot)
	codexOut := runADP(t, adpBin, repoRoot, env, "run", "codex", "--workspace", "game-a", "--", "--probe")
	claudeOut := runADP(t, adpBin, projectRoot, env, "run", "claude", "--", "--probe")

	if !strings.Contains(codexOut, "fake-codex") || !strings.Contains(claudeOut, "fake-claude") {
		t.Fatalf("fake agents did not run:\ncodex=%s\nclaude=%s", codexOut, claudeOut)
	}
	assertEventLines(t, filepath.Join(adpHome, "logs", "events.jsonl"), 4)
	assertProjectClean(t, projectRoot)
	assertRuntimeCleaned(t, runtimeDir)
}

func buildADP(t *testing.T, output string) {
	t.Helper()

	cmd := exec.Command("go", "build", "-o", output, "./cmd/adp")
	cmd.Dir = repositoryRoot(t)
	outputBytes, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build adp failed: %v\n%s", err, outputBytes)
	}
}

func runADP(t *testing.T, adpBin string, dir string, env []string, args ...string) string {
	t.Helper()

	cmd := exec.Command(adpBin, args...)
	cmd.Dir = dir
	cmd.Env = env

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("adp %v failed: %v\n%s", args, err, output)
	}
	return string(output)
}

func repositoryRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func fakeAgentScript(agent, instructions, config, linked string) string {
	return `#!/usr/bin/env sh
set -eu
printf 'fake-` + agent + ` cwd=%s args=%s\n' "$(pwd)" "$*"
test "$ADP_WORKSPACE" = "game-a"
test -n "$ADP_SESSION_ID"
test -n "$ADP_RUNTIME_ROOT"
test -f "` + instructions + `"
test -f "` + config + `"
test -L "` + linked + `"
test "$1" = "--probe"
`
}

func assertEventLines(t *testing.T, path string, want int) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read event log: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != want {
		t.Fatalf("event line count = %d, want %d\n%s", len(lines), want, data)
	}
}

func assertProjectClean(t *testing.T, projectRoot string) {
	t.Helper()
	for _, rel := range []string{"AGENTS.md", "CLAUDE.md", ".codex", ".claude"} {
		if _, err := os.Lstat(filepath.Join(projectRoot, rel)); err == nil {
			t.Fatalf("project root was polluted with %s", rel)
		} else if !os.IsNotExist(err) {
			t.Fatalf("inspect project path %s: %v", rel, err)
		}
	}
}

func assertRuntimeCleaned(t *testing.T, runtimeDir string) {
	t.Helper()
	entries, err := os.ReadDir(runtimeDir)
	if err != nil {
		t.Fatalf("read runtime dir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("runtime dir should be cleaned, found %d entries", len(entries))
	}
}

func mkdirAll(t *testing.T, paths ...string) {
	t.Helper()
	for _, path := range paths {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeExecutable(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}
