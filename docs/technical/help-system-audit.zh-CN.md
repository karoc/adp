# Help 系统审计与"另见"功能设计

**日期:** 2026-06-14  
**状态:** 设计文档  
**作者:** ADP 团队

## 执行摘要

本文档对 ADP 命令 help 系统进行全面审计,并提出实现交叉引用"另见"(See Also)功能的技术设计,以提升命令可发现性并减少用户混淆。

**核心发现:**
- ADP 使用自定义命令元数据系统(无 Cobra 依赖)
- Help 文本集中定义在 `internal/commandmeta/metadata.go`
- 当前"另见"仅存在于子命令 help(指向父命令)
- 18 个根命令,复杂度不一(2-12 个子命令)
- 在任务工作流、会话恢复和诊断方面有强烈的跨命令引用需求

**提议方案:**
- 扩展 `commandmeta.Command` 结构体,添加 `SeeAlso []string` 字段
- 在元数据层集中管理命令关系映射
- 实现双语支持(英文/中文)
- 分阶段推出,优先处理高混淆命令对

---

## 1. 当前 Help 系统架构

### 1.1 核心组件

**位置:** `/srv/agent-development-platform/internal/commandmeta/`

**关键文件:**
- `metadata.go` - 命令元数据定义和 help 渲染
- `examples.go` - 命令/子命令使用示例
- `cli.go` - 命令分发器和错误处理

**数据结构:**

```go
// internal/commandmeta/metadata.go
type Command struct {
    Name        string
    Description string
    Usage       []string      // 多行使用示例
    Subcommands []Value       // 嵌套子命令
    Options     []Value       // 可用标志/选项
}

type Value struct {
    Name        string
    Description string
}
```

### 1.2 Help 生成流程

```
用户运行: adp tasks take --help
    ↓
cli.Execute() 检查 --help 标志
    ↓
调用 commandHelp(["tasks", "take"])
    ↓
commandmeta.SubcommandHelp("tasks", "take")
    ↓
渲染: usage + examples + "See also: adp tasks --help"
```

**当前"另见"实现:**
- **位置:** `metadata.go:450-452`
- **范围:** 仅在子命令 help 中
- **格式:** 总是指向父命令
- **示例:** `adp tasks take --help` 显示 "See also: adp tasks --help"

**局限性:**
1. 无跨命令引用(如 `tasks` ↔ `run`, `sessions` ↔ `events`)
2. 无工作流指导(如 `tasks next` → `tasks take` → `run`)
3. 无诊断命令建议(如出错时建议 `doctor`)

### 1.3 命令清单

ADP 当前有 **18 个根命令**,分布在 6 个功能区域:

| 类别 | 命令 | 子命令数 | 复杂度 |
|------|------|---------|--------|
| **设置** | init, quickstart, doctor, version | 0, 0, 0, 0 | 低 |
| **工作区** | workspace | 6 (add, list, show, remove, rename, doctor) | 中 |
| **运行时** | enter, env, run, runtime | 0, 0, 0, 1 (prune) | 中 |
| **任务管理** | tasks, plan, phase, progress | 12, 3, 8, 1 | 高 |
| **可观测性** | events, sessions | 1 (list), 4 (list, show, restore-plan, resume-plan) | 中 |
| **Shell 集成** | shell-hook, completion | 0, 1 (values) | 低 |

**总计:** 18 个根命令, 36 个子命令

---

## 2. 命令关系分析

### 2.1 基于工作流的关系

#### **工作流 1: 初始设置**
```
init → workspace add → doctor → quickstart
```
- **混淆点:** 用户不知道 `workspace add` 后应该运行 `doctor`
- **需求:** `workspace add` 应建议 `doctor` 来验证设置

#### **工作流 2: 任务领取**
```
tasks next → tasks take → run --take → tasks renew → tasks done
         ↘ tasks claim ↗
```
- **混淆点:** `take`(原子领取下一个)与 `claim`(领取特定任务)的区别
- **需求:** `tasks next`、`tasks take`、`tasks claim` 和 `run --take` 之间的交叉引用

#### **工作流 3: 规划**
```
plan preview → plan apply → tasks list → phase status
```
- **混淆点:** plan 导入与任务创建的关系
- **需求:** `plan apply` 应引用 `tasks list` 和 `phase list`

#### **工作流 4: 会话恢复**
```
sessions list → sessions show → sessions restore-plan → run
                            ↘ sessions resume-plan ↗
```
- **混淆点:** 何时使用 `restore-plan` vs `resume-plan`
- **需求:** 恢复命令之间的清晰区分和交叉引用

#### **工作流 5: 诊断**
```
doctor → workspace doctor → plan doctor → runtime prune
```
- **混淆点:** 多个 `doctor` 变体对应不同作用域
- **需求:** 根命令 `doctor` 应引用专门的诊断命令

### 2.2 混淆矩阵

需要"另见"引用的高混淆命令对:

| 命令对 | 混淆类型 | 优先级 |
|--------|---------|--------|
| `tasks take` ↔ `tasks claim` | 语义重叠 | **P0** |
| `run --take` ↔ `tasks take` | 原子 vs 手动 | **P0** |
| `doctor` ↔ `workspace doctor` | 作用域差异 | **P0** |
| `sessions restore-plan` ↔ `sessions resume-plan` | 相似名称 | **P0** |
| `tasks next` ↔ `tasks take` | 预览 vs 操作 | **P1** |
| `plan apply` ↔ `tasks list` | 因果关系 | **P1** |
| `workspace add` ↔ `doctor` | 设置顺序 | **P1** |
| `tasks done` ↔ `phase accept` | 任务 vs 阶段完成 | **P1** |
| `events list` ↔ `sessions list` | 不同视图 | **P2** |
| `runtime prune` ↔ `enter` | 清理上下文 | **P2** |

### 2.3 命令别名关系

从 `cli.go:204-218`:
```go
aliases := map[string]string{
    "ws": "workspace",
    "t":  "tasks",
    "s":  "sessions",
    "e":  "events",
    "rt": "runtime",
    "p":  "phase",
}
```

**发现:** 别名功能正常但未在 help 文本中记录。  
**建议:** 在"另见"或命令描述中包含别名信息。

---

## 3. 设计提案:"另见"功能

### 3.1 设计目标

1. **可发现性:** 帮助用户在工作流中找到相关命令
2. **清晰性:** 减少相似命令之间的混淆
3. **最小摩擦:** 不要用过多引用压倒用户
4. **可维护性:** 集中管理关系映射,便于更新
5. **国际化:** 支持双语(英文/中文) help 文本

### 3.2 数据结构扩展

**文件:** `internal/commandmeta/metadata.go`

```go
// 扩展 Command 结构体
type Command struct {
    Name        string
    Description string
    Usage       []string
    Subcommands []Value
    Options     []Value
    SeeAlso     []string      // 新增: 相关命令引用
}

// 为子命令扩展 Value 结构体
type Value struct {
    Name        string
    Description string
    SeeAlso     []string      // 新增: 相关子命令引用(可选)
}
```

**替代设计(更明确):**

```go
// 更结构化的方法,包含关系类型
type RelatedCommand struct {
    Command     string
    Subcommand  string        // 根命令为空
    Relation    RelationType  // "workflow-next", "alternative", "diagnostic"
    Description string        // 可选上下文
}

type RelationType string
const (
    RelationWorkflowNext   RelationType = "workflow-next"    // 自然下一步
    RelationAlternative    RelationType = "alternative"      // 相似功能
    RelationDiagnostic     RelationType = "diagnostic"       // 故障排查
    RelationParent         RelationType = "parent"           // 父命令
    RelationChild          RelationType = "child"            // 子功能
)
```

**建议:** MVP 先使用简单的 `[]string` 方法,如果关系类型变得有价值再迁移到结构化方法。

### 3.3 命令关系映射

**中央映射位置:** `internal/commandmeta/metadata.go` 在 `rootCommands` 定义之后

```go
// 命令关系映射,用于"另见"部分
var commandRelationships = map[string][]string{
    // 设置和诊断
    "init":         {"workspace", "quickstart"},
    "quickstart":   {"doctor", "workspace add"},
    "doctor":       {"workspace doctor", "plan doctor"},
    
    // 工作区管理
    "workspace":    {"doctor", "tasks", "run"},
    
    // 任务工作流
    "tasks":        {"run", "phase", "progress"},
    "run":          {"tasks", "events", "sessions"},
    
    // 规划
    "plan":         {"tasks list", "phase list", "plan doctor"},
    "phase":        {"tasks", "progress", "plan"},
    "progress":     {"tasks", "phase", "sessions"},
    
    // 可观测性
    "events":       {"sessions", "tasks show"},
    "sessions":     {"events", "run", "tasks"},
    
    // 运行时管理
    "runtime":      {"enter", "env", "sessions"},
    "enter":        {"env", "run", "runtime prune"},
}

// 子命令关系 (command.subcommand 格式)
var subcommandRelationships = map[string][]string{
    // 工作区子命令
    "workspace.add":    {"doctor", "tasks add"},
    "workspace.doctor": {"doctor", "plan doctor"},
    
    // 任务子命令
    "tasks.next":       {"tasks take", "tasks claim"},
    "tasks.take":       {"run --take", "tasks next", "tasks renew"},
    "tasks.claim":      {"tasks take", "tasks renew"},
    "tasks.done":       {"phase accept", "tasks list"},
    "tasks.stale":      {"tasks renew", "tasks release"},
    
    // 会话子命令
    "sessions.restore-plan": {"sessions resume-plan", "run"},
    "sessions.resume-plan":  {"sessions restore-plan", "run --take"},
    
    // 规划子命令
    "plan.preview":  {"plan apply"},
    "plan.apply":    {"tasks list", "phase list"},
    "plan.doctor":   {"tasks list", "phase status"},
}
```

### 3.4 Help 渲染函数

**修改 `CommandHelp()` 函数:**

```go
// 文件: internal/commandmeta/metadata.go

func CommandHelp(name string) (string, bool) {
    command, ok := Lookup(name)
    if !ok {
        return "", false
    }

    var out strings.Builder
    out.WriteString("adp ")
    out.WriteString(command.Name)
    if command.Description != "" {
        out.WriteString(" - ")
        out.WriteString(command.Description)
    }
    out.WriteString("\n\nUsage:\n")
    writeUsageLines(&out, command.Usage)
    writeValuesSection(&out, "Subcommands", command.Subcommands)
    writeValuesSection(&out, "Options", command.Options)
    writeExamplesSection(&out, examplesForCommand(command.Name))
    
    // 新增: 添加"另见"部分
    writeSeeAlsoSection(&out, name, "")
    
    return out.String(), true
}

func SubcommandHelp(commandName, subcommand string) (string, bool) {
    command, ok := Lookup(commandName)
    if !ok || !hasValue(command.Subcommands, subcommand) {
        return "", false
    }

    usage := usageLinesForSubcommand(command, subcommand)
    if len(usage) == 0 {
        return "", false
    }

    var out strings.Builder
    out.WriteString("adp ")
    out.WriteString(command.Name)
    out.WriteByte(' ')
    out.WriteString(subcommand)
    if description := valueDescription(command.Subcommands, subcommand); description != "" {
        out.WriteString(" - ")
        out.WriteString(description)
    }
    out.WriteString("\n\nUsage:\n")
    writeUsageLines(&out, usage)
    writeExamplesSection(&out, examplesForSubcommand(command.Name, subcommand))
    
    // 新增: 增强的"另见"部分
    writeSeeAlsoSection(&out, command.Name, subcommand)
    
    return out.String(), true
}

// 新增辅助函数
func writeSeeAlsoSection(out *strings.Builder, commandName, subcommand string) {
    var related []string
    
    if subcommand != "" {
        // 子命令 help: 首先检查子命令关系
        key := commandName + "." + subcommand
        if refs, ok := subcommandRelationships[key]; ok {
            related = append(related, refs...)
        }
        // 始终包含父命令
        related = append(related, commandName+" --help")
    } else {
        // 根命令 help: 检查命令关系
        if refs, ok := commandRelationships[commandName]; ok {
            related = append(related, refs...)
        }
    }
    
    if len(related) == 0 {
        return
    }
    
    out.WriteString("\n另见:\n")
    for _, ref := range related {
        out.WriteString("  adp ")
        // 格式化引用(如果只是命令名则添加 --help)
        if !strings.Contains(ref, " ") {
            out.WriteString(ref)
            out.WriteString(" --help")
        } else {
            out.WriteString(ref)
        }
        out.WriteByte('\n')
    }
}
```

**实施后的使用示例:**

```bash
# 示例 1: tasks take
$ adp tasks take --help
adp tasks take - 原子领取下一项工作

Usage:
  adp tasks take [--workspace <name>] --owner <owner> [--lease <duration>]

Examples:
  adp tasks take --workspace game-a --owner codex-main --lease 4h

另见:
  adp run --take
  adp tasks next --help
  adp tasks renew --help
  adp tasks --help

# 示例 2: doctor
$ adp doctor --help
adp doctor - 诊断已注册的工作区

Usage:
  adp doctor [workspace] [--verbose] [--format <text|json>]

另见:
  adp workspace doctor --help
  adp plan doctor --help
```

---

## 4. 实施计划

### 4.1 分阶段推出

#### **阶段 1: 高优先级命令 (P0)**
**预计工作量:** 2-3 天

目标最高混淆的命令:
- `tasks take` ↔ `tasks claim` ↔ `run --take`
- `doctor` ↔ `workspace doctor`
- `sessions restore-plan` ↔ `sessions resume-plan`

**交付物:**
1. 添加 P0 命令的关系映射
2. 实现 `writeSeeAlsoSection()` 函数
3. 更新 `CommandHelp()` 和 `SubcommandHelp()`
4. 添加 see-also 渲染的单元测试

#### **阶段 2: 工作流序列 (P1)**
**预计工作量:** 3-4 天

目标基于工作流的关系:
- `tasks next` → `tasks take` → `tasks renew` → `tasks done`
- `workspace add` → `doctor` → `tasks add`
- `plan preview` → `plan apply` → `tasks list`

**交付物:**
1. 扩展 P1 命令的关系映射
2. 添加面向工作流的引用
3. 更新文档示例

#### **阶段 3: 全面覆盖 (P2)**
**预计工作量:** 2-3 天

完成剩余命令:
- 可观测性命令 (`events`, `sessions`)
- 运行时管理 (`runtime prune`, `enter`, `env`)
- 工具命令 (`completion`, `shell-hook`)

**交付物:**
1. 完整关系映射
2. 边缘情况处理
3. 全面测试覆盖

### 4.2 测试策略

**单元测试:** `internal/commandmeta/metadata_test.go`

```go
func TestSeeAlsoSection(t *testing.T) {
    tests := []struct {
        name       string
        command    string
        subcommand string
        wantRefs   []string
    }{
        {
            name:       "tasks take 子命令",
            command:    "tasks",
            subcommand: "take",
            wantRefs:   []string{"run --take", "tasks next", "tasks renew"},
        },
        {
            name:       "doctor 根命令",
            command:    "doctor",
            subcommand: "",
            wantRefs:   []string{"workspace doctor", "plan doctor"},
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            var buf strings.Builder
            writeSeeAlsoSection(&buf, tt.command, tt.subcommand)
            output := buf.String()
            
            for _, ref := range tt.wantRefs {
                if !strings.Contains(output, ref) {
                    t.Errorf("另见部分缺少引用: %q", ref)
                }
            }
        })
    }
}

func TestRelationshipIntegrity(t *testing.T) {
    // 验证所有引用的命令都存在
    allCommands := map[string]bool{}
    for _, cmd := range rootCommands {
        allCommands[cmd.Name] = true
        for _, sub := range cmd.Subcommands {
            allCommands[cmd.Name+"."+sub.Name] = true
        }
    }
    
    for cmd, refs := range commandRelationships {
        for _, ref := range refs {
            parts := strings.Fields(ref)
            refCmd := parts[0]
            if !allCommands[refCmd] {
                t.Errorf("命令 %q 引用了不存在的命令 %q", cmd, refCmd)
            }
        }
    }
}
```

### 4.3 向后兼容性

**考虑:** 现有 help 输出是 CLI 契约的一部分。

**方法:**
1. 仅添加功能 - 不破坏现有 help 格式
2. "另见"部分始终出现在末尾
3. 保持子命令 help 中现有的 "See also: adp <parent> --help"
4. 不更改 usage 行或选项描述

---

## 5. 风险与缓解

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|---------|
| **过多引用压倒用户** | 中 | 中 | 每个命令限制 3-5 个最相关的引用 |
| **关系映射变得过时** | 低 | 中 | 添加验证测试检查引用的命令是否存在 |
| **help 中的循环引用** | 低 | 低 | 记录关系模式,在测试中添加循环检测 |
| **破坏解析 help 的现有脚本** | 高 | 低 | 仅进行添加性更改,保留现有格式 |
| **不一致的关系定义** | 中 | 中 | 集中所有映射,使用代码审查检查清单 |

---

## 6. 建议

### 6.1 即时行动 (第 1 周)

1. **实施阶段 1 (P0 命令)**
   - 关注最高混淆对
   - 为关键工作流添加关系映射
   - 实现 `writeSeeAlsoSection()` 函数
   - 更新 help 渲染函数

2. **添加验证测试**
   - 引用完整性检查
   - Help 输出格式验证
   - 确保无循环依赖

3. **记录未来命令的模式**
   - 更新贡献指南
   - 为新命令添加添加检查清单
   - 记录关系类型及使用时机

### 6.2 后续行动 (第 2-3 周)

1. **阶段 2 实施**
   - 基于工作流的关系
   - 扩展测试覆盖
   - 更新操作员文档

2. **用户测试**
   - 从实际操作员获取反馈
   - 根据混淆模式调整关系
   - 迭代引用数量(过多 vs 过少)

3. **文档更新**
   - 更新 README 添加 see-also 示例
   - 添加到操作员入职指南
   - 更新中文文档

---

## 7. 附录

### 7.1 完整命令矩阵

包含所有子命令的完整命令清单:

```
adp
├── init (0 个子命令)
├── quickstart (0 个子命令)
├── doctor (0 个子命令)
├── version (0 个子命令)
├── workspace (6 个子命令)
│   ├── add
│   ├── list
│   ├── show
│   ├── remove
│   ├── rename
│   └── doctor
├── enter (0 个子命令)
├── env (0 个子命令)
├── shell-hook (0 个子命令)
├── completion (1 个子命令)
│   └── values
├── events (1 个子命令)
│   └── list
├── sessions (4 个子命令)
│   ├── list
│   ├── show
│   ├── restore-plan
│   └── resume-plan
├── runtime (1 个子命令)
│   └── prune
├── tasks (12 个子命令)
│   ├── add
│   ├── list
│   ├── next
│   ├── take
│   ├── stale
│   ├── show
│   ├── update
│   ├── claim
│   ├── renew
│   ├── release
│   ├── done
│   └── block
├── plan (3 个子命令)
│   ├── preview
│   ├── apply
│   └── doctor
├── phase (8 个子命令)
│   ├── add
│   ├── list
│   ├── show
│   ├── status
│   ├── start
│   ├── accept
│   ├── commit
│   └── push
├── progress (1 个子命令)
│   └── report
└── run (0 个子命令)
```

---

## 8. 结论

提议的"另见"功能是对 ADP 命令可发现性的低风险、高价值改进。通过实现集中式关系映射并扩展现有 help 系统,我们可以在不进行重大架构更改的情况下显著减少用户混淆。

**关键要点:**
1. 当前 help 系统结构良好且易于维护
2. "另见"可以增量添加,风险最小
3. 首先关注 P0 高混淆命令
4. 始终保持向后兼容性
5. 成功取决于精心策划的关系(质量重于数量)

**下一步:**
1. 与团队审查此设计文档
2. 获得阶段 1 实施的批准
3. 在 ADP 任务板中创建实施任务
4. 从 P0 命令关系开始编码

---

**文档版本:** 1.0  
**最后更新:** 2026-06-14  
**审查状态:** 准备团队审查