#!/usr/bin/env bash

fail() {
  printf 'runtime-audit-smoke: %s\n' "$*" >&2
  exit 1
}

info() {
  printf '[runtime-audit-smoke] %s\n' "$*"
}

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

event_log_count() {
  if [ -f "$EVENTS_FILE" ]; then
    line_count "$EVENTS_FILE"
  else
    printf '0\n'
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

assert_read_only_lease_state() {
  local label="$1"
  local tasks_before="$2"
  local phases_before="$3"
  local progress_before="$4"
  local events_before="$5"
  local runtime_before="$6"
  local project_before="$7"
  local git_before="$8"

  if [ "$(cat "$TASKS_FILE")" != "$tasks_before" ]; then fail "$label changed task state"; fi
  if [ "$(cat "$PHASES_FILE")" != "$phases_before" ]; then fail "$label changed phase state"; fi
  if [ "$(cat "$PROGRESS_FILE")" != "$progress_before" ]; then fail "$label changed progress events"; fi
  if [ "$(event_log_count)" != "$events_before" ]; then fail "$label changed event log"; fi
  if [ "$(runtime_dirs_state)" != "$runtime_before" ]; then fail "$label changed runtime dirs"; fi
  if [ "$(project_root_state)" != "$project_before" ]; then fail "$label changed project root"; fi
  if [ "$(git_state)" != "$git_before" ]; then fail "$label changed Git state"; fi
  assert_no_git_side_effects "$label"
  assert_absent_project_artifacts "$PROJECT_ROOT"
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
