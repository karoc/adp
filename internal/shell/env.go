package shell

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/karoc/adp/internal/adapters"
)

var ErrInvalidExportName = errors.New("invalid shell export name")

type ExportOptions struct {
	ChangeDir bool
}

func RenderExports(handle adapters.RuntimeHandle, opts ExportOptions) (string, error) {
	if opts.ChangeDir && handle.Root == "" {
		return "", ErrRuntimeRootRequired
	}

	keys := make([]string, 0, len(handle.Env))
	for key := range handle.Env {
		if !isRuntimeExportName(key) {
			continue
		}
		if !isShellName(key) {
			return "", fmt.Errorf("%w: %s", ErrInvalidExportName, key)
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var out strings.Builder
	for _, key := range keys {
		out.WriteString("export ")
		out.WriteString(key)
		out.WriteByte('=')
		out.WriteString(shellQuote(handle.Env[key]))
		out.WriteByte('\n')
	}

	if opts.ChangeDir {
		out.WriteString("cd ")
		out.WriteString(shellQuote(handle.Root))
		out.WriteByte('\n')
	}

	return out.String(), nil
}

func isRuntimeExportName(name string) bool {
	return strings.HasPrefix(name, "ADP_") || name == "GIT_CEILING_DIRECTORIES"
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func isShellName(name string) bool {
	if name == "" {
		return false
	}

	for index, r := range name {
		if index == 0 {
			if !isShellNameStart(r) {
				return false
			}
			continue
		}
		if !isShellNamePart(r) {
			return false
		}
	}
	return true
}

func isShellNameStart(r rune) bool {
	return r == '_' || ('A' <= r && r <= 'Z') || ('a' <= r && r <= 'z')
}

func isShellNamePart(r rune) bool {
	return isShellNameStart(r) || ('0' <= r && r <= '9')
}
