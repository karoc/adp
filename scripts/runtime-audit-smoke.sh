#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd -- "$SCRIPT_DIR/.." && pwd)
. "$SCRIPT_DIR/runtime-smoke-lib.sh"

fail() {
  printf 'runtime-audit-smoke: %s\n' "$*" >&2
  exit 1
}

info() {
  printf '[runtime-audit-smoke] %s\n' "$*"
}

. "$SCRIPT_DIR/smoke-git-tripwire-lib.sh"

run_adp_stdin() {
  local dir="$1"
  local input="$2"
  shift 2
  local output

  if ! output=$(printf '%s' "$input" | (cd "$dir" && "$ADP_BIN" "$@" 2>&1)); then
    printf '%s\n' "$output" >&2
    fail "adp $* with stdin failed"
  fi
  printf '%s\n' "$output"
}

run_adp_expect_code() {
  local want_code="$1"
  local dir="$2"
  local output
  local code
  shift 2

  set +e
  output=$(cd "$dir" && "$ADP_BIN" "$@" 2>&1)
  code=$?
  set -e

  if [ "$code" != "$want_code" ]; then
    printf '%s\n' "$output" >&2
    fail "adp $* exit code $code, want $want_code"
  fi
  printf '%s\n' "$output"
}

run_script_expect_code() {
  local want_code="$1"
  local output
  local code
  shift

  set +e
  output=$(cd "$REPO_ROOT" && "$@" 2>&1)
  code=$?
  set -e

  if [ "$code" != "$want_code" ]; then
    printf '%s\n' "$output" >&2
    fail "$* exit code $code, want $want_code"
  fi
  printf '%s\n' "$output"
}

assert_help() {
  local label="$1"
  local needle="$2"
  local output
  shift 2

  output=$(run_adp "$REPO_ROOT" "$@")
  assert_contains "$output" "Usage:" "$label"
  assert_contains "$output" "$needle" "$label"
}

assert_json_valid() {
  local output="$1"
  local label="$2"

  if ! printf '%s' "$output" | "$JSON_VALIDATOR" >/dev/null 2>&1; then
    printf '%s\n' "$output" >&2
    fail "$label was not valid JSON"
  fi
}

build_json_validator() {
  cat > "$TMP_ROOT/json-valid.go" <<'EOF'
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func main() {
	dec := json.NewDecoder(os.Stdin)
	dec.UseNumber()

	var value any
	if err := dec.Decode(&value); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			fmt.Fprintln(os.Stderr, "multiple JSON values")
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
EOF
  go build -o "$JSON_VALIDATOR" "$TMP_ROOT/json-valid.go"
}

write_enter_probe_shell() {
  local path="$1"

  cat > "$path" <<'EOF'
#!/usr/bin/env sh
set -eu

printf 'enter workspace=%s cwd=%s\n' "${ADP_WORKSPACE:-}" "$(pwd)"
test "${ADP_WORKSPACE:-}" = "game-a"
test -n "${ADP_RUNTIME_ROOT:-}"
test "$(pwd)" = "$ADP_RUNTIME_ROOT"
test -f "$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
test -L "$ADP_RUNTIME_ROOT/go.mod"
EOF
  chmod 755 "$path"
}

if ! command -v go >/dev/null 2>&1; then
  fail "Go is required to build cmd/adp"
fi

TMP_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/adp-runtime-audit-smoke.XXXXXX")
ADP_BIN="$TMP_ROOT/adp"
JSON_VALIDATOR="$TMP_ROOT/json-valid"

cleanup() {
  rm -rf "$TMP_ROOT"
}
trap cleanup EXIT INT TERM

PROJECT_ROOT="$TMP_ROOT/project"
ADP_HOME="$TMP_ROOT/adp-home"
ADP_RUNTIME_DIR="$TMP_ROOT/runtime"
FAKE_BIN="$TMP_ROOT/bin"
GIT_TRIPWIRE_LOG="$TMP_ROOT/git-side-effects.log"
EVENTS_FILE="$ADP_HOME/logs/events.jsonl"
TASKS_FILE="$ADP_HOME/workspaces/game-a/planning/tasks.yaml"
PHASES_FILE="$ADP_HOME/workspaces/game-a/planning/phases.yaml"
PROGRESS_FILE="$ADP_HOME/workspaces/game-a/planning/progress.jsonl"
PLAN_FILE="$TMP_ROOT/plan.yaml"

mkdir -p "$PROJECT_ROOT" "$ADP_HOME" "$ADP_RUNTIME_DIR" "$FAKE_BIN"
printf 'module example.com/adp-runtime-audit-smoke\n' > "$PROJECT_ROOT/go.mod"
printf 'package main\n' > "$PROJECT_ROOT/main.go"

write_fake_agent "$FAKE_BIN/codex" codex AGENTS.md .codex/config.toml go.mod
write_fake_agent "$FAKE_BIN/claude" claude CLAUDE.md .claude/settings.json main.go
write_enter_probe_shell "$FAKE_BIN/enter-shell"
setup_git_tripwire "$FAKE_BIN" "$GIT_TRIPWIRE_LOG"

export ADP_HOME
export ADP_RUNTIME_DIR
export PATH="$FAKE_BIN:$PATH"

info "auditing real-agent invocation smoke safety gates"
output=$(run_script_expect_code 0 scripts/real-agent-invocation-smoke.sh --help)
assert_contains "$output" "ADP_REAL_INVOKE_CODEX=1" "real invocation help output"
assert_contains "$output" "ADP_REAL_INVOKE_CLAUDE=1" "real invocation help output"

output=$(ADP_REAL_INVOKE_CODEX= ADP_REAL_INVOKE_CLAUDE= run_script_expect_code 1 scripts/real-agent-invocation-smoke.sh --codex)
assert_contains "$output" "requires ADP_REAL_INVOKE_CODEX=1" "real invocation codex missing-gate output"
assert_not_contains "$output" "building temporary adp binary" "real invocation codex missing-gate output"
assert_not_contains "$output" "running real Codex through ADP" "real invocation codex missing-gate output"

output=$(ADP_REAL_INVOKE_CODEX=1 ADP_REAL_INVOKE_CLAUDE= run_script_expect_code 1 scripts/real-agent-invocation-smoke.sh --all)
assert_contains "$output" "requires ADP_REAL_INVOKE_CLAUDE=1" "real invocation all missing-gate output"
assert_not_contains "$output" "building temporary adp binary" "real invocation all missing-gate output"
assert_not_contains "$output" "running real Codex through ADP" "real invocation all missing-gate output"
assert_not_contains "$output" "running real Claude through ADP" "real invocation all missing-gate output"

info "building temporary adp binary"
(cd "$REPO_ROOT" && go build -o "$ADP_BIN" ./cmd/adp)
build_json_validator

info "auditing help and command discoverability"
assert_help "root help" "adp run <agent>" --help
assert_help "init help" "adp init" init --help
assert_help "doctor help" "adp doctor [workspace]" doctor --help
assert_help "version help" "adp version" version --help
assert_help "workspace help" "adp workspace add" workspace --help
assert_help "workspace add help" "adp workspace add <name> <project-root>" workspace add --help
assert_help "workspace doctor help" "adp workspace doctor [name]" workspace doctor --help
assert_help "enter help" "adp enter <workspace>" enter --help
assert_help "env help" "adp env <workspace>" env --help
assert_help "shell-hook help" "adp shell-hook" shell-hook --help
assert_help "completion help" "adp completion values" completion --help
assert_help "completion values help" "adp completion values <agents|workspaces|profiles|tasks|phases|sessions|owners|statuses>" completion values --help
assert_help "events help" "adp events list" events --help
assert_help "events list help" "adp events list" events list --help
assert_help "sessions help" "adp sessions restore-plan" sessions --help
assert_help "sessions restore-plan help" "adp sessions restore-plan <session-id>" sessions restore-plan --help
assert_help "runtime help" "adp runtime prune" runtime --help
assert_help "runtime prune help" "adp runtime prune" runtime prune --help
assert_help "tasks help" "adp tasks next" tasks --help
assert_help "tasks help take" "adp tasks take" tasks --help
assert_help "tasks add help" "adp tasks add" tasks add --help
assert_help "tasks take help" "adp tasks take" tasks take --help
assert_help "tasks claim help" "adp tasks claim" tasks claim --help
assert_help "plan help" "adp plan preview" plan --help
assert_help "plan doctor help" "adp plan doctor" plan doctor --help
assert_help "phase help" "adp phase accept" phase --help
assert_help "phase commit help" "adp phase commit" phase commit --help
assert_help "progress help" "adp progress report" progress --help
assert_help "progress report help" "adp progress report" progress report --help
assert_help "run help" "adp run <agent>" run --help
output=$(run_adp_expect_fail "$REPO_ROOT" run)
assert_contains "$output" "--take --owner <owner>" "run usage output"
output=$(run_adp_expect_fail "$REPO_ROOT" run codex --take)
assert_contains "$output" "--owner is required with --take" "run take owner guard output"
output=$(run_adp_expect_fail "$REPO_ROOT" run codex --owner audit-agent)
assert_contains "$output" "--owner requires --take" "run owner guard output"

unknown_output=$(run_adp_expect_fail "$REPO_ROOT" bogus)
assert_contains "$unknown_output" 'unknown command "bogus"' "unknown command output"

info "auditing workspace, completion, shell hook, and diagnostics"
output=$(run_adp "$REPO_ROOT" init)
assert_contains "$output" "initialized ADP home" "init output"
output=$(run_adp "$REPO_ROOT" workspace add game-a "$PROJECT_ROOT")
assert_contains "$output" 'workspace "game-a" added' "workspace add output"
output=$(run_adp "$REPO_ROOT" workspace list)
assert_contains "$output" "game-a" "workspace list output"
output=$(run_adp "$REPO_ROOT" workspace show game-a)
assert_contains "$output" "project_root: $PROJECT_ROOT" "workspace show output"
output=$(run_adp "$REPO_ROOT" doctor game-a)
assert_contains "$output" "ok" "doctor output"
output=$(run_adp "$REPO_ROOT" workspace doctor game-a)
assert_contains "$output" "ok" "workspace doctor output"
output=$(run_adp "$REPO_ROOT" completion --shell bash --command tasks)
assert_contains "$output" "_tasks_completion" "completion output"
assert_contains "$output" "complete -F _tasks_completion tasks" "completion output"
output=$(run_adp "$REPO_ROOT" completion values workspaces)
assert_contains "$output" "game-a" "completion workspace values output"
output=$(run_adp "$REPO_ROOT" completion values profiles --workspace game-a)
assert_contains "$output" "default" "completion profile values output"
output=$(run_adp "$REPO_ROOT" completion values agents)
assert_contains "$output" "codex" "completion agent values output"
assert_contains "$output" "claude" "completion agent values output"
output=$(run_adp "$REPO_ROOT" shell-hook --shell bash --name adpx)
assert_contains "$output" "adpx()" "shell-hook output"
assert_contains "$output" "adp env" "shell-hook output"

info "auditing task and phase lifecycle commands"
output=$(run_adp "$REPO_ROOT" phase add --workspace game-a --goal "Audit local phase gates" p-audit "Audit phase")
assert_contains "$output" "phase p-audit added" "phase add output"
output=$(run_adp "$REPO_ROOT" completion values phases --workspace game-a)
assert_contains "$output" "p-audit" "completion phase values output"
output=$(run_adp "$REPO_ROOT" phase status --workspace game-a --format json)
assert_json_valid "$output" "phase status json output"
assert_contains "$output" '"can_start_next": true' "phase status json output"
output=$(run_adp "$REPO_ROOT" phase start --workspace game-a p-audit)
assert_contains "$output" "phase p-audit status: active" "phase start output"
output=$(run_adp "$REPO_ROOT" tasks add --workspace game-a --priority high --phase p-audit --description "runtime audit task" "Bind runtime session to task")
assert_contains "$output" "task task-" "tasks add output"
task_id=$(printf '%s\n' "$output" | sed -n 's/^task \(task-[^ ]*\) added$/\1/p')
if [ -z "$task_id" ]; then
  fail "could not parse task id from: $output"
fi
export ADP_EXPECT_TASK_ID="$task_id"
output=$(run_adp "$REPO_ROOT" completion values tasks --workspace game-a)
assert_contains "$output" "$task_id" "completion task values output"
output=$(run_adp "$REPO_ROOT" completion values statuses)
assert_contains "$output" "review" "completion status values output"
assert_contains "$output" "done" "completion status values output"
output=$(run_adp "$REPO_ROOT" tasks list --workspace game-a --format json)
assert_json_valid "$output" "tasks list json output"
assert_contains "$output" "$task_id" "tasks list json output"
output=$(run_adp "$REPO_ROOT" tasks next --workspace game-a --format json)
assert_json_valid "$output" "tasks next json output"
assert_contains "$output" "$task_id" "tasks next json output"
output=$(run_adp "$REPO_ROOT" tasks show --workspace game-a "$task_id" --format json)
assert_json_valid "$output" "tasks show json output"
assert_contains "$output" "Bind runtime session to task" "tasks show json output"
output=$(run_adp "$REPO_ROOT" tasks claim --workspace game-a "$task_id" --owner audit-agent --lease 10m)
assert_contains "$output" "claimed by audit-agent" "tasks claim output"
output=$(run_adp "$REPO_ROOT" completion values owners --workspace game-a)
assert_contains "$output" "audit-agent" "completion owner values output"
output=$(run_adp "$REPO_ROOT" tasks release --workspace game-a "$task_id" --owner audit-agent)
assert_contains "$output" "released" "tasks release output"
output=$(run_adp "$REPO_ROOT" tasks update --workspace game-a "$task_id" --status review)
assert_contains "$output" "status: review" "tasks update output"
output=$(run_adp "$REPO_ROOT" tasks block --workspace game-a "$task_id" --reason "audit blocker")
assert_contains "$output" "blocked" "tasks block output"
output=$(run_adp "$REPO_ROOT" tasks done --workspace game-a "$task_id")
assert_contains "$output" "done" "tasks done output"

info "auditing plan preview/apply/doctor and progress reports"
cat > "$PLAN_FILE" <<'YAML'
version: 1
phases:
  - id: p-audit-next
    title: Next audit phase
tasks:
  - title: Next audit task
    priority: medium
    phase: p-audit-next
YAML
output=$(run_adp "$REPO_ROOT" plan preview --workspace game-a --file "$PLAN_FILE" --format json)
assert_json_valid "$output" "plan preview json output"
assert_contains "$output" '"preview"' "plan preview json output"
output=$(run_adp "$REPO_ROOT" plan apply --workspace game-a --file "$PLAN_FILE" --format json)
assert_json_valid "$output" "plan apply json output"
assert_contains "$output" "p-audit-next" "plan apply json output"
stdin_preview_plan=$(cat <<'YAML'
version: 1
phases:
  - id: p-audit-stdin-preview
    title: Stdin preview audit phase
tasks:
  - title: Stdin preview audit task
    priority: low
    phase: p-audit-stdin-preview
YAML
)
output=$(run_adp_stdin "$REPO_ROOT" "$stdin_preview_plan" plan preview --workspace game-a --file -)
assert_contains "$output" "mode: preview" "stdin plan preview output"
assert_contains "$output" "p-audit-stdin-preview" "stdin plan preview output"
output=$(run_adp "$REPO_ROOT" plan doctor --workspace game-a --format json)
assert_json_valid "$output" "plan doctor json output"
assert_contains "$output" '"status": "ok"' "plan doctor json output"
output=$(run_adp "$REPO_ROOT" progress --workspace game-a --format json)
assert_json_valid "$output" "progress json output"
assert_contains "$output" '"workspace": "game-a"' "progress json output"
output=$(run_adp "$REPO_ROOT" progress report --workspace game-a --language en --format markdown)
assert_contains "$output" "# ADP Progress Report" "progress report English output"
output=$(run_adp "$REPO_ROOT" progress report --workspace game-a --language zh-CN --format markdown)
assert_contains "$output" "# ADP 执行进度报告" "progress report Chinese output"
output=$(run_adp "$REPO_ROOT" progress report --workspace game-a --format json)
assert_json_valid "$output" "progress report json output"
assert_contains "$output" '"runtime_sessions"' "progress report json output"

info "auditing runtime, events, sessions, restore plan, and prune"
env_output=$(run_adp "$REPO_ROOT" env game-a --cd)
runtime_root=$(parse_export "$env_output" ADP_RUNTIME_ROOT)
assert_contains "$env_output" "cd '$runtime_root'" "env output"
assert_file "$runtime_root/.adp-runtime.yaml"
output=$(run_adp "$REPO_ROOT" tasks add --workspace game-a --priority critical --phase p-audit --description "runtime take audit task" "Bind runtime session to task")
assert_contains "$output" "task task-" "run take task add output"
take_task_id=$(printf '%s\n' "$output" | sed -n 's/^task \(task-[^ ]*\) added$/\1/p')
if [ -z "$take_task_id" ]; then
  fail "could not parse run take task id from: $output"
fi
export ADP_EXPECT_TASK_ID="$take_task_id"
phases_before_take=$(cat "$PHASES_FILE")
progress_events_before_take=$(line_count "$PROGRESS_FILE")
events_before_take=0
if [ -f "$EVENTS_FILE" ]; then
  events_before_take=$(line_count "$EVENTS_FILE")
fi
runtime_entries_before_take=$(runtime_entry_count "$ADP_RUNTIME_DIR")
reset_git_tripwire
output=$(run_adp "$REPO_ROOT" run codex --workspace game-a --take --owner audit-run-take --lease 15m -- --probe codex-payload)
assert_contains "$output" "fake-codex" "codex run take output"
if [ "$(line_count "$EVENTS_FILE")" != $((events_before_take + 2)) ]; then
  fail "run --take should append two runtime events"
fi
if [ "$(line_count "$PROGRESS_FILE")" != $((progress_events_before_take + 1)) ]; then
  fail "run --take should append one task claim event"
fi
take_progress=$(tail -n 1 "$PROGRESS_FILE")
assert_contains "$take_progress" "\"type\":\"task_claimed\"" "run take progress event"
assert_contains "$take_progress" "\"task_id\":\"$take_task_id\"" "run take progress event"
assert_contains "$take_progress" "\"owner\":\"audit-run-take\"" "run take progress event"
take_events=$(cat "$EVENTS_FILE")
assert_contains "$take_events" "\"task_binding\":\"take\"" "run take event metadata"
assert_contains "$take_events" "\"owner\":\"audit-run-take\"" "run take event metadata"
assert_contains "$take_events" "\"lease_seconds\":900" "run take event metadata"
output=$(run_adp "$REPO_ROOT" tasks show --workspace game-a "$take_task_id" --format json)
assert_json_valid "$output" "run take task json output"
assert_contains "$output" "\"status\": \"in_progress\"" "run take task json output"
assert_contains "$output" "\"owner\": \"audit-run-take\"" "run take task json output"
take_session=$(session_id_by_agent "$EVENTS_FILE" codex)
output=$(run_adp "$REPO_ROOT" sessions show "$take_session")
assert_contains "$output" "task_id: $take_task_id" "run take session output"
assert_contains "$output" "run_finished" "run take session output"
if [ "$(cat "$PHASES_FILE")" != "$phases_before_take" ]; then
  fail "run --take changed phase evidence"
fi
if [ "$(runtime_entry_count "$ADP_RUNTIME_DIR")" != "$runtime_entries_before_take" ]; then
  fail "run --take leaked a runtime directory"
fi
assert_no_git_side_effects "runtime audit run --take"
assert_absent_project_artifacts "$PROJECT_ROOT"
export ADP_EXPECT_TASK_ID="$task_id"
output=$(run_adp "$REPO_ROOT" run codex --workspace game-a --task "$task_id" -- --probe codex-payload)
assert_contains "$output" "fake-codex" "codex run output"
output=$(run_adp "$REPO_ROOT" run claude --workspace game-a --task "$task_id" -- --probe claude-payload)
assert_contains "$output" "fake-claude" "claude run output"
output=$(SHELL="$FAKE_BIN/enter-shell" run_adp "$REPO_ROOT" enter game-a)
assert_contains "$output" "enter workspace=game-a" "enter output"
output=$(run_adp "$REPO_ROOT" events list --workspace game-a --task "$task_id" --type run_finished --limit 5)
assert_contains "$output" "run_finished" "events list output"
codex_session=$(session_id_by_agent "$EVENTS_FILE" codex)
if [ -z "$codex_session" ]; then
  cat "$EVENTS_FILE" >&2
  fail "codex session id missing in event log"
fi
output=$(run_adp "$REPO_ROOT" sessions list --workspace game-a --agent codex --task "$task_id")
assert_contains "$output" "$codex_session" "sessions list output"
output=$(run_adp "$REPO_ROOT" completion values sessions --workspace game-a)
assert_contains "$output" "$codex_session" "completion session values output"
output=$(run_adp "$REPO_ROOT" sessions show "$codex_session")
assert_contains "$output" "session_id: $codex_session" "sessions show output"
before_restore_lines=$(line_count "$EVENTS_FILE")
output=$(run_adp "$REPO_ROOT" sessions restore-plan "$codex_session")
assert_contains "$output" "status: ready" "sessions restore-plan output"
assert_contains "$output" "adp run codex --workspace game-a" "sessions restore-plan output"
after_restore_lines=$(line_count "$EVENTS_FILE")
if [ "$after_restore_lines" != "$before_restore_lines" ]; then
  fail "sessions restore-plan appended events"
fi
output=$(run_adp "$REPO_ROOT" runtime prune --older-than 0s --include-kept --dry-run)
assert_contains "$output" "would-remove" "runtime prune dry-run output"

info "auditing phase acceptance evidence commands"
reset_git_tripwire
output=$(run_adp "$REPO_ROOT" phase accept --workspace game-a p-audit --command "scripts/runtime-audit-smoke.sh" --result passed --notes "runtime audit smoke passed")
assert_contains "$output" "phase p-audit accepted: passed" "phase accept output"
output=$(run_adp "$REPO_ROOT" phase commit --workspace game-a p-audit --hash 0123456789abcdef0123456789abcdef01234567 --message "Audit smoke fixture")
assert_contains "$output" "phase p-audit commit" "phase commit output"
output=$(run_adp "$REPO_ROOT" phase push --workspace game-a p-audit --remote origin --branch main --result pushed)
assert_contains "$output" "phase p-audit push: origin/main pushed" "phase push output"
assert_no_git_side_effects "runtime audit phase evidence recording"
output=$(run_adp "$REPO_ROOT" phase show --workspace game-a p-audit --format json)
assert_json_valid "$output" "phase show json output"
assert_contains "$output" '"status": "pushed"' "phase show json output"

assert_absent_project_artifacts "$PROJECT_ROOT"
info "runtime audit smoke passed"
