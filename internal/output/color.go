// Package output provides terminal output formatting utilities.
package output

import (
	"fmt"
	"os"
)

// Color represents an ANSI color code.
type Color int

const (
	ColorReset Color = iota
	ColorRed
	ColorGreen
	ColorYellow
	ColorBlue
	ColorMagenta
	ColorCyan
	ColorBold
)

var (
	// colorEnabled is initialized once at package init time.
	colorEnabled = supportsColor()

	// ANSI escape codes for each color.
	colorCodes = map[Color]string{
		ColorRed:     "\033[31m",
		ColorGreen:   "\033[32m",
		ColorYellow:  "\033[33m",
		ColorBlue:    "\033[34m",
		ColorMagenta: "\033[35m",
		ColorCyan:    "\033[36m",
		ColorBold:    "\033[1m",
		ColorReset:   "\033[0m",
	}
)

// supportsColor checks if the terminal supports color output.
// It returns false if:
// - NO_COLOR environment variable is set (per https://no-color.org/)
// - stdout is not a terminal (e.g., piped to file)
func supportsColor() bool {
	// Respect NO_COLOR environment variable
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Check if stdout is a terminal
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}

	// If not a character device, it's not a terminal
	if (fileInfo.Mode() & os.ModeCharDevice) == 0 {
		return false
	}

	return true
}

// Colorize wraps text with ANSI color codes.
// If color is disabled, returns text unchanged.
func Colorize(color Color, text string) string {
	if !colorEnabled {
		return text
	}
	code, ok := colorCodes[color]
	if !ok {
		return text
	}
	return code + text + colorCodes[ColorReset]
}

// Colorizef formats and colorizes text.
func Colorizef(color Color, format string, args ...interface{}) string {
	return Colorize(color, fmt.Sprintf(format, args...))
}

// Error returns text formatted as an error (red).
func Error(text string) string {
	return Colorize(ColorRed, text)
}

// Errorf formats and returns text as an error (red).
func Errorf(format string, args ...interface{}) string {
	return Colorize(ColorRed, fmt.Sprintf(format, args...))
}

// Success returns text formatted as success (green).
func Success(text string) string {
	return Colorize(ColorGreen, text)
}

// Successf formats and returns text as success (green).
func Successf(format string, args ...interface{}) string {
	return Colorize(ColorGreen, fmt.Sprintf(format, args...))
}

// Warning returns text formatted as a warning (yellow).
func Warning(text string) string {
	return Colorize(ColorYellow, text)
}

// Warningf formats and returns text as a warning (yellow).
func Warningf(format string, args ...interface{}) string {
	return Colorize(ColorYellow, fmt.Sprintf(format, args...))
}

// Command returns text formatted as a command (cyan).
func Command(text string) string {
	return Colorize(ColorCyan, text)
}

// Commandf formats and returns text as a command (cyan).
func Commandf(format string, args ...interface{}) string {
	return Colorize(ColorCyan, fmt.Sprintf(format, args...))
}

// Bold returns text in bold.
func Bold(text string) string {
	return Colorize(ColorBold, text)
}

// Boldf formats and returns text in bold.
func Boldf(format string, args ...interface{}) string {
	return Colorize(ColorBold, fmt.Sprintf(format, args...))
}

// Enabled returns true if color output is enabled.
func Enabled() bool {
	return colorEnabled
}

// Disable disables color output (for testing).
func Disable() {
	colorEnabled = false
}

// Enable enables color output (for testing).
func Enable() {
	colorEnabled = true
}
