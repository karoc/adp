# ADP

ADP, short for Agent Development Platform, is an Agent Runtime Environment and Agent Workspace Manager for terminal-first AI agent workflows.

ADP keeps AI agent configuration outside the project directory, then builds a temporary runtime overlay when an agent starts. The agent sees generated files such as `AGENTS.md`, `CLAUDE.md`, `.codex/`, and `.claude/`, while the real project directory stays clean.

## Current MVP

Implemented Phase 1 foundations:

- `adp init`
- `adp workspace add <name> <project-root>`
- `adp run codex --workspace <name>`
- `adp run claude --workspace <name>`
- `adp enter <workspace>`
- local workspace registry under `$ADP_HOME`
- symlink-based runtime overlay under `$ADP_RUNTIME_DIR`
- Codex and Claude adapter layer
- JSONL event log
- process runner and workspace shell

## Quick Start

```bash
go run ./cmd/adp init
go run ./cmd/adp workspace add game-a /srv/game-a
go run ./cmd/adp run codex --workspace game-a
go run ./cmd/adp run claude --workspace game-a
go run ./cmd/adp enter game-a
```

Useful environment variables:

- `ADP_HOME`: ADP home directory. Defaults to `~/.adp`.
- `ADP_RUNTIME_DIR`: parent directory for temporary runtime overlays. Defaults to the system temp directory under `adp-runtime`.
- `ADP_WORKSPACE`: default workspace for commands that accept a workspace.

## Runtime Model

`adp run` builds a temporary runtime root that looks like the project root:

```txt
/tmp/adp-runtime/game-a-<session>/
├── AGENTS.md
├── CLAUDE.md
├── .codex/
├── .claude/
├── go.mod -> /srv/game-a/go.mod
└── internal -> /srv/game-a/internal
```

Agent-specific files are generated from the ADP workspace config. Real project files are linked into the runtime root. ADP-generated paths take priority inside the runtime view, and the original project directory is not modified.

## Development

Run the standard checks before handoff:

```bash
go test ./...
scripts/check-file-lines.sh
git diff --check
```

Project code files must stay at or below 700 physical lines. Split files by responsibility before they exceed the limit. See [docs/engineering-standards.md](docs/engineering-standards.md).

## License

ADP is publicly available for noncommercial use under the [PolyForm Noncommercial License 1.0.0](LICENSE).

Commercial use requires a separate paid license. See [COMMERCIAL.md](COMMERCIAL.md).
