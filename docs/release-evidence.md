# Release Evidence

Simplified Chinese: [release-evidence.zh-CN.md](release-evidence.zh-CN.md)

This template records the local evidence needed before publishing an ADP release artifact. It is a release note companion, not a hosted release system, cloud ledger, SaaS workflow, provider credential check, or replacement for the local release gate. ADP release evidence must stay terminal-first, local-first, and provider-neutral by default.

## Required Fields

Record these fields for every release artifact:

- Release version, such as `1.0.1`.
- Git tag when one is created, such as `v1.0.1`.
- Git commit hash or stable source archive identifier used for the build.
- Source form, such as Git checkout or source archive.
- UTC build date.
- Go version.
- Target operating system and architecture.
- Artifact filename, such as `adp-1.0.1-linux-amd64`.
- Artifact SHA-256 checksum and the checksum command used.
- Packaged binary `adp version` output.
- `scripts/check-all.sh` result from the released source form.
- Package contents manifest, either as an attached manifest path or a concise inline excerpt.
- Install-from-artifact rehearsal result.
- Source archive or no-`.git` rehearsal result when applicable.
- Explicit list of excluded local state, credentials, logs, runtime overlays, machine-specific files, and local ADP planning ledgers.
- Failure triage notes for any required check that failed before the final passing run.
- Optional real-agent operator evidence, separated by command availability, non-interactive invocation, and manual interactive acceptance when any tier was intentionally enabled.
- License notice: ADP is source-available for noncommercial learning, research, evaluation, and open collaboration; commercial use requires separate paid authorization.

## Evidence Tiers

Required evidence is provider-free and must be present for every release candidate:

- `scripts/check-all.sh` passed from the source form used to build or publish the artifacts.
- The release binary reports explicit version, commit, build date, Go version, and platform values.
- Artifact checksum generation and verification passed.
- Package manifest inspection passed and excluded local state was absent.
- Install-from-artifact rehearsal passed with temporary ADP directories, a temporary project root, fake agent commands, and no project-root pollution.
- Source archive or no-`.git` rehearsal passed when that source form is published or used for the build.

Optional real-agent evidence is supplemental. Record `not run` for each optional tier that was not intentionally enabled. A release candidate with complete required evidence and all optional tiers marked `not run` is still complete unless the release note claims real-agent behavior beyond the deterministic fake-agent gate.

## Build Evidence

The release note should include the exact build identity:

```bash
go version
dist/adp-1.0.1-linux-amd64 version
cd dist
sha256sum -c SHA256SUMS
cd ..
```

Expected `adp version` release output shape:

```text
adp version 1.0.1
commit: <commit>
built: <timestamp>
go: <go-version>
platform: <goos>/<goarch>
```

The `commit:` value must match the Git commit or stable source archive identifier recorded in the evidence. The `built:` value must match the UTC build date recorded for the artifact. A development build that omits `commit:` or `built:` is useful for local development but is not sufficient release artifact evidence.

If a source archive does not contain `.git`, record the explicit commit or archive identifier used before building:

```bash
COMMIT=source-archive-commit
```

Do not infer build identity from an unrelated local checkout when the source archive is the source form being released.

## Install Rehearsal Evidence

Record evidence that at least one binary was installed and run from an artifact path:

```bash
ADP_INSTALL_BIN="$(mktemp -d)"
install -m 0755 dist/adp-1.0.1-linux-amd64 "${ADP_INSTALL_BIN}/adp"
export PATH="${ADP_INSTALL_BIN}:${PATH}"
adp version
```

The install rehearsal should use temporary `ADP_HOME`, temporary `ADP_RUNTIME_DIR`, a temporary project root, and fake local agent commands. It should prove the installed binary can run the local-first workflow without real provider credentials:

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

Record the files included in each package. A release package should include one target-platform `adp` binary, `README.md`, `README.zh-CN.md`, `LICENSE`, `COMMERCIAL.md`, `COMMERCIAL.zh-CN.md`, `CONTRIBUTING.md`, `CONTRIBUTING.zh-CN.md`, `SECURITY.md`, `SECURITY.zh-CN.md`, `docs/license-policy.md`, `docs/license-policy.zh-CN.md`, `docs/release-packaging.md`, `docs/release-packaging.zh-CN.md`, `docs/release-evidence.md`, `docs/release-evidence.zh-CN.md`, and a release evidence note or short release note.

Also record that the package excludes `.envrc`, `mvp.md`, `$ADP_HOME`, `$ADP_RUNTIME_DIR`, runtime overlays, logs, task state, credentials, provider-native session state, machine-specific shell startup files, local ADP planning ledgers, and temporary release rehearsal directories.

Use a sorted archive listing or equivalent package tool output as the manifest:

```bash
tar -tf adp-1.0.1-linux-amd64.tar.gz | sort
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

Do not include raw credentials, account identifiers, private prompts, provider-native session state, or sensitive model output in release evidence. Summaries should state the command, environment tier, result, and any redacted failure class needed for troubleshooting.
