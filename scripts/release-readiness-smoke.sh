#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd -- "$SCRIPT_DIR/.." && pwd)

fail() {
  printf 'release-readiness-smoke: %s\n' "$*" >&2
  exit 1
}

info() {
  printf '[release-readiness-smoke] %s\n' "$*"
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

. "$SCRIPT_DIR/smoke-git-tripwire-lib.sh"

if ! command -v go >/dev/null 2>&1; then
  fail "Go is required to build cmd/adp"
fi

TMP_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/adp-release-readiness-smoke.XXXXXX")
ADP_BIN="$TMP_ROOT/adp"
PROJECT_ROOT="$TMP_ROOT/project"
ADP_HOME="$TMP_ROOT/adp-home"
ADP_RUNTIME_DIR="$TMP_ROOT/runtime"
FAKE_BIN="$TMP_ROOT/bin"
GIT_TRIPWIRE_LOG="$TMP_ROOT/git-side-effects.log"

cleanup() {
  rm -rf "$TMP_ROOT"
}
trap cleanup EXIT INT TERM

mkdir -p "$PROJECT_ROOT" "$ADP_HOME" "$ADP_RUNTIME_DIR" "$FAKE_BIN"
printf 'module example.com/adp-release-readiness-smoke\n' > "$PROJECT_ROOT/go.mod"

info "building temporary adp binary"
(cd "$REPO_ROOT" && go build -o "$ADP_BIN" ./cmd/adp)
setup_git_tripwire "$FAKE_BIN" "$GIT_TRIPWIRE_LOG"

export ADP_HOME
export ADP_RUNTIME_DIR
export PATH="$FAKE_BIN:$PATH"

info "checking phase evidence commands do not execute git"
output=$(run_adp "$REPO_ROOT" init)
assert_contains "$output" "initialized ADP home" "init output"
output=$(run_adp "$REPO_ROOT" workspace add game-a "$PROJECT_ROOT")
assert_contains "$output" 'workspace "game-a" added' "workspace add output"
output=$(run_adp "$REPO_ROOT" phase add --workspace game-a p-release "Release readiness")
assert_contains "$output" "phase p-release added" "phase add output"
output=$(run_adp "$REPO_ROOT" phase start --workspace game-a p-release)
assert_contains "$output" "phase p-release status: active" "phase start output"
reset_git_tripwire
output=$(run_adp "$REPO_ROOT" phase accept --workspace game-a p-release --command "scripts/check-all.sh" --result passed --notes "fake gate")
assert_contains "$output" "phase p-release accepted: passed" "phase accept output"
output=$(run_adp "$REPO_ROOT" phase commit --workspace game-a p-release --hash 0123456789abcdef0123456789abcdef01234567 --message "fake commit evidence")
assert_contains "$output" "phase p-release commit" "phase commit output"
output=$(run_adp "$REPO_ROOT" phase push --workspace game-a p-release --remote origin --branch main --result pushed)
assert_contains "$output" "phase p-release push: origin/main pushed" "phase push output"
output=$(run_adp "$REPO_ROOT" phase status --workspace game-a --format json)
assert_contains "$output" '"next_action": "plan_next_phase"' "phase status output"
assert_no_git_side_effects "release readiness phase evidence recording"

info "release readiness smoke passed"
