#!/usr/bin/env bash
set -euo pipefail

fail() {
  printf 'runtime-context-smoke: %s\n' "$*" >&2
  exit 1
}

info() {
  printf '[runtime-context-smoke] %s\n' "$*"
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

assert_symlink() {
  local path="$1"
  if [ ! -L "$path" ]; then
    fail "missing symlink: $path"
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

assert_file_contains() {
  local path="$1"
  local needle="$2"
  local label="$3"

  assert_file "$path"
  if ! grep -F -q "$needle" "$path"; then
    printf '%s\n' "$label file:" >&2
    cat "$path" >&2
    fail "$label missing expected text: $needle"
  fi
}

line_count() {
  local path="$1"

  assert_file "$path"
  wc -l < "$path" | tr -d '[:space:]'
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

with_dangerous_git_env() (
  local boundary_root="$1"
  shift

  mkdir -p "$boundary_root"
  export GIT_DIR="$boundary_root/git-dir"
  export GIT_WORK_TREE="$boundary_root/work-tree"
  export GIT_INDEX_FILE="$boundary_root/index"
  export GIT_OBJECT_DIRECTORY="$boundary_root/objects"
  export GIT_ALTERNATE_OBJECT_DIRECTORIES="$boundary_root/alt-objects-a:$boundary_root/alt-objects-b"
  export GIT_COMMON_DIR="$boundary_root/common"
  export GIT_NAMESPACE="adp-context-smoke"
  "$@"
)

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

if ! command -v go >/dev/null 2>&1; then
  fail "Go is required to build cmd/adp"
fi

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd -- "$SCRIPT_DIR/.." && pwd)
. "$SCRIPT_DIR/runtime-context-smoke-lib.sh"
TMP_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/adp-runtime-context-smoke.XXXXXX")
ADP_BIN="$TMP_ROOT/adp"

cleanup() {
  rm -rf "$TMP_ROOT"
}
trap cleanup EXIT INT TERM

PROJECT_GIT_ROOT="$TMP_ROOT/project"
PROJECT_ROOT="$PROJECT_GIT_ROOT/app"
ADP_HOME="$TMP_ROOT/adp-home"
ADP_RUNTIME_DIR="$TMP_ROOT/runtime"
FAKE_BIN="$TMP_ROOT/bin"
WORKSPACE_DIR="$ADP_HOME/workspaces/context-a"
EVENTS_FILE="$ADP_HOME/logs/events.jsonl"

mkdir -p "$PROJECT_ROOT/pkg" "$ADP_HOME" "$ADP_RUNTIME_DIR" "$FAKE_BIN"
printf 'module example.com/adp-runtime-context-smoke\n' > "$PROJECT_ROOT/go.mod"
printf 'package main\n' > "$PROJECT_ROOT/main.go"
printf 'package pkg\n' > "$PROJECT_ROOT/pkg/pkg.go"
printf 'dist\n' > "$PROJECT_GIT_ROOT/.gitignore"
git -C "$PROJECT_GIT_ROOT" init -q
git -C "$PROJECT_GIT_ROOT" config user.name adp-smoke
git -C "$PROJECT_GIT_ROOT" config user.email adp-smoke@example.invalid
git -C "$PROJECT_GIT_ROOT" add .gitignore app
git -C "$PROJECT_GIT_ROOT" commit -q -m "init context smoke project"
write_fake_agent "$FAKE_BIN/codex" codex senior-engineer AGENTS.md .codex/config.toml toml --context-codex
write_fake_agent "$FAKE_BIN/claude" claude architect CLAUDE.md .claude/settings.json json --context-claude

info "building temporary adp binary"
(cd "$REPO_ROOT" && go build -o "$ADP_BIN" ./cmd/adp)

export ADP_HOME
export ADP_RUNTIME_DIR
export PATH="$FAKE_BIN:$PATH"
export ADP_EXPECT_ADP_HOME="$ADP_HOME"
export ADP_EXPECT_ADP_CLI="$ADP_BIN"
export ADP_EXPECT_PROJECT_ROOT="$PROJECT_ROOT"
export ADP_EXPECT_GIT_ROOT="$PROJECT_GIT_ROOT"

info "initializing workspace with marked prompt, memory, MCP, and profiles"
output=$(run_adp "$REPO_ROOT" init)
assert_contains "$output" "initialized ADP home" "init output"
output=$(run_adp "$REPO_ROOT" workspace add context-a "$PROJECT_ROOT")
assert_contains "$output" 'workspace "context-a" added' "workspace add output"

cat > "$WORKSPACE_DIR/workspace.yaml" <<EOF
version: 1

workspace:
  name: context-a

project:
  root: $PROJECT_ROOT

memory:
  enabled: true
  shared: memory/shared.md

prompts:
  base: prompts/base.md

rules:
  coding_style: strict
  review_depth: context-audit

mcp:
  enabled: true
  config: mcp/config.yaml
  servers:
    - github
    - local-tools

agents:
  codex:
    enabled: true
    profile: senior-engineer
    command: codex
  claude:
    enabled: true
    profile: architect
    command: claude
EOF
cat > "$WORKSPACE_DIR/prompts/base.md" <<'EOF'
# P35 Base Prompt

P35 base prompt marker.
EOF
cat > "$WORKSPACE_DIR/memory/shared.md" <<'EOF'
# P35 Shared Memory

P35 shared memory marker.
EOF
cat > "$WORKSPACE_DIR/mcp/config.yaml" <<'EOF'
enabled: true
marker: p35-mcp-config-marker
servers:
  github:
    command: github-mcp-server
  local-tools:
    command: local-tools-mcp
EOF
cat > "$WORKSPACE_DIR/profiles/senior-engineer.yaml" <<'EOF'
profile: senior-engineer
notes:
  - P35 senior-engineer profile marker.
EOF
cat > "$WORKSPACE_DIR/profiles/architect.yaml" <<'EOF'
profile: architect
notes:
  - P35 architect profile marker.
EOF

info "checking workspace diagnostics before runtime launch"
output=$(run_adp "$REPO_ROOT" workspace doctor context-a)
assert_contains "$output" "context-a" "workspace doctor output"
assert_contains "$output" "ok" "workspace doctor output"
output=$(run_adp "$REPO_ROOT" workspace doctor context-a --format json)
assert_contains "$output" "\"git\": {" "workspace doctor git json output"
assert_contains "$output" "\"project_root\": \"$PROJECT_ROOT\"" "workspace doctor git json output"
assert_contains "$output" "\"git_root\": \"$PROJECT_GIT_ROOT\"" "workspace doctor git json output"
assert_contains "$output" "\"project_below_root\": true" "workspace doctor git json output"
assert_contains "$output" "\"relative_project_dir\": \"app\"" "workspace doctor git json output"
assert_contains "$output" "\"change_state\": \"clean\"" "workspace doctor git json output"
output=$(run_adp "$REPO_ROOT" workspace show context-a)
assert_contains "$output" "memory_enabled: true" "workspace show output"
assert_contains "$output" "mcp_enabled: true" "workspace show output"

info "creating task-bound phase context"
output=$(run_adp "$REPO_ROOT" phase add --workspace context-a --goal "runtime context audit" p-context "Runtime Context")
assert_contains "$output" "phase p-context added" "phase add output"
output=$(run_adp "$REPO_ROOT" phase start --workspace context-a p-context)
assert_contains "$output" "phase p-context status: active" "phase start output"
output=$(run_adp "$REPO_ROOT" tasks add --workspace context-a --priority critical --phase p-context --description "Verify generated context surface" "Audit runtime context")
assert_contains "$output" "task task-" "tasks add output"
TASK_ID=$(printf '%s\n' "$output" | sed -n 's/^task \(task-[^ ]*\) added$/\1/p')
if [ -z "$TASK_ID" ]; then
  fail "could not parse task id from: $output"
fi
export ADP_EXPECT_TASK_ID="$TASK_ID"
export ADP_EXPECT_TASK_TITLE="Audit runtime context"
export ADP_EXPECT_TASK_STATUS="ready"
export ADP_EXPECT_TASK_PRIORITY="critical"
export ADP_EXPECT_TASK_PHASE="p-context"
export ADP_EXPECT_TASK_DESCRIPTION="Verify generated context surface"

assert_file "$WORKSPACE_DIR/planning/tasks.yaml"
assert_file "$WORKSPACE_DIR/planning/phases.yaml"
assert_file "$WORKSPACE_DIR/planning/progress.jsonl"
assert_absent_project_artifacts "$PROJECT_ROOT"
assert_absent_project_artifacts "$PROJECT_GIT_ROOT"

info "running fake Codex and verifying generated context"
output=$(with_dangerous_git_env "$TMP_ROOT/git-boundary-env" run_adp "$REPO_ROOT" run codex --workspace context-a --task "$TASK_ID" -- --context-codex)
assert_contains "$output" "fake-codex" "codex output"
assert_contains "$output" "--context-codex" "codex output"
assert_absent_project_artifacts "$PROJECT_ROOT"
assert_absent_project_artifacts "$PROJECT_GIT_ROOT"

info "running fake Claude and verifying generated context"
output=$(with_dangerous_git_env "$TMP_ROOT/git-boundary-env" run_adp "$PROJECT_ROOT" run claude --task "$TASK_ID" -- --context-claude)
assert_contains "$output" "fake-claude" "claude output"
assert_contains "$output" "--context-claude" "claude output"
assert_absent_project_artifacts "$PROJECT_ROOT"
assert_absent_project_artifacts "$PROJECT_GIT_ROOT"
if [ -n "$(git -C "$PROJECT_GIT_ROOT" status --short)" ]; then
  git -C "$PROJECT_GIT_ROOT" status --short >&2
  fail "runtime launches changed Git worktree state"
fi
output=$(run_adp "$REPO_ROOT" tasks show --workspace context-a "$TASK_ID")
assert_contains "$output" "status: ready" "task-bound run task state"
assert_contains "$output" "owner: -" "task-bound run task state"

if [ "$(line_count "$EVENTS_FILE")" != "4" ]; then
  cat "$EVENTS_FILE" >&2
  fail "event log should contain four runtime events"
fi

codex_session=$(session_id_by_agent "$EVENTS_FILE" codex)
claude_session=$(session_id_by_agent "$EVENTS_FILE" claude)
if [ -z "$codex_session" ] || [ -z "$claude_session" ]; then
  cat "$EVENTS_FILE" >&2
  fail "missing codex or claude session id in event log"
fi

info "checking local evidence and diagnostics after runtime launch"
output=$(run_adp "$REPO_ROOT" events list --workspace context-a --task "$TASK_ID" --limit 4)
assert_contains "$output" "run_started" "events list output"
assert_contains "$output" "run_finished" "events list output"
assert_contains "$output" "codex" "events list output"
assert_contains "$output" "claude" "events list output"
assert_contains "$output" "$TASK_ID" "events list output"

output=$(run_adp "$REPO_ROOT" sessions list --workspace context-a --task "$TASK_ID")
assert_contains "$output" "$codex_session" "sessions list output"
assert_contains "$output" "$claude_session" "sessions list output"
assert_contains "$output" "context-a" "sessions list output"

output=$(run_adp "$REPO_ROOT" sessions show "$codex_session")
assert_contains "$output" "profile: senior-engineer" "codex session output"
assert_contains "$output" "task_id: $TASK_ID" "codex session output"

output=$(run_adp "$REPO_ROOT" sessions show "$claude_session")
assert_contains "$output" "profile: architect" "claude session output"
assert_contains "$output" "task_id: $TASK_ID" "claude session output"

events_before_restore=$(line_count "$EVENTS_FILE")
output=$(run_adp "$REPO_ROOT" sessions restore-plan "$codex_session")
assert_contains "$output" "status: ready" "codex restore-plan output"
assert_contains "$output" "adp run codex --workspace context-a --profile senior-engineer --task $TASK_ID" "codex restore-plan output"
assert_contains "$output" "-- --context-codex" "codex restore-plan output"

output=$(run_adp "$REPO_ROOT" sessions restore-plan "$claude_session")
assert_contains "$output" "status: ready" "claude restore-plan output"
assert_contains "$output" "adp run claude --workspace context-a --profile architect --task $TASK_ID" "claude restore-plan output"
assert_contains "$output" "-- --context-claude" "claude restore-plan output"
events_after_restore=$(line_count "$EVENTS_FILE")
if [ "$events_after_restore" != "$events_before_restore" ]; then
  cat "$EVENTS_FILE" >&2
  fail "sessions restore-plan appended events"
fi

output=$(run_adp "$REPO_ROOT" progress --workspace context-a --format json)
assert_contains "$output" '"workspace": "context-a"' "progress json output"
assert_contains "$output" '"ready": 1' "progress json output"
assert_contains "$output" '"priority": "critical"' "progress json output"

output=$(run_adp "$REPO_ROOT" plan doctor --workspace context-a --format json)
assert_contains "$output" '"workspace": "context-a"' "plan doctor json output"
assert_contains "$output" '"status": "ok"' "plan doctor json output"
assert_contains "$output" '"has_errors": false' "plan doctor json output"
assert_absent_project_artifacts "$PROJECT_ROOT"

info "running fake Codex with atomic task take"
output=$(run_adp "$REPO_ROOT" tasks update --workspace context-a "$TASK_ID" --status review)
assert_contains "$output" "status: review" "context task review output"
output=$(run_adp "$REPO_ROOT" tasks add --workspace context-a --priority critical --phase p-context --description "Verify generated take context surface" "Audit runtime take context")
assert_contains "$output" "task task-" "take task add output"
TAKE_TASK_ID=$(printf '%s\n' "$output" | sed -n 's/^task \(task-[^ ]*\) added$/\1/p')
if [ -z "$TAKE_TASK_ID" ]; then
  fail "could not parse take task id from: $output"
fi
export ADP_EXPECT_TASK_ID="$TAKE_TASK_ID"
export ADP_EXPECT_TASK_TITLE="Audit runtime take context"
export ADP_EXPECT_TASK_STATUS="in_progress"
export ADP_EXPECT_TASK_PRIORITY="critical"
export ADP_EXPECT_TASK_DESCRIPTION="Verify generated take context surface"
phases_before_take=$(cat "$WORKSPACE_DIR/planning/phases.yaml")
progress_events_before_take=$(line_count "$WORKSPACE_DIR/planning/progress.jsonl")
events_before_take=$(line_count "$EVENTS_FILE")
output=$(with_dangerous_git_env "$TMP_ROOT/git-boundary-env" run_adp "$REPO_ROOT" run codex --workspace context-a --take --owner context-run-take --lease 25m -- --context-codex)
assert_contains "$output" "fake-codex" "codex take output"
assert_contains "$output" "--context-codex" "codex take output"
take_session=$(session_id_by_agent "$EVENTS_FILE" codex)
if [ "$(line_count "$EVENTS_FILE")" != $((events_before_take + 2)) ]; then
  cat "$EVENTS_FILE" >&2
  fail "run --take should append two runtime events"
fi
if [ "$(line_count "$WORKSPACE_DIR/planning/progress.jsonl")" != $((progress_events_before_take + 1)) ]; then
  cat "$WORKSPACE_DIR/planning/progress.jsonl" >&2
  fail "run --take should append one task claim progress event"
fi
take_progress=$(tail -n 1 "$WORKSPACE_DIR/planning/progress.jsonl")
assert_contains "$take_progress" "\"type\":\"task_claimed\"" "run take progress event"
assert_contains "$take_progress" "\"task_id\":\"$TAKE_TASK_ID\"" "run take progress event"
assert_contains "$take_progress" "\"owner\":\"context-run-take\"" "run take progress event"
assert_contains "$(cat "$EVENTS_FILE")" "\"task_binding\":\"take\"" "run take event metadata"
assert_contains "$(cat "$EVENTS_FILE")" "\"owner\":\"context-run-take\"" "run take event metadata"
output=$(run_adp "$REPO_ROOT" tasks show --workspace context-a "$TAKE_TASK_ID")
assert_contains "$output" "status: in_progress" "take task show output"
assert_contains "$output" "owner: context-run-take" "take task show output"
assert_contains "$output" "lease_expires_at: 20" "take task show output"
output=$(run_adp "$REPO_ROOT" sessions show "$take_session")
assert_contains "$output" "task_id: $TAKE_TASK_ID" "take session output"
assert_contains "$output" "agent: codex" "take session output"
assert_contains "$output" "run_finished" "take session output"
if [ "$(cat "$WORKSPACE_DIR/planning/phases.yaml")" != "$phases_before_take" ]; then
  fail "run --take changed phase evidence"
fi
assert_absent_project_artifacts "$PROJECT_ROOT"
assert_absent_project_artifacts "$PROJECT_GIT_ROOT"
if [ -n "$(git -C "$PROJECT_GIT_ROOT" status --short)" ]; then
  git -C "$PROJECT_GIT_ROOT" status --short >&2
  fail "run --take changed Git worktree state"
fi

info "runtime context smoke passed"
