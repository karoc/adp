package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewDerivesLayoutPaths(t *testing.T) {
	l := New("/home/u/.adp", "/tmp/adp-runtime")
	cases := map[string]string{
		"Home":          "/home/u/.adp",
		"RuntimeParent": "/tmp/adp-runtime",
		"ConfigFile":    "/home/u/.adp/config.yaml",
		"WorkspacesDir": "/home/u/.adp/workspaces",
		"LogsDir":       "/home/u/.adp/logs",
		"EventsFile":    "/home/u/.adp/logs/events.jsonl",
	}
	got := map[string]string{
		"Home":          l.Home,
		"RuntimeParent": l.RuntimeParent,
		"ConfigFile":    l.ConfigFile,
		"WorkspacesDir": l.WorkspacesDir,
		"LogsDir":       l.LogsDir,
		"EventsFile":    l.EventsFile,
	}
	for field, want := range cases {
		if got[field] != want {
			t.Errorf("%s = %q, want %q", field, got[field], want)
		}
	}
}

func TestWorkspaceDirAndConfig(t *testing.T) {
	l := New("/home/u/.adp", "/tmp/rt")
	if got, want := l.WorkspaceDir("demo"), "/home/u/.adp/workspaces/demo"; got != want {
		t.Errorf("WorkspaceDir = %q, want %q", got, want)
	}
	if got, want := l.WorkspaceConfig("demo"), "/home/u/.adp/workspaces/demo/workspace.yaml"; got != want {
		t.Errorf("WorkspaceConfig = %q, want %q", got, want)
	}
}

func TestFromEnvUsesEnvironmentValues(t *testing.T) {
	home := t.TempDir()
	runtime := t.TempDir()
	t.Setenv(EnvHome, home)
	t.Setenv(EnvRuntimeDir, runtime)

	l, err := FromEnv()
	if err != nil {
		t.Fatalf("FromEnv: %v", err)
	}
	if l.Home != home {
		t.Errorf("Home = %q, want %q", l.Home, home)
	}
	if l.RuntimeParent != runtime {
		t.Errorf("RuntimeParent = %q, want %q", l.RuntimeParent, runtime)
	}
	if want := filepath.Join(home, "config.yaml"); l.ConfigFile != want {
		t.Errorf("ConfigFile = %q, want %q", l.ConfigFile, want)
	}
}

func TestFromEnvDefaultsWhenUnset(t *testing.T) {
	// Point HOME at a temp dir so the fallback ~/.adp is deterministic, and
	// clear the ADP-specific overrides to exercise both default branches.
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)
	t.Setenv(EnvHome, "")
	t.Setenv(EnvRuntimeDir, "")

	l, err := FromEnv()
	if err != nil {
		t.Fatalf("FromEnv: %v", err)
	}
	if want := filepath.Join(fakeHome, ".adp"); l.Home != want {
		t.Errorf("Home = %q, want default %q", l.Home, want)
	}
	if want := filepath.Join(os.TempDir(), "adp-runtime"); l.RuntimeParent != want {
		t.Errorf("RuntimeParent = %q, want default %q", l.RuntimeParent, want)
	}
}
