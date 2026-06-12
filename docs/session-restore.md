# Session Resume Plan

Simplified Chinese: [session-restore.zh-CN.md](session-restore.zh-CN.md)

ADP session resume planning is a read-only CLI aid for reconstructing how a local agent run was started and how another local run could continue the same ADP work context. It helps an operator review a previous session, inspect task and phase state, and decide whether to start a new run with the same or a different agent.

It is not provider-native Codex or Claude conversation resume, automatic replay, hosted orchestration, cloud sync, or remote issue tracking. ADP keeps the source of truth local and terminal-readable.

## Resume Plan Command

P50 provides this read-only command surface:

```bash
adp sessions resume-plan <session-id> [--workspace <name>] [--owner <owner>] [--lease <duration>] [--agent <agent>] [--format <text|json>]
```

The command is an inspection and proposal command only. It reads local ADP session evidence and prints a plan for a new `adp run ...` invocation plus any lease-aware preflight notes. It does not run the suggested commands.

Options are interpreted as planning inputs:

- `--workspace <name>` overrides the target workspace used for current task and phase inspection. If it differs from the source session workspace, output must say that the task and phase state came from the target workspace while the historical session came from the source workspace.
- `--owner <owner>` names the operator or worker that would continue the ADP task. It does not claim, renew, release, or complete the task.
- `--lease <duration>` sets the lease duration to show in suggested ownership commands. It does not create or extend a lease.
- `--agent <agent>` asks for a cross-tool plan using a different ADP adapter, such as planning a Claude continuation from a previous Codex session. It does not transfer provider-native conversation state.
- `--format text|json` controls only stdout shape. JSON output is for local tooling and must not become a second state store.

`adp sessions restore-plan <session-id>` is the earlier read-only rerun guidance name. `resume-plan` extends that boundary for the cross-tool ADP work-context semantics described here. References to restore planning should preserve the same read-only boundary.

## Resume Boundary

ADP work-context resume means starting a new local run with enough ADP-managed context to continue work:

- workspace and project-root identity from the local workspace registry;
- selected agent adapter and profile;
- non-sensitive agent arguments that followed `--`;
- task ID and task snapshot when the original session was task-bound;
- owner and lease guidance supplied by the operator;
- phase status and next gate hints from the ADP planning ledger;
- local event/session evidence that can be shown to the next terminal agent.

Provider-native resume means reattaching to a provider-private conversation or session handle owned by a tool such as Codex or Claude. ADP resume planning does not do that. It does not scrape provider transcripts, recover native task panels, trust provider plan panels, or pass provider-private session handles between tools.

Cross-tool resume is therefore an ADP handoff, not a provider conversation transfer. A `--agent claude` plan for a previous Codex session can suggest a new `adp run claude ...` command with the same workspace and task context, but the new Claude run starts as a new provider conversation. The durable handoff evidence remains in ADP tasks, phases, progress reports, events, and session history.

When the target agent and workspace match the source session, `resume-plan` can reuse non-sensitive launch details from the invocation snapshot: non-default `--profile`, `--keep-runtime`, and post-`--` agent arguments. When the target agent or workspace changes, ADP keeps ADP-owned runtime options such as `--keep-runtime`, but does not copy provider-specific profile or agent arguments. The output reports those omitted fields and the reason.

## Concepts

Session history answers "what happened":

- `adp events list [--format text|json]` reads local JSONL runtime events.
- `adp sessions list [--format text|json]` groups those events into local agent sessions.
- `adp sessions show <session-id> [--format text|json]` prints one session with its event timeline.

Resume plan answers "how could I start a new run that continues this ADP work context":

- `adp sessions resume-plan <session-id> [--workspace <name>] [--owner <owner>] [--lease <duration>] [--agent <agent>] [--format <text|json>]` inspects a historical session.
- It prints a suggested `adp run ...` command when enough invocation data is available.
- It can include suggested `adp tasks renew` or `adp tasks claim` preflight commands when task ownership guidance is available.
- It reports missing fields and context gaps for older sessions or incomplete event data; the plan is partial only when required launch or task context is unavailable.
- It does not execute suggested commands.
- It does not create a runtime workspace, launch an agent, append new events, change task or phase state, run Git, or write to the real project root.

Future replay would answer "start a new run for me":

- Replay is intentionally not part of this slice.
- A future replay command would still start a new local run, not resume a provider conversation.
- Any future execution command must remain explicit, locally gated, and separate from read-only inspection.

## Invocation Snapshot

Sessions created after the restore-planning foundation record a non-sensitive invocation snapshot on the `run_started` event under `fields.invocation`. The snapshot is designed to support resume planning without copying private runtime state.

The snapshot can include:

- `schema_version`: invocation snapshot schema version.
- `agent_args`: arguments passed after `--` to the provider or local agent command.
- `keep_runtime`: whether `--keep-runtime` was requested.
- `workspace_resolution`: how ADP resolved the selected workspace, such as an explicit flag or environment/default resolution.
- `profile_source`: how the agent profile was selected, such as explicit profile, default profile, or agent default.
- `original_cwd`: the directory where `adp run` was invoked, when available.
- `task_snapshot`: task context captured at run start, including `id`, `title`, `status`, `priority`, and `phase` when a task was bound.

The snapshot must not include:

- Full environment variables.
- Credentials, tokens, API keys, or shell secrets.
- Provider conversation state or provider session handles.
- Full generated adapter instructions.
- Project file contents.
- Remote tracker, hosted orchestration, or cloud synchronization state.

The resume-plan command combines this snapshot with normal session summary fields used to form a safe suggestion: workspace, agent, profile, task ID, and runtime options. Same-tool plans can reuse profile and agent arguments. Cross-tool or cross-workspace plans omit provider-specific profile and agent arguments with explicit guidance instead of silently forwarding them to another tool.

JSON output includes `suggested_commands`. Each suggested command has a machine-readable `side_effect`:

- `inspect`: read-only inspection commands such as `adp sessions show`, `adp progress report`, `adp tasks show`, or `adp tasks stale`.
- `task_mutation`: explicit ownership commands such as `adp tasks claim` or `adp tasks renew`; these are never run by `resume-plan`.
- `runtime_creation`: explicit launch commands such as `adp run ...`; these are never run by `resume-plan`.

For sessions started with `adp run --take`, the snapshot can explain which ADP task was bound at launch, but it is not lease recovery by itself. Long-running owners renew with `adp tasks renew --workspace <workspace> <task-id> --owner <owner> --lease <duration>`, while interrupted sessions become visible through `adp tasks stale --workspace <workspace> [--format text|json]`.

## CLI Usage

Inspect session history first:

```bash
adp events list --workspace game-a --task <task-id>
adp sessions list --workspace game-a --agent codex --task <task-id>
adp sessions show <session-id>
adp events list --workspace game-a --task <task-id> --format json
adp sessions list --workspace game-a --agent codex --task <task-id> --format json
adp sessions show <session-id> --format json
```

Then ask ADP for a read-only resume plan:

```bash
adp sessions resume-plan <session-id> --owner handoff-agent --lease 2h --format text
```

The output is intended to be reviewed by a human or another local tool before any command is run. A ready plan can include commands like:

```bash
adp tasks renew --workspace game-a task-20260608-0003 --owner handoff-agent --lease 2h
adp run codex --workspace game-a --profile senior-engineer --task task-20260608-0003 --keep-runtime -- --example-smoke
```

If data is missing, ADP should report missing fields and reasons instead of pretending the session is fully reconstructable. Old sessions created before invocation snapshots can still produce a ready task-level plan when task and workspace context are available, but the output must show the invocation snapshot gap.

## Cross-Tool Example

Use `--agent <agent>` when the operator wants the next run to use a different ADP adapter from the original session:

```bash
adp sessions resume-plan <session-id> --agent claude --owner reviewer --lease 1h --format json
```

The JSON plan should describe the recorded session, the selected target agent, the suggested local launch command, missing fields, omitted invocation fields, command `side_effect` values, and read-only warnings. If the original session was task-bound and the previous lease has expired, the operator still needs an explicit ADP ownership command before launching the next run:

```bash
adp tasks stale --workspace game-a --format json
adp tasks claim --workspace game-a task-20260608-0003 --owner reviewer --lease 1h
adp run claude --workspace game-a --task task-20260608-0003
```

Only run the claim command when ADP ownership rules allow it. If the same owner still has a valid lease, renew instead of claiming. If the work is unowned and the worker should select the next eligible board item atomically, use `adp run <agent> --take --owner <owner> --lease <duration>`.

## Fake Or Local Agent Workflow

Use a fake local agent when validating resume-plan behavior without depending on an external provider CLI:

```bash
fake_bin="$(mktemp -d)"
cat > "${fake_bin}/codex" <<'SH'
#!/usr/bin/env sh
printf 'fake codex received: %s\n' "$*"
SH
chmod +x "${fake_bin}/codex"
PATH="${fake_bin}:${PATH}"
```

Create a task and bind the runtime session to it:

```bash
TASK_ID=$(adp tasks add --workspace game-a --priority high "Exercise resume-plan guidance" | sed -n 's/^task \(task-[^ ]*\) added$/\1/p')
test -n "$TASK_ID"
adp run codex --workspace game-a --task "$TASK_ID" -- --example-smoke
```

Review the local evidence:

```bash
adp events list --workspace game-a --task "$TASK_ID"
adp sessions list --workspace game-a --agent codex --task "$TASK_ID"
adp sessions show <session-id>
adp events list --workspace game-a --task "$TASK_ID" --format json
adp sessions list --workspace game-a --agent codex --task "$TASK_ID" --format json
adp sessions show <session-id> --format json
adp sessions resume-plan <session-id> --owner handoff-agent --lease 2h --format text
```

Running `resume-plan` should only print inspection output. The event count, task state, phase state, runtime directories, and real project root should not change because of the resume-plan command itself.

If the suggested command is run manually, it starts a new local agent run with a new session ID. It does not attach to the previous provider conversation.

## Lease-Aware Handoff

`sessions resume-plan` is one handoff clue, not the handoff ledger. It can reconstruct a safe launch shape from local invocation evidence, but it does not prove that the previous worker still owns the task, renew a lease, reclaim interrupted work, or know anything from provider-private state.

For task-bound sessions, pair resume planning with ADP planning inspection:

```bash
adp tasks show --workspace <workspace> <task-id> --format json
adp tasks stale --workspace <workspace> --format json
adp phase status --workspace <workspace> --format json
```

When the same owner continues a long-running task, renew before the lease expires:

```bash
adp tasks renew --workspace <workspace> <task-id> --owner <owner> --lease <duration>
```

When a worker was interrupted and the lease has expired, reclaim only through `adp tasks take` or explicit `adp tasks claim`. Do not use provider conversation IDs, native task panels, plan panels, chat transcripts, process exits, or native resume state as ownership evidence.

Phase status in a resume plan is context only. It can tell the next worker whether a phase is open, accepted, blocked by gates, or waiting for evidence, but it must not accept a phase, record commit evidence, record push evidence, start a later phase, or run Git. Complete the normal phase commands only after the required validation has passed.

Plan-mode compatibility follows the same boundary. A provider-native plan panel can mirror a proposed restart or next-step checklist, but structured plan changes become durable only through `adp plan preview` and explicitly approved `adp plan apply`. Resume planning must not automatically edit files, complete tasks, accept phases, commit, push, run Git, apply plans, start runtimes, or write runtime/planning files into the real project root.

## Operating Rules

- Treat resume-plan output as guidance, not as an automatic repair or resume action.
- Review `suggested_commands[*].side_effect` before running any suggested command from JSON output.
- Bind work to tasks with `adp run <agent> --task <task-id>` when the session should be traceable in project planning.
- Prefer `adp run <agent> --take --owner <owner> --lease <duration>` when a worker should atomically take a board item at launch.
- Renew long-running ownership with `adp tasks renew`; use `adp tasks stale` to inspect expired in-progress claims after interruptions.
- Reclaim expired work only through ADP ownership commands such as `adp tasks take` or explicit `adp tasks claim`.
- Move task status explicitly with `adp tasks update`, `adp tasks done`, or `adp tasks block`.
- Move phase status explicitly with `adp phase start`, `adp phase accept`, `adp phase commit`, and `adp phase push`.
- Treat provider-native plan and task panels as mirror or scratch surfaces, not as recovery evidence.
- Use provider-native Codex or Claude resume only when the operator intentionally wants provider-private conversation state; do not treat it as ADP ownership, lease, task, phase, commit, or push evidence.
- Keep acceptance evidence local by pairing resume-plan checks with `adp events list`, `adp sessions list`, `adp sessions show`, `adp sessions restore-plan`, `adp tasks show`, `adp tasks stale`, and `adp phase status`. Use `--format json` when a local tool needs parseable inspection output.
- Do not describe resume-plan as cloud sync, remote issue tracking, hosted orchestration, provider-private state scraping, automatic task completion, automatic phase acceptance, or provider-native resume.
