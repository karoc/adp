package commandmeta

import "strings"

var commandHelpExamples = map[string][]string{
	"run": {
		"adp run codex --workspace game-a --take --owner codex-main --lease 4h",
		"adp run claude --workspace game-a --task task-20260611-0001 --keep-runtime",
	},
}

var subcommandHelpExamples = map[string]map[string][]string{
	"tasks": {
		"next": {
			"adp tasks next --workspace game-a --limit 3 --format json",
		},
		"take": {
			"adp tasks take --workspace game-a --owner codex-main --lease 4h --format json",
		},
		"claim": {
			"adp tasks claim --workspace game-a task-20260611-0001 --owner codex-main --lease 4h",
		},
	},
	"phase": {
		"status": {
			"adp phase status --workspace game-a --format json",
		},
		"accept": {
			`adp phase accept --workspace game-a P60 --command "scripts/check-all.sh" --result passed --notes "runtime smoke passed"`,
		},
		"commit": {
			`adp phase commit --workspace game-a P60 --hash 2d1b7fe8e2a6d1f46d0101b14efeff129aab68bc --message "Polish task ownership command errors"`,
		},
		"push": {
			"adp phase push --workspace game-a P60 --remote origin --branch main --result pushed",
		},
	},
}

func examplesForCommand(name string) []string {
	return cloneStrings(commandHelpExamples[name])
}

func examplesForSubcommand(commandName, subcommand string) []string {
	subcommands := subcommandHelpExamples[commandName]
	if subcommands == nil {
		return nil
	}
	return cloneStrings(subcommands[subcommand])
}

func writeExamplesSection(out *strings.Builder, examples []string) {
	if len(examples) == 0 {
		return
	}
	out.WriteString("\nExamples:\n")
	for _, example := range examples {
		out.WriteString("  ")
		out.WriteString(example)
		out.WriteByte('\n')
	}
}

func cloneStrings(values []string) []string {
	return append([]string(nil), values...)
}
