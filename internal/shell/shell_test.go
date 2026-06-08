package shell

import (
	"context"
	"errors"
	"testing"

	"github.com/karoc/adp/internal/adapters"
)

func TestNewSpecUsesPreferredShellRuntimeRootAndEnv(t *testing.T) {
	t.Parallel()

	handle := adapters.RuntimeHandle{
		Root: "/tmp/adp-runtime/workspace-session",
		Env: map[string]string{
			"ADP_RUNTIME_ROOT": "/tmp/adp-runtime/workspace-session",
			"ADP_WORKSPACE":    "game-a",
		},
	}

	spec := newSpec(handle, "/usr/local/bin/fish")
	if spec.Command != "/usr/local/bin/fish" {
		t.Fatalf("Command = %q, want preferred shell", spec.Command)
	}
	if spec.Dir != handle.Root {
		t.Fatalf("Dir = %q, want runtime root %q", spec.Dir, handle.Root)
	}
	if spec.Env["ADP_WORKSPACE"] != "game-a" {
		t.Fatalf("runtime env was not injected: %#v", spec.Env)
	}

	spec.Env["ADP_WORKSPACE"] = "changed"
	if handle.Env["ADP_WORKSPACE"] != "game-a" {
		t.Fatalf("NewSpec mutated the runtime handle env")
	}
}

func TestNewSpecDefaultsToBinSh(t *testing.T) {
	t.Parallel()

	spec := newSpec(adapters.RuntimeHandle{Root: "/runtime"}, "")
	if spec.Command != defaultShell {
		t.Fatalf("Command = %q, want %q", spec.Command, defaultShell)
	}
}

func TestEnterRejectsMissingRuntimeRoot(t *testing.T) {
	t.Parallel()

	err := Enter(context.Background(), adapters.RuntimeHandle{}, Streams{})
	if !errors.Is(err, ErrRuntimeRootRequired) {
		t.Fatalf("error = %v, want ErrRuntimeRootRequired", err)
	}
}
