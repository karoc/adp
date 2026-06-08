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
adp env game-a --cd
adp completion --shell bash
adp completion --shell zsh
adp run codex --workspace game-a -- --probe codex-payload
adp run claude -- --probe claude-payload
adp events list --workspace game-a --type run_finished --limit 2
adp sessions list --workspace game-a --agent codex
adp sessions show <session-id>
adp runtime prune --older-than 0s --include-kept --dry-run
adp runtime prune --older-than 0s --include-kept
```

fake Codex 和 Claude 命令会断言：

- 进程工作目录是 ADP runtime root。
- `ADP_WORKSPACE` 设置为已注册 workspace。
- `ADP_SESSION_ID` 存在。
- `ADP_RUNTIME_ROOT` 存在，并且与进程工作目录一致。
- `.adp-runtime.yaml` 存在于 runtime root。
- agent-specific 生成文件存在：
  - Codex：`AGENTS.md` 和 `.codex/config.toml`。
  - Claude：`CLAUDE.md` 和 `.claude/settings.json`。
- 真实项目文件可以通过 runtime root 中的 symlink 看到。
- `--` 之后的参数可以传递给 agent 进程。

脚本还会断言真实项目根目录没有被 ADP runtime artifact 污染：

- `AGENTS.md`。
- `CLAUDE.md`。
- `.codex/`。
- `.claude/`。

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
- 从 runtime root 启动 agent 命令。
- 写入本地 JSONL event log。
- 从本地 events 聚合 session history。
- 为 parent-shell workflow 渲染 shell exports。
- 为 bash 和 zsh 渲染 shell completion。
- 清理 ADP-owned runtime。
- 防止项目根目录污染。

它不验证 provider 账号、远程模型可用性、外部网络访问或交互式 agent 行为。这些属于 ADP 本地 runtime 边界之外，需要 operator-specific 手工验收。

## 本地发布门禁

将 runtime smoke 与标准仓库检查一起运行：

```bash
scripts/runtime-smoke.sh --fake
go test -count=1 ./...
go vet ./...
scripts/check-file-lines.sh
scripts/check-docs-bilingual.sh
git diff --check
```

真实 CLI 检查是可选 release evidence，运行后应单独记录：

```bash
ADP_SMOKE_REAL_CODEX=1 scripts/runtime-smoke.sh --real-codex
ADP_SMOKE_REAL_CLAUDE=1 scripts/runtime-smoke.sh --real-claude
```
