package shell

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/karoc/adp/internal/adapters"
	"github.com/karoc/adp/internal/runner"
)

const defaultShell = "/bin/sh"

var ErrRuntimeRootRequired = errors.New("runtime root is required")

type Streams struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

type ExitError struct {
	Code int
}

func (e ExitError) Error() string {
	return fmt.Sprintf("shell exited with code %d", e.Code)
}

func NewSpec(handle adapters.RuntimeHandle) adapters.LaunchSpec {
	return newSpec(handle, os.Getenv("SHELL"))
}

func Enter(ctx context.Context, handle adapters.RuntimeHandle, streams Streams) error {
	spec := NewSpec(handle)
	if spec.Dir == "" {
		return ErrRuntimeRootRequired
	}

	result, err := runner.Run(ctx, spec, runner.Streams{
		Stdin:  streams.Stdin,
		Stdout: streams.Stdout,
		Stderr: streams.Stderr,
	})
	if err != nil {
		return err
	}
	if result != nil && result.ExitCode != 0 {
		return ExitError{Code: result.ExitCode}
	}
	return nil
}

func newSpec(handle adapters.RuntimeHandle, shellPath string) adapters.LaunchSpec {
	if shellPath == "" {
		shellPath = defaultShell
	}

	return adapters.LaunchSpec{
		Command: shellPath,
		Dir:     handle.Root,
		Env:     copyEnv(handle.Env),
	}
}

func copyEnv(env map[string]string) map[string]string {
	if len(env) == 0 {
		return nil
	}

	copied := make(map[string]string, len(env))
	for key, value := range env {
		copied[key] = value
	}
	return copied
}
