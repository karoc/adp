# 基础工作区示例

此目录是可复制的 ADP 本地工作区配置。它不是能在任意机器上原样运行的固定项目。

示例 `workspace.yaml` 使用：

```yaml
workspace:
  name: game-a

project:
  root: /srv/game-a
```

使用前，必须把 `workspace.name` 替换为你的工作区名称，并把 `project.root: /srv/game-a` 替换为你本机真实项目的绝对路径。

## 目录内容

- `prompts/`：工作区引用的可复用提示词文件。
- `memory/`：提供给 Agent 上下文使用的本地共享记忆文件。
- `mcp/`：工作区使用的 MCP 配置。
- `profiles/`：Agent profile 文件，包含 Codex 和 Claude 示例。
- `workspace.yaml`：ADP 从 `$ADP_HOME/workspaces/<name>` 读取的工作区清单。

## 本地使用

如果你还没有设置 `ADP_HOME`，可以先设置：

```bash
export ADP_HOME="${HOME}/.adp"
```

把示例复制到工作区目录：

```bash
mkdir -p "${ADP_HOME}/workspaces"
cp -R examples/basic-workspace "${ADP_HOME}/workspaces/my-workspace"
```

编辑复制后的清单：

```bash
$EDITOR "${ADP_HOME}/workspaces/my-workspace/workspace.yaml"
```

更新这些字段：

```yaml
workspace:
  name: my-workspace

project:
  root: /absolute/path/to/your/project
```

启动 Agent 前先运行诊断：

```bash
adp workspace doctor my-workspace
```

输出工作区的 shell 环境提示：

```bash
adp env my-workspace --cd
```

通过 ADP 运行 Agent，使其进入隔离的 runtime workspace：

```bash
adp run codex --workspace my-workspace
adp run claude --workspace my-workspace
```

查看本地运行历史：

```bash
adp events list
adp sessions list --workspace my-workspace
adp sessions show <session-id>
```

为历史 session 输出只读 restore plan：

```bash
adp sessions restore-plan <session-id>
```

当非敏感 invocation 数据足够时，`restore-plan` 会打印建议的 `adp run ...` 命令。它不会执行命令，不会创建 runtime workspace，不会修改 task 状态，不会追加新 events，不会写入真实项目根目录，也不会恢复 provider 原生 conversation。参见 [../../docs/session-restore.zh-CN.md](../../docs/session-restore.zh-CN.md)。

## Task-Bound Fake Agent Workflow

如果你想在不依赖真实 provider CLI 的情况下验收 runtime history 和 restore-plan guidance，可以使用 fake local agent：

```bash
fake_bin="$(mktemp -d)"
cat > "${fake_bin}/codex" <<'SH'
#!/usr/bin/env sh
printf 'fake codex received: %s\n' "$*"
SH
chmod +x "${fake_bin}/codex"
PATH="${fake_bin}:${PATH}"
```

创建 task，把一次 runtime session 绑定到它，并在 `--` 后传入本地 Agent 参数：

```bash
TASK_ID=$(adp tasks add --workspace my-workspace --priority high --phase p4-session-restore "Exercise restore-plan guidance" | sed -n 's/^task \(task-[^ ]*\) added$/\1/p')
adp run codex --workspace my-workspace --task "$TASK_ID" -- --example-smoke
```

查看 task-bound evidence：

```bash
adp events list --workspace my-workspace --task "$TASK_ID"
adp sessions list --workspace my-workspace --agent codex --task "$TASK_ID"
adp sessions show <session-id>
adp sessions restore-plan <session-id>
```

如果你手动运行建议命令，ADP 会启动一次新的本地 run，并产生新的 session ID。该命令只是新运行的 guidance，不是 automatic replay，也不是 provider-native conversation resume。

ADP 将运行时状态保留在本地。此示例不需要 Web 服务、托管控制面或 SaaS 账号。
