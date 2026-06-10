# Session Resume Plan

English: [session-restore.md](session-restore.md)

ADP 的 session resume planning 是一个只读 CLI 辅助能力，用于还原一次本地 Agent 运行是如何启动的，以及另一次本地运行如何继续同一个 ADP work context。它帮助 operator 查看历史 session、检查 task 和 phase 状态，并判断是否要用相同或不同 Agent 启动一次新的运行。

它不是 provider 原生 Codex 或 Claude conversation resume，不是自动重放，不是托管编排，不是云同步，也不是远程 issue 跟踪。ADP 的事实源保持本地优先，并且可通过终端读取。

## Resume Plan Command

P50 提供以下只读命令面：

```bash
adp sessions resume-plan <session-id> [--workspace <name>] [--owner <owner>] [--lease <duration>] [--agent <agent>] [--format <text|json>]
```

该命令只是 inspection 和 proposal command。它读取本地 ADP session evidence，并打印一次新的 `adp run ...` invocation 方案，以及必要的 lease-aware preflight notes。它不会运行建议命令。

Options 会作为 planning inputs 解读：

- `--workspace <name>` 会覆盖用于当前 task 和 phase inspection 的 target workspace。如果它与 source session workspace 不同，输出必须说明 task 和 phase state 来自 target workspace，而 historical session 来自 source workspace。
- `--owner <owner>` 表示将继续 ADP task 的 operator 或 worker。它不会 claim、renew、release 或 complete 任务。
- `--lease <duration>` 设置建议 ownership commands 中展示的 lease duration。它不会创建或延长 lease。
- `--agent <agent>` 请求使用不同 ADP adapter 的跨工具方案，例如从之前的 Codex session 规划一次 Claude continuation。它不会转移 provider-native conversation state。
- `--format text|json` 只控制 stdout 形态。JSON 输出用于本地工具，不能成为第二份状态存储。

`adp sessions restore-plan <session-id>` 是较早的只读 rerun guidance 名称。`resume-plan` 会在同一个边界上扩展出本文描述的 cross-tool ADP work-context semantics。提到 restore planning 时仍必须保留同一个只读边界。

## Resume Boundary

ADP work-context resume 指的是用足够的 ADP-managed context 启动一次新的本地运行，以继续工作：

- 来自本地 workspace registry 的 workspace 和 project-root identity；
- 选定的 agent adapter 和 profile；
- 跟在 `--` 之后的非敏感 agent arguments；
- 原始 session 绑定 task 时的 task ID 和 task snapshot；
- operator 提供的 owner 和 lease guidance；
- 来自 ADP planning ledger 的 phase status 和 next gate hints；
- 可以展示给下一个终端 Agent 的本地 event/session evidence。

Provider-native resume 指的是重新 attach 到 Codex 或 Claude 等工具拥有的 provider-private conversation 或 session handle。ADP resume planning 不做这件事。它不会抓取 provider transcripts、恢复 native task panels、信任 provider plan panels，或在工具之间传递 provider-private session handles。

因此 cross-tool resume 是 ADP handoff，不是 provider conversation transfer。针对之前 Codex session 的 `--agent claude` plan，可以建议一条带有相同 workspace 和 task context 的新 `adp run claude ...` 命令，但新的 Claude run 会作为新的 provider conversation 启动。持久 handoff evidence 仍然保存在 ADP tasks、phases、progress reports、events 和 session history 中。

当 target agent 和 workspace 与 source session 相同时，`resume-plan` 可以复用 invocation snapshot 中非敏感的启动细节：非 default 的 `--profile`、`--keep-runtime`，以及 `--` 后置 agent arguments。当 target agent 或 workspace 发生变化时，ADP 会保留 `--keep-runtime` 这类 ADP-owned runtime options，但不会复制 provider-specific profile 或 agent arguments。输出会报告这些 omitted fields 和原因。

## 概念边界

Session history 回答“发生过什么”：

- `adp events list` 读取本地 JSONL runtime events。
- `adp sessions list` 将这些 events 归组为本地 Agent sessions。
- `adp sessions show <session-id>` 输出单个 session 及其 event timeline。

Resume plan 回答“如何启动一次新的运行来继续这个 ADP work context”：

- `adp sessions resume-plan <session-id> [--workspace <name>] [--owner <owner>] [--lease <duration>] [--agent <agent>] [--format <text|json>]` 检查一个历史 session。
- 如果 invocation 数据足够，它会打印建议的 `adp run ...` 命令。
- 当 task ownership guidance 可用时，它可以包含建议的 `adp tasks renew` 或 `adp tasks claim` preflight commands。
- 对于旧 session 或不完整 event 数据，它会报告 missing fields 和 context gaps；只有缺少必需 launch 或 task context 时，plan 才是 partial。
- 它不会执行建议命令。
- 它不会创建 runtime workspace，不会启动 Agent，不会追加新 events，不会修改 task 或 phase 状态，不会运行 Git，也不会写入真实项目根目录。

Future replay 回答“替我启动一次新运行”：

- Replay 不属于当前阶段。
- 未来的 replay 命令也只能启动一次新的本地运行，而不是恢复 provider conversation。
- 任何未来的执行型命令都必须显式触发、经过本地 gate，并且与只读 inspection 分离。

## Invocation Snapshot

在 restore-planning foundation 之后创建的 sessions，会在 `run_started` event 的 `fields.invocation` 下记录非敏感 invocation snapshot。该 snapshot 用于支持 resume planning，但不会复制私有 runtime 状态。

Snapshot 可以包含：

- `schema_version`：invocation snapshot schema 版本。
- `agent_args`：传给 provider 或本地 Agent 命令的 `--` 后置参数。
- `keep_runtime`：是否请求过 `--keep-runtime`。
- `workspace_resolution`：ADP 如何解析所选 workspace，例如显式 flag 或环境/default 解析。
- `profile_source`：Agent profile 如何被选择，例如显式 profile、default profile 或 agent default。
- `original_cwd`：执行 `adp run` 时所在目录，如果可用。
- `task_snapshot`：run start 时捕获的 task context；绑定 task 时可包含 `id`、`title`、`status`、`priority` 和 `phase`。

Snapshot 不得包含：

- 完整环境变量。
- 凭据、token、API key 或 shell secret。
- Provider conversation state 或 provider session handle。
- 完整生成的 adapter instructions。
- 项目文件内容。
- 远程 tracker、托管编排或云同步状态。

`resume-plan` 命令会把该 snapshot 与用于形成安全建议的普通 session summary 字段结合使用：workspace、agent、profile、task ID 和 runtime options。Same-tool plan 可以复用 profile 和 agent arguments。Cross-tool 或 cross-workspace plan 会省略 provider-specific profile 和 agent arguments，并输出明确 guidance，而不是静默转发给另一个工具。

JSON 输出包含 `suggested_commands`。每条建议命令都有机器可读的 `side_effect`：

- `inspect`：只读 inspection commands，例如 `adp sessions show`、`adp progress report`、`adp tasks show` 或 `adp tasks stale`。
- `task_mutation`：显式 ownership commands，例如 `adp tasks claim` 或 `adp tasks renew`；`resume-plan` 永远不会运行它们。
- `runtime_creation`：显式 launch commands，例如 `adp run ...`；`resume-plan` 永远不会运行它们。

对于通过 `adp run --take` 启动的 sessions，snapshot 可以说明启动时绑定的是哪个 ADP task，但它本身不是 lease recovery。长时间运行的 owner 使用 `adp tasks renew --workspace <workspace> <task-id> --owner <owner> --lease <duration>` 续租；中断 sessions 会通过 `adp tasks stale --workspace <workspace> [--format text|json]` 可见。

## CLI 使用

先查看 session history：

```bash
adp events list --workspace game-a --task <task-id>
adp sessions list --workspace game-a --agent codex --task <task-id>
adp sessions show <session-id>
```

然后让 ADP 输出只读 resume plan：

```bash
adp sessions resume-plan <session-id> --owner handoff-agent --lease 2h --format text
```

输出内容应由人或另一个本地工具复核后，再决定是否运行任何命令。一个 ready plan 可以包含类似命令：

```bash
adp tasks renew --workspace game-a task-20260608-0003 --owner handoff-agent --lease 2h
adp run codex --workspace game-a --profile senior-engineer --task task-20260608-0003 --keep-runtime -- --example-smoke
```

如果数据缺失，ADP 应报告 missing fields 和 reasons，而不是假装 session 可以完整重建。Invocation snapshot 加入前创建的旧 sessions 在 task 和 workspace context 可用时，仍然可以输出 ready 的 task-level plan，但输出必须显示 invocation snapshot gap。

## Cross-Tool Example

当 operator 希望下一次运行使用与原始 session 不同的 ADP adapter 时，使用 `--agent <agent>`：

```bash
adp sessions resume-plan <session-id> --agent claude --owner reviewer --lease 1h --format json
```

JSON plan 应描述 recorded session、selected target agent、suggested local launch command、missing fields、omitted invocation fields、command `side_effect` values 和 read-only warnings。如果原始 session 绑定了 task，且前一个 lease 已过期，operator 仍需要在启动下一次运行前显式执行 ADP ownership command：

```bash
adp tasks stale --workspace game-a --format json
adp tasks claim --workspace game-a task-20260608-0003 --owner reviewer --lease 1h
adp run claude --workspace game-a --task task-20260608-0003
```

只有在 ADP ownership rules 允许时，才运行 claim command。如果同一 owner 仍有有效 lease，应使用 renew 而不是 claim。如果工作未被领取，且 worker 应原子选择下一个 eligible board item，使用 `adp run <agent> --take --owner <owner> --lease <duration>`。

## Fake Or Local Agent Workflow

在不依赖外部 provider CLI 的情况下验收 resume-plan 行为时，可以使用 fake local agent：

```bash
fake_bin="$(mktemp -d)"
cat > "${fake_bin}/codex" <<'SH'
#!/usr/bin/env sh
printf 'fake codex received: %s\n' "$*"
SH
chmod +x "${fake_bin}/codex"
PATH="${fake_bin}:${PATH}"
```

创建 task，并把 runtime session 绑定到它：

```bash
TASK_ID=$(adp tasks add --workspace game-a --priority high "Exercise resume-plan guidance" | sed -n 's/^task \(task-[^ ]*\) added$/\1/p')
test -n "$TASK_ID"
adp run codex --workspace game-a --task "$TASK_ID" -- --example-smoke
```

查看本地 evidence：

```bash
adp events list --workspace game-a --task "$TASK_ID"
adp sessions list --workspace game-a --agent codex --task "$TASK_ID"
adp sessions show <session-id>
adp sessions resume-plan <session-id> --owner handoff-agent --lease 2h --format text
```

运行 `resume-plan` 只能打印 inspection output。Event count、task state、phase state、runtime directories 和真实项目根目录不应因为 resume-plan 命令本身而变化。

如果手动运行建议命令，它会启动一次新的本地 Agent run，并产生新的 session ID。它不会 attach 到之前的 provider conversation。

## Lease-Aware Handoff

`sessions resume-plan` 只是 handoff clue，不是 handoff ledger。它可以根据本地 invocation evidence 还原安全的启动形态，但不能证明前一个 worker 仍然拥有任务、不能续租、不能接管中断工作，也不能知道 provider-private state 中的任何信息。

对于 task-bound sessions，应把 resume planning 与 ADP planning inspection 配合使用：

```bash
adp tasks show --workspace <workspace> <task-id> --format json
adp tasks stale --workspace <workspace> --format json
adp phase status --workspace <workspace> --format json
```

同一 owner 继续长时间任务时，应在 lease 过期前续租：

```bash
adp tasks renew --workspace <workspace> <task-id> --owner <owner> --lease <duration>
```

worker 中断且 lease 已过期时，只能通过 `adp tasks take` 或显式 `adp tasks claim` 接管。不要把 provider conversation IDs、native task panels、plan panels、chat transcripts、process exits 或 native resume state 当作 ownership evidence。

Resume plan 中的 phase status 只是 context。它可以告诉下一个 worker 某个 phase 是否 open、accepted、被 gates 阻塞，或正在等待 evidence，但它不能 accept phase、记录 commit evidence、记录 push evidence、启动后续 phase，或运行 Git。只有在必需验证通过后，才能执行正常 phase commands。

Plan-mode compatibility 遵循同一边界。Provider 原生 plan panel 可以镜像建议的重启方案或下一步 checklist，但结构化 plan 变更只有通过 `adp plan preview` 和明确批准后的 `adp plan apply` 才会持久化。Resume planning 不能自动编辑文件、complete tasks、accept phases、commit、push、运行 Git、apply plans、启动 runtimes，或把 runtime/planning 文件写入真实 project root。

## 操作规则

- 将 resume-plan output 视为指导，而不是自动修复或 resume 动作。
- 运行 JSON 输出中的任何 suggested command 前，先检查 `suggested_commands[*].side_effect`。
- 如果 session 需要纳入项目规划追踪，使用 `adp run <agent> --task <task-id>` 绑定工作。
- worker 需要在启动时原子领取看板任务时，优先使用 `adp run <agent> --take --owner <owner> --lease <duration>`。
- 长时间 ownership 使用 `adp tasks renew` 续租；意外中断后使用 `adp tasks stale` 查看 lease 已过期的 in-progress claims。
- 只通过 `adp tasks take` 或显式 `adp tasks claim` 等 ADP ownership commands 接管过期工作。
- 通过 `adp tasks update`、`adp tasks done` 或 `adp tasks block` 显式推进 task 状态。
- 通过 `adp phase start`、`adp phase accept`、`adp phase commit` 和 `adp phase push` 显式推进 phase 状态。
- 将 provider 原生 plan 和 task panels 视为 mirror 或 scratch surfaces，而不是 recovery evidence。
- 只有当 operator 明确需要 provider-private conversation state 时，才使用 provider-native Codex 或 Claude resume；不要把它当作 ADP ownership、lease、task、phase、commit 或 push evidence。
- 将 resume-plan checks 与 `adp events list`、`adp sessions list`、`adp sessions show`、`adp tasks show`、`adp tasks stale` 和 `adp phase status` 配合使用，保留本地 acceptance evidence。
- 不要把 resume-plan 描述为云同步、远程 issue 跟踪、托管编排、provider-private state scraping、automatic task completion、automatic phase acceptance 或 provider-native resume。
