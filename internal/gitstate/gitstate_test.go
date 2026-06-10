package gitstate

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInspectDetectsCleanRepositoryAndGitfile(t *testing.T) {
	projectRoot := initRepo(t)
	state := Inspect(context.Background(), projectRoot)

	if !state.GitAvailable || state.InspectionError != "" || state.StatusError != "" {
		t.Fatalf("unexpected inspection failure: %+v", state)
	}
	if state.GitRoot != projectRoot {
		t.Fatalf("GitRoot = %q, want %q", state.GitRoot, projectRoot)
	}
	if state.MetadataKind != MetadataDirectory {
		t.Fatalf("MetadataKind = %q, want directory", state.MetadataKind)
	}
	if state.ChangeState != ChangeClean || state.ChangedEntries != 0 {
		t.Fatalf("change state = %+v, want clean", state)
	}
	if state.Branch == "" {
		t.Fatalf("branch should be detected: %+v", state)
	}

	worktreeRoot := filepath.Join(t.TempDir(), "linked-worktree")
	mustGit(t, projectRoot, "worktree", "add", "--detach", worktreeRoot, "HEAD")
	worktreeState := Inspect(context.Background(), worktreeRoot)
	if worktreeState.GitRoot != worktreeRoot {
		t.Fatalf("worktree GitRoot = %q, want %q", worktreeState.GitRoot, worktreeRoot)
	}
	if worktreeState.MetadataKind != MetadataFile {
		t.Fatalf("worktree MetadataKind = %q, want file: %+v", worktreeState.MetadataKind, worktreeState)
	}
}

func TestInspectDetectsSubdirectoryDirtyStateAndSanitizesGitEnv(t *testing.T) {
	projectRoot := initRepo(t)
	subdir := filepath.Join(projectRoot, "src")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatalf("create subdir: %v", err)
	}
	writeFile(t, filepath.Join(subdir, "changed.txt"), "changed\n")
	t.Setenv("GIT_DIR", filepath.Join(t.TempDir(), "poisoned-git-dir"))
	t.Setenv("GIT_WORK_TREE", t.TempDir())
	t.Setenv("GIT_CEILING_DIRECTORIES", projectRoot)

	state := Inspect(context.Background(), subdir)
	if state.GitRoot != projectRoot {
		t.Fatalf("GitRoot = %q, want %q: %+v", state.GitRoot, projectRoot, state)
	}
	if !state.ProjectBelowRoot || state.RelativeProjectDir != "src" {
		t.Fatalf("subdir relationship not detected: %+v", state)
	}
	if state.ChangeState != ChangeDirty || state.UntrackedEntries != 1 || state.ChangedEntries != 1 {
		t.Fatalf("dirty state not parsed: %+v", state)
	}
}

func TestDiscoverRootUsesRootInspectionOnly(t *testing.T) {
	binDir := t.TempDir()
	projectRoot := t.TempDir()
	gitRoot := filepath.Join(t.TempDir(), "repo")
	logPath := filepath.Join(t.TempDir(), "git-args.log")
	fakeGit := filepath.Join(binDir, "git")
	script := `#!/usr/bin/env sh
set -eu
printf '%s\n' "$*" >> "$ADP_FAKE_GIT_LOG"
for arg do
  if [ "$arg" = "status" ]; then
    printf 'status should not run during root discovery\n' >&2
    exit 43
  fi
done
printf 'true\n%s\n%s/.git\n' "$ADP_FAKE_GIT_ROOT" "$ADP_FAKE_GIT_ROOT"
`
	if err := os.WriteFile(fakeGit, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake git: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("ADP_FAKE_GIT_ROOT", gitRoot)
	t.Setenv("ADP_FAKE_GIT_LOG", logPath)

	if got := DiscoverRoot(context.Background(), projectRoot); got != gitRoot {
		t.Fatalf("DiscoverRoot() = %q, want %q", got, gitRoot)
	}
	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read fake git log: %v", err)
	}
	if strings.Contains(string(logData), "status") {
		t.Fatalf("DiscoverRoot invoked status: %s", logData)
	}
}

func TestInspectReportsNonGitProject(t *testing.T) {
	projectRoot := t.TempDir()
	state := Inspect(context.Background(), projectRoot)
	if state.GitRoot != "" || state.InspectionError == "" {
		t.Fatalf("non-Git project should report inspection error: %+v", state)
	}
	if !strings.Contains(state.InspectionError, "not a git repository") {
		t.Fatalf("unexpected inspection error: %q", state.InspectionError)
	}
	if state.ChangeState != ChangeError {
		t.Fatalf("ChangeState = %q, want error", state.ChangeState)
	}
}

func initRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustGit(t, root, "init", "-q")
	mustGit(t, root, "config", "user.name", "adp-test")
	mustGit(t, root, "config", "user.email", "adp-test@example.invalid")
	writeFile(t, filepath.Join(root, "README.md"), "# test\n")
	mustGit(t, root, "add", "README.md")
	mustGit(t, root, "commit", "-q", "-m", "init")
	return root
}

func mustGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git -C %s %s: %v\n%s", dir, strings.Join(args, " "), err, out)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create parent for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}
