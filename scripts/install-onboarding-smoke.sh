#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd -- "$SCRIPT_DIR/.." && pwd)

fail() {
  printf 'install-onboarding-smoke: %s\n' "$*" >&2
  exit 1
}

info() {
  printf '[install-onboarding-smoke] %s\n' "$*"
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

assert_executable() {
  local path="$1"

  if [ ! -x "$path" ]; then
    fail "missing executable: $path"
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

line_count() {
  local path="$1"

  assert_file "$path"
  wc -l < "$path" | tr -d '[:space:]'
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

release_ldflags() {
  printf '%s' "-s -w"
  printf ' %s' "-X github.com/karoc/adp/internal/cli.Version=$VERSION"
  printf ' %s' "-X github.com/karoc/adp/internal/cli.Commit=$COMMIT"
  printf ' %s' "-X github.com/karoc/adp/internal/cli.BuildDate=$BUILD_DATE"
}

build_local_binary() {
  local ldflags

  ldflags=$(release_ldflags)
  (
    cd "$REPO_ROOT"
    GOTOOLCHAIN=local GONOSUMDB='*' GOPROXY=off GOSUMDB=off \
      go build -buildvcs=false -mod=readonly -trimpath -ldflags="$ldflags" -o "$BUILD_BIN" ./cmd/adp
  )
  assert_executable "$BUILD_BIN"
}

install_to_temp_gobin() {
  local ldflags

  ldflags=$(release_ldflags)
  (
    cd "$REPO_ROOT"
    GOBIN="$INSTALL_BIN" GOTOOLCHAIN=local GONOSUMDB='*' GOPROXY=off GOSUMDB=off \
      go install -buildvcs=false -mod=readonly -trimpath -ldflags="$ldflags" ./cmd/adp
  )
  assert_executable "$INSTALL_BIN/adp"
}

write_fake_codex() {
  local path="$1"

  cat > "$path" <<'EOF'
#!/usr/bin/env sh
set -eu

printf 'fake-codex cwd=%s args=%s\n' "$(pwd)" "$*"

test "${ADP_WORKSPACE:-}" = "onboarding-a"
test -n "${ADP_SESSION_ID:-}"
test -n "${ADP_RUNTIME_ROOT:-}"
test -n "${ADP_TASK_ID:-}"
test "$(pwd)" = "$ADP_RUNTIME_ROOT"
test -f "$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
test -f "$ADP_RUNTIME_ROOT/AGENTS.md"
test -f "$ADP_RUNTIME_ROOT/.codex/config.toml"
test -L "$ADP_RUNTIME_ROOT/go.mod"
test -f "$ADP_RUNTIME_ROOT/go.mod"
test "$#" -eq 1

case "$1" in
  --install-onboarding)
    test "$ADP_TASK_ID" = "$ADP_EXPECT_TASK_ID"
    test "${ADP_TASK_TITLE:-}" = "Run install onboarding"
    grep -F -q "$ADP_EXPECT_TASK_ID" "$ADP_RUNTIME_ROOT/AGENTS.md"
    grep -F -q "Run install onboarding" "$ADP_RUNTIME_ROOT/AGENTS.md"
    grep -F -q "$ADP_EXPECT_TASK_ID" "$ADP_RUNTIME_ROOT/.codex/config.toml"
    ;;
  --trial-take)
    test "$ADP_TASK_ID" = "$ADP_EXPECT_TAKE_TASK_ID"
    test "${ADP_TASK_TITLE:-}" = "Claim trial workflow"
    test "${ADP_TASK_STATUS:-}" = "in_progress"
    test "${ADP_TASK_OWNER:-}" = "trial-agent"
    test -n "${ADP_TASK_CLAIMED_AT:-}"
    test -n "${ADP_TASK_LEASE_EXPIRES_AT:-}"
    grep -F -q "$ADP_EXPECT_TAKE_TASK_ID" "$ADP_RUNTIME_ROOT/AGENTS.md"
    grep -F -q "Claim trial workflow" "$ADP_RUNTIME_ROOT/AGENTS.md"
    grep -F -q "trial-agent" "$ADP_RUNTIME_ROOT/AGENTS.md"
    grep -F -q "$ADP_EXPECT_TAKE_TASK_ID" "$ADP_RUNTIME_ROOT/.codex/config.toml"
    ;;
  *)
    printf 'unexpected fake-codex argument: %s\n' "$1" >&2
    exit 99
    ;;
esac
EOF
  chmod 755 "$path"
}

write_fake_claude_guard() {
  local path="$1"

  cat > "$path" <<'EOF'
#!/usr/bin/env sh
set -eu

printf 'fake-claude guard should not be invoked by install onboarding smoke\n' >&2
exit 98
EOF
  chmod 755 "$path"
}

setup_git_tripwire() {
  local fake_bin="$1"
  local log_path="$2"
  local real_git

  real_git=$(command -v git || true)
  if [ -z "$real_git" ]; then
    fail "Git is required for smoke Git tripwire"
  fi

  export ADP_SMOKE_REAL_GIT="$real_git"
  export ADP_SMOKE_GIT_TRIPWIRE_LOG="$log_path"

  cat > "$fake_bin/git" <<'EOF'
#!/usr/bin/env sh
set -eu

: "${ADP_SMOKE_REAL_GIT:?}"
: "${ADP_SMOKE_GIT_TRIPWIRE_LOG:?}"

for arg do
  case "$arg" in
    commit|push|pull|fetch|clone|ls-remote|tag|branch|merge|rebase|checkout|switch|restore|reset)
      printf '%s\n' "$*" >> "$ADP_SMOKE_GIT_TRIPWIRE_LOG"
      printf 'fake git blocked install-onboarding side-effect command: %s\n' "$*" >&2
      exit 97
      ;;
  esac
done

exec "$ADP_SMOKE_REAL_GIT" "$@"
EOF
  chmod 755 "$fake_bin/git"
  reset_git_tripwire
}

reset_git_tripwire() {
  : "${ADP_SMOKE_GIT_TRIPWIRE_LOG:?}"
  : > "$ADP_SMOKE_GIT_TRIPWIRE_LOG"
}

assert_no_git_side_effects() {
  local label="$1"

  : "${ADP_SMOKE_GIT_TRIPWIRE_LOG:?}"
  if [ -s "$ADP_SMOKE_GIT_TRIPWIRE_LOG" ]; then
    printf '%s\n' "Git side-effect command log:" >&2
    cat "$ADP_SMOKE_GIT_TRIPWIRE_LOG" >&2
    fail "$label invoked a Git side-effect command"
  fi
}

init_project_git() {
  if ! command -v git >/dev/null 2>&1; then
    fail "Git is required for install onboarding smoke"
  fi
  git -C "$PROJECT_ROOT" init -q
  git -C "$PROJECT_ROOT" config user.name adp-smoke
  git -C "$PROJECT_ROOT" config user.email adp-smoke@example.invalid
  git -C "$PROJECT_ROOT" add go.mod main.go
  git -C "$PROJECT_ROOT" commit -q -m "init install onboarding project"
}

run_adp() {
  local dir="$1"
  shift
  local output

  if ! output=$(cd "$dir" && adp "$@" 2>&1); then
    printf '%s\n' "$output" >&2
    fail "adp $* failed"
  fi
  printf '%s\n' "$output"
}

for cmd in bash go git; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    fail "$cmd is required"
  fi
done

TMP_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/adp-install-onboarding.XXXXXX")
BUILD_BIN="$TMP_ROOT/build/adp"
INSTALL_BIN="$TMP_ROOT/gobin"
FAKE_BIN="$TMP_ROOT/fake-bin"
PROJECT_ROOT="$TMP_ROOT/project"
ADP_HOME="$TMP_ROOT/adp-home"
ADP_RUNTIME_DIR="$TMP_ROOT/runtime"
GIT_TRIPWIRE_LOG="$TMP_ROOT/git-side-effects.log"
EVENTS_FILE="$ADP_HOME/logs/events.jsonl"
TASKS_FILE="$ADP_HOME/workspaces/onboarding-a/planning/tasks.yaml"
PHASES_FILE="$ADP_HOME/workspaces/onboarding-a/planning/phases.yaml"
PROGRESS_FILE="$ADP_HOME/workspaces/onboarding-a/planning/progress.jsonl"

VERSION="0.1.0-install-onboarding"
COMMIT="install-onboarding-smoke"
BUILD_DATE="2026-06-09T00:00:00Z"

cleanup() {
  rm -rf "$TMP_ROOT"
}
trap cleanup EXIT INT TERM

mkdir -p "$INSTALL_BIN" "$FAKE_BIN" "$PROJECT_ROOT" "$ADP_HOME" "$ADP_RUNTIME_DIR" "$(dirname -- "$BUILD_BIN")"
printf 'module example.com/adp-install-onboarding\n' > "$PROJECT_ROOT/go.mod"
printf 'package main\n' > "$PROJECT_ROOT/main.go"
init_project_git
write_fake_codex "$FAKE_BIN/codex"
write_fake_claude_guard "$FAKE_BIN/claude"

info "building local binary with deterministic release metadata"
build_local_binary
output=$("$BUILD_BIN" version)
assert_contains "$output" "adp $VERSION commit $COMMIT built $BUILD_DATE" "local build version output"

info "installing adp into a temporary GOBIN"
install_to_temp_gobin

export ADP_HOME
export ADP_RUNTIME_DIR
export PATH="$INSTALL_BIN:$FAKE_BIN:$PATH"
hash -r

if [ "$(command -v adp)" != "$INSTALL_BIN/adp" ]; then
  fail "temporary installed adp is not first on PATH"
fi
if [ "$(command -v codex)" != "$FAKE_BIN/codex" ]; then
  fail "fake codex is not first on PATH"
fi
if [ "$(command -v claude)" != "$FAKE_BIN/claude" ]; then
  fail "fake claude guard is not first on PATH"
fi

setup_git_tripwire "$FAKE_BIN" "$GIT_TRIPWIRE_LOG"

info "running first-use onboarding commands through the installed binary"
output=$(run_adp "$TMP_ROOT" version)
assert_contains "$output" "adp $VERSION commit $COMMIT built $BUILD_DATE" "installed version output"
output=$(run_adp "$TMP_ROOT" init)
assert_contains "$output" "initialized ADP home" "init output"
output=$(run_adp "$TMP_ROOT" workspace add onboarding-a "$PROJECT_ROOT")
assert_contains "$output" 'workspace "onboarding-a" added' "workspace add output"
output=$(run_adp "$TMP_ROOT" workspace doctor onboarding-a)
assert_contains "$output" "onboarding-a" "workspace doctor output"
assert_contains "$output" "workspace.git.root.detected" "workspace doctor output"
output=$(run_adp "$TMP_ROOT" workspace show onboarding-a)
assert_contains "$output" "name: onboarding-a" "workspace show output"
assert_contains "$output" "project_root: $PROJECT_ROOT" "workspace show output"

info "creating a minimal phase and task for task-bound provider handoff"
output=$(run_adp "$TMP_ROOT" phase add --workspace onboarding-a --goal "install onboarding smoke" p-install "Install Onboarding")
assert_contains "$output" "phase p-install added" "phase add output"
output=$(run_adp "$TMP_ROOT" phase start --workspace onboarding-a p-install)
assert_contains "$output" "phase p-install status: active" "phase start output"
output=$(run_adp "$TMP_ROOT" tasks add --workspace onboarding-a --priority high --phase p-install --description "new operator onboarding path" "Run install onboarding")
assert_contains "$output" "task task-" "tasks add output"
TASK_ID=$(printf '%s\n' "$output" | sed -n 's/^task \(task-[^ ]*\) added$/\1/p')
if [ -z "$TASK_ID" ]; then
  fail "could not parse task id from: $output"
fi
export ADP_EXPECT_TASK_ID="$TASK_ID"

assert_file "$TASKS_FILE"
assert_file "$PHASES_FILE"
assert_file "$PROGRESS_FILE"
assert_project_root_clean

info "running task-bound fake codex from the installed binary"
reset_git_tripwire
output=$(run_adp "$TMP_ROOT" run codex --workspace onboarding-a --task "$TASK_ID" -- --install-onboarding)
assert_contains "$output" "fake-codex" "fake codex output"
assert_no_git_side_effects "install onboarding fake codex run"
assert_project_root_clean

info "checking events, sessions, progress, and planning diagnostics"
output=$(run_adp "$TMP_ROOT" events list --workspace onboarding-a --task "$TASK_ID" --type run_finished --limit 1)
assert_contains "$output" "run_finished" "events list output"
assert_contains "$output" "codex" "events list output"
assert_contains "$output" "$TASK_ID" "events list output"
assert_file "$EVENTS_FILE"

output=$(run_adp "$TMP_ROOT" sessions list --workspace onboarding-a --agent codex --task "$TASK_ID")
assert_contains "$output" "codex" "sessions list output"
assert_contains "$output" "$TASK_ID" "sessions list output"

output=$(run_adp "$TMP_ROOT" progress --workspace onboarding-a --format json)
assert_contains "$output" '"workspace": "onboarding-a"' "progress json output"
assert_contains "$output" '"total": 1' "progress json output"
assert_contains "$output" '"counts"' "progress json output"

output=$(run_adp "$TMP_ROOT" plan doctor --workspace onboarding-a --format json)
assert_contains "$output" '"workspace": "onboarding-a"' "plan doctor json output"
assert_contains "$output" '"status": "ok"' "plan doctor json output"
assert_contains "$output" '"task_count": 1' "plan doctor json output"
assert_contains "$output" '"phase_count": 1' "plan doctor json output"
assert_contains "$output" '"has_errors": false' "plan doctor json output"
assert_project_root_clean

info "checking friendly trial workflow pickup, lease, stale, and restore guidance"
output=$(run_adp "$TMP_ROOT" tasks add --workspace onboarding-a --priority critical --phase p-install --description "atomic worker pickup" "Claim trial workflow")
assert_contains "$output" "task task-" "trial take task add output"
TAKE_TASK_ID=$(printf '%s\n' "$output" | sed -n 's/^task \(task-[^ ]*\) added$/\1/p')
if [ -z "$TAKE_TASK_ID" ]; then
  fail "could not parse take task id from: $output"
fi
export ADP_EXPECT_TAKE_TASK_ID="$TAKE_TASK_ID"

output=$(run_adp "$TMP_ROOT" tasks next --workspace onboarding-a --limit 1 --format json)
assert_contains "$output" "\"$TAKE_TASK_ID\"" "tasks next json output"
assert_contains "$output" '"eligible_count": 1' "tasks next json output"
assert_contains "$output" '"claim_state": "unclaimed"' "tasks next json output"

reset_git_tripwire
events_before=$(line_count "$EVENTS_FILE")
output=$(run_adp "$TMP_ROOT" run codex --workspace onboarding-a --take --owner trial-agent --lease 30m -- --trial-take)
assert_contains "$output" "fake-codex" "run take fake codex output"
assert_contains "$output" "--trial-take" "run take fake codex output"
assert_no_git_side_effects "install onboarding run --take"
assert_project_root_clean
if [ "$(line_count "$EVENTS_FILE")" != $((events_before + 2)) ]; then
  fail "run --take should append two runtime events"
fi

take_session=$(session_id_by_agent "$EVENTS_FILE" codex)
if [ -z "$take_session" ]; then
  cat "$EVENTS_FILE" >&2
  fail "run --take session id missing in event log"
fi

output=$(run_adp "$TMP_ROOT" tasks show --workspace onboarding-a "$TAKE_TASK_ID")
assert_contains "$output" "status: in_progress" "taken task show output"
assert_contains "$output" "owner: trial-agent" "taken task show output"
assert_contains "$output" "claim_state: leased" "taken task show output"
assert_contains "$output" "lease_expires_at: 20" "taken task show output"

output=$(run_adp "$TMP_ROOT" tasks renew --workspace onboarding-a "$TAKE_TASK_ID" --owner trial-agent --lease 45m)
assert_contains "$output" "task $TAKE_TASK_ID lease renewed until" "tasks renew output"

output=$(run_adp "$TMP_ROOT" tasks add --workspace onboarding-a --priority low --phase p-install --description "expired worker claim" "Recover stale trial workflow")
assert_contains "$output" "task task-" "stale task add output"
STALE_TASK_ID=$(printf '%s\n' "$output" | sed -n 's/^task \(task-[^ ]*\) added$/\1/p')
if [ -z "$STALE_TASK_ID" ]; then
  fail "could not parse stale task id from: $output"
fi
output=$(run_adp "$TMP_ROOT" tasks claim --workspace onboarding-a "$STALE_TASK_ID" --owner abandoned-agent --lease 1ms)
assert_contains "$output" "task $STALE_TASK_ID claimed by abandoned-agent" "stale task claim output"
sleep 1
output=$(run_adp "$TMP_ROOT" tasks stale --workspace onboarding-a --format json)
assert_contains "$output" '"stale_count": 1' "tasks stale json output"
assert_contains "$output" "\"$STALE_TASK_ID\"" "tasks stale json output"
assert_contains "$output" '"owner": "abandoned-agent"' "tasks stale json output"
assert_contains "$output" '"claim_state": "stale"' "tasks stale json output"

output=$(run_adp "$TMP_ROOT" sessions restore-plan "$take_session")
assert_contains "$output" "session_id: $take_session" "take restore-plan output"
assert_contains "$output" "status: ready" "take restore-plan output"
assert_contains "$output" "adp run codex --workspace onboarding-a --task $TAKE_TASK_ID" "take restore-plan output"
assert_contains "$output" "-- --trial-take" "take restore-plan output"

output=$(run_adp "$TMP_ROOT" progress report --workspace onboarding-a)
assert_contains "$output" "# ADP Progress Report" "progress report output"
assert_contains "$output" "$TAKE_TASK_ID" "progress report output"
assert_contains "$output" "$STALE_TASK_ID" "progress report output"
assert_contains "$output" "$take_session" "progress report output"
assert_contains "$output" "Claim" "progress report output"
assert_contains "$output" "leased to trial-agent" "progress report output"
assert_contains "$output" "stale claim by abandoned-agent" "progress report output"
assert_project_root_clean

info "install onboarding smoke passed"
