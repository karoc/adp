# Operator Onboarding

Simplified Chinese: [operator-onboarding.zh-CN.md](operator-onboarding.zh-CN.md)

This guide is the concrete first-run path for a new ADP operator. It stays terminal-first and local-first: no Web UI, dashboard, SaaS tracker, cloud sync, hosted orchestration, automatic Git workflow, or real provider CLI is required for the default rehearsal.

For installation details, see [install.md](install.md). For a reusable workspace configuration example, see `examples/basic-workspace`.

## Current Start-Using Boundary

ADP is ready for a local technical operator to start with a provider-free first trial, then move to durable local workspaces after the isolated rehearsal passes. Treat this as local trial readiness for terminal-first workflows, not as a claim that every real provider environment or interactive session is production-ready.

Use this guide when you need to verify local install paths, workspace registration, diagnostics, task pickup, fake-provider runtime handoff, event/session/progress inspection, restore guidance, completion values, runtime prune dry-run, and project-root cleanliness. Keep real provider authentication, model access, quota, network behavior, and interactive session quality as separate opt-in acceptance checks.

## What The First Trial Proves

The first trial is a local rehearsal, not a production setup. By the end of it, you should have evidence that:

- the selected `adp` command runs and exposes copyable help for nested commands and common parser errors point back to the right help page;
- ADP can initialize isolated local state under a temporary `$ADP_HOME`;
- a workspace can point at a local project without writing ADP files into that project root;
- `workspace doctor` and `doctor` can inspect the local setup before an agent run;
- `adp run codex --take --owner --lease` can atomically claim a task, build a runtime overlay, and launch a provider command;
- events, sessions, progress, restore guidance, plan diagnostics, completion values, and runtime prune dry-run are readable from local ADP state; and
- the same board can be inspected or claimed without launching an agent through `tasks next` and `tasks take`.

The provider in this guide is a fake local `codex` shell script. A passing first trial does not prove real provider authentication, model access, quota, network behavior, or interactive session quality.

## Choose An ADP Command

Pick exactly one command form for the current shell.

From source while developing:

```bash
adp_local() { go run ./cmd/adp "$@"; }
adp_local version
```

From a local build:

```bash
mkdir -p bin
go build -o ./bin/adp ./cmd/adp
adp_local() { "$PWD/bin/adp" "$@"; }
adp_local version
```

From a temporary install path, including after unpacking a release package that contains `bin/adp`:

```bash
mkdir -p bin
go build -o ./bin/adp ./cmd/adp
ADP_INSTALL_DIR="$(mktemp -d)"
install -m 0755 ./bin/adp "${ADP_INSTALL_DIR}/adp"
adp_local() { "${ADP_INSTALL_DIR}/adp" "$@"; }
adp_local version
```

For a released artifact, replace `./bin/adp` in the last block with the unpacked artifact path. Keep the temporary install path outside the project root.

Before creating state, confirm the command surface from the same command form you chose:

```bash
adp_local --help
adp_local workspace --help
adp_local tasks --help
adp_local tasks take --help
adp_local sessions resume-plan --help
adp_local runtime prune --help
```

Use the same nesting pattern for other groups: `adp_local <command> --help` and `adp_local <command> <subcommand> --help`. Leaf help may include `See also:` for the parent command. If a build prints a friendly `try:` hint, read it as a suggested help command to run manually; it does not inspect projects, mutate `$ADP_HOME`, create runtimes, call providers, or run Git by itself.

Expected result: each command exits successfully and prints local help or version text with copyable examples for first-use commands. If this fails, fix the selected command path before creating any ADP state.

## ID Prefix Matching

ADP supports convenient prefix matching for task and session IDs. Instead of typing the full ID, you can use any unique prefix:

```bash
# Task ID prefix matching
adp tasks show task-20260611-0001    # Full ID
adp tasks show task-2026             # Prefix (if unique)
adp tasks claim task-001 --owner alice --lease 2h

# Session ID prefix matching
adp sessions show session-20260611T102030-abc123    # Full ID
adp sessions show 20260611T10                       # Prefix (if unique)
adp sessions restore-plan 2026061
```

When a prefix is ambiguous (matches multiple IDs), ADP returns an error with all matching IDs:

```bash
adp tasks show task-20
# Error: ambiguous task ID "task-20", matches:
#   - task-20260611-0001
#   - task-20260612-0002
```

Prefix matching is available in all commands that accept task or session IDs, including `tasks show/claim/renew/release/done/block`, `sessions show/restore-plan/resume-plan`, `events list`, and `run --task`.

Tips:
- Use longer prefixes when you have many tasks or sessions
- The shortest unique prefix is usually the date portion for recent IDs
- Full IDs always work and are never ambiguous

## Isolated First Run

Use temporary state until the install path is trusted. This rehearsal registers a temporary workspace, inspects the task board, runs a fake `codex` provider through atomic `run --take`, records local events and sessions, checks lease maintenance, and verifies that the project root stays clean.

```bash
ADP_ONBOARDING_ROOT="$(mktemp -d)"
export ADP_HOME="${ADP_ONBOARDING_ROOT}/adp-home"
export ADP_RUNTIME_DIR="${ADP_ONBOARDING_ROOT}/runtime"
mkdir -p "${ADP_ONBOARDING_ROOT}/project" "${ADP_ONBOARDING_ROOT}/fake-bin"
printf 'module example.com/adp-onboarding\n' > "${ADP_ONBOARDING_ROOT}/project/go.mod"
printf 'package main\n' > "${ADP_ONBOARDING_ROOT}/project/main.go"

cat > "${ADP_ONBOARDING_ROOT}/fake-bin/codex" <<'SH'
#!/usr/bin/env sh
printf 'fake codex cwd=%s args=%s\n' "$(pwd)" "$*"
test -n "${ADP_SESSION_ID:-}"
test -n "${ADP_RUNTIME_ROOT:-}"
test -n "${ADP_TASK_ID:-}"
test "$(pwd)" = "$ADP_RUNTIME_ROOT"
test -f "$ADP_RUNTIME_ROOT/AGENTS.md"
test -f "$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
SH
chmod +x "${ADP_ONBOARDING_ROOT}/fake-bin/codex"
export PATH="${ADP_ONBOARDING_ROOT}/fake-bin:${PATH}"

adp_local init
adp_local workspace add game-a "${ADP_ONBOARDING_ROOT}/project"
adp_local workspace list
adp_local workspace show game-a
adp_local workspace doctor game-a
adp_local doctor game-a
adp_local version

TASK_ID=$(adp_local tasks add --workspace game-a --priority high "Validate isolated first run" | sed -n 's/^task \(task-[^ ]*\) added$/\1/p')
test -n "$TASK_ID"
adp_local tasks next --workspace game-a --format json
adp_local run codex --workspace game-a --take --owner first-agent --lease 30m -- --onboarding-smoke
adp_local tasks show --workspace game-a "$TASK_ID"
adp_local tasks renew --workspace game-a "$TASK_ID" --owner first-agent --lease 30m
adp_local tasks stale --workspace game-a --format json
adp_local progress report --workspace game-a --format json
SESSION_ID=$(adp_local sessions list --workspace game-a --agent codex --task "$TASK_ID" | sed -n '2s/ .*//p')
test -n "$SESSION_ID"
adp_local sessions restore-plan "$SESSION_ID"
adp_local plan doctor --workspace game-a --format json

BOARD_TASK_ID=$(adp_local tasks add --workspace game-a --priority normal "Validate board pickup" | sed -n 's/^task \(task-[^ ]*\) added$/\1/p')
test -n "$BOARD_TASK_ID"
TAKEN_ID=$(adp_local tasks take --workspace game-a --owner second-agent --lease 30m | sed -n 's/^task \(task-[^ ]*\) taken .*/\1/p')
test -n "$TAKEN_ID"
adp_local tasks release --workspace game-a "$TAKEN_ID" --owner second-agent
adp_local tasks done --workspace game-a "$TASK_ID"
adp_local events list --workspace game-a --task "$TASK_ID" --limit 5
adp_local sessions list --workspace game-a --agent codex --task "$TASK_ID"
adp_local progress --workspace game-a --format json
adp_local runtime prune --older-than 24h --dry-run

ROOT_LEAKS="$(find "${ADP_ONBOARDING_ROOT}/project" -maxdepth 2 \( -name AGENTS.md -o -name CLAUDE.md -o -name .codex -o -name .claude -o -name .adp-runtime.yaml -o -name planning -o -name tasks.yaml -o -name phases.yaml -o -name progress.jsonl \) -print)"
test -z "$ROOT_LEAKS"
```

Expected result: the block exits successfully, the fake provider prints the runtime working directory, JSON commands print parseable local state, and the last command succeeds without printing project-root leaks. ADP state is under temporary `$ADP_HOME`, runtime overlays are under temporary `$ADP_RUNTIME_DIR`, and the provider command is the fake local `codex` script.

The visible workflow is:

- create local ADP state with `init`;
- register and inspect a workspace with `workspace add`, `workspace list`, `workspace show`, `workspace doctor`, and `doctor`;
- create a task with `tasks add`;
- preview eligible board work with read-only `tasks next`;
- claim work and launch a runtime in one boundary with `run --take --owner --lease`;
- inspect and maintain ownership with `tasks show`, `tasks renew`, and read-only `tasks stale`;
- inspect handoff evidence with `progress report`, `sessions list`, `sessions restore-plan`, `plan doctor`, `events list`, and `progress`;
- prove non-runtime board pickup with `tasks take`, then release that claim with `tasks release`; and
- close the completed trial task with `tasks done`.

Read-only inspection commands in the rehearsal include `tasks next`, `tasks stale`, `progress report`, `sessions list`, `sessions restore-plan`, `plan doctor`, `events list`, and `progress`; mutating commands include `tasks add`, `run --take`, `tasks renew`, `tasks take`, `tasks release`, and `tasks done`. The `tasks take` step proves board pickup without launching a runtime; the `run --take` step proves pickup and runtime launch in one command boundary. `tasks renew` refreshes the current owner's lease, while `tasks stale` is only the recovery inspection view for expired `in_progress` claims.

If the trial fails, keep the temporary root in place and inspect the last failed command first. Common causes are an `adp_local` function that points at the wrong binary, a shell that did not export the fake `codex` directory onto `PATH`, or an unsafe `ADP_RUNTIME_DIR`. Re-run `adp_local workspace doctor game-a` and `adp_local doctor game-a` before repeating the full block.

## Move To Durable Local Use

After the isolated rehearsal passes, choose durable local paths:

```bash
export ADP_HOME="${HOME}/.adp"
export ADP_RUNTIME_DIR="${TMPDIR:-/tmp}/adp-runtime"
adp_local init
adp_local workspace add game-a /absolute/path/to/project
adp_local workspace doctor game-a
```

Expected result: the durable workspace is registered under `${HOME}/.adp`, the project root remains free of ADP-generated files, and doctor output has no error-level diagnostics. Keep `$ADP_RUNTIME_DIR` outside project roots and outside directories that contain project roots. `adp doctor` and `adp workspace doctor` report unsafe runtime parents before real runs.

Use `examples/basic-workspace` when you need a copyable configuration reference with Codex and Claude profiles, base prompts, shared memory, and MCP settings. Copy it into the ADP home workspace configuration area, then update `project.root` before use. It is not required for the minimal smoke path above.

## Real Providers

Real Codex and Claude runs are opt-in operator checks. The default onboarding rehearsal above remains provider-free. Provider credentials, quota, model access, network behavior, and external CLI versions are operator environment concerns, not ADP quality guarantees.

For command availability evidence, intentionally enable the runtime smoke real flags. These checks confirm that the external command is present and can answer a lightweight `--version` or `--help` probe; they do not invoke a model.

```bash
ADP_SMOKE_REAL_CODEX=1 scripts/runtime-smoke.sh --real-codex
ADP_SMOKE_REAL_CLAUDE=1 scripts/runtime-smoke.sh --real-claude
```

For non-interactive real model invocation evidence, intentionally enable the dedicated invocation smoke. It may contact external providers and consume quota. It is not part of `scripts/check-all.sh` and must not become a default CI or release gate.

```bash
ADP_REAL_INVOKE_CODEX=1 scripts/real-agent-invocation-smoke.sh --codex
ADP_REAL_INVOKE_CLAUDE=1 scripts/real-agent-invocation-smoke.sh --claude
ADP_REAL_INVOKE_CODEX=1 ADP_REAL_INVOKE_CLAUDE=1 scripts/real-agent-invocation-smoke.sh --all
```

Manual interactive provider acceptance is separate from both smoke paths. Install and authenticate the external CLI on the operator machine first, then run:

```bash
adp_local run codex --workspace game-a -- <codex-args>
adp_local run claude --workspace game-a -- <claude-args>
```

Arguments after `--` are provider-specific. ADP forwards them but does not define their safety, model availability, quota use, network behavior, authentication state, or interactive session quality. Keep any operator acceptance notes non-sensitive and do not record credentials, tokens, account identifiers, private prompts, or sensitive model output. For the full compatibility procedure, see [real-agent-compatibility.md](real-agent-compatibility.md).
