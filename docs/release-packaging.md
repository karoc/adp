# Release Packaging

简体中文：[release-packaging.zh-CN.md](release-packaging.zh-CN.md)

This note defines the stable release packaging path for ADP as a terminal-first, local-first Go CLI. It keeps release artifacts aligned with the local runtime model and does not introduce hosted services, dashboards, cloud sync, or SaaS deployment assumptions.

## Release Gate

Run the same aggregate gate locally and in CI before preparing an artifact:

```bash
scripts/check-all.sh
```

The gate covers fake runtime acceptance, broad runtime audit smoke, focused runtime context smoke, release readiness smoke, release rehearsal smoke, release artifact smoke, release operator drill smoke, install onboarding smoke, example workspace smoke, task manager smoke, plan intake smoke, Go test and vet, file-line limits, bilingual documentation pairing, and whitespace checks. CI intentionally calls this same script so release evidence is not split between a local path and a separate GitHub Actions path.

Optional real Codex or Claude CLI checks remain operator evidence only:

```bash
ADP_SMOKE_REAL_CODEX=1 scripts/runtime-smoke.sh --real-codex
ADP_SMOKE_REAL_CLAUDE=1 scripts/runtime-smoke.sh --real-claude
```

They do not replace the aggregate gate and do not prove provider credentials, model access, quota, network reliability, or interactive session quality.

## Release Candidate Summary

The release-candidate path is deterministic by default:

1. Verify the exact source form with `scripts/check-all.sh`.
2. Build artifacts with the release script's explicit `-trimpath` and release ldflags.
3. Verify the generated checksums before packaging.
4. Stage the package from clean files only.
5. Inspect the package manifest for required notices and excluded local state.
6. Install a packaged binary on a temporary `PATH`.
7. Run the provider-free first-run rehearsal with fake agent commands and temporary ADP directories.
8. Record release evidence after the gate, checksum, manifest, install rehearsal, and source archive or no-`.git` rehearsal pass.

This path must not depend on real Codex or Claude CLIs, provider credentials, network access, hosted services, dashboards, cloud sync, or automatic Git execution. Optional real-agent evidence can be attached to a release note, but it is not part of the package assembly requirement unless the release explicitly claims that tier.

## Operator Drill

Use this sequence for stable release rehearsals:

1. Start from a clean Git checkout and record `git status --short --branch` and the commit hash. If a source archive without `.git` will also be published or used for builds, record the archive origin and set `COMMIT` explicitly before the no-`.git` build rehearsal.
2. Run `scripts/check-all.sh` from the clean checkout used to produce the artifacts or source archive. If an archive is missing test scripts or Go module files, rebuild it from that clean checkout instead of filling gaps from machine-local files.
3. Build the release artifacts with `scripts/build-release.sh`, using explicit `VERSION`, `COMMIT`, and `BUILD_DATE` values when the defaults do not describe the exact source form being released.
4. Verify the generated SHA-256 checksums for the artifacts that will be packaged.
5. Assemble the package from a clean staging directory, then record a sorted package manifest before publishing.
6. Install at least one packaged binary into a temporary directory on `PATH` and run the provider-free first-run rehearsal from that installed path.
7. Record release evidence only after the gate, checksum verification, package manifest inspection, install rehearsal, and source archive or no-`.git` rehearsal have passed.

If any required step fails, stop the release candidate, keep the failed command and output in the operator notes, and use [release-troubleshooting.md](release-troubleshooting.md). Do not repair a release failure by adding hosted orchestration, Web UI, cloud sync, automatic Git execution, provider-native resume, project-root planning exports, or default real Codex/Claude requirements.

## Build Artifacts

The canonical multi-platform release build path is `scripts/build-release.sh` from the repository root:

```bash
VERSION=1.0.1 ./scripts/build-release.sh
```

The script builds Linux, macOS, and Windows artifacts under `dist/`, injects release identity through `-ldflags`, uses `-trimpath`, strips debug metadata with `-s -w`, and writes `dist/SHA256SUMS`. By default it uses `VERSION=1.0.1`, the current Git commit from `git rev-parse HEAD`, and a UTC build timestamp. Operators may override all three values:

```bash
VERSION=1.0.1 \
COMMIT="$(git rev-parse HEAD)" \
BUILD_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
./scripts/build-release.sh
```

When debugging the release script from a single-platform source form, this manual fallback exposes the same release identity contract. Use it to isolate build or archive problems, then return to `scripts/build-release.sh` for published multi-platform artifacts:

```bash
VERSION="${VERSION:-1.0.1}"
COMMIT="${COMMIT:-source-archive-commit}"
BUILD_DATE="${BUILD_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"
LDFLAGS="-s -w -X github.com/karoc/adp/internal/cli.Version=${VERSION} -X github.com/karoc/adp/internal/cli.Commit=${COMMIT} -X github.com/karoc/adp/internal/cli.BuildDate=${BUILD_DATE}"
mkdir -p dist
go build -trimpath -ldflags="$LDFLAGS" -o dist/adp ./cmd/adp
sha256sum dist/adp > dist/adp.sha256
sha256sum -c dist/adp.sha256
ADP_INSTALL_BIN="$(mktemp -d)"
install -m 0755 dist/adp "${ADP_INSTALL_BIN}/adp"
```

The expected text output from a packaged release binary is multiline and begins with the injected version:

```text
adp version 1.0.1
commit: <commit>
built: <utc-timestamp>
go: <go-version>
platform: <goos>/<goarch>
```

The `-X` values target package variables in `github.com/karoc/adp/internal/cli`. A local source build without release ldflags uses the source default version and omits `commit:` and `built:` when those values were not injected; release artifacts should inject all three identity values so operators can connect a binary to the Git commit and build timestamp.

The default `COMMIT` lookup assumes a Git checkout. If building from a source archive without `.git`, set `COMMIT` explicitly before running the build script:

```bash
COMMIT=source-archive-commit VERSION=1.0.1 ./scripts/build-release.sh
```

When a source archive is used, the explicit `COMMIT` value should be the published commit hash or another stable archive identifier recorded in the release evidence. Do not infer build identity from a local checkout that is not the source form being released.

After building artifacts, verify checksums before distributing them:

```bash
cd dist
sha256sum -c SHA256SUMS
```

If the operator platform does not provide `sha256sum`, use an equivalent SHA-256 tool and record the exact command in the release evidence note.

## Install From Artifact

Validate at least one packaged binary from an installed location rather than by running directly from the source tree:

```bash
ADP_INSTALL_BIN="$(mktemp -d)"
install -m 0755 dist/adp-1.0.1-linux-amd64 "${ADP_INSTALL_BIN}/adp"
export PATH="${ADP_INSTALL_BIN}:${PATH}"
adp version
```

Then run a provider-free first-run rehearsal with temporary `ADP_HOME`, temporary `ADP_RUNTIME_DIR`, a temporary project root, and a fake local `codex` command. The rehearsal should prove the installed binary can initialize ADP state, register a workspace, pass doctor checks, run `adp run codex --workspace <name> --task <task-id> -- <agent-args>`, inspect events and sessions, and leave the real project root free of ADP-generated files such as `AGENTS.md`, `CLAUDE.md`, `.codex`, `.claude`, `.adp-runtime.yaml`, `planning`, `tasks.yaml`, `phases.yaml`, or `progress.jsonl`.

Use fake local agent commands for this default rehearsal. If real Codex or Claude evidence is collected, run it as a separate opt-in tier after the provider-free rehearsal has already passed and record the exact tier in [release-evidence.md](release-evidence.md).

## Package Contents

Each packaged archive should include:

- The `adp` binary for one target platform.
- `README.md`.
- `README.zh-CN.md`.
- `LICENSE`.
- `COMMERCIAL.md`.
- `COMMERCIAL.zh-CN.md`.
- `CONTRIBUTING.md`.
- `CONTRIBUTING.zh-CN.md`.
- `SECURITY.md`.
- `SECURITY.zh-CN.md`.
- `docs/license-policy.md`.
- `docs/license-policy.zh-CN.md`.
- `docs/release-packaging.md`.
- `docs/release-packaging.zh-CN.md`.
- `docs/release-evidence.md`.
- `docs/release-evidence.zh-CN.md`.
- A short release note with the Git commit, target platform, and gate evidence.

Keep `LICENSE`, `COMMERCIAL.md`, `CONTRIBUTING.md`, `SECURITY.md`, and `docs/license-policy.md` intact in every package. ADP is source-available for noncommercial learning, research, evaluation, and open collaboration under the public license; commercial use requires separate paid authorization from the copyright holder.

Do not include local `.envrc`, `mvp.md`, `$ADP_HOME`, `$ADP_RUNTIME_DIR`, runtime overlays, logs, task state, credentials, machine-specific shell startup files, or temporary release rehearsal directories.

Do not include provider credentials, account identifiers, private prompts, model output transcripts, provider-native session state, or machine-local ADP planning ledgers. Release evidence may summarize optional real-agent results, but the package itself should remain provider-neutral and safe to inspect without access to the operator environment.

Record a package manifest before publishing, for example:

```bash
tar -tf adp-1.0.1-linux-amd64.tar.gz | sort > adp-1.0.1-linux-amd64.manifest
```

Inspect the manifest before release. A manifest mismatch is a packaging failure, not a reason to weaken repository ignores or include local operator state.

## Release Scope

Stable release packages are local CLI artifacts. Users should install the binary somewhere on `PATH`, run `adp init`, register local workspaces, and keep agent configuration under `$ADP_HOME`.

The package should not claim:

- Hosted orchestration.
- Web or dashboard management.
- Cloud synchronization.
- Remote issue tracker synchronization.
- Managed Codex or Claude provider access.
- Production certification for external agent CLIs.

## Tagging Notes

Use explicit stable release tags, for example `v1.0.1`, only after the working tree is clean and the release gate has passed. The tag should point at the same commit used to build the binary artifacts.

Before publishing a release, record the evidence described in [release-evidence.md](release-evidence.md), including:

- Commit hash.
- Source form used for the build, such as a Git checkout or source archive.
- Target platform and architecture.
- Go version.
- `adp version` output from the packaged binary.
- Artifact filename and SHA-256 checksum.
- `scripts/check-all.sh` result.
- Install-from-artifact rehearsal result.
- Source archive or no-`.git` rehearsal result when applicable.
- Package manifest path or inline manifest excerpt.
- Any optional real CLI evidence that was intentionally collected.
