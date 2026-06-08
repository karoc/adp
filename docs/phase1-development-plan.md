# ADP Phase 1 Development Plan

Source: [mvp.md](../mvp.md)

Date: 2026-06-08

Simplified Chinese: [phase1-development-plan.zh-CN.md](phase1-development-plan.zh-CN.md)

This document turns the ADP MVP into an implementation plan that can be split across parallel agents without losing ownership boundaries. The goal is to freeze the architecture, contracts, module responsibilities, and validation gates before expanding implementation work.

## 1. MVP Scope

ADP Phase 1 is a terminal-first Agent Runtime Environment:

- Manage `$ADP_HOME`, defaulting to `~/.adp`.
- Register workspaces that map project roots to ADP runtime configuration.
- Build temporary runtime overlays so agents can see generated files such as `AGENTS.md`, `CLAUDE.md`, `.codex/`, and `.claude/` without polluting the real project directory.
- Provide Codex and Claude adapters.
- Support `adp init`, `adp workspace add/list/show/doctor/remove/rename`, `adp doctor`, `adp env`, `adp shell-hook`, `adp completion`, `adp completion values`, `adp version`, `adp events list`, `adp sessions list/show/restore-plan`, `adp runtime prune`, `adp enter`, `adp run`, and the local planning commands `adp tasks`, `adp phase`, `adp progress`, and `adp plan preview/apply`.
- Write a local JSONL event log for future replay, session restore, and multi-agent orchestration.

Phase 1 explicitly excludes:

- Web UI or dashboard.
- Cloud sync or SaaS features.
- Hosted trackers or project-root report exports.
- Graphical multi-agent orchestration.
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

## 3. Directory Structure

```txt
adp/
├── cmd/
│   └── adp/
│       └── main.go
├── internal/
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
│   └── events/
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
- `internal/paths`: resolve `ADP_HOME`, runtime temp roots, workspace paths, and log paths.
- `internal/schema`: define and validate the unified workspace schema.
- `internal/workspace`: create, load, and manage the workspace registry.
- `internal/overlay`: materialize project files and generated files into a runtime root.
- `internal/runtime`: orchestrate workspace config, adapter output, overlay lifecycle, runtime env, manifest handling, and runtime pruning.
- `internal/adapters`: adapter contracts, registry, concrete Codex/Claude adapters, and shared rendering.
- `internal/runner`: execute external commands with controlled cwd, env, and streams.
- `internal/shell`: implement `adp enter`, shell export rendering, parent-shell hook rendering, and shell completion rendering.
- `internal/sessions`: aggregate local events into session history summaries, detail views, and read-only restore plans.
- `internal/events`: append and query JSONL runtime events.
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

Concrete Codex and Claude CLI config formats can change over time. Keep format-specific behavior inside adapter packages.

## 8. CLI Behavior

### `adp init`

Creates `$ADP_HOME`, `workspaces/`, `logs/`, and default `config.yaml`. It is idempotent and must not destroy existing config.

### `adp workspace add <name> <project-root>`

Validates the workspace name and project root, stores the absolute project root, and creates default prompt, memory, MCP, profile, and workspace config files.

### `adp workspace list`

Prints registered workspace names, project roots, and ADP workspace directories.

### `adp workspace show <name>`

Prints operational details for one workspace, including project root, workspace directory, memory status, and MCP status.

### `adp workspace doctor [name]`

Checks one workspace, or all registered workspaces when no name is supplied. Diagnostics cover config loading and validation, project root reachability, runtime parent safety, prompt, memory, MCP, profile file references, path escapes, agent command defaults, inline command arguments, path-like command wrapper readiness, unknown enabled agents, and reserved project-root paths. Error-level diagnostics should be terminal-readable and return a non-zero process exit code; warning-only command/profile diagnostics should keep doctor exit code zero.

### `adp doctor [workspace]`

Provides the same local workspace diagnostics as a global command. It should accept one optional workspace name, fall back to all registered workspaces when omitted, and stay equivalent to the workspace command group for diagnostic behavior and exit codes.

### `adp workspace remove <name>`

Removes the ADP workspace directory without touching the real project root.

### `adp workspace rename <old-name> <new-name>`

Renames the ADP workspace directory and updates `workspace.yaml` while preserving project root, prompt, memory, MCP, and profile files.

### `adp enter <workspace> [--keep-runtime]`

Builds a runtime overlay and starts a child shell in the runtime root. A CLI process cannot change the parent shell cwd, so Phase 1 intentionally starts a child shell. Later shell-hook integration can be added separately.

### `adp env <workspace> [--cd]`

Builds a kept runtime overlay and prints POSIX shell exports for the ADP runtime env. With `--cd`, it also prints a quoted `cd` command for the runtime root.

### `adp shell-hook [--shell <sh|bash|zsh>] [--name <function-name>]`

Prints a shell function that calls `adp env <workspace> --cd` and evaluates the result in the parent shell. This provides a parent-shell workflow without changing the behavior of `adp enter`, which still starts a child shell.

### `adp completion [--shell <bash|zsh>] [--command <name>]`

Prints deterministic shell completion for supported shells, defaulting to bash when `--shell` is omitted. The completion script should cover the current command surface, including nested workspace, event, runtime, and session subcommands. The optional command name supports packaged binaries or aliases without hard-coding `adp`. Dynamic suggestions should be provided through read-only local endpoints: `adp completion values workspaces` and `adp completion values profiles [--workspace <name>]`.

P16 adds a local command metadata contract for the command surface. Usage text, dispatch wiring, and bash/zsh completion should be checked against that metadata so a command cannot be reachable in one place and missing from another. The contract is for local drift prevention only; it must not become a CLI framework migration, hosted command registry, Web UI, SaaS tracker, automatic Git path, automatic task or phase closure path, provider-native resume path, or project-root export mechanism.

### `adp version`

Prints the local CLI build identity. Development builds may report `dev`; preview release binaries should inject version, commit, and build-date values with Go linker flags.

### `adp events list [--workspace <name>] [--session <session-id>] [--type <event-type>] [--limit <n>]`

Reads `$ADP_HOME/logs/events.jsonl`, filters JSONL events, and prints recent matching events in a stable terminal table. Corrupted event log lines are reported with line numbers instead of being ignored.

### `adp sessions list [--workspace <name>] [--agent <agent>] [--task <task-id>] [--limit <n>]`

Groups local event log records by `session_id` and prints recent session summaries. Empty session IDs are ignored. Workspace and agent filters should be applied before limiting, and the selected sessions should remain in chronological session-start order for terminal readability.

### `adp sessions show <session-id>`

Prints the ordered events for one session. Missing sessions return a clear not-found error. The command is read-only and derives its data from the local JSONL event log.

### `adp sessions restore-plan <session-id>`

Prints a read-only suggested `adp run ...` command for a previous session when enough non-sensitive invocation snapshot data is available. The command must not execute the suggestion, launch an agent, create runtime state, append events, mutate task state, write to the project root, or resume provider-native conversations.

### `adp progress report [--workspace <name>] [--language <en|zh-CN>] [--format markdown|json]`

Prints a local planning/execution handoff snapshot to stdout. It reads the local planning ledger under `$ADP_HOME`. The default output remains English Markdown. `--language zh-CN` emits Simplified Chinese Markdown, and `--language` applies to Markdown only.

With `--format json`, the command emits a machine-readable, read-only handoff snapshot with workspace, total task count, phases, task counts, tasks, priority-sorted next work, phase evidence, and recent runtime session evidence when local JSONL runtime events and session data exist. JSON is for cross-tool parsing and must not become a separate state store.

When local JSONL runtime events and session data exist, Markdown and JSON reports include recent runtime session evidence derived from `$ADP_HOME/logs/events.jsonl`. This evidence is for inspection and handoff only.

The command is read-only. It must not append events, mutate task state, mutate phase state, create runtime directories, start agents, run Git, push, infer acceptance, close tasks, resume provider-native conversations, or write report files into the real project root.

### `adp tasks next [--workspace <name>] [--limit <n>] [--format text|json]`

Prints a compact local next-work snapshot to stdout. It reads the workspace planning ledger under `$ADP_HOME`, selects tasks with `ready`, `in_progress`, or `review` status, and sorts candidates by priority plus stable local tie-breakers so terminal users and sub-agents can choose follow-up work without parsing the full progress report.

Text is the default format and should be easy to scan in a terminal. `--limit <n>` caps candidates, defaults to 5, and accepts `0` for an untruncated snapshot. JSON output is for local cross-tool parsing and includes stable fields for workspace, planning source, generated timestamp, total task count, eligible candidate count, status counts, requested limit, ordered `candidates`, and a singular `next` first-candidate value when work is available.

The command is read-only. It must not claim tasks, mutate task status, change owners or leases, clear blockers, mutate phases, append events, create runtime directories, start agents, run Git, push, infer acceptance, close tasks, resume provider-native conversations, write files into the real project root, sync with hosted trackers, or maintain JSON as a second planning store.

### `adp plan preview|apply [--workspace <name>] --file <path|-> [--format text|json]`

Accepts structured local YAML/JSON planning input with phases and tasks. `preview` renders the proposed import to stdout without creating planning files or directories. `apply` explicitly writes the validated batch to `$ADP_HOME/workspaces/<workspace>/planning`.

JSON output is for inspection and local cross-tool parsing only. It must not become a second planning store. Plan intake must not split free-text natural language into tasks, sync hosted trackers, run Git, start agents, infer acceptance, claim or close tasks automatically, write planning files into the real project root, or mutate runtime state.

### `adp runtime prune [--older-than <duration>] [--include-kept] [--dry-run]`

Scans direct child directories under `$ADP_RUNTIME_DIR`. A directory becomes a prune candidate only when it contains a current-version, self-consistent `.adp-runtime.yaml` with `generated_by: adp`, non-empty workspace and session IDs, an absolute `project_root`, a matching `runtime_root`, and a valid `created_at`. The command removes ADP-owned runtime directories older than `--older-than`, skips `keep: true` by default unless `--include-kept` is passed, reports candidates without deleting when `--dry-run` is set, skips incompatible or self-inconsistent manifests, and never removes a target derived from the manifest project root.

### `adp run <agent> [--workspace <name>] [--profile <profile>] [--keep-runtime] [-- <agent-args>...]`

Resolves the workspace, renders adapter files, builds the runtime overlay, launches the agent, logs start/finish events, passes through streams, and returns the agent exit code.

Workspace resolution order:

- Explicit `--workspace`.
- `ADP_WORKSPACE`.
- Match the current directory to a registered project root, using the longest matching project root when workspaces are nested.

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

The repository started with a serial Contract Scaffold, then parallel work was split by disjoint paths.

Parallel slices:

- CLI/Foundation: `cmd/`, `internal/cli/`, root wiring.
- Workspace Registry: `internal/workspace/`, `internal/schema/`.
- Workspace Diagnostics: `internal/workspace/diagnostics*`, with CLI wiring in `internal/cli/`.
- Runtime Overlay: `internal/runtime/`, `internal/overlay/`.
- Adapter Layer: `internal/adapters/`.
- Runner/Shell/Events: `internal/runner/`, `internal/shell/`, `internal/events/`.
- Session History: `internal/sessions/`, with CLI wiring in `internal/cli/`.
- Docs/Examples: `README*`, `docs/`, `examples/`.
- Integration QA: `test/`, CI workflows, quality checks.

Rules:

- Each worker owns its path slice by default.
- Public contract changes must be explicit and validated across dependent packages.
- `LICENSE` and `COMMERCIAL*` are maintained by project maintainers.
- Hand-written code files must stay below 700 lines.
- Documentation must keep English default files and Simplified Chinese counterparts aligned.

## 11. Validation Gates

Required local checks:

```bash
scripts/check-all.sh
```

The aggregate gate runs the deterministic smoke and repository checks:

```bash
scripts/runtime-smoke.sh --fake
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

- `adp init` creates local ADP home.
- `adp workspace add` creates a workspace config without touching the project root.
- `adp workspace list` and `adp workspace show` expose registered workspace details.
- `adp workspace doctor` reports healthy workspaces and returns a non-zero exit code for error-level diagnostics.
- `adp doctor` exposes the same diagnostics globally.
- `adp workspace remove` and `adp workspace rename` modify only ADP workspace registry data.
- `adp env` prints shell-safe exports for a kept runtime overlay.
- `adp shell-hook` prints a deterministic shell function for `sh`, `bash`, and `zsh`.
- `adp completion` prints deterministic completion for `bash` and `zsh`, and `adp completion values` returns local workspace and profile candidates.
- The P16 command metadata drift check proves the local command inventory, usage text, dispatch wiring, and bash/zsh completion remain aligned without adopting a new CLI framework.
- `adp version` reports the CLI build identity.
- `adp events list` prints filtered run history from JSONL events.
- `adp sessions list`, `adp sessions show`, and `adp sessions restore-plan` expose local session history and read-only restore planning derived from JSONL events.
- `adp progress report [--workspace <name>] [--language <en|zh-CN>] [--format markdown|json]` prints a Markdown planning/execution report to stdout by default, emits a read-only JSON handoff snapshot with `--format json`, includes recent local runtime session evidence when JSONL event/session data exists, and leaves planning state, Git state, runtime state, event logs, and the real project root unchanged.
- `adp tasks next [--workspace <name>] [--limit <n>] [--format text|json]` prints a compact prioritized next-work snapshot to stdout, exposes a stable JSON contract for local tools, and leaves task state, phase state, Git state, runtime state, event logs, hosted services, and the real project root unchanged.
- `adp plan preview/apply [--workspace <name>] --file <path|-> [--format text|json]` accepts structured local planning input; preview stays read-only, apply writes only the local planning ledger under `$ADP_HOME`, and failed apply leaves no partial phase, task, or progress state.
- `adp runtime prune` reports and removes only current-version, self-consistent ADP-owned runtime directories.
- `adp run codex` and `adp run claude` build runtime overlays, and `--task <task-id>` binds runtime sessions to workspace task state.
- `examples/basic-workspace` remains a valid local workspace reference with bilingual Markdown prompt and memory files.
- Fake agent tests can assert cwd, env, generated files, symlinks, args, exit code, logs, and cleanup.
- The real project directory must not gain `AGENTS.md`, `CLAUDE.md`, `.codex/`, `.claude/`, `planning/`, `tasks.yaml`, `phases.yaml`, `progress.jsonl`, or report export files.

## 12. Next Work

Next work is prioritized by how much it improves ADP's terminal-first runtime and workspace management loop without drifting into hosted project management or a dashboard.

- P0 completed: Task and Progress Manager MVP. Store workspace-scoped task state under `$ADP_HOME/workspaces/<workspace>/planning`, expose `adp tasks` and `adp progress`, and validate it with a task-manager smoke.
- P1 completed: Runtime task binding. Add `adp run --task <task-id>`, inject task context into runtime env and generated adapter instructions, and connect task IDs to events and sessions.
- P2 completed: Early preview hardening. Dynamic workspace/profile completion, global `adp doctor`, version output, CI for `scripts/check-all.sh`, and release packaging notes are covered by the aggregate gate and runtime smoke.
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
- P19 planned: add runtime acceptance for workspace rename/remove and non-interactive `adp enter`.
- P20 planned: cover `adp plan --file -` stdin intake paths.
- P21 planned: split taskstore core files by model, persistence, events, ranking, and lifecycle responsibilities.
- P22 planned: normalize the Phase 1 English default roadmap and Simplified Chinese counterpart at the content level.
- P23 planned: add non-blocking line pressure audit tooling before files approach the hard 700-line cap.
- P3/P4/P5/P6/P7/P8/P9/P10/P11/P12/P13/P14/P15/P16/P17/P18 non-goals: no Web dashboard, SaaS tracker, cloud sync, hosted orchestration, hosted tracker sync, automatic Git execution, automatic claim/done/phase acceptance, provider-native conversation resume, remote issue-service integration, project-root report or planning exports, or hosted tracker semantics.

Each phase slice must be validated, committed, and pushed before the next slice starts.
