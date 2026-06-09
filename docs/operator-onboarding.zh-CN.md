# Operator 入门

English: [operator-onboarding.md](operator-onboarding.md)

本文档是给新 ADP operator 的具体首次运行路径。它保持 terminal-first 和 local-first：默认演练不需要 Web UI、dashboard、SaaS tracker、cloud sync、hosted orchestration、automatic Git workflow 或真实 provider CLI。

安装细节见 [install.zh-CN.md](install.zh-CN.md)。可复用的 workspace 配置示例见 `examples/basic-workspace`。

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

## 隔离首次运行

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

最后一条命令应该成功，并且不会打印 project-root 泄漏项。ADP 状态位于临时 `$ADP_HOME`，runtime overlay 位于临时 `$ADP_RUNTIME_DIR`，provider 命令是本地 fake `codex` 脚本。演练中的只读 inspection 命令包括 `tasks next`、`tasks stale`、`progress report`、`sessions list`、`sessions restore-plan`、`plan doctor`、`events list` 和 `progress`；会修改本地 ledger 的命令包括 `tasks add`、`run --take`、`tasks renew`、`tasks take`、`tasks release` 和 `tasks done`。

## 切换到持久本地使用

隔离演练通过后，再选择持久本地路径：

```bash
export ADP_HOME="${HOME}/.adp"
export ADP_RUNTIME_DIR="${TMPDIR:-/tmp}/adp-runtime"
adp_local init
adp_local workspace add game-a /absolute/path/to/project
adp_local workspace doctor game-a
```

保持 `$ADP_RUNTIME_DIR` 位于 project root 之外，也不要放在包含 project root 的目录下。`adp doctor` 和 `adp workspace doctor` 会在真实运行前报告不安全的 runtime parent。

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
