package tasks

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

func findPhase(phases []Phase, id string) (Phase, bool) {
	for _, phase := range phases {
		if phase.ID == id {
			return phase, true
		}
	}
	return Phase{}, false
}

func openPhase(phases []Phase, exceptID string) (Phase, bool) {
	for _, phase := range phases {
		if phase.ID == exceptID {
			continue
		}
		if phase.Status == PhaseStatusActive || phase.Status == PhaseStatusAccepted || phase.Status == PhaseStatusCommitted {
			return phase, true
		}
	}
	return Phase{}, false
}

func isValidPhaseStatus(status PhaseStatus) bool {
	for _, known := range phaseStatuses {
		if status == known {
			return true
		}
	}
	return false
}

func sortPhases(phases []Phase) {
	sort.SliceStable(phases, func(i, j int) bool {
		if phases[i].CreatedAt.Equal(phases[j].CreatedAt) {
			return phases[i].ID < phases[j].ID
		}
		return phases[i].CreatedAt.Before(phases[j].CreatedAt)
	})
}

func cleanStrings(values []string) []string {
	cleaned := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			cleaned = append(cleaned, value)
		}
	}
	return cleaned
}

func (s *Store) validateTaskPhase(ctx context.Context, phase string) error {
	phase = strings.TrimSpace(phase)
	if phase == "" || phase == defaultTaskPhase {
		return nil
	}
	file, err := s.loadPhases(ctx)
	if err != nil {
		return err
	}
	if len(file.Phases) == 0 {
		return nil
	}
	if _, ok := findPhase(file.Phases, phase); !ok {
		return fmt.Errorf("%w: task phase %s", ErrPhaseNotFound, phase)
	}
	return nil
}

func pushSucceeded(result string) bool {
	switch strings.ToLower(strings.TrimSpace(result)) {
	case "pushed", "passed", "success", "succeeded":
		return true
	default:
		return false
	}
}
