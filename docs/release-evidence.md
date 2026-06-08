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
- Package contents manifest.
- Explicit list of excluded local state, credentials, logs, and machine-specific files.
- Optional real Codex or Claude CLI evidence, only when it was intentionally enabled.
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
TASK_ID=$(adp tasks add --workspace artifact-a --priority high --phase artifact-smoke "Validate artifact install" | sed -n 's/^task \(task-[^ ]*\) added$/\1/p')
adp run codex --workspace artifact-a --task "$TASK_ID" -- --artifact-smoke
adp events list --workspace artifact-a --task "$TASK_ID" --limit 1
adp sessions list --workspace artifact-a --agent codex --task "$TASK_ID"
```

The project-root pollution scan should find no ADP-generated files:

```bash
find "${ADP_SMOKE_ROOT}/project" -maxdepth 2 \( -name AGENTS.md -o -name CLAUDE.md -o -name .codex -o -name .claude -o -name planning \)
```

## Package Contents Evidence

Record the files included in each package. A preview package should include one target-platform `adp` binary, `README.md`, `README.zh-CN.md`, `LICENSE`, `COMMERCIAL.md`, `COMMERCIAL.zh-CN.md`, `docs/release-packaging.md`, `docs/release-packaging.zh-CN.md`, `docs/release-evidence.md`, `docs/release-evidence.zh-CN.md`, and a short release note.

Also record that the package excludes `.envrc`, `mvp.md`, `$ADP_HOME`, `$ADP_RUNTIME_DIR`, runtime overlays, logs, task state, credentials, machine-specific shell startup files, and temporary release rehearsal directories.

## Optional Real CLI Evidence

Real Codex and Claude checks remain separate, opt-in operator evidence. They must not become default release gates because local credentials, provider access, quotas, network behavior, and external CLI versions vary by operator environment.

Only record these commands when intentionally run:

```bash
ADP_SMOKE_REAL_CODEX=1 scripts/runtime-smoke.sh --real-codex
ADP_SMOKE_REAL_CLAUDE=1 scripts/runtime-smoke.sh --real-claude
```

When they are not run, record `not run` rather than treating the release evidence as incomplete.
