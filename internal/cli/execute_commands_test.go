package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestExecuteShowsHelp(t *testing.T) {
	var stdout bytes.Buffer

	code := NewApp(Dependencies{}, &stdout, &bytes.Buffer{}).Execute(context.Background(), nil)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "adp run <agent>") {
		t.Fatalf("help output missing run usage: %q", stdout.String())
	}
}

func TestExecuteShowsCommandHelpBeforeInit(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := NewApp(Dependencies{InitError: errors.New("bad env")}, &stdout, &stderr).Execute(context.Background(), []string{"run", "--help"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	for _, want := range []string{"adp run - run an agent inside a runtime", "adp run <agent>", "--workspace"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("help output missing %q: %q", want, stdout.String())
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestExecuteShowsSubcommandHelpBeforeInit(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := NewApp(Dependencies{InitError: errors.New("bad env")}, &stdout, &stderr).Execute(context.Background(), []string{"tasks", "add", "--help"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	for _, want := range []string{"adp tasks add", "Usage:", "--priority <value>", "adp tasks --help"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("help output missing %q: %q", want, stdout.String())
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestExecuteReportsUnknownCommand(t *testing.T) {
	var stderr bytes.Buffer

	code := NewApp(Dependencies{}, &bytes.Buffer{}, &stderr).Execute(context.Background(), []string{"bogus"})

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), `adp: unknown command "bogus"`) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestExecuteReportsUnknownGlobalOption(t *testing.T) {
	var stderr bytes.Buffer

	code := NewApp(Dependencies{}, &bytes.Buffer{}, &stderr).Execute(context.Background(), []string{"--bogus"})

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), `adp: unknown global option "--bogus"`) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestExecuteReportsCommandSpecificWorkspaceOutputOption(t *testing.T) {
	var stderr bytes.Buffer

	code := NewApp(Dependencies{}, &bytes.Buffer{}, &stderr).Execute(context.Background(), []string{"tasks", "list", "--bogus"})

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), `adp: unknown tasks list option "--bogus"`) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
