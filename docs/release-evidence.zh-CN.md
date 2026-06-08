# 发布证据

English: [release-evidence.md](release-evidence.md)

本文档模板记录发布 ADP preview artifact 前需要保留的本地 evidence。它是 release note companion，不是 hosted release system、cloud ledger、SaaS workflow、provider credential check，也不能替代本地 phase gate。

## 必填字段

每个 preview artifact 都应记录这些字段：

- Release version，例如 `0.1.0-preview.1`。
- 构建使用的 Git commit hash。
- Source form，例如 Git checkout 或 source archive。
- UTC build date。
- Go version。
- 目标 operating system 和 architecture。
- Artifact filename。
- Artifact SHA-256 checksum，以及使用的 checksum command。
- Packaged binary 的 `adp version` output。
- `scripts/check-all.sh` result。
- Install-from-artifact rehearsal result。
- 适用时的 source archive 或 no-`.git` rehearsal result。
- Package contents manifest。
- 明确列出被排除的 local state、credentials、logs 和 machine-specific files。
- 可选真实 Codex 或 Claude CLI evidence，仅在有意启用时记录。
- License notice：ADP 以 source-available 形式提供给非商业学习、研究、评估和开源协作用途；商业使用必须取得单独的付费授权。

## 构建 Evidence

Release note 应包含准确的 build identity：

```bash
go version
dist/adp version
sha256sum dist/adp > dist/adp.sha256
sha256sum -c dist/adp.sha256
```

期望的 `dist/adp version` release 输出形态：

```txt
adp 0.1.0-preview.1 commit <commit> built <utc-timestamp>
```

如果 source archive 不包含 `.git`，应记录 build 前使用的显式 commit 值：

```bash
COMMIT=source-archive-commit
```

输出 `adp dev` 的 development build 对本地开发有用，但不足以作为 preview artifact evidence。

## 安装演练 Evidence

记录至少一个 binary 已从 artifact path 安装并运行：

```bash
ADP_INSTALL_BIN="$(mktemp -d)"
install -m 0755 dist/adp "${ADP_INSTALL_BIN}/adp"
export PATH="${ADP_INSTALL_BIN}:${PATH}"
adp version
```

安装演练应使用临时 `ADP_HOME`、临时 `ADP_RUNTIME_DIR`、临时 project root 和 fake local `codex` command。它应证明 installed binary 可以在没有真实 provider 凭据的情况下运行 local-first workflow：

```bash
export ADP_HOME="${ADP_SMOKE_ROOT}/adp-home"
export ADP_RUNTIME_DIR="${ADP_SMOKE_ROOT}/runtime"
adp init
adp workspace add artifact-a "${ADP_SMOKE_ROOT}/project"
adp workspace doctor artifact-a
TASK_ID=$(adp tasks add --workspace artifact-a --priority high --phase artifact-smoke "Validate artifact install" | sed -n 's/^task \(task-[^ ]*\) added$/\1/p')
adp run codex --workspace artifact-a --task "$TASK_ID" -- --artifact-smoke
adp events list --workspace artifact-a --task "$TASK_ID" --limit 1
adp sessions list --workspace artifact-a --agent codex --task "$TASK_ID"
```

Project-root pollution scan 应找不到任何 ADP-generated files：

```bash
find "${ADP_SMOKE_ROOT}/project" -maxdepth 2 \( -name AGENTS.md -o -name CLAUDE.md -o -name .codex -o -name .claude -o -name planning \)
```

## Package 内容 Evidence

记录每个 package 包含的文件。Preview package 应包含一个目标平台的 `adp` binary、`README.md`、`README.zh-CN.md`、`LICENSE`、`COMMERCIAL.md`、`COMMERCIAL.zh-CN.md`、`docs/release-packaging.md`、`docs/release-packaging.zh-CN.md`、`docs/release-evidence.md`、`docs/release-evidence.zh-CN.md`，以及一份简短 release note。

同时记录 package 排除了 `.envrc`、`mvp.md`、`$ADP_HOME`、`$ADP_RUNTIME_DIR`、runtime overlays、logs、task state、credentials、machine-specific shell startup files 和临时 release rehearsal directories。

## 可选真实 CLI Evidence

真实 Codex 和 Claude 检查仍然是独立、opt-in 的 operator evidence。由于本地 credentials、provider access、quotas、network behavior 和外部 CLI versions 会随 operator environment 变化，它们不能成为默认 release gates。

只有在有意运行时，才记录这些命令：

```bash
ADP_SMOKE_REAL_CODEX=1 scripts/runtime-smoke.sh --real-codex
ADP_SMOKE_REAL_CLAUDE=1 scripts/runtime-smoke.sh --real-claude
```

未运行时，应记录 `not run`，而不是把 release evidence 视为不完整。
