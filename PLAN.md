# ADP 易用性改进计划

基于易用性测试报告（usability-test-report.md），本计划列出所有改进项，按优先级组织，并设计支持多agent并行推进的实施方案。

---

## 改进项总览

| ID | 优先级 | 改进项 | 预估工作量 | 可并行 |
|----|--------|--------|-----------|--------|
| P0-1 | P0 | 修复workspace list --format json支持 | S | ✅ |
| P0-2 | P0 | 改进agent不存在时的错误提示 | M | ✅ |
| P0-3 | P0 | 增强版本信息显示 | S | ✅ |
| P1-1 | P1 | 为空列表添加友好提示 | M | ✅ |
| P1-2 | P1 | 统一输出格式（workspace show支持--format） | S | ✅ |
| P1-3 | P1 | 在adp --help中添加项目描述和快速入门链接 | S | ✅ |
| P1-4 | P1 | 改进init成功后的引导信息 | S | ✅ |
| P1-5 | P1 | workspace add时先检查名称冲突 | S | ✅ |
| P2-1 | P2 | 添加交互式引导命令（adp quickstart） | L | ⚠️ |
| P2-2 | P2 | 优化ID格式的可读性 | M | ⚠️ |

工作量：S=Small（<2h），M=Medium（2-4h），L=Large（>4h）

---

## Phase 1: P0高优先级改进（核心体验修复）

**目标**：修复明显的不一致和体验问题
**并行策略**：3个独立任务，可完全并行

### Task P0-1: 修复workspace list --format json支持

**问题描述**：
- 当前`adp workspace list`不支持`--format json`参数
- 其他命令（tasks、sessions、events、doctor等）都支持
- 造成命令接口不一致

**解决方案**：
1. 在`internal/cli/workspace.go`中为`WorkspaceListCmd`添加`--format`参数支持
2. 实现JSON输出格式，结构参考现有命令
3. 更新`internal/commandmeta/examples.go`添加JSON示例
4. 更新相关测试

**输出格式设计**：
```json
{
  "workspaces": [
    {
      "name": "game-a",
      "project_root": "/srv/game-a",
      "workspace_dir": "/home/user/.adp/workspaces/game-a",
      "memory_enabled": true,
      "mcp_enabled": true
    }
  ],
  "count": 1
}
```

**修改文件**：
- `internal/cli/workspace.go` - 添加format参数和JSON输出逻辑
- `internal/workspace/list.go` - 可能需要调整数据结构
- `internal/commandmeta/examples.go` - 添加示例
- `internal/cli/workspace_test.go` - 添加测试

**验证**：
- `scripts/runtime-audit-smoke.sh` 应该通过
- 手动测试：`adp workspace list --format json | jq`
- 确保与`adp workspace list`（text格式）输出一致

**独立性**：✅ 完全独立，不依赖其他任务

---

### Task P0-2: 改进agent不存在时的错误提示

**问题描述**：
- 当agent命令不存在时，显示"Error: stdin is not a terminal"
- 用户无法理解这是命令不存在的问题

**解决方案**：
1. 在`internal/runner/runner.go`或`internal/cli/run.go`中，启动agent前检查命令是否存在
2. 使用`exec.LookPath()`检查命令可执行性
3. 提供友好的错误提示，包括：
   - 明确说明agent命令未找到
   - 提示如何安装或配置
   - 列出workspace配置中的agent设置路径

**错误提示设计**：
```
adp: agent command not found: codex

The workspace is configured to use:
  command: codex
  path lookup: $PATH

Possible solutions:
  - Install codex CLI (see: https://docs.codex.com/install)
  - Configure explicit command path in workspace settings
  - Check if the command is in your $PATH

try: adp workspace doctor my-project
```

**修改文件**：
- `internal/runner/runner.go` - 添加命令存在性检查
- `internal/runner/errors.go` - 定义新的错误类型
- 相关smoke测试 - 验证错误提示

**验证**：
- 测试不存在的agent命令，确认错误提示清晰
- 测试存在的命令，确保正常运行不受影响

**独立性**：✅ 完全独立，不依赖其他任务

---

### Task P0-3: 增强版本信息显示

**问题描述**：
- 当前`adp version`只输出"adp dev"
- 缺少构建信息、commit hash、Go版本等

**解决方案**：
1. 在`cmd/adp/main.go`中使用build ldflags注入版本信息
2. 扩展`internal/cli/version.go`输出更多信息
3. 支持`--format json`输出结构化版本信息

**输出格式设计**：

Text格式：
```
adp version dev
commit: a966ad7
built: 2026-06-13T10:30:00Z
go: go1.21.0
platform: linux/amd64
```

JSON格式：
```json
{
  "version": "dev",
  "commit": "a966ad7",
  "built_at": "2026-06-13T10:30:00Z",
  "go_version": "go1.21.0",
  "platform": "linux/amd64"
}
```

**修改文件**：
- `cmd/adp/main.go` - 定义版本变量
- `internal/cli/version.go` - 扩展输出格式
- `Makefile` 或构建脚本 - 添加ldflags注入
- `scripts/release-*.sh` - 确保release构建注入版本信息

**验证**：
- 开发构建显示"dev"
- Release构建显示实际版本号和commit
- JSON格式可解析

**独立性**：✅ 完全独立，不依赖其他任务

---

## Phase 2: P1中优先级改进（用户体验增强）

**目标**：提升新用户体验和命令一致性
**并行策略**：5个独立任务，可完全并行

### Task P1-1: 为空列表添加友好提示

**问题描述**：
- 空列表时只显示表头，无任何提示
- 影响命令：tasks list/next, events list, sessions list, phase list

**解决方案**：
1. 在各个list命令的输出逻辑中检测空列表
2. 在表头后添加友好提示和下一步建议
3. JSON格式保持不变（只影响text格式）

**提示信息设计**：
```
# tasks list (empty)
ID  STATUS  OWNER  CLAIM  PRIORITY  PHASE  UPDATED  TITLE

No tasks found. Create one with 'adp tasks add --workspace <name> "<title>"'

# events list (empty)
TIME  TYPE  WORKSPACE  AGENT  SESSION  TASK  EXIT  RUNTIME

No events recorded yet. Events are created when you run agents with 'adp run'

# sessions list (empty)
SESSION  WORKSPACE  AGENT  PROFILE  TASK  STARTED  FINISHED  EXIT  DURATION  EVENTS  RUNTIME

No sessions found. Start an agent with 'adp run <agent> --workspace <name>'

# phase list (empty)
ID  STATUS  UPDATED  TITLE

No phases defined. Add one with 'adp phase add --workspace <name> <id> "<title>"'
```

**修改文件**：
- `internal/cli/tasks.go` - 添加空列表提示
- `internal/cli/events.go` - 添加空列表提示
- `internal/cli/sessions.go` - 添加空列表提示
- `internal/cli/phase.go` - 添加空列表提示
- 相关测试 - 验证提示信息

**验证**：
- 测试空列表场景，确认提示显示
- 测试非空列表，确认不显示提示
- JSON格式不受影响

**独立性**：✅ 完全独立，不依赖其他任务

---

### Task P1-2: workspace show支持--format参数

**问题描述**：
- `workspace list`使用表格格式
- `workspace show`使用YAML风格
- 输出格式不统一

**解决方案**：
1. 为`workspace show`添加`--format`参数（text/json）
2. text格式保持当前YAML风格，或改为表格
3. json格式输出结构化数据

**输出格式设计**：

JSON格式：
```json
{
  "name": "game-a",
  "project_root": "/srv/game-a",
  "workspace_dir": "/home/user/.adp/workspaces/game-a",
  "memory_enabled": true,
  "mcp_enabled": true,
  "agents": {
    "codex": {
      "command": "codex",
      "default_profile": "senior-engineer"
    },
    "claude": {
      "command": "claude",
      "default_profile": "architect"
    }
  },
  "profiles": ["default", "senior-engineer", "architect"],
  "prompts": ["base.md", "coding-standards.md"],
  "memory_files": 5,
  "mcp_servers": ["github", "postgres"]
}
```

**修改文件**：
- `internal/cli/workspace.go` - 添加format参数
- `internal/workspace/show.go` - 实现JSON输出
- `internal/commandmeta/examples.go` - 添加示例

**验证**：
- text和json格式都能正确显示
- 信息完整准确

**独立性**：✅ 完全独立，不依赖其他任务

---

### Task P1-3: 在adp --help中添加项目描述

**问题描述**：
- 根帮助信息只有命令列表
- 缺少项目简介和快速入门链接

**解决方案**：
1. 在`internal/cli/root.go`或帮助生成逻辑中添加描述头
2. 添加简短的项目描述（1-2句话）
3. 添加文档链接

**帮助信息设计**：
```
adp - Agent Development Platform

Manage AI agent workspaces, tasks, and runtime environments.
Keep agent configuration outside project roots with runtime overlays.

Documentation: https://github.com/karoc/adp
Quick start: https://github.com/karoc/adp#quick-start

Usage:
  adp init
  adp doctor [workspace] [--verbose] [--format <text|json>]
  ...
```

**修改文件**：
- `internal/cli/root.go` - 添加描述文本
- 或 `cmd/adp/main.go` - 如果帮助在主函数生成

**验证**：
- `adp --help`显示新的描述
- `adp`（无参数）显示相同信息

**独立性**：✅ 完全独立，不依赖其他任务

---

### Task P1-4: 改进init成功后的引导信息

**问题描述**：
- `adp init`成功后只输出"initialized ADP home"
- 新用户不知道下一步做什么

**解决方案**：
1. 在`internal/cli/init.go`中扩展成功消息
2. 添加下一步操作建议
3. 根据是否首次初始化显示不同信息

**输出信息设计**：
```
initialized ADP home at /home/user/.adp

Next steps:
  1. Add a workspace:
     adp workspace add <name> /path/to/project

  2. Check diagnostics:
     adp doctor

  3. See all commands:
     adp --help

Documentation: https://github.com/karoc/adp#quick-start
```

**修改文件**：
- `internal/cli/init.go` - 扩展输出信息

**验证**：
- 首次init显示完整引导
- 重复init不应显示（或显示简化版本）

**独立性**：✅ 完全独立，不依赖其他任务

---

### Task P1-5: workspace add时先检查名称冲突

**问题描述**：
- 添加重名workspace时，错误提示混淆（显示路径检查错误）
- 应该先检查名称，给出明确提示

**解决方案**：
1. 在`internal/cli/workspace.go`的add逻辑中，先检查名称是否存在
2. 再检查路径是否有效
3. 提供清晰的错误提示

**错误提示设计**：
```
# 名称冲突
adp: workspace "game-a" already exists
  current project root: /srv/game-a
  workspace dir: /home/user/.adp/workspaces/game-a

Use a different name or remove the existing workspace with:
  adp workspace remove game-a

# 路径不存在
adp: project root does not exist: /tmp/invalid-path

Create the directory first or check the path.
```

**修改文件**：
- `internal/cli/workspace.go` - 调整验证顺序
- `internal/workspace/add.go` - 改进错误处理

**验证**：
- 测试名称冲突场景
- 测试路径无效场景
- 确保错误提示清晰

**独立性**：✅ 完全独立，不依赖其他任务

---

## Phase 3: P2低优先级改进（高级功能）

**目标**：进一步提升用户体验
**并行策略**：2个任务，P2-2依赖多个模块，建议串行

### Task P2-1: 添加交互式引导命令（adp quickstart）

**问题描述**：
- 新用户需要记住多个命令才能完成初始设置
- 缺少交互式引导体验

**解决方案**：
1. 添加新命令`adp quickstart`
2. 交互式询问项目路径、workspace名称等
3. 自动执行init、workspace add、基础配置
4. 提供可选的workspace模板选择

**交互流程设计**：
```
$ adp quickstart

Welcome to ADP (Agent Development Platform)!

This wizard will help you set up your first workspace.

? ADP home directory [/home/user/.adp]: 
✓ Initialized ADP home

? Workspace name: my-project
? Project root: /srv/my-project
? Enable memory? (Y/n): y
? Enable MCP? (Y/n): y
? Which agents will you use?
  [x] codex
  [x] claude
  [ ] custom

✓ Workspace "my-project" created

? Run workspace diagnostics now? (Y/n): y
✓ No issues found

Next steps:
  - Start an agent: adp run codex --workspace my-project
  - Add a task: adp tasks add --workspace my-project "First task"
  - See all commands: adp --help
```

**修改文件**：
- `internal/cli/quickstart.go` - 新文件，实现交互逻辑
- `cmd/adp/main.go` - 注册新命令
- 需要引入交互式输入库（如survey或promptui）

**验证**：
- 完整走通quickstart流程
- 验证生成的workspace配置正确
- 测试各种用户输入场景（空值、取消等）

**独立性**：⚠️ 需要P0-3（版本信息）和P1-4（init引导）完成后效果更好，但可独立开发

---

### Task P2-2: 优化ID格式的可读性

**问题描述**：
- task ID: `task-20260611-0001`（日期格式冗长）
- session ID: `20260613T020120-270f1f79`（时间戳复杂）
- 不易记忆和输入

**解决方案**（多个选项）：

**选项A：保持ID格式，改进显示**
- 生成逻辑不变
- 在显示时格式化（如显示相对时间）
- 支持短ID前缀匹配

**选项B：简化ID格式**
- task ID: `task-001`, `task-002`（简单序号）
- session ID: `sess-abc123`（短hash）
- 需要维护全局计数器或改用随机短ID

**选项C：支持别名**
- 允许用户为task/session设置别名
- 通过别名引用ID

**推荐方案**：选项A（最小改动）
1. 保持现有ID生成逻辑（兼容性）
2. 实现短ID前缀匹配（如`task-202`可匹配`task-20260611-0001`）
3. 在list命令中支持`--short`显示简化版
4. 补全功能支持前缀匹配

**修改文件**：
- `internal/tasks/id.go` - 添加ID匹配逻辑
- `internal/sessions/id.go` - 添加ID匹配逻辑
- `internal/cli/*.go` - 各个接受ID参数的命令支持前缀匹配
- 补全逻辑 - 支持前缀补全

**验证**：
- 测试完整ID仍然工作
- 测试短前缀能正确匹配
- 测试歧义前缀给出清晰错误
- 测试补全功能

**独立性**：⚠️ 影响多个模块，建议在其他任务完成后进行

---

## 并行执行策略

### Wave 1: P0任务（可完全并行）
```
Agent-1: P0-1 (workspace list json)
Agent-2: P0-2 (agent error提示)  
Agent-3: P0-3 (version信息)
```
预计完成时间：2-4小时
验证：`scripts/check-all.sh`

### Wave 2: P1任务（可完全并行）
```
Agent-1: P1-1 (空列表提示)
Agent-2: P1-2 (workspace show格式)
Agent-3: P1-3 (帮助描述)
Agent-4: P1-4 (init引导)
Agent-5: P1-5 (名称冲突检查)
```
预计完成时间：2-4小时
验证：`scripts/check-all.sh`

### Wave 3: P2任务（建议串行）
```
Agent-1: P2-1 (quickstart命令)
等待验证通过后：
Agent-1: P2-2 (ID可读性)
```
预计完成时间：4-8小时
验证：`scripts/check-all.sh` + 手动测试

---

## 任务分配规范

每个任务agent应该：

1. **开始前**
   - 阅读本PLAN.md了解任务上下文
   - 阅读AGENTS.md了解项目约束
   - 确认没有其他agent在处理相同的文件

2. **实施中**
   - 严格按照700行文件限制
   - 保持双语文档同步
   - 添加相应的测试
   - 更新commandmeta/examples.go（如果涉及命令变更）

3. **完成后**
   - 运行`scripts/check-all.sh`
   - 提供修改文件列表
   - 提供测试验证结果
   - 报告是否发现新问题

---

## 验收标准

### P0任务验收
- [ ] workspace list支持--format json且输出正确
- [ ] agent不存在时错误提示清晰易懂
- [ ] version命令显示完整构建信息
- [ ] 所有smoke测试通过

### P1任务验收
- [ ] 所有空列表场景显示友好提示
- [ ] workspace show支持--format参数
- [ ] 根帮助包含项目描述和链接
- [ ] init成功后显示下一步指引
- [ ] workspace add错误提示清晰
- [ ] 所有smoke测试通过

### P2任务验收
- [ ] quickstart命令可交互完成初始化
- [ ] ID前缀匹配功能正常工作
- [ ] 补全支持短ID
- [ ] 所有smoke测试通过
- [ ] 手动测试新用户流程顺畅

---

## 风险与依赖

### 技术风险
- **低风险**: P0和P1任务都是常规CLI改进
- **中风险**: P2-1需要引入新依赖（交互库）
- **中风险**: P2-2涉及多个模块，可能引入ID解析bug

### 依赖关系
- P0任务：完全独立，无依赖
- P1任务：完全独立，无依赖
- P2-1：建议等P1-4完成（更好的一致性）
- P2-2：建议最后进行（影响范围大）

### 资源分配
- 5个并行agent可在4小时内完成P0+P1
- 1个agent可在1天内完成P2
- 总预计：1个工作日完成所有任务

---

## 后续计划

完成本计划后，建议：

1. **再次进行易用性测试**
   - 验证所有改进是否有效
   - 收集新的反馈

2. **更新文档**
   - 更新README.md反映新功能
   - 更新operator-onboarding.md包含quickstart
   - 更新双语文档

3. **发布新版本**
   - 标记为易用性增强版本
   - 准备release notes

4. **持续改进**
   - 监控用户反馈
   - 规划下一轮改进
