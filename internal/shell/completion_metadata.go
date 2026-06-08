package shell

import (
	"strings"

	"github.com/karoc/adp/internal/commandmeta"
)

func bashWords(values []commandmeta.Value) string {
	return commandmeta.ShellWords(values)
}

func bashProgressTopWords() string {
	return commandmeta.ShellWords(progressTopValues())
}

func completionTopValues() []commandmeta.Value {
	values := commandmeta.Subcommands("completion")
	values = append(values, completionRootOptions()...)
	return values
}

func completionRootOptions() []commandmeta.Value {
	return excludedOptions("completion", "--workspace", "-w")
}

func progressTopValues() []commandmeta.Value {
	values := commandmeta.Subcommands("progress")
	for _, option := range commandmeta.Options("progress") {
		if option.Name == "--language" {
			continue
		}
		values = append(values, option)
	}
	return values
}

func excludedOptions(command string, names ...string) []commandmeta.Value {
	excluded := make(map[string]bool, len(names))
	for _, name := range names {
		excluded[name] = true
	}

	var values []commandmeta.Value
	for _, option := range commandmeta.Options(command) {
		if !excluded[option.Name] {
			values = append(values, option)
		}
	}
	return values
}

func completionValuesOptions() []commandmeta.Value {
	return selectedOptions("completion", "--workspace", "-w")
}

func selectedOptions(command string, names ...string) []commandmeta.Value {
	wanted := make(map[string]bool, len(names))
	for _, name := range names {
		wanted[name] = true
	}

	var values []commandmeta.Value
	for _, option := range commandmeta.Options(command) {
		if wanted[option.Name] {
			values = append(values, option)
		}
	}
	return values
}

func zshRootCommandEntries() string {
	var out strings.Builder
	for _, command := range commandmeta.Commands() {
		out.WriteString("\t\t")
		out.WriteString(zshQuote(command.Name + ":" + command.Description))
		out.WriteByte('\n')
	}
	return out.String()
}

func zshArray(name string, values []string) string {
	var out strings.Builder
	out.WriteByte('\t')
	out.WriteString(name)
	out.WriteString("=(")
	out.WriteString(strings.Join(values, " "))
	out.WriteString(")\n")
	return out.String()
}

func zshValues(label string, values []commandmeta.Value) string {
	var out strings.Builder
	out.WriteString("_values ")
	out.WriteString(zshQuote(label))
	for _, value := range values {
		out.WriteByte(' ')
		out.WriteString(zshValue(value))
	}
	out.WriteByte('\n')
	return out.String()
}

func zshValue(value commandmeta.Value) string {
	if value.Description == "" {
		return zshQuote(value.Name)
	}
	return zshQuote(value.Name + "[" + value.Description + "]")
}

func zshQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
