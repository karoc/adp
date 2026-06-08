#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd -- "$SCRIPT_DIR/.." && pwd)
. "$SCRIPT_DIR/task-manager-smoke-lib.sh"

if ! command -v go >/dev/null 2>&1; then
  fail "Go is required to build cmd/adp"
fi

TMP_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/adp-task-manager-smoke.XXXXXX")
ADP_BIN="$TMP_ROOT/adp"
JSON_REPORT_ASSERT="$SCRIPT_DIR/progress-report-json-assert.go"

cleanup() {
  rm -rf "$TMP_ROOT"
}
trap cleanup EXIT INT TERM

PROJECT_ROOT="$TMP_ROOT/project"
ADP_HOME="$TMP_ROOT/adp-home"
ADP_RUNTIME_DIR="$TMP_ROOT/runtime"
FAKE_BIN="$TMP_ROOT/bin"
EVENTS_FILE="$ADP_HOME/logs/events.jsonl"
TASKS_FILE="$ADP_HOME/workspaces/game-a/planning/tasks.yaml"
PHASES_FILE="$ADP_HOME/workspaces/game-a/planning/phases.yaml"
PROGRESS_FILE="$ADP_HOME/workspaces/game-a/planning/progress.jsonl"

mkdir -p "$PROJECT_ROOT" "$ADP_HOME" "$ADP_RUNTIME_DIR" "$FAKE_BIN"
printf 'module example.com/adp-task-smoke\n' > "$PROJECT_ROOT/go.mod"
write_fake_codex "$FAKE_BIN/codex"

export ADP_HOME
export ADP_RUNTIME_DIR
export PATH="$FAKE_BIN:$PATH"

info "building temporary adp binary"
(cd "$REPO_ROOT" && go build -o "$ADP_BIN" ./cmd/adp)

info "initializing workspace"
output=$(run_adp "$REPO_ROOT" init)
assert_contains "$output" "initialized ADP home" "init output"

output=$(run_adp "$REPO_ROOT" workspace add game-a "$PROJECT_ROOT")
assert_contains "$output" 'workspace "game-a" added' "workspace add output"

info "creating phase"
output=$(run_adp "$REPO_ROOT" phase add --workspace game-a --goal "phase gate smoke" p3 "Phase Gate MVP")
assert_contains "$output" "phase p3 added" "phase add output"

output=$(run_adp "$REPO_ROOT" phase list --workspace game-a)
assert_contains "$output" "p3" "phase list output"
assert_contains "$output" "planned" "phase list output"
assert_contains "$output" "Phase Gate MVP" "phase list output"

output=$(run_adp "$REPO_ROOT" phase show --workspace game-a p3)
assert_contains "$output" "id: p3" "phase show output"
assert_contains "$output" "status: planned" "phase show output"
assert_contains "$output" "goal: phase gate smoke" "phase show output"

output=$(run_adp "$REPO_ROOT" phase start --workspace game-a p3)
assert_contains "$output" "phase p3 status: active" "phase start output"

info "checking phase JSON output"
output=$(run_adp "$REPO_ROOT" phase list --workspace game-a --format json)
assert_json_field "$output" "id" "phase list json output"
assert_json_field "$output" "status" "phase list json output"
assert_contains "$output" "\"p3\"" "phase list json output"
assert_contains "$output" "\"active\"" "phase list json output"

output=$(run_adp "$REPO_ROOT" phase show --workspace game-a p3 --format json)
assert_json_field "$output" "id" "phase show json output"
assert_json_field "$output" "status" "phase show json output"
assert_contains "$output" "\"p3\"" "phase show json output"
assert_contains "$output" "\"active\"" "phase show json output"

info "checking phase lifecycle guards"
output=$(run_adp "$REPO_ROOT" phase add --workspace game-a --goal "future gated work" p4 "Future Phase")
assert_contains "$output" "phase p4 added" "phase add p4 output"

output=$(run_adp_expect_fail "$REPO_ROOT" phase start --workspace game-a p4)
assert_contains "$output" "phase p3 must be pushed before phase p4 can start" "phase start order guard output"

output=$(run_adp_expect_fail "$REPO_ROOT" phase commit --workspace game-a p4 --hash blocked)
assert_contains "$output" "must be accepted before commit evidence is recorded" "phase commit guard output"

output=$(run_adp_expect_fail "$REPO_ROOT" phase push --workspace game-a p4 --remote origin --branch main --result pushed)
assert_contains "$output" "must have commit evidence before push evidence is recorded" "phase push guard output"

info "creating task"
output=$(run_adp "$REPO_ROOT" tasks add --workspace game-a --priority high --phase p3 --description "local task state" "Add task manager")
assert_contains "$output" "task task-" "tasks add output"
assert_contains "$output" "added" "tasks add output"
task_id=$(printf '%s\n' "$output" | sed -n 's/^task \(task-[^ ]*\) added$/\1/p')
if [ -z "$task_id" ]; then
  fail "could not parse task id from: $output"
fi

assert_file "$TASKS_FILE"
assert_file "$PHASES_FILE"
assert_file "$PROGRESS_FILE"

info "checking phase gate status read-only output"
tasks_before=$(cat "$TASKS_FILE")
phases_before=$(cat "$PHASES_FILE")
progress_before=$(cat "$PROGRESS_FILE")
runtime_dirs_before=$(runtime_dirs_state)
project_root_before=$(project_root_state)
git_before=$(git_state)

output=$(run_adp "$REPO_ROOT" phase status --workspace game-a)
assert_contains "$output" "workspace: game-a" "phase status output"
assert_contains "$output" "phase_count: 2" "phase status output"
assert_contains "$output" "open_phase: p3 [active] Phase Gate MVP" "phase status output"
assert_contains "$output" "next_planned_phase: p4 [planned] Future Phase" "phase status output"
assert_contains "$output" "can_start_next: false" "phase status output"
assert_contains "$output" "next_action: record_acceptance" "phase status output"

output=$(run_adp "$REPO_ROOT" phase status --workspace game-a --format json)
assert_json_field "$output" "workspace" "phase status json output"
assert_json_field "$output" "phase_count" "phase status json output"
assert_json_field "$output" "open_phase" "phase status json output"
assert_json_field "$output" "next_planned_phase" "phase status json output"
assert_json_field "$output" "can_start_next" "phase status json output"
assert_json_field "$output" "next_action" "phase status json output"
assert_contains "$output" "\"p3\"" "phase status json output"
assert_contains "$output" "\"p4\"" "phase status json output"
assert_contains "$output" "\"can_start_next\": false" "phase status json output"
assert_contains "$output" "\"next_action\": \"record_acceptance\"" "phase status json output"

assert_planning_state_unchanged "$tasks_before" "$phases_before" "$progress_before" "phase status"
assert_text_unchanged "$runtime_dirs_before" "$(runtime_dirs_state)" "phase status" "runtime dirs"
assert_text_unchanged "$project_root_before" "$(project_root_state)" "phase status" "project root"
assert_text_unchanged "$git_before" "$(git_state)" "phase status" "Git state"
assert_project_root_clean

info "checking planning doctor read-only output"
tasks_before=$(cat "$TASKS_FILE")
phases_before=$(cat "$PHASES_FILE")
progress_before=$(cat "$PROGRESS_FILE")
runtime_dirs_before=$(runtime_dirs_state)
project_root_before=$(project_root_state)
git_before=$(git_state)

output=$(run_adp "$REPO_ROOT" plan doctor --workspace game-a)
assert_contains "$output" "workspace: game-a" "plan doctor output"
assert_contains "$output" "planning_dir: $ADP_HOME/workspaces/game-a/planning" "plan doctor output"
assert_contains "$output" "status: ok" "plan doctor output"
assert_contains "$output" "task_count: 1" "plan doctor output"
assert_contains "$output" "phase_count: 2" "plan doctor output"
assert_contains "$output" "progress_event_count:" "plan doctor output"
assert_contains "$output" "phase_gate_next_action: record_acceptance" "plan doctor output"
assert_contains "$output" "phase_gate_can_start_next: false" "plan doctor output"
assert_contains "$output" "diagnostics: -" "plan doctor output"

output=$(run_adp "$REPO_ROOT" plan doctor --workspace game-a --format json)
assert_json_field "$output" "workspace" "plan doctor json output"
assert_json_field "$output" "planning_dir" "plan doctor json output"
assert_json_field "$output" "status" "plan doctor json output"
assert_json_field "$output" "diagnostics" "plan doctor json output"
assert_json_field "$output" "phase_gate" "plan doctor json output"
assert_contains "$output" "\"status\": \"ok\"" "plan doctor json output"
assert_contains "$output" "\"task_count\": 1" "plan doctor json output"
assert_contains "$output" "\"phase_count\": 2" "plan doctor json output"
assert_contains "$output" "\"has_errors\": false" "plan doctor json output"
assert_contains "$output" "\"diagnostics\": []" "plan doctor json output"

assert_planning_state_unchanged "$tasks_before" "$phases_before" "$progress_before" "plan doctor"
assert_text_unchanged "$runtime_dirs_before" "$(runtime_dirs_state)" "plan doctor" "runtime dirs"
assert_text_unchanged "$project_root_before" "$(project_root_state)" "plan doctor" "project root"
assert_text_unchanged "$git_before" "$(git_state)" "plan doctor" "Git state"
assert_project_root_clean

info "checking planning doctor broken-ledger diagnostics"
output=$(run_adp "$REPO_ROOT" workspace add broken "$PROJECT_ROOT")
assert_contains "$output" 'workspace "broken" added' "workspace add broken output"
BROKEN_PLANNING_DIR="$ADP_HOME/workspaces/broken/planning"
BROKEN_TASKS_FILE="$BROKEN_PLANNING_DIR/tasks.yaml"
BROKEN_PHASES_FILE="$BROKEN_PLANNING_DIR/phases.yaml"
BROKEN_PROGRESS_FILE="$BROKEN_PLANNING_DIR/progress.jsonl"
mkdir -p "$BROKEN_PLANNING_DIR"
printf '%s\n' \
  'version: 1' \
  'tasks:' \
  '  - id: task-broken' \
  '    title: Broken task' \
  '    status: ready' \
  '    priority: critical' \
  '    phase: missing-phase' \
  '    created_at: 2026-06-08T12:00:00Z' \
  '    updated_at: 2026-06-08T12:00:00Z' \
  > "$BROKEN_TASKS_FILE"
printf '%s\n' \
  'version: 1' \
  'phases:' \
  '  - id: p-broken' \
  '    title: Broken phase' \
  '    status: planned' \
  '    order: 1' \
  '    created_at: 2026-06-08T12:00:00Z' \
  '    updated_at: 2026-06-08T12:00:00Z' \
  > "$BROKEN_PHASES_FILE"
printf '%s\n' '{"type":"task_created","task_id":"task-broken"}' > "$BROKEN_PROGRESS_FILE"

broken_tasks_before=$(cat "$BROKEN_TASKS_FILE")
broken_phases_before=$(cat "$BROKEN_PHASES_FILE")
broken_progress_before=$(cat "$BROKEN_PROGRESS_FILE")
runtime_dirs_before=$(runtime_dirs_state)
project_root_before=$(project_root_state)
git_before=$(git_state)

output=$(run_adp_expect_code 2 "$REPO_ROOT" plan doctor --workspace broken --format json)
assert_contains "$output" "\"workspace\": \"broken\"" "broken plan doctor json output"
assert_contains "$output" "\"status\": \"error\"" "broken plan doctor json output"
assert_contains "$output" "\"has_errors\": true" "broken plan doctor json output"
assert_contains "$output" "\"level\": \"error\"" "broken plan doctor json output"
assert_contains "$output" "planning.task.phase.unknown" "broken plan doctor json output"

assert_text_unchanged "$broken_tasks_before" "$(cat "$BROKEN_TASKS_FILE")" "broken plan doctor" "broken tasks"
assert_text_unchanged "$broken_phases_before" "$(cat "$BROKEN_PHASES_FILE")" "broken plan doctor" "broken phases"
assert_text_unchanged "$broken_progress_before" "$(cat "$BROKEN_PROGRESS_FILE")" "broken plan doctor" "broken progress"
assert_text_unchanged "$runtime_dirs_before" "$(runtime_dirs_state)" "broken plan doctor" "runtime dirs"
assert_text_unchanged "$project_root_before" "$(project_root_state)" "broken plan doctor" "project root"
assert_text_unchanged "$git_before" "$(git_state)" "broken plan doctor" "Git state"
assert_project_root_clean

info "inspecting task list and detail"
output=$(run_adp "$REPO_ROOT" tasks list --workspace game-a)
assert_contains "$output" "$task_id" "tasks list output"
assert_contains "$output" "ready" "tasks list output"
assert_contains "$output" "Add task manager" "tasks list output"

output=$(run_adp "$REPO_ROOT" tasks show --workspace game-a "$task_id")
assert_contains "$output" "id: $task_id" "tasks show output"
assert_contains "$output" "title: Add task manager" "tasks show output"
assert_contains "$output" "description: local task state" "tasks show output"
assert_contains "$output" "phase: p3" "tasks show output"

info "checking task JSON output"
output=$(run_adp "$REPO_ROOT" tasks list --workspace game-a --format json)
assert_json_field "$output" "id" "tasks list json output"
assert_json_field "$output" "status" "tasks list json output"
assert_contains "$output" "\"$task_id\"" "tasks list json output"
assert_contains "$output" "\"ready\"" "tasks list json output"

output=$(run_adp "$REPO_ROOT" tasks show --workspace game-a "$task_id" --format json)
assert_json_field "$output" "id" "tasks show json output"
assert_json_field "$output" "status" "tasks show json output"
assert_contains "$output" "\"$task_id\"" "tasks show json output"
assert_contains "$output" "\"ready\"" "tasks show json output"

output=$(run_adp_expect_fail "$REPO_ROOT" tasks add --workspace game-a --phase missing-phase "Invalid phase task")
assert_contains "$output" "phase not found" "tasks add phase guard output"

info "claiming and releasing task"
output=$(run_adp "$REPO_ROOT" tasks claim --workspace game-a "$task_id" --owner smoke-agent --lease 30m)
assert_contains "$output" "task $task_id claimed by smoke-agent" "tasks claim output"

output=$(run_adp "$REPO_ROOT" tasks show --workspace game-a "$task_id")
assert_contains "$output" "owner: smoke-agent" "tasks show claimed output"
assert_contains "$output" "status: in_progress" "tasks show claimed output"
assert_contains "$output" "lease_expires_at: 20" "tasks show claimed output"

output=$(run_adp "$REPO_ROOT" tasks release --workspace game-a "$task_id" --owner smoke-agent)
assert_contains "$output" "task $task_id released" "tasks release output"

output=$(run_adp "$REPO_ROOT" tasks show --workspace game-a "$task_id")
assert_contains "$output" "owner: -" "tasks show released output"
assert_contains "$output" "status: ready" "tasks show released output"

info "updating task state"
output=$(run_adp "$REPO_ROOT" tasks update --workspace game-a "$task_id" --status in_progress)
assert_contains "$output" "status: in_progress" "tasks update output"

output=$(run_adp "$REPO_ROOT" tasks block --workspace game-a "$task_id" --reason "waiting for review")
assert_contains "$output" "blocked" "tasks block output"

output=$(run_adp "$REPO_ROOT" tasks done --workspace game-a "$task_id")
assert_contains "$output" "done" "tasks done output"

info "creating runtime session evidence for progress report"
export ADP_EXPECT_TASK_ID="$task_id"
output=$(run_adp "$REPO_ROOT" run codex --workspace game-a --task "$task_id" -- --report-smoke)
assert_contains "$output" "fake-codex" "codex run output"
assert_contains "$output" "--report-smoke" "codex run output"
assert_file "$EVENTS_FILE"

codex_session=$(session_id_by_agent "$EVENTS_FILE" codex)
if [ -z "$codex_session" ]; then
  cat "$EVENTS_FILE" >&2
  fail "codex session id missing in event log"
fi

output=$(run_adp "$REPO_ROOT" events list --workspace game-a --session "$codex_session" --task "$task_id" --limit 2)
assert_contains "$output" "$codex_session" "events list session output"
assert_contains "$output" "$task_id" "events list session output"
assert_contains "$output" "codex" "events list session output"
assert_contains "$output" "run_started" "events list session output"
assert_contains "$output" "run_finished" "events list session output"

output=$(run_adp "$REPO_ROOT" sessions show "$codex_session")
assert_contains "$output" "session_id: $codex_session" "sessions show output"
assert_contains "$output" "agent: codex" "sessions show output"
assert_contains "$output" "task_id: $task_id" "sessions show output"
assert_contains "$output" "run_finished" "sessions show output"
assert_project_root_clean

info "checking progress summary"
output=$(run_adp "$REPO_ROOT" progress --workspace game-a)
assert_contains "$output" "workspace: game-a" "progress output"
assert_contains "$output" "p3" "progress output"
assert_contains "$output" "total: 1" "progress output"
assert_contains "$output" "done" "progress output"

info "checking progress JSON output"
output=$(run_adp "$REPO_ROOT" progress --workspace game-a --format json)
assert_json_field "$output" "workspace" "progress json output"
assert_json_field "$output" "total" "progress json output"
assert_json_field "$output" "counts" "progress json output"
assert_contains "$output" "\"game-a\"" "progress json output"
assert_contains "$output" "\"done\"" "progress json output"

info "checking progress report output"
tasks_before=$(cat "$TASKS_FILE")
phases_before=$(cat "$PHASES_FILE")
progress_before=$(cat "$PROGRESS_FILE")
events_before=$(line_count "$EVENTS_FILE")

output=$(run_adp "$REPO_ROOT" progress report --workspace game-a)
assert_starts_with "$output" "# ADP Progress Report" "progress report output"
assert_contains "$output" "Workspace: game-a" "progress report output"
assert_contains "$output" "p3" "progress report output"
assert_contains "$output" "$task_id" "progress report output"
assert_contains "$output" "done" "progress report output"
assert_contains "$output" "## Runtime Sessions" "progress report output"
assert_contains "$output" "$codex_session" "progress report output"
assert_contains "$output" "codex" "progress report output"
assert_contains "$output" "$ADP_RUNTIME_DIR" "progress report output"
assert_contains_any "$output" "progress report output" "run_finished" "exit_code" "Exit Code" "Exit code" "| Exit |"

output=$(run_adp "$REPO_ROOT" progress report --workspace game-a --language zh-CN)
assert_starts_with "$output" "# ADP 执行进度报告" "progress report zh-CN output"
assert_contains "$output" "工作区：game-a" "progress report zh-CN output"
assert_contains "$output" "## Runtime 会话" "progress report zh-CN output"
assert_contains "$output" "$codex_session" "progress report zh-CN output"
assert_contains "$output" "codex" "progress report zh-CN output"
assert_contains "$output" "$task_id" "progress report zh-CN output"
assert_contains "$output" "$ADP_RUNTIME_DIR" "progress report zh-CN output"

assert_planning_state_unchanged "$tasks_before" "$phases_before" "$progress_before" "progress report"
assert_event_log_line_count_unchanged "$events_before" "progress report"
assert_project_root_clean

assert_contains "$(cat "$TASKS_FILE")" "$task_id" "tasks file"
assert_contains "$(cat "$PHASES_FILE")" "p3" "phases file"
assert_contains "$(cat "$PROGRESS_FILE")" "task_created" "progress file"
assert_contains "$(cat "$PROGRESS_FILE")" "task_claimed" "progress file"
assert_contains "$(cat "$PROGRESS_FILE")" "task_released" "progress file"
assert_contains "$(cat "$PROGRESS_FILE")" "task_blocked" "progress file"

info "recording phase gate evidence"
output=$(run_adp "$REPO_ROOT" phase accept --workspace game-a p3 --command "scripts/task-manager-smoke.sh" --result passed --notes "deterministic smoke")
assert_contains "$output" "phase p3 accepted: passed" "phase accept output"

output=$(run_adp "$REPO_ROOT" phase commit --workspace game-a p3 --hash abc123 --message "Phase Gate MVP smoke")
assert_contains "$output" "phase p3 commit: abc123" "phase commit output"

output=$(run_adp "$REPO_ROOT" phase push --workspace game-a p3 --remote origin --branch main --result pushed)
assert_contains "$output" "phase p3 push: origin/main pushed" "phase push output"

output=$(run_adp "$REPO_ROOT" phase show --workspace game-a p3)
assert_contains "$output" "status: pushed" "phase show pushed output"
assert_contains "$output" "acceptance_result: passed" "phase show pushed output"
assert_contains "$output" "acceptance_commands: scripts/task-manager-smoke.sh" "phase show pushed output"
assert_contains "$output" "commit_hash: abc123" "phase show pushed output"
assert_contains "$output" "push_remote: origin" "phase show pushed output"
assert_contains "$output" "push_branch: main" "phase show pushed output"
assert_contains "$output" "push_result: pushed" "phase show pushed output"

assert_contains "$(cat "$PHASES_FILE")" "abc123" "phases file"
assert_contains "$(cat "$PROGRESS_FILE")" "phase_accepted" "progress file"
assert_contains "$(cat "$PROGRESS_FILE")" "phase_committed" "progress file"
assert_contains "$(cat "$PROGRESS_FILE")" "phase_pushed" "progress file"

output=$(run_adp "$REPO_ROOT" phase status --workspace game-a --format json)
assert_contains "$output" "\"can_start_next\": true" "phase status after push json output"
assert_contains "$output" "\"next_action\": \"start_next_phase\"" "phase status after push json output"
assert_contains "$output" "\"p4\"" "phase status after push json output"

output=$(run_adp "$REPO_ROOT" phase start --workspace game-a p4)
assert_contains "$output" "phase p4 status: active" "phase start p4 after p3 push output"

info "checking progress report phase evidence"
tasks_before=$(cat "$TASKS_FILE")
phases_before=$(cat "$PHASES_FILE")
progress_before=$(cat "$PROGRESS_FILE")
events_before=$(line_count "$EVENTS_FILE")

output=$(run_adp "$REPO_ROOT" progress report --workspace game-a)
assert_starts_with "$output" "# ADP Progress Report" "progress report evidence output"
assert_contains "$output" "p3" "progress report evidence output"
assert_contains "$output" "## Runtime Sessions" "progress report evidence output"
assert_contains "$output" "$codex_session" "progress report evidence output"
assert_contains "$output" "codex" "progress report evidence output"
assert_contains "$output" "$task_id" "progress report evidence output"
assert_contains "$output" "$ADP_RUNTIME_DIR" "progress report evidence output"
assert_contains_any "$output" "progress report evidence output" "run_finished" "exit_code" "Exit Code" "Exit code" "| Exit |"
assert_contains "$output" "abc123" "progress report evidence output"
assert_contains "$output" "origin" "progress report evidence output"
assert_contains "$output" "main" "progress report evidence output"
assert_contains "$output" "pushed" "progress report evidence output"
assert_contains "$output" "scripts/task-manager-smoke.sh" "progress report evidence output"

assert_planning_state_unchanged "$tasks_before" "$phases_before" "$progress_before" "progress report evidence"
assert_event_log_line_count_unchanged "$events_before" "progress report evidence"
assert_project_root_clean

info "creating next-work report candidates"
output=$(run_adp "$REPO_ROOT" tasks add --workspace game-a --priority low --phase p4 --description "lower priority handoff candidate" "Low priority follow-up")
assert_contains "$output" "task task-" "low priority task add output"
assert_contains "$output" "added" "low priority task add output"
low_task_id=$(printf '%s\n' "$output" | sed -n 's/^task \(task-[^ ]*\) added$/\1/p')
if [ -z "$low_task_id" ]; then
  fail "could not parse low priority task id from: $output"
fi

output=$(run_adp "$REPO_ROOT" tasks add --workspace game-a --priority critical --phase p4 --description "critical handoff candidate" "Critical handoff follow-up")
assert_contains "$output" "task task-" "critical priority task add output"
assert_contains "$output" "added" "critical priority task add output"
critical_task_id=$(printf '%s\n' "$output" | sed -n 's/^task \(task-[^ ]*\) added$/\1/p')
if [ -z "$critical_task_id" ]; then
  fail "could not parse critical priority task id from: $output"
fi

info "checking tasks next read-only output"
tasks_before=$(cat "$TASKS_FILE")
phases_before=$(cat "$PHASES_FILE")
progress_before=$(cat "$PROGRESS_FILE")
events_before=$(line_count "$EVENTS_FILE")
runtime_dirs_before=$(runtime_dirs_state)
project_root_before=$(project_root_state)
git_before=$(git_state)

output=$(run_adp "$REPO_ROOT" tasks next --workspace game-a --limit 1)
assert_contains "$output" "workspace: game-a" "tasks next output"
assert_contains "$output" "limit: 1" "tasks next output"
assert_contains "$output" "$critical_task_id" "tasks next output"
case "$output" in
  *"$low_task_id"*) fail "tasks next --limit 1 included low priority task" ;;
esac

output=$(run_adp "$REPO_ROOT" tasks next --workspace game-a --limit 0 --format json)
assert_json_field "$output" "workspace" "tasks next json output"
assert_json_field "$output" "planning_source" "tasks next json output"
assert_json_field "$output" "generated_at" "tasks next json output"
assert_json_field "$output" "total" "tasks next json output"
assert_json_field "$output" "eligible_count" "tasks next json output"
assert_json_field "$output" "counts" "tasks next json output"
assert_json_field "$output" "limit" "tasks next json output"
assert_json_field "$output" "candidates" "tasks next json output"
assert_json_field "$output" "next" "tasks next json output"
assert_contains "$output" "\"$critical_task_id\"" "tasks next json output"
assert_contains "$output" "\"$low_task_id\"" "tasks next json output"
assert_contains "$output" "\"planning_source\": \"$TASKS_FILE\"" "tasks next json output"
assert_contains "$output" "\"eligible_count\": 2" "tasks next json output"
critical_line=$(printf '%s\n' "$output" | grep -n "\"$critical_task_id\"" | head -n 1 | cut -d: -f1)
low_line=$(printf '%s\n' "$output" | grep -n "\"$low_task_id\"" | head -n 1 | cut -d: -f1)
if [ -z "$critical_line" ] || [ -z "$low_line" ] || [ "$critical_line" -ge "$low_line" ]; then
  printf '%s\n' "$output" >&2
  fail "tasks next json did not prioritize critical task before low task"
fi

assert_planning_state_unchanged "$tasks_before" "$phases_before" "$progress_before" "tasks next"
assert_event_log_line_count_unchanged "$events_before" "tasks next"
assert_text_unchanged "$runtime_dirs_before" "$(runtime_dirs_state)" "tasks next" "runtime dirs"
assert_text_unchanged "$project_root_before" "$(project_root_state)" "tasks next" "project root"
assert_text_unchanged "$git_before" "$(git_state)" "tasks next" "Git state"
assert_project_root_clean

info "checking progress report JSON handoff snapshot"
tasks_before=$(cat "$TASKS_FILE")
phases_before=$(cat "$PHASES_FILE")
progress_before=$(cat "$PROGRESS_FILE")
events_before=$(line_count "$EVENTS_FILE")
runtime_dirs_before=$(runtime_dirs_state)
project_root_before=$(project_root_state)
git_before=$(git_state)

output=$(run_adp "$REPO_ROOT" progress report --workspace game-a --format json)
assert_progress_report_json "$output" "$task_id" "$critical_task_id" "$low_task_id" "$codex_session" "progress report json output"

assert_planning_state_unchanged "$tasks_before" "$phases_before" "$progress_before" "progress report json"
assert_event_log_line_count_unchanged "$events_before" "progress report json"
assert_text_unchanged "$runtime_dirs_before" "$(runtime_dirs_state)" "progress report json" "runtime dirs"
assert_text_unchanged "$project_root_before" "$(project_root_state)" "progress report json" "project root"
assert_text_unchanged "$git_before" "$(git_state)" "progress report json" "Git state"
assert_project_root_clean

info "task manager smoke passed"
