# Session Restore Plan

Simplified Chinese: [session-restore.zh-CN.md](session-restore.zh-CN.md)

ADP session restore planning is a read-only CLI aid for reconstructing how a local agent run was started. It helps an operator review a previous session and decide whether to start a new run with a similar command.

It is not provider-native conversation resume, automatic replay, hosted orchestration, cloud sync, or remote issue tracking. ADP keeps the source of truth local and terminal-readable.

## Concepts

Session history answers "what happened":

- `adp events list` reads local JSONL runtime events.
- `adp sessions list` groups those events into local agent sessions.
- `adp sessions show <session-id>` prints one session with its event timeline.

Restore plan answers "how could I start a similar new run":

- `adp sessions restore-plan <session-id>` inspects a historical session.
- It prints a suggested `adp run ...` command when enough invocation data is available.
- It may report a partial plan for older sessions or incomplete event data.
- It does not execute the suggested command.
- It does not create a runtime workspace, launch an agent, append new events, change task state, or write to the real project root.

Future replay would answer "start a new run for me":

- Replay is intentionally not part of this slice.
- A future replay command would still start a new local run, not resume a provider conversation.
- Any future execution command must remain explicit, locally gated, and separate from read-only inspection.

## Invocation Snapshot

Sessions created after this feature record a non-sensitive invocation snapshot on the `run_started` event under `fields.invocation`. The snapshot is designed to support restore planning without copying private runtime state.

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

The restore-plan command combines this snapshot with normal session summary fields used to form a safe suggestion: workspace, agent, profile, and task ID.

For sessions started with `adp run --take`, the snapshot can explain which ADP task was bound at launch, but it is not lease recovery by itself. Long-running owners renew with `adp tasks renew --workspace <workspace> <task-id> --owner <owner> --lease <duration>`, while interrupted sessions become visible through `adp tasks stale --workspace <workspace> [--format text|json]`.

## CLI Usage

Inspect session history first:

```bash
adp events list --workspace game-a --task <task-id>
adp sessions list --workspace game-a --agent codex --task <task-id>
adp sessions show <session-id>
```

Then ask ADP for a read-only restore plan:

```bash
adp sessions restore-plan <session-id>
```

The output is intended to be reviewed by a human or another local tool before any command is run. A ready plan can include a command like:

```bash
adp run codex --workspace game-a --profile senior-engineer --task task-20260608-0003 --keep-runtime -- --example-smoke
```

If data is missing, ADP should report missing fields and reasons instead of pretending the session is fully reconstructable. Old sessions created before invocation snapshots are expected to produce partial plans.

## Fake Or Local Agent Workflow

Use a fake local agent when validating restore-plan behavior without depending on an external provider CLI:

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
TASK_ID=$(adp tasks add --workspace game-a --priority high "Exercise restore-plan guidance" | sed -n 's/^task \(task-[^ ]*\) added$/\1/p')
test -n "$TASK_ID"
adp run codex --workspace game-a --task "$TASK_ID" -- --example-smoke
```

Review the local evidence:

```bash
adp events list --workspace game-a --task "$TASK_ID"
adp sessions list --workspace game-a --agent codex --task "$TASK_ID"
adp sessions show <session-id>
adp sessions restore-plan <session-id>
```

Running `restore-plan` should only print inspection output. The event count, task state, runtime directories, and real project root should not change because of the restore-plan command itself.

If the suggested command is run manually, it starts a new local agent run with a new session ID. It does not attach to the previous provider conversation.

## Lease-Aware Handoff

`sessions restore-plan` is one handoff clue, not the handoff ledger. It can reconstruct a safe launch shape from local invocation evidence, but it does not prove that the previous worker still owns the task, renew a lease, reclaim interrupted work, or know anything from provider-private state.

For task-bound sessions, pair restore planning with ADP planning inspection:

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

Plan-mode compatibility follows the same boundary. A provider-native plan panel can mirror a proposed restart or next-step checklist, but structured plan changes become durable only through `adp plan preview` and explicitly approved `adp plan apply`. Restore planning must not automatically edit files, complete tasks, accept phases, commit, push, run Git, apply plans, start runtimes, or write runtime/planning files into the real project root.

## Operating Rules

- Treat restore-plan output as guidance, not as an automatic repair or resume action.
- Bind work to tasks with `adp run <agent> --task <task-id>` when the session should be traceable in project planning.
- Prefer `adp run <agent> --take --owner <owner> --lease <duration>` when a worker should atomically take a board item at launch.
- Renew long-running ownership with `adp tasks renew`; use `adp tasks stale` to inspect expired in-progress claims after interruptions.
- Reclaim expired work only through ADP ownership commands such as `adp tasks take` or explicit `adp tasks claim`.
- Move task status explicitly with `adp tasks update`, `adp tasks done`, or `adp tasks block`.
- Treat provider-native plan and task panels as mirror or scratch surfaces, not as recovery evidence.
- Keep acceptance evidence local by pairing restore-plan checks with `adp events list`, `adp sessions list`, and `adp sessions show`.
- Do not describe restore-plan as cloud sync, remote issue tracking, hosted orchestration, provider-private state scraping, automatic task completion, or provider-native resume.
