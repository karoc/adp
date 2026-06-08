# Real Agent Compatibility Boundary

Simplified Chinese: [real-agent-compatibility.zh-CN.md](real-agent-compatibility.zh-CN.md)

This document defines what ADP guarantees when launching real external agent CLIs and what remains an operator-owned compatibility check. It intentionally records ADP's adapter contract rather than external CLI details that may change outside this repository.

## Compatibility Model

ADP owns the local runtime boundary:

- Resolve the workspace.
- Build an isolated runtime overlay under `ADP_RUNTIME_DIR`.
- Generate adapter-specific files inside the runtime overlay.
- Symlink real project files into the runtime overlay.
- Launch the external agent process with the runtime root as the working directory.
- Inject ADP environment variables while preserving the parent process environment.
- Forward arguments after `--` to the external command.
- Record local events and session history.
- Avoid writing ADP-generated files into the real project root.

The external agent CLI owns its own behavior after launch, including authentication, model selection, network access, tool permissions, prompt interpretation, and interactive behavior.

## Shared Runtime Contract

For `adp run <agent> --workspace <name> -- <agent-args>`, ADP builds a runtime root that includes:

- `.adp-runtime.yaml`, the ADP runtime manifest.
- Adapter-generated instruction and configuration files.
- Symlinks to files and directories from the real project root, unless a generated path takes precedence.

The launched process receives:

- Working directory: the runtime root.
- Parent environment: inherited from the shell that launched `adp`.
- ADP runtime variables:
  - `ADP_HOME`.
  - `ADP_WORKSPACE`.
  - `ADP_PROJECT_ROOT`.
  - `ADP_RUNTIME_ROOT`.
  - `ADP_SESSION_ID`.
  - `ADP_AGENT`.
  - `ADP_PROFILE`, when a profile is resolved.

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

The generated `AGENTS.md` contains ADP runtime instructions assembled from the workspace config, including workspace metadata, base prompt, shared memory, rules, MCP summary, and selected profile content when available.

The generated `.codex/config.toml` contains ADP metadata for the runtime session. It is an ADP-generated file and should not be treated as a complete statement of the external Codex CLI's current configuration schema.

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

The generated `CLAUDE.md` contains ADP runtime instructions assembled from the workspace config, including workspace metadata, base prompt, shared memory, rules, MCP summary, and selected profile content when available.

The generated `.claude/settings.json` contains ADP metadata for the runtime session. It is an ADP-generated file and should not be treated as a complete statement of the external Claude CLI's current settings schema.

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

Override command paths when needed:

```bash
ADP_SMOKE_REAL_CODEX=1 ADP_SMOKE_CODEX_BIN=/path/to/codex scripts/runtime-smoke.sh --real-codex
ADP_SMOKE_REAL_CLAUDE=1 ADP_SMOKE_CLAUDE_BIN=/path/to/claude scripts/runtime-smoke.sh --real-claude
```

These checks only confirm that the external command exists and that a lightweight `--version` or `--help` invocation completes. Default doctor diagnostics remain static and local: they can flag command shape, wrapper path, profile, and reserved-path risks, but they do not run provider CLIs. Neither path proves that a real interactive session can authenticate, select a model, reach a provider, or use external tools correctly.

## Manual Acceptance Steps

Use a temporary ADP home and runtime directory when validating real agents:

```bash
tmp="$(mktemp -d)"
export ADP_HOME="$tmp/adp-home"
export ADP_RUNTIME_DIR="$tmp/runtime"
mkdir -p "$tmp/project"
printf 'real-agent-smoke\n' > "$tmp/project/README.md"

adp init
adp workspace add real-agent-smoke "$tmp/project"
adp workspace doctor real-agent-smoke
```

Run the deterministic fake smoke first from the repository root:

```bash
scripts/runtime-smoke.sh --fake
```

Then run the opt-in command availability checks:

```bash
ADP_SMOKE_REAL_CODEX=1 scripts/runtime-smoke.sh --real-codex
ADP_SMOKE_REAL_CLAUDE=1 scripts/runtime-smoke.sh --real-claude
```

For real launch acceptance, choose operator-safe arguments supported by the installed external CLI:

```bash
adp run codex --workspace real-agent-smoke -- <operator-safe-codex-args>
adp run claude --workspace real-agent-smoke -- <operator-safe-claude-args>
```

After each run, inspect local ADP evidence:

```bash
adp events list --workspace real-agent-smoke
adp sessions list --workspace real-agent-smoke
adp sessions show <session-id>
adp sessions restore-plan <session-id>
```

`sessions restore-plan` remains read-only for real agents too. It can suggest a similar new `adp run ...` command from local invocation metadata, but it does not resume the provider-native conversation or execute the suggested command.

Confirm the real project root remains clean:

```bash
test ! -e "$tmp/project/AGENTS.md"
test ! -e "$tmp/project/CLAUDE.md"
test ! -e "$tmp/project/.codex"
test ! -e "$tmp/project/.claude"
```

Record the ADP commit, external command paths, external command versions or help output, workspace name, and any operator-specific flags used.

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
