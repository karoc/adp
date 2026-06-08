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
	ordered := append([]Phase(nil), phases...)
	sortPhases(ordered)
	for _, phase := range ordered {
		if phase.ID == exceptID {
			continue
		}
		if phase.Status == PhaseStatusActive || phase.Status == PhaseStatusAccepted || phase.Status == PhaseStatusCommitted {
			return phase, true
		}
	}
	return Phase{}, false
}

func phaseStartBlocker(phases []Phase, targetID string) (Phase, bool) {
	if phase, ok := openPhase(phases, targetID); ok {
		return phase, true
	}
	ordered := append([]Phase(nil), phases...)
	sortPhases(ordered)
	for _, phase := range ordered {
		if phase.ID == targetID {
			return Phase{}, false
		}
		if !phaseGateSatisfied(phase) {
			return phase, true
		}
	}
	return Phase{}, false
}

func phaseGateSatisfied(phase Phase) bool {
	return phase.Status == PhaseStatusPushed && pushSucceeded(phase.Push.Result)
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
		if phases[i].Order != phases[j].Order {
			if phases[i].Order == 0 {
				return true
			}
			if phases[j].Order == 0 {
				return false
			}
			return phases[i].Order < phases[j].Order
		}
		if phases[i].CreatedAt.Equal(phases[j].CreatedAt) {
			return phases[i].ID < phases[j].ID
		}
		return phases[i].CreatedAt.Before(phases[j].CreatedAt)
	})
}

func nextPhaseOrder(phases []Phase) int {
	ordered := append([]Phase(nil), phases...)
	sortPhases(ordered)
	maxOrder := 0
	for i, phase := range ordered {
		order := phase.Order
		if order == 0 {
			order = i + 1
		}
		if order > maxOrder {
			maxOrder = order
		}
	}
	return maxOrder + 1
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
