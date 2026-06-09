# Release Evidence

Simplified Chinese: [release-evidence.zh-CN.md](release-evidence.zh-CN.md)

This template records the local evidence needed before publishing an ADP preview artifact. It is a release note companion, not a hosted release system, cloud ledger, SaaS workflow, provider credential check, or replacement for the local phase gate.

## Required Fields

Record these fields for every preview artifact:

- Release version, such as `0.1.0-preview.1`.
- Git commit hash used for the build.
- Source form, such as Git checkout or source archive.
- Build date in UTC.
- Go version.
- Target operating system and architecture.
- Artifact filename.
- Artifact SHA-256 checksum and the checksum command used.
- Packaged binary `adp version` output.
- `scripts/check-all.sh` result.
- Install-from-artifact rehearsal result.
- Source archive or no-`.git` rehearsal result when applicable.
- Package contents manifest, either as an attached manifest path or a concise inline excerpt.
- Explicit list of excluded local state, credentials, logs, and machine-specific files.
- Failure triage notes for any required check that failed before the final passing run.
- Optional real-agent operator evidence, separated by command availability, non-interactive invocation, and manual interactive acceptance when any tier was intentionally enabled.
- License notice: ADP is source-available for noncommercial learning, research, evaluation, and open collaboration; commercial use requires separate paid authorization.

## Build Evidence

The release note should include the exact build identity:

```bash
go version
dist/adp version
sha256sum dist/adp > dist/adp.sha256
sha256sum -c dist/adp.sha256
```

Expected `dist/adp version` release output shape:

```txt
adp 0.1.0-preview.1 commit <commit> built <utc-timestamp>
```

If a source archive does not contain `.git`, record the explicit commit value used before building:

```bash
COMMIT=source-archive-commit
```

A development build that prints `adp dev` is useful for local development but is not sufficient preview artifact evidence.

## Install Rehearsal Evidence

Record evidence that at least one binary was installed and run from an artifact path:

```bash
ADP_INSTALL_BIN="$(mktemp -d)"
install -m 0755 dist/adp "${ADP_INSTALL_BIN}/adp"
export PATH="${ADP_INSTALL_BIN}:${PATH}"
adp version
```

The install rehearsal should use temporary `ADP_HOME`, temporary `ADP_RUNTIME_DIR`, a temporary project root, and a fake local `codex` command. It should prove the installed binary can run the local-first workflow without real provider credentials:

```bash
export ADP_HOME="${ADP_SMOKE_ROOT}/adp-home"
export ADP_RUNTIME_DIR="${ADP_SMOKE_ROOT}/runtime"
adp init
adp workspace add artifact-a "${ADP_SMOKE_ROOT}/project"
adp workspace doctor artifact-a
TASK_ID=$(adp tasks add --workspace artifact-a --priority high "Validate artifact install" | sed -n 's/^task \(task-[^ ]*\) added$/\1/p')
test -n "$TASK_ID"
adp run codex --workspace artifact-a --task "$TASK_ID" -- --artifact-smoke
adp events list --workspace artifact-a --task "$TASK_ID" --limit 1
adp sessions list --workspace artifact-a --agent codex --task "$TASK_ID"
```

The project-root pollution scan should find no ADP-generated files:

```bash
ROOT_LEAKS="$(find "${ADP_SMOKE_ROOT}/project" -maxdepth 2 \( -name AGENTS.md -o -name CLAUDE.md -o -name .codex -o -name .claude -o -name .adp-runtime.yaml -o -name planning -o -name tasks.yaml -o -name phases.yaml -o -name progress.jsonl \) -print)"
test -z "$ROOT_LEAKS"
```

## Package Contents Evidence

Record the files included in each package. A preview package should include one target-platform `adp` binary, `README.md`, `README.zh-CN.md`, `LICENSE`, `COMMERCIAL.md`, `COMMERCIAL.zh-CN.md`, `docs/release-packaging.md`, `docs/release-packaging.zh-CN.md`, `docs/release-evidence.md`, `docs/release-evidence.zh-CN.md`, and a short release note.

Also record that the package excludes `.envrc`, `mvp.md`, `$ADP_HOME`, `$ADP_RUNTIME_DIR`, runtime overlays, logs, task state, credentials, machine-specific shell startup files, and temporary release rehearsal directories.

Use a sorted archive listing or equivalent package tool output as the manifest:

```bash
tar -tf adp-0.1.0-preview.1-linux-amd64.tar.gz | sort
```

If the manifest includes local state or misses required notices, classify the release as failed until a rebuilt package passes manifest inspection and checksum verification.

## Failed Or Deferred Checks

Required gate failures must be recorded as failed operator evidence and must stop the release candidate. After a fix, rerun the failed command and the aggregate gate before replacing the failed note with passing evidence. Use [release-troubleshooting.md](release-troubleshooting.md) to classify build, checksum, manifest, install, source archive, and environment failures.

Optional real-agent evidence can be recorded as `not run` per tier when it was not intentionally enabled. A failed optional real-agent check blocks the release only when the release note claims that tier of real-agent compatibility beyond the deterministic fake gate.

## Optional Real-Agent Evidence

Real Codex and Claude checks remain separate, opt-in operator evidence. They must not become default release gates because provider credentials, quota, model access, network behavior, and external CLI versions are operator environment concerns, not ADP quality guarantees. `scripts/check-all.sh` must remain provider-free.

Record optional evidence in distinct tiers:

- Command availability evidence uses the runtime smoke real flags. It checks that the external command is available and can answer a lightweight `--version` or `--help` probe; it does not invoke a model.

```bash
ADP_SMOKE_REAL_CODEX=1 scripts/runtime-smoke.sh --real-codex
ADP_SMOKE_REAL_CLAUDE=1 scripts/runtime-smoke.sh --real-claude
```

- Non-interactive real model invocation evidence uses the dedicated invocation smoke. It may contact external providers and consume quota. It is not part of `scripts/check-all.sh` and must not become a default CI or release gate.

```bash
ADP_REAL_INVOKE_CODEX=1 scripts/real-agent-invocation-smoke.sh --codex
ADP_REAL_INVOKE_CLAUDE=1 scripts/real-agent-invocation-smoke.sh --claude
ADP_REAL_INVOKE_CODEX=1 ADP_REAL_INVOKE_CLAUDE=1 scripts/real-agent-invocation-smoke.sh --all
```

- Manual interactive provider acceptance is a separate operator note for real `adp run ...` sessions. It is required only for release claims about interactive provider behavior, and the note must avoid credentials, tokens, account identifiers, private prompts, and sensitive model output.

When a tier is not run, record `not run` for that tier rather than treating the release evidence as incomplete. For the full procedure and redaction guidance, see [real-agent-compatibility.md](real-agent-compatibility.md).
