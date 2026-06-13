# P2 任务细粒度拆解与并行执行方案

本文档将 P2 任务拆解为可并行执行的细粒度子任务。

---

## P2-1: 交互式快速入门命令 - 子任务拆解

### P2-1a: 交互库调研与选型（独立任务）

**目标**: 选择合适的交互式输入库

**工作内容**:
1. 评估候选库:
   - `github.com/AlecAivazis/survey/v2` - 功能丰富，社区活跃
   - `github.com/manifoldco/promptui` - 轻量级，API简洁
   - `github.com/charmbracelet/huh` - 现代化，TUI风格
2. 考虑因素:
   - 依赖大小和维护状态
   - API 易用性
   - 与现有代码风格的一致性
   - 是否支持非交互模式（CI环境）

**产出**: 推荐方案文档（200-300行文字）

**预计时间**: 30-60分钟

**可并行**: ✅ 完全独立

---

### P2-1b: quickstart 命令基础框架（依赖 P2-1a）

**目标**: 创建命令注册和基本结构

**工作内容**:
1. 创建 `internal/cli/quickstart.go`
2. 实现基础命令结构:
   ```go
   func (a *App) quickstart(ctx context.Context, args []string) error {
       // 解析参数（--non-interactive 等）
       // 调用交互流程
       return nil
   }
   ```
3. 在 `cmd/adp/main.go` 注册命令
4. 在 `internal/commandmeta/metadata.go` 添加元数据

**修改文件**:
- `internal/cli/quickstart.go` (新建)
- `cmd/adp/main.go` (1行)
- `internal/commandmeta/metadata.go` (~10行)

**预计时间**: 30分钟

**可并行**: 🔗 依赖 P2-1a（需要知道选用哪个库）

---

### P2-1c: 交互流程实现 - init 部分（依赖 P2-1b）

**目标**: 实现 ADP home 初始化交互

**工作内容**:
1. 在 `quickstart.go` 中添加:
   ```go
   // 询问 ADP home 路径
   // 调用 a.init()
   // 处理已存在的情况
   ```
2. 处理错误场景（home已存在、权限问题等）

**修改文件**:
- `internal/cli/quickstart.go` (~100行新增)

**预计时间**: 1小时

**可并行**: 🔗 依赖 P2-1b

---

### P2-1d: 交互流程实现 - workspace 部分（依赖 P2-1c）

**目标**: 实现 workspace 创建交互

**工作内容**:
1. 询问 workspace 名称
2. 询问 project root（带路径验证）
3. 询问配置选项（memory、MCP、agents）
4. 调用 `workspace add`
5. 可选：运行 `workspace doctor`

**修改文件**:
- `internal/cli/quickstart.go` (~150行新增)

**预计时间**: 1.5小时

**可并行**: 🔗 依赖 P2-1c（共享 quickstart.go）

---

### P2-1e: 非交互模式支持（依赖 P2-1d）

**目标**: 支持 CI/脚本环境

**工作内容**:
1. 添加 `--non-interactive` 标志
2. 添加 `--workspace-name`、`--project-root` 等参数
3. 非交互模式使用参数或合理默认值
4. 添加参数解析逻辑

**修改文件**:
- `internal/cli/quickstart.go` (~80行)
- `internal/commandmeta/metadata.go` (~5行)

**预计时间**: 45分钟

**可并行**: 🔗 依赖 P2-1d（修改同一文件）

---

### P2-1f: 测试与文档（可与 P2-1e 并行）

**目标**: 添加测试和使用文档

**工作内容**:
1. 添加单元测试（`quickstart_test.go`）
2. 添加集成测试脚本（`scripts/quickstart-smoke.sh`）
3. 更新 `internal/commandmeta/examples.go`
4. 更新 README.md 和 operator-onboarding.md

**修改文件**:
- `internal/cli/quickstart_test.go` (新建，~200行)
- `scripts/quickstart-smoke.sh` (新建，~50行)
- `internal/commandmeta/examples.go` (~5行)
- `README.md` (~20行)
- `operator-onboarding.md` (~30行)

**预计时间**: 1.5小时

**可并行**: ⚠️ 可与 P2-1e 并行（不同文件），但需要 P2-1d 完成

---

## P2-2: ID 格式可读性优化 - 子任务拆解

### P2-2a: task ID 前缀匹配核心逻辑（独立任务）

**目标**: 在 tasks 模块实现前缀匹配

**工作内容**:
1. 在 `internal/tasks/store.go` 添加 `FindByPrefix(prefix string) ([]Task, error)`
2. 实现匹配逻辑:
   - 精确匹配优先
   - 前缀匹配（返回所有匹配）
   - 无匹配返回错误
3. 处理歧义（多个匹配）

**修改文件**:
- `internal/tasks/store.go` (~50行新增)
- `internal/tasks/store_test.go` (~100行测试)

**预计时间**: 1小时

**可并行**: ✅ 完全独立

---

### P2-2b: session ID 前缀匹配核心逻辑（独立任务）

**目标**: 在 sessions 模块实现前缀匹配

**工作内容**:
1. 在 `internal/sessions/store.go` 添加 `FindByPrefix(prefix string) ([]SessionSummary, error)`
2. 实现与 P2-2a 相同的匹配逻辑
3. 添加测试

**修改文件**:
- `internal/sessions/store.go` (~50行新增)
- `internal/sessions/store_test.go` (~100行测试)

**预计时间**: 1小时

**可并行**: ✅ 完全独立（与 P2-2a 并行）

---

### P2-2c: tasks 命令支持前缀（依赖 P2-2a）

**目标**: 更新所有接受 task ID 的命令

**工作内容**:
1. 修改命令:
   - `tasks show <id>`
   - `tasks update <id>`
   - `tasks claim <id>`
   - `tasks renew <id>`
   - `tasks release <id>`
   - `tasks done <id>`
   - `tasks block <id>`
2. 每个命令调用 `FindByPrefix` 而非直接查询
3. 处理歧义错误（友好提示）

**修改文件**:
- `internal/cli/task_commands.go` (~70行修改)
- `internal/cli/task_commands_test.go` (~50行测试)

**预计时间**: 1小时

**可并行**: 🔗 依赖 P2-2a

---

### P2-2d: sessions 命令支持前缀（依赖 P2-2b）

**目标**: 更新所有接受 session ID 的命令

**工作内容**:
1. 修改命令:
   - `sessions show <id>`
   - `sessions restore-plan <id>`
   - `sessions resume-plan <id>`
2. 调用 `FindByPrefix`
3. 处理歧义错误

**修改文件**:
- `internal/cli/ops_commands.go` (~50行修改)
- `internal/cli/ops_commands_test.go` (~40行测试)

**预计时间**: 45分钟

**可并行**: ✅ 可与 P2-2c 并行（不同文件）

---

### P2-2e: events 命令支持前缀（可选，依赖 P2-2a 或 P2-2b）

**目标**: `events list --task <id>` 支持前缀

**工作内容**:
1. 在 `events list` 中，`--task` 参数支持前缀
2. 在 `events list` 中，`--session` 参数支持前缀

**修改文件**:
- `internal/cli/ops_commands.go` (~30行修改)

**预计时间**: 30分钟

**可并行**: ✅ 可与 P2-2c/2d 并行（不同函数）

---

### P2-2f: run 命令支持前缀（依赖 P2-2a）

**目标**: `adp run --task <id>` 支持前缀

**工作内容**:
1. 在 `internal/cli/run_commands.go` 的 task ID 解析处调用 `FindByPrefix`

**修改文件**:
- `internal/cli/run_commands.go` (~20行修改)

**预计时间**: 20分钟

**可并行**: ✅ 可与 P2-2c/2d/2e 并行（不同文件）

---

### P2-2g: 补全逻辑更新（依赖 P2-2a 和 P2-2b）

**目标**: shell 补全支持前缀

**工作内容**:
1. 更新 `internal/cli/completion.go` 中的 task ID 补全
2. 更新 session ID 补全
3. 测试 bash/zsh 补全

**修改文件**:
- `internal/cli/completion.go` (~40行修改)

**预计时间**: 45分钟

**可并行**: 🔗 需要等 P2-2a 和 P2-2b 完成（需要 FindByPrefix API）

---

### P2-2h: 文档与示例更新（可与 P2-2g 并行）

**目标**: 更新文档说明前缀匹配功能

**工作内容**:
1. 更新 `internal/commandmeta/examples.go`（添加短前缀示例）
2. 更新 README.md（说明 ID 前缀功能）
3. 更新 operator-onboarding.md

**修改文件**:
- `internal/commandmeta/examples.go` (~10行)
- `README.md` (~15行)
- `operator-onboarding.md` (~20行)

**预计时间**: 30分钟

**可并行**: ✅ 可与 P2-2g 并行

---

## 并行执行方案

### Wave P2-1: 调研与框架（2 agents）

```
Agent-A: P2-1a (交互库调研) ✅ 独立
Agent-B: P2-2a (task ID 前缀匹配) ✅ 独立
```

**预计时间**: 1小时  
**输出**: 库选型决策 + task 前缀匹配基础

---

### Wave P2-2: 核心逻辑（3 agents）

等待 Wave P2-1 完成后：

```
Agent-A: P2-1b → P2-1c (quickstart 框架 + init 流程) 🔗 串行
Agent-B: P2-2b (session ID 前缀匹配) ✅ 独立
Agent-C: P2-2c (tasks 命令前缀支持) 🔗 依赖 P2-2a
```

**预计时间**: 2小时  
**输出**: quickstart 基础 + 所有前缀匹配核心逻辑 + tasks 命令更新

---

### Wave P2-3: 功能完善（4 agents）

等待 Wave P2-2 完成后：

```
Agent-A: P2-1d (workspace 交互流程) 🔗 依赖 P2-1c
Agent-B: P2-2d (sessions 命令前缀支持) ✅ 独立
Agent-C: P2-2e (events 命令前缀支持) ✅ 独立
Agent-D: P2-2f (run 命令前缀支持) ✅ 独立
```

**预计时间**: 1.5小时  
**输出**: quickstart 完整流程 + 所有命令前缀支持

---

### Wave P2-4: 收尾与测试（3 agents）

等待 Wave P2-3 完成后：

```
Agent-A: P2-1e (非交互模式) 🔗 依赖 P2-1d
Agent-B: P2-1f (quickstart 测试文档) ⚠️ 需要 P2-1d
Agent-C: P2-2g (补全逻辑) 🔗 依赖 P2-2a/b
```

**预计时间**: 1.5小时  
**输出**: 非交互模式 + 测试 + 补全

---

### Wave P2-5: 文档整合（1 agent）

等待 Wave P2-4 完成后：

```
Agent-A: P2-2h (文档更新) ✅ 独立
```

**预计时间**: 30分钟  
**输出**: 完整文档

---

## 总体时间估算

- **Wave P2-1**: 1小时
- **Wave P2-2**: 2小时
- **Wave P2-3**: 1.5小时
- **Wave P2-4**: 1.5小时
- **Wave P2-5**: 0.5小时

**总计**: ~6.5小时（串行执行需 12-16小时）

---

## Agent 分配建议

### 方案 1: 4 agents 并行

- **Agent-1 (quickstart 专家)**: P2-1a → P2-1b → P2-1c → P2-1d → P2-1e → P2-1f
- **Agent-2 (ID 匹配专家)**: P2-2a → P2-2c → P2-2f
- **Agent-3 (ID 匹配专家)**: P2-2b → P2-2d → P2-2e
- **Agent-4 (集成专家)**: P2-2g → P2-2h

### 方案 2: 3 agents 并行（推荐）

- **Agent-1 (quickstart 全栈)**: P2-1a → P2-1b → P2-1c → P2-1d → P2-1e → P2-1f
- **Agent-2 (ID 匹配核心)**: P2-2a → P2-2b → P2-2c → P2-2d
- **Agent-3 (ID 匹配集成)**: 等待 P2-2a/b → P2-2e → P2-2f → P2-2g → P2-2h

---

## 任务协调规范

### 文件冲突预防

- **P2-1 系列**: 主要修改 `internal/cli/quickstart.go`（新文件）
- **P2-2 系列**: 分散在多个文件，低冲突风险

### 通信点

1. **P2-1a 完成后**: 通知 P2-1b 采用的交互库
2. **P2-2a 完成后**: 通知其他 agent `FindByPrefix` API 签名
3. **P2-2b 完成后**: 通知 P2-2d/2e/2g
4. **每个 wave 完成后**: 运行 `scripts/check-all.sh`

### 验收标准

- [ ] P2-1: `adp quickstart` 可完整运行并创建 workspace
- [ ] P2-1: 支持 `--non-interactive` 模式
- [ ] P2-1: smoke 测试通过
- [ ] P2-2: task ID 前缀匹配工作（精确、前缀、歧义）
- [ ] P2-2: session ID 前缀匹配工作
- [ ] P2-2: 所有命令支持前缀
- [ ] P2-2: 补全支持前缀
- [ ] 所有测试通过 `scripts/check-all.sh`

---

## 风险评估

### P2-1 风险

- **依赖引入**: 新交互库可能增加构建复杂度 → **缓解**: 选择轻量库
- **非交互模式**: 需要良好的参数设计 → **缓解**: 参考其他 CLI 工具

### P2-2 风险

- **歧义处理**: 前缀匹配可能混淆用户 → **缓解**: 清晰的错误提示
- **向后兼容**: 确保完整 ID 仍然工作 → **缓解**: 精确匹配优先

---

## 下一步行动

选择执行方案：

1. **快速迭代**: 启动方案 2（3 agents）
2. **稳健推进**: 先完成 P2-2（2 agents），验证通过后做 P2-1
3. **用户优先**: 先完成 P2-1（1 agent），快速改善新用户体验

推荐：**方案 2（3 agents 并行）** - 平衡速度和风险
