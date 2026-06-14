#!/usr/bin/env bash
# Command aliases verification script
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

# Initialize a git repo in test project
cd "$TEST_PROJECT"
git init -q
git config user.name "test"
git config user.email "test@example.com"
printf 'module example.com/test\n' > go.mod
printf 'package main\n' > main.go
git add .
git commit -q -m "init"
cd - >/dev/null

"$ADP_BIN" init >/dev/null 2>&1

# ============================================================================
# Test 1: ws alias for workspace
# ============================================================================

info "Test 1: 'ws' alias works for 'workspace'"
"$ADP_BIN" ws add test-ws "$TEST_PROJECT" >/dev/null 2>&1
output=$("$ADP_BIN" ws list 2>&1)

if ! echo "$output" | grep -q "test-ws"; then
  fail "'ws list' should work as alias for 'workspace list'"
fi

info "✓ Test 1 passed"

# ============================================================================
# Test 2: t alias for tasks
# ============================================================================

info "Test 2: 't' alias works for 'tasks'"
output=$("$ADP_BIN" t add --workspace test-ws "Test task" 2>&1)

if ! echo "$output" | grep -q "task.*added"; then
  fail "'t add' should work as alias for 'tasks add'"
fi

output=$("$ADP_BIN" t list --workspace test-ws 2>&1)
if ! echo "$output" | grep -q "Test task"; then
  fail "'t list' should work as alias for 'tasks list'"
fi

info "✓ Test 2 passed"

# ============================================================================
# Test 3: s alias for sessions
# ============================================================================

info "Test 3: 's' alias works for 'sessions'"
output=$("$ADP_BIN" s list --workspace test-ws 2>&1 || true)

# Should not show unknown command error
if echo "$output" | grep -q "unknown command"; then
  fail "'s' should be recognized as alias for 'sessions'"
fi

info "✓ Test 3 passed"

# ============================================================================
# Test 4: e alias for events
# ============================================================================

info "Test 4: 'e' alias works for 'events'"
output=$("$ADP_BIN" e list --workspace test-ws --limit 5 2>&1 || true)

# Should not show unknown command error
if echo "$output" | grep -q "unknown command"; then
  fail "'e' should be recognized as alias for 'events'"
fi

info "✓ Test 4 passed"

# ============================================================================
# Test 5: rt alias for runtime
# ============================================================================

info "Test 5: 'rt' alias works for 'runtime'"
output=$("$ADP_BIN" rt prune --older-than 0s --dry-run 2>&1 || true)

# Should not show unknown command error
if echo "$output" | grep -q "unknown command"; then
  fail "'rt' should be recognized as alias for 'runtime'"
fi

info "✓ Test 5 passed"

# ============================================================================
# Test 6: p alias for phase
# ============================================================================

info "Test 6: 'p' alias works for 'phase'"
output=$("$ADP_BIN" p list --workspace test-ws 2>&1 || true)

# Should not show unknown command error
if echo "$output" | grep -q "unknown command"; then
  fail "'p' should be recognized as alias for 'phase'"
fi

info "✓ Test 6 passed"

# ============================================================================
# Test 7: Aliases not shown in spelling suggestions
# ============================================================================

info "Test 7: Aliases not shown in spelling suggestions"
output=$("$ADP_BIN" workspac 2>&1 || true)

# Should suggest 'workspace', not 'ws'
if ! echo "$output" | grep -q "workspace"; then
  fail "should suggest 'workspace' for typo"
fi

if echo "$output" | grep -q "\\bws\\b"; then
  fail "should NOT suggest alias 'ws' in spelling suggestions"
fi

info "✓ Test 7 passed"

# ============================================================================
# Summary
# ============================================================================

info "All command alias tests passed!"
info ""
info "Summary:"
info "  ✓ ws → workspace"
info "  ✓ t → tasks"
info "  ✓ s → sessions"
info "  ✓ e → events"
info "  ✓ rt → runtime"
info "  ✓ p → phase"
info "  ✓ Aliases not shown in spelling suggestions"
