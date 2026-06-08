package overlay

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/karoc/adp/internal/adapters"
)

const defaultGeneratedFileMode fs.FileMode = 0644

type SymlinkBackend struct{}

func NewSymlinkBackend() *SymlinkBackend {
	return &SymlinkBackend{}
}

func (b *SymlinkBackend) Materialize(ctx context.Context, req Request) (*Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if req.WorkspaceName == "" {
		return nil, fmt.Errorf("workspace name is required")
	}

	projectRoot, err := requireAbsoluteDir(req.ProjectRoot, "project root")
	if err != nil {
		return nil, err
	}
	runtimeRoot, err := requireAbsolutePath(req.RuntimeRoot, "runtime root")
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(runtimeRoot, 0755); err != nil {
		return nil, fmt.Errorf("create runtime root: %w", err)
	}

	reserved := map[string]struct{}{}
	for _, path := range req.ReservedPaths {
		clean, err := safeRelativePath(path)
		if err != nil {
			return nil, fmt.Errorf("reserved path %q: %w", path, err)
		}
		reserved[topLevelPath(clean)] = struct{}{}
	}

	generatedPaths, err := writeGeneratedFiles(runtimeRoot, req.Files, reserved)
	if err != nil {
		return nil, err
	}
	linkedPaths, conflicts, err := linkProjectChildren(projectRoot, runtimeRoot, reserved)
	if err != nil {
		return nil, err
	}

	return &Handle{
		Root:           runtimeRoot,
		WorkspaceName:  req.WorkspaceName,
		ProjectRoot:    projectRoot,
		GeneratedPaths: generatedPaths,
		LinkedPaths:    linkedPaths,
		Conflicts:      conflicts,
		Keep:           req.Keep,
	}, nil
}

func (b *SymlinkBackend) Cleanup(ctx context.Context, handle Handle) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if handle.Keep {
		return nil
	}
	if handle.Root == "" {
		return fmt.Errorf("runtime root is required")
	}
	root, err := requireAbsolutePath(handle.Root, "runtime root")
	if err != nil {
		return err
	}
	if filepath.Dir(root) == root {
		return fmt.Errorf("refusing to clean filesystem root %q", root)
	}
	return os.RemoveAll(root)
}

func writeGeneratedFiles(runtimeRoot string, files []adapters.GeneratedFile, reserved map[string]struct{}) ([]string, error) {
	generatedPaths := make([]string, 0, len(files))
	seen := map[string]struct{}{}

	for _, file := range files {
		clean, err := safeRelativePath(file.Path)
		if err != nil {
			return nil, fmt.Errorf("generated file %q: %w", file.Path, err)
		}
		if file.Mode.IsDir() {
			return nil, fmt.Errorf("generated file %q must not be a directory", file.Path)
		}
		if _, exists := seen[clean]; exists {
			return nil, fmt.Errorf("generated file %q is duplicated", clean)
		}
		seen[clean] = struct{}{}
		reserved[topLevelPath(clean)] = struct{}{}

		target := filepath.Join(runtimeRoot, clean)
		if err := ensureWithinRoot(runtimeRoot, target); err != nil {
			return nil, fmt.Errorf("generated file %q: %w", file.Path, err)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return nil, fmt.Errorf("create generated file parent %q: %w", clean, err)
		}
		if err := ensureNoSymlinkParents(runtimeRoot, filepath.Dir(target)); err != nil {
			return nil, fmt.Errorf("generated file %q: %w", clean, err)
		}

		mode := file.Mode.Perm()
		if mode == 0 {
			mode = defaultGeneratedFileMode
		}
		if err := writeNewFile(target, file.Data, mode); err != nil {
			return nil, fmt.Errorf("write generated file %q: %w", clean, err)
		}
		if err := os.Chmod(target, mode); err != nil {
			return nil, fmt.Errorf("chmod generated file %q: %w", clean, err)
		}
		generatedPaths = append(generatedPaths, clean)
	}

	return generatedPaths, nil
}

func linkProjectChildren(projectRoot, runtimeRoot string, reserved map[string]struct{}) ([]string, []Conflict, error) {
	entries, err := os.ReadDir(projectRoot)
	if err != nil {
		return nil, nil, fmt.Errorf("read project root: %w", err)
	}

	linkedPaths := make([]string, 0, len(entries))
	conflicts := []Conflict{}
	for _, entry := range entries {
		name := entry.Name()
		if _, exists := reserved[name]; exists {
			conflicts = append(conflicts, Conflict{
				Path:   name,
				Reason: "project path conflicts with an ADP generated or reserved runtime path",
			})
			continue
		}

		target := filepath.Join(runtimeRoot, name)
		if _, err := os.Lstat(target); err == nil {
			conflicts = append(conflicts, Conflict{
				Path:   name,
				Reason: "runtime path already exists",
			})
			continue
		} else if !os.IsNotExist(err) {
			return nil, nil, fmt.Errorf("inspect runtime path %q: %w", name, err)
		}

		source := filepath.Join(projectRoot, name)
		if err := os.Symlink(source, target); err != nil {
			return nil, nil, fmt.Errorf("link project path %q: %w", name, err)
		}
		linkedPaths = append(linkedPaths, name)
	}

	return linkedPaths, conflicts, nil
}

func requireAbsoluteDir(path, label string) (string, error) {
	clean, err := requireAbsolutePath(path, label)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(clean)
	if err != nil {
		return "", fmt.Errorf("%s %q: %w", label, clean, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s %q is not a directory", label, clean)
	}
	return clean, nil
}

func requireAbsolutePath(path, label string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("%s is required", label)
	}
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("%s must be absolute", label)
	}
	return filepath.Clean(path), nil
}

func safeRelativePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("absolute paths are not allowed")
	}
	if hasParentSegment(path) {
		return "", fmt.Errorf("parent directory traversal is not allowed")
	}

	clean := filepath.Clean(path)
	if clean == "." || clean == string(filepath.Separator) {
		return "", fmt.Errorf("path must name a file or directory")
	}
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("parent directory traversal is not allowed")
	}
	return clean, nil
}

func hasParentSegment(path string) bool {
	for _, part := range strings.FieldsFunc(path, func(r rune) bool {
		return r == '/' || r == '\\'
	}) {
		if part == ".." {
			return true
		}
	}
	return false
}

func topLevelPath(path string) string {
	if idx := strings.IndexRune(path, filepath.Separator); idx >= 0 {
		return path[:idx]
	}
	return path
}

func ensureWithinRoot(root, target string) error {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("path escapes runtime root")
	}
	return nil
}

func ensureNoSymlinkParents(root, dir string) error {
	rel, err := filepath.Rel(root, dir)
	if err != nil {
		return err
	}
	if rel == "." {
		return nil
	}

	current := root
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("parent path %q is a symlink", current)
		}
	}
	return nil
}

func writeNewFile(path string, data []byte, mode fs.FileMode) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(data); err != nil {
		return err
	}
	return nil
}
