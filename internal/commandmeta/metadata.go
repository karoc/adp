package commandmeta

import "strings"

type Value struct {
	Name        string
	Description string
}

type Command struct {
	Name        string
	Description string
	Usage       []string
	Subcommands []Value
	Options     []Value
	SeeAlso     []string // Related commands for cross-referencing
}

type valueDescriptions map[string]string

var (
	HookShells          = values("sh", "bash", "zsh")
	CompletionShells    = values("bash", "zsh")
	Shells              = CompletionShells
	EventTypes          = values("run_started", "run_finished")
	RuntimeAges         = values("1h", "24h", "168h")
	TextJSONFormats     = describedValues(valueDescriptions{"text": "text output", "json": "JSON output"}, "text", "json")
	MarkdownJSONFormats = describedValues(valueDescriptions{"markdown": "Markdown report", "json": "JSON output"}, "markdown", "json")
	Languages           = describedValues(valueDescriptions{"en": "English", "zh-CN": "Simplified Chinese"}, "en", "zh-CN")
	CompletionKinds     = describedValues(valueDescriptions{
		"agents":     "registered agents",
		"workspaces": "registered workspaces",
		"profiles":   "workspace profiles",
		"tasks":      "workspace task ids",
		"phases":     "workspace phase ids",
		"sessions":   "session ids",
		"owners":     "task owners",
		"statuses":   "task statuses",
	}, "agents", "workspaces", "profiles", "tasks", "phases", "sessions", "owners", "statuses")
)

func Commands() []Command {
	commands := make([]Command, 0, len(rootCommands))
	for _, command := range rootCommands {
		commands = append(commands, cloneCommand(command))
	}
	return commands
}

func Lookup(name string) (Command, bool) {
	for _, command := range rootCommands {
		if command.Name == name {
			return cloneCommand(command), true
		}
	}
	return Command{}, false
}

func RootCommandNames() []string {
	names := make([]string, 0, len(rootCommands))
	for _, command := range rootCommands {
		names = append(names, command.Name)
	}
	return names
}

func SubcommandNames(command string) []string {
	meta, ok := Lookup(command)
	if !ok {
		return nil
	}
	return ValueNames(meta.Subcommands)
}

func Subcommands(command string) []Value {
	meta, ok := Lookup(command)
	if !ok {
		return nil
	}
	return append([]Value(nil), meta.Subcommands...)
}

func OptionNames(command string) []string {
	meta, ok := Lookup(command)
	if !ok {
		return nil
	}
	return ValueNames(meta.Options)
}

func Options(command string) []Value {
	meta, ok := Lookup(command)
	if !ok {
		return nil
	}
	return append([]Value(nil), meta.Options...)
}

func CommandValues(command string) []Value {
	meta, ok := Lookup(command)
	if !ok {
		return nil
	}
	values := append([]Value(nil), meta.Subcommands...)
	return append(values, meta.Options...)
}

func Usage() string {
	var out strings.Builder
	out.WriteString("adp - Agent Development Platform\n\n")
	out.WriteString("Manage AI agent workspaces, tasks, and runtime environments.\n")
	out.WriteString("Keep agent configuration outside project roots with runtime overlays.\n\n")
	out.WriteString("Documentation: https://github.com/karoc/adp\n")
	out.WriteString("Quick start: https://github.com/karoc/adp#quick-start\n\n")
	out.WriteString("Usage:\n")
	for _, command := range rootCommands {
		for _, line := range command.Usage {
			out.WriteString("  ")
			out.WriteString(line)
			out.WriteByte('\n')
		}
	}
	out.WriteByte('\n')
	return out.String()
}

func CommandHelp(name string) (string, bool) {
	command, ok := Lookup(name)
	if !ok {
		return "", false
	}

	var out strings.Builder
	out.WriteString("adp ")
	out.WriteString(command.Name)
	if command.Description != "" {
		out.WriteString(" - ")
		out.WriteString(command.Description)
	}
	out.WriteString("\n\nUsage:\n")
	writeUsageLines(&out, command.Usage)
	writeValuesSection(&out, "Subcommands", command.Subcommands)
	writeValuesSection(&out, "Options", command.Options)
	writeExamplesSection(&out, examplesForCommand(command.Name))
	writeSeeAlsoSection(&out, name, "")
	return out.String(), true
}

func SubcommandHelp(commandName, subcommand string) (string, bool) {
	command, ok := Lookup(commandName)
	if !ok || !hasValue(command.Subcommands, subcommand) {
		return "", false
	}

	usage := usageLinesForSubcommand(command, subcommand)
	if len(usage) == 0 {
		return "", false
	}

	var out strings.Builder
	out.WriteString("adp ")
	out.WriteString(command.Name)
	out.WriteByte(' ')
	out.WriteString(subcommand)
	if description := valueDescription(command.Subcommands, subcommand); description != "" {
		out.WriteString(" - ")
		out.WriteString(description)
	}
	out.WriteString("\n\nUsage:\n")
	writeUsageLines(&out, usage)
	writeValuesSection(&out, "Options", optionsForUsage(usage, command.Options))
	writeExamplesSection(&out, examplesForSubcommand(command.Name, subcommand))
	writeSeeAlsoSection(&out, command.Name, subcommand)
	return out.String(), true
}

func UsageLines() []string {
	var lines []string
	for _, command := range rootCommands {
		lines = append(lines, command.Usage...)
	}
	return lines
}

func FormatValues(command, subcommand string) []Value {
	if command == "progress" && subcommand == "report" {
		return append([]Value(nil), MarkdownJSONFormats...)
	}
	return append([]Value(nil), TextJSONFormats...)
}

func ValueNames(values []Value) []string {
	names := make([]string, 0, len(values))
	for _, value := range values {
		names = append(names, value.Name)
	}
	return names
}

func ShellWords(values []Value) string {
	return strings.Join(ValueNames(values), " ")
}

func cloneCommand(command Command) Command {
	command.Usage = append([]string(nil), command.Usage...)
	command.Subcommands = append([]Value(nil), command.Subcommands...)
	command.Options = append([]Value(nil), command.Options...)
	return command
}

func values(names ...string) []Value {
	out := make([]Value, 0, len(names))
	for _, name := range names {
		out = append(out, Value{Name: name})
	}
	return out
}

func describedValues(descriptions valueDescriptions, names ...string) []Value {
	out := make([]Value, 0, len(names))
	for _, name := range names {
		out = append(out, Value{Name: name, Description: descriptions[name]})
	}
	return out
}

func usageLinesForSubcommand(command Command, subcommand string) []string {
	prefix := "adp " + command.Name + " " + subcommand
	var lines []string
	for _, line := range command.Usage {
		if strings.HasPrefix(line, prefix) {
			lines = append(lines, line)
		}
	}
	return lines
}

func writeUsageLines(out *strings.Builder, lines []string) {
	for _, line := range lines {
		out.WriteString("  ")
		out.WriteString(line)
		out.WriteByte('\n')
	}
}

func writeValuesSection(out *strings.Builder, title string, values []Value) {
	if len(values) == 0 {
		return
	}
	out.WriteByte('\n')
	out.WriteString(title)
	out.WriteString(":\n")
	for _, value := range values {
		out.WriteString("  ")
		out.WriteString(value.Name)
		if value.Description != "" {
			out.WriteString(" - ")
			out.WriteString(value.Description)
		}
		out.WriteByte('\n')
	}
}

func hasValue(values []Value, name string) bool {
	return valueDescription(values, name) != "" || hasValueName(values, name)
}

func hasValueName(values []Value, name string) bool {
	for _, value := range values {
		if value.Name == name {
			return true
		}
	}
	return false
}

func valueDescription(values []Value, name string) string {
	for _, value := range values {
		if value.Name == name {
			return value.Description
		}
	}
	return ""
}

// writeSeeAlsoSection writes the "See also" cross-reference section
func writeSeeAlsoSection(out *strings.Builder, commandName, subcommand string) {
	var related []string

	if subcommand != "" {
		// Subcommand help: check subcommand relationships first
		key := commandName + "." + subcommand
		if refs, ok := subcommandRelationships[key]; ok {
			related = append(related, refs...)
		}
		// Always include parent command
		related = append(related, commandName+" --help")
	} else {
		// Root command help: check command relationships
		if refs, ok := commandRelationships[commandName]; ok {
			related = append(related, refs...)
		}
	}

	if len(related) == 0 {
		return
	}

	out.WriteString("\nSee also:\n")
	for _, ref := range related {
		out.WriteString("  adp ")
		// Check if ref already contains flags (like --take, --help)
		if strings.Contains(ref, "--") {
			// Already has flags, use as-is
			out.WriteString(ref)
		} else if strings.Contains(ref, " ") {
			// Multi-word reference like "workspace doctor", add --help
			out.WriteString(ref)
			out.WriteString(" --help")
		} else {
			// Single command name, add --help
			out.WriteString(ref)
			out.WriteString(" --help")
		}
		out.WriteByte('\n')
	}
}
