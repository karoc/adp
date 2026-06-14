# 游戏开发示例

English: [README.md](README.md)

> 通过专业化的游戏玩法和图形工程 agent 学习游戏开发的 agent 编排模式。

## 你将学到

- **Agent 专业化**：将领域特定任务分配给专业化 agent
- **任务依赖**：使用任务依赖协调 agent 之间的工作
- **阶段式开发**：将项目组织成逻辑开发阶段
- **协作模式**：agent 如何协同完成复杂功能

## 前置条件

- **已安装 ADP**：运行 `adp version` 验证
- **Go 1.21+**：构建示例游戏引擎所需
- **5 分钟**：从设置到运行 agent 的时间预算

## 快速开始

```bash
# 导航到此示例目录
cd examples/game-development

# 一键设置
./setup.sh

# 验证配置
adp workspace show game-dev

# 启动 agent
adp run codex --workspace game-dev
```

**就是这样！** 无需编辑配置。

## 项目结构

```
game-development/
├── README.md                    # 本文件
├── README.zh-CN.md              # 中文文档
├── setup.sh                     # 一键设置脚本
├── workspace.yaml               # 工作区配置
├── AGENTS.md                    # Agent 协作模式
├── tasks.yaml                   # 示例任务定义
├── phases.yaml                  # 开发阶段结构
│
├── profiles/                    # Agent 配置
│   ├── gameplay-dev.yaml        # 游戏玩法工程师
│   └── graphics-dev.yaml        # 图形工程师
│
├── prompts/                     # Agent 指令
│   ├── gameplay-engineer.md     # 游戏玩法工程指南
│   └── graphics-engineer.md     # 图形工程指南
│
├── memory/                      # 共享上下文
│   └── game-context.md          # 游戏常量和约定
│
├── mcp/                         # MCP 服务器配置
│   └── config.yaml
│
└── project/                     # 最小游戏引擎
    ├── main.go                  # 入口点
    ├── go.mod
    └── game/
        ├── engine.go            # 核心游戏循环
        ├── physics.go           # 物理模拟
        ├── renderer.go          # 渲染系统
        └── engine_test.go       # 测试
```

## Agent 编排

本示例展示**领域专业化**：

### gameplay-dev（游戏玩法工程师）
- **专注**：游戏逻辑、物理、AI
- **技能**：算法设计、数值方法、系统编程
- **分配任务**：T1（重力）、T3（碰撞检测）

### graphics-dev（图形工程师）
- **专注**：渲染、着色器、性能
- **技能**：图形 API、优化、性能分析
- **分配任务**：T2（渲染优化）、T4（精灵渲染）

### 协作
对于需要两个领域的功能（如粒子系统）：
- 两个 agent 通过 `AGENTS.md` 模式协调
- 任务依赖确保正确的顺序
- 共享内存维护通用约定

## 试一试

### 1. 探索游戏引擎

```bash
cd project

# 运行测试
go test ./...

# 测试模式
./game-engine --test
# 输出: Physics: OK, Renderer: OK

# 运行演示（60 FPS 持续 5 秒）
./game-engine
```

### 2. 查看 Agent 配置

```bash
# 查看 agent 配置
cat profiles/gameplay-dev.yaml
cat profiles/graphics-dev.yaml

# 查看协作模式
cat AGENTS.md

# 检查任务分配
cat tasks.yaml
```

### 3. 启动 Agent

```bash
# 启动游戏玩法工程师
adp run codex --workspace game-dev --profile gameplay-dev

# 或启动图形工程师
adp run codex --workspace game-dev --profile graphics-dev
```

### 4. 分配任务

Agent 运行后：

```
用户："处理任务 T1 - 实现重力物理"

Agent：[读取 tasks.yaml，实现重力，编写测试]
```

## 任务流示例

来自 `tasks.yaml`：

```yaml
- id: T1
  title: "实现重力物理"
  assignee: gameplay-dev
  priority: high

- id: T3
  title: "添加碰撞检测"
  assignee: gameplay-dev
  depends_on: [T1]  # 等待 T1
```

这创建了依赖链：T3 仅在 T1 完成后开始。

## 开发阶段

来自 `phases.yaml`：

- **阶段 1（核心引擎）**：物理 + 渲染基础
- **阶段 2（游戏功能）**：碰撞 + 精灵
- **阶段 3（优化）**：特效 + 视觉反馈

每个阶段建立在前一个基础上，具有明确的里程碑标准。

## 性能目标

定义在 `memory/game-context.md` 中：

- **帧率**：60 FPS（16.67ms 每帧）
- **物理时间步**：固定 16.67ms
- **最大对象数**：100+ 不掉帧

Agent 查阅这个共享内存以保持一致性。

## 下一步

- **修改任务**：编辑 `tasks.yaml` 添加你自己的任务
- **自定义 Agent**：调整 agent 配置和提示
- **扩展引擎**：添加碰撞检测、精灵等功能
- **尝试其他示例**：
  - `examples/web-application` - 全栈 Web 开发
  - `examples/data-pipeline` - 带质量检查的 ETL 管道

## 验证

运行工作区诊断验证配置：

```bash
adp workspace doctor game-dev
```

所有检查应该通过 ✓

## 时间预算验证

- **设置**：< 2 分钟（`./setup.sh`）
- **首次运行 Agent**：< 5 分钟（从克隆到运行）
- **总计**：符合"5 分钟规则" ✓

## 了解更多

- [ADP 文档](../../docs/)
- [工作区配置指南](../../docs/workspace.zh-CN.md)
- [Agent 编排模式](../../docs/agent-patterns.zh-CN.md)
- [任务管理](../../docs/tasks.zh-CN.md)

## 故障排除

**设置失败？**
- 验证已安装 Go 1.21+：`go version`
- 检查已安装 ADP：`adp version`
- 确保项目可构建：`cd project && go build`

**Agent 看不到任务？**
- 验证工作区已注册：`adp workspace list`
- 检查 tasks.yaml 存在：`cat tasks.yaml`
- 查看 agent 配置：`cat profiles/gameplay-dev.yaml`

**测试失败？**
- 检查 Go 依赖：`cd project && go mod tidy`
- 带输出运行测试：`go test -v ./...`
