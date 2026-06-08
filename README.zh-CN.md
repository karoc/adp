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
- `adp workspace remove <name>`
- `adp workspace rename <old-name> <new-name>`
- `adp env <workspace> [--cd]`
- `adp run codex --workspace <name>`
- `adp run claude --workspace <name>`
- `adp enter <workspace>`
- `$ADP_HOME` 下的本地 workspace registry
- `$ADP_RUNTIME_DIR` 下的 symlink runtime overlay
- Codex 和 Claude adapter layer
- JSONL event log
- process runner 和 workspace shell

## 快速开始

```bash
go run ./cmd/adp init
go run ./cmd/adp workspace add game-a /srv/game-a
go run ./cmd/adp workspace list
go run ./cmd/adp workspace show game-a
go run ./cmd/adp env game-a --cd
go run ./cmd/adp run codex --workspace game-a
cd /srv/game-a && go run /path/to/adp/cmd/adp run claude
go run ./cmd/adp run claude --workspace game-a
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

## 开发

交付前运行标准检查：

```bash
go test -count=1 ./...
go vet ./...
scripts/check-file-lines.sh
scripts/check-docs-bilingual.sh
git diff --check
```

项目代码文件必须控制在 700 行以内。超过前按职责拆分。详见 [docs/engineering-standards.zh-CN.md](docs/engineering-standards.zh-CN.md)。

文档默认语言为英文，并必须提供 `*.zh-CN.md` 简体中文对应文件。

## 许可证

ADP 在 [PolyForm Noncommercial License 1.0.0](LICENSE) 下提供公开非商业使用。

商业使用需要单独付费授权。详见 [COMMERCIAL.zh-CN.md](COMMERCIAL.zh-CN.md)。
