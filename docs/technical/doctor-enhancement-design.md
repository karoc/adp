# Doctor 命令输出增强设计

## 1. 当前分析

### 1.1 诊断架构

Doctor 命令采用模块化诊断架构：

- **入口**: `internal/cli/workspace_commands.go:doctor()`
- **核心逻辑**: `internal/workspace/diagnostics.go`
- **专项诊断模块**:
  - `config_diagnostics.go` - 配置文件验证
  - `agent_diagnostics.go` - Agent 配置检查
  - `git_state_diagnostics.go` - Git 状态检查
  - `runtime_diagnostics.go` - 运行时目录验证
  - `reserved_diagnostics.go` - 保留路径冲突检测

### 1.2 诊断代码分类

系统定义了 **35+ 诊断代码**，分为以下类别：

**A. 工作区配置类** (7 个代码)
- `workspace.name.*` - 名称验证
- `workspace.dir.*` - 目录完整性
- `workspace.config.*` - 配置加载

**B. 项目根目录类** (4 个代码)
- `workspace.project.root.*` - 根目录存在性和类型
- `workspace.project.reserved_path.*` - 保留路径冲突

**C. 文件引用类** (12 个代码)
- `workspace.prompt.*` - 提示文件
- `workspace.memory.shared.*` - 共享内存文件
- `workspace.mcp.config.*` - MCP 配置文件

**D. Agent 配置类** (9 个代码)
- `workspace.agent.unknown` - 未知 agent
- `workspace.agent.command.*` - 命令路径验证
- `workspace.agent.profile.*` - Profile 文件检查

**E. Git 状态类** (6 个代码)
- `workspace.git.env.*` - Git 环境变量
- `workspace.git.root.*` - Git 仓库检测
- `workspace.git.status.*` - 工作树状态

**F. 运行时目录类** (8 个代码)
- `workspace.runtime.parent.*` - 运行时父目录验证

### 1.3 当前输出格式

**文本模式** (默认):
```
WORKSPACE  LEVEL    CODE                           MESSAGE                              PATH
game-a     error    workspace.project.root.missing project root is missing             /path/to/missing
game-a     warning  workspace.prompt.missing       configured base prompt file is missing  /path/to/prompt.md
```

**特点**:
- 简洁的表格格式
- 按工作区分组
- 隐藏 info 级别（除非 --verbose）
- 无建议或下一步指引

**JSON 模式** (--format json):
- 完整结构化输出
- 包含 Git 上下文
- 机器可解析
- 无人类友好的建议

### 1.4 用户体验问题

当前输出存在以下问题：

1. **无指引**: 错误信息描述问题，但不告诉用户如何修复
2. **无优先级**: 所有错误平等显示，用户不知道先修什么
3. **查找困难**: 用户需要手动去 troubleshooting.md 查找解决方案
4. **上下文缺失**: 某些错误需要额外信息才能理解（如 runtime parent 为什么不能在项目内）

## 2. 错误-建议映射表

### 2.1 高优先级错误 (必须修复才能运行)

| 诊断代码 | 错误描述 | 建议命令 | 文档链接 |
|---------|---------|---------|---------|
| `workspace.config.missing` | 配置文件缺失 | `adp workspace add <name> /path` | operator-onboarding.md#工作区设置 |
| `workspace.config.invalid` | 配置文件无效 | 检查 workspace.yaml 语法 | - |
| `workspace.project.root.missing` | 项目根目录不存在 | `adp workspace remove <name>` 然后重新添加 | troubleshooting.md#project-root-does-not-exist |
| `workspace.runtime.parent.missing` | 运行时目录未配置 | `export ADP_RUNTIME_DIR="/tmp/adp-runtime"` | troubleshooting.md#failed-to-build-runtime |
| `workspace.runtime.parent.inside_project_root` | 运行时在项目内 | 设置 ADP_RUNTIME_DIR 到项目外 | - |

### 2.2 中优先级警告 (影响功能)

| 诊断代码 | 警告描述 | 建议命令 | 文档链接 |
|---------|---------|---------|---------|
| `workspace.prompt.missing` | 提示文件缺失 | 创建 prompts/base.md | operator-onboarding.md#提示文件 |
| `workspace.memory.shared.missing` | 共享内存文件缺失 | 创建 memory/shared.md 或禁用 memory | - |
| `workspace.mcp.config.missing` | MCP 配置缺失 | 创建 mcp-config.json 或禁用 MCP | - |
| `workspace.agent.command.missing` | Agent 命令不存在 | 检查命令路径或安装 agent | - |
| `workspace.project.reserved_path.present` | 项目包含保留路径 | 移除 AGENTS.md / CLAUDE.md 等文件 | - |

### 2.3 信息级提示 (仅供参考)

| 诊断代码 | 信息描述 | 说明 |
|---------|---------|------|
| `workspace.git.root.detected` | Git 仓库检测到 | 正常状态，ADP 将使用 Git 上下文 |
| `workspace.git.root.nested` | 嵌套 Git 仓库 | 项目在仓库子目录，ADP 会正确处理 |
| `workspace.agent.command.default` | 使用默认 agent 命令 | Codex/Claude 有内置默认值 |

### 2.4 特殊场景建议

**场景 1: 工作区首次设置**
- 错误: `workspace.prompt.missing`
- 建议: 运行 `adp quickstart <workspace>` 生成默认结构

**场景 2: Git 环境变量污染**
- 错误: `workspace.git.env.repository_directive`
- 建议: ADP 会自动中和这些变量，无需操作

**场景 3: 运行时目录冲突**
- 错误: `workspace.runtime.parent.contains_project_root`
- 建议: 使用 `/tmp/adp-runtime` 或其他临时目录

## 3. 增强方案设计

### 3.1 输出格式改进

#### 3.1.1 文本模式增强 (默认)

**当前格式**:
```
WORKSPACE  LEVEL    CODE                           MESSAGE                              PATH
game-a     error    workspace.project.root.missing project root is missing             /path/to/missing
```

**增强格式**:
```
✗ Workspace 'game-a' has 2 errors, 1 warning

  [ERROR] Project root does not exist
    Path: /home/user/projects/game-a
    
    下一步:
      1. 检查路径是否正确: ls -ld /home/user/projects/game-a
      2. 如果项目已移动: adp workspace remove game-a && adp workspace add game-a /new/path
      3. 查看详细指南: docs/troubleshooting.md#project-root-does-not-exist

  [ERROR] Runtime parent is inside project root
    Runtime: /home/user/projects/game-a/.adp-runtime
    Project: /home/user/projects/game-a
    
    原因: 运行时必须在项目外，以避免污染真实项目文件
    
    下一步:
      export ADP_RUNTIME_DIR="/tmp/adp-runtime"
      或修改 ~/.bashrc 使其持久化

  [WARNING] Base prompt file is missing
    Expected: /home/user/.adp/workspaces/game-a/prompts/base.md
    
    下一步:
      快速开始: adp quickstart game-a
      或手动创建: mkdir -p prompts && vim prompts/base.md

✓ Workspace 'game-b' is healthy
```

**设计要点**:
- 使用表情符号 (✗/✓) 快速识别状态
- 按工作区分组，每个工作区一个块
- 错误优先，然后警告
- 每个诊断包含：
  - 人类可读的标题
  - 关键路径信息
  - 原因说明（如果不明显）
  - 具体命令建议
  - 文档链接
- 健康的工作区仅显示一行确认

#### 3.1.2 详细模式 (--verbose)

在增强格式基础上添加：
- Info 级别诊断
- Git 上下文详情
- 诊断代码（用于脚本化）
- 完整路径展开

#### 3.1.3 JSON 模式 (--format json)

保持当前结构，添加新字段：

```json
{
  "report_count": 2,
  "has_errors": true,
  "reports": [
    {
      "workspace": "game-a",
      "has_errors": true,
      "diagnostics": [
        {
          "level": "error",
          "code": "workspace.project.root.missing",
          "message": "project root is missing",
          "path": "/path/to/missing",
          "suggestions": {
            "commands": [
              "ls -ld /path/to/missing",
              "adp workspace remove game-a && adp workspace add game-a /new/path"
            ],
            "doc_link": "docs/troubleshooting.md#project-root-does-not-exist"
          }
        }
      ]
    }
  ]
}
```

### 3.2 数据结构设计

#### 3.2.1 建议数据结构

在 `internal/workspace/diagnostics.go` 中添加：

```go
// DiagnosticSuggestion 包含诊断的建议信息
type DiagnosticSuggestion struct {
    // 简短的原因说明（可选）
    Reason string
    
    // 下一步命令列表
    Commands []string
    
    // 文档链接（相对于 docs/ 目录）
    DocLink string
    
    // 额外的文本说明
    Notes []string
}

// Diagnostic 扩展现有结构
type Diagnostic struct {
    Level      DiagnosticLevel
    Code       string
    Message    string
    Path       string
    Suggestion *DiagnosticSuggestion  // 新增字段
}
```

#### 3.2.2 建议生成器

创建 `internal/workspace/suggestions.go`:

```go
package workspace

// GenerateSuggestion 根据诊断代码生成建议
func GenerateSuggestion(code string, ctx SuggestionContext) *DiagnosticSuggestion {
    switch code {
    case DiagnosticCodeProjectRootMissing:
        return &DiagnosticSuggestion{
            Commands: []string{
                fmt.Sprintf("ls -ld %s", ctx.Path),
                fmt.Sprintf("adp workspace remove %s && adp workspace add %s /new/path", ctx.Workspace, ctx.Workspace),
            },
            DocLink: "troubleshooting.md#project-root-does-not-exist",
        }
    
    case DiagnosticCodeRuntimeParentInsideProjectRoot:
        return &DiagnosticSuggestion{
            Reason: "运行时必须在项目外，以避免污染真实项目文件",
            Commands: []string{
                `export ADP_RUNTIME_DIR="/tmp/adp-runtime"`,
            },
            Notes: []string{
                "将此行添加到 ~/.bashrc 使其持久化",
            },
        }
    
    case DiagnosticCodePromptMissing:
        return &DiagnosticSuggestion{
            Commands: []string{
                fmt.Sprintf("adp quickstart %s", ctx.Workspace),
                "mkdir -p prompts && vim prompts/base.md",
            },
            DocLink: "operator-onboarding.md#提示文件",
        }
    
    // ... 其他映射
    
    default:
        return nil
    }
}

// SuggestionContext 提供生成建议所需的上下文
type SuggestionContext struct {
    Workspace   string
    Path        string
    ProjectRoot string
    // ... 其他需要的字段
}
```

### 3.3 渲染层设计

#### 3.3.1 文本渲染器

创建 `internal/cli/doctor_renderer.go`:

```go
package cli

import (
    "fmt"
    "io"
    "strings"
    
    "github.com/karoc/adp/internal/output"
    "github.com/karoc/adp/internal/workspace"
)

// DoctorTextRenderer 渲染 doctor 诊断结果为人类可读格式
type DoctorTextRenderer struct {
    writer  io.Writer
    verbose bool
}

func (r *DoctorTextRenderer) Render(reports []workspace.DiagnosticReport) error {
    for _, report := range reports {
        if err := r.renderReport(report); err != nil {
            return err
        }
    }
    return nil
}

func (r *DoctorTextRenderer) renderReport(report workspace.DiagnosticReport) error {
    // 计算统计
    errorCount := countLevel(report.Diagnostics, workspace.DiagnosticLevelError)
    warningCount := countLevel(report.Diagnostics, workspace.DiagnosticLevelWarning)
    
    // 渲染标题
    if errorCount > 0 || warningCount > 0 {
        fmt.Fprintf(r.writer, "✗ Workspace '%s'", report.Workspace)
        if errorCount > 0 {
            fmt.Fprintf(r.writer, " has %d error(s)", errorCount)
        }
        if warningCount > 0 {
            if errorCount > 0 {
                fmt.Fprint(r.writer, ",")
            }
            fmt.Fprintf(r.writer, " %d warning(s)", warningCount)
        }
        fmt.Fprintln(r.writer)
        fmt.Fprintln(r.writer)
        
        // 渲染诊断
        for _, diag := range report.Diagnostics {
            if !r.verbose && diag.Level == workspace.DiagnosticLevelInfo {
                continue
            }
            if err := r.renderDiagnostic(diag, report); err != nil {
                return err
            }
        }
    } else {
        fmt.Fprintf(r.writer, "✓ Workspace '%s' is healthy\n", report.Workspace)
    }
    
    fmt.Fprintln(r.writer)
    return nil
}

func (r *DoctorTextRenderer) renderDiagnostic(diag workspace.Diagnostic, report workspace.DiagnosticReport) error {
    // 标题行
    levelIcon := map[workspace.DiagnosticLevel]string{
        workspace.DiagnosticLevelError:   "[ERROR]",
        workspace.DiagnosticLevelWarning: "[WARNING]",
        workspace.DiagnosticLevelInfo:    "[INFO]",
    }
    
    fmt.Fprintf(r.writer, "  %s %s\n", levelIcon[diag.Level], diag.Message)
    
    // 路径信息
    if diag.Path != "" {
        fmt.Fprintf(r.writer, "    Path: %s\n", diag.Path)
    }
    
    // 建议信息
    if diag.Suggestion != nil {
        fmt.Fprintln(r.writer)
        
        // 原因
        if diag.Suggestion.Reason != "" {
            fmt.Fprintf(r.writer, "    原因: %s\n", diag.Suggestion.Reason)
            fmt.Fprintln(r.writer)
        }
        
        // 下一步命令
        if len(diag.Suggestion.Commands) > 0 {
            fmt.Fprintln(r.writer, "    下一步:")
            for i, cmd := range diag.Suggestion.Commands {
                if i == 0 && len(diag.Suggestion.Commands) == 1 {
                    fmt.Fprintf(r.writer, "      %s\n", output.Command(cmd))
                } else {
                    fmt.Fprintf(r.writer, "      %d. %s\n", i+1, output.Command(cmd))
                }
            }
        }
        
        // 额外说明
        for _, note := range diag.Suggestion.Notes {
            fmt.Fprintf(r.writer, "      提示: %s\n", note)
        }
        
        // 文档链接
        if diag.Suggestion.DocLink != "" {
            fmt.Fprintf(r.writer, "      文档: docs/%s\n", diag.Suggestion.DocLink)
        }
    }
    
    if r.verbose {
        fmt.Fprintf(r.writer, "    Code: %s\n", diag.Code)
    }
    
    fmt.Fprintln(r.writer)
    return nil
}
```

### 3.4 实施计划

#### 阶段 1: 数据结构扩展 (1-2 小时)

1. **扩展 Diagnostic 结构**
   - 文件: `internal/workspace/diagnostics.go`
   - 添加 `DiagnosticSuggestion` 类型
   - 添加 `Suggestion *DiagnosticSuggestion` 字段到 `Diagnostic`

2. **创建建议生成器**
   - 新文件: `internal/workspace/suggestions.go`
   - 实现 `GenerateSuggestion()` 函数
   - 实现 `SuggestionContext` 结构

3. **更新诊断生成逻辑**
   - 修改 `diagnoseWorkspaceDir()` 函数
   - 在 `report.add()` 调用时生成建议
   - 传递必要的上下文信息

#### 阶段 2: 建议映射实现 (2-3 小时)

为每个诊断代码实现建议生成逻辑（优先级排序）：

**高优先级** (必须实现):
- `workspace.project.root.missing`
- `workspace.config.missing`
- `workspace.runtime.parent.inside_project_root`
- `workspace.runtime.parent.missing`
- `workspace.prompt.missing`

**中优先级**:
- `workspace.memory.shared.missing`
- `workspace.mcp.config.missing`
- `workspace.agent.command.missing`
- `workspace.project.reserved_path.present`

**低优先级** (可选):
- 所有 info 级别诊断
- 其他 warning 级别诊断

#### 阶段 3: 文本渲染器 (2-3 小时)

1. **创建渲染器**
   - 新文件: `internal/cli/doctor_renderer.go`
   - 实现 `DoctorTextRenderer` 类型
   - 实现 `Render()`, `renderReport()`, `renderDiagnostic()` 方法

2. **集成到 CLI**
   - 修改 `internal/cli/workspace_commands.go:workspaceDoctorReports()`
   - 使用新渲染器替代当前的 tabwriter 输出
   - 保持 `--verbose` 和 `--format json` 兼容

3. **中文本地化**
   - 建议文本使用中文
   - 错误消息保持英文（代码层面）
   - 文档链接优先指向中文版

#### 阶段 4: JSON 输出扩展 (1 小时)

1. **扩展 JSON 结构**
   - 文件: `internal/cli/workspace_doctor_json.go`
   - 添加 `suggestions` 字段到 `workspaceDiagnosticJSON`

2. **更新 JSON 转换**
   - 修改 `workspaceDoctorOutput()` 函数
   - 序列化建议信息

#### 阶段 5: 测试和文档 (2 小时)

1. **单元测试**
   - 测试建议生成逻辑
   - 测试渲染器输出格式
   - 测试 JSON 结构

2. **集成测试**
   - 使用真实工作区测试
   - 验证所有诊断代码的建议

3. **更新文档**
   - 更新 operator-onboarding.md
   - 在 troubleshooting.md 中引用 doctor 命令
   - 添加使用示例

## 4. 技术细节

### 4.1 保持向后兼容

**要求**:
- JSON 输出添加字段，不删除现有字段
- `--verbose` 行为保持不变
- 退出代码保持不变 (0=成功, 2=有错误)
- 诊断代码保持不变

**策略**:
- 新增 `Suggestion` 字段为可选 (指针类型)
- 如果生成建议失败，不影响诊断本身
- 保留当前的 tabwriter 实现作为备选

### 4.2 国际化考虑

**当前状态**: 文档有英文和中文双语版本

**建议文本策略**:
- 建议文本硬编码为中文（主要用户群）
- 未来可扩展为 i18n 机制
- 命令本身保持英文
- 文档链接优先中文，fallback 英文

**示例**:
```go
if diag.Suggestion != nil && diag.Suggestion.Reason != "" {
    fmt.Fprintf(r.writer, "    原因: %s\n", diag.Suggestion.Reason)
}
```

### 4.3 性能考虑

**建议生成开销**:
- 每个诊断生成一次建议
- 使用 switch/case 快速匹配
- 避免复杂的字符串处理
- 预期开销 < 1ms per diagnostic

**渲染开销**:
- 仅在文本模式下格式化输出
- JSON 模式保持结构化
- 使用 `io.Writer` 避免字符串拼接

### 4.4 可扩展性

**添加新诊断代码**:
1. 在 `diagnostics.go` 中定义代码常量
2. 在诊断函数中调用 `report.add()`
3. 在 `suggestions.go` 中添加 case 分支
4. 在 troubleshooting.md 中添加文档

**模板化建议**:
```go
type SuggestionTemplate struct {
    Reason     string
    Commands   []string
    DocLink    string
    Notes      []string
}

var suggestionTemplates = map[string]SuggestionTemplate{
    DiagnosticCodeProjectRootMissing: {
        Commands: []string{
            "ls -ld {{.Path}}",
            "adp workspace remove {{.Workspace}} && adp workspace add {{.Workspace}} /new/path",
        },
        DocLink: "troubleshooting.md#project-root-does-not-exist",
    },
    // ...
}
```

## 5. 代码示例

### 5.1 建议生成器完整示例

```go
package workspace

import "fmt"

// GenerateSuggestion 根据诊断代码和上下文生成建议
func GenerateSuggestion(code string, ctx SuggestionContext) *DiagnosticSuggestion {
    switch code {
    
    // === 工作区配置类 ===
    case DiagnosticCodeWorkspaceNameMismatch:
        return &DiagnosticSuggestion{
            Reason: "配置文件中的名称与目录名不匹配",
            Commands: []string{
                fmt.Sprintf("检查配置文件: cat %s", ctx.ConfigPath),
            },
            DocLink: "operator-onboarding.md#工作区配置",
        }
    
    case DiagnosticCodeConfigMissing:
        return &DiagnosticSuggestion{
            Commands: []string{
                fmt.Sprintf("adp workspace add %s /path/to/project", ctx.Workspace),
            },
            DocLink: "operator-onboarding.md#添加工作区",
        }
    
    // === 项目根目录类 ===
    case DiagnosticCodeProjectRootMissing:
        return &DiagnosticSuggestion{
            Commands: []string{
                fmt.Sprintf("检查路径: ls -ld %s", ctx.Path),
                fmt.Sprintf("如果项目已移动: adp workspace remove %s && adp workspace add %s /new/path", ctx.Workspace, ctx.Workspace),
            },
            DocLink: "troubleshooting.zh-CN.md#project-root-does-not-exist",
        }
    
    case DiagnosticCodeProjectRootReservedPath:
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
    case DiagnosticCodeRuntimeParentMissing:
        return &DiagnosticSuggestion{
            Commands: []string{
                `export ADP_RUNTIME_DIR="/tmp/adp-runtime"`,
            },
            Notes: []string{
                "将此行添加到 ~/.bashrc 或 ~/.zshrc 使其持久化",
            },
            DocLink: "troubleshooting.zh-CN.md#failed-to-build-runtime",
        }
    
    case DiagnosticCodeRuntimeParentInsideProjectRoot:
        return &DiagnosticSuggestion{
            Reason: "运行时目录必须在项目外，以避免污染真实项目文件和 Git 状态",
            Commands: []string{
                `export ADP_RUNTIME_DIR="/tmp/adp-runtime"`,
            },
            Notes: []string{
                "推荐使用 /tmp 或其他临时目录",
            },
        }
    
    case DiagnosticCodeRuntimeParentContainsProject:
        return &DiagnosticSuggestion{
            Reason: "运行时父目录不能包含项目根目录",
            Commands: []string{
                `export ADP_RUNTIME_DIR="/tmp/adp-runtime"`,
            },
        }
    
    // === 文件引用类 ===
    case DiagnosticCodePromptMissing:
        return &DiagnosticSuggestion{
            Commands: []string{
                fmt.Sprintf("快速开始: adp quickstart %s", ctx.Workspace),
                "或手动创建: mkdir -p prompts && vim prompts/base.md",
            },
            DocLink: "operator-onboarding.md#提示文件",
        }
    
    case DiagnosticCodeMemorySharedMissing:
        return &DiagnosticSuggestion{
            Commands: []string{
                "创建文件: mkdir -p memory && vim memory/shared.md",
                "或禁用: 在 workspace.yaml 中设置 memory.enabled: false",
            },
        }
    
    case DiagnosticCodeMCPConfigMissing:
        return &DiagnosticSuggestion{
            Commands: []string{
                "创建配置: vim mcp-config.json",
                "或禁用: 在 workspace.yaml 中设置 mcp.enabled: false",
            },
            DocLink: "operator-onboarding.md#mcp-配置",
        }
    
    // === Agent 配置类 ===
    case DiagnosticCodeAgentCommandMissing:
        return &DiagnosticSuggestion{
            Commands: []string{
                fmt.Sprintf("检查命令: which %s", ctx.AgentCommand),
                "或安装 agent: 参考 agent 安装文档",
            },
        }
    
    case DiagnosticCodeAgentProfileMissing:
        return &DiagnosticSuggestion{
            Commands: []string{
                fmt.Sprintf("创建 profile: vim profiles/%s.md", ctx.ProfileName),
                "或使用默认: 在 workspace.yaml 中移除 profile 配置",
            },
        }
    
    // === Git 状态类 ===
    case DiagnosticCodeGitEnvRepositoryDirective:
        return &DiagnosticSuggestion{
            Reason: "检测到 Git 环境变量 (GIT_DIR, GIT_WORK_TREE 等)",
            Notes: []string{
                "ADP 会自动中和这些变量，无需手动操作",
                "如果仍有问题，可以手动 unset GIT_DIR GIT_WORK_TREE",
            },
        }
    
    case DiagnosticCodeGitStatusDirty:
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
    
    default:
        return nil
    }
}

// SuggestionContext 包含生成建议所需的上下文信息
type SuggestionContext struct {
    Workspace    string
    WorkspaceDir string
    ConfigPath   string
    Path         string
    ProjectRoot  string
    AgentCommand string
    ProfileName  string
}
```

### 5.2 诊断生成集成示例

```go
// 在 internal/workspace/diagnostics.go 中修改 report.add()
func (r *DiagnosticReport) add(level DiagnosticLevel, code string, message string, path string) {
    ctx := SuggestionContext{
        Workspace:    r.Workspace,
        WorkspaceDir: r.WorkspaceDir,
        ConfigPath:   r.ConfigPath,
        Path:         path,
        ProjectRoot:  "", // 从 report 的其他字段获取
    }
    
    suggestion := GenerateSuggestion(code, ctx)
    
    r.Diagnostics = append(r.Diagnostics, Diagnostic{
        Level:      level,
        Code:       code,
        Message:    message,
        Path:       path,
        Suggestion: suggestion,
    })
}

// 或者在特定诊断函数中提供更多上下文
func checkProjectRoot(report *DiagnosticReport, root string) {
    info, err := os.Stat(root)
    if err != nil {
        if errors.Is(err, os.ErrNotExist) {
            ctx := SuggestionContext{
                Workspace:   report.Workspace,
                Path:        root,
                ProjectRoot: root,
            }
            suggestion := GenerateSuggestion(DiagnosticCodeProjectRootMissing, ctx)
            report.Diagnostics = append(report.Diagnostics, Diagnostic{
                Level:      DiagnosticLevelError,
                Code:       DiagnosticCodeProjectRootMissing,
                Message:    "project root is missing",
                Path:       root,
                Suggestion: suggestion,
            })
            return
        }
        // ...
    }
}
```

### 5.3 CLI 集成示例

```go
// 在 internal/cli/workspace_commands.go 中修改
func (a *App) workspaceDoctorReports(reports []workspace.DiagnosticReport, opts doctorOptions) error {
    if opts.format == outputFormatJSON {
        return a.workspaceDoctorJSON(reports)
    }

    // 使用新的渲染器
    renderer := &DoctorTextRenderer{
        writer:  a.stdout,
        verbose: opts.verbose,
    }
    
    if err := renderer.Render(reports); err != nil {
        return err
    }
    
    // 保持退出代码逻辑
    hasErrors := false
    for _, report := range reports {
        if report.HasErrors() {
            hasErrors = true
            break
        }
    }
    
    if hasErrors {
        return processExitError{code: 2}
    }
    return nil
}
```

## 6. 输出示例对比

### 6.1 当前输出 vs 增强输出

#### 场景 1: 项目根目录缺失

**当前输出**:
```
WORKSPACE  LEVEL  CODE                            MESSAGE                    PATH
game-a     error  workspace.project.root.missing  project root is missing   /home/user/projects/game-a
```

**增强输出**:
```
✗ Workspace 'game-a' has 1 error

  [ERROR] Project root is missing
    Path: /home/user/projects/game-a
    
    下一步:
      1. 检查路径: ls -ld /home/user/projects/game-a
      2. 如果项目已移动: adp workspace remove game-a && adp workspace add game-a /new/path
      文档: docs/troubleshooting.zh-CN.md#project-root-does-not-exist

```

#### 场景 2: 运行时目录冲突

**当前输出**:
```
WORKSPACE  LEVEL  CODE                                           MESSAGE                                          PATH
game-a     error  workspace.runtime.parent.inside_project_root  runtime parent must not be inside project root  /home/user/projects/game-a/.runtime
```

**增强输出**:
```
✗ Workspace 'game-a' has 1 error

  [ERROR] Runtime parent must not be inside the project root
    Runtime: /home/user/projects/game-a/.runtime
    Project: /home/user/projects/game-a
    
    原因: 运行时目录必须在项目外，以避免污染真实项目文件和 Git 状态
    
    下一步:
      export ADP_RUNTIME_DIR="/tmp/adp-runtime"
      提示: 推荐使用 /tmp 或其他临时目录

```

#### 场景 3: 多个问题混合

**当前输出**:
```
WORKSPACE  LEVEL    CODE                             MESSAGE                                PATH
game-a     error    workspace.project.root.missing   project root is missing               /path/to/missing
game-a     warning  workspace.prompt.missing         configured base prompt file is missing /path/to/prompt.md
game-a     warning  workspace.git.status.dirty       project has 3 changed Git entries     /path/to/project
```

**增强输出**:
```
✗ Workspace 'game-a' has 1 error, 2 warnings

  [ERROR] Project root is missing
    Path: /path/to/missing
    
    下一步:
      1. 检查路径: ls -ld /path/to/missing
      2. 如果项目已移动: adp workspace remove game-a && adp workspace add game-a /new/path
      文档: docs/troubleshooting.zh-CN.md#project-root-does-not-exist

  [WARNING] Base prompt file is missing
    Path: /path/to/prompt.md
    
    下一步:
      快速开始: adp quickstart game-a
      或手动创建: mkdir -p prompts && vim prompts/base.md
      文档: docs/operator-onboarding.md#提示文件

  [WARNING] Project has uncommitted changes
    Path: /path/to/project
    
    原因: 项目有未提交的更改
    
    下一步:
      查看状态: git -C /path/to/project status
      查看差异: git -C /path/to/project diff
      提示: 这是提示信息，不影响 ADP 运行

```

#### 场景 4: 健康的工作区

**当前输出**:
```
WORKSPACE  LEVEL  CODE  MESSAGE     PATH
game-a     ok     -     no issues   /home/user/.adp/workspaces/game-a
```

**增强输出**:
```
✓ Workspace 'game-a' is healthy
```

## 7. 风险评估和缓解

### 7.1 潜在风险

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|----------|
| 建议不准确或误导用户 | 高 | 中 | 仔细审查每个建议，优先实现高优先级错误的建议 |
| 输出过于冗长影响可读性 | 中 | 中 | 默认模式简洁，详细信息放在 --verbose |
| JSON 结构变更破坏下游工具 | 高 | 低 | 只添加字段，不删除或修改现有字段 |
| 国际化支持不足 | 低 | 高 | 当前硬编码中文，未来可扩展 i18n |
| 性能退化 | 低 | 低 | 建议生成逻辑简单，开销可忽略 |
| 维护成本增加 | 中 | 高 | 使用模板化设计，新诊断代码易于添加 |

### 7.2 回滚计划

如果增强导致问题，可以：
1. **保留旧渲染器**: 通过环境变量切换 `ADP_DOCTOR_LEGACY=1`
2. **禁用建议**: 在渲染时忽略 `Suggestion` 字段
3. **降级到 tabwriter**: 回退到 `workspace_commands.go` 的原始实现

### 7.3 测试策略

**单元测试**:
- 每个诊断代码的建议生成
- 渲染器输出格式
- JSON 结构正确性

**集成测试**:
- 使用真实工作区配置
- 覆盖所有错误场景
- 验证退出代码

**手动测试**:
- 创建故意损坏的工作区
- 验证建议是否有效
- 用户体验检查

## 8. 未来扩展方向

### 8.1 自动修复

在某些情况下，doctor 可以提供自动修复选项：

```bash
adp workspace doctor game-a --fix
```

例如：
- 创建缺失的目录结构
- 生成默认的 prompt 文件
- 修复配置文件中的简单错误

### 8.2 交互式模式

提供交互式引导：

```bash
adp workspace doctor game-a --interactive

✗ Workspace 'game-a' has 2 errors

[1/2] Project root is missing
  Path: /path/to/missing
  
  Do you want to:
  1. Check if path exists
  2. Remove and re-add workspace
  3. Skip
  
Your choice [1-3]:
```

### 8.3 健康评分

为工作区提供健康评分：

```bash
adp workspace doctor game-a

✓ Workspace 'game-a' - Health Score: 85/100

  Deductions:
  - Missing shared memory file (-5)
  - Project has uncommitted changes (-10)
  
  Recommendations:
  - Create memory/shared.md to enable memory features
  - Commit or stash changes for cleaner agent sessions
```

### 8.4 持续监控

定期运行诊断并报告：

```bash
adp workspace watch game-a --interval 5m

[14:00:00] ✓ All healthy
[14:05:00] ✗ New issue detected: project root missing
[14:05:01] Sending notification...
```

## 9. 总结

本设计文档提出了 doctor 命令的输出增强方案：

**核心改进**:
1. **更友好的输出** - 使用清晰的格式和表情符号
2. **实用的建议** - 为每个错误提供具体的修复命令
3. **上下文说明** - 解释为什么某些配置是错误的
4. **文档链接** - 引导用户到详细的故障排查文档

**实施成本**: 约 8-11 小时
**用户价值**: 显著降低故障排查时间，提升新用户体验

**下一步**: 开始阶段 1 的数据结构扩展实现

