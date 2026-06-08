#!/usr/bin/env bash

fail() {
  printf 'task-manager-smoke: %s\n' "$*" >&2
  exit 1
}

info() {
  printf '[task-manager-smoke] %s\n' "$*"
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

assert_contains_any() {
  local output="$1"
  local label="$2"
  shift 2
  local needle

  for needle in "$@"; do
    case "$output" in
      *"$needle"*) return 0 ;;
    esac
  done

  printf '%s\n' "$output" >&2
  fail "$label missing any expected text: $*"
}

assert_starts_with() {
  local output="$1"
  local prefix="$2"
  local label="$3"

  case "$output" in
    "$prefix"*) ;;
    *)
      printf '%s\n' "$output" >&2
      fail "$label did not start with expected text: $prefix"
      ;;
  esac
}

assert_json_field() {
  local output="$1"
  local field="$2"
  local label="$3"

  case "$output" in
    *'{'* | *'['*) ;;
    *)
      printf '%s\n' "$output" >&2
      fail "$label did not look like JSON"
      ;;
  esac
  assert_contains "$output" "\"$field\"" "$label"
}

assert_file() {
  local path="$1"
  if [ ! -f "$path" ]; then
    fail "missing file: $path"
  fi
}

assert_project_root_clean() {
  local rel

  for rel in AGENTS.md CLAUDE.md .codex .claude .adp-runtime.yaml planning tasks.yaml phases.yaml progress.jsonl; do
    if [ -e "$PROJECT_ROOT/$rel" ] || [ -L "$PROJECT_ROOT/$rel" ]; then
      fail "project root was polluted with $rel"
    fi
  done
}

assert_planning_state_unchanged() {
  local tasks_before="$1"
  local phases_before="$2"
  local progress_before="$3"
  local label="$4"
  local tasks_after
  local phases_after
  local progress_after

  tasks_after=$(cat "$TASKS_FILE")
  phases_after=$(cat "$PHASES_FILE")
  progress_after=$(cat "$PROGRESS_FILE")

  if [ "$tasks_after" != "$tasks_before" ]; then
    fail "$label changed task state"
  fi
  if [ "$phases_after" != "$phases_before" ]; then
    fail "$label changed phase state"
  fi
  if [ "$progress_after" != "$progress_before" ]; then
    fail "$label changed progress events"
  fi
}

line_count() {
  local path="$1"

  assert_file "$path"
  wc -l < "$path" | tr -d '[:space:]'
}

assert_event_log_line_count_unchanged() {
  local before="$1"
  local label="$2"
  local after

  after=$(line_count "$EVENTS_FILE")
  if [ "$after" != "$before" ]; then
    printf '%s\n' "event log:" >&2
    cat "$EVENTS_FILE" >&2
    fail "$label changed event log line count: before=$before after=$after"
  fi
}

runtime_dirs_state() {
  find "$ADP_RUNTIME_DIR" -mindepth 1 -maxdepth 1 -type d -print | LC_ALL=C sort
}

project_root_state() {
  find "$PROJECT_ROOT" -mindepth 1 -maxdepth 4 -print | LC_ALL=C sort
}

git_state() {
  git -C "$REPO_ROOT" status --short --branch --untracked-files=all
  git -C "$REPO_ROOT" rev-parse --verify HEAD
}

assert_text_unchanged() {
  local before="$1"
  local after="$2"
  local label="$3"
  local name="$4"

  if [ "$after" != "$before" ]; then
    printf '%s\n' "$name before:" >&2
    printf '%s\n' "$before" >&2
    printf '%s\n' "$name after:" >&2
    printf '%s\n' "$after" >&2
    fail "$label changed $name"
  fi
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

run_adp_expect_fail() {
  local dir="$1"
  shift
  local output

  if output=$(cd "$dir" && "$ADP_BIN" "$@" 2>&1); then
    printf '%s\n' "$output" >&2
    fail "adp $* succeeded unexpectedly"
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

assert_progress_report_json() {
  local output="$1"
  local done_task="$2"
  local critical_task="$3"
  local low_task="$4"
  local session="$5"
  local label="$6"
  local output_file="$TMP_ROOT/progress-report.json"
  local validation

  printf '%s\n' "$output" > "$output_file"
  if ! validation=$(go run "$JSON_REPORT_ASSERT" "$output_file" "$done_task" "$critical_task" "$low_task" "$session" "$ADP_RUNTIME_DIR" 2>&1); then
    printf '%s\n' "$output" >&2
    printf '%s\n' "$validation" >&2
    fail "$label failed JSON handoff contract"
  fi
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
test -f "$ADP_RUNTIME_ROOT/.codex/config.toml"
test -L "$ADP_RUNTIME_ROOT/go.mod"
test -f "$ADP_RUNTIME_ROOT/go.mod"
test "${ADP_TASK_ID:-}" = "$ADP_EXPECT_TASK_ID"
test "${ADP_TASK_TITLE:-}" = "Add task manager"
grep -F -q "$ADP_EXPECT_TASK_ID" "$ADP_RUNTIME_ROOT/AGENTS.md"
grep -F -q "Add task manager" "$ADP_RUNTIME_ROOT/AGENTS.md"
grep -F -q "$ADP_EXPECT_TASK_ID" "$ADP_RUNTIME_ROOT/.codex/config.toml"
test "$#" -eq 1
test "$1" = "--report-smoke"
EOF
  chmod 755 "$path"
}
