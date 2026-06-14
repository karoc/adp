#!/usr/bin/env bash

run_fake_workspace_lifecycle_checks() {
  local lifecycle_project_root="$smoke_root/lifecycle-project"
  local lifecycle_sentinel="$lifecycle_project_root/sentinel.txt"
  local old_workspace_dir="$adp_home/workspaces/game-lifecycle-a"
  local new_workspace_dir="$adp_home/workspaces/game-lifecycle-b"
  local lifecycle_project_snapshot="$smoke_root/lifecycle-project-before.txt"
  local output missing_output workspace_values
  local lifecycle_runtime_entries

  info "fake smoke: workspace rename and remove lifecycle"
  mkdir -p "$lifecycle_project_root"
  printf 'keep lifecycle project root\n' > "$lifecycle_sentinel"
  printf 'module example.com/adp-lifecycle-smoke\n' > "$lifecycle_project_root/go.mod"
  printf 'package main\n' > "$lifecycle_project_root/main.go"
  snapshot_tree_entries "$lifecycle_project_root" "$lifecycle_project_snapshot"
  lifecycle_runtime_entries=$(runtime_entry_count "$runtime_dir")

  output=$(run_adp "$REPO_ROOT" workspace add game-lifecycle-a "$lifecycle_project_root")
  assert_contains "$output" 'workspace "game-lifecycle-a" added' "lifecycle workspace add output"
  assert_file "$old_workspace_dir/workspace.yaml"
  assert_file "$lifecycle_sentinel"
  assert_absent_project_artifacts "$lifecycle_project_root"
  assert_tree_entries_unchanged "$lifecycle_project_root" "$lifecycle_project_snapshot" "lifecycle project root after add"
  assert_runtime_entries "$runtime_dir" "$lifecycle_runtime_entries"

  output=$(run_adp "$REPO_ROOT" workspace rename game-lifecycle-a game-lifecycle-b)
  assert_contains "$output" 'workspace "game-lifecycle-a" renamed to "game-lifecycle-b"' "workspace rename output"
  assert_absent_path "$old_workspace_dir"
  assert_file "$new_workspace_dir/workspace.yaml"
  assert_file "$lifecycle_sentinel"
  assert_absent_project_artifacts "$lifecycle_project_root"
  assert_tree_entries_unchanged "$lifecycle_project_root" "$lifecycle_project_snapshot" "lifecycle project root after rename"
  assert_runtime_entries "$runtime_dir" "$lifecycle_runtime_entries"

  output=$(run_adp "$REPO_ROOT" workspace show game-lifecycle-b)
  assert_contains "$output" "name: game-lifecycle-b" "renamed workspace show output"
  assert_contains "$output" "project_root: $lifecycle_project_root" "renamed workspace show output"
  assert_contains "$output" "workspace_dir: $new_workspace_dir" "renamed workspace show output"

  missing_output=$(run_adp_expect_fail "$REPO_ROOT" workspace show game-lifecycle-a)
  assert_contains "$missing_output" "workspace not found" "old workspace show output"

  workspace_values=$(run_adp "$REPO_ROOT" completion values workspaces)
  assert_contains "$workspace_values" "game-lifecycle-b" "workspace values after rename"
  assert_not_contains "$workspace_values" "game-lifecycle-a" "workspace values after rename"

  output=$(run_adp "$REPO_ROOT" workspace remove game-lifecycle-b --yes)
  assert_contains "$output" 'workspace "game-lifecycle-b" removed' "workspace remove output"
  assert_absent_path "$new_workspace_dir"

  assert_file "$lifecycle_sentinel"
  assert_file "$lifecycle_project_root/go.mod"
  assert_file "$lifecycle_project_root/main.go"
  assert_absent_project_artifacts "$lifecycle_project_root"
  assert_tree_entries_unchanged "$lifecycle_project_root" "$lifecycle_project_snapshot" "lifecycle project root after remove"
  assert_runtime_entries "$runtime_dir" "$lifecycle_runtime_entries"

  missing_output=$(run_adp_expect_fail "$REPO_ROOT" workspace show game-lifecycle-b)
  assert_contains "$missing_output" "workspace not found" "removed workspace show output"

  output=$(run_adp "$REPO_ROOT" workspace list)
  assert_contains "$output" "game-a" "workspace list after lifecycle output"
  assert_not_contains "$output" "game-lifecycle-a" "workspace list after lifecycle output"
  assert_not_contains "$output" "game-lifecycle-b" "workspace list after lifecycle output"

  workspace_values=$(run_adp "$REPO_ROOT" completion values workspaces)
  assert_contains "$workspace_values" "game-a" "workspace values after remove"
  assert_not_contains "$workspace_values" "game-lifecycle-a" "workspace values after remove"
  assert_not_contains "$workspace_values" "game-lifecycle-b" "workspace values after remove"
}

write_fake_enter_shell() {
  local path="$1"
  local expected_project_root="$2"

  cat > "$path" <<EOF
#!/usr/bin/env sh
set -eu

assert_git_env_unset() {
  name=\$1
  if env | grep -q "^\$name="; then
    value=\$(env | sed -n "s/^\$name=//p" | head -n 1)
    printf '%s leaked into fake enter shell environment: %s\n' "\$name" "\$value" >&2
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

test -n "\${ADP_ENTER_PROBE_OUT:-}"
{
  printf 'workspace=%s\n' "\${ADP_WORKSPACE:-}"
  printf 'runtime=%s\n' "\${ADP_RUNTIME_ROOT:-}"
  printf 'project=%s\n' "\${ADP_PROJECT_ROOT:-}"
  printf 'cwd=%s\n' "\$(pwd)"
} > "\$ADP_ENTER_PROBE_OUT"

test "\$(pwd)" = "\${ADP_RUNTIME_ROOT:-}"
test "\${ADP_WORKSPACE:-}" = "game-a"
test "\${ADP_PROJECT_ROOT:-}" = "$expected_project_root"
test "\${ADP_GIT_ROOT:-}" = "$expected_project_root"
case ":\${GIT_CEILING_DIRECTORIES:-}:" in
  *":\$ADP_RUNTIME_ROOT:"*) ;;
  *)
    printf 'GIT_CEILING_DIRECTORIES missing runtime root: %s\n' "\${GIT_CEILING_DIRECTORIES:-}" >&2
    exit 96
    ;;
esac
test -z "\${ADP_TASK_ID:-}"
test -z "\${ADP_TASK_TITLE:-}"
test -z "\${ADP_TASK_STATUS:-}"
test -z "\${ADP_TASK_PRIORITY:-}"
test -z "\${ADP_TASK_PHASE:-}"
test -f "\$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
grep -F -q "version: 1" "\$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
grep -F -q "workspace: game-a" "\$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
grep -F -q "project_root: $expected_project_root" "\$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
grep -F -q "git_root: $expected_project_root" "\$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
grep -F -q "git_metadata_skipped: true" "\$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
grep -F -q "runtime_root: \$ADP_RUNTIME_ROOT" "\$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
grep -F -q "generated_by: adp" "\$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
test ! -e "\$ADP_RUNTIME_ROOT/.git"
if git -C "\$ADP_RUNTIME_ROOT" status --short --branch >/dev/null 2>&1; then
  printf 'git status unexpectedly succeeded inside ADP runtime root\n' >&2
  exit 96
fi
git -C "\$ADP_PROJECT_ROOT" status --short --branch >/dev/null
test -L go.mod
test -f go.mod
test -L main.go
test -f main.go
test ! -e AGENTS.md
test ! -e CLAUDE.md

printf 'enter-shell ok\n'
EOF
  chmod 755 "$path"
}

probe_value() {
  local path="$1"
  local key="$2"

  sed -n "s/^${key}=//p" "$path" | tail -n 1
}

run_fake_enter_checks() {
  local enter_shell="$fake_bin/enter-shell"
  local enter_probe="$smoke_root/enter-probe.txt"
  local event_snapshot="$smoke_root/enter-events-before.jsonl"
  local project_snapshot="$smoke_root/enter-project-before.txt"
  local output enter_runtime enter_workspace enter_project enter_cwd
  local before_runtime_entries after_runtime_entries

  info "fake smoke: enter workspace through controlled shell"
  write_fake_enter_shell "$enter_shell" "$project_root"
  export SHELL="$enter_shell"
  export ADP_ENTER_PROBE_OUT="$enter_probe"
  unset ADP_TASK_ID ADP_TASK_TITLE ADP_TASK_STATUS ADP_TASK_PRIORITY ADP_TASK_PHASE
  before_runtime_entries=$(runtime_entry_count "$runtime_dir")
  cp "$events_file" "$event_snapshot"
  snapshot_tree_entries "$project_root" "$project_snapshot"

  rm -f "$enter_probe"
  output=$(with_dangerous_git_env "$smoke_root/git-boundary-env" run_adp "$REPO_ROOT" enter game-a)
  assert_contains "$output" "enter-shell ok" "enter output"
  assert_file "$enter_probe"
  enter_runtime=$(probe_value "$enter_probe" runtime)
  enter_workspace=$(probe_value "$enter_probe" workspace)
  enter_project=$(probe_value "$enter_probe" project)
  enter_cwd=$(probe_value "$enter_probe" cwd)
  if [ -z "$enter_runtime" ]; then
    cat "$enter_probe" >&2
    fail "enter runtime path missing from probe"
  fi
  if [ "$enter_workspace" != "game-a" ] || [ "$enter_project" != "$project_root" ] || [ "$enter_cwd" != "$enter_runtime" ]; then
    cat "$enter_probe" >&2
    fail "enter probe mismatch"
  fi
  if [ -e "$enter_runtime" ]; then
    fail "non-kept enter runtime still exists: $enter_runtime"
  fi
  assert_runtime_entries "$runtime_dir" "$before_runtime_entries"
  assert_file_unchanged "$event_snapshot" "$events_file" "event log after enter"
  assert_absent_project_artifacts "$project_root"
  assert_tree_entries_unchanged "$project_root" "$project_snapshot" "project root after enter"

  rm -f "$enter_probe"
  output=$(with_dangerous_git_env "$smoke_root/git-boundary-env" run_adp "$REPO_ROOT" enter game-a --keep-runtime)
  assert_contains "$output" "enter-shell ok" "enter --keep-runtime output"
  assert_file "$enter_probe"
  enter_runtime=$(probe_value "$enter_probe" runtime)
  enter_workspace=$(probe_value "$enter_probe" workspace)
  enter_project=$(probe_value "$enter_probe" project)
  enter_cwd=$(probe_value "$enter_probe" cwd)
  if [ -z "$enter_runtime" ]; then
    cat "$enter_probe" >&2
    fail "kept enter runtime path missing from probe"
  fi
  if [ "$enter_workspace" != "game-a" ] || [ "$enter_project" != "$project_root" ] || [ "$enter_cwd" != "$enter_runtime" ]; then
    cat "$enter_probe" >&2
    fail "enter --keep-runtime probe mismatch"
  fi
  assert_file "$enter_runtime/.adp-runtime.yaml"
  assert_contains "$(cat "$enter_runtime/.adp-runtime.yaml")" "keep: true" "enter kept runtime manifest"
  assert_symlink "$enter_runtime/go.mod"
  assert_symlink "$enter_runtime/main.go"
  after_runtime_entries=$((before_runtime_entries + 1))
  assert_runtime_entries "$runtime_dir" "$after_runtime_entries"
  assert_file_unchanged "$event_snapshot" "$events_file" "event log after enter --keep-runtime"
  assert_absent_project_artifacts "$project_root"
  assert_tree_entries_unchanged "$project_root" "$project_snapshot" "project root after enter --keep-runtime"

  rm -rf "$enter_runtime"
  assert_runtime_entries "$runtime_dir" "$before_runtime_entries"
}
