#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd -- "$SCRIPT_DIR/.." && pwd)
. "$SCRIPT_DIR/task-manager-smoke-lib.sh"

fail() {
  printf 'plan-intake-smoke: %s\n' "$*" >&2
  exit 1
}

info() {
  printf '[plan-intake-smoke] %s\n' "$*"
}

if ! command -v go >/dev/null 2>&1; then
  fail "Go is required to build cmd/adp"
fi

TMP_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/adp-plan-intake-smoke.XXXXXX")
ADP_BIN="$TMP_ROOT/adp"

cleanup() {
  rm -rf "$TMP_ROOT"
}
trap cleanup EXIT INT TERM

PROJECT_ROOT="$TMP_ROOT/project"
ADP_HOME="$TMP_ROOT/adp-home"
ADP_RUNTIME_DIR="$TMP_ROOT/runtime"
EVENTS_FILE="$ADP_HOME/logs/events.jsonl"
TASKS_FILE="$ADP_HOME/workspaces/game-a/planning/tasks.yaml"
PHASES_FILE="$ADP_HOME/workspaces/game-a/planning/phases.yaml"
PROGRESS_FILE="$ADP_HOME/workspaces/game-a/planning/progress.jsonl"
PLAN_FILE="$TMP_ROOT/plan.yaml"
PREVIEW_FILE="$TMP_ROOT/preview.yaml"
INVALID_FILE="$TMP_ROOT/invalid.yaml"
FRESH_INVALID_FILE="$TMP_ROOT/fresh-invalid.yaml"
FAULT_FILE="$TMP_ROOT/fault.yaml"

mkdir -p "$PROJECT_ROOT" "$ADP_HOME" "$ADP_RUNTIME_DIR"
printf 'module example.com/adp-plan-intake-smoke\n' > "$PROJECT_ROOT/go.mod"

export ADP_HOME
export ADP_RUNTIME_DIR

info "building temporary adp binary"
(cd "$REPO_ROOT" && go build -o "$ADP_BIN" ./cmd/adp)

info "initializing workspace"
output=$(run_adp "$REPO_ROOT" init)
assert_contains "$output" "initialized ADP home" "init output"
output=$(run_adp "$REPO_ROOT" workspace add game-a "$PROJECT_ROOT")
assert_contains "$output" 'workspace "game-a" added' "workspace add output"

cat > "$PLAN_FILE" <<'YAML'
version: 1
phases:
  - id: p14-plan-intake
    title: Local planning intake
    goal: Import structured plans into the local ledger.
tasks:
  - title: Define planning input schema
    description: Stable YAML/JSON input for cross-tool planning.
    priority: high
    phase: p14-plan-intake
  - title: Review planning intake
    priority: medium
    phase: p14-plan-intake
    status: review
YAML

cat > "$FRESH_INVALID_FILE" <<'YAML'
version: 1
tasks:
  - title: Missing phase task
    phase: missing-phase
YAML

info "checking invalid apply on a fresh workspace is side-effect-free"
runtime_dirs_before=$(runtime_dirs_state)
project_root_before=$(project_root_state)
git_before=$(git_state)
output=$(run_adp_expect_fail "$REPO_ROOT" plan apply --workspace game-a --file "$FRESH_INVALID_FILE")
assert_contains "$output" "phase not found" "fresh failed plan apply output"
if [ -e "$ADP_HOME/workspaces/game-a/planning" ]; then
  fail "fresh failed plan apply created planning directory"
fi
if [ -e "$EVENTS_FILE" ]; then
  fail "fresh failed plan apply created runtime event log"
fi
assert_text_unchanged "$runtime_dirs_before" "$(runtime_dirs_state)" "fresh failed plan apply" "runtime dirs"
assert_text_unchanged "$project_root_before" "$(project_root_state)" "fresh failed plan apply" "project root"
assert_text_unchanged "$git_before" "$(git_state)" "fresh failed plan apply" "Git state"
assert_project_root_clean

info "checking preview read-only behavior"
runtime_dirs_before=$(runtime_dirs_state)
project_root_before=$(project_root_state)
git_before=$(git_state)
output=$(run_adp "$REPO_ROOT" plan preview --workspace game-a --file "$PLAN_FILE")
assert_contains "$output" "workspace: game-a" "plan preview output"
assert_contains "$output" "mode: preview" "plan preview output"
assert_contains "$output" "p14-plan-intake" "plan preview output"
assert_contains "$output" "Define planning input schema" "plan preview output"
if [ -e "$ADP_HOME/workspaces/game-a/planning" ]; then
  fail "plan preview created planning directory"
fi
if [ -e "$EVENTS_FILE" ]; then
  fail "plan preview created runtime event log"
fi
assert_text_unchanged "$runtime_dirs_before" "$(runtime_dirs_state)" "plan preview" "runtime dirs"
assert_text_unchanged "$project_root_before" "$(project_root_state)" "plan preview" "project root"
assert_text_unchanged "$git_before" "$(git_state)" "plan preview" "Git state"
assert_project_root_clean

info "checking explicit apply behavior"
output=$(run_adp "$REPO_ROOT" plan apply --workspace game-a --file "$PLAN_FILE" --format json)
assert_json_field "$output" "workspace" "plan apply json output"
assert_json_field "$output" "mode" "plan apply json output"
assert_json_field "$output" "phases" "plan apply json output"
assert_json_field "$output" "tasks" "plan apply json output"
assert_contains "$output" "\"apply\"" "plan apply json output"
assert_contains "$output" "\"p14-plan-intake\"" "plan apply json output"
assert_contains "$output" "\"Define planning input schema\"" "plan apply json output"
assert_file "$TASKS_FILE"
assert_file "$PHASES_FILE"
assert_file "$PROGRESS_FILE"
assert_contains "$(cat "$TASKS_FILE")" "Define planning input schema" "tasks file"
assert_contains "$(cat "$TASKS_FILE")" "Review planning intake" "tasks file"
assert_contains "$(cat "$PHASES_FILE")" "p14-plan-intake" "phases file"
assert_contains "$(cat "$PROGRESS_FILE")" "phase_created" "progress file"
assert_contains "$(cat "$PROGRESS_FILE")" "task_created" "progress file"
if [ -e "$EVENTS_FILE" ]; then
  fail "plan apply created runtime event log"
fi
assert_text_unchanged "$runtime_dirs_before" "$(runtime_dirs_state)" "plan apply" "runtime dirs"
assert_text_unchanged "$project_root_before" "$(project_root_state)" "plan apply" "project root"
assert_text_unchanged "$git_before" "$(git_state)" "plan apply" "Git state"
assert_project_root_clean

output=$(run_adp "$REPO_ROOT" phase list --workspace game-a)
assert_contains "$output" "p14-plan-intake" "phase list after apply"
output=$(run_adp "$REPO_ROOT" tasks list --workspace game-a)
assert_contains "$output" "Define planning input schema" "tasks list after apply"
assert_contains "$output" "Review planning intake" "tasks list after apply"

info "checking preview json remains read-only after apply"
cat > "$PREVIEW_FILE" <<'YAML'
version: 1
phases:
  - id: p15-future
    title: Future local plan
tasks:
  - title: Future task
    phase: p15-future
    priority: low
YAML
tasks_before=$(cat "$TASKS_FILE")
phases_before=$(cat "$PHASES_FILE")
progress_before=$(cat "$PROGRESS_FILE")
output=$(run_adp "$REPO_ROOT" plan preview --workspace game-a --file "$PREVIEW_FILE" --format json)
assert_json_field "$output" "workspace" "plan preview json output"
assert_json_field "$output" "mode" "plan preview json output"
assert_json_field "$output" "source" "plan preview json output"
assert_contains "$output" "\"preview\"" "plan preview json output"
assert_contains "$output" "\"p15-future\"" "plan preview json output"
assert_planning_state_unchanged "$tasks_before" "$phases_before" "$progress_before" "plan preview json"
assert_text_unchanged "$runtime_dirs_before" "$(runtime_dirs_state)" "plan preview json" "runtime dirs"
assert_text_unchanged "$project_root_before" "$(project_root_state)" "plan preview json" "project root"
assert_text_unchanged "$git_before" "$(git_state)" "plan preview json" "Git state"
assert_project_root_clean

info "checking failed apply does not partially import"
cat > "$INVALID_FILE" <<'YAML'
version: 1
phases:
  - id: p-bad
    title: Bad import
tasks:
  - title: Missing phase task
    phase: missing-phase
YAML
output=$(run_adp_expect_fail "$REPO_ROOT" plan apply --workspace game-a --file "$INVALID_FILE")
assert_contains "$output" "phase not found" "failed plan apply output"
assert_planning_state_unchanged "$tasks_before" "$phases_before" "$progress_before" "failed plan apply"
if grep -F -q "p-bad" "$PHASES_FILE" || grep -F -q "Missing phase task" "$TASKS_FILE" || grep -F -q "Missing phase task" "$PROGRESS_FILE"; then
  fail "failed plan apply left partial state"
fi

cat > "$FAULT_FILE" <<'YAML'
version: 1
phases:
  - id: p-fault
    title: Fault import
tasks:
  - title: Fault task
    phase: p-fault
YAML
mkdir "$ADP_HOME/workspaces/game-a/planning/tasks.yaml.tmp"
output=$(run_adp_expect_fail "$REPO_ROOT" plan apply --workspace game-a --file "$FAULT_FILE")
assert_contains "$output" "write temporary planning file" "faulted plan apply output"
assert_planning_state_unchanged "$tasks_before" "$phases_before" "$progress_before" "faulted plan apply"
if grep -F -q "p-fault" "$PHASES_FILE" || grep -F -q "Fault task" "$TASKS_FILE" || grep -F -q "Fault task" "$PROGRESS_FILE"; then
  fail "faulted plan apply left partial state"
fi
if [ -e "$ADP_HOME/workspaces/game-a/planning/tasks.yaml.tmp" ]; then
  fail "faulted plan apply left staged task temp path"
fi

output=$(run_adp_expect_fail "$REPO_ROOT" plan apply --workspace game-a --file "$PLAN_FILE")
assert_contains "$output" "phase already exists" "duplicate plan apply output"
assert_planning_state_unchanged "$tasks_before" "$phases_before" "$progress_before" "duplicate plan apply"

info "plan intake smoke passed"
