package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/karoc/adp/internal/commandmeta"
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
	if !strings.Contains(stderr.String(), "try: adp --help") {
		t.Fatalf("stderr missing root help hint: %q", stderr.String())
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
	if !strings.Contains(stderr.String(), "try: adp --help") {
		t.Fatalf("stderr missing root help hint: %q", stderr.String())
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
	if !strings.Contains(stderr.String(), "try: adp tasks list --help") {
		t.Fatalf("stderr missing tasks list help hint: %q", stderr.String())
	}
}

func TestExecuteReportsCommandPositionUnknowns(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
		hint string
	}{
		{name: "completion", args: []string{"completion", "bogus"}, want: `adp: unknown completion command "bogus"`, hint: "try: adp completion --help"},
		{name: "progress", args: []string{"progress", "bogus"}, want: `adp: unknown progress command "bogus"`, hint: "try: adp progress --help"},
		{name: "doctor option", args: []string{"doctor", "--bogus"}, want: `adp: unknown doctor option "--bogus"`, hint: "try: adp doctor --help"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stderr bytes.Buffer

			code := NewApp(Dependencies{}, &bytes.Buffer{}, &stderr).Execute(context.Background(), test.args)

			if code != 1 {
				t.Fatalf("exit code = %d, want 1", code)
			}
			if !strings.Contains(stderr.String(), test.want) {
				t.Fatalf("stderr = %q, want %q", stderr.String(), test.want)
			}
			if !strings.Contains(stderr.String(), test.hint) {
				t.Fatalf("stderr = %q, want help hint %q", stderr.String(), test.hint)
			}
		})
	}
}

func TestExecuteReportsSubcommandUsageHelpHints(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
		hint string
	}{
		{
			name: "tasks take missing owner",
			args: []string{"tasks", "take"},
			want: "usage: adp tasks take",
			hint: "try: adp tasks take --help",
		},
		{
			name: "run missing agent",
			args: []string{"run"},
			want: "usage: adp run <agent>",
			hint: "try: adp run --help",
		},
		{
			name: "run take missing owner",
			args: []string{"run", "codex", "--take"},
			want: "--owner is required with --take",
			hint: "try: adp run --help",
		},
		{
			name: "progress report invalid language",
			args: []string{"progress", "report", "--language", "de"},
			want: `unknown progress report language "de"`,
			hint: "try: adp progress report --help",
		},
		{
			name: "completion values invalid kind",
			args: []string{"completion", "values", "widgets"},
			want: `unknown completion values kind "widgets"`,
			hint: "try: adp completion values --help",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stderr bytes.Buffer

			code := NewApp(Dependencies{}, &bytes.Buffer{}, &stderr).Execute(context.Background(), test.args)

			if code != 1 {
				t.Fatalf("exit code = %d, want 1", code)
			}
			if !strings.Contains(stderr.String(), test.want) {
				t.Fatalf("stderr = %q, want %q", stderr.String(), test.want)
			}
			if !strings.Contains(stderr.String(), test.hint) {
				t.Fatalf("stderr = %q, want help hint %q", stderr.String(), test.hint)
			}
		})
	}
}

func TestExecuteReportsMetadataSubcommandHelpHints(t *testing.T) {
	for _, command := range commandmeta.Commands() {
		command := command
		for _, subcommand := range command.Subcommands {
			subcommand := subcommand
			t.Run(command.Name+"/"+subcommand.Name, func(t *testing.T) {
				args := []string{command.Name, subcommand.Name, "--definitely-invalid"}
				want := "adp " + command.Name + " " + subcommand.Name + " --help"
				if got := helpHint(args); got != want {
					t.Fatalf("helpHint(%q) = %q, want %q", args, got, want)
				}
			})
		}
	}
}

func TestExecuteDoesNotHintForStateErrors(t *testing.T) {
	var stderr bytes.Buffer

	code := NewApp(Dependencies{WorkspaceStore: &fakeStore{}}, &bytes.Buffer{}, &stderr).Execute(context.Background(), []string{"workspace", "show", "missing"})

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "workspace not found") {
		t.Fatalf("stderr = %q, want workspace not found", stderr.String())
	}
	if strings.Contains(stderr.String(), "try: ") {
		t.Fatalf("stderr contains unexpected help hint for state error: %q", stderr.String())
	}
}
