# Runtime Acceptance

Simplified Chinese: [runtime-acceptance.zh-CN.md](runtime-acceptance.zh-CN.md)

This document defines the local runtime smoke acceptance path for ADP. The goal is to verify that the terminal-first, local-first runtime manager can build an isolated workspace overlay, launch agent commands from that overlay, record local runtime history, and clean up ADP-owned runtime directories without polluting the real project root.

## Smoke Script

Run the deterministic fake-agent smoke from the repository root:

```bash
scripts/runtime-smoke.sh --fake
```

`--fake` is also the default when no mode is provided:

```bash
scripts/runtime-smoke.sh
```

The script builds the current repository's `cmd/adp` binary into a temporary directory. It does not use a globally installed `adp` binary.

The script creates and removes temporary paths for:

- `ADP_HOME`.
- `ADP_RUNTIME_DIR`.
- A project root.
- A fake agent binary directory added to `PATH`.

The fake path requires only Go and a POSIX shell environment. It does not require real Codex or Claude CLIs.

## Fake Acceptance Coverage

The fake smoke executes the current CLI runtime path end to end:

```bash
adp init
adp workspace add game-a <temp-project-root>
adp workspace list
adp workspace show game-a
adp workspace doctor game-a
adp workspace doctor
adp tasks add --workspace game-a --priority high --phase p1 "Bind runtime session to task"
adp env game-a --cd
adp completion --shell bash
adp completion --shell zsh
adp run codex --workspace game-a --task <task-id> -- --probe codex-payload
adp run claude --task <task-id> -- --probe claude-payload
adp run codex --workspace game-a --task missing-task -- --probe codex-payload
adp events list --workspace game-a --task <task-id> --type run_finished --limit 2
adp sessions list --workspace game-a --agent codex --task <task-id>
adp sessions show <session-id>
adp runtime prune --older-than 0s --include-kept --dry-run
adp runtime prune --older-than 0s --include-kept
```

The fake Codex and Claude commands assert that:

- The process working directory is the ADP runtime root.
- `ADP_WORKSPACE` is set to the registered workspace.
- `ADP_SESSION_ID` is present.
- `ADP_RUNTIME_ROOT` is present and matches the process working directory.
- `ADP_TASK_ID` and task metadata are present for task-bound runs.
- `.adp-runtime.yaml` exists in the runtime root.
- `.adp-runtime.yaml` records the bound task ID.
- Agent-specific generated files exist:
  - Codex: `AGENTS.md` and `.codex/config.toml`.
  - Claude: `CLAUDE.md` and `.claude/settings.json`.
- Generated instructions contain the current task context.
- Real project files are visible through symlinks from the runtime root.
- Arguments after `--` reach the agent process.

The script also checks that a missing task ID fails before the fake agent command is launched.

The script also asserts that the real project root is not polluted with ADP runtime artifacts:

- `AGENTS.md`.
- `CLAUDE.md`.
- `.codex/`.
- `.claude/`.
- `planning/`.
- `tasks.yaml`.
- `progress.jsonl`.

## Real CLI Smoke

Real external agent checks are intentionally not part of the default path. They must be explicitly enabled with both a flag and an environment gate:

```bash
ADP_SMOKE_REAL_CODEX=1 scripts/runtime-smoke.sh --real-codex
ADP_SMOKE_REAL_CLAUDE=1 scripts/runtime-smoke.sh --real-claude
```

The real checks are conservative. They confirm that the external command exists and that a lightweight invocation completes:

- `codex --version`, falling back to `codex --help`.
- `claude --version`, falling back to `claude --help`.

The command names can be overridden:

```bash
ADP_SMOKE_REAL_CODEX=1 ADP_SMOKE_CODEX_BIN=/path/to/codex scripts/runtime-smoke.sh --real-codex
ADP_SMOKE_REAL_CLAUDE=1 ADP_SMOKE_CLAUDE_BIN=/path/to/claude scripts/runtime-smoke.sh --real-claude
```

These checks do not prove that a real interactive agent session is complete. Before a release that claims real-agent compatibility, an operator should also manually confirm that `adp run codex` and `adp run claude` can start the expected local CLI on that machine and that credentials, model selection, and external tool settings match the operator's environment.

## Acceptance Boundary

This smoke validates ADP's runtime responsibilities:

- Isolated runtime overlay creation.
- Runtime environment variable injection.
- Runtime task binding through `adp run --task <task-id>`.
- Agent command launch from the runtime root.
- Local JSONL event logging.
- Session history aggregation from local events.
- Shell export rendering for parent-shell workflows.
- Shell completion rendering for bash and zsh.
- ADP-owned runtime pruning.
- Protection against project-root pollution.

It does not validate provider accounts, remote model availability, external network access, or interactive agent behavior. Those are outside ADP's local runtime boundary and require operator-specific manual acceptance.

## Local Release Gate

Run the runtime smoke with the standard repository checks:

```bash
scripts/runtime-smoke.sh --fake
go test -count=1 ./...
go vet ./...
scripts/check-file-lines.sh
scripts/check-docs-bilingual.sh
git diff --check
```

Real CLI checks are optional release evidence and should be recorded separately when they are run:

```bash
ADP_SMOKE_REAL_CODEX=1 scripts/runtime-smoke.sh --real-codex
ADP_SMOKE_REAL_CLAUDE=1 scripts/runtime-smoke.sh --real-claude
```
