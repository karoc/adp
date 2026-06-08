package tasks

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

type PlanningDiagnosticLevel string

const (
	PlanningDiagnosticLevelInfo    PlanningDiagnosticLevel = "info"
	PlanningDiagnosticLevelWarning PlanningDiagnosticLevel = "warning"
	PlanningDiagnosticLevelError   PlanningDiagnosticLevel = "error"
)

const (
	PlanningDiagnosticCodeTaskFileInvalid      = "planning.tasks.file.invalid"
	PlanningDiagnosticCodeTaskIDMissing        = "planning.task.id.missing"
	PlanningDiagnosticCodeTaskIDDuplicate      = "planning.task.id.duplicate"
	PlanningDiagnosticCodeTaskPhaseUnknown     = "planning.task.phase.unknown"
	PlanningDiagnosticCodePhaseFileInvalid     = "planning.phases.file.invalid"
	PlanningDiagnosticCodePhaseIDMissing       = "planning.phase.id.missing"
	PlanningDiagnosticCodePhaseIDDuplicate     = "planning.phase.id.duplicate"
	PlanningDiagnosticCodePhaseOrderDuplicate  = "planning.phase.order.duplicate"
	PlanningDiagnosticCodePhaseMultipleOpen    = "planning.phase.open.multiple"
	PlanningDiagnosticCodePhaseGateSkipped     = "planning.phase.gate.skipped"
	PlanningDiagnosticCodePhaseEvidenceMissing = "planning.phase.evidence.missing"
	PlanningDiagnosticCodeLockStale            = "planning.lock.stale"
	PlanningDiagnosticCodeLockStatFailed       = "planning.lock.stat_failed"
	PlanningDiagnosticCodeProgressReadFailed   = "planning.progress.read_failed"
	PlanningDiagnosticCodeProgressInvalidJSON  = "planning.progress.invalid_json"
	PlanningDiagnosticCodeProgressTypeMissing  = "planning.progress.type.missing"
	PlanningDiagnosticCodeProgressTaskUnknown  = "planning.progress.task.unknown"
	PlanningDiagnosticCodeProgressPhaseUnknown = "planning.progress.phase.unknown"
)

type PlanningDiagnostic struct {
	Level   PlanningDiagnosticLevel
	Code    string
	Message string
	Path    string
	Line    int
}

type PlanningDiagnosticReport struct {
	WorkspaceDir       string
	PlanningDir        string
	TasksPath          string
	PhasesPath         string
	ProgressPath       string
	TaskCount          int
	PhaseCount         int
	ProgressEventCount int
	PhaseGate          PhaseGate
	Diagnostics        []PlanningDiagnostic
}

func (r PlanningDiagnosticReport) HasErrors() bool {
	for _, diagnostic := range r.Diagnostics {
		if diagnostic.Level == PlanningDiagnosticLevelError {
			return true
		}
	}
	return false
}

func (r PlanningDiagnosticReport) ErrorCount() int {
	return r.countLevel(PlanningDiagnosticLevelError)
}

func (r PlanningDiagnosticReport) WarningCount() int {
	return r.countLevel(PlanningDiagnosticLevelWarning)
}

func (s *Store) DiagnosePlanning(ctx context.Context) (PlanningDiagnosticReport, error) {
	if err := ctx.Err(); err != nil {
		return PlanningDiagnosticReport{}, err
	}
	report := PlanningDiagnosticReport{
		WorkspaceDir: s.WorkspaceDir,
		PlanningDir:  s.planningPath(),
		TasksPath:    s.tasksPath(),
		PhasesPath:   s.phasesPath(),
		ProgressPath: s.progressPath(),
	}

	taskData, tasksOK := s.diagnoseLoadTasks(ctx, &report)
	phaseData, phasesOK := s.diagnoseLoadPhases(ctx, &report)
	phaseIDs := map[string]struct{}{}
	if phasesOK {
		report.PhaseCount = len(phaseData.Phases)
		report.PhaseGate = PhaseGateStatus(phaseData.Phases)
		phaseIDs = diagnosePhaseRecords(&report, s.phasesPath(), phaseData.Phases)
	}
	taskIDs := map[string]struct{}{}
	if tasksOK {
		report.TaskCount = len(taskData.Tasks)
		taskIDs = diagnoseTaskRecords(&report, s.tasksPath(), taskData.Tasks, phaseIDs, phasesOK && len(phaseIDs) > 0)
	}
	s.diagnosePlanningLock(&report)
	report.ProgressEventCount = s.diagnoseProgressLog(&report, taskIDs, tasksOK, phaseIDs, phasesOK)
	return report, nil
}

func (s *Store) diagnoseLoadTasks(ctx context.Context, report *PlanningDiagnosticReport) (taskFile, bool) {
	data, err := s.load(ctx)
	if err == nil {
		return data, true
	}
	report.add(PlanningDiagnosticLevelError, PlanningDiagnosticCodeTaskFileInvalid, fmt.Sprintf("task ledger could not be loaded: %v", err), s.tasksPath(), 0)
	return taskFile{}, false
}

func (s *Store) diagnoseLoadPhases(ctx context.Context, report *PlanningDiagnosticReport) (phaseFile, bool) {
	data, err := s.loadPhases(ctx)
	if err == nil {
		return data, true
	}
	report.add(PlanningDiagnosticLevelError, PlanningDiagnosticCodePhaseFileInvalid, fmt.Sprintf("phase ledger could not be loaded: %v", err), s.phasesPath(), 0)
	return phaseFile{}, false
}

func diagnoseTaskRecords(report *PlanningDiagnosticReport, path string, tasks []Task, phaseIDs map[string]struct{}, validatePhases bool) map[string]struct{} {
	taskIDs := map[string]struct{}{}
	for _, task := range tasks {
		if task.ID == "" {
			report.add(PlanningDiagnosticLevelError, PlanningDiagnosticCodeTaskIDMissing, fmt.Sprintf("task %q has an empty id", task.Title), path, 0)
			continue
		}
		if _, ok := taskIDs[task.ID]; ok {
			report.add(PlanningDiagnosticLevelError, PlanningDiagnosticCodeTaskIDDuplicate, fmt.Sprintf("task id %s appears more than once", task.ID), path, 0)
		}
		taskIDs[task.ID] = struct{}{}
		if validatePhases && task.Phase != "" && task.Phase != defaultTaskPhase {
			if _, ok := phaseIDs[task.Phase]; !ok {
				report.add(PlanningDiagnosticLevelError, PlanningDiagnosticCodeTaskPhaseUnknown, fmt.Sprintf("task %s references unknown phase %s", task.ID, task.Phase), path, 0)
			}
		}
	}
	return taskIDs
}

func diagnosePhaseRecords(report *PlanningDiagnosticReport, path string, phases []Phase) map[string]struct{} {
	phaseIDs := map[string]struct{}{}
	orderIDs := map[int][]string{}
	openIDs := []string{}
	for _, phase := range phases {
		if phase.ID == "" {
			report.add(PlanningDiagnosticLevelError, PlanningDiagnosticCodePhaseIDMissing, fmt.Sprintf("phase %q has an empty id", phase.Title), path, 0)
			continue
		}
		if _, ok := phaseIDs[phase.ID]; ok {
			report.add(PlanningDiagnosticLevelError, PlanningDiagnosticCodePhaseIDDuplicate, fmt.Sprintf("phase id %s appears more than once", phase.ID), path, 0)
		}
		phaseIDs[phase.ID] = struct{}{}
		if phase.Order > 0 {
			orderIDs[phase.Order] = append(orderIDs[phase.Order], phase.ID)
		}
		if isOpenPhaseStatus(phase.Status) {
			openIDs = append(openIDs, phase.ID)
		}
		diagnosePhaseEvidence(report, path, phase)
	}
	for order, ids := range orderIDs {
		if len(ids) > 1 {
			report.add(PlanningDiagnosticLevelError, PlanningDiagnosticCodePhaseOrderDuplicate, fmt.Sprintf("phase order %d is shared by %s", order, strings.Join(ids, ", ")), path, 0)
		}
	}
	if len(openIDs) > 1 {
		report.add(PlanningDiagnosticLevelError, PlanningDiagnosticCodePhaseMultipleOpen, fmt.Sprintf("multiple open phases: %s", strings.Join(openIDs, ", ")), path, 0)
	}
	diagnoseSkippedPhaseGates(report, path, phases)
	return phaseIDs
}

func diagnosePhaseEvidence(report *PlanningDiagnosticReport, path string, phase Phase) {
	if phase.Status == PhaseStatusAccepted || phase.Status == PhaseStatusCommitted || phase.Status == PhaseStatusPushed {
		if phase.Acceptance.Result != "passed" {
			report.add(PlanningDiagnosticLevelError, PlanningDiagnosticCodePhaseEvidenceMissing, fmt.Sprintf("phase %s has status %s without passed acceptance evidence", phase.ID, phase.Status), path, 0)
		}
	}
	if phase.Status == PhaseStatusCommitted || phase.Status == PhaseStatusPushed {
		if strings.TrimSpace(phase.Commit.Hash) == "" {
			report.add(PlanningDiagnosticLevelError, PlanningDiagnosticCodePhaseEvidenceMissing, fmt.Sprintf("phase %s has status %s without commit evidence", phase.ID, phase.Status), path, 0)
		}
	}
	if phase.Status == PhaseStatusPushed && !pushSucceeded(phase.Push.Result) {
		report.add(PlanningDiagnosticLevelError, PlanningDiagnosticCodePhaseEvidenceMissing, fmt.Sprintf("phase %s is pushed without successful push evidence", phase.ID), path, 0)
	}
}

func diagnoseSkippedPhaseGates(report *PlanningDiagnosticReport, path string, phases []Phase) {
	ordered := append([]Phase(nil), phases...)
	sortPhases(ordered)
	var blocker *Phase
	for _, phase := range ordered {
		if blocker != nil && phase.ID != blocker.ID && phase.Status != PhaseStatusPlanned {
			report.add(PlanningDiagnosticLevelError, PlanningDiagnosticCodePhaseGateSkipped, fmt.Sprintf("phase %s [%s] appears after unsatisfied phase %s [%s]", phase.ID, phase.Status, blocker.ID, blocker.Status), path, 0)
		}
		if blocker == nil && !phaseGateSatisfied(phase) {
			copied := phase
			blocker = &copied
		}
	}
}

func (s *Store) diagnosePlanningLock(report *PlanningDiagnosticReport) {
	info, err := os.Stat(s.lockPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		report.add(PlanningDiagnosticLevelError, PlanningDiagnosticCodeLockStatFailed, fmt.Sprintf("planning lock could not be inspected: %v", err), s.lockPath(), 0)
		return
	}
	if !info.Mode().IsRegular() {
		report.add(PlanningDiagnosticLevelError, PlanningDiagnosticCodeLockStatFailed, "planning lock is not a regular file and may block mutating planning commands", s.lockPath(), 0)
		return
	}
	if s.now().Sub(info.ModTime()) > planningLockStaleAge {
		report.add(PlanningDiagnosticLevelWarning, PlanningDiagnosticCodeLockStale, "planning lock is stale and will be removed by the next mutating planning command", s.lockPath(), 0)
	}
}

func (s *Store) diagnoseProgressLog(report *PlanningDiagnosticReport, taskIDs map[string]struct{}, tasksOK bool, phaseIDs map[string]struct{}, phasesOK bool) int {
	file, err := os.Open(s.progressPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0
		}
		report.add(PlanningDiagnosticLevelError, PlanningDiagnosticCodeProgressReadFailed, fmt.Sprintf("progress log could not be read: %v", err), s.progressPath(), 0)
		return 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	count := 0
	for line := 1; scanner.Scan(); line++ {
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}
		count++
		var event progressEvent
		if err := json.Unmarshal([]byte(text), &event); err != nil {
			report.add(PlanningDiagnosticLevelError, PlanningDiagnosticCodeProgressInvalidJSON, fmt.Sprintf("progress event line %d is not valid JSON: %v", line, err), s.progressPath(), line)
			continue
		}
		if event.Type == "" {
			report.add(PlanningDiagnosticLevelWarning, PlanningDiagnosticCodeProgressTypeMissing, fmt.Sprintf("progress event line %d has no type", line), s.progressPath(), line)
		}
		if tasksOK && event.TaskID != "" {
			if _, ok := taskIDs[event.TaskID]; !ok {
				report.add(PlanningDiagnosticLevelWarning, PlanningDiagnosticCodeProgressTaskUnknown, fmt.Sprintf("progress event line %d references unknown task %s", line, event.TaskID), s.progressPath(), line)
			}
		}
		if phasesOK && event.PhaseID != "" {
			if _, ok := phaseIDs[event.PhaseID]; !ok {
				report.add(PlanningDiagnosticLevelWarning, PlanningDiagnosticCodeProgressPhaseUnknown, fmt.Sprintf("progress event line %d references unknown phase %s", line, event.PhaseID), s.progressPath(), line)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		report.add(PlanningDiagnosticLevelError, PlanningDiagnosticCodeProgressReadFailed, fmt.Sprintf("progress log could not be scanned: %v", err), s.progressPath(), 0)
	}
	return count
}

func (r *PlanningDiagnosticReport) add(level PlanningDiagnosticLevel, code string, message string, path string, line int) {
	r.Diagnostics = append(r.Diagnostics, PlanningDiagnostic{
		Level:   level,
		Code:    code,
		Message: message,
		Path:    path,
		Line:    line,
	})
}

func (r PlanningDiagnosticReport) countLevel(level PlanningDiagnosticLevel) int {
	count := 0
	for _, diagnostic := range r.Diagnostics {
		if diagnostic.Level == level {
			count++
		}
	}
	return count
}
