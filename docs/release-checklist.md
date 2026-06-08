# Release Checklist

Simplified Chinese: [release-checklist.zh-CN.md](release-checklist.zh-CN.md)

This checklist defines the local release gate for ADP. It keeps release validation aligned with the project boundary: ADP is a terminal-first, local-first Agent Runtime Environment and Agent Workspace Manager.

The release gate verifies ADP's own runtime, CLI, workspace, diagnostics, documentation, and repository hygiene. It does not turn release validation into a hosted service check, Web UI check, SaaS deployment check, or remote provider certification process.

## Required Gate

Run the unified gate before handoff, commit, push, or release candidate tagging:

```bash
scripts/check-all.sh
```

The script can be called from any current directory. It resolves the repository root from its own location before running checks.

The required gate runs these checks in order:

```bash
scripts/runtime-smoke.sh --fake
scripts/example-workspace-smoke.sh
go test -count=1 ./...
go vet ./...
scripts/check-file-lines.sh
scripts/check-docs-bilingual.sh
git diff --check
```

## Gate Coverage

`scripts/runtime-smoke.sh --fake` builds the current `cmd/adp` binary into a temporary directory and runs the deterministic fake-agent runtime acceptance path. It uses temporary `ADP_HOME`, `ADP_RUNTIME_DIR`, fake agent binaries, and a temporary project root.

The fake runtime smoke verifies:

- Runtime overlay creation.
- Runtime environment variables.
- Codex and Claude adapter launch paths through fake binaries.
- Event log writes.
- Session history queries.
- Workspace diagnostics.
- Shell export rendering.
- Bash and zsh completion rendering.
- ADP-owned runtime pruning.
- Protection against polluting the real project root with runtime artifacts.

`scripts/example-workspace-smoke.sh` builds the current `cmd/adp` binary, copies `examples/basic-workspace` into a temporary `ADP_HOME`, rewrites the copied `project.root` to a temporary project, and verifies `adp init`, `workspace doctor`, `workspace show`, and `env --cd` against that copied example.

The example workspace smoke verifies:

- The published example can be copied without depending on repository-local state.
- The example workspace schema remains compatible with the current CLI.
- A temporary project root can be linked into a kept runtime overlay.
- Example documentation and release claims stay connected to an executable path.

`go test -count=1 ./...` verifies the full Go test suite without using cached test results.

`go vet ./...` runs Go static checks.

`scripts/check-file-lines.sh` enforces the project rule that code files stay at or below 700 physical lines. It checks tracked files and non-ignored untracked files.

`scripts/check-docs-bilingual.sh` enforces the documentation pairing rule for tracked Markdown files and non-ignored untracked Markdown files. English is the default document, and maintained Markdown files need Simplified Chinese counterparts using `*.zh-CN.md`.

`git diff --check` catches whitespace errors in the current diff.

## Optional Real CLI Evidence

Real Codex and Claude CLI checks are not part of the default gate. They are opt-in release evidence because local installations, credentials, model availability, network access, and interactive behavior vary by operator environment.

Run the lightweight real Codex check only when the local Codex CLI is intentionally part of the release evidence:

```bash
ADP_SMOKE_REAL_CODEX=1 scripts/runtime-smoke.sh --real-codex
```

Run the lightweight real Claude check only when the local Claude CLI is intentionally part of the release evidence:

```bash
ADP_SMOKE_REAL_CLAUDE=1 scripts/runtime-smoke.sh --real-claude
```

The real CLI smoke confirms that the external command exists and that a lightweight `--version` or `--help` invocation completes. It does not prove that a full interactive agent session, provider credentials, account quota, model selection, external tool permission, or network path is ready.

When real CLI evidence is collected, record:

- The command that was run.
- The Codex or Claude CLI version when available.
- The operating system and shell.
- Any environment overrides such as `ADP_SMOKE_CODEX_BIN` or `ADP_SMOKE_CLAUDE_BIN`.
- Whether a separate manual interactive session was completed.

## Failure Triage

If `scripts/runtime-smoke.sh --fake` fails, inspect the reported step first. The fake smoke is the highest-signal check for runtime overlay behavior, adapter launch paths, local event history, session aggregation, and project-root pollution.

If `scripts/example-workspace-smoke.sh` fails, inspect whether the copied `examples/basic-workspace/workspace.yaml` still matches the current schema and whether `adp env <workspace> --cd` still produces a kept runtime with project-file symlinks.

If `go test -count=1 ./...` fails, narrow the failing package and rerun that package before making changes:

```bash
go test -count=1 ./internal/workspace
go test -count=1 ./internal/cli
go test -count=1 ./test/e2e
```

If `go vet ./...` fails, treat it as a code-quality gate and fix the reported package before rerunning the full gate.

If `scripts/check-file-lines.sh` fails, split the reported code file before adding more behavior. Do not bypass the 700-line limit with generated-looking manual files or unrelated formatting churn. If the reported file is a scratch file, remove it or ignore it deliberately.

If `scripts/check-docs-bilingual.sh` fails, add the missing English default or Simplified Chinese counterpart. New Markdown files should follow the default English path plus `*.zh-CN.md` counterpart pattern. If the reported Markdown is a local note, remove it or ignore it deliberately.

If `git diff --check` fails, remove trailing whitespace or conflict markers from the reported files.

## Manual Release Checks

Before a release candidate is announced, an operator should also confirm:

- `git status --short --branch` shows only intentional changes before commit and a clean tree after commit.
- `.envrc` and `mvp.md` remain ignored and uncommitted.
- Repository-local Git identity is not configured with `user.name` or `user.email`.
- The license files and PolyForm Noncommercial positioning were not changed unintentionally.
- README and focused docs describe the current CLI surface without Web, UI, SaaS, cloud sync, or hosted orchestration drift.
- Any claimed real-agent compatibility has matching opt-in real CLI evidence and, when needed, manual interactive acceptance notes.

## Out Of Scope

The release gate does not validate:

- Provider accounts or billing.
- Remote model availability.
- External network reliability.
- Real interactive Codex or Claude session quality.
- User-specific shell startup files.
- Hosted deployment, SaaS operations, dashboards, or Web UI behavior.

Those checks belong to operator-specific acceptance notes, not the default local release gate.
