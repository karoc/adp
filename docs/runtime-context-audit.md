# Runtime Context Audit

Simplified Chinese: [runtime-context-audit.zh-CN.md](runtime-context-audit.zh-CN.md)

This document audits the context ADP makes visible when it launches an agent. It is scoped to existing terminal-first, local-first runtime behavior: ADP prepares a local runtime overlay, launches a local external command, records local evidence, and keeps authoritative workspace and planning state outside the real project root.

It does not introduce a Web UI, dashboard, SaaS tracker, cloud sync, hosted orchestration, automatic Git workflow, provider-native conversation resume, or project-root report export path.

## Launch Boundary

The canonical launch shape is `adp run <agent> --workspace <name> --profile <profile> --task <task-id> -- <agent-args>`. The workspace, profile, and task flags are optional when normal workspace resolution or adapter defaults are enough, but this shape shows every context input that can affect what the agent sees.

At launch, ADP:

- Resolves the workspace from `--workspace`, `ADP_WORKSPACE`, or the current directory when it is inside a registered project root.
- Reads the workspace config from `$ADP_HOME/workspaces/<workspace>/workspace.yaml`.
- Resolves the adapter and selected profile from the explicit `--profile`, the workspace agent profile, or the `default` fallback.
- Loads task metadata from `$ADP_HOME/workspaces/<workspace>/planning` when `--task <task-id>` is provided.
- Builds a temporary runtime root under `$ADP_RUNTIME_DIR`.
- Writes ADP-generated files into the runtime root.
- Symlinks real project files into the runtime root, with generated paths taking precedence.
- Starts the agent command with the runtime root as the process working directory and forwards arguments after `--`.

`adp env <workspace> --cd` and `adp enter <workspace>` use the same runtime overlay boundary for shell workflows, but they do not render adapter-specific instruction files unless an agent adapter is launched through `adp run`.

## Runtime View

A Codex launch sees a runtime root shaped like:

```txt
$ADP_RUNTIME_DIR/<workspace>-<session>/
├── AGENTS.md
├── .adp-runtime.yaml
├── .codex/
│   └── config.toml
├── go.mod -> <project-root>/go.mod
└── internal -> <project-root>/internal
```

A Claude launch sees the same runtime model with `CLAUDE.md` and `.claude/settings.json` instead of `AGENTS.md` and `.codex/config.toml`.

If the real project already has provider-local configuration directories such as `.codex/` or `.claude/`, ADP merges non-conflicting children into the runtime overlay. For example, project-owned `.claude/settings.local.json` remains visible to a Claude runtime, while ADP-generated `.claude/settings.json` wins over any project file at that exact path. Conflicts are runtime evidence; the real project directory is not modified.

The generated `.adp-runtime.yaml` manifest records ADP ownership and cleanup metadata: manifest version, session ID, workspace name, optional task ID and title, project root, runtime root, creation time, keep flag, and `generated_by: adp`. Runtime pruning uses this manifest as compatibility evidence before deleting an ADP-owned runtime directory.

## Instruction Files

Generated instruction files are the primary human-readable context surface:

- Codex receives `AGENTS.md`.
- Claude receives `CLAUDE.md`.

Both files are generated from the same ADP renderer. The visible sections are:

- Workspace metadata: workspace name, real project root, adapter name, and effective profile.
- Current task: task ID, title, status, priority, phase, description, and blocked reason when a task is bound.
- ADP Planning Contract: ADP remains the authoritative local planning ledger, and durable task state changes must use ADP task and phase commands.
- Tool Taskbox Bridge: provider-native todo or task panels may mirror the active ADP task for local visibility, but they are not the source of truth.
- Tool Plan Mode Bridge: provider-native plan mode may organize proposals, but read-only ADP plan preview and explicitly approved plan apply are the durable planning path.
- Base prompt: the configured `prompts.base` file, or a local fallback message when no readable file is configured.
- Shared memory: the configured `memory.shared` file when memory is enabled, or a local disabled/missing fallback.
- Rules: sorted workspace rules from `workspace.yaml`.
- MCP: enabled server names and the configured MCP config file content, or a local disabled/missing fallback.
- Profile: effective profile name, agent enabled state, command summary, options, and the first matching profile file.

Profile file lookup uses the effective profile first when it is not `default`, then falls back to the adapter profile file such as `profiles/codex.yaml` or `profiles/claude.yaml`. Supported profile file suffixes are `.md`, `.yaml`, `.yml`, and `.json`.

These instruction files are runtime artifacts. They are not copied into the real project root by normal ADP workflows.

## Adapter Config Files

Adapter config files are generated beside the instruction file to expose ADP runtime metadata to the launched tool:

- Codex receives `.codex/config.toml`.
- Claude receives `.claude/settings.json`.

The Codex metadata contains an `[adp]` table with adapter name, workspace name, project root, effective profile, memory enabled state, MCP enabled state, and task fields when a task is bound.

The Claude metadata contains an `adp` JSON object with adapter name, workspace name, project root, effective profile, memory enabled state, MCP enabled state, and a task object when a task is bound.

These files are ADP metadata, not a full or current declaration of either external provider CLI's native configuration schema. External CLI authentication, model selection, network behavior, tool permissions, and prompt interpretation remain owned by the external command and the local operator.

Project-owned provider-local files that do not collide with these exact generated paths are linked into the runtime overlay. This preserves existing local provider configuration without letting project files override ADP's runtime metadata.

## Profile, Prompt, Memory, And MCP

The selected profile affects four launch surfaces:

- The effective profile line in the generated instruction file.
- The profile file content included in the generated instruction file.
- The generated adapter metadata.
- The `ADP_PROFILE` runtime environment variable.

The base prompt, shared memory, rules, and MCP references are read from the local ADP workspace directory under `$ADP_HOME`. ADP treats those files as local context inputs. Missing, empty, unreadable, or path-escaping files produce local fallback text in generated instructions instead of causing ADP to copy state into the project root.

MCP content in generated instructions is a reference and configuration summary for the launched agent. The runtime context audit must not describe MCP support as hosted orchestration or cloud sync.

## Task Metadata

When `--task <task-id>` is present, ADP loads the task from the workspace planning ledger before launching the agent. A missing task fails before the agent command starts.

Task metadata is visible in:

- The `Current Task` section of the generated instruction file.
- The generated adapter config file.
- Runtime environment variables.
- The `.adp-runtime.yaml` manifest through the task ID and task title.
- Local `run_started` and `run_finished` events.
- Session history derived from local events.

Binding a task to a runtime does not automatically claim, complete, block, accept, commit, or push that task or phase. Those remain explicit terminal commands, and Git remains operator-run outside ADP.

## Planning Contract And Taskbox Bridge

Generated instructions carry the ADP planning contract into each launched tool. The agent should treat ADP as the durable source of truth for workspace planning and progress, then use ADP commands for persistent state changes:

```bash
adp tasks next --workspace <workspace> --format json
adp tasks take --workspace <workspace> --owner <owner> --lease <duration> --format json
adp tasks add --workspace <workspace> --phase <phase-id> --priority <priority> "<title>"
adp tasks claim --workspace <workspace> <task-id> --owner <owner> --lease <duration>
adp tasks update --workspace <workspace> <task-id> --status in_progress
adp tasks block --workspace <workspace> <task-id> --reason "<reason>"
adp tasks release --workspace <workspace> <task-id> --owner <owner>
adp tasks done --workspace <workspace> <task-id>
adp phase status --workspace <workspace> --format json
adp progress report --workspace <workspace> --format json
```

If the external tool has a native task or todo panel, the agent should mirror the active ADP task there for visibility. That mirror can contain the task ID, title, status, phase, owner or lease, and local subtasks, but it is a working view only. Durable state still belongs in `$ADP_HOME/workspaces/<workspace>/planning/`.

This bridge is currently instruction-level unless a provider exposes a stable local API. ADP must not scrape provider-private todo state, treat a provider task panel as authoritative, infer completion from an agent exit code, auto-accept phases, or run Git automatically.

## Tool Plan Mode Bridge

When the launched provider tool supports plan mode, that mode is a proposal surface only. The agent may use it to shape and show candidate work, but plan-mode items are scratch state until they are validated and applied through ADP.

Plan-mode agents should not edit implementation files, mark tasks done, accept phases, commit, push, or perform execution side effects unless the user explicitly approves moving from planning into execution. A proposed structured plan should be checked read-only with:

```bash
adp plan preview --workspace <workspace> --file - --format json
```

After explicit user or operator approval, the same proposal can be made durable with:

```bash
adp plan apply --workspace <workspace> --file - --format json
```

Task ownership, status changes, blocker records, and phase evidence continue to use the task and phase commands in the ADP planning contract. Provider-native plan panels must not be treated as authoritative planning storage or recovery evidence.

## Runtime Environment Variables

The launched agent process inherits the parent shell environment and receives ADP runtime variables:

- `ADP_HOME`: local ADP home.
- `ADP_WORKSPACE`: selected workspace.
- `ADP_PROJECT_ROOT`: real project root.
- `ADP_RUNTIME_ROOT`: temporary runtime root and process working directory.
- `ADP_SESSION_ID`: ADP runtime session ID.
- `ADP_AGENT`: adapter name.
- `ADP_PROFILE`: effective profile when one is resolved.
- `ADP_TASK_ID`: bound task ID.
- `ADP_TASK_TITLE`: bound task title.
- `ADP_TASK_STATUS`: bound task status.
- `ADP_TASK_PRIORITY`: bound task priority.
- `ADP_TASK_PHASE`: bound task phase.

Task variables are present only for task-bound runs. The event logger sanitizes event fields named like full environments, so local session evidence should not store complete shell environments.

## Event And Session Evidence

ADP appends local JSONL runtime evidence under `$ADP_HOME/logs/events.jsonl`:

- `run_started` records workspace, agent, profile, runtime path, project root, session ID, task ID when present, and a non-sensitive invocation snapshot.
- `run_finished` records workspace, agent, profile, runtime path, project root, session ID, task ID when present, exit code, and duration.

The invocation snapshot can include schema version, forwarded agent arguments, keep-runtime choice, workspace resolution source, profile source, original current directory, and a task snapshot. It must not include credentials, tokens, full environment variables, generated instructions, provider conversation state, or project file contents.

Session commands read those local events:

- `adp events list`
- `adp sessions list`
- `adp sessions show <session-id>`
- `adp sessions restore-plan <session-id>`

`adp sessions restore-plan <session-id>` is read-only. It prints a suggested new local launch command when enough non-sensitive data exists; it does not launch an agent, create a runtime, append events, mutate task or phase state, write to the project root, or resume a provider-native conversation.

## Project-Root Cleanliness

The real project root must remain clean. Runtime context belongs under `$ADP_RUNTIME_DIR`; workspace config, prompts, shared memory, MCP config, profiles, planning ledgers, events, and sessions belong under `$ADP_HOME`.

Normal ADP runtime and reporting paths must not create these files or directories in the real project root:

- `AGENTS.md`
- `CLAUDE.md`
- `.codex`
- `.claude`
- `.adp-runtime.yaml`
- `planning`
- `tasks.yaml`
- `phases.yaml`
- `progress.jsonl`
- Markdown or JSON report exports

If a real project already contains a path that ADP reserves for generated runtime context, ADP-generated content takes precedence inside the runtime overlay and workspace doctor commands report local diagnostics. ADP should not repair that by editing the real project root unless an operator explicitly asks for such project-file edits.

## Audit Evidence

The current default evidence for runtime context is covered by local smokes:

- `scripts/runtime-smoke.sh --fake` verifies fake Codex and Claude runtime launches, generated files, task context, environment variables, local events, sessions, restore planning, pruning, and project-root cleanliness.
- `scripts/runtime-audit-smoke.sh` broadens coverage across CLI discovery, runtime entry points, task/phase/plan/progress flows, session views, JSON outputs, and local-first boundaries.
- `scripts/runtime-context-smoke.sh` focuses on launch-time context: generated instruction files, adapter metadata files, selected profiles, base prompt, shared memory, MCP references, task metadata, runtime environment variables, local event/session evidence, workspace diagnostics, and project-root cleanliness.

`scripts/runtime-context-smoke.sh` stays deterministic and local: temporary `ADP_HOME`, temporary `ADP_RUNTIME_DIR`, temporary project root, fake agents only, no network, no real provider CLI requirement, no Git execution, no hosted service dependency, and no project-root report or planning exports.
