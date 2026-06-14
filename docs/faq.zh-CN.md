# ADP FAQ

> Agent Development Platform 常见问题

English: [faq.md](faq.md)

本 FAQ 涵盖关于 ADP 架构、使用决策和集成场景的概念性问题。关于错误解决和诊断流程，请参阅[故障排除指南](troubleshooting.zh-CN.md)。

---

## 目录

1. [核心概念](#核心概念)
2. [使用决策](#使用决策)
3. [团队协作](#团队协作)
4. [集成场景](#集成场景)
5. [高级主题](#高级主题)

---

## 核心概念

### Q1: 什么是 ADP？为什么要使用它？

**简短回答：**

ADP（Agent Development Platform，智能体开发平台）是一个终端优先的工作空间管理器，它将 AI 智能体配置文件放在项目目录之外。当你需要管理多个项目、需要可复用的智能体配置，或希望保持项目根目录整洁时，可以使用它。

**详细说明：**

如果不使用 ADP，直接运行 `codex` 或 `claude` 会在项目根目录创建配置文件，如 `AGENTS.md`、`CLAUDE.md`、`.codex/` 和 `.claude/`。这会导致：
- Git 噪音：`git status` 中出现未跟踪的智能体文件
- 配置冲突：在智能体或项目间切换时的冲突
- 手动清理和 .gitignore 维护

ADP 通过以下方式解决这些问题：
1. **工作空间管理**：将智能体配置（配置文件、提示词、记忆、MCP 设置）存储在 `~/.adp` 中，而不是项目根目录
2. **运行时覆盖层**：创建临时目录，将项目文件（通过符号链接）与生成的智能体文件结合
3. **可复用性**：一次定义配置，多次复用于多个智能体运行
4. **团队一致性**：以模板形式共享工作空间配置，而不提交本地状态

**何时使用 ADP：**
- 需要不同智能体配置的多个项目
- 团队环境需要一致的智能体设置
- 项目中保持根目录整洁很重要
- 工作流需要任务跟踪和所有权管理

**何时不使用 ADP：**
- 一次性探索性智能体会话
- 单一项目，无需配置复用
- 快速实验，设置开销不值得

**示例：**

```bash
# 不使用 ADP：智能体文件污染项目根目录
cd /srv/my-project
codex
# 创建：/srv/my-project 中的 AGENTS.md、.codex/

# 使用 ADP：项目根目录保持整洁
adp workspace add my-project /srv/my-project
adp run codex --workspace my-project
# 配置在 ~/.adp，运行时在 /tmp，项目未改变
```

**另见：**
- [README：运行时模型](../README.zh-CN.md#运行时模型)
- [操作员入门指南](operator-onboarding.zh-CN.md)
- [Q7：何时使用 ADP 而不是直接运行 CLI？](#q7-何时应该使用-adp-而不是直接运行-codex-或-claude)

---

### Q2: 什么是工作空间？

**简短回答：**

工作空间是为在特定项目中运行智能体而命名的配置集。它定义了项目位置、要使用的智能体配置文件、要注入的提示词以及要启用的 MCP 服务器。

**详细说明：**

工作空间存储在 `$ADP_HOME/workspaces/<workspace-name>/` 目录下，包含：

- **workspace.yaml**：核心配置（项目根路径、适配器设置）
- **profiles/**：智能体特定配置（codex-architect.yaml、claude-reviewer.yaml）
- **prompts/**：可复用指令文件（base-prompt.md、task-instructions.md）
- **memory/**：共享上下文文件（project-conventions.md、architecture-decisions.md）
- **mcp/**：MCP 服务器配置（servers.json）
- **planning/**：任务和阶段状态（tasks.yaml、phases.yaml、progress.jsonl）

**类比：** 工作空间之于智能体配置，就像 Git 远程仓库之于仓库 URL——是一个命名引用，使重复操作变得方便。

**示例：**

```bash
# 创建工作空间
adp workspace add game-a /srv/game-a

# 工作空间结构
~/.adp/workspaces/game-a/
├── workspace.yaml
├── profiles/
│   ├── codex-architect.yaml
│   └── claude-engineer.yaml
├── prompts/
│   └── base-prompt.md
└── planning/
    └── tasks.yaml

# 使用工作空间
adp run codex --workspace game-a
adp run claude --workspace game-a --profile engineer
```

**常见陷阱：**
- ⚠️ 工作空间名称在每个 `$ADP_HOME` 中必须唯一
- ⚠️ 项目根路径必须是绝对路径
- ⚠️ 工作空间配置是本地的；以模板形式共享，不要提交 `$ADP_HOME`

**另见：**
- [examples/basic-workspace](../examples/basic-workspace/) - 可复制的工作空间模板
- [Q12：如何共享工作空间配置？](#q12-如何与团队共享工作空间配置)

---

### Q3: 什么是运行时覆盖层？

**简短回答：**

运行时覆盖层是一个临时目录，它将真实项目文件（通过符号链接）与 ADP 生成的智能体配置文件（AGENTS.md、CLAUDE.md、.codex/、.claude/）结合在一起。智能体在此覆盖层中工作，保持实际项目目录的整洁。

**工作原理：**

当你运行 `adp run <agent>` 时，ADP 在 `$ADP_RUNTIME_DIR` 下创建一个临时目录，构建一个看起来像项目根目录的视图：
- 生成的文件（AGENTS.md、CLAUDE.md、.codex/config.toml、.claude/settings.json）是写入覆盖层的真实文件
- 项目文件（源代码、go.mod、package.json 等）是从真实项目符号链接的
- 智能体的工作目录设置为此覆盖层根目录

**为什么重要：**

如果没有覆盖层，智能体会将配置文件直接写入项目。这会导致：
- Git 噪音：未跟踪的 AGENTS.md、CLAUDE.md 污染 `git status`
- 冲突：不同智能体覆盖彼此的配置
- 清理负担：记住要 .gitignore 智能体特定路径

运行时覆盖层通过为每次智能体运行提供独立视图和一致配置来解决这个问题，然后自动清理。

**示例：**

```bash
# 真实项目
/srv/my-project/
├── go.mod
├── main.go
└── internal/

# 运行时覆盖层（在 `adp run codex` 期间创建）
/tmp/adp-runtime/my-project-20260614T120000-abc123/
├── AGENTS.md              # 生成的
├── .codex/                # 生成的
│   └── config.toml
├── go.mod -> /srv/my-project/go.mod        # 符号链接
├── main.go -> /srv/my-project/main.go      # 符号链接
└── internal -> /srv/my-project/internal    # 符号链接

# 智能体在这里工作 ↑，真实项目未改变 ↓
```

**生命周期：**

1. `adp run` 在启动智能体前创建覆盖层
2. 智能体将覆盖层视为工作目录（`$ADP_RUNTIME_ROOT`）
3. 退出时，ADP 删除覆盖层（除非使用了 `--keep-runtime`）
4. 使用 `adp runtime prune` 清理保留的或陈旧的覆盖层

**常见陷阱：**
- ⚠️ 智能体退出后不要依赖覆盖层内容（默认会被删除）
- ⚠️ 不要将 `$ADP_RUNTIME_DIR` 指向项目内部（doctor 会警告）
- ⚠️ 保留的运行时会占用磁盘空间；定期用 `adp runtime prune` 清理

**另见：**
- [README：运行时模型](../README.zh-CN.md#运行时模型) - 详细的覆盖层机制
- [Q4：ADP 创建哪些文件？存放在哪里？](#q4-adp-创建哪些文件存放在哪里)
- [Q10：应该使用 --keep-runtime 吗？](#q10-应该使用---keep-runtime-还是让-adp-清理)

---

### Q4: ADP 创建哪些文件？存放在哪里？

**简短回答：**

ADP 在两个位置创建文件：`$ADP_HOME`（默认 `~/.adp`）下的持久配置和 `$ADP_RUNTIME_DIR`（默认 `/tmp/adp-runtime`）下的临时运行时覆盖层。你的真实项目根目录**永远不会**被修改。

**文件位置：**

| 文件类型 | 位置 | 生命周期 | 用途 |
|---------|------|---------|------|
| 工作空间配置 | `$ADP_HOME/workspaces/<name>/` | 持久 | 配置文件、提示词、记忆、MCP 设置 |
| 任务状态 | `$ADP_HOME/workspaces/<name>/planning/` | 持久 | tasks.yaml、phases.yaml、progress.jsonl |
| 事件日志 | `$ADP_HOME/logs/events.jsonl` | 持久 | 会话历史、运行时事件 |
| 运行时覆盖层 | `$ADP_RUNTIME_DIR/<name>-<timestamp>/` | 临时 | AGENTS.md、CLAUDE.md、.codex/、.claude/、符号链接 |
| 项目文件 | 项目根（未改变） | N/A | 你的源代码、配置等 |

**详细说明：**

**$ADP_HOME（持久状态）**
- 默认：`~/.adp`
- 显式设置：`export ADP_HOME=/path/to/adp-home`
- 包含工作空间注册表、任务账本、会话日志
- 重启后保留；可长期保存
- 备份此目录以保留工作空间配置

**$ADP_RUNTIME_DIR（临时覆盖层）**
- 默认：`$TMPDIR/adp-runtime` 或 `/tmp/adp-runtime`
- 显式设置：`export ADP_RUNTIME_DIR=/path/to/runtime`
- 仅包含活动的或保留的运行时目录
- 自动清理，除非使用 `--keep-runtime`
- 用 `adp runtime prune --older-than 24h` 清理

**项目根（永不修改）**
- ADP 永远不会向项目写入 AGENTS.md、CLAUDE.md、.codex/、.claude/ 或规划文件
- 智能体通过运行时覆盖层中的符号链接看到项目文件
- `adp workspace doctor` 验证没有 ADP 文件泄漏到项目根

**示例检查：**

```bash
# 检查 ADP 主目录结构
ls -la ~/.adp/
# workspaces/  logs/

# 检查运行时覆盖层
ls -la /tmp/adp-runtime/
# game-a-20260614T120000-abc123/  （如果保留）

# 验证项目根是干净的
cd /srv/my-project
ls -la
# 这里没有 AGENTS.md、CLAUDE.md、.codex/、.claude/
```

**另见：**
- [操作员入门：隔离的首次运行](operator-onboarding.zh-CN.md#隔离的首次运行)
- [Q3：什么是运行时覆盖层？](#q3-什么是运行时覆盖层)

---

### Q5: ADP 与 Codex 和 Claude 的关系是什么？

**简短回答：**

ADP 是智能体 CLI 的运行时环境管理器，不是替代品。Codex 和 Claude 是执行工作的智能体；ADP 提供一致的配置、工作空间隔离和跨多个智能体运行的任务协调。

**类比：**

ADP 之于智能体 CLI，就像 Docker 之于应用程序进程：
- Docker 管理容器环境；ADP 管理运行时覆盖层
- Docker 隔离进程状态；ADP 隔离智能体配置
- Docker 不替代你的应用；ADP 不替代 Codex 或 Claude

**ADP 做什么：**
1. **配置管理**：从工作空间模板生成 AGENTS.md、CLAUDE.md、.codex/config.toml、.claude/settings.json
2. **运行时隔离**：构建临时覆盖层，使智能体看到一致的配置，而不污染项目根
3. **任务协调**：跨智能体运行跟踪工作所有权、租约和交接
4. **会话历史**：记录事件并为跨智能体工作流提供恢复指导

**Codex/Claude 做什么：**
- 处理自然语言指令
- 读取和修改项目文件
- 执行代码、运行测试、调试问题
- 提供交互式智能体体验

**示例工作流：**

```bash
# ADP 设置环境
adp run codex --workspace game-a --task task-001
# → ADP 用生成的配置创建运行时覆盖层
# → ADP 启动：codex --config /tmp/.../codex/config.toml
# → Codex 在该环境中执行实际工作

# 稍后，交接给 Claude
adp sessions resume-plan <session-id> --agent claude --owner reviewer
# → ADP 建议：adp run claude --workspace game-a --task task-001
# → Claude 在具有相同任务上下文的新运行时中继续
```

**另见：**
- [README：核心价值主张](../README.zh-CN.md)
- [真实智能体兼容性](real-agent-compatibility.zh-CN.md)

---

### Q6: 什么是任务和阶段？

**简短回答：**

任务是具有所有权和租约的单个工作项。阶段是发布纪律的阶段门（规划 → 接受 → 提交 → 推送）。两者都是可选的——你可以在不使用任务管理的情况下使用 `adp run`。

**任务：**

任务是在 `$ADP_HOME/workspaces/<name>/planning/tasks.yaml` 中跟踪的工作项。每个任务有：
- **ID**：唯一标识符（task-20260614-0001）
- **标题**：简要描述
- **状态**：pending → in_progress → completed
- **所有者**：谁在处理它（可选）
- **租约**：所有权持续多长时间（可选，例如 2h、30m）
- **优先级**：high、normal、low
- **阶段**：它属于哪个发布阶段

**阶段：**

阶段为结构化发布工作流强制执行阶段门。常见阶段：
1. **planning**：定义和确定工作范围
2. **implementation**：构建功能、修复错误
3. **acceptance**：审查和验证
4. **commit**：在 Git 中记录更改
5. **push**：发布到远程

阶段门防止跳过步骤：不能在接受前提交，不能在提交前推送等。

**何时使用任务：**
- 多智能体协调（多个工作者从共享看板中选择）
- 工作交接（操作员 A 开始，操作员 B 继续）
- 审计追踪（谁在何时做了什么）
- 租约管理（防止并发工作流中的冲突）

**何时使用阶段：**
- 结构化发布纪律
- 需要审查门的团队工作流
- 合规需求（发布前的接受证据）

**示例：**

```bash
# 创建任务
adp tasks add --workspace game-a --phase implementation "Fix auth bug"
# task-20260614-0001 added

# 认领并处理它
adp run codex --workspace game-a --take --owner alice --lease 2h

# 检查进度
adp tasks show task-20260614-0001
# 状态：in_progress，所有者：alice，过期：2026-06-14 14:00

# 完成任务
adp tasks done task-20260614-0001
```

**另见：**
- [任务管理指南](task-management.zh-CN.md)
- [Q8：何时使用任务而不是直接 adp run？](#q8-何时应该使用任务而不是直接-adp-run)
- [Q14：多个操作员如何协调共享任务？](#q14-多个操作员如何协调共享任务)

---

## 使用决策

### Q7: 何时应该使用 ADP 而不是直接运行 `codex` 或 `claude`？

**简短回答：**

当你需要可复用的配置、多项目管理、团队一致性或任务跟踪时使用 ADP。对于一次性探索和快速实验使用直接 CLI。

**决策矩阵：**

| 场景 | 使用 ADP | 使用直接 CLI |
|------|---------|-------------|
| 具有不同配置的多个项目 | ✅ | ❌ |
| 可复用的智能体配置文件（架构师、审查者） | ✅ | ❌ |
| 团队需要一致设置 | ✅ | ❌ |
| 任务跟踪和所有权 | ✅ | ❌ |
| 保持项目根整洁 | ✅ | ❌ |
| 一次性探索 | ❌ | ✅ |
| 快速实验（无需设置） | ❌ | ✅ |
| 无需复用的交互式会话 | ❌ | ✅ |

**使用 ADP 的场景：**
1. 在 2 个以上需要不同智能体配置的项目上工作
2. 团队成员需要共享一致的智能体设置
3. 你希望项目根目录不含 AGENTS.md、CLAUDE.md、.codex/、.claude/
4. 多个智能体在具有所有权跟踪的共享任务上协调
5. 你需要谁做了什么的审计追踪

**使用直接 CLI 的场景：**
1. 第一次探索新工具或项目
2. 运行无需复用的快速一次性命令
3. 交互式实验，设置开销不值得
4. 没有配置复杂性的单一项目

**示例对比：**

```bash
# 直接 CLI：快速但在项目中留下文件
cd /srv/my-project
codex
# 创建：/srv/my-project/AGENTS.md、/srv/my-project/.codex/

# ADP：更多设置，但项目保持整洁
adp workspace add my-project /srv/my-project
adp run codex --workspace my-project
# 配置在 ~/.adp，运行时在 /tmp，/srv/my-project 未改变
```

**另见：**
- [Q1：什么是 ADP？为什么要使用它？](#q1-什么是-adp为什么要使用它)
- [操作员入门指南](operator-onboarding.zh-CN.md)

---

### Q8: 何时应该使用任务而不是直接 `adp run`？

**简短回答：**

当你需要工作协调、所有权跟踪或审计追踪时使用任务。对于探索性工作或不需要跟踪的一次性执行使用直接 `adp run`。

**使用任务：**

好处：
- 所有权跟踪（谁在做什么）
- 租约管理（防止并发工作冲突）
- 工作交接（开始任务，交给另一个操作员）
- 进度可见性（`adp tasks list`、`adp progress report`）
- 会话历史链接到任务上下文
- 基于看板工作流的优先级排序

工作流：
```bash
# 添加任务
adp tasks add --workspace game-a "Fix auth bug"

# 从看板中选择
adp run codex --workspace game-a --take --owner alice --lease 2h

# 检查进度
adp tasks show task-20260614-0001

# 完成
adp tasks done task-20260614-0001
```

**不使用任务：**

好处：
- 更简单的工作流（无需任务管理开销）
- 更快速的一次性工作
- 不需要所有权协调

工作流：
```bash
# 只运行智能体
adp run codex --workspace game-a
```

**决策指南：**

使用任务的情况：
- 多个智能体/操作员在共享工作上协调
- 你需要"谁做了这个？"审计追踪
- 工作跨越多个会话或交接
- 防止并发工作冲突很重要

跳过任务的情况：
- 探索性工作（无需复用或跟踪）
- 单个操作员，不需要协调
- 无需交接的交互式会话
- 快速修复，开销不值得

**另见：**
- [任务管理指南](task-management.zh-CN.md)
- [Q6：什么是任务和阶段？](#q6-什么是任务和阶段)
- [Q9：如何在 --task 和 --take 之间选择？](#q9-如何在-adp-run---task-和-adp-run---take-之间选择)

---

### Q9: 如何在 `adp run --task` 和 `adp run --take` 之间选择？

**简短回答：**

当你确切知道要处理哪个任务时使用 `--task <id>`。使用 `--take --owner <owner> --lease <duration>` 从看板原子性地认领第一个可用任务并启动智能体。

**三种模式：**

**1. 显式任务分配（`--task <id>`）**

何时使用：
- 你或前一步已经识别了具体任务
- 任务已预先分配给你
- 在已知任务上重新运行工作

```bash
# 显式任务目标
adp run codex --workspace game-a --task task-20260614-0001
```

**2. 原子看板提取（`--take --owner --lease`）**

何时使用：
- 工作者应从看板中选择第一个可用任务
- 在一个命令中原子性认领 + 启动
- 防止并发工作流中的竞态条件

```bash
# 认领第一个可用任务并启动
adp run codex --workspace game-a --take --owner alice --lease 2h
```

**3. 不启动认领（`adp tasks take`）**

何时使用：
- 认领任务但不立即启动智能体
- 在开始工作前审查任务
- 批量认领以供以后执行

```bash
# 先认领，稍后运行
TASK_ID=$(adp tasks take --workspace game-a --owner alice --lease 2h | grep -o 'task-[^ ]*')
# 审查任务详情
adp tasks show $TASK_ID
# 准备好后启动
adp run codex --workspace game-a --task $TASK_ID
```

**决策流程：**

```
你知道具体的任务 ID 吗？
├─ 是 → 使用 --task <id>
└─ 否 → 需要在开始前审查吗？
    ├─ 是 → 先使用 `adp tasks take`，然后 --task
    └─ 否 → 使用 --take --owner --lease（原子性）
```

**示例场景：**

```bash
# 场景 1：预先分配的任务
# 团队负责人："Alice，处理 task-20260614-0001"
adp run codex --workspace game-a --task task-20260614-0001

# 场景 2：工作者从看板中选择
adp run codex --workspace game-a --take --owner alice --lease 4h

# 场景 3：开始前审查
adp tasks take --workspace game-a --owner alice --lease 2h
# task-20260614-0002 taken
adp tasks show task-20260614-0002
# （审查详情）
adp run codex --workspace game-a --task task-20260614-0002
```

**另见：**
- [任务管理指南：工作流概览](task-management.zh-CN.md#工作流概览)
- [Q8：何时使用任务而不是直接 adp run？](#q8-何时应该使用任务而不是直接-adp-run)

---

### Q10: 应该使用 `--keep-runtime` 还是让 ADP 清理？

**简短回答：**

默认让 ADP 清理（退出时自动清理）。仅在调试运行时问题或需要手动检查生成的文件时使用 `--keep-runtime`。

**何时保留运行时：**

使用 `--keep-runtime` 的情况：
- 调试运行时覆盖层构建问题
- 手动检查生成的 AGENTS.md、CLAUDE.md、适配器配置
- 验证符号链接结构
- 排除智能体启动失败

```bash
# 保留运行时以供检查
adp run codex --workspace game-a --keep-runtime
# 运行时保留在：/tmp/adp-runtime/game-a-20260614T120000-abc123

# 检查生成的文件
ls -la /tmp/adp-runtime/game-a-20260614T120000-abc123
cat /tmp/adp-runtime/game-a-20260614T120000-abc123/AGENTS.md
```

**何时让 ADP 清理：**

默认行为（无 `--keep-runtime`）的情况：
- 正常工作流（无需调试）
- CI/CD 管道（每次运行干净状态）
- 自动化智能体运行
- 磁盘空间受限

```bash
# 自动清理（默认）
adp run codex --workspace game-a
# 退出时删除运行时
```

**管理保留的运行时：**

保留的运行时会占用磁盘空间。定期清理：

```bash
# 预览将被删除的内容
adp runtime prune --older-than 24h --dry-run

# 实际删除旧运行时
adp runtime prune --older-than 24h

# 删除所有保留的运行时（小心！）
adp runtime prune --older-than 0s --include-kept
```

**常见陷阱：**
- ⚠️ 保留的运行时不会自动过期；手动清理
- ⚠️ 修复问题后不要依赖保留的运行时内容
- ⚠️ 在 CI/CD 中，始终清理：`adp runtime prune --older-than 0s`

**另见：**
- [Q3：什么是运行时覆盖层？](#q3-什么是运行时覆盖层)
- [README：运行时清理](../README.zh-CN.md#运行时模型)

---

### Q11: 何时应该使用配置文件而不是工作空间默认值？

**简短回答：**

对特定角色的配置（架构师、工程师、审查者）使用配置文件。对适用于所有运行的一致团队基线使用工作空间默认值。

**配置层次：**

配置是分层的：**配置文件覆盖工作空间覆盖适配器默认值**

```
适配器默认值（内置）
    ↓
工作空间默认值（workspace.yaml）
    ↓
配置文件设置（profiles/architect.yaml）
```

**何时使用工作空间默认值：**

在 `workspace.yaml` 中设置适用于所有智能体运行的基线配置：
- 整个项目的共享基础提示词
- 所有智能体应看到的通用记忆文件
- 共享 MCP 服务器配置
- 默认模型设置

示例 workspace.yaml：
```yaml
codex:
  base_prompt: prompts/base-prompt.md
  memory:
    - memory/project-conventions.md
  mcp:
    servers: mcp/servers.json
```

**何时使用配置文件：**

在 `profiles/` 中为特定角色变体创建配置文件：
- 不同的指令集（架构 vs 实现）
- 特定角色的记忆（高级工程师上下文 vs 代码审查指南）
- 不同的模型设置（简单任务用快速模型，架构用强大模型）

示例配置文件：
```yaml
# profiles/architect.yaml
codex:
  base_prompt: prompts/architect-prompt.md
  memory:
    - memory/architecture-decisions.md
    - memory/design-patterns.md

# profiles/reviewer.yaml
codex:
  base_prompt: prompts/reviewer-prompt.md
  memory:
    - memory/code-review-checklist.md
```

**使用方法：**

```bash
# 使用工作空间默认值
adp run codex --workspace game-a

# 使用特定配置文件
adp run codex --workspace game-a --profile architect
adp run codex --workspace game-a --profile reviewer
```

**决策指南：**

使用工作空间默认值的情况：
- 配置适用于所有智能体运行
- 团队需要一致的基线
- 单一工作流，无角色变化

使用配置文件的情况：
- 不同角色需要不同配置
- 相同智能体，不同上下文（审查 vs 实现）
- 在不改变基线的情况下试验变化

**另见：**
- [examples/basic-workspace](../examples/basic-workspace/) - 配置文件示例
- [Q2：什么是工作空间？](#q2-什么是工作空间)

---

## 团队协作

### Q12: 如何与团队共享工作空间配置？

**简短回答：**

将示例工作空间配置提交到项目仓库（例如 `docs/adp-workspace-example/`）或使用 `examples/basic-workspace` 作为模板。永远不要提交 `$ADP_HOME` 本身——它包含本地状态和会话日志。

**三种共享方法：**

**选项 1：项目仓库中的示例（推荐）**

将工作空间配置模板提交到项目中：

```bash
# 项目结构
my-project/
├── docs/
│   └── adp-workspace/
│       ├── workspace.yaml
│       ├── prompts/
│       │   └── base-prompt.md
│       ├── profiles/
│       │   ├── architect.yaml
│       │   └── engineer.yaml
│       └── memory/
│           └── conventions.md
└── README.md

# 团队成员设置
cd my-project
adp workspace add my-project $PWD
# 将配置从 docs/adp-workspace/ 复制到 ~/.adp/workspaces/my-project/
cp -r docs/adp-workspace/* ~/.adp/workspaces/my-project/
```

**选项 2：共享模板仓库**

在单独的仓库中维护工作空间模板：

```bash
# 模板仓库
workspace-templates/
├── golang-service/
│   ├── workspace.yaml
│   └── prompts/
└── react-app/
    ├── workspace.yaml
    └── prompts/

# 团队成员克隆并复制
git clone https://git.example.com/team/workspace-templates
adp workspace add my-project /srv/my-project
cp -r workspace-templates/golang-service/* ~/.adp/workspaces/my-project/
```

**选项 3：使用 ADP 的基本工作空间示例**

从 ADP 的内置示例复制：

```bash
# 找到 ADP 示例工作空间
ls -la /path/to/adp/examples/basic-workspace/

# 复制到你的工作空间
adp workspace add my-project /srv/my-project
cp -r /path/to/adp/examples/basic-workspace/* ~/.adp/workspaces/my-project/

# 编辑项目根
vim ~/.adp/workspaces/my-project/workspace.yaml
# 更新：project.root: /srv/my-project
```

**共享与不共享的内容：**

✅ **可安全提交：**
- workspace.yaml（带占位符路径的模板）
- prompts/（智能体指令）
- profiles/（特定角色配置）
- memory/（共享项目知识）
- mcp/servers.json（MCP 服务器配置）

❌ **永远不要提交：**
- `$ADP_HOME/` 整个目录
- planning/（tasks.yaml、phases.yaml - 本地状态）
- logs/events.jsonl（会话历史）
- 凭证、API 密钥、令牌
- 机器特定的绝对路径

**另见：**
- [examples/basic-workspace](../examples/basic-workspace/)
- [Q13：应该将 .adp/ 提交到 Git 吗？](#q13-应该将-adp-或-adp_home-提交到-git-吗)

---

### Q13: 应该将 `.adp/` 或 `$ADP_HOME` 提交到 Git 吗？

**简短回答：**

**不**，永远不要提交 `$ADP_HOME`（默认 `~/.adp`）——它包含本地状态、任务所有权和会话日志。**是的**，在项目的 `docs/` 或 `examples/` 目录中提交示例工作空间配置。

**$ADP_HOME 包含什么：**

```
~/.adp/
├── workspaces/
│   ├── game-a/
│   │   ├── workspace.yaml       # ✅ 配置（作为模板共享）
│   │   ├── prompts/             # ✅ 指令（可共享）
│   │   ├── profiles/            # ✅ 角色配置（可共享）
│   │   └── planning/            # ❌ 本地状态（不要共享）
│   │       ├── tasks.yaml       # 任务所有权 - 机器特定
│   │       ├── phases.yaml      # 阶段状态 - 本地
│   │       └── progress.jsonl   # 执行日志 - 本地
└── logs/
    └── events.jsonl             # ❌ 会话历史（仅本地）
```

**推荐的 .gitignore：**

```gitignore
# 在项目根 .gitignore 中

# ADP 本地状态 - 永远不要提交
.adp/
$ADP_HOME/
**/planning/tasks.yaml
**/planning/phases.yaml
**/planning/progress.jsonl
**/logs/events.jsonl

# 但确实提交示例配置
!docs/adp-workspace/
!examples/adp-workspace/
```

**重启后保留什么：**

`$ADP_HOME` 下的 ADP 状态是持久的：
- ✅ 工作空间注册表（工作空间列表、配置）
- ✅ 任务状态（所有权、状态、租约）
- ✅ 阶段状态（当前阶段、接受记录）
- ✅ 会话历史（事件、运行时日志）

但这些是**每台机器本地的**——以模板形式共享配置，而不是作为提交的状态。

**推荐的团队工作流：**

1. 将工作空间配置模板提交到 `docs/adp-workspace/`
2. 在项目 README 中添加设置说明
3. 每个团队成员将模板复制到本地 `$ADP_HOME`
4. 任务状态保持本地（通过租约管理协调）

**另见：**
- [Q12：如何共享工作空间配置？](#q12-如何与团队共享工作空间配置)
- [Q14：多个操作员如何协调共享任务？](#q14-多个操作员如何协调共享任务)

---

### Q14: 多个操作员如何协调共享任务？

**简短回答：**

操作员通过基于租约的所有权进行协调：用 `adp tasks take --owner <owner> --lease <duration>` 认领任务，用 `adp tasks next` 预览可用工作，用 `adp tasks stale` 检测过期租约，用 `adp tasks renew` 延长所有权。

**协调工作流：**

**1. 预览可用工作**

```bash
# 查看看板上有什么
adp tasks next --workspace game-a

# JSON 用于机器解析
adp tasks next --workspace game-a --format json
```

**2. 用租约认领任务**

```bash
# 原子性认领
adp tasks take --workspace game-a --owner alice --lease 2h
# task-20260614-0001 taken

# 或在一个命令中认领 + 启动
adp run codex --workspace game-a --take --owner alice --lease 2h
```

**3. 为长时间运行的工作维护所有权**

```bash
# 在租约过期前延长
adp tasks renew --workspace game-a task-20260614-0001 --owner alice --lease 2h
```

**4. 检测陈旧所有权**

```bash
# 查找过期的租约
adp tasks stale --workspace game-a
# task-20260614-0002: owner bob, expired 30m ago

# 重新认领过期的任务
adp tasks take --workspace game-a --owner alice --lease 2h
```

**所有权规则：**

- **待处理任务**：任何人都可以认领（先到先得）
- **进行中且有所有者**：只有所有者可以续约或释放
- **进行中，租约过期**：任何人都可以通过 `adp tasks take` 重新认领
- **已完成任务**：不可变（用于审计追踪）

**冲突解决：**

ADP 使用"租约边界内最后写入者获胜"：
- 所有者在租约期间拥有独占控制权
- 租约过期后，任务可供重新认领
- 无分布式锁定（通过本地 `$ADP_HOME` 状态协调）

**示例多操作员流程：**

```bash
# 操作员 Alice
adp tasks add --workspace game-a "Fix auth bug"
adp run codex --workspace game-a --take --owner alice --lease 4h
# 工作 2 小时，需要更多时间
adp tasks renew task-20260614-0001 --owner alice --lease 2h

# 操作员 Bob（不同机器）
adp tasks next --workspace game-a
# 看到：无可用任务（Alice 拥有 task-20260614-0001）

# 6 小时后，Alice 的租约过期
adp tasks stale --workspace game-a
# task-20260614-0001: owner alice, expired 1h ago

# Bob 认领陈旧任务
adp tasks take --workspace game-a --owner bob --lease 2h
# task-20260614-0001 taken by bob
```

**常见陷阱：**
- ⚠️ 租约不会自动续约；设置提醒或使用更长的持续时间
- ⚠️ 无跨机器状态同步；通过 Git 提交或外部跟踪器协调
- ⚠️ `adp tasks next` 仅显示你的本地视图；其他操作员的状态可能不同

**另见：**
- [任务管理指南：租约管理](task-management.zh-CN.md)
- [Q15：如何在智能体或操作员之间交接工作？](#q15-如何在智能体或操作员之间交接工作)

---

### Q15: 如何在智能体或操作员之间交接工作？

**简短回答：**

通过释放所有权（`adp tasks release`）、让租约过期或使用 `adp sessions resume-plan` 进行跨工具交接来交接工作。下一个操作员认领任务并审查会话历史以获取上下文。

**三种交接方法：**

**方法 1：显式释放（立即）**

```bash
# 操作员 Alice 释放任务
adp tasks release --workspace game-a task-20260614-0001 --owner alice

# 操作员 Bob 认领它
adp tasks take --workspace game-a --owner bob --lease 2h
# 或
adp run claude --workspace game-a --take --owner bob --lease 2h
```

**方法 2：租约过期（延迟）**

```bash
# 操作员 Alice 的租约自然过期（无需操作）
# 2 小时后，任务变为陈旧

# 操作员 Bob 检查陈旧任务
adp tasks stale --workspace game-a
# task-20260614-0001: owner alice, expired 15m ago

# Bob 认领过期任务
adp tasks take --workspace game-a --owner bob --lease 2h
```

**方法 3：跨工具恢复（相同任务，不同智能体）**

```bash
# 操作员 Alice 用 Codex 工作
adp run codex --workspace game-a --task task-20260614-0001 --owner alice --lease 2h
# 会话：session-20260614T120000-abc123

# Alice 释放任务
adp tasks release --workspace game-a task-20260614-0001 --owner alice

# 操作员 Bob 想用 Claude 继续
adp sessions resume-plan session-20260614T120000-abc123 \
  --agent claude --owner bob --lease 2h

# 建议的命令（复制并运行）：
adp run claude --workspace game-a --task task-20260614-0001 \
  --owner bob --lease 2h
```

**交接证据追踪：**

认领前，审查上下文：

```bash
# 检查任务详情
adp tasks show task-20260614-0001

# 审查会话历史
adp sessions list --workspace game-a --task task-20260614-0001

# 获取进度快照
adp progress report --workspace game-a --format json
```

**另见：**
- [会话恢复规划](session-restore.zh-CN.md)
- [Q20：会话恢复和继续如何工作？](#q20-会话恢复和继续如何工作)
- [Q14：多个操作员如何协调？](#q14-多个操作员如何协调共享任务)

---

## 集成场景

### Q16: 如何在 CI/CD 管道中使用 ADP？

**简短回答：**

设置隔离的 `$ADP_HOME`，以编程方式创建工作空间，使用 `adp run --take` 进行任务提取，并始终用 `adp runtime prune` 清理运行时。避免在 CI 运行之间持久化本地状态。

**CI/CD 最佳实践：**

**1. 使用隔离的临时状态**

```bash
# 不要在运行之间持久化
export ADP_HOME="${RUNNER_TEMP}/adp-home"
export ADP_RUNTIME_DIR="${RUNNER_TEMP}/adp-runtime"
```

**2. 以编程方式创建工作空间**

```bash
# 初始化新的 ADP 状态
adp init

# 创建工作空间
adp workspace add ci-workspace "${GITHUB_WORKSPACE}"

# 从仓库复制配置
cp -r .github/adp-workspace/* "${ADP_HOME}/workspaces/ci-workspace/"
```

**3. 使用任务跟踪运行智能体**

```bash
# 创建任务
TASK_ID=$(adp tasks add --workspace ci-workspace \
  --priority high "CI: Automated code review" | \
  grep -o 'task-[^ ]*')

# 运行智能体
adp run codex --workspace ci-workspace \
  --task "${TASK_ID}" \
  --owner ci-bot \
  --lease 30m
```

**4. 始终清理**

```bash
# 清理所有运行时
adp runtime prune --older-than 0s

# 验证清理
find "${ADP_RUNTIME_DIR}" -type d -name "ci-workspace-*"
```

**示例 GitHub Actions 工作流：**

```yaml
name: ADP Code Review

on: [pull_request]

jobs:
  review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup ADP
        run: |
          # 安装 ADP
          curl -L https://github.com/example/adp/releases/latest/adp -o /usr/local/bin/adp
          chmod +x /usr/local/bin/adp
          
          # 设置隔离状态
          echo "ADP_HOME=${RUNNER_TEMP}/adp-home" >> $GITHUB_ENV
          echo "ADP_RUNTIME_DIR=${RUNNER_TEMP}/adp-runtime" >> $GITHUB_ENV

      - name: Initialize Workspace
        run: |
          adp init
          adp workspace add ci-workspace "${GITHUB_WORKSPACE}"
          cp -r .github/adp-workspace/* "${ADP_HOME}/workspaces/ci-workspace/"

      - name: Run Review
        run: |
          adp run codex --workspace ci-workspace \
            --take --owner ci-bot --lease 30m \
            -- --review-pr ${{ github.event.pull_request.number }}

      - name: Cleanup
        if: always()
        run: adp runtime prune --older-than 0s
```

**关键考虑因素：**

- **身份验证**：确保 Codex/Claude CLI 凭证在 CI 环境中可用
- **隔离**：每次运行使用单独的 `$ADP_HOME`（不持久化任务状态）
- **清理**：始终清理运行时，即使失败（使用 `if: always()`）
- **超时**：为 CI 环境设置合理的租约持续时间
- **审计**：记录会话 ID 和任务 ID 以供故障排除

**另见：**
- [Q17：可以在 Docker 容器中运行 ADP 吗？](#q17-可以在-docker-容器中运行-adp-吗)
- [真实智能体兼容性](real-agent-compatibility.zh-CN.md)

---

### Q17: 可以在 Docker 容器中运行 ADP 吗？

**简短回答：**

可以。将项目作为卷挂载，在容器内设置 `$ADP_HOME` 和 `$ADP_RUNTIME_DIR`，并确保 Codex/Claude CLI 在容器镜像中可用。

**示例 Dockerfile：**

```dockerfile
FROM ubuntu:22.04

# 安装依赖
RUN apt-get update && apt-get install -y \
    curl \
    git \
    && rm -rf /var/lib/apt/lists/*

# 安装 ADP
RUN curl -L https://github.com/example/adp/releases/latest/adp -o /usr/local/bin/adp \
    && chmod +x /usr/local/bin/adp

# 安装 Codex CLI（示例 - 根据你的提供商调整）
RUN curl -L https://codex-cli.example.com/install.sh | sh

# 设置 ADP 路径
ENV ADP_HOME=/workspace/.adp
ENV ADP_RUNTIME_DIR=/tmp/adp-runtime

# 工作目录
WORKDIR /workspace

CMD ["/bin/bash"]
```

**使用项目挂载运行容器：**

```bash
# 构建镜像
docker build -t adp-agent .

# 使用挂载的项目运行
docker run --rm -it \
  -v /srv/my-project:/workspace/project:ro \
  -v ~/.codex-credentials:/root/.codex-credentials:ro \
  -e ADP_HOME=/workspace/.adp \
  -e ADP_RUNTIME_DIR=/tmp/adp-runtime \
  adp-agent bash

# 容器内
adp init
adp workspace add my-project /workspace/project
adp run codex --workspace my-project
```

**Docker Compose 示例：**

```yaml
version: '3.8'

services:
  adp-agent:
    build: .
    volumes:
      - ./project:/workspace/project:ro
      - ~/.codex-credentials:/root/.codex-credentials:ro
      - adp-home:/workspace/.adp
      - adp-runtime:/tmp/adp-runtime
    environment:
      - ADP_HOME=/workspace/.adp
      - ADP_RUNTIME_DIR=/tmp/adp-runtime
    command: |
      bash -c "
        adp init
        adp workspace add my-project /workspace/project
        adp run codex --workspace my-project --take --owner docker-bot --lease 1h
      "

volumes:
  adp-home:
  adp-runtime:
```

**重要考虑因素：**

**身份验证：**
- 将提供商凭证挂载为只读卷
- 或使用环境变量（不太安全）
- 确保凭证在容器环境中有效

**文件权限：**
- 项目挂载为 `:ro`（只读）防止意外修改
- ADP 状态卷需要写访问权限
- 运行时目录需要写访问权限

**网络：**
- 确保容器可以访问提供商 API（Codex/Claude 端点）
- 如需要，配置代理

**清理：**
- 清理运行时：`docker exec <container> adp runtime prune --older-than 1h`
- 删除卷：`docker volume rm adp-home adp-runtime`

**另见：**
- [Q16：如何在 CI/CD 中使用 ADP？](#q16-如何在-cicd-管道中使用-adp)
- [真实智能体兼容性](real-agent-compatibility.zh-CN.md)

---

### Q18: 如何将 ADP 与现有工具（IDE、任务跟踪器）集成？

**简短回答：**

ADP 提供 JSON 输出模式（`--format json`）用于机器解析、shell 集成（`adp shell-hook`）、补全（`adp completion`）以及 `$ADP_HOME` 下基于文件的状态供外部工具集成。

**集成点：**

**1. JSON 输出用于解析**

所有检查命令都支持 `--format json`：

```bash
# 任务列表
adp tasks list --workspace game-a --format json | jq '.tasks[] | select(.status == "pending")'

# 进度报告
adp progress report --workspace game-a --format json

# 会话历史
adp sessions list --workspace game-a --format json

# 诊断
adp doctor game-a --format json
```

**2. Shell 集成**

```bash
# 为工作空间切换生成 shell 函数
adp shell-hook --shell bash >> ~/.bashrc

# 使用：进入工作空间环境
adp_env game-a

# 生成：export ADP_WORKSPACE=game-a; cd $ADP_RUNTIME_ROOT
```

**3. Shell 补全**

```bash
# Bash 补全
adp completion --shell bash > /etc/bash_completion.d/adp

# Zsh 补全
adp completion --shell zsh > /usr/share/zsh/site-functions/_adp

# 补全：工作空间、任务、会话、智能体、配置文件
```

**4. 基于文件的状态访问**

外部工具可以直接读取 ADP 状态：

```bash
# 工作空间配置
cat ~/.adp/workspaces/game-a/workspace.yaml

# 任务状态
cat ~/.adp/workspaces/game-a/planning/tasks.yaml

# 事件日志
tail -f ~/.adp/logs/events.jsonl | jq .
```

**另见：**
- [README：补全](../README.zh-CN.md#运行时模型)
- [Q19：ADP 如何与 Git 工作流配合？](#q19-adp-如何与-git-工作流配合)

---

### Q19: ADP 如何与 Git 工作流配合？

**简短回答：**

ADP 不包装 Git 命令或自动提交/推送。智能体通过符号链接看到项目文件，可以运行 `git -C $ADP_PROJECT_ROOT` 命令。阶段门可以记录提交/推送证据，但不执行 Git 操作。

**Git 交互模型：**

**ADP 做什么：**
- 从运行时覆盖层排除 `.git` 元数据
- 中和 Git 环境变量（`GIT_DIR`、`GIT_WORK_TREE` 等）
- 为智能体提供 `$ADP_PROJECT_ROOT` 以运行 Git 命令
- 在阶段门中记录提交/推送证据（只读跟踪）
- 在 `adp workspace doctor` 中运行只读 Git 诊断

**ADP 不做什么：**
- 包装 `git commit`、`git push`、`git checkout`
- 在任务完成时自动提交或推送
- 拦截或修改 Git 命令
- 跨 Git 分支恢复提供商原生对话

**从智能体运行 Git：**

在运行时覆盖层中工作的智能体可以对真实项目运行 Git：

```bash
# 智能体运行时内（例如 AGENTS.md 指令）
# 对真实项目根运行 Git 命令
git -C $ADP_PROJECT_ROOT status
git -C $ADP_PROJECT_ROOT diff
git -C $ADP_PROJECT_ROOT add .
git -C $ADP_PROJECT_ROOT commit -m "Fix auth bug"
git -C $ADP_PROJECT_ROOT push
```

**阶段门 Git 证据：**

阶段命令记录 Git 操作而不执行它们：

```bash
# 阶段工作流
adp phase accept --workspace game-a  # 标记阶段为已审查
adp phase commit --workspace game-a  # 记录提交证据（你单独运行 git commit）
adp phase push --workspace game-a    # 记录推送证据（你单独运行 git push）

# ADP 跟踪这些步骤已发生，但不运行 git 本身
```

**推荐的 ADP Git 工作流：**

**选项 1：手动 Git（完全控制）**

```bash
# 处理任务
adp run codex --workspace game-a --task task-001

# 智能体修改运行时覆盖层中的文件（符号链接指向真实项目）
# 更改出现在真实项目根

# 手动提交
cd /srv/my-project  # 真实项目根
git status
git add .
git commit -m "Fix: Auth bug in login flow"
git push

# 记录证据
adp phase commit --workspace game-a
adp phase push --workspace game-a
adp tasks done task-001
```

**另见：**
- [README：运行时模型 - Git 元数据](../README.zh-CN.md#运行时模型)
- [任务管理：阶段门](task-management.zh-CN.md)

---

## 高级主题

### Q20: 会话恢复和继续如何工作？

**简短回答：**

`adp sessions restore-plan` 建议用相同智能体重新运行相同会话。`adp sessions resume-plan` 建议继续 ADP 工作上下文，可能使用不同智能体。两者都不恢复提供商原生对话。

**两个恢复命令：**

**1. 恢复计划（同工具重运行）**

建议用相同智能体重新运行相同会话：

```bash
# 查看会话
adp sessions show session-20260614T120000-abc123

# 获取重运行建议
adp sessions restore-plan session-20260614T120000-abc123

# 输出：建议的命令
adp run codex --workspace game-a --task task-001 \
  --profile architect --keep-runtime
```

**2. 继续计划（跨工具交接）**

建议用不同智能体继续工作：

```bash
# 用 Codex 的原始会话
adp run codex --workspace game-a --task task-001 --owner alice

# 获取跨工具继续计划
adp sessions resume-plan session-20260614T120000-abc123 \
  --agent claude --owner bob --lease 2h

# 输出：建议的命令
adp run claude --workspace game-a --task task-001 \
  --owner bob --lease 2h
# 注意：配置文件和智能体特定参数被省略（不同工具）
```

**传输的内容：**

✅ **ADP 工作上下文（传输）**
- 工作空间和项目根标识
- 任务 ID 和任务快照
- 阶段状态和门
- 会话事件历史
- 所有者和租约指导

❌ **提供商状态（不传输）**
- Codex/Claude 对话历史
- 提供商原生任务面板
- 交互式会话句柄
- 提供商特定的内部状态

**恢复是 ADP 交接，不是提供商恢复：**

```
会话 A（Codex）  →  会话 B（Claude）
     ↓                      ↓
相同 ADP 任务上下文    新的提供商对话
相同工作空间配置      不同的智能体指令
```

**示例跨工具交接：**

```bash
# 步骤 1：Alice 用 Codex 工作
adp run codex --workspace game-a --task task-001 --owner alice --lease 2h
# 会话：session-20260614T120000-abc123

# 步骤 2：Alice 完成实现，释放任务
adp tasks release --workspace game-a task-001 --owner alice

# 步骤 3：Bob 审查上下文
adp sessions show session-20260614T120000-abc123
adp tasks show task-001
adp progress report --workspace game-a

# 步骤 4：Bob 获取 Claude 的继续计划
adp sessions resume-plan session-20260614T120000-abc123 \
  --agent claude --owner bob --lease 1h

# 步骤 5：Bob 运行建议的命令
adp run claude --workspace game-a --task task-001 \
  --owner bob --lease 1h --profile reviewer

# Claude 启动具有 ADP 任务上下文的新对话
```

**常见陷阱：**
- ⚠️ 恢复不会继续 Codex/Claude 对话
- ⚠️ 跨工具恢复启动新的提供商会话
- ⚠️ 恢复/继续命令是只读的；它们不执行建议的命令

**另见：**
- [会话恢复文档](session-restore.zh-CN.md)
- [Q15：如何在智能体之间交接工作？](#q15-如何在智能体或操作员之间交接工作)

---

### Q21: 运行时覆盖层的性能影响是什么？

**简短回答：**

运行时覆盖层使用符号链接，因此开销可以忽略不计——对于具有 <10k 文件的项目，通常构建时间 <100ms。符号链接不复制数据，因此磁盘空间和构建时间不受影响。

**性能细分：**

**运行时覆盖层创建：**
- **时间**：对于具有 <10k 文件的项目通常 <100ms
- **机制**：创建符号链接，不复制
- **磁盘 I/O**：最小（仅写入生成的配置文件）

**构建时间影响：**
- **零影响**：符号链接指向真实文件
- 工具（go build、npm install、cargo build）看到原始文件
- 无数据复制

**磁盘空间：**
- **生成的文件**：~10-50KB（AGENTS.md、CLAUDE.md、适配器配置）
- **符号链接**：每个符号链接 ~1KB
- **总计**：每个运行时覆盖层通常 <1MB
- **保留的运行时**：如果不清理会累积

**运行时性能：**
- **文件访问**：无开销（符号链接对程序透明）
- **智能体执行**：与在真实项目中运行相同
- **清理**：~10-50ms 删除覆盖层目录

**基准测试（示例项目）：**

```
项目：5,000 个文件，总大小 500MB
运行时覆盖层创建：80ms
生成的文件：45KB
符号链接计数：5,003
总覆盖层大小：~50KB（符号链接不复制数据）
清理时间：30ms
```

**大型项目考虑：**

对于具有 >50k 文件的项目：
- 覆盖层创建：~500ms-1s
- 考虑用 `--keep-runtime` 进行交互式调试
- 符号链接数量不影响运行时性能

**清理成本：**

```bash
# 检查保留的运行时磁盘使用情况
du -sh /tmp/adp-runtime/*

# 清理旧运行时
adp runtime prune --older-than 24h --dry-run
# 显示：5 个运行时，总计 250MB（主要是符号链接）

# 实际数据：~200KB 生成的文件
```

**优化提示：**
- 默认清理（无 `--keep-runtime`）防止累积
- 定期清理保留的运行时：`adp runtime prune --older-than 24h`
- 大多数项目不需要性能调优

**另见：**
- [Q3：什么是运行时覆盖层？](#q3-什么是运行时覆盖层)
- [Q10：应该使用 --keep-runtime 吗？](#q10-应该使用---keep-runtime-还是让-adp-清理)

---

### Q22: 可以用钩子或插件自定义 ADP 的行为吗？

**简短回答：**

目前没有插件系统或钩子。可扩展性通过工作空间配置、自定义配置文件、MCP 服务器集成以及围绕 `adp run` 的 shell 包装器来实现前/后逻辑。

**当前可扩展性选项：**

**1. 工作空间配置**

通过 workspace.yaml 自定义每个项目的行为：

```yaml
# workspace.yaml
project:
  root: /srv/my-project

codex:
  command: /custom/path/to/codex-wrapper.sh
  base_prompt: prompts/custom-prompt.md
  memory:
    - memory/custom-context.md
  mcp:
    servers: mcp/custom-servers.json
```

**2. 自定义配置文件**

创建特定角色的配置：

```bash
# profiles/custom-role.yaml
codex:
  base_prompt: prompts/custom-role-prompt.md
  memory:
    - memory/role-specific-context.md
```

**3. MCP 服务器集成**

通过模型上下文协议扩展智能体能力：

```json
// mcp/servers.json
{
  "mcpServers": {
    "custom-tool": {
      "command": "node",
      "args": ["/path/to/custom-mcp-server.js"]
    }
  }
}
```

**4. Shell 包装器模式**

在 `adp run` 周围添加前/后逻辑：

```bash
#!/bin/bash
# custom-adp-run.sh

# 运行前钩子
echo "Starting agent run at $(date)"
adp tasks show "${TASK_ID}"

# 运行 ADP
adp run codex --workspace game-a --task "${TASK_ID}" "$@"
EXIT_CODE=$?

# 运行后钩子
if [ $EXIT_CODE -eq 0 ]; then
  echo "Success! Sending notification..."
  curl -X POST https://slack.example.com/webhook \
    -d "{\"text\": \"Task ${TASK_ID} completed\"}"
fi

exit $EXIT_CODE
```

**不可用的内容：**

❌ 用于扩展 ADP 核心的插件系统
❌ Git 钩子集成（pre-commit、post-push）
❌ 事件驱动触发器（任务完成时、阶段接受时）
❌ 运行时覆盖层自定义钩子
❌ 自定义任务状态机

**变通方法：**

对于自动化需求：
1. **包装 `adp` 命令**在脚本中用于前/后逻辑
2. **轮询状态**通过 `adp tasks list --format json` 并对更改做出反应
3. **解析事件日志**（`~/.adp/logs/events.jsonl`）用于审计追踪
4. **通过 MCP 服务器和自定义提示词扩展智能体**

**未来考虑：**

ADP 保持终端优先的简洁性。插件系统增加复杂性。如果你有可扩展性需求，考虑：
- 打开 GitHub issue 描述用例
- 向社区贡献 shell 脚本模式
- 构建读取 ADP 状态文件的外部工具

**另见：**
- [Q18：如何与现有工具集成？](#q18-如何将-adp-与现有工具ide任务跟踪器集成)
- [examples/basic-workspace](../examples/basic-workspace/) - 配置模式

---

## 返回顶部

- [核心概念](#核心概念)
- [使用决策](#使用决策)
- [团队协作](#团队协作)
- [集成场景](#集成场景)
- [高级主题](#高级主题)

---

**有问题或反馈？** 参见[故障排除指南](troubleshooting.zh-CN.md)了解错误解决方法，或在项目仓库上打开 issue。












