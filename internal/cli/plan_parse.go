package cli

import (
	"errors"
	"fmt"
)

type planInputOptions struct {
	workspace string
	file      string
	format    outputFormat
}

func parsePlanInputArgs(args []string, usage string) (planInputOptions, error) {
	opts := planInputOptions{format: outputFormatText}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--workspace", "-w":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return planInputOptions{}, err
			}
			opts.workspace = value
			i = next
		case "--file", "-f":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return planInputOptions{}, err
			}
			opts.file = value
			i = next
		case "--format":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return planInputOptions{}, err
			}
			format, err := parseOutputFormat(value)
			if err != nil {
				return planInputOptions{}, err
			}
			opts.format = format
			i = next
		default:
			if len(arg) > 0 && arg[0] == '-' {
				return planInputOptions{}, fmt.Errorf("unknown plan option %q", arg)
			}
			return planInputOptions{}, errors.New("usage: " + usage)
		}
	}
	if opts.file == "" {
		return planInputOptions{}, errors.New("usage: " + usage)
	}
	return opts, nil
}
