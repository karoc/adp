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
TASK_ID=$(adp tasks add --workspace game-a --priority high --phase p4-session-restore "Exercise restore-plan guidance" | sed -n 's/^task \(task-[^ ]*\) added$/\1/p')
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

## Operating Rules

- Treat restore-plan output as guidance, not as an automatic repair or resume action.
- Bind work to tasks with `adp run <agent> --task <task-id>` when the session should be traceable in project planning.
- Move task status explicitly with `adp tasks update`, `adp tasks done`, or `adp tasks block`.
- Keep acceptance evidence local by pairing restore-plan checks with `adp events list`, `adp sessions list`, and `adp sessions show`.
- Do not describe restore-plan as cloud sync, remote issue tracking, hosted orchestration, or provider-native resume.
