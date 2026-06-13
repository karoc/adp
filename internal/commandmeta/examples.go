package commandmeta

import "strings"

var commandHelpExamples = map[string][]string{
	"quickstart": {
		"adp quickstart",
		"adp quickstart --non-interactive --workspace-name my-project --project-root /path/to/project --memory --mcp",
	},
	"workspace": {
		"adp workspace add game-a /absolute/path/to/project",
		"adp workspace list --format json",
		"adp workspace doctor game-a --format json",
	},
	"completion": {
		"source <(adp completion --shell bash)",
		"adp completion values tasks --workspace game-a",
	},
	"events": {
		"adp events list --workspace game-a --task task-20260611-0001 --format json",
		"adp events list --workspace game-a --task task-2026 --format json",
	},
	"sessions": {
		"adp sessions list --workspace game-a --agent codex --format json",
		"adp sessions show 20260611T10",
		"adp sessions resume-plan session-20260611-0001 --workspace game-a --agent claude --owner claude-main --lease 4h",
	},
	"runtime": {
		"adp runtime prune --older-than 24h --dry-run --format json",
	},
	"plan": {
		"adp plan doctor --workspace game-a --format json",
	},
	"progress": {
		"adp progress report --workspace game-a --format json",
	},
	"run": {
		"adp run codex --workspace game-a --take --owner codex-main --lease 4h",
		"adp run claude --workspace game-a --task task-20260611-0001 --keep-runtime",
		"adp run claude --workspace game-a --task task-001 --keep-runtime",
	},
	"version": {
		"adp version",
		"adp version --format json",
	},
}

var subcommandHelpExamples = map[string]map[string][]string{
	"workspace": {
		"add": {
			"adp workspace add game-a /absolute/path/to/project",
		},
		"list": {
			"adp workspace list",
			"adp workspace list --format json",
		},
		"show": {
			"adp workspace show game-a",
			"adp workspace show game-a --format json",
		},
		"doctor": {
			"adp workspace doctor game-a --verbose",
			"adp workspace doctor game-a --format json",
		},
	},
	"completion": {
		"values": {
			"adp completion values workspaces",
			"adp completion values tasks --workspace game-a",
		},
	},
	"events": {
		"list": {
			"adp events list --workspace game-a --task task-20260611-0001 --type run_finished --limit 5 --format json",
			"adp events list --workspace game-a --task task-2026 --limit 10",
		},
	},
	"sessions": {
		"list": {
			"adp sessions list --workspace game-a --agent codex --task task-20260611-0001 --format json",
			"adp sessions list --workspace game-a --task task-001 --format json",
		},
		"show": {
			"adp sessions show session-20260611-0001 --format json",
			"adp sessions show 20260611T10 --format json",
		},
		"restore-plan": {
			"adp sessions restore-plan session-20260611-0001 --format json",
			"adp sessions restore-plan 2026061 --format json",
		},
		"resume-plan": {
			"adp sessions resume-plan session-20260611-0001 --workspace game-a --agent claude --owner claude-main --lease 4h --format json",
			"adp sessions resume-plan 20260611-00 --workspace game-a --owner reviewer --lease 2h",
		},
	},
	"runtime": {
		"prune": {
			"adp runtime prune --older-than 24h --dry-run --format json",
		},
	},
	"tasks": {
		"next": {
			"adp tasks next --workspace game-a --limit 3 --format json",
		},
		"take": {
			"adp tasks take --workspace game-a --owner codex-main --lease 4h --format json",
		},
		"claim": {
			"adp tasks claim --workspace game-a task-20260611-0001 --owner codex-main --lease 4h",
			"adp tasks claim --workspace game-a task-001 --owner alice --lease 2h",
		},
		"renew": {
			"adp tasks renew --workspace game-a task-20260611-0001 --owner codex-main --lease 4h",
			"adp tasks renew --workspace game-a task-2026 --owner codex-main --lease 4h",
		},
		"stale": {
			"adp tasks stale --workspace game-a --format json",
		},
	},
	"plan": {
		"preview": {
			"adp plan preview --workspace game-a --file plan.yaml --format json",
		},
		"apply": {
			"adp plan apply --workspace game-a --file plan.yaml --format json",
		},
		"doctor": {
			"adp plan doctor --workspace game-a --format json",
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
	"progress": {
		"report": {
			"adp progress report --workspace game-a --language en --format markdown",
			"adp progress report --workspace game-a --format json",
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
