package commandmeta

import (
	"strings"
	"testing"
)

func TestMetadataHasUniqueRootCommandsAndUsage(t *testing.T) {
	t.Parallel()

	seen := map[string]bool{}
	for _, command := range Commands() {
		if command.Name == "" {
			t.Fatal("command name must not be empty")
		}
		if seen[command.Name] {
			t.Fatalf("duplicate root command %q", command.Name)
		}
		seen[command.Name] = true
		if len(command.Usage) == 0 {
			t.Fatalf("command %q has no usage lines", command.Name)
		}
		for _, line := range command.Usage {
			if !strings.HasPrefix(line, "adp "+command.Name) {
				t.Fatalf("usage line %q does not belong to command %q", line, command.Name)
			}
		}
		assertUniqueValues(t, command.Name+" subcommands", command.Subcommands)
		assertUniqueValues(t, command.Name+" options", command.Options)
	}
}

func TestUsageIncludesEveryMetadataLine(t *testing.T) {
	t.Parallel()

	usage := Usage()
	for _, line := range UsageLines() {
		if !strings.Contains(usage, "  "+line+"\n") {
			t.Fatalf("usage missing %q:\n%s", line, usage)
		}
	}
}

func TestUsageOptionsAreDeclared(t *testing.T) {
	t.Parallel()

	for _, command := range Commands() {
		declared := map[string]bool{}
		for _, option := range command.Options {
			declared[option.Name] = true
		}
		for _, line := range command.Usage {
			for _, field := range strings.Fields(line) {
				option := usageOption(field)
				if option == "" || declared[option] {
					continue
				}
				t.Fatalf("usage line %q references undeclared option %q", line, option)
			}
		}
	}
}

func TestCommandHelpIncludesUsageAndValues(t *testing.T) {
	t.Parallel()

	help, ok := CommandHelp("tasks")
	if !ok {
		t.Fatal("CommandHelp(tasks) returned false")
	}
	for _, want := range []string{
		"adp tasks - manage the local workspace task board",
		"Usage:",
		"adp tasks add",
		"Subcommands:",
		"take - atomically claim next work",
		"Options:",
		"--workspace - workspace name",
	} {
		if !strings.Contains(help, want) {
			t.Fatalf("command help missing %q:\n%s", want, help)
		}
	}
}

func TestSubcommandHelpIncludesFocusedUsage(t *testing.T) {
	t.Parallel()

	help, ok := SubcommandHelp("phase", "commit")
	if !ok {
		t.Fatal("SubcommandHelp(phase, commit) returned false")
	}
	for _, want := range []string{
		"adp phase commit",
		"Usage:",
		"adp phase commit [--workspace <name>] <phase-id> --hash <commit-hash> [--message <text>]",
		"See also:",
		"adp phase --help",
	} {
		if !strings.Contains(help, want) {
			t.Fatalf("subcommand help missing %q:\n%s", want, help)
		}
	}
	if strings.Contains(help, "adp phase add") {
		t.Fatalf("subcommand help included unrelated usage:\n%s", help)
	}
}

func TestHelpIncludesCopyableExamples(t *testing.T) {
	t.Parallel()

	help, ok := CommandHelp("run")
	if !ok {
		t.Fatal("CommandHelp(run) returned false")
	}
	for _, want := range []string{
		"Examples:",
		"adp run codex --workspace game-a --take --owner codex-main --lease 4h",
		"adp run claude --workspace game-a --task task-20260611-0001 --keep-runtime",
	} {
		if !strings.Contains(help, want) {
			t.Fatalf("run help missing example %q:\n%s", want, help)
		}
	}

	help, ok = SubcommandHelp("tasks", "take")
	if !ok {
		t.Fatal("SubcommandHelp(tasks, take) returned false")
	}
	for _, want := range []string{
		"Examples:",
		"adp tasks take --workspace game-a --owner codex-main --lease 4h --format json",
	} {
		if !strings.Contains(help, want) {
			t.Fatalf("tasks take help missing example %q:\n%s", want, help)
		}
	}
	if strings.Contains(help, "adp tasks claim --workspace") {
		t.Fatalf("tasks take help included claim example:\n%s", help)
	}

	help, ok = SubcommandHelp("phase", "accept")
	if !ok {
		t.Fatal("SubcommandHelp(phase, accept) returned false")
	}
	if want := `adp phase accept --workspace game-a P60 --command "scripts/check-all.sh" --result passed --notes "runtime smoke passed"`; !strings.Contains(help, want) {
		t.Fatalf("phase accept help missing example %q:\n%s", want, help)
	}
}

func assertUniqueValues(t *testing.T, label string, values []Value) {
	t.Helper()

	seen := map[string]bool{}
	for _, value := range values {
		if value.Name == "" {
			t.Fatalf("%s contains empty value", label)
		}
		if seen[value.Name] {
			t.Fatalf("%s contains duplicate value %q", label, value.Name)
		}
		seen[value.Name] = true
	}
}

func usageOption(field string) string {
	field = strings.Trim(field, "[],")
	if strings.HasPrefix(field, "--") {
		return field
	}
	if strings.HasPrefix(field, "-") && len(field) == 2 {
		return field
	}
	return ""
}
