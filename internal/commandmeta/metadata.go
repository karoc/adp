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
}

var rootCommands = []Command{
	{Name: "init", Description: "initialize ADP home", Usage: []string{"adp init"}},
	{Name: "doctor", Description: "diagnose registered workspaces", Usage: []string{"adp doctor [workspace]"}},
	{Name: "version", Description: "print version information", Usage: []string{"adp version"}},
	{
		Name:        "workspace",
		Description: "manage registered workspaces",
		Usage: []string{
			"adp workspace add <name> <project-root>",
			"adp workspace list",
			"adp workspace show <name>",
			"adp workspace remove <name>",
			"adp workspace rename <old-name> <new-name>",
			"adp workspace doctor [name]",
		},
		Subcommands: describedValues(valueDescriptions{
			"add":    "register a project root",
			"list":   "list registered workspaces",
			"show":   "show one workspace",
			"remove": "remove workspace registration",
			"rename": "rename workspace registration",
			"doctor": "diagnose workspace config",
		}, "add", "list", "show", "remove", "rename", "doctor"),
	},
	{
		Name:        "enter",
		Description: "enter a kept runtime shell",
		Usage:       []string{"adp enter <workspace> [--keep-runtime]"},
		Options:     describedValues(valueDescriptions{"--keep-runtime": "keep runtime directory after shell exits"}, "--keep-runtime"),
	},
	{
		Name:        "env",
		Description: "render runtime environment exports",
		Usage:       []string{"adp env <workspace> [--cd]"},
		Options:     describedValues(valueDescriptions{"--cd": "change directory to runtime root"}, "--cd"),
	},
	{
		Name:        "shell-hook",
		Description: "render shell helper function",
		Usage:       []string{"adp shell-hook [--shell <sh|bash|zsh>] [--name <function-name>]"},
		Options: describedValues(valueDescriptions{
			"--shell":    "render for shell",
			"-s":         "render for shell",
			"--name":     "function name",
			"--function": "function name",
			"-n":         "function name",
		}, "--shell", "-s", "--name", "--function", "-n"),
	},
	{
		Name:        "completion",
		Description: "render shell completion script",
		Usage: []string{
			"adp completion [--shell <bash|zsh>] [--command <name>]",
			"adp completion values <agents|workspaces|profiles|tasks|phases|sessions|owners|statuses> [--workspace <name>]",
		},
		Subcommands: describedValues(valueDescriptions{"values": "print dynamic completion values"}, "values"),
		Options: describedValues(valueDescriptions{
			"--shell":     "render for shell",
			"-s":          "render for shell",
			"--command":   "command name",
			"--workspace": "workspace name",
			"-w":          "workspace name",
		}, "--shell", "-s", "--command", "--workspace", "-w"),
	},
	{
		Name:        "events",
		Description: "read ADP event logs",
		Usage:       []string{"adp events list [--workspace <name>] [--session <session-id>] [--task <task-id>] [--type <event-type>] [--limit <n>]"},
		Subcommands: describedValues(valueDescriptions{"list": "list runtime events"}, "list"),
		Options: describedValues(valueDescriptions{
			"--workspace": "filter by workspace",
			"-w":          "filter by workspace",
			"--session":   "filter by session",
			"--task":      "filter by task",
			"--type":      "filter by event type",
			"--limit":     "limit result count",
		}, "--workspace", "-w", "--session", "--task", "--type", "--limit"),
	},
	{
		Name:        "sessions",
		Description: "summarize ADP session history",
		Usage: []string{
			"adp sessions list [--workspace <name>] [--agent <agent>] [--task <task-id>] [--limit <n>]",
			"adp sessions show <session-id>",
			"adp sessions restore-plan <session-id>",
			"adp sessions resume-plan <session-id> [--workspace <name>] [--owner <owner>] [--lease <duration>] [--agent <agent>] [--format <text|json>]",
		},
		Subcommands: describedValues(valueDescriptions{
			"list":         "list runtime sessions",
			"show":         "show one session",
			"restore-plan": "print read-only rerun guidance",
			"resume-plan":  "print read-only cross-tool resume guidance",
		}, "list", "show", "restore-plan", "resume-plan"),
		Options: describedValues(valueDescriptions{
			"--workspace": "filter by workspace",
			"-w":          "filter by workspace",
			"--agent":     "agent filter or resume target",
			"--task":      "filter by task",
			"--limit":     "limit result count",
			"--owner":     "task owner",
			"--lease":     "ownership lease duration",
			"--format":    "output format",
		}, "--workspace", "-w", "--agent", "--task", "--limit", "--owner", "--lease", "--format"),
	},
	{
		Name:        "runtime",
		Description: "manage ADP runtime directories",
		Usage:       []string{"adp runtime prune [--older-than <duration>] [--include-kept] [--dry-run]"},
		Subcommands: describedValues(valueDescriptions{"prune": "delete stale ADP-owned runtimes"}, "prune"),
		Options: describedValues(valueDescriptions{
			"--older-than":   "minimum runtime age",
			"--include-kept": "include kept runtimes",
			"--dry-run":      "print candidates without deleting",
		}, "--older-than", "--include-kept", "--dry-run"),
	},
	{
		Name:        "tasks",
		Description: "manage the local workspace task board",
		Usage: []string{
			"adp tasks add [--workspace <name>] [--priority <value>] [--phase <value>] [--description <text>] <title>",
			"adp tasks list [--workspace <name>] [--format <text|json>]",
			"adp tasks next [--workspace <name>] [--limit <n>] [--format <text|json>]",
			"adp tasks take [--workspace <name>] --owner <owner> [--lease <duration>] [--format <text|json>]",
			"adp tasks stale [--workspace <name>] [--format <text|json>]",
			"adp tasks show [--workspace <name>] <task-id> [--format <text|json>]",
			"adp tasks update [--workspace <name>] <task-id> --status <status>",
			"adp tasks claim [--workspace <name>] <task-id> --owner <owner> [--lease <duration>]",
			"adp tasks renew [--workspace <name>] <task-id> --owner <owner> --lease <duration>",
			"adp tasks release [--workspace <name>] <task-id> [--owner <owner>]",
			"adp tasks done [--workspace <name>] <task-id>",
			"adp tasks block [--workspace <name>] <task-id> --reason <reason>",
		},
		Subcommands: describedValues(valueDescriptions{
			"add":     "add a task to the board",
			"list":    "list tasks",
			"next":    "preview next claimable work",
			"take":    "atomically claim next work",
			"stale":   "inspect expired in-progress claims",
			"show":    "show one task",
			"update":  "set task status",
			"claim":   "claim a selected task",
			"renew":   "extend an owned task lease",
			"release": "release a claim",
			"done":    "mark a task done",
			"block":   "mark a task blocked",
		}, "add", "list", "next", "take", "stale", "show", "update", "claim", "renew", "release", "done", "block"),
		Options: describedValues(valueDescriptions{
			"--workspace":   "workspace name",
			"-w":            "workspace name",
			"--priority":    "task priority",
			"--phase":       "task phase",
			"--description": "task description",
			"--status":      "task status",
			"--reason":      "blocked reason",
			"--owner":       "task owner",
			"--lease":       "claim lease duration",
			"--limit":       "result limit",
			"--format":      "output format",
		}, "--workspace", "-w", "--priority", "--phase", "--description", "--status", "--reason", "--owner", "--lease", "--limit", "--format"),
	},
	{
		Name:        "plan",
		Description: "preview, apply, or diagnose local planning state",
		Usage: []string{
			"adp plan preview [--workspace <name>] --file <path|-> [--format <text|json>]",
			"adp plan apply [--workspace <name>] --file <path|-> [--format <text|json>]",
			"adp plan doctor [--workspace <name>] [--format <text|json>]",
		},
		Subcommands: describedValues(valueDescriptions{
			"preview": "validate plan input without writing",
			"apply":   "write validated plan input",
			"doctor":  "diagnose local planning ledger",
		}, "preview", "apply", "doctor"),
		Options: describedValues(valueDescriptions{
			"--workspace": "workspace name",
			"-w":          "workspace name",
			"--file":      "structured plan input path",
			"-f":          "structured plan input path",
			"--format":    "output format",
		}, "--workspace", "-w", "--file", "-f", "--format"),
	},
	{
		Name:        "phase",
		Description: "manage workspace phase gates",
		Usage: []string{
			"adp phase add [--workspace <name>] [--goal <text>] <phase-id> <title>",
			"adp phase list [--workspace <name>] [--format <text|json>]",
			"adp phase show [--workspace <name>] <phase-id> [--format <text|json>]",
			"adp phase status [--workspace <name>] [--format <text|json>]",
			"adp phase start [--workspace <name>] <phase-id>",
			"adp phase accept [--workspace <name>] <phase-id> [--command <cmd>] [--result <result>] [--notes <text>]",
			"adp phase commit [--workspace <name>] <phase-id> --hash <commit-hash> [--message <text>]",
			"adp phase push [--workspace <name>] <phase-id> --remote <remote> --branch <branch> [--result <result>]",
		},
		Subcommands: describedValues(valueDescriptions{
			"add":    "add a phase",
			"list":   "list phases",
			"show":   "show one phase",
			"status": "show next gate action",
			"start":  "start the next planned phase",
			"accept": "record validation evidence",
			"commit": "record commit evidence",
			"push":   "record push evidence",
		}, "add", "list", "show", "status", "start", "accept", "commit", "push"),
		Options: describedValues(valueDescriptions{
			"--workspace": "workspace name",
			"-w":          "workspace name",
			"--goal":      "phase goal",
			"--command":   "acceptance command",
			"--result":    "gate result",
			"--notes":     "gate notes",
			"--hash":      "commit hash",
			"--message":   "commit message",
			"--remote":    "push remote",
			"--branch":    "push branch",
			"--format":    "output format",
		}, "--workspace", "-w", "--goal", "--command", "--result", "--notes", "--hash", "--message", "--remote", "--branch", "--format"),
	},
	{
		Name:        "progress",
		Description: "summarize workspace execution progress",
		Usage: []string{
			"adp progress [--workspace <name>] [--format <text|json>]",
			"adp progress report [--workspace <name>] [--language <en|zh-CN>] [--format <markdown|json>]",
		},
		Subcommands: describedValues(valueDescriptions{"report": "render progress report"}, "report"),
		Options: describedValues(valueDescriptions{
			"--workspace": "workspace name",
			"-w":          "workspace name",
			"--format":    "output format",
			"--language":  "report language",
		}, "--workspace", "-w", "--format", "--language"),
	},
	{
		Name:        "run",
		Description: "run an agent inside a runtime",
		Usage:       []string{"adp run <agent> [--workspace <name>] [--profile <profile>] [--task <task-id>|--take --owner <owner> [--lease <duration>]] [--keep-runtime] [-- <agent-args>...]"},
		Options: describedValues(valueDescriptions{
			"--workspace":    "workspace name",
			"-w":             "workspace name",
			"--profile":      "profile name",
			"-p":             "profile name",
			"--task":         "task id",
			"--take":         "atomically take next task before launch",
			"--owner":        "task owner",
			"--lease":        "claim lease duration",
			"--keep-runtime": "keep runtime directory",
			"--":             "pass following args to agent",
		}, "--workspace", "-w", "--profile", "-p", "--task", "--take", "--owner", "--lease", "--keep-runtime", "--"),
	},
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
	out.WriteString("adp - Agent Development Platform\n\nUsage:\n")
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
	out.WriteString("\nSee also:\n  adp ")
	out.WriteString(command.Name)
	out.WriteString(" --help\n")
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
