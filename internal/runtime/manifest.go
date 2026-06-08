package runtime

import (
	"path/filepath"
	"strings"
)

func normalizeOwnedManifest(root string, manifest Manifest) (Manifest, bool) {
	if manifest.GeneratedBy != ManifestGeneratedBy {
		return Manifest{}, false
	}
	if manifest.Version != ManifestVersion {
		return Manifest{}, false
	}
	if strings.TrimSpace(manifest.SessionID) == "" || strings.TrimSpace(manifest.Workspace) == "" {
		return Manifest{}, false
	}
	if !filepath.IsAbs(manifest.ProjectRoot) || !sameAbsPath(root, manifest.RuntimeRoot) {
		return Manifest{}, false
	}
	if manifest.CreatedAt.IsZero() {
		return Manifest{}, false
	}
	manifest.CreatedAt = manifest.CreatedAt.UTC()
	return manifest, true
}

func sameAbsPath(left string, right string) bool {
	if right == "" {
		return false
	}
	leftAbs, err := filepath.Abs(left)
	if err != nil {
		return false
	}
	rightAbs, err := filepath.Abs(right)
	if err != nil {
		return false
	}
	return filepath.Clean(leftAbs) == filepath.Clean(rightAbs)
}
