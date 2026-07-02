package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// TestConcurrentTasksTakeAcrossProcessesNeverDuplicates spawns many real `adp`
// processes that race to `tasks take` from a shared pool. The cross-process
// file lock must serialize the read-modify-write of the task board so every
// task is claimed by exactly one owner: no task claimed twice, none lost. This
// exercises the lock across independent OS processes, which the in-process
// goroutine test in the tasks package cannot.
func TestConcurrentTasksTakeAcrossProcessesNeverDuplicates(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("cross-process spawn scenario is exercised on POSIX")
	}

	tmp := t.TempDir()
	projectRoot := filepath.Join(tmp, "project")
	adpHome := filepath.Join(tmp, "adp-home")
	runtimeDir := filepath.Join(tmp, "runtime")
	adpBin := filepath.Join(tmp, "adp")
	mkdirAll(t, projectRoot, runtimeDir)
	buildADP(t, adpBin)
	writeFile(t, filepath.Join(projectRoot, "go.mod"), "module example.com/game\n")
	writeFile(t, filepath.Join(projectRoot, "main.go"), "package main\n")
	initGitProject(t, projectRoot)

	env := append(os.Environ(),
		"ADP_HOME="+adpHome,
		"ADP_RUNTIME_DIR="+runtimeDir,
	)

	repoRoot := repositoryRoot(t)
	runADP(t, adpBin, repoRoot, env, "init")
	runADP(t, adpBin, repoRoot, env, "workspace", "add", "game-a", projectRoot)

	const (
		tasks   = 8
		workers = 16
	)
	for i := 1; i <= tasks; i++ {
		runADP(t, adpBin, repoRoot, env, "tasks", "add", "--workspace", "game-a", fmt.Sprintf("Task number %d", i), "--priority", "high")
	}

	type takeResult struct {
		taskID   string
		owner    string
		claimed  bool
		exitZero bool
		output   string
	}

	results := make([]takeResult, workers)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			owner := fmt.Sprintf("worker-%d", w)
			cmd := exec.Command(adpBin, "tasks", "take", "--workspace", "game-a", "--owner", owner, "--lease", "1h", "--format", "json")
			cmd.Dir = repoRoot
			cmd.Env = env
			out, err := cmd.CombinedOutput()
			res := takeResult{owner: owner, output: string(out), exitZero: err == nil}
			if err == nil {
				var task struct {
					ID    string `json:"id"`
					Owner string `json:"owner"`
				}
				if jsonErr := json.Unmarshal(out, &task); jsonErr != nil {
					res.output = fmt.Sprintf("claim output not JSON: %v\n%s", jsonErr, out)
				} else {
					res.claimed = true
					res.taskID = task.ID
					res.owner = task.Owner
				}
			}
			results[w] = res
		}(w)
	}
	wg.Wait()

	claimedBy := map[string]string{}
	claims := 0
	noWork := 0
	for _, res := range results {
		if res.claimed {
			claims++
			if prev, dup := claimedBy[res.taskID]; dup {
				t.Fatalf("task %s claimed twice: by %s and %s", res.taskID, prev, res.owner)
			}
			claimedBy[res.taskID] = res.owner
			continue
		}
		if res.exitZero {
			t.Fatalf("worker exited zero without a parsable claim: %q", res.output)
		}
		if !strings.Contains(res.output, "no claimable task") {
			t.Fatalf("non-claiming worker failed for an unexpected reason: %q", res.output)
		}
		noWork++
	}

	if claims != tasks {
		t.Fatalf("claimed %d tasks, want %d", claims, tasks)
	}
	if len(claimedBy) != tasks {
		t.Fatalf("distinct claimed tasks = %d, want %d", len(claimedBy), tasks)
	}
	if noWork != workers-tasks {
		t.Fatalf("no-work workers = %d, want %d", noWork, workers-tasks)
	}

	owners := map[string]struct{}{}
	for _, owner := range claimedBy {
		owners[owner] = struct{}{}
	}
	if len(owners) != tasks {
		t.Fatalf("distinct owners = %d, want %d (a worker claimed more than one task)", len(owners), tasks)
	}
}

// TestConcurrentRunsAppendIntactEventLines spawns many real `adp run`
// processes that append to the shared event log at once. Each finished run
// writes a run_started and run_finished line, and a single line can exceed the
// POSIX PIPE_BUF atomicity threshold once it carries command context. The
// cross-process append lock must keep every JSONL line intact and independently
// parseable — no interleaving, no truncation.
func TestConcurrentRunsAppendIntactEventLines(t *testing.T) {
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
	writeExecutable(t, filepath.Join(binDir, "codex"), fakeAgentScript("codex", "AGENTS.md", ".codex/config.toml", "go.mod"))

	env := append(os.Environ(),
		"ADP_HOME="+adpHome,
		"ADP_RUNTIME_DIR="+runtimeDir,
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
	)

	repoRoot := repositoryRoot(t)
	runADP(t, adpBin, repoRoot, env, "init")
	runADP(t, adpBin, repoRoot, env, "workspace", "add", "game-a", projectRoot)

	const runs = 12
	var wg sync.WaitGroup
	errCh := make(chan error, runs)
	for r := 0; r < runs; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cmd := exec.Command(adpBin, "run", "codex", "--workspace", "game-a", "--", "--probe")
			cmd.Dir = repoRoot
			cmd.Env = env
			if out, err := cmd.CombinedOutput(); err != nil {
				errCh <- fmt.Errorf("run failed: %v\n%s", err, out)
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(adpHome, "logs", "events.jsonl"))
	if err != nil {
		t.Fatalf("read events log: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	// Each run emits exactly run_started + run_finished.
	if len(lines) != runs*2 {
		t.Fatalf("event line count = %d, want %d", len(lines), runs*2)
	}
	started, finished := 0, 0
	for _, line := range lines {
		var event struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("event line is not intact JSON: %v\n%q", err, line)
		}
		switch event.Type {
		case "run_started":
			started++
		case "run_finished":
			finished++
		}
	}
	if started != runs || finished != runs {
		t.Fatalf("run_started=%d run_finished=%d, want %d each", started, finished, runs)
	}
}
