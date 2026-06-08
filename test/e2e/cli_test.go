package e2e

import (
	"encoding/json"
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
	listOut := runADP(t, adpBin, repoRoot, env, "workspace", "list")
	if !strings.Contains(listOut, "game-a") || !strings.Contains(listOut, projectRoot) {
		t.Fatalf("workspace list missing registered project: %q", listOut)
	}
	showOut := runADP(t, adpBin, repoRoot, env, "workspace", "show", "game-a")
	if !strings.Contains(showOut, "name: game-a") || !strings.Contains(showOut, "project_root: "+projectRoot) {
		t.Fatalf("workspace show missing details: %q", showOut)
	}
	doctorOut := runADP(t, adpBin, repoRoot, env, "workspace", "doctor", "game-a")
	if !strings.Contains(doctorOut, "game-a") || !strings.Contains(doctorOut, "ok") {
		t.Fatalf("workspace doctor missing healthy report: %q", doctorOut)
	}
	envOut := runADP(t, adpBin, repoRoot, env, "env", "game-a", "--cd")
	runtimeRoot := parseExport(t, envOut, "ADP_RUNTIME_ROOT")
	if !strings.Contains(envOut, "cd '"+runtimeRoot+"'") {
		t.Fatalf("env output missing cd to runtime root: %q", envOut)
	}
	assertFileExists(t, filepath.Join(runtimeRoot, ".adp-runtime.yaml"))
	assertProjectClean(t, projectRoot)

	hookOut := runADP(t, adpBin, repoRoot, env, "shell-hook", "--shell", "bash")
	if !strings.Contains(hookOut, "adp-enter()") || !strings.Contains(hookOut, `adp env "$1" --cd`) {
		t.Fatalf("shell-hook output missing bash hook: %q", hookOut)
	}
	completionOut := runADP(t, adpBin, repoRoot, env, "completion", "--shell", "bash")
	if !strings.Contains(completionOut, "complete -F _adp_completion adp") || !strings.Contains(completionOut, "sessions") {
		t.Fatalf("completion output missing bash completion: %q", completionOut)
	}

	codexOut := runADP(t, adpBin, repoRoot, env, "run", "codex", "--workspace", "game-a", "--", "--probe")
	claudeOut := runADP(t, adpBin, projectRoot, env, "run", "claude", "--", "--probe")

	if !strings.Contains(codexOut, "fake-codex") || !strings.Contains(claudeOut, "fake-claude") {
		t.Fatalf("fake agents did not run:\ncodex=%s\nclaude=%s", codexOut, claudeOut)
	}
	assertEventLines(t, filepath.Join(adpHome, "logs", "events.jsonl"), 4)
	eventsOut := runADP(t, adpBin, repoRoot, env, "events", "list", "--workspace", "game-a", "--type", "run_finished", "--limit", "2")
	if !strings.Contains(eventsOut, "run_finished") || !strings.Contains(eventsOut, "codex") || !strings.Contains(eventsOut, "claude") {
		t.Fatalf("events list missing run history: %q", eventsOut)
	}
	sessionIDs := sessionIDsByAgent(t, filepath.Join(adpHome, "logs", "events.jsonl"))
	codexSession := sessionIDs["codex"]
	if codexSession == "" {
		t.Fatalf("codex session id missing in event log: %#v", sessionIDs)
	}
	sessionsOut := runADP(t, adpBin, repoRoot, env, "sessions", "list", "--workspace", "game-a", "--agent", "codex")
	if !strings.Contains(sessionsOut, codexSession) || !strings.Contains(sessionsOut, "codex") {
		t.Fatalf("sessions list missing codex session: %q", sessionsOut)
	}
	sessionOut := runADP(t, adpBin, repoRoot, env, "sessions", "show", codexSession)
	if !strings.Contains(sessionOut, "session_id: "+codexSession) || !strings.Contains(sessionOut, "run_started") || !strings.Contains(sessionOut, "run_finished") {
		t.Fatalf("sessions show missing session detail: %q", sessionOut)
	}
	assertProjectClean(t, projectRoot)
	assertRuntimeEntries(t, runtimeDir, 1)
	pruneDryRunOut := runADP(t, adpBin, repoRoot, env, "runtime", "prune", "--older-than", "0s", "--include-kept", "--dry-run")
	if !strings.Contains(pruneDryRunOut, "would-remove") || !strings.Contains(pruneDryRunOut, runtimeRoot) {
		t.Fatalf("runtime prune dry-run missing kept runtime: %q", pruneDryRunOut)
	}
	assertRuntimeEntries(t, runtimeDir, 1)
	pruneOut := runADP(t, adpBin, repoRoot, env, "runtime", "prune", "--older-than", "0s", "--include-kept")
	if !strings.Contains(pruneOut, "removed") || !strings.Contains(pruneOut, runtimeRoot) {
		t.Fatalf("runtime prune missing removed runtime: %q", pruneOut)
	}
	assertRuntimeEntries(t, runtimeDir, 0)
	assertProjectClean(t, projectRoot)

	runADP(t, adpBin, repoRoot, env, "workspace", "rename", "game-a", "game-renamed")
	renamedOut := runADP(t, adpBin, repoRoot, env, "workspace", "show", "game-renamed")
	if !strings.Contains(renamedOut, "name: game-renamed") || !strings.Contains(renamedOut, "project_root: "+projectRoot) {
		t.Fatalf("renamed workspace show missing details: %q", renamedOut)
	}
	runADP(t, adpBin, repoRoot, env, "workspace", "remove", "game-renamed")
	removedList := runADP(t, adpBin, repoRoot, env, "workspace", "list")
	if strings.Contains(removedList, "game-renamed") {
		t.Fatalf("removed workspace still listed: %q", removedList)
	}
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

func sessionIDsByAgent(t *testing.T, path string) map[string]string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read event log: %v", err)
	}

	ids := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var event struct {
			Type      string `json:"type"`
			Agent     string `json:"agent"`
			SessionID string `json:"session_id"`
		}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("decode event line %q: %v", line, err)
		}
		if event.Type == "run_started" && event.Agent != "" && event.SessionID != "" {
			ids[event.Agent] = event.SessionID
		}
	}
	return ids
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

func assertRuntimeEntries(t *testing.T, runtimeDir string, want int) {
	t.Helper()
	entries, err := os.ReadDir(runtimeDir)
	if err != nil {
		t.Fatalf("read runtime dir: %v", err)
	}
	if len(entries) != want {
		t.Fatalf("runtime dir entries = %d, want %d", len(entries), want)
	}
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if info.IsDir() {
		t.Fatalf("%s is a directory, want file", path)
	}
}

func parseExport(t *testing.T, output string, name string) string {
	t.Helper()
	prefix := "export " + name + "="
	for _, line := range strings.Split(output, "\n") {
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		return strings.Trim(line[len(prefix):], "'")
	}
	t.Fatalf("export %s not found in:\n%s", name, output)
	return ""
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
