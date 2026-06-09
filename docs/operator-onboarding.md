# Operator Onboarding

Simplified Chinese: [operator-onboarding.zh-CN.md](operator-onboarding.zh-CN.md)

This guide is the concrete first-run path for a new ADP operator. It stays terminal-first and local-first: no Web UI, dashboard, SaaS tracker, cloud sync, hosted orchestration, automatic Git workflow, or real provider CLI is required for the default rehearsal.

For installation details, see [install.md](install.md). For a reusable workspace configuration example, see `examples/basic-workspace`.

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

## Isolated First Run

Use temporary state until the install path is trusted. This rehearsal registers a temporary workspace, runs a fake `codex` provider, records local events and sessions, and verifies that the project root stays clean.

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
adp_local run codex --workspace game-a --task "$TASK_ID" -- --onboarding-smoke
adp_local events list --workspace game-a --task "$TASK_ID" --limit 5
adp_local sessions list --workspace game-a --agent codex --task "$TASK_ID"
adp_local plan doctor --workspace game-a --format json
adp_local progress --workspace game-a --format json
adp_local runtime prune --older-than 24h --dry-run

ROOT_LEAKS="$(find "${ADP_ONBOARDING_ROOT}/project" -maxdepth 2 \( -name AGENTS.md -o -name CLAUDE.md -o -name .codex -o -name .claude -o -name .adp-runtime.yaml -o -name planning -o -name tasks.yaml -o -name phases.yaml -o -name progress.jsonl \) -print)"
test -z "$ROOT_LEAKS"
```

The last command should succeed without printing project-root leaks. ADP state is under temporary `$ADP_HOME`, runtime overlays are under temporary `$ADP_RUNTIME_DIR`, and the provider command is the fake local `codex` script.

## Move To Durable Local Use

After the isolated rehearsal passes, choose durable local paths:

```bash
export ADP_HOME="${HOME}/.adp"
export ADP_RUNTIME_DIR="${TMPDIR:-/tmp}/adp-runtime"
adp_local init
adp_local workspace add game-a /absolute/path/to/project
adp_local workspace doctor game-a
```

Keep `$ADP_RUNTIME_DIR` outside project roots and outside directories that contain project roots. `adp doctor` and `adp workspace doctor` report unsafe runtime parents before real runs.

Use `examples/basic-workspace` when you need a copyable configuration reference with Codex and Claude profiles, base prompts, shared memory, and MCP settings. Copy it into the ADP home workspace configuration area, then update `project.root` before use. It is not required for the minimal smoke path above.

## Real Providers

Real Codex and Claude runs are opt-in operator checks. Install and authenticate the external CLI on the operator machine first, then run:

```bash
adp_local run codex --workspace game-a -- <codex-args>
adp_local run claude --workspace game-a -- <claude-args>
```

Arguments after `--` are provider-specific. ADP forwards them but does not define their safety, model availability, network behavior, or authentication state.
