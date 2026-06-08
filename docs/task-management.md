# Task Management

Simplified Chinese: [task-management.zh-CN.md](task-management.zh-CN.md)

ADP's Task and Progress Manager is the local source of truth for workspace-scoped planning and execution state. It keeps project planning outside the real project root, then lets terminal users and agents inspect the same task list before choosing the next work item.

This layer is intentionally not a Web dashboard, SaaS tracker, or issue-hosting replacement. It is a terminal-first, local-first state manager for agent work.

## Implemented Scope

The first task-management slice provides:

- `adp tasks add`
- `adp tasks list`
- `adp tasks show`
- `adp tasks update`
- `adp tasks claim`
- `adp tasks release`
- `adp tasks done`
- `adp tasks block`
- `adp phase add`
- `adp phase list`
- `adp phase show`
- `adp phase start`
- `adp phase accept`
- `adp phase commit`
- `adp phase push`
- `adp progress`
- `adp run --task <task-id>` runtime binding.
- Workspace-local planning files under `$ADP_HOME/workspaces/<workspace>/planning/`.
- JSONL progress events for task creation and status changes.
- Runtime event and session evidence linked by task ID.

The active P3 work builds on this foundation. Smoke scripts should assert only the Phase Gate MVP commands that exist in the current tree, and should not add placeholder checks for planned commands.

## Storage

Task state lives under the ADP workspace directory:

```txt
$ADP_HOME/workspaces/<workspace>/
└── planning/
    ├── tasks.yaml
    └── progress.jsonl
```

ADP does not write these files into the real project root by default. Exporting task state into repository documentation should be an explicit user action in a future command, not automatic project-root mutation.

The Phase Gate MVP keeps using this local planning directory and extends it with structured phase and gate records. The storage remains local-first and terminal-readable:

- `tasks.yaml`: task list, status, priority, phase ID, and owner when a task is claimed.
- `phases.yaml`: phase records, phase status, acceptance records, commit records, push records, and gate summary.
- `progress.jsonl`: append-only audit events for task, phase, acceptance, commit, and push changes.

Repository documentation can summarize the plan, but the authoritative execution state stays under `$ADP_HOME`. ADP must not create `planning/`, `tasks.yaml`, `phases.yaml`, or `progress.jsonl` in the real project root during normal operation.

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
adp tasks claim --workspace adp <task-id> --owner codex-main
adp tasks update --workspace adp <task-id> --status in_progress
adp tasks block --workspace adp <task-id> --reason "waiting for real CLI evidence"
adp tasks release --workspace adp <task-id>
adp tasks done --workspace adp <task-id>
```

Record a phase and its gate evidence:

```bash
adp phase add --workspace adp --goal "phase gate MVP" p3 "Project planning and execution progress"
adp phase start --workspace adp p3
adp phase accept --workspace adp p3 --command "scripts/check-all.sh" --result passed --notes "local gate passed"
adp phase commit --workspace adp p3 --hash <commit-hash> --message "Implement phase gate MVP"
adp phase push --workspace adp p3 --remote origin --branch main --result pushed
adp phase show --workspace adp p3
```

Summarize progress:

```bash
adp progress --workspace adp
```

When `--workspace` is omitted, ADP uses the same workspace resolution model as other workspace-aware commands: `ADP_WORKSPACE` first, then the current directory if it is inside a registered project root.

## Phase Gate MVP

P3's first slice is the Phase Gate MVP. It turns the current task list into a phase-aware execution ledger that multiple terminal agents can share without adding a Web dashboard, SaaS tracker, cloud sync, hosted orchestration, or remote issue service.

The MVP makes these records explicit:

- Phase records: ID, title, status, objective or goal, acceptance command list, commit evidence, push evidence, and latest gate outcome.
- Task claim records: task ID, owner, claimed timestamp, released timestamp when applicable, and current ownership state.
- Acceptance records: phase ID, command list, result, timestamp, and short evidence text.
- Gate records: phase ID, gate status, required checks, and operator or agent notes when they are provided.
- Commit records: phase ID, commit hash, branch, summary, timestamp, and whether the commit contains only the accepted phase.
- Push records: phase ID, remote, branch, timestamp, and push result. Commit hash evidence is stored separately on the phase record before push evidence is recorded.

The first implementation can remain intentionally small. It should optimize for reliable local evidence over broad project-management features.

## Task Claim And Ownership

Task ownership is a coordination hint for local multi-agent execution. It is not an authorization system.

Phase Gate MVP claim rules:

- A ready task may be claimed by one owner at a time.
- The owner may be a human name, agent name, or stable local agent identifier.
- Claiming a task records ownership before implementation starts.
- Releasing a task clears ownership when the worker is done, blocked, or reassigned.
- A claimed task still has to respect its assigned file boundaries and phase scope.
- Sub-agents do not commit, push, change phase gates, or start work outside the active phase.

Claim and release actions should append progress events so another terminal or agent can reconstruct who worked on what and when.

## Acceptance, Commit, And Push Records

Phase completion is not just "code looks done." A phase slice is complete only after implementation, acceptance, commit, and push have all succeeded.

The Phase Gate MVP should record:

- The exact acceptance commands run for the phase.
- Pass or fail result for each command.
- Any skipped checks and the reason they were skipped.
- The accepted commit hash.
- The remote and branch used for push.
- The push remote, branch, and result, with the accepted commit hash stored in the commit record.

These records are local execution evidence. They should support handoff between tools and agents, but they should not require a hosted service.

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
- Do not let sub-agents start the next phase before the current phase is accepted, committed, and pushed.
- If acceptance fails, keep the phase active and record the failed gate before retrying.

This keeps task history, validation evidence, and Git history aligned.

## Boundary

The current task manager does not yet:

- Automatically split user intent into tasks.
- Enforce leases, claim conflicts, or automatic lifecycle closure.
- Export progress reports into repository documentation.
- Sync with GitHub Issues, Linear, Jira, Notion, or any hosted service.

Those are future slices. The first priority is a reliable local task state that all terminal agents can read.
