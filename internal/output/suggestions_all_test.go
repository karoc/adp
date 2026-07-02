package output

import (
	"strings"
	"testing"
)

// TestGenerateSuggestionCoversAllCodes asserts that every diagnostic code
// handled by GenerateSuggestion returns a non-nil, non-empty suggestion, and
// that context-dependent codes actually interpolate the provided context
// values (path, workspace, config path, etc.) into their command strings.
//
// This locks the operator-facing guidance contract: a diagnostic code that is
// emitted elsewhere in the codebase must always resolve to actionable advice
// here, and the advice must reference the concrete resource the operator needs
// to inspect.
func TestGenerateSuggestionCoversAllCodes(t *testing.T) {
	// ctx carries a distinct sentinel for each field so we can assert the
	// right field flows into the right code's commands.
	ctx := SuggestionContext{
		Workspace:     "game-a",
		WorkspaceDir:  "/home/u/.adp/workspaces/game-a",
		ConfigPath:    "/home/u/.adp/workspaces/game-a/workspace.yaml",
		Path:          "/tmp/resource/path",
		ProjectRoot:   "/tmp/project/root",
		AgentCommand:  "my-agent-bin",
		ProfileName:   "senior",
		AgentName:     "codex",
		ExpectedValue: "expected-x",
		ActualValue:   "actual-y",
	}

	cases := []struct {
		code string
		// mustContain lists substrings that must appear somewhere in the
		// combined Reason+Commands+Notes text (context interpolation check).
		mustContain []string
	}{
		{code: "workspace.name.invalid", mustContain: []string{"game-a"}},
		{code: "workspace.name.mismatch", mustContain: []string{ctx.ConfigPath}},
		{code: "workspace.dir.missing", mustContain: []string{ctx.WorkspaceDir, "game-a"}},
		{code: "workspace.dir.symlink", mustContain: []string{ctx.WorkspaceDir, "game-a"}},
		{code: "workspace.dir.not_directory", mustContain: []string{ctx.WorkspaceDir, "game-a"}},
		{code: "workspace.config.missing", mustContain: []string{"game-a"}},
		{code: "workspace.config.invalid", mustContain: []string{ctx.ConfigPath}},
		{code: "workspace.project.root.missing", mustContain: []string{ctx.Path, "game-a"}},
		{code: "workspace.project.root.not_directory", mustContain: []string{ctx.Path, "game-a"}},
		{code: "workspace.project.reserved_path.present", mustContain: []string{ctx.Path}},
		{code: "workspace.runtime.parent.missing", mustContain: []string{"ADP_RUNTIME_DIR"}},
		{code: "workspace.runtime.parent.inside_project_root", mustContain: []string{"ADP_RUNTIME_DIR"}},
		{code: "workspace.runtime.parent.contains_project_root", mustContain: []string{"ADP_RUNTIME_DIR"}},
		{code: "workspace.runtime.parent.project_root", mustContain: []string{"ADP_RUNTIME_DIR"}},
		{code: "workspace.runtime.parent.root", mustContain: []string{"ADP_RUNTIME_DIR"}},
		{code: "workspace.runtime.parent.not_directory", mustContain: []string{ctx.Path}},
		{code: "workspace.runtime.parent.symlink", mustContain: []string{"符号链接"}},
		{code: "workspace.prompt.missing", mustContain: []string{"game-a"}},
		{code: "workspace.prompt.outside_workspace", mustContain: []string{ctx.ConfigPath}},
		{code: "workspace.prompt.not_file", mustContain: []string{ctx.Path}},
		{code: "workspace.memory.shared.not_configured", mustContain: []string{"memory.shared"}},
		{code: "workspace.memory.shared.missing", mustContain: []string{"memory"}},
		{code: "workspace.memory.shared.outside_workspace", mustContain: []string{ctx.ConfigPath}},
		{code: "workspace.memory.shared.not_file", mustContain: []string{ctx.Path}},
		{code: "workspace.mcp.config.not_configured", mustContain: []string{"mcp.config"}},
		{code: "workspace.mcp.config.missing", mustContain: []string{"mcp"}},
		{code: "workspace.mcp.config.outside_workspace", mustContain: []string{ctx.ConfigPath}},
		{code: "workspace.mcp.config.not_file", mustContain: []string{ctx.Path}},
		{code: "workspace.agent.unknown", mustContain: []string{ctx.ConfigPath}},
		{code: "workspace.agent.command.default", mustContain: []string{"默认命令"}},
		{code: "workspace.agent.command.missing", mustContain: []string{ctx.AgentCommand}},
		{code: "workspace.agent.command.not_executable", mustContain: []string{ctx.Path}},
		{code: "workspace.agent.profile.invalid", mustContain: []string{ctx.ConfigPath}},
		{code: "workspace.agent.profile.outside_workspace", mustContain: []string{ctx.ConfigPath}},
		{code: "workspace.agent.profile.missing", mustContain: []string{ctx.ProfileName}},
		{code: "workspace.agent.profile.not_file", mustContain: []string{ctx.Path}},
		{code: "workspace.git.env.repository_directive", mustContain: []string{"GIT_DIR"}},
		{code: "workspace.git.root.absent", mustContain: []string{"Git"}},
		{code: "workspace.git.root.detected", mustContain: []string{"Git"}},
		{code: "workspace.git.root.nested_project", mustContain: []string{"子目录"}},
		{code: "workspace.git.metadata.file", mustContain: []string{".git"}},
		{code: "workspace.git.metadata.other", mustContain: []string{ctx.Path}},
		{code: "workspace.git.status.dirty", mustContain: []string{ctx.ProjectRoot}},
		{code: "workspace.git.status.unavailable", mustContain: []string{ctx.ProjectRoot}},
	}

	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			got := GenerateSuggestion(tc.code, ctx)
			if got == nil {
				t.Fatalf("code %q returned nil suggestion", tc.code)
			}
			// Every suggestion must carry at least one actionable field.
			if got.Reason == "" && len(got.Commands) == 0 && len(got.Notes) == 0 && got.DocLink == "" {
				t.Fatalf("code %q returned an empty suggestion", tc.code)
			}
			blob := got.Reason + "\n" + strings.Join(got.Commands, "\n") + "\n" + strings.Join(got.Notes, "\n")
			for _, want := range tc.mustContain {
				if !strings.Contains(blob, want) {
					t.Fatalf("code %q suggestion missing %q\n---\n%s", tc.code, want, blob)
				}
			}
		})
	}
}

// TestGenerateSuggestionUnknownCode confirms unrecognized codes yield nil so
// callers can fall back to their default messaging.
func TestGenerateSuggestionUnknownCode(t *testing.T) {
	if got := GenerateSuggestion("does.not.exist", SuggestionContext{}); got != nil {
		t.Fatalf("unknown code = %+v, want nil", got)
	}
	if got := GenerateSuggestion("", SuggestionContext{}); got != nil {
		t.Fatalf("empty code = %+v, want nil", got)
	}
}
