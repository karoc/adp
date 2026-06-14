package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/karoc/adp/internal/output"
)

// confirmDangerous prompts the user to confirm a dangerous operation.
// If yesFlag is true, confirmation is skipped (for --yes/-y flag).
// If not running in a TTY, requires --yes flag to proceed.
func (a *App) confirmDangerous(operation, details string, yesFlag bool) error {
	// If --yes flag provided, skip confirmation
	if yesFlag {
		return nil
	}

	// Non-TTY environment requires explicit --yes
	if !isTTY(os.Stdin) {
		return fmt.Errorf("operation requires confirmation; use --yes to proceed in non-interactive mode")
	}

	// Show warning-colored operation message
	fmt.Fprintf(a.stderr, "\n%s\n", output.Warning(operation))
	if details != "" {
		fmt.Fprintf(a.stderr, "\n%s\n", details)
	}
	fmt.Fprintf(a.stderr, "\nContinue? [y/N] ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return err
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		return fmt.Errorf("operation cancelled")
	}

	return nil
}

// isTTY checks if the given file is a terminal (TTY).
func isTTY(file *os.File) bool {
	fileInfo, err := file.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}
