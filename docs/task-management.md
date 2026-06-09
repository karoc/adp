# Task Management

Simplified Chinese: [task-management.zh-CN.md](task-management.zh-CN.md)

ADP's Task and Progress Manager is the local source of truth for workspace-scoped planning and execution state. It keeps project planning outside the real project root, then lets terminal users and agents inspect the same task list before choosing the next work item.

This layer is intentionally not a Web dashboard, SaaS tracker, or issue-hosting replacement. It is a terminal-first, local-first state manager for agent work.

## Implemented Scope

The first task-management slice provides:

- `adp tasks add`
- `adp tasks list`
- `adp tasks next`
- `adp tasks take`
- `adp tasks renew`
- `adp tasks stale`
- `adp tasks show`
- `adp tasks update`
- `adp tasks claim`
- `adp tasks release`
- `adp tasks done`
- `adp tasks block`
- `adp phase add`
- `adp phase list`
- `adp phase show`
- `adp phase status`
- `adp phase start`
- `adp phase accept`
- `adp phase commit`
- `adp phase push`
- `adp plan preview [--workspace <name>] --file <path|-> [--format text|json]`
- `adp plan apply [--workspace <name>] --file <path|-> [--format text|json]`
- `adp plan doctor [--workspace <name>] [--format text|json]`
- `adp progress`
- `adp progress report [--workspace <name>] [--language <en|zh-CN>] [--format markdown|json]`
- Read-only `--format json` output for task, phase, and progress inspection.
- `adp run <agent> --task <task-id>` runtime binding.
- `adp run <agent> --take --owner <owner> [--lease <duration>]` launch-time atomic task pickup and runtime binding.
- Workspace-local planning files under `$ADP_HOME/workspaces/<workspace>/planning/`.
- JSONL progress events for task creation and status changes.
- Runtime event and session evidence linked by task ID.
- Planning-file locking for mutating task and phase commands.
- Claim conflict handling, optional claim leases, and owner-checked release.
- Phase lifecycle guards for acceptance, commit evidence, push evidence, and next-phase start.

Smoke scripts should assert only the task-management commands that exist in the current tree, and should not add placeholder checks for planned commands.

## Command Discovery

Use the CLI itself as the command reference before changing local state:

```bash
adp --help
adp tasks --help
adp tasks take --help
adp tasks renew --help
adp tasks stale --help
```

The pattern is `adp <command> --help` for command groups and `adp <command> <subcommand> --help` for leaf commands. Leaf help currently prints usage plus `See also:` back to the parent help page. If a build adds friendly `try:` hints, treat them as navigation to run manually; hints do not mutate the planning ledger, start agents, call providers, run Git, or write runtime/planning files into the project root.

## Workflow At A Glance

For a first trial or a new sub-agent, read the command surface in this order:

1. Inspect before changing state:

```bash
adp workspace doctor <workspace>
adp tasks next --workspace <workspace> --format json
adp phase status --workspace <workspace> --format json
adp progress report --workspace <workspace> --format json
```

2. Create or import planned work explicitly:

```bash
adp tasks add --workspace <workspace> --phase <phase-id> --priority high "<title>"
adp plan preview --workspace <workspace> --file plan.yaml --format json
adp plan apply --workspace <workspace> --file plan.yaml --format json
```

3. Take one item from the board:

```bash
adp tasks take --workspace <workspace> --owner <owner> --lease 4h --format json
adp run codex --workspace <workspace> --take --owner <owner> --lease 4h -- <codex-args>
adp run claude --workspace <workspace> --take --owner <owner> --lease 4h -- <claude-args>
```

Use `tasks take` when a worker should atomically claim a task without launching a runtime. Use `run --take` when claim and runtime launch should happen together. Use `run --task <task-id>` only when a human or previous step has already assigned a specific task.

4. Maintain and hand off work:

```bash
adp tasks renew --workspace <workspace> <task-id> --owner <owner> --lease 4h
adp tasks stale --workspace <workspace> --format json
adp sessions restore-plan <session-id>
```

5. Finish through explicit local commands:

```bash
adp tasks done --workspace <workspace> <task-id>
adp phase accept --workspace <workspace> <phase-id> --command "scripts/check-all.sh" --result passed
adp phase commit --workspace <workspace> <phase-id> --hash <commit-hash>
adp phase push --workspace <workspace> <phase-id> --remote origin --branch main --result pushed
```

`tasks next`, `tasks stale`, `phase status`, `progress report`, `plan preview`, and `sessions restore-plan` are inspection or proposal commands. `tasks add`, `plan apply`, `tasks take`, `tasks claim`, `tasks renew`, `tasks release`, `tasks done`, `tasks block`, `phase start`, `phase accept`, `phase commit`, and `phase push` mutate the local ADP ledger. None of these commands run Git automatically, close a phase automatically, sync a hosted tracker, scrape provider-private task panels, or write planning/runtime files into the real project root.

## Reading The Task Board

Operators should treat the ADP task list as a local board. The fastest read path is:

```bash
adp tasks next --workspace <workspace> --format json
adp tasks stale --workspace <workspace> --format json
adp progress report --workspace <workspace> --format json
```

In task JSON, `claim_state` is the derived board state: `unclaimed`, `claimed`, `leased`, or `stale`. `owner` is the human or agent that currently holds the task. `claimed_at` is the UTC timestamp when ownership was recorded or refreshed. `lease_expires_at` is the UTC deadline for the current claim; when it is omitted, the claim has no lease expiration until it is released or reclaimed by the same owner. When `lease_expires_at` is present and no longer in the future, `claim_state` becomes `stale`.

Use `tasks next` to see prioritized board candidates before mutation. It is a preview, not a reservation. Its `candidates` and `next` entries include `claim_state` and can include active owner and lease fields when the underlying task has them, so an operator can distinguish unclaimed work from in-progress or review work before deciding whether to take, claim, release, renew, or inspect further.

Use `tasks stale` to inspect expired `in_progress` claims. Its JSON includes `stale_count` and a `tasks` list. An empty stale list means ADP does not currently see expired in-progress claims; it does not mean there is no ready work, no active work, or no phase gate left to finish.

Use `progress report --format json` for a broader handoff snapshot. It includes all task objects with `claim_state`, prioritized `next` work, phase evidence, and recent local runtime sessions when local JSONL evidence exists. It is the right board-wide view when starting another terminal agent, but it is still read-only and must not be persisted as a second planning store.

When starting a worker, prefer `adp run <agent> --take --owner <owner> --lease <duration>` so the board pickup and runtime launch share one command boundary. The selected task's `claim_state`, `owner`, `claimed_at`, and `lease_expires_at` are then visible in task JSON, with owner and lease timestamps also visible in runtime environment variables such as `ADP_TASK_OWNER`, `ADP_TASK_CLAIMED_AT`, and `ADP_TASK_LEASE_EXPIRES_AT`, generated adapter instructions, and runtime/session evidence. Provider-native task boxes may mirror those values for visibility, but ADP remains the authoritative board.

Expired or stale claims still require an explicit ADP command. Inspect with `adp tasks stale --workspace <workspace> --format json`, then either renew as the current owner with `adp tasks renew` before the lease expires or reclaim expired work through `adp tasks take` or explicit `adp tasks claim` after ADP ownership rules allow it. ADP does not automatically release a task, recover a worker, close work, accept a phase, run Git, push, scrape provider-private state, sync hosted trackers, or write planning/runtime files into the real project root.

## ADP Planning Contract

ADP is the authoritative local planning and progress ledger for a workspace. Terminal users, Codex, Claude, and future agent tools should create, claim, update, block, release, and complete durable work through ADP commands instead of keeping the only copy of task state in a provider-native todo list or chat transcript.

The durable task command flow for agents is, with `tasks take` preferred for atomic pickup:

```bash
adp tasks next --workspace <workspace> --format json
adp tasks add --workspace <workspace> --phase <phase-id> --priority <priority> "<title>"
adp tasks take --workspace <workspace> --owner <owner> --lease <duration> --format json
adp run <agent> --workspace <workspace> --take --owner <owner> --lease <duration> -- <agent-args>
adp tasks claim --workspace <workspace> <task-id> --owner <owner> --lease <duration>
adp tasks renew --workspace <workspace> <task-id> --owner <owner> --lease <duration>
adp tasks stale --workspace <workspace> --format json
adp tasks update --workspace <workspace> <task-id> --status in_progress
adp tasks block --workspace <workspace> <task-id> --reason "<reason>"
adp tasks release --workspace <workspace> <task-id> --owner <owner>
adp tasks done --workspace <workspace> <task-id>
adp progress report --workspace <workspace> --format json
```

`adp tasks next` is a read-only selection snapshot. It helps an agent choose work, but it does not claim the task. Durable ownership and recovery evidence start with `adp tasks claim` for a selected task or `adp tasks take` when the worker should atomically select and claim the next eligible task.

Multi-agent workers should prefer `adp tasks take` over a separate `tasks next` plus `tasks claim` sequence. `tasks take` combines selection and ownership into one planning-lock-protected mutation so two agents cannot take the same task concurrently.

When the worker is being launched through ADP, prefer `adp run <agent> --take --owner <owner> [--lease 4h]` so task selection, ownership recording, runtime context generation, and agent launch use one command boundary. `--take` and `--task <task-id>` are mutually exclusive: use `--task` only when the operator has already assigned a specific task.

Phase progress follows the same rule. Agents can inspect gates with `adp phase status --workspace <workspace> --format json`, but acceptance, commit evidence, and push evidence stay explicit through `adp phase accept`, `adp phase commit`, and `adp phase push`. ADP must not infer task completion, phase acceptance, or Git state from a provider session exit code.

## Tool Taskbox Bridge

If an agent tool exposes a native task or todo panel, the agent should mirror the active ADP task into that panel when work starts. The mirrored item can show the ADP task ID, title, status, phase, owner or lease, and short local subtasks so the tool's task box matches the work being executed. For `adp run --take`, the task selected and claimed at launch time is the active task to mirror.

The provider-native task box is a visual and scratch surface only. It must not become the durable task store, and ADP must not read or sync provider-private todo state as authoritative planning data. Any durable change still belongs in ADP through the task and phase commands above.

The current integration boundary is instruction-level mirroring unless a provider exposes a stable local API that ADP can call without making provider-private state authoritative. If a native panel is unavailable, agents continue normally with the ADP ledger and terminal commands.

## Tool Plan Mode Bridge

Provider-native plan mode or plan panels are proposal and scratch views. They can help an agent organize a candidate plan for the operator, but ADP remains the authoritative local planning and progress ledger.

While a tool is in plan mode, the agent should avoid implementation edits, task completion, phase acceptance, commits, pushes, or other execution side effects unless the user explicitly approves leaving planning and starting execution. Planning proposals should first be checked with the read-only intake path:

```bash
adp plan preview --workspace <workspace> --file - --format json
```

Only after explicit user or operator approval should the plan be written into ADP:

```bash
adp plan apply --workspace <workspace> --file - --format json
```

After a plan is applied, durable task state still follows the existing task and phase commands. Provider-native plan items may mirror the ADP plan for readability, but they are not a second ledger and must not be treated as recovery or progress evidence.

If `adp run --take` launches a provider in plan mode, the task ownership has already been recorded in ADP, but the native plan remains a proposal surface. The worker still needs explicit ADP commands for status changes and phase evidence after execution is approved.

## Command Surface Metadata And Drift Checks

P16 is a command-surface hardening slice, not a new task-management feature. It adds a local command metadata contract so usage text, dispatch wiring, and bash/zsh completion can be checked against the same command inventory instead of drifting independently.

The metadata contract is local maintenance evidence for the existing hand-written CLI. It is not a new CLI framework, Web dashboard, SaaS tracker, hosted orchestration service, hosted tracker sync, automatic Git workflow, automatic task closure path, provider-native resume mechanism, or project-root planning export.

Future command changes should update the metadata contract, usage text, dispatch path, bash/zsh completion, focused tests, and smoke or documentation acceptance in the same phase. Read-only command metadata must not claim tasks, close tasks, accept phases, append planning events, run Git, start agents, mutate runtime state, write project-root files, or become a second planning store.

## Progress Report Scope

P6 added Markdown reporting, and P8 extends the same read-only command with a JSON handoff snapshot:

- `adp progress report [--workspace <name>] [--language <en|zh-CN>] [--format markdown|json]`

The command prints a local planning/execution handoff snapshot to stdout. It reads workspace planning data from `$ADP_HOME`, uses English Markdown by default, and emits Simplified Chinese Markdown only when `--language zh-CN` is provided. `--language` applies to Markdown output only; JSON output keeps stable machine-readable field names and enum values for cross-tool parsing.

With `--format json`, the command emits a read-only handoff snapshot with workspace, total task count, phases, task counts, task objects including `claim_state` plus `owner`, `claimed_at`, and `lease_expires_at` when present, priority-sorted next work, phase evidence, and recent runtime session evidence when local JSONL runtime events and session data exist. The JSON snapshot is an inspection format, not a separate state store. The authoritative state remains the local planning ledger under `$ADP_HOME` plus local JSONL evidence such as `$ADP_HOME/logs/events.jsonl`.

The report is an inspection view, not a state transition. It does not append events, mutate task state, mutate phase state, create runtime directories, run agents, run Git, resume provider-native conversations, or write report files into project roots.

## Next Work Endpoint Scope

P10 defines a narrower read-only endpoint for choosing the next local task without parsing the full progress report:

- `adp tasks next [--workspace <name>] [--limit <n>] [--format text|json]`

The command reads the workspace planning ledger under `$ADP_HOME`, selects eligible tasks, sorts them by priority and stable local tie-breakers, and prints the best next work candidates to stdout. It is intended for terminal users and local sub-agents that need a small task-selection snapshot before explicitly claiming or updating a task.

Eligible statuses are `ready`, `in_progress`, and `review`. `planned`, `blocked`, `validated`, `done`, and `canceled` tasks stay visible in list, show, progress, and report views, but they are not selected by `adp tasks next`.

Text output is the default and is optimized for terminal scanning. `--limit <n>` caps the candidate list; the default is 5, and `--limit 0` means no truncation. JSON output uses stable machine-readable fields and enum values so cross-tool callers can select a task without scraping text.

The JSON contract includes:

- `workspace`: workspace name.
- `planning_source`: local planning file path used for the snapshot.
- `generated_at`: UTC snapshot timestamp.
- `total`: total task count in the workspace ledger.
- `eligible_count`: candidate count after status filtering and limit handling.
- `counts`: task counts by status across the full ledger.
- `limit`: requested limit.
- `candidates`: candidate task list in selection order.
- `next`: the first candidate when at least one task is eligible; omitted when there is no eligible work.

Each candidate uses the same task object shape as `adp tasks list --format json`, including task ID, title, status, priority, phase ID, `claim_state`, `owner`, `claimed_at`, `lease_expires_at` when present, and blocker summary when relevant.

The endpoint is read-only. It must not claim a task, update status, change owners or leases, clear blockers, mutate phases, append planning or runtime events, create runtime directories, start agents, run Git, infer acceptance, close tasks, write output files into the project root, sync with a hosted service, or maintain JSON output as a second planning store.

## Atomic Task Take Scope

P43 defines the mutating endpoint for taking one task from the local board in a single operation:

```bash
adp tasks take [--workspace <name>] --owner <owner> [--lease <duration>] [--format <text|json>]
```

The command is for multi-agent coordination. It acquires the workspace planning lock, selects the highest-priority claimable task using the same stable local ordering principles as `adp tasks next`, records the owner and optional lease, moves the task to `in_progress`, writes the local planning ledger, and returns the task that was taken.

By default, claimable tasks are:

- `ready` tasks with no active owner, including tasks whose previous lease has expired.
- `in_progress` tasks with an owner whose previous lease has expired.

By default, `planned`, `blocked`, `review`, `validated`, `done`, and `canceled` tasks are not taken. `review` can still appear in read-only `tasks next` snapshots, but atomic pickup does not assign review work unless a future explicit option adds that behavior.

The output should include the same task object shape used by task inspection commands, including task ID, title, status, priority, phase ID, `claim_state`, `owner`, `claimed_at`, and `lease_expires_at` when present. JSON output is a command result for local tools; it is not a second planning store.

`tasks take` is a task-ownership mutation only. It must not launch a runtime, run Git, accept a phase, record commit or push evidence, mark the task done, close a phase, infer completion from an agent exit code, sync with a hosted tracker, or write planning files into the real project root. Runtime launch remains a separate `adp run ...` decision, and task completion remains explicit through `adp tasks done` or another explicit status command.

## Run Take Bridge Scope

P44 connects atomic task pickup to runtime launch:

```bash
adp run <agent> [--workspace <name>] --take --owner <owner> [--lease <duration>] [--profile <profile>] [--keep-runtime] -- <agent-args>
```

`adp run --take` is for launch-time multi-agent ownership. Before building the runtime root or starting the external agent command, ADP takes the highest-priority claimable task under the workspace planning lock, records the owner and optional lease, moves the task to `in_progress`, then binds that task to the runtime exactly as if the operator had launched with `--task <task-id>`.

`--take` is mutually exclusive with `--task <task-id>`. Use `--task` for an already assigned task, and use `--take` when a worker should atomically pick up the next eligible item from the ADP board. If no claimable task exists, ADP should fail before launching the provider command.

The bridge does not complete the task, accept a phase, record commit or push evidence, run Git, infer success from the provider exit code, sync provider-native task panels, or write planning files into the real project root. Provider-native task boxes and plan panels may mirror the active task for local visibility, but ADP remains the authoritative ledger.

## Task Lease Maintenance Scope

P45 defines explicit lease maintenance for long-running and interrupted workers:

```bash
adp tasks renew --workspace <workspace> <task-id> --owner <owner> --lease <duration>
adp tasks stale --workspace <workspace> [--format text|json]
```

`tasks renew` extends an owned task lease under the workspace planning lock. The caller must provide the current owner and a new lease duration, and ADP updates the local planning ledger instead of relying on any provider-native task state.

`tasks stale` is read-only. It lists `in_progress` tasks whose leases have expired so operators and workers can see interrupted or abandoned work without mutating ownership. It must not append events, change task status, renew a lease, release a task, start an agent, run Git, accept a phase, or write files into the project root.

Workers launched with `adp run --take` should renew during long work before their lease expires. If a session is interrupted, the expired claim becomes visible through `tasks stale`; after expiration, another worker can reclaim the task with `tasks take` or explicit `tasks claim` according to ADP ownership rules. None of these commands automatically mark work done, accept phases, record commit or push evidence, execute Git, scrape provider-private task boxes, or trust provider plan panels as recovery state.

## Lease-Aware Runtime Handoff

Runtime handoff is ADP-led. A worker should hand off with enough local evidence for the next operator or terminal agent to answer four questions: which ADP task is active, who owns it, when its lease expires, and what explicit ADP command should happen next.

Recommended handoff inspection commands are:

```bash
adp tasks show --workspace <workspace> <task-id> --format json
adp tasks stale --workspace <workspace> --format json
adp phase status --workspace <workspace> --format json
adp progress report --workspace <workspace> --format json
adp sessions list --workspace <workspace> --task <task-id>
adp sessions restore-plan <session-id>
```

If the worker is still making progress, renew before handing off or before the lease expires:

```bash
adp tasks renew --workspace <workspace> <task-id> --owner <owner> --lease <duration>
```

If work was interrupted, inspect with `tasks stale` first. Reclaim expired work only through `adp tasks take` or explicit `adp tasks claim`; do not recover ownership from provider-private task boxes, plan panels, chat transcripts, conversation IDs, process exits, or native resume state.

Provider-native task panels may mirror the active ADP task, and provider-native plan mode may hold proposals, but neither surface is the handoff ledger. Handoff does not automatically complete a task, accept a phase, record commit or push evidence, run Git, apply a plan, start the next phase, or write planning/runtime files into the real project root. Runtime artifacts stay under `$ADP_RUNTIME_DIR`; planning ledgers, progress, events, and sessions stay under `$ADP_HOME`.

## Phase Gate Status Scope

P24 adds a read-only phase gate snapshot for local tools and terminal agents:

- `adp phase status [--workspace <name>] [--format text|json]`

The command reads the local phase ledger, identifies the earliest phase whose gate is not satisfied, reports any currently open phase, reports the next planned phase, tells whether another phase can start, and prints the next required action. The action is one of `record_acceptance`, `record_commit`, `record_push`, `start_next_phase`, or `plan_next_phase`.

The snapshot is inspection only. It does not start phases, accept phases, record commit or push evidence, mutate tasks, append events, run Git, push, start agents, sync hosted trackers, or write planning files into the project root.

Phase ordering is explicit for new phase records and plan imports. Existing phase records without an order keep their stable created-time and ID ordering for compatibility. A later phase cannot start until every earlier phase is satisfied, and a phase is satisfied only when it has successful pushed evidence recorded in the local phase ledger.

## Planning Intake Scope

P14 adds local planning intake commands for deterministic, cross-tool phase/task input:

```bash
adp plan preview --workspace <name> --file <path|-> [--format text|json]
adp plan apply --workspace <name> --file <path|-> [--format text|json]
```

The first version accepts structured YAML or JSON only. It does not split free-text natural language into tasks inside ADP.

`preview` is read-only: it parses and validates the input, then prints the phase and task changes that would be created. It must not write planning files, append events, create runtime directories, start agents, run Git, change task ownership, close tasks, accept phases, sync hosted trackers, or write report/planning exports into the project root.

`apply` is explicit. It writes only the local planning ledger under `$ADP_HOME/workspaces/<workspace>/planning` after validation succeeds, and JSON output remains an inspection format rather than a second planning store. The feature remains terminal-first and local-first; it is not a Web UI, dashboard, SaaS tracker, cloud sync layer, hosted orchestration service, hosted tracker sync, automatic Git workflow, automatic claim/done/phase acceptance flow, provider-native resume flow, or project-root report/planning export path.

## Planning Ledger Diagnostics Scope

P26 adds a read-only local planning ledger doctor:

```bash
adp plan doctor [--workspace <name>] [--format text|json]
```

The command inspects `$ADP_HOME/workspaces/<workspace>/planning/` and reports task, phase, progress-log, lock, and phase-gate consistency diagnostics. It treats `tasks.yaml` and `phases.yaml` as the current-state snapshots, and treats `progress.jsonl` as append-only audit evidence rather than a replay source for rebuilding state.

Text output prints a compact terminal summary with workspace, planning directory, status, task count, phase count, progress event count, error and warning counts, phase gate action, and diagnostics. JSON output is a single inspection object with the same counts, local file paths, phase gate snapshot, `has_errors`, and `diagnostics`.

Diagnostic levels are `info`, `warning`, and `error`. A healthy ledger and warning-only diagnostics return exit code `0`; error-level diagnostics return exit code `2` after printing the report. CLI usage or workspace resolution failures remain normal command failures.

The doctor is read-only. It does not repair files, remove locks, create missing planning files, append progress events, claim or close tasks, mutate phases, infer acceptance, record commit or push evidence, run Git, push, start agents, create runtime directories, sync hosted trackers, write project-root reports, or maintain JSON output as a second planning store.

## Storage

Task state lives under the ADP workspace directory:

```txt
$ADP_HOME/workspaces/<workspace>/
└── planning/
    ├── tasks.yaml
    ├── phases.yaml
    ├── .lock
    └── progress.jsonl
```

ADP does not write these files into the real project root. Planning and report output should stay on stdout or under `$ADP_HOME`; repository documentation may summarize accepted behavior manually, but ADP must not provide project-root planning or report export paths.

The Phase Gate MVP keeps using this local planning directory and extends it with structured phase and gate records. The storage remains local-first and terminal-readable:

- `tasks.yaml`: task list, status, priority, phase ID, owner, claim timestamp, and optional lease expiration.
- `phases.yaml`: phase records, phase status, acceptance records, commit records, push records, and gate summary.
- Phase records include a local order value for new phases so the gate can prevent skipping earlier planned or unfinished phases.
- `.lock`: short-lived local planning mutation lock. It is created around task and phase writes, and stale lock files are removed after the configured stale age.
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
adp tasks next --workspace adp
adp tasks next --workspace adp --limit 0
adp tasks list --workspace adp --format json
adp tasks show --workspace adp <task-id> --format json
adp tasks next --workspace adp --format json
```

Move a task through execution, using atomic pickup for parallel workers:

```bash
adp tasks take --workspace adp --owner codex-main --lease 30m --format json
adp run codex --workspace adp --take --owner codex-main --lease 30m -- --version
adp tasks claim --workspace adp <task-id> --owner codex-main --lease 30m
adp tasks renew --workspace adp <task-id> --owner codex-main --lease 30m
adp tasks stale --workspace adp --format json
adp tasks update --workspace adp <task-id> --status in_progress
adp tasks block --workspace adp <task-id> --reason "waiting for real CLI evidence"
adp tasks release --workspace adp <task-id> --owner codex-main
adp tasks done --workspace adp <task-id>
```

Record a phase and its gate evidence:

```bash
adp phase add --workspace adp --goal "phase gate MVP" p3 "Project planning and execution progress"
adp phase status --workspace adp
adp phase start --workspace adp p3
adp phase accept --workspace adp p3 --command "scripts/check-all.sh" --result passed --notes "local gate passed"
adp phase commit --workspace adp p3 --hash <commit-hash> --message "Implement phase gate MVP"
adp phase push --workspace adp p3 --remote origin --branch main --result pushed
adp phase show --workspace adp p3
adp phase list --workspace adp --format json
adp phase show --workspace adp p3 --format json
adp phase status --workspace adp --format json
```

Summarize progress:

```bash
adp progress --workspace adp
adp progress --workspace adp --format json
```

Progress report output:

```bash
adp progress report --workspace adp
adp progress report --workspace adp --language zh-CN
adp progress report --workspace adp --format json
```

The report command prints to stdout only. The default format is Markdown, and it does not create or update report files.

When `--workspace` is omitted, ADP uses the same workspace resolution model as other workspace-aware commands: `ADP_WORKSPACE` first, then the current directory if it is inside a registered project root.

## Machine-Readable Inspection

Read-only task, phase, and progress views support `--format json` so local tools and sub-agents can parse the planning ledger without scraping terminal text:

```bash
adp tasks list --workspace adp --format json
adp tasks show --workspace adp <task-id> --format json
adp tasks next --workspace adp --format json
adp phase list --workspace adp --format json
adp phase show --workspace adp <phase-id> --format json
adp phase status --workspace adp --format json
adp progress --workspace adp --format json
adp progress report --workspace adp --format json
```

The JSON output is an inspection format, not a separate state store. The authoritative planning state remains under `$ADP_HOME/workspaces/<workspace>/planning/`, and progress evidence remains in the local `progress.jsonl` ledger. Runtime session evidence remains derived from local JSONL events when those events exist. Repository docs may describe the plan, but they do not become the source of truth for execution state.

Cross-tool consumers should treat JSON output as a local snapshot for selecting work, showing status, or handing context to another terminal agent. They should still call explicit mutating commands when state needs to change:

- Use `adp tasks next --format json` when a local tool needs a compact prioritized task-selection snapshot.
- Use `adp tasks take --owner <owner> --lease <duration> --format json` when a local worker needs to atomically select and claim one task from the board.
- Use `adp tasks claim`, `adp tasks update`, `adp tasks done`, `adp tasks block`, or `adp tasks release` for task changes.
- Use `adp phase start`, `adp phase accept`, `adp phase commit`, and `adp phase push` for phase transitions.
- Use `adp phase status --format json` when a local tool needs to know whether the current phase needs acceptance, commit evidence, push evidence, or whether the next planned phase can start.
- Do not infer acceptance from a passing command or close a task automatically without an explicit task or phase command.
- Do not treat JSON output as permission to run Git, push changes, start the next phase, or modify the project root.

The phase discipline is unchanged: a phase is complete only after implementation, acceptance, commit evidence, and push evidence have been recorded through explicit local commands.

## Progress Report Outputs

`adp progress report [--workspace <name>] [--language <en|zh-CN>] [--format markdown|json]` produces a terminal-friendly Markdown report or a machine-readable JSON handoff snapshot. The output summarizes the local planning and execution state without becoming a separate source of truth.

Recommended Markdown report content:

- Workspace name and local planning source.
- Phase summary, including active, accepted, committed, and pushed phases.
- Prioritized next work from the local task ledger.
- Active owners, leases, blocked tasks, and acceptance evidence when available.
- Commit and push evidence already recorded in the phase ledger.
- Recent local runtime session evidence when JSONL event/session data exists, including available session IDs, agents, task IDs, statuses, exit codes, durations, and runtime paths.

Language behavior:

- `--language en` and omitted `--language` both produce English output.
- `--language zh-CN` produces Simplified Chinese output.
- `--language` applies to Markdown only. JSON output uses stable machine-readable fields and values.
- Other language values fail clearly.

Format behavior:

- `--format markdown` and omitted `--format` both produce Markdown.
- `--format json` produces a single read-only handoff snapshot for local tools and terminal agents.
- The JSON snapshot includes workspace, total task count, phases, task counts, task objects with `claim_state` plus owner and lease fields when present, priority-sorted next work, phase evidence, and recent runtime session evidence when local JSONL event/session data exists.
- The `next work` data should be sorted by priority so another local tool can choose likely follow-up work without scraping Markdown.
- JSON output is for cross-tool parsing and must not be persisted or treated as ADP's state store.

Read-only boundary:

- Do not update task status, owner, lease, or blocker records.
- Do not update phase status, acceptance records, commit records, or push records.
- Do not append planning or runtime events.
- Do not run Git commands, create commits, push, or infer Git state transitions.
- Do not build runtimes, create runtime directories, start agents, or prune runtime directories.
- Do not resume provider-native conversations or infer provider session state beyond local JSONL event evidence.
- Do not create or update Markdown files in the real project root.
- Do not create or update JSON report files in the real project root or use JSON output as a synchronized planning ledger.

## Phase Gate Ledger

P3's phase gate work turns the task list into a phase-aware execution ledger that multiple terminal agents can share without adding a Web dashboard, SaaS tracker, cloud sync, hosted orchestration, or remote issue service.

The ledger makes these records explicit:

- Phase records: ID, title, local order, status, objective or goal, acceptance command list, commit evidence, push evidence, and latest gate outcome.
- Task claim records: task ID, owner, claimed timestamp, optional lease expiration, release evidence, and current ownership state.
- Acceptance records: phase ID, command list, result, timestamp, and short evidence text.
- Gate records: phase ID, gate status, required checks, and operator or agent notes when they are provided.
- Commit records: phase ID, commit hash, branch, summary, timestamp, and whether the commit contains only the accepted phase.
- Push records: phase ID, remote, branch, timestamp, and push result. Commit hash evidence is stored separately on the phase record before push evidence is recorded.

The implementation stays intentionally small. It optimizes for reliable local evidence over broad project-management features.

## Task Claim And Ownership

Task ownership is a coordination hint for local multi-agent execution. It is not an authorization system.

Current claim rules:

- A ready task may be claimed by one owner at a time.
- The owner may be a human name, agent name, or stable local agent identifier.
- For parallel workers, prefer `tasks take` because it selects and claims under one planning lock.
- Claiming a task records ownership before implementation starts.
- `--lease <duration>` records an optional lease expiration. When the lease has expired, another owner may claim the task.
- `tasks renew --owner <owner> --lease <duration>` extends the lease for the current owner under the planning lock.
- `tasks stale` lists expired `in_progress` leases without changing the ledger.
- A claim without `--lease` is non-expiring until it is released or reclaimed by the same owner.
- Reclaiming with the same owner refreshes the claim timestamp and lease.
- A different owner cannot claim a task while the current owner's lease is still active.
- `tasks release --owner <owner>` checks ownership before clearing it. Omitting `--owner` performs an unowned release for manual recovery.
- Releasing a task clears ownership when the worker is done, blocked, or reassigned.
- `done` and `canceled` tasks cannot be claimed.
- A claimed task still has to respect its assigned file boundaries and phase scope.
- Sub-agents do not commit, push, change phase gates, or start work outside the active phase.

Claim and release actions should append progress events so another terminal or agent can reconstruct who worked on what and when.

## Acceptance, Commit, And Push Records

Phase completion is not just "code looks done." A phase slice is complete only after implementation, acceptance, commit, and push have all succeeded.

The phase ledger records:

- The exact acceptance commands run for the phase.
- Pass or fail acceptance result.
- Any skipped checks and the reason they were skipped.
- The accepted commit hash.
- The remote and branch used for push.
- The push remote, branch, and result, with the accepted commit hash stored in the commit record.

These records are local execution evidence. They should support handoff between tools and agents, but they should not require a hosted service.

Lifecycle guards are enforced locally:

- `phase start` follows phase order. A later phase cannot start while any earlier phase lacks successful pushed evidence, including earlier `planned`, `active`, `accepted`, or `committed` phases.
- `phase accept --result passed` moves a phase to `accepted`.
- `phase accept --result failed` records evidence and keeps or returns the phase to `active`.
- `phase commit` requires `accepted` status and `acceptance.result == passed`.
- `phase push` requires recorded commit evidence and a non-empty commit hash.
- `phase push --result failed` records evidence without advancing the phase beyond its prior committed state, and it cannot overwrite already recorded successful push evidence.
- `phase status` summarizes the current gate without changing the ledger.
- A phase is complete only after it reaches `pushed`.

## Runtime Binding

Attach a task to an agent runtime session:

```bash
adp run codex --workspace adp --task <task-id> -- --version
adp run codex --workspace adp --take --owner codex-main --lease 4h -- --version
```

The task ID is resolved inside the selected workspace. If the task does not exist in that workspace, ADP fails before building a runtime or launching the agent process. With `--take`, ADP atomically selects and claims the next eligible task before runtime creation; `--take` and `--task` cannot be used together.

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

`adp run <agent> --task <task-id>` and `adp run <agent> --take --owner <owner>` do not automatically complete work. `--take` records launch-time ownership and moves the task to `in_progress`, but later status changes remain explicit through `adp tasks update`, `adp tasks done`, and `adp tasks block`.

## Phase Discipline

Task management is intended to support phase-by-phase delivery:

- Prioritize planned work before starting execution.
- Complete one phase slice at a time. The local phase ledger enforces one open phase before the next phase can be started.
- Run the relevant runtime smoke and full repository gate for that phase.
- Commit and push the accepted phase before starting the next phase.
- Do not mix the next phase into the same commit just because the working tree is open.
- Do not let sub-agents start the next phase before the current phase is accepted, committed, and pushed.
- If acceptance fails, keep the phase active and record the failed gate before retrying.

This keeps task history, validation evidence, and Git history aligned.

## Boundary

The current task manager intentionally does not provide:

- Automatically split user intent into tasks.
- Write or export progress reports into repository documentation or project-root files automatically.
- Maintain JSON report output as a second planning store.
- Maintain `adp tasks next --format json` output as a second planning store.
- Sync with GitHub Issues, Linear, Jira, Notion, or any hosted service.
- Run Git commit or Git push commands automatically.
- Infer acceptance from command output or close tasks automatically without explicit task or phase commands.

These remain out of scope. Future slices should strengthen the local ledger, inspection views, diagnostics, and runtime binding without adding hosted sync, automatic Git, provider-native resume, or project-root exports.
