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
- Support `adp init`, `adp workspace add/list/show/doctor/remove/rename`, `adp doctor`, `adp env`, `adp shell-hook`, `adp completion`, `adp completion values`, `adp version`, `adp events list`, `adp sessions list/show`, `adp runtime prune`, `adp enter`, and `adp run`.
- Write a local JSONL event log for future replay, session restore, and multi-agent orchestration.

Phase 1 explicitly excludes:

- Web UI or dashboard.
- Cloud sync or SaaS features.
- Graphical multi-agent orchestration.
- Heavy permission sandboxing.
- Full Windows support. Interfaces should leave room for it, but Linux/macOS are the first target.
- Mandatory kernel overlayfs. The MVP defaults to portable symlink materialization, with bind mount and overlayfs reserved for later backends.

## 2. Technical Stack

Primary language: Go.

Dependencies should stay small and stable:

- CLI command wiring can use Go standard library first; add a CLI framework only when the command surface justifies it.
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

- `internal/paths`: resolve `ADP_HOME`, runtime temp roots, workspace paths, and log paths.
- `internal/schema`: define and validate the unified workspace schema.
- `internal/workspace`: create, load, and manage the workspace registry.
- `internal/overlay`: materialize project files and generated files into a runtime root.
- `internal/runtime`: orchestrate workspace config, adapter output, overlay lifecycle, runtime env, manifest handling, and runtime pruning.
- `internal/adapters`: adapter contracts, registry, concrete Codex/Claude adapters, and shared rendering.
- `internal/runner`: execute external commands with controlled cwd, env, and streams.
- `internal/shell`: implement `adp enter`, shell export rendering, parent-shell hook rendering, and shell completion rendering.
- `internal/sessions`: aggregate local events into session history summaries and detail views.
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
- Runtime pruning only deletes ADP-owned runtime directories that contain `.adp-runtime.yaml` with `generated_by: adp`.
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

Checks one workspace, or all registered workspaces when no name is supplied. Diagnostics cover config loading and validation, project root reachability, prompt, memory, MCP, profile file references, path escapes, and agent command defaults. Error-level diagnostics should be terminal-readable and return a non-zero process exit code.

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

### `adp version`

Prints the local CLI build identity. Development builds may report `dev`; preview release binaries should inject version, commit, and build-date values with Go linker flags.

### `adp events list [--workspace <name>] [--session <session-id>] [--type <event-type>] [--limit <n>]`

Reads `$ADP_HOME/logs/events.jsonl`, filters JSONL events, and prints recent matching events in a stable terminal table. Corrupted event log lines are reported with line numbers instead of being ignored.

### `adp sessions list [--workspace <name>] [--agent <agent>] [--limit <n>]`

Groups local event log records by `session_id` and prints recent session summaries. Empty session IDs are ignored. Workspace and agent filters should be applied before limiting, and the selected sessions should remain in chronological session-start order for terminal readability.

### `adp sessions show <session-id>`

Prints the ordered events for one session. Missing sessions return a clear not-found error. The command is read-only and derives its data from the local JSONL event log.

### `adp runtime prune [--older-than <duration>] [--include-kept] [--dry-run]`

Scans `$ADP_RUNTIME_DIR` for direct child directories containing an ADP runtime manifest. It removes stale ADP-owned runtime directories, never the real project root. `--dry-run` reports candidates without deleting, and kept runtimes require `--include-kept`.

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
- `adp sessions list/show` are read-only views over the same local log and must not create, mutate, or delete runtime state.

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
- `adp version` reports the CLI build identity.
- `adp events list` prints filtered run history from JSONL events.
- `adp sessions list` and `adp sessions show` expose local session history derived from JSONL events.
- `adp runtime prune` reports and removes only ADP-owned runtime directories.
- `adp run codex` and `adp run claude` build runtime overlays, and `--task <task-id>` binds runtime sessions to workspace task state.
- `examples/basic-workspace` remains a valid local workspace reference with bilingual Markdown prompt and memory files.
- Fake agent tests can assert cwd, env, generated files, symlinks, args, exit code, logs, and cleanup.
- The real project directory must not gain `AGENTS.md`, `CLAUDE.md`, `.codex/`, `.claude/`, `planning/`, `tasks.yaml`, or `progress.jsonl`.

## 12. Next Work

Next work is prioritized by how much it improves ADP's terminal-first runtime and workspace management loop without drifting into hosted project management or a dashboard.

- P0 completed: Task and Progress Manager MVP. Store workspace-scoped task state under `$ADP_HOME/workspaces/<workspace>/planning`, expose `adp tasks` and `adp progress`, and validate it with a task-manager smoke.
- P1 completed: Runtime task binding. Add `adp run --task <task-id>`, inject task context into runtime env and generated adapter instructions, and connect task IDs to events and sessions.
- P2 completed: Early preview hardening. Dynamic workspace/profile completion, global `adp doctor`, version output, CI for `scripts/check-all.sh`, and release packaging notes are covered by the aggregate gate and runtime smoke.
- P3 Phase Gate MVP completed: Project planning and execution progress management now has phase records, task claim and owner records, acceptance or gate records, commit records, push records, and task-manager smoke coverage. Next P3 hardening should add stricter lifecycle guards, leases, and conflict handling before broader runtime standards.
- P3 non-goals: no Web dashboard, SaaS tracker, cloud sync, hosted orchestration, or remote issue-service integration.

Each phase slice must be validated, committed, and pushed before the next slice starts.
