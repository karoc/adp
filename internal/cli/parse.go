package cli

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/karoc/adp/internal/shell"
)

const defaultEventLimit = 20
const defaultSessionLimit = 20

type runOptions struct {
	agent     string
	workspace string
	profile   string
	taskID    string
	take      bool
	owner     string
	lease     time.Duration
	keep      bool
	agentArgs []string
}

type eventsListOptions struct {
	workspace string
	sessionID string
	taskID    string
	eventType string
	limit     int
}

type sessionsListOptions struct {
	workspace string
	agent     string
	taskID    string
	limit     int
}

type sessionsResumePlanOptions struct {
	sessionID   string
	workspace   string
	owner       string
	lease       time.Duration
	targetAgent string
	format      outputFormat
}

type runtimePruneOptions struct {
	olderThan   time.Duration
	includeKept bool
	dryRun      bool
}

type completionValuesOptions struct {
	kind      string
	workspace string
}

func parseRunArgs(args []string) (runOptions, error) {
	if len(args) == 0 {
		return runOptions{}, errors.New("usage: adp run <agent> [--workspace <name>] [--profile <profile>] [--task <task-id>|--take --owner <owner> [--lease <duration>]] [--keep-runtime] [-- <agent-args>...]")
	}
	opts := runOptions{agent: args[0]}
	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--":
			opts.agentArgs = append(opts.agentArgs, args[i+1:]...)
			return opts, nil
		case "--workspace", "-w":
			if i+1 >= len(args) {
				return runOptions{}, fmt.Errorf("%s requires a value", arg)
			}
			i++
			opts.workspace = args[i]
		case "--profile", "-p":
			if i+1 >= len(args) {
				return runOptions{}, fmt.Errorf("%s requires a value", arg)
			}
			i++
			opts.profile = args[i]
		case "--task":
			if i+1 >= len(args) {
				return runOptions{}, fmt.Errorf("%s requires a value", arg)
			}
			i++
			opts.taskID = args[i]
		case "--take":
			opts.take = true
		case "--owner":
			if i+1 >= len(args) {
				return runOptions{}, fmt.Errorf("%s requires a value", arg)
			}
			i++
			opts.owner = args[i]
		case "--lease":
			if i+1 >= len(args) {
				return runOptions{}, fmt.Errorf("%s requires a value", arg)
			}
			i++
			lease, err := time.ParseDuration(args[i])
			if err != nil {
				return runOptions{}, fmt.Errorf("parse lease duration: %w", err)
			}
			if lease < 0 {
				return runOptions{}, errors.New("lease must not be negative")
			}
			opts.lease = lease
		case "--keep-runtime":
			opts.keep = true
		default:
			return runOptions{}, fmt.Errorf("unknown run option %q", arg)
		}
	}
	if opts.take && strings.TrimSpace(opts.taskID) != "" {
		return runOptions{}, errors.New("--take cannot be combined with --task")
	}
	if opts.take && strings.TrimSpace(opts.owner) == "" {
		return runOptions{}, errors.New("--owner is required with --take")
	}
	if !opts.take && strings.TrimSpace(opts.owner) != "" {
		return runOptions{}, errors.New("--owner requires --take")
	}
	if !opts.take && opts.lease != 0 {
		return runOptions{}, errors.New("--lease requires --take")
	}
	return opts, nil
}

func parseEnterArgs(args []string) (string, bool, error) {
	var name string
	var keep bool
	for _, arg := range args {
		switch arg {
		case "--keep-runtime":
			keep = true
		default:
			if strings.HasPrefix(arg, "-") {
				return "", false, fmt.Errorf("unknown enter option %q", arg)
			}
			if name != "" {
				return "", false, errors.New("usage: adp enter <workspace> [--keep-runtime]")
			}
			name = arg
		}
	}
	if name == "" {
		return "", false, errors.New("usage: adp enter <workspace> [--keep-runtime]")
	}
	return name, keep, nil
}

func parseEnvArgs(args []string) (string, bool, error) {
	var name string
	var changeDir bool
	for _, arg := range args {
		switch arg {
		case "--cd":
			changeDir = true
		default:
			if strings.HasPrefix(arg, "-") {
				return "", false, fmt.Errorf("unknown env option %q", arg)
			}
			if name != "" {
				return "", false, errors.New("usage: adp env <workspace> [--cd]")
			}
			name = arg
		}
	}
	if name == "" {
		return "", false, errors.New("usage: adp env <workspace> [--cd]")
	}
	return name, changeDir, nil
}

func parseShellHookArgs(args []string) (shell.HookOptions, error) {
	var opts shell.HookOptions
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--shell", "-s":
			if i+1 >= len(args) {
				return shell.HookOptions{}, fmt.Errorf("%s requires a value", arg)
			}
			i++
			opts.Shell = args[i]
		case "--name", "--function", "-n":
			if i+1 >= len(args) {
				return shell.HookOptions{}, fmt.Errorf("%s requires a value", arg)
			}
			i++
			opts.FunctionName = args[i]
		default:
			return shell.HookOptions{}, fmt.Errorf("unknown shell-hook option %q", arg)
		}
	}
	return opts, nil
}

func parseCompletionArgs(args []string) (shell.CompletionOptions, error) {
	var opts shell.CompletionOptions
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--shell", "-s":
			if i+1 >= len(args) {
				return shell.CompletionOptions{}, fmt.Errorf("%s requires a value", arg)
			}
			i++
			opts.Shell = args[i]
		case "--command":
			if i+1 >= len(args) {
				return shell.CompletionOptions{}, fmt.Errorf("%s requires a value", arg)
			}
			i++
			opts.CommandName = args[i]
		default:
			return shell.CompletionOptions{}, fmt.Errorf("unknown completion option %q", arg)
		}
	}
	return opts, nil
}

func parseCompletionValuesArgs(args []string) (completionValuesOptions, error) {
	if len(args) == 0 {
		return completionValuesOptions{}, errors.New("usage: adp completion values <agents|workspaces|profiles|tasks|phases|sessions|owners|statuses> [--workspace <name>]")
	}
	opts := completionValuesOptions{kind: args[0]}
	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--workspace", "-w":
			if i+1 >= len(args) {
				return completionValuesOptions{}, fmt.Errorf("%s requires a value", arg)
			}
			i++
			opts.workspace = args[i]
		default:
			return completionValuesOptions{}, fmt.Errorf("unknown completion values option %q", arg)
		}
	}
	return opts, nil
}

func parseEventsListArgs(args []string) (eventsListOptions, error) {
	opts := eventsListOptions{limit: defaultEventLimit}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--workspace", "-w":
			if i+1 >= len(args) {
				return eventsListOptions{}, fmt.Errorf("%s requires a value", arg)
			}
			i++
			opts.workspace = args[i]
		case "--session":
			if i+1 >= len(args) {
				return eventsListOptions{}, fmt.Errorf("%s requires a value", arg)
			}
			i++
			opts.sessionID = args[i]
		case "--task":
			if i+1 >= len(args) {
				return eventsListOptions{}, fmt.Errorf("%s requires a value", arg)
			}
			i++
			opts.taskID = args[i]
		case "--type":
			if i+1 >= len(args) {
				return eventsListOptions{}, fmt.Errorf("%s requires a value", arg)
			}
			i++
			opts.eventType = args[i]
		case "--limit":
			if i+1 >= len(args) {
				return eventsListOptions{}, fmt.Errorf("%s requires a value", arg)
			}
			i++
			limit, err := parseNonNegativeInt(args[i], "limit")
			if err != nil {
				return eventsListOptions{}, err
			}
			opts.limit = limit
		default:
			return eventsListOptions{}, fmt.Errorf("unknown events list option %q", arg)
		}
	}
	return opts, nil
}

func parseSessionsListArgs(args []string) (sessionsListOptions, error) {
	opts := sessionsListOptions{limit: defaultSessionLimit}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--workspace", "-w":
			if i+1 >= len(args) {
				return sessionsListOptions{}, fmt.Errorf("%s requires a value", arg)
			}
			i++
			opts.workspace = args[i]
		case "--agent":
			if i+1 >= len(args) {
				return sessionsListOptions{}, fmt.Errorf("%s requires a value", arg)
			}
			i++
			opts.agent = args[i]
		case "--task":
			if i+1 >= len(args) {
				return sessionsListOptions{}, fmt.Errorf("%s requires a value", arg)
			}
			i++
			opts.taskID = args[i]
		case "--limit":
			if i+1 >= len(args) {
				return sessionsListOptions{}, fmt.Errorf("%s requires a value", arg)
			}
			i++
			limit, err := parseNonNegativeInt(args[i], "limit")
			if err != nil {
				return sessionsListOptions{}, err
			}
			opts.limit = limit
		default:
			return sessionsListOptions{}, fmt.Errorf("unknown sessions list option %q", arg)
		}
	}
	return opts, nil
}

func parseSessionsShowArgs(args []string) (string, error) {
	if len(args) != 1 {
		return "", errors.New("usage: adp sessions show <session-id>")
	}
	return args[0], nil
}

func parseSessionsRestorePlanArgs(args []string) (string, error) {
	if len(args) != 1 {
		return "", errors.New("usage: adp sessions restore-plan <session-id>")
	}
	return args[0], nil
}

func parseSessionsResumePlanArgs(args []string) (sessionsResumePlanOptions, error) {
	opts := sessionsResumePlanOptions{format: outputFormatText}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--workspace", "-w":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return sessionsResumePlanOptions{}, err
			}
			opts.workspace, i = value, next
		case "--owner":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return sessionsResumePlanOptions{}, err
			}
			opts.owner, i = value, next
		case "--lease":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return sessionsResumePlanOptions{}, err
			}
			lease, err := time.ParseDuration(value)
			if err != nil {
				return sessionsResumePlanOptions{}, fmt.Errorf("parse lease duration: %w", err)
			}
			if lease < 0 {
				return sessionsResumePlanOptions{}, errors.New("lease must not be negative")
			}
			opts.lease, i = lease, next
		case "--agent":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return sessionsResumePlanOptions{}, err
			}
			opts.targetAgent, i = value, next
		case "--format":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return sessionsResumePlanOptions{}, err
			}
			format, err := parseOutputFormat(value)
			if err != nil {
				return sessionsResumePlanOptions{}, err
			}
			opts.format, i = format, next
		default:
			if strings.HasPrefix(arg, "-") {
				return sessionsResumePlanOptions{}, fmt.Errorf("unknown sessions resume-plan option %q", arg)
			}
			if opts.sessionID != "" {
				return sessionsResumePlanOptions{}, errors.New("usage: adp sessions resume-plan <session-id> [--workspace <name>] [--owner <owner>] [--lease <duration>] [--agent <agent>] [--format <text|json>]")
			}
			opts.sessionID = arg
		}
	}
	if opts.sessionID == "" {
		return sessionsResumePlanOptions{}, errors.New("usage: adp sessions resume-plan <session-id> [--workspace <name>] [--owner <owner>] [--lease <duration>] [--agent <agent>] [--format <text|json>]")
	}
	return opts, nil
}

func parseRuntimePruneArgs(args []string) (runtimePruneOptions, error) {
	opts := runtimePruneOptions{olderThan: 24 * time.Hour}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--older-than":
			if i+1 >= len(args) {
				return runtimePruneOptions{}, fmt.Errorf("%s requires a value", arg)
			}
			i++
			olderThan, err := time.ParseDuration(args[i])
			if err != nil {
				return runtimePruneOptions{}, fmt.Errorf("parse older-than duration: %w", err)
			}
			if olderThan < 0 {
				return runtimePruneOptions{}, errors.New("older-than must not be negative")
			}
			opts.olderThan = olderThan
		case "--include-kept":
			opts.includeKept = true
		case "--dry-run":
			opts.dryRun = true
		default:
			return runtimePruneOptions{}, fmt.Errorf("unknown runtime prune option %q", arg)
		}
	}
	return opts, nil
}
