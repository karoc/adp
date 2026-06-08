# 运行时审计

本文档记录打磨阶段的运行时审计标准。它只面向 ADP 已有行为；
同类工具只作为校准参照，不扩大 ADP 的产品范围。

## 同类工具信号

以下资料于 2026-06-09 查阅：

- [Claude Code CLI reference](https://code.claude.com/docs/en/cli-usage)：
  强调终端入口、命令和参数可发现性、认证状态、会话 continue/resume、
  后台会话，以及输错子命令时的清晰行为。
- [OpenAI Codex CLI docs](https://developers.openai.com/codex/cli) 和
  [openai/codex](https://github.com/openai/codex)：将 Codex CLI 定位为
  本地终端编码 Agent，项目规则、权限、沙箱、CLI 能力和非交互自动化
  都是明确的运行表面。
- [aider commands](https://aider.chat/docs/usage/commands.html) 和
  [aider lint/test](https://aider.chat/docs/usage/lint-test.html)：
  强调会话内命令可发现性、显式 `/run` 与 `/test` 反馈回路，以及可配置
  lint/test 门禁。
- [Continue CLI quickstart](https://docs.continue.dev/cli/quickstart) 和
  [Continue CLI tool permissions](https://docs.continue.dev/cli/tool-permissions)：
  强调终端 TUI/headless 模式、版本和帮助检查、resume、只读规划、工具
  allow/deny 控制，以及认证边界。

## ADP 审计边界

ADP 保持终端优先、本地优先：

- ADP 不能演变为托管 dashboard、SaaS tracker、云同步层，或 provider-native
  resume 实现。
- ADP 不能自动执行 Git。阶段 `commit` 和 `push` 命令只记录操作者证据。
- ADP 不能自动关闭任务或阶段。任务与阶段推进必须由显式命令完成。
- 真实 Codex 和 Claude CLI 检查保持可选的操作者证据；默认 CI 与 release
  gate 必须保持确定性、fake runtime、无网络依赖。
- Runtime 产物必须留在 ADP runtime/home 路径内。项目根目录不能出现
  `AGENTS.md`、`CLAUDE.md`、`.codex`、`.claude`、planning 文件或 runtime
  manifest。

## 运行时审计矩阵

审计 gate 必须覆盖以下已有表面：

- CLI 可发现性：根帮助、命令帮助、子命令帮助、版本输出、未知命令错误，
  以及 command metadata 漂移。
- 工作区生命周期：`init`、`workspace add/list/show/remove/rename/doctor`，
  以及顶层 `doctor`。
- Runtime 入口：`env`、`enter`、`run`、fake Codex/Claude adapter、shell hook、
  completion，以及动态 completion values。
- 事件与会话：`events list`、`sessions list/show/restore-plan`，以及
  restore-plan 只读行为。
- Runtime 清理：`runtime prune` dry-run 和 kept runtime 覆盖。
- Runtime hardening：runtime parent 等于、位于或包含项目根目录时，runtime
  入口必须直接拒绝，不能只依赖 `workspace doctor` 提示。
- 任务管理：`tasks add/list/next/show/update/claim/release/done/block`，
  并覆盖支持的 text 与 JSON 输出。
- 阶段门禁：`phase add/list/show/status/start/accept/commit/push`，包括
  acceptance-before-commit-before-push 顺序。
- 规划导入：`plan preview/apply/doctor`、stdin 输入、JSON 输出、失败 apply
  回滚，以及 preview 只读行为。
- 进度报告：`progress`、`progress report`、英文和简体中文 markdown 报告，
  以及 JSON handoff 输出。

## Gate 命令

使用 `scripts/runtime-audit-smoke.sh` 做广覆盖运行时审计。该脚本会构建临时
ADP 二进制文件，使用临时 ADP home/runtime/project 目录，只使用 fake agent，
并验证项目根目录没有被污染。

聚合本地 gate 是：

```bash
scripts/check-all.sh
```

阶段验收证据必须通过 `adp phase accept` 记录实际运行过的命令及其结果。只有验收通过后，才能记录
commit 和 push 证据。
