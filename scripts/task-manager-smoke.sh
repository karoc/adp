#!/usr/bin/env bash
set -euo pipefail

fail() {
  printf 'task-manager-smoke: %s\n' "$*" >&2
  exit 1
}

info() {
  printf '[task-manager-smoke] %s\n' "$*"
}

assert_contains() {
  local output="$1"
  local needle="$2"
  local label="$3"

  case "$output" in
    *"$needle"*) ;;
    *)
      printf '%s\n' "$output" >&2
      fail "$label missing expected text: $needle"
      ;;
  esac
}

assert_file() {
  local path="$1"
  if [ ! -f "$path" ]; then
    fail "missing file: $path"
  fi
}

run_adp() {
  local dir="$1"
  shift
  local output

  if ! output=$(cd "$dir" && "$ADP_BIN" "$@" 2>&1); then
    printf '%s\n' "$output" >&2
    fail "adp $* failed"
  fi
  printf '%s\n' "$output"
}

if ! command -v go >/dev/null 2>&1; then
  fail "Go is required to build cmd/adp"
fi

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd -- "$SCRIPT_DIR/.." && pwd)
TMP_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/adp-task-manager-smoke.XXXXXX")
ADP_BIN="$TMP_ROOT/adp"

cleanup() {
  rm -rf "$TMP_ROOT"
}
trap cleanup EXIT INT TERM

PROJECT_ROOT="$TMP_ROOT/project"
ADP_HOME="$TMP_ROOT/adp-home"
ADP_RUNTIME_DIR="$TMP_ROOT/runtime"
TASKS_FILE="$ADP_HOME/workspaces/game-a/planning/tasks.yaml"
PHASES_FILE="$ADP_HOME/workspaces/game-a/planning/phases.yaml"
PROGRESS_FILE="$ADP_HOME/workspaces/game-a/planning/progress.jsonl"

mkdir -p "$PROJECT_ROOT" "$ADP_HOME" "$ADP_RUNTIME_DIR"
printf 'module example.com/adp-task-smoke\n' > "$PROJECT_ROOT/go.mod"

export ADP_HOME
export ADP_RUNTIME_DIR

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

info "claiming and releasing task"
output=$(run_adp "$REPO_ROOT" tasks claim --workspace game-a "$task_id" --owner smoke-agent)
assert_contains "$output" "task $task_id claimed by smoke-agent" "tasks claim output"

output=$(run_adp "$REPO_ROOT" tasks show --workspace game-a "$task_id")
assert_contains "$output" "owner: smoke-agent" "tasks show claimed output"
assert_contains "$output" "status: in_progress" "tasks show claimed output"

output=$(run_adp "$REPO_ROOT" tasks release --workspace game-a "$task_id")
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

info "checking progress summary"
output=$(run_adp "$REPO_ROOT" progress --workspace game-a)
assert_contains "$output" "workspace: game-a" "progress output"
assert_contains "$output" "p3" "progress output"
assert_contains "$output" "total: 1" "progress output"
assert_contains "$output" "done" "progress output"

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

if [ -e "$PROJECT_ROOT/planning" ] || [ -e "$PROJECT_ROOT/tasks.yaml" ] || [ -e "$PROJECT_ROOT/phases.yaml" ] || [ -e "$PROJECT_ROOT/progress.jsonl" ]; then
  fail "project root was polluted with planning files"
fi

info "task manager smoke passed"
