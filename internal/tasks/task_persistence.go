package tasks

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

func (s *Store) load(ctx context.Context) (taskFile, error) {
	if err := ctx.Err(); err != nil {
		return taskFile{}, err
	}
	path := s.tasksPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return taskFile{Version: currentVersion}, nil
		}
		return taskFile{}, fmt.Errorf("read task file %s: %w", path, err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return taskFile{Version: currentVersion}, nil
	}

	var file taskFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return taskFile{}, fmt.Errorf("parse task file %s: %w", path, err)
	}
	if file.Version == 0 {
		file.Version = currentVersion
	}
	if file.Version != currentVersion {
		return taskFile{}, fmt.Errorf("unsupported task file version %d", file.Version)
	}
	for _, task := range file.Tasks {
		if !isValidStatus(task.Status) {
			return taskFile{}, fmt.Errorf("task %s has unknown status %q", task.ID, task.Status)
		}
	}
	return file, nil
}

func (s *Store) save(ctx context.Context, file taskFile) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	file.Version = currentVersion
	sortTasks(file.Tasks)

	if err := os.MkdirAll(s.planningPath(), 0o755); err != nil {
		return fmt.Errorf("create planning directory: %w", err)
	}
	data, err := yaml.Marshal(file)
	if err != nil {
		return fmt.Errorf("encode task file: %w", err)
	}
	path := s.tasksPath()
	tmpPath := path + ".tmp"
	defer os.Remove(tmpPath)

	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write temporary task file %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace task file %s: %w", path, err)
	}
	return nil
}

func (s *Store) nextID(tasks []Task, now time.Time) string {
	prefix := "task-" + now.UTC().Format("20060102") + "-"
	maxSeq := 0
	for _, task := range tasks {
		if !strings.HasPrefix(task.ID, prefix) {
			continue
		}
		seqText := strings.TrimPrefix(task.ID, prefix)
		var seq int
		if _, err := fmt.Sscanf(seqText, "%d", &seq); err == nil && seq > maxSeq {
			maxSeq = seq
		}
	}
	return fmt.Sprintf("%s%04d", prefix, maxSeq+1)
}
