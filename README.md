# ADP

Simplified Chinese: [README.zh-CN.md](README.zh-CN.md)

ADP, short for Agent Development Platform, is an Agent Runtime Environment and Agent Workspace Manager for terminal-first AI agent workflows.

ADP keeps AI agent configuration outside the project directory, then builds a temporary runtime overlay when an agent starts. The agent sees generated files such as `AGENTS.md`, `CLAUDE.md`, `.codex/`, and `.claude/`, while the real project directory stays clean.

## Current MVP

Implemented Phase 1 foundations:

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
- local workspace registry under `$ADP_HOME`
- symlink-based runtime overlay under `$ADP_RUNTIME_DIR`
- Codex and Claude adapter layer
- JSONL event log
- session history views derived from local events
- workspace diagnostics for local configuration issues
- `examples/basic-workspace` sample workspace configuration
- process runner and workspace shell

## Quick Start

For installation and bootstrap details, see [docs/install.md](docs/install.md).

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

Useful environment variables:

- `ADP_HOME`: ADP home directory. Defaults to `~/.adp`.
- `ADP_RUNTIME_DIR`: parent directory for temporary runtime overlays. Defaults to the system temp directory under `adp-runtime`.
- `ADP_WORKSPACE`: default workspace for commands that accept a workspace.

When `--workspace` and `ADP_WORKSPACE` are omitted, `adp run` tries to match the current directory to a registered project root.

## Runtime Model

`adp run` builds a temporary runtime root that looks like the project root:

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

Agent-specific files are generated from the ADP workspace config. Real project files are linked into the runtime root. ADP-generated paths take priority inside the runtime view, and the original project directory is not modified.

`adp env <workspace> --cd` prints POSIX shell exports for a kept runtime overlay. This is intended for shell-hook workflows and leaves the runtime directory in place so the calling shell can enter it.

`adp shell-hook --shell bash` prints a shell function that calls `adp env <workspace> --cd` and evaluates the result in the parent shell. `sh`, `bash`, and `zsh` are supported.

`adp completion [--shell <bash|zsh>] [--command <name>]` prints deterministic shell completion for the current CLI surface. It defaults to bash when `--shell` is omitted. The optional command name lets packaged binaries or aliases render completion for a command name other than `adp`.

`adp events list` reads `$ADP_HOME/logs/events.jsonl` and prints recent runtime events with optional workspace, session, type, and limit filters.

`adp sessions list [--workspace <name>] [--agent <agent>] [--limit <n>]` groups local event log entries by session so terminal users can inspect recent agent runs without leaving the CLI.

`adp sessions show <session-id>` prints the ordered events for one recorded session, including start, finish, workspace, agent, runtime path, exit code, and duration data when those fields are available.

`adp workspace doctor [name]` validates workspace configuration, project root reachability, referenced prompt, memory, MCP, and profile files, and agent command settings. Without a name it checks all registered workspaces and returns a non-zero exit code when error-level diagnostics are found.

`adp runtime prune` removes stale ADP-owned runtime directories under `$ADP_RUNTIME_DIR`. A directory is considered ADP-owned only when it contains `.adp-runtime.yaml` with `generated_by: adp`. Kept runtimes are preserved unless `--include-kept` is passed, and `--dry-run` reports candidates without deleting them.

The repository includes `examples/basic-workspace` as a copyable local workspace configuration with Codex and Claude profiles, base prompts, shared memory, and MCP settings. Replace its `project.root` before running it against a local project. It is intended as a terminal-first reference for how ADP keeps agent configuration outside the real project tree.

## Development

Use the aggregate validation gate before handoff:

```bash
scripts/check-all.sh
```

The aggregate gate covers deterministic runtime smoke, example workspace smoke, Go test and vet, file length limits, bilingual documentation pairing, and whitespace diff checks. For targeted example validation, run `scripts/example-workspace-smoke.sh`.

Project code files must stay at or below 700 physical lines. Split files by responsibility before they exceed the limit. See [docs/engineering-standards.md](docs/engineering-standards.md).

Documentation defaults to English and must include Simplified Chinese counterparts using `*.zh-CN.md`.

Runtime smoke acceptance is documented in [docs/runtime-acceptance.md](docs/runtime-acceptance.md).

Real agent compatibility boundaries are documented in [docs/real-agent-compatibility.md](docs/real-agent-compatibility.md), and release readiness is tracked in [docs/release-checklist.md](docs/release-checklist.md).

Agent execution standards, including multi-agent coordination rules and project constraints, are documented in [AGENTS.md](AGENTS.md).

## License

ADP is publicly available for noncommercial use under the [PolyForm Noncommercial License 1.0.0](LICENSE).

Commercial use requires a separate paid license. See [COMMERCIAL.md](COMMERCIAL.md).
