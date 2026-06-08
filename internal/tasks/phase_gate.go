package tasks

import "fmt"

const (
	PhaseGateActionRecordAcceptance = "record_acceptance"
	PhaseGateActionRecordCommit     = "record_commit"
	PhaseGateActionRecordPush       = "record_push"
	PhaseGateActionStartNextPhase   = "start_next_phase"
	PhaseGateActionPlanNextPhase    = "plan_next_phase"
)

type PhaseGate struct {
	PhaseCount       int
	OpenPhase        *Phase
	NextPlannedPhase *Phase
	CanStartNext     bool
	NextAction       string
	Reason           string
}

func PhaseGateStatus(phases []Phase) PhaseGate {
	ordered := append([]Phase(nil), phases...)
	sortPhases(ordered)

	gate := PhaseGate{PhaseCount: len(ordered)}
	var firstUnsatisfied *Phase
	var firstOpen *Phase
	for _, phase := range ordered {
		if gate.NextPlannedPhase == nil && phase.Status == PhaseStatusPlanned {
			gate.NextPlannedPhase = phaseCopy(phase)
		}
		if firstOpen == nil && isOpenPhaseStatus(phase.Status) {
			firstOpen = phaseCopy(phase)
		}
		if firstUnsatisfied == nil && !phaseGateSatisfied(phase) {
			firstUnsatisfied = phaseCopy(phase)
		}
	}

	if firstUnsatisfied == nil {
		gate.NextAction = PhaseGateActionPlanNextPhase
		gate.Reason = "no planned phase remains"
		return gate
	}
	if firstUnsatisfied.Status == PhaseStatusPlanned {
		if firstOpen != nil {
			gate.OpenPhase = firstOpen
			gate.NextAction, gate.Reason = blockedPhaseGateAction(*firstOpen)
			return gate
		}
		gate.CanStartNext = true
		gate.NextAction = PhaseGateActionStartNextPhase
		gate.Reason = fmt.Sprintf("phase %s is the next planned phase and can be started", firstUnsatisfied.ID)
		return gate
	}

	gate.OpenPhase = firstUnsatisfied
	if gate.OpenPhase != nil {
		gate.CanStartNext = false
		gate.NextAction, gate.Reason = blockedPhaseGateAction(*gate.OpenPhase)
		return gate
	}
	return gate
}

func isOpenPhaseStatus(status PhaseStatus) bool {
	return status == PhaseStatusActive || status == PhaseStatusAccepted || status == PhaseStatusCommitted
}

func blockedPhaseGateAction(phase Phase) (string, string) {
	switch phase.Status {
	case PhaseStatusActive:
		return PhaseGateActionRecordAcceptance, fmt.Sprintf("phase %s is active; record acceptance before commit evidence", phase.ID)
	case PhaseStatusAccepted:
		return PhaseGateActionRecordCommit, fmt.Sprintf("phase %s is accepted; record commit evidence before push evidence", phase.ID)
	case PhaseStatusCommitted:
		return PhaseGateActionRecordPush, fmt.Sprintf("phase %s is committed; record push evidence before starting another phase", phase.ID)
	case PhaseStatusPushed:
		return PhaseGateActionRecordPush, fmt.Sprintf("phase %s is pushed but successful push evidence is missing", phase.ID)
	default:
		return PhaseGateActionPlanNextPhase, fmt.Sprintf("phase %s has status %s", phase.ID, phase.Status)
	}
}

func phaseCopy(phase Phase) *Phase {
	copied := phase
	return &copied
}
