# 任务管理

English: [task-management.md](task-management.md)

ADP 的 Task and Progress Manager 是 workspace-scoped 规划和执行状态的本地事实源。它把项目规划保存在真实项目根目录之外，并让终端用户和 Agent 在选择下一项工作前读取同一份任务列表。

这一层刻意不做 Web dashboard、SaaS tracker 或 issue hosting 替代品。它是面向 Agent 工作的 terminal-first、local-first 状态管理器。

## 当前范围

第一段 task-management 能力提供：

- `adp tasks add`
- `adp tasks list`
- `adp tasks show`
- `adp tasks update`
- `adp tasks done`
- `adp tasks block`
- `adp progress`
- `$ADP_HOME/workspaces/<workspace>/planning/` 下的 workspace-local planning 文件。
- 用于记录任务创建和状态变化的 JSONL progress events。

下一段集成会通过 `adp run --task <task-id>`、runtime 环境变量、生成的 instruction context、event log task ID 和 session evidence，把任务与 runtime session 绑定。

## 存储

任务状态保存在 ADP workspace 目录下：

```txt
$ADP_HOME/workspaces/<workspace>/
└── planning/
    ├── tasks.yaml
    └── progress.jsonl
```

ADP 默认不会把这些文件写入真实项目根目录。未来如果要把任务状态导出到仓库文档，也应该通过显式用户命令完成，而不是自动修改 project root。

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
```

推进任务状态：

```bash
adp tasks update --workspace adp <task-id> --status in_progress
adp tasks block --workspace adp <task-id> --reason "waiting for real CLI evidence"
adp tasks done --workspace adp <task-id>
```

汇总进度：

```bash
adp progress --workspace adp
```

省略 `--workspace` 时，ADP 使用与其他 workspace-aware 命令相同的 workspace 解析模型：先看 `ADP_WORKSPACE`，再在当前目录位于已注册 project root 内时尝试匹配。

## 阶段纪律

任务管理用于支撑按阶段交付：

- 执行前先按优先级排列 planned work。
- 一次只完成一个阶段切片。
- 针对该阶段运行对应 runtime smoke 和完整仓库门禁。
- 验收通过后先 commit 和 push，再开始下一阶段。
- 不因为工作区已经打开就把下一阶段混进同一个提交。

这样可以让任务历史、验收证据和 Git 历史保持一致。

## 边界

当前 task manager 尚不支持：

- 自动把用户意图拆成任务。
- 为外部 Agent 领取任务。
- 把任务绑定到 `adp run` session。
- 把任务上下文注入生成的 `AGENTS.md` 或 `CLAUDE.md`。
- 把进度报告导出到项目文档。
- 与 GitHub Issues、Linear、Jira、Notion 或任何 hosted service 同步。

这些属于后续切片。第一优先级是提供可靠的本地任务状态，让所有终端 Agent 都能读取。
