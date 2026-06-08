package cli

import (
	"bytes"
	"context"
	"testing"

	"github.com/karoc/adp/internal/shell"
)

func TestCompletionCommandRendersCompletion(t *testing.T) {
	var stdout bytes.Buffer
	var gotOpts shell.CompletionOptions

	deps := Dependencies{
		RenderCompletion: func(opts shell.CompletionOptions) (string, error) {
			gotOpts = opts
			return "completion body\n", nil
		},
	}

	code := NewApp(deps, &stdout, &bytes.Buffer{}).Execute(
		context.Background(),
		[]string{"completion", "--shell", "zsh", "--command", "adp-dev"},
	)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if gotOpts.Shell != "zsh" || gotOpts.CommandName != "adp-dev" {
		t.Fatalf("completion options = %+v", gotOpts)
	}
	if stdout.String() != "completion body\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestCompletionCommandDefaultsToBash(t *testing.T) {
	var stdout bytes.Buffer
	var gotOpts shell.CompletionOptions

	deps := Dependencies{
		RenderCompletion: func(opts shell.CompletionOptions) (string, error) {
			gotOpts = opts
			return "completion body\n", nil
		},
	}

	code := NewApp(deps, &stdout, &bytes.Buffer{}).Execute(context.Background(), []string{"completion"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if gotOpts.Shell != "bash" {
		t.Fatalf("completion shell = %q, want bash", gotOpts.Shell)
	}
	if stdout.String() != "completion body\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}
