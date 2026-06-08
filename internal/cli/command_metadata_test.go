package cli

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/karoc/adp/internal/commandmeta"
)

func TestCommandMetadataMatchesDispatch(t *testing.T) {
	t.Parallel()

	app := NewApp(Dependencies{}, &bytes.Buffer{}, &bytes.Buffer{})
	got := make([]string, 0, len(app.commandHandlers()))
	for command := range app.commandHandlers() {
		got = append(got, command)
	}
	sort.Strings(got)

	want := append([]string(nil), commandmeta.RootCommandNames()...)
	sort.Strings(want)

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("dispatch commands = %v, want metadata commands %v", got, want)
	}
}

func TestHelpUsageMatchesCommandMetadata(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	code := NewApp(Dependencies{}, &stdout, &bytes.Buffer{}).Execute(context.Background(), []string{"--help"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if stdout.String() != commandmeta.Usage() {
		t.Fatalf("help output drifted from command metadata:\n%s", stdout.String())
	}
}

func TestMetadataHelpWorksBeforeInit(t *testing.T) {
	t.Parallel()

	for _, command := range commandmeta.Commands() {
		command := command
		t.Run(command.Name, func(t *testing.T) {
			t.Parallel()

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := NewApp(Dependencies{InitError: errors.New("bad env")}, &stdout, &stderr).Execute(context.Background(), []string{command.Name, "--help"})

			if code != 0 {
				t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
			}
			for _, usage := range command.Usage {
				if !strings.Contains(stdout.String(), usage) {
					t.Fatalf("%s help missing usage %q:\n%s", command.Name, usage, stdout.String())
				}
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}
		})
	}
}

func TestMetadataShortHelpMatchesLongHelpBeforeInit(t *testing.T) {
	t.Parallel()

	for _, command := range commandmeta.Commands() {
		command := command
		t.Run(command.Name, func(t *testing.T) {
			t.Parallel()

			longHelp := executeHelp(t, command.Name, "--help")
			shortHelp := executeHelp(t, command.Name, "-h")
			if shortHelp != longHelp {
				t.Fatalf("%s -h output drifted from --help\n-h:\n%s\n--help:\n%s", command.Name, shortHelp, longHelp)
			}
		})
	}
}

func TestMetadataSubcommandHelpWorksBeforeInit(t *testing.T) {
	t.Parallel()

	for _, command := range commandmeta.Commands() {
		command := command
		for _, subcommand := range command.Subcommands {
			subcommand := subcommand
			t.Run(command.Name+"/"+subcommand.Name, func(t *testing.T) {
				t.Parallel()

				var stdout bytes.Buffer
				var stderr bytes.Buffer
				code := NewApp(Dependencies{InitError: errors.New("bad env")}, &stdout, &stderr).Execute(context.Background(), []string{command.Name, subcommand.Name, "--help"})

				if code != 0 {
					t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
				}
				usagePrefix := "adp " + command.Name + " " + subcommand.Name
				if !strings.Contains(stdout.String(), usagePrefix) {
					t.Fatalf("%s %s help missing usage prefix %q:\n%s", command.Name, subcommand.Name, usagePrefix, stdout.String())
				}
				if stderr.Len() != 0 {
					t.Fatalf("stderr = %q, want empty", stderr.String())
				}
			})
		}
	}
}

func TestMetadataSubcommandShortHelpMatchesLongHelpBeforeInit(t *testing.T) {
	t.Parallel()

	for _, command := range commandmeta.Commands() {
		command := command
		for _, subcommand := range command.Subcommands {
			subcommand := subcommand
			t.Run(command.Name+"/"+subcommand.Name, func(t *testing.T) {
				t.Parallel()

				longHelp := executeHelp(t, command.Name, subcommand.Name, "--help")
				shortHelp := executeHelp(t, command.Name, subcommand.Name, "-h")
				if shortHelp != longHelp {
					t.Fatalf("%s %s -h output drifted from --help\n-h:\n%s\n--help:\n%s", command.Name, subcommand.Name, shortHelp, longHelp)
				}
			})
		}
	}
}

func executeHelp(t *testing.T, args ...string) string {
	t.Helper()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := NewApp(Dependencies{InitError: errors.New("bad env")}, &stdout, &stderr).Execute(context.Background(), args)
	if code != 0 {
		t.Fatalf("%v exit code = %d, want 0; stderr = %q", args, code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("%v stderr = %q, want empty", args, stderr.String())
	}
	return stdout.String()
}
