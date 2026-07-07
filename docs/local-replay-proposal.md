# Local Replay Proposal

Simplified Chinese: [local-replay-proposal.zh-CN.md](local-replay-proposal.zh-CN.md)

Status: P77 proposal only. No local replay command is implemented by this document.

Current baseline: ADP has `adp sessions list`, `adp sessions show`, `adp sessions restore-plan`, and `adp sessions resume-plan`. There is no `adp sessions replay` command in the current tree.

## Recommendation

ADP should consider a future explicit local replay command, but only as a narrow execution companion to `adp sessions resume-plan`. The value is real: operators already use ADP session evidence to copy a suggested `adp run ...` command, check task ownership, and launch a new worker. A first-class local replay command could reduce copy/paste errors and make the new run traceable to the previous ADP session.

The feature should not be implemented as automatic provider resume. It should start a new ADP runtime and a new provider process from ADP-owned local evidence. The existing `resume-plan` command must remain read-only.

Recommended direction:

- Keep `adp sessions resume-plan` as the inspection and proposal command.
- Add a separate execution command only in a later accepted phase, likely under `adp sessions replay`.
- Make task ownership behavior explicit. The command must not silently claim, renew, release, complete, or block tasks.
- Keep the first MVP free of task mutations. Require the operator to run `adp tasks claim` or `adp tasks renew` explicitly before replay when ownership changes are needed.
- Treat provider-native conversation state as out of scope.
- Treat redacted or incomplete invocation data as a replay blocker, not as something to guess.

## Problem

`resume-plan` currently answers "what should I run next?" It is intentionally read-only and can include suggested commands with side-effect labels such as `inspect`, `task_mutation`, and `runtime_creation`.

The remaining friction is the handoff from plan review to execution:

- The operator must copy a command from text or JSON output.
- The operator must separately decide whether a task claim, renew, or stale reclaim is required.
- A new run is not explicitly linked to the source session except by human notes.
- Same-tool reruns can reuse safe invocation shape, but cross-tool reruns intentionally omit provider-specific profile and agent arguments.

An explicit local replay command could make that transition less error-prone, while preserving the existing local-first and explicit-mutation contract.

## Existing Contract

The current implementation already has the required foundation:

- `adp run` records a `run_started` event before launching the adapter and a `run_finished` event afterward.
- `run_started` includes a non-sensitive `fields.invocation` snapshot with schema version, redacted agent args, `keep_runtime`, workspace resolution, profile source, task binding source, and task snapshot when present.
- `adp sessions restore-plan` reconstructs a read-only rerun command from one session.
- `adp sessions resume-plan` combines session evidence with current task, lease, phase, owner, and target-agent context.
- `resume-plan` marks suggested commands with `side_effect` so callers can distinguish inspection, task mutation, and runtime creation.

Those pieces are enough to build replay as "plan, validate, then execute a new local run." They are not enough to resume provider-private conversations, recover unrecorded environment variables, replay hidden shell state, or reconstruct secrets that were intentionally redacted.

## Proposed MVP

Candidate command shape:

```bash
adp sessions replay <session-id> \
  --dry-run \
  [--workspace <name>] \
  [--agent <agent>] \
  --owner <owner> \
  [--lease <duration>] \
  [--format text|json]

adp sessions replay <session-id> \
  --execute \
  [--workspace <name>] \
  [--agent <agent>] \
  --owner <owner> \
  [--lease <duration>] \
  [--format text|json]
```

This is a candidate API, not an accepted implementation contract. A bare `adp sessions replay <session-id>` should not execute by default. A later implementation phase should confirm final flag names through command metadata, help examples, completion, tests, and smoke coverage.

MVP behavior:

- Build the same internal plan used by `sessions resume-plan`.
- Refuse to execute unless the plan is `ready`.
- Refuse to execute if the plan has no `runtime_creation` command.
- Refuse to execute if invocation data contains redaction placeholders or missing launch fields.
- Refuse workspace-only replay by default unless the operator explicitly allows it.
- Require `--owner` whenever the source session is task-bound.
- Launch task-bound replay only when the current task is already owned by `--owner` and the lease is not stale.
- If the task is unowned, stale, owned by another owner, closed, or blocked, stop with the required explicit ADP task command instead of mutating task state.
- Do not claim, renew, release, complete, block, or update tasks in the MVP.
- Start a new local ADP runtime through the same path as `adp run`.
- Produce normal `run_started` and `run_finished` events for the new session.
- Add replay source metadata to the new run evidence, such as `replay_source_session_id`, without storing provider-native state.

Dry-run behavior:

- `--dry-run` should be read-only.
- It should print the exact task preflight decision and launch command that would run.
- JSON dry-run output should include `read_only: true`, `would_mutate_task: false`, `would_create_runtime`, and the source session ID.
- It must not append events, create runtimes, mutate tasks or phases, run Git, or write to the project root.

Possible later extension:

- A future post-MVP design may consider explicit `--renew` or `--claim` replay flags.
- Such flags should be reviewed as a separate phase because they combine task mutation with runtime creation.
- Dry-run JSON would need to classify those steps separately from runtime creation, using the same `task_mutation` and `runtime_creation` side-effect vocabulary as `resume-plan`.

## Non-MVP

The first replay implementation should not include:

- Provider-native Codex or Claude conversation resume.
- Provider-private session handle transfer.
- Provider transcript scraping or replay.
- Automatic use of native task or plan panels as recovery evidence.
- Automatic task completion, phase acceptance, commit evidence, push evidence, or Git execution.
- Task claim, renew, release, complete, block, or update shortcuts in the first MVP.
- Automatic stale reclaim.
- Cross-workspace replay that copies provider-specific profile or agent arguments.
- Replaying full environment variables, shell history, generated adapter instructions, project file contents, or secrets.
- Batch replay, daemon replay, scheduled replay, or hosted orchestration.
- Web UI, dashboard, SaaS tracker, cloud sync, or remote issue-service integration.

## Safety Rules

Local replay must preserve these invariants:

- The authoritative task and phase ledger stays under `$ADP_HOME`.
- Runtime artifacts stay under `$ADP_RUNTIME_DIR`.
- Real project roots stay clean except for work the launched agent intentionally performs.
- `resume-plan` stays read-only forever.
- Replay is an execution command and must be documented as such.
- If a future replay extension permits task mutation, every mutation must be explicit in flags and output.
- No phase mutation is allowed.
- No Git command is allowed.
- No provider-native resume is implied.
- Redaction is a hard boundary. If ADP recorded `***REDACTED***`, replay must stop and ask the operator to run an explicit `adp run ...` command with replacement values.
- Partial session data should produce a clear error and a `resume-plan` suggestion, not a best-effort launch.

## Suggested Output

Text output should be short and auditable:

```text
source_session: session-20260707-0001
status: ready
mode: dry_run
task_preflight: task is owned by reviewer and lease is valid
runtime: will create a new ADP runtime
provider_native_resume: false
git_side_effects: false
project_root_writes_by_adp: false
launch: adp run codex --workspace game-a --task task-20260707-0003 -- --example-smoke
```

JSON output should mirror the `resume-plan` structure where possible and add execution-specific fields:

- `source_session_id`
- `plan_status`
- `mode`
- `task_preflight`
- `executed_commands`
- `new_session_id`, after launch starts
- `side_effects`
- `guarantees`

## Validation For A Future Implementation

A future implementation phase should add focused unit and smoke coverage before it can be accepted:

- Parser tests for `sessions replay` flags, invalid combinations, and required `--owner` behavior.
- Resume planner or replay preflight tests for ready, partial, blocked, stale, unowned, same-owner, different-owner, and closed-task cases.
- Tests proving `--dry-run` is read-only: no task mutation, no phase mutation, no event append, no runtime creation, no project-root write, and no Git side effects.
- Tests proving MVP replay refuses to mutate task ownership and points operators to explicit `adp tasks claim` or `adp tasks renew` commands instead.
- Tests proving default replay refuses redacted agent args and incomplete invocation snapshots.
- Tests proving replay creates a new session rather than attaching to the old provider conversation.
- Runtime smoke coverage with fake Codex and fake Claude.
- Runtime audit smoke coverage for help text, JSON output, side-effect fields, and read-only dry-run behavior.
- Bilingual docs and command metadata examples.
- Full `scripts/check-all.sh` before phase acceptance.

## Open Questions

- Should the command be named `adp sessions replay` or should ADP expose a `run --from-session <session-id>` form?
- Should the first implementation require an interactive confirmation unless `--yes` is passed?
- Should workspace-only replay be allowed in MVP or deferred until task-bound replay is proven?
- Should replay source metadata be a field on `run_started` or a dedicated `replay_started` event?
- Should a future command allow operator-supplied replacement arguments for redacted values, or should operators always run `adp run` manually in that case?
- Should post-MVP replay add explicit task-mutation shortcuts such as `--renew` or `--claim`, or should ownership always stay outside replay?
