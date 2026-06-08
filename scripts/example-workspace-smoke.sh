#!/usr/bin/env bash
set -euo pipefail

fail() {
  printf 'example-workspace-smoke: %s\n' "$*" >&2
  exit 1
}

info() {
  printf '[example-workspace-smoke] %s\n' "$*"
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

if ! command -v go >/dev/null 2>&1; then
  fail "Go is required to build cmd/adp"
fi

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd -- "$SCRIPT_DIR/.." && pwd)
TMP_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/adp-example-workspace-smoke.XXXXXX")
ADP_BIN="$TMP_ROOT/adp"

cleanup() {
  rm -rf "$TMP_ROOT"
}
trap cleanup EXIT INT TERM

PROJECT_ROOT="$TMP_ROOT/project"
ADP_HOME="$TMP_ROOT/adp-home"
ADP_RUNTIME_DIR="$TMP_ROOT/runtime"
WORKSPACE_DIR="$ADP_HOME/workspaces/game-a"

mkdir -p "$PROJECT_ROOT" "$ADP_HOME/workspaces" "$ADP_RUNTIME_DIR"
printf 'module example.com/adp-example-smoke\n' > "$PROJECT_ROOT/go.mod"
printf 'package main\n' > "$PROJECT_ROOT/main.go"

info "building temporary adp binary"
(cd "$REPO_ROOT" && go build -o "$ADP_BIN" ./cmd/adp)

info "copying basic workspace example"
cp -R "$REPO_ROOT/examples/basic-workspace" "$WORKSPACE_DIR"
rewrite_project_root "$WORKSPACE_DIR/workspace.yaml" "$PROJECT_ROOT"

export ADP_HOME
export ADP_RUNTIME_DIR

info "initializing ADP home"
output=$(run_adp "$REPO_ROOT" init)
assert_contains "$output" "initialized ADP home" "init output"

info "validating copied workspace"
output=$(run_adp "$REPO_ROOT" workspace doctor game-a)
assert_contains "$output" "game-a" "workspace doctor output"
assert_contains "$output" "ok" "workspace doctor output"

output=$(run_adp "$REPO_ROOT" workspace show game-a)
assert_contains "$output" "name: game-a" "workspace show output"
assert_contains "$output" "project_root: $PROJECT_ROOT" "workspace show output"

info "building runtime env from copied workspace"
env_output=$(run_adp "$REPO_ROOT" env game-a --cd)
runtime_root=$(parse_export "$env_output" ADP_RUNTIME_ROOT)
assert_contains "$env_output" "cd '$runtime_root'" "env --cd output"
assert_file "$runtime_root/.adp-runtime.yaml"
assert_symlink "$runtime_root/go.mod"
assert_symlink "$runtime_root/main.go"

info "example workspace smoke passed"
