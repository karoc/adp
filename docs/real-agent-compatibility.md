# Real Agent Compatibility Boundary

简体中文：[real-agent-compatibility.zh-CN.md](real-agent-compatibility.zh-CN.md)

This document defines what ADP guarantees when launching real external agent CLIs and what remains an operator-owned compatibility check. It intentionally records ADP's adapter contract rather than external CLI details that may change outside this repository.

## Compatibility Model

ADP owns the local runtime boundary:

- Resolve the workspace.
- Build an isolated runtime overlay under `ADP_RUNTIME_DIR`.
- Generate adapter-specific files inside the runtime overlay.
- Symlink real project files into the runtime overlay.
- Launch the external agent process with the runtime root as the working directory.
- Inject ADP environment variables while preserving the parent process environment, except for repository-directing Git variables that would point the runtime at an unintended worktree, index, object directory, common directory, or namespace.
- Forward arguments after `--` to the external command.
- Record local events and session history.
- Avoid writing ADP-generated files into the real project root.

The external agent CLI owns its own behavior after launch, including authentication, model selection, network access, tool permissions, prompt interpretation, and interactive behavior.

Task ownership and lease recovery remain ADP-owned. A provider-native task panel, plan mode, or successful external process exit can mirror or inform local work, but it must not be treated as task completion, phase acceptance, commit evidence, push evidence, Git execution, or authoritative recovery state.

The runtime overlay is not the authoritative Git worktree and does not expose repository Git metadata. Normal project Git files such as `.gitignore`, `.gitattributes`, and `.gitmodules` remain project files, while `.git` metadata is excluded. Symlinked runtime project subpaths may still map to real files, so edits under those links can affect the real project even though Git metadata is absent from the overlay. Before launching a real agent, ADP neutralizes repository-directing Git environment variables such as `GIT_DIR`, `GIT_WORK_TREE`, `GIT_INDEX_FILE`, `GIT_OBJECT_DIRECTORY`, `GIT_ALTERNATE_OBJECT_DIRECTORIES`, `GIT_COMMON_DIR`, and `GIT_NAMESPACE`. Normal shell environment and auth-related variables remain available to the external CLI.

Generated real-agent context makes the repository boundary explicit. Instruction files, adapter metadata, runtime manifests, and runtime environment variables expose `ADP_PROJECT_ROOT` plus `ADP_GIT_ROOT`, `git_root`, or `gitRoot` when ADP can discover the worktree root. `ADP_PROJECT_ROOT` is the configured project root and operator workspace boundary; `ADP_GIT_ROOT` is the broader Git worktree root. In monorepos or registered subdirectories, use `git -C "$ADP_PROJECT_ROOT" ...` for project-scoped commands and reserve `git -C "$ADP_GIT_ROOT" ...` for intentional whole-worktree inspection. ADP may run read-only Git topology and status inspection for workspace diagnostics, but it does not stage, commit, push, pull, clean, checkout, or otherwise wrap or auto-run Git mutations.

## Shared Runtime Contract

The currently documented real-agent adapter contracts are `codex` and `claude`. Future adapter design notes should use neutral placeholders until an adapter is implemented and validated, and must not describe provider-native resume semantics.

For `adp run <agent> --workspace <name> -- <agent-args>`, ADP builds a runtime root that includes:

- `.adp-runtime.yaml`, the ADP runtime manifest.
- Adapter-generated instruction and configuration files.
- Symlinks to files and directories from the real project root, unless a generated path takes precedence.

When the operator passes `--task <task-id>`, ADP binds that existing task to the runtime. When the operator passes `--take --owner <owner> [--lease 4h]`, ADP atomically claims the next eligible task before runtime creation and binds the taken task to the runtime. `--take` and `--task` are mutually exclusive.

For long-running real-agent work, the owner renews ADP task ownership with `adp tasks renew --workspace <workspace> <task-id> --owner <owner> --lease <duration>`. Interrupted sessions become visible through the read-only `adp tasks stale --workspace <workspace> [--format text|json]` view, and expired work is reclaimed only through ADP ownership commands such as `tasks take` or explicit `tasks claim`.

Real-agent handoff must stay lease-aware. The operator or worker should inspect the ADP task and local session evidence, renew before continuing long work, and use `tasks stale` before reclaiming interrupted work:

```bash
adp tasks show --workspace <workspace> <task-id> --format json
adp tasks renew --workspace <workspace> <task-id> --owner <owner> --lease <duration>
adp tasks stale --workspace <workspace> --format json
adp sessions restore-plan <session-id>
```

Provider-native task panels and plan panels may mirror the active ADP task or a proposed next-step list, but they are compatibility surfaces, not the recovery ledger. ADP must not scrape provider-private state, infer completion from provider exit, accept phases, record commit or push evidence, run Git, apply plans, or start the next phase from those surfaces. Runtime artifacts remain under `ADP_RUNTIME_DIR`; planning ledgers, progress, events, and sessions remain under `ADP_HOME`.

When an external CLI has a native plan mode, treat it as proposal-only. Structured changes should pass read-only `adp plan preview --workspace <workspace> --file - --format json`, then become durable only after explicit operator approval through `adp plan apply --workspace <workspace> --file - --format json`. If the provider entered plan mode after `adp run --take`, ADP already owns the taken task for that session, but the native plan still cannot complete tasks, accept phases, run Git, or become recovery evidence.

The launched process receives:

- Working directory: the runtime root.
- Parent environment: inherited from the shell that launched `adp`, except repository-directing Git variables are removed before launch.
- ADP runtime variables:
  - `ADP_HOME`.
  - `ADP_WORKSPACE`.
  - `ADP_PROJECT_ROOT`.
  - `ADP_GIT_ROOT`, when ADP can discover a Git worktree root from the configured project root.
  - `ADP_RUNTIME_ROOT`.
  - `ADP_SESSION_ID`.
  - `ADP_AGENT`.
  - `ADP_PROFILE`, when a profile is resolved.

Generated `AGENTS.md`/`CLAUDE.md` and adapter metadata expose the same project-root and Git-root distinction visible in the environment. Runtime manifests also record `git_root` when available and `git_metadata_skipped: true`. ADP sets `GIT_CEILING_DIRECTORIES` for the runtime root so Git discovery does not treat the overlay as an authoritative repository. If `ADP_PROJECT_ROOT` is a subdirectory inside a larger worktree, `ADP_GIT_ROOT` points at that larger worktree while project-root commands should still use `git -C "$ADP_PROJECT_ROOT" ...` for the operator's configured workspace boundary. Use `git -C "$ADP_GIT_ROOT" ...` only for deliberate whole-worktree inspection.

Workspace doctor diagnostics may inspect Git topology and status from the real project root. These diagnostics can report that no usable Git worktree is available, that the project root is nested below the Git root, that `.git` is a gitfile as in linked worktrees or submodules, that status inspection failed, or that the worktree has changed/untracked entries. They are inspection-only compatibility signals. ADP does not stage files, clean files, checkout branches, pull, commit, push, or convert phase evidence into Git execution.

The workspace agent command can override the default command through `workspace.yaml`:

```yaml
agents:
  codex:
    enabled: true
    command: codex
  claude:
    enabled: true
    command: claude
```

Use a wrapper script as `command` if a local operator needs extra external CLI setup. Keep that wrapper outside the real project root when possible.

## Codex Adapter Contract

The Codex adapter name is `codex`.

Default launch command:

```text
codex
```

Generated runtime files:

- `AGENTS.md`.
- `.codex/config.toml`.

The generated `AGENTS.md` contains ADP runtime instructions assembled from the workspace config, including workspace metadata, project-root and Git-root context, base prompt, shared memory, rules, MCP summary, and selected profile content when available.

The generated `.codex/config.toml` contains ADP metadata for the runtime session, including the Git root when discovered. It is an ADP-generated file and should not be treated as a complete statement of the external Codex CLI's current configuration schema.

ADP forwards `--` arguments directly:

```bash
adp run codex --workspace game-a -- <codex-args>
```

ADP does not define which Codex arguments are supported. Verify supported arguments against the installed Codex CLI on the operator machine before relying on them.

## Claude Adapter Contract

The Claude adapter name is `claude`.

Default launch command:

```text
claude
```

Generated runtime files:

- `CLAUDE.md`.
- `.claude/settings.json`.

The generated `CLAUDE.md` contains ADP runtime instructions assembled from the workspace config, including workspace metadata, project-root and Git-root context, base prompt, shared memory, rules, MCP summary, and selected profile content when available.

The generated `.claude/settings.json` contains ADP metadata for the runtime session, including the Git root when discovered. It is an ADP-generated file and should not be treated as a complete statement of the external Claude CLI's current settings schema.

If the real project already has provider-local configuration such as `.claude/settings.local.json`, ADP preserves non-conflicting files in the runtime overlay. ADP-generated metadata still wins at the exact `.claude/settings.json` path so a project file cannot override the runtime session metadata.

ADP forwards `--` arguments directly:

```bash
adp run claude --workspace game-a -- <claude-args>
```

ADP does not define which Claude arguments are supported. Verify supported arguments against the installed Claude CLI on the operator machine before relying on them.

## Opt-In Real CLI Smoke

The default repository smoke uses fake agents. Real external CLI checks are opt-in:

```bash
ADP_SMOKE_REAL_CODEX=1 scripts/runtime-smoke.sh --real-codex
ADP_SMOKE_REAL_CLAUDE=1 scripts/runtime-smoke.sh --real-claude
```

Real CLI flags are additive to the default fake smoke. They do not replace `scripts/check-all.sh`, and they should not be treated as CI requirements unless a release explicitly claims real-agent evidence.

Override command paths when needed:

```bash
ADP_SMOKE_REAL_CODEX=1 ADP_SMOKE_CODEX_BIN=/path/to/codex scripts/runtime-smoke.sh --real-codex
ADP_SMOKE_REAL_CLAUDE=1 ADP_SMOKE_CLAUDE_BIN=/path/to/claude scripts/runtime-smoke.sh --real-claude
```

These checks only confirm that the external command exists and that a lightweight `--version` or `--help` invocation completes. They are command availability checks, not model invocation evidence. Default doctor diagnostics remain static and local: they can flag command shape, wrapper path, profile, and reserved-path risks, but they do not run provider CLIs. Neither path proves that a real interactive session can authenticate, select a model, reach a provider, consume quota, or use external tools correctly.

Manual real-agent acceptance is operator-owned. It may require local credentials, network access, provider account quota, model access, and external tool permissions that ADP cannot create or validate deterministically.

## Opt-In Real-Agent Invocation Smoke

`scripts/real-agent-invocation-smoke.sh` is the dedicated path for explicit non-interactive Codex and Claude invocation evidence through ADP. It is separate from `scripts/runtime-smoke.sh --real-codex` and `scripts/runtime-smoke.sh --real-claude`: the runtime smoke's real flags check command availability, while the invocation smoke is intended to prove that ADP can hand off a constrained non-interactive request to the installed external CLIs in the current operator environment.

The invocation smoke is not part of `scripts/check-all.sh`, and it must not become a default CI or release gate. Run it only when a release, audit, or operator note explicitly asks for real-agent invocation evidence and the operator accepts that the script may contact external providers, use account credentials already present on the machine, and consume provider quota.

Run the script only with the explicit opt-in gates documented by the script or release procedure:

```bash
ADP_REAL_INVOKE_CODEX=1 scripts/real-agent-invocation-smoke.sh --codex
ADP_REAL_INVOKE_CLAUDE=1 scripts/real-agent-invocation-smoke.sh --claude
ADP_REAL_INVOKE_CODEX=1 ADP_REAL_INVOKE_CLAUDE=1 scripts/real-agent-invocation-smoke.sh --all
```

Running the script without `--codex`, `--claude`, or `--all` is provider-free. It prints the opt-in guidance and exits successfully without building ADP, resolving external commands, creating runtime overlays, contacting providers, or consuming quota. Use that default run to verify the guidance path in local validation; it is not real-agent evidence.

The smoke should build or select the ADP binary under test, create temporary `ADP_HOME`, `ADP_RUNTIME_DIR`, and project root paths, register a temporary workspace, invoke Codex and Claude through `adp run ...`, inspect local events and sessions, and then remove temporary state. It should not write planning files, reports, generated instruction files, provider output, or runtime metadata into the real repository project root.

Evidence from this script must stay non-sensitive. Record only operational facts such as the ADP version or commit, external command paths, external command versions or first help lines, adapter names, workspace name, session IDs, exit statuses, sanitized timestamps, and whether each invocation path passed or failed. Do not record secrets, tokens, API keys, private prompts, account identifiers, full model responses, proprietary code excerpts, or provider-specific conversation IDs.

A passing invocation smoke is environment-specific evidence. It proves that ADP handed a constrained non-interactive request to the selected external CLI in that operator environment and that local ADP evidence was recorded. It does not guarantee that other operators have credentials, model access, available quota, stable network access, matching external CLI versions, equivalent tool permissions, or acceptable interactive session quality.

When a real invocation fails, first identify whether ADP reached the external CLI. ADP-side failures usually involve workspace resolution, runtime parent safety, command path configuration, runtime overlay creation, task binding, local event/session writes, or project-root cleanliness. External-environment failures usually involve authentication, account state, model names, quota, provider availability, network access, permissions, or external CLI argument changes. Triage those operator-environment causes before changing shared ADP adapter assumptions.

## Manual Acceptance Steps

Start by selecting the exact ADP binary being accepted. For a source checkout, build a temporary binary from the current commit and record its version output:

```bash
tmp="$(mktemp -d)"
ADP_BIN="$tmp/adp"
go build -o "$ADP_BIN" ./cmd/adp
"$ADP_BIN" version
```

For a packaged release candidate, use the packaged binary instead and record its version output:

```bash
tmp="$(mktemp -d)"
ADP_BIN=/path/to/adp
"$ADP_BIN" version
```

Run the deterministic repository gate first from the repository root:

```bash
scripts/check-all.sh
```

`scripts/runtime-smoke.sh` creates its own temporary ADP home and runtime directory. Treat its evidence as repository gate evidence, not as the manual real-launch sandbox.

Then run the opt-in command availability checks only when those checks are intentionally part of the release evidence:

```bash
ADP_SMOKE_REAL_CODEX=1 scripts/runtime-smoke.sh --real-codex
ADP_SMOKE_REAL_CLAUDE=1 scripts/runtime-smoke.sh --real-claude
```

When release evidence intentionally includes actual non-interactive model invocation through ADP, run the dedicated invocation smoke after the command availability checks and keep its output redacted:

```bash
ADP_REAL_INVOKE_CODEX=1 ADP_REAL_INVOKE_CLAUDE=1 scripts/real-agent-invocation-smoke.sh --all
```

For manual real-launch acceptance, create a separate temporary ADP home and runtime directory with the same `ADP_BIN`:

```bash
export ADP_HOME="$tmp/adp-home"
export ADP_RUNTIME_DIR="$tmp/runtime"
mkdir -p "$tmp/project"
printf 'real-agent-smoke\n' > "$tmp/project/README.md"

"$ADP_BIN" init
"$ADP_BIN" workspace add real-agent-smoke "$tmp/project"
"$ADP_BIN" workspace doctor real-agent-smoke
```

For real launch acceptance, choose operator-safe arguments supported by the installed external CLI:

```bash
"$ADP_BIN" run codex --workspace real-agent-smoke -- <operator-safe-codex-args>
"$ADP_BIN" run claude --workspace real-agent-smoke -- <operator-safe-claude-args>
```

Common safe candidates are `--version` or `--help` when the installed external CLI supports them, but ADP does not define external CLI arguments.

These commands may contact external providers. Do not run them as default CI or release gates, and do not record secrets, tokens, private prompts, account identifiers, or sensitive model output as evidence.

After each run, inspect local ADP evidence:

```bash
"$ADP_BIN" events list --workspace real-agent-smoke
"$ADP_BIN" sessions list --workspace real-agent-smoke
"$ADP_BIN" sessions show <session-id>
"$ADP_BIN" sessions restore-plan <session-id>
```

`sessions restore-plan` remains read-only for real agents too. It can suggest a similar new `adp run ...` command from local invocation metadata, but it does not resume the provider-native conversation or execute the suggested command.

Task recovery for real agents remains separate from provider-private session state. Use `tasks renew` before leases expire, `tasks stale` to inspect expired in-progress claims, and `tasks take` or explicit `tasks claim` to reclaim work after expiration. None of these paths complete tasks, accept phases, commit, push, run Git, or scrape native task boxes and plan panels.

Confirm the real project root remains clean:

```bash
test ! -e "$tmp/project/AGENTS.md"
test ! -e "$tmp/project/CLAUDE.md"
test ! -e "$tmp/project/.codex"
test ! -e "$tmp/project/.claude"
test ! -e "$tmp/project/planning"
test ! -e "$tmp/project/tasks.yaml"
test ! -e "$tmp/project/phases.yaml"
test ! -e "$tmp/project/progress.jsonl"
```

Record the ADP commit or packaged version, `"$ADP_BIN" version` output, external command paths, external command versions or help output, workspace name, and any operator-specific flags used.

Also record whether the evidence only proves command availability, whether `scripts/real-agent-invocation-smoke.sh` completed non-interactive model invocation, or whether a manual interactive `adp run ...` acceptance was completed.

## Boundary And Failure Handling

Treat these as outside ADP's deterministic compatibility guarantee:

- Provider account state.
- Network access.
- Model availability.
- External CLI release behavior.
- External CLI prompt/configuration schema.
- Interactive session semantics.
- External tool permissions granted by the external CLI.

When a real CLI behavior changes, verify it on the target machine before changing ADP adapter assumptions. If the change is local to one operator environment, prefer an explicit wrapper command in `workspace.yaml` over changing the shared adapter contract.
