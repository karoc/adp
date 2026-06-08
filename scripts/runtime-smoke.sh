#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage:
  scripts/runtime-smoke.sh [--fake] [--real-codex] [--real-claude]

Runs ADP runtime smoke acceptance from a temporary ADP home, runtime
directory, project root, and agent bin directory.

The fake smoke is the default path and is deterministic. Real external
CLI checks are opt-in and require both a flag and an environment gate:

  ADP_SMOKE_REAL_CODEX=1 scripts/runtime-smoke.sh --real-codex
  ADP_SMOKE_REAL_CLAUDE=1 scripts/runtime-smoke.sh --real-claude

Optional real CLI binary overrides:

  ADP_SMOKE_CODEX_BIN=/path/to/codex
  ADP_SMOKE_CLAUDE_BIN=/path/to/claude
USAGE
}

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd -- "$SCRIPT_DIR/.." && pwd)
. "$SCRIPT_DIR/runtime-smoke-lib.sh"
. "$SCRIPT_DIR/runtime-smoke-diagnostics.sh"
. "$SCRIPT_DIR/runtime-smoke-session.sh"
. "$SCRIPT_DIR/runtime-smoke-prune.sh"
. "$SCRIPT_DIR/runtime-smoke-fake.sh"

run_fake=1
run_real_codex=0
run_real_claude=0

while [ "$#" -gt 0 ]; do
  case "$1" in
    --fake)
      run_fake=1
      ;;
    --real-codex)
      run_real_codex=1
      ;;
    --real-claude)
      run_real_claude=1
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      usage >&2
      fail "unknown option: $1"
      ;;
  esac
  shift
done

if ! command -v go >/dev/null 2>&1; then
  fail "Go is required to build cmd/adp"
fi

TMP_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/adp-runtime-smoke.XXXXXX")
ADP_BIN="$TMP_ROOT/adp"

cleanup() {
  rm -rf "$TMP_ROOT"
}
trap cleanup EXIT INT TERM

info "building temporary adp binary"
(cd "$REPO_ROOT" && go build -o "$ADP_BIN" ./cmd/adp)

if [ "$run_fake" -eq 1 ]; then
  run_fake_smoke
fi

if [ "$run_real_codex" -eq 1 ]; then
  run_real_cli_smoke codex ADP_SMOKE_REAL_CODEX "${ADP_SMOKE_CODEX_BIN:-codex}"
fi

if [ "$run_real_claude" -eq 1 ]; then
  run_real_cli_smoke claude ADP_SMOKE_REAL_CLAUDE "${ADP_SMOKE_CLAUDE_BIN:-claude}"
fi

info "runtime smoke acceptance passed"
