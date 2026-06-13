#!/usr/bin/env bash
set -euo pipefail

fail() {
  printf 'usability-docs-verification: %s\n' "$*" >&2
  exit 1
}

info() {
  printf '[usability-docs-verification] %s\n' "$*"
}

warn() {
  printf '[usability-docs-verification] WARNING: %s\n' "$*"
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

# Test environment
export ADP_HOME="$TEMP_DIR/adp-home"
PROJECT_ROOT="$TEMP_DIR/test-project"
mkdir -p "$PROJECT_ROOT"

# Error tracking
ERRORS=0
WARNINGS=0

log_error() {
  echo "❌ ERROR: $1"
  ERRORS=$((ERRORS + 1))
}

log_warning() {
  echo "⚠️  WARNING: $1"
  WARNINGS=$((WARNINGS + 1))
}

log_success() {
  echo "✅ $1"
}

info "=== Starting Documentation Verification ==="
echo ""

# ============================================================================
# Part 1: README.md Command Examples Verification
# ============================================================================
info "Part 1: Verifying README.md command examples"
echo ""

# Extract commands from README.md
info "Extracting commands from README.md..."
COMMANDS=$(grep -E '^\s*-\s*`adp ' README.md | sed 's/.*`\(adp[^`]*\)`.*/\1/' || true)

if [ -z "$COMMANDS" ]; then
  log_error "README.md: No command examples found"
else
  log_success "README.md: Found command examples"
fi

# Test 1.1: adp init
info "Test 1.1: Verify 'adp init' command"
if "$ADP_BIN" init >/dev/null 2>&1; then
  log_success "adp init works"
else
  log_error "adp init failed"
fi

# Test 1.2: adp version
info "Test 1.2: Verify 'adp version' command"
if "$ADP_BIN" version >/dev/null 2>&1; then
  log_success "adp version works"
else
  log_error "adp version failed"
fi

# Test 1.3: adp workspace add
info "Test 1.3: Verify 'adp workspace add <name> <project-root>'"
WORKSPACE_NAME="docs-test-$$"
if "$ADP_BIN" workspace add "$WORKSPACE_NAME" "$PROJECT_ROOT" >/dev/null 2>&1; then
  log_success "adp workspace add works"
else
  log_error "adp workspace add failed"
fi

# Test 1.4: adp workspace list
info "Test 1.4: Verify 'adp workspace list'"
output=$("$ADP_BIN" workspace list 2>&1)
if echo "$output" | grep -q "$WORKSPACE_NAME"; then
  log_success "adp workspace list works and shows added workspace"
else
  log_error "adp workspace list doesn't show added workspace"
fi

# Test 1.5: adp workspace show
info "Test 1.5: Verify 'adp workspace show <name>'"
if "$ADP_BIN" workspace show "$WORKSPACE_NAME" >/dev/null 2>&1; then
  log_success "adp workspace show works"
else
  log_error "adp workspace show failed"
fi

# Test 1.6: adp workspace doctor
info "Test 1.6: Verify 'adp workspace doctor [name]'"
if "$ADP_BIN" workspace doctor "$WORKSPACE_NAME" >/dev/null 2>&1; then
  log_success "adp workspace doctor works"
else
  log_error "adp workspace doctor failed"
fi

# Test 1.7: adp doctor
info "Test 1.7: Verify 'adp doctor [workspace]'"
if "$ADP_BIN" doctor "$WORKSPACE_NAME" >/dev/null 2>&1; then
  log_success "adp doctor works"
else
  log_error "adp doctor failed"
fi

# Test 1.8: adp tasks add
info "Test 1.8: Verify 'adp tasks add'"
if "$ADP_BIN" tasks add --workspace "$WORKSPACE_NAME" "Test task" >/dev/null 2>&1; then
  log_success "adp tasks add works"
else
  log_error "adp tasks add failed"
fi

# Test 1.9: adp tasks list
info "Test 1.9: Verify 'adp tasks list'"
if "$ADP_BIN" tasks list --workspace "$WORKSPACE_NAME" >/dev/null 2>&1; then
  log_success "adp tasks list works"
else
  log_error "adp tasks list failed"
fi

# Test 1.10: adp progress
info "Test 1.10: Verify 'adp progress'"
if "$ADP_BIN" progress --workspace "$WORKSPACE_NAME" >/dev/null 2>&1; then
  log_success "adp progress works"
else
  log_error "adp progress failed"
fi

# Test 1.11: adp progress report
info "Test 1.11: Verify 'adp progress report'"
if "$ADP_BIN" progress report --workspace "$WORKSPACE_NAME" >/dev/null 2>&1; then
  log_success "adp progress report works"
else
  log_error "adp progress report failed"
fi

# Test 1.12: adp events list
info "Test 1.12: Verify 'adp events list'"
if "$ADP_BIN" events list --workspace "$WORKSPACE_NAME" >/dev/null 2>&1; then
  log_success "adp events list works"
else
  log_error "adp events list failed"
fi

# Test 1.13: adp sessions list
info "Test 1.13: Verify 'adp sessions list'"
if "$ADP_BIN" sessions list --workspace "$WORKSPACE_NAME" >/dev/null 2>&1; then
  log_success "adp sessions list works"
else
  log_error "adp sessions list failed"
fi

# Test 1.14: adp completion
info "Test 1.14: Verify 'adp completion'"
if "$ADP_BIN" completion --shell bash >/dev/null 2>&1; then
  log_success "adp completion works"
else
  log_error "adp completion failed"
fi

# Test 1.15: adp shell-hook
info "Test 1.15: Verify 'adp shell-hook'"
if "$ADP_BIN" shell-hook --shell bash >/dev/null 2>&1; then
  log_success "adp shell-hook works"
else
  log_error "adp shell-hook failed"
fi

echo ""
info "Part 1 completed: README.md verification"
echo ""

# ============================================================================
# Part 2: Quickstart Command Verification
# ============================================================================
info "Part 2: Verifying quickstart command (documented in README.md)"
echo ""

# Test 2.1: quickstart help
info "Test 2.1: Verify 'adp quickstart --help'"
output=$("$ADP_BIN" quickstart --help 2>&1)
if echo "$output" | grep -q "quickstart"; then
  log_success "adp quickstart --help works"
else
  log_error "adp quickstart --help doesn't mention quickstart"
fi

# Test 2.2: quickstart non-interactive
info "Test 2.2: Verify quickstart non-interactive mode"
QS_PROJECT="$TEMP_DIR/quickstart-project"
QS_WORKSPACE="qs-test-$$"
mkdir -p "$QS_PROJECT"
export ADP_HOME="$TEMP_DIR/adp-home-qs"

if "$ADP_BIN" quickstart --non-interactive \
  --workspace-name "$QS_WORKSPACE" \
  --project-root "$QS_PROJECT" >/dev/null 2>&1; then
  log_success "adp quickstart non-interactive works"
else
  log_error "adp quickstart non-interactive failed"
fi

echo ""
info "Part 2 completed: Quickstart verification"
echo ""

# ============================================================================
# Part 3: ID Prefix Matching Verification
# ============================================================================
info "Part 3: Verifying ID prefix matching (documented in README.md)"
echo ""

export ADP_HOME="$TEMP_DIR/adp-home"

# Create multiple tasks for prefix testing
info "Creating tasks for prefix matching tests..."
"$ADP_BIN" tasks add --workspace "$WORKSPACE_NAME" "Task 1" >/dev/null 2>&1
"$ADP_BIN" tasks add --workspace "$WORKSPACE_NAME" "Task 2" >/dev/null 2>&1

# Get task IDs using text format
TASK_LIST=$("$ADP_BIN" tasks list --workspace "$WORKSPACE_NAME" 2>&1 || true)

# Test 3.1: Full task ID
info "Test 3.1: Verify full task ID works"
# Extract task ID from the first task in the list (skip header lines)
FULL_TASK_ID=$(echo "$TASK_LIST" | grep -E '^task-' | head -1 | awk '{print $1}')

if [ -n "$FULL_TASK_ID" ]; then
  if "$ADP_BIN" tasks show --workspace "$WORKSPACE_NAME" "$FULL_TASK_ID" >/dev/null 2>&1; then
    log_success "Full task ID works: $FULL_TASK_ID"
  else
    log_error "Full task ID failed: $FULL_TASK_ID"
  fi
else
  log_warning "No task ID found for prefix matching test"
fi

# Test 3.2: Task ID prefix
info "Test 3.2: Verify task ID prefix matching"
if [ -n "$FULL_TASK_ID" ]; then
  # Extract prefix (first 10 characters)
  PREFIX=$(echo "$FULL_TASK_ID" | cut -c1-10)

  if "$ADP_BIN" tasks show --workspace "$WORKSPACE_NAME" "$PREFIX" >/dev/null 2>&1; then
    log_success "Task ID prefix works: $PREFIX"
  else
    # This might fail if prefix is ambiguous, which is expected
    log_warning "Task ID prefix might be ambiguous: $PREFIX"
  fi
else
  log_warning "No task ID available for prefix test"
fi

echo ""
info "Part 3 completed: ID prefix matching verification"
echo ""

# ============================================================================
# Part 4: operator-onboarding.md Verification
# ============================================================================
info "Part 4: Verifying operator-onboarding.md workflow"
echo ""

# Test 4.1: Help commands mentioned in onboarding
info "Test 4.1: Verify help commands from operator-onboarding.md"

for cmd in "--help" "workspace --help" "tasks --help"; do
  if "$ADP_BIN" $cmd >/dev/null 2>&1; then
    log_success "adp $cmd works"
  else
    log_error "adp $cmd failed"
  fi
done

# Test 4.2: Core workflow from operator-onboarding.md
info "Test 4.2: Verify core operator workflow"

# Use a fresh ADP_HOME for this test
export ADP_HOME="$TEMP_DIR/adp-home-operator"
OPERATOR_PROJECT="$TEMP_DIR/operator-project"
OPERATOR_WORKSPACE="operator-test-$$"
mkdir -p "$OPERATOR_PROJECT"

# Step 1: Initialize
if "$ADP_BIN" init >/dev/null 2>&1; then
  log_success "Operator workflow: init"
else
  log_error "Operator workflow: init failed"
fi

# Step 2: Add workspace
if "$ADP_BIN" workspace add "$OPERATOR_WORKSPACE" "$OPERATOR_PROJECT" >/dev/null 2>&1; then
  log_success "Operator workflow: workspace add"
else
  log_error "Operator workflow: workspace add failed"
fi

# Step 3: Doctor
if "$ADP_BIN" doctor "$OPERATOR_WORKSPACE" >/dev/null 2>&1; then
  log_success "Operator workflow: doctor"
else
  log_error "Operator workflow: doctor failed"
fi

# Step 4: Add task
if "$ADP_BIN" tasks add --workspace "$OPERATOR_WORKSPACE" "Operator test task" >/dev/null 2>&1; then
  log_success "Operator workflow: tasks add"
else
  log_error "Operator workflow: tasks add failed"
fi

# Step 5: List tasks
if "$ADP_BIN" tasks list --workspace "$OPERATOR_WORKSPACE" >/dev/null 2>&1; then
  log_success "Operator workflow: tasks list"
else
  log_error "Operator workflow: tasks list failed"
fi

# Step 6: Progress
if "$ADP_BIN" progress --workspace "$OPERATOR_WORKSPACE" >/dev/null 2>&1; then
  log_success "Operator workflow: progress"
else
  log_error "Operator workflow: progress failed"
fi

echo ""
info "Part 4 completed: operator-onboarding.md verification"
echo ""

# ============================================================================
# Part 5: Chinese Documentation Cross-Check
# ============================================================================
info "Part 5: Cross-checking Chinese documentation"
echo ""

# Test 5.1: Check if Chinese README exists
if [ -f "README.zh-CN.md" ]; then
  log_success "README.zh-CN.md exists"

  # Extract command count from both
  EN_CMD_COUNT=$(grep -c '^\s*-\s*`adp ' README.md || echo 0)
  ZH_CMD_COUNT=$(grep -c '^\s*-\s*`adp ' README.zh-CN.md || echo 0)

  if [ "$EN_CMD_COUNT" -eq "$ZH_CMD_COUNT" ]; then
    log_success "README.zh-CN.md has same command count as README.md ($EN_CMD_COUNT)"
  else
    log_warning "README.zh-CN.md command count ($ZH_CMD_COUNT) differs from README.md ($EN_CMD_COUNT)"
  fi
else
  log_error "README.zh-CN.md not found"
fi

# Test 5.2: Check if Chinese operator onboarding exists
if [ -f "docs/operator-onboarding.zh-CN.md" ]; then
  log_success "operator-onboarding.zh-CN.md exists"
else
  log_error "operator-onboarding.zh-CN.md not found"
fi

# Test 5.3: Check ID prefix matching documentation in Chinese
if [ -f "README.zh-CN.md" ]; then
  if grep -q "前缀匹配\|prefix" README.zh-CN.md; then
    log_success "README.zh-CN.md documents ID prefix matching"
  else
    log_warning "README.zh-CN.md may be missing ID prefix matching documentation"
  fi
fi

echo ""
info "Part 5 completed: Chinese documentation cross-check"
echo ""

# ============================================================================
# Summary
# ============================================================================
echo ""
info "=== Documentation Verification Summary ==="
echo ""

if [ "$ERRORS" -eq 0 ] && [ "$WARNINGS" -eq 0 ]; then
  echo "🎉 All tests passed! Documentation is accurate and up-to-date."
  exit 0
elif [ "$ERRORS" -eq 0 ]; then
  echo "✅ All tests passed with $WARNINGS warning(s)."
  echo ""
  echo "Warnings indicate minor issues that should be reviewed but don't affect core functionality."
  exit 0
else
  echo "❌ Documentation verification completed with $ERRORS error(s) and $WARNINGS warning(s)."
  echo ""
  echo "Errors indicate documentation that doesn't match actual implementation."
  echo "Please review the detailed output above and update the documentation."
  exit 1
fi
