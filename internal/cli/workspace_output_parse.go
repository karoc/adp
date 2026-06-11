package cli

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type outputFormat string

const (
	outputFormatText outputFormat = "text"
	outputFormatJSON outputFormat = "json"
)

type workspaceOutputOptions struct {
	workspace string
	format    outputFormat
}

type doctorOptions struct {
	workspace string
	format    outputFormat
	verbose   bool
}

func parseWorkspaceOnlyArgs(args []string, usage string) (string, error) {
	var workspace string
	command := usageCommandLabel(usage, "workspace")
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--workspace", "-w":
			if i+1 >= len(args) {
				return "", fmt.Errorf("%s requires a value", arg)
			}
			i++
			workspace = args[i]
		default:
			return "", fmt.Errorf("unknown %s option %q", command, arg)
		}
	}
	return workspace, nil
}

func parseWorkspaceOutputArgs(args []string, usage string) (workspaceOutputOptions, error) {
	opts := workspaceOutputOptions{format: outputFormatText}
	command := usageCommandLabel(usage, "workspace")
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--workspace", "-w":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return workspaceOutputOptions{}, err
			}
			opts.workspace, i = value, next
		case "--format":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return workspaceOutputOptions{}, err
			}
			format, err := parseOutputFormat(value)
			if err != nil {
				return workspaceOutputOptions{}, err
			}
			opts.format, i = format, next
		default:
			return workspaceOutputOptions{}, fmt.Errorf("unknown %s option %q", command, arg)
		}
	}
	return opts, nil
}

func parseDoctorArgs(args []string) (doctorOptions, error) {
	opts := doctorOptions{format: outputFormatText}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--format":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return doctorOptions{}, err
			}
			format, err := parseOutputFormat(value)
			if err != nil {
				return doctorOptions{}, err
			}
			opts.format, i = format, next
		case "--verbose":
			opts.verbose = true
		default:
			if strings.HasPrefix(arg, "-") {
				return doctorOptions{}, fmt.Errorf("unknown doctor option %q", arg)
			}
			if opts.workspace != "" {
				return doctorOptions{}, errors.New("usage: adp doctor [workspace] [--verbose] [--format <text|json>]")
			}
			opts.workspace = arg
		}
	}
	return opts, nil
}

func usageCommandLabel(usage string, fallback string) string {
	fields := strings.Fields(usage)
	if len(fields) < 2 || fields[0] != "adp" {
		return fallback
	}
	if len(fields) >= 3 && !strings.HasPrefix(fields[2], "[") && !strings.HasPrefix(fields[2], "<") {
		return fields[1] + " " + fields[2]
	}
	return fields[1]
}

func parseOutputFormat(value string) (outputFormat, error) {
	switch outputFormat(strings.TrimSpace(value)) {
	case "", outputFormatText:
		return outputFormatText, nil
	case outputFormatJSON:
		return outputFormatJSON, nil
	default:
		return "", fmt.Errorf("unknown output format %q", value)
	}
}

func parseNonNegativeInt(value string, name string) (int, error) {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	if parsed < 0 {
		return 0, fmt.Errorf("%s must not be negative", name)
	}
	return parsed, nil
}
