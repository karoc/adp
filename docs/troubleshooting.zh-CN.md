# 故障排查指南

English: [troubleshooting.md](troubleshooting.md)

本指南帮助你诊断和解决常见的 ADP 问题。问题按错误消息或症状组织，方便搜索。

---

## 目录

- [安装和设置](#安装和设置)
- [工作区问题](#工作区问题)
- [Agent 执行问题](#agent-执行问题)
- [运行时问题](#运行时问题)
- [任务管理问题](#任务管理问题)
- [阶段管理问题](#阶段管理问题)
- [会话问题](#会话问题)
- [交互和确认问题](#交互和确认问题)
- [参数验证问题](#参数验证问题)
- [环境变量](#环境变量)
- [权限问题](#权限问题)
- [诊断命令](#诊断命令)

---

## 安装和设置

### "command not found: adp"

**原因：**
- ADP 二进制文件不在 `$PATH` 中
- 二进制文件安装不正确
- Shell 未重新加载 `$PATH`

**诊断：**
```bash
# 检查二进制文件是否存在
ls -la ./bin/adp
which adp

# 检查 PATH
echo $PATH
```

**解决方法：**
1. 将 ADP 二进制目录添加到 `$PATH`：
   ```bash
   export PATH="$HOME/.local/bin:$PATH"
   ```
2. 或使用绝对路径：
   ```bash
   /path/to/adp --help
   ```
3. 重新加载 shell 配置：
   ```bash
   source ~/.bashrc  # 或 ~/.zshrc
   ```

---

### "ADP_HOME not set or invalid"

**原因：**
- 未设置 `$ADP_HOME` 环境变量
- 目录不存在或不可写

**诊断：**
```bash
# 检查环境变量
echo $ADP_HOME

# 检查目录是否存在
ls -ld $ADP_HOME
```

**解决方法：**
1. 设置 `ADP_HOME`（默认为 `~/.adp`）：
   ```bash
   export ADP_HOME="$HOME/.adp"
   ```
2. 初始化 ADP：
   ```bash
   adp init
   ```

---

## 工作区问题

### "workspace not found"

**原因：**
- 工作区名称拼写错误
- 工作区尚未创建
- `$ADP_HOME` 指向错误的目录

**诊断：**
```bash
# 列出所有工作区
adp workspace list

# 检查 ADP_HOME
echo $ADP_HOME
ls -la $ADP_HOME/workspaces/
```

**解决方法：**
1. 验证工作区名称拼写
2. 如需创建工作区：
   ```bash
   adp workspace add my-project /path/to/project
   ```
3. 检查 `$ADP_HOME` 是否正确

---

### "project root does not exist"

**原因：**
- 项目路径不正确
- 项目目录已移动或删除
- 符号链接损坏

**诊断：**
```bash
# 从工作区配置检查项目路径
adp workspace show my-workspace

# 验证目录存在
ls -ld /path/to/project
```

**解决方法：**
1. 如果项目已移动，更新工作区：
   ```bash
   adp workspace remove old-name
   adp workspace add new-name /new/path/to/project
   ```
2. 或使用正确路径重新创建工作区

---

### "workspace doctor reports errors"

**原因：**
- 配置文件缺失或无效
- 引用的文件（prompts、memory、MCP）不存在
- 运行时父目录不安全

**诊断：**
```bash
# 运行详细诊断
adp workspace doctor my-workspace --verbose

# JSON 输出用于机器解析
adp workspace doctor my-workspace --format json
```

**解决方法：**
- 按照 doctor 输出中的具体建议操作
- 检查工作区配置中引用的所有文件路径
- 验证 `$ADP_RUNTIME_DIR` 不在项目根目录内

---

## Agent 执行问题

### "agent command not found: <command>"

**原因：**
- Agent CLI（如 `codex`、`claude`）未安装
- Agent 二进制文件不在 `$PATH` 中
- 工作区配置了不存在的命令路径

**诊断：**
```bash
# 检查 agent 命令是否可解析
which codex
which claude

# 检查工作区命令配置
adp workspace show my-workspace

# 运行工作区诊断
adp workspace doctor my-workspace
```

**解决方法：**
1. 安装缺失的 agent CLI，然后验证：
   ```bash
   codex --version
   ```
2. 如果已安装但未找到，将其位置添加到 `$PATH`：
   ```bash
   export PATH="/path/to/agent/bin:$PATH"
   ```
3. 在工作区中配置显式命令路径，避免依赖 `$PATH` 查找：
   ```bash
   # 编辑工作区配置并显式设置 agent 命令路径
   adp workspace doctor my-workspace
   ```

---

### "unknown command" 附带 "Did you mean ...?"

**原因：**
- 顶层命令或子命令名称拼写错误
- ADP 无法匹配输入并建议相似命令

**诊断：**
```bash
# 示例：命令名称拼写错误
adp wrkspace list
# Error: unknown command "wrkspace"
# Did you mean one of these?
#   workspace
```

**解决方法：**
1. 使用 "Did you mean" 输出中的建议命令。
2. 列出所有顶层命令以确认正确拼写：
   ```bash
   adp --help
   ```
3. 常用快捷方式也提供了命令别名：
   - `ws` → `workspace`
   - `t` → `tasks`
   - `s` → `sessions`
   - `e` → `events`
   - `rt` → `runtime`
   - `p` → `phase`

---

## 运行时问题

### "failed to build runtime"

**原因：**
- `$ADP_RUNTIME_DIR` 不可写
- 磁盘空间耗尽
- 符号链接创建失败

**诊断：**
```bash
# 检查运行时目录
echo $ADP_RUNTIME_DIR
ls -ld $ADP_RUNTIME_DIR

# 检查磁盘空间
df -h $ADP_RUNTIME_DIR

# 检查权限
ls -ld $(dirname $ADP_RUNTIME_DIR)
```

**解决方法：**
1. 设置可写的运行时目录：
   ```bash
   export ADP_RUNTIME_DIR="/tmp/adp-runtime"
   ```
2. 清理旧运行时：
   ```bash
   adp runtime prune --older-than 24h
   ```
3. 检查文件系统权限

---

### "runtime directory not cleaned up"

**原因：**
- 运行时使用 `--keep-runtime` 创建
- Agent 崩溃前未清理
- 需要手动检查

**诊断：**
```bash
# 列出保留的运行时
adp runtime prune --dry-run --include-kept
```

**解决方法：**
1. 删除旧运行时：
   ```bash
   # 不包括保留的运行时
   adp runtime prune --older-than 1h

   # 包括保留的运行时
   adp runtime prune --older-than 1h --include-kept
   ```

---

### "symlink conflicts in runtime"

**原因：**
- 项目文件与生成的文件冲突
- 运行时未正确清理

**诊断：**
```bash
# 检查运行时结构
ls -la $ADP_RUNTIME_ROOT

# 检查工作区 doctor
adp workspace doctor --verbose
```

**解决方法：**
- 避免在项目根目录中使用 `AGENTS.md`、`CLAUDE.md` 等文件
- 清理运行时并重试
- 查看 workspace doctor 建议

---

### "unsafe runtime parent"

**原因：**
- `$ADP_RUNTIME_DIR` 解析为文件系统根目录
- 运行时父目录与项目根目录相同
- 运行时父目录在项目根目录内（或反之）

ADP 拒绝这些配置，以防止在运行时清理期间意外删除项目文件或整个文件系统。

**诊断：**
```bash
# 检查运行时父目录和项目根目录的关系
echo $ADP_RUNTIME_DIR
echo $ADP_HOME

# 检查工作区项目根目录
adp workspace show my-workspace

# 运行 doctor 进行安全检查
adp workspace doctor my-workspace --verbose
```

**解决方法：**
1. 将 `$ADP_RUNTIME_DIR` 指向项目外的专用目录：
   ```bash
   export ADP_RUNTIME_DIR="/tmp/adp-runtime"
   ```
2. 永远不要将运行时父目录设置为 `/`、项目根目录或任何包含项目的目录。
3. 修复后重新运行以确认安全检查通过：
   ```bash
   adp workspace doctor my-workspace
   ```

---

### ".adp-runtime.yaml is reserved"

**原因：**
- 项目文件或 adapter 生成的文件使用了保留名称 `.adp-runtime.yaml`
- 此文件名由 ADP 运行时清单所有，不能被覆盖

**诊断：**
```bash
# 在项目中搜索保留文件
find . -name '.adp-runtime.yaml' -not -path '*/.adp-runtime/*'

# 检查生成文件的工作区 adapter
adp workspace show my-workspace
```

**解决方法：**
1. 重命名或删除冲突的项目文件：
   ```bash
   git mv .adp-runtime.yaml runtime-config.yaml
   ```
2. 如果是 adapter 生成的，更新 adapter 配置以使用不同的输出路径。
3. 重建运行时：
   ```bash
   adp run codex --workspace my-workspace --keep-runtime
   ```

---

## 任务管理问题

### "task not found"

**原因：**
- 任务 ID 不正确或有歧义
- 任务属于不同的工作区
- 任务已被删除

**诊断：**
```bash
# 列出所有任务
adp tasks list --workspace my-workspace

# 使用前缀检查任务
adp tasks show task-2026
```

**解决方法：**
1. 使用正确的任务 ID 或唯一前缀
2. 验证工作区名称
3. 检查任务列表中是否存在任务

---

### "ambiguous task ID"

**原因：**
- 前缀匹配多个任务

**诊断：**
```bash
# 错误消息列出所有匹配项
adp tasks show task-20
# Error: ambiguous task ID "task-20", matches:
#   - task-20260611-0001
#   - task-20260612-0002
```

**解决方法：**
- 使用更长的前缀使其唯一：
  ```bash
  adp tasks show task-20260611
  ```
- 或使用完整的任务 ID

---

### "task already claimed"

**原因：**
- 任务当前被另一个 agent 拥有
- 租约尚未过期

**诊断：**
```bash
# 检查任务状态
adp tasks show task-123

# 检查过期任务
adp tasks stale --workspace my-workspace
```

**解决方法：**
- 等待租约过期
- 或释放任务（如果你拥有它）：
  ```bash
  adp tasks release task-123 --owner current-owner
  ```

---

### "task owner mismatch"

**原因：**
- 你正在尝试释放、续期或更新由不同 operator 拥有的任务
- `--owner` 值与任务的当前所有者不匹配

**诊断：**
```bash
# 检查谁当前拥有该任务
adp tasks show task-123
```

**解决方法：**
1. 使用与当前所有者匹配的正确 `--owner`。
2. 如果前一个所有者不可用，等待租约过期，以便任务再次可以被认领：
   ```bash
   adp tasks stale --workspace my-workspace
   ```
3. 显式协调所有权转移，而不是覆盖另一个 operator 的任务。

---

### "no claimable task"

**原因：**
- `adp run --take` 在工作区中找不到未认领、未阻塞的任务
- 所有任务要么被认领、被阻塞，要么已完成

**诊断：**
```bash
# 列出任务及其状态
adp tasks list --workspace my-workspace

# 显式查找下一个可认领的任务
adp tasks next --workspace my-workspace

# 单独检查被阻塞的任务
adp tasks show task-123
```

**解决方法：**
1. 添加新任务以供认领：
   ```bash
   adp tasks add --workspace my-workspace "Next piece of work"
   ```
2. 解除等待先决条件的任务的阻塞。
3. 如果任务被临时认领，等待现有租约过期。

---

## 阶段管理问题

### "phase not found"

**原因：**
- 阶段 ID 拼写错误或在工作区中不存在
- 阶段属于不同的工作区

**诊断：**
```bash
# 列出所有阶段
adp phase list --workspace my-workspace

# 显示特定阶段
adp phase show --workspace my-workspace phase1
```

**解决方法：**
1. 根据 `adp phase list` 验证阶段 ID。
2. 如果阶段不存在，创建它：
   ```bash
   adp phase add --workspace my-workspace phase1 "Phase 1 title"
   ```

---

### "invalid phase transition"

**原因：**
- 尝试启动不在允许状态的阶段（例如已经启动）
- 阻塞阶段尚未被接受/推送
- 按错误顺序记录接受、提交或推送证据

阶段门禁强制执行严格的生命周期：阶段必须在接受前启动，在提交证据前接受，在推送证据前有提交证据。

**诊断：**
```bash
# 检查当前阶段状态和门禁要求
adp phase status --workspace my-workspace phase1

adp phase show --workspace my-workspace phase1
```

**解决方法：**
1. 按顺序遵循文档化的阶段生命周期：
   ```bash
   adp phase start --workspace my-workspace phase1
   adp phase accept --workspace my-workspace phase1
   adp phase commit --workspace my-workspace phase1 --hash <commit-hash>
   adp phase push --workspace my-workspace phase1 --remote origin --branch main
   ```
2. 首先解决任何阻塞阶段——错误消息会指明阻塞阶段。

---

## 会话问题

### "session not found"

**原因：**
- 会话 ID 不正确或属于不同的工作区
- 会话前缀太短

**诊断：**
```bash
# 列出最近的会话
adp sessions list --workspace my-workspace --limit 20
```

**解决方法：**
1. 使用 `adp sessions list` 中的完整会话 ID。
2. 使用 `--workspace` 验证工作区名称。

---

### "ambiguous session ID"

**原因：**
- 会话 ID 前缀匹配多个会话

**诊断：**
```bash
# 错误列出每个匹配的会话
adp sessions show session-2026
# Error: ambiguous session ID "session-2026", matches:
#   - session-20260611T120000-aaa
#   - session-20260612T140000-bbb
```

**解决方法：**
- 提供更长的前缀或完整的会话 ID：
  ```bash
  adp sessions show session-20260611T120000
  ```

---

## 交互和确认问题

### "operation requires confirmation; use --yes to proceed in non-interactive mode"

**原因：**
- 在非交互上下文（管道、CI、脚本）中调用了破坏性操作，但没有 `--yes` 标志
- ADP 保护 `workspace remove` 和 `runtime prune --include-kept` 免遭意外删除

**诊断：**
```bash
# 在非 TTY 上下文中重现
echo n | adp workspace remove my-workspace
```

**解决方法：**
1. 在脚本中传递 `--yes`（或 `-y`）以非交互方式确认：
   ```bash
   adp workspace remove my-workspace --yes
   adp runtime prune --include-kept --older-than 1h --yes
   ```
2. `runtime prune --dry-run` 永远不需要确认——先预览：
   ```bash
   adp runtime prune --include-kept --dry-run
   ```
3. 在交互式 TTY 中，直接回答 `[y/N]` 提示。

---

## 参数验证问题

### "--take cannot be combined with --task"

**原因：**
- `adp run` 同时给出了 `--task <id>`（运行特定任务）和 `--take`（认领下一个可用任务）

**解决方法：**
- 选择一种模式：
  ```bash
  # 运行特定任务
  adp run codex --workspace my-workspace --task task-20260614-0001

  # 或认领并运行下一个可用任务
  adp run codex --workspace my-workspace --take --owner alice --lease 2h
  ```

---

### "--owner is required with --take"

**原因：**
- `--take` 认领任务，因此需要 `--owner` 来记录谁认领了它

**解决方法：**
```bash
adp run codex --workspace my-workspace --take --owner alice --lease 2h
```

---

### "lease must not be negative" / "parse lease duration"

**原因：**
- `--lease` 值为负数、零或不是有效的 Go 持续时间字符串

**诊断：**
```bash
# 有效的持续时间格式
echo "示例：30m, 2h, 1h30m, 480m"
```

**解决方法：**
- 使用正数 Go 持续时间：
  ```bash
  adp run codex --workspace my-workspace --take --owner alice --lease 2h
  adp tasks claim task-123 --owner alice --lease 90m
  ```

---

### "unknown option / unknown command"

**原因：**
- 标志或子命令名称拼写错误或该命令不支持

**诊断：**
```bash
# 显示命令接受的选项
adp run --help
adp tasks --help
```

**解决方法：**
1. 检查错误时打印的命令用法行——它列出接受的参数。
2. 在父命令上使用 `--help` 以枚举有效的子命令和标志。

---

## 环境变量

### 环境变量不工作

**原因：**
- 变量未 export
- 变量名拼写错误
- Shell 未重新加载

**诊断：**
```bash
# 检查所有 ADP 环境变量
env | grep ADP

# 检查特定变量
echo $ADP_HOME
echo $ADP_RUNTIME_DIR
echo $ADP_WORKSPACE
```

**解决方法：**
1. Export 变量：
   ```bash
   export ADP_HOME="$HOME/.adp"
   export ADP_RUNTIME_DIR="/tmp/adp-runtime"
   ```
2. 添加到 shell profile 以持久化：
   ```bash
   echo 'export ADP_HOME="$HOME/.adp"' >> ~/.bashrc
   source ~/.bashrc
   ```

---

### "dangerous Git environment variables"

**原因：**
- Git 特定变量干扰运行时

**诊断：**
```bash
# 检查 Git 环境
env | grep GIT_
```

**解决方法：**
- ADP 在运行时自动中和这些变量
- 如果问题持续存在，手动 unset：
  ```bash
  unset GIT_DIR GIT_WORK_TREE GIT_INDEX_FILE
  ```

---

## 权限问题

### "permission denied" 错误

**原因：**
- 二进制文件不可执行
- 目录不可写
- 文件所有权问题

**诊断：**
```bash
# 检查二进制权限
ls -la $(which adp)

# 检查 ADP_HOME 权限
ls -ld $ADP_HOME

# 检查运行时目录
ls -ld $ADP_RUNTIME_DIR
```

**解决方法：**
1. 使二进制文件可执行：
   ```bash
   chmod +x /path/to/adp
   ```
2. 修复目录权限：
   ```bash
   chmod 755 $ADP_HOME
   ```
3. 检查文件所有权：
   ```bash
   ls -la $ADP_HOME/workspaces/
   ```

---

## 诊断命令

### 快速健康检查

```bash
# 检查 ADP 安装
adp version

# 检查环境
echo $ADP_HOME
echo $ADP_RUNTIME_DIR

# 列出工作区
adp workspace list

# 对所有工作区运行诊断
adp doctor

# 检查特定工作区
adp workspace doctor my-workspace --verbose
```

---

### 调试任务问题

```bash
# 列出所有任务
adp tasks list --workspace my-workspace

# 显示任务详情
adp tasks show task-123 --format json

# 检查过期任务
adp tasks stale --workspace my-workspace

# 查看任务进度
adp progress --workspace my-workspace
```

---

### 调试运行时问题

```bash
# 检查运行时目录
ls -la $ADP_RUNTIME_DIR

# 列出运行时（dry-run）
adp runtime prune --dry-run

# 清理旧运行时
adp runtime prune --older-than 1h

# 检查事件
adp events list --workspace my-workspace --limit 20
```

---

### 调试会话问题

```bash
# 列出最近的会话
adp sessions list --workspace my-workspace --limit 10

# 显示会话详情
adp sessions show session-123

# 恢复会话计划
adp sessions restore-plan session-123
```

---

## 获取帮助

如果以上解决方案都不起作用：

1. **运行诊断：**
   ```bash
   adp doctor --verbose --format json > diagnostics.json
   ```

2. **检查日志：**
   ```bash
   # 事件日志
   cat $ADP_HOME/logs/events.jsonl

   # 最近事件
   adp events list --limit 50
   ```

3. **验证安装：**
   ```bash
   adp version
   go version
   ```

4. **全新测试：**
   ```bash
   # 使用临时 ADP_HOME
   ADP_HOME=$(mktemp -d) adp init
   ```

5. **报告问题：**
   - 包含 `adp version` 输出
   - 包含 `adp doctor --verbose` 输出
   - 包含相关错误消息
   - 描述重现步骤

---

## 常见模式

### 全新开始

```bash
# 如需要，备份现有 ADP_HOME
mv $ADP_HOME $ADP_HOME.backup

# 初始化全新环境
adp init

# 重新添加工作区
adp workspace add my-project /path/to/project

# 验证
adp workspace doctor my-project
```

---

### 工作区迁移

```bash
# 导出工作区配置
adp workspace show old-workspace --format json > workspace.json

# 使用更新的设置创建新工作区
adp workspace add new-workspace /new/path

# 如需要迁移任务（手动过程）
```

---

### 运行时清理

```bash
# 查看将被删除的内容
adp runtime prune --dry-run --older-than 0s

# 删除旧运行时
adp runtime prune --older-than 24h

# 包括保留的运行时
adp runtime prune --older-than 24h --include-kept
```

---

更多文档：
- [安装指南](install.zh-CN.md)
- [Operator 入门](operator-onboarding.zh-CN.md)
- [任务管理](task-management.zh-CN.md)
- [会话恢复](session-restore.zh-CN.md)
