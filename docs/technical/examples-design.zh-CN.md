# Examples 目录设计文档

## 概要

本文档定义 `examples/` 目录的组织结构、内容规范和质量标准。目标是提供特定领域的生产就绪示例，使用户能够在 5 分钟内从克隆到运行 agent。

## 当前状态审计

### 现有示例：`examples/basic-workspace/`

**优势：**
- ✅ 完整的 workspace 结构，包含所有必需目录（prompts, memory, mcp, profiles）
- ✅ 双语文档（英文 + 中文）
- ✅ 详细的设置说明，提供可复制粘贴的命令
- ✅ 使用 fake CLI 的无 provider 入门路径
- ✅ 清晰的关注点分离（workspace 配置、profiles、prompts、memory）

**局限性：**
- ❌ 通用示例，缺乏特定领域上下文
- ❌ 没有实际项目代码 - 用户必须自行提供
- ❌ 使用前需要多次手动编辑（workspace name、project root）
- ❌ 尽管 ADP 支持 tasks.yaml，但没有任务/阶段示例
- ❌ 没有展示 agent 协同模式的 AGENTS.md 示例

**价值实现时间：** 约 10-15 分钟（需要创建项目、编辑路径、验证）

## 行业最佳实践研究

### 模式分析

基于对 Docker、Vercel 和 Kubernetes 示例仓库的研究：

**成功模式：**
1. **基于分类的组织** - 按用例或领域分组示例
2. **自包含目录** - 每个示例都是完整的可运行单元
3. **一致的结构** - 所有示例采用可预测的文件布局
4. **README 优先** - 每个示例都以清晰的设置说明开始
5. **可复制粘贴的命令** - 无需手动编辑即可开始
6. **验证步骤** - 明确的成功标准和预期输出
7. **渐进式复杂度** - 基础 → 中级 → 高级路径

**每个示例的关键文件：**
- `README.md` - 包含前置条件、快速开始、架构说明的设置指南
- 配置文件 - 即用型，无需编辑的占位符
- 示例项目 - 最小但功能完整的代码库
- 验证脚本 - 自动验证设置成功

**参考来源：**
- [Docker 示例最佳实践](https://docs.docker.com/articles/dockerfile_best-practices/)
- [Vercel 示例仓库](https://github.com/vercel/examples)
- [Kubernetes 配置最佳实践](https://kubernetes.io/blog/2025/11/25/configuration-good-practices/)

## 建议的目录结构

```
examples/
├── README.md                          # 示例索引和导航
├── README.zh-CN.md
├── _templates/                        # 共享可复用组件
│   ├── workspace-base.yaml
│   ├── profiles/
│   │   ├── codex.yaml
│   │   ├── claude.yaml
│   │   └── architect.yaml
│   ├── prompts/
│   │   ├── coding-style.md
│   │   └── testing-requirements.md
│   └── mcp/
│       └── config.yaml
│
├── game-development/                  # 领域：游戏开发
│   ├── README.md
│   ├── README.zh-CN.md
│   ├── workspace.yaml
│   ├── AGENTS.md                     # Agent 协同模式
│   ├── tasks.yaml                    # 任务定义示例
│   ├── phases.yaml                   # 阶段结构示例
│   ├── prompts/
│   │   ├── gameplay-engineer.md
│   │   └── graphics-engineer.md
│   ├── profiles/
│   │   ├── gameplay-dev.yaml
│   │   └── graphics-dev.yaml
│   ├── memory/
│   │   └── game-context.md
│   ├── mcp/
│   │   └── config.yaml
│   └── project/                      # 最小可运行游戏
│       ├── main.go
│       ├── go.mod
│       ├── game/
│       │   ├── engine.go
│       │   └── physics.go
│       └── README.md
│
├── web-application/                   # 领域：Web 开发
│   ├── README.md
│   ├── README.zh-CN.md
│   ├── workspace.yaml
│   ├── AGENTS.md
│   ├── tasks.yaml
│   ├── phases.yaml
│   ├── prompts/
│   │   ├── frontend-engineer.md
│   │   └── backend-engineer.md
│   ├── profiles/
│   │   ├── frontend-dev.yaml
│   │   └── backend-dev.yaml
│   ├── memory/
│   │   └── api-contracts.md
│   ├── mcp/
│   │   └── config.yaml
│   └── project/                      # 最小 API + 前端
│       ├── backend/
│       │   ├── main.go
│       │   ├── go.mod
│       │   └── api/
│       ├── frontend/
│       │   ├── package.json
│       │   ├── src/
│       │   └── public/
│       └── README.md
│
└── data-pipeline/                     # 领域：数据工程
    ├── README.md
    ├── README.zh-CN.md
    ├── workspace.yaml
    ├── AGENTS.md
    ├── tasks.yaml
    ├── phases.yaml
    ├── prompts/
    │   ├── etl-engineer.md
    │   └── data-quality.md
    ├── profiles/
    │   ├── etl-dev.yaml
    │   └── qa-dev.yaml
    ├── memory/
    │   └── pipeline-schema.md
    ├── mcp/
    │   └── config.yaml
    └── project/                       # 最小 ETL 管道
        ├── main.go
        ├── go.mod
        ├── pipeline/
        │   ├── extract.go
        │   ├── transform.go
        │   └── load.go
        └── README.md
```

## 设计原则

### 1. 零编辑快速启动

**问题：** 当前 basic-workspace 使用前需要编辑 workspace.yaml。

**解决方案：** 每个示例都包含一个自包含的项目目录，绝对路径开箱即用。

```bash
# 用户工作流 - 无需编辑
cd examples/game-development
./setup.sh                           # 一键设置
adp workspace show game-dev          # 验证
adp run codex --workspace game-dev   # 启动 agent
```

### 2. 特定领域上下文

**问题：** 通用示例无法展示真实世界的 agent 协同模式。

**解决方案：** 每个示例代表一个完整领域，包含：
- 真实的项目结构
- 特定领域的 agent profiles（gameplay-dev, graphics-dev）
- 相关的任务定义（实现物理系统、优化渲染）
- 上下文化的 memory（游戏状态、物理常量）

### 3. 渐进式学习路径

示例按复杂度排序：

1. **game-development** - 单一领域，2 个 agents，简单任务
2. **web-application** - 多组件（前端/后端），2+ agents，API 契约
3. **data-pipeline** - 复杂编排，数据质量检查，多阶段工作流

### 4. 可复制粘贴的命令

每个 README 都包含：
```bash
# 完整命令序列，无占位符
export ADP_HOME="$(pwd)/.adp-state"
./setup.sh
adp workspace doctor game-dev
adp run codex --workspace game-dev
```

### 5. 内置验证

每个示例都包含：
- README 中的预期输出样本
- 自动验证脚本
- 成功标准检查清单

## 示例规格说明

### 示例 1：游戏开发

**领域：** 具有游戏玩法和图形专业化的游戏开发

**用例：** 小型游戏引擎项目，agents 协作处理游戏逻辑和渲染优化。

**项目结构：**
```
project/
├── main.go              # 入口点，约 50 行
├── go.mod
├── game/
│   ├── engine.go        # 核心游戏循环
│   ├── physics.go       # 物理模拟
│   └── renderer.go      # 渲染桩代码
└── README.md
```

**Agents：**
- `gameplay-dev` - 专注于游戏逻辑、物理、AI
- `graphics-dev` - 专注于渲染、着色器、优化

**示例任务（tasks.yaml）：**
```yaml
tasks:
  - id: T1
    title: "实现重力物理"
    description: "向物理引擎添加重力加速度"
    assignee: gameplay-dev
    
  - id: T2
    title: "优化渲染循环"
    description: "减少渲染器中的绘制调用"
    assignee: graphics-dev
```

**Memory 上下文：**
```markdown
# game-context.md
- 游戏目标帧率：60 FPS
- 物理时间步：16.67ms
- 坐标系统：Y 轴向上，右手系
```

**设置时间：** < 3 分钟

**验证：**
```bash
cd project
go build && ./game --test
# 预期输出："Physics: OK, Renderer: OK"
```

### 示例 2：Web 应用

**领域：** 全栈 Web 开发，包含 API 后端和 React 前端

**用例：** 带有 Web 界面的 REST API 服务，展示前端/后端 agent 协作。

**项目结构：**
```
project/
├── backend/
│   ├── main.go          # HTTP 服务器，约 80 行
│   ├── go.mod
│   └── api/
│       ├── handlers.go
│       └── models.go
├── frontend/
│   ├── package.json
│   ├── src/
│   │   ├── App.tsx
│   │   └── api.ts       # API 客户端
│   └── public/
└── README.md
```

**Agents：**
- `frontend-dev` - React、TypeScript、UI 组件
- `backend-dev` - Go、REST API、数据模型

**示例任务：**
```yaml
tasks:
  - id: T1
    title: "添加用户认证端点"
    description: "POST /api/auth/login 使用 JWT"
    assignee: backend-dev
    
  - id: T2
    title: "创建登录表单组件"
    description: "React 表单调用 /api/auth/login"
    assignee: frontend-dev
    depends_on: [T1]
```

**Memory 上下文：**
```markdown
# api-contracts.md
## 认证
- POST /api/auth/login
- 请求：{ username, password }
- 响应：{ token, expires_at }
```

**设置时间：** < 4 分钟

**验证：**
```bash
# 终端 1
cd project/backend && go run main.go

# 终端 2
cd project/frontend && npm install && npm start

# 检查：http://localhost:3000 加载，API 在 :8080 响应
```

### 示例 3：数据管道

**领域：** 带质量检查的 ETL 数据管道

**用例：** 提取、转换和加载数据的数据处理管道，带质量验证。

**项目结构：**
```
project/
├── main.go              # 管道编排器
├── go.mod
├── pipeline/
│   ├── extract.go       # 数据提取
│   ├── transform.go     # 数据转换
│   ├── load.go          # 数据加载
│   └── validate.go      # 质量检查
├── data/
│   ├── input/           # 示例输入数据
│   └── output/          # 预期输出
└── README.md
```

**Agents：**
- `etl-dev` - 管道逻辑、数据转换
- `qa-dev` - 质量检查、验证规则

**示例任务：**
```yaml
tasks:
  - id: T1
    title: "添加空值处理"
    description: "在转换步骤中处理缺失数据"
    assignee: etl-dev
    
  - id: T2
    title: "添加数据完整性检查"
    description: "验证所有必需字段存在"
    assignee: qa-dev
    
  - id: T3
    title: "集成测试"
    description: "端到端管道测试"
    depends_on: [T1, T2]
```

**阶段：**
```yaml
phases:
  - id: P1
    name: "开发"
    tasks: [T1, T2]
    
  - id: P2
    name: "测试"
    tasks: [T3]
    depends_on: [P1]
```

**设置时间：** < 5 分钟

**验证：**
```bash
cd project
go build && ./pipeline --input data/input --output /tmp/output
diff -r /tmp/output data/output/
# 预期：无差异
```

## 可复用组件（_templates/）

### 目的

通过提供示例可扩展的共享组件来避免重复：

```yaml
# game-development/workspace.yaml 包含模板
extends: ../_templates/workspace-base.yaml

workspace:
  name: game-dev

project:
  root: ./project  # 相对于捆绑项目的路径
```

### 模板内容

**workspace-base.yaml：**
```yaml
version: 1

memory:
  enabled: true

rules:
  coding_style: strict

mcp:
  enabled: true
  config: mcp/config.yaml
```

**profiles/codex.yaml：**
```yaml
profile: default
command: codex
notes:
  - 使用 AGENTS.md 作为主要运行时指令
  - 查阅 tasks.yaml 了解当前工作项
```

**prompts/coding-style.md：**
```markdown
# 编码风格指南
- 优先考虑可读性而非巧妙性
- 为新功能编写测试
- 保持函数在 50 行以内
```

## 设置脚本

每个示例都包含一个 `setup.sh` 脚本：

```bash
#!/usr/bin/env bash
set -euo pipefail

EXAMPLE_NAME="game-development"
EXAMPLE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 创建隔离的 ADP 状态
export ADP_HOME="${EXAMPLE_DIR}/.adp-state"
export ADP_RUNTIME_DIR="${EXAMPLE_DIR}/.adp-runtime"

echo "正在设置 ${EXAMPLE_NAME} 示例..."

# 初始化 ADP
adp init

# 复制 workspace 配置
mkdir -p "${ADP_HOME}/workspaces"
cp -R "${EXAMPLE_DIR}/workspace.yaml" "${ADP_HOME}/workspaces/${EXAMPLE_NAME}/"
cp -R "${EXAMPLE_DIR}/prompts" "${ADP_HOME}/workspaces/${EXAMPLE_NAME}/"
cp -R "${EXAMPLE_DIR}/profiles" "${ADP_HOME}/workspaces/${EXAMPLE_NAME}/"
cp -R "${EXAMPLE_DIR}/memory" "${ADP_HOME}/workspaces/${EXAMPLE_NAME}/"
cp -R "${EXAMPLE_DIR}/mcp" "${ADP_HOME}/workspaces/${EXAMPLE_NAME}/"

# 验证 workspace
adp workspace doctor "${EXAMPLE_NAME}"

echo "✓ 设置完成！"
echo ""
echo "下一步："
echo "  adp workspace show ${EXAMPLE_NAME}"
echo "  adp run codex --workspace ${EXAMPLE_NAME}"
```

## README 模板

每个示例 README 遵循以下结构：

```markdown
# [示例名称]

> [一句话描述此示例展示的内容]

## 你将学到什么

- [关键概念 1]
- [关键概念 2]
- [关键概念 3]

## 前置条件

- Go 1.21+（针对此示例）
- 已安装 ADP（`adp version`）

## 快速开始

[可复制粘贴的命令块 - 无需编辑]

## 项目结构

[project/ 目录的树视图及说明]

## Agent 协同

[说明 agents 如何在此领域协作]

## 试用

[建议的命令和预期输出]

## 下一步

- [相关示例链接]
- [相关文档链接]
```

## 验证标准

每个示例必须通过：

### 1. 时间预算
- 设置脚本在 < 2 分钟内完成
- 第一次 agent 运行在从克隆开始 < 5 分钟内启动

### 2. 零编辑要求
```bash
# 必须无需任何文件编辑即可工作
git clone <repo>
cd examples/game-development
./setup.sh
adp run codex --workspace game-dev
```

### 3. 项目有效性
- 项目代码编译/运行无错误
- 至少包含一个通过的测试
- README 验证步骤成功

### 4. 文档完整性
- README 包含前置条件
- README 包含可复制粘贴的设置命令
- README 包含预期输出样本
- 双语（英文 + 中文）

### 5. Workspace 验证
```bash
adp workspace doctor <name>  # 必须返回 0
adp workspace show <name>    # 必须打印有效 YAML
```

## 迁移计划

### 阶段 1：基础（第 1 周）
1. 创建包含可复用组件的 `_templates/`
2. 更新 `examples/README.md` 导航
3. 创建 `setup.sh` 模板脚本

### 阶段 2：第一个示例（第 1-2 周）
1. 实现 `game-development/` 示例
2. 创建最小游戏项目
3. 编写和测试 README
4. 验证 < 5 分钟时间预算

### 阶段 3：其他示例（第 2-3 周）
1. 实现 `web-application/` 示例
2. 实现 `data-pipeline/` 示例
3. 交叉链接 READMEs
4. 最终验证

### 阶段 4：完善（第 3 周）
1. 双语文档完成
2. 自动化 CI 验证
3. 更新主文档引用示例

## 成功指标

- **首次运行时间：** 任何示例 < 5 分钟
- **设置成功率：** 100% 无需手动编辑
- **文档清晰度：** 用户能从 README 理解 agent 协同模式
- **可复用性：** 用户将示例作为实际项目的起点

## 未来扩展

初始 3 个示例之后：

1. **CLI 工具开发** - 构建 CLI 应用的示例
2. **微服务** - 带服务网格的多服务架构
3. **机器学习** - 带实验跟踪的模型训练管道
4. **基础设施** - Terraform/Kubernetes 部署自动化

## 参考资料

- 当前 basic-workspace：`/srv/agent-development-platform/examples/basic-workspace/`
- Docker compose 示例：https://github.com/docker/awesome-compose
- Vercel 示例：https://github.com/vercel/examples
- Kubernetes 模式：https://kubernetes.io/examples/
