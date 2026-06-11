# ADP

Simplified Chinese: [README.zh-CN.md](README.zh-CN.md)

ADP, short for Agent Development Platform, is an Agent Runtime Environment and Agent Workspace Manager for terminal-first AI agent workflows.

ADP keeps AI agent configuration outside the project directory, then builds a temporary runtime overlay when an agent starts. The agent sees generated files such as `AGENTS.md`, `CLAUDE.md`, `.codex/`, and `.claude/`, while the real project directory stays clean.

## Current MVP

Implemented Phase 1 foundations. If you are trying ADP for the first time, start with [Quick Start](#quick-start); the list below is the command reference snapshot.

- `adp init`
- `adp workspace add <name> <project-root>`
- `adp workspace list`
- `adp workspace show <name>`
- `adp workspace doctor [name] [--verbose] [--format <text|json>]`
- `adp doctor [workspace] [--verbose] [--format <text|json>]`
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
- local workspace registry under `$ADP_HOME`
- symlink-based runtime overlay under `$ADP_RUNTIME_DIR`
- Codex and Claude adapter layer
- JSONL event log
- session history views derived from local events
- workspace diagnostics for local configuration issues
- `examples/basic-workspace` sample workspace configuration
- process runner and workspace shell

`adp sessions resume-plan <session-id> [--workspace <name>] [--owner <owner>] [--lease <duration>] [--agent <agent>] [--format <text|json>]` provides read-only cross-tool ADP work-context resume guidance. It complements `restore-plan`: `restore-plan` focuses on rerun guidance for one recorded session, while `resume-plan` adds owner, lease, task, phase, and target-agent context for operator review.

Command discovery stays inside the CLI. Use `adp --help` for the root command list, `adp <command> --help` for a command group such as `adp tasks --help`, and `adp <command> <subcommand> --help` for a leaf command such as `adp tasks take --help`. If you are using `ADP_BIN`, `adp_local`, or a packaged binary name, substitute that command name in the same pattern. Leaf help may point back to parent help with `See also:`; if a build prints a friendly `try:` hint, treat it as a pointer to the same help surface, not as an automatic action or state change.

## Quick Start

For installation and bootstrap details, see [docs/install.md](docs/install.md). For a concrete new-operator walkthrough, see [docs/operator-onboarding.md](docs/operator-onboarding.md).

If this is your first trial, use [docs/operator-onboarding.md](docs/operator-onboarding.md) as the guided path. It explains what the rehearsal proves, what output to expect, which commands are read-only, and when to move from temporary state to durable local use. The command block below is the compact smoke-first reference version.

Choose a source, built-binary, or temporary-install path first. The reference path below builds a local binary and uses `ADP_BIN`; for a release artifact, set `ADP_BIN` to the installed artifact path. To run from source during development, replace `"$ADP_BIN" <command>` with `go run ./cmd/adp <command>`.

The rehearsal uses temporary ADP state, a temporary project root, and a fake `codex` command. It does not require real Codex or Claude CLIs, does not run Git, and should leave the real project root free of ADP-generated files. The flow is intentionally close to mature CLI quickstarts: install one command, initialize local state, add a workspace, inspect before mutating, atomically take work when launching an agent, then verify local evidence.

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

Expected result: the fake provider prints a runtime working directory, local inspection commands return task/session/progress evidence, and the final project-root leak check passes without output. For durable local use, set `ADP_HOME` to a persistent directory such as `~/.adp`; for real agent runs, install and authenticate the external provider CLI first, then use `adp run codex ...` or `adp run claude ...`. Use `examples/basic-workspace` when you want a copyable workspace configuration with Codex and Claude profiles, base prompts, shared memory, and MCP settings.

Useful environment variables:

- `ADP_HOME`: ADP home directory. Defaults to `~/.adp`.
- `ADP_RUNTIME_DIR`: parent directory for temporary runtime overlays. Defaults to the system temp directory under `adp-runtime`. Do not point it at the filesystem root, a project root, a directory inside a project root, or a directory that contains the project root. Prefer a direct local directory; symlink runtime parents are reported as warnings by doctor commands.
- `ADP_WORKSPACE`: default workspace for commands that accept a workspace.
- `ADP_TASK_ID`, `ADP_TASK_TITLE`, `ADP_TASK_STATUS`, `ADP_TASK_PRIORITY`, `ADP_TASK_PHASE`, `ADP_TASK_OWNER`, `ADP_TASK_CLAIMED_AT`, and `ADP_TASK_LEASE_EXPIRES_AT`: available inside runtimes launched with `adp run <agent> --task <task-id>` or `adp run <agent> --take --owner <owner>` when the selected task has those values.

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

Agent-specific files are generated from the ADP workspace config. Real project files are linked into the runtime root. ADP-generated paths take priority inside the runtime view, and the original project directory is not modified. When a project already has provider-local configuration directories such as `.codex/` or `.claude/`, non-conflicting children remain visible in the runtime overlay; ADP wins only at exact generated paths such as `.codex/config.toml` and `.claude/settings.json`.

Runtime overlays do not expose repository Git metadata. Normal project Git files such as `.gitignore`, `.gitattributes`, and `.gitmodules` remain project files and may appear through the overlay, but `.git` metadata is excluded. The runtime also neutralizes repository-directing Git environment variables before launching agents, including `GIT_DIR`, `GIT_WORK_TREE`, `GIT_INDEX_FILE`, `GIT_OBJECT_DIRECTORY`, `GIT_ALTERNATE_OBJECT_DIRECTORIES`, `GIT_COMMON_DIR`, and `GIT_NAMESPACE`. ADP removes that Git routing state while preserving normal shell environment and auth-related variables. `$ADP_RUNTIME_ROOT` is not the authoritative Git worktree, though symlinked subpaths may still map to real project files. Run Git inspection or mutation from the real project root with `git -C "$ADP_PROJECT_ROOT" ...` or by `cd "$ADP_PROJECT_ROOT"` first. `adp workspace doctor` may run read-only Git topology and status checks for diagnostics, but ADP is not a Git workflow wrapper: it does not wrap, intercept, or auto-run Git mutations.

`adp env <workspace> --cd` prints POSIX shell commands for a kept runtime overlay. For shell-hook workflows, the output may unset dangerous repository-directing Git variables before exporting ADP runtime variables and changing to the runtime directory.

`adp shell-hook --shell bash` prints a shell function that calls `adp env <workspace> --cd` and evaluates the result in the parent shell. `sh`, `bash`, and `zsh` are supported.

`adp completion [--shell <bash|zsh>] [--command <name>]` prints deterministic shell completion for the current CLI surface. It defaults to bash when `--shell` is omitted. The optional command name lets packaged binaries or aliases render completion for a command name other than `adp`; that command must be available in the shell because generated scripts call it for dynamic values. Completion covers root commands, subcommands, options, finite option values such as shells, formats, languages, event types, and runtime ages, and read-only local candidates such as agents, workspaces, workspace profiles, task IDs, phase IDs, session IDs, task statuses, and owners. Dynamic candidates read `$ADP_HOME` state only and must not mutate planning, runtime, provider, Git, or project-root state.

P16 hardens the command surface with a local metadata contract that keeps usage text, dispatch wiring, and bash/zsh completion from drifting apart. This remains part of the existing hand-written CLI implementation; it does not adopt a new CLI framework or add a Web UI, dashboard, SaaS tracker, hosted orchestration, automatic Git workflow, automatic task closure, or provider-native resume path.

`adp events list` reads `$ADP_HOME/logs/events.jsonl` and prints recent runtime events with optional workspace, session, task, type, and limit filters.

`adp sessions list [--workspace <name>] [--agent <agent>] [--task <task-id>] [--limit <n>]` groups local event log entries by session so terminal users can inspect recent agent runs without leaving the CLI.

`adp sessions show <session-id>` prints the ordered events for one recorded session, including start, finish, workspace, agent, task ID, runtime path, exit code, and duration data when those fields are available.

`adp sessions restore-plan <session-id>` reads one recorded session and prints a read-only suggested `adp run ...` command when enough non-sensitive invocation data is available. It does not execute the command, launch an agent, create a runtime, append events, change task state, write to the project root, or resume provider-native conversations. See [docs/session-restore.md](docs/session-restore.md).

`adp sessions resume-plan <session-id> [--workspace <name>] [--owner <owner>] [--lease <duration>] [--agent <agent>] [--format <text|json>]` extends that read-only guidance into ADP work-context resume planning. It can suggest a new same-tool or cross-tool `adp run ...` command, include owner and lease preflight notes, show phase context from the local ledger, and classify each suggested command with a machine-readable `side_effect` such as `inspect`, `task_mutation`, or `runtime_creation`. Same-tool plans can reuse non-sensitive `--profile`, `--keep-runtime`, and post-`--` agent arguments from the invocation snapshot. Cross-tool plans keep ADP-safe runtime options such as `--keep-runtime`, but omit provider-specific profile and agent arguments with explicit guidance. The command itself must not claim or renew tasks, complete tasks, accept phases, run Git, create runtimes, append events, write to the project root, or attach to provider-native Codex or Claude conversations.

Operator examples:

```bash
adp sessions resume-plan <session-id> --owner handoff-agent --lease 2h --format text
adp sessions resume-plan <session-id> --agent claude --owner reviewer --lease 1h --format json
```

`adp workspace doctor [name] [--verbose] [--format <text|json>]` validates workspace configuration, project root reachability, runtime parent safety, referenced prompt, memory, MCP, and profile files, agent command settings, reserved project-root paths, and read-only Git topology/status. It reports adapter default command fallback, inline command arguments, missing or non-executable path-like command wrappers, missing, ambiguous, or escaping non-default profiles, and Git boundary caveats as local diagnostics. Without a name it checks all registered workspaces and returns a non-zero exit code when error-level diagnostics are found.

Default text output is operator-focused: it hides info-only diagnostics and prints `ok - no issues` when no warning or error diagnostics remain after that filter. Use `--verbose` when the terminal output should include info diagnostics such as Git topology details. Use `--format json` when tools or sub-agents need the complete machine-readable diagnostic report, including info, warning, and error diagnostics.

Git diagnostics stay read-only. Doctor commands may inspect topology and status, but they do not stage, checkout, commit, push, fetch, clean files, run release evidence commands, or infer phase acceptance, commit evidence, or push evidence.

`adp doctor [workspace] [--verbose] [--format <text|json>]` is the global diagnostics entry point for the same local workspace checks and output modes. It is intended for terminal workflows where diagnostics should be available without first entering the `workspace` command group.

`adp version` and `adp --version` print the CLI build identity. Packaged binaries can inject version, commit, and build-date values at build time; development builds fall back to `dev`.

`adp runtime prune` removes stale ADP-owned runtime directories under `$ADP_RUNTIME_DIR`. A directory is considered pruneable only when it contains a current-version, self-consistent `.adp-runtime.yaml` with `generated_by: adp` and a matching `runtime_root`. Kept runtimes are preserved unless `--include-kept` is passed, and `--dry-run` reports candidates without deleting them.

`adp tasks` and `adp progress` manage workspace-scoped planning and execution progress under `$ADP_HOME/workspaces/<workspace>/planning`. Read-only task, phase, and progress views support `--format json` for local tools and sub-agents that need machine-readable planning snapshots; task objects expose board visibility through `claim_state` plus `owner`, `claimed_at`, and `lease_expires_at` when present. `adp phase status` adds a compact gate snapshot with the open phase, next planned phase, whether the next phase can start, and the next required action. `adp plan doctor` adds read-only local diagnostics for task, phase, progress-log, lock, and phase-gate consistency and returns exit code `2` when error-level diagnostics exist. The authoritative state still stays under `$ADP_HOME`, and task or phase changes remain explicit commands. Use `adp tasks next --format json` to preview the board, `adp tasks take --owner <owner>` when a worker should atomically take one board item without launching an agent, and `adp run <agent> --take --owner <owner>` when task pickup and runtime launch should share one command boundary. Long-running workers can renew with `adp tasks renew`; interrupted leases are visible through read-only `adp tasks stale`, then reclaimed only through an explicit `adp tasks take` or `adp tasks claim` command after ADP ownership rules allow it. `adp run <agent> --task <task-id>` and `adp run <agent> --take ...` bind local task state to runtime environment variables, generated adapter instructions, events, and sessions without writing planning files into the real project root. See [docs/task-management.md](docs/task-management.md).

`adp progress report [--workspace <name>] [--language <en|zh-CN>] [--format markdown|json]` prints a local planning/execution handoff snapshot to stdout. The default output remains English Markdown; `--language zh-CN` applies to Markdown only. `--format json` emits a machine-readable, read-only snapshot with workspace, total task count, phases, task counts, task objects including `claim_state` and owner/lease fields when present, priority-sorted next work, phase evidence, and recent runtime session evidence when local JSONL event/session data exists. JSON output is for cross-tool parsing and must not become a separate state store. The report command is read-only and does not append events, mutate task or phase state, create runtime directories, run agents, run Git, resume provider-native conversations, or write report files into the project root.

P3 provides a local phase ledger for project planning and execution progress management. It records task ownership, optional claim leases, acceptance records, commit records, push records, and explicit stage gate discipline under `$ADP_HOME`. This remains terminal-first and local-first; it is not a Web dashboard, SaaS tracker, cloud sync layer, or hosted orchestration service.

`adp plan preview --workspace <name> --file <path|-> [--format text|json]` and `adp plan apply --workspace <name> --file <path|-> [--format text|json]` provide local planning intake for structured YAML/JSON phase and task input. `adp plan doctor [--workspace <name>] [--format text|json]` inspects the local planning ledger without repairing or mutating it. Preview and doctor are read-only. Apply explicitly writes only `$ADP_HOME/workspaces/<workspace>/planning` after validation succeeds. JSON output is not a second planning store, and ADP does not provide a Web UI, dashboard, SaaS tracker, cloud sync, hosted orchestration, hosted tracker sync, automatic Git, automatic claim/done/phase acceptance, provider-native resume, project-root report/planning exports, or free-text natural-language task splitting.

The repository includes `examples/basic-workspace` as a copyable local workspace configuration with Codex and Claude profiles, base prompts, shared memory, and MCP settings. Replace its `project.root` before running it against a local project. It is intended as a terminal-first reference for how ADP keeps agent configuration outside the real project tree.

## Development

Use the aggregate validation gate before handoff:

```bash
scripts/check-all.sh
```

The aggregate gate covers deterministic runtime smoke, broad runtime audit smoke, focused runtime context smoke, release readiness smoke, release rehearsal smoke, release artifact smoke, release operator drill smoke, install onboarding smoke, example workspace smoke, task manager smoke, plan intake smoke, Go test and vet, file length limits, bilingual documentation pairing and command-reference sync, and whitespace diff checks. CI uses the same `scripts/check-all.sh` gate so local and automated release evidence stay aligned. For targeted example validation, run `scripts/example-workspace-smoke.sh`.

Project code files must stay at or below 700 physical lines. Split files by responsibility before they exceed the limit. See [docs/engineering-standards.md](docs/engineering-standards.md).

Documentation defaults to English and must include Simplified Chinese counterparts using `*.zh-CN.md`.

Runtime smoke acceptance is documented in [docs/runtime-acceptance.md](docs/runtime-acceptance.md).

The launch-time context that agents see inside ADP runtime overlays is audited in [docs/runtime-context-audit.md](docs/runtime-context-audit.md).

Task management and P3 phase gate planning are documented in [docs/task-management.md](docs/task-management.md).

ADP development dogfoods the same local task and phase ledger. For work on this repository, register each phase in the `adp` workspace, validate the phase, record acceptance, commit, push, record commit and push evidence, and only then start the next phase. The planning ledger stays under `$ADP_HOME`; repository documents summarize accepted behavior but are not the execution state store.

Session resume planning and the cross-tool `resume-plan` command are documented in [docs/session-restore.md](docs/session-restore.md).

Real agent compatibility boundaries are documented in [docs/real-agent-compatibility.md](docs/real-agent-compatibility.md), release readiness is tracked in [docs/release-checklist.md](docs/release-checklist.md), and early preview packaging notes are in [docs/release-packaging.md](docs/release-packaging.md).

Agent execution standards, including multi-agent coordination rules and project constraints, are documented in [AGENTS.md](AGENTS.md).

Contribution, security, and licensing policy entry points are [CONTRIBUTING.md](CONTRIBUTING.md), [SECURITY.md](SECURITY.md), and [docs/license-policy.md](docs/license-policy.md).

## License

ADP is source-available for noncommercial learning, research, evaluation, and open collaboration under the [PolyForm Noncommercial License 1.0.0](LICENSE).

Noncommercial redistribution, forks, public references, and release packages must preserve the license text, required notices, and attribution to ADP and the copyright holder. Commercial use requires separate paid authorization; public availability is not a commercial license grant.

Release packages should keep `README.md`, `README.zh-CN.md`, `LICENSE`, `COMMERCIAL.md`, `COMMERCIAL.zh-CN.md`, `CONTRIBUTING.md`, `CONTRIBUTING.zh-CN.md`, `SECURITY.md`, `SECURITY.zh-CN.md`, and `docs/license-policy.md` with the binary and public docs. They must not include `.envrc`, `mvp.md`, local `$ADP_HOME` state, `$ADP_RUNTIME_DIR` contents, runtime overlays, logs, task state, credentials, or machine-specific shell configuration. See [COMMERCIAL.md](COMMERCIAL.md).
