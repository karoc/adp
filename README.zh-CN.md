# ADP

English: [README.md](README.md)

ADP 是 Agent Development Platform 的缩写。它是面向 terminal-first AI Agent 工作流的 Agent Runtime Environment 和 Agent Workspace Manager。

ADP 把 AI Agent 配置保存在项目目录之外，并在 Agent 启动时构建临时 runtime overlay。Agent 可以看到 `AGENTS.md`、`CLAUDE.md`、`.codex/`、`.claude/` 等生成文件，但真实项目目录保持干净。

## 当前 MVP

已实现 Phase 1 基础能力。如果你是首次试用 ADP，先看 [快速开始](#快速开始)；下面列表是当前命令面的参考快照。

- `adp init`
- `adp workspace add <name> <project-root>`
- `adp workspace list`
- `adp workspace show <name>`
- `adp workspace doctor [name]`
- `adp doctor [workspace]`
- `adp workspace remove <name>`
- `adp workspace rename <old-name> <new-name>`
- `adp env <workspace> [--cd]`
- `adp shell-hook [--shell <sh|bash|zsh>] [--name <function-name>]`
- `adp completion [--shell <bash|zsh>] [--command <name>]`
- `adp completion values <agents|workspaces|profiles|tasks|phases|sessions|owners|statuses> [--workspace <name>]`
- `adp version`
- `adp events list [--workspace <name>] [--session <session-id>] [--task <task-id>] [--type <event-type>] [--limit <n>]`
- `adp sessions list [--workspace <name>] [--agent <agent>] [--task <task-id>] [--limit <n>]`
- `adp sessions show <session-id>`
- `adp sessions restore-plan <session-id>`
- `adp sessions resume-plan <session-id> [--workspace <name>] [--owner <owner>] [--lease <duration>] [--agent <agent>] [--format <text|json>]`
- `adp runtime prune [--older-than <duration>] [--include-kept] [--dry-run]`
- `adp tasks add [--workspace <name>] [--priority <value>] [--phase <value>] [--description <text>] <title>`
- `adp tasks list/next/take/stale/show/update/claim/renew/release/done/block`
- `adp plan preview [--workspace <name>] --file <path|-> [--format <text|json>]`
- `adp plan apply [--workspace <name>] --file <path|-> [--format <text|json>]`
- `adp plan doctor [--workspace <name>] [--format <text|json>]`
- `adp phase add/list/show/status/start/accept/commit/push`
- `adp progress [--workspace <name>] [--format <text|json>]`
- `adp progress report [--workspace <name>] [--language <en|zh-CN>] [--format <markdown|json>]`
- `adp run <agent> [--workspace <name>] [--profile <profile>] [--task <task-id>|--take --owner <owner> [--lease <duration>]] [--keep-runtime] [-- <agent-args>...]`
- `adp enter <workspace> [--keep-runtime]`
- `$ADP_HOME` 下的本地 workspace registry
- `$ADP_RUNTIME_DIR` 下的 symlink runtime overlay
- Codex 和 Claude adapter layer
- JSONL event log
- 基于本地事件聚合的 session history 视图
- 检查本地 workspace 配置问题的 diagnostics
- `examples/basic-workspace` 示例 workspace 配置
- process runner 和 workspace shell

`adp sessions resume-plan <session-id> [--workspace <name>] [--owner <owner>] [--lease <duration>] [--agent <agent>] [--format <text|json>]` 提供只读的 cross-tool ADP work-context resume guidance。它补充 `restore-plan`：`restore-plan` 聚焦于某个已记录 session 的 rerun guidance，而 `resume-plan` 会增加 owner、lease、task、phase 和 target-agent context，供 operator 复核。

命令发现仍然留在 CLI 内完成。用 `adp --help` 查看根命令列表，用 `adp <command> --help` 查看命令组，例如 `adp tasks --help`，用 `adp <command> <subcommand> --help` 查看叶子命令，例如 `adp tasks take --help`。如果使用的是 `ADP_BIN`、`adp_local` 或打包后的二进制名，把同一模式中的命令名替换掉即可。叶子 help 可能通过 `See also:` 指回父级 help；如果某个构建打印友好的 `try:` hint，把它当作指向同一 help surface 的提示，而不是自动 action 或状态变更。

## 快速开始

安装与 bootstrap 细节见 [docs/install.zh-CN.md](docs/install.zh-CN.md)。面向新 operator 的具体演练见 [docs/operator-onboarding.zh-CN.md](docs/operator-onboarding.zh-CN.md)。

如果这是你的首次试运行，请把 [docs/operator-onboarding.zh-CN.md](docs/operator-onboarding.zh-CN.md) 作为引导路径。它会说明这次演练验证什么、应该期待什么输出、哪些命令只读，以及何时从临时状态切换到持久本地使用。下面的命令块是紧凑的 smoke-first 参考版本。

先选择源码、已构建二进制或临时安装路径。下面的参考路径会构建本地二进制并使用 `ADP_BIN`；如果要验证 release artifact，把 `ADP_BIN` 设置为已安装的 artifact 路径。开发时如果想从源码运行，可以把 `"$ADP_BIN" <command>` 替换为 `go run ./cmd/adp <command>`。

这次演练使用临时 ADP 状态、临时 project root 和 fake `codex` 命令。它不依赖真实 Codex 或 Claude CLI，不会运行 Git，也应该让真实 project root 保持没有 ADP 生成文件。流程有意接近成熟 CLI quickstart 的组织方式：先得到一个可运行命令，初始化本地状态，添加 workspace，先 inspect 再 mutation，启动 Agent 时原子领取任务，最后验证本地 evidence。

```bash
mkdir -p bin
go build -o ./bin/adp ./cmd/adp
ADP_BIN="$PWD/bin/adp"
"$ADP_BIN" version

ADP_SMOKE_ROOT="$(mktemp -d)"
export ADP_HOME="${ADP_SMOKE_ROOT}/adp-home"
export ADP_RUNTIME_DIR="${ADP_SMOKE_ROOT}/runtime"
mkdir -p "${ADP_SMOKE_ROOT}/project" "${ADP_SMOKE_ROOT}/fake-bin"
printf 'module example.com/adp-smoke\n' > "${ADP_SMOKE_ROOT}/project/go.mod"
printf 'package main\n' > "${ADP_SMOKE_ROOT}/project/main.go"

cat > "${ADP_SMOKE_ROOT}/fake-bin/codex" <<'SH'
#!/usr/bin/env sh
printf 'fake codex cwd=%s args=%s\n' "$(pwd)" "$*"
test -n "${ADP_SESSION_ID:-}"
test -n "${ADP_RUNTIME_ROOT:-}"
test -n "${ADP_TASK_ID:-}"
test "$(pwd)" = "$ADP_RUNTIME_ROOT"
test -f "$ADP_RUNTIME_ROOT/AGENTS.md"
test -f "$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
SH
chmod +x "${ADP_SMOKE_ROOT}/fake-bin/codex"
export PATH="${ADP_SMOKE_ROOT}/fake-bin:${PATH}"

"$ADP_BIN" init
"$ADP_BIN" workspace add game-a "${ADP_SMOKE_ROOT}/project"
"$ADP_BIN" workspace list
"$ADP_BIN" workspace show game-a
"$ADP_BIN" workspace doctor game-a
"$ADP_BIN" doctor game-a
"$ADP_BIN" env game-a --cd
"$ADP_BIN" completion values agents
"$ADP_BIN" completion values workspaces
"$ADP_BIN" completion values profiles --workspace game-a
"$ADP_BIN" completion values statuses
TASK_ID=$("$ADP_BIN" tasks add --workspace game-a --priority high "Validate isolated first run" | sed -n 's/^task \(task-[^ ]*\) added$/\1/p')
test -n "$TASK_ID"
"$ADP_BIN" completion values tasks --workspace game-a
"$ADP_BIN" tasks next --workspace game-a --format json
"$ADP_BIN" run codex --workspace game-a --take --owner first-agent --lease 30m -- --example-smoke
"$ADP_BIN" tasks show --workspace game-a "$TASK_ID"
"$ADP_BIN" tasks renew --workspace game-a "$TASK_ID" --owner first-agent --lease 30m
"$ADP_BIN" tasks stale --workspace game-a --format json
"$ADP_BIN" progress report --workspace game-a --format json
SESSION_ID=$("$ADP_BIN" sessions list --workspace game-a --agent codex --task "$TASK_ID" | sed -n '2s/ .*//p')
test -n "$SESSION_ID"
"$ADP_BIN" sessions restore-plan "$SESSION_ID"
"$ADP_BIN" plan doctor --workspace game-a --format json

BOARD_TASK_ID=$("$ADP_BIN" tasks add --workspace game-a --priority normal "Validate board pickup" | sed -n 's/^task \(task-[^ ]*\) added$/\1/p')
test -n "$BOARD_TASK_ID"
TAKEN_ID=$("$ADP_BIN" tasks take --workspace game-a --owner second-agent --lease 30m | sed -n 's/^task \(task-[^ ]*\) taken .*/\1/p')
test -n "$TAKEN_ID"
"$ADP_BIN" tasks release --workspace game-a "$TAKEN_ID" --owner second-agent
"$ADP_BIN" tasks done --workspace game-a "$TASK_ID"
"$ADP_BIN" events list --workspace game-a --task "$TASK_ID" --limit 5
"$ADP_BIN" tasks list --workspace game-a --format json
"$ADP_BIN" tasks next --workspace game-a --limit 0 --format json
"$ADP_BIN" progress --workspace game-a --format json
"$ADP_BIN" progress report --workspace game-a
"$ADP_BIN" sessions list --workspace game-a --agent codex --task "$TASK_ID"
"$ADP_BIN" completion values sessions --workspace game-a
"$ADP_BIN" runtime prune --older-than 24h --dry-run
ROOT_LEAKS="$(find "${ADP_SMOKE_ROOT}/project" -maxdepth 2 \( -name AGENTS.md -o -name CLAUDE.md -o -name .codex -o -name .claude -o -name .adp-runtime.yaml -o -name planning -o -name tasks.yaml -o -name phases.yaml -o -name progress.jsonl \) -print)"
test -z "$ROOT_LEAKS"
```

预期结果：fake provider 会打印 runtime 工作目录，本地 inspection 命令会返回 task/session/progress evidence，最后的 project-root 泄漏检查通过且没有输出。持久本地使用时，可以把 `ADP_HOME` 设置为 `~/.adp` 等稳定目录；运行真实 Agent 前，先安装并认证外部 provider CLI，然后再使用 `adp run codex ...` 或 `adp run claude ...`。当你需要一份包含 Codex 和 Claude profile、base prompt、shared memory 与 MCP 设置的可复制 workspace 配置时，再使用 `examples/basic-workspace`。

常用环境变量：

- `ADP_HOME`：ADP home 目录，默认 `~/.adp`。
- `ADP_RUNTIME_DIR`：临时 runtime overlay 的父目录，默认是系统临时目录下的 `adp-runtime`。不要把它指向文件系统根目录、project root、project root 内部目录，或包含 project root 的父目录。优先使用直接的本地目录；symlink runtime parent 会被 doctor 命令作为 warning 报告。
- `ADP_WORKSPACE`：可作为命令默认 workspace。
- `ADP_TASK_ID`、`ADP_TASK_TITLE`、`ADP_TASK_STATUS`、`ADP_TASK_PRIORITY`、`ADP_TASK_PHASE`、`ADP_TASK_OWNER`、`ADP_TASK_CLAIMED_AT` 和 `ADP_TASK_LEASE_EXPIRES_AT`：当被选中的 task 存在对应值时，在通过 `adp run <agent> --task <task-id>` 或 `adp run <agent> --take --owner <owner>` 启动的 runtime 内可用。

当没有传入 `--workspace` 且没有设置 `ADP_WORKSPACE` 时，`adp run` 会尝试用当前目录匹配已注册的 project root。

## Runtime 模型

`adp run` 会构建一个看起来像项目根目录的临时 runtime root：

```txt
/tmp/adp-runtime/game-a-<session>/
├── AGENTS.md
├── CLAUDE.md
├── .adp-runtime.yaml
├── .codex/
├── .claude/
├── go.mod -> /srv/game-a/go.mod
└── internal -> /srv/game-a/internal
```

Agent 专属文件由 ADP workspace config 生成。真实项目文件通过 symlink 进入 runtime root。ADP 生成路径在 runtime 视图中优先，原始项目目录不会被修改。当项目已经存在 `.codex/` 或 `.claude/` 等 provider-local configuration directories 时，非冲突子文件仍会在 runtime overlay 中可见；ADP 只在 `.codex/config.toml` 和 `.claude/settings.json` 等精确生成路径上优先。

`adp env <workspace> --cd` 会输出 POSIX shell exports，并保留 runtime overlay，适合 shell-hook 工作流让调用方 shell 进入 runtime。

`adp shell-hook --shell bash` 会输出一个 shell 函数，用 `adp env <workspace> --cd` 构建 runtime，并在父 shell 中执行返回的 exports 和 `cd`。当前支持 `sh`、`bash`、`zsh`。

`adp completion [--shell <bash|zsh>] [--command <name>]` 会为当前 CLI 命令面输出稳定的 shell completion。省略 `--shell` 时默认输出 bash completion。可选 command name 用于给打包后的二进制名或别名生成非 `adp` 名称的 completion；该命令必须在当前 shell 中可用，因为生成的脚本会调用它获取动态值。Completion 覆盖 root command、subcommand、option、shell/format/language/event type/runtime age 等有限 option 值，以及 agent、workspace、workspace profile、task ID、phase ID、session ID、task status 和 owner 等只读本地候选。动态候选只读取 `$ADP_HOME` 状态，不得修改 planning、runtime、provider、Git 或 project-root 状态。

P16 会用本地 metadata contract 强化命令面，防止 usage text、dispatch wiring 与 bash/zsh completion 彼此漂移。这仍属于现有手写 CLI 实现的一部分；不会引入新的 CLI 框架，也不会增加 Web UI、dashboard、SaaS tracker、hosted orchestration、automatic Git workflow、automatic task closure 或 provider-native resume 路径。

`adp events list` 会读取 `$ADP_HOME/logs/events.jsonl`，按 workspace、session、task、事件类型和数量限制输出最近的 runtime 事件。

`adp sessions list [--workspace <name>] [--agent <agent>] [--task <task-id>] [--limit <n>]` 会按 session 聚合本地 event log，方便在终端中查看最近的 Agent 运行记录。

`adp sessions show <session-id>` 会输出某个已记录 session 的有序事件；如果事件包含 workspace、agent、task ID、runtime path、exit code 和 duration 等字段，也会一起展示。

`adp sessions restore-plan <session-id>` 会读取一个已记录 session，并在非敏感 invocation 数据足够时打印只读的建议 `adp run ...` 命令。它不会执行命令、启动 Agent、创建 runtime、追加 events、修改 task 状态、写入项目根目录或恢复 provider 原生会话。详见 [docs/session-restore.zh-CN.md](docs/session-restore.zh-CN.md)。

`adp sessions resume-plan <session-id> [--workspace <name>] [--owner <owner>] [--lease <duration>] [--agent <agent>] [--format <text|json>]` 会把同一个只读 guidance 扩展为 ADP work-context resume planning。它可以建议一次 same-tool 或 cross-tool 的新 `adp run ...` 命令，包含 owner 和 lease preflight notes，展示来自本地 ledger 的 phase context，并用机器可读的 `side_effect` 给每条建议命令分类，例如 `inspect`、`task_mutation` 或 `runtime_creation`。Same-tool plan 可以复用 invocation snapshot 中非敏感的 `--profile`、`--keep-runtime` 和 `--` 后置 agent arguments。Cross-tool plan 会保留 `--keep-runtime` 这类 ADP-safe runtime options，但会省略 provider-specific profile 和 agent arguments，并明确输出 guidance。命令本身不得 claim 或 renew tasks、complete tasks、accept phases、运行 Git、创建 runtimes、追加 events、写入项目根目录，或 attach 到 provider-native Codex 或 Claude conversations。

Operator examples：

```bash
adp sessions resume-plan <session-id> --owner handoff-agent --lease 2h --format text
adp sessions resume-plan <session-id> --agent claude --owner reviewer --lease 1h --format json
```

`adp workspace doctor [name]` 会检查 workspace 配置、project root 可访问性、runtime parent 安全性、prompt、memory、MCP、profile 文件引用、agent command 设置，以及 project root 中的保留路径。它会把 adapter default command fallback、写在 command 字段里的 inline arguments、缺失或不可执行的路径型 command wrapper，以及缺失、重复或逃逸到 workspace 外部的非 default profile 报告为本地 diagnostics。不传 name 时检查所有已注册 workspace；发现 error 级 diagnostics 时返回非零退出码。

`adp doctor [workspace]` 是同一组本地 workspace 检查的全局 diagnostics 入口。它适合需要在终端中直接运行 diagnostics、但不想先进入 `workspace` 命令组的工作流。

`adp version` 和 `adp --version` 会输出 CLI build identity。打包 binary 可以在构建时注入 version、commit 和 build-date；开发构建默认回退为 `dev`。

`adp runtime prune` 会清理 `$ADP_RUNTIME_DIR` 下过期的 ADP runtime 目录。只有包含当前版本且结构自洽的 `.adp-runtime.yaml`，其中 `generated_by: adp` 且 `runtime_root` 与目录一致的 runtime 才会进入 prune 候选；默认保留 `keep: true` 的 runtime，除非传入 `--include-kept`；`--dry-run` 只报告候选项，不删除。

`adp tasks` 和 `adp progress` 管理 `$ADP_HOME/workspaces/<workspace>/planning` 下的 workspace-scoped 规划和执行进度。只读 task、phase 和 progress 视图支持 `--format json`，方便本地工具和子 Agent 获取机器可读 planning snapshot；task objects 会通过 `claim_state` 以及存在时的 `owner`、`claimed_at` 和 `lease_expires_at` 暴露看板可见性。`adp phase status` 会输出紧凑的 gate snapshot，包含当前 open phase、下一个 planned phase、是否可以启动下一阶段，以及下一步必需动作。`adp plan doctor` 会对 task、phase、progress log、lock 和 phase gate 一致性做只读本地 diagnostics；存在 error-level diagnostics 时返回退出码 `2`。权威状态仍保存在 `$ADP_HOME` 下，task 或 phase 变更仍然必须通过显式命令完成。使用 `adp tasks next --format json` 预览看板；当 worker 只需要从看板原子领取一项、但不启动 Agent 时，使用 `adp tasks take --owner <owner>`；当任务领取和 runtime 启动需要处在同一个命令边界时，使用 `adp run <agent> --take --owner <owner>`。长时间执行的 worker 可以用 `adp tasks renew` 续租；中断后的过期 lease 可通过只读 `adp tasks stale` 查看，然后只在 ADP ownership rules 允许后，通过显式 `adp tasks take` 或 `adp tasks claim` 接管。`adp run <agent> --task <task-id>` 和 `adp run <agent> --take ...` 会把本地任务状态绑定到 runtime 环境变量、生成的 adapter instructions、events 和 sessions，同时不会把 planning 文件写入真实项目根目录。详见 [docs/task-management.zh-CN.md](docs/task-management.zh-CN.md)。

`adp progress report [--workspace <name>] [--language <en|zh-CN>] [--format markdown|json]` 会向 stdout 打印本地 planning/execution handoff snapshot。默认输出仍然是英文 Markdown；`--language zh-CN` 只作用于 Markdown。`--format json` 会输出机器可读的只读 snapshot，包含 workspace、task 总数、phases、task counts、带有 `claim_state` 且在存在时带有 owner/lease 字段的 task objects、按优先级排序的 next work、phase evidence，以及在本地 JSONL event/session 数据存在时的最近 runtime session evidence。JSON 输出用于跨工具解析，不能成为单独的状态存储。该 report 命令是只读的，不会追加 events、修改 task 或 phase 状态、创建 runtime 目录、启动 Agent、运行 Git、恢复 provider 原生会话，或把报告文件写入项目根目录。

P3 提供项目规划和执行进度管理的本地 phase ledger。它会在 `$ADP_HOME` 下记录任务归属、可选 claim lease、验收记录、commit 记录、push 记录和明确的阶段门禁纪律。该能力仍然保持 terminal-first、local-first；它不是 Web dashboard、SaaS tracker、cloud sync layer 或 hosted orchestration service。

`adp plan preview --workspace <name> --file <path|-> [--format text|json]` 和 `adp plan apply --workspace <name> --file <path|-> [--format text|json]` 提供本地 planning intake，用于结构化 YAML/JSON phase 和 task 输入。`adp plan doctor [--workspace <name>] [--format text|json]` 会在不修复、不修改 ledger 的前提下检查本地 planning ledger。Preview 和 doctor 保持只读。Apply 会在校验通过后显式写入 `$ADP_HOME/workspaces/<workspace>/planning`。JSON 输出不能成为第二份 planning store；ADP 不提供 Web UI、dashboard、SaaS tracker、cloud sync、hosted orchestration、hosted tracker sync、automatic Git、automatic claim/done/phase acceptance、provider-native resume、project-root report/planning export，或自由文本自然语言拆任务。

仓库包含 `examples/basic-workspace`，作为可复制的本地 workspace 配置示例，内含 Codex 和 Claude profile、base prompt、shared memory 与 MCP 设置。实际运行前需要替换其中的 `project.root`。它展示了 ADP 如何在保持 terminal-first 的前提下，把 Agent 配置保存在真实项目目录之外。

## 开发

交付前运行聚合验证 gate：

```bash
scripts/check-all.sh
```

聚合 gate 覆盖确定性 runtime smoke、广覆盖 runtime audit smoke、聚焦 runtime context smoke、release readiness smoke、release rehearsal smoke、release artifact smoke、release operator drill smoke、install onboarding smoke、示例 workspace smoke、task manager smoke、plan intake smoke、Go test 和 vet、文件行数限制、双语文档配对和命令引用同步，以及 diff 空白检查。CI 使用同一个 `scripts/check-all.sh` gate，确保本地和自动化 release evidence 对齐。针对示例 workspace 的独立验证可运行 `scripts/example-workspace-smoke.sh`。

项目代码文件必须控制在 700 行以内。超过前按职责拆分。详见 [docs/engineering-standards.zh-CN.md](docs/engineering-standards.zh-CN.md)。

文档默认语言为英文，并必须提供 `*.zh-CN.md` 简体中文对应文件。

Runtime smoke 验收说明见 [docs/runtime-acceptance.zh-CN.md](docs/runtime-acceptance.zh-CN.md)。

Agent 在 ADP runtime overlay 中启动时可见的上下文，见 [docs/runtime-context-audit.zh-CN.md](docs/runtime-context-audit.zh-CN.md)。

任务管理和 P3 phase gate 规划见 [docs/task-management.zh-CN.md](docs/task-management.zh-CN.md)。

ADP 自身开发也 dogfood 同一套本地 task 和 phase ledger。在本仓库工作时，每个阶段都应登记到 `adp` workspace，先验证阶段、记录验收、提交、推送、记录 commit 和 push evidence，然后才能启动下一阶段。planning ledger 保存在 `$ADP_HOME` 下；仓库文档只总结已验收行为，不是执行状态存储。

Session resume planning 和 cross-tool `resume-plan` 命令见 [docs/session-restore.zh-CN.md](docs/session-restore.zh-CN.md)。

真实 Agent 兼容边界见 [docs/real-agent-compatibility.zh-CN.md](docs/real-agent-compatibility.zh-CN.md)，发布就绪检查见 [docs/release-checklist.zh-CN.md](docs/release-checklist.zh-CN.md)，early preview 打包说明见 [docs/release-packaging.zh-CN.md](docs/release-packaging.zh-CN.md)。

Agent 执行标准，包括多 Agent 协作规则与项目约束，见 [AGENTS.zh-CN.md](AGENTS.zh-CN.md)。

贡献、安全和许可证策略入口见 [CONTRIBUTING.zh-CN.md](CONTRIBUTING.zh-CN.md)、[SECURITY.zh-CN.md](SECURITY.zh-CN.md) 和 [docs/license-policy.zh-CN.md](docs/license-policy.zh-CN.md)。

## 许可证

ADP 在 [PolyForm Noncommercial License 1.0.0](LICENSE) 下以 source-available 形式提供给非商业学习、研究、评估和开放协作使用。

非商业再分发、fork、公开引用和 release package 必须保留许可证文本、必要声明，以及对 ADP 和版权持有人的署名。商业使用需要单独付费授权；公开可访问不代表已经授予商业许可。

Release package 应随 binary 和公开文档保留 `README.md`、`README.zh-CN.md`、`LICENSE`、`COMMERCIAL.md`、`COMMERCIAL.zh-CN.md`、`CONTRIBUTING.md`、`CONTRIBUTING.zh-CN.md`、`SECURITY.md`、`SECURITY.zh-CN.md` 和 `docs/license-policy.md`。不得包含 `.envrc`、`mvp.md`、本地 `$ADP_HOME` 状态、`$ADP_RUNTIME_DIR` 内容、runtime overlay、日志、任务状态、凭据或机器特定的 shell 配置。详见 [COMMERCIAL.zh-CN.md](COMMERCIAL.zh-CN.md)。
