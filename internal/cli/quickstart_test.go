package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/karoc/adp/internal/events"
	"github.com/karoc/adp/internal/paths"
)

// TestParseQuickstartArgs tests argument parsing.
func TestParseQuickstartArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    QuickstartOptions
		wantErr bool
	}{
		{
			name: "no args",
			args: []string{},
			want: QuickstartOptions{},
		},
		{
			name: "non-interactive flag",
			args: []string{"--non-interactive"},
			want: QuickstartOptions{NonInteractive: true},
		},
		{
			name: "workspace name",
			args: []string{"--workspace-name", "my-workspace"},
			want: QuickstartOptions{WorkspaceName: "my-workspace"},
		},
		{
			name: "project root",
			args: []string{"--project-root", "/path/to/project"},
			want: QuickstartOptions{ProjectRoot: "/path/to/project"},
		},
		{
			name: "adp home",
			args: []string{"--adp-home", "/custom/home"},
			want: QuickstartOptions{ADPHome: "/custom/home"},
		},
		{
			name: "enable memory",
			args: []string{"--memory"},
			want: QuickstartOptions{EnableMemory: true},
		},
		{
			name: "enable mcp",
			args: []string{"--mcp"},
			want: QuickstartOptions{EnableMCP: true},
		},
		{
			name: "enable agents",
			args: []string{"--agents"},
			want: QuickstartOptions{EnableAgents: true},
		},
		{
			name: "full non-interactive",
			args: []string{
				"--non-interactive",
				"--workspace-name", "test-ws",
				"--project-root", "/tmp/project",
				"--memory",
				"--mcp",
			},
			want: QuickstartOptions{
				NonInteractive: true,
				WorkspaceName:  "test-ws",
				ProjectRoot:    "/tmp/project",
				EnableMemory:   true,
				EnableMCP:      true,
			},
		},
		{
			name:    "workspace-name without value",
			args:    []string{"--workspace-name"},
			wantErr: true,
		},
		{
			name:    "project-root without value",
			args:    []string{"--project-root"},
			wantErr: true,
		},
		{
			name:    "adp-home without value",
			args:    []string{"--adp-home"},
			wantErr: true,
		},
		{
			name:    "unknown flag",
			args:    []string{"--unknown-flag"},
			wantErr: true,
		},
		{
			name:    "help flag",
			args:    []string{"--help"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseQuickstartArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseQuickstartArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseQuickstartArgs() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestValidateWorkspaceName tests workspace name validation.
func TestValidateWorkspaceName(t *testing.T) {
	tests := []struct {
		name    string
		wsName  string
		wantErr bool
	}{
		{
			name:   "valid simple name",
			wsName: "my-workspace",
		},
		{
			name:   "valid with underscores",
			wsName: "my_workspace",
		},
		{
			name:   "valid with dots",
			wsName: "my.workspace",
		},
		{
			name:   "valid alphanumeric",
			wsName: "workspace123",
		},
		{
			name:   "valid mixed case",
			wsName: "MyWorkspace",
		},
		{
			name:   "valid complex",
			wsName: "my-workspace_v1.0",
		},
		{
			name:    "empty name",
			wsName:  "",
			wantErr: true,
		},
		{
			name:    "contains space",
			wsName:  "my workspace",
			wantErr: true,
		},
		{
			name:    "contains slash",
			wsName:  "my/workspace",
			wantErr: true,
		},
		{
			name:    "contains special char",
			wsName:  "my@workspace",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWorkspaceName(tt.wsName)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateWorkspaceName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateProjectRoot tests project root validation.
func TestValidateProjectRoot(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
		{
			name:    "non-existent path",
			path:    "/this/path/does/not/exist",
			wantErr: true,
		},
		{
			name: "valid existing directory",
			path: "/tmp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProjectRoot(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateProjectRoot() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestExpandPath tests path expansion with ~.
func TestExpandPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
		check   func(string) bool
	}{
		{
			name: "empty path",
			path: "",
			check: func(result string) bool {
				return result == ""
			},
		},
		{
			name: "absolute path",
			path: "/tmp/test",
			check: func(result string) bool {
				return result == "/tmp/test"
			},
		},
		{
			name: "tilde only",
			path: "~",
			check: func(result string) bool {
				return result != "" && result != "~"
			},
		},
		{
			name: "tilde with path",
			path: "~/test",
			check: func(result string) bool {
				return strings.HasSuffix(result, "/test") && !strings.HasPrefix(result, "~")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := expandPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("expandPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil && !tt.check(got) {
				t.Errorf("expandPath() = %v, check failed", got)
			}
		})
	}
}

// TestQuickstartNonInteractiveRequiresWorkspaceName tests that non-interactive mode
// requires workspace name.
func TestQuickstartNonInteractiveRequiresWorkspaceName(t *testing.T) {
	app := &App{
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
		deps: Dependencies{
			Layout:         paths.Layout{Home: "/tmp/adp-home"},
			WorkspaceStore: &fakeStore{},
		},
	}

	err := app.quickstart(context.Background(), []string{
		"--non-interactive",
		"--project-root", "/tmp",
	})

	if err == nil {
		t.Fatal("expected error when workspace-name is missing in non-interactive mode")
	}

	if !strings.Contains(err.Error(), "workspace-name") {
		t.Errorf("error should mention workspace-name, got: %v", err)
	}
}

// TestQuickstartNonInteractiveRequiresProjectRoot tests that non-interactive mode
// requires project root.
func TestQuickstartNonInteractiveRequiresProjectRoot(t *testing.T) {
	app := &App{
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
		deps: Dependencies{
			Layout:         paths.Layout{Home: "/tmp/adp-home"},
			WorkspaceStore: &fakeStore{},
		},
	}

	err := app.quickstart(context.Background(), []string{
		"--non-interactive",
		"--workspace-name", "test-ws",
	})

	if err == nil {
		t.Fatal("expected error when project-root is missing in non-interactive mode")
	}

	if !strings.Contains(err.Error(), "project-root") {
		t.Errorf("error should mention project-root, got: %v", err)
	}
}

// TestQuickstartNonInteractiveValidatesWorkspaceName tests workspace name validation
// in non-interactive mode.
func TestQuickstartNonInteractiveValidatesWorkspaceName(t *testing.T) {
	app := &App{
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
		deps: Dependencies{
			Layout:         paths.Layout{Home: "/tmp/adp-home"},
			WorkspaceStore: &fakeStore{},
		},
	}

	err := app.quickstart(context.Background(), []string{
		"--non-interactive",
		"--workspace-name", "invalid name",
		"--project-root", "/tmp",
	})

	if err == nil {
		t.Fatal("expected error for invalid workspace name")
	}

	if !strings.Contains(err.Error(), "workspace name") {
		t.Errorf("error should mention workspace name, got: %v", err)
	}
}

// TestQuickstartNonInteractiveValidatesProjectRoot tests project root validation
// in non-interactive mode.
func TestQuickstartNonInteractiveValidatesProjectRoot(t *testing.T) {
	app := &App{
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
		deps: Dependencies{
			Layout:         paths.Layout{Home: "/tmp/adp-home"},
			WorkspaceStore: &fakeStore{},
		},
	}

	err := app.quickstart(context.Background(), []string{
		"--non-interactive",
		"--workspace-name", "test-ws",
		"--project-root", "/this/does/not/exist",
	})

	if err == nil {
		t.Fatal("expected error for non-existent project root")
	}

	if !strings.Contains(err.Error(), "project root") {
		t.Errorf("error should mention project root, got: %v", err)
	}
}

// TestQuickstartNonInteractiveSuccess tests successful non-interactive quickstart.
func TestQuickstartNonInteractiveSuccess(t *testing.T) {
	store := &fakeStore{
		cfg: testConfig(),
	}

	eventLogger := eventLoggerFunc(func(context.Context, events.Event) error {
		return nil
	})

	app := &App{
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
		deps: Dependencies{
			Layout:         paths.Layout{Home: "/tmp/adp-home"},
			WorkspaceStore: store,
			EventLogger:    eventLogger,
		},
	}

	err := app.quickstart(context.Background(), []string{
		"--non-interactive",
		"--workspace-name", "test-ws",
		"--project-root", "/tmp",
		"--memory",
		"--mcp",
	})

	if err != nil {
		t.Fatalf("quickstart failed: %v", err)
	}

	// Verify init was called
	if !store.initCalled {
		t.Error("expected Init to be called")
	}

	// Verify workspace was added
	if store.addName != "test-ws" {
		t.Errorf("expected workspace name 'test-ws', got %q", store.addName)
	}

	if store.addRoot != "/tmp" {
		t.Errorf("expected project root '/tmp', got %q", store.addRoot)
	}

	// Verify output contains success messages
	output := app.stdout.(*bytes.Buffer).String()
	if !strings.Contains(output, "Initialized ADP home") {
		t.Error("output should contain initialization message")
	}
	if !strings.Contains(output, "created") {
		t.Error("output should contain creation message")
	}
	if !strings.Contains(output, "complete") {
		t.Error("output should contain completion message")
	}
}
