# Web 应用示例

English: [README.md](README.md)

> 通过 backend 和 frontend agent 专业化学习 Go REST API 和 React 应用的全栈开发模式。

## 你将学到

- **API 契约优先开发**：在实现前定义契约
- **Agent 专业化**：Backend vs Frontend 职责划分
- **任务依赖**：Frontend 任务等待 backend API 可用
- **全栈集成**：连接 Go backend 与 React frontend

## 前置条件

- **已安装 ADP**：运行 `adp version` 验证
- **Go 1.21+**：Backend API 所需
- **Node.js 16+**：React frontend 所需
- **4 分钟**：从设置到运行的时间预算

## 快速开始

```bash
# 导航到此示例目录
cd examples/web-application

# 一键设置
./setup.sh

# 启动 backend（终端 1）
cd project/backend
./backend-server

# 启动 frontend（终端 2）
cd project/frontend
npm start

# 打开浏览器
# Frontend: http://localhost:3000
# Backend API: http://localhost:8080
```

**就是这样！** 无需编辑配置。

## 项目结构

```
web-application/
├── README.md                    # 本文件
├── README.zh-CN.md              # 中文文档
├── setup.sh                     # 一键设置脚本
├── workspace.yaml               # 工作区配置
├── AGENTS.md                    # Agent 协作模式
├── tasks.yaml                   # 示例任务定义
├── phases.yaml                  # 开发阶段结构
│
├── profiles/                    # Agent 配置
│   ├── backend-dev.yaml         # Backend 工程师
│   └── frontend-dev.yaml        # Frontend 工程师
│
├── prompts/                     # Agent 指令
│   ├── backend-engineer.md      # Backend 指南
│   └── frontend-engineer.md     # Frontend 指南
│
├── memory/                      # 共享上下文
│   └── api-contracts.md         # API 端点契约
│
├── mcp/                         # MCP 服务器配置
│   └── config.yaml
│
└── project/                     # 全栈应用
    ├── backend/                 # Go REST API
    │   ├── main.go              # HTTP 服务器
    │   ├── go.mod
    │   └── api/
    │       ├── handlers.go      # API 端点
    │       └── handlers_test.go # 测试
    │
    └── frontend/                # React 应用
        ├── package.json
        ├── public/
        │   └── index.html
        └── src/
            ├── App.js           # 主组件
            ├── api.js           # API 客户端
            ├── App.test.js      # 组件测试
            └── api.test.js      # API 测试
```

## Agent 编排

本示例展示**API 契约优先开发**：

### backend-dev（Backend 工程师）
- **专注**：REST API、数据模型、认证
- **技能**：Go、HTTP 服务器、API 设计、验证
- **分配任务**：T1-T3（health、users、login 端点）、T7（分页）

### frontend-dev（Frontend 工程师）
- **专注**：React 组件、UI/UX、API 集成
- **技能**：React、JavaScript、响应式设计
- **分配任务**：T4-T6（API 客户端、登录 UI、用户列表）、T8（搜索）

### 协作模式：契约优先

1. **两个 agent 同意 API 契约** - 定义在 `memory/api-contracts.md`
2. **Backend 实现端点** - 遵循契约规范
3. **Frontend 集成 API** - 使用符合契约的端点
4. **集成测试验证** - 确保两部分协同工作

这使得**并行开发**成为可能，无需阻塞。

## 试一试

### 1. 探索 API

```bash
cd project/backend

# 启动服务器
./backend-server

# 在另一个终端测试端点
curl http://localhost:8080/api/health
curl http://localhost:8080/api/users
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"test"}'
```

### 2. 探索 Frontend

```bash
cd project/frontend

# 启动开发服务器
npm start

# 在浏览器打开 http://localhost:3000
```

**尝试 UI**：
- 查看服务器健康状态
- 点击"Fetch Users"加载用户列表
- 使用用户名"alice"或"bob"登录（任意密码）

### 3. 查看 API 契约

```bash
# 查看端点定义
cat memory/api-contracts.md
```

此文档定义了 backend 和 frontend 之间的契约。

### 4. 启动 Agent

```bash
# 启动 backend 工程师
adp run codex --workspace web-app --profile backend-dev

# 或启动 frontend 工程师
adp run codex --workspace web-app --profile frontend-dev
```

### 5. 分配任务

Agent 运行后：

```
用户："处理任务 T7 - 为用户端点添加分页"

Agent：[读取 tasks.yaml，实现分页，更新 API 契约]
```

## 任务流示例

来自 `tasks.yaml`：

```yaml
- id: T3
  title: "实现登录端点"
  assignee: backend-dev
  priority: high

- id: T4
  title: "创建 API 客户端模块"
  assignee: frontend-dev
  depends_on: [T1, T2, T3]  # 等待 backend
  
- id: T5
  title: "构建登录表单组件"
  assignee: frontend-dev
  depends_on: [T4]  # 等待 API 客户端
```

这创建了依赖链：Frontend 等待 backend API 就绪。

## 开发阶段

来自 `phases.yaml`：

- **阶段 1（API 基础）**：Backend 端点和测试
- **阶段 2（Frontend 集成）**：React UI 使用 API
- **阶段 3（功能增强）**：分页、搜索
- **阶段 4（集成与部署）**：端到端测试

每个阶段都有明确的里程碑，并依赖前一阶段。

## API 端点

定义在 `memory/api-contracts.md` 中：

### GET /api/health
服务器健康检查

### GET /api/users
列出所有用户（返回 {id, username, email} 数组）

### POST /api/auth/login
用户认证（返回 {token, expires_at}）

## 测试

```bash
# Backend 测试
cd project/backend
go test ./...
# 输出：6/6 测试通过

# Frontend 测试
cd project/frontend
npm test
```

## 演示凭证

登录端点使用：
- **用户名**：`alice` 或 `bob`
- **密码**：任意值（演示模式，不验证）

## 下一步

- **修改 API 契约**：编辑 `memory/api-contracts.md` 并实现
- **添加新端点**：更新 backend，文档化契约，在 frontend 集成
- **自定义 Agent**：调整配置和提示以适应你的工作流
- **尝试其他示例**：
  - `examples/game-development` - 带物理的游戏引擎
  - `examples/data-pipeline` - 带质量检查的 ETL 管道

## 验证

运行工作区诊断验证配置：

```bash
adp workspace doctor web-app
```

所有检查应该通过 ✓

## 时间预算验证

- **设置**：< 4 分钟（`./setup.sh` + npm install）
- **Backend 启动**：< 5 秒
- **Frontend 启动**：< 30 秒（首次）
- **总计**：符合"5 分钟规则" ✓

## 架构亮点

### Backend (Go)
- 标准库 HTTP 服务器（无框架）
- JSON 请求/响应处理
- 开发用 CORS 中间件
- 一致的错误格式：`{"error": "message"}`

### Frontend (React)
- 函数式组件和 hooks
- 集中化 API 客户端
- 加载和错误状态处理
- 响应式 CSS 设计

### 通信
- HTTP 上的 REST API
- JSON 负载格式
- 启用 CORS 支持跨域请求

## 了解更多

- [ADP 文档](../../docs/)
- [工作区配置指南](../../docs/workspace.zh-CN.md)
- [Agent 编排模式](../../docs/agent-patterns.zh-CN.md)
- [任务管理](../../docs/tasks.zh-CN.md)

## 故障排除

**设置失败？**
- 验证 Go 1.21+：`go version`
- 验证 Node.js 16+：`node --version`
- 检查端口 8080 和 3000 可用

**Backend 无法启动？**
- 检查端口 8080 是否被占用：`lsof -i :8080`
- 查看 backend 日志错误

**Frontend 无法连接？**
- 验证 backend 在端口 8080 运行
- 检查浏览器控制台 CORS 错误
- 验证 frontend/.env 中的 API_BASE_URL

**Agent 看不到任务？**
- 验证工作区已注册：`adp workspace list`
- 检查 tasks.yaml 存在且是有效 YAML
- 查看 agent 配置：`cat profiles/backend-dev.yaml`
