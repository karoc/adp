#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

run_step() {
  printf '\n==> %s\n' "$*"
  "$@"
}

run_step scripts/runtime-smoke.sh --fake
run_step scripts/runtime-audit-smoke.sh
run_step scripts/release-readiness-smoke.sh
run_step scripts/release-rehearsal-smoke.sh
run_step scripts/release-artifact-smoke.sh
run_step scripts/example-workspace-smoke.sh
run_step scripts/task-manager-smoke.sh
run_step scripts/plan-intake-smoke.sh
run_step go test -count=1 ./...
run_step go vet ./...
run_step scripts/check-file-lines.sh
run_step scripts/check-docs-bilingual.sh
run_step git diff --check

printf '\ncheck-all passed\n'
