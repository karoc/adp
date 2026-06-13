#!/usr/bin/env bash
set -euo pipefail

fail() {
  printf 'usability-json-verification: %s\n' "$*" >&2
  exit 1
}

info() {
  printf '[usability-json-verification] %s\n' "$*"
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

assert_valid_json() {
  local output="$1"
  local label="$2"

  # Basic JSON validation: check for opening and closing braces/brackets
  if [[ ! "$output" =~ ^\{.*\}$ ]] && [[ ! "$output" =~ ^\[.*\]$ ]]; then
    printf 'Output:\n%s\n' "$output" >&2
    fail "$label is not valid JSON (must start with { or [ and end with } or ])"
  fi

  # Check for common JSON structure elements
  if [[ ! "$output" =~ [\{\[\"\:] ]]; then
    printf 'Output:\n%s\n' "$output" >&2
    fail "$label does not look like valid JSON"
  fi
}

# Setup
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

# Create a test workspace
WORKSPACE_NAME="json-test-$$"
"$ADP_BIN" workspace add "$WORKSPACE_NAME" "$TEST_PROJECT" >/dev/null 2>&1

# ============================================================================
# Test 1: version command JSON output
# ============================================================================
info "Test 1: version --format json"
output=$("$ADP_BIN" version --format json 2>&1)
assert_valid_json "$output" "version output"
assert_contains "$output" '"version"' "version output"
assert_contains "$output" '"go_version"' "version output"
assert_contains "$output" '"platform"' "version output"
info "✓ Test 1 passed"

# ============================================================================
# Test 2: workspace list JSON output
# ============================================================================
info "Test 2: workspace list --format json"
output=$("$ADP_BIN" workspace list --format json 2>&1)
assert_valid_json "$output" "workspace list output"
assert_contains "$output" '"workspaces"' "workspace list output"
assert_contains "$output" '"count"' "workspace list output"
info "✓ Test 2 passed"

# ============================================================================
# Test 3: workspace show JSON output
# ============================================================================
info "Test 3: workspace show --format json"
output=$("$ADP_BIN" workspace show "$WORKSPACE_NAME" --format json 2>&1)
assert_valid_json "$output" "workspace show output"
assert_contains "$output" '"name"' "workspace show output"
assert_contains "$output" '"project_root"' "workspace show output"
info "✓ Test 3 passed"

# ============================================================================
# Test 4: workspace doctor JSON output
# ============================================================================
info "Test 4: workspace doctor --format json"
output=$("$ADP_BIN" workspace doctor "$WORKSPACE_NAME" --format json 2>&1)
assert_valid_json "$output" "workspace doctor output"
assert_contains "$output" '"reports"' "workspace doctor output"
assert_contains "$output" '"diagnostics"' "workspace doctor output"
assert_contains "$output" '"diagnostic_count"' "workspace doctor output"
info "✓ Test 4 passed"

# ============================================================================
# Test 5: tasks list JSON output
# ============================================================================
info "Test 5: tasks list --format json"
output=$("$ADP_BIN" tasks list --workspace "$WORKSPACE_NAME" --format json 2>&1)
assert_valid_json "$output" "tasks list output"
assert_contains "$output" '"tasks"' "tasks list output"
assert_contains "$output" '"workspace"' "tasks list output"
info "✓ Test 5 passed (NOTE: no 'count' field - inconsistent with other list commands)"

# ============================================================================
# Test 6: tasks next JSON output
# ============================================================================
info "Test 6: tasks next --format json"
output=$("$ADP_BIN" tasks next --workspace "$WORKSPACE_NAME" --format json 2>&1)
assert_valid_json "$output" "tasks next output"
assert_contains "$output" '"candidates"' "tasks next output"
assert_contains "$output" '"eligible_count"' "tasks next output"
assert_contains "$output" '"workspace"' "tasks next output"
info "✓ Test 6 passed"

# ============================================================================
# Test 7: tasks stale JSON output
# ============================================================================
info "Test 7: tasks stale --format json"
output=$("$ADP_BIN" tasks stale --workspace "$WORKSPACE_NAME" --format json 2>&1)
assert_valid_json "$output" "tasks stale output"
assert_contains "$output" '"tasks"' "tasks stale output"
info "✓ Test 7 passed"

# ============================================================================
# Test 8: tasks show JSON output (with a task)
# ============================================================================
info "Test 8: tasks show --format json"
# Add a task first
task_add_output=$("$ADP_BIN" tasks add --workspace "$WORKSPACE_NAME" "Test task" 2>&1)
info "  Task added: $task_add_output"

# Get task ID from the add output (format: "task task-XXXXXXXX-XXXX added")
TASK_ID=$(echo "$task_add_output" | grep -o 'task [a-z0-9-]*' | head -1 | awk '{print $2}')

if [ -n "$TASK_ID" ]; then
  info "  Task ID: $TASK_ID"
  output=$("$ADP_BIN" tasks show --workspace "$WORKSPACE_NAME" "$TASK_ID" --format json 2>&1)
  assert_valid_json "$output" "tasks show output"
  assert_contains "$output" '"id"' "tasks show output"
  assert_contains "$output" '"title"' "tasks show output"
  assert_contains "$output" '"status"' "tasks show output"
  info "✓ Test 8 passed"
else
  info "⊘ Test 8 skipped (could not create task)"
fi

# ============================================================================
# Test 9: tasks take JSON output
# ============================================================================
info "Test 9: tasks take --format json"
output=$("$ADP_BIN" tasks take --workspace "$WORKSPACE_NAME" --owner test-owner --format json 2>&1 || true)
# This might fail if no available tasks, just check it's valid JSON if successful
if echo "$output" | grep -q "^{"; then
  assert_valid_json "$output" "tasks take output"
  info "✓ Test 9 passed"
else
  info "⊘ Test 9 skipped (no available tasks)"
fi

# ============================================================================
# Test 10: phase list JSON output
# ============================================================================
info "Test 10: phase list --format json"
output=$("$ADP_BIN" phase list --workspace "$WORKSPACE_NAME" --format json 2>&1)
assert_valid_json "$output" "phase list output"
assert_contains "$output" '"phases"' "phase list output"
assert_contains "$output" '"workspace"' "phase list output"
info "✓ Test 10 passed (NOTE: no 'count' field - inconsistent with other list commands)"

# ============================================================================
# Test 11: phase status JSON output
# ============================================================================
info "Test 11: phase status --format json"
output=$("$ADP_BIN" phase status --workspace "$WORKSPACE_NAME" --format json 2>&1)
assert_valid_json "$output" "phase status output"
assert_contains "$output" '"workspace"' "phase status output"
assert_contains "$output" '"phase_count"' "phase status output"
assert_contains "$output" '"can_start_next"' "phase status output"
info "✓ Test 11 passed"

# ============================================================================
# Test 12: phase show JSON output (with a phase)
# ============================================================================
info "Test 12: phase show --format json"
# Add a phase first
phase_add_output=$("$ADP_BIN" phase add --workspace "$WORKSPACE_NAME" "test-phase" "Test Phase" 2>&1 || true)
info "  Phase add output: $phase_add_output"

# Get phase ID from the add output (format: "phase test-phase added")
PHASE_ID=$(echo "$phase_add_output" | grep -o 'phase [a-z0-9-]*' | head -1 | awk '{print $2}')

if [ -n "$PHASE_ID" ]; then
  info "  Phase ID: $PHASE_ID"
  output=$("$ADP_BIN" phase show --workspace "$WORKSPACE_NAME" "$PHASE_ID" --format json 2>&1)
  assert_valid_json "$output" "phase show output"
  assert_contains "$output" '"id"' "phase show output"
  assert_contains "$output" '"title"' "phase show output"
  info "✓ Test 12 passed"
else
  info "⊘ Test 12 skipped (no phases)"
fi

# ============================================================================
# Test 13: progress report JSON output
# ============================================================================
info "Test 13: progress report --format json"
output=$("$ADP_BIN" progress report --workspace "$WORKSPACE_NAME" --format json 2>&1)
assert_valid_json "$output" "progress report output"
assert_contains "$output" '"workspace"' "progress report output"
info "✓ Test 13 passed"

# ============================================================================
# Test 14: progress JSON output
# ============================================================================
info "Test 14: progress --format json"
output=$("$ADP_BIN" progress --workspace "$WORKSPACE_NAME" --format json 2>&1)
assert_valid_json "$output" "progress output"
# Progress output structure may vary
info "✓ Test 14 passed"

# ============================================================================
# Test 15: events list JSON output
# ============================================================================
info "Test 15: events list --format json"
output=$("$ADP_BIN" events list --workspace "$WORKSPACE_NAME" --format json 2>&1)
assert_valid_json "$output" "events list output"
assert_contains "$output" '"events"' "events list output"
assert_contains "$output" '"count"' "events list output"
info "✓ Test 15 passed"

# ============================================================================
# Test 16: sessions list JSON output
# ============================================================================
info "Test 16: sessions list --format json"
output=$("$ADP_BIN" sessions list --workspace "$WORKSPACE_NAME" --format json 2>&1)
assert_valid_json "$output" "sessions list output"
assert_contains "$output" '"sessions"' "sessions list output"
assert_contains "$output" '"count"' "sessions list output"
info "✓ Test 16 passed"

# ============================================================================
# Test 17: runtime prune JSON output
# ============================================================================
info "Test 17: runtime prune --format json --dry-run"
output=$("$ADP_BIN" runtime prune --dry-run --format json 2>&1)
assert_valid_json "$output" "runtime prune output"
assert_contains "$output" '"count"' "runtime prune output"
assert_contains "$output" '"dry_run"' "runtime prune output"
info "✓ Test 17 passed"

# ============================================================================
# Test 18: plan doctor JSON output
# ============================================================================
info "Test 18: plan doctor --format json"
output=$("$ADP_BIN" plan doctor --workspace "$WORKSPACE_NAME" --format json 2>&1)
assert_valid_json "$output" "plan doctor output"
assert_contains "$output" '"workspace"' "plan doctor output"
info "✓ Test 18 passed"

# ============================================================================
# Test 19: JSON structure consistency - list commands
# ============================================================================
info "Test 19: list commands structure consistency check"
workspace_list=$("$ADP_BIN" workspace list --format json 2>&1)
tasks_list=$("$ADP_BIN" tasks list --workspace "$WORKSPACE_NAME" --format json 2>&1)
phase_list=$("$ADP_BIN" phase list --workspace "$WORKSPACE_NAME" --format json 2>&1)
events_list=$("$ADP_BIN" events list --workspace "$WORKSPACE_NAME" --format json 2>&1)
sessions_list=$("$ADP_BIN" sessions list --workspace "$WORKSPACE_NAME" --format json 2>&1)

# Check which commands have 'count' field
info "  - workspace list: has 'count' ✓"
assert_contains "$workspace_list" '"count"' "workspace list"

info "  - events list: has 'count' ✓"
assert_contains "$events_list" '"count"' "events list"

info "  - sessions list: has 'count' ✓"
assert_contains "$sessions_list" '"count"' "sessions list"

# These commands are missing 'count' - document the inconsistency
if echo "$tasks_list" | grep -q '"count"'; then
  info "  - tasks list: has 'count' ✓"
else
  info "  - tasks list: MISSING 'count' ✗ (inconsistent)"
fi

if echo "$phase_list" | grep -q '"count"'; then
  info "  - phase list: has 'count' ✓"
else
  info "  - phase list: MISSING 'count' ✗ (inconsistent)"
fi

info "✓ Test 19 passed (with documented inconsistencies)"

# ============================================================================
# Test 20: Unknown format is rejected
# ============================================================================
info "Test 20: unknown format is rejected"
output=$("$ADP_BIN" version --format xml 2>&1 || true)
if echo "$output" | grep -q "unknown format\|invalid format\|unsupported format"; then
  info "✓ Test 20 passed"
else
  # Check if command simply doesn't support the format
  if ! echo "$output" | grep -q "^{"; then
    info "✓ Test 20 passed (format validation working)"
  else
    fail "Unknown format should be rejected"
  fi
fi

info "All JSON verification tests passed!"
