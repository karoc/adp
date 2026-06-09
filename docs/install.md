# Install And Bootstrap

Simplified Chinese: [install.zh-CN.md](install.zh-CN.md)

This document describes local installation and first-run bootstrap for ADP. ADP is a terminal-first, local-first runtime manager; installation is a local CLI workflow and does not require a hosted service. For a single operator-facing walkthrough, see [operator-onboarding.md](operator-onboarding.md).

## Prerequisites

- Go installed locally.
- A POSIX shell environment for the repository scripts.
- A local project directory with an absolute path.
- Optional external agent CLIs, such as Codex or Claude, only when you are ready to run real agents. The default smoke path uses fake agents and does not require them.

## Run From Source

From a source checkout:

```bash
git clone git@github.com:karoc/adp.git
cd adp
go run ./cmd/adp --help
```

You can run every CLI command through `go run` while developing:

```bash
go run ./cmd/adp init
go run ./cmd/adp workspace add game-a /absolute/path/to/project
go run ./cmd/adp workspace doctor game-a
```

Use this path when you want to test the current working tree without installing a binary.

## Build A Local Binary

Build the CLI from the repository root:

```bash
mkdir -p bin
go build -o ./bin/adp ./cmd/adp
./bin/adp --help
```

Use the built binary for local bootstrap:

```bash
./bin/adp init
./bin/adp workspace add game-a /absolute/path/to/project
./bin/adp workspace doctor game-a
```

## Install With Go

From a source checkout:

```bash
go install ./cmd/adp
```

Make sure the Go install directory is on `PATH`. It is usually `$(go env GOPATH)/bin` unless `GOBIN` is set:

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
adp --help
```

When installing a published module version, use the versioned module path:

```bash
go install github.com/karoc/adp/cmd/adp@<version>
```

Use an explicit version for repeatable installs.

## Choose An Operator Path

Use one of these paths before the first workspace registration:

- Source checkout: run `go run ./cmd/adp version`, then use `go run ./cmd/adp <command>` while developing.
- Built binary: run `mkdir -p bin && go build -o ./bin/adp ./cmd/adp`, then validate with `./bin/adp version`.
- Temporary install path: copy a built or unpacked release binary into a temporary directory on `PATH`, then validate with `adp version`.

Example temporary install path:

```bash
mkdir -p bin
go build -o ./bin/adp ./cmd/adp
ADP_INSTALL_DIR="$(mktemp -d)"
install -m 0755 ./bin/adp "${ADP_INSTALL_DIR}/adp"
export PATH="${ADP_INSTALL_DIR}:${PATH}"
command -v adp
adp version
```

For a release package, unpack the package first and use its `bin/adp` artifact in the `install -m 0755` command. Keep temporary install directories outside project roots.

## Environment Variables

ADP can run with defaults, but these variables are useful for local bootstrap and repeatable testing:

- `ADP_HOME`: ADP home directory. Defaults to `~/.adp`. Workspace configs live under `$ADP_HOME/workspaces`, and local event logs live under `$ADP_HOME/logs`.
- `ADP_RUNTIME_DIR`: parent directory for temporary runtime overlays. Defaults to the system temp directory under `adp-runtime`. Do not point it at the filesystem root, a project root, a directory inside a project root, or a directory that contains the project root. Prefer a direct local directory; symlink runtime parents are reported as warnings by `adp doctor` and `adp workspace doctor`.
- `ADP_WORKSPACE`: default workspace for commands that accept a workspace. `adp run` resolves the workspace from `--workspace`, then `ADP_WORKSPACE`, then the current directory if it is inside a registered project root.

For isolated validation, use temporary directories:

```bash
export ADP_HOME="$(mktemp -d)"
export ADP_RUNTIME_DIR="$(mktemp -d)"
```

## Isolated First-Run Rehearsal

After using one of the install paths above, run a provider-free rehearsal from a shell where the chosen `adp` command is available. If you built `./bin/adp` instead of installing `adp` on `PATH`, replace `adp` with `./bin/adp` in this block.

```bash
command -v adp
adp version

ADP_REHEARSAL_ROOT="$(mktemp -d)"
export ADP_HOME="${ADP_REHEARSAL_ROOT}/adp-home"
export ADP_RUNTIME_DIR="${ADP_REHEARSAL_ROOT}/runtime"
mkdir -p "${ADP_REHEARSAL_ROOT}/project" "${ADP_REHEARSAL_ROOT}/fake-bin"
printf 'module example.com/adp-rehearsal\n' > "${ADP_REHEARSAL_ROOT}/project/go.mod"
printf 'package main\n' > "${ADP_REHEARSAL_ROOT}/project/main.go"

cat > "${ADP_REHEARSAL_ROOT}/fake-bin/codex" <<'SH'
#!/usr/bin/env sh
printf 'fake codex cwd=%s args=%s\n' "$(pwd)" "$*"
test -n "${ADP_SESSION_ID:-}"
test -n "${ADP_RUNTIME_ROOT:-}"
test "$(pwd)" = "$ADP_RUNTIME_ROOT"
test -f "$ADP_RUNTIME_ROOT/AGENTS.md"
SH
chmod +x "${ADP_REHEARSAL_ROOT}/fake-bin/codex"
export PATH="${ADP_REHEARSAL_ROOT}/fake-bin:${PATH}"

adp init
adp workspace add game-a "${ADP_REHEARSAL_ROOT}/project"
adp workspace list
adp workspace show game-a
adp workspace doctor game-a
adp doctor game-a
TASK_ID=$(adp tasks add --workspace game-a --priority high "Validate isolated first run" | sed -n 's/^task \(task-[^ ]*\) added$/\1/p')
test -n "$TASK_ID"
adp run codex --workspace game-a --task "$TASK_ID" -- --example-smoke
adp events list --workspace game-a --task "$TASK_ID" --limit 5
adp sessions list --workspace game-a --agent codex --task "$TASK_ID"
adp plan doctor --workspace game-a --format json
adp progress --workspace game-a --format json
ROOT_LEAKS="$(find "${ADP_REHEARSAL_ROOT}/project" -maxdepth 2 \( -name AGENTS.md -o -name CLAUDE.md -o -name .codex -o -name .claude -o -name .adp-runtime.yaml -o -name planning -o -name tasks.yaml -o -name phases.yaml -o -name progress.jsonl \) -print)"
test -z "$ROOT_LEAKS"
```

The final project-root leak check should pass without output. This rehearsal keeps ADP state under temporary `$ADP_HOME`, keeps runtime overlays under temporary `$ADP_RUNTIME_DIR`, uses a fake local `codex`, does not run Git, and does not write planning or report exports into the project root.

## Bootstrap A Workspace

Initialize ADP:

```bash
adp init
```

Register a local project. The project root must be an absolute path:

```bash
adp workspace add game-a /absolute/path/to/project
```

Inspect and validate the workspace:

```bash
adp workspace list
adp workspace show game-a
adp workspace doctor game-a
```

`adp workspace doctor` checks local configuration, project root reachability, runtime parent safety, referenced prompt, memory, MCP, and profile files, agent command settings, and reserved project-root paths. It reports adapter default command fallback, inline command arguments, missing or non-executable path-like command wrappers, and missing, ambiguous, or escaping non-default profiles as local diagnostics. Fix doctor errors before running real agents; warning-only command/profile diagnostics do not prove or disprove real provider CLI authentication, network access, or model availability.

## Enter Or Run A Runtime

Render shell exports for a kept runtime overlay:

```bash
adp env game-a --cd
```

Run an agent through ADP:

```bash
adp run codex --workspace game-a -- <agent-args>
adp run claude --workspace game-a -- <agent-args>
```

These commands require the corresponding external CLI to be installed and authenticated. Arguments after `--` are forwarded to the external agent command. ADP does not define which arguments are safe or supported by a particular external CLI; verify that with the installed CLI on the operator machine. Use the isolated rehearsal above for the default provider-free validation path. Use `examples/basic-workspace` and `scripts/example-workspace-smoke.sh` when you need a copyable workspace configuration with Codex and Claude profiles, base prompts, shared memory, and MCP settings.

Inspect local history:

```bash
adp events list --workspace game-a
adp sessions list --workspace game-a
adp sessions show <session-id>
adp sessions restore-plan <session-id>
```

`sessions restore-plan` prints a read-only suggested `adp run ...` command for a previous session when enough non-sensitive invocation data is available. It does not execute the command, launch an agent, append events, mutate task state, write to the real project root, or resume a provider-native conversation.

Clean old ADP-owned runtime directories:

```bash
adp runtime prune --older-than 24h --dry-run
adp runtime prune --older-than 24h
```

`runtime prune` only removes directories that contain a current-version ADP runtime manifest whose `runtime_root` matches the directory being removed. Incompatible, malformed, foreign, or self-inconsistent manifests are skipped. Use `--dry-run` before deleting.

## Deterministic Bootstrap Smoke

From the repository root, run the aggregate validation gate:

```bash
scripts/check-all.sh
```

The aggregate gate includes fake-agent runtime smoke, broad runtime audit smoke, focused runtime context smoke, release readiness smoke, release rehearsal smoke, release artifact smoke, release operator drill smoke, install onboarding smoke, example workspace smoke, task manager smoke, plan intake smoke, Go tests, vet, file line checks, bilingual documentation checks, and diff whitespace checks.

For targeted bootstrap checks, run:

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
```

The runtime smoke builds the current `cmd/adp` binary into a temporary directory and uses temporary `ADP_HOME`, `ADP_RUNTIME_DIR`, fake agent binaries, and a temporary project root. It verifies the runtime overlay path without requiring real Codex or Claude CLIs. The runtime audit smoke broadens coverage across CLI help, JSON outputs, task/phase/plan/progress flows, sessions, restore planning, completion values, and local-first runtime boundaries. The runtime context smoke focuses on the exact launch context fake agents receive: generated instruction files, adapter metadata, selected profiles, prompt, shared memory, MCP references, task metadata, runtime environment variables, local evidence, diagnostics, and project-root cleanliness. The release readiness smoke verifies release-gate invariants such as phase commit and push evidence recording without Git execution. The release rehearsal smoke copies the current non-ignored repository files into a temporary clean workspace, builds a preview binary with release ldflags, verifies copied docs and file limits, bootstraps the copied example workspace, and checks phase evidence recording with a fake Git tripwire. The release artifact smoke validates package contents, checksums, install-from-artifact, source archive builds without `.git`, and local-only exclusion boundaries. The release operator drill smoke rehearses the operator sequence from clean source form through checksum, temporary install, fake-provider handoff, and local phase evidence recording. The install onboarding smoke validates source/build/temp-install onboarding, fake-provider first run, event and session evidence, project-root cleanliness, and Git side-effect guards. The example workspace smoke copies `examples/basic-workspace` into a temporary `ADP_HOME` and verifies that the published example still bootstraps against a temporary project. The plan intake smoke verifies that structured local planning input can be previewed read-only and explicitly applied to `$ADP_HOME` without project-root, runtime, Git, or partial-write side effects.
