# Session Restore Plan

English: [session-restore.md](session-restore.md)

ADP 的 session restore planning 是一个只读 CLI 辅助能力，用于还原一次本地 Agent 运行是如何启动的。它帮助操作者查看历史 session，并判断是否要用相近命令启动一次新的运行。

它不是 provider 原生会话恢复，不是自动重放，不是托管编排，不是云同步，也不是远程 issue 跟踪。ADP 的事实源保持本地优先，并且可通过终端读取。

## 概念边界

Session history 回答“发生过什么”：

- `adp events list` 读取本地 JSONL runtime events。
- `adp sessions list` 将这些 events 归组为本地 Agent sessions。
- `adp sessions show <session-id>` 输出单个 session 及其 event timeline。

Restore plan 回答“如何启动一次相似的新运行”：

- `adp sessions restore-plan <session-id>` 检查一个历史 session。
- 如果 invocation 数据足够，它会打印建议的 `adp run ...` 命令。
- 对于旧 session 或不完整 event 数据，它可能输出 partial plan。
- 它不会执行建议命令。
- 它不会创建 runtime workspace，不会启动 Agent，不会追加新 events，不会修改 task 状态，也不会写入真实项目根目录。

Future replay 回答“替我启动一次新运行”：

- Replay 不属于当前阶段。
- 未来的 replay 命令也只能启动一次新的本地运行，而不是恢复 provider conversation。
- 任何未来的执行型命令都必须显式触发、经过本地 gate，并且与只读 inspection 分离。

## Invocation Snapshot

在该能力加入后创建的 sessions，会在 `run_started` event 的 `fields.invocation` 下记录非敏感 invocation snapshot。该 snapshot 用于支持 restore planning，但不会复制私有 runtime 状态。

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

`restore-plan` 命令会把该 snapshot 与用于形成安全建议的普通 session summary 字段结合使用：workspace、agent、profile 和 task ID。

## CLI 使用

先查看 session history：

```bash
adp events list --workspace game-a --task <task-id>
adp sessions list --workspace game-a --agent codex --task <task-id>
adp sessions show <session-id>
```

然后让 ADP 输出只读 restore plan：

```bash
adp sessions restore-plan <session-id>
```

输出内容应由人或另一个本地工具复核后，再决定是否运行任何命令。一个 ready plan 可以包含类似命令：

```bash
adp run codex --workspace game-a --profile senior-engineer --task task-20260608-0003 --keep-runtime -- --example-smoke
```

如果数据缺失，ADP 应报告 missing fields 和 reasons，而不是假装 session 可以完整重建。Invocation snapshot 加入前创建的旧 sessions 预期会输出 partial plans。

## Fake Or Local Agent Workflow

在不依赖外部 provider CLI 的情况下验收 restore-plan 行为时，可以使用 fake local agent：

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
TASK_ID=$(adp tasks add --workspace game-a --priority high "Exercise restore-plan guidance" | sed -n 's/^task \(task-[^ ]*\) added$/\1/p')
test -n "$TASK_ID"
adp run codex --workspace game-a --task "$TASK_ID" -- --example-smoke
```

查看本地 evidence：

```bash
adp events list --workspace game-a --task "$TASK_ID"
adp sessions list --workspace game-a --agent codex --task "$TASK_ID"
adp sessions show <session-id>
adp sessions restore-plan <session-id>
```

运行 `restore-plan` 只能打印 inspection output。Event count、task state、runtime directories 和真实项目根目录不应因为 restore-plan 命令本身而变化。

如果手动运行建议命令，它会启动一次新的本地 Agent run，并产生新的 session ID。它不会 attach 到之前的 provider conversation。

## 操作规则

- 将 restore-plan output 视为指导，而不是自动修复或 resume 动作。
- 如果 session 需要纳入项目规划追踪，使用 `adp run <agent> --task <task-id>` 绑定工作。
- 通过 `adp tasks update`、`adp tasks done` 或 `adp tasks block` 显式推进 task 状态。
- 将 restore-plan checks 与 `adp events list`、`adp sessions list`、`adp sessions show` 配合使用，保留本地 acceptance evidence。
- 不要把 restore-plan 描述为云同步、远程 issue 跟踪、托管编排或 provider-native resume。
