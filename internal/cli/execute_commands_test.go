package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestExecuteShowsHelp(t *testing.T) {
	var stdout bytes.Buffer

	code := NewApp(Dependencies{}, &stdout, &bytes.Buffer{}).Execute(context.Background(), nil)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "adp run <agent>") {
		t.Fatalf("help output missing run usage: %q", stdout.String())
	}
}

func TestExecuteReportsUnknownCommand(t *testing.T) {
	var stderr bytes.Buffer

	code := NewApp(Dependencies{}, &bytes.Buffer{}, &stderr).Execute(context.Background(), []string{"bogus"})

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), `adp: unknown command "bogus"`) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
