# 同类工具边界说明

English: [comparable-tools.md](comparable-tools.md)

本文档用于让 ADP 的打磨工作保持和相邻工具生态对齐，同时不扩大 MVP
范围。它只记录产品形态信号，不是功能矩阵、排名或竞品批评。

以下资料于 2026-06-09 查阅：

- [Claude Code CLI reference](https://docs.anthropic.com/en/docs/claude-code/cli-usage)
- [OpenAI Codex CLI getting started](https://help.openai.com/en/articles/11096431)
- [openai/codex](https://github.com/openai/codex)
- [aider in-chat commands](https://aider.chat/docs/usage/commands.html)
- [aider Git integration](https://aider.chat/docs/git.html)
- [aider linting and testing](https://aider.chat/docs/usage/lint-test.html)
- [Continue CLI quickstart](https://docs.continue.dev/cli/quickstart)
- [Continue CLI tool permissions](https://docs.continue.dev/cli/tool-permissions)
- [just manual](https://just.systems/man/en/)

## 边界信号

- Claude Code、Codex CLI、aider 和 Continue CLI 都把终端作为一等编码
  表面。ADP 应继续打磨 terminal entry points、command discoverability、
  diagnostics 和确定性的本地 release gates。
- 相邻 Agent CLI 会在自身产品表面提供 provider authentication、model
  selection、permissions、session resume 或 background/headless automation。
  ADP 应通过 adapter 和 runtime overlay 集成外部 Agent CLI，而不是重新实现
  provider-native account、model、approval 或 session semantics。
- 一些相邻工具也提供 IDE、web、cloud、platform 或 remote-control 表面。
  ADP 在 MVP 打磨阶段不应吸收这些范围。当前产品仍保持 terminal-first、
  local-first，并由 operator 显式驱动。
- 不同工具的 Git 行为不同。Aider 文档把自动 Git commit 作为其工作流的一部分；
  而 ADP 的阶段 `commit` 和 `push` 命令只记录 operator evidence。ADP 应保持
  Git execution 手工且显式。
- Continue 和 Claude Code 文档中都有 tool permissions 或 permission modes。
  ADP 应保持默认测试 fake-runtime、无网络依赖，并把真实 provider CLI 检查
  作为 opt-in operator evidence。
- `just` 这类本地 task runner 可以作为可预测 project commands 的校准参照，
  但它们不是 Agent orchestrator。ADP 应把 task 和 progress management 继续
  聚焦在 Agent workspace、phase、lease、evidence 和 runtime context，而不是
  演变成通用 command runner。

## ADP 应继续坚持

- 把 ADP 生成状态保存在 `$ADP_HOME` 和 `$ADP_RUNTIME_DIR` 下。
- 保持真实 project root 干净，除非 operator 明确编辑项目文件。
- 保持 adapter 轻量且了解 provider 边界，把 provider-specific 假设隔离在
  adapter behavior 和 compatibility docs 中。
- 保持 task、phase 和 progress records 本地、显式、可审计。
- 保持默认 release validation 确定性：fake agent、临时目录、无 provider
  credentials、无网络要求、无自动 Git side effects。
- 把双语文档和 700 行 code-file cap 继续作为有效打磨约束。

## MVP 打磨阶段不应吸收

- Hosted dashboard、SaaS tracker、cloud sync、cloud task execution 或
  remote-control server。
- IDE extension、editor-native chat panel、browser plugin 或图形化 multi-agent
  workbench。
- Provider-native resume/session semantics；ADP 只记录本地 runtime sessions
  和 restore plans。
- 基于 Agent output 自动关闭 task 或 phase。
- 自动 Git commit、push、pull、fetch、clone、branch 或 merge 行为。
- Provider account management、billing、model registry、quota handling 或
  remote approval policy。

## 打磨含义

同类工具支持 ADP 当前方向：先把本地 CLI 打磨到稳定可靠，再增加广度。因此，
下一轮打磨应优先关注 operator drill、smoke coverage、error messages、
documentation precision、package evidence 和 adapter boundary checks。不能把
相邻工具中的 web、IDE、hosted 或 Git automation 能力作为扩大 MVP 范围的理由。
