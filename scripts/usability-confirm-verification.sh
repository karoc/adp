#!/usr/bin/env bash
# Dangerous operation confirmation verification script
set -euo pipefail

fail() {
  printf '[%s] %s\n' "$(basename "$0" .sh)" "$*" >&2
  exit 1
}

info() {
  printf '[%s] %s\n' "$(basename "$0" .sh)" "$*"
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

export ADP_HOME="$TEMP_DIR/adp-home"
TEST_PROJECT="$TEMP_DIR/test-project"
mkdir -p "$TEST_PROJECT"

# ============================================================================
# Test 1: workspace remove with --yes
# ============================================================================

info "Test 1: workspace remove with --yes bypasses confirmation"
"$ADP_BIN" workspace add test-ws "$TEST_PROJECT" >/dev/null 2>&1

output=$("$ADP_BIN" workspace remove test-ws --yes 2>&1)

if ! echo "$output" | grep -q "workspace.*removed"; then
  fail "workspace remove --yes did not succeed"
fi

info "✓ Test 1 passed"

# ============================================================================
# Test 2: workspace remove without --yes in non-TTY fails
# ============================================================================

info "Test 2: workspace remove without --yes in non-TTY environment fails"
"$ADP_BIN" workspace add test-ws2 "$TEST_PROJECT" >/dev/null 2>&1

# Explicitly redirect stdin from /dev/null to simulate non-TTY
output=$("$ADP_BIN" workspace remove test-ws2 </dev/null 2>&1 || true)

if ! echo "$output" | grep -q "requires confirmation\|EOF"; then
  fail "workspace remove without --yes should fail in non-TTY (got: $output)"
fi

info "✓ Test 2 passed"

# ============================================================================
# Test 3: -y short form works
# ============================================================================

info "Test 3: -y short form works"
output=$("$ADP_BIN" workspace remove test-ws2 -y 2>&1)

if ! echo "$output" | grep -q "workspace.*removed"; then
  fail "workspace remove -y did not succeed"
fi

info "✓ Test 3 passed"

# ============================================================================
# Test 4: Error message is clear
# ============================================================================

info "Test 4: Confirmation shows warning-colored message"
"$ADP_BIN" workspace add test-ws3 "$TEST_PROJECT" >/dev/null 2>&1

# In non-interactive mode with /dev/null, should get EOF or confirmation required
output=$("$ADP_BIN" workspace remove test-ws3 </dev/null 2>&1 || true)

if ! echo "$output" | grep -q "Remove workspace"; then
  fail "confirmation message should show operation description"
fi

info "✓ Test 4 passed"

# ============================================================================
# Summary
# ============================================================================

info "All confirmation tests passed!"
info ""
info "Summary:"
info "  ✓ workspace remove --yes bypasses confirmation"
info "  ✓ workspace remove without --yes requires interaction"
info "  ✓ -y short form works"
info "  ✓ Confirmation message shows operation details"
