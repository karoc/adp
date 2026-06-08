# 发布打包

English: [release-packaging.md](release-packaging.md)

本文档定义 ADP early preview 的打包路径。ADP 是 terminal-first、local-first 的 Go CLI，发布 artifact 应与本地 runtime 模型保持一致，不引入 hosted service、dashboard、cloud sync 或 SaaS deployment 假设。

## 发布门禁

准备 artifact 前，在本地和 CI 中运行同一个聚合门禁：

```bash
scripts/check-all.sh
```

该门禁覆盖 fake runtime acceptance、example workspace smoke、task manager smoke、Go test 和 vet、文件行数限制、双语文档配对以及 whitespace 检查。CI 有意调用同一个脚本，避免 release evidence 被拆成本地路径和独立的 GitHub Actions 路径。

可选的真实 Codex 或 Claude CLI 检查只作为 operator evidence：

```bash
ADP_SMOKE_REAL_CODEX=1 scripts/runtime-smoke.sh --real-codex
ADP_SMOKE_REAL_CLAUDE=1 scripts/runtime-smoke.sh --real-claude
```

它们不能替代聚合门禁，也不能证明 provider 凭据、模型访问、额度、网络可靠性或交互式 session 质量。

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

这些 `-X` 值对应 `github.com/karoc/adp/internal/cli` 包中的变量。省略时，`adp version` 会回退到开发构建标识 `dev`；release artifact 应注入全部三个值，方便 operator 把 binary 关联到 Git commit 和构建时间。

如果需要跨平台 preview artifact，应显式设置 `GOOS` 和 `GOARCH`，并使用带平台信息的名称：

```bash
GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="$LDFLAGS" -o dist/adp-linux-amd64 ./cmd/adp
GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="$LDFLAGS" -o dist/adp-darwin-arm64 ./cmd/adp
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="$LDFLAGS" -o dist/adp-windows-amd64.exe ./cmd/adp
```

每个打包 archive 应包含：

- 单一目标平台的 `adp` binary。
- `README.md`。
- `LICENSE`。
- `COMMERCIAL.md`。
- 一份简短 release note，记录 Git commit、目标平台和门禁证据。

不要包含本地 `.envrc`、`mvp.md`、`$ADP_HOME`、`$ADP_RUNTIME_DIR`、runtime overlay、日志、task state、凭据或机器特定 shell startup files。

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

发布 preview 前应记录：

- Commit hash。
- 目标平台和架构。
- Go version。
- packaged binary 的 `adp version` 输出。
- `scripts/check-all.sh` 结果。
- 任何有意收集的可选真实 CLI evidence。
