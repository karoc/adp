# Release Checklist

Simplified Chinese: [release-checklist.zh-CN.md](release-checklist.zh-CN.md)

This checklist defines the local release gate for ADP. It keeps release validation aligned with the project boundary: ADP is a terminal-first, local-first Agent Runtime Environment and Agent Workspace Manager.

The release gate verifies ADP's own runtime, CLI, workspace, diagnostics, documentation, and repository hygiene. It does not turn release validation into a hosted service check, Web UI check, SaaS deployment check, or remote provider certification process.

For early preview artifact layout and CLI build commands, see [release-packaging.md](release-packaging.md).

## Required Gate

Run the unified gate before handoff, commit, push, or release candidate tagging:

```bash
scripts/check-all.sh
```

The script can be called from any current directory. It resolves the repository root from its own location before running checks. CI should call this same script instead of maintaining a separate release gate path.

`scripts/check-all.sh` remains the aggregate gate even when an individual smoke is internally split for maintainability.

The required gate runs these checks in order:

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

## Phase Slice Discipline

For normal development handoff, a phase slice is not complete when implementation stops. It is complete only after:

- The relevant acceptance commands have passed.
- The phase has a recorded gate result.
- The accepted changes have been committed.
- The commit has been pushed to the configured remote branch.
- The next phase has not been mixed into the same commit.

P3 phase gate work turns this discipline into local records under `$ADP_HOME/workspaces/<workspace>/planning`. Release evidence should include both the positive lifecycle path and the local guards that reject out-of-order phase evidence.

## Gate Coverage

`scripts/runtime-smoke.sh --fake` builds the current `cmd/adp` binary into a temporary directory and runs the deterministic fake-agent runtime acceptance path. It uses temporary `ADP_HOME`, `ADP_RUNTIME_DIR`, fake agent binaries, and a temporary project root.

P17 may split shared helpers and fake diagnostics/session/prune slices into helper files under `scripts/`. That split is an implementation detail for maintainability under the 700-line file cap; callers still run `scripts/runtime-smoke.sh --fake`, and the aggregate release gate still runs it through `scripts/check-all.sh`.

The fake runtime smoke verifies:

- Runtime overlay creation.
- Runtime environment variables.
- Task-bound runtime context through `adp run --task <task-id>`.
- Codex and Claude adapter launch paths through fake binaries.
- Event log writes.
- Session history queries.
- Read-only session restore-plan output, including original agent arguments and no event-log mutation from inspection.
- Workspace diagnostics through `adp workspace doctor` and `adp doctor`.
- Runtime parent safety diagnostics: fake smoke covers project-root overlap rejection, while Go tests cover filesystem-root, containing-project-root, symlink, and non-directory risks.
- Agent command/profile diagnostics: fake smoke covers reserved project-root paths, adapter default command fallback, inline command arguments, missing non-default profiles, escaping profile symlinks, and unknown enabled agent entries; Go tests cover path-like missing or non-executable command wrappers and ambiguous profile files.
- Shell export rendering.
- Bash and zsh completion rendering.
- Dynamic completion value endpoints for local workspaces and profiles.
- Global `adp doctor [workspace]` diagnostics.
- `adp version` and `adp --version` output.
- ADP-owned runtime pruning.
- Runtime manifest compatibility checks that keep prune limited to current-version, self-consistent ADP runtime directories.
- Protection against polluting the real project root with runtime artifacts or planning files.

`scripts/example-workspace-smoke.sh` builds the current `cmd/adp` binary, copies `examples/basic-workspace` into a temporary `ADP_HOME`, rewrites the copied `project.root` to a temporary project, and verifies `adp init`, `workspace doctor`, `workspace show`, `env --cd`, fake Codex runtime launch, local events, sessions, and restore-plan output against that copied example.

The example workspace smoke verifies:

- The published example can be copied without depending on repository-local state.
- The example workspace schema remains compatible with the current CLI.
- A temporary project root can be linked into a kept runtime overlay.
- Fake local agent execution through the copied example records session history and supports read-only restore planning.
- Example documentation and release claims stay connected to an executable path.

`scripts/task-manager-smoke.sh` remains the public entry point for workspace-local task, phase, and progress report runtime acceptance. It builds the current `cmd/adp` binary, creates a temporary workspace, exercises `adp tasks add/list/next/show/update/claim/release/block/done`, `adp phase add/list/show/start/accept/commit/push`, `adp progress`, and `adp progress report`, then verifies that planning files are written under `$ADP_HOME/workspaces/<workspace>/planning`, next-work/report generation is read-only, and no planning or report artifacts are written into the real project root.

P9 may move shared smoke helpers and the JSON report validator into helper files under `scripts/`. That split is an implementation detail for maintenance and hardening; callers still run `scripts/task-manager-smoke.sh`, and the release gate still runs it through `scripts/check-all.sh`.

The phase gate smoke path covers phase records, task claim ownership with leases, owner-checked release, task phase validation, acceptance or gate records, commit records, push records, lifecycle ordering guards, and project-root pollution protection. Go tests additionally cover planning lock behavior, claim conflicts, lease expiry, terminal-task claim rejection, failed acceptance, and failed push semantics. Do not add placeholder assertions for commands that do not exist yet.

`scripts/plan-intake-smoke.sh` builds the current `cmd/adp` binary, creates a temporary workspace, and verifies `adp plan preview` and `adp plan apply` with structured YAML input. It proves preview stays read-only, apply writes only the local planning ledger under `$ADP_HOME/workspaces/<workspace>/planning`, JSON output remains an inspection format, invalid input on a fresh workspace leaves no planning directory, staging failures leave no partial phase/task/progress state, and no runtime, Git, event-log, or real project-root side effects occur.

`go test -count=1 ./...` verifies the full Go test suite without using cached test results.

`go vet ./...` runs Go static checks.

`scripts/check-file-lines.sh` enforces the project rule that code files stay at or below 700 physical lines. It checks tracked files and non-ignored untracked files.

`scripts/check-docs-bilingual.sh` enforces the documentation pairing rule for tracked Markdown files and non-ignored untracked Markdown files. English is the default document, and maintained Markdown files need Simplified Chinese counterparts using `*.zh-CN.md`.

`git diff --check` catches whitespace errors in the current diff.

## Optional Real CLI Evidence

Real Codex and Claude CLI checks are not part of the default gate. They are opt-in release evidence because local installations, credentials, model availability, network access, and interactive behavior vary by operator environment.

Run the lightweight real Codex check only when the local Codex CLI is intentionally part of the release evidence:

```bash
ADP_SMOKE_REAL_CODEX=1 scripts/runtime-smoke.sh --real-codex
```

Run the lightweight real Claude check only when the local Claude CLI is intentionally part of the release evidence:

```bash
ADP_SMOKE_REAL_CLAUDE=1 scripts/runtime-smoke.sh --real-claude
```

The real CLI smoke confirms that the external command exists and that a lightweight `--version` or `--help` invocation completes. It does not prove that a full interactive agent session, provider credentials, account quota, model selection, external tool permission, or network path is ready.

When real CLI evidence is collected, record:

- The command that was run.
- The Codex or Claude CLI version when available.
- The operating system and shell.
- Any environment overrides such as `ADP_SMOKE_CODEX_BIN` or `ADP_SMOKE_CLAUDE_BIN`.
- Whether a separate manual interactive session was completed.

## Failure Triage

If `scripts/runtime-smoke.sh --fake` fails, inspect the reported step first. The fake smoke is the highest-signal check for runtime overlay behavior, runtime manifest fields, adapter launch paths, local event history, session aggregation, and project-root pollution.

If a task-bound runtime smoke step fails, inspect workspace resolution, task lookup under `$ADP_HOME/workspaces/<workspace>/planning`, generated task context in `AGENTS.md` or `CLAUDE.md`, `ADP_TASK_ID` in the runtime environment, and task IDs in events and sessions.

If a diagnostics step fails, compare `adp doctor [workspace]` with `adp workspace doctor [name]` and inspect the local workspace registry, project root, `ADP_RUNTIME_DIR`, referenced prompts, memory files, MCP files, profile files, and agent command settings. For runtime parent failures, confirm `ADP_RUNTIME_DIR` is not the filesystem root, not equal to the project root, not inside the project root, not a parent directory containing the project root, not a file, and not an unintended symlink. For agent command/profile warnings, check whether the enabled agent has an adapter default, whether `command` contains inline arguments that should be passed after `--` or moved into a wrapper, whether path-like command wrappers exist and are executable, whether non-default profile files are missing or duplicated, and whether profile files escape the workspace through symlinks or path traversal.

If a completion value step fails, inspect local workspace name discovery under `$ADP_HOME/workspaces`, `ADP_WORKSPACE` or `--workspace` resolution, workspace agent profiles, and files under the workspace `profiles/` directory. Completion value endpoints must stay read-only and local.

If a version step fails, inspect the CLI build variables in `internal/cli` and the release `-ldflags` described in [release-packaging.md](release-packaging.md). Development builds may print `dev`; packaged preview binaries should inject version, commit, and build date.

If `scripts/example-workspace-smoke.sh` fails, inspect whether the copied `examples/basic-workspace/workspace.yaml` still matches the current schema and whether `adp env <workspace> --cd` still produces a kept runtime with project-file symlinks.

If `scripts/task-manager-smoke.sh` fails, inspect task CLI parsing, workspace resolution, task storage under `planning/`, next-work JSON selection, helper wiring, JSON report validation, next-work/report read-only checks, and project-root pollution checks.

If `scripts/plan-intake-smoke.sh` fails, inspect plan input parsing, plan preview read-only behavior, explicit apply writes under `$ADP_HOME`, planning batch rollback behavior, JSON output shape, and the project-root/runtime/Git side-effect checks.

If a phase-gate smoke step fails, inspect phase record storage, task owner state, claim lease parsing, owner-checked release, append-only progress events, acceptance result recording, commit hash recording, push result recording, and lifecycle ordering. The expected state must remain local under `$ADP_HOME`; failures should not be fixed by writing planning artifacts into the project root.

If `go test -count=1 ./...` fails, narrow the failing package and rerun that package before making changes:

```bash
go test -count=1 ./internal/workspace
go test -count=1 ./internal/cli
go test -count=1 ./test/e2e
```

If `go vet ./...` fails, treat it as a code-quality gate and fix the reported package before rerunning the full gate.

If `scripts/check-file-lines.sh` fails, split the reported code file before adding more behavior. Do not bypass the 700-line limit with generated-looking manual files or unrelated formatting churn. If the reported file is a scratch file, remove it or ignore it deliberately.

If `scripts/check-docs-bilingual.sh` fails, add the missing English default or Simplified Chinese counterpart. New Markdown files should follow the default English path plus `*.zh-CN.md` counterpart pattern. If the reported Markdown is a local note, remove it or ignore it deliberately.

If `git diff --check` fails, remove trailing whitespace or conflict markers from the reported files.

## Manual Release Checks

Before a release candidate is announced, an operator should also confirm:

- `git status --short --branch` shows only intentional changes before commit and a clean tree after commit.
- `.envrc` and `mvp.md` remain ignored and uncommitted.
- Repository-local Git identity is not configured with `user.name` or `user.email`.
- The license files and PolyForm Noncommercial positioning were not changed unintentionally.
- Packaged CLI artifacts were built with version, commit, and build-date ldflags and `adp version` reports the expected values.
- README and focused docs describe the current CLI surface without Web, UI, SaaS, cloud sync, hosted tracker, hosted orchestration, automatic Git execution, automatic task closure, provider-native resume, or project-root report export drift.
- Active development phases have local evidence for acceptance, commit, and push before the next phase starts.
- Any claimed real-agent compatibility has matching opt-in real CLI evidence and, when needed, manual interactive acceptance notes.

## Out Of Scope

The release gate does not validate:

- Provider accounts or billing.
- Remote model availability.
- External network reliability.
- Real interactive Codex or Claude session quality.
- User-specific shell startup files.
- Hosted deployment, SaaS operations, dashboards, Web UI behavior, hosted trackers, automatic Git execution, automatic task closure, provider-native resume, or project-root report export behavior.

Those checks belong to operator-specific acceptance notes, not the default local release gate.
