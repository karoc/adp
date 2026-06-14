package output

import (
	"testing"
)

func TestGenerateSuggestion(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		ctx      SuggestionContext
		wantNil  bool
		checkFn  func(*testing.T, *DiagnosticSuggestion)
	}{
		{
			name: "workspace.config.missing",
			code: "workspace.config.missing",
			ctx: SuggestionContext{
				Workspace: "test-ws",
			},
			wantNil: false,
			checkFn: func(t *testing.T, s *DiagnosticSuggestion) {
				if len(s.Commands) == 0 {
					t.Error("expected commands")
				}
				if s.DocLink == "" {
					t.Error("expected doc link")
				}
			},
		},
		{
			name: "workspace.project.root.missing",
			code: "workspace.project.root.missing",
			ctx: SuggestionContext{
				Workspace: "test-ws",
				Path:      "/tmp/missing",
			},
			wantNil: false,
			checkFn: func(t *testing.T, s *DiagnosticSuggestion) {
				if len(s.Commands) != 2 {
					t.Errorf("expected 2 commands, got %d", len(s.Commands))
				}
				if s.DocLink == "" {
					t.Error("expected doc link")
				}
			},
		},
		{
			name: "workspace.runtime.parent.missing",
			code: "workspace.runtime.parent.missing",
			ctx:  SuggestionContext{},
			wantNil: false,
			checkFn: func(t *testing.T, s *DiagnosticSuggestion) {
				if len(s.Commands) == 0 {
					t.Error("expected commands")
				}
				if len(s.Notes) == 0 {
					t.Error("expected notes")
				}
				if s.DocLink == "" {
					t.Error("expected doc link")
				}
			},
		},
		{
			name: "workspace.runtime.parent.inside_project_root",
			code: "workspace.runtime.parent.inside_project_root",
			ctx:  SuggestionContext{},
			wantNil: false,
			checkFn: func(t *testing.T, s *DiagnosticSuggestion) {
				if s.Reason == "" {
					t.Error("expected reason")
				}
				if len(s.Commands) == 0 {
					t.Error("expected commands")
				}
				if len(s.Notes) == 0 {
					t.Error("expected notes")
				}
			},
		},
		{
			name: "workspace.prompt.missing",
			code: "workspace.prompt.missing",
			ctx: SuggestionContext{
				Workspace: "test-ws",
			},
			wantNil: false,
			checkFn: func(t *testing.T, s *DiagnosticSuggestion) {
				if len(s.Commands) != 2 {
					t.Errorf("expected 2 commands, got %d", len(s.Commands))
				}
				if s.DocLink == "" {
					t.Error("expected doc link")
				}
			},
		},
		{
			name: "workspace.git.status.dirty",
			code: "workspace.git.status.dirty",
			ctx: SuggestionContext{
				ProjectRoot: "/tmp/project",
			},
			wantNil: false,
			checkFn: func(t *testing.T, s *DiagnosticSuggestion) {
				if s.Reason == "" {
					t.Error("expected reason")
				}
				if len(s.Commands) != 2 {
					t.Errorf("expected 2 commands, got %d", len(s.Commands))
				}
				if len(s.Notes) == 0 {
					t.Error("expected notes")
				}
			},
		},
		{
			name: "workspace.git.root.detected",
			code: "workspace.git.root.detected",
			ctx:  SuggestionContext{},
			wantNil: false,
			checkFn: func(t *testing.T, s *DiagnosticSuggestion) {
				if len(s.Notes) == 0 {
					t.Error("expected notes")
				}
			},
		},
		{
			name: "workspace.agent.command.missing",
			code: "workspace.agent.command.missing",
			ctx: SuggestionContext{
				AgentCommand: "my-agent",
			},
			wantNil: false,
			checkFn: func(t *testing.T, s *DiagnosticSuggestion) {
				if len(s.Commands) != 2 {
					t.Errorf("expected 2 commands, got %d", len(s.Commands))
				}
			},
		},
		{
			name: "workspace.memory.shared.missing",
			code: "workspace.memory.shared.missing",
			ctx:  SuggestionContext{},
			wantNil: false,
			checkFn: func(t *testing.T, s *DiagnosticSuggestion) {
				if len(s.Commands) != 2 {
					t.Errorf("expected 2 commands, got %d", len(s.Commands))
				}
			},
		},
		{
			name: "unknown code",
			code: "unknown.code",
			ctx:  SuggestionContext{},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateSuggestion(tt.code, tt.ctx)
			if tt.wantNil {
				if got != nil {
					t.Errorf("expected nil suggestion for unknown code, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil suggestion")
			}
			if tt.checkFn != nil {
				tt.checkFn(t, got)
			}
		})
	}
}

func TestSuggestionContext(t *testing.T) {
	ctx := SuggestionContext{
		Workspace:    "test",
		WorkspaceDir: "/home/user/.adp/workspaces/test",
		ConfigPath:   "/home/user/.adp/workspaces/test/workspace.yaml",
		Path:         "/tmp/test",
		ProjectRoot:  "/tmp/project",
	}

	suggestion := GenerateSuggestion("workspace.project.root.missing", ctx)
	if suggestion == nil {
		t.Fatal("expected non-nil suggestion")
	}

	// 检查命令是否包含上下文信息（路径或工作区名称）
	if len(suggestion.Commands) < 2 {
		t.Fatalf("expected at least 2 commands, got %d", len(suggestion.Commands))
	}

	// 检查第一个命令是否包含路径
	firstCmd := suggestion.Commands[0]
	if !contains(firstCmd, ctx.Path) {
		t.Errorf("expected first command to contain path %s, got: %s", ctx.Path, firstCmd)
	}

	// 检查第二个命令是否包含工作区名称
	secondCmd := suggestion.Commands[1]
	if !contains(secondCmd, ctx.Workspace) {
		t.Errorf("expected second command to contain workspace %s, got: %s", ctx.Workspace, secondCmd)
	}
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) && hasSubstring(s, substr))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
