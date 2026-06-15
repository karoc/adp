#!/usr/bin/env bash
# Phase 5 最终验收测试脚本
# 覆盖 Day 1-2（颜色）、Day 3（确认）、Day 4（建议）的运行时行为
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd -- "$SCRIPT_DIR/.." && pwd)

echo "=== Phase 5 最终验收测试 ==="
echo ""

# 构建二进制
echo "1. 构建 ADP 二进制..."
go build -o /tmp/adp-phase5-acceptance "$REPO_ROOT/cmd/adp"
echo "  ✓ 构建成功"
echo ""

# 准备测试环境
export ADP_HOME=/tmp/adp-phase5-test
PROJECTS=/tmp/phase5-projects
rm -rf $ADP_HOME $PROJECTS
mkdir -p $PROJECTS/{ws-a,ws-b,ws-c,ws-check}
ADP=/tmp/adp-phase5-acceptance

# 辅助：断言输出包含某字符串
assert_contains() {
    local label="$1" expected="$2" actual="$3"
    if echo "$actual" | grep -q -- "$expected"; then
        echo "  ✓ $label"
    else
        echo "  ✗ $label"
        echo "    期望包含: $expected"
        echo "    实际输出: $actual"
        exit 1
    fi
}

# 辅助：断言输出不包含某字符串
assert_not_contains() {
    local label="$1" expected="$2" actual="$3"
    if echo "$actual" | grep -q -- "$expected"; then
        echo "  ✗ $label"
        echo "    期望不包含: $expected"
        echo "    实际输出: $actual"
        exit 1
    else
        echo "  ✓ $label"
    fi
}

echo "2. [Day 1-2] 颜色输出运行时验证..."
$ADP init >/dev/null 2>&1

# 场景 A: PTY + 无 NO_COLOR → 应有 ANSI 颜色码
color_count=$(script -qec "$ADP workspace add color-pty $PROJECTS/ws-a" /dev/null 2>&1 | grep -c $'\033\[' || true)
if [ "$color_count" -gt 0 ]; then
    echo "  ✓ PTY + 无 NO_COLOR → 颜色开启（$color_count 个 ANSI 序列）"
else
    echo "  ✗ PTY 环境颜色未启用"
    exit 1
fi

# 场景 B: PTY + NO_COLOR=1 → 应无 ANSI 颜色码
rm -rf $ADP_HOME && $ADP init >/dev/null 2>&1
color_count=$(script -qec "NO_COLOR=1 $ADP workspace add color-nocolor $PROJECTS/ws-a" /dev/null 2>&1 | grep -c $'\033\[' || true)
if [ "$color_count" -eq 0 ]; then
    echo "  ✓ PTY + NO_COLOR=1 → 颜色正确禁用"
else
    echo "  ✗ NO_COLOR=1 未正确禁用颜色"
    exit 1
fi

# 场景 C: 管道（非 TTY）→ 应无 ANSI 颜色码（优雅降级）
rm -rf $ADP_HOME && $ADP init >/dev/null 2>&1
color_count=$($ADP workspace add color-pipe $PROJECTS/ws-a 2>&1 | grep -c $'\033\[' || true)
if [ "$color_count" -eq 0 ]; then
    echo "  ✓ 管道输出 → 颜色正确禁用（不污染管道）"
else
    echo "  ✗ 管道输出包含 ANSI 码（会破坏脚本解析）"
    exit 1
fi
echo ""

echo "3. [Day 3] 危险操作确认验证..."
rm -rf $ADP_HOME && $ADP init >/dev/null 2>&1
$ADP workspace add ws-a $PROJECTS/ws-a >/dev/null 2>&1

# workspace remove 非 TTY 无 --yes → 应拒绝并提示
out=$(echo "n" | $ADP workspace remove ws-a 2>&1 || true)
assert_contains "workspace remove 非 TTY 要求确认" "requires confirmation" "$out"

# workspace remove --yes → 跳过确认
out=$($ADP workspace remove ws-a --yes 2>&1)
assert_contains "workspace remove --yes 跳过确认" "removed" "$out"

# runtime prune --include-kept 非 TTY 无 --yes → 应拒绝
$ADP workspace add ws-b $PROJECTS/ws-b >/dev/null 2>&1
mkdir -p $ADP_HOME/runtime/ws-b-20260615T120000
touch $ADP_HOME/runtime/ws-b-20260615T120000/.keep
out=$(echo "n" | $ADP runtime prune --older-than 0s --include-kept 2>&1 || true)
assert_contains "runtime prune --include-kept 要求确认" "requires confirmation" "$out"

# runtime prune --dry-run → 不要求确认，正常扫描
out=$($ADP runtime prune --older-than 0s --include-kept --dry-run 2>&1)
assert_contains "runtime prune --dry-run 不要求确认" "dry run" "$out"

# runtime prune --include-kept --yes → 跳过确认
out=$($ADP runtime prune --older-than 0s --include-kept --yes 2>&1)
assert_contains "runtime prune --yes 跳过确认" "Scanning" "$out"
echo ""

echo "4. [Day 4] 成功消息建议验证..."
# phase add 建议
$ADP workspace add ws-c $PROJECTS/ws-c >/dev/null 2>&1
out=$($ADP phase add --workspace ws-c phase1 "Test Phase" 2>&1)
assert_contains "phase add 显示下一步建议" "Next steps" "$out"
assert_contains "phase add 建议含 show 命令" "phase show phase1" "$out"

# quickstart 非交互模式建议
out=$($ADP quickstart --non-interactive --workspace-name ws-check --project-root $PROJECTS/ws-check 2>&1)
assert_contains "quickstart 显示操作员指南建议" "operator-onboarding" "$out"
assert_contains "quickstart 显示 agent 启动建议" "run codex" "$out"
echo ""

echo "5. 单元测试与 CI 检查..."
cd "$REPO_ROOT"
if go test ./internal/output/... ./internal/cli/... >/dev/null 2>&1; then
    echo "  ✓ output + cli 包单元测试通过"
else
    echo "  ✗ 单元测试失败"
    exit 1
fi
if "$REPO_ROOT/scripts/check-all.sh" >/dev/null 2>&1; then
    echo "  ✓ 全部 CI 检查通过（含双语文档、文件行数、vet）"
else
    echo "  ✗ CI 检查失败"
    exit 1
fi
echo ""

# 清理
rm -rf $ADP_HOME $PROJECTS /tmp/adp-phase5-acceptance

echo "=== Phase 5 验收测试全部通过 ✅ ==="
echo ""
echo "功能验收结果:"
echo "  ✓ Day 1-2: 颜色输出（PTY 开启 / NO_COLOR 禁用 / 管道降级）"
echo "  ✓ Day 3: 危险操作确认（workspace remove + runtime prune --include-kept）"
echo "  ✓ Day 4: 成功消息建议（phase add + quickstart）"
echo "  ✓ 单元测试（output + cli 包）"
echo "  ✓ CI 检查（check-all.sh）"
