# 真实 Agent 兼容边界

English: [real-agent-compatibility.md](real-agent-compatibility.md)

本文档定义 ADP 启动真实外部 Agent CLI 时保证什么，以及哪些内容仍然属于 operator 负责的兼容性检查。本文刻意记录 ADP 自身的 adapter contract，而不是记录可能在本仓库之外变化的外部 CLI 细节。

## 兼容模型

ADP 负责本地 runtime 边界：

- 解析 workspace。
- 在 `ADP_RUNTIME_DIR` 下构建隔离 runtime overlay。
- 在 runtime overlay 内生成 adapter-specific 文件。
- 将真实项目文件 symlink 到 runtime overlay。
- 以 runtime root 作为工作目录启动外部 agent 进程。
- 保留父进程环境，并注入 ADP 环境变量。
- 将 `--` 之后的参数透传给外部命令。
- 记录本地 events 和 session history。
- 避免把 ADP 生成文件写入真实项目根目录。

外部 Agent CLI 负责启动后的自身行为，包括认证、模型选择、网络访问、工具权限、prompt 解释和交互行为。

## 共享 Runtime Contract

对于 `adp run <agent> --workspace <name> -- <agent-args>`，ADP 会构建一个 runtime root，其中包含：

- `.adp-runtime.yaml`，即 ADP runtime manifest。
- Adapter 生成的 instruction 和配置文件。
- 指向真实项目根目录中文件和目录的 symlink，除非某个生成路径优先生效。

被启动的进程会收到：

- 工作目录：runtime root。
- 父进程环境：继承自启动 `adp` 的 shell。
- ADP runtime 变量：
  - `ADP_HOME`。
  - `ADP_WORKSPACE`。
  - `ADP_PROJECT_ROOT`。
  - `ADP_RUNTIME_ROOT`。
  - `ADP_SESSION_ID`。
  - `ADP_AGENT`。
  - `ADP_PROFILE`，当 profile 被解析出来时存在。

Workspace agent command 可以通过 `workspace.yaml` 覆盖默认命令：

```yaml
agents:
  codex:
    enabled: true
    command: codex
  claude:
    enabled: true
    command: claude
```

如果本地 operator 需要额外的外部 CLI 设置，可以把 wrapper script 设置为 `command`。尽量把 wrapper 放在真实项目根目录之外。

## Codex Adapter Contract

Codex adapter 名称是 `codex`。

默认启动命令：

```text
codex
```

生成的 runtime 文件：

- `AGENTS.md`。
- `.codex/config.toml`。

生成的 `AGENTS.md` 包含从 workspace config 组装出的 ADP runtime instructions，包括 workspace metadata、base prompt、shared memory、rules、MCP summary，以及可用时的 selected profile 内容。

生成的 `.codex/config.toml` 包含本次 runtime session 的 ADP metadata。它是 ADP 生成文件，不应被视为外部 Codex CLI 当前配置 schema 的完整声明。

ADP 会直接透传 `--` 之后的参数：

```bash
adp run codex --workspace game-a -- <codex-args>
```

ADP 不定义 Codex 支持哪些参数。依赖这些参数前，应在 operator 机器上根据已安装的 Codex CLI 验证。

## Claude Adapter Contract

Claude adapter 名称是 `claude`。

默认启动命令：

```text
claude
```

生成的 runtime 文件：

- `CLAUDE.md`。
- `.claude/settings.json`。

生成的 `CLAUDE.md` 包含从 workspace config 组装出的 ADP runtime instructions，包括 workspace metadata、base prompt、shared memory、rules、MCP summary，以及可用时的 selected profile 内容。

生成的 `.claude/settings.json` 包含本次 runtime session 的 ADP metadata。它是 ADP 生成文件，不应被视为外部 Claude CLI 当前 settings schema 的完整声明。

ADP 会直接透传 `--` 之后的参数：

```bash
adp run claude --workspace game-a -- <claude-args>
```

ADP 不定义 Claude 支持哪些参数。依赖这些参数前，应在 operator 机器上根据已安装的 Claude CLI 验证。

## Opt-In 真实 CLI Smoke

默认仓库 smoke 使用 fake agent。真实外部 CLI 检查是 opt-in：

```bash
ADP_SMOKE_REAL_CODEX=1 scripts/runtime-smoke.sh --real-codex
ADP_SMOKE_REAL_CLAUDE=1 scripts/runtime-smoke.sh --real-claude
```

需要时可以覆盖命令路径：

```bash
ADP_SMOKE_REAL_CODEX=1 ADP_SMOKE_CODEX_BIN=/path/to/codex scripts/runtime-smoke.sh --real-codex
ADP_SMOKE_REAL_CLAUDE=1 ADP_SMOKE_CLAUDE_BIN=/path/to/claude scripts/runtime-smoke.sh --real-claude
```

这些检查只确认外部命令存在，并且轻量的 `--version` 或 `--help` invocation 能完成。它们不能证明真实交互 session 可以完成认证、选择模型、访问 provider 或正确使用外部工具。

## 手工验收步骤

验证真实 Agent 时，使用临时 ADP home 和 runtime 目录：

```bash
tmp="$(mktemp -d)"
export ADP_HOME="$tmp/adp-home"
export ADP_RUNTIME_DIR="$tmp/runtime"
mkdir -p "$tmp/project"
printf 'real-agent-smoke\n' > "$tmp/project/README.md"

adp init
adp workspace add real-agent-smoke "$tmp/project"
adp workspace doctor real-agent-smoke
```

先在仓库根目录运行确定性的 fake smoke：

```bash
scripts/runtime-smoke.sh --fake
```

然后运行 opt-in 的命令可用性检查：

```bash
ADP_SMOKE_REAL_CODEX=1 scripts/runtime-smoke.sh --real-codex
ADP_SMOKE_REAL_CLAUDE=1 scripts/runtime-smoke.sh --real-claude
```

真实启动验收时，选择已安装外部 CLI 支持的 operator-safe 参数：

```bash
adp run codex --workspace real-agent-smoke -- <operator-safe-codex-args>
adp run claude --workspace real-agent-smoke -- <operator-safe-claude-args>
```

每次运行后，检查本地 ADP evidence：

```bash
adp events list --workspace real-agent-smoke
adp sessions list --workspace real-agent-smoke
adp sessions show <session-id>
```

确认真实项目根目录保持干净：

```bash
test ! -e "$tmp/project/AGENTS.md"
test ! -e "$tmp/project/CLAUDE.md"
test ! -e "$tmp/project/.codex"
test ! -e "$tmp/project/.claude"
```

记录 ADP commit、外部命令路径、外部命令版本或 help 输出、workspace 名称，以及使用过的 operator-specific 参数。

## 边界与失败处理

以下内容不属于 ADP 确定性兼容保证：

- Provider 账号状态。
- 网络访问。
- 模型可用性。
- 外部 CLI release 行为。
- 外部 CLI prompt/configuration schema。
- 交互 session 语义。
- 外部 CLI 授予的外部工具权限。

当真实 CLI 行为变化时，先在目标机器上验证，再调整 ADP adapter 假设。如果变化只属于某个 operator 的本地环境，优先在 `workspace.yaml` 中使用显式 wrapper command，而不是修改共享 adapter contract。
