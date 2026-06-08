package workspace

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/karoc/adp/internal/schema"
)

func checkAgents(ctx context.Context, report *DiagnosticReport, projectRoot string, agents map[string]schema.AgentConfig) error {
	names := make([]string, 0, len(agents))
	for name := range agents {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return err
		}

		agent := agents[name]
		if !agent.Enabled {
			continue
		}
		if !agentHasDefaultCommand(name) {
			report.add(DiagnosticLevelWarning, DiagnosticCodeAgentUnknown, fmt.Sprintf("enabled agent %q has no registered adapter", name), report.ConfigPath)
		}
		command := strings.TrimSpace(agent.Command)
		if command == "" {
			checkDefaultAgentCommand(report, name)
		} else {
			checkConfiguredAgentCommand(report, projectRoot, name, command)
		}
		profile := strings.TrimSpace(agent.Profile)
		if profile != "" && profile != "default" {
			checkProfileFile(report, name, profile)
		}
	}
	return nil
}

func agentHasDefaultCommand(name string) bool {
	switch name {
	case "codex", "claude":
		return true
	default:
		return false
	}
}

func checkDefaultAgentCommand(report *DiagnosticReport, name string) {
	level := DiagnosticLevelWarning
	message := fmt.Sprintf("enabled agent %q has no command override; configure a command unless an adapter default is available", name)
	if agentHasDefaultCommand(name) {
		level = DiagnosticLevelInfo
		message = fmt.Sprintf("enabled agent %q has no command override; adapter default command will be used", name)
	}
	report.add(level, DiagnosticCodeAgentCommandDefault, message, report.ConfigPath)
}

func checkConfiguredAgentCommand(report *DiagnosticReport, projectRoot string, agentName string, command string) {
	if commandLikelyContainsArguments(command) && !configuredCommandPathExists(projectRoot, command) {
		report.add(DiagnosticLevelWarning, DiagnosticCodeAgentCommandArguments, fmt.Sprintf("agent %q command looks like it contains arguments or shell syntax; configure only the executable path and pass runtime arguments after --", agentName), report.ConfigPath)
		return
	}

	checkAgentCommandPath(report, projectRoot, agentName, command)
}

func commandLikelyContainsArguments(command string) bool {
	return strings.ContainsAny(command, " \t\r\n\"'`;|&<>$()")
}

func configuredCommandPathExists(projectRoot string, command string) bool {
	path, ok := commandPathForInspection(projectRoot, command)
	if !ok {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func checkAgentCommandPath(report *DiagnosticReport, projectRoot string, agentName string, command string) {
	path, ok := commandPathForInspection(projectRoot, command)
	if !ok {
		return
	}

	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			report.add(DiagnosticLevelWarning, DiagnosticCodeAgentCommandMissing, fmt.Sprintf("configured command for agent %q does not exist", agentName), path)
			return
		}
		report.add(DiagnosticLevelWarning, DiagnosticCodeAgentCommandStatFailed, fmt.Sprintf("configured command for agent %q could not be inspected: %v", agentName, err), path)
		return
	}
	if info.IsDir() || info.Mode().Perm()&0o111 == 0 {
		report.add(DiagnosticLevelWarning, DiagnosticCodeAgentCommandNotExecutable, fmt.Sprintf("configured command for agent %q is not an executable file", agentName), path)
	}
}

func commandPathForInspection(projectRoot string, command string) (string, bool) {
	if filepath.IsAbs(command) {
		return filepath.Clean(command), true
	}
	if strings.ContainsRune(command, os.PathSeparator) {
		return filepath.Join(projectRoot, filepath.Clean(command)), true
	}
	return "", false
}

func checkProfileFile(report *DiagnosticReport, agentName string, profile string) {
	if !validProfileName(profile) {
		report.add(DiagnosticLevelWarning, DiagnosticCodeAgentProfileInvalid, fmt.Sprintf("profile %q for agent %q must be a file basename under profiles/", profile, agentName), report.ConfigPath)
		return
	}

	candidates := profileCandidatePaths(profile)
	existingFiles := []string{}
	notFilePaths := []string{}
	var outsidePath string
	var statFailedPath string
	var statFailedErr error
	for _, candidate := range candidates {
		inspection := inspectWorkspacePath(report.WorkspaceDir, candidate)
		if inspection.Outside {
			if outsidePath == "" {
				outsidePath = inspection.Path
			}
			continue
		}
		if inspection.Err != nil {
			if !errors.Is(inspection.Err, os.ErrNotExist) && statFailedPath == "" {
				statFailedPath = inspection.Path
				statFailedErr = inspection.Err
			}
			continue
		}
		if inspection.Info.IsDir() {
			notFilePaths = append(notFilePaths, inspection.Path)
			continue
		}
		existingFiles = append(existingFiles, inspection.Path)
	}

	switch {
	case len(existingFiles) > 1:
		report.add(DiagnosticLevelWarning, DiagnosticCodeAgentProfileAmbiguous, fmt.Sprintf("profile %q for agent %q matches multiple profile files; remove duplicates to make precedence explicit", profile, agentName), profilePatternPath(report.WorkspaceDir, profile))
	case len(existingFiles) == 1:
		return
	case outsidePath != "":
		report.add(DiagnosticLevelWarning, DiagnosticCodeAgentProfileOutside, fmt.Sprintf("profile %q for agent %q resolves outside the ADP workspace directory", profile, agentName), outsidePath)
	case len(notFilePaths) > 0:
		report.add(DiagnosticLevelWarning, DiagnosticCodeAgentProfileNotFile, fmt.Sprintf("profile %q for agent %q resolves to a directory, not a file", profile, agentName), notFilePaths[0])
	case statFailedPath != "":
		report.add(DiagnosticLevelWarning, DiagnosticCodeAgentProfileStatFailed, fmt.Sprintf("profile %q for agent %q could not be inspected: %v", profile, agentName, statFailedErr), statFailedPath)
	default:
		report.add(DiagnosticLevelWarning, DiagnosticCodeAgentProfileMissing, fmt.Sprintf("non-default profile %q for agent %q has no profile file", profile, agentName), profilePatternPath(report.WorkspaceDir, profile))
	}
}

func validProfileName(profile string) bool {
	if profile == "." || profile == ".." || filepath.IsAbs(profile) {
		return false
	}
	if strings.ContainsAny(profile, `/\`) {
		return false
	}
	return filepath.Clean(profile) == profile
}

func profilePatternPath(workspaceDir string, profile string) string {
	return filepath.Join(workspaceDir, "profiles", profile+".{md,yaml,yml,json}")
}
