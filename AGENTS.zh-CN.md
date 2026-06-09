# Agent 工作指南

English: [AGENTS.md](AGENTS.md)

本文沉淀 ADP 项目中 Agent 参与开发时必须遵守的工作规则。它是规划、分工、实现、验证和交付的项目级约定。

## 产品边界

ADP 是 terminal-first、local-first 的 Agent Runtime Environment 和 Agent Workspace Manager。

所有工作都必须保持在这个边界内：

- 优先建设本地 CLI workflow、runtime overlay、workspace registry、adapter、shell integration、event log、session、diagnostics 和 release gate。
- 不偏向 Web UI、dashboard、SaaS、cloud sync、托管编排或图形化多 Agent 产品。
- 真实项目根目录必须保持干净。`AGENTS.md`、`CLAUDE.md`、`.codex/`、`.claude/` 等 ADP 生成文件应进入 runtime overlay，不应污染用户项目根目录；除非用户明确要求编辑本仓库自身文件。
- 外部 Agent CLI 是兼容性边界。修改 adapter 假设前必须验证当前行为。

## 硬约束

- 代码文件必须控制在 700 个物理行以内，超过前先拆分。
- 规划拆分或 hardening 阶段时，可用 `scripts/check-file-lines.sh --audit` 做非阻断 pressure report。它不能替代必跑硬门禁。
- 文档默认语言为英文。每个维护中的 Markdown 文档都必须有 `*.zh-CN.md` 简体中文 counterpart。
- `.envrc` 和 `mvp.md` 必须保持 ignored，不提交。
- 不配置仓库本地 Git `user.name` 或 `user.email`。
- 提交只使用一次性身份：

```bash
GIT_AUTHOR_NAME=karoc GIT_COMMITTER_NAME=karoc git commit -m "<message>"
```

- 直接推送：

```bash
git push
```

- 项目采用 PolyForm Noncommercial 授权模式。没有维护者明确要求时，不替换或重新解释授权策略。

## 标准门禁

交付、提交或推送前运行：

```bash
scripts/check-all.sh
```

如果变更早期还没有 `scripts/check-all.sh`，则运行底层门禁：

```bash
scripts/runtime-smoke.sh --fake
scripts/runtime-audit-smoke.sh
scripts/runtime-context-smoke.sh
scripts/release-readiness-smoke.sh
scripts/release-rehearsal-smoke.sh
scripts/release-artifact-smoke.sh
scripts/release-operator-drill-smoke.sh
scripts/install-onboarding-smoke.sh
scripts/example-workspace-smoke.sh
scripts/task-manager-smoke.sh
scripts/plan-intake-smoke.sh
go test -count=1 ./...
go vet ./...
scripts/check-file-lines.sh
scripts/check-docs-bilingual.sh
git diff --check
```

行数检查和双语文档门禁会覆盖 tracked 文件以及未被 ignored 的 untracked 文件。不要把不满足项目约束的临时 source、script、config 或 Markdown 文件留在工作区；确实不应进入项目的文件必须显式 ignored。

## 多 Agent 执行标准

当用户要求并行或多 Agent 工作，且任务能拆分为互不冲突的写入范围时，使用子 Agent。

主线程职责：

- 启动子 Agent 前明确目标、约束和互斥写入范围。
- 使用 ADP 作为共享任务看板。当 worker 需要在启动时原子领取任务时，优先使用 `adp run <agent> --take --owner <owner> [--lease 4h]`。只需要手工领取而不启动 Agent 时，使用 `adp tasks take`；长时间运行的 worker 应在 lease 过期前使用 `adp tasks renew` 续租。
- 阻塞集成主线的关键工作留在主线程处理。
- 不把同一组文件交给多个写入型子 Agent；只读 review Agent 例外。
- 每个子 Agent 返回后必须审阅 diff。
- 集成后跑全仓门禁，不能只依赖子 Agent 的局部检查。
- 每个阶段切片完成后，先验收、记录验收、提交、推送并记录 push evidence，再开始下一阶段。
- 只有集成树验证通过后才能提交和推送。门禁失败表示该阶段仍未完成。

适合并行拆分的任务边界：

- Runtime acceptance：`scripts/runtime-smoke.sh`、`docs/runtime-acceptance*.md`。
- Release gates：`scripts/check-all.sh`、`docs/release-checklist*.md`。
- Workspace diagnostics：`internal/workspace/diagnostics*`。
- CLI behavior：`internal/cli/` 和相关 CLI 测试。
- Examples：`examples/` 和 example 专属文档。
- Documentation：明确列出的 Markdown 文件及其 `.zh-CN.md` counterpart。

子 Agent 任务说明必须写清：

- 目标。
- 允许写入路径。
- 禁止写入路径。
- ADP task ownership 预期，包括该 worker 是使用 `adp run --take`、`adp tasks take`，还是使用已明确分配的 task ID。
- 必守约束。
- 必跑验证命令。
- 最终汇报格式：修改文件、行为变化、测试结果。

只读审查 Agent 必须明确说明不得编辑文件。

中断 worker 的恢复必须通过 ADP，而不是 provider-private state。Operator 可以用 `adp tasks stale --workspace <workspace> [--format text|json]` 查看 lease 已过期的 in-progress claims；lease 过期后，其他 worker 可以按 ADP ownership rules 通过 `adp tasks take` 或显式 `adp tasks claim` 接管任务。不能根据 provider task box、plan panel 或进程退出推断 completion、phase acceptance、commit evidence、push evidence 或 Git state。

## 实现原则

- 优先沿用现有 package 边界和本地模式，不轻易引入新抽象。
- 修改范围紧贴请求的行为。
- 有结构化 parser 和 typed API 时优先使用。
- CLI 命令变更必须通过本地 command metadata contract 保持一致。Usage text、dispatch wiring、bash/zsh completion、tests 以及 smoke 或文档验收必须描述同一套命令面；P16 会在不引入新 CLI 框架的前提下强化这一点。
- 测试规模随风险增加。改动 shared behavior、CLI contract、runtime behavior 或 workspace safety 时必须扩大测试。
- 保持 local-first。测试应使用临时 `ADP_HOME`、临时 `ADP_RUNTIME_DIR`、fake binary 和临时 project root。
- 默认测试不能调用真实外部 CLI。真实 Codex/Claude 检查必须显式 opt-in。

## 工具 Plan Mode

Provider 原生 plan mode 和 plan panels 是 proposal surfaces。它们可以帮助 Agent 展示候选工作，但 ADP 仍然是权威本地 planning 和 progress ledger。

在 plan mode 中，不要编辑 implementation files、complete tasks、accept phases、commit、push，或以其他方式执行计划；除非用户明确批准离开 planning。结构化 proposals 用只读路径验证：

```bash
adp plan preview --workspace <workspace> --file - --format json
```

只有在用户或 operator 明确批准后才能 apply plan：

```bash
adp plan apply --workspace <workspace> --file - --format json
```

批准后的 plan apply 完成后，继续使用 ADP task 和 phase commands 维护持久 task ownership、progress、blockers、acceptance、commit evidence 和 push evidence。原生 plan panels 可以为了可读性镜像 ADP items，但它们只是 scratch views。

如果工具通过 `adp run --take` 进入 plan mode，已领取的 task 是该 session 的 ADP-owned active work item，但 provider 原生 plan 仍然只是 proposal view。不能因为 native plan item 被勾选或 provider session 退出，就把 task 标记为 done、accept phase、commit、push，或运行 Git。

## Runtime 验收

确定性的 runtime smoke 路径是：

```bash
scripts/runtime-smoke.sh --fake
```

它验证本地 runtime overlay、fake Codex/Claude 启动链路、event log、session history、runtime pruning，以及不污染 project root。

广覆盖 runtime audit 路径是：

```bash
scripts/runtime-audit-smoke.sh
```

它使用 fake agent 和临时目录验证已发布 CLI 命令面、help 输出、JSON 可解析性、task/phase/plan/progress flow、session、restore planning、completion values，以及 local-first runtime 边界。

聚焦 runtime context smoke 路径是：

```bash
scripts/runtime-context-smoke.sh
```

它通过 generated instruction files、adapter metadata、selected profiles、prompt、shared memory、MCP references、task metadata、runtime environment variables、本地 event/session evidence、workspace diagnostics 和 project-root cleanliness 验证 launch-time context。

release readiness smoke 路径是：

```bash
scripts/release-readiness-smoke.sh
```

它验证不依赖真实 provider CLI 的 release gate invariant，包括 phase commit 和 push 命令只记录证据、不会执行 Git。

release rehearsal smoke 路径是：

```bash
scripts/release-rehearsal-smoke.sh
```

它会把当前未被 ignored 的仓库文件复制到临时干净 workspace，使用 release ldflags 构建 preview binary，验证复制后的文档和文件行数，使用隔离 ADP 路径 bootstrap 复制后的 example workspace，并通过 fake Git tripwire 检查 phase evidence recording。

release artifact smoke 路径是：

```bash
scripts/release-artifact-smoke.sh
```

它验证 package staging、checksums、manifest boundaries、install-from-artifact 行为、provider-free first-run rehearsal，以及不依赖 `.git` 的 source archive build。

release operator drill smoke 路径是：

```bash
scripts/release-operator-drill-smoke.sh
```

它验证文档化 release commands、no-`.git` operator source 处理、release script syntax checks、显式 commit build metadata、checksum verification、installed `PATH` 行为、fake Codex handoff、本地 phase evidence、fake Git tripwire protection，以及 project-root cleanliness。

install onboarding smoke 路径是：

```bash
scripts/install-onboarding-smoke.sh
```

它验证安装到临时 `GOBIN`、`PATH` precedence、首次使用 workspace registration、fake Codex/Claude command handling、task-bound context、本地 event/session/progress evidence、Git side-effect guards，以及 project-root cleanliness。

可复制 example workspace smoke 路径是：

```bash
scripts/example-workspace-smoke.sh
```

它验证 `examples/basic-workspace` 可以被复制到临时 `ADP_HOME`，指向临时项目根目录，并完成 diagnostics、show 和 kept runtime overlay 构建。

task manager smoke 路径是：

```bash
scripts/task-manager-smoke.sh
```

它验证 workspace-local task、phase、planning doctor、next-work、progress、progress report、本地 phase evidence、read-only report generation，以及 project-root pollution protection。

plan intake smoke 路径是：

```bash
scripts/plan-intake-smoke.sh
```

它验证来自文件和 stdin 的本地结构化 plan preview/apply、显式写入 `$ADP_HOME` 下的 ledger、failed 或 duplicate apply rollback、read-only preview、JSON inspection output，以及无 runtime、Git、event-log 或 project-root side effects。

真实外部 CLI 检查是可选 release evidence，必须显式启用：

```bash
ADP_SMOKE_REAL_CODEX=1 scripts/runtime-smoke.sh --real-codex
ADP_SMOKE_REAL_CLAUDE=1 scripts/runtime-smoke.sh --real-claude
```

这些检查不能替代凭据、模型、网络行为或交互式 session 的人工真实 Agent 验收。

## 文档规则

- 英文是默认文档。
- 简体中文 counterpart 必须包含同等操作内容，不应只是摘要。
- README 保持简洁，把细节链接到专题文档。
- 新增脚本或 release gate 时，必须说明运行时机和不覆盖的验收边界。
- 不加入 Web/SaaS 定位。

## 当前项目 Dogfooding

ADP 自身开发从 P24 开始使用 ADP 自己的本地 planning ledger。把 `adp` workspace 视为执行状态事实源：

- 每个新的实现切片开始前，先登记为 phase 和按优先级排序的 tasks。
- 权威 phase/task/progress records 保存在 `$ADP_HOME` 下；正常流程中不要把 planning state 导出到仓库根目录。
- 需要在 Agent 启动时原子领取任务时，使用 `adp run <agent> --workspace adp --take --owner <owner> --lease <duration> -- <agent-args>`；长时间执行时用 `adp tasks renew --workspace adp <task-id> --owner <owner> --lease <duration>` 续租；主线程和子 Agent 协作交接时，使用 `adp tasks next --workspace adp --limit 0 --format json` 和 `adp phase status --workspace adp --format json` 作为本地 snapshot。
- 重新分配工作前，使用 `adp tasks stale --workspace adp` 找出 in-progress lease 已过期的中断 worker。
- 当 Codex、Claude 或其他工具提供原生 task/todo panel 时，可以把当前 ADP task 镜像进去提升可见性，但持久 status、ownership、progress 和恢复证据仍必须维护在 ADP 中。
- 当工具提供 plan mode 时，只用它起草或展示候选 plans；proposal 通过 `adp plan preview` 并获得明确批准执行 `adp plan apply` 前，不写入持久 ledger。
- 当前 phase 未通过验证、未记录验收、未提交、未推送、未记录 commit 和 push evidence 前，不启动后续 phase。
- 仓库文档可以总结已验收行为，但不是执行 ledger。

## 阶段纪律

一个规划阶段切片完成后：

1. 运行该阶段对应的 runtime smoke。
2. 运行 `scripts/check-all.sh`。
3. 只有这些门禁通过后，才能记录验收。
4. 提交已验收的阶段。
5. 推送该提交。
6. 记录 push evidence。
7. 推送成功且 phase record 更新后，才开始下一阶段。

不要把后续阶段工作混入同一个提交。这样可以让规划、执行进度、验收证据和 Git 历史保持一致。

## Git 工作流

提交前：

```bash
git status --short --branch
git config --local --get-regexp '^user\.' || true
git check-ignore -v .envrc mvp.md || true
scripts/check-all.sh
git diff --check
```

提交后：

```bash
git status --short --branch
git log --oneline --decorate -5
git config --local --get-regexp '^user\.' || true
git push
```

最终汇报必须包含 commit hash、已推送分支、已运行门禁，以及仍需人工验收的缺口。
