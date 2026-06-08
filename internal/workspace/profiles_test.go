package workspace

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/karoc/adp/internal/schema"
)

func TestListProfilesCollectsDefaultsConfigAndProfileFiles(t *testing.T) {
	workspaceDir := t.TempDir()
	profilesDir := filepath.Join(workspaceDir, "profiles")
	if err := os.Mkdir(profilesDir, 0o755); err != nil {
		t.Fatalf("create profiles dir: %v", err)
	}

	for _, name := range []string{"zeta.md", "alpha.yaml", "beta.yml", "gamma.json", "ignored.txt"} {
		if err := os.WriteFile(filepath.Join(profilesDir, name), []byte("profile\n"), 0o644); err != nil {
			t.Fatalf("write profile file %s: %v", name, err)
		}
	}
	if err := os.Mkdir(filepath.Join(profilesDir, "nested.md"), 0o755); err != nil {
		t.Fatalf("create profile-like directory: %v", err)
	}

	got, err := ListProfiles(context.Background(), workspaceDir, schema.Config{
		Agents: map[string]schema.AgentConfig{
			"codex":  {Profile: "senior"},
			"claude": {Profile: "architect"},
			"api":    {Profile: " "},
		},
	})
	if err != nil {
		t.Fatalf("ListProfiles() error = %v", err)
	}

	want := []string{"alpha", "architect", "beta", "default", "gamma", "senior", "zeta"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ListProfiles() = %#v, want %#v", got, want)
	}
}

func TestListProfilesAllowsMissingProfilesDir(t *testing.T) {
	got, err := ListProfiles(context.Background(), t.TempDir(), schema.Config{
		Agents: map[string]schema.AgentConfig{
			"codex": {Profile: "senior"},
		},
	})
	if err != nil {
		t.Fatalf("ListProfiles() error = %v", err)
	}

	want := []string{"default", "senior"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ListProfiles() = %#v, want %#v", got, want)
	}
}
