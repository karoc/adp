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
		"init doctor version workspace enter env shell-hook completion events sessions runtime tasks progress run",
		"add list show remove rename doctor",
		"--shell -s --command",
		"--workspace -w --session --task --type --limit",
		"list show",
		"--workspace -w --agent --task --limit",
		"--older-than --include-kept --dry-run",
		"add list show update done block",
		"--workspace -w --priority --phase --description --status --reason",
		"codex claude",
		"complete -F _adp_completion adp\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("bash completion missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "eval") {
		t.Fatalf("bash completion should not contain eval:\n%s", got)
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
		"_adp_completion() {\n",
		"'doctor:diagnose registered workspaces'",
		"'version:print version information'",
		"'workspace:manage registered workspaces'",
		"'completion:render shell completion script'",
		"'sessions:summarize ADP session history'",
		"'tasks:manage workspace task state'",
		"'progress:summarize workspace execution progress'",
		"workspace_commands=(add list show remove rename doctor)",
		"events_commands=(list)",
		"sessions_commands=(list show)",
		"runtime_commands=(prune)",
		"tasks_commands=(add list show update done block)",
		"'--shell[render for shell]'",
		"'--command[command name]'",
		"'--agent[filter by agent]'",
		"'--task[filter by task]'",
		"'--workspace[workspace name]'",
		"compdef _adp_completion adp\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("zsh completion missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "eval") {
		t.Fatalf("zsh completion should not contain eval:\n%s", got)
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
