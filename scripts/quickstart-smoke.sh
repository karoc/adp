#!/usr/bin/env bash
set -euo pipefail

fail() {
  printf 'quickstart-smoke: %s\n' "$*" >&2
  exit 1
}

info() {
  printf '[quickstart-smoke] %s\n' "$*"
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

# Setup
info "Building adp binary..."
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

ADP_BIN="$TEMP_DIR/adp"
if ! go build -o "$ADP_BIN" ./cmd/adp; then
  fail "failed to build adp binary"
fi

# Test 1: Missing workspace name in non-interactive mode
info "Test 1: non-interactive mode requires workspace name"
if "$ADP_BIN" quickstart --non-interactive --project-root /tmp 2>&1; then
  fail "quickstart should fail without workspace name"
fi
info "✓ Test 1 passed"

# Test 2: Missing project root in non-interactive mode
info "Test 2: non-interactive mode requires project root"
if "$ADP_BIN" quickstart --non-interactive --workspace-name test-ws 2>&1; then
  fail "quickstart should fail without project root"
fi
info "✓ Test 2 passed"

# Test 3: Invalid workspace name
info "Test 3: validates workspace name"
if "$ADP_BIN" quickstart --non-interactive --workspace-name "invalid name" --project-root /tmp 2>&1; then
  fail "quickstart should fail with invalid workspace name"
fi
info "✓ Test 3 passed"

# Test 4: Non-existent project root
info "Test 4: validates project root exists"
if "$ADP_BIN" quickstart --non-interactive --workspace-name test-ws --project-root /this/does/not/exist 2>&1; then
  fail "quickstart should fail with non-existent project root"
fi
info "✓ Test 4 passed"

# Test 5: Successful non-interactive quickstart
info "Test 5: successful non-interactive quickstart"
PROJECT_ROOT="$TEMP_DIR/test-project"
WORKSPACE_NAME="smoke-test-$$-$(date +%s)"
mkdir -p "$PROJECT_ROOT"

# Use a separate ADP_HOME for this test
export ADP_HOME="$TEMP_DIR/adp-home"

output=$("$ADP_BIN" quickstart \
  --non-interactive \
  --workspace-name "$WORKSPACE_NAME" \
  --project-root "$PROJECT_ROOT" \
  --memory \
  --mcp 2>&1)

assert_contains "$output" "Initialized ADP home" "quickstart output"
assert_contains "$output" "created" "quickstart output"
assert_contains "$output" "complete" "quickstart output"

# Verify ADP home was created
if [ ! -d "$ADP_HOME" ]; then
  fail "ADP home was not created at $ADP_HOME"
fi

# Verify workspace can be shown
workspace_output=$("$ADP_BIN" workspace show "$WORKSPACE_NAME" 2>&1)
assert_contains "$workspace_output" "$WORKSPACE_NAME" "workspace show output"

info "✓ Test 5 passed"

# Test 6: --help flag
info "Test 6: help output"
help_output=$("$ADP_BIN" quickstart --help 2>&1 || true)
assert_contains "$help_output" "quickstart" "help output"
info "✓ Test 6 passed"

info "All tests passed!"
