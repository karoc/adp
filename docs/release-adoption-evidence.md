# Release Adoption Evidence

Simplified Chinese: [release-adoption-evidence.zh-CN.md](release-adoption-evidence.zh-CN.md)

This note records post-publish adoption evidence for released ADP artifacts. It is an operator evidence note, not a hosted release ledger, SaaS workflow, provider credential check, or replacement for the required release gate. ADP's default validation remains provider-free.

## v1.0.1 Fresh Post-Publish Adoption Smoke

Evidence date:

- UTC: 2026-07-06T22:56:28Z
- Local operator date: 2026-07-07

Scope:

- Release: `v1.0.1`
- Artifact: `adp-1.0.1-linux-amd64`
- Artifact source: `https://github.com/karoc/adp/releases/download/v1.0.1/adp-1.0.1-linux-amd64`
- Checksum source: `https://github.com/karoc/adp/releases/download/v1.0.1/SHA256SUMS`
- Execution environment: fresh temporary `HOME`, `ADP_HOME`, `ADP_RUNTIME_DIR`, `PATH`, and project root
- Binary source: downloaded release artifact installed into a temporary directory
- Provider mode: fake local Codex command only; no real provider credentials, model invocation, quota, or networked provider session

Fresh-environment proof:

- A new temporary root was created with `mktemp -d`.
- The release artifact and `SHA256SUMS` were downloaded into that temporary root.
- The downloaded artifact was installed as `adp` into a temporary install directory.
- `PATH` was set so the temporary install directory came before system paths.
- `HOME`, `ADP_HOME`, and `ADP_RUNTIME_DIR` all pointed inside the temporary root before the first `adp` command.
- `command -v adp` resolved to the temporary install directory, not to the repository checkout, `dist/`, or a source-built binary.

Artifact verification:

```text
adp-1.0.1-linux-amd64: OK
sha256: 9fba1e473e5997124f92e1e43377d7132bc2766a1c71788a937e8bae0bf2c1a4
```

Installed binary identity:

```text
/tmp/adp-p72-adoption.<redacted>/install-bin/adp
adp version 1.0.1
commit: c02033a45c7440bb116538dac6a00bb6c4ca864d
built: 2026-07-06T20:10:40Z
go: go1.25.7
platform: linux/amd64
```

Sanitized command sequence:

```bash
TMP_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/adp-p72-adoption.XXXXXX")
curl -fsSL https://github.com/karoc/adp/releases/download/v1.0.1/adp-1.0.1-linux-amd64 -o "$TMP_ROOT/download/adp-1.0.1-linux-amd64"
curl -fsSL https://github.com/karoc/adp/releases/download/v1.0.1/SHA256SUMS -o "$TMP_ROOT/download/SHA256SUMS"
(cd "$TMP_ROOT/download" && grep 'adp-1.0.1-linux-amd64$' SHA256SUMS | sha256sum -c -)
install -m 0755 "$TMP_ROOT/download/adp-1.0.1-linux-amd64" "$TMP_ROOT/install-bin/adp"
export HOME="$TMP_ROOT/home"
export ADP_HOME="$TMP_ROOT/adp-home"
export ADP_RUNTIME_DIR="$TMP_ROOT/runtime"
export PATH="$TMP_ROOT/install-bin:$TMP_ROOT/fake-bin:/usr/bin:/bin"
command -v adp
adp version
adp init
adp workspace add p72-adoption "$TMP_ROOT/project"
adp workspace doctor p72-adoption
adp phase add --workspace p72-adoption p72 "P72 adoption phase"
adp phase start --workspace p72-adoption p72
adp tasks add --workspace p72-adoption --phase p72 --priority high "Validate published v1.0.1 adoption"
adp run codex --workspace p72-adoption --task task-20260706-0001 -- --p72-adoption
adp events list --workspace p72-adoption --task task-20260706-0001 --limit 5
adp sessions list --workspace p72-adoption --agent codex --task task-20260706-0001
adp progress report --workspace p72-adoption --format markdown
adp plan doctor --workspace p72-adoption --format text
find "$TMP_ROOT/project" -maxdepth 2 \( -name AGENTS.md -o -name CLAUDE.md -o -name .codex -o -name .claude -o -name .adp-runtime.yaml -o -name planning -o -name tasks.yaml -o -name phases.yaml -o -name progress.jsonl \) -print
```

Provider-free workflow evidence:

- `adp init` created a new temporary ADP home.
- `adp workspace add p72-adoption <temporary-project>` registered a fresh temporary project root.
- `adp workspace doctor p72-adoption` completed with one expected warning because the temporary project root was not a Git worktree. This warning does not block local-first ADP operation.
- `adp phase add` and `adp phase start` created and activated a local phase in the temporary workspace.
- `adp tasks add` created `task-20260706-0001` in the temporary planning ledger.
- `adp run codex --workspace p72-adoption --task task-20260706-0001 -- --p72-adoption` ran through a fake local Codex command and exited `0`.
- `adp events list` showed `run_started` and `run_finished` for the task-bound fake Codex session.
- `adp sessions list` showed the completed task-bound Codex session.
- `adp progress report --format markdown` completed against the temporary workspace.
- `adp plan doctor --workspace p72-adoption --format text` reported `status: ok`, `error_count: 0`, and `warning_count: 0` for the temporary planning ledger.
- The temporary project root remained clean: no `AGENTS.md`, `CLAUDE.md`, `.codex/`, `.claude/`, `.adp-runtime.yaml`, `planning/`, `tasks.yaml`, `phases.yaml`, or `progress.jsonl` files were written into it.

P72 scope note:

P69 already verified broad release publication, downloaded assets, checksums, package boundaries, package install, fake provider handoff, events, sessions, progress, and project-root cleanliness. P72 does not replace or expand that package verification. This note records a separately timed fresh post-publish adoption smoke from public release URLs with isolated `HOME`, `ADP_HOME`, `ADP_RUNTIME_DIR`, `PATH`, install directory, and project root. It is still provider-free and does not claim real Codex or Claude model readiness.

Optional real-agent evidence:

- Command availability: not run for this adoption note.
- Non-interactive real model invocation: not run for this adoption note.
- Manual interactive provider acceptance: not run for this adoption note.

Residual risk:

- This evidence verifies the `linux/amd64` artifact on one fresh local operator environment. It does not verify other platforms, package managers, real provider credentials, billing, quota, model access, networked model behavior, or interactive provider session quality.
