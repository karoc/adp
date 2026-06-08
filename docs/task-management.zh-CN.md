# 任务管理

English: [task-management.md](task-management.md)

ADP 的 Task and Progress Manager 是 workspace-scoped 规划和执行状态的本地事实源。它把项目规划保存在真实项目根目录之外，并让终端用户和 Agent 在选择下一项工作前读取同一份任务列表。

这一层刻意不做 Web dashboard、SaaS tracker 或 issue hosting 替代品。它是面向 Agent 工作的 terminal-first、local-first 状态管理器。

## 已实现范围

第一段 task-management 能力提供：

- `adp tasks add`
- `adp tasks list`
- `adp tasks next`
- `adp tasks show`
- `adp tasks update`
- `adp tasks claim`
- `adp tasks release`
- `adp tasks done`
- `adp tasks block`
- `adp phase add`
- `adp phase list`
- `adp phase show`
- `adp phase start`
- `adp phase accept`
- `adp phase commit`
- `adp phase push`
- `adp plan preview [--workspace <name>] --file <path|-> [--format text|json]`
- `adp plan apply [--workspace <name>] --file <path|-> [--format text|json]`
- `adp progress`
- `adp progress report [--workspace <name>] [--language <en|zh-CN>] [--format markdown|json]`
- 面向 task、phase 和 progress inspection 的只读 `--format json` 输出。
- `adp run --task <task-id>` runtime binding。
- `$ADP_HOME/workspaces/<workspace>/planning/` 下的 workspace-local planning 文件。
- 用于记录任务创建和状态变化的 JSONL progress events。
- 通过 task ID 关联的 runtime event 和 session evidence。
- 针对 task 和 phase 修改命令的 planning-file lock。
- claim conflict 处理、可选 claim lease，以及带 owner 校验的 release。
- 针对验收、commit evidence、push evidence 和下一阶段启动的 phase lifecycle guards。

smoke 脚本只能断言当前工作树中实际存在的 task-management 命令，不能为计划中的命令添加 placeholder checks。

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

## Planning Intake 范围

P14 增加了本地 planning intake 命令，用于接收确定性的跨工具 phase/task 输入：

```bash
adp plan preview --workspace <name> --file <path|-> [--format text|json]
adp plan apply --workspace <name> --file <path|-> [--format text|json]
```

第一版只接受结构化 YAML 或 JSON。ADP 内部不做自由文本自然语言拆任务。

`preview` 保持只读：它只解析和校验输入，然后打印将要创建的 phase 和 task 变更。它不能写入 planning 文件、追加 events、创建 runtime 目录、启动 Agent、运行 Git、修改 task 归属、关闭任务、验收 phase、同步 hosted tracker，或把 report/planning export 写入 project root。

`apply` 必须由用户显式执行。它只会在校验通过后写入 `$ADP_HOME/workspaces/<workspace>/planning` 下的本地 planning ledger；JSON 输出仍然只是 inspection format，不能成为第二份 planning store。该能力仍然保持 terminal-first 和 local-first；它不是 Web UI、dashboard、SaaS tracker、cloud sync layer、hosted orchestration service、hosted tracker sync、automatic Git workflow、automatic claim/done/phase acceptance flow、provider-native resume flow 或 project-root report/planning export path。

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

ADP 默认不会把这些文件写入真实项目根目录。未来如果要把任务状态导出到仓库文档，也应该通过显式用户命令完成，而不是自动修改 project root。

Phase Gate MVP 继续使用这个本地 planning 目录，并用结构化 phase 和 gate 记录进行扩展。存储仍然保持 local-first，且便于终端读取：

- `tasks.yaml`：任务列表、状态、优先级、phase ID、owner、claim timestamp，以及可选 lease expiration。
- `phases.yaml`：阶段记录、阶段状态、验收记录、commit 记录、push 记录和 gate 摘要。
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

推进任务状态：

```bash
adp tasks claim --workspace adp <task-id> --owner codex-main --lease 30m
adp tasks update --workspace adp <task-id> --status in_progress
adp tasks block --workspace adp <task-id> --reason "waiting for real CLI evidence"
adp tasks release --workspace adp <task-id> --owner codex-main
adp tasks done --workspace adp <task-id>
```

记录阶段和 gate 证据：

```bash
adp phase add --workspace adp --goal "phase gate MVP" p3 "Project planning and execution progress"
adp phase start --workspace adp p3
adp phase accept --workspace adp p3 --command "scripts/check-all.sh" --result passed --notes "local gate passed"
adp phase commit --workspace adp p3 --hash <commit-hash> --message "Implement phase gate MVP"
adp phase push --workspace adp p3 --remote origin --branch main --result pushed
adp phase show --workspace adp p3
adp phase list --workspace adp --format json
adp phase show --workspace adp p3 --format json
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
adp progress --workspace adp --format json
adp progress report --workspace adp --format json
```

JSON 输出是 inspection format，不是单独的状态存储。权威 planning 状态仍保存在 `$ADP_HOME/workspaces/<workspace>/planning/` 下，progress evidence 仍保存在本地 `progress.jsonl` ledger 中。runtime session evidence 在对应本地 JSONL events 存在时仍然从这些 events 派生。仓库文档可以描述计划，但不会变成执行状态的事实源。

跨工具消费者应该把 JSON 输出视为本地 snapshot，用于选择工作、展示状态，或把上下文交给另一个终端 Agent。需要改变状态时，仍然必须调用显式 mutating commands：

- 本地工具需要紧凑的优先级任务选择 snapshot 时，使用 `adp tasks next --format json`。
- 使用 `adp tasks claim`、`adp tasks update`、`adp tasks done`、`adp tasks block` 或 `adp tasks release` 修改任务。
- 使用 `adp phase start`、`adp phase accept`、`adp phase commit` 和 `adp phase push` 推进阶段。
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

- Phase records：ID、标题、状态、目标或 goal、验收命令列表、commit 证据、push 证据和最近一次 gate 结果。
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
- 领取任务必须在开始实现前记录 ownership。
- `--lease <duration>` 会记录可选 lease expiration。lease 过期后，另一个 owner 可以接管该任务。
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

- `phase start` 保持一次只有一个 open phase。当已有 phase 处于 `active`、`accepted` 或 `committed` 时，不能启动另一个 phase。
- `phase accept --result passed` 会把 phase 推进到 `accepted`。
- `phase accept --result failed` 会记录 evidence，并让 phase 保持或回到 `active`。
- `phase commit` 要求 phase 已处于 `accepted`，且 `acceptance.result == passed`。
- `phase push` 要求已经记录 commit evidence，且 commit hash 非空。
- `phase push --result failed` 会记录 evidence，但不会把 phase 推进到超过此前 committed 或 pushed 状态的位置。
- phase 只有到达 `pushed` 才算完成。

## Runtime Binding

把任务绑定到 Agent runtime session：

```bash
adp run codex --workspace adp --task <task-id> -- --version
```

task ID 会在选定 workspace 内解析。如果该 task 不存在于这个 workspace，ADP 会在构建 runtime 或启动 agent process 之前失败。

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

`adp run --task` 不会自动推进任务状态。状态变化仍然通过 `adp tasks update`、`adp tasks done` 和 `adp tasks block` 显式完成。

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

当前 task manager 尚不支持：

- 自动把用户意图拆成任务。
- 自动把进度报告写入或导出到仓库文档或 project-root 文件。
- 把 JSON report 输出维护为第二份 planning store。
- 把 `adp tasks next --format json` 输出维护为第二份 planning store。
- 与 GitHub Issues、Linear、Jira、Notion 或任何 hosted service 同步。
- 自动运行 Git commit 或 Git push 命令。
- 从命令输出推断验收结果，或在没有显式 task / phase 命令时自动关闭任务。

这些属于后续切片。第一优先级是提供可靠的本地任务状态，让所有终端 Agent 都能读取。
