# 文档错误和改进清单

**生成日期**: 2026-06-13  
**验证脚本**: `scripts/usability-docs-verification.sh`  
**验证报告**: `docs/verification/documentation-verification.md`

---

## ❌ 发现的错误

### 无严重错误

所有验证的命令示例都与实际实现一致，未发现错误或不准确的文档。

---

## ⚠️ 发现的问题

### 问题 1: 中文 README 缺少 ID 前缀匹配文档

**严重程度**: 中等  
**影响**: 中文用户不了解 ID 前缀匹配功能  

**位置**: 
- ✅ 英文版有文档: `README.md` 第 55-84 行
- ❌ 中文版缺失: `README.zh-CN.md` (无对应章节)

**英文版内容** (README.md:55-84):
```markdown
### ID Prefix Matching

Task and session IDs support prefix matching for convenience...

```bash
# Full task ID
adp tasks show task-20260611-0001

# Prefix matching (if unique)
adp tasks show task-2026
```

**修复建议**:

在 `README.zh-CN.md` 第 54 行之后（"## 快速开始" 之前）添加：

```markdown
### ID 前缀匹配

Task 和 session ID 支持前缀匹配以便使用。接受 task ID 或 session ID 的命令会匹配最短的唯一前缀：

```bash
# 完整的 task ID
adp tasks show task-20260611-0001

# 前缀匹配（如果唯一）
adp tasks show task-2026
adp tasks claim task-001 --owner alice --lease 2h

# 完整的 session ID
adp sessions show session-20260611T102030-abc123

# 前缀匹配（如果唯一）
adp sessions show 20260611T10
adp sessions restore-plan 2026061
```

当前缀匹配多个 ID 时，ADP 会返回错误并列出所有匹配项：

```bash
adp tasks show task-20
# Error: ambiguous task ID "task-20", matches:
#   - task-20260611-0001
#   - task-20260612-0002
```

前缀匹配适用于所有 task 和 session 命令，包括 `tasks show`、`tasks claim`、`tasks renew`、`tasks release`、`tasks done`、`tasks block`、`sessions show`、`sessions restore-plan`、`sessions resume-plan`、`events list --task`、`events list --session` 和 `run --task`。
```

**预计工作量**: 10-15 分钟

---

### 问题 2: Operator Onboarding 中文版可能缺少 ID 前缀匹配

**严重程度**: 低  
**影响**: 中文操作员不了解便捷功能  

**位置**: 
- ✅ 英文版有文档: `docs/operator-onboarding.md` 第 77-100 行
- ⚠️ 中文版需确认: `docs/operator-onboarding.zh-CN.md`

**修复建议**: 检查中文版是否有对应章节，如无则添加。

---

## 💡 改进建议

### 建议 1: ID 前缀匹配增强说明

**优先级**: P2  
**位置**: `README.md` 第 55-84 行

**当前问题**: 文档说明了前缀匹配功能，但缺少：
- 最短前缀长度要求
- 如何选择合适的前缀长度
- 前缀歧义的实际示例

**建议内容**:

在 README.md 第 84 行之后添加：

```markdown
#### Best Practices for Prefix Matching

**Minimum prefix length**: While ADP accepts any prefix length, using at least 4-8 characters helps avoid ambiguity:
- Task IDs: Use at least `task-YYYY` (9 chars) for date-based uniqueness
- Session IDs: Use at least 8-12 characters including the date portion

**Example workflow**:
```bash
# List tasks to see IDs
adp tasks list --workspace my-project

# Use a safe prefix (date-based)
adp tasks show task-20260613    # Matches all tasks from June 13
adp tasks show task-20260613-00 # Narrows to first few tasks

# If ambiguous, ADP guides you
$ adp tasks show task-20260613
Error: ambiguous task ID "task-20260613", matches:
  - task-20260613-0001
  - task-20260613-0002
  
# Add more characters to disambiguate
adp tasks show task-20260613-0001
```

**Tip**: Start with date-based prefixes (`task-YYYYMMDD`) for day-level uniqueness, then add more digits as needed.
```

**预计工作量**: 15-20 分钟

---

### 建议 2: Quickstart 成功指标文档

**优先级**: P2  
**位置**: `README.md` 第 86-99 行

**当前问题**: 文档说明了如何运行 quickstart，但没有说明：
- 成功后应该看到什么输出
- 如何确认设置完成
- 推荐的下一步操作

**建议内容**:

在 README.md 第 99 行之后添加：

```markdown
#### What to Expect

On successful quickstart, you'll see:
```
✓ Initialized ADP home at ~/.adp
✓ Created workspace: my-project
✓ Workspace ready at /path/to/project

Next steps:
  adp tasks add --workspace my-project "Your first task"
  adp run <agent> --workspace my-project
```

Verify the setup:
```bash
# Confirm workspace is registered
adp workspace show my-project

# Check workspace health
adp doctor my-project

# View workspace from project directory
cd /path/to/project
adp workspace show  # Auto-detects workspace
```

If quickstart fails, check:
- Project root exists and is a directory
- Workspace name is valid (alphanumeric, dash, underscore, dot)
- No existing workspace with the same name
```

**预计工作量**: 20-25 分钟

---

### 建议 3: 文档同步检查清单

**优先级**: P3  
**位置**: 项目维护流程 (可以添加到 CONTRIBUTING.md)

**当前问题**: 中英文档可能不同步

**建议内容**:

在 `CONTRIBUTING.md` 或创建 `docs/documentation-guide.md` 添加：

```markdown
## Documentation Update Checklist

When updating English documentation:
- [ ] Update corresponding Chinese version (*.zh-CN.md)
- [ ] Verify command examples are identical
- [ ] Check code block formatting consistency
- [ ] Verify internal links work in both versions
- [ ] Run documentation verification script:
  ```bash
  ./scripts/usability-docs-verification.sh
  ```
- [ ] Update line number references if structure changed

### Documentation Verification

Before committing documentation changes:
```bash
# Verify all command examples work
./scripts/usability-docs-verification.sh

# Check Chinese command count matches English
grep -c '^\s*-\s*`adp ' README.md
grep -c '^\s*-\s*`adp ' README.zh-CN.md
# Numbers should match
```

### Quick Chinese Translation Check

Key sections that must be synchronized:
1. Command list in README.md (line 13-40)
2. Quick Start section (line 86-99)
3. ID Prefix Matching (line 55-84)
4. Operator onboarding workflow

Use diff to spot missing sections:
```bash
# Compare structure (markdown headers)
diff <(grep '^#' README.md) <(grep '^#' README.zh-CN.md)
```
```

**预计工作量**: 30-40 分钟

---

### 建议 4: 命令示例添加注释

**优先级**: P3  
**位置**: README.md 多处命令示例

**当前问题**: 某些命令示例缺少说明

**示例改进**:

**当前** (README.md:64):
```bash
adp tasks show task-2026
```

**建议**:
```bash
# Show task using date prefix (matches all tasks from 2026)
adp tasks show task-2026

# More specific prefix for unique match
adp tasks show task-20260613-00
```

**预计工作量**: 40-50 分钟（需要审查所有示例）

---

## 📊 问题统计

| 类别 | 数量 | 优先级分布 |
|------|------|-----------|
| 错误 | 0 | - |
| 问题 | 2 | P1: 0, P2: 2, P3: 0 |
| 改进建议 | 4 | P1: 0, P2: 2, P3: 2 |
| **总计** | **6** | **待修复: 2** |

---

## ✅ 修复优先级

### 立即修复 (本周)
1. ⚠️ **问题 1**: 中文 README 添加 ID 前缀匹配文档 (~15分钟)
2. ⚠️ **问题 2**: 确认中文 operator-onboarding 完整性 (~10分钟)

### 短期改进 (本月)
3. 💡 **建议 1**: ID 前缀匹配最佳实践 (~20分钟)
4. 💡 **建议 2**: Quickstart 成功指标 (~25分钟)

### 长期改进 (季度)
5. 💡 **建议 3**: 文档同步检查清单 (~40分钟)
6. 💡 **建议 4**: 命令示例注释增强 (~50分钟)

**预计总工作量**: 2.5-3 小时

---

## 📝 验证清单

修复完成后，运行以下验证：

```bash
# 1. 运行文档验证脚本
./scripts/usability-docs-verification.sh

# 2. 检查中英文档命令数量一致
echo "English commands: $(grep -c '^\s*-\s*`adp ' README.md)"
echo "Chinese commands: $(grep -c '^\s*-\s*`adp ' README.zh-CN.md)"

# 3. 检查 ID 前缀匹配文档
grep -n "prefix" README.md README.zh-CN.md

# 4. 确认所有 markdown 链接有效
find docs -name "*.md" -exec grep -l "](.*\.md)" {} \;
```

---

**维护者**: 保持此清单更新，每次文档修改后重新运行验证脚本。
