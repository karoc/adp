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

runtime_entry_count() {
  local runtime_dir="$1"

  find "$runtime_dir" -mindepth 1 -maxdepth 1 -type d | wc -l | tr -d '[:space:]'
}

assert_json_valid() {
  local output="$1"
  local label="$2"

  if ! printf '%s' "$output" | "$JSON_VALIDATOR" >/dev/null 2>&1; then
    printf '%s\n' "$output" >&2
    fail "$label was not valid JSON"
  fi
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

build_json_validator() {
  cat > "$TMP_ROOT/json-valid.go" <<'EOF'
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func main() {
	dec := json.NewDecoder(os.Stdin)
	dec.UseNumber()

	var value any
	if err := dec.Decode(&value); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			fmt.Fprintln(os.Stderr, "multiple JSON values")
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
EOF
  go build -o "$JSON_VALIDATOR" "$TMP_ROOT/json-valid.go"
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

run_adp_expect_fail() {
  local dir="$1"
  local output
  local code
  shift

  set +e
  output=$(cd "$dir" && adp "$@" 2>&1)
  code=$?
  set -e

  if [ "$code" = "0" ]; then
    printf '%s\n' "$output" >&2
    fail "adp $* unexpectedly succeeded"
  fi
  printf '%s\n' "$output"
}
