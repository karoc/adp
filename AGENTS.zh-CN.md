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
scripts/example-workspace-smoke.sh
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
- 阻塞集成主线的关键工作留在主线程处理。
- 不把同一组文件交给多个写入型子 Agent；只读 review Agent 例外。
- 每个子 Agent 返回后必须审阅 diff。
- 集成后跑全仓门禁，不能只依赖子 Agent 的局部检查。
- 每个阶段切片完成后，先验收、提交并推送，再开始下一阶段。
- 只有集成树验证通过后才能提交和推送。

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
- 必守约束。
- 必跑验证命令。
- 最终汇报格式：修改文件、行为变化、测试结果。

只读审查 Agent 必须明确说明不得编辑文件。

## 实现原则

- 优先沿用现有 package 边界和本地模式，不轻易引入新抽象。
- 修改范围紧贴请求的行为。
- 有结构化 parser 和 typed API 时优先使用。
- 测试规模随风险增加。改动 shared behavior、CLI contract、runtime behavior 或 workspace safety 时必须扩大测试。
- 保持 local-first。测试应使用临时 `ADP_HOME`、临时 `ADP_RUNTIME_DIR`、fake binary 和临时 project root。
- 默认测试不能调用真实外部 CLI。真实 Codex/Claude 检查必须显式 opt-in。

## Runtime 验收

确定性的 runtime smoke 路径是：

```bash
scripts/runtime-smoke.sh --fake
```

它验证本地 runtime overlay、fake Codex/Claude 启动链路、event log、session history、runtime pruning，以及不污染 project root。

可复制 example workspace smoke 路径是：

```bash
scripts/example-workspace-smoke.sh
```

它验证 `examples/basic-workspace` 可以被复制到临时 `ADP_HOME`，指向临时项目根目录，并完成 diagnostics、show 和 kept runtime overlay 构建。

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

## 阶段纪律

一个规划阶段切片完成后：

1. 运行该阶段对应的 runtime smoke。
2. 运行 `scripts/check-all.sh`。
3. 提交已验收的阶段。
4. 推送该提交。
5. 推送成功后才开始下一阶段。

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
