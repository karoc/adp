#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

run_step() {
  printf '\n==> %s\n' "$*"
  "$@"
}

# Smoke scripts each build ./cmd/adp on first use. Pre-warm the shared Go build
# cache once up front so the workers below hit a warm cache (~0.2s each) instead
# of racing to pay the ~3.5s cold compile independently. go test below also
# benefits because cmd/adp's dependencies are already compiled.
printf '\n==> warming build cache (go build ./cmd/adp)\n'
go build -o /dev/null ./cmd/adp

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
  for entry in "${SMOKE_SCRIPTS[@]}"; do
    # shellcheck disable=SC2086 # entries are trusted static command strings
    run_step $entry
  done
else
  printf '\n==> running %d smoke scripts in parallel\n' "${#SMOKE_SCRIPTS[@]}"
  smoke_log_dir="$(mktemp -d "${TMPDIR:-/tmp}/adp-check-all.XXXXXX")"
  trap 'rm -rf "$smoke_log_dir"' EXIT

  pids=()
  for idx in "${!SMOKE_SCRIPTS[@]}"; do
    # shellcheck disable=SC2086 # entries are trusted static command strings
    ( ${SMOKE_SCRIPTS[$idx]} ) >"$smoke_log_dir/$idx.log" 2>&1 &
    pids+=("$!")
  done

  smoke_failed=0
  for idx in "${!pids[@]}"; do
    if wait "${pids[$idx]}"; then
      status="ok"
    else
      status="FAILED"
      smoke_failed=1
    fi
    printf '\n==> [%s] %s\n' "$status" "${SMOKE_SCRIPTS[$idx]}"
    cat "$smoke_log_dir/$idx.log"
  done

  if [ "$smoke_failed" -ne 0 ]; then
    printf '\nsmoke suite FAILED\n' >&2
    exit 1
  fi
fi

# Runs the full test suite with coverage instrumentation and enforces the
# regression floor, so tests execute once here rather than twice (plain + cover).
run_step scripts/check-coverage.sh
run_step go vet ./...
run_step scripts/check-file-lines.sh
run_step scripts/check-docs-bilingual.sh
run_step git diff --check

printf '\ncheck-all passed\n'
