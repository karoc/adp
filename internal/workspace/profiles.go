package workspace

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/karoc/adp/internal/schema"
)

var profileExtensions = []string{".md", ".yaml", ".yml", ".json"}

func ListProfiles(ctx context.Context, workspaceDir string, cfg schema.Config) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	seen := map[string]struct{}{"default": {}}
	for _, agent := range cfg.Agents {
		addProfile(seen, agent.Profile)
	}
	if err := addProfileFiles(ctx, seen, filepath.Join(workspaceDir, "profiles")); err != nil {
		return nil, err
	}

	profiles := make([]string, 0, len(seen))
	for profile := range seen {
		profiles = append(profiles, profile)
	}
	sort.Strings(profiles)
	return profiles, nil
}

func addProfileFiles(ctx context.Context, profiles map[string]struct{}, profilesDir string) error {
	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read profiles directory: %w", err)
	}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if !isProfileExtension(ext) {
			continue
		}
		addProfile(profiles, strings.TrimSuffix(name, filepath.Ext(name)))
	}
	return nil
}

func profileCandidatePaths(profile string) []string {
	candidates := make([]string, 0, len(profileExtensions))
	for _, ext := range profileExtensions {
		candidates = append(candidates, filepath.Join("profiles", profile+ext))
	}
	return candidates
}

func isProfileExtension(ext string) bool {
	for _, known := range profileExtensions {
		if ext == known {
			return true
		}
	}
	return false
}

func addProfile(profiles map[string]struct{}, profile string) {
	profile = strings.TrimSpace(profile)
	if profile == "" {
		return
	}
	profiles[profile] = struct{}{}
}
