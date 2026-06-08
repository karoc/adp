package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestPhaseCommandsRecordGateLifecycle(t *testing.T) {
	store := &fakeTaskStore{}
	deps := Dependencies{
		WorkspaceStore:   &fakeStore{cfg: testConfig()},
		TaskStoreFactory: func(string) TaskStore { return store },
	}
	var stdout bytes.Buffer

	commands := [][]string{
		{"phase", "add", "--workspace", "game-a", "--goal", "local phase gates", "p3", "Project", "planning"},
		{"phase", "start", "--workspace", "game-a", "p3"},
		{"phase", "accept", "--workspace", "game-a", "p3", "--command", "scripts/task-manager-smoke.sh", "--result", "passed", "--notes", "runtime smoke accepted"},
		{"phase", "commit", "--workspace", "game-a", "p3", "--hash", "abc123", "--message", "Add phase gates"},
		{"phase", "push", "--workspace", "game-a", "p3", "--remote", "origin", "--branch", "main", "--result", "pushed"},
		{"phase", "list", "--workspace", "game-a"},
		{"phase", "show", "--workspace", "game-a", "p3"},
	}
	for _, args := range commands {
		code := NewApp(deps, &stdout, &bytes.Buffer{}).Execute(context.Background(), args)
		if code != 0 {
			t.Fatalf("adp %v exit code = %d, want 0", args, code)
		}
	}

	output := stdout.String()
	for _, want := range []string{"phase p3 added", "status: active", "accepted: passed", "commit: abc123", "push: origin/main pushed", "Project planning", "commit_hash: abc123", "push_result: pushed"} {
		if !strings.Contains(output, want) {
			t.Fatalf("phase output missing %q: %q", want, output)
		}
	}
}
