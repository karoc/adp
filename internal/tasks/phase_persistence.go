package tasks

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

func (s *Store) loadPhases(ctx context.Context) (phaseFile, error) {
	if err := ctx.Err(); err != nil {
		return phaseFile{}, err
	}
	path := s.phasesPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return phaseFile{Version: currentVersion}, nil
		}
		return phaseFile{}, fmt.Errorf("read phase file %s: %w", path, err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return phaseFile{Version: currentVersion}, nil
	}
	var file phaseFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return phaseFile{}, fmt.Errorf("parse phase file %s: %w", path, err)
	}
	if file.Version == 0 {
		file.Version = currentVersion
	}
	if file.Version != currentVersion {
		return phaseFile{}, fmt.Errorf("unsupported phase file version %d", file.Version)
	}
	for _, phase := range file.Phases {
		if !isValidPhaseStatus(phase.Status) {
			return phaseFile{}, fmt.Errorf("phase %s has unknown status %q", phase.ID, phase.Status)
		}
	}
	return file, nil
}

func (s *Store) savePhases(ctx context.Context, file phaseFile) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	file.Version = currentVersion
	sortPhases(file.Phases)
	if err := os.MkdirAll(s.planningPath(), 0o755); err != nil {
		return fmt.Errorf("create planning directory: %w", err)
	}
	data, err := yaml.Marshal(file)
	if err != nil {
		return fmt.Errorf("encode phase file: %w", err)
	}
	path := s.phasesPath()
	tmpPath := path + ".tmp"
	defer os.Remove(tmpPath)
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write temporary phase file %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace phase file %s: %w", path, err)
	}
	return nil
}
