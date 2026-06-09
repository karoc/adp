package shell

import (
	"errors"
	"strings"
	"testing"
)

func TestRenderCompletionDefaultsToBash(t *testing.T) {
	t.Parallel()

	got, err := RenderCompletion(CompletionOptions{})
	if err != nil {
		t.Fatalf("RenderCompletion returned error: %v", err)
	}

	for _, want := range []string{
		"# bash completion for adp\n",
		"_adp_completion() {\n",
		"completion values workspaces",
		"completion values profiles",
		"completion values agents",
		"completion values tasks",
		"completion values phases",
		"completion values sessions",
		"completion values owners",
		"completion values statuses",
		"init doctor version workspace enter env shell-hook completion events sessions runtime tasks plan phase progress run",
		"add list show remove rename doctor",
		"--shell -s --command",
		"[ \"$subcommand\" = \"values\" ] && [ \"$COMP_CWORD\" -eq 3 ]",
		"compgen -W \"--workspace -w\"",
		"--workspace -w --session --task --type --limit",
		"_adp_completion_dynamic_values sessions",
		"list show restore-plan",
		"--workspace -w --agent --task --limit",
		"--older-than --include-kept --dry-run",
		"add list next take show update claim release done block",
		"--workspace -w --priority --phase --description --status --reason --owner --lease --limit --format",
		"_adp_completion_dynamic_values tasks",
		"preview apply doctor",
		"--workspace -w --file -f --format",
		"add list show status start accept commit push",
		"--workspace -w --goal --command --result --notes --hash --message --remote --branch --format",
		"_adp_completion_dynamic_values phases",
		"report --workspace -w --format",
		"--workspace -w --format --language",
		"--workspace -w --profile -p --task --take --owner --lease --keep-runtime --",
		"markdown json",
		"text json",
		"en zh-CN",
		"complete -F _adp_completion adp\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("bash completion missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "eval") {
		t.Fatalf("bash completion should not contain eval:\n%s", got)
	}
	if strings.Contains(got, "codex claude") {
		t.Fatalf("bash completion should not hard-code run agents:\n%s", got)
	}

	gotAgain, err := RenderCompletion(CompletionOptions{})
	if err != nil {
		t.Fatalf("RenderCompletion returned error on second render: %v", err)
	}
	if got != gotAgain {
		t.Fatal("RenderCompletion output is not deterministic")
	}
}

func TestRenderCompletionNormalizesBashShellPathAndCommandName(t *testing.T) {
	t.Parallel()

	got, err := RenderCompletion(CompletionOptions{
		Shell:       "/usr/local/bin/bash.exe",
		CommandName: "adp-dev",
	})
	if err != nil {
		t.Fatalf("RenderCompletion returned error: %v", err)
	}

	for _, want := range []string{
		"# bash completion for adp-dev\n",
		"_adp_dev_completion() {\n",
		"complete -F _adp_dev_completion adp-dev\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("bash completion missing %q:\n%s", want, got)
		}
	}
}

func TestRenderCompletionSupportsZsh(t *testing.T) {
	t.Parallel()

	got, err := RenderCompletion(CompletionOptions{Shell: "-zsh"})
	if err != nil {
		t.Fatalf("RenderCompletion returned error: %v", err)
	}

	for _, want := range []string{
		"#compdef adp\n",
		"completion values workspaces",
		"completion values profiles",
		"completion values agents",
		"completion values tasks",
		"completion values phases",
		"completion values sessions",
		"completion values owners",
		"completion values statuses",
		"_adp_completion() {\n",
		"'doctor:diagnose registered workspaces'",
		"'version:print version information'",
		"'workspace:manage registered workspaces'",
		"'completion:render shell completion script'",
		"'sessions:summarize ADP session history'",
		"'tasks:manage workspace task state'",
		"'plan:preview, apply, or diagnose local planning state'",
		"'phase:manage workspace phase gates'",
		"'progress:summarize workspace execution progress'",
		"workspace_commands=(add list show remove rename doctor)",
		"events_commands=(list)",
		"sessions_commands=(list show restore-plan)",
		"runtime_commands=(prune)",
		"tasks_commands=(add list next take show update claim release done block)",
		"plan_commands=(preview apply doctor)",
		"phase_commands=(add list show status start accept commit push)",
		"'run:run an agent inside a runtime'",
		"'--shell[render for shell]'",
		"'--command[command name]'",
		"[[ \"${words[3]}\" == \"values\" && CURRENT == 4 ]]",
		"_values 'completion values option' '--workspace[workspace name]' '-w[workspace name]'",
		"'--agent[filter by agent]'",
		"'--task[filter by task]'",
		"_adp_completion_dynamic_values tasks",
		"_adp_completion_dynamic_values phases",
		"_adp_completion_dynamic_values sessions",
		"'--workspace[workspace name]'",
		"'--limit[result limit]'",
		"'--format[output format]'",
		"'--file[structured plan input path]'",
		"'report[render progress report]'",
		"'text[text output]'",
		"'markdown[Markdown report]'",
		"'--language[report language]'",
		"'zh-CN[Simplified Chinese]'",
		"_values 'run option' '--workspace[workspace name]' '-w[workspace name]' '--profile[profile name]' '-p[profile name]' '--task[task id]' '--take[atomically take next task before launch]' '--owner[task owner]' '--lease[claim lease duration]' '--keep-runtime[keep runtime directory]' '--[pass following args to agent]'",
		"compdef _adp_completion adp\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("zsh completion missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "eval") {
		t.Fatalf("zsh completion should not contain eval:\n%s", got)
	}
	if strings.Contains(got, "run_agents=(codex claude)") {
		t.Fatalf("zsh completion should not hard-code run agents:\n%s", got)
	}
}

func TestRenderCompletionRejectsUnsupportedShell(t *testing.T) {
	t.Parallel()

	for _, shellName := range []string{"sh", "/bin/fish"} {
		shellName := shellName
		t.Run(shellName, func(t *testing.T) {
			t.Parallel()

			_, err := RenderCompletion(CompletionOptions{Shell: shellName})
			if !errors.Is(err, ErrUnsupportedCompletionShell) {
				t.Fatalf("error = %v, want ErrUnsupportedCompletionShell", err)
			}
		})
	}
}

func TestRenderCompletionRejectsInvalidCommandNames(t *testing.T) {
	t.Parallel()

	tests := []string{
		"1adp",
		"-adp",
		"adp;rm",
		"adp$(whoami)",
		"adp/cli",
		"if",
	}

	for _, commandName := range tests {
		commandName := commandName
		t.Run(commandName, func(t *testing.T) {
			t.Parallel()

			_, err := RenderCompletion(CompletionOptions{CommandName: commandName})
			if !errors.Is(err, ErrInvalidCompletionCommandName) {
				t.Fatalf("error = %v, want ErrInvalidCompletionCommandName", err)
			}
		})
	}
}
