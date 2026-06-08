package shell

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

const (
	defaultCompletionShell       = "bash"
	defaultCompletionCommandName = "adp"
)

var (
	ErrUnsupportedCompletionShell   = errors.New("unsupported shell for completion")
	ErrInvalidCompletionCommandName = errors.New("invalid shell completion command name")
)

type CompletionOptions struct {
	Shell       string
	CommandName string
}

func RenderCompletion(opts CompletionOptions) (string, error) {
	shellName := normalizeCompletionShell(opts.Shell)
	if !isSupportedCompletionShell(shellName) {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedCompletionShell, opts.Shell)
	}

	commandName := opts.CommandName
	if commandName == "" {
		commandName = defaultCompletionCommandName
	}
	if !isValidCompletionCommandName(commandName) {
		return "", fmt.Errorf("%w: %s", ErrInvalidCompletionCommandName, commandName)
	}

	switch shellName {
	case "bash":
		return renderBashCompletion(commandName), nil
	case "zsh":
		return renderZshCompletion(commandName), nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedCompletionShell, opts.Shell)
	}
}

func normalizeCompletionShell(shellName string) string {
	shellName = strings.TrimSpace(shellName)
	if shellName == "" {
		return defaultCompletionShell
	}

	shellName = filepath.Base(shellName)
	shellName = strings.TrimLeft(shellName, "-")
	shellName = strings.ToLower(shellName)
	return strings.TrimSuffix(shellName, ".exe")
}

func isSupportedCompletionShell(shellName string) bool {
	switch shellName {
	case "bash", "zsh":
		return true
	default:
		return false
	}
}

func isValidCompletionCommandName(commandName string) bool {
	if commandName == "" || isShellReservedWord(commandName) {
		return false
	}

	for index, r := range commandName {
		if index == 0 {
			if !isShellNameStart(r) {
				return false
			}
			continue
		}
		if r != '-' && !isShellNamePart(r) {
			return false
		}
	}
	return true
}

func completionFunctionName(commandName string) string {
	return "_" + strings.ReplaceAll(commandName, "-", "_") + "_completion"
}
