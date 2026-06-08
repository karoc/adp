#!/usr/bin/env bash

run_fake_smoke() (
  local smoke_root="$TMP_ROOT/fake"
  local project_root="$smoke_root/project"
  local diag_project_root="$smoke_root/diagnostics-project"
  local fake_bin="$smoke_root/bin"
  local adp_home="$smoke_root/adp-home"
  local runtime_dir="$smoke_root/runtime"
  local events_file="$adp_home/logs/events.jsonl"
  local output env_output runtime_root task_output task_id codex_output claude_output
  local completion_output zsh_completion_output workspace_values profile_values version_output events_output
  local invalid_output task_event_count

  mkdir -p "$project_root" "$diag_project_root" "$fake_bin" "$adp_home" "$runtime_dir"
  printf 'module example.com/adp-smoke\n' > "$project_root/go.mod"
  printf 'package main\n' > "$project_root/main.go"
  printf 'module example.com/adp-diagnostics-smoke\n' > "$diag_project_root/go.mod"
  printf 'package main\n' > "$diag_project_root/main.go"

  write_fake_agent "$fake_bin/codex" codex AGENTS.md .codex/config.toml go.mod
  write_fake_agent "$fake_bin/claude" claude CLAUDE.md .claude/settings.json main.go

  export ADP_HOME="$adp_home"
  export ADP_RUNTIME_DIR="$runtime_dir"
  export PATH="$fake_bin:$PATH"

  info "fake smoke: init and register workspace"
  output=$(run_adp "$REPO_ROOT" init)
  assert_contains "$output" "initialized ADP home" "init output"

  output=$(run_adp "$REPO_ROOT" workspace add game-a "$project_root")
  assert_contains "$output" 'workspace "game-a" added' "workspace add output"

  output=$(run_adp "$REPO_ROOT" workspace list)
  assert_contains "$output" "game-a" "workspace list output"
  assert_contains "$output" "$project_root" "workspace list output"

  output=$(run_adp "$REPO_ROOT" workspace show game-a)
  assert_contains "$output" "name: game-a" "workspace show output"
  assert_contains "$output" "project_root: $project_root" "workspace show output"

  output=$(run_adp "$REPO_ROOT" workspace doctor game-a)
  assert_contains "$output" "game-a" "workspace doctor output"
  assert_contains "$output" "ok" "workspace doctor output"

  output=$(run_adp "$REPO_ROOT" workspace doctor)
  assert_contains "$output" "game-a" "workspace doctor all output"
  assert_contains "$output" "ok" "workspace doctor all output"

  output=$(run_adp "$REPO_ROOT" doctor game-a)
  assert_contains "$output" "game-a" "doctor output"
  assert_contains "$output" "ok" "doctor output"

  output=$(run_adp "$REPO_ROOT" doctor)
  assert_contains "$output" "game-a" "doctor all output"
  assert_contains "$output" "ok" "doctor all output"

  run_fake_workspace_lifecycle_checks
  run_fake_diagnostics_checks

  version_output=$(run_adp "$REPO_ROOT" version)
  assert_contains "$version_output" "adp dev" "version output"

  version_output=$(run_adp "$REPO_ROOT" --version)
  assert_contains "$version_output" "adp dev" "--version output"

  workspace_values=$(run_adp "$REPO_ROOT" completion values workspaces)
  assert_contains "$workspace_values" "game-a" "completion workspace values output"

  profile_values=$(run_adp "$REPO_ROOT" completion values profiles --workspace game-a)
  assert_contains "$profile_values" "default" "completion profile values output"
  assert_contains "$profile_values" "codex" "completion profile values output"
  assert_contains "$profile_values" "claude" "completion profile values output"

  info "fake smoke: create task for runtime binding"
  task_output=$(run_adp "$REPO_ROOT" tasks add --workspace game-a --priority high --phase p1 --description "runtime binding smoke" "Bind runtime session to task")
  assert_contains "$task_output" "task task-" "tasks add output"
  task_id=$(printf '%s\n' "$task_output" | sed -n 's/^task \(task-[^ ]*\) added$/\1/p')
  if [ -z "$task_id" ]; then
    fail "could not parse task id from: $task_output"
  fi
  export ADP_EXPECT_TASK_ID="$task_id"

  info "fake smoke: build kept runtime with env --cd"
  env_output=$(run_adp "$REPO_ROOT" env game-a --cd)
  runtime_root=$(parse_export "$env_output" ADP_RUNTIME_ROOT)
  assert_contains "$env_output" "cd '$runtime_root'" "env --cd output"
  assert_file "$runtime_root/.adp-runtime.yaml"
  assert_contains "$(cat "$runtime_root/.adp-runtime.yaml")" "version: 1" "runtime manifest"
  assert_contains "$(cat "$runtime_root/.adp-runtime.yaml")" "runtime_root: $runtime_root" "runtime manifest"
  assert_contains "$(cat "$runtime_root/.adp-runtime.yaml")" "generated_by: adp" "runtime manifest"
  assert_symlink "$runtime_root/go.mod"
  assert_absent_project_artifacts "$project_root"

  completion_output=$(run_adp "$REPO_ROOT" completion --shell bash)
  assert_contains "$completion_output" "complete -F _adp_completion adp" "bash completion output"
  assert_contains "$completion_output" "sessions" "bash completion output"
  assert_contains "$completion_output" "completion values workspaces" "bash completion output"
  assert_contains "$completion_output" "completion values profiles" "bash completion output"

  zsh_completion_output=$(run_adp "$REPO_ROOT" completion --shell zsh)
  assert_contains "$zsh_completion_output" "compdef _adp_completion adp" "zsh completion output"
  assert_contains "$zsh_completion_output" "workspace" "zsh completion output"
  assert_contains "$zsh_completion_output" "completion values workspaces" "zsh completion output"
  assert_contains "$zsh_completion_output" "completion values profiles" "zsh completion output"

  info "fake smoke: run codex and claude through runtime overlays"
  codex_output=$(run_adp "$REPO_ROOT" run codex --workspace game-a --task "$task_id" -- --probe codex-payload)
  assert_contains "$codex_output" "fake-codex" "codex run output"
  assert_contains "$codex_output" "--probe codex-payload" "codex run output"

  claude_output=$(run_adp "$project_root" run claude --task "$task_id" -- --probe claude-payload)
  assert_contains "$claude_output" "fake-claude" "claude run output"
  assert_contains "$claude_output" "--probe claude-payload" "claude run output"

  assert_absent_project_artifacts "$project_root"
  assert_line_count "$events_file" 4
  task_event_count=$({ grep "\"task_id\":\"$task_id\"" "$events_file" || true; } | wc -l | tr -d '[:space:]')
  if [ "$task_event_count" != "4" ]; then
    cat "$events_file" >&2
    fail "task-bound event count is $task_event_count, expected 4"
  fi

  invalid_output=$(run_adp_expect_fail "$REPO_ROOT" run codex --workspace game-a --task missing-task -- --probe codex-payload)
  assert_contains "$invalid_output" 'load task "missing-task"' "missing task run output"

  run_fake_enter_checks
  run_fake_session_checks
  run_fake_prune_checks

  assert_absent_project_artifacts "$project_root"
  info "fake smoke passed"
)
