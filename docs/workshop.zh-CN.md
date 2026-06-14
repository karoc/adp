# ADP Workshop：实战学习指南

English: [workshop.md](workshop.md)

**⏱️ 总时长：30 分钟**

本实战 workshop 通过渐进式、任务驱动的练习教授 ADP 核心工作流程。完成后，你将理解工作区管理、任务协调、agent 执行和调试技术。

**你将学到：**
- ✅ 设置和验证 ADP 工作区
- ✅ 创建和管理任务的整个生命周期
- ✅ 使用原子化任务领取运行 agent
- ✅ 通过 session 和 event 检查执行历史
- ✅ 使用诊断命令调试问题
- ✅ 协调多 agent 工作流

**前置条件：**
- 已安装 ADP（参见 [install.md](install.md)）
- 已安装 Go（用于示例项目）
- 基本命令行操作能力
- 30 分钟专注时间

**Workshop 与 Onboarding 的区别：**
- **Operator Onboarding**（[operator-onboarding.zh-CN.md](operator-onboarding.zh-CN.md)）：验证"ADP 在你机器上能工作"（15-20 分钟）
- **本 Workshop**：通过实践教授"如何有效使用 ADP"（30 分钟）

## 快速开始

运行自动设置脚本：

```bash
cd examples/workshop
./setup.sh
```

这将创建：
- 示例 Go 项目在 `~/adp-workshop-project`
- 名为 `workshop` 的 ADP 工作区
- 用于学习的假 `workshop-agent` 命令

然后按照下面的模块学习。

---

## 模块 1：工作区设置与验证

**⏱️ 时间：5 分钟**

### 学习目标

- 创建和检查工作区
- 在运行 agent 前验证配置
- 理解工作区诊断

### 场景

你正在加入一个使用 ADP 的项目。设置你的本地工作区并验证它已准备好运行 agent。

### 实战步骤

**步骤 1.1：验证你的 workshop 环境**

```bash
# 检查工作区是否存在（setup.sh 已创建）
adp workspace list

# 查看工作区详情
adp workspace show workshop
```

**✓ 你应该看到**：工作区 `workshop` 指向 `~/adp-workshop-project`。

**步骤 1.2：运行诊断**

```bash
# 检查工作区健康状态
adp workspace doctor workshop

# 运行综合诊断
adp doctor workshop --verbose
```

**✓ 你应该看到**：显示工作区验证的状态报告。如果未安装 codex/claude CLI，一些关于 agent 命令的警告是正常的（没关系——我们使用 `workshop-agent`）。

**步骤 1.3：探索示例项目**

```bash
# 进入项目目录
cd ~/adp-workshop-project

# 构建示例 CLI
go build -o task-cli main.go

# 试用一下
./task-cli add "测试任务"
./task-cli list
```

**✓ 你应该看到**：简单的任务管理器 CLI 正常工作。注意：这个项目有一个你稍后会发现的故意 bug！

### 你学到了什么

- **工作区**：将名称（`workshop`）映射到项目根目录 + 配置
- **Doctor 命令**：在运行 agent 前验证设置
- **配置可见性**：`show` 和 `doctor` 展示 ADP 对你环境的视图

### 故障排查

**如果工作区不存在：**
- 运行设置脚本：`./examples/workshop/setup.sh`
- 或手动创建：`adp workspace add workshop ~/adp-workshop-project`

**如果 doctor 显示错误：**
- 检查项目目录是否存在：`ls -ld ~/adp-workshop-project`
- 关于缺少 codex/claude 的警告在本 workshop 中是正常的
- 使用 `--verbose` 查看详细诊断输出

---

## 模块 2：任务生命周期管理

**⏱️ 时间：10 分钟**

### 学习目标

- 创建带描述和优先级的任务
- 理解任务状态转换
- 使用原子化 `--take` 领取运行 agent
- 管理任务所有权和租约
- 完成和跟踪进度

### 场景

创建任务，将其分配给 agent，监控执行，并标记为完成。

### 实战步骤

**步骤 2.1：创建你的第一个任务**

```bash
# 添加任务
TASK_ID=$(adp tasks add \
  --workspace workshop \
  --priority high \
  "修复 CompleteTask 函数的边界检查" | \
  sed -n 's/^task \(task-[^ ]*\) added$/\1/p')

echo "创建的任务：$TASK_ID"

# 查看任务看板
adp tasks list --workspace workshop
```

**✓ 你应该看到**：新任务状态为 `pending`，优先级为 `high`。

**步骤 2.2：预览可用工作**

```bash
# 查看可领取的任务
adp tasks next --workspace workshop

# 机器可读格式
adp tasks next --workspace workshop --format json | jq '.'
```

**✓ 你应该看到**：你的任务列为可用工作。

**步骤 2.3：使用原子化任务领取运行 agent**

```bash
# 原子化地领取任务并启动 agent
adp run workshop-agent \
  --workspace workshop \
  --take \
  --owner alice \
  --lease 30m
```

**✓ 你应该看到**：
- Workshop agent 横幅和环境显示
- 运行时覆盖层验证
- 模拟工作步骤
- 任务被 `alice` 领取

**步骤 2.4：检查任务状态**

```bash
# 显示任务详情
adp tasks show --workspace workshop "$TASK_ID"

# 检查进度
adp progress --workspace workshop
```

**✓ 你应该看到**：任务现在状态为 `in_progress`，所有者为 `alice`，有租约到期时间。

**步骤 2.5：管理租约**

```bash
# 续约租约（模拟持续工作）
adp tasks renew \
  --workspace workshop \
  "$TASK_ID" \
  --owner alice \
  --lease 1h

# 检查过期任务（目前没有）
adp tasks stale --workspace workshop
```

**✓ 你应该看到**：更新后的租约到期时间。

**步骤 2.6：完成任务**

```bash
# 标记任务完成
adp tasks done --workspace workshop "$TASK_ID"

# 查看最终进度
adp progress report --workspace workshop
```

**✓ 你应该看到**：任务状态为 `completed`，进度显示 1 个已完成任务。

### 你学到了什么

- **任务状态**：`pending` → `in_progress` → `completed`
- **原子化领取**：`run --take` 在一个操作中领取 + 启动
- **所有权模型**：任务有所有者和时间限制的租约
- **租约管理**：为长时间运行的工作续约
- **进度跟踪**：监控工作区范围的完成情况

### 故障排查

**如果 `run --take` 失败提示"没有可用任务"：**
- 检查任务是否存在：`adp tasks list --workspace workshop`
- 确保任务是 `pending`（未被领取）
- 尝试显式任务：`adp run workshop-agent --workspace workshop --task $TASK_ID`

**如果找不到 workshop-agent：**
- 检查 PATH：`which workshop-agent`
- 重新运行设置：`./examples/workshop/setup.sh`
- 手动添加：`export PATH="$HOME/.local/bin:$PATH"`

**如果任务一直是 `in_progress`：**
- Agent 退出不会自动完成任务（这是设计）
- 手动完成允许在标记完成前检查
- 使用完成命令：`adp tasks done --workspace workshop $TASK_ID`

---

## 模块 3：运行时检查与调试

**⏱️ 时间：10 分钟**

### 学习目标

- 检查 agent 执行历史
- 理解运行时覆盖层结构
- 使用诊断命令调试
- 检查任务到 session 的关系
- 生成执行报告

### 场景

调查一个已完成的 agent 运行，了解发生了什么并调试潜在问题。

### 实战步骤

**步骤 3.1：创建任务并保留运行时**

```bash
# 添加新任务
DEBUG_TASK=$(adp tasks add \
  --workspace workshop \
  --priority normal \
  "调查 task-cli 的内存使用" | \
  sed -n 's/^task \(task-[^ ]*\) added$/\1/p')

# 运行 agent 并保留运行时
adp run workshop-agent \
  --workspace workshop \
  --task "$DEBUG_TASK" \
  --keep-runtime
```

**✓ 你应该看到**：Agent 运行，退出后运行时目录被保留。

**步骤 3.2：检查 session**

```bash
# 列出最近的 session
adp sessions list --workspace workshop --limit 5

# 获取最新 session ID
LATEST_SESSION=$(adp sessions list --workspace workshop | sed -n '2s/ .*//p')
echo "最新 session：$LATEST_SESSION"

# 显示 session 详情
adp sessions show "$LATEST_SESSION"
```

**✓ 你应该看到**：Session 详情包括任务 ID、运行时路径、持续时间、agent 命令和退出码。

**步骤 3.3：检查事件**

```bash
# 列出此任务的所有事件
adp events list \
  --workspace workshop \
  --task "$DEBUG_TASK" \
  --limit 10

# 按事件类型过滤
adp events list \
  --workspace workshop \
  --type task_claimed

# Session 特定事件
adp events list \
  --workspace workshop \
  --session "$LATEST_SESSION"
```

**✓ 你应该看到**：事件时间线显示 `task_claimed`、`runtime_created`、`agent_started`、`agent_exited`、`runtime_kept`。

**步骤 3.4：探索运行时覆盖层**

```bash
# 查找运行时目录（从 session show 输出）
RUNTIME_DIR=$(adp sessions show "$LATEST_SESSION" | grep -oP 'runtime_path: \K.*')

if [ -d "$RUNTIME_DIR" ]; then
  echo "运行时目录存在：$RUNTIME_DIR"
  
  # 检查结构
  ls -la "$RUNTIME_DIR"
  
  # 检查 AGENTS.md
  cat "$RUNTIME_DIR/AGENTS.md"
  
  # 检查运行时元数据
  cat "$RUNTIME_DIR/.adp-runtime.yaml"
else
  echo "运行时已清理。下次使用 --keep-runtime 运行。"
fi
```

**✓ 你应该看到**：运行时目录包含 AGENTS.md、.adp-runtime.yaml 和到项目文件的符号链接。

**步骤 3.5：生成恢复计划**

```bash
# 获取恢复指导
adp sessions restore-plan "$LATEST_SESSION"

# 机器可读格式
adp sessions restore-plan "$LATEST_SESSION" --format json | jq '.'

# 跨 agent 交接计划
adp sessions resume-plan "$LATEST_SESSION" \
  --agent claude \
  --owner bob \
  --lease 2h
```

**✓ 你应该看到**：建议的 `adp run` 命令来重现或恢复 session，带有交接的上下文。

**步骤 3.6：生成进度报告**

```bash
# 人类可读报告
adp progress report --workspace workshop

# 导出为 JSON
adp progress report --workspace workshop --format json > /tmp/workshop-progress.json
cat /tmp/workshop-progress.json | jq '.summary'
```

**✓ 你应该看到**：包含任务计数、最近活动和 session 证据的综合报告。

**步骤 3.7：清理运行时**

```bash
# 预览清理（dry-run）
adp runtime prune --older-than 24h --dry-run

# 实际清理 workshop 运行时
adp runtime prune --older-than 5m
```

**✓ 你应该看到**：被清理的运行时目录列表。

### 你学到了什么

- **Session**：记录每次 agent 运行（任务 + 运行时 + 时间）
- **Event**：所有 ADP 操作的详细日志
- **运行时覆盖层**：包含生成文件的临时目录
- **恢复计划**：重现或恢复 session
- **进度报告**：生成状态文档
- **运行时清理**：管理磁盘空间

### 故障排查

**如果 session 列表为空：**
- 验证 agent 已运行：`adp events list --workspace workshop`
- 检查工作区名称是否正确

**如果运行时目录缺失：**
- 运行时已被清理
- 下次运行使用 `--keep-runtime` 标志
- 检查：`ls /tmp/adp-runtime/workshop-*`

**如果 restore-plan 显示"数据不足"：**
- 用新 session 尝试
- 使用 `sessions show` 查看可用数据

---

## 模块 4：跨 Session 工作流

**⏱️ 时间：5 分钟**

### 学习目标

- 协调多 agent 工作流
- 在 agent 之间交接工作
- 使用任务依赖
- 跟踪跨 agent 进度

### 场景

协调多个 agent 处理相互依赖的任务。

### 实战步骤

**步骤 4.1：创建依赖任务**

```bash
# 创建基础任务
FOUNDATION=$(adp tasks add \
  --workspace workshop \
  --priority high \
  "为 TaskManager 添加单元测试" | \
  sed -n 's/^task \(task-[^ ]*\) added$/\1/p')

# 创建依赖任务
DEPENDENT=$(adp tasks add \
  --workspace workshop \
  --priority normal \
  "使用 TaskManager 添加集成测试" | \
  sed -n 's/^task \(task-[^ ]*\) added$/\1/p')

# 设置依赖关系
adp tasks block \
  --workspace workshop \
  --task "$DEPENDENT" \
  --blocked-by "$FOUNDATION"

echo "基础任务：$FOUNDATION"
echo "依赖任务：$DEPENDENT（被阻塞）"
```

**✓ 你应该看到**：创建了两个任务并建立了阻塞关系。

**步骤 4.2：观察任务可见性**

```bash
# 检查可用工作
adp tasks next --workspace workshop

# 显示依赖任务详情
adp tasks show --workspace workshop "$DEPENDENT"
```

**✓ 你应该看到**：基础任务出现在 `next` 中，依赖任务显示 `blocked_by` 且不出现在 `next` 中。

**步骤 4.3：完成基础任务并解除阻塞**

```bash
# 处理基础任务
adp run workshop-agent \
  --workspace workshop \
  --task "$FOUNDATION" \
  --owner alice \
  --lease 30m

# 完成它
adp tasks done --workspace workshop "$FOUNDATION"

# 再次检查看板
adp tasks next --workspace workshop
```

**✓ 你应该看到**：依赖任务现在出现在 `next` 输出中（已解除阻塞）。

**步骤 4.4：交接给不同的 agent**

```bash
# 获取基础任务的 session
FOUNDATION_SESSION=$(adp sessions list \
  --workspace workshop \
  --task "$FOUNDATION" | \
  sed -n '2s/ .*//p')

# 生成跨 agent 交接
adp sessions resume-plan "$FOUNDATION_SESSION" \
  --agent claude \
  --owner bob \
  --lease 1h

# Bob 领取依赖任务
adp tasks claim "$DEPENDENT" \
  --workspace workshop \
  --owner bob \
  --lease 1h

# 检查所有权
adp tasks list --workspace workshop
```

**✓ 你应该看到**：带上下文的恢复计划，任务所有权显示 alice（已完成）和 bob（进行中）。

**步骤 4.5：生成最终报告**

```bash
# 完成依赖任务
adp tasks done --workspace workshop "$DEPENDENT"

# 最终进度报告
adp progress report --workspace workshop

# 为工具导出
adp progress report --workspace workshop --format json > /tmp/workshop-final.json
cat /tmp/workshop-final.json | jq '.summary'
```

**✓ 你应该看到**：报告显示多个已完成任务及 session 证据。

### 你学到了什么

- **任务依赖**：使用 `block` 强制完成顺序
- **任务可见性**：被阻塞任务在解除前隐藏于 `next`
- **跨 agent 交接**：使用 `resume-plan` 提供上下文
- **所有权跟踪**：查看谁做了什么
- **进度报告**：随时生成报告

### 故障排查

**如果被阻塞任务出现在 `next` 中：**
- 验证阻塞：`adp tasks show $TASK_ID | grep blocked_by`
- 检查阻塞者是否已完成
- 阻塞影响可见性，不影响领取

**如果 `tasks claim` 失败：**
- 任务可能已被领取
- 检查：`adp tasks list --workspace workshop`
- 对过期租约使用 `tasks take`

---

## Workshop 完成

### 总结

✅ **恭喜！** 你已完成 ADP workshop。

你现在知道如何：
- 设置和验证工作区
- 管理任务的整个生命周期
- 使用原子化领取运行 agent
- 检查执行历史
- 使用诊断命令调试
- 协调多 agent 工作流
- 生成进度报告

### 后续步骤

**日常使用：**
1. 安装真实 agent CLI（codex/claude）
2. 设置你的实际项目工作区
3. 配置 profile（参见 `examples/basic-workspace`）
4. 从简单任务开始

**高级工作流：**
1. 探索阶段管理：`adp phase --help`
2. 设置 MCP 集成
3. 使用工作区 profile
4. 使用计划导入自动化

**文档：**
- [安装指南](install.md)
- [Operator Onboarding](operator-onboarding.zh-CN.md)
- [任务管理](task-management.md)
- [Session 恢复](session-restore.md)
- [真实 Agent 设置](real-agent-compatibility.md)

### 清理

删除 workshop 工件：

```bash
# 删除假 agent
rm ~/.local/bin/workshop-agent

# 删除工作区（可选）
adp workspace remove workshop

# 删除项目（可选）
rm -rf ~/adp-workshop-project

# 清理运行时
adp runtime prune --older-than 1h
```

---

## 常见问题

**问：为什么使用假 agent 而不是真实的 codex/claude？**  
答：假 agent 消除了设置摩擦（安装、认证、网络、配额），让你可以专注于学习 ADP 概念。对于实际工作，安装真实的 agent CLI。

**问：task-cli 中的 bug 是故意的吗？**  
答：是的！`CompleteTask` 中的 off-by-one 错误是故意的。它演示了在开发过程中发现问题，你会用 ADP 任务来协调。

**问：我可以用这个 workshop 来培训团队吗？**  
答：当然可以！设置脚本使其可重现。每个人运行 `./setup.sh` 并跟随模块学习。

**问：这与 operator-onboarding.md 有什么区别？**  
答：Onboarding 验证安装（15-20 分钟，验证导向）。Workshop 教授使用（30 分钟，实战技能培养）。

**问：如果我想用真实 agent 尝试怎么办？**  
答：完成 workshop 后，参见 [real-agent-compatibility.md](real-agent-compatibility.md) 进行 codex/claude 设置，然后用真实 agent 重复模块。

---

**版本**：1.0  
**最后更新**：2026-06-14  
**反馈**：欢迎提交 issue 或 PR 提出建议
