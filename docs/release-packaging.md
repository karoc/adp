# Release Packaging

Simplified Chinese: [release-packaging.zh-CN.md](release-packaging.zh-CN.md)

This note defines the early preview packaging path for ADP as a terminal-first, local-first Go CLI. It keeps release artifacts aligned with the local runtime model and does not introduce hosted services, dashboards, cloud sync, or SaaS deployment assumptions.

## Release Gate

Run the same aggregate gate locally and in CI before preparing an artifact:

```bash
scripts/check-all.sh
```

The gate covers fake runtime acceptance, broad runtime audit smoke, release readiness smoke, release rehearsal smoke, release artifact smoke, release operator drill smoke, example workspace smoke, task manager smoke, plan intake smoke, Go test and vet, file-line limits, bilingual documentation pairing, and whitespace checks. CI intentionally calls this same script so release evidence is not split between a local path and a separate GitHub Actions path.

Optional real Codex or Claude CLI checks remain operator evidence only:

```bash
ADP_SMOKE_REAL_CODEX=1 scripts/runtime-smoke.sh --real-codex
ADP_SMOKE_REAL_CLAUDE=1 scripts/runtime-smoke.sh --real-claude
```

They do not replace the aggregate gate and do not prove provider credentials, model access, quota, network reliability, or interactive session quality.

## Operator Drill

Use this sequence for preview release rehearsals:

1. Start from a clean Git checkout and record `git status --short --branch` and the commit hash. If a source archive without `.git` will also be published or used for builds, record the archive origin and set `COMMIT` explicitly before the no-`.git` build rehearsal.
2. Run `scripts/check-all.sh` from the clean checkout used to produce the artifacts or source archive. If an archive is missing test scripts or Go module files, rebuild it from that clean checkout instead of filling gaps from machine-local files.
3. Build the target-platform artifact with explicit `VERSION`, `COMMIT`, and `BUILD_DATE` values.
4. Generate and verify the SHA-256 checksum for the artifact that will be packaged.
5. Assemble the package from a clean staging directory, then record a sorted package manifest before publishing.
6. Install at least one packaged binary into a temporary directory on `PATH` and run the provider-free first-run rehearsal from that installed path.
7. Record release evidence only after the gate, checksum verification, package manifest inspection, install rehearsal, and source archive or no-`.git` rehearsal have passed.

If any required step fails, stop the release candidate, keep the failed command and output in the operator notes, and use [release-troubleshooting.md](release-troubleshooting.md). Do not repair a release failure by adding hosted orchestration, Web UI, cloud sync, automatic Git execution, provider-native resume, project-root planning exports, or default real Codex/Claude requirements.

## Build Artifacts

For an early preview binary, build the CLI from the repository root:

```bash
mkdir -p dist
VERSION=${VERSION:-0.1.0-preview.1}
COMMIT=${COMMIT:-$(git rev-parse --short HEAD)}
BUILD_DATE=${BUILD_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}

LDFLAGS="-s -w"
LDFLAGS="$LDFLAGS -X github.com/karoc/adp/internal/cli.Version=$VERSION"
LDFLAGS="$LDFLAGS -X github.com/karoc/adp/internal/cli.Commit=$COMMIT"
LDFLAGS="$LDFLAGS -X github.com/karoc/adp/internal/cli.BuildDate=$BUILD_DATE"

go build -trimpath -ldflags="$LDFLAGS" -o dist/adp ./cmd/adp
dist/adp version
```

The expected release output shape is `adp 0.1.0-preview.1 commit <commit> built <utc-timestamp>`. The `-X` values target package variables in `github.com/karoc/adp/internal/cli`. When they are omitted, `adp version` falls back to the development identity `dev`; release artifacts should inject all three values so operators can connect a binary to the Git commit and build timestamp.

The default `COMMIT` command assumes a Git checkout. If building from a source archive without `.git`, set `COMMIT` explicitly before running the build command:

```bash
COMMIT=source-archive-commit
```

For cross-platform preview artifacts, set `GOOS` and `GOARCH` explicitly and use platform-specific names:

```bash
GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="$LDFLAGS" -o dist/adp-linux-amd64 ./cmd/adp
GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="$LDFLAGS" -o dist/adp-darwin-arm64 ./cmd/adp
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="$LDFLAGS" -o dist/adp-windows-amd64.exe ./cmd/adp
```

After building an artifact, generate and verify a checksum before distributing it:

```bash
sha256sum dist/adp > dist/adp.sha256
sha256sum -c dist/adp.sha256
```

If the operator platform does not provide `sha256sum`, use an equivalent SHA-256 tool and record the exact command in the release evidence note.

## Install From Artifact

Validate at least one packaged binary from an installed location rather than by running directly from the source tree:

```bash
ADP_INSTALL_BIN="$(mktemp -d)"
install -m 0755 dist/adp "${ADP_INSTALL_BIN}/adp"
export PATH="${ADP_INSTALL_BIN}:${PATH}"
adp version
```

Then run a provider-free first-run rehearsal with temporary `ADP_HOME`, temporary `ADP_RUNTIME_DIR`, a temporary project root, and a fake local `codex` command. The rehearsal should prove the installed binary can initialize ADP state, register a workspace, pass doctor checks, run `adp run codex --workspace <name> --task <task-id> -- <agent-args>`, inspect events and sessions, and leave the real project root free of ADP-generated files such as `AGENTS.md`, `CLAUDE.md`, `.codex`, `.claude`, or `planning`.

## Package Contents

Each packaged archive should include:

- The `adp` binary for one target platform.
- `README.md`.
- `README.zh-CN.md`.
- `LICENSE`.
- `COMMERCIAL.md`.
- `COMMERCIAL.zh-CN.md`.
- `docs/release-packaging.md`.
- `docs/release-packaging.zh-CN.md`.
- `docs/release-evidence.md`.
- `docs/release-evidence.zh-CN.md`.
- A short release note with the Git commit, target platform, and gate evidence.

Keep `LICENSE` and `COMMERCIAL.md` intact in every package. ADP is source-available for noncommercial learning, research, evaluation, and open collaboration under the public license; commercial use requires separate paid authorization from the copyright holder.

Do not include local `.envrc`, `mvp.md`, `$ADP_HOME`, `$ADP_RUNTIME_DIR`, runtime overlays, logs, task state, credentials, machine-specific shell startup files, or temporary release rehearsal directories.

Record a package manifest before publishing, for example:

```bash
tar -tf adp-0.1.0-preview.1-linux-amd64.tar.gz | sort > adp-0.1.0-preview.1-linux-amd64.manifest
```

Inspect the manifest before release. A manifest mismatch is a packaging failure, not a reason to weaken repository ignores or include local operator state.

## Preview Scope

Early preview packages are local CLI artifacts. Users should install the binary somewhere on `PATH`, run `adp init`, register local workspaces, and keep agent configuration under `$ADP_HOME`.

The package should not claim:

- Hosted orchestration.
- Web or dashboard management.
- Cloud synchronization.
- Remote issue tracker synchronization.
- Managed Codex or Claude provider access.
- Production certification for external agent CLIs.

## Tagging Notes

Use explicit preview tags, for example `v0.1.0-preview.1`, only after the working tree is clean and the release gate has passed. The tag should point at the same commit used to build the binary artifacts.

Before publishing a preview, record the evidence described in [release-evidence.md](release-evidence.md), including:

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
