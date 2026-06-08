# Agent Operating Guide

Simplified Chinese: [AGENTS.zh-CN.md](AGENTS.zh-CN.md)

This guide captures the working rules for agents contributing to ADP. It is a project-level contract for planning, delegation, implementation, validation, and handoff.

## Product Boundary

ADP is a terminal-first, local-first Agent Runtime Environment and Agent Workspace Manager.

Keep work aligned with this boundary:

- Build local CLI workflows, runtime overlays, workspace registry behavior, adapters, shell integration, event logs, sessions, diagnostics, and release gates.
- Do not drift into Web UI, dashboard, SaaS, cloud sync, hosted orchestration, or graphical multi-agent products.
- Real project roots must stay clean. ADP-generated files such as `AGENTS.md`, `CLAUDE.md`, `.codex/`, and `.claude/` belong in runtime overlays, not in user project roots, unless the user explicitly asks to edit this repository's own files.
- Treat external agent CLIs as compatibility boundaries. Verify current behavior before changing adapter assumptions.

## Hard Constraints

- Code files must stay at or below 700 physical lines. Split before exceeding the limit.
- Documentation defaults to English. Every maintained Markdown document must have a Simplified Chinese counterpart using `*.zh-CN.md`.
- Keep `.envrc` and `mvp.md` ignored and uncommitted.
- Do not configure repository-local Git `user.name` or `user.email`.
- Commit with one-shot identity only:

```bash
GIT_AUTHOR_NAME=karoc GIT_COMMITTER_NAME=karoc git commit -m "<message>"
```

- Push directly with:

```bash
git push
```

- The project uses PolyForm Noncommercial licensing. Do not replace or reinterpret the license model without an explicit maintainer request.

## Standard Gates

Before handoff, commit, or push, run:

```bash
scripts/check-all.sh
```

If `scripts/check-all.sh` is unavailable while bootstrapping a change, run the underlying gates:

```bash
scripts/runtime-smoke.sh --fake
scripts/example-workspace-smoke.sh
scripts/task-manager-smoke.sh
scripts/plan-intake-smoke.sh
go test -count=1 ./...
go vet ./...
scripts/check-file-lines.sh
scripts/check-docs-bilingual.sh
git diff --check
```

The line-count and bilingual-document gates include tracked files and non-ignored untracked files. Do not leave draft source, script, config, or Markdown files in the working tree unless they satisfy the same project constraints or are intentionally ignored.

## Multi-Agent Execution Standard

Use sub-agents when the user asks for parallel or multi-agent work and the task can be split into independent write scopes.

Main-thread responsibilities:

- Define the goal, constraints, and disjoint write scopes before spawning agents.
- Keep the immediate blocking integration path local.
- Do not delegate the exact same file set to multiple agents unless one is read-only review.
- Review every returned diff before integration.
- Run full repository gates after integration, not just sub-agent local checks.
- Close each phase slice before starting the next: validate it, record acceptance, commit it, push it, and record push evidence first.
- Commit and push only after the integrated tree is validated. A failed gate means the phase is still open.

Good sub-agent task boundaries:

- Runtime acceptance: `scripts/runtime-smoke.sh`, `docs/runtime-acceptance*.md`.
- Release gates: `scripts/check-all.sh`, `docs/release-checklist*.md`.
- Workspace diagnostics: `internal/workspace/diagnostics*`.
- CLI behavior: `internal/cli/` and related CLI tests.
- Examples: `examples/` and example-specific docs.
- Documentation: explicitly listed Markdown files and their `.zh-CN.md` counterparts.

Sub-agent prompts must specify:

- Objective.
- Allowed write paths.
- Disallowed paths.
- Required constraints.
- Required validation commands.
- Expected final report: files changed, behavior changed, tests run.

Read-only review agents must be told not to edit files.

## Implementation Principles

- Prefer existing package boundaries and local patterns over new abstractions.
- Keep changes scoped to the requested behavior.
- Use structured parsers and typed APIs where available.
- Keep CLI command changes aligned through the local command metadata contract. Usage text, dispatch wiring, bash/zsh completion, tests, and smoke or documentation acceptance must describe the same command surface; P16 hardens this without adopting a new CLI framework.
- Add tests proportional to risk. Broaden tests when changing shared behavior, CLI contracts, runtime behavior, or workspace safety.
- Preserve local-first behavior. Tests should use temporary `ADP_HOME`, temporary `ADP_RUNTIME_DIR`, fake binaries, and temporary project roots.
- Avoid real external CLI calls in default tests. Real Codex/Claude checks must be explicit opt-in.

## Runtime Acceptance

The deterministic runtime smoke path is:

```bash
scripts/runtime-smoke.sh --fake
```

It verifies the local runtime overlay, fake Codex/Claude launch path, event log, session history, runtime pruning, and protection against project-root pollution.

The copyable example workspace smoke path is:

```bash
scripts/example-workspace-smoke.sh
```

It verifies that `examples/basic-workspace` can be copied into a temporary `ADP_HOME`, pointed at a temporary project root, diagnosed, shown, and used to build a kept runtime overlay.

Real external CLI checks are optional release evidence and must be explicitly enabled:

```bash
ADP_SMOKE_REAL_CODEX=1 scripts/runtime-smoke.sh --real-codex
ADP_SMOKE_REAL_CLAUDE=1 scripts/runtime-smoke.sh --real-claude
```

These checks do not replace manual real-agent acceptance for credentials, models, network behavior, or interactive sessions.

## Documentation Rules

- English is the default document.
- The Simplified Chinese counterpart should carry the same operational content, not a shorter summary.
- Keep README concise and link to focused docs for details.
- When adding scripts or release gates, document when they should run and what they do not validate.
- Do not add Web/SaaS positioning.

## Phase Discipline

After a planned phase slice is complete:

1. Run the relevant runtime smoke for that phase.
2. Run `scripts/check-all.sh`.
3. Record acceptance only if those gates pass.
4. Commit the accepted phase.
5. Push the commit.
6. Record push evidence.
7. Start the next phase only after the push succeeds and the phase record is updated.

Do not mix later-phase work into the same commit. This keeps planning, execution progress, validation evidence, and Git history aligned.

## Git Workflow

Before committing:

```bash
git status --short --branch
git config --local --get-regexp '^user\.' || true
git check-ignore -v .envrc mvp.md || true
scripts/check-all.sh
git diff --check
```

After committing:

```bash
git status --short --branch
git log --oneline --decorate -5
git config --local --get-regexp '^user\.' || true
git push
```

Report the commit hash, pushed branch, gates run, and any remaining manual acceptance gaps.
