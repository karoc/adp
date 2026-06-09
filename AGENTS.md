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
- Use `scripts/check-file-lines.sh --audit` as a non-blocking pressure report when planning split or hardening phases. It does not replace the required hard gate.
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
scripts/runtime-audit-smoke.sh
scripts/runtime-context-smoke.sh
scripts/release-readiness-smoke.sh
scripts/release-rehearsal-smoke.sh
scripts/release-artifact-smoke.sh
scripts/release-operator-drill-smoke.sh
scripts/install-onboarding-smoke.sh
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
- Use ADP as the shared task board. Prefer `adp run <agent> --take --owner <owner> [--lease 4h]` when a worker should atomically pick up work at launch time. Use `adp tasks take` for manual pickup without launching an agent.
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
- ADP task ownership expectations, including whether the worker should use `adp run --take`, `adp tasks take`, or an explicitly assigned task ID.
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

## Tool Plan Mode

Provider-native plan mode and plan panels are proposal surfaces. They can help an agent show candidate work, but ADP remains the authoritative local planning and progress ledger.

When operating in plan mode, do not edit implementation files, complete tasks, accept phases, commit, push, or otherwise execute the plan unless the user explicitly approves moving beyond planning. Validate structured proposals with the read-only path:

```bash
adp plan preview --workspace <workspace> --file - --format json
```

Apply a plan only after explicit user or operator approval:

```bash
adp plan apply --workspace <workspace> --file - --format json
```

After an approved plan is applied, continue to use ADP task and phase commands for durable task ownership, progress, blockers, acceptance, commit evidence, and push evidence. Native plan panels may mirror ADP items for readability, but they are scratch views only.

If a tool is launched in plan mode through `adp run --take`, the taken task is the active ADP-owned work item for that session, but the provider-native plan remains a proposal view. The worker must not mark the task done, accept a phase, commit, push, or run Git just because a native plan item was checked off or the provider session exited.

## Runtime Acceptance

The deterministic runtime smoke path is:

```bash
scripts/runtime-smoke.sh --fake
```

It verifies the local runtime overlay, fake Codex/Claude launch path, event log, session history, runtime pruning, and protection against project-root pollution.

The broad runtime audit path is:

```bash
scripts/runtime-audit-smoke.sh
```

It verifies the published CLI command surface, help output, JSON parseability, task/phase/plan/progress flows, sessions, restore planning, completion values, and local-first runtime boundaries using fake agents and temporary directories only.

The focused runtime context smoke path is:

```bash
scripts/runtime-context-smoke.sh
```

It verifies launch-time context through generated instruction files, adapter metadata, selected profiles, prompt, shared memory, MCP references, task metadata, runtime environment variables, local event/session evidence, workspace diagnostics, and project-root cleanliness.

The release readiness smoke path is:

```bash
scripts/release-readiness-smoke.sh
```

It verifies release-gate invariants that are not tied to a real provider CLI, including that phase commit and push commands record evidence without executing Git.

The release rehearsal smoke path is:

```bash
scripts/release-rehearsal-smoke.sh
```

It copies the current non-ignored repository files into a temporary clean workspace, builds a preview binary with release ldflags, verifies copied docs and file limits, bootstraps the copied example workspace with isolated ADP paths, and checks phase evidence recording with a fake Git tripwire.

The release artifact smoke path is:

```bash
scripts/release-artifact-smoke.sh
```

It verifies package staging, checksums, manifest boundaries, install-from-artifact behavior, provider-free first-run rehearsal, and source archive builds without relying on `.git`.

The release operator drill smoke path is:

```bash
scripts/release-operator-drill-smoke.sh
```

It verifies documented release commands, no-`.git` operator source handling, release script syntax checks, explicit commit build metadata, checksum verification, installed `PATH` behavior, fake Codex handoff, local phase evidence, fake Git tripwire protection, and project-root cleanliness.

The install onboarding smoke path is:

```bash
scripts/install-onboarding-smoke.sh
```

It verifies local install into a temporary `GOBIN`, `PATH` precedence, first-use workspace registration, fake Codex/Claude command handling, task-bound context, local event/session/progress evidence, Git side-effect guards, and project-root cleanliness.

The copyable example workspace smoke path is:

```bash
scripts/example-workspace-smoke.sh
```

It verifies that `examples/basic-workspace` can be copied into a temporary `ADP_HOME`, pointed at a temporary project root, diagnosed, shown, and used to build a kept runtime overlay.

The task manager smoke path is:

```bash
scripts/task-manager-smoke.sh
```

It verifies workspace-local task, phase, planning doctor, next-work, progress, progress report, local phase evidence, read-only report generation, and project-root pollution protection.

The plan intake smoke path is:

```bash
scripts/plan-intake-smoke.sh
```

It verifies local structured plan preview/apply from files and stdin, explicit ledger writes under `$ADP_HOME`, rollback on failed or duplicate apply, read-only preview, JSON inspection output, and no runtime, Git, event-log, or project-root side effects.

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

## Current Project Dogfooding

ADP development uses ADP's own local planning ledger for P24 and later work. Treat the `adp` workspace as the execution source of truth:

- Register each new implementation slice as a phase and prioritized tasks before starting it.
- Keep the authoritative phase/task/progress records under `$ADP_HOME`; do not export planning state into the repository root as a normal workflow.
- Use `adp run <agent> --workspace adp --take --owner <owner> --lease <duration> -- <agent-args>` for launch-time atomic pickup, and use `adp tasks next --workspace adp --limit 0 --format json` plus `adp phase status --workspace adp --format json` as local handoff snapshots for main-thread and sub-agent coordination.
- When Codex, Claude, or another tool exposes a native task/todo panel, mirror the active ADP task there for visibility, but keep durable status, ownership, progress, and recovery evidence in ADP.
- When a tool exposes plan mode, use it only to draft or display candidate plans until the proposal passes `adp plan preview` and receives explicit approval for `adp plan apply`.
- Do not start a later phase until the current phase has passed validation, recorded acceptance, been committed, been pushed, and recorded commit plus push evidence.
- Repository docs may summarize accepted behavior, but they are not the execution ledger.

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
