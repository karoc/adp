# Task Management

Simplified Chinese: [task-management.zh-CN.md](task-management.zh-CN.md)

ADP's Task and Progress Manager is the local source of truth for workspace-scoped planning and execution state. It keeps project planning outside the real project root, then lets terminal users and agents inspect the same task list before choosing the next work item.

This layer is intentionally not a Web dashboard, SaaS tracker, or issue-hosting replacement. It is a terminal-first, local-first state manager for agent work.

## Current Scope

The first task-management slice provides:

- `adp tasks add`
- `adp tasks list`
- `adp tasks show`
- `adp tasks update`
- `adp tasks done`
- `adp tasks block`
- `adp progress`
- `adp run --task <task-id>` runtime binding.
- Workspace-local planning files under `$ADP_HOME/workspaces/<workspace>/planning/`.
- JSONL progress events for task creation and status changes.
- Runtime event and session evidence linked by task ID.

## Storage

Task state lives under the ADP workspace directory:

```txt
$ADP_HOME/workspaces/<workspace>/
└── planning/
    ├── tasks.yaml
    └── progress.jsonl
```

ADP does not write these files into the real project root by default. Exporting task state into repository documentation should be an explicit user action in a future command, not automatic project-root mutation.

## Task Status

Task status values are:

- `planned`: known work that is not ready to execute.
- `ready`: work that can be picked up now.
- `in_progress`: work currently being executed.
- `blocked`: work that cannot continue without a recorded reason.
- `review`: implementation is ready for review.
- `validated`: acceptance has passed but the phase is not yet closed.
- `done`: work has been accepted and committed or otherwise delivered.
- `canceled`: work is no longer planned.

## CLI

Create a task:

```bash
adp tasks add --workspace adp --priority high --phase phase-1.5 "Add task manager"
```

List and inspect tasks:

```bash
adp tasks list --workspace adp
adp tasks show --workspace adp <task-id>
```

Move a task through execution:

```bash
adp tasks update --workspace adp <task-id> --status in_progress
adp tasks block --workspace adp <task-id> --reason "waiting for real CLI evidence"
adp tasks done --workspace adp <task-id>
```

Summarize progress:

```bash
adp progress --workspace adp
```

When `--workspace` is omitted, ADP uses the same workspace resolution model as other workspace-aware commands: `ADP_WORKSPACE` first, then the current directory if it is inside a registered project root.

## Runtime Binding

Attach a task to an agent runtime session:

```bash
adp run codex --workspace adp --task <task-id> -- --version
```

The task ID is resolved inside the selected workspace. If the task does not exist in that workspace, ADP fails before building a runtime or launching the agent process.

When a task is bound, ADP injects task context into:

- Runtime environment variables such as `ADP_TASK_ID`, `ADP_TASK_TITLE`, `ADP_TASK_STATUS`, `ADP_TASK_PRIORITY`, and `ADP_TASK_PHASE`.
- The runtime manifest `.adp-runtime.yaml`.
- Generated adapter instructions such as `AGENTS.md` and `CLAUDE.md`.
- Generated adapter metadata such as `.codex/config.toml` and `.claude/settings.json`.
- `run_started` and `run_finished` events.
- Session summaries and session detail output.

Task-bound history can be inspected with:

```bash
adp events list --workspace adp --task <task-id>
adp sessions list --workspace adp --task <task-id>
adp sessions show <session-id>
```

`adp run --task` does not automatically move task status. Status changes remain explicit through `adp tasks update`, `adp tasks done`, and `adp tasks block`.

## Phase Discipline

Task management is intended to support phase-by-phase delivery:

- Prioritize planned work before starting execution.
- Complete one phase slice at a time.
- Run the relevant runtime smoke and full repository gate for that phase.
- Commit and push the accepted phase before starting the next phase.
- Do not mix the next phase into the same commit just because the working tree is open.

This keeps task history, validation evidence, and Git history aligned.

## Boundary

The current task manager does not yet:

- Automatically split user intent into tasks.
- Claim tasks for external agents.
- Export progress reports into project documentation.
- Sync with GitHub Issues, Linear, Jira, Notion, or any hosted service.

Those are future slices. The first priority is a reliable local task state that all terminal agents can read.
