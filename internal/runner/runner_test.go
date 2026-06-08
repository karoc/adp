package runner

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/karoc/adp/internal/adapters"
)

func TestRunSuccessPassesStreamsEnvAndCWD(t *testing.T) {
	t.Parallel()

	runtimeRoot := t.TempDir()
	spec := adapters.LaunchSpec{
		Command: "/bin/sh",
		Args: []string{"-c", `
printf 'cwd=%s\n' "$(pwd)"
printf 'env=%s\n' "$ADP_RUNNER_TEST"
printf 'stdin='
cat
printf '\nstderr=%s\n' "$ADP_RUNNER_TEST" >&2
`},
		Dir: runtimeRoot,
		Env: map[string]string{
			"ADP_RUNNER_TEST": "visible",
		},
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	result, err := Run(context.Background(), spec, Streams{
		Stdin:  strings.NewReader("payload"),
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", result.ExitCode)
	}

	out := stdout.String()
	if !strings.Contains(out, "cwd="+runtimeRoot+"\n") {
		t.Fatalf("stdout missing cwd %q:\n%s", runtimeRoot, out)
	}
	if !strings.Contains(out, "env=visible\n") {
		t.Fatalf("stdout missing env override:\n%s", out)
	}
	if !strings.Contains(out, "stdin=payload") {
		t.Fatalf("stdout missing stdin payload:\n%s", out)
	}
	if !strings.Contains(stderr.String(), "stderr=visible\n") {
		t.Fatalf("stderr was not passed through:\n%s", stderr.String())
	}
}

func TestRunReturnsFailedExitCodeWithoutError(t *testing.T) {
	t.Parallel()

	result, err := Run(context.Background(), adapters.LaunchSpec{
		Command: "/bin/sh",
		Args:    []string{"-c", "exit 37"},
	}, Streams{})
	if err != nil {
		t.Fatalf("Run returned error for process exit: %v", err)
	}
	if result.ExitCode != 37 {
		t.Fatalf("ExitCode = %d, want 37", result.ExitCode)
	}
}

func TestRunRejectsEmptyCommand(t *testing.T) {
	t.Parallel()

	result, err := Run(context.Background(), adapters.LaunchSpec{}, Streams{})
	if !errors.Is(err, ErrCommandRequired) {
		t.Fatalf("error = %v, want ErrCommandRequired", err)
	}
	if result != nil {
		t.Fatalf("result = %#v, want nil", result)
	}
}
