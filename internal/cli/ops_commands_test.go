package cli

import (
	"bytes"
	"context"
	"testing"

	"github.com/karoc/adp/internal/shell"
)

func TestShellHookCommandRendersHook(t *testing.T) {
	var stdout bytes.Buffer
	var gotOpts shell.HookOptions

	deps := Dependencies{
		RenderHook: func(opts shell.HookOptions) (string, error) {
			gotOpts = opts
			return "hook body\n", nil
		},
	}

	code := NewApp(deps, &stdout, &bytes.Buffer{}).Execute(
		context.Background(),
		[]string{"shell-hook", "--shell", "bash", "--name", "adp-enter"},
	)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if gotOpts.Shell != "bash" || gotOpts.FunctionName != "adp-enter" {
		t.Fatalf("hook options = %+v", gotOpts)
	}
	if stdout.String() != "hook body\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}
