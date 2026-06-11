# 安装与 Bootstrap

English: [install.md](install.md)

本文档说明 ADP 的本地安装和首次 bootstrap 流程。ADP 是 terminal-first、local-first 的 runtime manager；安装是本地 CLI 工作流，不需要托管服务。面向 operator 的单一路径见 [operator-onboarding.zh-CN.md](operator-onboarding.zh-CN.md)。

## 前置要求

- 本机已安装 Go。
- 用于运行仓库脚本的 POSIX shell 环境。
- 一个具有绝对路径的本地项目目录。
- 可选的外部 Agent CLI，例如 Codex 或 Claude。只有准备运行真实 Agent 时才需要它们；默认 smoke 路径使用 fake agent，不依赖真实外部 CLI。

## 从源码运行

从源码 checkout 开始：

```bash
git clone git@github.com:karoc/adp.git
cd adp
go run ./cmd/adp --help
```

开发时可以通过 `go run` 运行所有 CLI 命令：

```bash
go run ./cmd/adp init
go run ./cmd/adp workspace add game-a /absolute/path/to/project
go run ./cmd/adp workspace doctor game-a
```

当你想测试当前工作区而不安装二进制时，使用这种方式。

## 构建本地二进制

在仓库根目录构建 CLI：

```bash
mkdir -p bin
go build -o ./bin/adp ./cmd/adp
./bin/adp --help
```

使用构建出的二进制进行本地 bootstrap：

```bash
./bin/adp init
./bin/adp workspace add game-a /absolute/path/to/project
./bin/adp workspace doctor game-a
```

## 使用 Go 安装

从源码 checkout 安装：

```bash
go install ./cmd/adp
```

确保 Go 安装目录在 `PATH` 中。除非设置了 `GOBIN`，通常是 `$(go env GOPATH)/bin`：

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
adp --help
```

如果安装已发布的 module 版本，使用带版本的 module path：

```bash
go install github.com/karoc/adp/cmd/adp@<version>
```

使用显式版本可以保证安装可复现。

## 选择 Operator 路径

首次注册 workspace 前，先选择一种路径：

- 源码 checkout：运行 `go run ./cmd/adp version`，开发时继续用 `go run ./cmd/adp <command>`。
- 已构建二进制：运行 `mkdir -p bin && go build -o ./bin/adp ./cmd/adp`，再用 `./bin/adp version` 验证。
- 临时安装路径：把已构建或从 release package 解压出的二进制复制到临时目录并加入 `PATH`，再用 `adp version` 验证。

临时安装路径示例：

```bash
mkdir -p bin
go build -o ./bin/adp ./cmd/adp
ADP_INSTALL_DIR="$(mktemp -d)"
install -m 0755 ./bin/adp "${ADP_INSTALL_DIR}/adp"
export PATH="${ADP_INSTALL_DIR}:${PATH}"
command -v adp
adp version
```

如果使用 release package，先解压 package，并在 `install -m 0755` 命令中使用其中的 `bin/adp` artifact。临时安装目录应放在 project root 之外。

## 环境变量

ADP 可以使用默认值运行，但以下变量对本地 bootstrap 和可复验测试很有用：

- `ADP_HOME`：ADP home 目录，默认 `~/.adp`。Workspace 配置位于 `$ADP_HOME/workspaces`，本地事件日志位于 `$ADP_HOME/logs`。
- `ADP_RUNTIME_DIR`：临时 runtime overlay 的父目录，默认是系统临时目录下的 `adp-runtime`。不要把它指向文件系统根目录、project root、project root 内部目录，或包含 project root 的父目录。优先使用直接的本地目录；symlink runtime parent 会被 `adp doctor` 和 `adp workspace doctor` 作为 warning 报告。
- `ADP_WORKSPACE`：接受 workspace 的命令所使用的默认 workspace。`adp run` 会先使用 `--workspace`，其次使用 `ADP_WORKSPACE`，最后在当前目录位于已注册 project root 内时尝试匹配。

Runtime overlay 会排除仓库 Git metadata。`.gitignore`、`.gitattributes`、`.gitmodules` 等普通项目 Git 文件仍然是 project files，但 `$ADP_RUNTIME_ROOT` 不是权威 Git worktree。Runtime 中 symlinked subpaths 仍可能映射到真实 project files，因此在这些 link 下编辑文件会影响真实项目，即使 overlay 中没有 `.git` metadata。ADP runtime 会在启动 Agent 前 neutralize 指向仓库的 Git environment variables，包括 `GIT_DIR`、`GIT_WORK_TREE`、`GIT_INDEX_FILE`、`GIT_OBJECT_DIRECTORY`、`GIT_ALTERNATE_OBJECT_DIRECTORIES`、`GIT_COMMON_DIR` 和 `GIT_NAMESPACE`。普通 shell environment 和 auth-related variables 会被保留。Runtime manifest 会在可用时记录 `git_root` 和 `git_metadata_skipped: true`；runtime environment 会在 ADP 发现 Git worktree root 时暴露 `ADP_GIT_ROOT`，并为 runtime root 设置 `GIT_CEILING_DIRECTORIES`。Git 命令应从真实 project root 执行，可以使用 `git -C "$ADP_PROJECT_ROOT" ...`，也可以先 `cd "$ADP_PROJECT_ROOT"`。ADP 可能为了 diagnostics 运行只读 Git inspection，但不会 wrap 或自动运行 Git mutation。

Shell helpers 遵循同一边界。`adp env <workspace> --cd` 和 shell-hook 输出可能会在 export ADP runtime environment 前，为危险的仓库指向 Git variables 生成 `unset` 命令。

如果需要隔离验证，可以使用临时目录：

```bash
export ADP_HOME="$(mktemp -d)"
export ADP_RUNTIME_DIR="$(mktemp -d)"
```

## Shell Completion

ADP 会在本地为 bash 和 zsh 渲染 shell completion。生成的脚本会补全 command、subcommand、option、有限 option 值和只读本地候选。Completion 脚本里的 command name 必须能在 operator shell 中解析，因为动态候选会通过运行 `adp completion values ...` 获取。

临时 bash session：

```bash
source <(adp completion --shell bash)
```

临时 zsh session：

```bash
autoload -Uz compinit
compinit
source <(adp completion --shell zsh)
```

如果二进制以其他命令名安装，需要为该名称渲染脚本：

```bash
source <(adp-dev completion --shell bash --command adp-dev)
```

持久配置时，把渲染出的脚本放到 operator shell 会加载的 completion 目录，或在 `adp` 已经进入 `PATH` 后从 shell startup file 中 source。Completion 保持 terminal-first 和 local-first：动态候选只读取 `$ADP_HOME` 下的本地 adapter、workspace、profile、planning 和 session 状态，包括 task ID、phase ID、session ID、task status 和 owner。Completion 不得运行 Agent、调用 provider CLI、运行 Git、创建 runtime overlay、写入 project-root 文件，或修改 task 和 phase 状态。

## 隔离首次运行演练

使用上面任一安装路径后，在选定的 `adp` 命令可用的 shell 中运行一次不依赖 provider 的演练。如果你构建的是 `./bin/adp`，而不是把 `adp` 安装到 `PATH`，则把下面命令块中的 `adp` 替换为 `./bin/adp`。如需同一流程的逐步预期结果和失败排查，请使用 [operator-onboarding.zh-CN.md](operator-onboarding.zh-CN.md)。

```bash
command -v adp
adp version

ADP_REHEARSAL_ROOT="$(mktemp -d)"
export ADP_HOME="${ADP_REHEARSAL_ROOT}/adp-home"
export ADP_RUNTIME_DIR="${ADP_REHEARSAL_ROOT}/runtime"
mkdir -p "${ADP_REHEARSAL_ROOT}/project" "${ADP_REHEARSAL_ROOT}/fake-bin"
printf 'module example.com/adp-rehearsal\n' > "${ADP_REHEARSAL_ROOT}/project/go.mod"
printf 'package main\n' > "${ADP_REHEARSAL_ROOT}/project/main.go"

cat > "${ADP_REHEARSAL_ROOT}/fake-bin/codex" <<'SH'
#!/usr/bin/env sh
printf 'fake codex cwd=%s args=%s\n' "$(pwd)" "$*"
test -n "${ADP_SESSION_ID:-}"
test -n "${ADP_RUNTIME_ROOT:-}"
test -n "${ADP_TASK_ID:-}"
test "$(pwd)" = "$ADP_RUNTIME_ROOT"
test -f "$ADP_RUNTIME_ROOT/AGENTS.md"
SH
chmod +x "${ADP_REHEARSAL_ROOT}/fake-bin/codex"
export PATH="${ADP_REHEARSAL_ROOT}/fake-bin:${PATH}"

adp init
adp workspace add game-a "${ADP_REHEARSAL_ROOT}/project"
adp workspace list
adp workspace show game-a
adp workspace doctor game-a
adp doctor game-a
TASK_ID=$(adp tasks add --workspace game-a --priority high "Validate isolated first run" | sed -n 's/^task \(task-[^ ]*\) added$/\1/p')
test -n "$TASK_ID"
adp tasks next --workspace game-a --format json
adp run codex --workspace game-a --take --owner first-agent --lease 30m -- --example-smoke
adp tasks renew --workspace game-a "$TASK_ID" --owner first-agent --lease 30m
adp tasks stale --workspace game-a --format json
adp progress report --workspace game-a --format json
SESSION_ID=$(adp sessions list --workspace game-a --agent codex --task "$TASK_ID" | sed -n '2s/ .*//p')
test -n "$SESSION_ID"
adp sessions restore-plan "$SESSION_ID"
adp plan doctor --workspace game-a --format json
adp events list --workspace game-a --task "$TASK_ID" --limit 5
adp sessions list --workspace game-a --agent codex --task "$TASK_ID"
adp progress --workspace game-a --format json
ROOT_LEAKS="$(find "${ADP_REHEARSAL_ROOT}/project" -maxdepth 2 \( -name AGENTS.md -o -name CLAUDE.md -o -name .codex -o -name .claude -o -name .adp-runtime.yaml -o -name planning -o -name tasks.yaml -o -name phases.yaml -o -name progress.jsonl \) -print)"
test -z "$ROOT_LEAKS"
```

预期结果：fake provider 会打印 runtime 工作目录，JSON 命令会打印可解析的本地状态，最后的 project-root 泄漏检查通过且没有输出。这次演练会把 ADP 状态放在临时 `$ADP_HOME` 下，把 runtime overlay 放在临时 `$ADP_RUNTIME_DIR` 下，使用 fake local `codex`，不会运行 Git，也不会把 planning 或 report export 写入 project root。可见的任务流是：用 `tasks next` 做只读选择，用 `run --take --owner --lease` 在启动时原子领取任务，用 `tasks renew` 和 `tasks stale` 维护 lease，用 `progress report` 交接上下文，用 `sessions restore-plan` 获取只读重启建议，并用 `plan doctor` 做本地 ledger diagnostics。

## Bootstrap Workspace

初始化 ADP：

```bash
adp init
```

注册一个本地项目。Project root 必须是绝对路径：

```bash
adp workspace add game-a /absolute/path/to/project
```

查看并验证 workspace：

```bash
adp workspace list
adp workspace show game-a
adp workspace doctor game-a
```

`adp workspace doctor` 会检查本地配置、project root 可访问性、runtime parent 安全性、prompt、memory、MCP、profile 文件引用、agent command 设置、project root 中的保留路径、继承的仓库指向 Git environment variables，以及只读 Git topology/status。它会把 adapter default command fallback、写在 command 字段里的 inline arguments、缺失或不可执行的路径型 command wrapper、缺失、重复或逃逸到 workspace 外部的非 default profile、不可用或 nested Git worktree、linked worktree 或 submodule 常见的 gitfile metadata、Git status inspection 不可用，以及 dirty Git status 报告为本地 diagnostics。运行真实 Agent 前应先修复 doctor error；warning-only command/profile/Git diagnostics 不能证明或否定真实 provider CLI 的认证、网络访问、模型可用性，或某个 Git mutation 是否安全。

默认 text 输出会隐藏 info-level diagnostics；当剩余的 diagnostics 只有 info 时，会输出 `ok - no issues`。使用 `adp workspace doctor game-a --verbose` 可以显示 Git topology 等本地 info diagnostics；当脚本、子 Agent 或外部工具需要完整机器可读报告时，使用 `adp workspace doctor game-a --format json`。Git inspection 实际运行时，JSON report 会包含只读 `git` object，提供 project root、Git root、Git directory、metadata kind、nested-workspace flag、relative project path、branch/upstream delta、change state，以及 changed/untracked counts。这些模式仍然是只读的：不会 stage、checkout、commit、push、fetch、clean，也不会推断 release evidence。

## 进入或运行 Runtime

为一个保留的 runtime overlay 渲染 shell exports：

```bash
adp env game-a --cd
```

通过 ADP 运行 Agent：

```bash
adp run codex --workspace game-a -- <agent-args>
adp run claude --workspace game-a -- <agent-args>
```

当 Agent 应在启动时领取下一项符合条件的 ADP task，可以把领取和 runtime 创建放在同一个命令里：

```bash
adp run codex --workspace game-a --take --owner codex-main --lease 4h -- <agent-args>
adp run claude --workspace game-a --take --owner claude-main --lease 4h -- <agent-args>
```

这些命令需要已安装并完成认证的对应外部 CLI。`--` 之后的参数会透传给外部 agent 命令。ADP 不定义某个外部 CLI 支持或安全的具体参数；这需要在 operator 机器上用已安装的 CLI 验证。默认不依赖 provider 的验证路径使用上面的隔离演练。当你需要一份包含 Codex 和 Claude profile、base prompt、shared memory 与 MCP 设置的可复制 workspace 配置时，再使用 `examples/basic-workspace` 和 `scripts/example-workspace-smoke.sh`。

查看本地历史：

```bash
adp events list --workspace game-a
adp sessions list --workspace game-a
adp sessions show <session-id>
adp sessions restore-plan <session-id>
```

`sessions restore-plan` 会在非敏感 invocation 数据足够时，为历史 session 打印只读的建议 `adp run ...` 命令。它不会执行命令、启动 Agent、追加 events、修改 task 状态、写入真实项目根目录或恢复 provider 原生会话。

清理旧的 ADP-owned runtime 目录：

```bash
adp runtime prune --older-than 24h --dry-run
adp runtime prune --older-than 24h
```

`runtime prune` 只删除包含当前版本 ADP runtime manifest，且 `runtime_root` 与待删除目录一致的目录。不兼容、格式错误、外部系统生成或自相矛盾的 manifest 会被跳过。删除前先使用 `--dry-run`。

## 确定性 Bootstrap Smoke

在仓库根目录运行聚合验证 gate：

```bash
scripts/check-all.sh
```

聚合 gate 包含 fake-agent runtime smoke、广覆盖 runtime audit smoke、聚焦 runtime context smoke、release readiness smoke、release rehearsal smoke、release artifact smoke、release operator drill smoke、install onboarding smoke、示例 workspace smoke、task manager smoke、plan intake smoke、Go tests、vet、文件行数检查、双语文档检查和 diff 空白检查。

如需定向 bootstrap 检查，运行：

```bash
scripts/runtime-smoke.sh --fake
scripts/runtime-audit-smoke.sh
scripts/runtime-context-smoke.sh
scripts/release-readiness-smoke.sh
scripts/release-rehearsal-smoke.sh
scripts/release-artifact-smoke.sh
scripts/release-operator-drill-smoke.sh
scripts/install-onboarding-smoke.sh
scripts/example-workspace-smoke.sh
scripts/task-manager-smoke.sh
scripts/plan-intake-smoke.sh
```

runtime smoke 会把当前 `cmd/adp` 二进制构建到临时目录，并使用临时 `ADP_HOME`、`ADP_RUNTIME_DIR`、fake agent binary 和临时 project root。它可以在不安装真实 Codex 或 Claude CLI 的情况下验证 runtime overlay 路径。runtime audit smoke 会扩大覆盖面，验证 CLI help、JSON output、task/phase/plan/progress flow、session、restore planning、completion values 和 local-first runtime 边界。runtime context smoke 聚焦 fake agents 收到的精确 launch context：generated instruction files、adapter metadata、selected profiles、prompt、shared memory、MCP references、task metadata、runtime environment variables、本地 evidence、diagnostics 和 project-root cleanliness。release readiness smoke 会验证 release gate invariant，例如 phase commit 和 push 只记录 evidence 而不会执行 Git。release rehearsal smoke 会把当前未被 ignored 的仓库文件复制到临时干净 workspace，使用 release ldflags 构建 preview binary，验证复制后的文档和文件行数，bootstrap 复制后的 example workspace，并通过 fake Git tripwire 检查 phase evidence recording。release artifact smoke 会验证 package contents、checksum、install-from-artifact、无 `.git` source archive 构建和 local-only 排除边界。release operator drill smoke 会按 operator 顺序演练 clean source form、checksum、临时安装、fake-provider handoff 和本地 phase evidence 记录。install onboarding smoke 会验证源码、构建、临时安装 onboarding、fake-provider 首次运行、event 和 session evidence、project-root 干净状态，以及 Git 副作用 guard。example workspace smoke 会把 `examples/basic-workspace` 复制到临时 `ADP_HOME`，并验证发布的示例仍能针对临时项目完成 bootstrap。plan intake smoke 会验证结构化本地 planning 输入可以只读 preview，并且只能显式 apply 到 `$ADP_HOME`，不会产生 project-root、runtime、Git 或 partial-write 副作用。
