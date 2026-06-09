# 发布打包

English: [release-packaging.md](release-packaging.md)

本文档定义 ADP early preview 的打包路径。ADP 是 terminal-first、local-first 的 Go CLI，发布 artifact 应与本地 runtime 模型保持一致，不引入 hosted service、dashboard、cloud sync 或 SaaS deployment 假设。

## 发布门禁

准备 artifact 前，在本地和 CI 中运行同一个聚合门禁：

```bash
scripts/check-all.sh
```

该门禁覆盖 fake runtime acceptance、广覆盖 runtime audit smoke、聚焦 runtime context smoke、release readiness smoke、release rehearsal smoke、release artifact smoke、release operator drill smoke、install onboarding smoke、example workspace smoke、task manager smoke、plan intake smoke、Go test 和 vet、文件行数限制、双语文档配对以及 whitespace 检查。CI 有意调用同一个脚本，避免 release evidence 被拆成本地路径和独立的 GitHub Actions 路径。

可选的真实 Codex 或 Claude CLI 检查只作为 operator evidence：

```bash
ADP_SMOKE_REAL_CODEX=1 scripts/runtime-smoke.sh --real-codex
ADP_SMOKE_REAL_CLAUDE=1 scripts/runtime-smoke.sh --real-claude
```

它们不能替代聚合门禁，也不能证明 provider 凭据、模型访问、额度、网络可靠性或交互式 session 质量。

## Operator 演练

preview release rehearsal 使用这条顺序：

1. 从干净 Git checkout 开始，记录 `git status --short --branch` 和 commit hash。如果还要发布没有 `.git` 的 source archive 或用它构建，记录 archive 来源，并在 no-`.git` build rehearsal 前显式设置 `COMMIT`。
2. 从用于生成 artifacts 或 source archive 的干净 checkout 运行 `scripts/check-all.sh`。如果 archive 缺少测试脚本或 Go module 文件，应从该干净 checkout 重新生成 archive，而不是用本机本地文件补洞。
3. 使用明确的 `VERSION`、`COMMIT` 和 `BUILD_DATE` 构建目标平台 artifact。
4. 为将要打包的 artifact 生成并验证 SHA-256 checksum。
5. 从干净 staging directory 组装 package，然后在发布前记录排序后的 package manifest。
6. 至少把一个 packaged binary 安装到临时 `PATH` 目录，并从该 installed path 运行 provider-free first-run rehearsal。
7. 只有在 gate、checksum verification、package manifest inspection、install rehearsal，以及适用的 source archive 或 no-`.git` rehearsal 都通过后，才记录 release evidence。

如果任何 required step 失败，应停止该 release candidate，在 operator notes 中保留失败 command 和 output，并使用 [release-troubleshooting.zh-CN.md](release-troubleshooting.zh-CN.md)。不能通过增加 hosted orchestration、Web UI、cloud sync、automatic Git execution、provider-native resume、project-root planning export，或默认真实 Codex/Claude 要求来修复 release failure。

## 构建 Artifact

early preview binary 应从仓库根目录构建 CLI：

```bash
mkdir -p dist
VERSION=${VERSION:-0.1.0-preview.1}
COMMIT=${COMMIT:-$(git rev-parse --short HEAD)}
BUILD_DATE=${BUILD_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}

LDFLAGS="-s -w"
LDFLAGS="$LDFLAGS -X github.com/karoc/adp/internal/cli.Version=$VERSION"
LDFLAGS="$LDFLAGS -X github.com/karoc/adp/internal/cli.Commit=$COMMIT"
LDFLAGS="$LDFLAGS -X github.com/karoc/adp/internal/cli.BuildDate=$BUILD_DATE"

go build -trimpath -ldflags="$LDFLAGS" -o dist/adp ./cmd/adp
dist/adp version
```

期望的 release 输出形态是 `adp 0.1.0-preview.1 commit <commit> built <utc-timestamp>`。这些 `-X` 值对应 `github.com/karoc/adp/internal/cli` 包中的变量。省略时，`adp version` 会回退到开发构建标识 `dev`；release artifact 应注入全部三个值，方便 operator 把 binary 关联到 Git commit 和构建时间。

默认的 `COMMIT` 命令假设当前是 Git checkout。如果从没有 `.git` 的 source archive 构建，应在运行 build command 前显式设置 `COMMIT`：

```bash
COMMIT=source-archive-commit
```

如果需要跨平台 preview artifact，应显式设置 `GOOS` 和 `GOARCH`，并使用带平台信息的名称：

```bash
GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="$LDFLAGS" -o dist/adp-linux-amd64 ./cmd/adp
GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="$LDFLAGS" -o dist/adp-darwin-arm64 ./cmd/adp
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="$LDFLAGS" -o dist/adp-windows-amd64.exe ./cmd/adp
```

构建 artifact 后，应先生成并验证 checksum，再进行分发：

```bash
sha256sum dist/adp > dist/adp.sha256
sha256sum -c dist/adp.sha256
```

如果 operator 平台没有 `sha256sum`，可以使用等价的 SHA-256 工具，并在 release evidence note 中记录实际命令。

## 从 Artifact 安装

至少应从已安装位置验证一个 packaged binary，而不是直接从源码树运行：

```bash
ADP_INSTALL_BIN="$(mktemp -d)"
install -m 0755 dist/adp "${ADP_INSTALL_BIN}/adp"
export PATH="${ADP_INSTALL_BIN}:${PATH}"
adp version
```

随后使用临时 `ADP_HOME`、临时 `ADP_RUNTIME_DIR`、临时 project root 和 fake local `codex` command 运行一次 provider-free first-run rehearsal。该演练应证明 installed binary 可以初始化 ADP state、注册 workspace、通过 doctor 检查、运行 `adp run codex --workspace <name> --task <task-id> -- <agent-args>`、检查 events 和 sessions，并且不会在真实 project root 中留下 `AGENTS.md`、`CLAUDE.md`、`.codex`、`.claude`、`.adp-runtime.yaml`、`planning`、`tasks.yaml`、`phases.yaml` 或 `progress.jsonl` 等 ADP-generated files。

## Package 内容

每个打包 archive 应包含：

- 单一目标平台的 `adp` binary。
- `README.md`。
- `README.zh-CN.md`。
- `LICENSE`。
- `COMMERCIAL.md`。
- `COMMERCIAL.zh-CN.md`。
- `docs/release-packaging.md`。
- `docs/release-packaging.zh-CN.md`。
- `docs/release-evidence.md`。
- `docs/release-evidence.zh-CN.md`。
- 一份简短 release note，记录 Git commit、目标平台和门禁证据。

每个 package 都必须保留完整的 `LICENSE` 和 `COMMERCIAL.md`。ADP 在公共许可下以 source-available 形式提供给非商业学习、研究、评估和开源协作用途；任何商业使用都必须从版权持有人取得单独的付费授权。

不要包含本地 `.envrc`、`mvp.md`、`$ADP_HOME`、`$ADP_RUNTIME_DIR`、runtime overlay、日志、task state、凭据、机器特定 shell startup files 或临时 release rehearsal directories。

发布前记录 package manifest，例如：

```bash
tar -tf adp-0.1.0-preview.1-linux-amd64.tar.gz | sort > adp-0.1.0-preview.1-linux-amd64.manifest
```

发布前检查该 manifest。manifest mismatch 是 packaging failure，不能通过削弱 repository ignores 或包含本地 operator state 来修复。

## Preview 范围

early preview package 是本地 CLI artifact。用户应把 binary 安装到 `PATH` 中，运行 `adp init`，注册本地 workspace，并把 agent 配置保存在 `$ADP_HOME` 下。

package 不应声明：

- Hosted orchestration。
- Web 或 dashboard management。
- Cloud synchronization。
- Remote issue tracker synchronization。
- Managed Codex 或 Claude provider access。
- 外部 Agent CLI 的 production certification。

## Tagging 说明

仅在 working tree 干净且发布门禁通过后，才使用明确的 preview tag，例如 `v0.1.0-preview.1`。tag 应指向构建 binary artifacts 使用的同一个 commit。

发布 preview 前应记录 [release-evidence.zh-CN.md](release-evidence.zh-CN.md) 中描述的 evidence，包括：

- Commit hash。
- 构建使用的 source 形态，例如 Git checkout 或 source archive。
- 目标平台和架构。
- Go version。
- packaged binary 的 `adp version` 输出。
- Artifact 文件名和 SHA-256 checksum。
- `scripts/check-all.sh` 结果。
- Install-from-artifact rehearsal 结果。
- 适用时的 source archive 或 no-`.git` rehearsal 结果。
- Package manifest path 或 inline manifest excerpt。
- 任何有意收集的可选真实 CLI evidence。
