# Runtime Audit

This document records the runtime audit standard for the polishing phase.
It is deliberately scoped to existing ADP behavior. Comparable tools are
used as calibration only; they do not expand ADP's product scope.

## Comparable Tool Signals

Sources reviewed on 2026-06-09:

- [Claude Code CLI reference](https://code.claude.com/docs/en/cli-usage):
  emphasizes terminal entry points, command/flag discoverability,
  authentication state, session continue/resume, background sessions,
  and clear behavior for mistyped subcommands.
- [OpenAI Codex CLI docs](https://developers.openai.com/codex/cli) and
  [openai/codex](https://github.com/openai/codex): frame Codex CLI as a
  local terminal coding agent, with project rules, permissions,
  sandboxing, CLI features, and non-interactive automation as explicit
  operating surfaces.
- [aider commands](https://aider.chat/docs/usage/commands.html) and
  [aider lint/test](https://aider.chat/docs/usage/lint-test.html):
  emphasize in-session command discoverability, explicit `/run` and
  `/test` feedback loops, and configurable lint/test gates.
- [Continue CLI quickstart](https://docs.continue.dev/cli/quickstart)
  and [Continue CLI tool permissions](https://docs.continue.dev/cli/tool-permissions):
  emphasize terminal TUI/headless modes, version/help checks, resume,
  read-only planning, tool allow/deny controls, and authentication
  boundaries.

## ADP Audit Boundary

ADP remains terminal-first and local-first:

- ADP must not become a hosted dashboard, SaaS tracker, cloud sync layer,
  or provider-native resume implementation.
- ADP must not execute Git automatically. Phase `commit` and `push`
  commands record operator evidence only.
- ADP must not auto-close tasks or phases. Explicit task and phase
  commands remain required.
- Real Codex and Claude CLI checks remain opt-in operator evidence.
  Default CI and release gates stay deterministic, fake-runtime, and
  network-free.
- Runtime artifacts must stay inside ADP runtime/home paths. Project roots
  must not receive `AGENTS.md`, `CLAUDE.md`, `.codex`, `.claude`,
  planning files, or runtime manifests.

## Runtime Audit Matrix

The audit gate must cover these existing surfaces:

- CLI discoverability: root help, command help, subcommand help,
  version output, unknown-command errors, and command metadata drift.
- Workspace lifecycle: `init`, `workspace add/list/show/remove/rename/doctor`,
  and top-level `doctor`.
- Runtime entry points: `env`, `enter`, `run`, fake Codex/Claude adapters,
  shell hooks, completions, and dynamic completion values.
- Events and sessions: `events list`,
  `sessions list/show/restore-plan/resume-plan`, and restore/resume-plan
  read-only behavior, including cross-tool guidance that does not launch
  agents or copy provider-private conversation state.
- Runtime cleanup: `runtime prune` dry-run and kept-runtime coverage.
- Runtime hardening: unsafe runtime parents equal to, inside, or containing
  the project root must be rejected by runtime entry points, not only
  reported by `workspace doctor`.
- Task manager: `tasks add/list/next/show/update/claim/release/done/block`
  with text and JSON outputs where supported.
- Phase gates: `phase add/list/show/status/start/accept/commit/push`,
  including acceptance-before-commit-before-push ordering.
- Plan intake: `plan preview/apply/doctor`, stdin input, JSON output,
  failed-apply rollback, and read-only preview behavior.
- Progress reporting: `progress`, `progress report`, English and
  Simplified Chinese markdown reports, and JSON handoff output.

## Gate Commands

Use `scripts/runtime-audit-smoke.sh` for the broad runtime audit.
It builds a temporary ADP binary, uses temporary ADP home/runtime/project
directories, uses fake agents only, and verifies that the project root is
not polluted.

The aggregate local gate is:

```bash
scripts/check-all.sh
```

For phase acceptance evidence, record the actual commands run and their
result with `adp phase accept`. The phase can be committed and pushed only
after acceptance passes.
