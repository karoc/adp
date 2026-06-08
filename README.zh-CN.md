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
- `adp workspace remove <name>`
- `adp workspace rename <old-name> <new-name>`
- `adp env <workspace> [--cd]`
- `adp shell-hook [--shell <sh|bash|zsh>]`
- `adp completion [--shell <bash|zsh>] [--command <name>]`
- `adp events list [--workspace <name>]`
- `adp sessions list [--workspace <name>] [--agent <agent>] [--limit <n>]`
- `adp sessions show <session-id>`
- `adp runtime prune [--older-than <duration>] [--dry-run]`
- `adp run codex --workspace <name>`
- `adp run claude --workspace <name>`
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

```bash
go run ./cmd/adp init
go run ./cmd/adp workspace add game-a /srv/game-a
go run ./cmd/adp workspace list
go run ./cmd/adp workspace show game-a
go run ./cmd/adp workspace doctor game-a
go run ./cmd/adp env game-a --cd
go run ./cmd/adp shell-hook --shell bash
go run ./cmd/adp completion --shell bash
go run ./cmd/adp run codex --workspace game-a
cd /srv/game-a && go run /path/to/adp/cmd/adp run claude
go run ./cmd/adp run claude --workspace game-a
go run ./cmd/adp events list --workspace game-a
go run ./cmd/adp sessions list --workspace game-a --agent codex
go run ./cmd/adp sessions show <session-id>
go run ./cmd/adp runtime prune --older-than 24h --dry-run
go run ./cmd/adp enter game-a
```

常用环境变量：

- `ADP_HOME`：ADP home 目录，默认 `~/.adp`。
- `ADP_RUNTIME_DIR`：临时 runtime overlay 的父目录，默认是系统临时目录下的 `adp-runtime`。
- `ADP_WORKSPACE`：可作为命令默认 workspace。

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

`adp completion [--shell <bash|zsh>] [--command <name>]` 会为当前 CLI 命令面输出稳定的 shell completion。省略 `--shell` 时默认输出 bash completion。可选 command name 用于给打包后的二进制名或别名生成非 `adp` 名称的 completion。

`adp events list` 会读取 `$ADP_HOME/logs/events.jsonl`，按 workspace、session、事件类型和数量限制输出最近的 runtime 事件。

`adp sessions list [--workspace <name>] [--agent <agent>] [--limit <n>]` 会按 session 聚合本地 event log，方便在终端中查看最近的 Agent 运行记录。

`adp sessions show <session-id>` 会输出某个已记录 session 的有序事件；如果事件包含 workspace、agent、runtime path、exit code 和 duration 等字段，也会一起展示。

`adp workspace doctor [name]` 会检查 workspace 配置、project root 是否可访问、prompt、memory、MCP、profile 文件引用以及 agent command 设置。不传 name 时检查所有已注册 workspace；发现 error 级 diagnostics 时返回非零退出码。

`adp runtime prune` 会清理 `$ADP_RUNTIME_DIR` 下过期的 ADP runtime 目录。只有包含 `.adp-runtime.yaml` 且 `generated_by: adp` 的目录才会被视为 ADP 创建；默认保留 `keep: true` 的 runtime，除非传入 `--include-kept`；`--dry-run` 只报告候选项，不删除。

仓库包含 `examples/basic-workspace`，作为可复制的本地 workspace 配置示例，内含 Codex 和 Claude profile、base prompt、shared memory 与 MCP 设置。实际运行前需要替换其中的 `project.root`。它展示了 ADP 如何在保持 terminal-first 的前提下，把 Agent 配置保存在真实项目目录之外。

## 开发

交付前运行标准检查：

```bash
scripts/runtime-smoke.sh --fake
go test -count=1 ./...
go vet ./...
scripts/check-file-lines.sh
scripts/check-docs-bilingual.sh
git diff --check
```

项目代码文件必须控制在 700 行以内。超过前按职责拆分。详见 [docs/engineering-standards.zh-CN.md](docs/engineering-standards.zh-CN.md)。

文档默认语言为英文，并必须提供 `*.zh-CN.md` 简体中文对应文件。

Runtime smoke 验收说明见 [docs/runtime-acceptance.zh-CN.md](docs/runtime-acceptance.zh-CN.md)。

## 许可证

ADP 在 [PolyForm Noncommercial License 1.0.0](LICENSE) 下提供公开非商业使用。

商业使用需要单独付费授权。详见 [COMMERCIAL.zh-CN.md](COMMERCIAL.zh-CN.md)。
