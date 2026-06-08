#!/usr/bin/env bash

run_fake_prune_checks() {
  local incompatible_root prune_output

  info "fake smoke: prune kept runtime"
  incompatible_root="$runtime_dir/incompatible-runtime"
  mkdir -p "$incompatible_root"
  cat > "$incompatible_root/.adp-runtime.yaml" <<EOF
version: 999
session_id: incompatible-session
workspace: game-a
project_root: $project_root
runtime_root: $incompatible_root
created_at: "2026-06-08T12:00:00Z"
generated_by: adp
EOF
  assert_runtime_entries "$runtime_dir" 2

  prune_output=$(run_adp "$REPO_ROOT" runtime prune --older-than 0s --include-kept --dry-run)
  assert_contains "$prune_output" "would-remove" "runtime prune dry-run output"
  assert_contains "$prune_output" "$runtime_root" "runtime prune dry-run output"
  case "$prune_output" in
    *"$incompatible_root"*) fail "runtime prune reported incompatible manifest: $prune_output" ;;
  esac
  assert_runtime_entries "$runtime_dir" 2

  prune_output=$(run_adp "$REPO_ROOT" runtime prune --older-than 0s --include-kept)
  assert_contains "$prune_output" "removed" "runtime prune output"
  assert_contains "$prune_output" "$runtime_root" "runtime prune output"
  case "$prune_output" in
    *"$incompatible_root"*) fail "runtime prune removed incompatible manifest: $prune_output" ;;
  esac
  assert_runtime_entries "$runtime_dir" 1
  assert_file "$incompatible_root/.adp-runtime.yaml"
}
