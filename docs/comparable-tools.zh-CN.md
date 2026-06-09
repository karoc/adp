# 同类工具边界说明

English: [comparable-tools.md](comparable-tools.md)

本文档用于让 ADP 的打磨工作保持和相邻工具生态对齐，同时不扩大 MVP
范围。它只记录产品形态信号，不是功能矩阵、排名或竞品批评。

以下资料于 2026-06-09 查阅：

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

## P35 校准

P35 应聚焦 ADP 的本地 runtime context 和 configuration audit，而不是扩大产品
表面。同类工具给出的窄校准方向是 context quality、configuration visibility
和可重复 checks：

- Codex 和 Claude Code 都强调终端 Agent context 与控制面：durable instruction
  files、rules、settings 或 config layers、permission controls，以及 MCP
  configuration。
- Aider 的 repository map 说明 context quality 很重要。精简且相关的 repository
  structure 视图可以帮助模型在编辑前理解 symbols、dependencies 和周边代码。
- Continue checks 说明 versioned agent checks 可以先在本地运行，再作为 CI/PR
  status checks 运行。ADP 对应的压力应继续落在本地、确定性的 runtime audit
  和 release evidence 上。

对 ADP 来说，结论不是复制这些工具，而是验证哪些 context 和 configuration 进入
runtime overlay，adapter assumptions 如何被限定，以及默认 gates 是否保持本地、
fake-runtime、可审计。

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
- Continue 和 Claude Code 文档中都有 tool permissions、permission modes 和
  configuration scopes。ADP 应保持默认测试 fake-runtime、无网络依赖，并把真实
  provider CLI 检查作为 opt-in operator evidence。
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

- Web UI、hosted dashboard、SaaS tracker、hosted orchestration、cloud sync、
  cloud task execution 或 remote-control server。
- IDE plugin、IDE extension、editor-native chat panel、browser plugin 或图形化
  multi-agent workbench。
- Provider-native resume/session semantics；ADP 只记录本地 runtime sessions
  和 restore plans。
- 基于 Agent output 自动关闭 task 或 phase。
- 自动 Git execution，包括 commit、push、pull、fetch、clone、branch 或 merge
  行为。
- Real provider default gates。真实 Codex、Claude 或其他 provider CLI 检查必须
  保持显式 opt-in evidence，而不是默认 validation path。
- Provider account management、billing、model registry、quota handling 或
  remote approval policy。

## 打磨含义

同类工具支持 ADP 当前方向：先把本地 CLI 打磨到稳定可靠，再增加广度。因此，
下一轮打磨应优先关注 local runtime context/configuration audit、operator drill、
smoke coverage、error messages、documentation precision、package evidence 和
adapter boundary checks。不能把相邻工具中的 web、IDE、hosted、cloud-sync、
provider-resume、real-provider-default 或 Git automation 能力作为扩大 MVP 范围的
理由。
