package shell

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

const (
	defaultCompletionShell       = "bash"
	defaultCompletionCommandName = "adp"
)

var (
	ErrUnsupportedCompletionShell   = errors.New("unsupported shell for completion")
	ErrInvalidCompletionCommandName = errors.New("invalid shell completion command name")
)

type CompletionOptions struct {
	Shell       string
	CommandName string
}

func RenderCompletion(opts CompletionOptions) (string, error) {
	shellName := normalizeCompletionShell(opts.Shell)
	if !isSupportedCompletionShell(shellName) {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedCompletionShell, opts.Shell)
	}

	commandName := opts.CommandName
	if commandName == "" {
		commandName = defaultCompletionCommandName
	}
	if !isValidCompletionCommandName(commandName) {
		return "", fmt.Errorf("%w: %s", ErrInvalidCompletionCommandName, commandName)
	}

	switch shellName {
	case "bash":
		return renderBashCompletion(commandName), nil
	case "zsh":
		return renderZshCompletion(commandName), nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedCompletionShell, opts.Shell)
	}
}

func renderBashCompletion(commandName string) string {
	functionName := completionFunctionName(commandName)

	var out strings.Builder
	out.WriteString("# bash completion for ")
	out.WriteString(commandName)
	out.WriteByte('\n')
	out.WriteString(functionName)
	out.WriteString("() {\n")
	out.WriteString("\tlocal cur prev command subcommand\n")
	out.WriteString("\tCOMPREPLY=()\n")
	out.WriteString("\tcur=\"${COMP_WORDS[COMP_CWORD]}\"\n")
	out.WriteString("\tprev=\"${COMP_WORDS[COMP_CWORD-1]}\"\n")
	out.WriteString("\tcommand=\"${COMP_WORDS[1]}\"\n")
	out.WriteString("\tsubcommand=\"${COMP_WORDS[2]}\"\n")
	out.WriteByte('\n')
	out.WriteString("\tcase \"$prev\" in\n")
	out.WriteString("\t\t--shell|-s)\n")
	out.WriteString("\t\t\tCOMPREPLY=( $(compgen -W \"bash zsh\" -- \"$cur\") )\n")
	out.WriteString("\t\t\treturn 0\n")
	out.WriteString("\t\t\t;;\n")
	out.WriteString("\t\t--type)\n")
	out.WriteString("\t\t\tCOMPREPLY=( $(compgen -W \"run_started run_finished\" -- \"$cur\") )\n")
	out.WriteString("\t\t\treturn 0\n")
	out.WriteString("\t\t\t;;\n")
	out.WriteString("\t\t--older-than)\n")
	out.WriteString("\t\t\tCOMPREPLY=( $(compgen -W \"1h 24h 168h\" -- \"$cur\") )\n")
	out.WriteString("\t\t\treturn 0\n")
	out.WriteString("\t\t\t;;\n")
	out.WriteString("\tesac\n")
	out.WriteByte('\n')
	out.WriteString("\tif [ \"$COMP_CWORD\" -eq 1 ]; then\n")
	out.WriteString("\t\tCOMPREPLY=( $(compgen -W \"init workspace enter env shell-hook completion events sessions runtime tasks progress run\" -- \"$cur\") )\n")
	out.WriteString("\t\treturn 0\n")
	out.WriteString("\tfi\n")
	out.WriteByte('\n')
	out.WriteString("\tcase \"$command\" in\n")
	out.WriteString("\t\tworkspace)\n")
	out.WriteString("\t\t\tif [ \"$COMP_CWORD\" -eq 2 ]; then\n")
	out.WriteString("\t\t\t\tCOMPREPLY=( $(compgen -W \"add list show remove rename doctor\" -- \"$cur\") )\n")
	out.WriteString("\t\t\tfi\n")
	out.WriteString("\t\t\t;;\n")
	out.WriteString("\t\tenter)\n")
	out.WriteString("\t\t\tCOMPREPLY=( $(compgen -W \"--keep-runtime\" -- \"$cur\") )\n")
	out.WriteString("\t\t\t;;\n")
	out.WriteString("\t\tenv)\n")
	out.WriteString("\t\t\tCOMPREPLY=( $(compgen -W \"--cd\" -- \"$cur\") )\n")
	out.WriteString("\t\t\t;;\n")
	out.WriteString("\t\tshell-hook)\n")
	out.WriteString("\t\t\tCOMPREPLY=( $(compgen -W \"--shell -s --name --function -n\" -- \"$cur\") )\n")
	out.WriteString("\t\t\t;;\n")
	out.WriteString("\t\tcompletion)\n")
	out.WriteString("\t\t\tCOMPREPLY=( $(compgen -W \"--shell -s --command\" -- \"$cur\") )\n")
	out.WriteString("\t\t\t;;\n")
	out.WriteString("\t\tevents)\n")
	out.WriteString("\t\t\tif [ \"$COMP_CWORD\" -eq 2 ]; then\n")
	out.WriteString("\t\t\t\tCOMPREPLY=( $(compgen -W \"list\" -- \"$cur\") )\n")
	out.WriteString("\t\t\telif [ \"$subcommand\" = \"list\" ]; then\n")
	out.WriteString("\t\t\t\tCOMPREPLY=( $(compgen -W \"--workspace -w --session --type --limit\" -- \"$cur\") )\n")
	out.WriteString("\t\t\tfi\n")
	out.WriteString("\t\t\t;;\n")
	out.WriteString("\t\tsessions)\n")
	out.WriteString("\t\t\tif [ \"$COMP_CWORD\" -eq 2 ]; then\n")
	out.WriteString("\t\t\t\tCOMPREPLY=( $(compgen -W \"list show\" -- \"$cur\") )\n")
	out.WriteString("\t\t\telif [ \"$subcommand\" = \"list\" ]; then\n")
	out.WriteString("\t\t\t\tCOMPREPLY=( $(compgen -W \"--workspace -w --agent --limit\" -- \"$cur\") )\n")
	out.WriteString("\t\t\tfi\n")
	out.WriteString("\t\t\t;;\n")
	out.WriteString("\t\truntime)\n")
	out.WriteString("\t\t\tif [ \"$COMP_CWORD\" -eq 2 ]; then\n")
	out.WriteString("\t\t\t\tCOMPREPLY=( $(compgen -W \"prune\" -- \"$cur\") )\n")
	out.WriteString("\t\t\telif [ \"$subcommand\" = \"prune\" ]; then\n")
	out.WriteString("\t\t\t\tCOMPREPLY=( $(compgen -W \"--older-than --include-kept --dry-run\" -- \"$cur\") )\n")
	out.WriteString("\t\t\tfi\n")
	out.WriteString("\t\t\t;;\n")
	out.WriteString("\t\ttasks)\n")
	out.WriteString("\t\t\tif [ \"$COMP_CWORD\" -eq 2 ]; then\n")
	out.WriteString("\t\t\t\tCOMPREPLY=( $(compgen -W \"add list show update done block\" -- \"$cur\") )\n")
	out.WriteString("\t\t\telse\n")
	out.WriteString("\t\t\t\tCOMPREPLY=( $(compgen -W \"--workspace -w --priority --phase --description --status --reason\" -- \"$cur\") )\n")
	out.WriteString("\t\t\tfi\n")
	out.WriteString("\t\t\t;;\n")
	out.WriteString("\t\tprogress)\n")
	out.WriteString("\t\t\tCOMPREPLY=( $(compgen -W \"--workspace -w\" -- \"$cur\") )\n")
	out.WriteString("\t\t\t;;\n")
	out.WriteString("\t\trun)\n")
	out.WriteString("\t\t\tif [ \"$COMP_CWORD\" -eq 2 ]; then\n")
	out.WriteString("\t\t\t\tCOMPREPLY=( $(compgen -W \"codex claude\" -- \"$cur\") )\n")
	out.WriteString("\t\t\telse\n")
	out.WriteString("\t\t\t\tCOMPREPLY=( $(compgen -W \"--workspace -w --profile -p --keep-runtime --\" -- \"$cur\") )\n")
	out.WriteString("\t\t\tfi\n")
	out.WriteString("\t\t\t;;\n")
	out.WriteString("\tesac\n")
	out.WriteString("}\n")
	out.WriteByte('\n')
	out.WriteString("complete -F ")
	out.WriteString(functionName)
	out.WriteByte(' ')
	out.WriteString(commandName)
	out.WriteByte('\n')
	return out.String()
}

func renderZshCompletion(commandName string) string {
	functionName := completionFunctionName(commandName)

	var out strings.Builder
	out.WriteString("#compdef ")
	out.WriteString(commandName)
	out.WriteString("\n\n")
	out.WriteString(functionName)
	out.WriteString("() {\n")
	out.WriteString("\tlocal -a commands workspace_commands events_commands sessions_commands runtime_commands tasks_commands run_agents\n")
	out.WriteString("\tcommands=(\n")
	out.WriteString("\t\t'init:initialize ADP home'\n")
	out.WriteString("\t\t'workspace:manage registered workspaces'\n")
	out.WriteString("\t\t'enter:enter a kept runtime shell'\n")
	out.WriteString("\t\t'env:render runtime environment exports'\n")
	out.WriteString("\t\t'shell-hook:render shell helper function'\n")
	out.WriteString("\t\t'completion:render shell completion script'\n")
	out.WriteString("\t\t'events:read ADP event logs'\n")
	out.WriteString("\t\t'sessions:summarize ADP session history'\n")
	out.WriteString("\t\t'runtime:manage ADP runtime directories'\n")
	out.WriteString("\t\t'tasks:manage workspace task state'\n")
	out.WriteString("\t\t'progress:summarize workspace execution progress'\n")
	out.WriteString("\t\t'run:run an agent inside a runtime'\n")
	out.WriteString("\t)\n")
	out.WriteString("\tworkspace_commands=(add list show remove rename doctor)\n")
	out.WriteString("\tevents_commands=(list)\n")
	out.WriteString("\tsessions_commands=(list show)\n")
	out.WriteString("\truntime_commands=(prune)\n")
	out.WriteString("\ttasks_commands=(add list show update done block)\n")
	out.WriteString("\trun_agents=(codex claude)\n")
	out.WriteByte('\n')
	out.WriteString("\tif (( CURRENT == 2 )); then\n")
	out.WriteString("\t\t_describe -t commands 'adp command' commands\n")
	out.WriteString("\t\treturn\n")
	out.WriteString("\tfi\n")
	out.WriteByte('\n')
	out.WriteString("\tcase \"${words[2]}\" in\n")
	out.WriteString("\t\tworkspace)\n")
	out.WriteString("\t\t\tif (( CURRENT == 3 )); then\n")
	out.WriteString("\t\t\t\t_describe -t workspace-commands 'workspace command' workspace_commands\n")
	out.WriteString("\t\t\tfi\n")
	out.WriteString("\t\t\t;;\n")
	out.WriteString("\t\tenter)\n")
	out.WriteString("\t\t\t_values 'enter option' '--keep-runtime[keep runtime directory after shell exits]'\n")
	out.WriteString("\t\t\t;;\n")
	out.WriteString("\t\tenv)\n")
	out.WriteString("\t\t\t_values 'env option' '--cd[change directory to runtime root]'\n")
	out.WriteString("\t\t\t;;\n")
	out.WriteString("\t\tshell-hook)\n")
	out.WriteString("\t\t\t_values 'shell-hook option' '--shell[render for shell]' '-s[render for shell]' '--name[function name]' '--function[function name]' '-n[function name]'\n")
	out.WriteString("\t\t\t;;\n")
	out.WriteString("\t\tcompletion)\n")
	out.WriteString("\t\t\t_values 'completion option' '--shell[render for shell]' '-s[render for shell]' '--command[command name]'\n")
	out.WriteString("\t\t\t;;\n")
	out.WriteString("\t\tevents)\n")
	out.WriteString("\t\t\tif (( CURRENT == 3 )); then\n")
	out.WriteString("\t\t\t\t_describe -t events-commands 'events command' events_commands\n")
	out.WriteString("\t\t\telif [[ \"${words[3]}\" == \"list\" ]]; then\n")
	out.WriteString("\t\t\t\t_values 'events list option' '--workspace[filter by workspace]' '-w[filter by workspace]' '--session[filter by session]' '--type[filter by event type]' '--limit[limit result count]'\n")
	out.WriteString("\t\t\tfi\n")
	out.WriteString("\t\t\t;;\n")
	out.WriteString("\t\tsessions)\n")
	out.WriteString("\t\t\tif (( CURRENT == 3 )); then\n")
	out.WriteString("\t\t\t\t_describe -t sessions-commands 'sessions command' sessions_commands\n")
	out.WriteString("\t\t\telif [[ \"${words[3]}\" == \"list\" ]]; then\n")
	out.WriteString("\t\t\t\t_values 'sessions list option' '--workspace[filter by workspace]' '-w[filter by workspace]' '--agent[filter by agent]' '--limit[limit result count]'\n")
	out.WriteString("\t\t\tfi\n")
	out.WriteString("\t\t\t;;\n")
	out.WriteString("\t\truntime)\n")
	out.WriteString("\t\t\tif (( CURRENT == 3 )); then\n")
	out.WriteString("\t\t\t\t_describe -t runtime-commands 'runtime command' runtime_commands\n")
	out.WriteString("\t\t\telif [[ \"${words[3]}\" == \"prune\" ]]; then\n")
	out.WriteString("\t\t\t\t_values 'runtime prune option' '--older-than[minimum runtime age]' '--include-kept[include kept runtimes]' '--dry-run[print candidates without deleting]'\n")
	out.WriteString("\t\t\tfi\n")
	out.WriteString("\t\t\t;;\n")
	out.WriteString("\t\ttasks)\n")
	out.WriteString("\t\t\tif (( CURRENT == 3 )); then\n")
	out.WriteString("\t\t\t\t_describe -t tasks-commands 'tasks command' tasks_commands\n")
	out.WriteString("\t\t\telse\n")
	out.WriteString("\t\t\t\t_values 'tasks option' '--workspace[workspace name]' '-w[workspace name]' '--priority[task priority]' '--phase[task phase]' '--description[task description]' '--status[task status]' '--reason[blocked reason]'\n")
	out.WriteString("\t\t\tfi\n")
	out.WriteString("\t\t\t;;\n")
	out.WriteString("\t\tprogress)\n")
	out.WriteString("\t\t\t_values 'progress option' '--workspace[workspace name]' '-w[workspace name]'\n")
	out.WriteString("\t\t\t;;\n")
	out.WriteString("\t\trun)\n")
	out.WriteString("\t\t\tif (( CURRENT == 3 )); then\n")
	out.WriteString("\t\t\t\t_describe -t agents 'agent' run_agents\n")
	out.WriteString("\t\t\telse\n")
	out.WriteString("\t\t\t\t_values 'run option' '--workspace[workspace name]' '-w[workspace name]' '--profile[profile name]' '-p[profile name]' '--keep-runtime[keep runtime directory]' '--[pass following args to agent]'\n")
	out.WriteString("\t\t\tfi\n")
	out.WriteString("\t\t\t;;\n")
	out.WriteString("\tesac\n")
	out.WriteString("}\n")
	out.WriteByte('\n')
	out.WriteString("compdef ")
	out.WriteString(functionName)
	out.WriteByte(' ')
	out.WriteString(commandName)
	out.WriteByte('\n')
	return out.String()
}

func normalizeCompletionShell(shellName string) string {
	shellName = strings.TrimSpace(shellName)
	if shellName == "" {
		return defaultCompletionShell
	}

	shellName = filepath.Base(shellName)
	shellName = strings.TrimLeft(shellName, "-")
	shellName = strings.ToLower(shellName)
	return strings.TrimSuffix(shellName, ".exe")
}

func isSupportedCompletionShell(shellName string) bool {
	switch shellName {
	case "bash", "zsh":
		return true
	default:
		return false
	}
}

func isValidCompletionCommandName(commandName string) bool {
	if commandName == "" || isShellReservedWord(commandName) {
		return false
	}

	for index, r := range commandName {
		if index == 0 {
			if !isShellNameStart(r) {
				return false
			}
			continue
		}
		if r != '-' && !isShellNamePart(r) {
			return false
		}
	}
	return true
}

func completionFunctionName(commandName string) string {
	return "_" + strings.ReplaceAll(commandName, "-", "_") + "_completion"
}
