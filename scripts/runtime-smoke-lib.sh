#!/usr/bin/env bash

fail() {
  printf 'runtime-smoke: %s\n' "$*" >&2
  exit 1
}

info() {
  printf '[runtime-smoke] %s\n' "$*"
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

assert_not_contains() {
  local output="$1"
  local needle="$2"
  local label="$3"

  case "$output" in
    *"$needle"*)
      printf '%s\n' "$output" >&2
      fail "$label contained unexpected text: $needle"
      ;;
  esac
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
  export GIT_NAMESPACE="adp-smoke-namespace"
  "$@"
)

assert_file() {
  local path="$1"
  if [ ! -f "$path" ]; then
    fail "missing file: $path"
  fi
}

assert_file_unchanged() {
  local before="$1"
  local after="$2"
  local label="$3"

  assert_file "$before"
  assert_file "$after"
  if ! cmp -s "$before" "$after"; then
    printf '%s\n' "$label changed:" >&2
    diff -u "$before" "$after" >&2 || true
    fail "$label changed"
  fi
}

assert_symlink() {
  local path="$1"
  if [ ! -L "$path" ]; then
    fail "missing symlink: $path"
  fi
}

assert_absent_path() {
  local path="$1"
  if [ -e "$path" ] || [ -L "$path" ]; then
    fail "path should be absent: $path"
  fi
}

snapshot_tree_entries() {
  local root="$1"
  local output="$2"

  (cd "$root" && find . -mindepth 1 -print | LC_ALL=C sort) > "$output"
}

assert_tree_entries_unchanged() {
  local root="$1"
  local before="$2"
  local label="$3"
  local after="${before}.after"

  snapshot_tree_entries "$root" "$after"
  if ! cmp -s "$before" "$after"; then
    printf '%s\n' "$label changed:" >&2
    diff -u "$before" "$after" >&2 || true
    fail "$label changed"
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

assert_runtime_entries() {
  local runtime_dir="$1"
  local want="$2"
  local got

  got=$(runtime_entry_count "$runtime_dir")
  if [ "$got" != "$want" ]; then
    fail "runtime dir entry count is $got, expected $want"
  fi
}

runtime_entry_count() {
  local runtime_dir="$1"

  find "$runtime_dir" -mindepth 1 -maxdepth 1 -type d | wc -l | tr -d '[:space:]'
}

assert_line_count() {
  local path="$1"
  local want="$2"
  local got

  assert_file "$path"
  got=$(wc -l < "$path" | tr -d '[:space:]')
  if [ "$got" != "$want" ]; then
    printf '%s\n' "event log:" >&2
    cat "$path" >&2
    fail "$path line count is $got, expected $want"
  fi
}

line_count() {
  local path="$1"

  assert_file "$path"
  wc -l < "$path" | tr -d '[:space:]'
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
    fail "adp $* succeeded, expected failure"
  fi
  printf '%s\n' "$output"
}

write_fake_agent() {
  local path="$1"
  local agent="$2"
  local instructions="$3"
  local config="$4"
  local linked="$5"

  cat > "$path" <<EOF
#!/usr/bin/env sh
set -eu

printf 'fake-$agent cwd=%s args=%s\n' "\$(pwd)" "\$*"

assert_git_env_unset() {
  name=\$1
  if env | grep -q "^\$name="; then
    value=\$(env | sed -n "s/^\$name=//p" | head -n 1)
    printf '%s leaked into fake-$agent environment: %s\n' "\$name" "\$value" >&2
    exit 96
  fi
}

assert_git_env_unset GIT_ALTERNATE_OBJECT_DIRECTORIES
assert_git_env_unset GIT_COMMON_DIR
assert_git_env_unset GIT_DIR
assert_git_env_unset GIT_INDEX_FILE
assert_git_env_unset GIT_NAMESPACE
assert_git_env_unset GIT_OBJECT_DIRECTORY
assert_git_env_unset GIT_WORK_TREE

test "\${ADP_WORKSPACE:-}" = "game-a"
test "\${ADP_PROJECT_ROOT:-}" = "\$ADP_EXPECT_PROJECT_ROOT"
test "\${ADP_GIT_ROOT:-}" = "\$ADP_EXPECT_PROJECT_ROOT"
test -n "\${ADP_SESSION_ID:-}"
test -n "\${ADP_RUNTIME_ROOT:-}"
test "\$(pwd)" = "\$ADP_RUNTIME_ROOT"
case ":\${GIT_CEILING_DIRECTORIES:-}:" in
  *":\$ADP_RUNTIME_ROOT:"*) ;;
  *)
    printf 'GIT_CEILING_DIRECTORIES missing runtime root: %s\n' "\${GIT_CEILING_DIRECTORIES:-}" >&2
    exit 96
    ;;
esac
test -f "\$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
grep -F -q "version: 1" "\$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
grep -F -q "git_root: \$ADP_EXPECT_PROJECT_ROOT" "\$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
grep -F -q "git_metadata_skipped: true" "\$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
grep -F -q "runtime_root: \$ADP_RUNTIME_ROOT" "\$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
grep -F -q "generated_by: adp" "\$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
test ! -e "\$ADP_RUNTIME_ROOT/.git"
if git -C "\$ADP_RUNTIME_ROOT" status --short --branch >/dev/null 2>&1; then
  printf 'git status unexpectedly succeeded inside ADP runtime root\n' >&2
  exit 96
fi
git -C "\$ADP_PROJECT_ROOT" status --short --branch >/dev/null
test -f "$instructions"
grep -F -q "## Git Boundary" "$instructions"
grep -F -q 'git -C "\$ADP_PROJECT_ROOT" status --short --branch' "$instructions"
test -f "$config"
test -L "$linked"
test -f "$linked"
if [ -n "\${ADP_EXPECT_TASK_ID:-}" ]; then
  test "\${ADP_TASK_ID:-}" = "\$ADP_EXPECT_TASK_ID"
  test "\${ADP_TASK_TITLE:-}" = "Bind runtime session to task"
  grep -q "\$ADP_EXPECT_TASK_ID" "$instructions"
  grep -q "Bind runtime session to task" "$instructions"
  grep -q "task_id: \$ADP_EXPECT_TASK_ID" "\$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
fi
test "\$#" -eq 2
test "\$1" = "--probe"
test "\$2" = "$agent-payload"
EOF
  chmod 755 "$path"
}

first_line() {
  printf '%s\n' "$1" | sed -n '1p'
}

run_real_cli_smoke() {
  local label="$1"
  local gate_var="$2"
  local bin="$3"
  local output

  if [ "${!gate_var:-}" != "1" ]; then
    fail "real $label smoke requires $gate_var=1"
  fi
  if ! command -v "$bin" >/dev/null 2>&1; then
    fail "real $label smoke requested, but command is not available: $bin"
  fi

  if output=$("$bin" --version 2>&1); then
    info "real $label CLI responded to --version: $(first_line "$output")"
    return
  fi
  if output=$("$bin" --help 2>&1); then
    info "real $label CLI responded to --help: $(first_line "$output")"
    return
  fi

  printf '%s\n' "$output" >&2
  fail "real $label CLI did not complete --version or --help"
}
