package output

import (
	"fmt"
)

// DiagnosticSuggestion 包含诊断的建议信息
type DiagnosticSuggestion struct {
	// Reason 简短的原因说明（可选）
	Reason string

	// Commands 下一步命令列表
	Commands []string

	// DocLink 文档链接（相对于 docs/ 目录）
	DocLink string

	// Notes 额外的文本说明
	Notes []string
}

// SuggestionContext 提供生成建议所需的上下文信息
type SuggestionContext struct {
	Workspace     string
	WorkspaceDir  string
	ConfigPath    string
	Path          string
	ProjectRoot   string
	AgentCommand  string
	ProfileName   string
	AgentName     string
	ExpectedValue string
	ActualValue   string
}

// GenerateSuggestion 根据诊断代码和上下文生成建议
func GenerateSuggestion(code string, ctx SuggestionContext) *DiagnosticSuggestion {
	switch code {

	// === 工作区配置类 ===
	case "workspace.name.invalid":
		return &DiagnosticSuggestion{
			Reason: "工作区名称不符合规范",
			Commands: []string{
				fmt.Sprintf("adp workspace remove %s", ctx.Workspace),
				"使用有效名称重新添加工作区（只能包含字母、数字、连字符和下划线）",
			},
			DocLink: "operator-onboarding.md#工作区设置",
		}

	case "workspace.name.mismatch":
		return &DiagnosticSuggestion{
			Reason: "配置文件中的名称与目录名不匹配",
			Commands: []string{
				fmt.Sprintf("检查配置文件: cat %s", ctx.ConfigPath),
				"修改配置文件中的 workspace.name 字段，或重命名工作区目录",
			},
			DocLink: "operator-onboarding.md#工作区配置",
		}

	case "workspace.dir.missing":
		return &DiagnosticSuggestion{
			Commands: []string{
				fmt.Sprintf("检查目录是否存在: ls -ld %s", ctx.WorkspaceDir),
				fmt.Sprintf("重新创建工作区: adp workspace add %s /path/to/project", ctx.Workspace),
			},
		}

	case "workspace.dir.symlink":
		return &DiagnosticSuggestion{
			Reason: "工作区目录不能是符号链接",
			Commands: []string{
				fmt.Sprintf("检查链接目标: ls -l %s", ctx.WorkspaceDir),
				fmt.Sprintf("adp workspace remove %s", ctx.Workspace),
				"使用真实路径重新添加工作区",
			},
		}

	case "workspace.dir.not_directory":
		return &DiagnosticSuggestion{
			Reason: "工作区路径必须是目录",
			Commands: []string{
				fmt.Sprintf("检查文件类型: file %s", ctx.WorkspaceDir),
				fmt.Sprintf("adp workspace remove %s", ctx.Workspace),
				"删除该文件后重新添加工作区",
			},
		}

	case "workspace.config.missing":
		return &DiagnosticSuggestion{
			Commands: []string{
				fmt.Sprintf("adp workspace add %s /path/to/project", ctx.Workspace),
			},
			DocLink: "operator-onboarding.md#添加工作区",
		}

	case "workspace.config.invalid":
		return &DiagnosticSuggestion{
			Commands: []string{
				fmt.Sprintf("检查配置语法: cat %s", ctx.ConfigPath),
				"修复 YAML 语法错误，或删除后重新生成",
			},
			DocLink: "operator-onboarding.md#工作区配置",
		}

	// === 项目根目录类 ===
	case "workspace.project.root.missing":
		return &DiagnosticSuggestion{
			Commands: []string{
				fmt.Sprintf("检查路径是否存在: ls -ld %s", ctx.Path),
				fmt.Sprintf("如果项目已移动: adp workspace remove %s && adp workspace add %s /new/path", ctx.Workspace, ctx.Workspace),
			},
			DocLink: "troubleshooting.zh-CN.md#project-root-does-not-exist",
		}

	case "workspace.project.root.not_directory":
		return &DiagnosticSuggestion{
			Reason: "项目根目录必须是目录",
			Commands: []string{
				fmt.Sprintf("检查文件类型: file %s", ctx.Path),
				fmt.Sprintf("adp workspace remove %s && adp workspace add %s /correct/path", ctx.Workspace, ctx.Workspace),
			},
		}

	case "workspace.project.reserved_path.present":
		return &DiagnosticSuggestion{
			Reason: "项目包含 ADP 保留路径，可能与运行时生成的文件冲突",
			Commands: []string{
				fmt.Sprintf("检查文件: ls -la %s", ctx.Path),
				fmt.Sprintf("移除或重命名: mv %s %s.bak", ctx.Path, ctx.Path),
			},
			Notes: []string{
				"保留路径包括: AGENTS.md, CLAUDE.md, tasks.yaml, phases.yaml 等",
			},
		}

	// === 运行时目录类 ===
	case "workspace.runtime.parent.missing":
		return &DiagnosticSuggestion{
			Commands: []string{
				`export ADP_RUNTIME_DIR="/tmp/adp-runtime"`,
			},
			Notes: []string{
				"将此行添加到 ~/.bashrc 或 ~/.zshrc 使其持久化",
			},
			DocLink: "troubleshooting.zh-CN.md#failed-to-build-runtime",
		}

	case "workspace.runtime.parent.inside_project_root":
		return &DiagnosticSuggestion{
			Reason: "运行时目录必须在项目外，以避免污染真实项目文件和 Git 状态",
			Commands: []string{
				`export ADP_RUNTIME_DIR="/tmp/adp-runtime"`,
			},
			Notes: []string{
				"推荐使用 /tmp 或其他临时目录",
			},
		}

	case "workspace.runtime.parent.contains_project_root":
		return &DiagnosticSuggestion{
			Reason: "运行时父目录不能包含项目根目录",
			Commands: []string{
				`export ADP_RUNTIME_DIR="/tmp/adp-runtime"`,
			},
		}

	case "workspace.runtime.parent.project_root":
		return &DiagnosticSuggestion{
			Reason: "运行时目录不能与项目根目录相同",
			Commands: []string{
				`export ADP_RUNTIME_DIR="/tmp/adp-runtime"`,
			},
		}

	case "workspace.runtime.parent.root":
		return &DiagnosticSuggestion{
			Reason: "运行时目录不能是文件系统根目录",
			Commands: []string{
				`export ADP_RUNTIME_DIR="/tmp/adp-runtime"`,
			},
		}

	case "workspace.runtime.parent.not_directory":
		return &DiagnosticSuggestion{
			Reason: "运行时父目录必须是目录",
			Commands: []string{
				fmt.Sprintf("检查文件类型: file %s", ctx.Path),
				"删除该文件后设置正确的运行时目录",
			},
		}

	case "workspace.runtime.parent.symlink":
		return &DiagnosticSuggestion{
			Reason: "建议使用直接目录而非符号链接",
			Notes: []string{
				"符号链接可以使用，但直接目录路径更清晰",
			},
		}

	// === 文件引用类 ===
	case "workspace.prompt.missing":
		return &DiagnosticSuggestion{
			Commands: []string{
				fmt.Sprintf("快速开始: adp quickstart %s", ctx.Workspace),
				"或手动创建: mkdir -p prompts && vim prompts/base.md",
			},
			DocLink: "operator-onboarding.md#提示文件",
		}

	case "workspace.prompt.outside_workspace":
		return &DiagnosticSuggestion{
			Reason: "提示文件路径必须在工作区目录内",
			Commands: []string{
				fmt.Sprintf("检查配置: cat %s", ctx.ConfigPath),
				"修改配置文件中的 prompts.base 为相对路径",
			},
		}

	case "workspace.prompt.not_file":
		return &DiagnosticSuggestion{
			Reason: "提示路径必须指向文件而非目录",
			Commands: []string{
				fmt.Sprintf("检查路径: ls -ld %s", ctx.Path),
				"修改配置文件中的 prompts.base 为文件路径",
			},
		}

	case "workspace.memory.shared.not_configured":
		return &DiagnosticSuggestion{
			Reason: "共享内存已启用但未配置路径",
			Commands: []string{
				"在 workspace.yaml 中配置 memory.shared: \"memory/shared.md\"",
				"或禁用: memory.enabled: false",
			},
		}

	case "workspace.memory.shared.missing":
		return &DiagnosticSuggestion{
			Commands: []string{
				"创建文件: mkdir -p memory && vim memory/shared.md",
				"或禁用: 在 workspace.yaml 中设置 memory.enabled: false",
			},
		}

	case "workspace.memory.shared.outside_workspace":
		return &DiagnosticSuggestion{
			Reason: "共享内存文件必须在工作区目录内",
			Commands: []string{
				fmt.Sprintf("检查配置: cat %s", ctx.ConfigPath),
				"修改配置文件中的 memory.shared 为相对路径",
			},
		}

	case "workspace.memory.shared.not_file":
		return &DiagnosticSuggestion{
			Reason: "共享内存路径必须指向文件而非目录",
			Commands: []string{
				fmt.Sprintf("检查路径: ls -ld %s", ctx.Path),
				"修改配置文件中的 memory.shared 为文件路径",
			},
		}

	case "workspace.mcp.config.not_configured":
		return &DiagnosticSuggestion{
			Reason: "MCP 已启用但未配置路径",
			Commands: []string{
				"在 workspace.yaml 中配置 mcp.config: \"mcp-config.json\"",
				"或禁用: mcp.enabled: false",
			},
		}

	case "workspace.mcp.config.missing":
		return &DiagnosticSuggestion{
			Commands: []string{
				"创建配置: vim mcp-config.json",
				"或禁用: 在 workspace.yaml 中设置 mcp.enabled: false",
			},
			DocLink: "operator-onboarding.md#mcp-配置",
		}

	case "workspace.mcp.config.outside_workspace":
		return &DiagnosticSuggestion{
			Reason: "MCP 配置文件必须在工作区目录内",
			Commands: []string{
				fmt.Sprintf("检查配置: cat %s", ctx.ConfigPath),
				"修改配置文件中的 mcp.config 为相对路径",
			},
		}

	case "workspace.mcp.config.not_file":
		return &DiagnosticSuggestion{
			Reason: "MCP 配置路径必须指向文件而非目录",
			Commands: []string{
				fmt.Sprintf("检查路径: ls -ld %s", ctx.Path),
				"修改配置文件中的 mcp.config 为文件路径",
			},
		}

	// === Agent 配置类 ===
	case "workspace.agent.unknown":
		return &DiagnosticSuggestion{
			Reason: "配置中引用了未知的 agent",
			Commands: []string{
				fmt.Sprintf("检查配置: cat %s", ctx.ConfigPath),
				"移除未使用的 agent 配置，或安装对应的 agent",
			},
		}

	case "workspace.agent.command.default":
		return &DiagnosticSuggestion{
			Notes: []string{
				"这是信息提示：agent 使用内置的默认命令",
			},
		}

	case "workspace.agent.command.missing":
		return &DiagnosticSuggestion{
			Commands: []string{
				fmt.Sprintf("检查命令: which %s", ctx.AgentCommand),
				"安装对应的 agent 或修改配置文件中的命令路径",
			},
		}

	case "workspace.agent.command.not_executable":
		return &DiagnosticSuggestion{
			Reason: "agent 命令文件不可执行",
			Commands: []string{
				fmt.Sprintf("添加执行权限: chmod +x %s", ctx.Path),
				fmt.Sprintf("检查文件: ls -l %s", ctx.Path),
			},
		}

	case "workspace.agent.profile.invalid":
		return &DiagnosticSuggestion{
			Reason: "profile 路径格式无效",
			Commands: []string{
				fmt.Sprintf("检查配置: cat %s", ctx.ConfigPath),
				"修改为有效的 profile 路径（如 \"profiles/expert.md\"）",
			},
		}

	case "workspace.agent.profile.outside_workspace":
		return &DiagnosticSuggestion{
			Reason: "profile 文件必须在工作区目录内",
			Commands: []string{
				fmt.Sprintf("检查配置: cat %s", ctx.ConfigPath),
				"修改配置文件中的 profile 为相对路径",
			},
		}

	case "workspace.agent.profile.missing":
		return &DiagnosticSuggestion{
			Commands: []string{
				fmt.Sprintf("创建 profile: vim profiles/%s.md", ctx.ProfileName),
				"或使用默认: 在 workspace.yaml 中移除 profile 配置",
			},
		}

	case "workspace.agent.profile.not_file":
		return &DiagnosticSuggestion{
			Reason: "profile 路径必须指向文件而非目录",
			Commands: []string{
				fmt.Sprintf("检查路径: ls -ld %s", ctx.Path),
				"修改配置文件中的 profile 为文件路径",
			},
		}

	// === Git 状态类 ===
	case "workspace.git.env.repository_directive":
		return &DiagnosticSuggestion{
			Reason: "检测到 Git 环境变量 (GIT_DIR, GIT_WORK_TREE 等)",
			Notes: []string{
				"ADP 会自动中和这些变量，无需手动操作",
				"如果仍有问题，可以手动 unset GIT_DIR GIT_WORK_TREE",
			},
		}

	case "workspace.git.root.absent":
		return &DiagnosticSuggestion{
			Reason: "项目根目录不在 Git 仓库中",
			Notes: []string{
				"这是提示信息，不影响 ADP 运行",
				"但 phase evidence 和 agent handoff 在 Git 仓库中更易于审计",
			},
		}

	case "workspace.git.root.detected":
		return &DiagnosticSuggestion{
			Notes: []string{
				"这是正常状态，ADP 将使用 Git 上下文",
			},
		}

	case "workspace.git.root.nested_project":
		return &DiagnosticSuggestion{
			Notes: []string{
				"项目在仓库子目录中，ADP 会正确处理",
			},
		}

	case "workspace.git.metadata.file":
		return &DiagnosticSuggestion{
			Notes: []string{
				"检测到 .git 文件（worktree 或 submodule）",
				"ADP 会排除此元数据，Git 命令应显式指定 ADP_PROJECT_ROOT",
			},
		}

	case "workspace.git.metadata.other":
		return &DiagnosticSuggestion{
			Reason: "Git 元数据路径存在但不是标准的 .git 目录或文件",
			Commands: []string{
				fmt.Sprintf("检查元数据: ls -la %s", ctx.Path),
				"在依赖 Git 状态前检查项目根目录",
			},
		}

	case "workspace.git.status.dirty":
		return &DiagnosticSuggestion{
			Reason: "项目有未提交的更改",
			Commands: []string{
				fmt.Sprintf("查看状态: git -C %s status", ctx.ProjectRoot),
				fmt.Sprintf("查看差异: git -C %s diff", ctx.ProjectRoot),
			},
			Notes: []string{
				"这是提示信息，不影响 ADP 运行",
			},
		}

	case "workspace.git.status.unavailable":
		return &DiagnosticSuggestion{
			Reason: "检测到 Git 仓库但无法读取状态",
			Commands: []string{
				fmt.Sprintf("尝试手动检查: git -C %s status", ctx.ProjectRoot),
			},
		}

	default:
		return nil
	}
}
