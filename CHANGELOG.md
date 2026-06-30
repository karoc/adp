# Changelog

All notable changes to ADP (Agent Development Platform) will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Security
- **Redact secrets from agent arguments in event logs and resume planning** — Arguments passed after `--` to `adp run` (for example `--api-key sk-...`) were recorded verbatim in the world-readable `events.jsonl` and echoed back by `sessions restore-plan` / `sessions resume-plan`. Secret-looking values are now masked with `***REDACTED***` while the flag name is preserved, the event log is created with owner-only `0600` permissions (and pre-existing loose permissions are tightened on next write), and the redaction is applied both at write time and when plans are rendered (text and JSON). A new `internal/redact` package detects credentials by sensitive flag name (`key`, `secret`, `token`, `password`, `auth`, `credential`, ...), known provider prefixes (`sk-`, `ghp_`, `AKIA`, `eyJ`, ...), and high-entropy bare values.
- **Restrict `$ADP_HOME` to owner-only permissions** — The ADP home tree (`$ADP_HOME`, `workspaces/`, `logs/`, and each workspace's `planning/`) was created world-readable (`0755`), and planning ledgers (`tasks.yaml`, `phases.yaml`, `progress.jsonl`) were written `0644`, exposing project paths, task content, command history, and git remotes (which may embed tokens) to other local users (CWE-732). All ADP-managed directories are now created `0700` and tightened on next access if pre-existing, and planning ledgers are written `0600`. A new `internal/paths.EnsurePrivateDir` helper centralizes the directory hardening.
- **Restrict runtime overlay to owner-only permissions** — The runtime overlay tree under `$ADP_RUNTIME_DIR` (default `$TMPDIR/adp-runtime`, a shared temp location) was materialized world-readable: directories `0755` and generated agent config files (`CLAUDE.md`, `AGENTS.md`, `.codex/config.toml`, `.claude/settings.json`) `0644`. These files are rendered from workspace prompts, memory, and profiles, so other local users could read them (CWE-732). The runtime root and generated parent directories are now created `0700`, and a group/other mask is applied to every generated file so adapter-requested modes can no longer widen access beyond the owner (effectively `0600`).
- **Sanitize terminal control characters in rendered planning text** — Task titles, descriptions, phase goals, owners, and other user-controlled planning fields were written directly to the terminal without filtering control characters (CWE-150). Because the shared workspace planning YAML can be edited by hand, a malicious entry could embed ANSI escape sequences that clear the screen, move the cursor, or spoof output when another operator runs `adp tasks list`/`show`, `adp phase list`/`show`, or `adp progress report`. A new `safeText` helper now neutralizes control runes (ESC, CR, BEL, backspace, C1 range) while preserving legitimate newline/tab; it is applied at every text sink (the `valueOrDash`, `formatStringList`, and `markdownCell` helpers, plus direct field prints in the task, phase, plan, progress, and sessions resume-plan renderers — including guidance lines that embed task `blocked_reason`/`owner`/`id` and phase-gate reasons). JSON output already escapes control characters via the encoder.
- **Complete terminal-control sanitization for planning identifiers and remaining report fields** — The earlier `safeText` pass covered task/phase titles, owners, and goals but left sibling fields from the same hand-editable planning YAML unsanitized: task and phase identifiers (printed in list/show/status tables, "Next steps" suggested commands, and the `phase status` gate summary), the progress report's "Next Work" bullet list (task id, title, and owner rendered via the claim-handoff helper), phase lifecycle status messages (commit hash, push remote/branch/result), and the "ambiguous task ID" error list. Because the shared planning YAML is editable by hand, a malicious entry could embed ANSI escape sequences in any of these to clear the screen or spoof output when another operator lists or shows tasks/phases. `safeText` is now applied at all of these display sinks (a new `formatAmbiguousTaskIDList` helper centralizes the sanitized error-list rendering); JSON output remains covered by the encoder. Identifiers used as lookup keys are intentionally left unsanitized so prefix matching is unaffected.
- **Close TOCTOU race in the planning lock's stale-lock recovery** — The planning store serializes mutations with an `O_EXCL` lock file and breaks locks older than 30 minutes (left behind by a crashed process). The stale-break path used `os.Remove` followed by a retry, so two concurrent commands that both judged the same lock stale could each remove a lock the other had just recreated, letting both enter the critical section and causing lost updates or double-claimed tasks (CWE-367). The break now atomically claims the stale lock with `os.Rename` (into a `.lock.stale` trash file); a breaker that lost the race gets `os.ErrNotExist` and retries the exclusive create, so mutual exclusion holds across all interleavings. Atomic tmp-file-then-rename writes already prevented torn files; this closes the remaining mutual-exclusion gap.

### Phase 1-5 Foundation (Pre-release Development)

The following sections document ADP's evolution from initial concept to production-ready 1.0 candidate. All features have undergone systematic acceptance testing and are considered stable for local terminal-first AI agent workflows.

---

## Phase 1: Core Runtime & Workspace Foundation

**Status**: ✅ Completed
**Focus**: Terminal-first runtime isolation and workspace management

### Added
- **Core CLI** — `adp` binary with subcommand architecture
- **Workspace Management**
  - `adp workspace add/list/show/remove/rename` — register and manage workspaces
  - `adp workspace doctor` — comprehensive configuration diagnostics
  - `adp doctor` — global health check across all workspaces
  - Workspace configuration under `$ADP_HOME/workspaces/`
  - Project root validation and safety checks
- **Runtime Overlay System**
  - Symlink-based runtime under `$ADP_RUNTIME_DIR`
  - Generated files (`AGENTS.md`, `CLAUDE.md`, `.codex/`, `.claude/`) isolated from project
  - Automatic cleanup on agent exit (configurable with `--keep-runtime`)
  - Runtime manifest (`.adp-runtime.yaml`) for inspection
- **Agent Adapters**
  - Codex adapter with profile support
  - Claude adapter with profile support
  - Process runner with environment isolation
  - `adp run <agent>` — launch agents with runtime overlay
  - `adp enter <workspace>` — interactive shell with runtime
- **Event & Session Tracking**
  - JSONL event log under `$ADP_HOME/logs/events.jsonl`
  - `adp events list` — query events by workspace/session/task/type
  - `adp sessions list/show` — inspect past agent runs
  - Session history derived from local events (no external database)
- **Task Management**
  - `adp tasks add/list/show/update` — create and manage tasks
  - `adp tasks claim/renew/release/done` — ownership and lease management
  - `adp tasks next/take` — preview and claim work atomically
  - `adp tasks stale` — find expired leases
  - `adp tasks block` — mark tasks as blocked
  - Task persistence under `$ADP_HOME/workspaces/<name>/planning/`
  - Task ID prefix matching for convenience
- **Phase Gates**
  - `adp phase add/list/show/status` — define phases
  - `adp phase start/accept/commit/push` — record lifecycle evidence
  - Phase-based workflow enforcement with blocking dependencies
- **Plan Import**
  - `adp plan preview/apply` — import tasks and phases from structured plans
  - `adp plan doctor` — validate plan integrity
- **Progress Reporting**
  - `adp progress` — view workspace progress summary
  - `adp progress report` — generate markdown/JSON reports (English/Chinese)
- **Session Restore**
  - `adp sessions restore-plan` — extract rerun guidance from past sessions
  - `adp sessions resume-plan` — generate work context for operator handoff
- **Runtime Maintenance**
  - `adp runtime prune` — clean up old runtimes with age-based filtering
  - `--dry-run` mode for safe preview
  - `--include-kept` flag to prune manually-kept runtimes
- **Shell Integration**
  - `adp shell-hook` — shell function for runtime-aware navigation
  - `adp completion` — bash/zsh completion scripts
  - `adp completion values` — dynamic completion for workspaces/tasks/sessions
  - `adp env --cd` — print cd command for workspace runtime
- **Utilities**
  - `adp init` — initialize `$ADP_HOME`
  - `adp version` — show version and build info
  - JSON output (`--format json`) for all list/show commands
  - `--verbose` flag for detailed diagnostics

### Infrastructure
- Go 1.24+ codebase with no external dependencies (standard library only)
- File-based storage (no database required)
- Cross-platform support (Linux, macOS, Windows via WSL)
- Session-local state isolation for safe concurrent use
- Git environment neutralization (GIT_DIR, GIT_WORK_TREE, etc.)

---

## Phase 2-3: Documentation & Bilingual Support

**Status**: ✅ Completed
**Focus**: Production-grade documentation and Chinese localization

### Added
- **Comprehensive Documentation**
  - `README.md` / `README.zh-CN.md` — project overview with quick start
  - `docs/install.md` — installation guide (source/binary/release)
  - `docs/operator-onboarding.md` — hands-on tutorial with checkpoints
  - `docs/task-management.md` — task workflow deep dive
  - `docs/session-restore.md` — restore/resume patterns
  - `docs/faq.md` — 22 questions covering common workflows
  - `docs/engineering-standards.md` — project conventions
  - `docs/license-policy.md` — license compliance guidelines
- **Bilingual Support**
  - Full English / 简体中文 documentation parity
  - Automated bilingual checks in CI (`scripts/check-docs-bilingual.sh`)
  - Command reference sync validation (same `adp` commands in both languages)
- **Contribution & Security**
  - `CONTRIBUTING.md` — contribution guidelines (dual-licensed)
  - `SECURITY.md` — security policy and reporting process
  - `COMMERCIAL.md` — commercial licensing information
- **License**
  - PolyForm Noncommercial 1.0.0 for source distribution
  - Commercial licensing available (see COMMERCIAL.md)

---

## Phase 4: Documentation Excellence

**Status**: ✅ Completed
**Focus**: Actionable diagnostics, enhanced help, and hands-on learning

### Added
- **Enhanced Diagnostics** (`adp workspace doctor`)
  - 50+ diagnostic codes with actionable suggestions
  - ✗/✓ symbols for visual clarity
  - Contextual fix commands in output
  - Structured JSON output for automation
- **Help System Improvements**
  - "See also" cross-references in all help pages
  - Related command suggestions (19 top-level, 5 subcommand mappings)
  - Smart deduplication of suggestions
- **Workshop Tutorial** (`docs/workshop.md`)
  - 4-module hands-on tutorial (workspace → task → agent → progress)
  - Self-contained with verification steps
  - Bilingual (English / 简体中文)
- **FAQ Expansion**
  - Q15: Complete handoff example (operator transition)
  - Q18: IDE integration examples (VS Code, TypeScript, external trackers)
  - Q19: Agent-driven Git workflow + safety considerations
  - Q22: External tool integration (Python monitoring script)
  - 132 code blocks with consistent command format
- **Example Enhancements**
  - `examples/basic-workspace` — production-ready workspace template
  - Profile examples for Codex and Claude
  - Memory and MCP configuration samples

### Quality Improvements
- Documentation score: 4.9/5 → 5.0/5
- Average task quality: 9.88/10
- Completion efficiency: 12x faster than estimated

---

## Phase 5: Usability Excellence

**Status**: ✅ Completed
**Focus**: Terminal output polish and interactive safety

### Added
- **Color Output** (`internal/output/color.go`)
  - ANSI color support with TTY auto-detection
  - `NO_COLOR` environment variable support (https://no-color.org/)
  - 7 color constants: Red (errors), Green (success), Yellow (warnings), Cyan (commands), Bold (emphasis)
  - Applied to: error messages, success confirmations, diagnostic output, command examples
  - PTY-verified three-state behavior (TTY on / NO_COLOR / piped)
- **Dangerous Operation Confirmation** (`internal/cli/confirm.go`)
  - Interactive confirmation for destructive operations
  - `--yes` / `-y` flag for non-interactive automation
  - Non-TTY safety (requires explicit `--yes` in scripts/CI)
  - Applied to: `workspace remove`, `runtime prune --include-kept`
  - Comprehensive unit tests (8/8 passing)
- **Success Message Guidance**
  - "Next steps" suggestions after key operations
  - Context-aware command recommendations
  - Applied to: `workspace add`, `task add`, `phase add`, `quickstart`, `run`
  - Color-highlighted commands for clarity
- **Command Aliases**
  - `ws` → `workspace`
  - `t` → `tasks`
  - `s` → `sessions`
  - `e` → `events`
  - `rt` → `runtime`
  - `p` → `phase`
- **Spelling Suggestions**
  - Levenshtein distance-based command matching
  - "Did you mean" output for typos (max edit distance 3)
  - Up to 3 suggestions per unknown command
  - Consistent across top-level and subcommands

### Quality Improvements
- Usability score: 4.6/5 → 5.0/5
- All Phase 5 acceptance tests passed (color PTY, confirmation, suggestions)
- Zero breaking changes (backward compatible)

---

## Phase 6: Documentation Refinement

**Status**: ✅ Completed
**Focus**: Troubleshooting, visual polish, and operator onboarding

### Added
- **Troubleshooting Guide** (`docs/troubleshooting.md` / `.zh-CN.md`)
  - 954 lines (66% expansion from initial draft)
  - 12 categorized sections: Installation, Workspace, Agent Execution, Runtime, Tasks, Phases, Sessions, Confirmation, Parameters, Environment, Permissions, Diagnostics
  - Error-message-indexed organization for fast lookup
  - 6 new error categories:
    - Agent execution issues (agent command not found, spelling suggestions)
    - Runtime safety (unsafe runtime parent, reserved filenames)
    - Task ownership (owner mismatch, no claimable task)
    - Phase management (phase not found, invalid phase transition)
    - Session issues (session not found, ambiguous session ID)
    - Interactive & confirmation (operation requires confirmation)
    - Parameter validation (--take conflicts, lease validation)
  - Command reference parity enforcement (English/Chinese)
- **README Visual Polish**
  - ✨ Key features spotlight (5-point summary with emoji)
  - 📖 "Start here" navigation for new users (install/onboarding/troubleshooting/FAQ)
  - Emoji section headers retained (🚀⚙️💡🏗️🔧📄)
  - Layered navigation: new-user links (top) vs developer docs (bottom)
- **Operator Onboarding Enhancement**
  - ✓ Checkpoint count: 4 → 6 (added "Move To Durable" and "Real Providers" sections)
  - ⏱️ Expected Time prompts: 4 → 6 (synchronized with checkpoints)
  - Diagnostic command examples in every checkpoint
  - Cross-links to troubleshooting guide
  - Bilingual parity (English/Chinese: 6/6 checkpoints, 6/6 time prompts)

---

## [1.0.0] - 2026-06-15

### Overview
ADP 1.0.0 marks the first production-ready release for local terminal-first AI agent workflows. All Phase 1-6 foundations are complete, tested, and documented.

### Recommended Upgrade Path
This is the first official release. New users should follow the [Installation Guide](docs/install.md) and [Operator Onboarding](docs/operator-onboarding.md).

### Known Limitations
- **Provider Support**: Codex and Claude adapters are local-process wrappers. External CLI authentication, model availability, quota, and network behavior are operator environment concerns (not ADP guarantees).
- **Real Agent Testing**: Default CI uses fake providers. Real model invocation requires opt-in smoke tests (`ADP_REAL_INVOKE_CODEX=1`).
- **Concurrency**: ADP state is session-local and file-based. High-concurrency multi-operator workflows may require external coordination.
- **Platform**: Primary testing on Linux. macOS and Windows (via WSL) are supported but less extensively tested.

### Future Roadmap
See [docs/project-roadmap-2026-06.md](docs/project-roadmap-2026-06.md) for post-1.0 priorities:
- Remote agent support (SSH, container runtimes)
- Web dashboard (optional observability layer)
- Real-time collaboration (shared task boards)
- Advanced phase gates (approval workflows, policy hooks)

---

## Migration Guide

### From Dev Builds to 1.0.0
No breaking changes. Existing `$ADP_HOME` state is forward-compatible.

**Optional cleanup**:
```bash
# Clean old runtimes
adp runtime prune --older-than 24h --include-kept --yes

# Verify workspace health
adp doctor --verbose
```

### Environment Variables
No changes. Continue using:
- `$ADP_HOME` — workspace/task/event storage (default: `~/.adp`)
- `$ADP_RUNTIME_DIR` — runtime overlay directory (default: `$TMPDIR/adp-runtime` or `/tmp/adp-runtime`)
- `$ADP_WORKSPACE` — default workspace for commands

---

## Deprecations

None. All Phase 1-5 APIs are stable for 1.0.0.

---

## Security

### Addressed in Phase 1-5
- **Runtime Isolation**: Project root safety checks prevent ADP from overwriting project files
- **Git Environment Neutralization**: Dangerous Git variables (GIT_DIR, GIT_WORK_TREE) are cleared during runtime
- **Unsafe Runtime Parent Detection**: `workspace doctor` catches filesystem root, project root, or overlapping runtime parents
- **Confirmation Protection**: Destructive operations (`workspace remove`, `runtime prune --include-kept`) require explicit confirmation

### Reporting
See [SECURITY.md](SECURITY.md) for the security policy and reporting process.

---

## Contributors

ADP is developed and maintained by [@karoc](https://github.com/karoc).

For contribution guidelines, see [CONTRIBUTING.md](CONTRIBUTING.md).

---

## License

ADP is source-available under the [PolyForm Noncommercial License 1.0.0](LICENSE).

Commercial use requires separate paid authorization. See [COMMERCIAL.md](COMMERCIAL.md) for details.
