package output

import (
	"os"
	"testing"
)

func TestSupportsColor(t *testing.T) {
	// Save original state
	origEnabled := colorEnabled
	defer func() { colorEnabled = origEnabled }()

	tests := []struct {
		name        string
		noColor     string
		wantEnabled bool
	}{
		{
			name:        "NO_COLOR set",
			noColor:     "1",
			wantEnabled: false,
		},
		{
			name:        "NO_COLOR empty",
			noColor:     "",
			wantEnabled: true, // Assumes stdout is TTY in test
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.noColor != "" {
				os.Setenv("NO_COLOR", tt.noColor)
				defer os.Unsetenv("NO_COLOR")
			}

			// Note: This test may fail if stdout is not a TTY
			// In CI environments, supportsColor() will return false
			got := supportsColor()
			if got && tt.noColor != "" {
				t.Errorf("supportsColor() with NO_COLOR=%q = %v, want false", tt.noColor, got)
			}
		})
	}
}

func TestColorize(t *testing.T) {
	// Force enable color for testing
	origEnabled := colorEnabled
	colorEnabled = true
	defer func() { colorEnabled = origEnabled }()

	tests := []struct {
		name  string
		color Color
		text  string
		want  string
	}{
		{
			name:  "red text",
			color: ColorRed,
			text:  "error",
			want:  "\033[31merror\033[0m",
		},
		{
			name:  "green text",
			color: ColorGreen,
			text:  "success",
			want:  "\033[32msuccess\033[0m",
		},
		{
			name:  "yellow text",
			color: ColorYellow,
			text:  "warning",
			want:  "\033[33mwarning\033[0m",
		},
		{
			name:  "cyan text",
			color: ColorCyan,
			text:  "command",
			want:  "\033[36mcommand\033[0m",
		},
		{
			name:  "bold text",
			color: ColorBold,
			text:  "important",
			want:  "\033[1mimportant\033[0m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Colorize(tt.color, tt.text)
			if got != tt.want {
				t.Errorf("Colorize(%v, %q) = %q, want %q", tt.color, tt.text, got, tt.want)
			}
		})
	}
}

func TestColorizeDisabled(t *testing.T) {
	// Force disable color for testing
	origEnabled := colorEnabled
	colorEnabled = false
	defer func() { colorEnabled = origEnabled }()

	text := "test"
	got := Colorize(ColorRed, text)
	if got != text {
		t.Errorf("Colorize with color disabled = %q, want %q", got, text)
	}
}

func TestHelperFunctions(t *testing.T) {
	// Force enable color for testing
	origEnabled := colorEnabled
	colorEnabled = true
	defer func() { colorEnabled = origEnabled }()

	tests := []struct {
		name string
		fn   func(string) string
		text string
		want string
	}{
		{"Error", Error, "failed", "\033[31mfailed\033[0m"},
		{"Success", Success, "ok", "\033[32mok\033[0m"},
		{"Warning", Warning, "caution", "\033[33mcaution\033[0m"},
		{"Command", Command, "adp run", "\033[36madp run\033[0m"},
		{"Bold", Bold, "title", "\033[1mtitle\033[0m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn(tt.text)
			if got != tt.want {
				t.Errorf("%s(%q) = %q, want %q", tt.name, tt.text, got, tt.want)
			}
		})
	}
}

func TestHelperFunctionsWithFormat(t *testing.T) {
	// Force enable color for testing
	origEnabled := colorEnabled
	colorEnabled = true
	defer func() { colorEnabled = origEnabled }()

	tests := []struct {
		name   string
		fn     func(string, ...interface{}) string
		format string
		args   []interface{}
		want   string
	}{
		{"Errorf", Errorf, "error: %s", []interface{}{"not found"}, "\033[31merror: not found\033[0m"},
		{"Successf", Successf, "created %d items", []interface{}{5}, "\033[32mcreated 5 items\033[0m"},
		{"Warningf", Warningf, "caution: %v", []interface{}{"slow"}, "\033[33mcaution: slow\033[0m"},
		{"Commandf", Commandf, "adp %s", []interface{}{"run"}, "\033[36madp run\033[0m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn(tt.format, tt.args...)
			if got != tt.want {
				t.Errorf("%s(%q, %v) = %q, want %q", tt.name, tt.format, tt.args, got, tt.want)
			}
		})
	}
}

func TestEnabledToggle(t *testing.T) {
	// Save original state
	orig := colorEnabled
	defer func() { colorEnabled = orig }()

	Enable()
	if !Enabled() {
		t.Error("Enable() did not enable color")
	}

	Disable()
	if Enabled() {
		t.Error("Disable() did not disable color")
	}
}
