package shell

import (
	"errors"
	"testing"

	"github.com/karoc/adp/internal/adapters"
)

func TestRenderExportsOutputsADPEnvInStableOrder(t *testing.T) {
	t.Parallel()

	got, err := RenderExports(adapters.RuntimeHandle{
		Env: map[string]string{
			"PATH":                    "/usr/bin",
			"ADP_WORKSPACE":           "game-a",
			"ADP_SESSION_ID":          "session-1",
			"ADP_HOME":                "/tmp/adp",
			"ADP_PROJECT_ROOT":        "/repo",
			"ADP_RUNTIME_ROOT":        "/tmp/adp/runtime",
			"GIT_CEILING_DIRECTORIES": "/tmp/adp/runtime",
		},
	}, ExportOptions{})
	if err != nil {
		t.Fatalf("RenderExports returned error: %v", err)
	}

	want := "" +
		"unset GIT_ALTERNATE_OBJECT_DIRECTORIES\n" +
		"unset GIT_COMMON_DIR\n" +
		"unset GIT_DIR\n" +
		"unset GIT_INDEX_FILE\n" +
		"unset GIT_NAMESPACE\n" +
		"unset GIT_OBJECT_DIRECTORY\n" +
		"unset GIT_WORK_TREE\n" +
		"export ADP_HOME='/tmp/adp'\n" +
		"export ADP_PROJECT_ROOT='/repo'\n" +
		"export ADP_RUNTIME_ROOT='/tmp/adp/runtime'\n" +
		"export ADP_SESSION_ID='session-1'\n" +
		"export ADP_WORKSPACE='game-a'\n" +
		"export GIT_CEILING_DIRECTORIES='/tmp/adp/runtime'\n"
	if got != want {
		t.Fatalf("RenderExports() = %q, want %q", got, want)
	}
}

func TestRenderExportsQuotesShellValues(t *testing.T) {
	t.Parallel()

	got, err := RenderExports(adapters.RuntimeHandle{
		Env: map[string]string{
			"ADP_HOME":         "/tmp/adp dir",
			"ADP_PROJECT_ROOT": "/repo/it's here; rm -rf /",
			"ADP_RUNTIME_ROOT": "",
			"ADP_SESSION_ID":   "$(whoami)",
		},
	}, ExportOptions{})
	if err != nil {
		t.Fatalf("RenderExports returned error: %v", err)
	}

	want := "" +
		"unset GIT_ALTERNATE_OBJECT_DIRECTORIES\n" +
		"unset GIT_COMMON_DIR\n" +
		"unset GIT_DIR\n" +
		"unset GIT_INDEX_FILE\n" +
		"unset GIT_NAMESPACE\n" +
		"unset GIT_OBJECT_DIRECTORY\n" +
		"unset GIT_WORK_TREE\n" +
		"export ADP_HOME='/tmp/adp dir'\n" +
		"export ADP_PROJECT_ROOT='/repo/it'\\''s here; rm -rf /'\n" +
		"export ADP_RUNTIME_ROOT=''\n" +
		"export ADP_SESSION_ID='$(whoami)'\n"
	if got != want {
		t.Fatalf("RenderExports() = %q, want %q", got, want)
	}
}

func TestRenderExportsCanChangeDirectory(t *testing.T) {
	t.Parallel()

	got, err := RenderExports(adapters.RuntimeHandle{
		Root: "/tmp/runtime dir/it's safe",
		Env: map[string]string{
			"ADP_WORKSPACE": "game-a",
		},
	}, ExportOptions{ChangeDir: true})
	if err != nil {
		t.Fatalf("RenderExports returned error: %v", err)
	}

	want := "" +
		"unset GIT_ALTERNATE_OBJECT_DIRECTORIES\n" +
		"unset GIT_COMMON_DIR\n" +
		"unset GIT_DIR\n" +
		"unset GIT_INDEX_FILE\n" +
		"unset GIT_NAMESPACE\n" +
		"unset GIT_OBJECT_DIRECTORY\n" +
		"unset GIT_WORK_TREE\n" +
		"export ADP_WORKSPACE='game-a'\n" +
		"cd '/tmp/runtime dir/it'\\''s safe'\n"
	if got != want {
		t.Fatalf("RenderExports() = %q, want %q", got, want)
	}
}

func TestRenderExportsRejectsMissingRuntimeRootWhenChangingDirectory(t *testing.T) {
	t.Parallel()

	_, err := RenderExports(adapters.RuntimeHandle{}, ExportOptions{ChangeDir: true})
	if !errors.Is(err, ErrRuntimeRootRequired) {
		t.Fatalf("error = %v, want ErrRuntimeRootRequired", err)
	}
}

func TestRenderExportsRejectsInvalidADPExportName(t *testing.T) {
	t.Parallel()

	_, err := RenderExports(adapters.RuntimeHandle{
		Env: map[string]string{
			"ADP_BAD-NAME": "value",
		},
	}, ExportOptions{})
	if !errors.Is(err, ErrInvalidExportName) {
		t.Fatalf("error = %v, want ErrInvalidExportName", err)
	}
}
