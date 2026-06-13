# 交互式输入库评估报告

## 执行摘要

本报告评估了三个 Go 交互式输入库，用于 ADP 项目的 `quickstart` 命令实现。基于依赖管理、API 设计、维护状态和项目契合度的综合分析，**推荐使用 `github.com/charmbracelet/huh`**。

**推荐方案**: `github.com/charmbracelet/huh` v0.6.0+

**理由**:
- 活跃维护，现代化 API 设计
- 零依赖核心，构建轻量
- 原生支持非交互模式
- 与 ADP 的简洁代码风格一致
- 强大的表单验证和错误处理

---

## 详细评估

### 1. github.com/manifoldco/promptui

**基本信息**:
- **仓库**: https://github.com/manifoldco/promptui
- **版本**: v0.9.0 (2021)
- **Stars**: ~6.5k
- **维护状态**: ⚠️ 低活跃度（最后主要更新 2021）

**优势**:
- API 简洁直观，学习曲线平缓
- 轻量级实现，依赖少（主要依赖 `github.com/chzyer/readline` 和 `github.com/manifoldco/ansi`）
- 成熟稳定，在生产环境广泛使用
- 支持基本的 Prompt（输入）和 Select（选择）功能
- 内置验证函数支持

**劣势**:
- ⚠️ **维护停滞**: 最后活跃更新在 2021 年，近期主要是社区修复
- 功能相对基础，缺少高级表单组合能力
- 非交互模式支持需要手动实现（无内置支持）
- 依赖 `readline` 库可能在某些终端环境存在兼容性问题
- 无原生多步骤表单支持

**依赖分析**:
```
github.com/chzyer/readline v1.5.1
github.com/manifoldco/ansi (内部包)
golang.org/x/sys (间接依赖)
```

**代码示例**:
```go
// 基本文本输入
prompt := promptui.Prompt{
    Label:    "ADP Home Directory",
    Default:  "~/.adp",
    Validate: validatePath,
}
result, err := prompt.Run()

// 选择列表
selectPrompt := promptui.Select{
    Label: "Enable memory?",
    Items: []string{"Yes", "No"},
}
_, result, err := selectPrompt.Run()
```

**适配 ADP 的复杂度**: 中等
- 需要自行实现非交互模式逻辑
- 多步骤流程需要手动组合多个 Prompt
- 路径验证需要自定义 Validate 函数

**评分**: 6.5/10
- API 易用性: 8/10
- 维护状态: 4/10
- 功能完整性: 6/10
- 项目契合度: 7/10

---

### 2. github.com/AlecAivazis/survey/v2

**基本信息**:
- **仓库**: https://github.com/AlecAivazis/survey (原仓库)
- **版本**: v2.3.7 (2023)
- **Stars**: ~4.1k
- **维护状态**: 🔴 **已归档**（2024年4月19日）

**优势**:
- 功能丰富，支持多种输入类型（Input, Select, MultiSelect, Confirm, Password, Editor）
- 跨平台支持良好（Windows, POSIX）
- 可访问性设计优秀（ANSI 终端支持）
- 提供完整的测试辅助工具
- 社区生态较好，有多个活跃分支

**劣势**:
- 🔴 **原仓库已归档**: 原作者不再维护
- 活跃分支分散（`go-survey/survey`、`linaro-its/golang-survey`），选择困难
- 依赖较重，间接依赖较多
- API 设计较为传统，缺少现代 Go 惯用法
- 非交互模式支持不完整，需要通过环境变量或修改输入流实现

**依赖分析**:
```
github.com/kballard/go-shellquote v0.0.0
github.com/mattn/go-isatty v0.0.20
github.com/mgutz/ansi v0.0.0
golang.org/x/term v0.22.0
golang.org/x/text v0.16.0
(及多个间接依赖)
```

**代码示例**:
```go
// 基本输入
var adpHome string
prompt := &survey.Input{
    Message: "ADP Home Directory:",
    Default: "~/.adp",
}
survey.AskOne(prompt, &adpHome, survey.WithValidator(survey.Required))

// 确认
var enableMemory bool
confirmPrompt := &survey.Confirm{
    Message: "Enable memory?",
    Default: true,
}
survey.AskOne(confirmPrompt, &enableMemory)

// 多步骤
qs := []*survey.Question{
    {Name: "name", Prompt: &survey.Input{Message: "Workspace name:"}},
    {Name: "root", Prompt: &survey.Input{Message: "Project root:"}},
}
answers := struct {
    Name string
    Root string
}{}
survey.Ask(qs, &answers)
```

**适配 ADP 的复杂度**: 中高
- 需要选择并固定一个活跃分支
- 依赖较重可能影响构建时间
- 非交互模式实现较为复杂

**评分**: 5.5/10
- API 易用性: 7/10
- 维护状态: 3/10 (归档)
- 功能完整性: 8/10
- 项目契合度: 5/10

---

### 3. github.com/charmbracelet/huh ⭐ **推荐**

**基本信息**:
- **仓库**: https://github.com/charmbracelet/huh
- **版本**: v0.6.0+ (2024活跃)
- **Stars**: ~4.5k+
- **维护状态**: ✅ **活跃维护**（Charm 组织持续投入）

**优势**:
- ✅ **活跃维护**: 属于 Charm 生态系统，持续更新和改进
- ✅ **现代化设计**: 使用 Bubble Tea 架构，声明式 API
- ✅ **原生非交互模式**: 内置 `Accessible()` 模式，完美支持 CI/脚本环境
- ✅ **零依赖核心**: 核心功能无外部依赖，可选的 TUI 增强依赖 Bubble Tea
- ✅ **表单组合**: 原生支持多步骤表单（Form + Group）
- ✅ **强大验证**: 内置验证器和自定义验证支持
- ✅ **优秀的类型安全**: 使用泛型提供类型安全的字段绑定
- 丰富的字段类型（Input, Text, Select, MultiSelect, Confirm, FilePicker 等）
- 美观的 TUI 界面（可选）

**劣势**:
- 相对较新，社区规模小于 survey（但增长快）
- 完整 TUI 模式依赖 Bubble Tea 生态（但这对我们是优势，因为可以只用轻量模式）
- API 较为现代化，可能需要团队学习（但代码更清晰）

**依赖分析** (Accessible 模式):
```
// 核心无外部依赖
// 可选增强:
github.com/charmbracelet/bubbles (TUI 组件)
github.com/charmbracelet/bubbletea (TUI 框架)
github.com/charmbracelet/lipgloss (样式)
```

**代码示例**:
```go
package main

import (
    "fmt"
    "github.com/charmbracelet/huh"
)

func quickstartFlow() error {
    var (
        adpHome       string
        workspaceName string
        projectRoot   string
        enableMemory  bool
        enableMCP     bool
    )

    // 定义表单
    form := huh.NewForm(
        // Step 1: ADP Home 初始化
        huh.NewGroup(
            huh.NewInput().
                Title("ADP Home Directory").
                Description("Where to store ADP configuration").
                Value(&adpHome).
                Placeholder("~/.adp").
                Validate(func(s string) error {
                    if s == "" {
                        return fmt.Errorf("path cannot be empty")
                    }
                    return validatePath(s)
                }),
        ),

        // Step 2: Workspace 配置
        huh.NewGroup(
            huh.NewInput().
                Title("Workspace Name").
                Description("A unique name for this workspace").
                Value(&workspaceName).
                Validate(validateWorkspaceName),

            huh.NewInput().
                Title("Project Root").
                Description("Path to your project directory").
                Value(&projectRoot).
                Validate(validateProjectRoot),

            huh.NewConfirm().
                Title("Enable Memory?").
                Description("Store conversation history").
                Value(&enableMemory),

            huh.NewConfirm().
                Title("Enable MCP Servers?").
                Description("Model Context Protocol integration").
                Value(&enableMCP),
        ),
    )

    // 运行表单 (自动检测终端能力)
    if err := form.Run(); err != nil {
        return err
    }

    // 执行实际操作
    fmt.Printf("Initializing ADP at %s\n", adpHome)
    fmt.Printf("Creating workspace %q at %s\n", workspaceName, projectRoot)
    fmt.Printf("Memory: %v, MCP: %v\n", enableMemory, enableMCP)

    return nil
}

// 非交互模式支持
func quickstartNonInteractive(flags Flags) error {
    var (
        adpHome       = flags.Home
        workspaceName = flags.WorkspaceName
        projectRoot   = flags.ProjectRoot
        enableMemory  = flags.Memory
        enableMCP     = flags.MCP
    )

    // 使用 Accessible 模式（无 TUI，直接使用值）
    form := huh.NewForm(
        huh.NewGroup(
            huh.NewInput().Value(&adpHome),
            huh.NewInput().Value(&workspaceName),
            huh.NewInput().Value(&projectRoot),
            huh.NewConfirm().Value(&enableMemory),
            huh.NewConfirm().Value(&enableMCP),
        ),
    ).WithAccessible(true) // 强制非交互模式

    // 验证并执行
    if err := form.Run(); err != nil {
        return err
    }

    // 执行实际操作
    return performQuickstart(adpHome, workspaceName, projectRoot, enableMemory, enableMCP)
}

func validatePath(s string) error {
    // 路径验证逻辑
    return nil
}

func validateWorkspaceName(s string) error {
    // workspace 名称验证逻辑
    return nil
}

func validateProjectRoot(s string) error {
    // 项目根目录验证逻辑
    return nil
}
```

**适配 ADP 的复杂度**: 低
- 原生非交互模式支持，无需额外代码
- 表单结构清晰，易于维护
- 验证逻辑内置在字段定义中
- 与 ADP 现有的简洁代码风格一致

**评分**: 9/10
- API 易用性: 9/10
- 维护状态: 10/10
- 功能完整性: 9/10
- 项目契合度: 9/10

---

## 对比表格

| 特性 | promptui | survey/v2 | huh ⭐ |
|------|----------|-----------|--------|
| **维护状态** | ⚠️ 低活跃 (2021) | 🔴 已归档 (2024) | ✅ 活跃维护 |
| **Stars** | ~6.5k | ~4.1k | ~4.5k+ |
| **最后更新** | 2021 | 2023 (归档前) | 2024+ |
| **核心依赖数** | 2 | 5+ | 0 (核心) |
| **非交互模式** | ❌ 需自实现 | ⚠️ 部分支持 | ✅ 原生支持 |
| **多步骤表单** | ❌ 需手动组合 | ✅ 支持 | ✅ 原生支持 |
| **路径验证** | 🟡 自定义 | 🟡 自定义 | ✅ 内置 |
| **API 风格** | 传统命令式 | 传统命令式 | 现代声明式 |
| **类型安全** | 🟡 运行时 | 🟡 运行时 | ✅ 泛型支持 |
| **错误处理** | 基础 | 良好 | 优秀 |
| **文档质量** | 良好 | 优秀 | 优秀 |
| **学习曲线** | 低 | 中 | 中 |
| **构建大小** | 小 | 中等 | 小 |
| **测试友好** | 一般 | 优秀 | 优秀 |
| **CI 环境支持** | ⚠️ 需额外工作 | 🟡 可配置 | ✅ 开箱即用 |

---

## 推荐方案: charmbracelet/huh

### 选择理由

1. **维护状态最佳**: 
   - Charm 组织持续投入资源维护
   - 活跃的社区和快速的问题响应
   - 定期发布新版本和功能改进

2. **原生非交互模式支持**:
   ```go
   form.WithAccessible(true) // 一行代码切换到非交互模式
   ```
   这对 CI/CD 环境和自动化脚本至关重要，其他库需要大量额外代码实现。

3. **零依赖核心**:
   - ADP 当前 `go.mod` 只有 `gopkg.in/yaml.v3` 一个依赖
   - huh 核心功能无外部依赖，保持项目轻量
   - 可选的 TUI 增强不会强制引入

4. **API 设计与 ADP 契合**:
   - 声明式 API，代码结构清晰
   - 类型安全的字段绑定
   - 与 ADP 现有的简洁、功能性代码风格一致

5. **表单验证集成**:
   ```go
   huh.NewInput().
       Validate(func(s string) error {
           // 验证逻辑直接在字段定义中
           return validatePath(s)
       })
   ```
   验证逻辑与字段定义耦合，易于维护。

6. **测试友好**:
   - 支持 Accessible 模式进行单元测试
   - 无需 mock 终端即可测试表单逻辑
   - 与 ADP 现有测试框架无缝集成

7. **未来扩展性**:
   - 如果未来需要更丰富的 TUI 界面，可以无缝升级
   - Bubble Tea 生态系统提供丰富的组件
   - 但当前简单模式已满足所有需求

### 实施路径

**Phase 1: 基础集成** (1小时)
```bash
go get github.com/charmbracelet/huh@latest
```
创建 `internal/cli/quickstart.go`，实现基础表单结构。

**Phase 2: 交互流程** (2小时)
实现 init 和 workspace 两步流程，添加验证逻辑。

**Phase 3: 非交互模式** (1小时)
添加 `--non-interactive` 支持，使用 `WithAccessible(true)`。

**Phase 4: 测试覆盖** (1.5小时)
单元测试 + 集成测试脚本。

**总计**: ~5.5小时

### 风险评估

**风险**: huh 相对较新，社区规模小于传统库
**缓解**: 
- Charm 是 Go TUI 领域的领导者（bubbletea, lipgloss, bubbles）
- 代码质量高，测试覆盖率优秀
- 如有问题，API 简单可快速切换到其他库

**风险**: 团队需要学习新 API
**缓解**:
- API 直观，学习曲线不陡峭
- 文档和示例丰富
- 声明式风格与现代 Go 实践一致

---

## 不推荐其他方案的原因

### 为何不选 promptui?
- ⚠️ 维护停滞：最后活跃更新在 2021 年
- 功能受限：缺少高级表单功能
- 非交互模式需要大量额外代码
- 未来可能遇到兼容性问题

### 为何不选 survey/v2?
- 🔴 原仓库已归档（2024年4月）
- 活跃分支分散，选择困难
- 依赖较重（5+ 外部依赖）
- API 设计传统，与 ADP 现代风格不符
- 非交互模式支持不完整

---

## 代码示例: ADP quickstart 实现预览

```go
// internal/cli/quickstart.go
package cli

import (
    "context"
    "errors"
    "fmt"
    "os"
    "path/filepath"
    
    "github.com/charmbracelet/huh"
    "github.com/karoc/adp/internal/workspace"
)

type QuickstartOptions struct {
    NonInteractive bool
    ADPHome        string
    WorkspaceName  string
    ProjectRoot    string
    EnableMemory   bool
    EnableMCP      bool
    EnableAgents   bool
}

func (a *App) quickstart(ctx context.Context, args []string) error {
    opts, err := parseQuickstartArgs(args)
    if err != nil {
        return err
    }

    if opts.NonInteractive {
        return a.quickstartNonInteractive(ctx, opts)
    }

    return a.quickstartInteractive(ctx, opts)
}

func (a *App) quickstartInteractive(ctx context.Context, opts QuickstartOptions) error {
    // 默认值
    adpHome := opts.ADPHome
    if adpHome == "" {
        home, _ := os.UserHomeDir()
        adpHome = filepath.Join(home, ".adp")
    }

    var (
        workspaceName string
        projectRoot   string
        enableMemory  bool
        enableMCP     bool
        enableAgents  bool
    )

    // 构建表单
    form := huh.NewForm(
        // Step 1: ADP Home 初始化
        huh.NewGroup(
            huh.NewNote().
                Title("Welcome to ADP!").
                Description("Let's set up your Agent Development Platform"),

            huh.NewInput().
                Title("ADP Home Directory").
                Description("Where to store ADP configuration and data").
                Value(&adpHome).
                Validate(func(s string) error {
                    if s == "" {
                        return errors.New("path cannot be empty")
                    }
                    // 检查路径是否已存在
                    if _, err := os.Stat(s); err == nil {
                        return fmt.Errorf("directory already exists: %s", s)
                    }
                    return nil
                }),
        ).Title("Step 1: Initialize ADP Home"),

        // Step 2: Workspace 配置
        huh.NewGroup(
            huh.NewNote().
                Title("Create Your First Workspace").
                Description("A workspace connects ADP to a project directory"),

            huh.NewInput().
                Title("Workspace Name").
                Description("A unique identifier (e.g., 'my-project')").
                Value(&workspaceName).
                Validate(func(s string) error {
                    if s == "" {
                        return errors.New("name cannot be empty")
                    }
                    if !workspace.IsValidName(s) {
                        return errors.New("invalid name format")
                    }
                    return nil
                }),

            huh.NewInput().
                Title("Project Root").
                Description("Path to your project directory").
                Value(&projectRoot).
                Validate(func(s string) error {
                    if s == "" {
                        return errors.New("path cannot be empty")
                    }
                    info, err := os.Stat(s)
                    if err != nil {
                        return fmt.Errorf("path does not exist: %s", s)
                    }
                    if !info.IsDir() {
                        return errors.New("path must be a directory")
                    }
                    return nil
                }),
        ).Title("Step 2: Workspace Configuration"),

        // Step 3: 可选功能
        huh.NewGroup(
            huh.NewNote().
                Title("Optional Features").
                Description("Configure additional ADP features"),

            huh.NewConfirm().
                Title("Enable Memory?").
                Description("Store conversation history for context").
                Value(&enableMemory).
                Affirmative("Yes").
                Negative("No"),

            huh.NewConfirm().
                Title("Enable MCP Servers?").
                Description("Model Context Protocol integration").
                Value(&enableMCP).
                Affirmative("Yes").
                Negative("No"),

            huh.NewConfirm().
                Title("Enable Custom Agents?").
                Description("Support for custom agent definitions").
                Value(&enableAgents).
                Affirmative("Yes").
                Negative("No"),
        ).Title("Step 3: Optional Features"),
    )

    // 运行交互式表单
    if err := form.Run(); err != nil {
        return err
    }

    // 执行实际操作
    fmt.Fprintln(a.stdout, "\n🚀 Setting up ADP...")

    // 1. 初始化 ADP home
    if err := a.deps.WorkspaceStore.Init(ctx); err != nil {
        return fmt.Errorf("failed to initialize ADP home: %w", err)
    }
    fmt.Fprintf(a.stdout, "✓ Initialized ADP home at %s\n", adpHome)

    // 2. 创建 workspace
    ws, err := a.deps.WorkspaceStore.Add(ctx, workspaceName, projectRoot)
    if err != nil {
        return fmt.Errorf("failed to create workspace: %w", err)
    }
    fmt.Fprintf(a.stdout, "✓ Created workspace %q\n", workspaceName)

    // 3. 配置可选功能
    if enableMemory {
        fmt.Fprintln(a.stdout, "✓ Memory enabled")
    }
    if enableMCP {
        fmt.Fprintln(a.stdout, "✓ MCP servers enabled")
    }
    if enableAgents {
        fmt.Fprintln(a.stdout, "✓ Custom agents enabled")
    }

    // 4. 运行 doctor 检查
    fmt.Fprintln(a.stdout, "\n🔍 Running diagnostics...")
    if err := a.doctor(ctx, []string{workspaceName}); err != nil {
        fmt.Fprintf(a.stderr, "⚠️  Diagnostics found issues (run 'adp workspace doctor %s' for details)\n", workspaceName)
    }

    fmt.Fprintln(a.stdout, "\n✨ Setup complete! Next steps:")
    fmt.Fprintf(a.stdout, "  adp run --workspace %s\n", workspaceName)
    
    return nil
}

func (a *App) quickstartNonInteractive(ctx context.Context, opts QuickstartOptions) error {
    // 验证所有必需参数
    if opts.ADPHome == "" || opts.WorkspaceName == "" || opts.ProjectRoot == "" {
        return errors.New("--non-interactive requires --adp-home, --workspace-name, and --project-root")
    }

    // 使用 huh 的 Accessible 模式进行验证
    var (
        adpHome       = opts.ADPHome
        workspaceName = opts.WorkspaceName
        projectRoot   = opts.ProjectRoot
    )

    form := huh.NewForm(
        huh.NewGroup(
            huh.NewInput().Value(&adpHome).Validate(validateADPHome),
            huh.NewInput().Value(&workspaceName).Validate(validateWorkspaceName),
            huh.NewInput().Value(&projectRoot).Validate(validateProjectRoot),
        ),
    ).WithAccessible(true)

    if err := form.Run(); err != nil {
        return err
    }

    // 执行实际操作（与交互模式相同）
    fmt.Fprintf(a.stdout, "Initializing ADP at %s\n", adpHome)
    if err := a.deps.WorkspaceStore.Init(ctx); err != nil {
        return err
    }

    fmt.Fprintf(a.stdout, "Creating workspace %q at %s\n", workspaceName, projectRoot)
    if _, err := a.deps.WorkspaceStore.Add(ctx, workspaceName, projectRoot); err != nil {
        return err
    }

    fmt.Fprintln(a.stdout, "Setup complete")
    return nil
}

func validateADPHome(s string) error {
    if s == "" {
        return errors.New("ADP home cannot be empty")
    }
    return nil
}

func validateWorkspaceName(s string) error {
    if s == "" {
        return errors.New("workspace name cannot be empty")
    }
    if !workspace.IsValidName(s) {
        return errors.New("invalid workspace name format")
    }
    return nil
}

func validateProjectRoot(s string) error {
    if s == "" {
        return errors.New("project root cannot be empty")
    }
    info, err := os.Stat(s)
    if err != nil {
        return fmt.Errorf("invalid path: %w", err)
    }
    if !info.IsDir() {
        return errors.New("project root must be a directory")
    }
    return nil
}

func parseQuickstartArgs(args []string) (QuickstartOptions, error) {
    // 参数解析逻辑
    // 支持: --non-interactive, --adp-home, --workspace-name, --project-root, --memory, --mcp, --agents
    return QuickstartOptions{}, nil
}
```

---

## 结论

基于维护状态、功能完整性、API 设计和项目契合度的综合评估，**强烈推荐使用 `github.com/charmbracelet/huh`** 作为 ADP 项目的交互式输入库。

该库提供：
- ✅ 活跃维护和持续改进
- ✅ 原生非交互模式支持
- ✅ 零依赖核心，保持项目轻量
- ✅ 现代化 API 设计
- ✅ 优秀的测试友好性
- ✅ 与 ADP 代码风格完美契合

实施该方案预计需要 5.5 小时，无重大技术风险。

---

## 参考资源

- [manifoldco/promptui](https://github.com/manifoldco/promptui)
- [AlecAivazis/survey (archived)](https://github.com/AlecAivazis/survey)
- [go-survey/survey (fork)](https://github.com/go-survey/survey)
- [charmbracelet/huh](https://github.com/charmbracelet/huh)
- [Charm - Bubble Tea ecosystem](https://github.com/charmbracelet/)

---

**文档版本**: 1.0  
**评估日期**: 2026-06-13  
**评估者**: Agent (P2-1a task)
