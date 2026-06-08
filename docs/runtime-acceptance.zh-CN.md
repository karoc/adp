# 运行时验收

English: [runtime-acceptance.md](runtime-acceptance.md)

本文档定义 ADP 的本地 runtime smoke 验收路径。目标是验证这个 terminal-first、local-first runtime manager 能构建隔离的 workspace overlay，从该 overlay 启动 agent 命令，记录本地 runtime 历史，并清理 ADP-owned runtime 目录，同时不污染真实项目根目录。

## Smoke 脚本

在仓库根目录运行确定性的 fake-agent smoke：

```bash
scripts/runtime-smoke.sh --fake
```

不传 mode 时，`--fake` 也是默认路径：

```bash
scripts/runtime-smoke.sh
```

脚本会把当前仓库的 `cmd/adp` 二进制构建到临时目录。它不使用全局已安装的 `adp` 二进制。

脚本会创建并在退出时删除以下临时路径：

- `ADP_HOME`。
- `ADP_RUNTIME_DIR`。
- 项目根目录。
- 加入 `PATH` 的 fake agent binary 目录。

fake 路径只要求 Go 和 POSIX shell 环境，不要求安装真实 Codex 或 Claude CLI。

## Fake 验收覆盖

fake smoke 会端到端执行当前 CLI runtime 路径：

```bash
adp init
adp workspace add game-a <temp-project-root>
adp workspace list
adp workspace show game-a
adp workspace doctor game-a
adp workspace doctor
adp doctor game-a
adp doctor
adp version
adp --version
adp tasks add --workspace game-a --priority high --phase p1 "Bind runtime session to task"
adp env game-a --cd
adp completion --shell bash
adp completion --shell zsh
adp completion values workspaces
adp completion values profiles --workspace game-a
adp run codex --workspace game-a --task <task-id> -- --probe codex-payload
adp run claude --task <task-id> -- --probe claude-payload
adp run codex --workspace game-a --task missing-task -- --probe codex-payload
adp events list --workspace game-a --task <task-id> --type run_finished --limit 2
adp sessions list --workspace game-a --agent codex --task <task-id>
adp sessions show <session-id>
adp sessions restore-plan <session-id>
adp runtime prune --older-than 0s --include-kept --dry-run
adp runtime prune --older-than 0s --include-kept
```

fake Codex 和 Claude 命令会断言：

- 进程工作目录是 ADP runtime root。
- `ADP_WORKSPACE` 设置为已注册 workspace。
- `ADP_SESSION_ID` 存在。
- `ADP_RUNTIME_ROOT` 存在，并且与进程工作目录一致。
- 对于 task-bound run，`ADP_TASK_ID` 和任务 metadata 存在。
- `.adp-runtime.yaml` 存在于 runtime root。
- `.adp-runtime.yaml` 会记录 manifest version `1`、`generated_by: adp`、runtime root path，以及绑定的 task ID。
- agent-specific 生成文件存在：
  - Codex：`AGENTS.md` 和 `.codex/config.toml`。
  - Claude：`CLAUDE.md` 和 `.claude/settings.json`。
- 生成的 instructions 包含当前 task context。
- 真实项目文件可以通过 runtime root 中的 symlink 看到。
- `--` 之后的参数可以传递给 agent 进程。

脚本还会检查本地 CLI hardening surface：

- `adp doctor [workspace]` 输出与 workspace 命令组一致的 workspace diagnostics，并支持检查单个 workspace 或全部已注册 workspace；fake smoke 会覆盖 runtime parent 等于 project root 和位于 project root 内部时的拒绝路径。Go 测试覆盖更完整的 runtime parent guard：文件系统根目录、等于 project root、位于 project root 内部、包含 project root、symlink warning 和非目录路径。
- fake smoke 也会通过两个 doctor 入口检查 warning-only agent command/profile diagnostics：project root 中的保留路径、adapter default command fallback、inline command arguments、缺失的非 default profile、逃逸到 workspace 外部的 profile symlink，以及 enabled 但未知的 agent 配置。这些 diagnostics 只做本地静态检查，不运行真实 provider CLI。
- `adp version` 和 `adp --version` 可以在不访问网络、不依赖 provider CLI 的情况下输出 CLI build identity。
- bash 和 zsh completion 脚本包含动态值端点调用。
- `adp completion values workspaces` 从本地状态返回已注册 workspace 名称。
- `adp completion values profiles --workspace <name>` 从 workspace 配置和 profile 文件中返回本地 profile 名称。

脚本还会验收 session restore planning：

- `adp events list --session <session-id> --task <task-id>` 可以查到 Codex session 的 task-bound `run_started` 和 `run_finished` events。
- `adp sessions restore-plan <session-id>` 会打印只读的建议命令，并保留原始 agent arguments。
- 运行 `restore-plan` 不会追加 event log、创建 runtime 状态、修改 task 状态或写入项目根目录。

脚本还会检查缺失的 task ID 会在 fake agent 命令启动前失败。

脚本还会断言真实项目根目录没有被 ADP runtime artifact 污染：

- `AGENTS.md`。
- `CLAUDE.md`。
- `.codex/`。
- `.claude/`。
- `planning/`。
- `tasks.yaml`。
- `phases.yaml`。
- `progress.jsonl`。

## Task Manager 与 Phase Gate 验收

`scripts/task-manager-smoke.sh` 是 workspace-local task、next-work、phase 和 progress report runtime 行为的公开入口和聚焦验收路径。它使用确定性的临时 `ADP_HOME`、临时 `ADP_RUNTIME_DIR` 和临时项目根目录。它不能依赖仓库本地用户状态、全局 `adp` 二进制、provider CLI、网络访问，或写入真实项目根目录的文件。

P9 task-manager smoke modularization 可以把共享 shell helpers 和 JSON report validator 移到 `scripts/` 下的 helper files 中，但这些 helper 属于实现细节。用户和 release gates 仍然运行 `scripts/task-manager-smoke.sh`，`scripts/check-all.sh` 仍然是聚合门禁。

这次拆分只属于维护和 hardening。它不能削弱 runtime acceptance：smoke 仍必须证明 next-work 和 report 生成保持只读，并且没有 planning 或 report artifacts 污染真实项目根目录。

当前 smoke 覆盖已经实现的 task CLI：

- `adp tasks add`
- `adp tasks list`
- `adp tasks next`
- `adp tasks show`
- `adp tasks update`
- `adp tasks claim`
- `adp tasks release`
- `adp tasks block`
- `adp tasks done`
- `adp phase add`
- `adp phase list`
- `adp phase show`
- `adp phase start`
- `adp phase accept`
- `adp phase commit`
- `adp phase push`
- `adp progress`
- `adp progress report [--workspace <name>] [--language <en|zh-CN>] [--format markdown|json]`
- `$ADP_HOME/workspaces/<workspace>/planning` 下的 planning 文件。
- 只读 Markdown 和 JSON progress report 输出，包含专用 JSON validation，并在本地 JSONL event/session 数据存在时包含最近 runtime session evidence。
- 防止项目根目录出现 `planning/`、`tasks.yaml`、`phases.yaml`、`progress.jsonl` 和 report export 污染。

对于 Phase Gate MVP 行为，该 smoke 只能验证实际存在的 CLI。它应覆盖：

- 可以创建、列出、查看 phase records，并推进其生命周期。
- task claim 和 release 命令一次只记录一个 owner，并覆盖 `--lease` 和带 owner 校验的 release。
- acceptance 或 gate records 能记录命令、结果、时间戳和失败证据。
- commit records 能记录已验收阶段的 commit hash 和 branch。
- push records 能记录 remote、branch 和 push 结果；commit 证据保存在同一个 phase record 中。
- progress report 默认输出英文 Markdown，`--language zh-CN` 只作用于 Markdown，并在本地 JSONL events 中存在相应数据时包含 runtime session evidence。
- `adp tasks next --format json` 输出只读 next-work snapshot，包含 workspace、planning source、snapshot 时间、task counts、status counts、请求的 limit、排序后的 candidates，以及存在 eligible work 时的 singular first-candidate `next` 值。
- `adp progress report --format json` 输出机器可读的只读 handoff snapshot，包含 workspace、task 总数、phases、task counts、tasks、按优先级排序的 next work、phase evidence，以及在本地 JSONL event/session 数据存在时的最近 runtime session evidence。
- JSON report 输出保持为跨工具解析 snapshot，不能创建单独的状态存储。
- happy path 会在阶段被视为 pushed 前记录 acceptance、commit 和 push 证据。
- lifecycle guard 检查会拒绝未通过验收前记录 commit、拒绝未记录 commit evidence 前记录 push，并在 phase ledger 存在时拒绝把任务分配到未知 phase。
- next-work 和 report 生成不会追加 events、修改 task 或 phase 状态、创建 runtime 目录、启动 Agent、运行 Git、推断 acceptance、关闭 task、恢复 provider 原生会话、同步 hosted tracker，或把 Markdown 或 JSON report 文件写入项目根目录。
- 所有状态都留在临时 `$ADP_HOME` 下，不污染项目根目录。

不要向 smoke 脚本添加 placeholder commands、TODO assertions、Web UI 检查、SaaS 检查、cloud sync 检查、hosted tracker 检查、hosted orchestration 检查、automatic Git execution、automatic task closure、provider-native resume 或 project-root report export 行为。

## 真实 CLI Smoke

真实外部 agent 检查刻意不进入默认路径。必须同时使用 flag 和环境变量 gate 显式启用：

```bash
ADP_SMOKE_REAL_CODEX=1 scripts/runtime-smoke.sh --real-codex
ADP_SMOKE_REAL_CLAUDE=1 scripts/runtime-smoke.sh --real-claude
```

真实检查是保守的。它会确认外部命令存在，并执行一个轻量 invocation：

- `codex --version`，失败时回退到 `codex --help`。
- `claude --version`，失败时回退到 `claude --help`。

命令名称可以覆盖：

```bash
ADP_SMOKE_REAL_CODEX=1 ADP_SMOKE_CODEX_BIN=/path/to/codex scripts/runtime-smoke.sh --real-codex
ADP_SMOKE_REAL_CLAUDE=1 ADP_SMOKE_CLAUDE_BIN=/path/to/claude scripts/runtime-smoke.sh --real-claude
```

这些检查不能证明真实交互式 agent session 已完整可用。在声明 real-agent compatibility 的 release 前，operator 还应手工确认 `adp run codex` 和 `adp run claude` 能在该机器上启动预期的本地 CLI，并确认凭据、模型选择和外部工具设置符合 operator 环境。

## 验收边界

该 smoke 验证 ADP 的 runtime 责任：

- 创建隔离 runtime overlay。
- 注入 runtime 环境变量。
- 通过 `adp run --task <task-id>` 进行 runtime task binding。
- 从 runtime root 启动 agent 命令。
- 写入本地 JSONL event log。
- 从本地 events 聚合 session history。
- 基于非敏感 invocation snapshot 打印只读 session restore plan。
- 为 parent-shell workflow 渲染 shell exports。
- 为 bash 和 zsh 渲染 shell completion。
- 为 workspace 和 profile 提供动态本地 completion 值端点。
- 通过 `adp doctor` 提供全局 workspace diagnostics。
- 通过 workspace 和全局 doctor 命令检查 runtime parent 安全性，覆盖文件系统根目录、project-root overlap、symlink warning 和非目录场景。
- 通过 workspace 和全局 doctor 命令检查 agent command/profile diagnostics，覆盖 adapter default fallback、inline command arguments、路径型 command wrapper、缺失或重复的 profile 文件、profile path escape、未知 enabled agent，以及 project root 中的保留路径。
- 通过 `adp version` 输出本地 build identity。
- 通过 `scripts/task-manager-smoke.sh` 验收 workspace-local task manager。
- 验收 Phase Gate ledger evidence、claim lease、release owner check 和 lifecycle ordering。
- 验收 progress report 的 Markdown 和 JSON 输出；当 event/session 数据存在时，验收从本地 JSONL events 派生的 runtime session evidence，并确认不会追加 event log、创建 runtime、写入 project root、导出 report 文件或创建单独状态存储。
- 清理 ADP-owned runtime。
- 针对当前版本 ADP manifest 的 runtime prune compatibility checks。
- 防止项目根目录污染。

它不验证 provider 账号、远程模型可用性、外部网络访问或交互式 agent 行为。这些属于 ADP 本地 runtime 边界之外，需要 operator-specific 手工验收。

## 本地发布门禁

将 runtime smoke 与标准仓库检查一起运行：

```bash
scripts/check-all.sh
scripts/runtime-smoke.sh --fake
go test -count=1 ./...
go vet ./...
scripts/check-file-lines.sh
scripts/check-docs-bilingual.sh
git diff --check
```

`scripts/check-all.sh` 是本地 handoff 和 CI 使用的聚合门禁。展开后的命令列表适合在失败时单独定位问题。

真实 CLI 检查是可选 release evidence，运行后应单独记录：

```bash
ADP_SMOKE_REAL_CODEX=1 scripts/runtime-smoke.sh --real-codex
ADP_SMOKE_REAL_CLAUDE=1 scripts/runtime-smoke.sh --real-claude
```
