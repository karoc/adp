#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage:
  scripts/real-agent-invocation-smoke.sh [--codex] [--claude] [--all]

Runs explicit real-agent invocation evidence through ADP. This script may
contact external providers and consume account quota. It is intentionally not
part of scripts/check-all.sh. With no provider target selected, it performs a
provider-free opt-in guidance check and exits successfully without building ADP,
resolving Codex or Claude, creating runtimes, or invoking external CLIs.

Each provider requires both a command-line flag and an environment gate:

  ADP_REAL_INVOKE_CODEX=1 scripts/real-agent-invocation-smoke.sh --codex
  ADP_REAL_INVOKE_CLAUDE=1 scripts/real-agent-invocation-smoke.sh --claude
  ADP_REAL_INVOKE_CODEX=1 ADP_REAL_INVOKE_CLAUDE=1 scripts/real-agent-invocation-smoke.sh --all

Optional command/model overrides:

  ADP_REAL_CODEX_BIN=/path/to/codex
  ADP_REAL_CLAUDE_BIN=/path/to/claude
  ADP_REAL_CODEX_MODEL=<model>
  ADP_REAL_CLAUDE_MODEL=<model>

Optional timeout/budget controls:

  ADP_REAL_AGENT_TIMEOUT=180s
  ADP_REAL_CLAUDE_MAX_BUDGET_USD=0.20
USAGE
}

provider_free_guidance() {
  cat <<'GUIDANCE'
[real-agent-invocation-smoke] no real provider target selected
[real-agent-invocation-smoke] provider-free guidance check passed

This script only invokes Codex or Claude when both a provider flag and its
matching environment gate are present, for example:

  ADP_REAL_INVOKE_CODEX=1 scripts/real-agent-invocation-smoke.sh --codex
  ADP_REAL_INVOKE_CLAUDE=1 scripts/real-agent-invocation-smoke.sh --claude

ADP can validate local launch wiring, runtime overlays, task binding, events,
sessions, and project-root cleanliness. Real provider credentials, model
access, quota, network access, external CLI release behavior, and interactive
provider behavior remain operator-environment concerns.
GUIDANCE
}

fail() {
  printf 'real-agent-invocation-smoke: %s\n' "$*" >&2
  exit 1
}

info() {
  printf '[real-agent-invocation-smoke] %s\n' "$*"
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

assert_file() {
  local path="$1"
  if [ ! -f "$path" ]; then
    fail "missing file: $path"
  fi
}

assert_absent_project_artifacts() {
  local project_root="$1"
  local rel

  for rel in AGENTS.md CLAUDE.md .codex .claude .adp-runtime.yaml planning tasks.yaml phases.yaml progress.jsonl; do
    if [ -e "$project_root/$rel" ] || [ -L "$project_root/$rel" ]; then
      fail "project root was polluted with $rel"
    fi
  done
}

yaml_quote() {
  printf '"%s"' "$(printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g')"
}

require_gate() {
  local gate_var="$1"
  local label="$2"

  if [ "${!gate_var:-}" != "1" ]; then
    fail "real $label invocation requires $gate_var=1"
  fi
}

resolve_command() {
  local command_name="$1"
  local label="$2"
  local resolved

  if ! resolved=$(command -v "$command_name" 2>/dev/null); then
    fail "real $label invocation requested, but command is not available: $command_name"
  fi
  printf '%s\n' "$resolved"
}

run_adp() {
  local dir="$1"
  shift
  local output

  if command -v timeout >/dev/null 2>&1; then
    if ! output=$(cd "$dir" && timeout "$ADP_REAL_AGENT_TIMEOUT" "$ADP_BIN" "$@" 2>&1); then
      printf '%s\n' "$output" >&2
      fail "adp $* failed. If the external CLI was reached, triage provider credentials, model access, quota, network access, and installed CLI behavior before changing ADP launch wiring."
    fi
  elif ! output=$(cd "$dir" && "$ADP_BIN" "$@" 2>&1); then
    printf '%s\n' "$output" >&2
    fail "adp $* failed. If the external CLI was reached, triage provider credentials, model access, quota, network access, and installed CLI behavior before changing ADP launch wiring."
  fi
  printf '%s\n' "$output"
}

session_id_by_agent() {
  local events_file="$1"
  local agent="$2"
  local id

  id=$(
    {
      grep '"type":"run_started"' "$events_file" |
        grep "\"agent\":\"$agent\"" |
        sed -n 's/.*"session_id":"\([^"]*\)".*/\1/p' |
        tail -n 1
    } || true
  )
  printf '%s\n' "$id"
}

run_real_codex() {
  local output_file="$TMP_ROOT/codex-last-message.txt"
  local marker="ADP_REAL_CODEX_OK"
  local prompt="Reply with exactly $marker and nothing else."
  local args
  local output
  local evidence
  local session

  require_gate ADP_REAL_INVOKE_CODEX codex
  info "running real Codex through ADP"

  args=(exec --skip-git-repo-check --sandbox read-only --ephemeral --color never --output-last-message "$output_file")
  if [ -n "${ADP_REAL_CODEX_MODEL:-}" ]; then
    args+=(--model "$ADP_REAL_CODEX_MODEL")
  fi
  args+=("$prompt")

  output=$(run_adp "$REPO_ROOT" run codex --workspace real-agent-smoke --task "$TASK_ID" -- "${args[@]}")
  evidence="$output"
  if [ -f "$output_file" ]; then
    evidence="$evidence
$(cat "$output_file")"
  fi
  assert_contains "$evidence" "$marker" "real Codex output"
  assert_absent_project_artifacts "$PROJECT_ROOT"

  session=$(session_id_by_agent "$EVENTS_FILE" codex)
  if [ -z "$session" ]; then
    fail "missing Codex session id in event log"
  fi
  output=$(run_adp "$REPO_ROOT" sessions show "$session")
  assert_contains "$output" "agent: codex" "codex session output"
  assert_contains "$output" "task_id: $TASK_ID" "codex session output"
  output=$(run_adp "$REPO_ROOT" sessions restore-plan "$session")
  assert_contains "$output" "adp run codex --workspace real-agent-smoke" "codex restore-plan output"
}

run_real_claude() {
  local marker="ADP_REAL_CLAUDE_OK"
  local prompt="Reply with exactly $marker and nothing else."
  local args
  local output
  local session

  require_gate ADP_REAL_INVOKE_CLAUDE claude
  info "running real Claude through ADP"

  args=(-p --no-session-persistence --permission-mode plan --max-budget-usd "$ADP_REAL_CLAUDE_MAX_BUDGET_USD" --output-format text)
  if [ -n "${ADP_REAL_CLAUDE_MODEL:-}" ]; then
    args+=(--model "$ADP_REAL_CLAUDE_MODEL")
  fi
  args+=("$prompt")

  output=$(run_adp "$REPO_ROOT" run claude --workspace real-agent-smoke --task "$TASK_ID" -- "${args[@]}")
  assert_contains "$output" "$marker" "real Claude output"
  assert_absent_project_artifacts "$PROJECT_ROOT"

  session=$(session_id_by_agent "$EVENTS_FILE" claude)
  if [ -z "$session" ]; then
    fail "missing Claude session id in event log"
  fi
  output=$(run_adp "$REPO_ROOT" sessions show "$session")
  assert_contains "$output" "agent: claude" "claude session output"
  assert_contains "$output" "task_id: $TASK_ID" "claude session output"
  output=$(run_adp "$REPO_ROOT" sessions restore-plan "$session")
  assert_contains "$output" "adp run claude --workspace real-agent-smoke" "claude restore-plan output"
}

run_codex=0
run_claude=0

while [ "$#" -gt 0 ]; do
  case "$1" in
    --codex)
      run_codex=1
      ;;
    --claude)
      run_claude=1
      ;;
    --all)
      run_codex=1
      run_claude=1
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

if [ "$run_codex" -eq 0 ] && [ "$run_claude" -eq 0 ]; then
  provider_free_guidance
  exit 0
fi
if [ "$run_codex" -eq 1 ]; then
  require_gate ADP_REAL_INVOKE_CODEX codex
fi
if [ "$run_claude" -eq 1 ]; then
  require_gate ADP_REAL_INVOKE_CLAUDE claude
fi
if ! command -v go >/dev/null 2>&1; then
  fail "Go is required to build cmd/adp"
fi

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd -- "$SCRIPT_DIR/.." && pwd)
TMP_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/adp-real-agent-invocation.XXXXXX")
ADP_BIN="$TMP_ROOT/adp"
PROJECT_ROOT="$TMP_ROOT/project"
ADP_HOME="$TMP_ROOT/adp-home"
ADP_RUNTIME_DIR="$TMP_ROOT/runtime"
EVENTS_FILE="$ADP_HOME/logs/events.jsonl"
ADP_REAL_AGENT_TIMEOUT="${ADP_REAL_AGENT_TIMEOUT:-180s}"
ADP_REAL_CLAUDE_MAX_BUDGET_USD="${ADP_REAL_CLAUDE_MAX_BUDGET_USD:-0.20}"

cleanup() {
  rm -rf "$TMP_ROOT"
}
trap cleanup EXIT INT TERM

CODEX_COMMAND="${ADP_REAL_CODEX_BIN:-codex}"
CLAUDE_COMMAND="${ADP_REAL_CLAUDE_BIN:-claude}"
if [ "$run_codex" -eq 1 ]; then
  CODEX_COMMAND=$(resolve_command "$CODEX_COMMAND" codex)
fi
if [ "$run_claude" -eq 1 ]; then
  CLAUDE_COMMAND=$(resolve_command "$CLAUDE_COMMAND" claude)
fi
CODEX_COMMAND_YAML=$(yaml_quote "$CODEX_COMMAND")
CLAUDE_COMMAND_YAML=$(yaml_quote "$CLAUDE_COMMAND")

mkdir -p "$PROJECT_ROOT" "$ADP_HOME" "$ADP_RUNTIME_DIR"
printf 'real-agent-invocation-smoke\n' > "$PROJECT_ROOT/README.md"
printf 'package main\n\nfunc main() {}\n' > "$PROJECT_ROOT/main.go"

info "building temporary adp binary"
(cd "$REPO_ROOT" && go build -o "$ADP_BIN" ./cmd/adp)

export ADP_HOME
export ADP_RUNTIME_DIR

info "initializing temporary ADP workspace"
output=$(run_adp "$REPO_ROOT" init)
assert_contains "$output" "initialized ADP home" "init output"
output=$(run_adp "$REPO_ROOT" workspace add real-agent-smoke "$PROJECT_ROOT")
assert_contains "$output" 'workspace "real-agent-smoke" added' "workspace add output"

WORKSPACE_DIR="$ADP_HOME/workspaces/real-agent-smoke"
mkdir -p "$WORKSPACE_DIR/prompts" "$WORKSPACE_DIR/memory"
cat > "$WORKSPACE_DIR/workspace.yaml" <<EOF
version: 1

workspace:
  name: real-agent-smoke

project:
  root: $PROJECT_ROOT

prompts:
  base: prompts/base.md

memory:
  enabled: true
  shared: memory/shared.md

rules:
  evidence_scope: real-agent-invocation
  default_gate: provider-free

agents:
  codex:
    enabled: true
    command: $CODEX_COMMAND_YAML
  claude:
    enabled: true
    command: $CLAUDE_COMMAND_YAML
EOF
cat > "$WORKSPACE_DIR/prompts/base.md" <<'EOF'
# Real Agent Invocation Smoke

This run is explicit operator evidence. Keep the response bounded and do not
modify files.
EOF
cat > "$WORKSPACE_DIR/memory/shared.md" <<'EOF'
# Shared Memory

The default ADP release gate remains provider-free.
EOF

output=$(run_adp "$REPO_ROOT" workspace doctor real-agent-smoke)
assert_contains "$output" "real-agent-smoke" "workspace doctor output"
assert_contains "$output" "ok" "workspace doctor output"

info "creating task-bound phase context"
output=$(run_adp "$REPO_ROOT" phase add --workspace real-agent-smoke --goal "real agent invocation evidence" p-real-agent "Real Agent Invocation")
assert_contains "$output" "phase p-real-agent added" "phase add output"
output=$(run_adp "$REPO_ROOT" phase start --workspace real-agent-smoke p-real-agent)
assert_contains "$output" "phase p-real-agent status: active" "phase start output"
output=$(run_adp "$REPO_ROOT" tasks add --workspace real-agent-smoke --priority high --phase p-real-agent --description "Collect explicit real provider invocation evidence" "Run real agent invocation")
assert_contains "$output" "task task-" "tasks add output"
TASK_ID=$(printf '%s\n' "$output" | sed -n 's/^task \(task-[^ ]*\) added$/\1/p')
if [ -z "$TASK_ID" ]; then
  fail "could not parse task id from: $output"
fi
assert_absent_project_artifacts "$PROJECT_ROOT"

if [ "$run_codex" -eq 1 ]; then
  run_real_codex
fi
if [ "$run_claude" -eq 1 ]; then
  run_real_claude
fi

output=$(run_adp "$REPO_ROOT" events list --workspace real-agent-smoke --task "$TASK_ID")
assert_contains "$output" "run_started" "events list output"
assert_contains "$output" "run_finished" "events list output"
output=$(run_adp "$REPO_ROOT" sessions list --workspace real-agent-smoke --task "$TASK_ID")
assert_contains "$output" "real-agent-smoke" "sessions list output"
output=$(run_adp "$REPO_ROOT" progress --workspace real-agent-smoke --format json)
assert_contains "$output" '"workspace": "real-agent-smoke"' "progress json output"
assert_contains "$output" '"ready": 1' "progress json output"
output=$(run_adp "$REPO_ROOT" plan doctor --workspace real-agent-smoke --format json)
assert_contains "$output" '"status": "ok"' "plan doctor output"
assert_contains "$output" '"has_errors": false' "plan doctor output"
assert_absent_project_artifacts "$PROJECT_ROOT"

info "real agent invocation evidence passed"
