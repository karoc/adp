package cli

import (
	"bytes"
	"context"
	"reflect"
	"sort"
	"testing"

	"github.com/karoc/adp/internal/commandmeta"
)

func TestCommandMetadataMatchesDispatch(t *testing.T) {
	t.Parallel()

	app := NewApp(Dependencies{}, &bytes.Buffer{}, &bytes.Buffer{})
	got := make([]string, 0, len(app.commandHandlers()))
	for command := range app.commandHandlers() {
		got = append(got, command)
	}
	sort.Strings(got)

	want := append([]string(nil), commandmeta.RootCommandNames()...)
	sort.Strings(want)

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("dispatch commands = %v, want metadata commands %v", got, want)
	}
}

func TestHelpUsageMatchesCommandMetadata(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	code := NewApp(Dependencies{}, &stdout, &bytes.Buffer{}).Execute(context.Background(), []string{"--help"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if stdout.String() != commandmeta.Usage() {
		t.Fatalf("help output drifted from command metadata:\n%s", stdout.String())
	}
}
