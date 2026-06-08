# 安装与 Bootstrap

English: [install.md](install.md)

本文档说明 ADP 的本地安装和首次 bootstrap 流程。ADP 是 terminal-first、local-first 的 runtime manager；安装是本地 CLI 工作流，不需要托管服务。

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

## 环境变量

ADP 可以使用默认值运行，但以下变量对本地 bootstrap 和可复验测试很有用：

- `ADP_HOME`：ADP home 目录，默认 `~/.adp`。Workspace 配置位于 `$ADP_HOME/workspaces`，本地事件日志位于 `$ADP_HOME/logs`。
- `ADP_RUNTIME_DIR`：临时 runtime overlay 的父目录，默认是系统临时目录下的 `adp-runtime`。不要把它指向文件系统根目录、project root、project root 内部目录，或包含 project root 的父目录。优先使用直接的本地目录；symlink runtime parent 会被 `adp doctor` 和 `adp workspace doctor` 作为 warning 报告。
- `ADP_WORKSPACE`：接受 workspace 的命令所使用的默认 workspace。`adp run` 会先使用 `--workspace`，其次使用 `ADP_WORKSPACE`，最后在当前目录位于已注册 project root 内时尝试匹配。

如果需要隔离验证，可以使用临时目录：

```bash
export ADP_HOME="$(mktemp -d)"
export ADP_RUNTIME_DIR="$(mktemp -d)"
```

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

`adp workspace doctor` 会检查本地配置、project root 可访问性、runtime parent 安全性、prompt、memory、MCP、profile 文件引用以及 agent command 设置。运行真实 Agent 前应先修复 doctor error。

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

`--` 之后的参数会透传给外部 agent 命令。ADP 不定义某个外部 CLI 支持或安全的具体参数；这需要在 operator 机器上用已安装的 CLI 验证。

查看本地历史：

```bash
adp events list --workspace game-a
adp sessions list --workspace game-a
adp sessions show <session-id>
```

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

聚合 gate 包含 fake-agent runtime smoke、示例 workspace smoke、task manager smoke、Go tests、vet、文件行数检查、双语文档检查和 diff 空白检查。

如需定向 bootstrap 检查，运行：

```bash
scripts/runtime-smoke.sh --fake
scripts/example-workspace-smoke.sh
scripts/task-manager-smoke.sh
```

runtime smoke 会把当前 `cmd/adp` 二进制构建到临时目录，并使用临时 `ADP_HOME`、`ADP_RUNTIME_DIR`、fake agent binary 和临时 project root。它可以在不安装真实 Codex 或 Claude CLI 的情况下验证 runtime overlay 路径。example workspace smoke 会把 `examples/basic-workspace` 复制到临时 `ADP_HOME`，并验证发布的示例仍能针对临时项目完成 bootstrap。
