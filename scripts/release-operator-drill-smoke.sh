#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd -- "$SCRIPT_DIR/.." && pwd)

fail() {
  printf 'release-operator-drill-smoke: %s\n' "$*" >&2
  exit 1
}

info() {
  printf '[release-operator-drill-smoke] %s\n' "$*"
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

assert_absent() {
  local path="$1"
  local label="$2"

  if [ -e "$path" ] || [ -L "$path" ]; then
    fail "$label unexpectedly exists: $path"
  fi
}

copy_current_tree() {
  local destination="$1"
  local path

  mkdir -p "$destination"
  while IFS= read -r -d '' path; do
    mkdir -p "$destination/$(dirname -- "$path")"
    cp -Pp "$REPO_ROOT/$path" "$destination/$path"
  done < <(cd "$REPO_ROOT" && git ls-files --cached --others --exclude-standard -z)
}

assert_clean_source_boundary() {
  local source_root="$1"
  local rel

  for rel in .git .envrc mvp.md ADP_HOME ADP_RUNTIME_DIR .codex .claude planning logs credentials; do
    assert_absent "$source_root/$rel" "clean operator source"
  done
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

assert_release_docs_have_operator_commands() {
  local source_root="$1"
  local packaging="$source_root/docs/release-packaging.md"
  local packaging_cn="$source_root/docs/release-packaging.zh-CN.md"

  assert_file "$packaging"
  assert_file "$packaging_cn"
  assert_contains "$(cat "$packaging")" 'COMMIT=source-archive-commit' "release packaging source archive command"
  assert_contains "$(cat "$packaging")" 'go build -trimpath -ldflags="$LDFLAGS" -o dist/adp ./cmd/adp' "release packaging build command"
  assert_contains "$(cat "$packaging")" 'sha256sum -c dist/adp.sha256' "release packaging checksum command"
  assert_contains "$(cat "$packaging")" 'install -m 0755 dist/adp "${ADP_INSTALL_BIN}/adp"' "release packaging install command"
  assert_contains "$(cat "$packaging_cn")" 'COMMIT=source-archive-commit' "release packaging zh source archive command"
  assert_contains "$(cat "$packaging_cn")" 'sha256sum -c dist/adp.sha256' "release packaging zh checksum command"
}

release_ldflags() {
  printf '%s' "-s -w"
  printf ' %s' "-X github.com/karoc/adp/internal/cli.Version=$VERSION"
  printf ' %s' "-X github.com/karoc/adp/internal/cli.Commit=$COMMIT"
  printf ' %s' "-X github.com/karoc/adp/internal/cli.BuildDate=$BUILD_DATE"
}

build_release_binary() {
  local source_root="$1"
  local output_path="$2"
  local ldflags

  ldflags=$(release_ldflags)
  mkdir -p "$(dirname -- "$output_path")"
  (
    cd "$source_root"
    GOTOOLCHAIN=local GONOSUMDB='*' GOPROXY=off GOSUMDB=off \
      go build -buildvcs=false -trimpath -ldflags="$ldflags" -o "$output_path" ./cmd/adp
  )
  assert_executable "$output_path"
}

sha256_write() {
  local path="$1"
  local output_path="$2"
  local base

  base=$(basename -- "$path")
  if command -v sha256sum >/dev/null 2>&1; then
    (cd "$(dirname -- "$path")" && sha256sum "$base") > "$output_path"
    return
  fi
  if command -v shasum >/dev/null 2>&1; then
    (cd "$(dirname -- "$path")" && shasum -a 256 "$base") > "$output_path"
    return
  fi
  fail "sha256sum or shasum is required"
}

sha256_verify() {
  local checksum_path="$1"

  if command -v sha256sum >/dev/null 2>&1; then
    (cd "$(dirname -- "$checksum_path")" && sha256sum -c "$(basename -- "$checksum_path")")
    return
  fi
  if command -v shasum >/dev/null 2>&1; then
    (cd "$(dirname -- "$checksum_path")" && shasum -a 256 -c "$(basename -- "$checksum_path")")
    return
  fi
  fail "sha256sum or shasum is required"
}

write_fake_codex() {
  local path="$1"

  cat > "$path" <<'EOF'
#!/usr/bin/env sh
set -eu

printf 'fake-codex cwd=%s args=%s\n' "$(pwd)" "$*"
test "${ADP_WORKSPACE:-}" = "operator-a"
test -n "${ADP_SESSION_ID:-}"
test -n "${ADP_RUNTIME_ROOT:-}"
test -n "${ADP_TASK_ID:-}"
test "$(pwd)" = "$ADP_RUNTIME_ROOT"
test -f "$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
test -f "$ADP_RUNTIME_ROOT/AGENTS.md"
test -f "$ADP_RUNTIME_ROOT/.codex/config.toml"
test -L "$ADP_RUNTIME_ROOT/go.mod"
test "$#" -eq 1
test "$1" = "--operator-drill"
EOF
  chmod 755 "$path"
}

write_fake_claude_guard() {
  local path="$1"

  cat > "$path" <<'EOF'
#!/usr/bin/env sh
set -eu

printf 'fake-claude guard should not be invoked by operator drill\n' >&2
exit 98
EOF
  chmod 755 "$path"
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

. "$SCRIPT_DIR/smoke-git-tripwire-lib.sh"

for cmd in go git install bash; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    fail "$cmd is required"
  fi
done

TMP_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/adp-release-operator-drill.XXXXXX")
SOURCE_ROOT="$TMP_ROOT/source"
DIST_DIR="$TMP_ROOT/dist"
INSTALL_BIN="$TMP_ROOT/install-bin"
FAKE_BIN="$TMP_ROOT/fake-bin"
PROJECT_ROOT="$TMP_ROOT/project"
ADP_HOME="$TMP_ROOT/adp-home"
ADP_RUNTIME_DIR="$TMP_ROOT/runtime"
GIT_TRIPWIRE_LOG="$TMP_ROOT/git-side-effects.log"

VERSION="0.1.0-operator-drill"
COMMIT="operator-drill-source-archive"
BUILD_DATE="2026-06-09T00:00:00Z"

cleanup() {
  rm -rf "$TMP_ROOT"
}
trap cleanup EXIT INT TERM

mkdir -p "$SOURCE_ROOT" "$DIST_DIR" "$INSTALL_BIN" "$FAKE_BIN" "$PROJECT_ROOT" "$ADP_HOME" "$ADP_RUNTIME_DIR"
printf 'module example.com/adp-release-operator-drill\n' > "$PROJECT_ROOT/go.mod"
printf 'package main\n' > "$PROJECT_ROOT/main.go"
write_fake_codex "$FAKE_BIN/codex"
write_fake_claude_guard "$FAKE_BIN/claude"

info "copying current repository files into a no-git operator source tree"
copy_current_tree "$SOURCE_ROOT"
assert_clean_source_boundary "$SOURCE_ROOT"
assert_release_docs_have_operator_commands "$SOURCE_ROOT"

info "syntax-checking release scripts from the clean source tree"
(
  cd "$SOURCE_ROOT"
  bash -n \
    scripts/check-all.sh \
    scripts/release-readiness-smoke.sh \
    scripts/release-rehearsal-smoke.sh \
    scripts/release-artifact-smoke.sh \
    scripts/release-operator-drill-smoke.sh
)

info "building documented source-archive release binary with explicit COMMIT"
build_release_binary "$SOURCE_ROOT" "$DIST_DIR/adp"
output=$("$DIST_DIR/adp" version)
assert_contains "$output" "adp $VERSION commit $COMMIT built $BUILD_DATE" "release binary version output"

info "generating and verifying release checksum"
sha256_write "$DIST_DIR/adp" "$DIST_DIR/adp.sha256"
sha256_verify "$DIST_DIR/adp.sha256" >/dev/null

info "installing release artifact into a temporary PATH"
install -m 0755 "$DIST_DIR/adp" "$INSTALL_BIN/adp"
export ADP_HOME
export ADP_RUNTIME_DIR
export PATH="$INSTALL_BIN:$FAKE_BIN:$PATH"
hash -r
if [ "$(command -v adp)" != "$INSTALL_BIN/adp" ]; then
  fail "temporary adp binary is not first on PATH"
fi
setup_git_tripwire "$FAKE_BIN" "$GIT_TRIPWIRE_LOG"

info "running provider-free operator handoff sequence"
output=$(run_adp "$TMP_ROOT" version)
assert_contains "$output" "adp $VERSION commit $COMMIT built $BUILD_DATE" "installed version output"
output=$(run_adp "$TMP_ROOT" init)
assert_contains "$output" "initialized ADP home" "init output"
output=$(run_adp "$TMP_ROOT" workspace add operator-a "$PROJECT_ROOT")
assert_contains "$output" 'workspace "operator-a" added' "workspace add output"
output=$(run_adp "$TMP_ROOT" workspace doctor operator-a)
assert_contains "$output" "operator-a" "workspace doctor output"
assert_contains "$output" "ok" "workspace doctor output"
output=$(run_adp "$TMP_ROOT" phase add --workspace operator-a --goal "operator release drill" p-operator "Operator Release Drill")
assert_contains "$output" "phase p-operator added" "phase add output"
output=$(run_adp "$TMP_ROOT" phase start --workspace operator-a p-operator)
assert_contains "$output" "phase p-operator status: active" "phase start output"
output=$(run_adp "$TMP_ROOT" tasks add --workspace operator-a --priority high --phase p-operator --description "operator release handoff" "Run operator drill")
assert_contains "$output" "task task-" "tasks add output"
TASK_ID=$(printf '%s\n' "$output" | sed -n 's/^task \(task-[^ ]*\) added$/\1/p')
if [ -z "$TASK_ID" ]; then
  fail "could not parse task id from: $output"
fi
reset_git_tripwire
output=$(run_adp "$TMP_ROOT" run codex --workspace operator-a --task "$TASK_ID" -- --operator-drill)
assert_contains "$output" "fake-codex" "fake codex output"
output=$(run_adp "$TMP_ROOT" phase accept --workspace operator-a p-operator --command "scripts/check-all.sh" --command "scripts/release-operator-drill-smoke.sh" --result passed --notes "operator drill smoke")
assert_contains "$output" "phase p-operator accepted: passed" "phase accept output"
output=$(run_adp "$TMP_ROOT" phase commit --workspace operator-a p-operator --hash 0123456789abcdef0123456789abcdef01234567 --message "operator drill evidence")
assert_contains "$output" "phase p-operator commit" "phase commit output"
output=$(run_adp "$TMP_ROOT" phase push --workspace operator-a p-operator --remote origin --branch main --result pushed)
assert_contains "$output" "phase p-operator push: origin/main pushed" "phase push output"
output=$(run_adp "$TMP_ROOT" phase status --workspace operator-a --format json)
assert_contains "$output" '"next_action": "plan_next_phase"' "phase status output"
assert_no_git_side_effects "operator drill handoff sequence"
assert_absent_project_artifacts "$PROJECT_ROOT"

info "release operator drill smoke passed"
