package paths

import (
	"fmt"
	"os"
)

// PrivateDirMode is the permission ADP uses for directories under $ADP_HOME.
// The tree records project paths, task content, git remotes, and command
// history, so it is kept owner-only to avoid disclosing that context to other
// local users on shared hosts.
const PrivateDirMode os.FileMode = 0o700

// PrivateFileMode is the permission ADP uses for state files under $ADP_HOME
// that may carry sensitive recorded context.
const PrivateFileMode os.FileMode = 0o600

// EnsurePrivateDir creates dir (and any missing parents) with owner-only
// permission. When the directory already exists with a looser mode — for
// example one created by an older ADP build or under a permissive umask — its
// permission is tightened to PrivateDirMode. MkdirAll alone does not adjust the
// mode of an existing directory, so the explicit Chmod is what closes the gap
// for upgrades.
func EnsurePrivateDir(dir string) error {
	if dir == "" {
		return fmt.Errorf("directory path is required")
	}
	if err := os.MkdirAll(dir, PrivateDirMode); err != nil {
		return err
	}
	if info, err := os.Stat(dir); err == nil && info.IsDir() && info.Mode().Perm() != PrivateDirMode {
		// Best-effort tightening; a failure here should not block the caller
		// because the directory is still usable, only less private than ideal.
		_ = os.Chmod(dir, PrivateDirMode)
	}
	return nil
}
