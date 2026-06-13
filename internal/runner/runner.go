package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/karoc/adp/internal/adapters"
	"github.com/karoc/adp/internal/gitenv"
)

var ErrCommandRequired = errors.New("launch command is required")

var ErrCommandNotFound = errors.New("agent command not found")

type Streams struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

type Result struct {
	ExitCode int
}

func Run(ctx context.Context, spec adapters.LaunchSpec, streams Streams) (*Result, error) {
	if spec.Command == "" {
		return nil, ErrCommandRequired
	}

	// Check if command exists in PATH
	cmdPath, err := exec.LookPath(spec.Command)
	if err != nil {
		return nil, &CommandNotFoundError{
			Command: spec.Command,
			Err:     err,
		}
	}

	cmd := exec.CommandContext(ctx, cmdPath, spec.Args...)
	cmd.Dir = spec.Dir
	cmd.Env = mergedEnv(spec.Env)
	cmd.Stdin = streams.Stdin
	cmd.Stdout = streams.Stdout
	cmd.Stderr = streams.Stderr

	err = cmd.Run()
	if err == nil {
		return &Result{ExitCode: 0}, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result := &Result{ExitCode: exitErr.ExitCode()}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return result, ctxErr
		}
		return result, nil
	}

	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}

	return nil, fmt.Errorf("start %q: %w", spec.Command, err)
}

// CommandNotFoundError is returned when an agent command is not found in PATH.
type CommandNotFoundError struct {
	Command string
	Err     error
}

func (e *CommandNotFoundError) Error() string {
	return fmt.Sprintf("agent command not found: %s", e.Command)
}

func (e *CommandNotFoundError) Unwrap() error {
	return e.Err
}

func mergedEnv(overrides map[string]string) []string {
	env := map[string]string{}
	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		if gitenv.IsRepositoryDirective(key) {
			continue
		}
		env[key] = value
	}
	for key, value := range overrides {
		if gitenv.IsRepositoryDirective(key) {
			continue
		}
		env[key] = value
	}

	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	merged := make([]string, 0, len(keys))
	for _, key := range keys {
		merged = append(merged, key+"="+env[key])
	}
	return merged
}
