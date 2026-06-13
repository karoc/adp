# ADP 易用性测试报告

测试日期：2026-06-13
测试者视角：普通用户（首次接触 ADP）

## 测试方法

模拟一个普通开发者第一次使用 ADP 的完整流程，从安装、初始化、配置到日常使用，记录所有发现的问题和改进建议。

## 测试场景覆盖

- ✅ 查看帮助信息
- ✅ 初始化 ADP
- ✅ 工作空间管理（添加、列表、显示、诊断）
- ✅ 任务管理（添加、列表、查询）
- ✅ 阶段管理（添加、列表、状态）
- ✅ 进度报告（文本、JSON、中英文）
- ✅ 补全功能
- ✅ 错误处理
- ✅ 从项目目录内自动检测工作空间
- ✅ 会话和事件查询

---

## 发现的问题

### 🟢 优点（做得好的地方）

1. **错误提示清晰且友好**
   - 缺少参数时会显示 usage 和 `try: adp <command> --help`
   - 错误信息具体明确，如："workspace not found: invalid-ws"

2. **帮助系统完善**
   - 每个命令都有 `--help`
   - 提供了实际可复制的示例
   - 命令层级清晰（命令组 -> 子命令）

3. **自动工作空间检测**
   - 可以从项目目录内运行命令，自动识别 workspace
   - 减少了 `--workspace` 参数的重复输入

4. **多格式输出**
   - 支持 text 和 JSON 格式
   - JSON 输出结构清晰，适合工具集成
   - 双语支持（英文/简体中文）

5. **补全功能**
   - 提供了 completion values 命令
   - 支持动态值补全（workspaces、tasks、sessions 等）

6. **诊断功能详细**
   - doctor 命令提供 verbose 模式
   - 支持 JSON 输出以便自动化处理
   - 诊断信息包含具体的问题代码和路径

### 🟡 中等问题（可改进）

1. **workspace list 不支持 --format 参数**
   ```
   错误：adp: usage: adp workspace list
   预期：应该支持 --format json
   ```
   - 其他命令都支持 JSON 输出，workspace list 应该保持一致

2. **初次运行时缺少引导信息**
   - `adp init` 成功后只输出 "initialized ADP home"
   - 建议：可以提示下一步操作，例如："Next: add a workspace with 'adp workspace add <name> <path>'"

3. **运行不存在的 agent 时错误信息不够清晰**
   ```
   实际输出：Error: stdin is not a terminal
   ```
   - 这个错误信息与"codex 命令不存在"的关系不明显
   - 建议：检测命令是否存在，给出更明确的提示

4. **workspace add 时相同名称的提示不够友好**
   ```
   实际：adp: stat project root /tmp/another-path: stat /tmp/another-path: no such file or directory
   ```
   - 看起来是在检查路径，但实际是名称冲突
   - 建议：先检查名称是否已存在，给出明确提示

5. **空列表时的展示**
   - events、sessions、tasks 的空列表都只显示表头
   - 建议：可以添加提示信息，如："No tasks found. Create one with 'adp tasks add'"

### 🔴 需要改进的问题

1. **版本信息过于简单**
   ```
   当前输出：adp dev
   建议输出：
   adp version dev
   built: 2026-06-13
   commit: a966ad7
   go: go1.21.0
   ```

2. **workspace show 输出格式不统一**
   - workspace list 使用表格格式
   - workspace show 使用 YAML 风格的 key: value 格式
   - 建议：统一使用表格或支持 --format 参数

3. **缺少快速入门指导**
   - 运行 `adp` 或 `adp --help` 时，只有命令列表
   - 建议：添加一段简短的描述和快速入门链接
   ```
   adp - Agent Development Platform
   
   Manage AI agent workspaces, tasks, and runtime environments.
   
   Quick start: https://github.com/karoc/adp#quick-start
   
   Usage:
   ...
   ```

4. **示例中的时间格式不够友好**
   - task-20260611-0001（日期格式不易读）
   - session-20260613T020120-270f1f79（时间戳格式复杂）
   - 建议：考虑更简短的 ID 格式，或在显示时格式化

5. **缺少交互式引导**
   - 没有类似 `adp quickstart` 或 `adp wizard` 的交互式设置
   - 新用户可能不知道从何开始
   - 建议：添加交互式初始化命令

---

## 易用性评分

| 维度 | 评分 | 说明 |
|------|------|------|
| 文档完整性 | ⭐⭐⭐⭐☆ | 帮助文档完善，但缺少快速入门引导 |
| 错误提示 | ⭐⭐⭐⭐⭐ | 错误信息清晰，提供了有用的提示 |
| 命令一致性 | ⭐⭐⭐⭐☆ | 大部分命令风格一致，个别命令缺少 --format 支持 |
| 学习曲线 | ⭐⭐⭐☆☆ | 命令较多，需要一定学习时间 |
| 自动化友好 | ⭐⭐⭐⭐⭐ | JSON 输出完善，适合脚本集成 |
| **总体评分** | **⭐⭐⭐⭐☆** | **4/5 - 良好，有改进空间** |

---

## 优先级改进建议

### P0（高优先级）

1. 修复 `workspace list --format json` 支持
2. 改进 agent 不存在时的错误提示
3. 增强版本信息显示

### P1（中优先级）

4. 为空列表添加友好提示
5. 统一输出格式（支持 --format 参数）
6. 在 `adp --help` 中添加项目描述和快速入门链接
7. 改进首次运行后的引导信息

### P2（低优先级）

8. 考虑添加交互式引导命令
9. 优化 ID 格式的可读性
10. 添加更多实用的别名命令

---

## 测试结论

ADP 作为一个 terminal-first 的工具，在命令行体验方面做得很好：
- ✅ 错误处理规范
- ✅ 帮助系统完善
- ✅ JSON 输出适合自动化
- ✅ 补全功能完整

主要改进空间在于：
- ⚠️ 新用户引导不足
- ⚠️ 部分命令格式不一致
- ⚠️ 空状态提示不够友好

整体而言，这是一个**设计良好、实现扎实的 CLI 工具**，适合有一定命令行经验的开发者使用。建议优先改进新用户引导，降低初次使用的门槛。
