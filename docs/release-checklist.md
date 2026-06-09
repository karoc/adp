# Release Checklist

Simplified Chinese: [release-checklist.zh-CN.md](release-checklist.zh-CN.md)

This checklist defines the local release gate for ADP. It keeps release validation aligned with the project boundary: ADP is a terminal-first, local-first Agent Runtime Environment and Agent Workspace Manager.

The release gate verifies ADP's own runtime, CLI, workspace, diagnostics, documentation, and repository hygiene. It does not turn release validation into a hosted service check, Web UI check, SaaS deployment check, or remote provider certification process.

For early preview artifact layout and CLI build commands, see [release-packaging.md](release-packaging.md). For operator failure triage, see [release-troubleshooting.md](release-troubleshooting.md). For adjacent-tool scope calibration, see [comparable-tools.md](comparable-tools.md).

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

## Release Package Contents

For preview artifacts, inspect the package before publishing it. The package should include the target-platform `adp` binary, `README.md`, `README.zh-CN.md`, `LICENSE`, `COMMERCIAL.md`, `COMMERCIAL.zh-CN.md`, `docs/release-packaging.md`, `docs/release-packaging.zh-CN.md`, `docs/release-evidence.md`, `docs/release-evidence.zh-CN.md`, and a release evidence note or release note that records the commit, version, target platform, gate result, and checksum.

The package must preserve the PolyForm Noncommercial and source-available positioning. Noncommercial redistribution must keep the license text, required notices, and attribution to ADP and the copyright holder. Any commercial use requires separate paid authorization; do not describe a preview package as granting commercial rights.

The package must exclude local or sensitive operator state, including `.envrc`, `mvp.md`, `$ADP_HOME`, `$ADP_RUNTIME_DIR`, runtime overlays, event logs, session logs, task or phase state, credentials, tokens, account identifiers, private prompts, and machine-specific shell startup files.

## Phase Slice Discipline

For normal development handoff, a phase slice is not complete when implementation stops. It is complete only after:

- The relevant acceptance commands have passed.
- The phase has a recorded gate result.
- The accepted changes have been committed.
- The commit has been pushed to the configured remote branch.
- Commit and push evidence have been recorded in the local phase ledger before any next phase starts.
- The next phase has not been mixed into the same commit.

P3 phase gate work turns this discipline into local records under `$ADP_HOME/workspaces/<workspace>/planning`. Release evidence should include both the positive lifecycle path and the local guards that reject out-of-order phase evidence.

## Gate Coverage

`scripts/runtime-smoke.sh --fake` builds the current `cmd/adp` binary into a temporary directory and runs the deterministic fake-agent runtime acceptance path. It uses temporary `ADP_HOME`, `ADP_RUNTIME_DIR`, fake agent binaries, and a temporary project root.

P17 may split shared helpers and fake diagnostics/session/prune slices into helper files under `scripts/`. That split is an implementation detail for maintainability under the 700-line file cap; callers still run `scripts/runtime-smoke.sh --fake`, and the aggregate release gate still runs it through `scripts/check-all.sh`.

The fake runtime smoke verifies:

- Runtime overlay creation.
- Runtime environment variables.
- Task-bound runtime context through `adp run <agent> --task <task-id>`.
- Codex and Claude adapter launch paths through fake binaries.
- Event log writes.
- Session history queries.
- Read-only session restore-plan output, including original agent arguments and no event-log mutation from inspection.
- Workspace diagnostics through `adp workspace doctor` and `adp doctor`.
- Workspace rename/remove lifecycle checks that prove only temporary ADP registry data is mutated while the real project root entry snapshot and runtime entry count remain unchanged.
- Controlled `adp enter` child-shell execution through a fake `SHELL`, including runtime env/cwd, project symlinks, default cleanup, `--keep-runtime`, unchanged project-root entries, and content-level no event-log mutation.
- Runtime parent safety diagnostics: fake smoke covers project-root overlap rejection, while Go tests cover filesystem-root, containing-project-root, symlink, and non-directory risks.
- Agent command/profile diagnostics: fake smoke covers reserved project-root paths, adapter default command fallback, inline command arguments, missing non-default profiles, escaping profile symlinks, and unknown enabled agent entries; Go tests cover path-like missing or non-executable command wrappers and ambiguous profile files.
- Shell export rendering.
- Bash and zsh completion rendering.
- Dynamic completion value endpoints for local agents, workspaces, and profiles.
- Global `adp doctor [workspace]` diagnostics.
- `adp version` and `adp --version` output.
- ADP-owned runtime pruning.
- Runtime manifest compatibility checks that keep prune limited to current-version, self-consistent ADP runtime directories.
- Protection against polluting the real project root with runtime artifacts or planning files.

P25 splits bash and zsh completion renderers into shell-specific implementation files to remove line pressure from `internal/shell/completion.go`. This is an internal maintenance boundary: `adp completion`, bash/zsh output semantics, metadata drift checks, dynamic value endpoints, and the default fake runtime smoke remain the release evidence. It does not add interactive completion simulation or new shell support.

`scripts/runtime-audit-smoke.sh` builds the current `cmd/adp` binary, uses temporary `ADP_HOME`, `ADP_RUNTIME_DIR`, fake agent binaries, and a temporary project root, and verifies the broad runtime audit matrix without real provider CLIs or network access.

The runtime audit smoke verifies:

- CLI discoverability and command metadata drift coverage.
- Runtime entry points, events, sessions, restore-plan, runtime pruning, and project-root pollution guards.
- Workspace lifecycle, diagnostics, task manager, phase gate, plan intake, and progress report surfaces through the current CLI.
- Local-first boundaries: no hosted tracker sync, no automatic Git execution, no automatic task closure, no provider-native resume, and no project-root planning or report exports.

`scripts/runtime-context-smoke.sh` builds the current `cmd/adp` binary, uses temporary `ADP_HOME`, `ADP_RUNTIME_DIR`, fake agent binaries, and a temporary project root, then verifies the launch-time runtime context without real provider CLIs or network access.

The runtime context smoke verifies:

- Generated Codex and Claude instruction files.
- Adapter metadata files and selected workspace profiles.
- Base prompt, shared memory, MCP references, task metadata, and runtime environment variables.
- Local event/session evidence, workspace diagnostics, and project-root cleanliness.
- Local-first boundaries: no hosted services, no automatic Git execution, no provider-native resume, and no project-root report or planning exports.

`scripts/release-readiness-smoke.sh` builds the current `cmd/adp` binary, uses temporary `ADP_HOME`, `ADP_RUNTIME_DIR`, a temporary project root, and a fake Git tripwire, then verifies release-readiness invariants that are independent of real provider CLIs.

The release readiness smoke verifies:

- Phase acceptance, commit, and push evidence can be recorded through the CLI.
- Phase gate state reaches `plan_next_phase` only after accepted, committed, and pushed evidence exists.
- Phase evidence commands do not execute Git side-effect commands such as `git commit`, `git push`, `git pull`, `git fetch`, `git clone`, or `git ls-remote`.
- The default release path remains local, deterministic, and independent of real Codex or Claude CLIs.

`scripts/release-rehearsal-smoke.sh` copies the current non-ignored repository files into a temporary clean workspace, builds a preview binary with release ldflags, verifies copied docs and file limits, bootstraps the copied example workspace with isolated `ADP_HOME` and `ADP_RUNTIME_DIR`, and checks phase evidence recording with a fake Git tripwire.

The release rehearsal smoke verifies:

- The documented preview build/version path can produce a binary whose `adp version` reports injected version, commit, and build date values.
- The copied docs and code files still satisfy the bilingual-document and 700-line gates from a clean temporary workspace.
- The published example workspace can bootstrap against a temporary project without relying on the developer's local `$ADP_HOME` or runtime state.
- Release phase evidence remains local and does not execute Git side-effect commands.

`scripts/release-artifact-smoke.sh` builds preview artifacts in a temporary source tree, assembles a local release package, verifies checksums, checks package include/exclude boundaries, installs the packaged binary into a temporary `PATH`, runs a provider-free first-run rehearsal, and verifies a source archive build without `.git` by using an explicit commit value.

The release artifact smoke verifies:

- Target-platform artifacts can be built with release ldflags and report the injected version, commit, and build date.
- The package includes the required binary, license, commercial notice, README, release packaging, and release evidence files.
- The package excludes `.envrc`, `mvp.md`, local ADP state, runtime overlays, logs, task state, credentials, and machine-specific shell startup files.
- Checksums are generated for packaged artifacts before install rehearsal.
- The installed binary runs from the package path with temporary `ADP_HOME`, temporary `ADP_RUNTIME_DIR`, fake Codex, local events, local sessions, and no project-root pollution.
- Source archive builds without `.git` remain valid when `COMMIT` is explicit.

`scripts/release-operator-drill-smoke.sh` copies the repository into a no-`.git` operator source tree, verifies the release packaging docs expose the required source archive, build, checksum, and install commands, syntax-checks release scripts from that source tree, builds a release binary with explicit commit metadata, verifies the checksum, installs the artifact into a temporary `PATH`, and runs a provider-free handoff sequence with fake Codex and a fake Git tripwire.

The release operator drill smoke verifies:

- The release path works from a clean source form without relying on repository `.git` metadata.
- The installed binary can initialize ADP state, add a workspace, start a phase, add a task, run fake Codex with task context, and record accept, commit, and push evidence locally.
- The operator handoff sequence reaches `plan_next_phase` without executing Git side-effect commands.
- Temporary ADP state and runtime state stay outside the real project root.

`scripts/install-onboarding-smoke.sh` builds a local binary, installs ADP into a temporary `GOBIN`, verifies the installed binary is first on `PATH`, and runs the first-use operator path with temporary ADP state, a temporary project root, fake Codex, fake Claude guard, local events, local sessions, planning diagnostics, progress JSON, Git side-effect guards, and project-root pollution checks.

The install onboarding smoke verifies:

- A new operator can validate the installed binary before registering a workspace.
- Temporary `ADP_HOME` and `ADP_RUNTIME_DIR` are enough for the first local workspace.
- Task-bound fake Codex execution records local event, session, progress, and plan-doctor evidence.
- Missing real Codex or Claude CLIs do not block deterministic onboarding validation.
- The first-use path does not execute Git side-effect commands or write ADP artifacts into the real project root.

`scripts/example-workspace-smoke.sh` builds the current `cmd/adp` binary, copies `examples/basic-workspace` into a temporary `ADP_HOME`, rewrites the copied `project.root` to a temporary project, and verifies `adp init`, `workspace doctor`, `workspace show`, `env --cd`, fake Codex runtime launch, local events, sessions, and restore-plan output against that copied example.

The example workspace smoke verifies:

- The published example can be copied without depending on repository-local state.
- The example workspace schema remains compatible with the current CLI.
- A temporary project root can be linked into a kept runtime overlay.
- Fake local agent execution through the copied example records session history and supports read-only restore planning.
- Example documentation and release claims stay connected to an executable path.

`scripts/task-manager-smoke.sh` remains the public entry point for workspace-local task, phase, planning doctor, and progress report runtime acceptance. It builds the current `cmd/adp` binary, creates a temporary workspace, exercises `adp tasks add/list/next/show/update/claim/release/block/done`, `adp phase add/list/show/status/start/accept/commit/push`, `adp plan doctor`, `adp progress`, and `adp progress report`, then verifies that planning files are written under `$ADP_HOME/workspaces/<workspace>/planning`, next-work/plan-doctor/report generation is read-only, error-level planning doctor diagnostics return exit code `2`, and no planning or report artifacts are written into the real project root.

P9 may move shared smoke helpers and the JSON report validator into helper files under `scripts/`. That split is an implementation detail for maintenance and hardening; callers still run `scripts/task-manager-smoke.sh`, and the release gate still runs it through `scripts/check-all.sh`.

The phase gate smoke path covers phase records, task claim ownership with leases, owner-checked release, task phase validation, acceptance or gate records, commit records, push records, read-only phase gate status snapshots, read-only planning ledger diagnostics, lifecycle ordering guards, and project-root pollution protection. It asserts that a later phase cannot start before the earlier phase has successful pushed evidence, and that it can start after that evidence is recorded. Go tests additionally cover planning lock behavior, claim conflicts, lease expiry, terminal-task claim rejection, failed acceptance, failed push semantics, explicit phase ordering, and planning doctor invariants. Do not add placeholder assertions for commands that do not exist yet.

`scripts/plan-intake-smoke.sh` builds the current `cmd/adp` binary, creates a temporary workspace, and verifies `adp plan preview` and `adp plan apply` with structured YAML input from both files and stdin through `--file -`. It proves preview stays read-only, apply writes only the local planning ledger under `$ADP_HOME/workspaces/<workspace>/planning`, JSON output remains an inspection format, invalid input on a fresh workspace leaves no planning directory, staging failures leave no partial phase/task/progress state, and no runtime, Git, event-log, or real project-root side effects occur.

`go test -count=1 ./...` verifies the full Go test suite without using cached test results.

`go vet ./...` runs Go static checks.

`scripts/check-file-lines.sh` enforces the project rule that code files stay at or below 700 physical lines. It checks tracked files and non-ignored untracked files.

`scripts/check-file-lines.sh --audit` is a non-blocking line pressure report for planning future splits before files approach the hard cap. It reports files at or above `LINE_PRESSURE_WARN_LINES`, defaults to 600 lines, and exits zero. The audit is not part of `scripts/check-all.sh` by default and must not replace the hard line-count gate.

`scripts/check-docs-bilingual.sh` enforces the documentation pairing rule for tracked Markdown files and non-ignored untracked Markdown files. English is the default document, and maintained Markdown files need Simplified Chinese counterparts using `*.zh-CN.md`.

`git diff --check` catches whitespace errors in the current diff.

## Optional Real-Agent Evidence

Real Codex and Claude checks are not part of the default gate. They are opt-in release evidence because provider credentials, quota, model access, network behavior, external CLI versions, and interactive behavior vary by operator environment. These are operator environment concerns, not ADP quality guarantees.

Keep this evidence separate from default gate evidence. `scripts/check-all.sh` remains provider-free. `scripts/real-agent-invocation-smoke.sh` is not part of `scripts/check-all.sh` and must not become a default CI or release gate. A failure in optional real-agent evidence does not fail the default release gate unless the release explicitly claims that evidence tier.

Command availability evidence uses the runtime smoke real flags. It confirms that the external command exists and that a lightweight `--version` or `--help` invocation completes. It does not invoke a model or prove provider account readiness.

```bash
ADP_SMOKE_REAL_CODEX=1 scripts/runtime-smoke.sh --real-codex
ADP_SMOKE_REAL_CLAUDE=1 scripts/runtime-smoke.sh --real-claude
```

Non-interactive real model invocation evidence uses the dedicated invocation smoke. It proves only that, in the current operator environment, ADP can hand off a constrained non-interactive request to the installed external CLI. It may contact providers and consume quota.

```bash
ADP_REAL_INVOKE_CODEX=1 scripts/real-agent-invocation-smoke.sh --codex
ADP_REAL_INVOKE_CLAUDE=1 scripts/real-agent-invocation-smoke.sh --claude
ADP_REAL_INVOKE_CODEX=1 ADP_REAL_INVOKE_CLAUDE=1 scripts/real-agent-invocation-smoke.sh --all
```

Manual interactive provider acceptance is a third, separate operator note for real `adp run ...` sessions:

```bash
adp run codex --workspace <name> --task <task-id> -- <codex-args>
adp run claude --workspace <name> --task <task-id> -- <claude-args>
```

Manual interactive evidence is required for any release note that claims interactive provider workflow readiness. Non-interactive invocation evidence does not validate interactive session quality, provider-native resume, external tool permissions, user-specific shell startup behavior, or broad model performance.

When optional real-agent evidence is collected, record:

- The evidence tier: command availability, non-interactive invocation, or manual interactive acceptance.
- The command that was run and the enabled gate variable, such as `ADP_SMOKE_REAL_CODEX=1`, `ADP_SMOKE_REAL_CLAUDE=1`, `ADP_REAL_INVOKE_CODEX=1`, or `ADP_REAL_INVOKE_CLAUDE=1`.
- The resolved command path from `command -v` or the explicit override path.
- The Codex or Claude CLI version when available, or the first `--help` line when `--version` is not supported.
- Whether the smoke passed through `--version` or fell back to `--help`.
- The operating system and shell.
- Any environment overrides such as `ADP_SMOKE_CODEX_BIN`, `ADP_SMOKE_CLAUDE_BIN`, `ADP_REAL_CODEX_BIN`, `ADP_REAL_CLAUDE_BIN`, model overrides, timeout overrides, or budget overrides.
- For non-interactive invocation, non-sensitive session/event evidence and project-root cleanliness evidence.
- Whether a separate manual interactive session was completed, when claimed.

Do not paste credentials, tokens, account identifiers, private prompts, or sensitive model output into release notes; record only non-sensitive operator evidence. For the full procedure, use [real-agent-compatibility.md](real-agent-compatibility.md).

## Failure Triage

This section is a quick map for default gate failures. For the operator drill flow, package manifest failures, checksum mismatches, and source archive issues, use [release-troubleshooting.md](release-troubleshooting.md).

If `scripts/runtime-smoke.sh --fake` fails, inspect the reported step first. The fake smoke is the highest-signal check for runtime overlay behavior, runtime manifest fields, adapter launch paths, local event history, session aggregation, and project-root pollution.

If `scripts/runtime-audit-smoke.sh` fails, inspect whether the documented runtime audit matrix still matches the current CLI surface. The audit smoke is intentionally fake-runtime and local-only; failures should not be fixed by adding real Codex/Claude defaults, hosted services, automatic Git execution, provider-native resume, or project-root exports.

If `scripts/runtime-context-smoke.sh` fails, inspect the generated instruction files, adapter metadata files, selected profiles, prompt, shared memory, MCP references, task metadata, runtime environment variables, local event/session evidence, workspace diagnostics, and project-root cleanliness. Do not fix context failures by adding hosted services, automatic Git execution, provider-native resume, real-provider default checks, or project-root report/planning exports.

If `scripts/release-readiness-smoke.sh` fails, inspect phase evidence recording and the fake Git tripwire first. Phase accept, commit, and push commands must record local evidence only; failures should not be fixed by making ADP run Git automatically or by weakening the phase lifecycle gate.

If `scripts/release-rehearsal-smoke.sh` fails, inspect the temporary clean workspace step first: copied non-ignored files, release ldflags, `adp version`, copied example workspace bootstrap, isolated `ADP_HOME` and `ADP_RUNTIME_DIR`, and fake Git tripwire output. Do not fix rehearsal failures by relying on machine-local ADP state, real provider CLIs, network access, automatic Git execution, or project-root exports.

If `scripts/release-artifact-smoke.sh` fails, inspect the package staging directory, artifact checksum, package manifest, install-from-artifact path, explicit source archive `COMMIT`, temporary `ADP_HOME`, temporary `ADP_RUNTIME_DIR`, fake Codex command, and project-root pollution scan. Do not fix artifact failures by running from the source tree, including local state in the package, relying on `.git` inside a source archive, or turning real Codex/Claude checks into default gates.

If `scripts/release-operator-drill-smoke.sh` fails, inspect the no-`.git` source copy, documented release commands, release script syntax checks, explicit commit build, checksum verification, installed `PATH` binary, fake Codex handoff sequence, phase evidence records, fake Git tripwire, and project-root pollution scan. Do not repair drill failures by adding machine-local source files, automatic Git execution, or hosted orchestration.

If `scripts/install-onboarding-smoke.sh` fails, inspect local build metadata, temporary `GOBIN`, `PATH` ordering, temporary `ADP_HOME`, temporary `ADP_RUNTIME_DIR`, workspace registration, fake Codex path, fake Claude guard, task-bound context, local event/session/progress evidence, fake Git tripwire output, and project-root pollution scan. Do not repair onboarding failures by requiring real provider CLIs, writing ADP state into the project root, or adding hosted setup steps.

If an optional command availability check fails because `ADP_SMOKE_REAL_CODEX=1` or `ADP_SMOKE_REAL_CLAUDE=1` is missing, treat it as an intentionally unenabled operator check. If the command is unavailable, install the external CLI on that machine or set `ADP_SMOKE_CODEX_BIN` or `ADP_SMOKE_CLAUDE_BIN` to the intended command path. If both `--version` and `--help` fail, classify it as an external CLI, wrapper, or operator-environment evidence gap unless the deterministic fake gate or ADP launch contract also fails.

If `scripts/real-agent-invocation-smoke.sh` fails because `ADP_REAL_INVOKE_CODEX=1` or `ADP_REAL_INVOKE_CLAUDE=1` is missing, treat it as an intentionally unenabled operator check. Authentication, quota, billing, model access, provider response, and network failures are operator environment evidence gaps unless the default fake gate or ADP's local launch/session/project-root cleanliness contract also fails. Manual interactive provider failures belong in operator acceptance notes and do not change the default release gate.

If an operator drill fails before a smoke script starts because the workspace cannot be resolved, distinguish the two common cases. `workspace not found: <name>` means the requested name is absent from the local registry; run `adp workspace list`, add the workspace with `adp workspace add <name> <project-root>`, or pass the registered name through `--workspace` or `ADP_WORKSPACE`. `workspace is required; pass --workspace, set ADP_WORKSPACE, or run from inside a registered project` means no workspace was selected. Do not create project-root planning files as a workaround.

If a manual task-bound run was typed as `adp run --task <task-id>`, correct the command shape to `adp run <agent> --workspace <name> --task <task-id> -- <agent-args>`. `--task` is a run option after the agent name, not a replacement for the required agent argument.

If a task-bound runtime smoke step fails, inspect workspace resolution, task lookup under `$ADP_HOME/workspaces/<workspace>/planning`, generated task context in `AGENTS.md` or `CLAUDE.md`, `ADP_TASK_ID` in the runtime environment, and task IDs in events and sessions.

If a diagnostics step fails, compare `adp doctor [workspace]` with `adp workspace doctor [name]` and inspect the local workspace registry, project root, `ADP_RUNTIME_DIR`, referenced prompts, memory files, MCP files, profile files, and agent command settings. For runtime parent failures, confirm `ADP_RUNTIME_DIR` is not the filesystem root, not equal to the project root, not inside the project root, not a parent directory containing the project root, not a file, and not an unintended symlink. A runtime parent error should return exit code `2` from doctor and should stop `adp env <workspace> --cd`, `adp enter <workspace>`, or `adp run <agent> --workspace <name>` before runtime creation. Fix the directory boundary instead of forcing the run.

For agent command/profile warnings, check whether the enabled agent has an adapter default, whether `command` contains inline arguments that should be passed after `--` or moved into a wrapper, whether path-like command wrappers exist and are executable, whether non-default profile files are missing or duplicated, and whether profile files escape the workspace through symlinks or path traversal. A missing non-default profile is warning-only in doctor output; fix it by adding one matching file under `$ADP_HOME/workspaces/<workspace>/profiles/` or by selecting `default` or an existing profile. Treat it as release-blocking only when the release evidence depends on that profile.

If a completion value step fails, inspect the local adapter registry, local workspace name discovery under `$ADP_HOME/workspaces`, `ADP_WORKSPACE` or `--workspace` resolution, workspace agent profiles, and files under the workspace `profiles/` directory. Completion value endpoints must stay read-only and local.

If a version step fails, inspect the CLI build variables in `internal/cli` and the release `-ldflags` described in [release-packaging.md](release-packaging.md). Development builds may print `dev`; packaged preview binaries should inject version, commit, and build date.

If `scripts/example-workspace-smoke.sh` fails, inspect whether the copied `examples/basic-workspace/workspace.yaml` still matches the current schema and whether `adp env <workspace> --cd` still produces a kept runtime with project-file symlinks.

If `scripts/task-manager-smoke.sh` fails, inspect task CLI parsing, workspace resolution, task storage under `planning/`, planning doctor diagnostics and exit code `2` behavior, next-work JSON selection, helper wiring, JSON report validation, next-work/plan-doctor/report read-only checks, and project-root pollution checks.

If `scripts/plan-intake-smoke.sh` fails, inspect plan input parsing, plan preview read-only behavior, explicit apply writes under `$ADP_HOME`, planning batch rollback behavior, JSON output shape, and the project-root/runtime/Git side-effect checks.

If a phase-gate smoke step fails, inspect phase record storage, task owner state, claim lease parsing, owner-checked release, append-only progress events, acceptance result recording, commit hash recording, push result recording, and lifecycle ordering. Expected operator errors include commit evidence before passed acceptance, push evidence before commit evidence, and starting a later phase before the earlier phase has pushed evidence. The repair path is the explicit sequence `adp phase accept`, `adp phase commit`, and `adp phase push` after real validation, commit, and push have happened outside ADP. The expected state must remain local under `$ADP_HOME`; failures should not be fixed by writing planning artifacts into the project root or by making ADP run Git automatically.

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
- Preview packages include `LICENSE`, `COMMERCIAL.md`, and `COMMERCIAL.zh-CN.md`, preserve required notices and attribution, and do not include `.envrc`, `mvp.md`, local ADP state, runtime overlays, logs, task state, credentials, or machine-specific shell configuration.
- A sorted package manifest was recorded and checked against the required include/exclude list before publishing.
- At least one artifact was installed from the package into a temporary `PATH` and exercised without using the source-tree binary.
- The license files and PolyForm Noncommercial/source-available positioning were not changed unintentionally, and public docs do not imply that noncommercial availability grants commercial rights.
- Packaged CLI artifacts were built with version, commit, and build-date ldflags and `adp version` reports the expected values.
- README and focused docs describe the current CLI surface without Web, UI, SaaS, cloud sync, hosted tracker, hosted orchestration, automatic Git execution, automatic task closure, provider-native resume, or project-root report export drift.
- Active development phases have local evidence for acceptance, commit, and successful push before the next phase starts, and `adp phase status --workspace <name> --format json` agrees that the next planned phase can start.
- Any claimed real-agent compatibility names the evidence tier, and each claimed tier has matching opt-in evidence: command availability, non-interactive invocation, and/or manual interactive acceptance.

## Out Of Scope

The release gate does not validate:

- Provider credentials, accounts, quota, or billing.
- Remote model access or availability.
- External network reliability.
- Real interactive Codex or Claude session quality.
- User-specific shell startup files.
- Hosted deployment, SaaS operations, dashboards, Web UI behavior, hosted trackers, automatic Git execution, automatic task closure, provider-native resume, or project-root report export behavior.

Those checks belong to operator-specific acceptance notes, not the default local release gate.
