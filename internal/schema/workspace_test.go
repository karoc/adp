package schema

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/karoc/adp/internal/paths"
)

// validConfig returns a minimal Config that passes Validate, so individual
// tests can mutate one field to exercise a single validation branch.
func validConfig() Config {
	return Config{
		Version:   CurrentVersion,
		Workspace: Workspace{Name: "demo"},
		Project:   Project{Root: "/abs/project"},
	}
}

func TestValidateAcceptsMinimalConfig(t *testing.T) {
	if err := validConfig().Validate(); err != nil {
		t.Fatalf("Validate() on valid config: %v", err)
	}
}

func TestValidateRejectsWrongVersion(t *testing.T) {
	cfg := validConfig()
	cfg.Version = CurrentVersion + 1
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for unsupported version, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported workspace schema version") {
		t.Fatalf("error = %q, want it to mention unsupported version", err)
	}
}

func TestValidateRejectsBadWorkspaceName(t *testing.T) {
	cfg := validConfig()
	cfg.Workspace.Name = "bad name" // space is not allowed by the pattern
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for invalid workspace name, got nil")
	}
}

func TestValidateRequiresProjectRoot(t *testing.T) {
	cfg := validConfig()
	cfg.Project.Root = ""
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for empty project root, got nil")
	}
	if !strings.Contains(err.Error(), "project root is required") {
		t.Fatalf("error = %q, want it to mention required project root", err)
	}
}

func TestValidateRequiresAbsoluteProjectRoot(t *testing.T) {
	cfg := validConfig()
	cfg.Project.Root = "relative/path"
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for relative project root, got nil")
	}
	if !strings.Contains(err.Error(), "must be absolute") {
		t.Fatalf("error = %q, want it to mention absolute path", err)
	}
}

func TestValidateWorkspaceName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"simple", "demo", false},
		{"with digits", "web2", false},
		{"leading digit", "2web", false},
		{"dots dashes underscores", "a.b-c_d", false},
		{"empty", "", true},
		{"leading dot", ".hidden", true},
		{"leading dash", "-flag", true},
		{"space", "has space", true},
		{"slash", "a/b", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWorkspaceName(tt.input)
			if tt.wantErr && err == nil {
				t.Fatalf("ValidateWorkspaceName(%q) = nil, want error", tt.input)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("ValidateWorkspaceName(%q) = %v, want nil", tt.input, err)
			}
		})
	}
}

func TestSaveConfigThenLoadConfigRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workspace.yaml")
	want := validConfig()
	want.Memory = Memory{Enabled: true, Shared: "team"}
	want.MCP = MCP{Enabled: true, Servers: []string{"fs", "git"}}

	if err := SaveConfig(path, &want); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	got, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if got.Workspace.Name != want.Workspace.Name || got.Project.Root != want.Project.Root {
		t.Fatalf("round-trip mismatch: got %+v, want %+v", got, want)
	}
	if !got.Memory.Enabled || got.Memory.Shared != "team" {
		t.Fatalf("memory not round-tripped: %+v", got.Memory)
	}
	if len(got.MCP.Servers) != 2 {
		t.Fatalf("mcp servers not round-tripped: %+v", got.MCP)
	}
}

func TestSaveConfigWritesOwnerOnlyPermission(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workspace.yaml")
	cfg := validConfig()
	if err := SaveConfig(path, &cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if got := info.Mode().Perm(); got != paths.PrivateFileMode {
		t.Fatalf("permission = %o, want %o", got, paths.PrivateFileMode)
	}
}

func TestSaveConfigRejectsNil(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workspace.yaml")
	if err := SaveConfig(path, nil); err == nil {
		t.Fatal("expected error for nil config, got nil")
	}
}

func TestSaveConfigRejectsInvalidConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workspace.yaml")
	cfg := validConfig()
	cfg.Project.Root = "" // invalid: fails Validate before writing
	if err := SaveConfig(path, &cfg); err == nil {
		t.Fatal("expected error for invalid config, got nil")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("invalid config should not have written a file")
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	_, err := LoadConfig(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
	if !strings.Contains(err.Error(), "read workspace config") {
		t.Fatalf("error = %q, want it to mention read failure", err)
	}
}

func TestLoadConfigMalformedYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workspace.yaml")
	if err := os.WriteFile(path, []byte("version: [not-an-int\n"), 0o600); err != nil {
		t.Fatalf("seed file: %v", err)
	}
	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for malformed yaml, got nil")
	}
	if !strings.Contains(err.Error(), "decode workspace config") {
		t.Fatalf("error = %q, want it to mention decode failure", err)
	}
}

func TestLoadConfigValidButFailsValidation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workspace.yaml")
	// Well-formed YAML but semantically invalid (version 0, no project root).
	if err := os.WriteFile(path, []byte("version: 0\n"), 0o600); err != nil {
		t.Fatalf("seed file: %v", err)
	}
	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported workspace schema version") {
		t.Fatalf("error = %q, want validation failure", err)
	}
}
