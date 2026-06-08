package shell

import (
	"errors"
	"testing"
)

func TestRenderHookDefaultsToPOSIXShellFunction(t *testing.T) {
	t.Parallel()

	got, err := RenderHook(HookOptions{})
	if err != nil {
		t.Fatalf("RenderHook returned error: %v", err)
	}

	want := `adp_enter() {
	if [ "$#" -ne 1 ]; then
		printf '%s\n' 'usage: adp_enter <workspace>' >&2
		return 2
	fi
	eval "$(adp env "$1" --cd)"
}
`
	if got != want {
		t.Fatalf("RenderHook() = %q, want %q", got, want)
	}
}

func TestRenderHookSupportsBashDefaultCommandName(t *testing.T) {
	t.Parallel()

	got, err := RenderHook(HookOptions{Shell: "/usr/local/bin/bash"})
	if err != nil {
		t.Fatalf("RenderHook returned error: %v", err)
	}

	want := `adp-enter() {
	if [ "$#" -ne 1 ]; then
		printf '%s\n' 'usage: adp-enter <workspace>' >&2
		return 2
	fi
	eval "$(adp env "$1" --cd)"
}
`
	if got != want {
		t.Fatalf("RenderHook() = %q, want %q", got, want)
	}
}

func TestRenderHookSupportsZshAndCustomFunctionName(t *testing.T) {
	t.Parallel()

	got, err := RenderHook(HookOptions{
		Shell:        "-zsh",
		FunctionName: "adp_workspace",
	})
	if err != nil {
		t.Fatalf("RenderHook returned error: %v", err)
	}

	want := `adp_workspace() {
	if [ "$#" -ne 1 ]; then
		printf '%s\n' 'usage: adp_workspace <workspace>' >&2
		return 2
	fi
	eval "$(adp env "$1" --cd)"
}
`
	if got != want {
		t.Fatalf("RenderHook() = %q, want %q", got, want)
	}
}

func TestRenderHookRejectsUnsupportedShell(t *testing.T) {
	t.Parallel()

	_, err := RenderHook(HookOptions{Shell: "fish"})
	if !errors.Is(err, ErrUnsupportedHookShell) {
		t.Fatalf("error = %v, want ErrUnsupportedHookShell", err)
	}
}

func TestRenderHookRejectsInvalidFunctionNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		shell        string
		functionName string
	}{
		{
			name:         "posix hyphen",
			shell:        "sh",
			functionName: "adp-enter",
		},
		{
			name:         "leading digit",
			shell:        "bash",
			functionName: "1adp",
		},
		{
			name:         "command separator",
			shell:        "zsh",
			functionName: "adp;enter",
		},
		{
			name:         "command substitution",
			shell:        "bash",
			functionName: "adp$(enter)",
		},
		{
			name:         "slash",
			shell:        "bash",
			functionName: "adp/enter",
		},
		{
			name:         "reserved word",
			shell:        "bash",
			functionName: "if",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := RenderHook(HookOptions{
				Shell:        tt.shell,
				FunctionName: tt.functionName,
			})
			if !errors.Is(err, ErrInvalidHookFunctionName) {
				t.Fatalf("error = %v, want ErrInvalidHookFunctionName", err)
			}
		})
	}
}
