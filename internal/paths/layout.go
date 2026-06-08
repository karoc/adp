package paths

import (
	"os"
	"path/filepath"
)

const (
	EnvHome       = "ADP_HOME"
	EnvRuntimeDir = "ADP_RUNTIME_DIR"
)

type Layout struct {
	Home          string
	RuntimeParent string
	ConfigFile    string
	WorkspacesDir string
	LogsDir       string
	EventsFile    string
}

func New(home, runtimeParent string) Layout {
	return Layout{
		Home:          home,
		RuntimeParent: runtimeParent,
		ConfigFile:    filepath.Join(home, "config.yaml"),
		WorkspacesDir: filepath.Join(home, "workspaces"),
		LogsDir:       filepath.Join(home, "logs"),
		EventsFile:    filepath.Join(home, "logs", "events.jsonl"),
	}
}

func FromEnv() (Layout, error) {
	home := os.Getenv(EnvHome)
	if home == "" {
		userHome, err := os.UserHomeDir()
		if err != nil {
			return Layout{}, err
		}
		home = filepath.Join(userHome, ".adp")
	}

	runtimeParent := os.Getenv(EnvRuntimeDir)
	if runtimeParent == "" {
		runtimeParent = filepath.Join(os.TempDir(), "adp-runtime")
	}

	return New(home, runtimeParent), nil
}

func (l Layout) WorkspaceDir(name string) string {
	return filepath.Join(l.WorkspacesDir, name)
}

func (l Layout) WorkspaceConfig(name string) string {
	return filepath.Join(l.WorkspaceDir(name), "workspace.yaml")
}
