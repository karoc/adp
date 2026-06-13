#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd -- "$SCRIPT_DIR/.." && pwd)

fail() {
  printf 'release-rehearsal-smoke: %s\n' "$*" >&2
  exit 1
}

info() {
  printf '[release-rehearsal-smoke] %s\n' "$*"
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

rewrite_project_root() {
  local workspace_yaml="$1"
  local project_root="$2"
  local tmp_file="${workspace_yaml}.tmp"

  awk -v project_root="$project_root" '
    /^project:[[:space:]]*$/ {
      in_project = 1
      print
      next
    }
    in_project && /^[[:space:]]*root:[[:space:]]*/ {
      print "  root: " project_root
      in_project = 0
      next
    }
    /^[^[:space:]]/ {
      in_project = 0
    }
    {
      print
    }
  ' "$workspace_yaml" > "$tmp_file"
  mv "$tmp_file" "$workspace_yaml"
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
test "$(pwd)" = "$ADP_RUNTIME_ROOT"
test -f "$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
test -f "$ADP_RUNTIME_ROOT/AGENTS.md"
test -f "$ADP_RUNTIME_ROOT/.codex/config.toml"
test -L "$ADP_RUNTIME_ROOT/go.mod"
test "$#" -eq 1
test "$1" = "--rehearsal"
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

init_project_git() {
  git -C "$PROJECT_ROOT" init -q
  git -C "$PROJECT_ROOT" config user.name adp-smoke
  git -C "$PROJECT_ROOT" config user.email adp-smoke@example.invalid
  git -C "$PROJECT_ROOT" add go.mod main.go
  git -C "$PROJECT_ROOT" commit -q -m "init release rehearsal project"
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

. "$SCRIPT_DIR/smoke-git-tripwire-lib.sh"

if ! command -v go >/dev/null 2>&1; then
  fail "Go is required to build cmd/adp"
fi

if ! command -v git >/dev/null 2>&1; then
  fail "Git is required for release rehearsal smoke"
fi

TMP_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/adp-release-rehearsal.XXXXXX")
CHECKOUT_ROOT="$TMP_ROOT/source"
ADP_BIN="$CHECKOUT_ROOT/dist/adp"
PROJECT_ROOT="$TMP_ROOT/project"
ADP_HOME="$TMP_ROOT/adp-home"
ADP_RUNTIME_DIR="$TMP_ROOT/runtime"
FAKE_BIN="$TMP_ROOT/bin"
GIT_TRIPWIRE_LOG="$TMP_ROOT/git-side-effects.log"
WORKSPACE_DIR="$ADP_HOME/workspaces/game-a"

cleanup() {
  rm -rf "$TMP_ROOT"
}
trap cleanup EXIT INT TERM

mkdir -p "$PROJECT_ROOT" "$ADP_HOME/workspaces" "$ADP_RUNTIME_DIR" "$FAKE_BIN"
printf 'module example.com/adp-release-rehearsal\n' > "$PROJECT_ROOT/go.mod"
printf 'package main\n' > "$PROJECT_ROOT/main.go"
init_project_git
write_fake_codex "$FAKE_BIN/codex"
write_fake_claude "$FAKE_BIN/claude"

info "copying current repository files into a clean rehearsal workspace"
copy_current_tree "$CHECKOUT_ROOT"
git -C "$CHECKOUT_ROOT" init -q

export ADP_HOME
export ADP_RUNTIME_DIR
export PATH="$FAKE_BIN:$PATH"

info "building preview binary with release ldflags"
mkdir -p "$CHECKOUT_ROOT/dist"
VERSION="0.1.0-rehearsal"
COMMIT="rehearsal"
BUILD_DATE="2026-06-09T00:00:00Z"
LDFLAGS="-s -w"
LDFLAGS="$LDFLAGS -X github.com/karoc/adp/internal/cli.Version=$VERSION"
LDFLAGS="$LDFLAGS -X github.com/karoc/adp/internal/cli.Commit=$COMMIT"
LDFLAGS="$LDFLAGS -X github.com/karoc/adp/internal/cli.BuildDate=$BUILD_DATE"
(cd "$CHECKOUT_ROOT" && go build -trimpath -ldflags="$LDFLAGS" -o "$ADP_BIN" ./cmd/adp)
output=$("$ADP_BIN" version)
assert_contains "$output" "adp version $VERSION" "version output"
assert_contains "$output" "commit: $COMMIT" "version output"
assert_contains "$output" "built: $BUILD_DATE" "version output"

info "checking copied docs and file limits"
(cd "$CHECKOUT_ROOT" && scripts/check-file-lines.sh)
(cd "$CHECKOUT_ROOT" && scripts/check-docs-bilingual.sh)

info "bootstrapping copied example workspace with isolated ADP paths"
cp -R "$CHECKOUT_ROOT/examples/basic-workspace" "$WORKSPACE_DIR"
rewrite_project_root "$WORKSPACE_DIR/workspace.yaml" "$PROJECT_ROOT"
output=$(run_adp "$CHECKOUT_ROOT" init)
assert_contains "$output" "initialized ADP home" "init output"
output=$(run_adp "$CHECKOUT_ROOT" workspace doctor game-a)
assert_contains "$output" "game-a" "workspace doctor output"
assert_contains "$output" "ok" "workspace doctor output"
assert_contains "$output" "no issues" "workspace doctor output"
output=$(run_adp "$CHECKOUT_ROOT" workspace doctor game-a --verbose)
assert_contains "$output" "workspace.git.root.detected" "workspace doctor verbose output"
output=$(run_adp "$CHECKOUT_ROOT" workspace doctor game-a --format json)
assert_contains "$output" '"code": "workspace.git.root.detected"' "workspace doctor json output"
output=$(run_adp "$CHECKOUT_ROOT" env game-a --cd)
assert_contains "$output" "ADP_RUNTIME_ROOT" "env output"
output=$(run_adp "$CHECKOUT_ROOT" run codex --workspace game-a -- --rehearsal)
assert_contains "$output" "fake-codex" "fake codex output"
output=$(run_adp "$CHECKOUT_ROOT" sessions list --workspace game-a --agent codex)
assert_contains "$output" "codex" "sessions output"
assert_absent_project_artifacts "$PROJECT_ROOT"

info "checking release phase evidence remains local"
setup_git_tripwire "$FAKE_BIN" "$GIT_TRIPWIRE_LOG"
output=$(run_adp "$CHECKOUT_ROOT" phase add --workspace game-a p-rehearsal "Release rehearsal")
assert_contains "$output" "phase p-rehearsal added" "phase add output"
output=$(run_adp "$CHECKOUT_ROOT" phase start --workspace game-a p-rehearsal)
assert_contains "$output" "phase p-rehearsal status: active" "phase start output"
reset_git_tripwire
output=$(run_adp "$CHECKOUT_ROOT" phase accept --workspace game-a p-rehearsal --command "scripts/check-all.sh" --result passed --notes "release rehearsal")
assert_contains "$output" "phase p-rehearsal accepted: passed" "phase accept output"
output=$(run_adp "$CHECKOUT_ROOT" phase commit --workspace game-a p-rehearsal --hash 0123456789abcdef0123456789abcdef01234567 --message "release rehearsal evidence")
assert_contains "$output" "phase p-rehearsal commit" "phase commit output"
output=$(run_adp "$CHECKOUT_ROOT" phase push --workspace game-a p-rehearsal --remote origin --branch main --result pushed)
assert_contains "$output" "phase p-rehearsal push: origin/main pushed" "phase push output"
assert_no_git_side_effects "release rehearsal phase evidence recording"

info "release rehearsal smoke passed"
