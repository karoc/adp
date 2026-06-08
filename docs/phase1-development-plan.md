# ADP Phase 1 Development Plan

Historical source: local `mvp.md` input. That file is intentionally ignored and is not a maintained repository document.

Date: 2026-06-08

Simplified Chinese: [phase1-development-plan.zh-CN.md](phase1-development-plan.zh-CN.md)

This document turns the ADP MVP into the current Phase 1 implementation plan and operating roadmap. It records the product boundary, architecture, module ownership, multi-agent split points, and validation gates that must remain true as implementation continues.

## 1. MVP Scope

ADP Phase 1 is a terminal-first Agent Runtime Environment:

- Manage `$ADP_HOME`, defaulting to `~/.adp`.
- Register workspaces that map project roots to ADP runtime configuration.
- Build temporary runtime overlays so agents can see generated files such as `AGENTS.md`, `CLAUDE.md`, `.codex/`, and `.claude/` without polluting the real project directory.
- Provide Codex and Claude adapters.
- Support `adp init`, `adp workspace add/list/show/doctor/remove/rename`, `adp doctor`, `adp env`, `adp shell-hook`, `adp completion`, `adp completion values`, `adp version`, `adp events list`, `adp sessions list/show/restore-plan`, `adp runtime prune`, `adp enter`, `adp run`, and the local planning commands `adp tasks`, `adp phase`, `adp phase status`, `adp progress`, and `adp plan preview/apply/doctor`.
- Write a local JSONL event log for future replay, session restore, inspection-only handoff evidence, and terminal-based multi-agent coordination.

Phase 1 explicitly excludes:

- Web UI or dashboard.
- Cloud sync or SaaS features.
- Hosted trackers, hosted tracker semantics, issue-service sync, or SaaS task management.
- Project-root planning, progress, or report exports.
- Automatic Git execution, including automatic commits or pushes.
- Automatic task closure, phase acceptance, commit evidence, or push evidence inference.
- Provider-native conversation resume.
- Graphical or hosted multi-agent orchestration.
- Heavy permission sandboxing.
- Full Windows support. Interfaces should leave room for it, but Linux/macOS are the first target.
- Mandatory kernel overlayfs. The MVP defaults to portable symlink materialization, with bind mount and overlayfs reserved for later backends.

## 2. Technical Stack

Primary language: Go.

Dependencies should stay small and stable:

- CLI command wiring remains hand-written with the Go standard library for Phase 1. P16 adds a local command metadata contract to prevent drift between usage text, dispatch wiring, and bash/zsh completion without adopting a new CLI framework.
- YAML: `gopkg.in/yaml.v3`.
- Tests: Go standard `testing`, with CLI integration tests using temporary `ADP_HOME`, temporary `ADP_RUNTIME_DIR`, and fake agent binaries.

Dependency rules:

- Keep dependencies minimal and defensible.
- Adapter packages must not depend on CLI command wiring.
- Runtime and overlay packages must not depend on concrete adapters.
- Business packages must not read real `~/.adp` directly. Path resolution goes through `internal/paths`.
- New third-party dependencies must be checked for license compatibility with ADP's source-available noncommercial distribution model.

## 2.1 License, Documentation, and Engineering Gates

ADP uses a source-available noncommercial model, not an OSI-approved open-source license.

- Public noncommercial use: `PolyForm Noncommercial License 1.0.0`.
- Personal and organizational use for learning, research, evaluation, and noncommercial open collaboration is allowed.
- Required notices and attribution must be preserved.
- Commercial use requires a separate paid commercial license. See [COMMERCIAL.md](../COMMERCIAL.md).

Documentation rule:

- Default documentation files are English.
- Every project-maintained documentation file should have a Simplified Chinese counterpart named `*.zh-CN.md`.
- `LICENSE` remains the authoritative English legal text. Any translation of legal terms is explanatory only and must not replace the English license.

Code size rule:

- Hand-written project code files must stay at or below 700 physical lines.
- Split files by responsibility before they exceed the limit.
- Generated files, vendored code, lockfiles, license files, and long-form documentation are exempt.
- Local check: `scripts/check-file-lines.sh`.
- Non-blocking pressure audit for split planning: `scripts/check-file-lines.sh --audit`, with `LINE_PRESSURE_WARN_LINES` defaulting to 600.

## 3. Directory Structure

```txt
adp/
├── cmd/
│   └── adp/
│       └── main.go
├── internal/
│   ├── commandmeta/
│   ├── cli/
│   ├── paths/
│   ├── schema/
│   ├── workspace/
│   ├── runtime/
│   ├── overlay/
│   ├── adapters/
│   │   ├── adapter.go
│   │   ├── registry.go
│   │   ├── api/
│   │   ├── claude/
│   │   ├── codex/
│   │   └── shared/
│   ├── runner/
│   ├── shell/
│   ├── sessions/
│   ├── events/
│   ├── planinput/
│   └── tasks/
├── templates/
│   ├── claude/
│   └── codex/
├── test/
│   └── e2e/
├── scripts/
├── docs/
├── examples/
│   └── basic-workspace/
├── README.md
├── README.zh-CN.md
├── COMMERCIAL.md
├── COMMERCIAL.zh-CN.md
├── LICENSE
└── go.mod
```

Package responsibilities:

- `internal/cli`: wire the hand-written command dispatcher, usage text, command metadata contract, and command-focused tests.
- `internal/commandmeta`: define the local command inventory used for usage, dispatch, completion, and drift tests.
- `internal/paths`: resolve `ADP_HOME`, runtime temp roots, workspace paths, and log paths.
- `internal/schema`: define and validate the unified workspace schema.
- `internal/workspace`: create, load, and manage the workspace registry.
- `internal/overlay`: materialize project files and generated files into a runtime root.
- `internal/runtime`: orchestrate workspace config, adapter output, overlay lifecycle, runtime env, manifest handling, and runtime pruning.
- `internal/adapters`: adapter contracts, registry, concrete Codex/Claude adapters, compatibility API aliases, and shared rendering.
- `internal/runner`: execute external commands with controlled cwd, env, and streams.
- `internal/shell`: implement `adp enter`, shell export rendering, parent-shell hook rendering, and shell completion rendering.
- `internal/sessions`: aggregate local events into session history summaries, detail views, and read-only restore plans.
- `internal/events`: append and query JSONL runtime events.
- `internal/planinput`: parse and validate structured local planning imports for `adp plan preview/apply`.
- `internal/tasks`: store workspace-scoped tasks, phases, progress events, ranking, owner leases, and phase acceptance/commit/push evidence under `$ADP_HOME`.
- `templates`: adapter-facing default runtime template material for Codex and Claude.
- `test/e2e`: end-to-end CLI tests using fake agents.

## 4. Local Data Layout

Default ADP home:

```txt
~/.adp/
├── config.yaml
├── workspaces/
│   └── game-a/
│       ├── workspace.yaml
│       ├── prompts/
│       │   └── base.md
│       ├── memory/
│       │   └── shared.md
│       ├── mcp/
│       │   └── config.yaml
│       └── profiles/
│           ├── codex.yaml
│           └── claude.yaml
└── logs/
    └── events.jsonl
```

Tests and local development must support:

- `ADP_HOME` overriding the default home.
- `ADP_RUNTIME_DIR` overriding the default runtime parent directory.

The repository also includes `examples/basic-workspace` as a copyable workspace configuration with base prompts, shared memory, MCP configuration, and Codex/Claude profiles. It is documentation and test material for local-first workspace configuration, not a hosted template service.

## 5. Unified Workspace Schema

The MVP schema stays small and extensible:

```yaml
version: 1

workspace:
  name: game-a

project:
  root: /srv/game-a

memory:
  enabled: true
  shared: memory/shared.md

prompts:
  base: prompts/base.md

rules:
  coding_style: strict

mcp:
  enabled: true
  config: mcp/config.yaml
  servers:
    - github
    - postgres

agents:
  codex:
    enabled: true
    profile: senior-engineer
    command: codex
  claude:
    enabled: true
    profile: architect
    command: claude
```

Schema constraints:

- `workspace.name` must be a safe directory name: `^[a-zA-Z0-9][a-zA-Z0-9._-]*$`.
- `project.root` is stored as an absolute path.
- Prompt, memory, MCP, and profile paths are relative to the workspace directory.
- Adapter-specific settings can later be attached under `agents.<name>.options`.

## 6. Runtime Overlay

MVP backend: symlink materialization.

Runtime root example:

```txt
${ADP_RUNTIME_DIR:-/tmp/adp-runtime}/
└── game-a-20260608T120102-8f3a/
    ├── .adp-runtime.yaml
    ├── AGENTS.md
    ├── CLAUDE.md
    ├── .codex/
    ├── .claude/
    ├── go.mod -> /srv/game-a/go.mod
    ├── cmd -> /srv/game-a/cmd
    └── internal -> /srv/game-a/internal
```

Design choices:

- Agent cwd is the runtime root, not `runtime/source`, so the agent sees a normal project-root shape.
- Real project children are linked into the runtime root.
- ADP-generated files are written directly into the runtime root.
- ADP-generated or reserved paths win over real project paths in the runtime view.
- Reserved-path conflicts are reported as warnings and runtime evidence.
- The real project root is never modified.
- Runtime directories are cleaned after `adp run` and `adp enter` unless `--keep-runtime` is set.
- Kept or stale runtime directories can be inspected and removed with `adp runtime prune`.
- Runtime pruning only deletes direct child directories that contain a current-version, self-consistent `.adp-runtime.yaml` with `generated_by: adp`, non-empty workspace and session IDs, an absolute `project_root`, a matching `runtime_root`, and a valid `created_at`.
- Kept runtimes are preserved by default and are pruned only when `--include-kept` is passed.

Reserved future backends:

- `symlink`: MVP default for Linux/macOS portability.
- `bind`: Linux optional backend when permissions allow.
- `overlayfs`: Linux optional backend for higher fidelity.

## 7. Adapter Contract

Adapters translate the unified schema into agent-specific runtime files and launch specs.

```go
type Adapter interface {
    Name() string
    Validate(context.Context, Context) error
    Render(context.Context, Context) (*RenderResult, error)
    Launch(context.Context, Context, RuntimeHandle, []string) (*LaunchSpec, error)
}
```

Adapter boundaries:

- Adapters generate files, env, and launch specs.
- Adapters do not create runtime directories.
- Adapters do not write event logs.
- Adapters do not parse CLI flags.
- Adapters do not read real home directories directly.

MVP adapters:

- Codex:
  - Generates `AGENTS.md`.
  - Generates `.codex/config.toml`.
  - Defaults to command `codex`, with workspace command override.
- Claude:
  - Generates `CLAUDE.md`.
  - Generates `.claude/settings.json`.
  - Defaults to command `claude`, with workspace command override.

Concrete Codex and Claude CLI config formats can change over time. Verify current provider CLI behavior or official documentation before freezing adapter assumptions, and keep format-specific behavior inside adapter packages.

## 8. CLI Behavior and Acceptance

### `adp init`

Behavior:

- Creates `$ADP_HOME`, `workspaces/`, `logs/`, and default `config.yaml`.
- Is idempotent and must not destroy existing configuration.

Acceptance:

- Empty temporary `ADP_HOME` initializes successfully.
- Existing configuration remains intact on repeated runs.
- Tests use temporary `ADP_HOME`, not the real user home.

### `adp workspace add <name> <project-root>`

Behavior:

- Validates the workspace name and project root.
- Stores the project root as an absolute path.
- Creates default prompt, memory, MCP, profile, and workspace config files.
- Returns clear errors for duplicate workspaces and invalid names.

Acceptance:

- Relative project roots are converted to absolute paths.
- Duplicate workspace names fail without modifying the existing workspace.
- Invalid names fail before writing ADP home data.

### `adp workspace list`

Behavior:

- Prints registered workspace names, project roots, and ADP workspace directories.

Acceptance:

- Empty registries remain readable.
- Multiple workspaces are listed in stable order.

### `adp workspace show <name>`

Behavior:

- Prints operational details for one workspace, including project root, workspace directory, memory status, and MCP status.

Acceptance:

- Missing workspaces return a clear not-found error.
- Output fields remain stable enough for terminal users and tests.

### `adp workspace doctor [name]`

Behavior:

- Checks one workspace, or all registered workspaces when no name is supplied.
- Covers config loading and validation, project root reachability, runtime parent safety, prompt, memory, MCP, profile file references, path escapes, agent command defaults, inline command arguments, path-like command wrapper readiness, unknown enabled agents, and reserved project-root paths.
- Reports diagnostics in a stable terminal format.

Acceptance:

- Healthy workspaces report `ok - no issues`.
- Error-level diagnostics return a non-zero exit code.
- Warning-only command/profile diagnostics keep exit code zero.
- A bad workspace does not prevent reporting diagnostics for other workspaces.

### `adp doctor [workspace]`

Behavior:

- Provides the same local workspace diagnostics as a global command.
- Accepts one optional workspace name; checks all registered workspaces when omitted.

Acceptance:

- `adp doctor game-a` and `adp workspace doctor game-a` have equivalent diagnostic semantics.
- The command does not access the network or require provider CLIs.

### `adp workspace remove <name>`

Behavior:

- Removes the ADP workspace directory without touching the real project root.

Acceptance:

- Missing workspaces return clear errors.
- Invalid names do not mutate ADP home or the project root.

### `adp workspace rename <old-name> <new-name>`

Behavior:

- Renames the ADP workspace directory.
- Updates `workspace.yaml` with the new workspace name.
- Preserves project root, prompt, memory, MCP, and profile files.

Acceptance:

- Missing old names and existing new names fail clearly.
- `workspace show` can read the renamed workspace.
- The old name disappears from local completion candidates.

### `adp enter <workspace> [--keep-runtime]`

Behavior:

- Builds a runtime overlay.
- Sets `ADP_HOME`, `ADP_WORKSPACE`, `ADP_PROJECT_ROOT`, `ADP_RUNTIME_ROOT`, and `ADP_SESSION_ID`.
- Starts a child shell in the runtime root. A CLI process cannot change the parent shell cwd, so this command intentionally starts a child shell.
- Cleans the runtime on shell exit unless `--keep-runtime` is set.

Acceptance:

- Shell cwd is the runtime root.
- `pwd`, env values, generated files, and project symlinks are visible from the child shell.
- Runtime cleanup and `--keep-runtime` preservation are both covered without requiring a real interactive shell in default smoke.

### `adp env <workspace> [--cd]`

Behavior:

- Builds a kept runtime overlay.
- Prints POSIX-compatible shell exports for the ADP runtime environment.
- With `--cd`, also prints a quoted `cd` command for the runtime root.

Acceptance:

- Output order is deterministic.
- Shell quoting handles spaces, single quotes, and special characters.
- The runtime root contains `.adp-runtime.yaml`.

### `adp shell-hook [--shell <sh|bash|zsh>] [--name <function-name>]`

Behavior:

- Prints a shell function that calls `adp env <workspace> --cd`.
- Lets users evaluate exports and `cd` in the parent shell.
- Does not change `adp enter`; `enter` still starts a child shell.

Acceptance:

- Supports `sh`, `bash`, and `zsh`.
- Function names are conservatively validated to avoid shell injection.
- Output remains deterministic for tests and shell configuration.

### `adp completion [--shell <bash|zsh>] [--command <name>]`

Behavior:

- Prints deterministic shell completion for supported shells, defaulting to bash when `--shell` is omitted.
- Covers the current command surface, including nested workspace, event, runtime, session, task, phase, progress, and plan subcommands.
- Supports packaged binary names or aliases through `--command`.
- Uses read-only dynamic endpoints for local candidates: `adp completion values agents`, `adp completion values workspaces`, and `adp completion values profiles [--workspace <name>]`.

P16 adds a local command metadata contract for the command surface. Usage text, dispatch wiring, and bash/zsh completion should be checked against that metadata so a command cannot be reachable in one place and missing from another. The contract is for local drift prevention only; it must not become a CLI framework migration, hosted command registry, Web UI, SaaS tracker, automatic Git path, automatic task or phase closure path, provider-native resume path, or project-root export mechanism.

Acceptance:

- Supports bash and zsh.
- Shell names and command names are conservatively validated.
- Dynamic value endpoints read the local adapter registry and workspace/profile state only; they do not initialize workspaces, access networks, or mutate planning/runtime state.

### `adp version`

Behavior:

- Prints the local CLI build identity.
- Development builds may report `dev`.
- Preview release binaries should inject version, commit, and build-date values with Go linker flags.

Acceptance:

- `adp version` and `adp --version` output remain stable.
- Missing linker flags still produce a usable development identity.
- Release packaging docs describe the linker flag fields.

### `adp events list [--workspace <name>] [--session <session-id>] [--task <task-id>] [--type <event-type>] [--limit <n>]`

Behavior:

- Reads `$ADP_HOME/logs/events.jsonl`.
- Filters JSONL events by workspace, session, task, event type, and limit.
- Prints recent matching events in a stable terminal table.
- Reports corrupted event log lines with line numbers instead of silently ignoring them.

Acceptance:

- Missing event logs produce an empty result.
- `--limit` returns the most recent matching records while preserving chronological output order.
- The command remains read-only.

### `adp sessions list [--workspace <name>] [--agent <agent>] [--task <task-id>] [--limit <n>]`

Behavior:

- Groups local event log records by `session_id`.
- Supports workspace, agent, task, and limit filters.
- Ignores events with empty session IDs.

Acceptance:

- Filters are applied before limiting.
- Selected sessions remain in chronological session-start order for terminal readability.
- Missing event logs produce an empty result without creating runtime state.

### `adp sessions show <session-id>`

Behavior:

- Prints ordered events for one session.
- Derives data from the local JSONL event log.

Acceptance:

- Missing sessions return a clear not-found error.
- The command is read-only and does not create, mutate, or delete runtime state.

### `adp sessions restore-plan <session-id>`

Behavior:

- Prints a read-only suggested `adp run ...` command for a previous session when enough non-sensitive invocation snapshot data is available.
- Emits partial plans with missing fields and reasons when historical data is incomplete.

Acceptance:

- The command does not execute the suggestion, launch an agent, create runtime state, append events, mutate task state, write to the project root, or resume provider-native conversations.
- Output event ordering follows the source log ordering.

### `adp progress report [--workspace <name>] [--language <en|zh-CN>] [--format markdown|json]`

Behavior:

- Prints a local planning/execution handoff snapshot to stdout.
- Reads the local planning ledger under `$ADP_HOME`.
- Defaults to English Markdown.
- Emits Simplified Chinese Markdown only when `--language zh-CN` is passed.
- Applies `--language` to Markdown only.

With `--format json`, the command emits a machine-readable, read-only handoff snapshot with workspace, total task count, phases, task counts, tasks, priority-sorted next work, phase evidence, and recent runtime session evidence when local JSONL runtime events and session data exist. JSON is for cross-tool parsing and must not become a separate state store.

When local JSONL runtime events and session data exist, Markdown and JSON reports include recent runtime session evidence derived from `$ADP_HOME/logs/events.jsonl`. This evidence is for inspection and handoff only.

Acceptance:

- Output goes to stdout; the command never auto-creates or updates report files.
- It must not append events, mutate task state, mutate phase state, create runtime directories, start agents, run Git, push, infer acceptance, close tasks, resume provider-native conversations, or write report files into the real project root.
- Task state, phase state, event log, runtime directories, Git state, and the real project root remain unchanged.

### `adp progress [--workspace <name>] [--format text|json]`

Behavior:

- Prints current workspace planning progress.
- Counts task statuses and exposes priority-sorted next work.
- Supports text for terminal scanning and JSON for local cross-tool parsing.

Acceptance:

- The command is read-only.
- JSON output is a snapshot, not a second planning store.
- It does not mutate task, phase, event, runtime, Git, hosted service, or project-root state.

### `adp tasks add|list|show|update|claim|release|done|block`

Behavior:

- Stores workspace-scoped tasks under `$ADP_HOME/workspaces/<workspace>/planning`.
- Supports local task creation, listing, showing, status updates, owner claims with optional leases, owner-checked release, done transitions, and blocker recording.
- Uses local file locking around mutating planning operations.
- Validates task phase IDs once the phase ledger exists.

Acceptance:

- Mutations affect only the local planning ledger under `$ADP_HOME`.
- Owner conflicts and non-expired leases are enforced.
- Read-only task views support stable JSON output for local tools.
- Task commands do not run Git, start agents, write project-root planning files, or sync hosted trackers.

### `adp tasks next [--workspace <name>] [--limit <n>] [--format text|json]`

Behavior:

- Prints a compact local next-work snapshot to stdout.
- Reads the workspace planning ledger under `$ADP_HOME`.
- Selects tasks with `ready`, `in_progress`, or `review` status.
- Sorts candidates by priority and stable local tie-breakers so terminal users and sub-agents can choose follow-up work without parsing the full progress report.

Text is the default format and should be easy to scan in a terminal. `--limit <n>` caps candidates, defaults to 5, and accepts `0` for an untruncated snapshot. JSON output is for local cross-tool parsing and includes stable fields for workspace, planning source, generated timestamp, total task count, eligible candidate count, status counts, requested limit, ordered `candidates`, and a singular `next` first-candidate value when work is available.

Acceptance:

- The command is read-only.
- It must not claim tasks, mutate task status, change owners or leases, clear blockers, mutate phases, append events, create runtime directories, start agents, run Git, push, infer acceptance, close tasks, resume provider-native conversations, write files into the real project root, sync with hosted trackers, or maintain JSON as a second planning store.

### `adp phase add|list|show|status|start|accept|commit|push`

Behavior:

- Stores workspace-scoped phase records under `$ADP_HOME/workspaces/<workspace>/planning`.
- Tracks phase status, goal, acceptance command evidence, commit evidence, and push evidence.
- Assigns an explicit local order to new phases and plan-imported phases so later phases cannot skip earlier planned or unfinished phases.
- `adp phase status [--workspace <name>] [--format text|json]` prints a read-only gate snapshot with the open phase, next planned phase, whether the next phase can start, and the next required action.
- Enforces phase lifecycle ordering: planned, active, accepted, committed, pushed.
- Guards the phase process: acceptance before commit evidence, commit before push evidence, and successful pushed evidence for every earlier phase before starting a later phase.

Acceptance:

- Phase evidence is local ledger data, not Git automation.
- `phase commit` records a commit hash and message but does not create the commit.
- `phase push` records remote, branch, and result but does not run `git push`.
- Later phases must not start until every earlier phase is accepted, committed, successfully pushed, and recorded.
- `phase status` is read-only and must not mutate tasks, phases, events, runtime directories, Git, hosted services, or the real project root.

### `adp plan preview|apply [--workspace <name>] --file <path|-> [--format text|json]`

Behavior:

- Accepts structured local YAML/JSON planning input with phases and tasks.
- Supports regular files and stdin through `--file -`.
- `preview` renders the proposed import to stdout without creating planning files or directories.
- `apply` explicitly writes the validated batch to `$ADP_HOME/workspaces/<workspace>/planning`.

JSON output is for inspection and local cross-tool parsing only. It must not become a second planning store. Plan intake must not split free-text natural language into tasks, sync hosted trackers, run Git, start agents, infer acceptance, claim or close tasks automatically, write planning files into the real project root, or mutate runtime state.

Acceptance:

- Preview is read-only.
- Apply writes only the local planning ledger under `$ADP_HOME`.
- Failed apply leaves no partial phase, task, or progress state.
- Stdin intake preserves the same preview/apply mutation boundaries as file intake.

### `adp runtime prune [--older-than <duration>] [--include-kept] [--dry-run]`

Behavior:

- Scans direct child directories under `$ADP_RUNTIME_DIR`.
- Treats a directory as a prune candidate only when it contains a current-version, self-consistent `.adp-runtime.yaml` with `generated_by: adp`, non-empty workspace and session IDs, an absolute `project_root`, a matching `runtime_root`, and a valid `created_at`.
- Removes ADP-owned runtime directories older than `--older-than`.
- Skips `keep: true` by default unless `--include-kept` is passed.
- Reports candidates without deleting when `--dry-run` is set.

Acceptance:

- Incompatible, malformed, externally generated, or self-inconsistent manifests are skipped.
- Delete targets are scanned runtime child directories, never paths derived from manifest project roots.
- Kept runtime behavior is covered for default and `--include-kept` modes.

### `adp run <agent> [--workspace <name>] [--profile <profile>] [--task <task-id>] [--keep-runtime] [-- <agent-args>...]`

Behavior:

- Resolves the workspace, renders adapter files, builds the runtime overlay, launches the agent, logs start/finish events, passes through streams, and returns the agent exit code.
- With `--task`, binds runtime sessions to local task state and injects task context into runtime env and generated adapter instructions.

Workspace resolution order:

- Explicit `--workspace`.
- `ADP_WORKSPACE`.
- Match the current directory to a registered project root, using the longest matching project root when workspaces are nested.

Acceptance:

- Fake Codex and Claude binaries verify cwd, env, generated files, project symlinks, args, exit code, events, sessions, task binding, and cleanup.
- Agent cwd is the runtime root.
- The real project root is not modified.

## 9. Event Log

Format: JSONL.

Path:

```txt
$ADP_HOME/logs/events.jsonl
```

Example events:

```json
{"ts":"2026-06-08T12:01:02Z","type":"run_started","workspace":"game-a","agent":"codex","profile":"senior-engineer","runtime_path":"/tmp/adp-runtime/game-a-...","project_root":"/srv/game-a","session_id":"..."}
{"ts":"2026-06-08T12:03:44Z","type":"run_finished","workspace":"game-a","agent":"codex","session_id":"...","exit_code":0,"duration_ms":162000}
```

Constraints:

- Do not log API keys, tokens, or full env maps.
- Event log write failures should warn on stderr but should not prevent agent startup.
- One complete JSON object per line.
- `adp events list` must return the most recent matching events while preserving chronological output order.
- `adp sessions list/show/restore-plan` are read-only views over the same local log and must not create, mutate, or delete runtime state.

## 10. Parallel Development Boundaries

Parallel work is allowed only when ownership boundaries are explicit and the phase gate can still be closed as one integrated slice.

Current parallel slices:

- CLI command surface: `cmd/`, `internal/cli/`, `internal/commandmeta/`, and command-focused tests.
- Workspace registry and diagnostics: `internal/workspace/`, `internal/schema/`, and related CLI wiring.
- Runtime and overlay: `internal/runtime/`, `internal/overlay/`, manifest handling, lifecycle smoke, and runtime acceptance docs.
- Adapter layer: `internal/adapters/`, `templates/`, adapter tests, and provider compatibility docs.
- Runner, shell, events, and sessions: `internal/runner/`, `internal/shell/`, `internal/events/`, `internal/sessions/`.
- Local planning manager: `internal/tasks/`, `internal/planinput/`, task/phase/progress CLI wiring, and planning smoke.
- Docs and examples: `README*`, `AGENTS*`, `docs/`, and `examples/`.
- Integration QA and gates: `scripts/`, `test/e2e/`, `.github/`, and release checklist docs.

Rules:

- Main-thread integration owns the immediate blocking path.
- Sub-agents get disjoint write scopes; overlapping file sets should be read-only review unless there is an explicit handoff.
- Public contract changes must be explicit and validated across dependent packages.
- `LICENSE` and `COMMERCIAL*` are maintained by project maintainers.
- Hand-written code files must stay at or below 700 lines.
- Documentation must keep English default files and Simplified Chinese counterparts aligned.
- Each phase slice is validated, accepted, committed, pushed, and recorded before the next phase starts.

Sub-agent task prompts should state objective, allowed write paths, disallowed paths, constraints, validation commands, and expected final report. Read-only review agents must be told not to edit files.

## 11. Validation Gates

Required local checks:

```bash
scripts/check-all.sh
```

The aggregate gate runs the deterministic smoke and repository checks:

```bash
scripts/runtime-smoke.sh --fake
scripts/runtime-audit-smoke.sh
scripts/release-readiness-smoke.sh
scripts/release-rehearsal-smoke.sh
scripts/example-workspace-smoke.sh
scripts/task-manager-smoke.sh
scripts/plan-intake-smoke.sh
go test -count=1 ./...
go vet ./...
scripts/check-file-lines.sh
scripts/check-docs-bilingual.sh
git diff --check
```

End-to-end expectations:

```bash
export ADP_HOME="$(mktemp -d)"
export ADP_RUNTIME_DIR="$(mktemp -d)"

adp init
adp workspace add game-a /srv/game-a
adp workspace list
adp workspace show game-a
adp workspace doctor game-a
adp env game-a --cd
adp shell-hook --shell bash
adp completion --shell bash
adp tasks add --workspace game-a --priority high --phase phase-1 "Bind runtime session to task"
adp tasks next --workspace game-a --limit 0 --format json
adp plan preview --workspace game-a --file plan.yaml
adp plan doctor --workspace game-a --format json
adp run codex --workspace game-a --task <task-id> -- --version
cd /srv/game-a && adp run claude -- --version
adp events list --workspace game-a --task <task-id>
adp sessions list --workspace game-a --agent codex --task <task-id>
adp sessions show <session-id>
adp sessions restore-plan <session-id>
adp runtime prune --older-than 24h --dry-run
adp enter game-a
adp workspace rename game-a game-renamed
adp workspace remove game-renamed
```

- `adp init` creates local ADP home.
- `adp workspace add` creates a workspace config without touching the project root.
- `adp workspace list` and `adp workspace show` expose registered workspace details.
- `adp workspace doctor` reports healthy workspaces and returns a non-zero exit code for error-level diagnostics.
- `adp doctor` exposes the same diagnostics globally.
- `adp workspace remove` and `adp workspace rename` modify only ADP workspace registry data.
- `adp env` prints shell-safe exports for a kept runtime overlay.
- `adp shell-hook` prints a deterministic shell function for `sh`, `bash`, and `zsh`.
- `adp completion` prints deterministic completion for `bash` and `zsh`, and `adp completion values` returns local agent, workspace, and profile candidates.
- The P16 command metadata drift check proves the local command inventory, usage text, dispatch wiring, and bash/zsh completion remain aligned without adopting a new CLI framework.
- `adp version` reports the CLI build identity.
- `adp events list` prints filtered run history from JSONL events.
- `adp sessions list`, `adp sessions show`, and `adp sessions restore-plan` expose local session history and read-only restore planning derived from JSONL events.
- `adp progress report [--workspace <name>] [--language <en|zh-CN>] [--format markdown|json]` prints a Markdown planning/execution report to stdout by default, emits a read-only JSON handoff snapshot with `--format json`, includes recent local runtime session evidence when JSONL event/session data exists, and leaves planning state, Git state, runtime state, event logs, and the real project root unchanged.
- `adp tasks next [--workspace <name>] [--limit <n>] [--format text|json]` prints a compact prioritized next-work snapshot to stdout, exposes a stable JSON contract for local tools, and leaves task state, phase state, Git state, runtime state, event logs, hosted services, and the real project root unchanged.
- `adp phase status [--workspace <name>] [--format text|json]` prints a compact read-only phase gate snapshot to stdout, exposes a stable JSON contract for local tools, and leaves task state, phase state, Git state, runtime state, event logs, hosted services, and the real project root unchanged.
- `adp plan preview/apply [--workspace <name>] --file <path|-> [--format text|json]` accepts structured local planning input; preview stays read-only, apply writes only the local planning ledger under `$ADP_HOME`, and failed apply leaves no partial phase, task, or progress state.
- `adp plan doctor [--workspace <name>] [--format text|json]` prints read-only local planning ledger diagnostics for task, phase, progress-log, lock, and phase-gate invariants, returns exit code `2` for error-level diagnostics, and leaves planning state, Git state, runtime state, event logs, hosted services, and the real project root unchanged.
- `adp runtime prune` reports and removes only current-version, self-consistent ADP-owned runtime directories.
- `adp run codex` and `adp run claude` build runtime overlays, and `--task <task-id>` binds runtime sessions to workspace task state.
- `examples/basic-workspace` remains a valid local workspace reference with bilingual Markdown prompt and memory files.
- Fake agent tests can assert cwd, env, generated files, symlinks, args, exit code, logs, and cleanup.
- The real project directory must not gain `AGENTS.md`, `CLAUDE.md`, `.codex/`, `.claude/`, `planning/`, `tasks.yaml`, `phases.yaml`, `progress.jsonl`, or report export files.

Phase process gate:

1. Complete one planned phase slice.
2. Run focused validation and the required aggregate gate.
3. Record phase acceptance only when validation passes.
4. Commit the accepted phase.
5. Push the commit.
6. Record commit and push evidence in the local phase ledger.
7. Start the next phase only after the push succeeds and the phase record is updated.

## 12. Next Work

Next work is prioritized by how much it improves ADP's terminal-first runtime and workspace management loop without drifting into hosted project management or a dashboard.

- P0 completed: Task and Progress Manager MVP. Store workspace-scoped task state under `$ADP_HOME/workspaces/<workspace>/planning`, expose `adp tasks` and `adp progress`, and validate it with a task-manager smoke.
- P1 completed: Runtime task binding. Add `adp run <agent> --task <task-id>`, inject task context into runtime env and generated adapter instructions, and connect task IDs to events and sessions.
- P2 completed: Early preview hardening. Dynamic agent/workspace/profile completion, global `adp doctor`, version output, CI for `scripts/check-all.sh`, and release packaging notes are covered by the aggregate gate and runtime smoke.
- P3 Phase Gate MVP completed: Project planning and execution progress management now has phase records, task claim and owner records, acceptance or gate records, commit records, push records, and task-manager smoke coverage.
- P3 planning coordination hardening completed: Mutating planning operations use a local lock, task claims enforce owner conflicts and optional leases, owner-checked release is available, tasks validate phase IDs once a phase ledger exists, and phase lifecycle guards enforce accept-before-commit, commit-before-push, and push-before-next-phase discipline.
- P4 runtime manifest compatibility completed: runtime manifests now use an explicit manifest version, runtime smoke checks core manifest fields, and pruning skips incompatible or self-inconsistent manifests instead of treating every `generated_by: adp` file as safe deletion evidence.
- P4 workspace runtime-parent diagnostics completed: workspace and global doctor now reject runtime parents placed at the filesystem root, equal to the project root, inside the project root, or containing the project root, and warn on symlinked runtime parents.
- P4 agent command/profile diagnostics completed: workspace and global doctor now report reserved project-root paths, adapter default command fallback, inline command arguments, missing or non-executable path-like command wrappers, invalid, missing, ambiguous, not-file, or escaping non-default profiles, and unknown enabled agents without running provider CLIs.
- P4 session restore foundation completed: `run_started` events now record non-sensitive invocation snapshots, `adp sessions restore-plan <session-id>` prints read-only suggested commands, and runtime plus example smoke cover session events, session history, restore-plan event-log immutability, and examples/docs polish.
- P5 planning JSON output completed: read-only `--format json` output for task, phase, and progress views gives local tools and sub-agents machine-readable planning snapshots without scraping terminal text or changing state.
- P6 progress report output completed: `adp progress report [--workspace <name>] [--language <en|zh-CN>]` prints a read-only local Markdown planning/execution report to stdout. English is the default; Simplified Chinese requires `--language zh-CN`. Task-manager smoke proves the report leaves task, phase, Git, runtime, event log, and project-root state unchanged.
- P7 progress report runtime session evidence completed: when local JSONL runtime events and session data exist, `adp progress report [--workspace <name>] [--language <en|zh-CN>]` includes recent runtime session evidence for inspection-only handoff. It does not append events, mutate tasks or phases, create runtime directories, run agents, run Git, write report files into project roots, or resume provider-native conversations.
- P8 progress report JSON handoff snapshot completed: `adp progress report [--workspace <name>] [--language <en|zh-CN>] [--format markdown|json]` keeps default output as English Markdown, applies `--language zh-CN` to Markdown only, and emits a read-only machine-readable snapshot with `--format json`. The JSON snapshot includes workspace, total task count, phases, task counts, tasks, priority-sorted next work, phase evidence, and recent runtime session evidence when local JSONL event/session data exists. It is for cross-tool parsing and must not become a separate state store.
- P9 task-manager smoke modularization completed: the oversized task-manager runtime smoke is split into a smaller public entry point, shared shell helper library, and dedicated JSON report validator before breaching the 700-line code-file limit. `scripts/task-manager-smoke.sh` remains the public entry point for workspace-local task, phase, and progress report runtime acceptance, and `scripts/check-all.sh` remains the aggregate gate.
- P9 is maintenance and hardening only. It preserves coverage for project-root pollution protection and read-only progress report behavior.
- P10 task next-work endpoint completed: `adp tasks next [--workspace <name>] [--limit <n>] [--format text|json]` provides a compact read-only local task-selection snapshot for terminal users and sub-agents. It narrows the existing progress-report next-work data into a focused command without claiming tasks, changing state, running Git, starting agents, writing project-root files, syncing hosted trackers, or creating another planning store.
- P11 task command test split completed: the growing task command test coverage is split into focused test files before any hand-written code file breaches the 700-line limit. P11 is maintenance and hardening only: no runtime behavior change, no new product command, no Web/SaaS/hosted orchestration drift, no Git automation, and no change to the terminal-first local planning boundary.
- P12 CLI parse helper split completed: the growing `internal/cli/parse.go` helper surface is split into focused files before it breaches the 700-line code-file limit. P12 is maintenance and hardening only: no runtime behavior change, no new product command, no Web/SaaS/hosted orchestration drift, no Git automation, and no change to the terminal-first local planning boundary.
- P13 CLI base test split completed: the growing `internal/cli/cli_test.go` base CLI coverage is split before it breaches the 700-line code-file limit. P13 is maintenance-only: no runtime behavior change, no new product command, no Web/SaaS/hosted orchestration drift, no Git automation, and no change to the terminal-first local planning boundary.
- P14 local planning intake preview/apply completed: `adp plan preview --workspace <name> --file <path|-> [--format text|json]` and `adp plan apply --workspace <name> --file <path|-> [--format text|json]` accept structured YAML/JSON phase and task input. Preview is read-only; apply explicitly writes only `$ADP_HOME/workspaces/<workspace>/planning`; JSON output is not a second planning store; ADP does not split free-text natural language into tasks in this first version.
- P15 MVP completion audit completed: command/runtime coverage, release-gate documentation, maintainability pressure, and bilingual roadmap drift were audited. The audit seeded the next local planning backlog as planned phases P16-P23 without starting any later phase.
- P16 command surface hardening completed: the local command metadata contract now keeps usage text, dispatch wiring, bash/zsh completion, focused tests, and smoke or documentation acceptance aligned. This is local CLI maintenance only and does not introduce a new CLI framework or hosted command surface.
- P17 runtime smoke split completed: `scripts/runtime-smoke.sh` remains the public entry point, while shared helpers and fake diagnostics/session/prune slices are split into focused implementation files under the 700-line cap. The phase is maintenance-only and preserves fake smoke coverage, fake subshell isolation, real CLI opt-in gates, and `scripts/check-all.sh` as the aggregate gate.
- P18 CLI command test split completed: the remaining large mixed CLI tests are split into focused task CRUD/progress/report/phase/helper files and shell/completion/events/sessions/runtime-prune files. The phase is maintenance-only: no runtime behavior change, no product command change, no Web/SaaS/hosted orchestration drift, and no Git automation.
- P19 workspace lifecycle and enter acceptance completed: runtime smoke now covers workspace rename/remove with project-root sentinel preservation, stale workspace completion checks, and controlled non-interactive `adp enter` execution through a fake `SHELL`. The enter smoke verifies runtime env/cwd, project symlinks, cleanup versus `--keep-runtime`, and no event-log mutation without launching a real interactive shell.
- P20 plan stdin coverage completed: focused CLI tests and plan-intake smoke now cover `adp plan preview --file -` and `adp plan apply --file -` with piped YAML/JSON, preserving preview read-only behavior, explicit apply boundaries, local planning-ledger writes only, JSON inspection semantics, and no runtime/Git/event-log/project-root side effects.
- P21 taskstore maintainability split completed: `internal/tasks` core responsibilities are now separated into same-package store, task model, task lifecycle, task persistence, progress events, task ranking, phase model, phase lifecycle, phase persistence, and phase helper files. The split is mechanical and preserves public APIs, local ledger semantics, plan-import atomic staging, phase-gate lifecycle behavior, and runtime acceptance coverage while keeping all touched code files well below the 700-line cap.
- P22 Phase 1 bilingual roadmap normalization completed: the English default roadmap and Simplified Chinese counterpart now share the same section tree, current command surface, directory responsibilities, local-first non-goals, validation gates, E2E expectations, and validate/accept/commit/push/record phase discipline.
- P23 line pressure audit tooling completed: `scripts/check-file-lines.sh --audit` reports files at or above `LINE_PRESSURE_WARN_LINES`, defaulting to 600, and exits zero so split phases can be planned before the hard 700-line cap is breached. The required `scripts/check-file-lines.sh` hard gate and `scripts/check-all.sh` pass/fail semantics remain unchanged.
- P24 phase gate status and ordering hardening completed: `adp phase status [--workspace <name>] [--format text|json]` exposes a read-only local gate snapshot, new phases carry explicit local order, phase start rejects skipped earlier planned or unfinished phases, and successful push evidence cannot be overwritten by failed push evidence.
- P25 shell completion renderer split completed: bash and zsh completion rendering are split into shell-specific files while `RenderCompletion`, command-name validation, metadata-backed candidates, dynamic local value endpoints, and public `adp completion` behavior remain unchanged. This is maintenance-only line-pressure work and does not add commands, shell types, Web/SaaS behavior, automatic Git execution, hosted orchestration, provider-native resume, or project-root exports.
- P26 planning ledger doctor completed: `adp plan doctor [--workspace <name>] [--format text|json]` reports read-only local diagnostics for task, phase, progress-log, lock, and phase-gate invariants; error diagnostics return exit code `2`; healthy and broken ledger paths are covered by focused tests and task-manager smoke without automatic repair, Git execution, runtime mutation, hosted tracker sync, or project-root exports.
- Completed Phase 1 slices keep the same non-goals: no Web dashboard, SaaS tracker, cloud sync, hosted orchestration, hosted tracker sync, automatic Git execution, automatic claim/done/phase acceptance, provider-native conversation resume, remote issue-service integration, project-root report or planning exports, or hosted tracker semantics.

Each phase slice must be validated, accepted, committed, pushed, and recorded before the next slice starts.
