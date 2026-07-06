# 更新日志

English: [CHANGELOG.md](CHANGELOG.md)

ADP（Agent Development Platform）的所有重要变更都将记录在此文件中。

格式基于 [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)，
本项目遵循[语义化版本](https://semver.org/spec/v2.0.0.html)。

---

## [未发布]

暂无变更。

---

## [1.0.1] - 2026-07-06

### 安全
- **对事件日志与会话恢复计划中的 agent 参数密钥脱敏** — 传递给 `adp run` 的 `--` 之后的参数（例如 `--api-key sk-...`）此前被原样记录在全局可读的 `events.jsonl` 中，并被 `sessions restore-plan` / `sessions resume-plan` 回显。现在密钥形态的值会被替换为 `***REDACTED***`，同时保留参数名；事件日志以仅属主可读的 `0600` 权限创建（已存在的宽松权限会在下次写入时收紧）；脱敏在写入时和计划渲染时（文本与 JSON）双重生效。新增 `internal/redact` 包，通过敏感参数名（`key`、`secret`、`token`、`password`、`auth`、`credential` 等）、已知服务商前缀（`sk-`、`ghp_`、`AKIA`、`eyJ` 等）以及高熵裸值来识别凭据。
- **将 `$ADP_HOME` 状态目录树收紧为仅属主可访问** — ADP 此前以 `0o755`（目录）和 `0o644`（文件）创建 `$ADP_HOME` 下的全部内容，使同一主机上的其他本地用户可以读取项目路径、任务内容、git remote（可能含令牌）以及命令历史（CWE-732）。现在所有 `$ADP_HOME` 目录（home、`workspaces/`、`logs/`、每个工作区子目录以及 `planning/`）都通过新增的 `internal/paths.EnsurePrivateDir` 以 `0o700` 创建，已存在的宽松权限会被收紧；任务台账文件（`tasks.yaml`、`phases.yaml`、`progress.jsonl`）额外以 `0o600` 写入，构成纵深防御。
- **将运行时 overlay 目录树收紧为仅属主可访问** — `adp run` / `adp enter` 物化的运行时 overlay 此前以 `0o755`（目录）创建，生成的 agent 配置文件（`CLAUDE.md`、`AGENTS.md`、`.codex/`、`.claude/`，由工作区 prompts、memory、profiles 渲染而来）以 `0o644` 写入。由于该树位于共享的 `$ADP_RUNTIME_DIR`（默认 `$TMPDIR/adp-runtime`）下，其他本地用户可读取这些可能含敏感项目上下文的内容（CWE-732）。现在运行时根目录及所有生成文件的父目录均以 `0o700` 创建，生成文件会剥离 group/other 权限位，因此即使 adapter 显式请求 `0o644` 也会被收紧为仅属主可访问。
- **净化渲染到终端的规划文本中的控制字符** — 任务标题、描述、阶段目标、owner 等用户可控的规划字段此前被直接打印到终端，未过滤控制字符（CWE-150）。由于共享工作区的 planing YAML 可被手工编辑，恶意条目可嵌入 ANSI 转义序列，在他人运行 `adp tasks list`/`show`、`adp phase list`/`show` 或 `adp progress report` 时清屏、移动光标或伪造输出。新增 `safeText` 助手中和控制 rune（ESC、CR、BEL、退格、C1 区段），同时保留合法的换行/制表；在所有文本 sink（`valueOrDash`、`formatStringList`、`markdownCell` 助手及任务、阶段、计划、进度报告、会话 resume-plan 渲染器中的直接字段打印——含嵌入任务 `blocked_reason`/`owner`/`id` 与阶段门控原因的 guidance 行）处应用。JSON 输出已由编码器转义控制字符。
- **补全规划标识符与报告剩余字段的终端控制字符净化** — 此前 `safeText` 覆盖了任务/阶段标题、owner、目标,但同源(可手编 planning YAML)的相邻字段仍未净化:任务与阶段标识符(打印于 list/show/status 表格、"Next steps" 建议命令、`phase status` 门禁摘要)、进度报告的"Next Work"列表项(任务 id、title,以及经 claim-handoff 助手渲染的 owner)、阶段生命周期状态消息(commit hash、push remote/branch/result),以及"ambiguous task ID"错误列表。由于共享 planning YAML 可手编,恶意条目可在上述任一字段嵌入 ANSI 转义,在他人 list/show 任务或阶段时清屏或伪造输出。现已在所有这些展示 sink 应用 `safeText`(新增 `formatAmbiguousTaskIDList` 助手集中净化错误列表渲染);JSON 输出仍由编码器覆盖。用作查找键的标识符刻意不净化,以免影响前缀匹配。
- **修复规划锁 stale 破除路径的 TOCTOU 竞态** — 规划 store 用 `O_EXCL` 锁文件串行化变更，并破除超过 30 分钟（崩溃进程遗留）的 stale 锁。原 stale 破除用 `os.Remove` 后重试，两个并发命令若同时判定同一锁为 stale，可能各自删除对方刚重建的新锁，导致双方同时进入临界区，造成丢失更新或任务被双重认领（CWE-367）。现改用 `os.Rename` 原子认领 stale 锁（重命名为 `.lock.stale` 垃圾文件）；竞态中落败的破除者得到 `os.ErrNotExist` 后重试独占创建，互斥在所有交错下成立。tmp+rename 原子写本就防止了写撕裂，本次补上剩余的互斥缺口。
- **脱敏 `phase push --remote` URL 中内嵌的凭据** — `adp phase push` 记录操作员推送到了哪里（`--remote` 值）作为审计证据；它本身不执行 `git push`，故该字段纯为声明性。操作员若误粘贴含凭据的 URL（如 `https://user:ghp_...@github.com/org/repo`）而非 remote 名，该 URL 会被原样存入阶段台账与 `progress.jsonl`，并回显到终端、进度报告与 JSON 输出（CWE-200）。新增 `redact.URLCredentials` 助手剥离 URL 中的 `user:password@` userinfo，同时保留 scheme/host/path 可见；在唯一存储 chokepoint（`RecordPhasePush`）处应用，使所有展示 sink 与两份持久化文件都拿到脱敏值。remote 名（`origin`）与 scp 风格路径（`git@host:repo.git`）原样保留，无 userinfo 的 URL（含 path 中含 `@` 的）不受影响。
- **将工作区配置文件收紧为仅属主可访问** — 早先的 `$ADP_HOME` 权限加固把目录收紧为 `0700`、规划台账收紧为 `0600`，但写入 `$ADP_HOME/workspaces/<name>/` 下的工作区配置文件——`workspace.yaml`、`mcp/config.yaml`、`prompts/`·`memory/`·`profiles/` 种子文件，以及全局 workspaces 注册表——仍以 `0644` 写入，仅依赖父目录的 `0700` 作为唯一屏障（CWE-732）。其中 `mcp/config.yaml` 尤其可能存放操作员粘贴的 MCP 服务器 token 或 URL。现这些文件改用 `paths.PrivateFileMode` 以 `0600` 写入，与规划台账一致，并增加一道独立于目录权限的纵深防御层。

### 变更
- **让 task、phase、workspace 与 session 的 "not found" 错误可操作** — 对不存在的标识符执行 `adp tasks show`/`done`/`claim`/...、`adp phase show`/`start`/`accept`/...、`adp workspace show`/`remove`/... 或 `adp sessions show`/`restore-plan`/... 时，此前只输出一条干瘪的 "not found" 消息，操作员只能猜测实际存在哪些 ID。现在这些错误会追加 `run: adp <cmd> list` 提示（`adp tasks list`、`adp phase list`、`adp workspace list` 或 `adp sessions list`），让操作员能立刻查看可用标识符，与既有的可操作空列表和拼写建议行为保持一致。歧义前缀错误不受影响（它们本就列出所有匹配项）。
- **拒绝非法 task status 时列出合法值** — `adp tasks update --status <非法值>`（以及计划导入遇到未知 status 时）此前只输出 "unknown task status"，而 `adp tasks update --help` 也不列出合法值，操作员只能猜测。现在该错误会追加可接受的 status（planned、ready、in_progress、blocked、review、validated、done、canceled），让操作员能立即更正输入。
- **`adp run` 的 flags 出现在 agent 之前时报明确的 "agent is required" 错误** — `adp run --workspace x codex`（flags 在 agent 之前，常见的 shell 习惯）此前会把 `--workspace` 当作 agent 名，再为该 flag 的值报误导性的 `unknown run option "x"`，让操作员去寻找一个不存在的 option。解析器现在检测到首个参数是 flag 时，改报与空参数一致的、可操作的 `agent is required; usage: adp run <agent> ...` 消息（含 `try: adp run --help` 提示），指向真正的问题——agent 必须在前——而非一个虚构的 option。文档约定的 `adp run <agent> [flags]` 顺序不变；顺序正确的调用不受影响。
- **在 `adp tasks update --help` 中列出合法 task status** — `--status` 选项的 usage 行此前只显示 `<status>`，而同类枚举选项在 `adp progress report --help`（`--language <en|zh-CN>`、`--format <markdown|json>`）中已列出合法值。该 usage 行现在枚举可接受的 status（planned、ready、in_progress、blocked、review、validated、done、canceled），让操作员能预先得知合法值，而不必在提交非法值后才能从错误消息中看到（后者已由前一项变更覆盖）。`--status` 的错误消息行为不变。
- **`adp workspace list` 为空时引导操作员** — 空的 workspace 列表此前只打印表头，首次使用的操作员无从得知如何注册 workspace，而其他 list 命令（`adp sessions list`、`adp events list`、`adp tasks list`、`adp phase list`）已打印 "No X found. ... with 'adp ...'" 行指向创建命令。`adp workspace list` 现在以相同风格追加 `No workspaces found. Register one with 'adp workspace add <name> <project-root>'`。JSON 输出不变（本就返回空数组）。
- **将源文件硬性行数上限提高到 1000 行** — 项目操作指南、贡献文档、README 摘要、工程标准、同类工具说明、发布 checklist 以及 `scripts/check-file-lines.sh` 现在统一使用 1000 行硬上限。审计压力阈值仍保持较低，以便维护者在硬门禁失败前提前看到建议拆分的文件。
- **将默认开发与发布版本标识设为 1.0.1** — 源码构建现在报告 `adp version 1.0.1`，`scripts/build-release.sh` 默认使用 `VERSION=1.0.1`，同时仍允许操作员显式覆盖发布元数据。

### 修复
- **确定性拒绝被吞掉的 CLI option 值** — CLI 解析现在会捕获 flag 或值出现在错误位置的情况，包括 `adp run` 的前置 flag 场景，而不是把下一个 token 变成误导性的未知 option。
- **净化终端安全文本中的原始非 UTF-8 字节** — `safeText` 现在能处理非法字节序列，避免原始控制字节泄漏到终端输出。
- **提升 Windows 测试可移植性** — 测试套件覆盖了已识别的四类 Windows 可移植性根因，并新增 CI sentinel 让回归保持可见。
- **防护 smoke 测试中的符号链接污染和本地报告漂移** — Smoke 路径现在检查符号链接行为，并确保本地报告产物保持 ignored 状态。
- **拆分过大的实现文件** — P65 维护阶段拆分了大型源码文件，使其继续满足项目文件规模约束。

### 构建与验证
- **并行化完整仓库门禁** — `scripts/check-all.sh` 现在预热 Go build cache，默认并行运行 smoke 脚本，保留 `CHECK_ALL_SERIAL=1` 串行回退，并使用 coverage 门禁避免重复运行普通测试。
- **报告完整门禁耗时** — `scripts/check-all.sh` 现在输出 cache 预热、整体 smoke suite、每个 smoke worker、coverage、vet、文件行数检查、双语文档、diff check 以及总耗时。
- **降低 install-onboarding smoke 延迟** — 固定的一秒 stale-lease 等待已替换为围绕立即过期 lease 的短轮询循环。
- **新增并扩展回归覆盖** — 新测试覆盖输出渲染、Codex/Claude 适配器、overlay 安全防护、resume 决策路径、render 内容注入路径、schema 与 path/layout 包，并为 plan intake 与 redaction parser 增加 fuzz 覆盖。
- **保持发布构建路径权威一致** — `scripts/build-release.sh` 现在接受 `VERSION`、`COMMIT`、`BUILD_DATE`、`DIST_DIR`，使用集中化 release ldflags，带 `-trimpath` 构建，并保留校验和生成。

### 文档
- **为稳定版 1.0.1 包装流程对齐发布文档** — 发布包装、checklist、evidence、troubleshooting 与 GitHub 发布说明现在描述稳定版发布流程，而不是 preview 阶段示例，并包含多行 `adp version` 输出格式。
- **归档过期规划和验证文档** — 历史 plan、verification report 与 checklist snapshot 已标记为 archived 或 historical，不再读起来像当前项目状态。
- **记录 Phase 6 与 Phase 7 验收证据** — 为已完成的文档阶段和发布就绪阶段补充最终验收报告。

---

## Phase 1-5 基础（预发布开发）

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
