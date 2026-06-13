# Operator 入门

English: [operator-onboarding.md](operator-onboarding.md)

本文档是给新 ADP operator 的具体首次运行路径。它保持 terminal-first 和 local-first：默认演练不需要 Web UI、dashboard、SaaS tracker、cloud sync、hosted orchestration、automatic Git workflow 或真实 provider CLI。

安装细节见 [install.zh-CN.md](install.zh-CN.md)。可复用的 workspace 配置示例见 `examples/basic-workspace`。

## 当前开始使用边界

ADP 已经适合本地技术 operator 从不依赖 provider 的首次试运行开始；隔离演练通过后，再切换到持久本地 workspace 使用。请把这理解为 terminal-first workflow 的本地试用准备度，而不是所有真实 provider 环境或交互式 session 都已经 production-ready 的声明。

当你需要验证 local install path、workspace registration、diagnostics、task pickup、fake-provider runtime handoff、event/session/progress inspection、restore guidance、completion values、runtime prune dry-run 和 project-root cleanliness 时，使用本指南。真实 provider authentication、model access、quota、network behavior 和 interactive session quality 仍然是单独的 opt-in acceptance checks。

## 首次试运行验证什么

首次试运行是本地演练，不是生产配置。完成后，你应该得到以下 evidence：

- 选定的 `adp` 命令可以运行，并能输出带可复制示例的嵌套命令 help，常见 parser error 会指向正确的 help 页面；
- ADP 可以在临时 `$ADP_HOME` 下初始化隔离的本地状态；
- workspace 可以指向本地项目，同时不把 ADP 文件写入该 project root；
- `workspace doctor` 和 `doctor` 可以在 Agent 运行前检查本地配置；
- `adp run codex --take --owner --lease` 可以原子领取 task、构建 runtime overlay，并启动 provider 命令；
- events、sessions、progress、restore guidance、plan diagnostics、completion values 和 runtime prune dry-run 都可以从本地 ADP 状态读取；并且
- 同一个看板可以通过 `tasks next` 只读检查，也可以通过 `tasks take` 在不启动 Agent 的情况下领取。

本指南里的 provider 是一个本地 fake `codex` shell 脚本。首次试运行通过，并不证明真实 provider 的认证、模型访问、quota、网络行为或交互式 session 质量。

## 选择 ADP 命令

在当前 shell 中只选择一种命令形式。

开发时从源码运行：

```bash
adp_local() { go run ./cmd/adp "$@"; }
adp_local version
```

使用本地构建的二进制：

```bash
mkdir -p bin
go build -o ./bin/adp ./cmd/adp
adp_local() { "$PWD/bin/adp" "$@"; }
adp_local version
```

使用临时安装路径，包括解压包含 `bin/adp` 的 release package 后：

```bash
mkdir -p bin
go build -o ./bin/adp ./cmd/adp
ADP_INSTALL_DIR="$(mktemp -d)"
install -m 0755 ./bin/adp "${ADP_INSTALL_DIR}/adp"
adp_local() { "${ADP_INSTALL_DIR}/adp" "$@"; }
adp_local version
```

如果使用已发布 artifact，把最后一个命令块中的 `./bin/adp` 替换为解压后的 artifact 路径。临时安装路径应放在 project root 之外。

创建状态前，先用你选定的同一命令形式确认命令面：

```bash
adp_local --help
adp_local workspace --help
adp_local tasks --help
adp_local tasks take --help
adp_local sessions resume-plan --help
adp_local runtime prune --help
```

其他命令组也使用同样的嵌套模式：`adp_local <command> --help` 和 `adp_local <command> <subcommand> --help`。叶子 help 可能包含指向父命令的 `See also:`。如果某个构建打印友好的 `try:` hint，把它理解为建议手动运行的 help 命令；它本身不会 inspect project、修改 `$ADP_HOME`、创建 runtime、调用 provider 或运行 Git。

预期结果：每条命令成功退出，并打印本地 help 或 version 文本，首用命令会带有可复制示例。如果这里失败，先修复选定的命令路径，再创建任何 ADP 状态。

## ID 前缀匹配

ADP 支持方便的 task 和 session ID 前缀匹配。无需输入完整 ID，你可以使用任何唯一前缀：

```bash
# Task ID 前缀匹配
adp tasks show task-20260611-0001    # Full ID
adp tasks show task-2026             # Prefix (if unique)
adp tasks claim task-001 --owner alice --lease 2h

# Session ID 前缀匹配
adp sessions show session-20260611T102030-abc123    # Full ID
adp sessions show 20260611T10                       # Prefix (if unique)
adp sessions restore-plan 2026061
```

当前缀有歧义（匹配多个 ID）时，ADP 会返回错误并列出所有匹配的 ID：

```bash
adp tasks show task-20
# Error: ambiguous task ID "task-20", matches:
#   - task-20260611-0001
#   - task-20260612-0002
```

前缀匹配在所有接受 task 或 session ID 的命令中可用，包括 `tasks show/claim/renew/release/done/block`、`sessions show/restore-plan/resume-plan`、`events list` 和 `run --task`。

提示：
- 当有许多 task 或 session 时使用较长的前缀
- 对于最近的 ID，最短的唯一前缀通常是日期部分

## 隔离首次运行

### 使用 Quickstart 命令快速开始（推荐）

最快速的入门方式是使用 `quickstart` 命令，它会自动完成初始化和 workspace 设置：

```bash
# 交互模式 - 引导你完成设置
adp_local quickstart

# 非交互模式（用于脚本/自动化）
ADP_ONBOARDING_ROOT="$(mktemp -d)"
export ADP_HOME="${ADP_ONBOARDING_ROOT}/adp-home"
mkdir -p "${ADP_ONBOARDING_ROOT}/project"
printf 'module example.com/adp-onboarding\n' > "${ADP_ONBOARDING_ROOT}/project/go.mod"
printf 'package main\n' > "${ADP_ONBOARDING_ROOT}/project/main.go"

adp_local quickstart \
  --non-interactive \
  --workspace-name game-a \
  --project-root "${ADP_ONBOARDING_ROOT}/project" \
  --memory \
  --mcp
```

`quickstart` 命令会：
1. 初始化你的 ADP home 目录
2. 使用推荐设置创建你的第一个 workspace
3. 可选地运行诊断以验证设置

quickstart 完成后，你可以直接跳到添加任务和运行 Agent（见下面"添加任务并运行 Agent"）。

### 手动设置（备选方案）

如果你更喜欢手动控制或需要理解每一步，使用下面的详细设置。

在确认安装路径可信前，先使用临时状态。这次演练会注册一个临时 workspace，检查任务看板，通过原子 `run --take` 运行 fake `codex` provider，记录本地 events 和 sessions，检查 lease 维护，并验证 project root 保持干净。

```bash
ADP_ONBOARDING_ROOT="$(mktemp -d)"
export ADP_HOME="${ADP_ONBOARDING_ROOT}/adp-home"
export ADP_RUNTIME_DIR="${ADP_ONBOARDING_ROOT}/runtime"
mkdir -p "${ADP_ONBOARDING_ROOT}/project" "${ADP_ONBOARDING_ROOT}/fake-bin"
printf 'module example.com/adp-onboarding\n' > "${ADP_ONBOARDING_ROOT}/project/go.mod"
printf 'package main\n' > "${ADP_ONBOARDING_ROOT}/project/main.go"

cat > "${ADP_ONBOARDING_ROOT}/fake-bin/codex" <<'SH'
#!/usr/bin/env sh
printf 'fake codex cwd=%s args=%s\n' "$(pwd)" "$*"
test -n "${ADP_SESSION_ID:-}"
test -n "${ADP_RUNTIME_ROOT:-}"
test -n "${ADP_TASK_ID:-}"
test "$(pwd)" = "$ADP_RUNTIME_ROOT"
test -f "$ADP_RUNTIME_ROOT/AGENTS.md"
test -f "$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
SH
chmod +x "${ADP_ONBOARDING_ROOT}/fake-bin/codex"
export PATH="${ADP_ONBOARDING_ROOT}/fake-bin:${PATH}"

adp_local init
adp_local workspace add game-a "${ADP_ONBOARDING_ROOT}/project"
adp_local workspace list
adp_local workspace show game-a
adp_local workspace doctor game-a
adp_local doctor game-a
adp_local version
```

### 添加任务并运行 Agent

无论使用 `quickstart` 还是手动设置，都继续进行任务管理和 Agent 运行：

```bash
# 创建 fake codex provider（如果还没创建）
if [ ! -f "${ADP_ONBOARDING_ROOT}/fake-bin/codex" ]; then
  mkdir -p "${ADP_ONBOARDING_ROOT}/fake-bin"
  cat > "${ADP_ONBOARDING_ROOT}/fake-bin/codex" <<'SH'
#!/usr/bin/env sh
printf 'fake codex cwd=%s args=%s\n' "$(pwd)" "$*"
test -n "${ADP_SESSION_ID:-}"
test -n "${ADP_RUNTIME_ROOT:-}"
test -n "${ADP_TASK_ID:-}"
test "$(pwd)" = "$ADP_RUNTIME_ROOT"
test -f "$ADP_RUNTIME_ROOT/AGENTS.md"
test -f "$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
SH
  chmod +x "${ADP_ONBOARDING_ROOT}/fake-bin/codex"
  export PATH="${ADP_ONBOARDING_ROOT}/fake-bin:${PATH}"
fi

TASK_ID=$(adp_local tasks add --workspace game-a --priority high "Validate isolated first run" | sed -n 's/^task \(task-[^ ]*\) added$/\1/p')
test -n "$TASK_ID"
adp_local tasks next --workspace game-a --format json
adp_local run codex --workspace game-a --take --owner first-agent --lease 30m -- --onboarding-smoke
adp_local tasks show --workspace game-a "$TASK_ID"
adp_local tasks renew --workspace game-a "$TASK_ID" --owner first-agent --lease 30m
adp_local tasks stale --workspace game-a --format json
adp_local progress report --workspace game-a --format json
SESSION_ID=$(adp_local sessions list --workspace game-a --agent codex --task "$TASK_ID" | sed -n '2s/ .*//p')
test -n "$SESSION_ID"
adp_local sessions restore-plan "$SESSION_ID"
adp_local plan doctor --workspace game-a --format json

BOARD_TASK_ID=$(adp_local tasks add --workspace game-a --priority normal "Validate board pickup" | sed -n 's/^task \(task-[^ ]*\) added$/\1/p')
test -n "$BOARD_TASK_ID"
TAKEN_ID=$(adp_local tasks take --workspace game-a --owner second-agent --lease 30m | sed -n 's/^task \(task-[^ ]*\) taken .*/\1/p')
test -n "$TAKEN_ID"
adp_local tasks release --workspace game-a "$TAKEN_ID" --owner second-agent
adp_local tasks done --workspace game-a "$TASK_ID"
adp_local events list --workspace game-a --task "$TASK_ID" --limit 5
adp_local sessions list --workspace game-a --agent codex --task "$TASK_ID"
adp_local progress --workspace game-a --format json
adp_local runtime prune --older-than 24h --dry-run

ROOT_LEAKS="$(find "${ADP_ONBOARDING_ROOT}/project" -maxdepth 2 \( -name AGENTS.md -o -name CLAUDE.md -o -name .codex -o -name .claude -o -name .adp-runtime.yaml -o -name planning -o -name tasks.yaml -o -name phases.yaml -o -name progress.jsonl \) -print)"
test -z "$ROOT_LEAKS"
```

预期结果：整个命令块成功退出，fake provider 打印 runtime 工作目录，JSON 命令打印可解析的本地状态，最后一条命令成功且不会打印 project-root 泄漏项。ADP 状态位于临时 `$ADP_HOME`，runtime overlay 位于临时 `$ADP_RUNTIME_DIR`，provider 命令是本地 fake `codex` 脚本。

可见工作流是：

- 用 `init` 创建本地 ADP 状态；
- 用 `workspace add`、`workspace list`、`workspace show`、`workspace doctor` 和 `doctor` 注册并检查 workspace；
- 用 `tasks add` 创建 task；
- 用只读 `tasks next` 预览可领取的看板工作；
- 用 `run --take --owner --lease` 在同一个边界里领取工作并启动 runtime；
- 用 `tasks show`、`tasks renew` 和只读 `tasks stale` 检查并维护 ownership；
- 用 `progress report`、`sessions list`、`sessions restore-plan`、`plan doctor`、`events list` 和 `progress` 检查 handoff evidence；
- 用 `tasks take` 证明不启动 runtime 的看板领取，再用 `tasks release` 释放该 claim；并且
- 用 `tasks done` 关闭已完成的试运行 task。

演练中的只读 inspection 命令包括 `tasks next`、`tasks stale`、`progress report`、`sessions list`、`sessions restore-plan`、`plan doctor`、`events list` 和 `progress`；会修改本地 ledger 的命令包括 `tasks add`、`run --take`、`tasks renew`、`tasks take`、`tasks release` 和 `tasks done`。`tasks take` 步骤证明不启动 runtime 时也能从看板领取任务；`run --take` 步骤证明任务领取和 runtime 启动处在同一个命令边界。`tasks renew` 刷新当前 owner 的 lease，而 `tasks stale` 只是检查已过期 `in_progress` claim 的 recovery inspection 视图。

如果试运行失败，先保留临时 root，并检查最后一个失败命令。常见原因包括 `adp_local` function 指向了错误 binary、当前 shell 没有把 fake `codex` 目录 export 到 `PATH`，或 `ADP_RUNTIME_DIR` 不安全。重复完整命令块前，先重新运行 `adp_local workspace doctor game-a` 和 `adp_local doctor game-a`。

## 切换到持久本地使用

隔离演练通过后，再选择持久本地路径：

```bash
export ADP_HOME="${HOME}/.adp"
export ADP_RUNTIME_DIR="${TMPDIR:-/tmp}/adp-runtime"
adp_local init
adp_local workspace add game-a /absolute/path/to/project
adp_local workspace doctor game-a
```

预期结果：持久 workspace 注册到 `${HOME}/.adp` 下，project root 仍然没有 ADP 生成文件，doctor 输出没有 error-level diagnostics。保持 `$ADP_RUNTIME_DIR` 位于 project root 之外，也不要放在包含 project root 的目录下。`adp doctor` 和 `adp workspace doctor` 会在真实运行前报告不安全的 runtime parent。

当你需要一份可复制的配置参考时，再使用 `examples/basic-workspace`。它包含 Codex 和 Claude profile、base prompt、shared memory 与 MCP 设置。复制到 ADP home 的 workspace 配置区域后，先更新 `project.root` 再使用。上面的最小 smoke 路径不依赖这个示例。

## 真实 Provider

真实 Codex 和 Claude 运行属于 opt-in operator check。上面的默认 onboarding 演练仍然保持 provider-free。Provider credentials、quota、model access、network behavior 和外部 CLI versions 都属于 operator environment concerns，不是 ADP quality guarantees。

Command availability evidence 需要有意启用 runtime smoke 的真实 flag。这些检查确认外部命令存在，并且可以完成轻量 `--version` 或 `--help` probe；它们不会调用模型。

```bash
ADP_SMOKE_REAL_CODEX=1 scripts/runtime-smoke.sh --real-codex
ADP_SMOKE_REAL_CLAUDE=1 scripts/runtime-smoke.sh --real-claude
```

非交互真实模型 invocation evidence 需要有意启用专用 invocation smoke。它可能联系外部 provider 并消耗 quota。它不属于 `scripts/check-all.sh`，也不得变成默认 CI 或 release gate。

```bash
ADP_REAL_INVOKE_CODEX=1 scripts/real-agent-invocation-smoke.sh --codex
ADP_REAL_INVOKE_CLAUDE=1 scripts/real-agent-invocation-smoke.sh --claude
ADP_REAL_INVOKE_CODEX=1 ADP_REAL_INVOKE_CLAUDE=1 scripts/real-agent-invocation-smoke.sh --all
```

手工交互式 provider acceptance 独立于上述两条 smoke path。先在 operator 机器上安装并认证外部 CLI，然后运行：

```bash
adp_local run codex --workspace game-a -- <codex-args>
adp_local run claude --workspace game-a -- <claude-args>
```

`--` 之后的参数由 provider 定义。ADP 会透传这些参数，但不定义其安全性、模型可用性、quota 使用、网络行为、认证状态或交互式 session 质量。Operator acceptance notes 应保持非敏感，不要记录凭据、token、账号标识、私有 prompt 或敏感模型输出。完整兼容性流程见 [real-agent-compatibility.zh-CN.md](real-agent-compatibility.zh-CN.md)。
