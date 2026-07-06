#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd -- "$SCRIPT_DIR/.." && pwd)
. "$SCRIPT_DIR/task-manager-smoke-lib.sh"

fail() {
  printf 'planning-concurrency-smoke: %s\n' "$*" >&2
  exit 1
}

info() {
  printf '[planning-concurrency-smoke] %s\n' "$*"
}

if ! command -v go >/dev/null 2>&1; then
  fail "Go is required to build cmd/adp"
fi

TMP_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/adp-planning-concurrency-smoke.XXXXXX")
ADP_BIN="$TMP_ROOT/adp"

cleanup() {
  rm -rf "$TMP_ROOT"
}
trap cleanup EXIT INT TERM

PROJECT_ROOT="$TMP_ROOT/project"
ADP_HOME="$TMP_ROOT/adp-home"
ADP_RUNTIME_DIR="$TMP_ROOT/runtime"
TASKS_FILE="$ADP_HOME/workspaces/concurrency/planning/tasks.yaml"
PHASES_FILE="$ADP_HOME/workspaces/concurrency/planning/phases.yaml"
PROGRESS_FILE="$ADP_HOME/workspaces/concurrency/planning/progress.jsonl"

mkdir -p "$PROJECT_ROOT" "$ADP_HOME" "$ADP_RUNTIME_DIR"
printf 'module example.com/adp-planning-concurrency-smoke\n' > "$PROJECT_ROOT/go.mod"

export ADP_HOME
export ADP_RUNTIME_DIR

wait_all_success() {
  local label="$1"
  shift
  local pid
  local failed=0

  for pid in "$@"; do
    if ! wait "$pid"; then
      failed=1
    fi
  done
  if [ "$failed" -ne 0 ]; then
    find "$TMP_ROOT" -maxdepth 1 \( -name "$label-*.out" -o -name "$label-*.err" \) -print -exec sed -n '1,12p' {} \; >&2
    fail "$label workers failed"
  fi
}

task_count() {
  grep -c '^[[:space:]]*- id: task-' "$TASKS_FILE"
}

phase_count() {
  grep -c '^[[:space:]]*- id:' "$PHASES_FILE"
}

assert_no_duplicates() {
  local label="$1"
  local values="$2"
  local duplicates

  duplicates=$(printf '%s\n' "$values" | sed '/^[[:space:]]*$/d' | LC_ALL=C sort | uniq -d)
  if [ -n "$duplicates" ]; then
    printf '%s\n' "$duplicates" >&2
    fail "$label has duplicate values"
  fi
}

assert_progress_line_count_at_least() {
  local want_min="$1"
  local got

  assert_file "$PROGRESS_FILE"
  got=$(wc -l < "$PROGRESS_FILE" | tr -d '[:space:]')
  if [ "$got" -lt "$want_min" ]; then
    printf '%s\n' "progress log:" >&2
    cat "$PROGRESS_FILE" >&2
    fail "progress event count is $got, want at least $want_min"
  fi
}

info "building temporary adp binary"
(cd "$REPO_ROOT" && go build -o "$ADP_BIN" ./cmd/adp)

info "initializing isolated workspace"
output=$(run_adp "$REPO_ROOT" init)
assert_contains "$output" "initialized ADP home" "init output"
output=$(run_adp "$REPO_ROOT" workspace add concurrency "$PROJECT_ROOT")
assert_contains "$output" 'workspace "concurrency" added' "workspace add output"
output=$(run_adp "$REPO_ROOT" phase add --workspace concurrency p-base "Base phase")
assert_contains "$output" "phase p-base added" "base phase add output"
output=$(run_adp "$REPO_ROOT" phase start --workspace concurrency p-base)
assert_contains "$output" "phase p-base status: active" "base phase start output"

info "checking concurrent tasks add against an existing phase"
pids=()
for i in $(seq 1 20); do
  (
    cd "$REPO_ROOT"
    "$ADP_BIN" tasks add --workspace concurrency --phase p-base --priority high "Concurrent task $i"
  ) >"$TMP_ROOT/task-add-$i.out" 2>"$TMP_ROOT/task-add-$i.err" &
  pids+=("$!")
done
wait_all_success "task-add" "${pids[@]}"
if [ "$(task_count)" -ne 20 ]; then
  cat "$TASKS_FILE" >&2
  fail "task count after concurrent add is $(task_count), want 20"
fi
task_ids=$(sed -n 's/^[[:space:]]*- id: //p' "$TASKS_FILE")
assert_no_duplicates "task ids after concurrent add" "$task_ids"

info "checking concurrent phase add order allocation"
pids=()
for i in $(seq 1 12); do
  phase_id=$(printf 'p-concurrent-%02d' "$i")
  (
    cd "$REPO_ROOT"
    "$ADP_BIN" phase add --workspace concurrency "$phase_id" "Concurrent phase $i"
  ) >"$TMP_ROOT/phase-add-$i.out" 2>"$TMP_ROOT/phase-add-$i.err" &
  pids+=("$!")
done
wait_all_success "phase-add" "${pids[@]}"
if [ "$(phase_count)" -ne 13 ]; then
  cat "$PHASES_FILE" >&2
  fail "phase count after concurrent add is $(phase_count), want 13"
fi
phase_ids=$(sed -n 's/^[[:space:]]*- id: //p' "$PHASES_FILE")
phase_orders=$(sed -n 's/^[[:space:]]*order: //p' "$PHASES_FILE")
assert_no_duplicates "phase ids after concurrent add" "$phase_ids"
assert_no_duplicates "phase orders after concurrent add" "$phase_orders"

info "checking duplicate phase add has a single winner"
pids=()
for i in $(seq 1 8); do
  (
    set +e
    cd "$REPO_ROOT"
    "$ADP_BIN" phase add --workspace concurrency p-duplicate "Duplicate phase" >"$TMP_ROOT/duplicate-phase-$i.out" 2>"$TMP_ROOT/duplicate-phase-$i.err"
    printf '%s\n' "$?" >"$TMP_ROOT/duplicate-phase-$i.status"
  ) &
  pids+=("$!")
done
wait_all_success "duplicate-phase-launch" "${pids[@]}"
duplicate_successes=0
duplicate_failures=0
for i in $(seq 1 8); do
  status=$(cat "$TMP_ROOT/duplicate-phase-$i.status")
  if [ "$status" -eq 0 ]; then
    duplicate_successes=$((duplicate_successes + 1))
    continue
  fi
  duplicate_failures=$((duplicate_failures + 1))
  assert_contains "$(cat "$TMP_ROOT/duplicate-phase-$i.err")" "phase already exists" "duplicate phase error"
done
if [ "$duplicate_successes" -ne 1 ] || [ "$duplicate_failures" -ne 7 ]; then
  fail "duplicate phase successes=$duplicate_successes failures=$duplicate_failures, want 1/7"
fi

info "checking concurrent plan apply batches"
pids=()
for i in $(seq 1 8); do
  plan="$TMP_ROOT/plan-$i.yaml"
  phase_id=$(printf 'p-plan-%02d' "$i")
  task_title=$(printf 'Plan task %02d' "$i")
  cat > "$plan" <<YAML
version: 1
phases:
  - id: $phase_id
    title: Plan phase $i
tasks:
  - title: $task_title
    priority: medium
    phase: $phase_id
YAML
  (
    cd "$REPO_ROOT"
    "$ADP_BIN" plan apply --workspace concurrency --file "$plan" --format json
  ) >"$TMP_ROOT/plan-apply-$i.out" 2>"$TMP_ROOT/plan-apply-$i.err" &
  pids+=("$!")
done
wait_all_success "plan-apply" "${pids[@]}"
if [ "$(task_count)" -ne 28 ]; then
  cat "$TASKS_FILE" >&2
  fail "task count after concurrent plan apply is $(task_count), want 28"
fi
if [ "$(phase_count)" -ne 22 ]; then
  cat "$PHASES_FILE" >&2
  fail "phase count after concurrent plan apply is $(phase_count), want 22"
fi
task_ids=$(sed -n 's/^[[:space:]]*- id: //p' "$TASKS_FILE")
phase_ids=$(sed -n 's/^[[:space:]]*- id: //p' "$PHASES_FILE")
phase_orders=$(sed -n 's/^[[:space:]]*order: //p' "$PHASES_FILE")
assert_no_duplicates "task ids after concurrent plan apply" "$task_ids"
assert_no_duplicates "phase ids after concurrent plan apply" "$phase_ids"
assert_no_duplicates "phase orders after concurrent plan apply" "$phase_orders"

info "checking plan doctor and progress evidence"
output=$(run_adp "$REPO_ROOT" plan doctor --workspace concurrency --format json)
assert_contains "$output" '"status": "ok"' "plan doctor json output"
assert_contains "$output" '"has_errors": false' "plan doctor json output"
assert_progress_line_count_at_least 42
assert_project_root_clean

printf '[planning-concurrency-smoke] planning concurrency smoke passed\n'
