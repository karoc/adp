#!/usr/bin/env bash

run_fake_diagnostics_checks() {
  local output

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

  output=$(
    export ADP_RUNTIME_DIR="$project_root/.adp-runtime-parent"
    run_adp_expect_fail "$REPO_ROOT" workspace doctor game-a
  )
  assert_contains "$output" "workspace.runtime.parent.inside_project_root" "workspace doctor runtime parent output"
}
