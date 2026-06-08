# Install And Bootstrap

Simplified Chinese: [install.zh-CN.md](install.zh-CN.md)

This document describes local installation and first-run bootstrap for ADP. ADP is a terminal-first, local-first runtime manager; installation is a local CLI workflow and does not require a hosted service.

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

Arguments after `--` are forwarded to the external agent command. ADP does not define which arguments are safe or supported by a particular external CLI; verify that with the installed CLI on the operator machine.

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

The aggregate gate includes fake-agent runtime smoke, example workspace smoke, task manager smoke, Go tests, vet, file line checks, bilingual documentation checks, and diff whitespace checks.

For targeted bootstrap checks, run:

```bash
scripts/runtime-smoke.sh --fake
scripts/example-workspace-smoke.sh
scripts/task-manager-smoke.sh
```

The runtime smoke builds the current `cmd/adp` binary into a temporary directory and uses temporary `ADP_HOME`, `ADP_RUNTIME_DIR`, fake agent binaries, and a temporary project root. It verifies the runtime overlay path without requiring real Codex or Claude CLIs. The example workspace smoke copies `examples/basic-workspace` into a temporary `ADP_HOME` and verifies that the published example still bootstraps against a temporary project.
