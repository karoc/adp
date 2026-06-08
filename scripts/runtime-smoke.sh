#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage:
  scripts/runtime-smoke.sh [--fake] [--real-codex] [--real-claude]

Runs ADP runtime smoke acceptance from a temporary ADP home, runtime
directory, project root, and agent bin directory.

The fake smoke is the default path and is deterministic. Real external
CLI checks are opt-in and require both a flag and an environment gate:

  ADP_SMOKE_REAL_CODEX=1 scripts/runtime-smoke.sh --real-codex
  ADP_SMOKE_REAL_CLAUDE=1 scripts/runtime-smoke.sh --real-claude

Optional real CLI binary overrides:

  ADP_SMOKE_CODEX_BIN=/path/to/codex
  ADP_SMOKE_CLAUDE_BIN=/path/to/claude
USAGE
}

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

  for rel in AGENTS.md CLAUDE.md .codex .claude planning tasks.yaml phases.yaml progress.jsonl; do
    if [ -e "$project_root/$rel" ] || [ -L "$project_root/$rel" ]; then
      fail "project root was polluted with $rel"
    fi
  done
}

assert_runtime_entries() {
  local runtime_dir="$1"
  local want="$2"
  local got

  got=$(find "$runtime_dir" -mindepth 1 -maxdepth 1 -type d | wc -l | tr -d '[:space:]')
  if [ "$got" != "$want" ]; then
    fail "runtime dir entry count is $got, expected $want"
  fi
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

test "\${ADP_WORKSPACE:-}" = "game-a"
test -n "\${ADP_SESSION_ID:-}"
test -n "\${ADP_RUNTIME_ROOT:-}"
test "\$(pwd)" = "\$ADP_RUNTIME_ROOT"
test -f "\$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
grep -F -q "version: 1" "\$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
grep -F -q "runtime_root: \$ADP_RUNTIME_ROOT" "\$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
grep -F -q "generated_by: adp" "\$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
test -f "$instructions"
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
  local codex_session sessions_output session_output restore_plan_output prune_output invalid_output
  local task_event_count before_restore_lines after_restore_lines

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

  info "fake smoke: inspect events and sessions"
  events_output=$(run_adp "$REPO_ROOT" events list --workspace game-a --task "$task_id" --type run_finished --limit 2)
  assert_contains "$events_output" "run_finished" "events list output"
  assert_contains "$events_output" "codex" "events list output"
  assert_contains "$events_output" "claude" "events list output"
  assert_contains "$events_output" "$task_id" "events list output"

  codex_session=$(session_id_by_agent "$events_file" codex)
  if [ -z "$codex_session" ]; then
    cat "$events_file" >&2
    fail "codex session id missing in event log"
  fi

  sessions_output=$(run_adp "$REPO_ROOT" sessions list --workspace game-a --agent codex --task "$task_id")
  assert_contains "$sessions_output" "$codex_session" "sessions list output"
  assert_contains "$sessions_output" "codex" "sessions list output"
  assert_contains "$sessions_output" "$task_id" "sessions list output"

  events_output=$(run_adp "$REPO_ROOT" events list --session "$codex_session" --task "$task_id" --limit 2)
  assert_contains "$events_output" "$codex_session" "events list session output"
  assert_contains "$events_output" "$task_id" "events list session output"
  assert_contains "$events_output" "codex" "events list session output"
  assert_contains "$events_output" "run_started" "events list session output"
  assert_contains "$events_output" "run_finished" "events list session output"

  session_output=$(run_adp "$REPO_ROOT" sessions show "$codex_session")
  assert_contains "$session_output" "session_id: $codex_session" "sessions show output"
  assert_contains "$session_output" "task_id: $task_id" "sessions show output"
  assert_contains "$session_output" "run_started" "sessions show output"
  assert_contains "$session_output" "run_finished" "sessions show output"

  before_restore_lines=$(line_count "$events_file")
  restore_plan_output=$(run_adp "$REPO_ROOT" sessions restore-plan "$codex_session")
  assert_contains "$restore_plan_output" "session_id: $codex_session" "sessions restore-plan output"
  assert_contains "$restore_plan_output" "status: ready" "sessions restore-plan output"
  assert_contains "$restore_plan_output" "suggested_command:" "sessions restore-plan output"
  assert_contains "$restore_plan_output" "adp run codex --workspace game-a" "sessions restore-plan output"
  assert_contains "$restore_plan_output" "--task $task_id" "sessions restore-plan output"
  assert_contains "$restore_plan_output" "-- --probe codex-payload" "sessions restore-plan output"
  assert_contains "$restore_plan_output" "missing_fields: -" "sessions restore-plan output"
  after_restore_lines=$(line_count "$events_file")
  if [ "$after_restore_lines" != "$before_restore_lines" ]; then
    cat "$events_file" >&2
    fail "sessions restore-plan appended events: before=$before_restore_lines after=$after_restore_lines"
  fi

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

  assert_absent_project_artifacts "$project_root"
  info "fake smoke passed"
)

run_fake=1
run_real_codex=0
run_real_claude=0

while [ "$#" -gt 0 ]; do
  case "$1" in
    --fake)
      run_fake=1
      ;;
    --real-codex)
      run_real_codex=1
      ;;
    --real-claude)
      run_real_claude=1
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      usage >&2
      fail "unknown option: $1"
      ;;
  esac
  shift
done

if ! command -v go >/dev/null 2>&1; then
  fail "Go is required to build cmd/adp"
fi

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd -- "$SCRIPT_DIR/.." && pwd)
TMP_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/adp-runtime-smoke.XXXXXX")
ADP_BIN="$TMP_ROOT/adp"

cleanup() {
  rm -rf "$TMP_ROOT"
}
trap cleanup EXIT INT TERM

info "building temporary adp binary"
(cd "$REPO_ROOT" && go build -o "$ADP_BIN" ./cmd/adp)

if [ "$run_fake" -eq 1 ]; then
  run_fake_smoke
fi

if [ "$run_real_codex" -eq 1 ]; then
  run_real_cli_smoke codex ADP_SMOKE_REAL_CODEX "${ADP_SMOKE_CODEX_BIN:-codex}"
fi

if [ "$run_real_claude" -eq 1 ]; then
  run_real_cli_smoke claude ADP_SMOKE_REAL_CLAUDE "${ADP_SMOKE_CLAUDE_BIN:-claude}"
fi

info "runtime smoke acceptance passed"
