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
- `.adp-runtime.yaml` records manifest version `1`, `generated_by: adp`, the runtime root path, and the bound task ID.
- Agent-specific generated files exist:
  - Codex: `AGENTS.md` and `.codex/config.toml`.
  - Claude: `CLAUDE.md` and `.claude/settings.json`.
- Generated instructions contain the current task context.
- Real project files are visible through symlinks from the runtime root.
- Arguments after `--` reach the agent process.

The script also checks the local CLI hardening surface:

- `adp doctor [workspace]` reports the same workspace diagnostics as the workspace command group and works for one workspace or all registered workspaces.
- `adp version` and `adp --version` print the CLI build identity without requiring network access or provider CLIs.
- Bash and zsh completion scripts include dynamic value endpoint calls.
- `adp completion values workspaces` returns registered workspace names from local state.
- `adp completion values profiles --workspace <name>` returns local profile names from workspace configuration and profile files.

The script also checks that a missing task ID fails before the fake agent command is launched.

The script also asserts that the real project root is not polluted with ADP runtime artifacts:

- `AGENTS.md`.
- `CLAUDE.md`.
- `.codex/`.
- `.claude/`.
- `planning/`.
- `tasks.yaml`.
- `phases.yaml`.
- `progress.jsonl`.

## Task Manager And Phase Gate Acceptance

`scripts/task-manager-smoke.sh` is the focused acceptance path for task-management behavior. It uses a deterministic temporary `ADP_HOME`, temporary `ADP_RUNTIME_DIR`, and temporary project root. It must not depend on repository-local user state, global `adp` binaries, provider CLIs, network access, or files written into the real project root.

The current smoke covers the implemented task CLI:

- `adp tasks add`
- `adp tasks list`
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
- Planning files under `$ADP_HOME/workspaces/<workspace>/planning`.
- Protection against project-root `planning/`, `tasks.yaml`, `phases.yaml`, and `progress.jsonl` pollution.

For Phase Gate MVP behavior, this smoke should verify only CLI that actually exists. It should cover:

- Phase records can be created, listed, inspected, and advanced through their lifecycle.
- Task claim and release commands record one owner at a time, including `--lease` and owner-checked release.
- Acceptance or gate records capture command, result, timestamp, and failure evidence.
- Commit records capture the accepted phase commit hash and branch.
- Push records capture the remote, branch, and push result, while commit evidence is stored on the same phase record.
- The happy path records acceptance, commit, and push evidence before a phase is treated as pushed.
- Lifecycle guard checks reject commit before passed acceptance, reject push before commit evidence, and reject tasks assigned to unknown phases when a phase ledger exists.
- All state remains under temporary `$ADP_HOME`, with no project-root pollution.

Do not add placeholder commands, TODO assertions, Web UI checks, SaaS checks, cloud sync checks, or hosted orchestration checks to smoke scripts.

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
- Dynamic local completion value endpoints for workspaces and profiles.
- Global workspace diagnostics through `adp doctor`.
- Local build identity output through `adp version`.
- Workspace-local task manager smoke through `scripts/task-manager-smoke.sh`.
- Phase Gate ledger evidence, claim leases, release owner checks, and lifecycle ordering.
- ADP-owned runtime pruning.
- Runtime prune compatibility checks for current-version ADP manifests.
- Protection against project-root pollution.

It does not validate provider accounts, remote model availability, external network access, or interactive agent behavior. Those are outside ADP's local runtime boundary and require operator-specific manual acceptance.

## Local Release Gate

Run the runtime smoke with the standard repository checks:

```bash
scripts/check-all.sh
scripts/runtime-smoke.sh --fake
go test -count=1 ./...
go vet ./...
scripts/check-file-lines.sh
scripts/check-docs-bilingual.sh
git diff --check
```

`scripts/check-all.sh` is the aggregate gate used by local handoff and CI. The expanded command list above is useful when a failure needs to be isolated.

Real CLI checks are optional release evidence and should be recorded separately when they are run:

```bash
ADP_SMOKE_REAL_CODEX=1 scripts/runtime-smoke.sh --real-codex
ADP_SMOKE_REAL_CLAUDE=1 scripts/runtime-smoke.sh --real-claude
```
