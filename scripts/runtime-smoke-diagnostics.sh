#!/usr/bin/env bash

run_fake_diagnostics_checks() {
  local output project_snapshot

  info "fake smoke: inspect agent diagnostics"
  mkdir -p "$diag_project_root/.codex" "$diag_project_root/planning"
  printf '# project agents\n' > "$diag_project_root/AGENTS.md"
  printf 'runtime_root: %s\n' "$diag_project_root" > "$diag_project_root/.adp-runtime.yaml"
  output=$(run_adp "$REPO_ROOT" workspace add diag-a "$diag_project_root")
  assert_contains "$output" 'workspace "diag-a" added' "diagnostics workspace add output"
  printf '# escaped profile\n' > "$smoke_root/escaped-profile.yaml"
  rm -f "$adp_home/workspaces/diag-a/profiles/escaped-profile.yaml"
  ln -s "$smoke_root/escaped-profile.yaml" "$adp_home/workspaces/diag-a/profiles/escaped-profile.yaml"
  cat > "$adp_home/workspaces/diag-a/workspace.yaml" <<YAML
version: 1
workspace:
  name: diag-a
project:
  root: $diag_project_root
memory:
  enabled: false
prompts:
  base: prompts/base.md
mcp:
  enabled: false
agents:
  codex:
    enabled: true
    profile: missing-profile
    command: ""
  claude:
    enabled: true
    profile: escaped-profile
    command: claude
  local:
    enabled: true
    profile: default
    command: "codex --model test"
YAML
  output=$(run_adp "$REPO_ROOT" workspace doctor diag-a)
  assert_contains "$output" "diag-a" "agent diagnostics output"
  assert_contains "$output" "workspace.project.reserved_path.present" "agent diagnostics output"
  assert_contains "$output" "workspace.agent.command.default" "agent diagnostics output"
  assert_contains "$output" "workspace.agent.command.arguments" "agent diagnostics output"
  assert_contains "$output" "workspace.agent.profile.missing" "agent diagnostics output"
  assert_contains "$output" "workspace.agent.profile.outside_workspace" "agent diagnostics output"
  assert_contains "$output" "workspace.agent.unknown" "agent diagnostics output"

  output=$(run_adp "$REPO_ROOT" doctor diag-a)
  assert_contains "$output" "diag-a" "global agent diagnostics output"
  assert_contains "$output" "workspace.project.reserved_path.present" "global agent diagnostics output"
  assert_contains "$output" "workspace.agent.profile.outside_workspace" "global agent diagnostics output"

  output=$(
    export ADP_RUNTIME_DIR="$project_root"
    run_adp_expect_fail "$REPO_ROOT" doctor game-a
  )
  assert_contains "$output" "workspace.runtime.parent.project_root" "doctor runtime parent output"
  project_snapshot="$smoke_root/runtime-parent-project-root-before.txt"
  snapshot_tree_entries "$project_root" "$project_snapshot"
  output=$(
    export ADP_RUNTIME_DIR="$project_root"
    run_adp_expect_fail "$REPO_ROOT" env game-a --cd
  )
  assert_contains "$output" "must not be the project root" "env runtime parent project root output"
  assert_tree_entries_unchanged "$project_root" "$project_snapshot" "project root after project-root runtime parent"

  output=$(
    export ADP_RUNTIME_DIR="$project_root/.adp-runtime-parent"
    run_adp_expect_fail "$REPO_ROOT" workspace doctor game-a
  )
  assert_contains "$output" "workspace.runtime.parent.inside_project_root" "workspace doctor runtime parent output"
  project_snapshot="$smoke_root/runtime-parent-inside-project-before.txt"
  snapshot_tree_entries "$project_root" "$project_snapshot"
  output=$(
    export ADP_RUNTIME_DIR="$project_root/.adp-runtime-parent"
    run_adp_expect_fail "$REPO_ROOT" env game-a --cd
  )
  assert_contains "$output" "must not be inside the project root" "env runtime parent inside project output"
  assert_tree_entries_unchanged "$project_root" "$project_snapshot" "project root after inside-project runtime parent"
}
