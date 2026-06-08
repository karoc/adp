package cli

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type tasksClaimOptions struct {
	workspace string
	taskID    string
	owner     string
	lease     time.Duration
}

type tasksReleaseOptions struct {
	workspace string
	taskID    string
	owner     string
}

type phaseAddOptions struct {
	workspace string
	id        string
	title     string
	goal      string
}

type phaseAcceptOptions struct {
	workspace string
	id        string
	commands  []string
	result    string
	notes     string
}

type phaseCommitOptions struct {
	workspace string
	id        string
	hash      string
	message   string
}

type phasePushOptions struct {
	workspace string
	id        string
	remote    string
	branch    string
	result    string
}

func parseTasksClaimArgs(args []string) (tasksClaimOptions, error) {
	opts := tasksClaimOptions{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--workspace", "-w":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return tasksClaimOptions{}, err
			}
			opts.workspace, i = value, next
		case "--owner":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return tasksClaimOptions{}, err
			}
			opts.owner, i = value, next
		case "--lease":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return tasksClaimOptions{}, err
			}
			lease, err := time.ParseDuration(value)
			if err != nil {
				return tasksClaimOptions{}, fmt.Errorf("parse lease duration: %w", err)
			}
			if lease < 0 {
				return tasksClaimOptions{}, errors.New("lease must not be negative")
			}
			opts.lease, i = lease, next
		default:
			if strings.HasPrefix(arg, "-") {
				return tasksClaimOptions{}, fmt.Errorf("unknown tasks claim option %q", arg)
			}
			if opts.taskID != "" {
				return tasksClaimOptions{}, errors.New("usage: adp tasks claim [--workspace <name>] <task-id> --owner <owner> [--lease <duration>]")
			}
			opts.taskID = arg
		}
	}
	if opts.taskID == "" || opts.owner == "" {
		return tasksClaimOptions{}, errors.New("usage: adp tasks claim [--workspace <name>] <task-id> --owner <owner> [--lease <duration>]")
	}
	return opts, nil
}

func parsePhaseAddArgs(args []string) (phaseAddOptions, error) {
	opts := phaseAddOptions{}
	titleParts := []string{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--workspace", "-w":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return phaseAddOptions{}, err
			}
			opts.workspace, i = value, next
		case "--goal":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return phaseAddOptions{}, err
			}
			opts.goal, i = value, next
		default:
			if strings.HasPrefix(arg, "-") {
				return phaseAddOptions{}, fmt.Errorf("unknown phase add option %q", arg)
			}
			if opts.id == "" {
				opts.id = arg
				continue
			}
			titleParts = append(titleParts, arg)
		}
	}
	opts.title = joinTitle(titleParts)
	if opts.id == "" || opts.title == "" {
		return phaseAddOptions{}, errors.New("usage: adp phase add [--workspace <name>] [--goal <text>] <phase-id> <title>")
	}
	return opts, nil
}

func parseTasksReleaseArgs(args []string) (tasksReleaseOptions, error) {
	opts := tasksReleaseOptions{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--workspace", "-w":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return tasksReleaseOptions{}, err
			}
			opts.workspace, i = value, next
		case "--owner":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return tasksReleaseOptions{}, err
			}
			opts.owner, i = value, next
		default:
			if strings.HasPrefix(arg, "-") {
				return tasksReleaseOptions{}, fmt.Errorf("unknown tasks release option %q", arg)
			}
			if opts.taskID != "" {
				return tasksReleaseOptions{}, errors.New("usage: adp tasks release [--workspace <name>] <task-id> [--owner <owner>]")
			}
			opts.taskID = arg
		}
	}
	if opts.taskID == "" {
		return tasksReleaseOptions{}, errors.New("usage: adp tasks release [--workspace <name>] <task-id> [--owner <owner>]")
	}
	return opts, nil
}

func parsePhaseAcceptArgs(args []string) (phaseAcceptOptions, error) {
	opts := phaseAcceptOptions{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--workspace", "-w":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return phaseAcceptOptions{}, err
			}
			opts.workspace, i = value, next
		case "--command":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return phaseAcceptOptions{}, err
			}
			opts.commands, i = append(opts.commands, value), next
		case "--result":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return phaseAcceptOptions{}, err
			}
			opts.result, i = value, next
		case "--notes":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return phaseAcceptOptions{}, err
			}
			opts.notes, i = value, next
		default:
			if strings.HasPrefix(arg, "-") {
				return phaseAcceptOptions{}, fmt.Errorf("unknown phase accept option %q", arg)
			}
			if opts.id != "" {
				return phaseAcceptOptions{}, errors.New("usage: adp phase accept [--workspace <name>] <phase-id> [--command <cmd>] [--result <result>] [--notes <text>]")
			}
			opts.id = arg
		}
	}
	if opts.id == "" {
		return phaseAcceptOptions{}, errors.New("usage: adp phase accept [--workspace <name>] <phase-id> [--command <cmd>] [--result <result>] [--notes <text>]")
	}
	return opts, nil
}

func parsePhaseCommitArgs(args []string) (phaseCommitOptions, error) {
	opts := phaseCommitOptions{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--workspace", "-w":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return phaseCommitOptions{}, err
			}
			opts.workspace, i = value, next
		case "--hash":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return phaseCommitOptions{}, err
			}
			opts.hash, i = value, next
		case "--message":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return phaseCommitOptions{}, err
			}
			opts.message, i = value, next
		default:
			if strings.HasPrefix(arg, "-") {
				return phaseCommitOptions{}, fmt.Errorf("unknown phase commit option %q", arg)
			}
			if opts.id != "" {
				return phaseCommitOptions{}, errors.New("usage: adp phase commit [--workspace <name>] <phase-id> --hash <commit-hash> [--message <text>]")
			}
			opts.id = arg
		}
	}
	if opts.id == "" || opts.hash == "" {
		return phaseCommitOptions{}, errors.New("usage: adp phase commit [--workspace <name>] <phase-id> --hash <commit-hash> [--message <text>]")
	}
	return opts, nil
}

func parsePhasePushArgs(args []string) (phasePushOptions, error) {
	opts := phasePushOptions{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--workspace", "-w":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return phasePushOptions{}, err
			}
			opts.workspace, i = value, next
		case "--remote":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return phasePushOptions{}, err
			}
			opts.remote, i = value, next
		case "--branch":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return phasePushOptions{}, err
			}
			opts.branch, i = value, next
		case "--result":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return phasePushOptions{}, err
			}
			opts.result, i = value, next
		default:
			if strings.HasPrefix(arg, "-") {
				return phasePushOptions{}, fmt.Errorf("unknown phase push option %q", arg)
			}
			if opts.id != "" {
				return phasePushOptions{}, errors.New("usage: adp phase push [--workspace <name>] <phase-id> --remote <remote> --branch <branch> [--result <result>]")
			}
			opts.id = arg
		}
	}
	if opts.id == "" || opts.remote == "" || opts.branch == "" {
		return phasePushOptions{}, errors.New("usage: adp phase push [--workspace <name>] <phase-id> --remote <remote> --branch <branch> [--result <result>]")
	}
	return opts, nil
}

func parsePhaseIDArgs(args []string, usage string) (string, string, error) {
	var workspace string
	var phaseID string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--workspace", "-w":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return "", "", err
			}
			workspace, i = value, next
		default:
			if strings.HasPrefix(arg, "-") {
				return "", "", fmt.Errorf("unknown phase option %q", arg)
			}
			if phaseID != "" {
				return "", "", errors.New("usage: " + usage)
			}
			phaseID = arg
		}
	}
	if phaseID == "" {
		return "", "", errors.New("usage: " + usage)
	}
	return workspace, phaseID, nil
}

func requireValue(args []string, index int, option string) (string, int, error) {
	if index+1 >= len(args) {
		return "", index, fmt.Errorf("%s requires a value", option)
	}
	return args[index+1], index + 1, nil
}
