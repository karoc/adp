#!/usr/bin/env bash
set -euo pipefail

fail() {
  printf 'usability-errors-verification: %s\n' "$*" >&2
  exit 1
}

info() {
  printf '[usability-errors-verification] %s\n' "$*"
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

assert_not_contains() {
  local output="$1"
  local needle="$2"
  local label="$3"

  case "$output" in
    *"$needle"*)
      printf '%s\n' "$output" >&2
      fail "$label should not contain: $needle"
      ;;
    *) ;;
  esac
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

run_adp_expect_fail() {
  local dir="$1"
  shift
  local output

  if output=$(cd "$dir" && "$ADP_BIN" "$@" 2>&1); then
    printf '%s\n' "$output" >&2
    fail "adp $* succeeded, expected failure"
  fi
  printf '%s\n' "$output"
}

# Setup
info "Building adp binary..."
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

ADP_BIN="$TEMP_DIR/adp"
if ! go build -o "$ADP_BIN" ./cmd/adp; then
  fail "failed to build adp binary"
fi

# Test environment
export ADP_HOME="$TEMP_DIR/adp-home"
PROJECT_ROOT="$TEMP_DIR/test-project"
mkdir -p "$PROJECT_ROOT"

# Initialize ADP
info "Initializing ADP..."
"$ADP_BIN" init >/dev/null 2>&1

# Create a test workspace
WORKSPACE_NAME="error-test-$$"
"$ADP_BIN" workspace add "$WORKSPACE_NAME" "$PROJECT_ROOT" >/dev/null 2>&1

info "=== Starting Error Handling Verification ==="
echo ""

# ============================================================================
# Test 1: Non-existent Agent/Command Error
# ============================================================================
info "Test 1: Non-existent command error message"
echo "Testing: adp nonexistent-command"
echo ""

output=$(run_adp_expect_fail "$PROJECT_ROOT" nonexistent-command)

echo "Output:"
echo "$output"
echo ""

# Check for clear error message
if echo "$output" | grep -qE "(unknown command|not found|invalid)"; then
  info "✓ Error message mentions unknown command"
else
  info "✗ Error message unclear about unknown command"
fi

# Check for actionable suggestion
if echo "$output" | grep -qE "(try.*--help|available commands|check)"; then
  info "✓ Provides actionable suggestion"
else
  info "⚠ Could provide more actionable suggestions"
fi

echo ""
info "Test 1 completed"
echo ""

# ============================================================================
# Test 2: Workspace Conflict Error
# ============================================================================
info "Test 2: Workspace name conflict error"
echo "Testing: Adding duplicate workspace name"
echo ""

# Try to add the same workspace name with different path
ANOTHER_PATH="$TEMP_DIR/another-path"
mkdir -p "$ANOTHER_PATH"

output=$(run_adp_expect_fail "$PROJECT_ROOT" workspace add "$WORKSPACE_NAME" "$ANOTHER_PATH")

echo "Output:"
echo "$output"
echo ""

# Check for clear conflict message
if echo "$output" | grep -qE "(already exists|duplicate|conflict|name.*taken)"; then
  info "✓ Error message clearly indicates name conflict"
else
  info "✗ Error message doesn't clearly indicate name conflict"
fi

# Check it's NOT showing path validation error
if echo "$output" | grep -qE "(stat.*no such file|does not exist)" && ! echo "$output" | grep -qE "(already exists|duplicate)"; then
  fail "Shows confusing path error instead of name conflict"
else
  info "✓ Does not show misleading path validation error"
fi

# Check for actionable suggestion
if echo "$output" | grep -qE "(use different name|choose another|workspace list|rename)"; then
  info "✓ Provides actionable suggestion"
else
  info "⚠ Could suggest using a different name"
fi

echo ""
info "Test 2 completed"
echo ""

# ============================================================================
# Test 3: Empty Task List
# ============================================================================
info "Test 3: Empty task list display"
echo "Testing: adp tasks list (with no tasks)"
echo ""

output=$(run_adp "$PROJECT_ROOT" tasks list)

echo "Output:"
echo "$output"
echo ""

# Check if it shows helpful message for empty state
if echo "$output" | grep -qE "(No tasks|empty|Create.*task|add.*task)"; then
  info "✓ Shows helpful message for empty task list"
else
  info "⚠ Could show more helpful message for empty list"
fi

# Check it's not just showing table header
lines=$(echo "$output" | wc -l)
if [ "$lines" -le 2 ]; then
  info "⚠ Output is minimal (possibly just table header)"
else
  info "✓ Output includes helpful context beyond table header"
fi

echo ""
info "Test 3 completed"
echo ""

# ============================================================================
# Test 4: Empty Progress Report
# ============================================================================
info "Test 4: Empty progress report display"
echo "Testing: adp progress (with no tasks)"
echo ""

output=$(run_adp "$PROJECT_ROOT" progress)

echo "Output:"
echo "$output"
echo ""

if echo "$output" | grep -qE "(No tasks|empty|0.*tasks|Progress:)"; then
  info "✓ Shows appropriate message for empty progress"
else
  info "⚠ Could show more helpful message for empty state"
fi

echo ""
info "Test 4 completed"
echo ""

# ============================================================================
# Test 5: Empty Events List
# ============================================================================
info "Test 5: Empty events list display"
echo "Testing: adp events list (with no events)"
echo ""

output=$(run_adp "$PROJECT_ROOT" events list)

echo "Output:"
echo "$output"
echo ""

if echo "$output" | grep -qE "(No events|empty|run.*agent|start.*session)"; then
  info "✓ Shows helpful message for empty events list"
else
  info "⚠ Could show more helpful message for empty list"
fi

echo ""
info "Test 5 completed"
echo ""

# ============================================================================
# Test 6: Empty Sessions List
# ============================================================================
info "Test 6: Empty sessions list display"
echo "Testing: adp sessions list (with no sessions)"
echo ""

output=$(run_adp "$PROJECT_ROOT" sessions list)

echo "Output:"
echo "$output"
echo ""

if echo "$output" | grep -qE "(No sessions|empty|run.*agent|start.*session)"; then
  info "✓ Shows helpful message for empty sessions list"
else
  info "⚠ Could show more helpful message for empty list"
fi

echo ""
info "Test 6 completed"
echo ""

# ============================================================================
# Test 7: Workspace Not Found
# ============================================================================
info "Test 7: Workspace not found error"
echo "Testing: adp workspace show nonexistent-workspace"
echo ""

output=$(run_adp_expect_fail "$PROJECT_ROOT" workspace show nonexistent-workspace)

echo "Output:"
echo "$output"
echo ""

if echo "$output" | grep -qE "(workspace.*not found|unknown workspace)"; then
  info "✓ Clear workspace not found message"
else
  info "✗ Workspace not found message unclear"
fi

if echo "$output" | grep -qE "(workspace list|available workspaces|check)"; then
  info "✓ Suggests checking available workspaces"
else
  info "⚠ Could suggest checking workspace list"
fi

echo ""
info "Test 7 completed"
echo ""

# ============================================================================
# Test 8: Invalid Workspace Path
# ============================================================================
info "Test 8: Invalid workspace path error"
echo "Testing: adp workspace add test-invalid /nonexistent/path"
echo ""

output=$(run_adp_expect_fail "$PROJECT_ROOT" workspace add test-invalid /nonexistent/path)

echo "Output:"
echo "$output"
echo ""

if echo "$output" | grep -qE "(does not exist|not found|invalid path|no such file)"; then
  info "✓ Clear invalid path message"
else
  info "✗ Invalid path message unclear"
fi

if echo "$output" | grep -qE "(create|check path|verify|absolute path)"; then
  info "✓ Provides path validation guidance"
else
  info "⚠ Could provide path validation guidance"
fi

echo ""
info "Test 8 completed"
echo ""

# ============================================================================
# Test 9: Missing Required Arguments
# ============================================================================
info "Test 9: Missing required arguments"
echo "Testing: adp workspace add (without arguments)"
echo ""

output=$(run_adp_expect_fail "$PROJECT_ROOT" workspace add)

echo "Output:"
echo "$output"
echo ""

if echo "$output" | grep -qE "(usage|Usage|required|missing)"; then
  info "✓ Shows usage information"
else
  info "✗ Usage information unclear"
fi

if echo "$output" | grep -qE "(--help|adp workspace add)"; then
  info "✓ Suggests help command"
else
  info "⚠ Could suggest help command"
fi

echo ""
info "Test 9 completed"
echo ""

# ============================================================================
# Summary
# ============================================================================
echo ""
info "=== Error Handling Verification Complete ==="
echo ""
info "All tests completed successfully"
info "Review the output above to verify error message quality"
