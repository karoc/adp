# 发布检查清单

English: [release-checklist.md](release-checklist.md)

本文档定义 ADP 的本地发布门禁。它让发布验证保持在项目边界内：ADP 是 terminal-first、local-first 的 Agent Runtime Environment 和 Agent Workspace Manager。

发布门禁验证 ADP 自身的 runtime、CLI、workspace、diagnostics、文档和仓库卫生。它不会把发布验证扩展为 hosted service 检查、Web UI 检查、SaaS deployment 检查或远程 provider certification 流程。

## 必跑门禁

在 handoff、commit、push 或 release candidate tag 前运行统一门禁：

```bash
scripts/check-all.sh
```

该脚本可以从任意当前目录调用。它会根据自身位置解析仓库根目录，然后再运行检查。

必跑门禁按以下顺序执行：

```bash
scripts/runtime-smoke.sh --fake
scripts/example-workspace-smoke.sh
scripts/task-manager-smoke.sh
go test -count=1 ./...
go vet ./...
scripts/check-file-lines.sh
scripts/check-docs-bilingual.sh
git diff --check
```

## 门禁覆盖范围

`scripts/runtime-smoke.sh --fake` 会把当前 `cmd/adp` 二进制构建到临时目录，并运行确定性的 fake-agent runtime acceptance 路径。它使用临时 `ADP_HOME`、`ADP_RUNTIME_DIR`、fake agent binary 和临时项目根目录。

fake runtime smoke 验证：

- runtime overlay 创建。
- runtime 环境变量。
- 通过 fake binary 覆盖 Codex 和 Claude adapter 启动路径。
- event log 写入。
- session history 查询。
- workspace diagnostics。
- shell export 渲染。
- bash 和 zsh completion 渲染。
- ADP-owned runtime 清理。
- 防止 runtime artifact 污染真实项目根目录。

`scripts/example-workspace-smoke.sh` 会构建当前 `cmd/adp` 二进制，把 `examples/basic-workspace` 复制到临时 `ADP_HOME`，把复制后的 `project.root` 改写为临时项目，并用该示例验证 `adp init`、`workspace doctor`、`workspace show` 和 `env --cd`。

example workspace smoke 验证：

- 发布的示例可以被复制使用，不依赖仓库本地状态。
- 示例 workspace schema 与当前 CLI 保持兼容。
- 临时项目根目录可以被链接进 kept runtime overlay。
- 示例文档和发布声明有可执行路径支撑。

`scripts/task-manager-smoke.sh` 会构建当前 `cmd/adp` 二进制，创建临时 workspace，执行 `adp tasks add/list/show/update/block/done` 和 `adp progress`，并验证 planning 文件写入 `$ADP_HOME/workspaces/<workspace>/planning`，而不是写入真实项目根目录。

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

如果 `scripts/runtime-smoke.sh --fake` 失败，优先查看报告的失败步骤。fake smoke 是 runtime overlay 行为、adapter 启动路径、本地 event history、session 聚合和项目根目录污染防护的最高信号检查。

如果 `scripts/example-workspace-smoke.sh` 失败，优先检查复制后的 `examples/basic-workspace/workspace.yaml` 是否仍匹配当前 schema，以及 `adp env <workspace> --cd` 是否仍能生成带项目文件 symlink 的 kept runtime。

如果 `scripts/task-manager-smoke.sh` 失败，优先检查 task CLI 解析、workspace 解析、`planning/` 下的 task 存储，以及项目根目录污染防护。

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
- README 和 focused docs 描述当前 CLI surface，且没有 Web、UI、SaaS、cloud sync 或 hosted orchestration 偏移。
- 任何声明的 real-agent compatibility 都有对应的 opt-in real CLI evidence，必要时还有手工交互式验收记录。

## 范围之外

发布门禁不验证：

- provider 账号或 billing。
- 远程模型可用性。
- 外部网络可靠性。
- 真实交互式 Codex 或 Claude session 质量。
- 用户特定的 shell startup files。
- hosted deployment、SaaS operations、dashboard 或 Web UI 行为。

这些检查属于 operator-specific acceptance notes，不属于默认本地发布门禁。
