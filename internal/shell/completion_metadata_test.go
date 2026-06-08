package shell

import (
	"strings"
	"testing"

	"github.com/karoc/adp/internal/commandmeta"
)

func TestRenderCompletionTracksCommandMetadata(t *testing.T) {
	t.Parallel()

	bashCompletion, err := RenderCompletion(CompletionOptions{Shell: "bash"})
	if err != nil {
		t.Fatalf("RenderCompletion(bash) returned error: %v", err)
	}
	zshCompletion, err := RenderCompletion(CompletionOptions{Shell: "zsh"})
	if err != nil {
		t.Fatalf("RenderCompletion(zsh) returned error: %v", err)
	}

	assertContains(t, bashCompletion, commandmeta.ShellWords(rootValues()), "bash root commands")
	for _, command := range commandmeta.Commands() {
		assertContains(t, zshCompletion, zshQuote(command.Name+":"+command.Description), "zsh root commands")
		assertCompletionValues(t, bashCompletion, zshCompletion, command.Name+" subcommands", command.Subcommands)
		assertCompletionValues(t, bashCompletion, zshCompletion, command.Name+" options", command.Options)
	}
	for name, values := range map[string][]commandmeta.Value{
		"shell names":            commandmeta.Shells,
		"event types":            commandmeta.EventTypes,
		"runtime ages":           commandmeta.RuntimeAges,
		"text/json formats":      commandmeta.TextJSONFormats,
		"markdown/json formats":  commandmeta.MarkdownJSONFormats,
		"languages":              commandmeta.Languages,
		"completion value kinds": commandmeta.CompletionKinds,
	} {
		assertCompletionValues(t, bashCompletion, zshCompletion, name, values)
	}
}

func rootValues() []commandmeta.Value {
	commands := commandmeta.Commands()
	values := make([]commandmeta.Value, 0, len(commands))
	for _, command := range commands {
		values = append(values, commandmeta.Value{Name: command.Name, Description: command.Description})
	}
	return values
}

func assertCompletionValues(t *testing.T, bashCompletion, zshCompletion, label string, values []commandmeta.Value) {
	t.Helper()

	if len(values) == 0 {
		return
	}
	assertContains(t, bashCompletion, commandmeta.ShellWords(values), "bash "+label)
	for _, value := range values {
		assertZshCandidate(t, zshCompletion, value, "zsh "+label)
	}
}

func assertZshCandidate(t *testing.T, completion string, value commandmeta.Value, label string) {
	t.Helper()

	if strings.Contains(completion, zshValue(value)) || strings.Contains(completion, value.Name) {
		return
	}
	t.Fatalf("%s missing %q:\n%s", label, value.Name, completion)
}

func assertContains(t *testing.T, haystack, needle, label string) {
	t.Helper()

	if !strings.Contains(haystack, needle) {
		t.Fatalf("%s missing %q:\n%s", label, needle, haystack)
	}
}
