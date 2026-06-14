# Runtime Acceptance

简体中文：[runtime-acceptance.zh-CN.md](runtime-acceptance.zh-CN.md)

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

P17 runtime-smoke modularization keeps `scripts/runtime-smoke.sh` as the only public entry point while moving shared helpers and fake diagnostics, session, and prune slices into focused helper files under `scripts/`. Those files are implementation details: they must not execute smoke work when sourced, and release gates still run `scripts/runtime-smoke.sh --fake` through `scripts/check-all.sh`.

This split is maintenance and hardening only. It must not weaken runtime acceptance, change the fake default path, remove the fake subshell isolation, or relax the explicit real-CLI environment gates.

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
adp workspace add game-lifecycle-a <temp-lifecycle-project-root>
adp workspace rename game-lifecycle-a game-lifecycle-b
adp workspace show game-lifecycle-b
adp workspace remove game-lifecycle-b
adp version
adp --version
adp tasks add --workspace game-a --priority high --phase p1 "Bind runtime session to task"
adp env game-a --cd
adp completion --shell bash
adp completion --shell zsh
adp completion values agents
adp completion values workspaces
adp completion values profiles --workspace game-a
adp completion values statuses
adp completion values tasks --workspace game-a
adp tasks claim --workspace game-a <task-id> --owner smoke-agent --lease 1m
adp completion values owners --workspace game-a
adp tasks release --workspace game-a <task-id> --owner smoke-agent
adp run codex --workspace game-a --task <task-id> -- --probe codex-payload
adp run claude --task <task-id> -- --probe claude-payload
adp run codex --workspace game-a --task missing-task -- --probe codex-payload
adp enter game-a
adp enter game-a --keep-runtime
adp events list --workspace game-a --task <task-id> --type run_finished --limit 2
adp events list --workspace game-a --task <task-id> --type run_finished --limit 2 --format json
adp sessions list --workspace game-a --agent codex --task <task-id>
adp sessions list --workspace game-a --agent codex --task <task-id> --format json
adp completion values sessions --workspace game-a
adp sessions show <session-id>
adp sessions show <session-id> --format json
adp sessions restore-plan <session-id>
adp sessions restore-plan <session-id> --format json
adp runtime prune --older-than 0s --include-kept --dry-run
adp runtime prune --older-than 0s --include-kept --dry-run --format json
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

- `adp doctor [workspace]` reports the same workspace diagnostics as the workspace command group, works for one workspace or all registered workspaces, and the fake smoke exercises runtime parent rejection for project-root and inside-project-root values. Default text output hides info diagnostics and reports `ok - no issues` when no warning or error remains, `--verbose` shows info diagnostics such as Git topology, and `--format json` emits the complete machine-readable report for local tooling. Go tests cover the broader runtime parent guard set: filesystem root, project root, inside project root, containing project root, symlink warning, and non-directory paths.
- The fake smoke also checks warning-only agent command/profile diagnostics through both doctor entry points: reserved project-root paths, adapter default command fallback, inline command arguments, missing non-default profiles, escaping profile symlinks, and unknown enabled agent entries. These diagnostics stay local and static; they do not run real provider CLIs.
- `adp version` and `adp --version` print the CLI build identity without requiring network access or provider CLIs.
- Bash and zsh completion scripts include dynamic value endpoint calls.
- `adp completion values agents` returns registered adapter names from the local registry.
- `adp completion values workspaces` returns registered workspace names from local state.
- `adp completion values profiles --workspace <name>` returns local profile names from workspace configuration and profile files.
- `adp completion values tasks --workspace <name>` returns local task IDs from the workspace planning ledger.
- `adp completion values phases --workspace <name>` returns local phase IDs from the workspace planning ledger.
- `adp completion values owners --workspace <name>` returns current task owners from the workspace planning ledger without claiming or releasing tasks.
- `adp completion values statuses` returns the task status enum used by `adp tasks update --status`.
- `adp completion values sessions --workspace <name>` returns local session IDs from JSONL runtime events.
- `adp workspace rename` and `adp workspace remove` mutate only the ADP workspace registry under the temporary `ADP_HOME`; the lifecycle smoke keeps sentinel project files in place, compares project-root entry snapshots so no new project files appear, verifies the runtime directory entry count stays unchanged after add/rename/remove, and checks completion values do not retain stale workspace names.
- `adp enter` is exercised through a controlled shell wrapper by setting `SHELL` to a temporary executable. The wrapper proves the child shell starts in `ADP_RUNTIME_ROOT`, receives the ADP runtime environment, sees project files through runtime symlinks, and does not receive task-bound runtime variables. The smoke checks default `enter` cleanup removes its runtime, `enter --keep-runtime` leaves a kept runtime until the smoke removes it, the project root entry snapshot stays unchanged, and neither path changes event log contents.

The script also verifies session restore planning:

- `adp events list --session <session-id> --task <task-id>` exposes the task-bound `run_started` and `run_finished` events for the Codex session.
- `adp events list --format json`, `adp sessions list --format json`, `adp sessions show <session-id> --format json`, and `adp sessions restore-plan <session-id> --format json` emit parseable read-only JSON inspection output.
- `adp sessions restore-plan <session-id>` prints a read-only suggested command that includes the original agent arguments.
- Running `restore-plan` does not append event log entries, create runtime state, change task state, or write to the project root.

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

The fake smoke uses a clean temporary project root for this pollution check. In real projects that already contain provider-local directories such as `.codex/` or `.claude/`, ADP should preserve non-conflicting children in the runtime overlay while still warning only on exact ADP-generated paths such as `.codex/config.toml` or `.claude/settings.json`.

## Runtime Context Audit

[docs/runtime-context-audit.md](runtime-context-audit.md) documents the launch-time context an agent sees inside an ADP runtime overlay: generated instruction files, adapter config files, selected profile, base prompt, shared memory, MCP references, task metadata, runtime environment variables, local event/session evidence, and project-root cleanliness.

The fake runtime, broad audit, and focused runtime context smokes cover this behavior through `scripts/runtime-smoke.sh --fake`, `scripts/runtime-audit-smoke.sh`, and `scripts/runtime-context-smoke.sh`. The focused smoke stays local and deterministic, uses fake agents, and avoids network access, Git execution, hosted services, provider-native resume, and project-root report or planning exports.

## Operator Failure Drill Expectations

Release candidate rehearsal should include negative drills in throwaway `ADP_HOME`, `ADP_RUNTIME_DIR`, and project roots when operator guidance is being audited. These drills are not new product scope; they verify that existing CLI errors and diagnostics point to the documented local fixes.

Useful drill commands include:

```bash
adp workspace show missing
adp run codex -- --probe
adp doctor <workspace>
adp env <workspace> --cd
adp run --task <task-id>
adp phase commit --workspace <workspace> <phase-id> --hash <commit-hash>
adp phase push --workspace <workspace> <phase-id> --remote origin --branch main --result pushed
adp phase start --workspace <workspace> <later-phase-id>
```

Expected results:

- Missing workspace names should fail with `workspace not found: <name>`. Missing workspace selection should fail with `workspace is required; pass --workspace, set ADP_WORKSPACE, or run from inside a registered project`. Operators should use `adp workspace list`, `adp workspace add <name> <project-root>`, `--workspace`, `ADP_WORKSPACE`, or a registered project directory to select the workspace.
- Unsafe runtime parents should produce error-level diagnostics from `adp doctor <workspace>` or `adp workspace doctor <workspace>` and should stop `adp env <workspace> --cd`, `adp enter <workspace>`, and `adp run <agent> --workspace <name>` before runtime creation. `ADP_RUNTIME_DIR` must stay outside the project tree and must not contain the project tree.
- Missing non-default profiles should appear as warning-only diagnostics such as `workspace.agent.profile.missing`. Operators should add one matching file under `$ADP_HOME/workspaces/<workspace>/profiles/`, select `default`, or select an existing profile. The warning is not a default release gate failure unless the release evidence depends on that profile.
- The command typo `adp run --task <task-id>` should fail because `adp run` requires an agent name before run options. The supported shape is `adp run <agent> --workspace <name> --task <task-id> -- <agent-args>`.
- Phase ordering guards should reject `adp phase commit` before passed acceptance, `adp phase push` before commit evidence, and `adp phase start` for a later phase before the earlier phase has pushed evidence. ADP records phase evidence locally; it must not run Git automatically or infer acceptance from command output.

## Task Manager And Phase Gate Acceptance

`scripts/task-manager-smoke.sh` is the public entry point and focused acceptance path for workspace-local task, next-work, phase, and progress report runtime behavior. It uses a deterministic temporary `ADP_HOME`, temporary `ADP_RUNTIME_DIR`, and temporary project root. It must not depend on repository-local user state, global `adp` binaries, provider CLIs, network access, or files written into the real project root.

P9 task-manager smoke modularization may move shared shell helpers and the JSON report validator into helper files under `scripts/`, but those helpers are implementation details. Users and release gates still run `scripts/task-manager-smoke.sh`, and `scripts/check-all.sh` remains the aggregate gate.

The split is maintenance and hardening only. It must not weaken runtime acceptance: the smoke still proves that next-work and report generation are read-only and that no planning or report artifacts pollute the real project root.

The current smoke covers the implemented task CLI:

- `adp tasks add`
- `adp tasks list`
- `adp tasks next`
- `adp tasks take`
- `adp tasks renew`
- `adp tasks stale`
- `adp tasks show`
- `adp tasks update`
- `adp tasks claim`
- `adp tasks release`
- `adp tasks block`
- `adp tasks done`
- `adp phase add`
- `adp phase list`
- `adp phase show`
- `adp phase status`
- `adp phase start`
- `adp phase accept`
- `adp phase commit`
- `adp phase push`
- `adp plan doctor [--workspace <name>] [--format text|json]`
- `adp progress`
- `adp progress report [--workspace <name>] [--language <en|zh-CN>] [--format markdown|json]`
- Planning files under `$ADP_HOME/workspaces/<workspace>/planning`.
- Read-only Markdown and JSON progress report output, including dedicated JSON validation and recent runtime session evidence when local JSONL event/session data exists.
- Protection against project-root `planning/`, `tasks.yaml`, `phases.yaml`, `progress.jsonl`, and report export pollution.

For Phase Gate MVP behavior, this smoke should verify only CLI that actually exists. It should cover:

- Phase records can be created, listed, inspected, and advanced through their lifecycle.
- Task claim, take, renew, stale, and release behavior records one owner at a time, exposes `claim_state` plus `owner`, `claimed_at`, and `lease_expires_at` when present, preserves owner-checked release, and keeps stale inspection read-only.
- Acceptance or gate records capture command, result, timestamp, and failure evidence.
- Commit records capture the accepted phase commit hash and branch.
- Push records capture the remote, branch, and push result, while commit evidence is stored on the same phase record.
- `adp phase status --format json` emits a read-only gate snapshot with the open phase, next planned phase, next required action, and whether the next phase can start.
- `adp plan doctor --format json` emits a read-only planning ledger diagnostics snapshot for healthy and broken local ledgers, including error-level diagnostics and exit code `2` for broken ledger invariants.
- Progress reports default to English Markdown, apply `--language zh-CN` to Markdown only, and include runtime session evidence from local JSONL events when that evidence exists.
- `adp tasks next --format json` emits a read-only next-work snapshot with workspace, planning source, snapshot time, task counts, status counts, requested limit, ordered candidates, `claim_state`, owner/lease fields when candidates have them, and a singular first-candidate `next` value when eligible work exists.
- `adp progress report --format json` emits a read-only machine-readable handoff snapshot with workspace, total task count, phases, task counts, tasks including `claim_state` and owner/lease fields when present, priority-sorted next work, phase evidence, and recent runtime session evidence when local JSONL event/session data exists.
- JSON report output remains a cross-tool parsing snapshot and does not create a separate state store.
- The happy path records acceptance, commit, and push evidence before a phase is treated as pushed.
- Lifecycle guard checks reject commit before passed acceptance, reject push before commit evidence, reject skipped earlier planned or unfinished phases, and reject tasks assigned to unknown phases when a phase ledger exists.
- Next-work, stale inspection, plan doctor, and report generation do not append events, mutate task or phase state, remove locks, create planning files, create runtime directories, run agents, run Git, infer acceptance, close tasks, resume provider-native conversations, sync hosted trackers, or write Markdown or JSON report files into project roots.
- All state remains under temporary `$ADP_HOME`, with no project-root pollution.

Do not add placeholder commands, TODO assertions, Web UI checks, SaaS checks, cloud sync checks, hosted tracker checks, hosted orchestration checks, automatic Git execution, automatic task closure, provider-native resume, or project-root report export behavior to smoke scripts.

## Plan Intake Acceptance

`scripts/plan-intake-smoke.sh` is the focused acceptance path for structured local planning intake. It uses a deterministic temporary `ADP_HOME`, temporary `ADP_RUNTIME_DIR`, and temporary project root, then verifies `adp plan preview` and `adp plan apply` with structured YAML input from both files and stdin through `--file -`.

The smoke covers:

- `adp plan preview --workspace <name> --file <path>` prints planned phases and tasks without creating the planning directory.
- `adp plan preview --workspace <name> --file -` accepts piped YAML from stdin and remains read-only.
- `adp plan apply --workspace <name> --file <path> --format json` explicitly writes only `$ADP_HOME/workspaces/<workspace>/planning`.
- `adp plan apply --workspace <name> --file - --format json` accepts piped YAML from stdin and still requires explicit apply.
- JSON output remains an inspection format, not a second planning store.
- Preview after apply remains read-only.
- Invalid apply on a fresh workspace leaves no empty `planning` directory.
- Failed or duplicate apply leaves phase, task, and progress state unchanged.
- Staging failures leave no partial `phases.yaml`, `tasks.yaml`, or `progress.jsonl` state.
- Preview and apply, including stdin intake through `--file -`, do not create runtime event logs, mutate runtime directories, run Git, or write planning artifacts into the real project root.

## Real CLI Smoke

Real external agent checks are intentionally not part of the default path. They must be explicitly enabled with both a flag and an environment gate:

```bash
ADP_SMOKE_REAL_CODEX=1 scripts/runtime-smoke.sh --real-codex
ADP_SMOKE_REAL_CLAUDE=1 scripts/runtime-smoke.sh --real-claude
```

Real CLI flags are additive. `scripts/runtime-smoke.sh` still runs the deterministic fake smoke first, then runs any requested real CLI checks. `scripts/check-all.sh` remains the default aggregate gate and does not pass real CLI flags, so the standard release path stays local, deterministic, and network-free.

The real checks are conservative. They confirm that the external command exists and that a lightweight invocation completes:

- `codex --version`, falling back to `codex --help`.
- `claude --version`, falling back to `claude --help`.

The command names can be overridden:

```bash
ADP_SMOKE_REAL_CODEX=1 ADP_SMOKE_CODEX_BIN=/path/to/codex scripts/runtime-smoke.sh --real-codex
ADP_SMOKE_REAL_CLAUDE=1 ADP_SMOKE_CLAUDE_BIN=/path/to/claude scripts/runtime-smoke.sh --real-claude
```

These checks do not prove that a real interactive agent session is complete. Before a release that claims real-agent compatibility, an operator should also manually confirm that `adp run codex` and `adp run claude` can start the expected local CLI on that machine and that credentials, model selection, and external tool settings match the operator's environment.

If the only claim is command availability, the opt-in real CLI smoke is enough evidence for that narrow claim. Any claim about real-agent compatibility needs separate manual acceptance notes from the operator environment.

## Acceptance Boundary

This smoke validates ADP's runtime responsibilities:

- Isolated runtime overlay creation.
- Runtime environment variable injection.
- Runtime task binding through `adp run <agent> --task <task-id>`.
- Agent command launch from the runtime root.
- Local JSONL event logging.
- Session history aggregation from local events.
- Read-only session restore planning from non-sensitive invocation snapshots.
- Shell export rendering for parent-shell workflows.
- Shell completion rendering for bash and zsh.
- Dynamic local completion value endpoints for agents, workspaces, profiles, tasks, phases, sessions, owners, and task statuses.
- Global workspace diagnostics through `adp doctor`.
- Runtime parent safety diagnostics through workspace and global doctor commands, covering filesystem root, project-root overlap, symlink warning, and non-directory cases.
- Agent command/profile diagnostics through workspace and global doctor commands, covering adapter default fallback, inline command arguments, path-like command wrappers, missing or ambiguous profile files, profile path escapes, unknown enabled agents, and reserved project-root paths.
- Local build identity output through `adp version`.
- Broad runtime audit smoke through `scripts/runtime-audit-smoke.sh`.
- Focused runtime context smoke through `scripts/runtime-context-smoke.sh`.
- Release readiness smoke through `scripts/release-readiness-smoke.sh`.
- Release rehearsal smoke through `scripts/release-rehearsal-smoke.sh`.
- Workspace-local task manager smoke through `scripts/task-manager-smoke.sh`.
- Local plan intake preview/apply smoke through `scripts/plan-intake-smoke.sh`.
- Phase Gate ledger evidence, claim leases, release owner checks, and lifecycle ordering.
- Progress report Markdown and JSON output, including runtime session evidence derived from local JSONL events when event/session data exists, with no event-log append, runtime creation, project-root writes, report exports, or separate state store creation.
- ADP-owned runtime pruning.
- Runtime prune compatibility checks for current-version ADP manifests.
- Protection against project-root pollution.

It does not validate provider accounts, remote model availability, external network access, or interactive agent behavior. Those are outside ADP's local runtime boundary and require operator-specific manual acceptance.

## Local Release Gate

Run the runtime smoke with the standard repository checks:

```bash
scripts/check-all.sh
scripts/runtime-smoke.sh --fake
scripts/runtime-audit-smoke.sh
scripts/runtime-context-smoke.sh
scripts/release-readiness-smoke.sh
scripts/release-rehearsal-smoke.sh
scripts/release-artifact-smoke.sh
scripts/release-operator-drill-smoke.sh
scripts/install-onboarding-smoke.sh
scripts/example-workspace-smoke.sh
scripts/task-manager-smoke.sh
scripts/plan-intake-smoke.sh
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

Record the default gate evidence separately from optional real CLI evidence. Optional real CLI failures do not fail the default release gate unless that release explicitly claims real-agent evidence.

When non-interactive real-agent invocation evidence is explicitly required, use the dedicated invocation smoke with both the provider flag and matching environment gate:

```bash
ADP_REAL_INVOKE_CODEX=1 scripts/real-agent-invocation-smoke.sh --codex
ADP_REAL_INVOKE_CLAUDE=1 scripts/real-agent-invocation-smoke.sh --claude
ADP_REAL_INVOKE_CODEX=1 ADP_REAL_INVOKE_CLAUDE=1 scripts/real-agent-invocation-smoke.sh --all
```

Running `scripts/real-agent-invocation-smoke.sh` without a provider target is provider-free: it prints opt-in guidance and exits successfully without building ADP, resolving external commands, creating runtimes, contacting providers, or consuming quota. Treat that default run as local guidance validation only, not as real-agent evidence.

If a real invocation fails after ADP reaches the external CLI, triage credentials, account state, model access, quota, network access, provider availability, external CLI argument changes, and tool permissions before changing ADP launch wiring. ADP-owned failures should be limited to the local runtime boundary: workspace resolution, command path configuration, runtime overlay creation, task binding, local event/session evidence, and project-root cleanliness.
