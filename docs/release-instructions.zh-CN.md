# 发布指南

English: [release-instructions.md](release-instructions.md)

本文档是发布 ADP stable release artifacts 的 operator procedure。它适用于当前 `1.0.1` patch-prep release，也适用于后续只需调整版本号的 patch releases。ADP 仍然是 terminal-first、local-first 的 CLI；本流程不会引入 hosted orchestration、dashboard、cloud sync、SaaS release tracking、automatic Git execution，或 provider-managed Codex/Claude access。

## 发布范围

当前发布目标是 `1.0.1`。源码默认版本也已经是 `1.0.1`，但 release artifacts 仍必须携带明确的 build identity：

- `VERSION`，通常是 `1.0.1`。
- `COMMIT`，通常是本次发布使用的 Git commit。
- `BUILD_DATE`，通常是 UTC timestamp。

标准构建脚本是 `scripts/build-release.sh`。除非是在排查失败构建，并且随后会回到标准脚本，否则不要用临时 `go build` 命令生成 release artifacts。

## 预检

从将要发布的 source form 开始，通常是一个干净的 Git checkout。构建前记录本地状态：

```bash
git status --short --branch
git rev-parse HEAD
git check-ignore -v .envrc mvp.md || true
```

运行标准聚合门禁：

```bash
scripts/check-all.sh
```

`scripts/check-all.sh` 是 release decision gate。它会预热 Go build cache，默认并行运行 provider-free smoke suite 并输出 timing，然后运行 `scripts/check-coverage.sh`、`go vet ./...`、`scripts/check-file-lines.sh`、`scripts/check-docs-bilingual.sh` 和 `git diff --check`。

如果门禁失败，停止该 candidate，并使用 [release-troubleshooting.zh-CN.md](release-troubleshooting.zh-CN.md)。不要从失败的 source form 创建 tag、发布公告或发布 artifacts。

## 构建 Artifacts

在仓库根目录用明确的 identity values 构建：

```bash
VERSION=1.0.1 \
COMMIT="$(git rev-parse HEAD)" \
BUILD_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
./scripts/build-release.sh
```

该脚本会把平台 binaries 和 checksums 写入 `dist/`：

```text
dist/SHA256SUMS
dist/adp-1.0.1-linux-amd64
dist/adp-1.0.1-linux-arm64
dist/adp-1.0.1-darwin-amd64
dist/adp-1.0.1-darwin-arm64
dist/adp-1.0.1-windows-amd64.exe
```

如果 build source 是不含 `.git` 的 archive，应把 `COMMIT` 显式设置为已发布的 commit hash，或 release evidence 中记录的另一个稳定 archive identifier：

```bash
VERSION=1.0.1 \
COMMIT=<published-commit-or-archive-id> \
BUILD_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
./scripts/build-release.sh
```

## 验证构建 Identity

至少运行一个 packaged binary，并确认它报告 release identity：

```bash
dist/adp-1.0.1-linux-amd64 version
```

预期的 release 输出形态：

```text
adp version 1.0.1
commit: <commit>
built: <timestamp>
go: <go-version>
platform: <goos>/<goarch>
```

如果 artifact 缺少 `commit:` 或 `built:`，说明它没有携带完整 release identity，必须重新构建。不要修改 release evidence 去匹配一个意外 binary。

## 验证 Checksums

打包或上传 artifacts 前，先验证构建脚本生成的 checksum file：

```bash
cd dist
sha256sum -c SHA256SUMS
cd ..
```

如果 operator 平台没有 `sha256sum`，可以使用等价的 SHA-256 verification command，并在 [release-evidence.zh-CN.md](release-evidence.zh-CN.md) 中记录该命令。

## Package 内容

每个 release package 应包含一个目标平台的 `adp` binary，以及必需的 notices 和 release documents。以 [release-packaging.zh-CN.md](release-packaging.zh-CN.md) 中的 packaging contract 为准。当前 required contents 包括：

- `README.md` 和 `README.zh-CN.md`。
- `LICENSE`。
- `COMMERCIAL.md` 和 `COMMERCIAL.zh-CN.md`。
- `CONTRIBUTING.md` 和 `CONTRIBUTING.zh-CN.md`。
- `SECURITY.md` 和 `SECURITY.zh-CN.md`。
- `docs/license-policy.md` 和 `docs/license-policy.zh-CN.md`。
- `docs/release-packaging.md` 和 `docs/release-packaging.zh-CN.md`。
- `docs/release-evidence.md` 和 `docs/release-evidence.zh-CN.md`。
- 一份简短 release evidence note 或 release note，记录 version、commit、target platform、gate result 和 checksum。

Packages 必须排除 `.envrc`、`mvp.md`、`$ADP_HOME`、`$ADP_RUNTIME_DIR`、runtime overlays、logs、task state、credentials、机器特定 shell startup files、临时 release rehearsal directories、provider-native session state 和本地 ADP planning ledgers。

发布前记录排序后的 package manifest，例如：

```bash
tar -tf adp-1.0.1-linux-amd64.tar.gz | sort > adp-1.0.1-linux-amd64.manifest
```

manifest mismatch 是 packaging failure。应从干净 staging directory 重新构建，而不是削弱 local-first package boundary。

## 安装演练

至少把一个 packaged binary 安装到临时 `PATH` 目录，并从该位置运行：

```bash
ADP_INSTALL_BIN="$(mktemp -d)"
install -m 0755 dist/adp-1.0.1-linux-amd64 "${ADP_INSTALL_BIN}/adp"
export PATH="${ADP_INSTALL_BIN}:${PATH}"
adp version
```

随后使用临时 `ADP_HOME`、临时 `ADP_RUNTIME_DIR`、临时 project root 和 fake local agent commands 运行 provider-free first-run rehearsal。该 rehearsal 应证明 installed binary 可以初始化 ADP state、注册 workspace、运行 workspace diagnostics、对 fake `codex` 执行 `adp run codex --workspace <name> --task <task-id> -- <agent-args>`、检查 events 和 sessions，并且不会在真实 project root 中留下 ADP-generated files。

可选真实 Codex 或 Claude 检查是独立的 operator evidence。不要把真实 provider credentials、quota、model access、network availability 或 interactive session behavior 变成默认 release gate。

## Release Evidence

在 gate、build identity check、checksum verification、package manifest inspection、install rehearsal，以及适用的 source archive rehearsal 全部通过后，再记录 release evidence。使用 [release-evidence.zh-CN.md](release-evidence.zh-CN.md)，并把 required provider-free evidence 与 optional real-agent evidence 分开。

Evidence 至少应包含：

- Version `1.0.1`。
- Commit 和 source form。
- UTC build date。
- Go version。
- Target platform 和 artifact filename。
- packaged binary 的 `adp version` output。
- SHA-256 checksum 和 verification command。
- `scripts/check-all.sh` result。
- Package manifest path 或 inline manifest excerpt。
- Install-from-artifact rehearsal result。
- 适用时的 source archive 或 no-`.git` rehearsal result。
- Optional real-agent tiers 记录为 passed、failed、deferred 或 `not run`。

不要在 evidence 或 packages 中包含 credentials、账号标识、私有 prompts、provider-native session files、敏感 model output 或机器本地 ADP ledgers。

## Tag 和发布

只有在 source form 干净、门禁通过、artifacts 已构建并验证、release evidence 完整后，才创建 release tag。Tag 必须指向 artifacts 中 `COMMIT` 使用的同一个 commit。

```bash
git tag -a v1.0.1 -m "ADP 1.0.1"
git push origin v1.0.1
```

只有在 tag 指向 release commit 后，才发布 artifacts。如果使用 GitHub Releases，应上传已检查的 binaries、`dist/SHA256SUMS`、package archives、package manifests，以及 release evidence 或 release note。Operator 权限足够时，可以使用 `gh` CLI：

```bash
gh release create v1.0.1 \
  --title "ADP 1.0.1" \
  --notes-file <release-note-file> \
  dist/adp-1.0.1-linux-amd64 \
  dist/adp-1.0.1-linux-arm64 \
  dist/adp-1.0.1-darwin-amd64 \
  dist/adp-1.0.1-darwin-arm64 \
  dist/adp-1.0.1-windows-amd64.exe \
  dist/SHA256SUMS
```

如果 release creation 因 token permissions 失败，保持 artifacts 不变，刷新或替换 operator token，然后只重试 publishing step。除非 artifacts 或 evidence 有误，否则不要重新构建。

## 发布后验证

发布后，确认 tag、release metadata、artifacts 和 checksums 已公开可见，或对预期 repository audience 可见：

```bash
gh release view v1.0.1
curl -LO https://github.com/karoc/adp/releases/download/v1.0.1/adp-1.0.1-linux-amd64
curl -LO https://github.com/karoc/adp/releases/download/v1.0.1/SHA256SUMS
sha256sum -c SHA256SUMS
```

从临时位置下载并运行至少一个已发布 binary，并把它的 `adp version` output 与 release evidence 对比。如果已发布 artifact 与验证过的本地 artifact 不一致，应移除或替换该 release asset，并在 operator notes 中记录修正。
