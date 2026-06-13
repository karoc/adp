#!/usr/bin/env bash
# Verification Script Template
# Copy this template and customize for your verification task

set -euo pipefail

# ============================================================================
# Helper Functions
# ============================================================================

fail() {
  printf '[%s] %s\n' "$(basename "$0" .sh)" "$*" >&2
  exit 1
}

info() {
  printf '[%s] %s\n' "$(basename "$0" .sh)" "$*"
}

assert_contains() {
  local output="$1"
  local needle="$2"
  local label="$3"

  case "$output" in
    *"$needle"*) ;;
    *)
      printf 'Output:\n%s\n' "$output" >&2
      fail "$label missing expected text: $needle"
      ;;
  esac
}

assert_success() {
  local exit_code="$1"
  local label="$2"

  if [ "$exit_code" -ne 0 ]; then
    fail "$label failed with exit code $exit_code"
  fi
}

# ============================================================================
# Setup
# ============================================================================

info "Building adp binary..."
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

ADP_BIN="$TEMP_DIR/adp"
if ! go build -o "$ADP_BIN" ./cmd/adp; then
  fail "failed to build adp binary"
fi

# Setup test environment
export ADP_HOME="$TEMP_DIR/adp-home"
TEST_PROJECT="$TEMP_DIR/test-project"
mkdir -p "$TEST_PROJECT"

# Create test workspace
WORKSPACE_NAME="test-$$"
"$ADP_BIN" workspace add "$WORKSPACE_NAME" "$TEST_PROJECT" >/dev/null 2>&1

# ============================================================================
# Test Cases
# ============================================================================

# Test 1: Example test
info "Test 1: Example test description"
output=$("$ADP_BIN" version 2>&1)
assert_contains "$output" "adp" "version output"
info "✓ Test 1 passed"

# Test 2: Add more tests here
info "Test 2: Another test description"
# Your test logic here
info "✓ Test 2 passed"

# ============================================================================
# Summary
# ============================================================================

info "All verification tests passed!"
