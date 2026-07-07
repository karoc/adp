# Real-Agent Optional Evidence

Simplified Chinese: [real-agent-optional-evidence.zh-CN.md](real-agent-optional-evidence.zh-CN.md)

This note records optional real-agent operator evidence collected outside the default ADP release gate. It is not a hosted release ledger, provider credential check, model readiness guarantee, or replacement for `scripts/check-all.sh`. ADP's required validation remains provider-free.

## P74 Command Availability Drill

Evidence date:

- UTC: 2026-07-07T00:39:01Z
- Local operator date: 2026-07-07

Scope:

- ADP phase: `P74` real-agent optional evidence drill
- Source checkout at evidence time: `b58231baba7667bb7ee5e272e4e52a9239a1e25d`
- ADP version output from the checkout: `adp version 1.0.1`
- Go version: `go1.25.7 linux/amd64`
- Evidence tier collected: command availability
- Evidence tiers not collected: non-interactive real model invocation and manual interactive provider acceptance

Provider-free safety-gate guidance check:

```bash
scripts/real-agent-invocation-smoke.sh
```

Result:

```text
[real-agent-invocation-smoke] no real provider target selected
[real-agent-invocation-smoke] provider-free guidance check passed
```

This default invocation did not build ADP, resolve external commands, create runtime overlays, contact providers, or consume quota. It only verified the local opt-in guidance path.

Command availability evidence:

```bash
command -v codex
command -v claude
ADP_SMOKE_REAL_CODEX=1 ADP_SMOKE_REAL_CLAUDE=1 scripts/runtime-smoke.sh --real-codex --real-claude
```

Resolved commands:

```text
/home/karoc/.npm-global/bin/codex
/home/karoc/.local/bin/claude
```

Observed command availability output:

```text
[runtime-smoke] real codex CLI responded to --version: codex-cli 0.142.5
[runtime-smoke] real claude CLI responded to --version: 2.1.201 (Claude Code)
[runtime-smoke] runtime smoke acceptance passed
```

The runtime smoke real flags are additive. The command above first ran the deterministic fake runtime smoke, then checked real Codex and Claude command availability through lightweight `--version` probes. It did not invoke a model and does not prove credentials, account state, model access, quota, network behavior, provider availability, external tool permissions, or interactive session quality.

Deferred optional tiers:

- Non-interactive real model invocation: `not run`. The command would require explicit `ADP_REAL_INVOKE_CODEX=1` and/or `ADP_REAL_INVOKE_CLAUDE=1` plus a provider flag, and may contact providers or consume quota.
- Manual interactive provider acceptance: `not run`. No claim is made about interactive Codex or Claude workflow readiness from this evidence.

Residual risk:

- This evidence is specific to one operator environment and the command versions above.
- It supports only the command availability tier.
- It does not change the default provider-free release gate or CI boundary.
- It does not justify adding real-provider checks to `scripts/check-all.sh`.

