# 基础工作区示例

此目录是可复制的 ADP 本地工作区配置。把它复制到
`$ADP_HOME/workspaces/<name>`，将 `project.root` 指向真实本地项目，并在启动
任何 Agent 前完成验证。

仓库中的 `workspace.yaml` 使用：

```yaml
workspace:
  name: game-a

project:
  root: /srv/game-a
```

使用此示例前，必须把 `workspace.name` 替换为你的工作区名称，并把
`project.root: /srv/game-a` 替换为你本机的绝对路径。

## 目录内容

- `prompts/`：工作区引用的可复用提示词文件。
- `memory/`：提供给 Agent 上下文使用的本地共享记忆文件。
- `mcp/`：工作区使用的 MCP 配置。
- `profiles/`：Agent profile 文件，包含 Codex 和 Claude 示例。
- `workspace.yaml`：ADP 从 `$ADP_HOME/workspaces/<name>` 读取的工作区清单。

## 本地快速演练

安装或构建 `adp` 后，在 ADP 仓库根目录运行以下命令。演练使用临时目录，
因此不依赖已有的操作者状态。此示例是可复制配置路径；如果需要通过 CLI 命令
创建 workspace 的最小首次试运行路径，见
[../../docs/operator-onboarding.zh-CN.md](../../docs/operator-onboarding.zh-CN.md)。

准备隔离的 ADP 状态和一个很小的本地项目：

```bash
export ADP_HOME="$(mktemp -d)"
export ADP_RUNTIME_DIR="$(mktemp -d)"
project_root="$(mktemp -d)"
printf 'module example.com/adp-basic-workspace\n' > "${project_root}/go.mod"
printf 'package main\n' > "${project_root}/main.go"
adp init
```

把示例复制到 `$ADP_HOME`：

```bash
mkdir -p "${ADP_HOME}/workspaces"
cp -R examples/basic-workspace "${ADP_HOME}/workspaces/my-workspace"
```

编辑复制后的清单：

```bash
$EDITOR "${ADP_HOME}/workspaces/my-workspace/workspace.yaml"
```

设置工作区名称和项目根目录：

```yaml
workspace:
  name: my-workspace

project:
  root: /absolute/path/from/project_root
```

使用 `project_root` shell 变量中保存的绝对路径。不要把 `project.root` 指向
`$ADP_HOME`、`$ADP_RUNTIME_DIR` 或复制后的工作区目录。

启动任何 run 前，先验证复制后的工作区：

```bash
adp workspace doctor my-workspace
adp workspace show my-workspace
adp env my-workspace --cd
```

`adp env my-workspace --cd` 会在 `$ADP_RUNTIME_DIR` 下创建或定位 runtime
overlay，并输出 shell exports 和进入该 overlay 的 `cd` 命令。真实项目根目录应
只保留你创建或本来拥有的文件。
预期结果：`workspace doctor` 成功退出，`workspace show` 打印复制后的清单
细节，`env --cd` 打印指向真实 project root 之外 runtime overlay 的 shell
exports。

## 不依赖 Provider 的运行

onboarding 时使用 fake local `codex` 命令。这样可以证明 ADP 能通过复制后的
工作区启动 Agent，同时不要求真实 provider CLI、账号、网络访问或托管服务。

```bash
fake_bin="$(mktemp -d)"
cat > "${fake_bin}/codex" <<'SH'
#!/usr/bin/env sh
printf 'fake codex received: %s\n' "$*"
SH
chmod +x "${fake_bin}/codex"
export PATH="${fake_bin}:${PATH}"
adp run codex --workspace my-workspace -- --example-smoke
```

预期结果：fake provider 打印透传参数，ADP 记录本地 event/session evidence，
且不需要真实 provider 账号。

查看本地 runtime evidence：

```bash
adp events list --workspace my-workspace
adp sessions list --workspace my-workspace
adp sessions show <session-id>
adp sessions restore-plan <session-id>
```

当非敏感 invocation 数据足够时，`restore-plan` 会打印建议的 `adp run ...`
命令。它不会执行命令，不会创建 runtime workspace，不会修改 task 状态，不会
追加新 events，不会写入真实项目根目录，也不会恢复 provider-native
conversation。参见 [../../docs/session-restore.zh-CN.md](../../docs/session-restore.zh-CN.md)。

运行后，真实项目根目录不应出现 ADP 生成的文件，例如 `AGENTS.md`、
`CLAUDE.md`、`.codex`、`.claude`、`.adp-runtime.yaml`、`planning`、
`tasks.yaml`、`phases.yaml` 或 `progress.jsonl`。Runtime overlay 应位于
`$ADP_RUNTIME_DIR` 下；工作区配置和本地 planning 状态应位于 `$ADP_HOME`
下。

## 可选真实 Agent 运行

只有在对应外部 CLI 已安装并完成认证后，才通过 ADP 运行真实 Agent：

```bash
adp run codex --workspace my-workspace
adp run claude --workspace my-workspace
```

缺少真实 Codex 或 Claude CLI 不是 onboarding 失败。确定性的本地验证应使用
上面的 provider-free run。

ADP 将 runtime 状态保留在本地。此示例不需要 Web 服务、托管控制面、SaaS
账号、cloud sync 或 automatic Git 行为。
