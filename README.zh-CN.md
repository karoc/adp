# ADP

English: [README.md](README.md)

ADP 是 Agent Development Platform 的缩写。它是面向 terminal-first AI Agent 工作流的 Agent Runtime Environment 和 Agent Workspace Manager。

ADP 把 AI Agent 配置保存在项目目录之外，并在 Agent 启动时构建临时 runtime overlay。Agent 可以看到 `AGENTS.md`、`CLAUDE.md`、`.codex/`、`.claude/` 等生成文件，但真实项目目录保持干净。

## 当前 MVP

已实现 Phase 1 基础能力：

- `adp init`
- `adp workspace add <name> <project-root>`
- `adp workspace list`
- `adp workspace show <name>`
- `adp workspace doctor [name]`
- `adp doctor [workspace]`
- `adp workspace remove <name>`
- `adp workspace rename <old-name> <new-name>`
- `adp env <workspace> [--cd]`
- `adp shell-hook [--shell <sh|bash|zsh>]`
- `adp completion [--shell <bash|zsh>] [--command <name>]`
- `adp completion values <workspaces|profiles> [--workspace <name>]`
- `adp version`
- `adp events list [--workspace <name>] [--task <task-id>]`
- `adp sessions list [--workspace <name>] [--agent <agent>] [--task <task-id>] [--limit <n>]`
- `adp sessions show <session-id>`
- `adp sessions restore-plan <session-id>`
- `adp runtime prune [--older-than <duration>] [--include-kept] [--dry-run]`
- `adp tasks add/list/show/update/claim/release/done/block`
- `adp phase add/list/show/start/accept/commit/push`
- `adp progress [--workspace <name>]`
- `adp progress report [--workspace <name>] [--language <en|zh-CN>]`
- `adp run codex --workspace <name> [--task <task-id>]`
- `adp run claude --workspace <name> [--task <task-id>]`
- `adp enter <workspace>`
- `$ADP_HOME` 下的本地 workspace registry
- `$ADP_RUNTIME_DIR` 下的 symlink runtime overlay
- Codex 和 Claude adapter layer
- JSONL event log
- 基于本地事件聚合的 session history 视图
- 检查本地 workspace 配置问题的 diagnostics
- `examples/basic-workspace` 示例 workspace 配置
- process runner 和 workspace shell

## 快速开始

安装与 bootstrap 细节见 [docs/install.zh-CN.md](docs/install.zh-CN.md)。

```bash
go run ./cmd/adp init
go run ./cmd/adp workspace add game-a /srv/game-a
go run ./cmd/adp workspace list
go run ./cmd/adp workspace show game-a
go run ./cmd/adp workspace doctor game-a
go run ./cmd/adp doctor game-a
go run ./cmd/adp env game-a --cd
go run ./cmd/adp shell-hook --shell bash
go run ./cmd/adp completion --shell bash
go run ./cmd/adp completion values workspaces
go run ./cmd/adp completion values profiles --workspace game-a
go run ./cmd/adp version
TASK_ID=$(go run ./cmd/adp tasks add --workspace game-a --priority high --phase phase-1 "Bind runtime session to task" | sed -n 's/^task \(task-[^ ]*\) added$/\1/p')
go run ./cmd/adp run codex --workspace game-a --task "$TASK_ID"
cd /srv/game-a && go run /path/to/adp/cmd/adp run claude
go run ./cmd/adp run claude --workspace game-a
go run ./cmd/adp events list --workspace game-a --task "$TASK_ID"
go run ./cmd/adp tasks list --workspace game-a --format json
go run ./cmd/adp progress --workspace game-a --format json
go run ./cmd/adp progress report --workspace game-a
go run ./cmd/adp sessions list --workspace game-a --agent codex --task "$TASK_ID"
go run ./cmd/adp sessions show <session-id>
go run ./cmd/adp sessions restore-plan <session-id>
go run ./cmd/adp runtime prune --older-than 24h --dry-run
go run ./cmd/adp enter game-a
```

常用环境变量：

- `ADP_HOME`：ADP home 目录，默认 `~/.adp`。
- `ADP_RUNTIME_DIR`：临时 runtime overlay 的父目录，默认是系统临时目录下的 `adp-runtime`。不要把它指向文件系统根目录、project root、project root 内部目录，或包含 project root 的父目录。优先使用直接的本地目录；symlink runtime parent 会被 doctor 命令作为 warning 报告。
- `ADP_WORKSPACE`：可作为命令默认 workspace。
- `ADP_TASK_ID`、`ADP_TASK_TITLE`、`ADP_TASK_STATUS`、`ADP_TASK_PRIORITY` 和 `ADP_TASK_PHASE`：在通过 `adp run --task <task-id>` 启动的 runtime 内可用。

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

Agent 专属文件由 ADP workspace config 生成。真实项目文件通过 symlink 进入 runtime root。ADP 生成路径在 runtime 视图中优先，原始项目目录不会被修改。

`adp env <workspace> --cd` 会输出 POSIX shell exports，并保留 runtime overlay，适合 shell-hook 工作流让调用方 shell 进入 runtime。

`adp shell-hook --shell bash` 会输出一个 shell 函数，用 `adp env <workspace> --cd` 构建 runtime，并在父 shell 中执行返回的 exports 和 `cd`。当前支持 `sh`、`bash`、`zsh`。

`adp completion [--shell <bash|zsh>] [--command <name>]` 会为当前 CLI 命令面输出稳定的 shell completion。省略 `--shell` 时默认输出 bash completion。可选 command name 用于给打包后的二进制名或别名生成非 `adp` 名称的 completion。生成的 completion 脚本会调用只读的本地值端点 `adp completion values workspaces` 和 `adp completion values profiles [--workspace <name>]`，用于补全已注册 workspace 名称和 workspace profile 名称。

`adp events list` 会读取 `$ADP_HOME/logs/events.jsonl`，按 workspace、session、task、事件类型和数量限制输出最近的 runtime 事件。

`adp sessions list [--workspace <name>] [--agent <agent>] [--task <task-id>] [--limit <n>]` 会按 session 聚合本地 event log，方便在终端中查看最近的 Agent 运行记录。

`adp sessions show <session-id>` 会输出某个已记录 session 的有序事件；如果事件包含 workspace、agent、task ID、runtime path、exit code 和 duration 等字段，也会一起展示。

`adp sessions restore-plan <session-id>` 会读取一个已记录 session，并在非敏感 invocation 数据足够时打印只读的建议 `adp run ...` 命令。它不会执行命令、启动 Agent、创建 runtime、追加 events、修改 task 状态、写入项目根目录或恢复 provider 原生会话。详见 [docs/session-restore.zh-CN.md](docs/session-restore.zh-CN.md)。

`adp workspace doctor [name]` 会检查 workspace 配置、project root 可访问性、runtime parent 安全性、prompt、memory、MCP、profile 文件引用、agent command 设置，以及 project root 中的保留路径。它会把 adapter default command fallback、写在 command 字段里的 inline arguments、缺失或不可执行的路径型 command wrapper，以及缺失、重复或逃逸到 workspace 外部的非 default profile 报告为本地 diagnostics。不传 name 时检查所有已注册 workspace；发现 error 级 diagnostics 时返回非零退出码。

`adp doctor [workspace]` 是同一组本地 workspace 检查的全局 diagnostics 入口。它适合需要在终端中直接运行 diagnostics、但不想先进入 `workspace` 命令组的工作流。

`adp version` 和 `adp --version` 会输出 CLI build identity。打包 binary 可以在构建时注入 version、commit 和 build-date；开发构建默认回退为 `dev`。

`adp runtime prune` 会清理 `$ADP_RUNTIME_DIR` 下过期的 ADP runtime 目录。只有包含当前版本且结构自洽的 `.adp-runtime.yaml`，其中 `generated_by: adp` 且 `runtime_root` 与目录一致的 runtime 才会进入 prune 候选；默认保留 `keep: true` 的 runtime，除非传入 `--include-kept`；`--dry-run` 只报告候选项，不删除。

`adp tasks` 和 `adp progress` 管理 `$ADP_HOME/workspaces/<workspace>/planning` 下的 workspace-scoped 规划和执行进度。只读 task、phase 和 progress 视图支持 `--format json`，方便本地工具和子 Agent 获取机器可读 planning snapshot；权威状态仍保存在 `$ADP_HOME` 下，task 或 phase 变更仍然必须通过显式命令完成。`adp run --task <task-id>` 会把这份本地任务状态绑定到 runtime 环境变量、生成的 adapter instructions、events 和 sessions，同时不会把 planning 文件写入真实项目根目录。详见 [docs/task-management.zh-CN.md](docs/task-management.zh-CN.md)。

`adp progress report [--workspace <name>] [--language <en|zh-CN>]` 会向 stdout 打印本地 Markdown 规划/执行报告。报告默认语言是英文；简体中文必须显式传入 `--language zh-CN`。该 report 命令是只读的，不会修改 task 状态、phase 状态、Git 状态、runtime 状态、event log 或 project-root 文件。

P3 提供项目规划和执行进度管理的本地 phase ledger。它会在 `$ADP_HOME` 下记录任务归属、可选 claim lease、验收记录、commit 记录、push 记录和明确的阶段门禁纪律。该能力仍然保持 terminal-first、local-first；它不是 Web dashboard、SaaS tracker、cloud sync layer 或 hosted orchestration service。

仓库包含 `examples/basic-workspace`，作为可复制的本地 workspace 配置示例，内含 Codex 和 Claude profile、base prompt、shared memory 与 MCP 设置。实际运行前需要替换其中的 `project.root`。它展示了 ADP 如何在保持 terminal-first 的前提下，把 Agent 配置保存在真实项目目录之外。

## 开发

交付前运行聚合验证 gate：

```bash
scripts/check-all.sh
```

聚合 gate 覆盖确定性 runtime smoke、示例 workspace smoke、task manager smoke、Go test 和 vet、文件行数限制、双语文档配对以及 diff 空白检查。CI 使用同一个 `scripts/check-all.sh` gate，确保本地和自动化 release evidence 对齐。针对示例 workspace 的独立验证可运行 `scripts/example-workspace-smoke.sh`。

项目代码文件必须控制在 700 行以内。超过前按职责拆分。详见 [docs/engineering-standards.zh-CN.md](docs/engineering-standards.zh-CN.md)。

文档默认语言为英文，并必须提供 `*.zh-CN.md` 简体中文对应文件。

Runtime smoke 验收说明见 [docs/runtime-acceptance.zh-CN.md](docs/runtime-acceptance.zh-CN.md)。

任务管理和 P3 phase gate 规划见 [docs/task-management.zh-CN.md](docs/task-management.zh-CN.md)。

Session restore planning 见 [docs/session-restore.zh-CN.md](docs/session-restore.zh-CN.md)。

真实 Agent 兼容边界见 [docs/real-agent-compatibility.zh-CN.md](docs/real-agent-compatibility.zh-CN.md)，发布就绪检查见 [docs/release-checklist.zh-CN.md](docs/release-checklist.zh-CN.md)，early preview 打包说明见 [docs/release-packaging.zh-CN.md](docs/release-packaging.zh-CN.md)。

Agent 执行标准，包括多 Agent 协作规则与项目约束，见 [AGENTS.zh-CN.md](AGENTS.zh-CN.md)。

## 许可证

ADP 在 [PolyForm Noncommercial License 1.0.0](LICENSE) 下提供公开非商业使用。

商业使用需要单独付费授权。详见 [COMMERCIAL.zh-CN.md](COMMERCIAL.zh-CN.md)。
