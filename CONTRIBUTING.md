# Contributing

Simplified Chinese: [CONTRIBUTING.zh-CN.md](CONTRIBUTING.zh-CN.md)

Thank you for taking time to improve ADP. This project is terminal-first, local-first, and source-available for noncommercial use. Contributions should preserve that product boundary and the repository's PolyForm Noncommercial licensing model.

## Scope

Good contributions improve local CLI workflows, runtime overlays, workspace registry behavior, adapters, shell integration, local event/session evidence, diagnostics, task and phase ledgers, release gates, tests, examples, or bilingual documentation.

Avoid proposals that move ADP toward a Web dashboard, SaaS tracker, cloud sync service, hosted orchestration platform, graphical multi-agent product, automatic Git executor, provider-private state scraper, or provider-native conversation resume implementation.

## License Boundary

ADP is available under the [PolyForm Noncommercial License 1.0.0](LICENSE). Commercial use requires separate paid authorization from the copyright holder. Public availability does not grant commercial rights.

By submitting a contribution, you confirm that you have the right to contribute it and that it can be distributed as part of ADP under the repository's existing licensing model. Do not submit code, docs, generated output, or dependency changes that conflict with the noncommercial source-available distribution model.

For details, see [docs/license-policy.md](docs/license-policy.md) and [COMMERCIAL.md](COMMERCIAL.md).

## Development Rules

Before opening or handing off a change:

- Keep code files at or below 700 physical lines.
- Keep English documentation as the default and add equivalent Simplified Chinese `*.zh-CN.md` counterparts for maintained Markdown files.
- Keep `.envrc` and `mvp.md` ignored and uncommitted.
- Keep ADP runtime and planning state outside real project roots.
- Do not configure repository-local Git `user.name` or `user.email`.
- Preserve provider-free default tests; real Codex and Claude checks must remain explicit opt-in.

Run the standard gate:

```bash
scripts/check-all.sh
```

If a change is narrow, focused local checks are useful during development, but the full gate is still required before commit or handoff.

## Planning And Handoff

ADP development dogfoods ADP's own local task and phase ledger. Register planned implementation slices as ADP phases and tasks, keep execution state under `$ADP_HOME`, and record acceptance, commit evidence, and push evidence before starting the next phase.

When using multiple agents, assign disjoint write scopes, make ADP the authoritative task board, review each returned diff, and run full repository gates after integration.

See [AGENTS.md](AGENTS.md) for the full contributor and agent operating contract.
