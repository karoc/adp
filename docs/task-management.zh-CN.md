# 任务管理

English: [task-management.md](task-management.md)

ADP 的 Task and Progress Manager 是 workspace-scoped 规划和执行状态的本地事实源。它把项目规划保存在真实项目根目录之外，并让终端用户和 Agent 在选择下一项工作前读取同一份任务列表。

这一层刻意不做 Web dashboard、SaaS tracker 或 issue hosting 替代品。它是面向 Agent 工作的 terminal-first、local-first 状态管理器。

## 已实现范围

第一段 task-management 能力提供：

- `adp tasks add`
- `adp tasks list`
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
- `adp progress`
- `adp progress report`
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

P6 增加一个只读报告命令：

- `adp progress report [--workspace <name>] [--language <en|zh-CN>]`

该命令会向 stdout 打印本地 Markdown 规划/执行报告。它从 `$ADP_HOME` 读取 workspace planning 数据，默认输出英文；只有显式传入 `--language zh-CN` 时才输出简体中文。

该报告是 inspection view，不是状态流转命令。它不会修改 task 状态、phase 状态、Git 状态、runtime 状态、event log 或 project-root 文件。

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
adp tasks list --workspace adp --format json
adp tasks show --workspace adp <task-id> --format json
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

Markdown 报告输出：

```bash
adp progress report --workspace adp
adp progress report --workspace adp --language zh-CN
```

report 命令只向 stdout 打印 Markdown。它是 inspection 命令，不会创建或更新报告文件。

省略 `--workspace` 时，ADP 使用与其他 workspace-aware 命令相同的 workspace 解析模型：先看 `ADP_WORKSPACE`，再在当前目录位于已注册 project root 内时尝试匹配。

## 机器可读检查

只读的 task、phase 和 progress 视图支持 `--format json`，方便本地工具和子 Agent 解析 planning ledger，而不需要抓取终端文本：

```bash
adp tasks list --workspace adp --format json
adp tasks show --workspace adp <task-id> --format json
adp phase list --workspace adp --format json
adp phase show --workspace adp <phase-id> --format json
adp progress --workspace adp --format json
```

JSON 输出是 inspection format，不是单独的状态存储。权威 planning 状态仍保存在 `$ADP_HOME/workspaces/<workspace>/planning/` 下，progress evidence 仍保存在本地 `progress.jsonl` ledger 中。仓库文档可以描述计划，但不会变成执行状态的事实源。

跨工具消费者应该把 JSON 输出视为本地 snapshot，用于选择工作、展示状态，或把上下文交给另一个终端 Agent。需要改变状态时，仍然必须调用显式 mutating commands：

- 使用 `adp tasks claim`、`adp tasks update`、`adp tasks done`、`adp tasks block` 或 `adp tasks release` 修改任务。
- 使用 `adp phase start`、`adp phase accept`、`adp phase commit` 和 `adp phase push` 推进阶段。
- 不要从通过的命令输出自动推断验收结果，也不要在没有显式 task 或 phase 命令时自动关闭任务。
- 不要把 JSON 输出当作运行 Git、推送变更、启动下一阶段或修改项目根目录的许可。

阶段纪律不变：只有通过显式本地命令记录实现完成、验收、commit evidence 和 push evidence 后，一个阶段才算完成。

## Markdown 进度报告

`adp progress report [--workspace <name>] [--language <en|zh-CN>]` 命令会生成适合终端阅读的 Markdown 报告，用于人工查看和跨工具 handoff。输出会总结本地规划与执行状态，但不能成为新的事实源。

建议报告内容：

- Workspace 名称和本地 planning source。
- Phase 摘要，包括 active、accepted、committed 和 pushed 阶段。
- 来自本地 task ledger 的优先级排序 next work。
- 可用时展示 active owners、leases、blocked tasks 和 acceptance evidence。
- 已记录在 phase ledger 中的 commit 和 push evidence。

语言行为：

- `--language en` 和省略 `--language` 都输出英文。
- `--language zh-CN` 输出简体中文。
- 其他 language 值会给出明确失败。

只读边界：

- 不更新 task status、owner、lease 或 blocker records。
- 不更新 phase status、acceptance records、commit records 或 push records。
- 不运行 Git 命令、不创建 commit、不 push，也不推断 Git state transitions。
- 不构建 runtime、不启动 Agent、不追加 runtime events，也不 prune runtime directories。
- 不在真实项目根目录创建或更新 Markdown 文件。

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
- 与 GitHub Issues、Linear、Jira、Notion 或任何 hosted service 同步。
- 自动运行 Git commit 或 Git push 命令。
- 从命令输出推断验收结果，或在没有显式 task / phase 命令时自动关闭任务。

这些属于后续切片。第一优先级是提供可靠的本地任务状态，让所有终端 Agent 都能读取。
