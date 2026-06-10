package cli

import (
	"errors"
	"fmt"
	"strings"
)

type tasksAddOptions struct {
	workspace   string
	title       string
	description string
	priority    string
	phase       string
}

type tasksUpdateOptions struct {
	workspace string
	taskID    string
	status    string
}

type tasksBlockOptions struct {
	workspace string
	taskID    string
	reason    string
}

type taskIDOutputOptions struct {
	workspace string
	taskID    string
	format    outputFormat
}

func parseTasksAddArgs(args []string) (tasksAddOptions, error) {
	opts := tasksAddOptions{}
	titleParts := []string{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--workspace", "-w":
			if i+1 >= len(args) {
				return tasksAddOptions{}, fmt.Errorf("%s requires a value", arg)
			}
			i++
			opts.workspace = args[i]
		case "--description":
			if i+1 >= len(args) {
				return tasksAddOptions{}, fmt.Errorf("%s requires a value", arg)
			}
			i++
			opts.description = args[i]
		case "--priority":
			if i+1 >= len(args) {
				return tasksAddOptions{}, fmt.Errorf("%s requires a value", arg)
			}
			i++
			opts.priority = args[i]
		case "--phase":
			if i+1 >= len(args) {
				return tasksAddOptions{}, fmt.Errorf("%s requires a value", arg)
			}
			i++
			opts.phase = args[i]
		default:
			if strings.HasPrefix(arg, "-") {
				return tasksAddOptions{}, fmt.Errorf("unknown tasks add option %q", arg)
			}
			titleParts = append(titleParts, arg)
		}
	}
	opts.title = joinTitle(titleParts)
	if opts.title == "" {
		return tasksAddOptions{}, errors.New("title is required; usage: adp tasks add [--workspace <name>] [--priority <value>] [--phase <value>] [--description <text>] <title>")
	}
	return opts, nil
}

func parseTasksUpdateArgs(args []string) (tasksUpdateOptions, error) {
	opts := tasksUpdateOptions{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--workspace", "-w":
			if i+1 >= len(args) {
				return tasksUpdateOptions{}, fmt.Errorf("%s requires a value", arg)
			}
			i++
			opts.workspace = args[i]
		case "--status":
			if i+1 >= len(args) {
				return tasksUpdateOptions{}, fmt.Errorf("%s requires a value", arg)
			}
			i++
			opts.status = args[i]
		default:
			if strings.HasPrefix(arg, "-") {
				return tasksUpdateOptions{}, fmt.Errorf("unknown tasks update option %q", arg)
			}
			if opts.taskID != "" {
				return tasksUpdateOptions{}, errors.New("usage: adp tasks update [--workspace <name>] <task-id> --status <status>")
			}
			opts.taskID = arg
		}
	}
	switch {
	case opts.taskID == "":
		return tasksUpdateOptions{}, errors.New("task-id is required; usage: adp tasks update [--workspace <name>] <task-id> --status <status>")
	case opts.status == "":
		return tasksUpdateOptions{}, errors.New("--status is required; usage: adp tasks update [--workspace <name>] <task-id> --status <status>")
	}
	return opts, nil
}

func parseTasksBlockArgs(args []string) (tasksBlockOptions, error) {
	opts := tasksBlockOptions{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--workspace", "-w":
			if i+1 >= len(args) {
				return tasksBlockOptions{}, fmt.Errorf("%s requires a value", arg)
			}
			i++
			opts.workspace = args[i]
		case "--reason":
			if i+1 >= len(args) {
				return tasksBlockOptions{}, fmt.Errorf("%s requires a value", arg)
			}
			i++
			opts.reason = args[i]
		default:
			if strings.HasPrefix(arg, "-") {
				return tasksBlockOptions{}, fmt.Errorf("unknown tasks block option %q", arg)
			}
			if opts.taskID != "" {
				return tasksBlockOptions{}, errors.New("usage: adp tasks block [--workspace <name>] <task-id> --reason <reason>")
			}
			opts.taskID = arg
		}
	}
	switch {
	case opts.taskID == "":
		return tasksBlockOptions{}, errors.New("task-id is required; usage: adp tasks block [--workspace <name>] <task-id> --reason <reason>")
	case opts.reason == "":
		return tasksBlockOptions{}, errors.New("--reason is required; usage: adp tasks block [--workspace <name>] <task-id> --reason <reason>")
	}
	return opts, nil
}

func parseTaskIDArgs(args []string, usage string) (string, string, error) {
	var workspace string
	var taskID string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--workspace", "-w":
			if i+1 >= len(args) {
				return "", "", fmt.Errorf("%s requires a value", arg)
			}
			i++
			workspace = args[i]
		default:
			if strings.HasPrefix(arg, "-") {
				return "", "", fmt.Errorf("unknown task option %q", arg)
			}
			if taskID != "" {
				return "", "", errors.New("usage: " + usage)
			}
			taskID = arg
		}
	}
	if taskID == "" {
		return "", "", errors.New("task-id is required; usage: " + usage)
	}
	return workspace, taskID, nil
}

func parseTaskIDOutputArgs(args []string, usage string) (taskIDOutputOptions, error) {
	opts := taskIDOutputOptions{format: outputFormatText}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--workspace", "-w":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return taskIDOutputOptions{}, err
			}
			opts.workspace, i = value, next
		case "--format":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return taskIDOutputOptions{}, err
			}
			format, err := parseOutputFormat(value)
			if err != nil {
				return taskIDOutputOptions{}, err
			}
			opts.format, i = format, next
		default:
			if strings.HasPrefix(arg, "-") {
				return taskIDOutputOptions{}, fmt.Errorf("unknown task option %q", arg)
			}
			if opts.taskID != "" {
				return taskIDOutputOptions{}, errors.New("usage: " + usage)
			}
			opts.taskID = arg
		}
	}
	if opts.taskID == "" {
		return taskIDOutputOptions{}, errors.New("task-id is required; usage: " + usage)
	}
	return opts, nil
}
