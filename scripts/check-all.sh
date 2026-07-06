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

run_timed_step() {
  local label="$1"
  local had_errexit
  local started
  local status
  shift

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
  fi
  return "$status"
}

run_step() {
  run_timed_step "$*" "$@"
}

run_serial_smokes() {
  local started
  local smoke_failed
  local entry

  started=$SECONDS
  smoke_failed=0
  printf '\n==> smoke suite (%d scripts serial)\n' "${#SMOKE_SCRIPTS[@]}"
  for entry in "${SMOKE_SCRIPTS[@]}"; do
    # shellcheck disable=SC2086 # entries are trusted static command strings
    if ! run_step $entry; then
      smoke_failed=1
      break
    fi
  done
  if [ "$smoke_failed" -ne 0 ]; then
    printf '\nsmoke suite FAILED (elapsed %s)\n' "$(format_elapsed "$((SECONDS - started))")" >&2
    exit 1
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

  started=$SECONDS
  printf '\n==> smoke suite (%d scripts parallel)\n' "${#SMOKE_SCRIPTS[@]}"
  check_all_smoke_log_dir="$(mktemp -d "${TMPDIR:-/tmp}/adp-check-all.XXXXXX")"
  trap 'rm -rf "${check_all_smoke_log_dir:-}"' EXIT

  pids=()
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
    fi
    printf '\n==> [%s elapsed %s] %s\n' "$status_label" "$(format_elapsed "$worker_elapsed")" "${SMOKE_SCRIPTS[$idx]}"
    cat "$check_all_smoke_log_dir/$idx.log"
  done

  if [ "$smoke_failed" -ne 0 ]; then
    printf '\nsmoke suite FAILED (elapsed %s)\n' "$(format_elapsed "$((SECONDS - started))")" >&2
    exit 1
  fi
  printf '\n==> smoke suite passed (elapsed %s)\n' "$(format_elapsed "$((SECONDS - started))")"
}

check_all_started=$SECONDS

# Smoke scripts each build ./cmd/adp on first use. Pre-warm the shared Go build
# cache once up front so the workers below hit a warm cache (~0.2s each) instead
# of racing to pay the ~3.5s cold compile independently. go test below also
# benefits because cmd/adp's dependencies are already compiled.
run_timed_step "warming build cache (go build ./cmd/adp)" go build -o /dev/null ./cmd/adp

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
)

if [ "${CHECK_ALL_SERIAL:-0}" = "1" ]; then
  run_serial_smokes
else
  run_parallel_smokes
fi

# Runs the full test suite with coverage instrumentation and enforces the
# regression floor, so tests execute once here rather than twice (plain + cover).
run_step scripts/check-coverage.sh
run_step go vet ./...
run_step scripts/check-file-lines.sh
run_step scripts/check-docs-bilingual.sh
run_step git diff --check

printf '\ncheck-all passed (elapsed %s)\n' "$(format_elapsed "$((SECONDS - check_all_started))")"
