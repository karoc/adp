# Comparable Tools Boundary Note

Simplified Chinese: [comparable-tools.zh-CN.md](comparable-tools.zh-CN.md)

This note keeps ADP polishing grounded in adjacent tools without expanding
the MVP scope. It records product-shape signals only; it is not a feature
matrix, ranking, or competitive critique.

Sources reviewed on 2026-06-09:

- [Claude Code CLI reference](https://code.claude.com/docs/en/cli-usage)
- [Claude Code settings](https://code.claude.com/docs/en/settings)
- [OpenAI Codex CLI](https://developers.openai.com/codex/cli)
- [OpenAI Codex config basics](https://developers.openai.com/codex/config-basic)
- [OpenAI Codex rules](https://developers.openai.com/codex/rules)
- [OpenAI Codex MCP](https://developers.openai.com/codex/mcp)
- [aider repository map](https://aider.chat/docs/repomap.html)
- [aider in-chat commands](https://aider.chat/docs/usage/commands.html)
- [aider Git integration](https://aider.chat/docs/git.html)
- [Continue run checks locally](https://docs.continue.dev/checks/running-locally)
- [Continue run checks in CI](https://docs.continue.dev/checks/running-in-ci)
- [just manual](https://just.systems/man/en/)

## P35 Calibration

P35 should focus on ADP's local runtime context and configuration audit rather
than expanding the product surface. Comparable tools point to context quality,
configuration visibility, and repeatable checks as the narrow calibration area:

- Codex and Claude Code both emphasize terminal agent context and controls:
  durable instruction files, rules, settings or config layers, permission
  controls, and MCP configuration.
- Aider's repository map shows that context quality matters. A concise,
  relevant view of repository structure helps the model understand symbols,
  dependencies, and surrounding code before edits.
- Continue checks show that versioned agent checks can run locally before CI
  and then run again as PR status checks. ADP's equivalent pressure should stay
  with local, deterministic runtime audit and release evidence.

For ADP, the implication is not to clone those tools. It is to verify what
context and configuration enter the runtime overlay, how adapter assumptions are
bounded, and whether default gates remain local, fake-runtime, and auditable.

## Boundary Signals

- Claude Code, Codex CLI, aider, and Continue CLI all treat the terminal as
  a first-class coding surface. ADP should keep polishing terminal entry
  points, command discoverability, diagnostics, and deterministic local
  release gates.
- Adjacent agent CLIs expose provider authentication, model selection,
  permissions, session resume, or background/headless automation in their own
  product surfaces. ADP should integrate with external agent CLIs through
  adapters and runtime overlays, not reimplement provider-native account,
  model, approval, or session semantics.
- Some adjacent tools also expose IDE, web, cloud, platform, or remote-control
  surfaces. ADP should not absorb these during MVP polishing. The current
  product remains terminal-first, local-first, and operator-driven.
- Git behavior differs across tools. Aider documents automatic Git commits as
  part of its workflow, while ADP's phase `commit` and `push` commands record
  operator evidence only. ADP should keep Git execution manual and explicit.
- Continue and Claude Code document tool permissions, permission modes, and
  configuration scopes. ADP should keep default tests fake-runtime and
  network-free, and treat real provider CLI checks as opt-in operator evidence.
- Local task runners such as `just` are useful calibration for predictable
  project commands, but they are not agent orchestrators. ADP should keep
  task and progress management focused on agent workspaces, phases, leases,
  evidence, and runtime context rather than becoming a general command runner.

## ADP Should Keep Doing

- Keep ADP-generated state under `$ADP_HOME` and `$ADP_RUNTIME_DIR`.
- Keep real project roots clean unless the operator explicitly edits the
  project files.
- Keep adapters thin and provider-aware, with provider-specific assumptions
  isolated inside adapter behavior and compatibility docs.
- Keep task, phase, and progress records local, explicit, and auditable.
- Keep release validation deterministic by default: fake agents, temporary
  directories, no provider credentials, no network requirement, and no
  automatic Git side effects.
- Keep bilingual documentation and the 700-line code-file cap as active
  polishing constraints.

## ADP Should Not Absorb In MVP Polishing

- Web UI, hosted dashboards, SaaS trackers, hosted orchestration, cloud sync,
  cloud task execution, or remote-control servers.
- IDE plugins, IDE extensions, editor-native chat panels, browser plugins, or
  graphical multi-agent workbenches.
- Provider-native resume/session semantics beyond recording ADP's local
  runtime sessions and restore plans.
- Automatic task or phase closure based on agent output.
- Automatic Git execution, including commit, push, pull, fetch, clone, branch,
  or merge behavior.
- Real provider default gates. Real Codex, Claude, or other provider CLI checks
  must remain explicit opt-in evidence, not the default validation path.
- Provider account management, billing, model registry, quota handling, or
  remote approval policy.

## Polishing Implications

Comparable tools support the current ADP direction: make the local CLI
boringly dependable before adding breadth. The next polishing work should
therefore prioritize local runtime context/configuration audit, operator
drills, smoke coverage, error messages, documentation precision, package
evidence, and adapter boundary checks. It should not use adjacent web, IDE,
hosted, cloud-sync, provider-resume, real-provider-default, or Git-automating
features as a reason to widen the MVP.
