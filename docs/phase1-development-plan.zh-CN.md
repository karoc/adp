# ADP Phase 1 Development Plan

历史来源：本地 `mvp.md` 输入。该文件被有意忽略，不是仓库维护文档。

日期：2026-06-08

English: [phase1-development-plan.md](phase1-development-plan.md)

本文档把 ADP MVP 转换为当前 Phase 1 实施计划和运行 roadmap。它记录产品边界、架构、模块所有权、多 Agent 拆分点，以及后续实现必须继续遵守的验收门禁。

## 1. MVP 结论

ADP Phase 1 的产品核心是一个 terminal-first 的 Agent Runtime Environment：

- 管理 `$ADP_HOME`，默认 `~/.adp`。
- 注册 workspace，保存真实项目根目录与 ADP runtime 配置的映射。
- 为 Agent 构建临时 runtime overlay，让 Agent 在不污染真实项目目录的前提下看到 `AGENTS.md`、`CLAUDE.md`、`.codex/`、`.claude/` 等配置文件。
- 提供 Claude Code CLI 与 Codex CLI 两个 adapter。
- 支持 `adp init`、`adp workspace add/list/show/doctor/remove/rename`、`adp doctor`、`adp env`、`adp shell-hook`、`adp completion`、`adp completion values`、`adp version`、`adp events list`、`adp sessions list/show/restore-plan`、`adp runtime prune`、`adp enter`、`adp run`，以及本地 planning 命令 `adp tasks`、`adp phase`、`adp phase status`、`adp progress` 和 `adp plan preview/apply/doctor`。
- 记录本地 JSONL event log，为后续 replay、session restore、inspection-only handoff evidence 和 terminal-based 多 Agent 协作预留数据基础。

Phase 1 明确不做：

- Web UI / Dashboard。
- 云同步 / SaaS。
- hosted tracker、hosted tracker semantics、issue-service sync 或 SaaS task management。
- project-root planning、progress 或 report export。
- automatic Git execution，包括自动 commit 或 push。
- automatic task closure、phase acceptance、commit evidence 或 push evidence 推断。
- provider-native conversation resume。
- graphical 或 hosted multi-agent orchestration。
- 复杂权限沙箱。
- Windows 完整支持。保留接口，优先实现 Linux/macOS 可用路径。
- 真正的内核级 overlayfs 强依赖。MVP 默认使用可移植 symlink materialization，后续扩展 bind mount / overlayfs backend。

## 2. 技术栈建议

主实现语言：Go。

建议依赖：

- CLI：Phase 1 继续使用 Go 标准库手写命令解析。P16 增加本地 command metadata contract，用来防止 usage text、dispatch wiring 和 bash/zsh completion 之间漂移，并且不引入新的 CLI 框架。
- YAML：`gopkg.in/yaml.v3`。
- 测试：Go 标准库 `testing`，CLI 集成测试使用临时 HOME、临时 PATH 和 fake agent binary。

依赖原则：

- 依赖少而稳定。
- adapter 不直接依赖具体 CLI 框架。
- runtime/overlay 不依赖 adapter 具体实现。
- 业务包不能读取真实 `~/.adp`，必须通过 path/env abstraction 注入，保证测试可控。
- 新增第三方依赖前必须确认许可证与 ADP 的非商业源码可用授权策略兼容。

## 2.1 License、文档与工程门禁

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
- 用于拆分规划的非阻断 pressure audit：`scripts/check-file-lines.sh --audit`，`LINE_PRESSURE_WARN_LINES` 默认 600。

## 3. 推荐目录结构

```txt
adp/
├── cmd/
│   └── adp/
│       └── main.go
├── internal/
│   ├── commandmeta/
│   ├── cli/
│   ├── paths/
│   ├── schema/
│   ├── workspace/
│   ├── runtime/
│   ├── overlay/
│   ├── adapters/
│   │   ├── adapter.go
│   │   ├── registry.go
│   │   ├── api/
│   │   ├── claude/
│   │   ├── codex/
│   │   └── shared/
│   ├── runner/
│   ├── shell/
│   ├── sessions/
│   ├── events/
│   ├── planinput/
│   └── tasks/
├── templates/
│   ├── claude/
│   └── codex/
├── test/
│   └── e2e/
├── scripts/
├── docs/
├── examples/
│   └── basic-workspace/
├── README.md
├── README.zh-CN.md
├── COMMERCIAL.md
├── COMMERCIAL.zh-CN.md
├── LICENSE
└── go.mod
```

关键包职责：

- `internal/cli`：维护手写 command dispatcher、usage text、command metadata contract 和命令聚焦测试。
- `internal/commandmeta`：定义本地 command inventory，供 usage、dispatch、completion 和 drift tests 共同使用。
- `internal/paths`：解析 `ADP_HOME`、runtime tmp dir、workspace 路径、日志路径。
- `internal/schema`：统一 YAML schema，仅定义数据结构和校验。
- `internal/workspace`：workspace registry 的创建、查询、列表、读写。
- `internal/overlay`：把 project root 和 generated files materialize 到 runtime dir。
- `internal/runtime`：编排 workspace config、adapter output、overlay lifecycle、runtime env、manifest 和 runtime pruning。
- `internal/adapters`：adapter contracts、registry、具体 Codex/Claude adapters、兼容 API aliases 和公共渲染 helper。
- `internal/runner`：执行外部命令，处理 stdin/stdout/stderr、exit code、signal。
- `internal/shell`：实现 `adp enter`、shell exports 渲染、parent-shell hook 渲染和 shell completion 渲染。
- `internal/sessions`：把本地事件聚合为 session history summary、detail 视图和只读 restore plan。
- `internal/events`：JSONL event log 的写入和查询。
- `internal/planinput`：解析和校验 `adp plan preview/apply` 的结构化本地 planning import。
- `internal/tasks`：在 `$ADP_HOME` 下保存 workspace-scoped tasks、phases、progress events、ranking、owner leases，以及 phase acceptance/commit/push evidence。
- `templates`：面向 Codex 和 Claude adapter 的默认 runtime template material。
- `test/e2e`：使用 fake agents 的端到端 CLI 测试。

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
- `ADP_RUNTIME_DIR` 覆盖默认 runtime parent 目录。

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
${ADP_RUNTIME_DIR:-/tmp/adp-runtime}/
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
- `adp run` 和 `adp enter` 结束后默认清理 runtime 目录，除非传入 `--keep-runtime`。
- 保留或过期的 runtime 目录可通过 `adp runtime prune` 检查和清理。
- runtime prune 只删除包含当前版本且结构自洽的 `.adp-runtime.yaml` 的直接子目录；manifest 必须包含 `generated_by: adp`、非空 workspace 和 session ID、绝对 `project_root`、与目录一致的 `runtime_root`，以及有效的 `created_at`。
- 默认保留 `keep: true` 的 runtime，只有传入 `--include-kept` 才会纳入清理候选。

后续 backend 预留：

- `symlink`：MVP 默认，跨 Linux/macOS。
- `bind`：Linux 可选，需要权限检查。
- `overlayfs`：Linux 可选，适合更高保真 runtime。

## 7. Adapter Contract

Adapters 把 unified schema 转换成 agent-specific runtime files 和 launch specs。

```go
type Adapter interface {
    Name() string
    Validate(context.Context, Context) error
    Render(context.Context, Context) (*RenderResult, error)
    Launch(context.Context, Context, RuntimeHandle, []string) (*LaunchSpec, error)
}
```

边界原则：

- adapter 只负责生成 files、env 和 launch specs。
- adapter 不创建 runtime dir。
- adapter 不读写 event log。
- adapter 不解析 CLI flags。
- adapter 不直接读取真实 home directory。需要的路径由 context 提供。

MVP adapter 输出：

- Codex：
  - 生成 `AGENTS.md`。
  - 生成 `.codex/config.toml`。
  - 默认命令 `codex`，支持 workspace command override。
- Claude：
  - 生成 `CLAUDE.md`。
  - 生成 `.claude/settings.json`。
  - 默认命令 `claude`，支持 workspace command override。

Codex CLI 和 Claude Code CLI 的具体配置格式可能随版本变化。冻结 adapter 假设前必须验证当前 provider CLI 行为或官方文档，并把格式细节限制在 adapter package 内。

## 8. CLI 行为与验收

### `adp init`

行为：

- 创建 `$ADP_HOME`、`workspaces/`、`logs/` 和默认 `config.yaml`。
- 幂等执行，不能破坏已有配置。

验收：

- 空的临时 `ADP_HOME` 可以初始化成功。
- 重复执行时已有配置保持不变。
- 测试使用临时 `ADP_HOME`，不使用真实用户 home。

### `adp workspace add <name> <project-root>`

行为：

- 校验 workspace name 和 project root。
- 将 project root 存为绝对路径。
- 创建默认 prompt、memory、MCP、profile 和 workspace config 文件。
- 对重复 workspace 和无效 name 返回明确错误。

验收：

- 相对 project root 会转换为绝对路径。
- 重名 workspace 失败且不修改已有 workspace。
- 无效 name 在写入 ADP home 前失败。

### `adp workspace list`

行为：

- 输出已注册 workspace 的名称、project roots 和 ADP workspace directories。

验收：

- 空 registry 仍然可读。
- 多个 workspace 以稳定顺序输出。

### `adp workspace show <name>`

行为：

- 输出单个 workspace 的 project root、workspace directory、memory status 和 MCP status。

验收：

- workspace 不存在时返回明确 not-found 错误。
- 输出字段足够稳定，便于终端用户和测试使用。

### `adp workspace doctor [name]`

行为：

- 不传 name 时检查全部已注册 workspace；传 name 时检查指定 workspace。
- 覆盖 config load/validation、project root reachability、runtime parent safety、prompt、memory、MCP、profile 文件引用、路径逃逸、agent command 默认值、inline command arguments、路径型 command wrapper readiness、unknown enabled agents 和 reserved project-root paths。
- 以稳定的终端格式输出 diagnostics。

验收：

- 健康 workspace 输出 `ok - no issues`。
- error 级 diagnostics 返回非零退出码。
- warning-only command/profile diagnostics 保持退出码为零。
- 单个坏 workspace 不阻断其它 workspace 的 diagnostics。

### `adp doctor [workspace]`

行为：

- 作为全局命令提供同一组本地 workspace diagnostics。
- 接受一个可选 workspace name；省略时检查全部已注册 workspace。

验收：

- `adp doctor game-a` 与 `adp workspace doctor game-a` 具有等价诊断语义。
- 命令不访问网络，也不要求 provider CLI 存在。

### `adp workspace remove <name>`

行为：

- 删除 ADP workspace directory，不触碰真实 project root。

验收：

- workspace 不存在时返回明确错误。
- 无效 name 不修改 ADP home 或 project root。

### `adp workspace rename <old-name> <new-name>`

行为：

- 重命名 ADP workspace directory。
- 更新 `workspace.yaml` 中的 workspace name。
- 保留 project root、prompt、memory、MCP 和 profile 文件。

验收：

- old name 不存在和 new name 已存在都会明确失败。
- `workspace show` 可以读取重命名后的 workspace。
- local completion candidates 中不再出现旧名称。

### `adp enter <workspace> [--keep-runtime]`

行为：

- 构建 runtime overlay。
- 设置 `ADP_HOME`、`ADP_WORKSPACE`、`ADP_PROJECT_ROOT`、`ADP_RUNTIME_ROOT` 和 `ADP_SESSION_ID`。
- 在 runtime root 启动子 shell。CLI 进程不能改变父 shell 的 cwd，因此该命令有意启动 child shell。
- shell 退出后默认清理 runtime，除非传入 `--keep-runtime`。

验收：

- shell cwd 是 runtime root。
- `pwd`、env values、generated files 和 project symlinks 在 child shell 中可见。
- 默认 smoke 不需要真实交互 shell，也能覆盖 runtime cleanup 和 `--keep-runtime` preservation。

### `adp env <workspace> [--cd]`

行为：

- 构建保留的 runtime overlay。
- 输出 POSIX-compatible shell exports。
- 传入 `--cd` 时额外输出进入 runtime root 的 quoted `cd` command。

验收：

- 输出顺序确定。
- shell quoting 能处理空格、单引号和特殊字符。
- runtime root 中存在 `.adp-runtime.yaml`。

### `adp shell-hook [--shell <sh|bash|zsh>] [--name <function-name>]`

行为：

- 输出一个调用 `adp env <workspace> --cd` 的 shell function。
- 让用户在 parent shell 中 evaluate exports 和 `cd`。
- 不改变 `adp enter`；`enter` 仍然启动 child shell。

验收：

- 支持 `sh`、`bash` 和 `zsh`。
- 函数名保守校验，避免 shell injection。
- 输出稳定，便于测试和 shell 配置。

### `adp completion [--shell <bash|zsh>] [--command <name>]`

行为：

- 为支持的 shell 输出确定性的 completion，省略 `--shell` 时默认 bash。
- 覆盖当前 command surface，包括 nested workspace、event、runtime、session、task、phase、progress 和 plan subcommands。
- 通过 `--command` 支持打包二进制名或 alias。
- 使用只读动态端点提供本地候选值：`adp completion values workspaces` 和 `adp completion values profiles [--workspace <name>]`。

P16 增加本地 command metadata contract。Usage text、dispatch wiring 和 bash/zsh completion 都应对照该 metadata 检查，避免命令在一处可达而在另一处缺失。该 contract 只用于本地 drift prevention；不能变成 CLI framework migration、hosted command registry、Web UI、SaaS tracker、automatic Git path、automatic task 或 phase closure path、provider-native resume path，或 project-root export mechanism。

验收：

- 支持 bash 和 zsh。
- shell name 和 command name 做保守校验。
- 动态值端点只读取本地 workspace/profile 状态；不初始化 workspace、不访问网络、不修改 planning/runtime state。

### `adp version`

行为：

- 输出本地 CLI build identity。
- 开发构建可以输出 `dev`。
- preview release binary 应通过 Go linker flags 注入 version、commit 和 build-date。

验收：

- `adp version` 和 `adp --version` 输出稳定。
- 缺少 linker flags 时仍能输出可用的 development identity。
- release packaging docs 记录 linker flag 字段。

### `adp events list [--workspace <name>] [--session <session-id>] [--task <task-id>] [--type <event-type>] [--limit <n>]`

行为：

- 读取 `$ADP_HOME/logs/events.jsonl`。
- 按 workspace、session、task、event type 和 limit 过滤 JSONL events。
- 以稳定终端表格输出最近匹配事件。
- 损坏 event log 行必须带行号报告，不能静默忽略。

验收：

- event log 不存在时输出空结果。
- `--limit` 返回最近匹配记录，同时保持输出为时间顺序。
- 命令保持只读。

### `adp sessions list [--workspace <name>] [--agent <agent>] [--task <task-id>] [--limit <n>]`

行为：

- 按 `session_id` 聚合本地 event log records。
- 支持 workspace、agent、task 和 limit filters。
- 忽略空 `session_id` 的 events。

验收：

- 先应用 filters，再执行 limit。
- 选中的 sessions 保持 session-start 时间顺序，便于终端阅读。
- event log 不存在时输出空结果且不创建 runtime state。

### `adp sessions show <session-id>`

行为：

- 输出一个 session 的有序 events。
- 数据来源是本地 JSONL event log。

验收：

- session 不存在时返回明确 not-found 错误。
- 命令只读，不创建、修改或删除 runtime state。

### `adp sessions restore-plan <session-id>`

行为：

- 当历史 session 中有足够的非敏感 invocation snapshot 数据时，打印只读建议 `adp run ...` command。
- 历史数据不完整时输出 partial plan、missing fields 和 reasons。

验收：

- 命令不执行建议、不启动 Agent、不创建 runtime state、不追加 events、不修改 task state、不写入 project root，也不恢复 provider-native conversations。
- 输出事件顺序遵循 source log 顺序。

### `adp progress report [--workspace <name>] [--language <en|zh-CN>] [--format markdown|json]`

行为：

- 向 stdout 打印本地 planning/execution handoff snapshot。
- 读取 `$ADP_HOME` 下的本地 planning ledger。
- 默认输出英文 Markdown。
- 只有传入 `--language zh-CN` 时输出简体中文 Markdown。
- `--language` 只作用于 Markdown。

传入 `--format json` 时，命令输出机器可读、只读的 handoff snapshot，包含 workspace、task 总数、phases、task counts、tasks、按优先级排序的 next work、phase evidence，以及本地 JSONL runtime events 和 session 数据存在时的 recent runtime session evidence。JSON 用于本地跨工具解析，不能成为第二份状态存储。

当本地 JSONL runtime events 和 session 数据存在时，Markdown 和 JSON report 都会包含从 `$ADP_HOME/logs/events.jsonl` 派生的 recent runtime session evidence。该 evidence 只用于 inspection 和 handoff。

验收：

- 输出到 stdout；命令永远不自动创建或更新 report files。
- 不能追加 events、修改 task state、修改 phase state、创建 runtime directories、启动 agents、运行 Git、push、推断 acceptance、关闭 tasks、恢复 provider-native conversations，或把 report files 写入真实 project root。
- task state、phase state、event log、runtime directories、Git state 和真实 project root 保持不变。

### `adp progress [--workspace <name>] [--format text|json]`

行为：

- 输出当前 workspace planning progress。
- 统计 task statuses，并暴露按优先级排序的 next work。
- 支持 text 供终端扫描，支持 JSON 供本地跨工具解析。

验收：

- 命令只读。
- JSON 输出是 snapshot，不是第二份 planning store。
- 不修改 task、phase、event、runtime、Git、hosted service 或 project-root state。

### `adp tasks add|list|show|update|claim|release|done|block`

行为：

- 在 `$ADP_HOME/workspaces/<workspace>/planning` 下保存 workspace-scoped tasks。
- 支持本地 task creation、listing、showing、status updates、带可选 lease 的 owner claims、owner-checked release、done transitions 和 blocker recording。
- 对 mutating planning operations 使用本地文件锁。
- phase ledger 存在后校验 task phase IDs。

验收：

- mutations 只影响 `$ADP_HOME` 下的本地 planning ledger。
- owner conflicts 和未过期 leases 会被强制执行。
- read-only task views 提供稳定 JSON 输出给本地工具。
- task commands 不运行 Git、不启动 agents、不写入 project-root planning files、不同步 hosted trackers。

### `adp tasks next [--workspace <name>] [--limit <n>] [--format text|json]`

行为：

- 向 stdout 输出紧凑的本地 next-work snapshot。
- 读取 `$ADP_HOME` 下的 workspace planning ledger。
- 选择状态为 `ready`、`in_progress` 或 `review` 的 tasks。
- 按 priority 和稳定本地 tie-breakers 排序 candidates，便于终端用户和子 Agent 不解析完整 progress report 也能选择后续工作。

Text 是默认格式，面向终端快速扫描。`--limit <n>` 限制 candidates，默认 5，`0` 表示不截断。JSON 输出用于本地跨工具解析，包含 workspace、planning source、generated timestamp、total task count、eligible candidate count、status counts、requested limit、排序后的 `candidates`，以及存在可执行任务时的 singular `next` first-candidate value。

验收：

- 命令只读。
- 不能 claim tasks、修改 task status、改变 owners 或 leases、清理 blockers、修改 phases、追加 events、创建 runtime directories、启动 agents、运行 Git、push、推断 acceptance、关闭 tasks、恢复 provider-native conversations、写入真实 project root、同步 hosted trackers，或把 JSON 维护成第二份 planning store。

### `adp phase add|list|show|status|start|accept|commit|push`

行为：

- 在 `$ADP_HOME/workspaces/<workspace>/planning` 下保存 workspace-scoped phase records。
- 跟踪 phase status、goal、acceptance command evidence、commit evidence 和 push evidence。
- 给新建 phase 和 plan-imported phase 分配显式本地 order，防止后续 phase 跳过更早的 planned 或 unfinished phase。
- `adp phase status [--workspace <name>] [--format text|json]` 输出只读 gate snapshot，包含 open phase、下一个 planned phase、是否可以启动下一阶段，以及下一步必需动作。
- 强制 phase lifecycle 顺序：planned、active、accepted、committed、pushed。
- 强制阶段流程：acceptance 先于 commit evidence，commit 先于 push evidence，且启动后续 phase 前，每个更早 phase 都必须已有 successful pushed evidence。

验收：

- phase evidence 是本地 ledger 数据，不是 Git automation。
- `phase commit` 记录 commit hash 和 message，但不创建 commit。
- `phase push` 记录 remote、branch 和 result，但不运行 `git push`。
- 后续 phase 必须等所有更早 phase accepted、committed、successfully pushed 并完成记录后才能开始。
- `phase status` 只读，不能修改 tasks、phases、events、runtime directories、Git、hosted services 或真实 project root。

### `adp plan preview|apply [--workspace <name>] --file <path|-> [--format text|json]`

行为：

- 接收包含 phases 和 tasks 的结构化本地 YAML/JSON planning input。
- 支持普通文件和通过 `--file -` 读取 stdin。
- `preview` 只把拟导入内容输出到 stdout，不创建 planning files 或 directories。
- `apply` 必须显式执行，并把校验后的 batch 写入 `$ADP_HOME/workspaces/<workspace>/planning`。

JSON 输出只用于 inspection 和本地跨工具解析，不能成为第二份 planning store。Plan intake 不能把自由文本自然语言拆成任务、同步 hosted trackers、运行 Git、启动 agents、推断 acceptance、自动 claim 或 close tasks、把 planning files 写入真实 project root，或修改 runtime state。

验收：

- Preview 保持只读。
- Apply 只写 `$ADP_HOME` 下的本地 planning ledger。
- 失败的 apply 不留下 partial phase、task 或 progress state。
- stdin intake 与 file intake 保持相同 preview/apply mutation boundaries。

### `adp runtime prune [--older-than <duration>] [--include-kept] [--dry-run]`

行为：

- 扫描 `$ADP_RUNTIME_DIR` 下的直接子目录。
- 只有目录包含 current-version 且 self-consistent 的 `.adp-runtime.yaml`，并且 manifest 中包含 `generated_by: adp`、非空 workspace 和 session IDs、绝对 `project_root`、与目录匹配的 `runtime_root`、有效 `created_at` 时，才把它视为 prune candidate。
- 删除超过 `--older-than` 的 ADP-owned runtime directories。
- 默认跳过 `keep: true`，除非传入 `--include-kept`。
- 传入 `--dry-run` 时只报告 candidates，不删除。

验收：

- 不兼容、格式错误、外部系统生成或自相矛盾的 manifests 会被跳过。
- 删除目标只能是扫描到的 runtime child directories，不能来自 manifest project roots。
- 默认和 `--include-kept` 模式都覆盖 kept runtime 行为。

### `adp run <agent> [--workspace <name>] [--profile <profile>] [--task <task-id>] [--keep-runtime] [-- <agent-args>...]`

行为：

- 解析 workspace、渲染 adapter files、构建 runtime overlay、启动 agent、记录 start/finish events、透传 streams，并返回 agent exit code。
- 传入 `--task` 时，把 runtime sessions 绑定到本地 task state，并把 task context 注入 runtime env 和生成的 adapter instructions。

Workspace 解析顺序：

- 显式 `--workspace`。
- `ADP_WORKSPACE`。
- 将当前目录匹配到已注册 project root；嵌套 workspaces 时选择最长匹配的 project root。

验收：

- fake Codex 和 Claude binaries 会验证 cwd、env、generated files、project symlinks、args、exit code、events、sessions、task binding 和 cleanup。
- Agent cwd 是 runtime root。
- 真实 project root 不被修改。

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
- `adp sessions list/show/restore-plan` 是同一本地 log 的只读视图，不能创建、修改或删除 runtime 状态。

## 10. 并行开发边界

只有在 ownership boundaries 明确，且 phase gate 仍然能作为一个 integrated slice 关闭时，才允许并行工作。

当前并行切片：

- CLI command surface：`cmd/`、`internal/cli/`、`internal/commandmeta/` 和 command-focused tests。
- Workspace registry and diagnostics：`internal/workspace/`、`internal/schema/` 和相关 CLI wiring。
- Runtime and overlay：`internal/runtime/`、`internal/overlay/`、manifest handling、lifecycle smoke 和 runtime acceptance docs。
- Adapter layer：`internal/adapters/`、`templates/`、adapter tests 和 provider compatibility docs。
- Runner、shell、events、sessions：`internal/runner/`、`internal/shell/`、`internal/events/`、`internal/sessions/`。
- Local planning manager：`internal/tasks/`、`internal/planinput/`、task/phase/progress CLI wiring 和 planning smoke。
- Docs and examples：`README*`、`AGENTS*`、`docs/` 和 `examples/`。
- Integration QA and gates：`scripts/`、`test/e2e/`、`.github/` 和 release checklist docs。

规则：

- 主线程 integration 负责 immediate blocking path。
- 子 Agent 必须拿到 disjoint write scopes；重叠文件集默认只做 read-only review，除非存在明确 handoff。
- Public contract changes 必须明确说明，并跨 dependent packages 验证。
- `LICENSE` 和 `COMMERCIAL*` 由项目维护者维护。
- 手写代码文件必须保持在 700 行以内。
- 文档必须保持英文默认文件和简体中文 counterpart 对齐。
- 每个 phase slice 都必须先 validation、acceptance、commit、push 并完成记录，然后才能开始下一阶段。

子 Agent 任务说明必须写明 objective、allowed write paths、disallowed paths、constraints、validation commands 和 expected final report。Read-only review agents 必须明确告知不得编辑文件。

## 11. Validation Gates

必跑本地检查：

```bash
scripts/check-all.sh
```

聚合门禁会运行 deterministic smoke 和仓库检查：

```bash
scripts/runtime-smoke.sh --fake
scripts/example-workspace-smoke.sh
scripts/task-manager-smoke.sh
scripts/plan-intake-smoke.sh
go test -count=1 ./...
go vet ./...
scripts/check-file-lines.sh
scripts/check-docs-bilingual.sh
git diff --check
```

端到端期望：

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
adp tasks add --workspace game-a --priority high --phase phase-1 "Bind runtime session to task"
adp tasks next --workspace game-a --limit 0 --format json
adp plan preview --workspace game-a --file plan.yaml
adp plan doctor --workspace game-a --format json
adp run codex --workspace game-a --task <task-id> -- --version
cd /srv/game-a && adp run claude -- --version
adp events list --workspace game-a --task <task-id>
adp sessions list --workspace game-a --agent codex --task <task-id>
adp sessions show <session-id>
adp sessions restore-plan <session-id>
adp runtime prune --older-than 24h --dry-run
adp enter game-a
adp workspace rename game-a game-renamed
adp workspace remove game-renamed
```

- `$ADP_HOME/workspaces/game-a/workspace.yaml` 存在且 project root 正确。
- `adp workspace list` / `show` 能输出已注册 workspace。
- `adp workspace doctor` 能报告健康 workspace，并对 error 级 diagnostics 返回非零退出码。
- `adp doctor` 能作为全局入口输出同一组 workspace diagnostics。
- `adp env` 能输出 shell-safe exports，并保留 runtime manifest。
- `adp shell-hook` 能输出稳定 shell 函数。
- `adp completion` 能输出 `bash` 和 `zsh` 的稳定 completion，`adp completion values` 能返回本地 workspace 和 profile 候选值。
- P16 command metadata drift check 能证明本地命令清单、usage text、dispatch wiring 和 bash/zsh completion 保持一致，并且不引入新的 CLI 框架。
- `adp version` 能输出 CLI build identity。
- `adp events list` 能查询 run start/finish 历史。
- `adp sessions list` / `show` / `restore-plan` 能从 event log 查询 session history 和只读 restore planning。
- `adp progress report [--workspace <name>] [--language <en|zh-CN>] [--format markdown|json]` 默认能向 stdout 打印 Markdown 规划/执行报告，传入 `--format json` 时输出只读 JSON handoff snapshot，在 JSONL event/session 数据存在时包含最近本地 runtime session evidence，并保持 planning state、Git state、runtime state、event log 和真实项目根目录不变。
- `adp tasks next [--workspace <name>] [--limit <n>] [--format text|json]` 会向 stdout 打印紧凑的优先级 next-work snapshot，为本地工具提供稳定 JSON contract，并保持 task state、phase state、Git state、runtime state、event log、hosted service state 和真实项目根目录不变。
- `adp phase status [--workspace <name>] [--format text|json]` 会向 stdout 打印紧凑的只读 phase gate snapshot，为本地工具提供稳定 JSON contract，并保持 task state、phase state、Git state、runtime state、event log、hosted service state 和真实项目根目录不变。
- `adp plan preview/apply [--workspace <name>] --file <path|-> [--format text|json]` 接收结构化本地 planning 输入；preview 保持只读，apply 只写 `$ADP_HOME` 下的本地 planning ledger，失败 apply 不留下 partial phase、task 或 progress state。
- `adp plan doctor [--workspace <name>] [--format text|json]` 会输出只读本地 planning ledger diagnostics，覆盖 task、phase、progress-log、lock 和 phase-gate invariants；存在 error-level diagnostics 时返回退出码 `2`，并保持 planning state、Git state、runtime state、event log、hosted service state 和真实项目根目录不变。
- `adp run --task <task-id>` 能把 task context 注入 runtime env、生成指令、events 和 sessions。
- `adp runtime prune` 只报告或删除当前版本且结构自洽的 ADP-owned runtime 目录。
- `adp workspace rename` / `remove` 只修改 ADP workspace registry。
- `examples/basic-workspace` 保持为有效本地 workspace 示例，其中 Markdown prompt 和 memory 文件保持英文默认与简体中文 counterpart 配对。
- runtime root 中存在 ADP 生成的 Agent 配置文件。
- 真实 `/srv/game-a` 不新增 `AGENTS.md`、`CLAUDE.md`、`.codex/`、`.claude/`、`planning/`、`tasks.yaml`、`phases.yaml`、`progress.jsonl` 或 report export 文件。
- event log 记录 run start/finish。
- fake agent e2e 能断言 cwd、env、参数透传、exit code。

Phase process gate：

1. 完成一个 planned phase slice。
2. 运行 focused validation 和 required aggregate gate。
3. 只有 validation 通过后，才能记录 phase acceptance。
4. 提交已验收的 phase。
5. 推送 commit。
6. 在本地 phase ledger 中记录 commit 和 push evidence。
7. 只有 push 成功且 phase record 更新后，才能开始下一阶段。

## 12. 下一步优先级

后续工作按“是否能增强 ADP 的 terminal-first runtime 和 workspace 管理闭环”排序，同时避免偏向 hosted project management 或 dashboard。

- P0 已完成：Task and Progress Manager MVP。把 workspace-scoped 任务状态保存在 `$ADP_HOME/workspaces/<workspace>/planning` 下，提供 `adp tasks` 和 `adp progress`，并通过 task-manager smoke 验收。
- P1 已完成：Runtime task binding。增加 `adp run --task <task-id>`，把 task context 注入 runtime env 和 adapter 生成指令，并把 task ID 关联到 events 和 sessions。
- P2 已完成：Early preview hardening。动态 workspace/profile completion、全局 `adp doctor`、version 输出、`scripts/check-all.sh` CI 和发布打包说明已纳入聚合门禁和 runtime smoke。
- P3 Phase Gate MVP 已完成：项目规划与执行进度管理现在具备 phase records、task claim 和 owner records、acceptance 或 gate records、commit records、push records，并已纳入 task-manager smoke。
- P3 planning coordination hardening 已完成：会用本地 lock 保护 planning 修改操作，task claim 会强制 owner conflict 和可选 lease，release 支持 owner 校验；phase ledger 存在后 task 会校验 phase ID；phase lifecycle guards 会强制 accept-before-commit、commit-before-push，以及 push-before-next-phase 纪律。
- P4 runtime manifest compatibility 已完成：runtime manifest 现在使用显式 manifest version，runtime smoke 会检查核心 manifest 字段，pruning 会跳过不兼容或自相矛盾的 manifest，而不是把每个 `generated_by: adp` 文件都当作可安全删除的证据。
- P4 workspace runtime-parent diagnostics 已完成：workspace 和全局 doctor 现在会拒绝位于文件系统根目录、等于 project root、位于 project root 内部或包含 project root 的 runtime parent，并对 symlink runtime parent 发出 warning。
- P4 agent command/profile diagnostics 已完成：workspace 和全局 doctor 现在会报告 project root 中的保留路径、adapter default command fallback、inline command arguments、缺失或不可执行的路径型 command wrapper、无效、缺失、重复、非文件或逃逸到 workspace 外部的非 default profile，以及 enabled 但未知的 agent 配置，并且不会运行 provider CLI。
- P4 session restore foundation 已完成：`run_started` event 会记录非敏感 invocation snapshot，`adp sessions restore-plan <session-id>` 会输出只读建议命令，runtime 和 example smoke 会验收 session events、session history、restore-plan 不追加 events，以及 examples/docs polish。
- P5 planning JSON output 已完成：task、phase 和 progress 只读视图支持 `--format json`，方便本地工具和子 Agent 获取机器可读 planning snapshot，而不需要抓取终端文本或改变状态。
- P6 progress report output 已完成：`adp progress report [--workspace <name>] [--language <en|zh-CN>]` 会向 stdout 打印只读的本地 Markdown 规划/执行报告。默认语言是英文；简体中文必须显式传入 `--language zh-CN`。task-manager smoke 会证明 report 不会修改 task、phase、Git、runtime、event log 或 project-root 状态。
- P7 progress report runtime session evidence 已完成：当本地 JSONL runtime events 和 session 数据存在时，`adp progress report [--workspace <name>] [--language <en|zh-CN>]` 会包含最近 runtime session evidence，供 inspection-only handoff 使用。它不会追加 events、修改 tasks 或 phases、创建 runtime 目录、启动 Agent、运行 Git、把报告文件写入项目根目录，或恢复 provider 原生会话。
- P8 progress report JSON handoff snapshot 已完成：`adp progress report [--workspace <name>] [--language <en|zh-CN>] [--format markdown|json]` 保持默认输出为英文 Markdown，`--language zh-CN` 只作用于 Markdown，并通过 `--format json` 输出机器可读的只读 snapshot。JSON snapshot 包含 workspace、task 总数、phases、task counts、tasks、按优先级排序的 next work、phase evidence，以及在本地 JSONL event/session 数据存在时的最近 runtime session evidence。它用于跨工具解析，不能成为单独的状态存储。
- P9 task-manager smoke modularization 已完成：oversized task-manager runtime smoke 已在触及 700 行代码文件限制前拆分为更小的公开入口、共享 shell helper library 和专用 JSON report validator。`scripts/task-manager-smoke.sh` 仍然是 workspace-local task、phase 和 progress report runtime acceptance 的公开入口，`scripts/check-all.sh` 仍然是聚合门禁。
- P9 只属于维护和 hardening。它保留了项目根目录污染防护和只读 progress report 行为的覆盖。
- P10 task next-work endpoint 已完成：`adp tasks next [--workspace <name>] [--limit <n>] [--format text|json]` 提供紧凑的只读本地 task-selection snapshot，供终端用户和子 Agent 使用。它把既有 progress-report next-work 数据收窄为专用命令，但不领取任务、不修改状态、不运行 Git、不启动 Agent、不写入 project-root 文件、不同步 hosted tracker，也不创建另一份 planning store。
- P11 task command test split 已完成：在任何手写代码文件触及 700 行限制前，把持续增长的 task command 测试覆盖拆分到更聚焦的测试文件中。P11 只属于维护和 hardening：不改变 runtime behavior，不新增 product command，不偏向 Web/SaaS/hosted orchestration，不做 Git automation，也不改变 terminal-first 的本地 planning 边界。
- P12 CLI parse helper split 已完成：在 `internal/cli/parse.go` helper 面触及 700 行代码文件限制前，将持续增长的解析辅助逻辑拆分到更聚焦的文件中。P12 只属于维护和 hardening：不改变 runtime behavior，不新增 product command，不偏向 Web/SaaS/hosted orchestration，不做 Git automation，也不改变 terminal-first 的本地 planning 边界。
- P13 CLI base test split 已完成：在 `internal/cli/cli_test.go` 基础 CLI 测试覆盖触及 700 行代码文件限制前，已将其拆分为更聚焦的测试文件。P13 只属于维护：不改变 runtime behavior，不新增 product command，不偏向 Web/SaaS/hosted orchestration，不做 Git automation，也不改变 terminal-first 的本地 planning 边界。
- P14 local planning intake preview/apply 已完成：`adp plan preview --workspace <name> --file <path|-> [--format text|json]` 和 `adp plan apply --workspace <name> --file <path|-> [--format text|json]` 接收结构化 YAML/JSON phase 和 task 输入。Preview 保持只读；apply 必须显式执行，并且只写入 `$ADP_HOME/workspaces/<workspace>/planning`；JSON 输出不能成为第二份 planning store；第一版 ADP 不做自由文本自然语言拆任务。
- P15 MVP completion audit 已完成：已审计 command/runtime coverage、release-gate 文档、maintainability pressure 和双语 roadmap 漂移。该审计把下一批本地 planning backlog 写入 planned phases P16-P23，但没有启动任何后续阶段。
- P16 command surface hardening 已完成：本地 command metadata contract 已让 usage text、dispatch wiring、bash/zsh completion、聚焦测试、smoke 或文档验收保持一致。该阶段只属于本地 CLI 维护，不引入新的 CLI 框架或 hosted command surface。
- P17 runtime smoke split 已完成：`scripts/runtime-smoke.sh` 仍然是公开入口，共享 helpers 和 fake diagnostics/session/prune slices 已拆到更聚焦的实现文件中，并保持在 700 行上限以内。该阶段只属于维护，不削弱 fake smoke 覆盖、fake subshell 隔离、真实 CLI opt-in 门禁，也继续保持 `scripts/check-all.sh` 作为聚合门禁。
- P18 CLI command test split 已完成：剩余的大型混合 CLI tests 已拆分为聚焦的 task CRUD/progress/report/phase/helper 文件，以及 shell/completion/events/sessions/runtime-prune 文件。该阶段只属于维护：不改变 runtime behavior，不改变 product command，不偏向 Web/SaaS/hosted orchestration，也不做 Git automation。
- P19 workspace lifecycle and enter acceptance 已完成：runtime smoke 现在会覆盖 workspace rename/remove，并验证 project-root sentinel 保持存在、completion 不保留 stale workspace names，以及通过 fake `SHELL` 执行受控的非交互式 `adp enter`。enter smoke 会在不启动真实交互 shell 的前提下验证 runtime env/cwd、project symlink、默认 cleanup 与 `--keep-runtime`，以及不会修改 event log。
- P20 plan stdin coverage 已完成：focused CLI tests 和 plan-intake smoke 现在覆盖通过 pipe 输入的 `adp plan preview --file -` 和 `adp plan apply --file -` YAML/JSON，保持 preview 只读、apply 必须显式执行、只写本地 planning ledger、JSON 仅用于 inspection，并且不产生 runtime/Git/event-log/project-root 副作用。
- P21 taskstore maintainability split 已完成：`internal/tasks` core responsibilities 现在已拆分为同 package 下的 store、task model、task lifecycle、task persistence、progress events、task ranking、phase model、phase lifecycle、phase persistence 和 phase helper 文件。该拆分是机械维护，不改变 public APIs、本地 ledger 语义、plan-import atomic staging、phase-gate lifecycle 行为或 runtime acceptance 覆盖，并让所有 touched code files 都明显低于 700 行上限。
- P22 Phase 1 bilingual roadmap normalization 已完成：英文默认 roadmap 与简体中文 counterpart 现在拥有相同章节树、当前 command surface、目录职责、local-first 非目标、validation gates、E2E expectations，以及 validate/accept/commit/push/record 阶段纪律。
- P23 line pressure audit tooling 已完成：`scripts/check-file-lines.sh --audit` 会报告达到或超过 `LINE_PRESSURE_WARN_LINES` 的文件，默认阈值为 600，并以退出码 0 结束，便于在触及 700 行硬限制前规划拆分阶段。必跑的 `scripts/check-file-lines.sh` 硬门禁和 `scripts/check-all.sh` pass/fail 语义保持不变。
- P24 phase gate status and ordering hardening 已完成：`adp phase status [--workspace <name>] [--format text|json]` 暴露只读本地 gate snapshot；新 phase 带有显式本地 order；phase start 会拒绝跳过更早 planned 或 unfinished phases；successful push evidence 不能被 failed push evidence 覆盖。
- P25 shell completion renderer split 已完成：bash 和 zsh completion rendering 已拆到按 shell 区分的文件中，同时 `RenderCompletion`、command-name validation、基于 metadata 的候选项、动态本地 value endpoints 以及公开 `adp completion` 行为保持不变。该阶段只属于维护性的 line-pressure 工作，不新增命令、shell 类型、Web/SaaS 行为、automatic Git execution、hosted orchestration、provider-native resume 或 project-root exports。
- P26 planning ledger doctor 已完成：`adp plan doctor [--workspace <name>] [--format text|json]` 会报告 task、phase、progress-log、lock 和 phase-gate invariants 的只读本地 diagnostics；error diagnostics 返回退出码 `2`；focused tests 和 task-manager smoke 覆盖健康与坏账本路径，并且不做 automatic repair、Git execution、runtime mutation、hosted tracker sync 或 project-root exports。
- 已完成的 Phase 1 slices 保持同一组非目标：不做 Web dashboard、SaaS tracker、cloud sync、hosted orchestration、hosted tracker sync、automatic Git execution、automatic claim/done/phase acceptance、provider-native conversation resume、远程 issue-service 集成、project-root report 或 planning export，或 hosted tracker semantics。

每个阶段切片必须先完成 validation、acceptance、commit、push 和 evidence record，然后再开始下一阶段。
