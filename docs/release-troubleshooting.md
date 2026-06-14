# Release Troubleshooting

简体中文：[release-troubleshooting.zh-CN.md](release-troubleshooting.zh-CN.md)

This note is the operator triage path for preview release failures. It keeps failure handling local-first and terminal-first; it does not add hosted orchestration, dashboards, cloud sync, SaaS release tracking, automatic Git execution, provider-native resume, or default real Codex/Claude gates.

## First Response

When a required release check fails:

- Stop the release candidate. Do not tag, announce, or publish the artifact.
- Keep the failed command, exit status, relevant output, source form, commit value, `VERSION`, `BUILD_DATE`, Go version, and any environment overrides in operator notes.
- Rerun the smallest failing command from the same source form before editing. If it fails only in `scripts/check-all.sh`, inspect the aggregate ordering and temporary directory setup.
- Classify the failure as an ADP regression, documentation drift, package assembly error, source archive error, checksum mismatch, install rehearsal error, or operator environment issue.
- After a fix, rerun the failed command and then rerun `scripts/check-all.sh` before recording passing release evidence.

Do not treat optional real Codex or Claude evidence as a default gate. A real CLI failure blocks the release only when the release note claims real-agent compatibility beyond deterministic fake-provider evidence.

## Release Candidate Decision

Classify the candidate before continuing:

- `blocked`: any required gate, checksum, package manifest, install rehearsal, source archive rehearsal, or project-root cleanliness check failed.
- `passing-provider-free`: every required deterministic check passed and optional real-agent tiers are either `not run` or not claimed.
- `passing-with-real-agent-evidence`: every required deterministic check passed and the explicitly claimed optional real-agent tiers also passed.

Do not downgrade a required failure to optional evidence. Do not upgrade optional real-agent evidence into a default release requirement. If a release note claimed a real-agent tier and that tier failed, either remove that claim and record the tier as failed or deferred, or stop the candidate until the tier passes.

## Source Form Failures

For a Git checkout, start with:

```bash
git status --short --branch
git rev-parse HEAD
```

Unexpected tracked changes mean the release source is not clean. Ignored local files such as `.envrc` and `mvp.md` should remain ignored and uncommitted.

For a source archive without `.git`, build with an explicit commit value:

```bash
COMMIT=<published-commit-or-archive-id>
```

If the archive build needs files from the operator machine, rebuild the archive from the clean checkout. Do not mix archive contents with machine-local ADP state.

## Build And Version Failures

If `adp version` prints `adp dev`, the release ldflags were not injected. Rebuild with explicit `VERSION`, `COMMIT`, and `BUILD_DATE` values from [release-packaging.md](release-packaging.md).

If the reported commit or build date does not match the release evidence, discard the artifact and rebuild. Do not edit evidence to match an accidental binary.

## Checksum Failures

Checksum evidence must refer to the exact artifact that will be packaged:

```bash
sha256sum dist/adp > dist/adp.sha256
sha256sum -c dist/adp.sha256
```

If verification fails, discard the checksum and artifact pair, rebuild the artifact, regenerate the checksum, and verify again. Do not modify an artifact after recording its checksum.

## Package Manifest Failures

Inspect the archive contents before publishing:

```bash
tar -tf adp-0.1.0-preview.1-linux-amd64.tar.gz | sort
```

The package must include the binary, `README.md`, `README.zh-CN.md`, `LICENSE`, `COMMERCIAL.md`, `COMMERCIAL.zh-CN.md`, release packaging docs, release evidence docs, and a short release note. It must exclude `.envrc`, `mvp.md`, `$ADP_HOME`, `$ADP_RUNTIME_DIR`, runtime overlays, logs, task state, credentials, shell startup files, and temporary rehearsal directories.

If required files are missing, fix the clean staging directory and rebuild the package. If excluded files appear, fix the package assembly path; do not weaken the local-first boundary or publish operator state.

## Install Rehearsal Failures

Install and run from the packaged artifact path, not from the source tree:

```bash
ADP_INSTALL_BIN="$(mktemp -d)"
install -m 0755 dist/adp "${ADP_INSTALL_BIN}/adp"
PATH="${ADP_INSTALL_BIN}:${PATH}" adp version
```

If the installed binary fails but `dist/adp` succeeds, inspect file permissions, package extraction, target platform, and `PATH` ordering. The rehearsal must use temporary `ADP_HOME`, temporary `ADP_RUNTIME_DIR`, a temporary project root, and fake provider commands unless optional real CLI evidence was intentionally enabled.

If the project-root pollution scan finds ADP files, fix runtime or planning output boundaries. Do not accept `AGENTS.md`, `CLAUDE.md`, `.codex`, `.claude`, `.adp-runtime.yaml`, `planning`, task files, phase files, or progress reports in the real project root.

## Gate Failures

For `scripts/check-all.sh`, inspect the first failing child command and rerun that child directly from the same source form. The aggregate gate is the release decision point, but the smallest failing command is usually the fastest triage target.

For `scripts/release-artifact-smoke.sh`, inspect package staging, checksums, manifest assertions, install-from-artifact, source archive `COMMIT`, fake Codex command, temporary ADP directories, and project-root pollution output first.

For `scripts/release-operator-drill-smoke.sh`, inspect the no-`.git` source copy, documented release commands, release script syntax checks, explicit commit build, checksum verification, installed `PATH` binary, fake Codex handoff sequence, local phase evidence records, fake Git tripwire, and project-root pollution scan.

For `scripts/install-onboarding-smoke.sh`, inspect deterministic build metadata, temporary `GOBIN` install, `PATH` ordering, temporary `ADP_HOME`, temporary `ADP_RUNTIME_DIR`, workspace registration, fake Codex and fake Claude commands, task-bound evidence, fake Git tripwire output, and project-root pollution scan.

For `scripts/release-rehearsal-smoke.sh`, inspect the clean workspace copy, release ldflags, copied example workspace bootstrap, isolated runtime directories, and fake Git tripwire output.

For `scripts/check-docs-bilingual.sh`, add the missing English default or Simplified Chinese counterpart. For `scripts/check-file-lines.sh`, split the reported code file before adding behavior. For `git diff --check`, remove whitespace errors or conflict markers.

When in doubt, keep the failure narrow: rerun the failing step, fix the local cause, and rerun the aggregate gate. Do not solve release failures by adding new product scope.

## Optional Real-Agent Failures

For command availability failures, verify the external CLI is installed on `PATH`, then rerun only the opted-in real flag. Do not add the real flag to `scripts/check-all.sh` or CI.

For non-interactive invocation failures, record whether the failure was command launch, authentication, quota, network, model access, prompt handling, timeout, or unexpected output. These failures describe the operator environment unless the deterministic fake-agent gate also fails.

For manual interactive acceptance failures, keep notes redacted and separate from package evidence. Do not include credentials, account identifiers, private prompts, provider-native session files, or sensitive model output in release artifacts.
