# 可选 Real-Agent 证据

English: [real-agent-optional-evidence.md](real-agent-optional-evidence.md)

本文档记录默认 ADP release gate 之外收集的可选 real-agent operator evidence。它不是 hosted release ledger、provider credential check、model readiness guarantee，也不能替代 `scripts/check-all.sh`。ADP 的必跑验证仍然保持 provider-free。

## P74 命令可用性演练

证据日期：

- UTC：2026-07-07T00:39:01Z
- 本地 operator 日期：2026-07-07

范围：

- ADP phase：`P74` real-agent optional evidence drill
- 收集 evidence 时的 source checkout：`b58231baba7667bb7ee5e272e4e52a9239a1e25d`
- 该 checkout 的 ADP version 输出：`adp version 1.0.1`
- Go version：`go1.25.7 linux/amd64`
- 已收集的 evidence tier：command availability
- 未收集的 evidence tiers：非交互真实模型 invocation 和手工交互式 provider acceptance

Provider-free safety-gate guidance check：

```bash
scripts/real-agent-invocation-smoke.sh
```

结果：

```text
[real-agent-invocation-smoke] no real provider target selected
[real-agent-invocation-smoke] provider-free guidance check passed
```

这次默认 invocation 没有构建 ADP、解析外部命令、创建 runtime overlay、访问 provider 或消耗 quota。它只验证了本地 opt-in guidance path。

Command availability evidence：

```bash
command -v codex
command -v claude
ADP_SMOKE_REAL_CODEX=1 ADP_SMOKE_REAL_CLAUDE=1 scripts/runtime-smoke.sh --real-codex --real-claude
```

解析到的命令：

```text
/home/karoc/.npm-global/bin/codex
/home/karoc/.local/bin/claude
```

观察到的 command availability 输出：

```text
[runtime-smoke] real codex CLI responded to --version: codex-cli 0.142.5
[runtime-smoke] real claude CLI responded to --version: 2.1.201 (Claude Code)
[runtime-smoke] runtime smoke acceptance passed
```

Runtime smoke 的真实 flag 是 additive。上述命令先运行 deterministic fake runtime smoke，然后通过轻量 `--version` probe 检查真实 Codex 和 Claude command availability。它没有调用模型，也不能证明 credentials、account state、model access、quota、network behavior、provider availability、外部工具权限或交互式 session 质量。

延期的可选 tiers：

- 非交互真实模型 invocation：`not run`。该命令需要显式设置 `ADP_REAL_INVOKE_CODEX=1` 和/或 `ADP_REAL_INVOKE_CLAUDE=1`，并传入 provider flag，且可能访问 provider 或消耗 quota。
- 手工交互式 provider acceptance：`not run`。本 evidence 不声明交互式 Codex 或 Claude workflow readiness。

剩余风险：

- 该 evidence 只适用于一个 operator environment 和上述命令版本。
- 它只支撑 command availability tier。
- 它不改变默认 provider-free release gate 或 CI 边界。
- 它不能作为把真实 provider 检查加入 `scripts/check-all.sh` 的依据。

