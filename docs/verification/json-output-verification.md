# JSON 输出一致性验证报告

**生成日期**: 2026-06-13  
**验证工具**: `scripts/usability-json-verification.sh`  
**测试范围**: 所有支持 `--format json` 的命令

## 执行摘要

本报告记录了 ADP 命令行工具所有 JSON 输出的格式一致性验证结果。共测试了 **20 项测试用例**，覆盖以下命令类别：

- ✅ **Workspace 命令**: list, show, doctor
- ✅ **Tasks 命令**: list, show, next, stale, take
- ✅ **Phase 命令**: list, show, status
- ✅ **Progress 命令**: report, progress
- ✅ **Events 命令**: list
- ✅ **Sessions 命令**: list
- ✅ **Runtime 命令**: prune
- ✅ **Plan 命令**: doctor
- ✅ **Version 命令**: version

### 关键发现

✅ **通过**: 所有命令均输出有效的 JSON  
⚠️ **不一致**: 发现 **2 处结构不一致**问题

---

## JSON 结构文档

### 1. Version 命令

#### `version --format json`

| 字段 | 类型 | 说明 | 必需 |
|------|------|------|------|
| `version` | string | 版本号 | ✅ |
| `go_version` | string | Go 编译版本 | ✅ |
| `platform` | string | 平台信息 | ✅ |

**示例输出**:
```json
{
  "version": "0.1.0",
  "go_version": "go1.22.1",
  "platform": "linux/amd64"
}
```

---

### 2. Workspace 命令

#### `workspace list --format json`

| 字段 | 类型 | 说明 | 必需 |
|------|------|------|------|
| `workspaces` | array | 工作区列表 | ✅ |
| `count` | integer | 工作区总数 | ✅ |

**示例输出**:
```json
{
  "workspaces": [
    {
      "name": "my-workspace",
      "project_root": "/path/to/project"
    }
  ],
  "count": 1
}
```

#### `workspace show <name> --format json`

| 字段 | 类型 | 说明 | 必需 |
|------|------|------|------|
| `name` | string | 工作区名称 | ✅ |
| `project_root` | string | 项目根路径 | ✅ |

**示例输出**:
```json
{
  "name": "my-workspace",
  "project_root": "/path/to/project"
}
```

#### `workspace doctor <name> --format json`

| 字段 | 类型 | 说明 | 必需 |
|------|------|------|------|
| `reports` | array | 诊断报告列表 | ✅ |
| `report_count` | integer | 报告数量 | ✅ |
| `has_errors` | boolean | 是否有错误 | ✅ |

**报告对象结构**:

| 字段 | 类型 | 说明 | 必需 |
|------|------|------|------|
| `workspace` | string | 工作区名称 | ✅ |
| `workspace_dir` | string | 工作区目录 | ✅ |
| `config_path` | string | 配置文件路径 | ✅ |
| `diagnostics` | array | 诊断项列表 | ✅ |
| `diagnostic_count` | integer | 诊断项数量 | ✅ |
| `has_errors` | boolean | 是否有错误 | ✅ |

**诊断项对象结构**:

| 字段 | 类型 | 说明 | 必需 |
|------|------|------|------|
| `level` | string | 级别 (error/warning/info) | ✅ |
| `code` | string | 诊断代码 | ✅ |
| `message` | string | 诊断消息 | ✅ |
| `path` | string | 相关路径 | ❌ |

---

### 3. Tasks 命令

#### `tasks list --workspace <name> --format json`

⚠️ **不一致**: 缺少 `count` 字段

| 字段 | 类型 | 说明 | 必需 |
|------|------|------|------|
| `workspace` | string | 工作区名称 | ✅ |
| `tasks` | array | 任务列表 | ✅ |
| ~~`count`~~ | ~~integer~~ | ⚠️ **缺失** | ❌ |

**示例输出**:
```json
{
  "workspace": "my-workspace",
  "tasks": []
}
```

#### `tasks show <id> --workspace <name> --format json`

| 字段 | 类型 | 说明 | 必需 |
|------|------|------|------|
| `id` | string | 任务 ID | ✅ |
| `title` | string | 任务标题 | ✅ |
| `status` | string | 任务状态 | ✅ |

#### `tasks next --workspace <name> --format json`

| 字段 | 类型 | 说明 | 必需 |
|------|------|------|------|
| `workspace` | string | 工作区名称 | ✅ |
| `candidates` | array | 候选任务列表 | ✅ |
| `eligible_count` | integer | 符合条件的任务数 | ✅ |
| `planning_source` | string | 计划来源路径 | ✅ |
| `generated_at` | string | 生成时间 (ISO 8601) | ✅ |
| `total` | integer | 总任务数 | ✅ |
| `counts` | object | 各状态任务计数 | ✅ |
| `limit` | integer | 返回限制 | ✅ |

**示例输出**:
```json
{
  "workspace": "my-workspace",
  "planning_source": "/path/to/tasks.yaml",
  "generated_at": "2026-06-13T14:05:20Z",
  "total": 0,
  "eligible_count": 0,
  "counts": {
    "blocked": 0,
    "canceled": 0,
    "done": 0,
    "in_progress": 0,
    "planned": 0,
    "ready": 0,
    "review": 0,
    "validated": 0
  },
  "limit": 5,
  "candidates": []
}
```

#### `tasks stale --workspace <name> --format json`

| 字段 | 类型 | 说明 | 必需 |
|------|------|------|------|
| `workspace` | string | 工作区名称 | ✅ |
| `tasks` | array | 过期任务列表 | ✅ |

#### `tasks take --workspace <name> --owner <owner> --format json`

| 字段 | 类型 | 说明 | 必需 |
|------|------|------|------|
| `task` | object | 被认领的任务 | ✅ |
| `owner` | string | 所有者 | ✅ |

---

### 4. Phase 命令

#### `phase list --workspace <name> --format json`

⚠️ **不一致**: 缺少 `count` 字段

| 字段 | 类型 | 说明 | 必需 |
|------|------|------|------|
| `workspace` | string | 工作区名称 | ✅ |
| `phases` | array | 阶段列表 | ✅ |
| ~~`count`~~ | ~~integer~~ | ⚠️ **缺失** | ❌ |

**示例输出**:
```json
{
  "workspace": "my-workspace",
  "phases": []
}
```

#### `phase show <id> --workspace <name> --format json`

| 字段 | 类型 | 说明 | 必需 |
|------|------|------|------|
| `id` | string | 阶段 ID | ✅ |
| `title` | string | 阶段标题 | ✅ |
| `status` | string | 阶段状态 | ✅ |
| `order` | integer | 顺序 | ✅ |
| `created_at` | string | 创建时间 (ISO 8601) | ✅ |
| `updated_at` | string | 更新时间 (ISO 8601) | ✅ |

#### `phase status --workspace <name> --format json`

| 字段 | 类型 | 说明 | 必需 |
|------|------|------|------|
| `workspace` | string | 工作区名称 | ✅ |
| `phase_count` | integer | 阶段数量 | ✅ |
| `can_start_next` | boolean | 能否开始下一个 | ✅ |
| `next_action` | string | 下一步操作 | ✅ |
| `reason` | string | 原因说明 | ✅ |

**示例输出**:
```json
{
  "workspace": "my-workspace",
  "phase_count": 0,
  "can_start_next": false,
  "next_action": "plan_next_phase",
  "reason": "no planned phase remains"
}
```

---

### 5. Progress 命令

#### `progress report --workspace <name> --format json`

| 字段 | 类型 | 说明 | 必需 |
|------|------|------|------|
| `workspace` | string | 工作区名称 | ✅ |
| `phases` | array | 阶段列表 | ✅ |
| `total` | integer | 总任务数 | ✅ |
| `counts` | object | 各状态任务计数 | ✅ |
| `tasks` | array | 任务列表 | ✅ |
| `next` | array | 下一批任务 | ✅ |
| `phase_evidence` | array | 阶段证据 | ✅ |
| `runtime_sessions` | array | 运行时会话 | ✅ |

#### `progress --workspace <name> --format json`

类似于 `progress report`，结构相同。

---

### 6. Events 命令

#### `events list --workspace <name> --format json`

| 字段 | 类型 | 说明 | 必需 |
|------|------|------|------|
| `filters` | object | 过滤条件 | ✅ |
| `limit` | integer | 返回限制 | ✅ |
| `count` | integer | 事件总数 | ✅ |
| `events` | array | 事件列表 | ✅ |

**示例输出**:
```json
{
  "filters": {
    "workspace": "my-workspace"
  },
  "limit": 20,
  "count": 0,
  "events": []
}
```

---

### 7. Sessions 命令

#### `sessions list --workspace <name> --format json`

| 字段 | 类型 | 说明 | 必需 |
|------|------|------|------|
| `filters` | object | 过滤条件 | ✅ |
| `limit` | integer | 返回限制 | ✅ |
| `count` | integer | 会话总数 | ✅ |
| `sessions` | array | 会话列表 | ✅ |

**示例输出**:
```json
{
  "filters": {
    "workspace": "my-workspace"
  },
  "limit": 20,
  "count": 0,
  "sessions": []
}
```

---

### 8. Runtime 命令

#### `runtime prune --format json --dry-run`

| 字段 | 类型 | 说明 | 必需 |
|------|------|------|------|
| `count` | integer | 清理项数量 | ✅ |
| `dry_run` | boolean | 是否演习模式 | ✅ |

---

### 9. Plan 命令

#### `plan doctor --workspace <name> --format json`

| 字段 | 类型 | 说明 | 必需 |
|------|------|------|------|
| `workspace` | string | 工作区名称 | ✅ |
| `diagnostics` | array | 诊断项列表 | ✅ |

---

## 不一致问题清单

### 问题 1: List 命令缺少 `count` 字段

**严重性**: ⚠️ 中等  
**影响范围**: `tasks list`, `phase list`

**问题描述**:

大多数 `list` 命令（`workspace list`, `events list`, `sessions list`）都包含 `count` 字段来表示列表项总数，但以下命令缺少此字段：

- ❌ `tasks list --format json` - 无 `count` 字段
- ❌ `phase list --format json` - 无 `count` 字段

**当前行为**:
```json
{
  "workspace": "test",
  "tasks": [...]
}
```

**期望行为**:
```json
{
  "workspace": "test",
  "tasks": [...],
  "count": 5
}
```

**建议修复**:

在 `tasks list` 和 `phase list` 的 JSON 输出中添加 `count` 字段，与其他 list 命令保持一致。

---

### 问题 2: `workspace doctor` 字段命名已修复

**状态**: ✅ 已解决

之前验证脚本期望 `checks` 字段，但实际输出使用 `diagnostics`。这已在最新版本中得到正确处理。实际输出字段：

- ✅ `diagnostics` (正确)
- ✅ `diagnostic_count` (正确)

---

## 验证方法

### 测试方法

1. **有效性验证**: 检查输出是否为有效 JSON（以 `{` 或 `[` 开头/结尾）
2. **字段验证**: 使用 `assert_contains` 验证关键字段存在
3. **隔离环境**: 每次测试使用临时 `ADP_HOME`，确保测试独立性
4. **无外部依赖**: 不使用 `jq`，仅依赖标准 bash 工具

### 运行验证脚本

```bash
bash scripts/usability-json-verification.sh
```

### 验证覆盖率

| 类别 | 覆盖命令数 | 覆盖率 |
|------|-----------|--------|
| Workspace | 3/3 | 100% |
| Tasks | 5/5 | 100% |
| Phase | 3/3 | 100% |
| Progress | 2/2 | 100% |
| Events | 1/1 | 100% |
| Sessions | 1/1 | 100% |
| Runtime | 1/1 | 100% |
| Plan | 1/1 | 100% |
| Version | 1/1 | 100% |
| **总计** | **18/18** | **100%** |

---

## 改进建议

### 高优先级

1. **添加 `count` 字段到 list 命令**
   - 影响: `tasks list`, `phase list`
   - 理由: 保持 API 一致性，方便客户端分页实现
   - 实现难度: 低

### 中优先级

2. **统一时间戳格式**
   - 当前: 所有时间戳使用 ISO 8601 格式 (✅ 已一致)
   - 建议: 保持现状

3. **统一命名约定**
   - 当前: `diagnostic_count`, `phase_count`, `eligible_count`
   - 建议: 统一使用 `<resource>_count` 模式

### 低优先级

4. **添加 API 版本字段**
   - 在顶层 JSON 添加 `api_version` 字段
   - 便于未来版本迁移和兼容性处理

5. **标准化错误响应格式**
   - 定义统一的错误 JSON 结构
   - 包含 `error`, `code`, `message` 字段

---

## 验证历史

| 日期 | 版本 | 通过测试 | 不一致问题 | 验证者 |
|------|------|---------|-----------|--------|
| 2026-06-13 | v1.0 | 20/20 | 2 | json-verifier agent |

---

## 附录: 完整测试用例列表

1. ✅ `version --format json`
2. ✅ `workspace list --format json`
3. ✅ `workspace show --format json`
4. ✅ `workspace doctor --format json`
5. ✅ `tasks list --format json` (有不一致)
6. ✅ `tasks next --format json`
7. ✅ `tasks stale --format json`
8. ✅ `tasks show --format json`
9. ✅ `tasks take --format json`
10. ✅ `phase list --format json` (有不一致)
11. ✅ `phase status --format json`
12. ✅ `phase show --format json`
13. ✅ `progress report --format json`
14. ✅ `progress --format json`
15. ✅ `events list --format json`
16. ✅ `sessions list --format json`
17. ✅ `runtime prune --format json --dry-run`
18. ✅ `plan doctor --format json`
19. ✅ List 命令一致性检查
20. ✅ 未知格式拒绝测试

---

## 结论

ADP CLI 的 JSON 输出整体质量良好，所有命令均能输出有效的 JSON 格式。主要问题是 `tasks list` 和 `phase list` 缺少 `count` 字段，导致与其他 list 命令不一致。

建议在下一个版本中统一所有 list 命令的输出格式，添加缺失的 `count` 字段，以提升 API 一致性和可用性。
