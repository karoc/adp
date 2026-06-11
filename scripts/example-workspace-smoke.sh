#!/usr/bin/env bash
set -euo pipefail

fail() {
  printf 'example-workspace-smoke: %s\n' "$*" >&2
  exit 1
}

info() {
  printf '[example-workspace-smoke] %s\n' "$*"
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

line_count() {
  local path="$1"

  assert_file "$path"
  wc -l < "$path" | tr -d '[:space:]'
}

assert_line_count() {
  local path="$1"
  local want="$2"
  local got

  got=$(line_count "$path")
  if [ "$got" != "$want" ]; then
    printf '%s\n' "event log:" >&2
    cat "$path" >&2
    fail "$path line count is $got, expected $want"
  fi
}

parse_export() {
  local output="$1"
  local name="$2"
  local value

  value=$(printf '%s\n' "$output" | sed -n "s/^export ${name}='\\(.*\\)'$/\\1/p" | head -n 1)
  if [ -z "$value" ]; then
    printf '%s\n' "$output" >&2
    fail "export $name not found"
  fi
  printf '%s\n' "$value"
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

init_project_git() {
  if ! command -v git >/dev/null 2>&1; then
    fail "Git is required for example workspace smoke"
  fi
  git -C "$PROJECT_ROOT" init -q
  git -C "$PROJECT_ROOT" config user.name adp-smoke
  git -C "$PROJECT_ROOT" config user.email adp-smoke@example.invalid
  git -C "$PROJECT_ROOT" add go.mod main.go
  git -C "$PROJECT_ROOT" commit -q -m "init example workspace project"
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

rewrite_project_root() {
  local workspace_yaml="$1"
  local project_root="$2"
  local tmp_file="${workspace_yaml}.tmp"

  awk -v project_root="$project_root" '
    /^project:[[:space:]]*$/ {
      in_project = 1
      print
      next
    }
    in_project && /^[[:space:]]*root:[[:space:]]*/ {
      print "  root: " project_root
      in_project = 0
      next
    }
    /^[^[:space:]]/ {
      in_project = 0
    }
    {
      print
    }
  ' "$workspace_yaml" > "$tmp_file"
  mv "$tmp_file" "$workspace_yaml"
}

write_fake_codex() {
  local path="$1"

  cat > "$path" <<'EOF'
#!/usr/bin/env sh
set -eu

printf 'fake-codex cwd=%s args=%s\n' "$(pwd)" "$*"

test "${ADP_WORKSPACE:-}" = "game-a"
test -n "${ADP_SESSION_ID:-}"
test -n "${ADP_RUNTIME_ROOT:-}"
test "$(pwd)" = "$ADP_RUNTIME_ROOT"
test -f "$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
test -f "$ADP_RUNTIME_ROOT/AGENTS.md"
test -L "$ADP_RUNTIME_ROOT/go.mod"
test -f "$ADP_RUNTIME_ROOT/go.mod"
test "$#" -eq 1
test "$1" = "--example-smoke"
EOF
  chmod 755 "$path"
}

write_fake_claude() {
  local path="$1"

  cat > "$path" <<'EOF'
#!/usr/bin/env sh
set -eu

printf 'fake-claude cwd=%s args=%s\n' "$(pwd)" "$*"
EOF
  chmod 755 "$path"
}

if ! command -v go >/dev/null 2>&1; then
  fail "Go is required to build cmd/adp"
fi

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd -- "$SCRIPT_DIR/.." && pwd)
TMP_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/adp-example-workspace-smoke.XXXXXX")
ADP_BIN="$TMP_ROOT/adp"

cleanup() {
  rm -rf "$TMP_ROOT"
}
trap cleanup EXIT INT TERM

PROJECT_ROOT="$TMP_ROOT/project"
ADP_HOME="$TMP_ROOT/adp-home"
ADP_RUNTIME_DIR="$TMP_ROOT/runtime"
FAKE_BIN="$TMP_ROOT/bin"
WORKSPACE_DIR="$ADP_HOME/workspaces/game-a"
EVENTS_FILE="$ADP_HOME/logs/events.jsonl"
before_restore_lines=""
after_restore_lines=""

mkdir -p "$PROJECT_ROOT" "$ADP_HOME/workspaces" "$ADP_RUNTIME_DIR" "$FAKE_BIN"
printf 'module example.com/adp-example-smoke\n' > "$PROJECT_ROOT/go.mod"
printf 'package main\n' > "$PROJECT_ROOT/main.go"
init_project_git
write_fake_codex "$FAKE_BIN/codex"
write_fake_claude "$FAKE_BIN/claude"

info "building temporary adp binary"
(cd "$REPO_ROOT" && go build -o "$ADP_BIN" ./cmd/adp)

info "copying basic workspace example"
cp -R "$REPO_ROOT/examples/basic-workspace" "$WORKSPACE_DIR"
rewrite_project_root "$WORKSPACE_DIR/workspace.yaml" "$PROJECT_ROOT"

export ADP_HOME
export ADP_RUNTIME_DIR
export PATH="$FAKE_BIN:$PATH"

info "initializing ADP home"
output=$(run_adp "$REPO_ROOT" init)
assert_contains "$output" "initialized ADP home" "init output"

info "validating copied workspace"
output=$(run_adp "$REPO_ROOT" workspace doctor game-a)
assert_contains "$output" "game-a" "workspace doctor output"
assert_contains "$output" "ok" "workspace doctor output"
assert_contains "$output" "no issues" "workspace doctor output"
output=$(run_adp "$REPO_ROOT" workspace doctor game-a --verbose)
assert_contains "$output" "workspace.git.root.detected" "workspace doctor verbose output"
output=$(run_adp "$REPO_ROOT" workspace doctor game-a --format json)
assert_contains "$output" '"code": "workspace.git.root.detected"' "workspace doctor json output"

output=$(run_adp "$REPO_ROOT" workspace show game-a)
assert_contains "$output" "name: game-a" "workspace show output"
assert_contains "$output" "project_root: $PROJECT_ROOT" "workspace show output"

info "building runtime env from copied workspace"
env_output=$(run_adp "$REPO_ROOT" env game-a --cd)
runtime_root=$(parse_export "$env_output" ADP_RUNTIME_ROOT)
assert_contains "$env_output" "cd '$runtime_root'" "env --cd output"
assert_file "$runtime_root/.adp-runtime.yaml"
assert_symlink "$runtime_root/go.mod"
assert_symlink "$runtime_root/main.go"
assert_absent_project_artifacts "$PROJECT_ROOT"

info "running fake codex through copied workspace"
codex_output=$(run_adp "$REPO_ROOT" run codex --workspace game-a -- --example-smoke)
assert_contains "$codex_output" "fake-codex" "codex run output"
assert_contains "$codex_output" "--example-smoke" "codex run output"
assert_absent_project_artifacts "$PROJECT_ROOT"
assert_line_count "$EVENTS_FILE" 2

codex_session=$(session_id_by_agent "$EVENTS_FILE" codex)
if [ -z "$codex_session" ]; then
  cat "$EVENTS_FILE" >&2
  fail "codex session id missing in event log"
fi

info "inspecting copied workspace events and sessions"
events_output=$(run_adp "$REPO_ROOT" events list --workspace game-a --session "$codex_session" --limit 2)
assert_contains "$events_output" "$codex_session" "events list output"
assert_contains "$events_output" "run_started" "events list output"
assert_contains "$events_output" "run_finished" "events list output"
assert_contains "$events_output" "codex" "events list output"

sessions_output=$(run_adp "$REPO_ROOT" sessions list --workspace game-a --agent codex)
assert_contains "$sessions_output" "$codex_session" "sessions list output"
assert_contains "$sessions_output" "game-a" "sessions list output"
assert_contains "$sessions_output" "codex" "sessions list output"

session_output=$(run_adp "$REPO_ROOT" sessions show "$codex_session")
assert_contains "$session_output" "session_id: $codex_session" "sessions show output"
assert_contains "$session_output" "workspace: game-a" "sessions show output"
assert_contains "$session_output" "agent: codex" "sessions show output"
assert_contains "$session_output" "run_started" "sessions show output"
assert_contains "$session_output" "run_finished" "sessions show output"

before_restore_lines=$(line_count "$EVENTS_FILE")
restore_plan_output=$(run_adp "$REPO_ROOT" sessions restore-plan "$codex_session")
assert_contains "$restore_plan_output" "session_id: $codex_session" "sessions restore-plan output"
assert_contains "$restore_plan_output" "status: ready" "sessions restore-plan output"
assert_contains "$restore_plan_output" "suggested_command:" "sessions restore-plan output"
assert_contains "$restore_plan_output" "adp run codex --workspace game-a" "sessions restore-plan output"
assert_contains "$restore_plan_output" "-- --example-smoke" "sessions restore-plan output"
assert_contains "$restore_plan_output" "missing_fields: -" "sessions restore-plan output"
after_restore_lines=$(line_count "$EVENTS_FILE")
if [ "$after_restore_lines" != "$before_restore_lines" ]; then
  cat "$EVENTS_FILE" >&2
  fail "sessions restore-plan appended events: before=$before_restore_lines after=$after_restore_lines"
fi
assert_absent_project_artifacts "$PROJECT_ROOT"

info "example workspace smoke passed"
