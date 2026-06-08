package cli

import (
	"fmt"
	"strings"
)

const defaultTasksNextLimit = 5

type tasksNextOptions struct {
	workspace string
	format    outputFormat
	limit     int
}

func parseTasksNextArgs(args []string) (tasksNextOptions, error) {
	opts := tasksNextOptions{format: outputFormatText, limit: defaultTasksNextLimit}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--workspace", "-w":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return tasksNextOptions{}, err
			}
			opts.workspace, i = value, next
		case "--format":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return tasksNextOptions{}, err
			}
			format, err := parseOutputFormat(value)
			if err != nil {
				return tasksNextOptions{}, err
			}
			opts.format, i = format, next
		case "--limit":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return tasksNextOptions{}, err
			}
			limit, err := parseNonNegativeInt(value, "limit")
			if err != nil {
				return tasksNextOptions{}, err
			}
			opts.limit, i = limit, next
		default:
			if strings.HasPrefix(arg, "-") {
				return tasksNextOptions{}, fmt.Errorf("unknown tasks next option %q", arg)
			}
			return tasksNextOptions{}, fmt.Errorf("usage: %s", tasksNextUsage)
		}
	}
	return opts, nil
}
