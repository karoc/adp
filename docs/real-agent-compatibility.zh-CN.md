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

Task ownership 仍由 ADP 管理。Provider 原生 task panel、plan mode 或外部进程成功退出可以镜像或辅助本地工作，但不能被视为 task completion、phase acceptance、commit evidence、push evidence、Git execution，或权威 recovery state。

## 共享 Runtime Contract

当前已文档化的真实 Agent adapter contract 是 `codex` 和 `claude`。未来 adapter design notes 在 adapter 实现并验收前应使用中性 placeholder，并且不得描述 provider-native resume 语义。

对于 `adp run <agent> --workspace <name> -- <agent-args>`，ADP 会构建一个 runtime root，其中包含：

- `.adp-runtime.yaml`，即 ADP runtime manifest。
- Adapter 生成的 instruction 和配置文件。
- 指向真实项目根目录中文件和目录的 symlink，除非某个生成路径优先生效。

当 operator 传入 `--task <task-id>` 时，ADP 会把这个既有 task 绑定到 runtime。当 operator 传入 `--take --owner <owner> [--lease 4h]` 时，ADP 会在 runtime 创建前原子领取下一个 eligible task，并把被领取的 task 绑定到 runtime。`--take` 与 `--task` 互斥。

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

如果真实项目已经存在 `.claude/settings.local.json` 等 provider-local configuration，ADP 会在 runtime overlay 中保留非冲突文件。ADP 生成的 metadata 仍会在精确的 `.claude/settings.json` 路径上优先，避免项目文件覆盖本次 runtime session metadata。

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

真实 CLI flag 会叠加在默认 fake smoke 之后运行。它们不会替代 `scripts/check-all.sh`，除非某个 release 明确声明 real-agent evidence，否则也不应被当作 CI 要求。

需要时可以覆盖命令路径：

```bash
ADP_SMOKE_REAL_CODEX=1 ADP_SMOKE_CODEX_BIN=/path/to/codex scripts/runtime-smoke.sh --real-codex
ADP_SMOKE_REAL_CLAUDE=1 ADP_SMOKE_CLAUDE_BIN=/path/to/claude scripts/runtime-smoke.sh --real-claude
```

这些检查只确认外部命令存在，并且轻量的 `--version` 或 `--help` invocation 能完成。它们是 command availability checks，不是 model invocation evidence。默认 doctor diagnostics 仍然是静态且本地的：它们可以提示 command 形态、wrapper 路径、profile 和 reserved path 风险，但不会运行 provider CLI。两类路径都不能证明真实交互 session 可以完成认证、选择模型、访问 provider、消耗 quota，或正确使用外部工具。

手工 real-agent acceptance 由 operator 负责。它可能需要本地凭据、网络访问、provider 账号额度、模型访问权限和外部工具权限，这些不是 ADP 能确定性创建或验证的内容。

## Opt-In 真实 Agent 调用 Smoke

`scripts/real-agent-invocation-smoke.sh` 是通过 ADP 显式收集非交互 Codex 和 Claude invocation evidence 的专用路径。它独立于 `scripts/runtime-smoke.sh --real-codex` 和 `scripts/runtime-smoke.sh --real-claude`：runtime smoke 的真实 flag 检查 command availability，而 invocation smoke 用于证明在当前 operator 环境中，ADP 可以把受限的非交互请求交给已安装的外部 CLI。

invocation smoke 不属于 `scripts/check-all.sh`，并且不能变成默认 CI 或 release gate。只有当某个 release、audit 或 operator note 明确要求 real-agent invocation evidence，且 operator 接受该脚本可能访问外部 provider、使用机器上已有账号凭据并消耗 provider quota 时，才运行它。

只能在脚本或 release procedure 记录的显式 opt-in gates 已满足时运行该脚本：

```bash
ADP_REAL_INVOKE_CODEX=1 scripts/real-agent-invocation-smoke.sh --codex
ADP_REAL_INVOKE_CLAUDE=1 scripts/real-agent-invocation-smoke.sh --claude
ADP_REAL_INVOKE_CODEX=1 ADP_REAL_INVOKE_CLAUDE=1 scripts/real-agent-invocation-smoke.sh --all
```

该 smoke 应构建或选择正在被验收的 ADP binary，创建临时 `ADP_HOME`、`ADP_RUNTIME_DIR` 和 project root，注册临时 workspace，通过 `adp run ...` 调用 Codex 和 Claude，检查本地 events 和 sessions，然后删除临时状态。它不应把 planning files、reports、generated instruction files、provider output 或 runtime metadata 写入真实仓库项目根目录。

该脚本产生的 evidence 必须保持非敏感。只记录 ADP version 或 commit、外部命令路径、外部命令版本或首行 help 输出、adapter names、workspace name、session IDs、exit statuses、已脱敏 timestamps，以及每条 invocation path 的 passed 或 failed 结果。不要记录 secrets、tokens、API keys、私有 prompts、账号标识、完整模型回复、专有代码片段，或 provider-specific conversation IDs。

通过 invocation smoke 只是一份环境相关 evidence。它不保证其他 operator 拥有凭据、模型访问权限、可用 quota、稳定网络、相同外部 CLI 版本、等价工具权限或可接受的交互 session 质量。失败时应先按 operator evidence 排查：先验证本地认证、命令路径、provider 可用性、quota、网络访问和外部 CLI 变化，再修改 ADP adapter 假设。

## 手工验收步骤

先选择正在被验收的同一个 ADP binary。对于源码 checkout，从当前 commit 构建一个临时 binary，并记录 version 输出：

```bash
tmp="$(mktemp -d)"
ADP_BIN="$tmp/adp"
go build -o "$ADP_BIN" ./cmd/adp
"$ADP_BIN" version
```

对于 packaged release candidate，改用 packaged binary，并记录 version 输出：

```bash
tmp="$(mktemp -d)"
ADP_BIN=/path/to/adp
"$ADP_BIN" version
```

先在仓库根目录运行确定性的 repository gate：

```bash
scripts/check-all.sh
```

`scripts/runtime-smoke.sh` 会创建自己的临时 ADP home 和 runtime 目录。它的 evidence 应视为 repository gate evidence，不应和手工真实启动 sandbox 混用。

只有在这些检查被明确纳入 release evidence 时，才运行 opt-in 的命令可用性检查：

```bash
ADP_SMOKE_REAL_CODEX=1 scripts/runtime-smoke.sh --real-codex
ADP_SMOKE_REAL_CLAUDE=1 scripts/runtime-smoke.sh --real-claude
```

当 release evidence 明确包含通过 ADP 进行的真实非交互模型调用时，在命令可用性检查之后运行专用 invocation smoke，并保持输出脱敏：

```bash
ADP_REAL_INVOKE_CODEX=1 ADP_REAL_INVOKE_CLAUDE=1 scripts/real-agent-invocation-smoke.sh --all
```

手工真实启动验收时，使用同一个 `ADP_BIN` 创建单独的临时 ADP home 和 runtime 目录：

```bash
export ADP_HOME="$tmp/adp-home"
export ADP_RUNTIME_DIR="$tmp/runtime"
mkdir -p "$tmp/project"
printf 'real-agent-smoke\n' > "$tmp/project/README.md"

"$ADP_BIN" init
"$ADP_BIN" workspace add real-agent-smoke "$tmp/project"
"$ADP_BIN" workspace doctor real-agent-smoke
```

真实启动验收时，选择已安装外部 CLI 支持的 operator-safe 参数：

```bash
"$ADP_BIN" run codex --workspace real-agent-smoke -- <operator-safe-codex-args>
"$ADP_BIN" run claude --workspace real-agent-smoke -- <operator-safe-claude-args>
```

常见安全候选是已安装外部 CLI 支持的 `--version` 或 `--help`，但 ADP 不定义外部 CLI 参数。

这些命令可能访问外部 provider。不要把它们作为默认 CI 或 release gate，也不要把 secret、token、私有 prompt、账号标识或敏感模型输出记录为 evidence。

每次运行后，检查本地 ADP evidence：

```bash
"$ADP_BIN" events list --workspace real-agent-smoke
"$ADP_BIN" sessions list --workspace real-agent-smoke
"$ADP_BIN" sessions show <session-id>
"$ADP_BIN" sessions restore-plan <session-id>
```

即使对真实 Agent，`sessions restore-plan` 也保持只读。它可以基于本地 invocation metadata 建议一条相似的新 `adp run ...` 命令，但不会恢复 provider 原生 conversation，也不会执行建议命令。

确认真实项目根目录保持干净：

```bash
test ! -e "$tmp/project/AGENTS.md"
test ! -e "$tmp/project/CLAUDE.md"
test ! -e "$tmp/project/.codex"
test ! -e "$tmp/project/.claude"
test ! -e "$tmp/project/planning"
test ! -e "$tmp/project/tasks.yaml"
test ! -e "$tmp/project/phases.yaml"
test ! -e "$tmp/project/progress.jsonl"
```

记录 ADP commit 或 packaged version、`"$ADP_BIN" version` 输出、外部命令路径、外部命令版本或 help 输出、workspace 名称，以及使用过的 operator-specific 参数。

同时记录该 evidence 只证明命令可用性，还是已经完成 `scripts/real-agent-invocation-smoke.sh` 的非交互模型调用，或已经完成手工交互式 `adp run ...` 验收。

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
