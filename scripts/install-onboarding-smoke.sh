#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd -- "$SCRIPT_DIR/.." && pwd)

. "$SCRIPT_DIR/install-onboarding-smoke-lib.sh"
. "$SCRIPT_DIR/smoke-helpers.sh"

for cmd in bash go git; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    fail "$cmd is required"
  fi
done

TMP_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/adp-install-onboarding.XXXXXX")
BUILD_BIN="$TMP_ROOT/build/adp"
INSTALL_BIN="$TMP_ROOT/gobin"
FAKE_BIN="$TMP_ROOT/fake-bin"
JSON_VALIDATOR="$TMP_ROOT/json-valid"
PROJECT_ROOT="$TMP_ROOT/project"
ADP_HOME="$TMP_ROOT/adp-home"
ADP_RUNTIME_DIR="$TMP_ROOT/runtime"
GIT_TRIPWIRE_LOG="$TMP_ROOT/git-side-effects.log"
EVENTS_FILE="$ADP_HOME/logs/events.jsonl"
TASKS_FILE="$ADP_HOME/workspaces/onboarding-a/planning/tasks.yaml"
PHASES_FILE="$ADP_HOME/workspaces/onboarding-a/planning/phases.yaml"
PROGRESS_FILE="$ADP_HOME/workspaces/onboarding-a/planning/progress.jsonl"

VERSION="0.1.0-install-onboarding"
COMMIT="install-onboarding-smoke"
BUILD_DATE="2026-06-09T00:00:00Z"

cleanup() {
  rm -rf "$TMP_ROOT"
}
trap cleanup EXIT INT TERM

mkdir -p "$INSTALL_BIN" "$FAKE_BIN" "$PROJECT_ROOT" "$ADP_HOME" "$ADP_RUNTIME_DIR" "$(dirname -- "$BUILD_BIN")"
printf 'module example.com/adp-install-onboarding\n' > "$PROJECT_ROOT/go.mod"
printf 'package main\n' > "$PROJECT_ROOT/main.go"
init_project_git
write_fake_codex "$FAKE_BIN/codex"
write_fake_claude_guard "$FAKE_BIN/claude"

info "building local binary with deterministic release metadata"
build_local_binary
output=$("$BUILD_BIN" version)
assert_contains "$output" "adp version $VERSION" "local build version output"
assert_contains "$output" "commit: $COMMIT" "local build version output"
assert_contains "$output" "built: $BUILD_DATE" "local build version output"

info "installing adp into a temporary GOBIN"
install_to_temp_gobin
build_json_validator

export ADP_HOME
export ADP_RUNTIME_DIR
export PATH="$INSTALL_BIN:$FAKE_BIN:$PATH"
hash -r

if [ "$(command -v adp)" != "$INSTALL_BIN/adp" ]; then
  fail "temporary installed adp is not first on PATH"
fi
if [ "$(command -v codex)" != "$FAKE_BIN/codex" ]; then
  fail "fake codex is not first on PATH"
fi
if [ "$(command -v claude)" != "$FAKE_BIN/claude" ]; then
  fail "fake claude guard is not first on PATH"
fi

setup_git_tripwire "$FAKE_BIN" "$GIT_TRIPWIRE_LOG"

info "checking first-use help, examples, and parser hints"
output=$(run_adp "$TMP_ROOT" --help)
assert_contains "$output" "adp init" "root help output"
assert_contains "$output" "adp workspace add" "root help output"
assert_contains "$output" "adp run <agent>" "root help output"

output=$(run_adp "$TMP_ROOT" workspace --help)
assert_contains "$output" "Examples:" "workspace help output"
assert_contains "$output" "adp workspace add game-a /absolute/path/to/project" "workspace help output"
assert_contains "$output" "adp workspace doctor game-a --format json" "workspace help output"
output=$(run_adp "$TMP_ROOT" tasks take --help)
assert_contains "$output" "adp tasks take --workspace game-a --owner codex-main --lease 4h --format json" "tasks take help output"
output=$(run_adp "$TMP_ROOT" sessions resume-plan --help)
assert_contains "$output" "adp sessions resume-plan session-20260611-0001" "sessions resume-plan help output"
output=$(run_adp "$TMP_ROOT" runtime prune --help)
assert_contains "$output" "adp runtime prune --older-than 24h --dry-run --format json" "runtime prune help output"
output=$(run_adp "$TMP_ROOT" progress report --help)
assert_contains "$output" "adp progress report --workspace game-a --format json" "progress report help output"

output=$(run_adp_expect_fail "$TMP_ROOT" run codex --take)
assert_contains "$output" "--owner is required with --take" "run --take missing owner output"
assert_contains "$output" "try: adp run --help" "run --take missing owner output"
output=$(run_adp_expect_fail "$TMP_ROOT" tasks take task-123 --owner trial-agent)
assert_contains "$output" 'tasks take does not accept task id "task-123"' "tasks take task id guard output"
assert_contains "$output" "try: adp tasks take --help" "tasks take task id guard output"
output=$(run_adp_expect_fail "$TMP_ROOT" completion values widgets)
assert_contains "$output" 'unknown completion values kind "widgets"' "completion values guard output"
assert_contains "$output" "try: adp completion values --help" "completion values guard output"

assert_project_root_clean
if [ -e "$EVENTS_FILE" ]; then
  fail "help and parser checks created event log"
fi
if [ "$(runtime_entry_count "$ADP_RUNTIME_DIR")" != "0" ]; then
  fail "help and parser checks created runtime directories"
fi

info "running first-use onboarding commands through the installed binary"
output=$(run_adp "$TMP_ROOT" version)
assert_contains "$output" "adp version $VERSION" "installed version output"
assert_contains "$output" "commit: $COMMIT" "installed version output"
assert_contains "$output" "built: $BUILD_DATE" "installed version output"
output=$(run_adp "$TMP_ROOT" init)
assert_contains "$output" "initialized ADP home" "init output"
output=$(run_adp "$TMP_ROOT" workspace add onboarding-a "$PROJECT_ROOT")
assert_contains "$output" 'workspace "onboarding-a" added' "workspace add output"
output=$(run_adp "$TMP_ROOT" workspace list)
assert_contains "$output" "onboarding-a" "workspace list output"
output=$(run_adp "$TMP_ROOT" workspace doctor onboarding-a)
assert_contains "$output" "onboarding-a" "workspace doctor output"
assert_contains "$output" "ok" "workspace doctor output"
assert_contains "$output" "no issues" "workspace doctor output"
output=$(run_adp "$TMP_ROOT" doctor onboarding-a)
assert_contains "$output" "onboarding-a" "doctor output"
assert_contains "$output" "ok" "doctor output"
assert_contains "$output" "no issues" "doctor output"
output=$(run_adp "$TMP_ROOT" workspace doctor onboarding-a --verbose)
assert_contains "$output" "workspace.git.root.detected" "workspace doctor verbose output"
output=$(run_adp "$TMP_ROOT" workspace doctor onboarding-a --format json)
assert_json_valid "$output" "workspace doctor json output"
assert_contains "$output" '"code": "workspace.git.root.detected"' "workspace doctor json output"
output=$(run_adp "$TMP_ROOT" workspace show onboarding-a)
assert_contains "$output" "name: onboarding-a" "workspace show output"
assert_contains "$output" "project_root: $PROJECT_ROOT" "workspace show output"

info "creating a minimal phase and task for task-bound provider handoff"
output=$(run_adp "$TMP_ROOT" phase add --workspace onboarding-a --goal "install onboarding smoke" p-install "Install Onboarding")
assert_contains "$output" "phase p-install added" "phase add output"
output=$(run_adp "$TMP_ROOT" phase start --workspace onboarding-a p-install)
assert_contains "$output" "phase p-install status: active" "phase start output"
output=$(run_adp "$TMP_ROOT" tasks add --workspace onboarding-a --priority high --phase p-install --description "new operator onboarding path" "Run install onboarding")
assert_contains "$output" "task task-" "tasks add output"
TASK_ID=$(printf '%s\n' "$output" | sed -n 's/^task \(task-[^ ]*\) added$/\1/p')
if [ -z "$TASK_ID" ]; then
  fail "could not parse task id from: $output"
fi
export ADP_EXPECT_TASK_ID="$TASK_ID"

output=$(run_adp "$TMP_ROOT" completion values workspaces)
assert_contains "$output" "onboarding-a" "completion workspace values output"
output=$(run_adp "$TMP_ROOT" completion values agents)
assert_contains "$output" "codex" "completion agent values output"
assert_contains "$output" "claude" "completion agent values output"
output=$(run_adp "$TMP_ROOT" completion values phases --workspace onboarding-a)
assert_contains "$output" "p-install" "completion phase values output"
output=$(run_adp "$TMP_ROOT" completion values tasks --workspace onboarding-a)
assert_contains "$output" "$TASK_ID" "completion task values output"
output=$(run_adp "$TMP_ROOT" completion values statuses)
assert_contains "$output" "ready" "completion status values output"
assert_contains "$output" "done" "completion status values output"

assert_file "$TASKS_FILE"
assert_file "$PHASES_FILE"
assert_file "$PROGRESS_FILE"
assert_project_root_clean

info "running task-bound fake codex from the installed binary"
smoke_require_symlinks
reset_git_tripwire
output=$(run_adp "$TMP_ROOT" run codex --workspace onboarding-a --task "$TASK_ID" -- --install-onboarding)
assert_contains "$output" "fake-codex" "fake codex output"
assert_no_git_side_effects "install onboarding fake codex run"
assert_project_root_clean

info "checking events, sessions, progress, and planning diagnostics"
output=$(run_adp "$TMP_ROOT" events list --workspace onboarding-a --task "$TASK_ID" --type run_finished --limit 1)
assert_contains "$output" "run_finished" "events list output"
assert_contains "$output" "codex" "events list output"
assert_contains "$output" "$TASK_ID" "events list output"
output=$(run_adp "$TMP_ROOT" events list --workspace onboarding-a --task "$TASK_ID" --type run_finished --limit 1 --format json)
assert_json_valid "$output" "events list json output"
assert_contains "$output" "\"task_id\": \"$TASK_ID\"" "events list json output"
assert_contains "$output" '"events": [' "events list json output"
assert_file "$EVENTS_FILE"

output=$(run_adp "$TMP_ROOT" sessions list --workspace onboarding-a --agent codex --task "$TASK_ID")
assert_contains "$output" "codex" "sessions list output"
assert_contains "$output" "$TASK_ID" "sessions list output"
output=$(run_adp "$TMP_ROOT" sessions list --workspace onboarding-a --agent codex --task "$TASK_ID" --format json)
assert_json_valid "$output" "sessions list json output"
assert_contains "$output" "\"task_id\": \"$TASK_ID\"" "sessions list json output"
assert_contains "$output" '"sessions": [' "sessions list json output"

output=$(run_adp "$TMP_ROOT" progress --workspace onboarding-a --format json)
assert_json_valid "$output" "progress json output"
assert_contains "$output" '"workspace": "onboarding-a"' "progress json output"
assert_contains "$output" '"total": 1' "progress json output"
assert_contains "$output" '"counts"' "progress json output"

output=$(run_adp "$TMP_ROOT" plan doctor --workspace onboarding-a --format json)
assert_json_valid "$output" "plan doctor json output"
assert_contains "$output" '"workspace": "onboarding-a"' "plan doctor json output"
assert_contains "$output" '"status": "ok"' "plan doctor json output"
assert_contains "$output" '"task_count": 1' "plan doctor json output"
assert_contains "$output" '"phase_count": 1' "plan doctor json output"
assert_contains "$output" '"has_errors": false' "plan doctor json output"
assert_project_root_clean

info "checking friendly trial workflow pickup, lease, stale, and restore guidance"
output=$(run_adp "$TMP_ROOT" tasks add --workspace onboarding-a --priority critical --phase p-install --description "atomic worker pickup" "Claim trial workflow")
assert_contains "$output" "task task-" "trial take task add output"
TAKE_TASK_ID=$(printf '%s\n' "$output" | sed -n 's/^task \(task-[^ ]*\) added$/\1/p')
if [ -z "$TAKE_TASK_ID" ]; then
  fail "could not parse take task id from: $output"
fi
export ADP_EXPECT_TAKE_TASK_ID="$TAKE_TASK_ID"

output=$(run_adp "$TMP_ROOT" tasks next --workspace onboarding-a --limit 1 --format json)
assert_json_valid "$output" "tasks next json output"
assert_contains "$output" "\"$TAKE_TASK_ID\"" "tasks next json output"
assert_contains "$output" '"eligible_count": 1' "tasks next json output"
assert_contains "$output" '"claim_state": "unclaimed"' "tasks next json output"

reset_git_tripwire
events_before=$(line_count "$EVENTS_FILE")
output=$(run_adp "$TMP_ROOT" run codex --workspace onboarding-a --take --owner trial-agent --lease 30m -- --trial-take)
assert_contains "$output" "fake-codex" "run take fake codex output"
assert_contains "$output" "--trial-take" "run take fake codex output"
assert_no_git_side_effects "install onboarding run --take"
assert_project_root_clean
if [ "$(line_count "$EVENTS_FILE")" != $((events_before + 2)) ]; then
  fail "run --take should append two runtime events"
fi

take_session=$(session_id_by_agent "$EVENTS_FILE" codex)
if [ -z "$take_session" ]; then
  cat "$EVENTS_FILE" >&2
  fail "run --take session id missing in event log"
fi

output=$(run_adp "$TMP_ROOT" completion values sessions --workspace onboarding-a)
assert_contains "$output" "$take_session" "completion session values output"

output=$(run_adp "$TMP_ROOT" tasks show --workspace onboarding-a "$TAKE_TASK_ID")
assert_contains "$output" "status: in_progress" "taken task show output"
assert_contains "$output" "owner: trial-agent" "taken task show output"
assert_contains "$output" "claim_state: leased" "taken task show output"
assert_contains "$output" "lease_expires_at: 20" "taken task show output"
output=$(run_adp "$TMP_ROOT" completion values owners --workspace onboarding-a)
assert_contains "$output" "trial-agent" "completion owner values output"

output=$(run_adp "$TMP_ROOT" tasks renew --workspace onboarding-a "$TAKE_TASK_ID" --owner trial-agent --lease 45m)
assert_contains "$output" "task $TAKE_TASK_ID lease renewed until" "tasks renew output"

output=$(run_adp "$TMP_ROOT" tasks add --workspace onboarding-a --priority low --phase p-install --description "expired worker claim" "Recover stale trial workflow")
assert_contains "$output" "task task-" "stale task add output"
STALE_TASK_ID=$(printf '%s\n' "$output" | sed -n 's/^task \(task-[^ ]*\) added$/\1/p')
if [ -z "$STALE_TASK_ID" ]; then
  fail "could not parse stale task id from: $output"
fi
output=$(run_adp "$TMP_ROOT" tasks claim --workspace onboarding-a "$STALE_TASK_ID" --owner abandoned-agent --lease 1ms)
assert_contains "$output" "task $STALE_TASK_ID claimed by abandoned-agent" "stale task claim output"
sleep 1
output=$(run_adp "$TMP_ROOT" tasks stale --workspace onboarding-a --format json)
assert_json_valid "$output" "tasks stale json output"
assert_contains "$output" '"stale_count": 1' "tasks stale json output"
assert_contains "$output" "\"$STALE_TASK_ID\"" "tasks stale json output"
assert_contains "$output" '"owner": "abandoned-agent"' "tasks stale json output"
assert_contains "$output" '"claim_state": "stale"' "tasks stale json output"

output=$(run_adp "$TMP_ROOT" sessions restore-plan "$take_session")
assert_contains "$output" "session_id: $take_session" "take restore-plan output"
assert_contains "$output" "status: ready" "take restore-plan output"
assert_contains "$output" "adp run codex --workspace onboarding-a --task $TAKE_TASK_ID" "take restore-plan output"
assert_contains "$output" "-- --trial-take" "take restore-plan output"
output=$(run_adp "$TMP_ROOT" sessions restore-plan "$take_session" --format json)
assert_json_valid "$output" "take restore-plan json output"
assert_contains "$output" "\"session_id\": \"$take_session\"" "take restore-plan json output"
assert_contains "$output" '"status": "ready"' "take restore-plan json output"

output=$(run_adp "$TMP_ROOT" progress report --workspace onboarding-a)
assert_contains "$output" "# ADP Progress Report" "progress report output"
assert_contains "$output" "$TAKE_TASK_ID" "progress report output"
assert_contains "$output" "$STALE_TASK_ID" "progress report output"
assert_contains "$output" "$take_session" "progress report output"
assert_contains "$output" "Claim" "progress report output"
assert_contains "$output" "leased to trial-agent" "progress report output"
assert_contains "$output" "stale claim by abandoned-agent" "progress report output"
output=$(run_adp "$TMP_ROOT" progress report --workspace onboarding-a --format json)
assert_json_valid "$output" "progress report json output"
assert_contains "$output" '"runtime_sessions"' "progress report json output"
assert_contains "$output" "\"$take_session\"" "progress report json output"
output=$(run_adp "$TMP_ROOT" runtime prune --older-than 24h --dry-run --format json)
assert_json_valid "$output" "runtime prune dry-run json output"
assert_contains "$output" '"dry_run": true' "runtime prune dry-run json output"
assert_contains "$output" '"results": [' "runtime prune dry-run json output"
assert_project_root_clean

info "install onboarding smoke passed"
