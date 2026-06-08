# 发布检查清单

English: [release-checklist.md](release-checklist.md)

本文档定义 ADP 的本地发布门禁。它让发布验证保持在项目边界内：ADP 是 terminal-first、local-first 的 Agent Runtime Environment 和 Agent Workspace Manager。

发布门禁验证 ADP 自身的 runtime、CLI、workspace、diagnostics、文档和仓库卫生。它不会把发布验证扩展为 hosted service 检查、Web UI 检查、SaaS deployment 检查或远程 provider certification 流程。

early preview artifact 布局和 CLI 构建命令见 [release-packaging.zh-CN.md](release-packaging.zh-CN.md)。

## 必跑门禁

在 handoff、commit、push 或 release candidate tag 前运行统一门禁：

```bash
scripts/check-all.sh
```

该脚本可以从任意当前目录调用。它会根据自身位置解析仓库根目录，然后再运行检查。CI 应调用同一个脚本，而不是维护一条独立的 release gate 路径。

即使单个 smoke 为了可维护性在内部拆分，`scripts/check-all.sh` 仍然是聚合门禁。

必跑门禁按以下顺序执行：

```bash
scripts/runtime-smoke.sh --fake
scripts/example-workspace-smoke.sh
scripts/task-manager-smoke.sh
scripts/plan-intake-smoke.sh
go test -count=1 ./...
go vet ./...
scripts/check-file-lines.sh
scripts/check-docs-bilingual.sh
git diff --check
```

## 阶段切片纪律

对于正常开发 handoff，阶段切片不会因为实现停止就算完成。只有满足以下条件后，阶段才算完成：

- 相关验收命令已经通过。
- 阶段已经记录 gate 结果。
- 已验收变更已经提交。
- 该 commit 已推送到配置的远端分支。
- 下一阶段没有混入同一个 commit。

P3 phase gate 工作会把这条纪律转化为 `$ADP_HOME/workspaces/<workspace>/planning` 下的本地记录。release evidence 应同时包含正向 lifecycle path，以及拒绝乱序 phase evidence 的本地 guards。

## 门禁覆盖范围

`scripts/runtime-smoke.sh --fake` 会把当前 `cmd/adp` 二进制构建到临时目录，并运行确定性的 fake-agent runtime acceptance 路径。它使用临时 `ADP_HOME`、`ADP_RUNTIME_DIR`、fake agent binary 和临时项目根目录。

fake runtime smoke 验证：

- runtime overlay 创建。
- runtime 环境变量。
- 通过 `adp run --task <task-id>` 注入 task-bound runtime context。
- 通过 fake binary 覆盖 Codex 和 Claude adapter 启动路径。
- event log 写入。
- session history 查询。
- 只读 session restore-plan 输出，包括原始 agent arguments，并且 inspection 不会修改 event log。
- 通过 `adp workspace doctor` 和 `adp doctor` 提供 workspace diagnostics。
- runtime parent safety diagnostics：fake smoke 覆盖 project-root overlap 拒绝路径，Go 测试覆盖文件系统根目录、包含 project root、symlink 和非目录风险。
- agent command/profile diagnostics：fake smoke 覆盖 project root 中的保留路径、adapter default command fallback、inline command arguments、缺失的非 default profile、逃逸到 workspace 外部的 profile symlink，以及 enabled 但未知的 agent 配置；Go 测试覆盖缺失或不可执行的路径型 command wrapper 和重复 profile 文件。
- shell export 渲染。
- bash 和 zsh completion 渲染。
- 本地 workspace 和 profile 的动态 completion 值端点。
- 全局 `adp doctor [workspace]` diagnostics。
- `adp version` 和 `adp --version` 输出。
- ADP-owned runtime 清理。
- runtime manifest compatibility checks，确保 prune 只处理当前版本且结构自洽的 ADP runtime 目录。
- 防止 runtime artifact 或 planning 文件污染真实项目根目录。

`scripts/example-workspace-smoke.sh` 会构建当前 `cmd/adp` 二进制，把 `examples/basic-workspace` 复制到临时 `ADP_HOME`，把复制后的 `project.root` 改写为临时项目，并用该示例验证 `adp init`、`workspace doctor`、`workspace show`、`env --cd`、fake Codex runtime launch、本地 events、sessions 和 restore-plan 输出。

example workspace smoke 验证：

- 发布的示例可以被复制使用，不依赖仓库本地状态。
- 示例 workspace schema 与当前 CLI 保持兼容。
- 临时项目根目录可以被链接进 kept runtime overlay。
- 通过复制后的示例执行 fake local agent 会记录 session history，并支持只读 restore planning。
- 示例文档和发布声明有可执行路径支撑。

`scripts/task-manager-smoke.sh` 仍然是 workspace-local task、phase 和 progress report runtime acceptance 的公开入口。它会构建当前 `cmd/adp` 二进制，创建临时 workspace，执行 `adp tasks add/list/next/show/update/claim/release/block/done`、`adp phase add/list/show/start/accept/commit/push`、`adp progress` 和 `adp progress report`，并验证 planning 文件写入 `$ADP_HOME/workspaces/<workspace>/planning`，next-work/report 生成保持只读，且没有 planning 或 report artifacts 写入真实项目根目录。

P9 可以把共享 smoke helpers 和 JSON report validator 移到 `scripts/` 下的 helper files 中。这种拆分只是维护和 hardening 的实现细节；调用者仍然运行 `scripts/task-manager-smoke.sh`，release gate 仍然通过 `scripts/check-all.sh` 运行它。

phase gate smoke 路径覆盖 phase records、带 lease 的 task claim ownership、带 owner 校验的 release、task phase validation、acceptance 或 gate records、commit records、push records、lifecycle ordering guards，以及项目根目录污染防护。Go 测试还会覆盖 planning lock 行为、claim conflicts、lease expiry、terminal-task claim rejection、failed acceptance 和 failed push 语义。不要为尚不存在的命令添加 placeholder assertions。

`scripts/plan-intake-smoke.sh` 会构建当前 `cmd/adp` 二进制，创建临时 workspace，并用结构化 YAML 输入验证 `adp plan preview` 和 `adp plan apply`。它证明 preview 保持只读，apply 只写 `$ADP_HOME/workspaces/<workspace>/planning` 下的本地 planning ledger，JSON 输出仍是 inspection format，fresh workspace 上的 invalid input 不会留下 planning 目录，staging failure 不会留下 partial phase/task/progress state，并且不会产生 runtime、Git、event log 或真实 project-root 副作用。

`go test -count=1 ./...` 会运行完整 Go 测试套件，并且不使用缓存测试结果。

`go vet ./...` 运行 Go 静态检查。

`scripts/check-file-lines.sh` 执行项目规则：代码文件必须控制在 700 物理行以内。它会检查 tracked 文件以及未被 ignored 的 untracked 文件。

`scripts/check-docs-bilingual.sh` 执行 tracked Markdown 文件以及未被 ignored 的 untracked Markdown 文件的文档配对规则。英文是默认文档，维护中的 Markdown 文件需要使用 `*.zh-CN.md` 作为简体中文 counterpart。

`git diff --check` 检查当前 diff 中的空白错误。

## 可选真实 CLI 证据

真实 Codex 和 Claude CLI 检查不属于默认门禁。它们是 opt-in release evidence，因为本地安装、凭据、模型可用性、网络访问和交互行为都会随 operator 环境变化。

只有在本地 Codex CLI 被明确纳入 release evidence 时，才运行轻量真实 Codex 检查：

```bash
ADP_SMOKE_REAL_CODEX=1 scripts/runtime-smoke.sh --real-codex
```

只有在本地 Claude CLI 被明确纳入 release evidence 时，才运行轻量真实 Claude 检查：

```bash
ADP_SMOKE_REAL_CLAUDE=1 scripts/runtime-smoke.sh --real-claude
```

真实 CLI smoke 会确认外部命令存在，并且轻量 `--version` 或 `--help` invocation 可以完成。它不能证明完整交互式 agent session、provider 凭据、账号额度、模型选择、外部工具权限或网络路径已经 ready。

收集真实 CLI evidence 时，应记录：

- 执行过的命令。
- 可用时记录 Codex 或 Claude CLI 版本。
- 操作系统和 shell。
- `ADP_SMOKE_CODEX_BIN` 或 `ADP_SMOKE_CLAUDE_BIN` 等环境覆盖。
- 是否完成了单独的手工交互式 session。

## 失败定位

如果 `scripts/runtime-smoke.sh --fake` 失败，优先查看报告的失败步骤。fake smoke 是 runtime overlay 行为、runtime manifest 字段、adapter 启动路径、本地 event history、session 聚合和项目根目录污染防护的最高信号检查。

如果 task-bound runtime smoke 步骤失败，优先检查 workspace 解析、`$ADP_HOME/workspaces/<workspace>/planning` 下的 task lookup、`AGENTS.md` 或 `CLAUDE.md` 中生成的 task context、runtime 环境变量中的 `ADP_TASK_ID`，以及 events 和 sessions 中的 task ID。

如果 diagnostics 步骤失败，对比 `adp doctor [workspace]` 和 `adp workspace doctor [name]`，并检查本地 workspace registry、project root、`ADP_RUNTIME_DIR`、引用的 prompt、memory、MCP、profile 文件和 agent command 设置。对于 runtime parent 失败，确认 `ADP_RUNTIME_DIR` 不是文件系统根目录、不等于 project root、不位于 project root 内部、不是包含 project root 的父目录、不是文件，也不是非预期的 symlink。对于 agent command/profile warning，检查 enabled agent 是否有 adapter default、`command` 是否包含应该放到 `--` 之后或移入 wrapper 的 inline arguments、路径型 command wrapper 是否存在且可执行、非 default profile 文件是否缺失或重复，以及 profile 文件是否通过 symlink 或 path traversal 逃逸出 workspace。

如果 completion value 步骤失败，检查 `$ADP_HOME/workspaces` 下的本地 workspace 名称发现、`ADP_WORKSPACE` 或 `--workspace` 解析、workspace agent profiles，以及 workspace `profiles/` 目录下的文件。completion value endpoints 必须保持只读和本地化。

如果 version 步骤失败，检查 `internal/cli` 中的 CLI build variables，以及 [release-packaging.zh-CN.md](release-packaging.zh-CN.md) 中记录的 release `-ldflags`。开发构建可以输出 `dev`；packaged preview binary 应注入 version、commit 和 build date。

如果 `scripts/example-workspace-smoke.sh` 失败，优先检查复制后的 `examples/basic-workspace/workspace.yaml` 是否仍匹配当前 schema，以及 `adp env <workspace> --cd` 是否仍能生成带项目文件 symlink 的 kept runtime。

如果 `scripts/task-manager-smoke.sh` 失败，优先检查 task CLI 解析、workspace 解析、`planning/` 下的 task 存储、next-work JSON selection、helper wiring、JSON report validation、next-work/report read-only 检查，以及项目根目录污染防护。

如果 `scripts/plan-intake-smoke.sh` 失败，优先检查 plan input 解析、plan preview 只读行为、显式 apply 写入 `$ADP_HOME`、planning batch rollback 行为、JSON 输出结构，以及 project-root/runtime/Git 副作用检查。

如果 phase-gate smoke 步骤失败，优先检查 phase record 存储、task owner 状态、claim lease parsing、owner-checked release、append-only progress events、acceptance 结果记录、commit hash 记录、push 结果记录和 lifecycle ordering。预期状态必须继续保存在 `$ADP_HOME` 下，不能通过把 planning artifacts 写进项目根目录来修复失败。

如果 `go test -count=1 ./...` 失败，先定位失败 package，并在修改前单独重跑该 package：

```bash
go test -count=1 ./internal/workspace
go test -count=1 ./internal/cli
go test -count=1 ./test/e2e
```

如果 `go vet ./...` 失败，把它作为 code-quality gate 处理，先修复报告的 package，再重跑完整门禁。

如果 `scripts/check-file-lines.sh` 失败，在继续增加行为前先拆分被报告的代码文件。不要通过手工制造 generated-looking 文件或无关格式调整绕过 700 行限制。如果被报告的是 scratch file，应删除或明确 ignore。

如果 `scripts/check-docs-bilingual.sh` 失败，补齐缺失的英文默认文档或简体中文 counterpart。新增 Markdown 文件应遵循默认英文路径加 `*.zh-CN.md` counterpart 的模式。如果被报告的是本地 note，应删除或明确 ignore。

如果 `git diff --check` 失败，清理被报告文件中的 trailing whitespace 或 conflict marker。

## 手工发布检查

发布 release candidate 前，operator 还应确认：

- `git status --short --branch` 在提交前只显示有意变更，提交后工作区干净。
- `.envrc` 和 `mvp.md` 仍被忽略且未提交。
- 仓库本地 Git identity 没有配置 `user.name` 或 `user.email`。
- license 文件和 PolyForm Noncommercial 定位没有被意外修改。
- packaged CLI artifact 使用 version、commit 和 build-date ldflags 构建，且 `adp version` 报告符合预期。
- README 和 focused docs 描述当前 CLI surface，且没有 Web、UI、SaaS、cloud sync、hosted tracker、hosted orchestration、automatic Git execution、automatic task closure、provider-native resume 或 project-root report export 偏移。
- 活跃开发阶段在下一阶段开始前，已有 acceptance、commit 和 push 的本地证据。
- 任何声明的 real-agent compatibility 都有对应的 opt-in real CLI evidence，必要时还有手工交互式验收记录。

## 范围之外

发布门禁不验证：

- provider 账号或 billing。
- 远程模型可用性。
- 外部网络可靠性。
- 真实交互式 Codex 或 Claude session 质量。
- 用户特定的 shell startup files。
- hosted deployment、SaaS operations、dashboard、Web UI behavior、hosted tracker、automatic Git execution、automatic task closure、provider-native resume 或 project-root report export 行为。

这些检查属于 operator-specific acceptance notes，不属于默认本地发布门禁。
