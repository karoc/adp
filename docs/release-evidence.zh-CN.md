# 发布证据

English: [release-evidence.md](release-evidence.md)

本文档模板记录发布 ADP release artifact 前需要保留的本地 evidence。它是 release note companion，不是 hosted release system、cloud ledger、SaaS workflow、provider credential check，也不能替代本地 release gate。ADP release evidence 默认必须保持 terminal-first、local-first 和 provider-neutral。

## 必填字段

每个 release artifact 都应记录这些字段：

- Release version，例如 `1.0.1`。
- 已创建时的 Git tag，例如 `v1.0.1`。
- 构建使用的 Git commit hash 或稳定的 source archive identifier。
- Source form，例如 Git checkout 或 source archive。
- UTC build date。
- Go version。
- 目标 operating system 和 architecture。
- Artifact filename，例如 `adp-1.0.1-linux-amd64`。
- Artifact SHA-256 checksum，以及使用的 checksum command。
- Packaged binary 的 `adp version` output。
- 来自被发布 source form 的 `scripts/check-all.sh` result。
- Package contents manifest，可以是 attached manifest path，也可以是简短 inline excerpt。
- Install-from-artifact rehearsal result。
- 适用时的 source archive 或 no-`.git` rehearsal result。
- 明确列出被排除的 local state、credentials、logs、runtime overlays、machine-specific files 和 local ADP planning ledgers。
- 最终 passing run 之前任何 required check 失败时的 failure triage notes。
- 可选 real-agent operator evidence；如果有意启用，应按 command availability、非交互 invocation、手工交互式 acceptance 分开记录。
- License notice：ADP 以 source-available 形式提供给非商业学习、研究、评估和开源协作用途；商业使用必须取得单独的付费授权。

## 证据分层

Required evidence 是 provider-free 的，并且每个 release candidate 都必须具备：

- `scripts/check-all.sh` 已从用于构建或发布 artifact 的 source form 通过。
- Release binary 报告明确的 version、commit、build date、Go version 和 platform 值。
- Artifact checksum generation 和 verification 已通过。
- Package manifest inspection 已通过，并且没有被排除的 local state。
- Install-from-artifact rehearsal 已使用临时 ADP directories、临时 project root、fake agent commands 通过，并且没有 project-root pollution。
- 当 source archive 或 no-`.git` source form 会被发布或用于构建时，对应 rehearsal 已通过。

Optional real-agent evidence 是补充证据。每个未被有意启用的 optional tier 都应记录为 `not run`。如果 required evidence 完整且所有 optional tiers 都记录为 `not run`，只要 release note 没有声明 deterministic fake-agent gate 之外的 real-agent 行为，该 release candidate 仍然完整。

## 构建证据

Release note 应包含准确的 build identity：

```bash
go version
dist/adp-1.0.1-linux-amd64 version
cd dist
sha256sum -c SHA256SUMS
cd ..
```

期望的 `adp version` release 输出形态：

```text
adp version 1.0.1
commit: <commit>
built: <timestamp>
go: <go-version>
platform: <goos>/<goarch>
```

`commit:` 值必须匹配 evidence 中记录的 Git commit 或稳定 source archive identifier。`built:` 值必须匹配该 artifact 记录的 UTC build date。缺少 `commit:` 或 `built:` 的 development build 对本地开发有用，但不足以作为 release artifact evidence。

如果 source archive 不包含 `.git`，应记录 build 前使用的显式 commit 或 archive identifier：

```bash
COMMIT=source-archive-commit
```

当 source archive 是被发布的 source form 时，不要从无关的本地 checkout 推断 build identity。

## 安装演练证据

记录至少一个 binary 已从 artifact path 安装并运行：

```bash
ADP_INSTALL_BIN="$(mktemp -d)"
install -m 0755 dist/adp-1.0.1-linux-amd64 "${ADP_INSTALL_BIN}/adp"
export PATH="${ADP_INSTALL_BIN}:${PATH}"
adp version
```

安装演练应使用临时 `ADP_HOME`、临时 `ADP_RUNTIME_DIR`、临时 project root 和 fake local agent commands。它应证明 installed binary 可以在没有真实 provider 凭据的情况下运行 local-first workflow：

```bash
export ADP_HOME="${ADP_SMOKE_ROOT}/adp-home"
export ADP_RUNTIME_DIR="${ADP_SMOKE_ROOT}/runtime"
adp init
adp workspace add artifact-a "${ADP_SMOKE_ROOT}/project"
adp workspace doctor artifact-a
TASK_ID=$(adp tasks add --workspace artifact-a --priority high "Validate artifact install" | sed -n 's/^task \(task-[^ ]*\) added$/\1/p')
test -n "$TASK_ID"
adp run codex --workspace artifact-a --task "$TASK_ID" -- --artifact-smoke
adp events list --workspace artifact-a --task "$TASK_ID" --limit 1
adp sessions list --workspace artifact-a --agent codex --task "$TASK_ID"
```

Project-root pollution scan 应找不到任何 ADP-generated files：

```bash
ROOT_LEAKS="$(find "${ADP_SMOKE_ROOT}/project" -maxdepth 2 \( -name AGENTS.md -o -name CLAUDE.md -o -name .codex -o -name .claude -o -name .adp-runtime.yaml -o -name planning -o -name tasks.yaml -o -name phases.yaml -o -name progress.jsonl \) -print)"
test -z "$ROOT_LEAKS"
```

## Package 内容证据

记录每个 package 包含的文件。Release package 应包含一个目标平台的 `adp` binary、`README.md`、`README.zh-CN.md`、`LICENSE`、`COMMERCIAL.md`、`COMMERCIAL.zh-CN.md`、`CONTRIBUTING.md`、`CONTRIBUTING.zh-CN.md`、`SECURITY.md`、`SECURITY.zh-CN.md`、`docs/license-policy.md`、`docs/license-policy.zh-CN.md`、`docs/release-packaging.md`、`docs/release-packaging.zh-CN.md`、`docs/release-evidence.md`、`docs/release-evidence.zh-CN.md`，以及一份 release evidence note 或简短 release note。

同时记录 package 排除了 `.envrc`、`mvp.md`、`$ADP_HOME`、`$ADP_RUNTIME_DIR`、runtime overlays、logs、task state、credentials、provider-native session state、machine-specific shell startup files、local ADP planning ledgers 和临时 release rehearsal directories。

使用排序后的 archive listing 或等价 package tool output 作为 manifest：

```bash
tar -tf adp-1.0.1-linux-amd64.tar.gz | sort
```

如果 manifest 包含本地状态或缺少 required notices，应把该 release 归类为 failed，直到重新构建的 package 通过 manifest inspection 和 checksum verification。

## 失败或延期检查

Required gate failures 必须记录为 failed operator evidence，并且必须停止该 release candidate。修复后，应先重新运行失败 command 和 aggregate gate，再把失败记录替换为 passing evidence。使用 [release-troubleshooting.zh-CN.md](release-troubleshooting.zh-CN.md) 对 build、checksum、manifest、install、source archive 和 environment failures 进行分类。

如果 real-agent evidence 没有被有意启用，可以按 tier 记录为 `not run`。只有当 release note 声明了 deterministic fake gate 之外的对应 real-agent compatibility tier 时，失败的 optional real-agent check 才会阻塞 release。

## 可选 Real-Agent 证据

真实 Codex 和 Claude 检查仍然是独立、opt-in 的 operator evidence。Provider credentials、quota、model access、network behavior 和外部 CLI versions 都属于 operator environment concerns，不是 ADP quality guarantees。`scripts/check-all.sh` 必须保持 provider-free。

可选 evidence 应按不同 tier 记录：

- Command availability evidence 使用 runtime smoke 的真实 flag。它检查外部命令可用，并且可以完成轻量 `--version` 或 `--help` probe；它不会调用模型。

```bash
ADP_SMOKE_REAL_CODEX=1 scripts/runtime-smoke.sh --real-codex
ADP_SMOKE_REAL_CLAUDE=1 scripts/runtime-smoke.sh --real-claude
```

- 非交互真实模型 invocation evidence 使用专用 invocation smoke。它可能联系外部 provider 并消耗 quota。它不属于 `scripts/check-all.sh`，也不得变成默认 CI 或 release gate。

```bash
ADP_REAL_INVOKE_CODEX=1 scripts/real-agent-invocation-smoke.sh --codex
ADP_REAL_INVOKE_CLAUDE=1 scripts/real-agent-invocation-smoke.sh --claude
ADP_REAL_INVOKE_CODEX=1 ADP_REAL_INVOKE_CLAUDE=1 scripts/real-agent-invocation-smoke.sh --all
```

- 手工交互式 provider acceptance 是真实 `adp run ...` session 的独立 operator note。只有 release claims 涉及交互式 provider 行为时才需要它，并且 note 不能包含凭据、token、账号标识、私有 prompt 或敏感模型输出。

某个 tier 未运行时，应为该 tier 记录 `not run`，而不是把 release evidence 视为不完整。完整流程和脱敏要求见 [real-agent-compatibility.zh-CN.md](real-agent-compatibility.zh-CN.md)。

不要在 release evidence 中包含原始 credentials、账号标识、私有 prompts、provider-native session state 或敏感 model output。Summary 应说明 command、environment tier、result，以及 troubleshooting 所需的任何已脱敏 failure class。
