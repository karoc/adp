package tasks

import (
	"path/filepath"
	"time"

	"github.com/karoc/adp/internal/paths"
)

const (
	planningDir      = "planning"
	tasksFile        = "tasks.yaml"
	phasesFile       = "phases.yaml"
	progressFile     = "progress.jsonl"
	currentVersion   = 1
	defaultNextLimit = 5
	defaultPriority  = "normal"
	defaultTaskPhase = "unassigned"
)

type Store struct {
	WorkspaceDir string
	Now          func() time.Time
}

func NewStore(workspaceDir string) *Store {
	return &Store{WorkspaceDir: workspaceDir, Now: time.Now}
}

func (s *Store) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC().Truncate(time.Second)
	}
	return time.Now().UTC().Truncate(time.Second)
}

func (s *Store) planningPath() string {
	return filepath.Join(s.WorkspaceDir, planningDir)
}

// ensurePlanningDir creates the planning directory with owner-only
// permissions (0o700). Planning data can contain project paths, task
// content, and command history that must not be exposed to other local
// users (CWE-732).
func (s *Store) ensurePlanningDir() error {
	return paths.EnsurePrivateDir(s.planningPath())
}

func (s *Store) tasksPath() string {
	return filepath.Join(s.planningPath(), tasksFile)
}

func (s *Store) phasesPath() string {
	return filepath.Join(s.planningPath(), phasesFile)
}

func (s *Store) progressPath() string {
	return filepath.Join(s.planningPath(), progressFile)
}

func (s *Store) lockPath() string {
	return filepath.Join(s.planningPath(), planningLockFile)
}
