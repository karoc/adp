package runtime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/karoc/adp/internal/paths"
	"gopkg.in/yaml.v3"
)

// PruneRequest controls runtime pruning scope and safety behavior.
type PruneRequest struct {
	Layout      paths.Layout
	OlderThan   time.Duration
	Now         time.Time
	IncludeKept bool
	DryRun      bool
}

// PruneResult describes a stale runtime selected by Prune.
type PruneResult struct {
	Root      string
	Workspace string
	SessionID string
	CreatedAt time.Time
	Keep      bool
	Removed   bool
	DryRun    bool
}

// Prune scans the configured runtime parent and removes stale ADP-owned runtimes.
func Prune(ctx context.Context, req PruneRequest) ([]PruneResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if req.Layout.RuntimeParent == "" {
		return nil, fmt.Errorf("runtime parent is required")
	}
	if req.OlderThan < 0 {
		return nil, fmt.Errorf("older than must not be negative")
	}

	runtimeParent, err := filepath.Abs(req.Layout.RuntimeParent)
	if err != nil {
		return nil, fmt.Errorf("resolve runtime parent: %w", err)
	}
	if filepath.Dir(runtimeParent) == runtimeParent {
		return nil, fmt.Errorf("refusing to prune filesystem root runtime parent %q", runtimeParent)
	}

	entries, err := os.ReadDir(runtimeParent)
	if os.IsNotExist(err) {
		return []PruneResult{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read runtime parent: %w", err)
	}

	now := req.Now
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	cutoff := now.Add(-req.OlderThan)

	results := []PruneResult{}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return results, err
		}
		if !entry.IsDir() {
			continue
		}

		root := filepath.Join(runtimeParent, entry.Name())
		manifest, owned, err := readOwnedManifest(root)
		if err != nil {
			return results, err
		}
		if !owned || !manifest.CreatedAt.Before(cutoff) {
			continue
		}
		if manifest.Keep && !req.IncludeKept {
			continue
		}
		if isManifestProjectRoot(root, manifest) {
			continue
		}

		result := PruneResult{
			Root:      root,
			Workspace: manifest.Workspace,
			SessionID: manifest.SessionID,
			CreatedAt: manifest.CreatedAt.UTC(),
			Keep:      manifest.Keep,
			DryRun:    req.DryRun,
		}
		if !req.DryRun {
			if err := ctx.Err(); err != nil {
				return results, err
			}
			if err := os.RemoveAll(root); err != nil {
				return results, fmt.Errorf("remove runtime %q: %w", root, err)
			}
			result.Removed = true
		}
		results = append(results, result)
	}

	return results, nil
}

func readOwnedManifest(root string) (Manifest, bool, error) {
	data, err := os.ReadFile(filepath.Join(root, ManifestPath))
	if os.IsNotExist(err) {
		return Manifest{}, false, nil
	}
	if err != nil {
		return Manifest{}, false, fmt.Errorf("read runtime manifest %q: %w", root, err)
	}

	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, false, nil
	}
	manifest, ok := normalizeOwnedManifest(root, manifest)
	if !ok {
		return Manifest{}, false, nil
	}
	return manifest, true, nil
}

func isManifestProjectRoot(root string, manifest Manifest) bool {
	if manifest.ProjectRoot == "" {
		return false
	}
	projectRoot, err := filepath.Abs(manifest.ProjectRoot)
	if err != nil {
		return false
	}
	return filepath.Clean(root) == filepath.Clean(projectRoot)
}
