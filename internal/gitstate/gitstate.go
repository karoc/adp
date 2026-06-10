package gitstate

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/karoc/adp/internal/gitenv"
)

type MetadataKind string

const (
	MetadataAbsent    MetadataKind = "absent"
	MetadataDirectory MetadataKind = "directory"
	MetadataFile      MetadataKind = "file"
	MetadataOther     MetadataKind = "other"
)

type ChangeState string

const (
	ChangeClean ChangeState = "clean"
	ChangeDirty ChangeState = "dirty"
	ChangeError ChangeState = "error"
)

type State struct {
	ProjectRoot        string
	GitRoot            string
	GitDir             string
	MetadataPath       string
	MetadataKind       MetadataKind
	InsideWorkTree     bool
	ProjectBelowRoot   bool
	RelativeProjectDir string
	Branch             string
	Upstream           string
	Ahead              int
	Behind             int
	ChangeState        ChangeState
	ChangedEntries     int
	UntrackedEntries   int
	InspectionError    string
	StatusError        string
	GitAvailable       bool
}

func Inspect(ctx context.Context, projectRoot string) State {
	state := State{
		ProjectRoot:     filepath.Clean(projectRoot),
		MetadataKind:    MetadataAbsent,
		ChangeState:     ChangeError,
		GitAvailable:    true,
		InspectionError: "",
	}
	if strings.TrimSpace(projectRoot) == "" {
		state.InspectionError = "project root is required"
		return state
	}
	if _, err := exec.LookPath("git"); err != nil {
		state.GitAvailable = false
		state.InspectionError = "git executable was not found in PATH"
		return state
	}
	metadataPath, kind := inspectMetadataPath(state.ProjectRoot)
	state.MetadataPath = metadataPath
	state.MetadataKind = kind

	if err := inspectRoot(ctx, &state); err != nil {
		state.InspectionError = cleanGitError(err)
		state.ChangeState = ChangeError
		return state
	}

	if statusOutput, err := runGit(ctx, state.ProjectRoot, "status", "--porcelain=v2", "--branch"); err == nil {
		applyStatus(&state, statusOutput)
	} else {
		state.StatusError = cleanGitError(err)
		state.ChangeState = ChangeError
	}
	return state
}

func DiscoverRoot(ctx context.Context, projectRoot string) string {
	state := State{
		ProjectRoot:  filepath.Clean(projectRoot),
		GitAvailable: true,
	}
	if strings.TrimSpace(projectRoot) == "" {
		return ""
	}
	if _, err := exec.LookPath("git"); err != nil {
		return ""
	}
	if err := inspectRoot(ctx, &state); err != nil {
		return ""
	}
	return state.GitRoot
}

func inspectRoot(ctx context.Context, state *State) error {
	output, err := runGit(ctx, state.ProjectRoot,
		"rev-parse",
		"--is-inside-work-tree",
		"--show-toplevel",
		"--absolute-git-dir",
	)
	if err != nil {
		return err
	}
	lines := splitOutputLines(output)
	if len(lines) >= 1 {
		state.InsideWorkTree = lines[0] == "true"
	}
	if len(lines) >= 2 {
		state.GitRoot = filepath.Clean(lines[1])
	}
	if len(lines) >= 3 {
		state.GitDir = filepath.Clean(lines[2])
	}
	if state.GitRoot != "" {
		state.ProjectBelowRoot = state.GitRoot != state.ProjectRoot && isPathInside(state.GitRoot, state.ProjectRoot)
		if rel, err := filepath.Rel(state.GitRoot, state.ProjectRoot); err == nil && rel != "." {
			state.RelativeProjectDir = rel
		}
		if state.MetadataKind == MetadataAbsent {
			state.MetadataPath, state.MetadataKind = inspectMetadataPath(state.GitRoot)
		}
	}
	return nil
}

func inspectMetadataPath(projectRoot string) (string, MetadataKind) {
	path := filepath.Join(projectRoot, ".git")
	info, err := os.Lstat(path)
	if err != nil {
		return path, MetadataAbsent
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return path, MetadataOther
	}
	if info.IsDir() {
		return path, MetadataDirectory
	}
	if info.Mode().IsRegular() {
		return path, MetadataFile
	}
	return path, MetadataOther
}

func runGit(ctx context.Context, dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	cmd.Env = sanitizedEnv()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		return stdout.String(), ctx.Err()
	}
	if err != nil {
		return stdout.String(), gitCommandError{err: err, stderr: stderr.String()}
	}
	return stdout.String(), nil
}

func sanitizedEnv() []string {
	env := os.Environ()
	out := make([]string, 0, len(env))
	for _, entry := range env {
		key, _, ok := strings.Cut(entry, "=")
		if !ok || gitenv.IsRepositoryDirective(key) || key == "GIT_CEILING_DIRECTORIES" {
			continue
		}
		out = append(out, entry)
	}
	return out
}

func splitOutputLines(output string) []string {
	raw := strings.Split(strings.TrimRight(output, "\n"), "\n")
	lines := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func applyStatus(state *State, output string) {
	state.ChangeState = ChangeClean
	for _, line := range splitOutputLines(output) {
		switch {
		case strings.HasPrefix(line, "# branch.head "):
			state.Branch = strings.TrimPrefix(line, "# branch.head ")
		case strings.HasPrefix(line, "# branch.upstream "):
			state.Upstream = strings.TrimPrefix(line, "# branch.upstream ")
		case strings.HasPrefix(line, "# branch.ab "):
			parseAheadBehind(state, strings.TrimPrefix(line, "# branch.ab "))
		case strings.HasPrefix(line, "? "):
			state.UntrackedEntries++
			state.ChangedEntries++
			state.ChangeState = ChangeDirty
		case strings.HasPrefix(line, "1 ") || strings.HasPrefix(line, "2 ") || strings.HasPrefix(line, "u "):
			state.ChangedEntries++
			state.ChangeState = ChangeDirty
		}
	}
}

func parseAheadBehind(state *State, value string) {
	fields := strings.Fields(value)
	for _, field := range fields {
		if strings.HasPrefix(field, "+") {
			if parsed, err := strconv.Atoi(strings.TrimPrefix(field, "+")); err == nil {
				state.Ahead = parsed
			}
			continue
		}
		if strings.HasPrefix(field, "-") {
			if parsed, err := strconv.Atoi(strings.TrimPrefix(field, "-")); err == nil {
				state.Behind = parsed
			}
		}
	}
}

func isPathInside(base string, candidate string) bool {
	rel, err := filepath.Rel(base, candidate)
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

type gitCommandError struct {
	err    error
	stderr string
}

func (e gitCommandError) Error() string {
	message := strings.TrimSpace(e.stderr)
	if message == "" {
		return e.err.Error()
	}
	return message
}

func cleanGitError(err error) string {
	if err == nil {
		return ""
	}
	var gitErr gitCommandError
	if errors.As(err, &gitErr) {
		return gitErr.Error()
	}
	return fmt.Sprint(err)
}
