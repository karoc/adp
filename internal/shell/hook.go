package shell

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

const (
	defaultHookShell                   = "sh"
	defaultInteractiveHookFunctionName = "adp-enter"
	defaultPOSIXHookFunctionName       = "adp_enter"
)

var (
	ErrUnsupportedHookShell    = errors.New("unsupported shell for hook")
	ErrInvalidHookFunctionName = errors.New("invalid shell hook function name")
)

type HookOptions struct {
	Shell        string
	FunctionName string
}

func RenderHook(opts HookOptions) (string, error) {
	shellName := normalizeHookShell(opts.Shell)
	if !isSupportedHookShell(shellName) {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedHookShell, opts.Shell)
	}

	functionName := opts.FunctionName
	if functionName == "" {
		functionName = defaultHookFunctionName(shellName)
	}
	if !isValidHookFunctionName(shellName, functionName) {
		return "", fmt.Errorf("%w: %s", ErrInvalidHookFunctionName, functionName)
	}

	return renderHookFunction(functionName), nil
}

func renderHookFunction(functionName string) string {
	var out strings.Builder
	out.WriteString(functionName)
	out.WriteString("() {\n")
	out.WriteString("\tif [ \"$#\" -ne 1 ]; then\n")
	out.WriteString("\t\tprintf '%s\\n' 'usage: ")
	out.WriteString(functionName)
	out.WriteString(" <workspace>' >&2\n")
	out.WriteString("\t\treturn 2\n")
	out.WriteString("\tfi\n")
	out.WriteString("\teval \"$(adp env \"$1\" --cd)\"\n")
	out.WriteString("}\n")
	return out.String()
}

func normalizeHookShell(shellName string) string {
	shellName = strings.TrimSpace(shellName)
	if shellName == "" {
		return defaultHookShell
	}

	shellName = filepath.Base(shellName)
	shellName = strings.TrimLeft(shellName, "-")
	shellName = strings.ToLower(shellName)
	return strings.TrimSuffix(shellName, ".exe")
}

func isSupportedHookShell(shellName string) bool {
	switch shellName {
	case "sh", "bash", "zsh":
		return true
	default:
		return false
	}
}

func defaultHookFunctionName(shellName string) string {
	if shellName == "sh" {
		return defaultPOSIXHookFunctionName
	}
	return defaultInteractiveHookFunctionName
}

func isValidHookFunctionName(shellName, functionName string) bool {
	if isShellReservedWord(functionName) {
		return false
	}
	if shellName == "sh" {
		return isShellName(functionName)
	}
	return isExtendedShellFunctionName(functionName)
}

func isShellReservedWord(name string) bool {
	switch name {
	case "!", "{", "}", "case", "do", "done", "elif", "else", "esac",
		"fi", "for", "if", "in", "then", "until", "while":
		return true
	default:
		return false
	}
}

func isExtendedShellFunctionName(functionName string) bool {
	if functionName == "" {
		return false
	}

	for index, r := range functionName {
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
