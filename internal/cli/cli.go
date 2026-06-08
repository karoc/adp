package cli

import (
	"context"
	"fmt"
	"io"
)

const usage = `adp - Agent Development Platform

Usage:
  adp init
  adp workspace add <name> <project-root>
  adp enter <workspace>
  adp run <agent> [--workspace <name>] [-- <agent-args>...]
`

// Execute is the temporary CLI entrypoint used by the scaffold.
// The CLI/Foundation task will replace this dispatcher with the full command tree.
func Execute(_ context.Context, args []string, stdout io.Writer, _ io.Writer) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Fprint(stdout, usage)
		return 0
	}

	fmt.Fprintln(stdout, "adp scaffold: command wiring is not implemented yet")
	return 2
}
