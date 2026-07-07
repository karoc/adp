#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

format_elapsed() {
  local seconds="$1"
  local minutes
  local remainder

  if [ "$seconds" -ge 60 ]; then
    minutes=$((seconds / 60))
    remainder=$((seconds % 60))
    printf '%dm%02ds' "$minutes" "$remainder"
  else
    printf '%ds' "$seconds"
  fi
}

print_failure_guidance() {
  local command="$1"

  printf 'triage: rerun failed command from repo root: %s\n' "$command" >&2
  printf 'triage: after fixing, rerun aggregate gate: scripts/check-all.sh\n' >&2
}

print_smoke_failure_guidance() {
  local log_dir="$1"
  local command
  shift

  printf 'failed smoke commands:\n' >&2
  for command in "$@"; do
    printf '  %s\n' "$command" >&2
  done
  printf 'triage: rerun the smallest failing smoke command directly from repo root.\n' >&2
  printf 'triage: for sequential aggregate output: CHECK_ALL_SERIAL=1 scripts/check-all.sh\n' >&2
  if [ -n "$log_dir" ]; then
    if [ "${CHECK_ALL_KEEP_LOGS:-0}" = "1" ]; then
      printf 'triage: per-smoke logs kept at %s\n' "$log_dir" >&2
    else
      printf 'triage: preserve per-smoke logs with CHECK_ALL_KEEP_LOGS=1 scripts/check-all.sh\n' >&2
    fi
  fi
  printf 'triage: after fixing, rerun aggregate gate: scripts/check-all.sh\n' >&2
}

record_check_all_failure() {
  CHECK_ALL_FAILURES+=("$1")
}

finish_check_all_failures() {
  local failure

  if [ "${#CHECK_ALL_FAILURES[@]}" -eq 0 ]; then
    return 0
  fi

  printf '\ncheck-all FAILED (elapsed %s)\n' "$(format_elapsed "$((SECONDS - check_all_started))")" >&2
  printf 'failed gate commands:\n' >&2
  for failure in "${CHECK_ALL_FAILURES[@]}"; do
    printf '  %s\n' "$failure" >&2
  done
  printf 'triage: rerun each failed command directly from repo root, then rerun scripts/check-all.sh\n' >&2
  exit 1
}

run_timed_step() {
  local label="$1"
  local command
  local had_errexit
  local started
  local status
  shift
  command="$*"

  had_errexit=0
  case $- in
    *e*) had_errexit=1 ;;
  esac
  started=$SECONDS
  printf '\n==> %s\n' "$label"
  set +e
  "$@"
  status=$?
  if [ "$had_errexit" -eq 1 ]; then
    set -e
  else
    set +e
  fi
  if [ "$status" -eq 0 ]; then
    printf '==> completed %s (elapsed %s)\n' "$label" "$(format_elapsed "$((SECONDS - started))")"
  else
    printf '==> FAILED %s (elapsed %s)\n' "$label" "$(format_elapsed "$((SECONDS - started))")" >&2
    print_failure_guidance "$command"
  fi
  return "$status"
}

run_step() {
  run_timed_step "$*" "$@"
}

run_required_step() {
  if run_step "$@"; then
    return 0
  fi
  record_check_all_failure "$*"
  if [ "${CHECK_ALL_KEEP_GOING:-0}" != "1" ]; then
    finish_check_all_failures
  fi
  return 0
}

run_named_required_step() {
  local label="$1"
  shift

  if run_timed_step "$label" "$@"; then
    return 0
  fi
  record_check_all_failure "$*"
  if [ "${CHECK_ALL_KEEP_GOING:-0}" != "1" ]; then
    finish_check_all_failures
  fi
  return 0
}

run_serial_smokes() {
  local started
  local smoke_failed
  local entry
  local -a failed_commands

  started=$SECONDS
  smoke_failed=0
  failed_commands=()
  printf '\n==> smoke suite (%d scripts serial)\n' "${#SMOKE_SCRIPTS[@]}"
  for entry in "${SMOKE_SCRIPTS[@]}"; do
    # shellcheck disable=SC2086 # entries are trusted static command strings
    if ! run_step $entry; then
      smoke_failed=1
      failed_commands+=("$entry")
      if [ "${CHECK_ALL_KEEP_GOING:-0}" != "1" ]; then
        break
      fi
    fi
  done
  if [ "$smoke_failed" -ne 0 ]; then
    printf '\nsmoke suite FAILED (elapsed %s)\n' "$(format_elapsed "$((SECONDS - started))")" >&2
    print_smoke_failure_guidance "" "${failed_commands[@]}"
    return 1
  fi
  printf '==> smoke suite passed (elapsed %s)\n' "$(format_elapsed "$((SECONDS - started))")"
}

run_parallel_smokes() {
  local started
  local smoke_failed
  local idx
  local wait_status
  local worker_status
  local worker_elapsed
  local status_label
  local -a pids
  local -a failed_commands
  local -a failed_indexes
  local -a worker_elapseds
  local -a worker_statuses

  started=$SECONDS
  printf '\n==> smoke suite (%d scripts parallel)\n' "${#SMOKE_SCRIPTS[@]}"
  check_all_smoke_log_dir="$(mktemp -d "${TMPDIR:-/tmp}/adp-check-all.XXXXXX")"
  if [ "${CHECK_ALL_KEEP_LOGS:-0}" = "1" ]; then
    printf '==> per-smoke logs: %s\n' "$check_all_smoke_log_dir"
  else
    trap 'rm -rf "${check_all_smoke_log_dir:-}"' EXIT
  fi

  pids=()
  failed_commands=()
  failed_indexes=()
  worker_elapseds=()
  worker_statuses=()
  for idx in "${!SMOKE_SCRIPTS[@]}"; do
    (
      SECONDS=0
      set +e
      # shellcheck disable=SC2086 # entries are trusted static command strings
      ${SMOKE_SCRIPTS[$idx]}
      worker_status=$?
      set -e
      printf '%s %s\n' "$worker_status" "$SECONDS" >"$check_all_smoke_log_dir/$idx.status"
      exit "$worker_status"
    ) >"$check_all_smoke_log_dir/$idx.log" 2>&1 &
    pids+=("$!")
  done

  smoke_failed=0
  for idx in "${!pids[@]}"; do
    if wait "${pids[$idx]}"; then
      wait_status=0
    else
      wait_status=$?
    fi
    if [ -f "$check_all_smoke_log_dir/$idx.status" ] &&
      read -r worker_status worker_elapsed <"$check_all_smoke_log_dir/$idx.status"; then
      :
    else
      worker_status="$wait_status"
      worker_elapsed=0
    fi
    if [ "$wait_status" -eq 0 ] && [ "$worker_status" -eq 0 ]; then
      status_label="ok"
    else
      status_label="FAILED"
      smoke_failed=1
      failed_commands+=("${SMOKE_SCRIPTS[$idx]}")
      failed_indexes+=("$idx")
    fi
    worker_statuses[$idx]="$status_label"
    worker_elapseds[$idx]="$worker_elapsed"
  done

  printf '\n==> smoke suite summary\n'
  for idx in "${!SMOKE_SCRIPTS[@]}"; do
    printf '==> [%s elapsed %s] %s\n' "${worker_statuses[$idx]}" "$(format_elapsed "${worker_elapseds[$idx]}")" "${SMOKE_SCRIPTS[$idx]}"
  done

  if [ "$smoke_failed" -ne 0 ]; then
    printf '\n==> failed smoke logs\n'
    for idx in "${failed_indexes[@]}"; do
      printf '\n==> [FAILED elapsed %s] %s\n' "$(format_elapsed "${worker_elapseds[$idx]}")" "${SMOKE_SCRIPTS[$idx]}"
      cat "$check_all_smoke_log_dir/$idx.log"
    done
    if [ "${CHECK_ALL_SHOW_PASSED_LOGS:-0}" = "1" ]; then
      printf '\n==> passed smoke logs\n'
      for idx in "${!SMOKE_SCRIPTS[@]}"; do
        if [ "${worker_statuses[$idx]}" = "ok" ]; then
          printf '\n==> [ok elapsed %s] %s\n' "$(format_elapsed "${worker_elapseds[$idx]}")" "${SMOKE_SCRIPTS[$idx]}"
          cat "$check_all_smoke_log_dir/$idx.log"
        fi
      done
    fi
    printf '\nsmoke suite FAILED (elapsed %s)\n' "$(format_elapsed "$((SECONDS - started))")" >&2
    print_smoke_failure_guidance "$check_all_smoke_log_dir" "${failed_commands[@]}"
    return 1
  fi

  printf '\n==> smoke logs\n'
  for idx in "${!SMOKE_SCRIPTS[@]}"; do
    printf '\n==> [ok elapsed %s] %s\n' "$(format_elapsed "${worker_elapseds[$idx]}")" "${SMOKE_SCRIPTS[$idx]}"
    cat "$check_all_smoke_log_dir/$idx.log"
  done
  printf '\n==> smoke suite passed (elapsed %s)\n' "$(format_elapsed "$((SECONDS - started))")"
}

CHECK_ALL_FAILURES=()
check_all_started=$SECONDS

# Smoke scripts each build ./cmd/adp on first use. Pre-warm the shared Go build
# cache once up front so the workers below hit a warm cache (~0.2s each) instead
# of racing to pay the ~3.5s cold compile independently. go test below also
# benefits because cmd/adp's dependencies are already compiled.
run_named_required_step "warming build cache (go build ./cmd/adp)" go build -o /dev/null ./cmd/adp

# The smoke scripts are mutually isolated: each creates its own mktemp TMP_ROOT
# and points ADP_HOME/ADP_RUNTIME_DIR inside it, so they share no writable state
# beyond the (concurrency-safe) Go build cache and the read-only source tree.
# That makes them safe to run in parallel. Set CHECK_ALL_SERIAL=1 to fall back
# to sequential execution when debugging an interleaved failure.
SMOKE_SCRIPTS=(
  "scripts/runtime-smoke.sh --fake"
  "scripts/runtime-audit-smoke.sh"
  "scripts/runtime-context-smoke.sh"
  "scripts/release-readiness-smoke.sh"
  "scripts/release-rehearsal-smoke.sh"
  "scripts/release-artifact-smoke.sh"
  "scripts/release-operator-drill-smoke.sh"
  "scripts/install-onboarding-smoke.sh"
  "scripts/example-workspace-smoke.sh"
  "scripts/task-manager-smoke.sh"
  "scripts/plan-intake-smoke.sh"
  "scripts/planning-concurrency-smoke.sh"
)

if [ "${CHECK_ALL_SERIAL:-0}" = "1" ]; then
  if ! run_serial_smokes; then
    record_check_all_failure "smoke suite"
    if [ "${CHECK_ALL_KEEP_GOING:-0}" != "1" ]; then
      finish_check_all_failures
    fi
  fi
else
  if ! run_parallel_smokes; then
    record_check_all_failure "smoke suite"
    if [ "${CHECK_ALL_KEEP_GOING:-0}" != "1" ]; then
      finish_check_all_failures
    fi
  fi
fi

# Runs the full test suite with coverage instrumentation and enforces the
# regression floor, so tests execute once here rather than twice (plain + cover).
run_required_step scripts/check-coverage.sh
run_required_step go vet ./...
run_required_step scripts/check-file-lines.sh
run_required_step scripts/check-docs-bilingual.sh
run_required_step git diff --check

finish_check_all_failures

printf '\ncheck-all passed (elapsed %s)\n' "$(format_elapsed "$((SECONDS - check_all_started))")"
