# 任务管理

English: [task-management.md](task-management.md)

ADP 的 Task and Progress Manager 是 workspace-scoped 规划和执行状态的本地事实源。它把项目规划保存在真实项目根目录之外，并让终端用户和 Agent 在选择下一项工作前读取同一份任务列表。

这一层刻意不做 Web dashboard、SaaS tracker 或 issue hosting 替代品。它是面向 Agent 工作的 terminal-first、local-first 状态管理器。

## 已实现范围

第一段 task-management 能力提供：

- `adp tasks add`
- `adp tasks list`
- `adp tasks next`
- `adp tasks take`
- `adp tasks renew`
- `adp tasks stale`
- `adp tasks show`
- `adp tasks update`
- `adp tasks claim`
- `adp tasks release`
- `adp tasks done`
- `adp tasks block`
- `adp phase add`
- `adp phase list`
- `adp phase show`
- `adp phase status`
- `adp phase start`
- `adp phase accept`
- `adp phase commit`
- `adp phase push`
- `adp plan preview [--workspace <name>] --file <path|-> [--format text|json]`
- `adp plan apply [--workspace <name>] --file <path|-> [--format text|json]`
- `adp plan doctor [--workspace <name>] [--format text|json]`
- `adp progress`
- `adp progress report [--workspace <name>] [--language <en|zh-CN>] [--format markdown|json]`
- 面向 task、phase 和 progress inspection 的只读 `--format json` 输出。
- `adp run <agent> --task <task-id>` runtime binding。
- `adp run <agent> --take --owner <owner> [--lease <duration>]` 启动时原子领取任务并绑定 runtime。
- `$ADP_HOME/workspaces/<workspace>/planning/` 下的 workspace-local planning 文件。
- 用于记录任务创建和状态变化的 JSONL progress events。
- 通过 task ID 关联的 runtime event 和 session evidence。
- 针对 task 和 phase 修改命令的 planning-file lock。
- claim conflict 处理、可选 claim lease，以及带 owner 校验的 release。
- 针对验收、commit evidence、push evidence 和下一阶段启动的 phase lifecycle guards。

smoke 脚本只能断言当前工作树中实际存在的 task-management 命令，不能为计划中的命令添加 placeholder checks。

## ADP 规划契约

ADP 是 workspace 的权威本地 planning 和 progress ledger。终端用户、Codex、Claude 以及后续 Agent 工具，都应该通过 ADP 命令创建、领取、更新、阻塞、释放和完成持久任务，而不是把任务状态的唯一副本保存在 provider 原生 todo list 或聊天记录里。

Agent 的持久任务命令流如下；原子领取优先使用 `tasks take`：

```bash
adp tasks next --workspace <workspace> --format json
adp tasks add --workspace <workspace> --phase <phase-id> --priority <priority> "<title>"
adp tasks take --workspace <workspace> --owner <owner> --lease <duration> --format json
adp run <agent> --workspace <workspace> --take --owner <owner> --lease <duration> -- <agent-args>
adp tasks claim --workspace <workspace> <task-id> --owner <owner> --lease <duration>
adp tasks renew --workspace <workspace> <task-id> --owner <owner> --lease <duration>
adp tasks stale --workspace <workspace> --format json
adp tasks update --workspace <workspace> <task-id> --status in_progress
adp tasks block --workspace <workspace> <task-id> --reason "<reason>"
adp tasks release --workspace <workspace> <task-id> --owner <owner>
adp tasks done --workspace <workspace> <task-id>
adp progress report --workspace <workspace> --format json
```

`adp tasks next` 是只读 selection snapshot。它可以帮助 Agent 选择工作，但不会领取任务。持久 owner 和恢复证据从 `adp tasks claim` 开始，包括 owner name 和可选 lease。

多 Agent worker 应优先使用 `adp tasks take`，而不是拆成 `tasks next` 加 `tasks claim` 两步。`tasks take` 会在一次受 planning lock 保护的 mutation 中完成选择和归属记录，避免两个 Agent 并发拿到同一个任务。

当 worker 通过 ADP 启动时，优先使用 `adp run <agent> --take --owner <owner> [--lease 4h]`，让任务选择、ownership 记录、runtime context 生成和 Agent 启动位于同一个命令边界。`--take` 与 `--task <task-id>` 互斥：只有 operator 已明确分配具体任务时才使用 `--task`。

阶段推进遵循同一规则。Agent 可以用 `adp phase status --workspace <workspace> --format json` 检查 gate，但 acceptance、commit evidence 和 push evidence 仍然必须分别通过 `adp phase accept`、`adp phase commit` 和 `adp phase push` 显式记录。ADP 不能根据 provider session exit code 推断 task completion、phase acceptance 或 Git state。

## 工具任务框桥接

如果 Agent 工具提供原生 task 或 todo panel，Agent 应该在开始工作时把当前 ADP task 镜像到该 panel。镜像项可以展示 ADP task ID、title、status、phase、owner 或 lease，以及短小的本地 subtasks，让工具任务框与正在执行的工作保持一致。对于 `adp run --take`，启动时被选择并领取的 task 就是需要镜像的 active task。

Provider 原生 task box 只作为视觉和 scratch surface。它不能成为持久任务存储，ADP 也不能把 provider-private todo state 读取或同步为权威 planning data。任何持久变更仍然必须通过上面的 task 和 phase 命令写回 ADP。

当前集成边界是 instruction-level mirroring；只有当 provider 暴露稳定的本地 API，且不会让 provider-private state 成为权威状态时，ADP 才能进一步调用该 API。没有 native panel 时，Agent 继续使用 ADP ledger 和终端命令即可。

## 工具 Plan Mode 桥接

Provider 原生 plan mode 或 plan panel 是 proposal 和 scratch views。它们可以帮助 Agent 为 operator 组织候选计划，但 ADP 仍然是权威本地 planning 和 progress ledger。

工具处于 plan mode 时，Agent 应避免 implementation edits、task completion、phase acceptance、commit、push 或其他执行 side effects；除非用户明确批准离开规划并开始执行。Planning proposals 应先通过只读 intake path 检查：

```bash
adp plan preview --workspace <workspace> --file - --format json
```

只有在用户或 operator 明确批准后，才能把 plan 写入 ADP：

```bash
adp plan apply --workspace <workspace> --file - --format json
```

Plan apply 之后，持久 task state 仍然遵循已有 task 和 phase commands。Provider 原生 plan items 可以为了可读性镜像 ADP plan，但它们不是第二份 ledger，也不能作为 recovery 或 progress evidence。

如果 `adp run --take` 启动的 provider 处于 plan mode，task ownership 已经记录在 ADP 中，但原生 plan 仍然只是 proposal surface。执行获得批准后，worker 仍需通过显式 ADP 命令更新 status 和 phase evidence。

## Command Surface Metadata 与 Drift Checks

P16 是 command-surface hardening 切片，不是新的 task-management 功能。它会增加一份本地 command metadata contract，让 usage text、dispatch wiring 和 bash/zsh completion 都能对照同一份命令清单检查，而不是各自漂移。

这份 metadata contract 是现有手写 CLI 的本地维护证据。它不是新的 CLI 框架、Web dashboard、SaaS tracker、hosted orchestration service、hosted tracker sync、automatic Git workflow、automatic task closure path、provider-native resume mechanism 或 project-root planning export。

后续命令变更应该在同一阶段同步更新 metadata contract、usage text、dispatch path、bash/zsh completion、聚焦测试，以及 smoke 或文档验收。只读 command metadata 不能 claim 任务、关闭任务、验收 phase、追加 planning events、运行 Git、启动 Agent、修改 runtime state、写入 project-root 文件，或成为第二份 planning store。

## Progress Report 范围

P6 增加了 Markdown report，P8 在同一个只读命令上扩展 JSON handoff snapshot：

- `adp progress report [--workspace <name>] [--language <en|zh-CN>] [--format markdown|json]`

该命令会向 stdout 打印本地 planning/execution handoff snapshot。它从 `$ADP_HOME` 读取 workspace planning 数据，默认输出英文 Markdown；只有显式传入 `--language zh-CN` 时才输出简体中文 Markdown。`--language` 只作用于 Markdown 输出；JSON 输出保留稳定的机器可读字段名和枚举值，供跨工具解析。

传入 `--format json` 时，该命令会输出只读 handoff snapshot，包含 workspace、task 总数、phases、task counts、tasks、按优先级排序的 next work、phase evidence，以及在本地 JSONL runtime events 和 session 数据存在时的最近 runtime session evidence。JSON snapshot 是 inspection format，不是单独的状态存储。权威状态仍然是 `$ADP_HOME` 下的本地 planning ledger，以及 `$ADP_HOME/logs/events.jsonl` 等本地 JSONL evidence。

该报告是 inspection view，不是状态流转命令。它不会追加 events、修改 task 状态、修改 phase 状态、创建 runtime 目录、启动 Agent、运行 Git、恢复 provider 原生会话，或把报告文件写入项目根目录。

## Next Work Endpoint 范围

P10 定义一个更窄的只读 endpoint，用于在不解析完整 progress report 的情况下选择下一项本地任务：

- `adp tasks next [--workspace <name>] [--limit <n>] [--format text|json]`

该命令读取 `$ADP_HOME` 下的 workspace planning ledger，选择符合条件的任务，按优先级和稳定的本地 tie-breakers 排序，并把最佳 next work candidates 输出到 stdout。它面向终端用户和本地子 Agent，用于在显式 claim 或更新任务前获取小型任务选择 snapshot。

符合条件的状态是 `ready`、`in_progress` 和 `review`。`planned`、`blocked`、`validated`、`done` 和 `canceled` 任务仍会出现在 list、show、progress 和 report 视图中，但不会被 `adp tasks next` 选中。

Text 输出是默认格式，并针对终端快速扫描优化。`--limit <n>` 用于限制 candidate list；默认值是 5，`--limit 0` 表示不截断。JSON 输出使用稳定的机器可读字段和枚举值，方便跨工具调用方无需抓取文本即可选择任务。

JSON contract 包含：

- `workspace`：workspace 名称。
- `planning_source`：用于生成 snapshot 的本地 planning 文件路径。
- `generated_at`：UTC snapshot 时间戳。
- `total`：workspace ledger 中的任务总数。
- `eligible_count`：状态过滤和 limit 处理后的 candidate 数量。
- `counts`：完整 ledger 中按状态统计的任务数量。
- `limit`：请求的 limit。
- `candidates`：按选择顺序排列的 candidate task list。
- `next`：至少有一个 eligible task 时的第一个 candidate；没有 eligible work 时省略。

每个 candidate 使用与 `adp tasks list --format json` 相同的 task object 形态，包括 task ID、标题、状态、优先级、phase ID、存在时的 owner 或 lease 信息，以及相关时的 blocker 摘要。

该 endpoint 保持只读。它不能领取任务、更新状态、修改 owner 或 lease、清理 blocker、修改 phase、追加 planning 或 runtime events、创建 runtime 目录、启动 Agent、运行 Git、推断验收、关闭任务、把输出文件写入项目根目录、与 hosted service 同步，或把 JSON 输出维护为第二份 planning store。

## 原子任务领取范围

P43 定义从本地看板一次领取一个任务的 mutating endpoint：

```bash
adp tasks take [--workspace <name>] --owner <owner> [--lease <duration>] [--format <text|json>]
```

该命令用于多 Agent 协调。它会获取 workspace planning lock，使用与 `adp tasks next` 相同的稳定本地排序原则选择优先级最高的可领取任务，记录 owner 和可选 lease，把任务推进到 `in_progress`，写入本地 planning ledger，并返回被领取的任务。

默认可领取任务包括：

- 没有 active owner 的 `ready` 任务，包括 previous lease 已过期的任务。
- 带有 owner 且 previous lease 已过期的 `in_progress` 任务。

默认情况下，`planned`、`blocked`、`review`、`validated`、`done` 和 `canceled` 任务不会被领取。`review` 仍可出现在只读 `tasks next` snapshot 中，但 atomic pickup 不会分配 review work，除非后续明确增加对应选项。

输出应包含与 task inspection commands 相同的 task object 形态，包括 task ID、title、status、priority、phase ID、owner、claimed timestamp，以及存在时的 lease expiration。JSON 输出只是本地工具的命令结果，不能成为第二份 planning store。

`tasks take` 只做 task ownership mutation。它不能启动 runtime、运行 Git、验收 phase、记录 commit 或 push evidence、把 task 标记为 done、关闭 phase、根据 agent exit code 推断完成、同步 hosted tracker，或把 planning 文件写入真实 project root。Runtime launch 仍然是独立的 `adp run ...` 决策，task completion 仍然必须通过 `adp tasks done` 或其他显式 status 命令完成。

## Run Take 桥接范围

P44 把原子任务领取连接到 runtime 启动：

```bash
adp run <agent> [--workspace <name>] --take --owner <owner> [--lease <duration>] [--profile <profile>] [--keep-runtime] -- <agent-args>
```

`adp run --take` 用于启动时的多 Agent ownership。ADP 会先在 workspace planning lock 下领取优先级最高的可领取任务，记录 owner 和可选 lease，把任务推进到 `in_progress`，然后再构建 runtime root 或启动外部 agent command。被领取的 task 会像 operator 使用 `--task <task-id>` 启动时一样绑定到 runtime。

`--take` 与 `--task <task-id>` 互斥。已经明确分配任务时使用 `--task`；需要 worker 从 ADP 看板原子领取下一个 eligible item 时使用 `--take`。如果没有可领取任务，ADP 应在启动 provider command 前失败。

该 bridge 不会完成 task、accept phase、记录 commit 或 push evidence、运行 Git、根据 provider exit code 推断成功、同步 provider 原生 task panel，或把 planning 文件写入真实 project root。Provider 原生 task box 和 plan panel 可以为了本地可见性镜像 active task，但 ADP 仍然是权威 ledger。

## 任务 Lease 维护范围

P45 定义长时间运行和中断 worker 的显式 lease 维护能力：

```bash
adp tasks renew --workspace <workspace> <task-id> --owner <owner> --lease <duration>
adp tasks stale --workspace <workspace> [--format text|json]
```

`tasks renew` 会在 workspace planning lock 下延长当前 owner 的 task lease。调用方必须提供当前 owner 和新的 lease duration，ADP 会更新本地 planning ledger，而不是依赖任何 provider 原生任务状态。

`tasks stale` 保持只读。它列出 lease 已过期的 `in_progress` tasks，让 operator 和 worker 能看到中断或遗留工作，而不修改 ownership。它不能追加 events、修改 task status、续租、释放任务、启动 Agent、运行 Git、验收 phase，或向 project root 写入文件。

通过 `adp run --take` 启动的 worker 应在长时间执行期间、lease 过期前主动续租。如果 session 意外中断，过期 claim 会通过 `tasks stale` 可见；lease 过期后，其他 worker 可以按 ADP ownership rules 使用 `tasks take` 或显式 `tasks claim` 接管任务。这些命令都不会自动把任务标记为 done、accept phase、记录 commit 或 push evidence、执行 Git、抓取 provider-private task box，或把 provider plan panel 当作恢复状态。

## Phase Gate Status 范围

P24 增加一个只读的 phase gate snapshot，面向本地工具和终端 Agent：

- `adp phase status [--workspace <name>] [--format text|json]`

该命令读取本地 phase ledger，找出最早尚未满足 gate 的 phase，报告当前 open phase、下一个 planned phase、是否可以启动下一阶段，以及下一步必需动作。动作值包括 `record_acceptance`、`record_commit`、`record_push`、`start_next_phase` 和 `plan_next_phase`。

该 snapshot 只用于 inspection。它不会启动 phase、验收 phase、记录 commit 或 push evidence、修改 tasks、追加 events、运行 Git、push、启动 Agent、同步 hosted tracker，或把 planning 文件写入 project root。

新 phase records 和 plan import 会带有显式本地顺序。没有 order 的既有 phase records 会继续使用稳定的 created-time 和 ID 顺序以保持兼容。后续 phase 不能跳过更早的 planned 或未完成 phase；只有在本地 phase ledger 中记录了 successful pushed evidence 的 phase 才算满足 gate。

## Planning Intake 范围

P14 增加了本地 planning intake 命令，用于接收确定性的跨工具 phase/task 输入：

```bash
adp plan preview --workspace <name> --file <path|-> [--format text|json]
adp plan apply --workspace <name> --file <path|-> [--format text|json]
```

第一版只接受结构化 YAML 或 JSON。ADP 内部不做自由文本自然语言拆任务。

`preview` 保持只读：它只解析和校验输入，然后打印将要创建的 phase 和 task 变更。它不能写入 planning 文件、追加 events、创建 runtime 目录、启动 Agent、运行 Git、修改 task 归属、关闭任务、验收 phase、同步 hosted tracker，或把 report/planning export 写入 project root。

`apply` 必须由用户显式执行。它只会在校验通过后写入 `$ADP_HOME/workspaces/<workspace>/planning` 下的本地 planning ledger；JSON 输出仍然只是 inspection format，不能成为第二份 planning store。该能力仍然保持 terminal-first 和 local-first；它不是 Web UI、dashboard、SaaS tracker、cloud sync layer、hosted orchestration service、hosted tracker sync、automatic Git workflow、automatic claim/done/phase acceptance flow、provider-native resume flow 或 project-root report/planning export path。

## Planning Ledger Diagnostics 范围

P26 增加一个只读的本地 planning ledger doctor：

```bash
adp plan doctor [--workspace <name>] [--format text|json]
```

该命令检查 `$ADP_HOME/workspaces/<workspace>/planning/`，并报告 task、phase、progress log、lock 和 phase gate 的一致性 diagnostics。它把 `tasks.yaml` 和 `phases.yaml` 视为当前状态 snapshot，把 `progress.jsonl` 视为 append-only audit evidence，而不是用于 replay 并重建状态的事实源。

Text 输出会打印适合终端阅读的摘要，包括 workspace、planning directory、status、task count、phase count、progress event count、error 和 warning 数量、phase gate action，以及 diagnostics。JSON 输出是单个 inspection object，包含同一组计数、本地文件路径、phase gate snapshot、`has_errors` 和 `diagnostics`。

Diagnostic level 包括 `info`、`warning` 和 `error`。健康 ledger 和仅 warning diagnostics 返回退出码 `0`；存在 error-level diagnostics 时会先打印报告，然后返回退出码 `2`。CLI 用法错误或 workspace 解析失败仍然按普通命令失败处理。

doctor 保持只读。它不会修复文件、删除 lock、创建缺失的 planning 文件、追加 progress events、claim 或关闭 task、修改 phase、推断 acceptance、记录 commit 或 push evidence、运行 Git、push、启动 Agent、创建 runtime 目录、同步 hosted tracker、写入 project-root report，或把 JSON 输出维护为第二份 planning store。

## 存储

任务状态保存在 ADP workspace 目录下：

```txt
$ADP_HOME/workspaces/<workspace>/
└── planning/
    ├── tasks.yaml
    ├── phases.yaml
    ├── .lock
    └── progress.jsonl
```

ADP 不会把这些文件写入真实项目根目录。Planning 和 report 输出应保留在 stdout 或 `$ADP_HOME` 下；仓库文档可以手工总结已验收行为，但 ADP 不能提供 project-root planning 或 report export path。

Phase Gate MVP 继续使用这个本地 planning 目录，并用结构化 phase 和 gate 记录进行扩展。存储仍然保持 local-first，且便于终端读取：

- `tasks.yaml`：任务列表、状态、优先级、phase ID、owner、claim timestamp，以及可选 lease expiration。
- `phases.yaml`：阶段记录、阶段状态、验收记录、commit 记录、push 记录和 gate 摘要。
- 新 phase records 会包含本地 order 值，让 gate 能阻止跳过更早 planned 或 unfinished phases。
- `.lock`：短生命周期的本地 planning mutation lock。task 和 phase 写入会围绕它执行；超过 stale age 的 lock file 会被清理。
- `progress.jsonl`：用于记录 task、phase、acceptance、commit 和 push 变化的 append-only audit events。

仓库文档可以总结规划，但权威执行状态仍保存在 `$ADP_HOME` 下。正常运行时，ADP 不能在真实项目根目录创建 `planning/`、`tasks.yaml`、`phases.yaml` 或 `progress.jsonl`。

## 任务状态

任务状态值包括：

- `planned`：已知但尚未准备执行的工作。
- `ready`：当前可以领取执行的工作。
- `in_progress`：正在执行的工作。
- `blocked`：无法继续，且需要记录阻塞原因的工作。
- `review`：实现已完成，等待审查。
- `validated`：验收已通过，但阶段尚未关闭。
- `done`：工作已验收并提交或以其他方式交付。
- `canceled`：不再计划执行的工作。

## CLI

创建任务：

```bash
adp tasks add --workspace adp --priority high --phase phase-1.5 "Add task manager"
```

列出和查看任务：

```bash
adp tasks list --workspace adp
adp tasks show --workspace adp <task-id>
adp tasks next --workspace adp
adp tasks next --workspace adp --limit 0
adp tasks list --workspace adp --format json
adp tasks show --workspace adp <task-id> --format json
adp tasks next --workspace adp --format json
```

推进任务状态；并行 worker 优先使用原子领取：

```bash
adp tasks take --workspace adp --owner codex-main --lease 30m --format json
adp run codex --workspace adp --take --owner codex-main --lease 30m -- --version
adp tasks claim --workspace adp <task-id> --owner codex-main --lease 30m
adp tasks renew --workspace adp <task-id> --owner codex-main --lease 30m
adp tasks stale --workspace adp --format json
adp tasks update --workspace adp <task-id> --status in_progress
adp tasks block --workspace adp <task-id> --reason "waiting for real CLI evidence"
adp tasks release --workspace adp <task-id> --owner codex-main
adp tasks done --workspace adp <task-id>
```

记录阶段和 gate 证据：

```bash
adp phase add --workspace adp --goal "phase gate MVP" p3 "Project planning and execution progress"
adp phase status --workspace adp
adp phase start --workspace adp p3
adp phase accept --workspace adp p3 --command "scripts/check-all.sh" --result passed --notes "local gate passed"
adp phase commit --workspace adp p3 --hash <commit-hash> --message "Implement phase gate MVP"
adp phase push --workspace adp p3 --remote origin --branch main --result pushed
adp phase show --workspace adp p3
adp phase list --workspace adp --format json
adp phase show --workspace adp p3 --format json
adp phase status --workspace adp --format json
```

汇总进度：

```bash
adp progress --workspace adp
adp progress --workspace adp --format json
```

Progress report 输出：

```bash
adp progress report --workspace adp
adp progress report --workspace adp --language zh-CN
adp progress report --workspace adp --format json
```

report 命令只向 stdout 输出。默认 format 是 Markdown，并且不会创建或更新报告文件。

省略 `--workspace` 时，ADP 使用与其他 workspace-aware 命令相同的 workspace 解析模型：先看 `ADP_WORKSPACE`，再在当前目录位于已注册 project root 内时尝试匹配。

## 机器可读检查

只读的 task、phase 和 progress 视图支持 `--format json`，方便本地工具和子 Agent 解析 planning ledger，而不需要抓取终端文本：

```bash
adp tasks list --workspace adp --format json
adp tasks show --workspace adp <task-id> --format json
adp tasks next --workspace adp --format json
adp phase list --workspace adp --format json
adp phase show --workspace adp <phase-id> --format json
adp phase status --workspace adp --format json
adp progress --workspace adp --format json
adp progress report --workspace adp --format json
```

JSON 输出是 inspection format，不是单独的状态存储。权威 planning 状态仍保存在 `$ADP_HOME/workspaces/<workspace>/planning/` 下，progress evidence 仍保存在本地 `progress.jsonl` ledger 中。runtime session evidence 在对应本地 JSONL events 存在时仍然从这些 events 派生。仓库文档可以描述计划，但不会变成执行状态的事实源。

跨工具消费者应该把 JSON 输出视为本地 snapshot，用于选择工作、展示状态，或把上下文交给另一个终端 Agent。需要改变状态时，仍然必须调用显式 mutating commands：

- 本地工具需要紧凑的优先级任务选择 snapshot 时，使用 `adp tasks next --format json`。
- 本地 worker 需要从看板原子选择并领取一个任务时，使用 `adp tasks take --owner <owner> --lease <duration> --format json`。
- 使用 `adp tasks claim`、`adp tasks update`、`adp tasks done`、`adp tasks block` 或 `adp tasks release` 修改任务。
- 使用 `adp phase start`、`adp phase accept`、`adp phase commit` 和 `adp phase push` 推进阶段。
- 本地工具需要判断当前 phase 是否还需要验收、commit evidence、push evidence，或者下一 planned phase 是否能启动时，使用 `adp phase status --format json`。
- 不要从通过的命令输出自动推断验收结果，也不要在没有显式 task 或 phase 命令时自动关闭任务。
- 不要把 JSON 输出当作运行 Git、推送变更、启动下一阶段或修改项目根目录的许可。

阶段纪律不变：只有通过显式本地命令记录实现完成、验收、commit evidence 和 push evidence 后，一个阶段才算完成。

## Progress Report 输出

`adp progress report [--workspace <name>] [--language <en|zh-CN>] [--format markdown|json]` 命令会生成适合终端阅读的 Markdown 报告，或机器可读的 JSON handoff snapshot。输出会总结本地规划与执行状态，但不能成为新的事实源。

建议 Markdown report 内容：

- Workspace 名称和本地 planning source。
- Phase 摘要，包括 active、accepted、committed 和 pushed 阶段。
- 来自本地 task ledger 的优先级排序 next work。
- 可用时展示 active owners、leases、blocked tasks 和 acceptance evidence。
- 已记录在 phase ledger 中的 commit 和 push evidence。
- 当 JSONL event/session 数据存在时，展示最近的本地 runtime session evidence，包括可用的 session ID、Agent、task ID、状态、exit code、duration 和 runtime path。

语言行为：

- `--language en` 和省略 `--language` 都输出英文。
- `--language zh-CN` 输出简体中文。
- `--language` 只作用于 Markdown。JSON 输出使用稳定的机器可读字段和值。
- 其他 language 值会给出明确失败。

Format 行为：

- `--format markdown` 和省略 `--format` 都输出 Markdown。
- `--format json` 输出单个只读 handoff snapshot，供本地工具和终端 Agent 使用。
- JSON snapshot 包含 workspace、task 总数、phases、task counts、tasks、按优先级排序的 next work、phase evidence，以及在本地 JSONL event/session 数据存在时的最近 runtime session evidence。
- `next work` 数据应按优先级排序，方便另一个本地工具不抓取 Markdown 也能选择可能的后续工作。
- JSON 输出用于跨工具解析，不能被持久化或当作 ADP 的状态存储。

只读边界：

- 不更新 task status、owner、lease 或 blocker records。
- 不更新 phase status、acceptance records、commit records 或 push records。
- 不追加 planning 或 runtime events。
- 不运行 Git 命令、不创建 commit、不 push，也不推断 Git state transitions。
- 不构建 runtime、不创建 runtime directories、不启动 Agent，也不 prune runtime directories。
- 不恢复 provider 原生会话，也不根据本地 JSONL event evidence 之外的信息推断 provider session state。
- 不在真实项目根目录创建或更新 Markdown 文件。
- 不在真实项目根目录创建或更新 JSON report 文件，也不把 JSON 输出用作同步 planning ledger。

## Phase Gate Ledger

P3 的 phase gate 工作会把任务列表打磨为 phase-aware 的执行台账，供多个终端 Agent 共享使用，但不会增加 Web dashboard、SaaS tracker、cloud sync、hosted orchestration 或远程 issue service。

该 ledger 会把以下记录显式化：

- Phase records：ID、标题、本地 order、状态、目标或 goal、验收命令列表、commit 证据、push 证据和最近一次 gate 结果。
- Task claim records：task ID、owner、领取时间、可选 lease expiration、release evidence，以及当前 ownership 状态。
- Acceptance records：phase ID、命令列表、结果、时间戳，以及简短证据文本。
- Gate records：phase ID、gate 状态、必跑检查，以及有提供时的 operator 或 Agent notes。
- Commit records：phase ID、commit hash、branch、摘要、时间戳，以及该 commit 是否只包含已验收阶段。
- Push records：phase ID、remote、branch、时间戳和 push 结果。Commit hash 证据在记录 push evidence 前单独保存在 phase record 上。

实现应保持克制，优先提供可靠的本地证据，而不是追求宽泛的项目管理功能。

## 任务领取与归属

任务 owner 是本地多 Agent 协作的协调线索，不是授权系统。

当前领取规则：

- ready 状态任务一次只能被一个 owner 领取。
- owner 可以是人名、Agent 名称或稳定的本地 Agent 标识。
- 并行 worker 应优先使用 `tasks take`，因为它会在同一把 planning lock 下完成选择和领取。
- 领取任务必须在开始实现前记录 ownership。
- `--lease <duration>` 会记录可选 lease expiration。lease 过期后，另一个 owner 可以接管该任务。
- `tasks renew --owner <owner> --lease <duration>` 会在 planning lock 下为当前 owner 延长 lease。
- `tasks stale` 会列出 lease 已过期的 `in_progress` tasks，但不修改 ledger。
- 不带 `--lease` 的 claim 不会自动过期，直到它被 release 或被同一 owner 重新 claim。
- 同一 owner 重新 claim 会刷新 claim timestamp 和 lease。
- 现有 owner 的 lease 仍然有效时，其他 owner 不能 claim 该任务。
- `tasks release --owner <owner>` 会在清空 ownership 前校验 owner。不传 `--owner` 时执行无 owner 限制的手工恢复 release。
- worker 完成、阻塞或被重新分配时，应释放任务并清空 ownership。
- `done` 和 `canceled` 状态任务不能被 claim。
- 已领取任务仍必须遵守分配给它的文件边界和阶段范围。
- 子 Agent 不提交、不推送、不修改阶段门禁，也不启动活跃阶段之外的工作。

claim 和 release 行为应追加 progress events，方便另一个终端或 Agent 重建谁在什么时间处理了什么任务。

## 验收、提交与推送记录

阶段完成不能只靠“代码看起来做完了”。一个阶段切片只有在实现、验收、提交和推送都成功后才算完成。

phase ledger 会记录：

- 该阶段实际执行的验收命令。
- 验收通过或失败结果。
- 被跳过的检查及其跳过原因。
- 已验收的 commit hash。
- push 使用的 remote 和 branch。
- push remote、branch 和结果；已验收 commit hash 保存在 commit record 中。

这些记录是本地执行证据。它们应该支持工具和 Agent 之间的 handoff，但不应要求 hosted service。

本地会强制 lifecycle guards：

- `phase start` 按 phase order 执行。任何更早 phase 尚未记录 successful pushed evidence 时，不能启动后续 phase；这包括更早的 `planned`、`active`、`accepted` 或 `committed` phase。
- `phase accept --result passed` 会把 phase 推进到 `accepted`。
- `phase accept --result failed` 会记录 evidence，并让 phase 保持或回到 `active`。
- `phase commit` 要求 phase 已处于 `accepted`，且 `acceptance.result == passed`。
- `phase push` 要求已经记录 commit evidence，且 commit hash 非空。
- `phase push --result failed` 会记录 evidence，但不会把 phase 推进到超过此前 committed 状态的位置，也不能覆盖已经记录的 successful push evidence。
- `phase status` 会在不改变 ledger 的前提下总结当前 gate。
- phase 只有到达 `pushed` 才算完成。

## Runtime Binding

把任务绑定到 Agent runtime session：

```bash
adp run codex --workspace adp --task <task-id> -- --version
adp run codex --workspace adp --take --owner codex-main --lease 4h -- --version
```

task ID 会在选定 workspace 内解析。如果该 task 不存在于这个 workspace，ADP 会在构建 runtime 或启动 agent process 之前失败。使用 `--take` 时，ADP 会在 runtime 创建前原子选择并领取下一个 eligible task；`--take` 与 `--task` 不能同时使用。

绑定 task 后，ADP 会把 task context 注入：

- `ADP_TASK_ID`、`ADP_TASK_TITLE`、`ADP_TASK_STATUS`、`ADP_TASK_PRIORITY`、`ADP_TASK_PHASE` 等 runtime 环境变量。
- runtime manifest `.adp-runtime.yaml`。
- `AGENTS.md` 和 `CLAUDE.md` 等生成的 adapter instructions。
- `.codex/config.toml` 和 `.claude/settings.json` 等生成的 adapter metadata。
- `run_started` 和 `run_finished` events。
- session summary 和 session detail 输出。

可以用以下命令查看 task-bound history：

```bash
adp events list --workspace adp --task <task-id>
adp sessions list --workspace adp --task <task-id>
adp sessions show <session-id>
```

`adp run <agent> --task <task-id>` 和 `adp run <agent> --take --owner <owner>` 都不会自动完成工作。`--take` 会记录启动时 ownership 并把任务推进到 `in_progress`，但后续状态变化仍然通过 `adp tasks update`、`adp tasks done` 和 `adp tasks block` 显式完成。

## 阶段纪律

任务管理用于支撑按阶段交付：

- 执行前先按优先级排列 planned work。
- 一次只完成一个阶段切片。本地 phase ledger 会在启动下一阶段前强制只有一个 open phase。
- 针对该阶段运行对应 runtime smoke 和完整仓库门禁。
- 验收通过后先 commit 和 push，再开始下一阶段。
- 不因为工作区已经打开就把下一阶段混进同一个提交。
- 当前阶段完成验收、提交和推送前，不允许子 Agent 启动下一阶段。
- 如果验收失败，阶段保持 active，并先记录失败 gate 后再重试。

这样可以让任务历史、验收证据和 Git 历史保持一致。

## 边界

当前 task manager 有意不提供：

- 自动把用户意图拆成任务。
- 自动把进度报告写入或导出到仓库文档或 project-root 文件。
- 把 JSON report 输出维护为第二份 planning store。
- 把 `adp tasks next --format json` 输出维护为第二份 planning store。
- 与 GitHub Issues、Linear、Jira、Notion 或任何 hosted service 同步。
- 自动运行 Git commit 或 Git push 命令。
- 从命令输出推断验收结果，或在没有显式 task / phase 命令时自动关闭任务。

这些内容保持在范围之外。后续切片应强化本地 ledger、inspection view、diagnostics 和 runtime binding，而不是增加 hosted sync、automatic Git、provider-native resume 或 project-root export。
