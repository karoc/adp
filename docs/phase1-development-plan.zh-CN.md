# ADP Phase 1 Development Plan

来源：[mvp.md](../mvp.md)

日期：2026-06-08

English: [phase1-development-plan.md](phase1-development-plan.md)

本文档用于把 ADP MVP 拆成可以并行推进的工程任务。目标不是现在实现全部功能，而是先固定架构边界、公共契约、目录所有权和验收标准，方便后续启动多个子 Agent 并行开发。

## 1. MVP 结论

ADP Phase 1 的产品核心是一个 terminal-first 的 Agent Runtime Environment：

- 管理 `$ADP_HOME`，默认 `~/.adp`。
- 注册 workspace，保存真实项目根目录与 ADP runtime 配置的映射。
- 为 Agent 构建临时 runtime overlay，让 Agent 在不污染真实项目目录的前提下看到 `AGENTS.md`、`CLAUDE.md`、`.codex/`、`.claude/` 等配置文件。
- 提供 Claude Code CLI 与 Codex CLI 两个 adapter。
- 支持 `adp init`、`adp workspace add/list/show/doctor/remove/rename`、`adp env`、`adp shell-hook`、`adp completion`、`adp events list`、`adp sessions list/show`、`adp runtime prune`、`adp enter`、`adp run`。
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

- CLI：优先使用 Go 标准库手写命令解析；只有命令面复杂到明显需要时再引入 CLI 框架。
- YAML：`gopkg.in/yaml.v3`。
- 测试：Go 标准库 `testing`，CLI 集成测试使用临时 HOME、临时 PATH 和 fake agent binary。

依赖原则：

- 依赖少而稳定。
- adapter 不直接依赖具体 CLI 框架。
- runtime/overlay 不依赖 adapter 具体实现。
- 业务包不能读取真实 `~/.adp`，必须通过 path/env abstraction 注入，保证测试可控。
- 新增第三方依赖前必须确认许可证与 ADP 的非商业源码可用授权策略兼容。

## 2.1 License and Engineering Gates

ADP 采用 source-available non-commercial 授权模式，而不是 OSI 定义下的开源许可证。

- 公共非商业使用：`PolyForm Noncommercial License 1.0.0`。
- 允许个人和组织为了学习、研究、评估、非商业开源协作等目的免费使用。
- 必须保留 `LICENSE` 中的 `Required Notice:`，声明 ADP 和版权归属。
- 任何商业使用都必须取得单独的付费商业授权，详见 `COMMERCIAL.md`。

文档约束：

- 默认文档文件使用英文。
- 项目维护的文档都应提供 `*.zh-CN.md` 简体中文 counterpart。
- `LICENSE` 是权威英文法律文本；任何法律条款翻译只用于说明，不能替代英文许可证。

代码规模约束：

- 项目代码文件必须控制在 700 行以内。
- 超过 700 行前必须按职责拆分。
- 生成文件、vendor、lockfile、许可证和长文档不受此限制。
- 本地检查命令：`scripts/check-file-lines.sh`。

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
│   ├── sessions/
│   └── events/
├── templates/
│   ├── claude/
│   └── codex/
├── examples/
│   └── basic-workspace/
├── scripts/
│   └── check-file-lines.sh
├── docs/
│   └── engineering-standards.md
├── COMMERCIAL.md
├── LICENSE
├── README.md
└── go.mod
```

关键包职责：

- `internal/paths`：解析 `ADP_HOME`、runtime tmp dir、workspace 路径、日志路径。
- `internal/schema`：统一 YAML schema，仅定义数据结构和校验。
- `internal/workspace`：workspace registry 的创建、查询、列表、读写。
- `internal/overlay`：把 project root 和 generated files materialize 到 runtime dir。
- `internal/runtime`：编排 workspace config、adapter output、overlay lifecycle、runtime env、manifest 和 runtime pruning。
- `internal/adapters`：adapter interface、registry、公共渲染 helper。
- `internal/adapters/codex`：Codex runtime 文件、环境变量、启动命令。
- `internal/adapters/claude`：Claude runtime 文件、环境变量、启动命令。
- `internal/runner`：执行外部命令，处理 stdin/stdout/stderr、exit code、signal。
- `internal/shell`：实现 `adp enter`、shell exports 渲染、parent-shell hook 渲染和 shell completion 渲染。
- `internal/sessions`：把本地事件聚合为 session history summary 和 detail 视图。
- `internal/events`：JSONL event log 的写入和查询。

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

仓库也包含 `examples/basic-workspace`，作为可复制的 workspace 配置示例，内含 base prompt、shared memory、MCP config 和 Codex/Claude profiles。它是 local-first workspace 配置的文档和测试材料，不是托管模板服务。

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
- 保留或过期的 runtime 目录可通过 `adp runtime prune` 检查和清理。
- runtime prune 只删除包含 `.adp-runtime.yaml` 且 `generated_by: adp` 的 ADP runtime 目录。
- 默认保留 `keep: true` 的 runtime，只有传入 `--include-kept` 才会纳入清理候选。

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

### `adp workspace list`

职责：

- 列出已注册 workspace 的名称、project root 和 ADP workspace dir。

验收：

- 空 registry 输出表头。
- 多个 workspace 按名称排序。

### `adp workspace show <name>`

职责：

- 输出单个 workspace 的 project root、workspace dir、memory 状态和 MCP 状态。

验收：

- workspace 不存在时有明确错误。
- 输出字段稳定，便于终端阅读。

### `adp workspace doctor [name]`

职责：

- 不传 name 时检查全部已注册 workspace；传 name 时只检查指定 workspace。
- 检查 config load/validation、project root 是否可访问、prompt、memory、MCP、profile 文件引用、路径逃逸和 agent command 默认值。
- 以稳定终端表格输出 diagnostics。

验收：

- 健康 workspace 输出 `ok - no issues`。
- error 级 diagnostics 返回非零退出码。
- `DiagnoseAll` 对单个异常 workspace 给出报告，不阻断其它 workspace 的检查。

### `adp workspace remove <name>`

职责：

- 删除 ADP workspace 目录，不触碰真实 project root。

验收：

- workspace 不存在时返回明确错误。
- invalid name 不触碰 ADP home。

### `adp workspace rename <old-name> <new-name>`

职责：

- 重命名 ADP workspace 目录。
- 更新 `workspace.yaml` 中的 `workspace.name`。
- 保留 project root、prompt、memory、MCP 和 profile 文件。

验收：

- old 不存在时返回明确错误。
- new 已存在时返回明确错误。
- 重命名后 `workspace show` 能读取新名称。

### `adp enter <workspace>`

职责：

- 解析 workspace。
- 构建 runtime overlay。
- 设置 `ADP_HOME`、`ADP_WORKSPACE`、`ADP_PROJECT_ROOT`、`ADP_RUNTIME_ROOT`、`ADP_SESSION_ID`。
- 在 runtime root 启动交互 shell。

限制：

- CLI 进程不能改变父 shell 的 cwd，所以 MVP 行为是启动一个子 shell。
- 需要改变父 shell cwd 时，使用 `adp shell-hook` 生成的 shell 函数。

验收：

- shell cwd 是 runtime root。
- `pwd`、env、runtime 文件可见。
- shell 退出后默认清理 runtime，`--keep-runtime` 保留。

### `adp env <workspace> [--cd]`

职责：

- 构建保留的 runtime overlay。
- 输出 POSIX shell 兼容的 `export ADP_*` 内容。
- `--cd` 时额外输出进入 runtime root 的 `cd` 命令。

验收：

- 输出顺序稳定。
- shell quote 能处理空格、单引号和特殊字符。
- runtime root 中存在 `.adp-runtime.yaml`。

### `adp shell-hook [--shell <sh|bash|zsh>] [--name <function-name>]`

职责：

- 输出一个 shell 函数。
- 函数内部调用 `adp env <workspace> --cd`。
- 在父 shell 中 `eval` 返回的 exports 和 `cd` 命令。

验收：

- 支持 `sh`、`bash`、`zsh`。
- 函数名做保守校验，避免 shell injection。
- 输出稳定，便于写入 shell 配置或被测试断言。

### `adp completion [--shell <bash|zsh>] [--command <name>]`

职责：

- 为支持的 shell 输出确定性的 completion 脚本；省略 `--shell` 时默认输出 bash completion。
- 覆盖当前 CLI 命令面，包括 workspace、events、runtime、sessions 等嵌套子命令。
- 可选 command name 支持打包后的二进制名或别名，不把 `adp` 写死为唯一命令名。

验收：

- 支持 `bash` 和 `zsh`。
- 对 shell 名称和 command name 做保守校验。
- 输出稳定，便于测试断言和写入 shell 配置。

### `adp events list [--workspace <name>] [--session <session-id>] [--type <event-type>] [--limit <n>]`

职责：

- 读取 `$ADP_HOME/logs/events.jsonl`。
- 按 workspace、session、event type 和 limit 过滤。
- 以稳定表格输出最近匹配事件。

验收：

- event log 不存在时输出空表。
- 损坏 JSON 行返回带行号的明确错误。
- `--limit` 返回最近 N 条匹配事件，并保持输出顺序为时间顺序。

### `adp sessions list [--workspace <name>] [--agent <agent>] [--limit <n>]`

职责：

- 读取 `$ADP_HOME/logs/events.jsonl`。
- 按 `session_id` 聚合本地事件。
- 支持 workspace、agent 和 limit 过滤。
- 忽略空 `session_id` 的事件。

验收：

- 先应用 workspace/agent 过滤，再按最近 session 限制数量。
- 输出保持 session start 的时间顺序，便于终端阅读。
- event log 不存在时输出空结果，不创建 runtime 状态。

### `adp sessions show <session-id>`

职责：

- 输出一个 session 的有序事件。
- 展示事件中已有的 workspace、agent、runtime path、exit code、duration 等信息。
- 只读取本地 JSONL event log，不修改 runtime 或 workspace。

验收：

- session 不存在时返回明确 not-found 错误。
- 输出事件顺序与 log 中记录顺序一致。
- 损坏 JSON 行与 `adp events list` 保持一致的错误行为。

### `adp runtime prune [--older-than <duration>] [--include-kept] [--dry-run]`

职责：

- 扫描 `$ADP_RUNTIME_DIR` 的直接子目录。
- 只把包含 `.adp-runtime.yaml` 且 `generated_by: adp` 的目录视为 ADP-owned runtime。
- 清理过期 runtime 目录。

验收：

- 默认跳过 `keep: true` 的 runtime。
- `--dry-run` 只报告候选项，不删除。
- 删除目标只能是扫描到的 runtime 子目录，不能来自 manifest 中的 project root。

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
- `adp events list` 返回最近匹配事件时，输出顺序仍保持时间顺序。
- `adp sessions list/show` 是同一本地 log 的只读视图，不能创建、修改或删除 runtime 状态。

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
| B. Workspace Registry | `adp init`、workspace add/list/show/doctor 的底层能力、schema load/save、workspace diagnostics | `internal/workspace/`, `internal/schema/` | `internal/paths` | `Registry` API、workspace config 文件、diagnostic report | temp `ADP_HOME` 单测通过 |
| C. Runtime Overlay | runtime session、symlink materialization、冲突策略、cleanup | `internal/runtime/`, `internal/overlay/` | schema + adapter `GeneratedFile` | `BuildRuntime(workspace, generatedFiles)` 返回 `RuntimeHandle` | temp project 集成测试通过 |
| D. Adapter Layer | adapter registry、Codex/Claude render、模板 | `internal/adapters/`, `templates/` | schema + adapter contract | `Adapter` 实现注册到 registry | 生成文件 golden tests 通过 |
| E. Runner/Shell/Events | 外部进程执行、交互 shell、shell completion、event JSONL、session history | `internal/runner/`, `internal/shell/`, `internal/events/`, `internal/sessions/` | paths + runtime handle | `Run(spec)`、`EnterShell(handle)`、`RenderCompletion()`、`Log(event)`、`ListSessions()` | fake binary、completion 和 event/session log 测试通过 |
| F. Docs/Examples | README、CLI reference、example workspace、双语文档配对 | `README.md`, `README.zh-CN.md`, `docs/`, `examples/` | CLI 行为草案 | 可运行示例和用户文档 | 文档命令与实现保持一致，双语检查通过 |
| G. Integration QA | 端到端测试、fake agents、CI 脚本 | `test/`, `.github/`, 允许少量 test helper | A-F 初版 | `go test ./...` + CLI e2e + file-line check | 覆盖 init/add/run/enter 基本流，且代码文件不超过 700 行 |

### 文件修改规则

- 每个子 Agent 默认只能修改自己的独占路径。
- 修改公共 contract 时必须先声明原因，并同步更新依赖方测试。
- `go.mod` 由 A 负责；其他子 Agent 如需新增依赖，先在任务说明里列出，不直接引入大依赖。
- `README.md` 由 F 负责；其他子 Agent 可在实现说明中提出文档变更点。
- 默认文档为英文，F 负责同步维护 `.zh-CN.md` 简体中文 counterpart。
- `internal/schema` 初版由 Contract Scaffold 创建，B 可以继续完善；其他子 Agent 只消费 schema。
- `internal/adapters/adapter.go` 初版由 Contract Scaffold 创建，D 可以继续完善；C/E 只消费接口。
- `LICENSE` 与 `COMMERCIAL.md` 只由项目维护者修改；子 Agent 不应自行更改授权策略。
- 任何手写代码文件超过 700 行都必须先拆分再交付。

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
- scripts/check-file-lines.sh 通过。
- 支持 ADP_HOME 指向临时目录。
- 不读取或污染真实 ~/.adp。
- 不修改真实项目目录，只写 runtime/temp 和测试目录。
- 手写项目代码文件不超过 700 行，超过前必须拆分。

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
adp workspace list
adp workspace show game-a
adp workspace doctor game-a
adp env game-a --cd
adp shell-hook --shell bash
adp completion --shell bash
adp run codex --workspace game-a -- --version
cd /srv/game-a && adp run claude -- --version
adp events list --workspace game-a
adp sessions list --workspace game-a --agent codex
adp sessions show <session-id>
adp runtime prune --older-than 24h --dry-run
adp enter game-a
adp workspace rename game-a game-renamed
adp workspace remove game-renamed
```

验收点：

- `$ADP_HOME/workspaces/game-a/workspace.yaml` 存在且 project root 正确。
- `adp workspace list` / `show` 能输出已注册 workspace。
- `adp workspace doctor` 能报告健康 workspace，并对 error 级 diagnostics 返回非零退出码。
- `adp env` 能输出 shell-safe exports，并保留 runtime manifest。
- `adp shell-hook` 能输出稳定 shell 函数。
- `adp completion` 能输出 `bash` 和 `zsh` 的稳定 completion。
- `adp events list` 能查询 run start/finish 历史。
- `adp sessions list` / `show` 能从 event log 查询 session history。
- `adp runtime prune` 只报告或删除 ADP-owned runtime 目录。
- `adp workspace rename` / `remove` 只修改 ADP workspace registry。
- `examples/basic-workspace` 保持为有效本地 workspace 示例，其中 Markdown prompt 和 memory 文件保持英文默认与简体中文 counterpart 配对。
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
- `adp shell-hook` 提供基础 parent-shell workflow。
- 后续继续补动态 completion 和更完整的 session restore。

并行开发接口漂移：

- 先做 Contract Scaffold。
- 公共接口变更集中在短窗口内处理。
- G 负责端到端测试把漂移暴露出来。

真实 HOME 污染：

- 所有测试必须通过 `ADP_HOME`。
- 业务包通过 `internal/paths` 获取路径，不直接调用 `os.UserHomeDir()`。
