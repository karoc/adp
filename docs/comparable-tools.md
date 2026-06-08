# Comparable Tools Boundary Note

Simplified Chinese: [comparable-tools.zh-CN.md](comparable-tools.zh-CN.md)

This note keeps ADP polishing grounded in adjacent tools without expanding
the MVP scope. It records product-shape signals only; it is not a feature
matrix, ranking, or competitive critique.

Sources reviewed on 2026-06-09:

- [Claude Code CLI reference](https://docs.anthropic.com/en/docs/claude-code/cli-usage)
- [OpenAI Codex CLI getting started](https://help.openai.com/en/articles/11096431)
- [openai/codex](https://github.com/openai/codex)
- [aider in-chat commands](https://aider.chat/docs/usage/commands.html)
- [aider Git integration](https://aider.chat/docs/git.html)
- [aider linting and testing](https://aider.chat/docs/usage/lint-test.html)
- [Continue CLI quickstart](https://docs.continue.dev/cli/quickstart)
- [Continue CLI tool permissions](https://docs.continue.dev/cli/tool-permissions)
- [just manual](https://just.systems/man/en/)

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
- Continue and Claude Code document tool permissions or permission modes.
  ADP should keep default tests fake-runtime and network-free, and treat real
  provider CLI checks as opt-in operator evidence.
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

- Hosted dashboards, SaaS trackers, cloud sync, cloud task execution, or
  remote-control servers.
- IDE extensions, editor-native chat panels, browser plugins, or graphical
  multi-agent workbenches.
- Provider-native resume/session semantics beyond recording ADP's local
  runtime sessions and restore plans.
- Automatic task or phase closure based on agent output.
- Automatic Git commit, push, pull, fetch, clone, branch, or merge behavior.
- Provider account management, billing, model registry, quota handling, or
  remote approval policy.

## Polishing Implications

Comparable tools support the current ADP direction: make the local CLI
boringly dependable before adding breadth. The next polishing work should
therefore prioritize operator drills, smoke coverage, error messages,
documentation precision, package evidence, and adapter boundary checks. It
should not use adjacent web, IDE, hosted, or Git-automating features as a
reason to widen the MVP.
