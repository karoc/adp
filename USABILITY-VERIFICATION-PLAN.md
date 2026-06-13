# 易用性验证执行方案

**目标**: 验证 P0-P2 改进的实际效果，发现新的可用性问题  
**预估时间**: 2-3 小时（并行执行）  
**并行策略**: 3 波次，最多 4 个 agents 并行

---

## 验证范围

### 已完成的改进（需要验证）

**P0 改进** (核心体验修复):
- ✅ workspace list --format json 支持
- ✅ agent 不存在时的友好错误提示
- ✅ 增强的版本信息显示

**P1 改进** (用户体验增强):
- ✅ 空列表友好提示（tasks, phases, events, sessions）
- ✅ workspace show --format json 支持
- ✅ adp --help 项目描述和链接
- ✅ init 成功后的引导信息
- ✅ workspace add 名称冲突检查

**P2 改进** (高级功能):
- ✅ quickstart 交互式命令
- ✅ task/session ID 前缀匹配

---

## 验证方法论

采用 **模拟用户场景** + **自动化测试** 双轨验证：

1. **场景模拟**: 构建真实用户的使用路径，记录体验
2. **自动化验证**: 编写测试脚本验证功能正确性
3. **文档审查**: 检查文档是否准确反映新功能
4. **边界测试**: 验证错误处理和边界情况

---

## 任务拆分（支持并行）

### Wave 1: 核心流程验证（3 个独立任务，可完全并行）

#### Task V1-1: 新用户首次体验验证

**验证目标**: quickstart 命令的完整流程

**测试场景**:
1. **场景 1.1**: 非交互式设置自动化测试
   - 运行 `adp quickstart --non-interactive`
   - 验证所有参数正确处理
   - 检查成功流程的输出内容
   - 验证 workspace 创建成功

2. **场景 1.2**: 参数验证边界测试
   - 缺少必需参数（workspace-name, project-root）
   - 非法 workspace 名称
   - 不存在的项目路径
   - 验证错误消息友好性和可操作性

3. **场景 1.3**: 交互式体验手动验证
   - 手动运行 `adp quickstart` 交互模式
   - 记录用户输入流程体验
   - 评估提示清晰度和引导信息
   - 测试错误恢复能力

**验证脚本**: `scripts/usability-quickstart-verification.sh` (扩展现有 quickstart-smoke.sh)

**产出**:
- 验证报告: `docs/verification/quickstart-verification.md`
- 自动化测试脚本（基于现有 quickstart-smoke.sh）
- 手动测试记录
- 发现的问题清单
- 改进建议

**独立性**: ✅ 完全独立

**注**: 自动化测试使用 non-interactive 模式（现有模式），交互式体验通过手动测试验证

---

#### Task V1-2: ID 前缀匹配体验验证

**验证目标**: task/session ID 前缀功能的易用性

**测试场景**:
1. **场景 2.1**: 前缀匹配成功路径
   - 创建多个 tasks
   - 使用短前缀（如 `task-001`）
   - 使用日期前缀（如 `task-2026`）
   - 验证命令执行成功

2. **场景 2.2**: 歧义处理体验
   - 创建多个匹配同一前缀的 tasks
   - 使用歧义前缀
   - 验证错误消息的可读性和可操作性

3. **场景 2.3**: 跨命令一致性
   - 测试所有支持前缀的命令
   - 验证行为一致性（tasks, sessions, events, run）

**验证脚本**: `scripts/usability-prefix-verification.sh`

**产出**:
- 验证报告: `docs/verification/prefix-matching-verification.md`
- 命令覆盖清单
- 用户体验评分

**独立性**: ✅ 完全独立

---

#### Task V1-3: 错误处理改进验证

**验证目标**: P0-P2 中所有错误提示的友好性

**测试场景**:
1. **场景 3.1**: Agent 不存在错误
   - 配置不存在的 agent
   - 运行 `adp run nonexistent-agent`
   - 验证错误消息的清晰度和可操作性

2. **场景 3.2**: Workspace 冲突错误
   - 尝试添加已存在的 workspace
   - 验证冲突提示的友好性
   - 检查建议操作是否明确

3. **场景 3.3**: 空列表提示
   - 测试所有空列表场景（tasks, phases, events, sessions）
   - 验证提示消息的可操作性
   - 检查建议命令是否正确

**验证脚本**: `scripts/usability-errors-verification.sh`

**产出**:
- 验证报告: `docs/verification/error-handling-verification.md`
- 错误消息质量评分
- 改进建议

**独立性**: ✅ 完全独立

---

### Wave 2: 输出格式和文档验证（3 个独立任务，可完全并行）

#### Task V2-1: JSON 输出一致性验证

**验证目标**: 所有命令的 JSON 输出格式一致性

**测试场景**:
1. **场景 1.1**: JSON 输出可用性
   - 列出所有支持 `--format json` 的命令
   - 逐一执行验证 JSON 输出可用
   - 验证 JSON 语法正确性（可解析）

2. **场景 1.2**: JSON 结构一致性
   - 使用 `assert_contains` 验证关键字段存在性
   - 检查字段命名规范一致性
   - 验证数据类型正确性（通过字段值检查）

3. **场景 1.3**: JSON 结构文档化
   - 记录每个命令的 JSON 输出结构
   - 列出必需字段和可选字段
   - 使用 Markdown 表格格式文档化

**验证脚本**: `scripts/usability-json-verification.sh`

**产出**:
- 验证报告: `docs/verification/json-consistency-verification.md`
- JSON 结构文档（Markdown 表格，非正式 schema 文件）
- 不一致问题清单
- 改进建议

**独立性**: ✅ 完全独立

**注**: 使用项目现有的 `assert_contains` 模式验证，不依赖 jq 或生成正式 JSON schema

---

#### Task V2-2: 文档准确性验证

**验证目标**: 文档与实际功能的一致性

**测试场景**:
1. **场景 2.1**: README.md 验证
   - 使用 grep/sed 提取命令示例
   - 逐一执行高频命令示例验证
   - 检查输出是否与文档描述一致

2. **场景 2.2**: operator-onboarding.md 验证
   - 按文档步骤完整走通核心流程
   - 记录任何偏差或过时内容
   - 验证新功能（quickstart, ID 前缀匹配）是否被正确记录

3. **场景 2.3**: 中英文档一致性
   - 对比英文和中文版本关键章节
   - 检查命令示例一致性
   - 验证技术术语翻译准确性

**验证脚本**: `scripts/usability-docs-verification.sh`

**产出**:
- 验证报告: `docs/verification/documentation-verification.md`
- 文档错误清单（过时、不准确、缺失）
- 修复建议（具体行号和修改内容）

**独立性**: ✅ 完全独立

**注**: 示例提取使用简单的 grep/sed 脚本，重点验证高频命令而非全覆盖

---

#### Task V2-3: Help 信息完整性验证

**验证目标**: 所有 help 信息的准确性和完整性

**测试场景**:
1. **场景 3.1**: Help 文本覆盖
   - 测试所有命令的 `--help`
   - 验证示例命令可执行
   - 检查选项描述准确性

2. **场景 3.2**: 错误提示的 help 建议
   - 触发各种错误
   - 验证是否提供 help 建议
   - 检查建议的准确性

3. **场景 3.3**: 新功能的 help 覆盖
   - quickstart --help
   - 前缀匹配的文档
   - 检查描述清晰度

**验证脚本**: `scripts/usability-help-verification.sh`

**产出**:
- 验证报告: `docs/verification/help-completeness-verification.md`
- Help 覆盖矩阵
- 改进建议

**独立性**: ✅ 完全独立

---

### Wave 3: 综合分析和报告（2 个任务，部分依赖）

#### Task V3-1: 易用性评分和对比

**验证目标**: 量化改进效果

**依赖**: Wave 1 和 Wave 2 的所有验证报告

**分析内容**:
1. **对比分析**
   - 改进前（usability-test-report.md）vs 改进后
   - 量化指标对比（完成时间、错误率、满意度）

2. **评分卡**
   - 新用户首次体验得分
   - 常用操作流畅度得分
   - 错误恢复能力得分
   - 文档清晰度得分

3. **用户画像验证**
   - 初学者体验
   - 中级用户体验
   - 高级用户体验

**产出**:
- 综合报告: `docs/verification/usability-improvement-report.md`
- 评分对比表
- 可视化图表（如果可能）

**独立性**: ⚠️ 需要 Wave 1-2 完成

---

#### Task V3-2: 遗留问题和后续规划

**验证目标**: 识别新的改进机会

**依赖**: Wave 1, 2, 3-1 完成

**分析内容**:
1. **新发现的问题**
   - 从验证中发现的问题
   - 优先级评估
   - 工作量估算

2. **用户反馈总结**
   - 积极反馈
   - 负面反馈
   - 中性观察

3. **下一阶段规划**
   - P3 优先级改进建议
   - 功能扩展方向
   - 性能优化机会

**产出**:
- 问题清单: `docs/verification/issues-discovered.md`
- 下一阶段计划: `PLAN-P3.md`
- Release notes 草稿

**独立性**: ⚠️ 需要 V3-1 完成

---

## 执行策略

### 并行执行时间表

```
Wave 1 (并行): V1-1, V1-2, V1-3 (预估 60 分钟)
    |
    v
Wave 2 (并行): V2-1, V2-2, V2-3 (预估 60 分钟)
    |
    v
Wave 3 (串行): V3-1 -> V3-2 (预估 60 分钟)

总时间: ~180 分钟（3 小时）
实际可能: 3-4 小时（考虑问题发现和调整）
```

### Agent 分配

- **Wave 1**: 3 agents (quickstart-verifier, prefix-verifier, errors-verifier)
- **Wave 2**: 3 agents (json-verifier, docs-verifier, help-verifier)
- **Wave 3**: 2 agents (scorer, planner)

### 协调机制

1. **统一测试环境**
   - 所有 agents 使用独立的临时 ADP_HOME
   - 避免测试数据污染
   - 清理脚本统一管理（使用 `trap cleanup EXIT` 模式）

2. **报告格式规范**
   - 使用统一的 Markdown 模板
   - 评分标准一致（1-5 分）
   - 问题分类统一（critical/major/minor/enhancement）

3. **依赖管理**
   - Wave 1-2 完全独立
   - Wave 3 需要前两波的输出文件
   - 使用文件系统作为协调机制

4. **产出目录结构**
   - 第一个执行的 agent 创建 `docs/verification/` 目录
   - 验证脚本: `scripts/usability-*-verification.sh`
   - 验证报告: `docs/verification/*-verification.md`

5. **测试方法统一**
   - 使用现有 smoke test 模式（`assert_contains`, `run_adp`, `fail`, `info`）
   - 不依赖外部工具（expect, jq）
   - 临时环境隔离：`mktemp -d`, `trap cleanup EXIT`

---

## 验收标准

### 每个验证任务的产出

✅ **验证脚本**: 可重复执行的自动化脚本  
✅ **验证报告**: Markdown 格式，包含场景、结果、评分  
✅ **问题清单**: 发现的问题及严重程度  
✅ **改进建议**: 具体可执行的改进方案

### 整体验收

✅ **覆盖率**: 所有 P0-P2 改进都有验证  
✅ **可重复性**: 所有测试脚本可自动运行  
✅ **量化指标**: 至少 3 个量化对比指标  
✅ **可操作性**: 发现的问题有明确的解决方案

---

## 风险与缓解

### 风险 1: 测试环境冲突

**影响**: 并行 agents 可能共享测试数据

**缓解**:
- 每个 agent 使用独立的临时目录
- ADP_HOME 环境变量隔离
- 测试后自动清理

### 风险 2: 主观评分不一致

**影响**: 不同 agents 的评分标准可能不同

**缓解**:
- 提供明确的评分标准（1-5 分量表）
- 使用客观指标辅助（时间、错误次数）
- 最终由 V3-1 统一校准

### 风险 3: 验证发现重大问题

**影响**: 可能需要立即修复，影响发布计划

**缓解**:
- 按严重程度分类（critical 立即修复，minor 下一版本）
- 准备快速修复流程
- 更新发布计划

### 风险 4: P0-P2 改进未完全实现

**影响**: 验证发现某些声称已完成的改进实际未生效

**缓解**:
- 优先验证核心改进（workspace list --format json, ID 前缀匹配）
- 发现问题立即标记为 critical
- 在 V3-2 中明确记录实现缺口

### 风险 5: 测试工具缺失

**影响**: 方案假设的工具不可用（如 expect, jq, JSON schema 生成器）

**缓解**:
- 已在方案中使用项目现有模式（assert_contains, non-interactive）
- 降级方案：自动化改为手动，JSON schema 改为文档化
- Agent 自适应调整测试方法

---

## 下一步行动

如果批准此方案，我将：

1. **立即启动 Wave 1** (3 agents 并行)
   - V1-1: quickstart-verifier
   - V1-2: prefix-verifier
   - V1-3: errors-verifier

2. **Wave 1 完成后启动 Wave 2** (3 agents 并行)

3. **Wave 2 完成后启动 Wave 3** (串行执行)

4. **生成最终综合报告**

预计总时间：2.5-3 小时

---

## 备注

- 所有验证脚本将保存在 `scripts/` 目录
- 所有验证报告将保存在 `docs/verification/` 目录
- 发现的问题将创建为 GitHub issues（如果需要）
- 最终报告将作为 v0.2.0 release 的质量保证依据

---

## 方案修订记录

**修订日期**: 2026-06-13  
**修订原因**: 审计发现部分假设与项目实际情况不符

### 主要修订内容

1. **Task V1-1 (quickstart 验证)**:
   - ❌ 移除：使用 expect 工具模拟交互输入
   - ✅ 改为：non-interactive 自动化 + 手动交互测试记录
   - **理由**: 项目不使用 expect，所有现有测试都是 non-interactive 模式

2. **Task V2-1 (JSON 一致性)**:
   - ❌ 移除：使用 jq 解析、生成 JSON schema 文件
   - ✅ 改为：使用 assert_contains 验证、Markdown 文档化结构
   - **理由**: 项目使用字符串匹配模式，无 JSON schema 工具链

3. **时间估算调整**:
   - Wave 2-1: 45分钟 → 60分钟
   - 总时间: 2.5小时 → 3-4小时
   - **理由**: JSON 验证和文档提取需要更多时间

4. **协调机制补充**:
   - 明确产出目录结构（docs/verification/ 需创建）
   - 统一测试方法（现有 smoke test 模式）
   - 明确不依赖外部工具

5. **风险缓解补充**:
   - 新增"P0-P2 未完全实现"风险
   - 新增"测试工具缺失"风险及降级方案

### 审计结论

✅ **方案可行性**: 8.5/10  
✅ **批准状态**: 已批准执行  
⚠️ **注意事项**: Agent 团队需自适应调整细节

完整审计报告: `usability-verification-audit.md`
