# ADP Phase 1 Development Plan

来源：[mvp.md](../mvp.md)

日期：2026-06-08

本文档用于把 ADP MVP 拆成可以并行推进的工程任务。目标不是现在实现全部功能，而是先固定架构边界、公共契约、目录所有权和验收标准，方便后续启动多个子 Agent 并行开发。

## 1. MVP 结论

ADP Phase 1 的产品核心是一个 terminal-first 的 Agent Runtime Environment：

- 管理 `$ADP_HOME`，默认 `~/.adp`。
- 注册 workspace，保存真实项目根目录与 ADP runtime 配置的映射。
- 为 Agent 构建临时 runtime overlay，让 Agent 在不污染真实项目目录的前提下看到 `AGENTS.md`、`CLAUDE.md`、`.codex/`、`.claude/` 等配置文件。
- 提供 Claude Code CLI 与 Codex CLI 两个 adapter。
- 支持 `adp init`、`adp workspace add`、`adp enter`、`adp run`。
- 记录本地 JSONL event log，为后续 replay、session restore、多 Agent 编排预留数据基础。

Phase 1 明确不做：

- Web UI / Dashboard。
- 云同步 / SaaS。
- 多 Agent 图形编排。
- 复杂权限沙箱。
- Windows 完整支持。保留接口，优先实现 Linux/macOS 可用路径。
- 真正的内核级 overlayfs 强依赖。MVP 默认使用可移植 symlink materialization，后续扩展 bind mount / overlayfs backend。

## 2. 技术栈建议

主实现语言：Go。

建议依赖：

- CLI：`spf13/cobra`，降低命令扩展成本。
- YAML：`gopkg.in/yaml.v3`。
- 测试：Go 标准库 `testing`，CLI 集成测试使用临时 HOME、临时 PATH 和 fake agent binary。

依赖原则：

- 依赖少而稳定。
- adapter 不直接依赖具体 CLI 框架。
- runtime/overlay 不依赖 adapter 具体实现。
- 业务包不能读取真实 `~/.adp`，必须通过 path/env abstraction 注入，保证测试可控。

## 3. 推荐目录结构

```txt
adp/
├── cmd/
│   └── adp/
│       └── main.go
├── internal/
│   ├── cli/
│   ├── paths/
│   ├── schema/
│   ├── workspace/
│   ├── runtime/
│   ├── overlay/
│   ├── adapters/
│   │   ├── registry.go
│   │   ├── adapter.go
│   │   ├── claude/
│   │   ├── codex/
│   │   └── shared/
│   ├── runner/
│   ├── shell/
│   └── events/
├── templates/
│   ├── claude/
│   └── codex/
├── examples/
│   └── basic-workspace/
├── docs/
├── README.md
└── go.mod
```

关键包职责：

- `internal/paths`：解析 `ADP_HOME`、runtime tmp dir、workspace 路径、日志路径。
- `internal/schema`：统一 YAML schema，仅定义数据结构和校验。
- `internal/workspace`：workspace registry 的创建、查询、列表、读写。
- `internal/overlay`：把 project root 和 generated files materialize 到 runtime dir。
- `internal/runtime`：编排 workspace config、adapter output、overlay lifecycle。
- `internal/adapters`：adapter interface、registry、公共渲染 helper。
- `internal/adapters/codex`：Codex runtime 文件、环境变量、启动命令。
- `internal/adapters/claude`：Claude runtime 文件、环境变量、启动命令。
- `internal/runner`：执行外部命令，处理 stdin/stdout/stderr、exit code、signal。
- `internal/shell`：实现 `adp enter` 的交互式 shell。
- `internal/events`：JSONL event log。

## 4. 本地数据布局

默认：

```txt
~/.adp/
├── config.yaml
├── workspaces/
│   └── game-a/
│       ├── workspace.yaml
│       ├── prompts/
│       │   └── base.md
│       ├── memory/
│       │   └── shared.md
│       ├── mcp/
│       │   └── config.yaml
│       └── profiles/
│           ├── codex.yaml
│           └── claude.yaml
└── logs/
    └── events.jsonl
```

测试和开发必须支持：

- `ADP_HOME` 覆盖默认 home。
- `ADP_RUNTIME_DIR` 覆盖默认临时 runtime 根目录。

## 5. Unified Workspace Schema

MVP schema 保持小而可扩展：

```yaml
version: 1

workspace:
  name: game-a

project:
  root: /srv/game-a

memory:
  enabled: true
  shared: memory/shared.md

prompts:
  base: prompts/base.md

rules:
  coding_style: strict

mcp:
  enabled: true
  config: mcp/config.yaml
  servers:
    - github
    - postgres

agents:
  codex:
    enabled: true
    profile: senior-engineer
    command: codex
  claude:
    enabled: true
    profile: architect
    command: claude
```

schema 约束：

- `workspace.name` 必须是安全目录名：建议 `^[a-zA-Z0-9][a-zA-Z0-9._-]*$`。
- `project.root` 保存绝对路径。
- prompt、memory、mcp 路径相对于 workspace config 目录。
- adapter 专属字段后续可以挂在 `agents.<name>.options`，MVP 不需要过度建模。

## 6. Runtime Overlay 设计

MVP 默认 backend：symlink materialization。

运行时目录示例：

```txt
${ADP_RUNTIME_DIR:-/tmp}/adp-runtime/
└── game-a-20260608T120102-8f3a/
    ├── .adp-runtime.yaml
    ├── AGENTS.md
    ├── CLAUDE.md
    ├── .codex/
    ├── .claude/
    ├── go.mod -> /srv/game-a/go.mod
    ├── cmd -> /srv/game-a/cmd
    ├── internal -> /srv/game-a/internal
    └── README.md -> /srv/game-a/README.md
```

设计选择：

- Agent 的 cwd 默认是 runtime root，而不是 `runtime/source`。这样 Agent 看到的目录形态更像真实项目根目录。
- 真实项目根目录下的文件和目录通过 symlink 镜像到 runtime root。
- ADP 生成的 Agent 配置文件直接写入 runtime root。
- 如果真实项目已有 `AGENTS.md`、`CLAUDE.md`、`.codex/`、`.claude/` 等保留路径，runtime 中以 ADP 生成内容为准，并把冲突写入 warning/event log。真实项目不被修改。
- 默认在 `adp run` 结束后清理 runtime 目录。
- 提供 `--keep-runtime` 便于调试。

后续 backend 预留：

- `symlink`：MVP 默认，跨 Linux/macOS。
- `bind`：Linux 可选，需要权限检查。
- `overlayfs`：Linux 可选，适合更高保真 runtime。

## 7. Adapter Contract

建议先冻结一个最小 adapter contract，再并行开发具体 adapter。

概念接口：

```go
type Adapter interface {
    Name() string
    Validate(ctx AdapterContext) error
    Render(ctx AdapterContext) (*RenderResult, error)
    Launch(ctx AdapterContext, runtime RuntimeHandle, extraArgs []string) (*LaunchSpec, error)
}

type RenderResult struct {
    Files []GeneratedFile
    Env   map[string]string
}

type GeneratedFile struct {
    Path string
    Mode fs.FileMode
    Data []byte
}

type LaunchSpec struct {
    Command string
    Args    []string
    Env     map[string]string
    Dir     string
}
```

边界原则：

- adapter 只负责“生成哪些文件、注入哪些环境变量、如何启动 Agent”。
- adapter 不创建 runtime dir。
- adapter 不读写 event log。
- adapter 不解析 CLI flags。
- adapter 不直接读取 `$HOME`。需要的路径由 `AdapterContext` 提供。

MVP adapter 输出：

- Codex：
  - 生成 `AGENTS.md`。
  - 生成 `.codex/config.toml` 或 `.codex/` 下的最小 runtime 配置，具体格式由实现阶段结合 Codex CLI 当前约定确认。
  - 默认命令 `codex`。
- Claude：
  - 生成 `CLAUDE.md`。
  - 生成 `.claude/` 下的最小 runtime 配置，具体格式由实现阶段结合 Claude Code CLI 当前约定确认。
  - 默认命令 `claude`。

需要注意：Codex CLI、Claude Code CLI 的配置格式可能随版本变化。实现 adapter 时要优先读取官方当前文档或本机 CLI 行为，避免凭记忆固化格式。

## 8. CLI 行为设计

### `adp init`

职责：

- 创建 `$ADP_HOME`。
- 创建 `workspaces/`、`logs/`。
- 创建默认 `config.yaml`。
- 幂等执行，多次运行不破坏已有配置。

验收：

- 空环境运行成功。
- 已存在环境运行成功。
- 支持 `ADP_HOME` 临时目录测试。

### `adp workspace add <name> <project-root>`

职责：

- 校验 workspace name。
- 校验 project root 存在且是目录。
- 写入 `$ADP_HOME/workspaces/<name>/workspace.yaml`。
- 初始化 prompts、memory、mcp、profiles 默认文件。
- 如果 workspace 已存在，默认报错；后续可加 `--force`。

验收：

- 相对 project root 会转换为绝对路径。
- 重名 workspace 有明确错误。
- 无效 name 有明确错误。

### `adp enter <workspace>`

职责：

- 解析 workspace。
- 构建 runtime overlay。
- 设置 `ADP_HOME`、`ADP_WORKSPACE`、`ADP_PROJECT_ROOT`、`ADP_RUNTIME_ROOT`、`ADP_SESSION_ID`。
- 在 runtime root 启动交互 shell。

限制：

- CLI 进程不能改变父 shell 的 cwd，所以 MVP 行为是启动一个子 shell。
- 后续可以增加 `adp shell-hook` 或 `adp env` 支持 parent shell integration。

验收：

- shell cwd 是 runtime root。
- `pwd`、env、runtime 文件可见。
- shell 退出后默认清理 runtime，`--keep-runtime` 保留。

### `adp run <agent> [--workspace <name>] [--profile <profile>] [--keep-runtime] [-- <agent-args>...]`

职责：

- 解析 workspace：优先 `--workspace`，其次 `ADP_WORKSPACE`，再考虑通过当前目录匹配已注册 project root。
- 查找 adapter。
- 构建 runtime overlay。
- 调用 adapter render。
- 执行 agent command。
- 透传 stdin/stdout/stderr 和 exit code。
- 记录 event log。

验收：

- 支持 fake `codex` / fake `claude` binary 集成测试。
- agent cwd 是 runtime root。
- agent 能看到 ADP 生成的配置文件。
- exit code 与 Agent 进程一致。

## 9. Event Log

格式：JSONL。

文件：

```txt
$ADP_HOME/logs/events.jsonl
```

事件建议：

```json
{"ts":"2026-06-08T12:01:02Z","type":"run_started","workspace":"game-a","agent":"codex","profile":"senior-engineer","runtime_path":"/tmp/adp-runtime/game-a-...","project_root":"/srv/game-a","session_id":"...","pid":12345}
{"ts":"2026-06-08T12:03:44Z","type":"run_finished","workspace":"game-a","agent":"codex","session_id":"...","exit_code":0,"duration_ms":162000}
```

约束：

- 不记录 API key、token、完整 env。
- 写入失败不应导致 Agent 无法启动，但要给 stderr warning。
- 每行一条完整 JSON，便于后续 streaming 和 grep。

## 10. 并行开发边界

### 串行前置：Contract Scaffold

从空仓库开始，不建议直接启动所有子 Agent 同时写代码。必须先完成一个很短的前置任务，冻结公共目录和接口。

负责人：Foundation Agent。

只做：

- 初始化 Go module。
- 建立目录结构。
- 定义 `internal/schema` 核心 struct。
- 定义 `internal/adapters` interface。
- 定义 `internal/overlay` input/output type。
- 定义 `internal/paths` layout type。
- 放置 TODO/stub，保证包可编译。

完成后才能大规模并行。

### 并行任务矩阵

| 子 Agent | 负责范围 | 独占路径 | 依赖 | 输出契约 | 验收 |
| --- | --- | --- | --- | --- | --- |
| A. CLI/Foundation | Cobra 命令骨架、全局 flag、错误格式、main wiring | `cmd/`, `internal/cli/`, `go.mod` | Contract Scaffold | 调用 workspace/runtime/runner 接口，不实现业务细节 | `adp --help`、命令参数解析测试通过 |
| B. Workspace Registry | `adp init`、workspace add/list/show 的底层能力、schema load/save | `internal/workspace/`, `internal/schema/` | `internal/paths` | `Registry` API、workspace config 文件 | temp `ADP_HOME` 单测通过 |
| C. Runtime Overlay | runtime session、symlink materialization、冲突策略、cleanup | `internal/runtime/`, `internal/overlay/` | schema + adapter `GeneratedFile` | `BuildRuntime(workspace, generatedFiles)` 返回 `RuntimeHandle` | temp project 集成测试通过 |
| D. Adapter Layer | adapter registry、Codex/Claude render、模板 | `internal/adapters/`, `templates/` | schema + adapter contract | `Adapter` 实现注册到 registry | 生成文件 golden tests 通过 |
| E. Runner/Shell/Events | 外部进程执行、交互 shell、event JSONL | `internal/runner/`, `internal/shell/`, `internal/events/` | paths + runtime handle | `Run(spec)`、`EnterShell(handle)`、`Log(event)` | fake binary 和 event log 测试通过 |
| F. Docs/Examples | README、CLI reference、example workspace | `README.md`, `docs/`, `examples/` | CLI 行为草案 | 可运行示例和用户文档 | 文档命令与实现保持一致 |
| G. Integration QA | 端到端测试、fake agents、CI 脚本 | `test/`, `.github/`, 允许少量 test helper | A-F 初版 | `go test ./...` + CLI e2e | 覆盖 init/add/run/enter 基本流 |

### 文件修改规则

- 每个子 Agent 默认只能修改自己的独占路径。
- 修改公共 contract 时必须先声明原因，并同步更新依赖方测试。
- `go.mod` 由 A 负责；其他子 Agent 如需新增依赖，先在任务说明里列出，不直接引入大依赖。
- `README.md` 由 F 负责；其他子 Agent 可在实现说明中提出文档变更点。
- `internal/schema` 初版由 Contract Scaffold 创建，B 可以继续完善；其他子 Agent 只消费 schema。
- `internal/adapters/adapter.go` 初版由 Contract Scaffold 创建，D 可以继续完善；C/E 只消费接口。

### 合并顺序

推荐合并顺序：

1. Contract Scaffold。
2. A + B，可以先合并，形成可运行 CLI 与 workspace registry。
3. C + D，可以并行开发，随后由 A 或 Integration Agent 做 wiring。
4. E 合并 runner/events/shell。
5. G 做端到端修正。
6. F 最后校正文档，或与实现并行但最终以实现为准。

## 11. 子 Agent 任务说明模板

后续启动子 Agent 时，建议把下面模板填入任务：

```txt
你负责 ADP Phase 1 的 <任务名>。

背景文档：
- mvp.md
- docs/phase1-development-plan.md

你的独占路径：
- <paths>

你可以读取全仓库，但默认不要修改独占路径之外的文件。
如果必须修改公共接口，先在回复中说明原因和影响面。

必须保持：
- go test ./... 通过。
- 支持 ADP_HOME 指向临时目录。
- 不读取或污染真实 ~/.adp。
- 不修改真实项目目录，只写 runtime/temp 和测试目录。

交付：
- 实现代码。
- 对应单测或集成测试。
- 简短说明已覆盖的命令/行为。
```

## 12. 端到端验收场景

Phase 1 完成时，至少满足以下场景：

```bash
export ADP_HOME="$(mktemp -d)"
export ADP_RUNTIME_DIR="$(mktemp -d)"

adp init
adp workspace add game-a /srv/game-a
adp run codex --workspace game-a -- --version
adp run claude --workspace game-a -- --version
adp enter game-a
```

验收点：

- `$ADP_HOME/workspaces/game-a/workspace.yaml` 存在且 project root 正确。
- runtime root 中存在 ADP 生成的 Agent 配置文件。
- 真实 `/srv/game-a` 不新增 `AGENTS.md`、`CLAUDE.md`、`.codex/`、`.claude/`。
- event log 记录 run start/finish。
- fake agent e2e 能断言 cwd、env、参数透传、exit code。

## 13. 主要风险和处理策略

Codex/Claude 配置格式变化：

- adapter 实现阶段必须验证当前 CLI 或官方文档。
- 把格式细节限制在 adapter 包内，不泄漏到 runtime/workspace。

symlink overlay 与真实项目已有配置冲突：

- 保留路径由 ADP 生成内容覆盖 runtime 视图。
- 真实项目不修改。
- event log 记录冲突 warning。

`adp enter` 改变父 shell cwd 的误解：

- MVP 明确启动子 shell。
- 文档说明后续 shell hook 能力。

并行开发接口漂移：

- 先做 Contract Scaffold。
- 公共接口变更集中在短窗口内处理。
- G 负责端到端测试把漂移暴露出来。

真实 HOME 污染：

- 所有测试必须通过 `ADP_HOME`。
- 业务包通过 `internal/paths` 获取路径，不直接调用 `os.UserHomeDir()`。

