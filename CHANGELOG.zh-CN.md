# 更新日志

English: [CHANGELOG.md](CHANGELOG.md)

ADP（Agent Development Platform）的所有重要变更都将记录在此文件中。

格式基于 [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)，
本项目遵循[语义化版本](https://semver.org/spec/v2.0.0.html)。

---

## [未发布]

### Phase 1-5 基础（预发布开发）

以下章节记录了 ADP 从初始概念到生产就绪 1.0 候选版本的演进。所有功能都经过系统化验收测试，对于本地 terminal-first AI agent 工作流被认为是稳定的。

---

## Phase 1：核心运行时与工作区基础

**状态**：✅ 已完成
**重点**：Terminal-first 运行时隔离和工作区管理

### 新增
- **核心 CLI** — `adp` 二进制文件，具有子命令架构
- **工作区管理**
  - `adp workspace add/list/show/remove/rename` — 注册和管理工作区
  - `adp workspace doctor` — 综合配置诊断
  - `adp doctor` — 所有工作区的全局健康检查
  - 工作区配置位于 `$ADP_HOME/workspaces/`
  - 项目根目录验证和安全检查
- **运行时 Overlay 系统**
  - 基于符号链接的运行时位于 `$ADP_RUNTIME_DIR`
  - 生成的文件（`AGENTS.md`、`CLAUDE.md`、`.codex/`、`.claude/`）与项目隔离
  - Agent 退出时自动清理（可通过 `--keep-runtime` 配置）
  - 运行时清单（`.adp-runtime.yaml`）用于检查
- **Agent 适配器**
  - Codex 适配器，支持 profile
  - Claude 适配器，支持 profile
  - 进程运行器，具有环境隔离
  - `adp run <agent>` — 使用运行时 overlay 启动 agent
  - `adp enter <workspace>` — 带运行时的交互式 shell
- **事件与会话跟踪**
  - JSONL 事件日志位于 `$ADP_HOME/logs/events.jsonl`
  - `adp events list` — 按 workspace/session/task/type 查询事件
  - `adp sessions list/show` — 检查过去的 agent 运行
  - 会话历史从本地事件派生（无需外部数据库）
- **任务管理**
  - `adp tasks add/list/show/update` — 创建和管理任务
  - `adp tasks claim/renew/release/done` — 所有权和租约管理
  - `adp tasks next/take` — 原子化预览和认领工作
  - `adp tasks stale` — 查找过期租约
  - `adp tasks block` — 将任务标记为阻塞
  - 任务持久化在 `$ADP_HOME/workspaces/<name>/planning/`
  - 任务 ID 前缀匹配以便使用
- **阶段门禁**
  - `adp phase add/list/show/status` — 定义阶段
  - `adp phase start/accept/commit/push` — 记录生命周期证据
  - 基于阶段的工作流强制执行，带阻塞依赖
- **计划导入**
  - `adp plan preview/apply` — 从结构化计划导入任务和阶段
  - `adp plan doctor` — 验证计划完整性
- **进度报告**
  - `adp progress` — 查看工作区进度摘要
  - `adp progress report` — 生成 markdown/JSON 报告（英文/中文）
- **会话恢复**
  - `adp sessions restore-plan` — 从过去会话提取重新运行指导
  - `adp sessions resume-plan` — 为 operator 交接生成工作上下文
- **运行时维护**
  - `adp runtime prune` — 使用基于时间的过滤清理旧运行时
  - `--dry-run` 模式用于安全预览
  - `--include-kept` 标志清理手动保留的运行时
- **Shell 集成**
  - `adp shell-hook` — 用于运行时感知导航的 shell 函数
  - `adp completion` — bash/zsh 补全脚本
  - `adp completion values` — 工作区/任务/会话的动态补全
  - `adp env --cd` — 为工作区运行时打印 cd 命令
- **工具**
  - `adp init` — 初始化 `$ADP_HOME`
  - `adp version` — 显示版本和构建信息
  - JSON 输出（`--format json`）用于所有 list/show 命令
  - `--verbose` 标志用于详细诊断

### 基础设施
- Go 1.24+ 代码库，无外部依赖（仅标准库）
- 基于文件的存储（无需数据库）
- 跨平台支持（Linux、macOS、Windows via WSL）
- 会话本地状态隔离，安全并发使用
- Git 环境中和（GIT_DIR、GIT_WORK_TREE 等）

---

## Phase 2-3：文档与双语支持

**状态**：✅ 已完成
**重点**：生产级文档和中文本地化

### 新增
- **综合文档**
  - `README.md` / `README.zh-CN.md` — 项目概览和快速入门
  - `docs/install.md` — 安装指南（源码/二进制/发布）
  - `docs/operator-onboarding.md` — 带检查点的实践教程
  - `docs/task-management.md` — 任务工作流深入探讨
  - `docs/session-restore.md` — 恢复/继续模式
  - `docs/faq.md` — 22 个问题，涵盖常见工作流
  - `docs/engineering-standards.md` — 项目约定
  - `docs/license-policy.md` — 许可证合规指南
- **双语支持**
  - 完整的 English / 简体中文 文档对等
  - CI 中的自动双语检查（`scripts/check-docs-bilingual.sh`）
  - 命令引用同步验证（两种语言中相同的 `adp` 命令）
- **贡献与安全**
  - `CONTRIBUTING.md` — 贡献指南（双重许可）
  - `SECURITY.md` — 安全策略和报告流程
  - `COMMERCIAL.md` — 商业许可信息
- **许可证**
  - PolyForm Noncommercial 1.0.0 用于源码分发
  - 商业许可可用（见 COMMERCIAL.md）

---

## Phase 4：文档卓越

**状态**：✅ 已完成
**重点**：可操作诊断、增强帮助和实践学习

### 新增
- **增强诊断**（`adp workspace doctor`）
  - 50+ 诊断代码，带可操作建议
  - ✗/✓ 符号用于视觉清晰度
  - 输出中的上下文修复命令
  - 用于自动化的结构化 JSON 输出
- **帮助系统改进**
  - 所有帮助页面中的 "See also" 交叉引用
  - 相关命令建议（19 个顶层，5 个子命令映射）
  - 建议的智能去重
- **Workshop 教程**（`docs/workshop.md`）
  - 4 模块实践教程（工作区 → 任务 → agent → 进度）
  - 自包含，带验证步骤
  - 双语（English / 简体中文）
- **FAQ 扩展**
  - Q15：完整交接示例（operator 过渡）
  - Q18：IDE 集成示例（VS Code、TypeScript、外部跟踪器）
  - Q19：Agent 驱动的 Git 工作流 + 安全考虑
  - Q22：外部工具集成（Python 监控脚本）
  - 132 个代码块，一致的命令格式
- **示例增强**
  - `examples/basic-workspace` — 生产就绪的工作区模板
  - Codex 和 Claude 的 Profile 示例
  - Memory 和 MCP 配置示例

### 质量改进
- 文档评分：4.9/5 → 5.0/5
- 平均任务质量：9.88/10
- 完成效率：比预期快 12 倍

---

## Phase 5：易用性卓越

**状态**：✅ 已完成
**重点**：终端输出打磨和交互安全

### 新增
- **颜色输出**（`internal/output/color.go`）
  - ANSI 颜色支持，具有 TTY 自动检测
  - `NO_COLOR` 环境变量支持（https://no-color.org/）
  - 7 种颜色常量：Red（错误）、Green（成功）、Yellow（警告）、Cyan（命令）、Bold（强调）
  - 应用于：错误消息、成功确认、诊断输出、命令示例
  - PTY 验证的三态行为（TTY on / NO_COLOR / 管道）
- **危险操作确认**（`internal/cli/confirm.go`）
  - 破坏性操作的交互式确认
  - `--yes` / `-y` 标志用于非交互自动化
  - 非 TTY 安全（脚本/CI 中需要显式 `--yes`）
  - 应用于：`workspace remove`、`runtime prune --include-kept`
  - 综合单元测试（8/8 通过）
- **成功消息指导**
  - 关键操作后的 "Next steps" 建议
  - 上下文感知的命令推荐
  - 应用于：`workspace add`、`task add`、`phase add`、`quickstart`、`run`
  - 颜色高亮命令以提高清晰度
- **命令别名**
  - `ws` → `workspace`
  - `t` → `tasks`
  - `s` → `sessions`
  - `e` → `events`
  - `rt` → `runtime`
  - `p` → `phase`
- **拼写建议**
  - 基于 Levenshtein 距离的命令匹配
  - 拼写错误的 "Did you mean" 输出（最大编辑距离 3）
  - 每个未知命令最多 3 个建议
  - 顶层和子命令间一致

### 质量改进
- 易用性评分：4.6/5 → 5.0/5
- 所有 Phase 5 验收测试通过（颜色 PTY、确认、建议）
- 零破坏性变更（向后兼容）

---

## Phase 6：文档精炼

**状态**：✅ 已完成
**重点**：故障排查、视觉打磨和 operator 入门

### 新增
- **故障排查指南**（`docs/troubleshooting.md` / `.zh-CN.md`）
  - 954 行（从初始草稿扩展 66%）
  - 12 个分类章节：安装、工作区、Agent 执行、运行时、任务、阶段、会话、确认、参数、环境、权限、诊断
  - 按错误消息索引组织，便于快速查找
  - 6 个新错误类别：
    - Agent 执行问题（agent command not found、拼写建议）
    - 运行时安全（unsafe runtime parent、保留文件名）
    - 任务所有权（owner mismatch、no claimable task）
    - 阶段管理（phase not found、invalid phase transition）
    - 会话问题（session not found、ambiguous session ID）
    - 交互和确认（operation requires confirmation）
    - 参数验证（--take 冲突、lease 验证）
  - 命令引用对等强制执行（英文/中文）
- **README 视觉打磨**
  - ✨ 核心特性聚焦（5 点摘要，带 emoji）
  - 📖 "从这里开始" 新用户导航（install/onboarding/troubleshooting/FAQ）
  - 保留 emoji 章节标题（🚀⚙️💡🏗️🔧📄）
  - 分层导航：新用户链接（顶部）vs 开发者文档（底部）
- **Operator 入门增强**
  - ✓ 检查点数量：4 → 6（新增 "Move To Durable" 和 "Real Providers" 章节）
  - ⏱️ 预期时间提示：4 → 6（与检查点同步）
  - 每个检查点中的诊断命令示例
  - 故障排查指南的交叉链接
  - 双语对等（英文/中文：6/6 检查点，6/6 时间提示）

---

## [1.0.0] - 2026-06-15（计划中）

### 概述
ADP 1.0.0 标志着本地 terminal-first AI agent 工作流的首个生产就绪版本。所有 Phase 1-6 基础已完成、测试并记录。

### 推荐升级路径
这是首个正式发布版本。新用户应遵循[安装指南](docs/install.zh-CN.md)和 [Operator 入门](docs/operator-onboarding.zh-CN.md)。

### 已知限制
- **Provider 支持**：Codex 和 Claude 适配器是本地进程包装器。外部 CLI 认证、模型可用性、quota 和网络行为是 operator 环境关注点（不是 ADP 保证）。
- **真实 Agent 测试**：默认 CI 使用 fake provider。真实模型调用需要 opt-in smoke 测试（`ADP_REAL_INVOKE_CODEX=1`）。
- **并发性**：ADP 状态是会话本地和基于文件的。高并发多 operator 工作流可能需要外部协调。
- **平台**：主要在 Linux 上测试。macOS 和 Windows（通过 WSL）受支持但测试较少。

### 未来路线图
参见 [docs/project-roadmap-2026-06.md](docs/project-roadmap-2026-06.md) 了解 1.0 后的优先事项：
- 远程 agent 支持（SSH、容器运行时）
- Web 仪表板（可选的可观测性层）
- 实时协作（共享任务板）
- 高级阶段门禁（审批工作流、策略钩子）

---

## 迁移指南

### 从 Dev 构建到 1.0.0
无破坏性变更。现有 `$ADP_HOME` 状态向前兼容。

**可选清理**：
```bash
# 清理旧运行时
adp runtime prune --older-than 24h --include-kept --yes

# 验证工作区健康
adp doctor --verbose
```

### 环境变量
无变更。继续使用：
- `$ADP_HOME` — 工作区/任务/事件存储（默认：`~/.adp`）
- `$ADP_RUNTIME_DIR` — 运行时 overlay 目录（默认：`$TMPDIR/adp-runtime` 或 `/tmp/adp-runtime`）
- `$ADP_WORKSPACE` — 命令的默认工作区

---

## 弃用

无。所有 Phase 1-5 API 对于 1.0.0 是稳定的。

---

## 安全

### Phase 1-5 中已解决
- **运行时隔离**：项目根目录安全检查防止 ADP 覆盖项目文件
- **Git 环境中和**：危险的 Git 变量（GIT_DIR、GIT_WORK_TREE）在运行时被清除
- **不安全运行时父目录检测**：`workspace doctor` 捕获文件系统根、项目根或重叠的运行时父目录
- **确认保护**：破坏性操作（`workspace remove`、`runtime prune --include-kept`）需要显式确认

### 报告
参见 [SECURITY.md](SECURITY.zh-CN.md) 了解安全策略和报告流程。

---

## 贡献者

ADP 由 [@karoc](https://github.com/karoc) 开发和维护。

贡献指南请参见 [CONTRIBUTING.md](CONTRIBUTING.zh-CN.md)。

---

## 许可证

ADP 在 [PolyForm Noncommercial License 1.0.0](LICENSE) 下开源。

商业使用需要单独的付费授权。详见 [COMMERCIAL.md](COMMERCIAL.zh-CN.md)。
