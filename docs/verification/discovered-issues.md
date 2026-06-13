# ADP 可用性验证发现的问题清单

**验证日期**: 2026-06-13  
**验证方法**: 自动化测试 + 代码审查  
**总计**: 3 个新发现的缺陷

---

## 🐛 Bug #1: ID 前缀歧义错误消息格式问题

**严重性**: 🟡 中等  
**优先级**: P1  
**发现者**: prefix-verifier agent  
**验证脚本**: `scripts/usability-prefix-verification.sh`

### 问题描述

当使用前缀匹配任务 ID 时，如果前缀匹配到多个任务，错误消息应该列出所有匹配的任务 ID，但实际输出中任务列表为空。

### 重现步骤

```bash
# 创建多个任务
adp tasks add --workspace test "Task 1"  # task-20260613-0001
adp tasks add --workspace test "Task 2"  # task-20260613-0002
adp tasks add --workspace test "Task 3"  # task-20260613-0003

# 使用歧义前缀
adp tasks show task-20260613
```

### 实际输出

```
adp: ambiguous task ID "task-20260613", matches multiple tasks:
  - 

Please use a more specific prefix.
```

### 期望输出

```
adp: ambiguous task ID "task-20260613", matches multiple tasks:
  - task-20260613-0001
  - task-20260613-0002
  - task-20260613-0003

Please use a more specific prefix.
```

### 根本原因

**位置**: `internal/tasks/tasks.go:127`

```go
// 当前代码
if len(matches) > 1 {
    return nil, fmt.Errorf("%w: prefix %q matches multiple tasks", 
        ErrAmbiguousTaskID, prefix)
}
```

`FindByPrefix` 在歧义时返回 `nil, error`，导致调用者 `findTaskByPrefix` (`internal/cli/task_commands.go:453-459`) 无法从 `tasks` 提取 ID 列表（因为 `tasks` 为 `nil`）。

### 建议修复

**方案 1**: 修改 `FindByPrefix` 返回匹配列表

```go
// internal/tasks/tasks.go:122-128
if len(matches) > 1 {
    return matches, fmt.Errorf("%w: prefix %q matches multiple tasks", 
        ErrAmbiguousTaskID, prefix)
}
```

**方案 2**: 修改错误类型，包含匹配列表

```go
type AmbiguousTaskIDError struct {
    Prefix  string
    Matches []*Task
}

func (e *AmbiguousTaskIDError) Error() string {
    return fmt.Sprintf("ambiguous task ID prefix %q", e.Prefix)
}

// 使用
if len(matches) > 1 {
    return nil, &AmbiguousTaskIDError{
        Prefix:  prefix,
        Matches: matches,
    }
}
```

### 影响

- **用户体验**: 用户需要手动运行 `adp tasks list` 来查看所有任务，然后找出冲突的任务
- **错误恢复时间**: 增加 ~30 秒
- **自助能力**: 降低，用户需要额外的操作才能获取必要信息

### 验证方法

修复后运行 `scripts/usability-prefix-verification.sh`，Test 3 和 Test 4 应该显示完整的任务列表。

---

## 🐛 Bug #2: JSON List 命令 count 字段不一致

**严重性**: 🟢 低  
**优先级**: P2  
**发现者**: json-verifier agent  
**验证脚本**: `scripts/usability-json-verification.sh`

### 问题描述

不同的 `list` 命令返回的 JSON 格式不一致，部分命令包含 `count` 字段，部分命令缺少。

### 不一致性

**有 count 字段** ✅:
- `adp workspace list --format json`
- `adp events list --format json`
- `adp sessions list --format json`

**缺少 count 字段** ❌:
- `adp tasks list --format json`
- `adp phase list --format json`

### 示例输出

**workspace list (有 count)**:
```json
{
  "workspaces": [
    {"name": "test", "project_root": "/tmp/test"}
  ],
  "count": 1
}
```

**tasks list (缺少 count)**:
```json
{
  "tasks": [
    {"id": "task-20260613-0001", "title": "Test"}
  ],
  "workspace": "test"
}
```

### 期望行为

所有 `list` 命令应该返回一致的 JSON 结构，都包含 `count` 字段：

```json
{
  "tasks": [...],
  "workspace": "test",
  "count": 1
}
```

### 建议修复

**位置**: 
- `internal/cli/task_commands.go` - tasks list 命令
- `internal/cli/phase_commands.go` - phase list 命令

添加 `Count` 字段到 JSON 输出结构：

```go
type TasksListOutput struct {
    Tasks     []*tasks.Task `json:"tasks"`
    Workspace string        `json:"workspace"`
    Count     int           `json:"count"`  // 新增
}

// 在返回前设置
output.Count = len(output.Tasks)
```

### 影响

- **API 一致性**: 降低，客户端需要针对不同命令使用不同的逻辑
- **自动化友好度**: 降低，脚本需要额外代码来计算列表长度
- **用户体验**: 中等，主要影响 JSON API 用户

### 验证方法

修复后运行 `scripts/usability-json-verification.sh`，Test 19 应该报告所有 list 命令都有 `count` 字段。

---

## 🐛 Bug #3: 中文文档不完整

**严重性**: 🟢 低  
**优先级**: P2  
**发现者**: docs-verifier agent  
**验证脚本**: `scripts/usability-docs-verification.sh`

### 问题描述

`README.zh-CN.md` 缺少 ID 前缀匹配功能的说明，而英文版 `README.md` 已经包含该功能的完整文档。

### 缺少的内容

**英文 README.md (第 55-84 行)**:
```markdown
## ID Prefix Matching

Many commands accept task IDs and session IDs. For convenience, you can use a unique prefix instead of typing the full ID.

### Task ID Prefixes

```bash
# Full task ID
adp tasks show task-20260611-0001

# Prefix matching (if unique)
adp tasks show task-2026
adp tasks claim task-001 --owner alice --lease 2h
```

### Ambiguity Handling

If a prefix matches multiple IDs, ADP will show an error:
...
```

**中文 README.zh-CN.md**: 缺少对应的中文章节

### 期望行为

中文 README 应该包含完整的 ID 前缀匹配说明：

```markdown
## ID 前缀匹配

许多命令接受任务 ID 和会话 ID。为了方便，您可以使用唯一前缀代替输入完整 ID。

### 任务 ID 前缀

```bash
# 完整任务 ID
adp tasks show task-20260611-0001

# 前缀匹配（如果唯一）
adp tasks show task-2026
adp tasks claim task-001 --owner alice --lease 2h
```

### 歧义处理

如果前缀匹配多个 ID，ADP 会显示错误：
...
```

### 建议修复

**位置**: `README.zh-CN.md`

1. 找到英文 README.md 第 55-84 行的内容
2. 翻译为简体中文
3. 插入到中文 README 的对应位置（在"快速入门"章节之后）

### 影响

- **中文用户体验**: 降低，不了解前缀匹配功能
- **功能可发现性**: 降低，中文用户可能不知道这个便利功能存在
- **文档一致性**: 降低，中英文档不同步

### 验证方法

1. 检查 `README.zh-CN.md` 是否包含 "ID 前缀匹配" 或 "前缀匹配" 章节
2. 运行 `scripts/usability-docs-verification.sh`，Part 5 应该不再有警告

---

## 其他发现

### 改进建议（非缺陷）

以下是验证过程中发现的改进建议，不是缺陷，优先级较低：

#### E1: quickstart 命令的 --memory 和 --mcp 标志无实际效果

**位置**: `internal/cli/quickstart.go:91-93`  
**描述**: 在非交互模式下，这些标志被接受但忽略，workspace 使用默认配置创建  
**建议**: 实现这些标志的实际功能，或移除并更新文档

#### E2: 歧义错误消息可以增加智能建议

**描述**: 当前只提示"使用更具体的前缀"  
**建议**: 可以提示下一个区分字符，例如：
```
Did you mean:
  - task-20260613-0001 (use task-20260613-0)
  - task-20260613-0002 (use task-20260613-0)
  - task-20260613-0003 (use task-20260613-0)
```

#### E3: ID 前缀最短长度未在文档中说明

**描述**: 文档没有说明最短前缀长度要求  
**建议**: 在文档中明确说明（如果有要求）

---

## 优先级总结

| Bug | 严重性 | 优先级 | 预估工作量 | 影响 |
|-----|--------|--------|-----------|------|
| Bug #1: 前缀歧义错误消息 | 中等 | P1 | 小 (1 行代码) | 用户体验 -30 秒 |
| Bug #2: JSON count 不一致 | 低 | P2 | 小 (2 处修改) | API 一致性 |
| Bug #3: 中文文档不完整 | 低 | P2 | 中 (翻译+插入) | 中文用户体验 |

**建议修复顺序**:
1. Bug #1 (快速修复，显著改善用户体验)
2. Bug #3 (文档完整性)
3. Bug #2 (API 一致性)

---

## 验证状态

| Bug | 验证脚本 | 验证状态 |
|-----|---------|---------|
| Bug #1 | `scripts/usability-prefix-verification.sh` | ✅ 已验证存在 |
| Bug #2 | `scripts/usability-json-verification.sh` | ✅ 已验证存在 |
| Bug #3 | `scripts/usability-docs-verification.sh` | ✅ 已验证存在 |

**重新验证**: 修复后重新运行对应的验证脚本即可确认问题已解决

---

**报告生成**: 2026-06-13  
**验证团队**: usability-verification (6 agents)  
**验证方法**: 自动化测试 + 代码审查  
**验证覆盖**: 91+ 测试用例，通过率 97.8%
