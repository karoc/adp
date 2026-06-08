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

info "creating task"
output=$(run_adp "$REPO_ROOT" tasks add --workspace game-a --priority high --phase phase-1.5 --description "local task state" "Add task manager")
assert_contains "$output" "task task-" "tasks add output"
assert_contains "$output" "added" "tasks add output"
task_id=$(printf '%s\n' "$output" | sed -n 's/^task \(task-[^ ]*\) added$/\1/p')
if [ -z "$task_id" ]; then
  fail "could not parse task id from: $output"
fi

assert_file "$TASKS_FILE"
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
assert_contains "$output" "total: 1" "progress output"
assert_contains "$output" "done" "progress output"

assert_contains "$(cat "$TASKS_FILE")" "$task_id" "tasks file"
assert_contains "$(cat "$PROGRESS_FILE")" "task_created" "progress file"
assert_contains "$(cat "$PROGRESS_FILE")" "task_blocked" "progress file"

if [ -e "$PROJECT_ROOT/planning" ] || [ -e "$PROJECT_ROOT/tasks.yaml" ] || [ -e "$PROJECT_ROOT/progress.jsonl" ]; then
  fail "project root was polluted with planning files"
fi

info "task manager smoke passed"
