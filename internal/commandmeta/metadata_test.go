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
