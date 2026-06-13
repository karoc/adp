#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd -- "$SCRIPT_DIR/.." && pwd)

fail() {
  printf 'release-artifact-smoke: %s\n' "$*" >&2
  exit 1
}

info() {
  printf '[release-artifact-smoke] %s\n' "$*"
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

assert_absent_project_artifacts() {
  local project_root="$1"
  local rel

  for rel in AGENTS.md CLAUDE.md .codex .claude .adp-runtime.yaml planning tasks.yaml phases.yaml progress.jsonl; do
    if [ -e "$project_root/$rel" ] || [ -L "$project_root/$rel" ]; then
      fail "project root was polluted with $rel"
    fi
  done
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

init_project_git() {
  git -C "$PROJECT_ROOT" init -q
  git -C "$PROJECT_ROOT" config user.name adp-smoke
  git -C "$PROJECT_ROOT" config user.email adp-smoke@example.invalid
  git -C "$PROJECT_ROOT" add go.mod main.go
  git -C "$PROJECT_ROOT" commit -q -m "init artifact smoke project"
}

release_ldflags() {
  local commit="$1"

  printf '%s' "-s -w"
  printf ' %s' "-X github.com/karoc/adp/internal/cli.Version=$VERSION"
  printf ' %s' "-X github.com/karoc/adp/internal/cli.Commit=$commit"
  printf ' %s' "-X github.com/karoc/adp/internal/cli.BuildDate=$BUILD_DATE"
}

build_adp() {
  local source_root="$1"
  local output_path="$2"
  local commit="$3"
  local ldflags

  ldflags=$(release_ldflags "$commit")
  mkdir -p "$(dirname -- "$output_path")"
  (
    cd "$source_root"
    GONOSUMDB='*' GOPROXY=off GOSUMDB=off \
      go build -buildvcs=false -trimpath -ldflags="$ldflags" -o "$output_path" ./cmd/adp
  )
  assert_executable "$output_path"
}

build_target_artifact() {
  local source_root="$1"
  local goos="$2"
  local goarch="$3"
  local output_path="$4"
  local ldflags

  ldflags=$(release_ldflags "$COMMIT")
  mkdir -p "$(dirname -- "$output_path")"
  (
    cd "$source_root"
    CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" GONOSUMDB='*' GOPROXY=off GOSUMDB=off \
      go build -buildvcs=false -trimpath -ldflags="$ldflags" -o "$output_path" ./cmd/adp
  )
  assert_file "$output_path"
}

artifact_name() {
  local goos="$1"
  local goarch="$2"
  local suffix=""

  if [ "$goos" = "windows" ]; then
    suffix=".exe"
  fi
  printf 'adp-%s-%s%s\n' "$goos" "$goarch" "$suffix"
}

sha256_file() {
  local path="$1"

  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$path" | awk '{print $1}'
    return
  fi
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$path" | awk '{print $1}'
    return
  fi
  fail "sha256sum or shasum is required"
}

plant_forbidden_state() {
  local source_root="$1"

  mkdir -p \
    "$source_root/ADP_HOME/logs" \
    "$source_root/ADP_HOME/workspaces/game-a" \
    "$source_root/ADP_RUNTIME_DIR/game-a-runtime/.codex" \
    "$source_root/.claude" \
    "$source_root/.codex" \
    "$source_root/planning" \
    "$source_root/logs" \
    "$source_root/credentials"

  touch \
    "$source_root/.envrc" \
    "$source_root/mvp.md" \
    "$source_root/ADP_HOME/logs/events.jsonl" \
    "$source_root/ADP_HOME/workspaces/game-a/tasks.yaml" \
    "$source_root/ADP_HOME/workspaces/game-a/phases.yaml" \
    "$source_root/ADP_HOME/workspaces/game-a/progress.jsonl" \
    "$source_root/ADP_RUNTIME_DIR/game-a-runtime/.adp-runtime.yaml" \
    "$source_root/ADP_RUNTIME_DIR/game-a-runtime/AGENTS.md" \
    "$source_root/ADP_RUNTIME_DIR/game-a-runtime/CLAUDE.md" \
    "$source_root/.codex/config.toml" \
    "$source_root/.claude/settings.json" \
    "$source_root/planning/tasks.yaml" \
    "$source_root/logs/adp.log" \
    "$source_root/credentials/token" \
    "$source_root/.bashrc" \
    "$source_root/.zshrc" \
    "$source_root/.profile"
}

stage_release_package() {
  local source_root="$1"
  local binary_path="$2"
  local package_root="$3"
  local checksum

  mkdir -p "$package_root/bin"
  cp "$binary_path" "$package_root/bin/adp"
  chmod 755 "$package_root/bin/adp"

  cp "$source_root/README.md" "$package_root/README.md"
  cp "$source_root/README.zh-CN.md" "$package_root/README.zh-CN.md"
  cp "$source_root/LICENSE" "$package_root/LICENSE"
  cp "$source_root/COMMERCIAL.md" "$package_root/COMMERCIAL.md"
  cp "$source_root/COMMERCIAL.zh-CN.md" "$package_root/COMMERCIAL.zh-CN.md"
  cp "$source_root/CONTRIBUTING.md" "$package_root/CONTRIBUTING.md"
  cp "$source_root/CONTRIBUTING.zh-CN.md" "$package_root/CONTRIBUTING.zh-CN.md"
  cp "$source_root/SECURITY.md" "$package_root/SECURITY.md"
  cp "$source_root/SECURITY.zh-CN.md" "$package_root/SECURITY.zh-CN.md"
  mkdir -p "$package_root/docs"
  cp "$source_root/docs/license-policy.md" "$package_root/docs/license-policy.md"
  cp "$source_root/docs/license-policy.zh-CN.md" "$package_root/docs/license-policy.zh-CN.md"
  cp "$source_root/docs/release-packaging.md" "$package_root/docs/release-packaging.md"
  cp "$source_root/docs/release-packaging.zh-CN.md" "$package_root/docs/release-packaging.zh-CN.md"
  cp "$source_root/docs/release-evidence.md" "$package_root/docs/release-evidence.md"
  cp "$source_root/docs/release-evidence.zh-CN.md" "$package_root/docs/release-evidence.zh-CN.md"

  checksum=$(sha256_file "$package_root/bin/adp")
  printf '%s  %s\n' "$checksum" "bin/adp" > "$package_root/SHA256SUMS"
  {
    printf 'version=%s\n' "$VERSION"
    printf 'commit=%s\n' "$COMMIT"
    printf 'build_date=%s\n' "$BUILD_DATE"
    printf 'binary_sha256=%s\n' "$checksum"
    printf 'real_provider_checks=not-run\n'
  } > "$package_root/RELEASE-EVIDENCE.txt"
}

assert_package_required_files() {
  local package_root="$1"

  assert_executable "$package_root/bin/adp"
  assert_file "$package_root/README.md"
  assert_file "$package_root/README.zh-CN.md"
  assert_file "$package_root/LICENSE"
  assert_file "$package_root/COMMERCIAL.md"
  assert_file "$package_root/COMMERCIAL.zh-CN.md"
  assert_file "$package_root/CONTRIBUTING.md"
  assert_file "$package_root/CONTRIBUTING.zh-CN.md"
  assert_file "$package_root/SECURITY.md"
  assert_file "$package_root/SECURITY.zh-CN.md"
  assert_file "$package_root/docs/license-policy.md"
  assert_file "$package_root/docs/license-policy.zh-CN.md"
  assert_file "$package_root/docs/release-packaging.md"
  assert_file "$package_root/docs/release-packaging.zh-CN.md"
  assert_file "$package_root/docs/release-evidence.md"
  assert_file "$package_root/docs/release-evidence.zh-CN.md"
  assert_file "$package_root/SHA256SUMS"
  assert_file "$package_root/RELEASE-EVIDENCE.txt"
}

assert_package_excludes_forbidden_state() {
  local package_tar="$1"
  local list_file="$2"
  local entry

  tar -tzf "$package_tar" > "$list_file"
  while IFS= read -r entry; do
    entry=${entry%/}
    case "$entry" in
      */.envrc|*/mvp.md|*/ADP_HOME|*/ADP_HOME/*|*/ADP_RUNTIME_DIR|*/ADP_RUNTIME_DIR/*)
        fail "package includes local-only state path: $entry"
        ;;
      */.adp-runtime.yaml|*/AGENTS.md|*/CLAUDE.md|*/.codex|*/.codex/*|*/.claude|*/.claude/*)
        fail "package includes runtime overlay path: $entry"
        ;;
      */planning|*/planning/*|*/logs|*/logs/*|*/tasks.yaml|*/phases.yaml|*/progress.jsonl|*/events.jsonl)
        fail "package includes planning, log, or task state path: $entry"
        ;;
      */credentials|*/credentials/*|*/.credentials|*/.credentials/*|*/credentials.*|*/token|*/token.*)
        fail "package includes credential-like path: $entry"
        ;;
      */.bashrc|*/.bash_profile|*/.zshrc|*/.zprofile|*/.profile|*/.config/fish/config.fish)
        fail "package includes shell startup file: $entry"
        ;;
    esac
  done < "$list_file"
}

write_fake_codex() {
  local path="$1"

  cat > "$path" <<'EOF'
#!/usr/bin/env sh
set -eu

printf 'fake-codex cwd=%s args=%s\n' "$(pwd)" "$*"
test "${ADP_WORKSPACE:-}" = "game-a"
test -n "${ADP_SESSION_ID:-}"
test -n "${ADP_RUNTIME_ROOT:-}"
test -n "${ADP_TASK_ID:-}"
test "$(pwd)" = "$ADP_RUNTIME_ROOT"
test -f "$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
test -f "$ADP_RUNTIME_ROOT/AGENTS.md"
test -f "$ADP_RUNTIME_ROOT/.codex/config.toml"
test -L "$ADP_RUNTIME_ROOT/go.mod"
test "$#" -eq 1
test "$1" = "--artifact-smoke"
EOF
  chmod 755 "$path"
}

write_fake_claude() {
  local path="$1"

  cat > "$path" <<'EOF'
#!/usr/bin/env sh
set -eu

printf 'fake-claude cwd=%s args=%s\n' "$(pwd)" "$*"
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

for cmd in go git tar awk; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    fail "$cmd is required"
  fi
done

TMP_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/adp-release-artifact-smoke.XXXXXX")
SOURCE_ROOT="$TMP_ROOT/source"
SOURCE_ARCHIVE_ROOT="$TMP_ROOT/source-archive"
DIST_DIR="$TMP_ROOT/dist"
PACKAGE_PARENT="$TMP_ROOT/package"
PACKAGE_NAME="adp-0.1.0-preview-artifact"
PACKAGE_ROOT="$PACKAGE_PARENT/$PACKAGE_NAME"
PACKAGE_TAR="$TMP_ROOT/$PACKAGE_NAME.tar.gz"
PACKAGE_LIST="$TMP_ROOT/package-files.txt"
EXTRACT_ROOT="$TMP_ROOT/extract"
INSTALL_BIN="$TMP_ROOT/install-bin"
FAKE_BIN="$TMP_ROOT/fake-bin"
PROJECT_ROOT="$TMP_ROOT/project"
ADP_HOME="$TMP_ROOT/adp-home"
ADP_RUNTIME_DIR="$TMP_ROOT/runtime"
GIT_TRIPWIRE_LOG="$TMP_ROOT/git-side-effects.log"

VERSION="0.1.0-preview.artifact"
COMMIT="0123456789abcdef0123456789abcdef01234567"
SOURCE_COMMIT="fedcba9876543210fedcba9876543210fedcba98"
BUILD_DATE="2026-06-09T00:00:00Z"

cleanup() {
  rm -rf "$TMP_ROOT"
}
trap cleanup EXIT INT TERM

mkdir -p "$SOURCE_ROOT" "$SOURCE_ARCHIVE_ROOT" "$DIST_DIR" "$INSTALL_BIN" "$FAKE_BIN" "$PROJECT_ROOT" "$ADP_HOME" "$ADP_RUNTIME_DIR"
printf 'module example.com/adp-artifact-smoke\n' > "$PROJECT_ROOT/go.mod"
printf 'package main\n' > "$PROJECT_ROOT/main.go"
init_project_git
write_fake_codex "$FAKE_BIN/codex"
write_fake_claude "$FAKE_BIN/claude"

info "copying repository files into temporary source trees"
copy_current_tree "$SOURCE_ROOT"
copy_current_tree "$SOURCE_ARCHIVE_ROOT"
plant_forbidden_state "$SOURCE_ROOT"

info "building preview artifacts with release ldflags"
build_adp "$SOURCE_ROOT" "$DIST_DIR/adp" "$COMMIT"
build_target_artifact "$SOURCE_ROOT" linux amd64 "$DIST_DIR/$(artifact_name linux amd64)"
build_target_artifact "$SOURCE_ROOT" darwin arm64 "$DIST_DIR/$(artifact_name darwin arm64)"
build_target_artifact "$SOURCE_ROOT" windows amd64 "$DIST_DIR/$(artifact_name windows amd64)"
output=$("$DIST_DIR/adp" version)
assert_contains "$output" "adp version $VERSION" "preview artifact version output"
assert_contains "$output" "commit: $COMMIT" "preview artifact version output"
assert_contains "$output" "built: $BUILD_DATE" "preview artifact version output"

info "verifying source archive build without .git and with explicit COMMIT"
if [ -d "$SOURCE_ARCHIVE_ROOT/.git" ]; then
  fail "source archive rehearsal unexpectedly contains .git"
fi
build_adp "$SOURCE_ARCHIVE_ROOT" "$TMP_ROOT/source-archive-bin/adp" "$SOURCE_COMMIT"
output=$("$TMP_ROOT/source-archive-bin/adp" version)
assert_contains "$output" "adp version $VERSION" "source archive version output"
assert_contains "$output" "commit: $SOURCE_COMMIT" "source archive version output"
assert_contains "$output" "built: $BUILD_DATE" "source archive version output"

info "staging release package and checking excluded local state"
stage_release_package "$SOURCE_ROOT" "$DIST_DIR/adp" "$PACKAGE_ROOT"
assert_package_required_files "$PACKAGE_ROOT"
tar -czf "$PACKAGE_TAR" -C "$PACKAGE_PARENT" "$PACKAGE_NAME"
assert_package_excludes_forbidden_state "$PACKAGE_TAR" "$PACKAGE_LIST"

info "installing package artifact into a temporary PATH"
mkdir -p "$EXTRACT_ROOT"
tar -xzf "$PACKAGE_TAR" -C "$EXTRACT_ROOT"
cp "$EXTRACT_ROOT/$PACKAGE_NAME/bin/adp" "$INSTALL_BIN/adp"
chmod 755 "$INSTALL_BIN/adp"

export ADP_HOME
export ADP_RUNTIME_DIR
export PATH="$INSTALL_BIN:$FAKE_BIN:$PATH"
hash -r
if [ "$(command -v adp)" != "$INSTALL_BIN/adp" ]; then
  fail "temporary adp binary is not first on PATH"
fi
setup_git_tripwire "$FAKE_BIN" "$GIT_TRIPWIRE_LOG"

info "running provider-free first-run rehearsal from installed artifact"
output=$(run_adp "$TMP_ROOT" version)
assert_contains "$output" "adp version $VERSION" "installed version output"
assert_contains "$output" "commit: $COMMIT" "installed version output"
assert_contains "$output" "built: $BUILD_DATE" "installed version output"
output=$(run_adp "$TMP_ROOT" init)
assert_contains "$output" "initialized ADP home" "init output"
output=$(run_adp "$TMP_ROOT" workspace add game-a "$PROJECT_ROOT")
assert_contains "$output" 'workspace "game-a" added' "workspace add output"
output=$(run_adp "$TMP_ROOT" workspace doctor game-a)
assert_contains "$output" "game-a" "workspace doctor output"
assert_contains "$output" "ok" "workspace doctor output"
assert_contains "$output" "no issues" "workspace doctor output"
output=$(run_adp "$TMP_ROOT" workspace doctor game-a --verbose)
assert_contains "$output" "workspace.git.root.detected" "workspace doctor verbose output"
output=$(run_adp "$TMP_ROOT" workspace doctor game-a --format json)
assert_contains "$output" '"code": "workspace.git.root.detected"' "workspace doctor json output"
output=$(run_adp "$TMP_ROOT" tasks add --workspace game-a --priority high --phase artifact-smoke "Validate artifact install")
assert_contains "$output" "task task-" "tasks add output"
TASK_ID=$(printf '%s\n' "$output" | sed -n 's/^task \(task-[^ ]*\) added$/\1/p')
if [ -z "$TASK_ID" ]; then
  fail "could not parse task id from: $output"
fi
reset_git_tripwire
output=$(run_adp "$TMP_ROOT" run codex --workspace game-a --task "$TASK_ID" -- --artifact-smoke)
assert_contains "$output" "fake-codex" "fake codex output"
output=$(run_adp "$TMP_ROOT" events list --workspace game-a --task "$TASK_ID" --limit 2)
assert_contains "$output" "run_started" "events output"
output=$(run_adp "$TMP_ROOT" sessions list --workspace game-a --agent codex --task "$TASK_ID")
assert_contains "$output" "codex" "sessions output"
output=$(run_adp "$TMP_ROOT" plan doctor --workspace game-a --format json)
assert_contains "$output" '"status": "ok"' "plan doctor output"
output=$(run_adp "$TMP_ROOT" progress --workspace game-a --format json)
assert_contains "$output" '"total"' "progress output"
assert_no_git_side_effects "artifact first-run rehearsal"
assert_absent_project_artifacts "$PROJECT_ROOT"

info "release artifact smoke passed"
