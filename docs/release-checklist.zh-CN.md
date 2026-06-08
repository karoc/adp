# 发布检查清单

English: [release-checklist.md](release-checklist.md)

本文档定义 ADP 的本地发布门禁。它让发布验证保持在项目边界内：ADP 是 terminal-first、local-first 的 Agent Runtime Environment 和 Agent Workspace Manager。

发布门禁验证 ADP 自身的 runtime、CLI、workspace、diagnostics、文档和仓库卫生。它不会把发布验证扩展为 hosted service 检查、Web UI 检查、SaaS deployment 检查或远程 provider certification 流程。

early preview artifact 布局和 CLI 构建命令见 [release-packaging.zh-CN.md](release-packaging.zh-CN.md)。operator failure triage 见 [release-troubleshooting.zh-CN.md](release-troubleshooting.zh-CN.md)。相邻工具范围校准见 [comparable-tools.zh-CN.md](comparable-tools.zh-CN.md)。

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
scripts/runtime-audit-smoke.sh
scripts/release-readiness-smoke.sh
scripts/release-rehearsal-smoke.sh
scripts/release-artifact-smoke.sh
scripts/release-operator-drill-smoke.sh
scripts/example-workspace-smoke.sh
scripts/task-manager-smoke.sh
scripts/plan-intake-smoke.sh
go test -count=1 ./...
go vet ./...
scripts/check-file-lines.sh
scripts/check-docs-bilingual.sh
git diff --check
```

## 发布包内容

发布 preview artifact 前，应先检查 package 内容。Package 应包含目标平台的 `adp` binary、`README.md`、`README.zh-CN.md`、`LICENSE`、`COMMERCIAL.md`、`COMMERCIAL.zh-CN.md`、`docs/release-packaging.md`、`docs/release-packaging.zh-CN.md`、`docs/release-evidence.md`、`docs/release-evidence.zh-CN.md`，以及记录 commit、version、target platform、gate result 和 checksum 的 release evidence note 或 release note。

Package 必须保留 PolyForm Noncommercial 和 source-available 定位。非商业再分发必须保留许可证文本、必要声明，以及对 ADP 和版权持有人的署名。任何商业使用都必须取得单独付费授权；不得把 preview package 描述成已经授予商业权利。

Package 必须排除本地或敏感 operator 状态，包括 `.envrc`、`mvp.md`、`$ADP_HOME`、`$ADP_RUNTIME_DIR`、runtime overlay、event log、session log、task 或 phase 状态、凭据、token、账号标识、私有 prompt，以及机器特定的 shell startup file。

## 阶段切片纪律

对于正常开发 handoff，阶段切片不会因为实现停止就算完成。只有满足以下条件后，阶段才算完成：

- 相关验收命令已经通过。
- 阶段已经记录 gate 结果。
- 已验收变更已经提交。
- 该 commit 已推送到配置的远端分支。
- 启动任何下一阶段前，本地 phase ledger 已记录 commit 和 push evidence。
- 下一阶段没有混入同一个 commit。

P3 phase gate 工作会把这条纪律转化为 `$ADP_HOME/workspaces/<workspace>/planning` 下的本地记录。release evidence 应同时包含正向 lifecycle path，以及拒绝乱序 phase evidence 的本地 guards。

## 门禁覆盖范围

`scripts/runtime-smoke.sh --fake` 会把当前 `cmd/adp` 二进制构建到临时目录，并运行确定性的 fake-agent runtime acceptance 路径。它使用临时 `ADP_HOME`、`ADP_RUNTIME_DIR`、fake agent binary 和临时项目根目录。

P17 可以把共享 helper 以及 fake diagnostics/session/prune slices 拆到 `scripts/` 下的 helper 文件。这只是为了在 700 行文件上限内保持可维护性的实现细节；调用方仍然运行 `scripts/runtime-smoke.sh --fake`，聚合 release gate 仍然通过 `scripts/check-all.sh` 运行它。

fake runtime smoke 验证：

- runtime overlay 创建。
- runtime 环境变量。
- 通过 `adp run <agent> --task <task-id>` 注入 task-bound runtime context。
- 通过 fake binary 覆盖 Codex 和 Claude adapter 启动路径。
- event log 写入。
- session history 查询。
- 只读 session restore-plan 输出，包括原始 agent arguments，并且 inspection 不会修改 event log。
- 通过 `adp workspace doctor` 和 `adp doctor` 提供 workspace diagnostics。
- workspace rename/remove lifecycle 检查，证明只修改临时 ADP registry 数据，真实 project root entry snapshot 和 runtime entry count 都保持不变。
- 通过 fake `SHELL` 执行受控的 `adp enter` child shell，覆盖 runtime env/cwd、project symlink、默认 cleanup、`--keep-runtime`、project-root entries 不变，以及内容级不修改 event log。
- runtime parent safety diagnostics：fake smoke 覆盖 project-root overlap 拒绝路径，Go 测试覆盖文件系统根目录、包含 project root、symlink 和非目录风险。
- agent command/profile diagnostics：fake smoke 覆盖 project root 中的保留路径、adapter default command fallback、inline command arguments、缺失的非 default profile、逃逸到 workspace 外部的 profile symlink，以及 enabled 但未知的 agent 配置；Go 测试覆盖缺失或不可执行的路径型 command wrapper 和重复 profile 文件。
- shell export 渲染。
- bash 和 zsh completion 渲染。
- 本地 agent、workspace 和 profile 的动态 completion 值端点。
- 全局 `adp doctor [workspace]` diagnostics。
- `adp version` 和 `adp --version` 输出。
- ADP-owned runtime 清理。
- runtime manifest compatibility checks，确保 prune 只处理当前版本且结构自洽的 ADP runtime 目录。
- 防止 runtime artifact 或 planning 文件污染真实项目根目录。

P25 将 bash 和 zsh completion renderer 拆分到按 shell 区分的实现文件中，用来消除 `internal/shell/completion.go` 的行数压力。这是内部维护边界：`adp completion`、bash/zsh 输出语义、metadata drift checks、动态 value endpoints 和默认 fake runtime smoke 仍然是发布证据。它不新增交互式 completion 模拟，也不新增 shell 支持。

`scripts/runtime-audit-smoke.sh` 会构建当前 `cmd/adp` 二进制，使用临时 `ADP_HOME`、`ADP_RUNTIME_DIR`、fake agent binary 和临时项目根目录，在不依赖真实 provider CLI 或网络访问的前提下验证广覆盖 runtime audit matrix。

runtime audit smoke 验证：

- CLI 可发现性和 command metadata drift 覆盖。
- Runtime 入口、events、sessions、restore-plan、runtime pruning 和项目根目录污染防护。
- 通过当前 CLI 覆盖 workspace lifecycle、diagnostics、task manager、phase gate、plan intake 和 progress report 表面。
- 本地优先边界：不做 hosted tracker sync、不自动执行 Git、不自动关闭任务、不恢复 provider-native session，也不导出 project-root planning 或 report。

`scripts/release-readiness-smoke.sh` 会构建当前 `cmd/adp` 二进制，使用临时 `ADP_HOME`、`ADP_RUNTIME_DIR`、临时项目根目录和 fake Git tripwire，验证不依赖真实 provider CLI 的 release-readiness invariant。

release readiness smoke 验证：

- 可以通过 CLI 记录 phase acceptance、commit 和 push evidence。
- 只有 accepted、committed、pushed evidence 都存在后，phase gate 状态才会进入 `plan_next_phase`。
- Phase evidence 命令不会执行 `git commit`、`git push`、`git pull`、`git fetch`、`git clone` 或 `git ls-remote` 等 Git 副作用命令。
- 默认 release path 保持本地、确定性，并且不依赖真实 Codex 或 Claude CLI。

`scripts/release-rehearsal-smoke.sh` 会把当前未被 ignored 的仓库文件复制到临时干净 workspace，使用 release ldflags 构建 preview binary，验证复制后的文档和文件行数，使用隔离的 `ADP_HOME` 和 `ADP_RUNTIME_DIR` bootstrap 复制后的 example workspace，并通过 fake Git tripwire 检查 phase evidence recording。

release rehearsal smoke 验证：

- 文档化的 preview build/version 路径可以生成 binary，并且 `adp version` 会报告注入的 version、commit 和 build date。
- 在临时干净 workspace 中，复制后的文档和代码文件仍满足双语文档和 700 行门禁。
- 发布的 example workspace 可以针对临时项目完成 bootstrap，不依赖开发机本地 `$ADP_HOME` 或 runtime state。
- Release phase evidence 保持本地记录，不执行 Git 副作用命令。

`scripts/release-artifact-smoke.sh` 会在临时 source tree 中构建 preview artifacts，组装本地 release package，验证 checksums，检查 package include/exclude boundaries，把 packaged binary 安装到临时 `PATH`，运行 provider-free first-run rehearsal，并通过显式 commit 值验证没有 `.git` 的 source archive build。

release artifact smoke 验证：

- target-platform artifacts 可以使用 release ldflags 构建，并报告注入的 version、commit 和 build date。
- package 包含 required binary、license、commercial notice、README、release packaging 和 release evidence 文件。
- package 排除 `.envrc`、`mvp.md`、本地 ADP state、runtime overlays、logs、task state、credentials 和 machine-specific shell startup files。
- packaged artifacts 在 install rehearsal 前已生成 checksums。
- installed binary 从 package path 运行，使用临时 `ADP_HOME`、临时 `ADP_RUNTIME_DIR`、fake Codex、本地 events、本地 sessions，且不污染 project root。
- 没有 `.git` 的 source archive builds 在 `COMMIT` 显式时仍然有效。

`scripts/release-operator-drill-smoke.sh` 会把仓库复制到没有 `.git` 的 operator source tree，验证 release packaging docs 暴露 required source archive、build、checksum 和 install commands，从该 source tree syntax-check release scripts，使用显式 commit metadata 构建 release binary，验证 checksum，把 artifact 安装到临时 `PATH`，并用 fake Codex 和 fake Git tripwire 运行 provider-free handoff sequence。

release operator drill smoke 验证：

- release path 可以从干净 source form 运行，不依赖 repository `.git` metadata。
- installed binary 可以初始化 ADP state、添加 workspace、启动 phase、添加 task、用 task context 运行 fake Codex，并在本地记录 accept、commit 和 push evidence。
- operator handoff sequence 可以到达 `plan_next_phase`，且不执行 Git side-effect commands。
- 临时 ADP state 和 runtime state 保持在真实 project root 之外。

`scripts/example-workspace-smoke.sh` 会构建当前 `cmd/adp` 二进制，把 `examples/basic-workspace` 复制到临时 `ADP_HOME`，把复制后的 `project.root` 改写为临时项目，并用该示例验证 `adp init`、`workspace doctor`、`workspace show`、`env --cd`、fake Codex runtime launch、本地 events、sessions 和 restore-plan 输出。

example workspace smoke 验证：

- 发布的示例可以被复制使用，不依赖仓库本地状态。
- 示例 workspace schema 与当前 CLI 保持兼容。
- 临时项目根目录可以被链接进 kept runtime overlay。
- 通过复制后的示例执行 fake local agent 会记录 session history，并支持只读 restore planning。
- 示例文档和发布声明有可执行路径支撑。

`scripts/task-manager-smoke.sh` 仍然是 workspace-local task、phase、planning doctor 和 progress report runtime acceptance 的公开入口。它会构建当前 `cmd/adp` 二进制，创建临时 workspace，执行 `adp tasks add/list/next/show/update/claim/release/block/done`、`adp phase add/list/show/status/start/accept/commit/push`、`adp plan doctor`、`adp progress` 和 `adp progress report`，并验证 planning 文件写入 `$ADP_HOME/workspaces/<workspace>/planning`，next-work/plan-doctor/report 生成保持只读，error-level planning doctor diagnostics 返回退出码 `2`，且没有 planning 或 report artifacts 写入真实项目根目录。

P9 可以把共享 smoke helpers 和 JSON report validator 移到 `scripts/` 下的 helper files 中。这种拆分只是维护和 hardening 的实现细节；调用者仍然运行 `scripts/task-manager-smoke.sh`，release gate 仍然通过 `scripts/check-all.sh` 运行它。

phase gate smoke 路径覆盖 phase records、带 lease 的 task claim ownership、带 owner 校验的 release、task phase validation、acceptance 或 gate records、commit records、push records、只读 phase gate status snapshots、只读 planning ledger diagnostics、lifecycle ordering guards，以及项目根目录污染防护。它会断言 earlier phase 没有 successful pushed evidence 时不能启动 later phase，并在该 evidence 记录后可以启动 later phase。Go 测试还会覆盖 planning lock 行为、claim conflicts、lease expiry、terminal-task claim rejection、failed acceptance、failed push 语义、显式 phase ordering 和 planning doctor invariants。不要为尚不存在的命令添加 placeholder assertions。

`scripts/plan-intake-smoke.sh` 会构建当前 `cmd/adp` 二进制，创建临时 workspace，并用来自文件以及通过 `--file -` 从 stdin 传入的结构化 YAML 输入验证 `adp plan preview` 和 `adp plan apply`。它证明 preview 保持只读，apply 只写 `$ADP_HOME/workspaces/<workspace>/planning` 下的本地 planning ledger，JSON 输出仍是 inspection format，fresh workspace 上的 invalid input 不会留下 planning 目录，staging failure 不会留下 partial phase/task/progress state，并且不会产生 runtime、Git、event log 或真实 project-root 副作用。

`go test -count=1 ./...` 会运行完整 Go 测试套件，并且不使用缓存测试结果。

`go vet ./...` 运行 Go 静态检查。

`scripts/check-file-lines.sh` 执行项目规则：代码文件必须控制在 700 物理行以内。它会检查 tracked 文件以及未被 ignored 的 untracked 文件。

`scripts/check-file-lines.sh --audit` 是非阻断 line pressure report，用于在文件接近硬上限前规划后续拆分。它会报告达到或超过 `LINE_PRESSURE_WARN_LINES` 的文件，默认阈值为 600 行，并且退出码保持为零。该 audit 默认不属于 `scripts/check-all.sh`，也不能替代硬性的行数门禁。

`scripts/check-docs-bilingual.sh` 执行 tracked Markdown 文件以及未被 ignored 的 untracked Markdown 文件的文档配对规则。英文是默认文档，维护中的 Markdown 文件需要使用 `*.zh-CN.md` 作为简体中文 counterpart。

`git diff --check` 检查当前 diff 中的空白错误。

## 可选真实 CLI 证据

真实 Codex 和 Claude CLI 检查不属于默认门禁。它们是 opt-in release evidence，因为本地安装、凭据、模型可用性、网络访问和交互行为都会随 operator 环境变化。

这类证据必须与默认门禁证据分开记录。除非该 release 明确声明 real-agent evidence，否则可选真实 CLI 检查失败不应导致默认 release gate 失败。

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
- `command -v` 解析到的命令路径，或显式 override 路径。
- 已启用的 gate 变量，例如 `ADP_SMOKE_REAL_CODEX=1` 或 `ADP_SMOKE_REAL_CLAUDE=1`。
- 可用时记录 Codex 或 Claude CLI 版本；当 `--version` 不支持时，记录第一行 `--help` 输出。
- smoke 是通过 `--version` 通过，还是回退到 `--help` 通过。
- 操作系统和 shell。
- `ADP_SMOKE_CODEX_BIN` 或 `ADP_SMOKE_CLAUDE_BIN` 等环境覆盖。
- 是否完成了单独的手工交互式 session。

任何超出命令可用性的 real-agent compatibility release note，都必须有手工交互式 evidence。不要把凭据、token、账号标识、私有 prompt 或敏感模型输出粘贴到 release notes；只记录非敏感 operator evidence。

## 失败定位

本节是 default gate failures 的快速索引。operator drill flow、package manifest failures、checksum mismatches 和 source archive issues 见 [release-troubleshooting.zh-CN.md](release-troubleshooting.zh-CN.md)。

如果 `scripts/runtime-smoke.sh --fake` 失败，优先查看报告的失败步骤。fake smoke 是 runtime overlay 行为、runtime manifest 字段、adapter 启动路径、本地 event history、session 聚合和项目根目录污染防护的最高信号检查。

如果 `scripts/runtime-audit-smoke.sh` 失败，优先检查文档中的 runtime audit matrix 是否仍匹配当前 CLI surface。audit smoke 刻意保持 fake-runtime 和 local-only；不能通过增加真实 Codex/Claude 默认路径、hosted service、automatic Git execution、provider-native resume 或 project-root export 来修复失败。

如果 `scripts/release-readiness-smoke.sh` 失败，优先检查 phase evidence recording 和 fake Git tripwire。Phase accept、commit 和 push 命令只能记录本地 evidence；不能通过让 ADP 自动执行 Git 或削弱 phase lifecycle gate 来修复失败。

如果 `scripts/release-rehearsal-smoke.sh` 失败，优先检查临时干净 workspace 步骤：复制后的非 ignored 文件、release ldflags、`adp version`、复制后的 example workspace bootstrap、隔离 `ADP_HOME` 和 `ADP_RUNTIME_DIR`，以及 fake Git tripwire 输出。不能通过依赖机器本地 ADP state、真实 provider CLI、网络访问、automatic Git execution 或 project-root export 来修复 rehearsal 失败。

如果 `scripts/release-artifact-smoke.sh` 失败，优先检查 package staging directory、artifact checksum、package manifest、install-from-artifact path、显式 source archive `COMMIT`、临时 `ADP_HOME`、临时 `ADP_RUNTIME_DIR`、fake Codex command 和 project-root pollution scan。不能通过从 source tree 直接运行、把本地状态放进 package、在 source archive 里依赖 `.git`，或把真实 Codex/Claude 检查变成默认门禁来修复 artifact failures。

如果 `scripts/release-operator-drill-smoke.sh` 失败，优先检查 no-`.git` source copy、文档化 release commands、release script syntax checks、显式 commit build、checksum verification、installed `PATH` binary、fake Codex handoff sequence、phase evidence records、fake Git tripwire 和 project-root pollution scan。不能通过增加本机 source files、automatic Git execution 或 hosted orchestration 来修复 drill failures。

如果可选真实 CLI 检查因为缺少 `ADP_SMOKE_REAL_CODEX=1` 或 `ADP_SMOKE_REAL_CLAUDE=1` 而失败，应把它视为 operator 尚未显式启用该检查。如果命令不可用，应在该机器上安装外部 CLI，或通过 `ADP_SMOKE_CODEX_BIN` 或 `ADP_SMOKE_CLAUDE_BIN` 指向预期命令路径。如果 `--version` 和 `--help` 都失败，应归类为外部 CLI、wrapper 或 operator 环境的 evidence gap，除非确定性 fake gate 或 ADP launch contract 也同时失败。

如果 operator drill 在 smoke 脚本启动前就因为 workspace 无法解析而失败，要区分两个常见情况。`workspace not found: <name>` 表示请求的名称不在本地 registry 中；运行 `adp workspace list`，用 `adp workspace add <name> <project-root>` 添加 workspace，或通过 `--workspace` / `ADP_WORKSPACE` 传入已注册名称。`workspace is required; pass --workspace, set ADP_WORKSPACE, or run from inside a registered project` 表示没有选择 workspace。不要用在项目根目录创建 planning 文件的方式绕过它。

如果手工 task-bound run 被误写成 `adp run --task <task-id>`，应改为 `adp run <agent> --workspace <name> --task <task-id> -- <agent-args>`。`--task` 是 agent 名称之后的 run option，不能替代必需的 agent 参数。

如果 task-bound runtime smoke 步骤失败，优先检查 workspace 解析、`$ADP_HOME/workspaces/<workspace>/planning` 下的 task lookup、`AGENTS.md` 或 `CLAUDE.md` 中生成的 task context、runtime 环境变量中的 `ADP_TASK_ID`，以及 events 和 sessions 中的 task ID。

如果 diagnostics 步骤失败，对比 `adp doctor [workspace]` 和 `adp workspace doctor [name]`，并检查本地 workspace registry、project root、`ADP_RUNTIME_DIR`、引用的 prompt、memory、MCP、profile 文件和 agent command 设置。对于 runtime parent 失败，确认 `ADP_RUNTIME_DIR` 不是文件系统根目录、不等于 project root、不位于 project root 内部、不是包含 project root 的父目录、不是文件，也不是非预期的 symlink。runtime parent error 应让 doctor 返回退出码 `2`，并在 runtime 创建前阻止 `adp env <workspace> --cd`、`adp enter <workspace>` 或 `adp run <agent> --workspace <name>`。应修复目录边界，而不是强行继续运行。

对于 agent command/profile warning，检查 enabled agent 是否有 adapter default、`command` 是否包含应该放到 `--` 之后或移入 wrapper 的 inline arguments、路径型 command wrapper 是否存在且可执行、非 default profile 文件是否缺失或重复，以及 profile 文件是否通过 symlink 或 path traversal 逃逸出 workspace。缺失的非 default profile 在 doctor 输出中是 warning-only；通过在 `$ADP_HOME/workspaces/<workspace>/profiles/` 下添加一个匹配文件，或改用 `default` / 已存在 profile 来修复。只有 release evidence 依赖该 profile 时，才应把它视为 release blocker。

如果 completion value 步骤失败，检查本地 adapter registry、`$ADP_HOME/workspaces` 下的本地 workspace 名称发现、`ADP_WORKSPACE` 或 `--workspace` 解析、workspace agent profiles，以及 workspace `profiles/` 目录下的文件。completion value endpoints 必须保持只读和本地化。

如果 version 步骤失败，检查 `internal/cli` 中的 CLI build variables，以及 [release-packaging.zh-CN.md](release-packaging.zh-CN.md) 中记录的 release `-ldflags`。开发构建可以输出 `dev`；packaged preview binary 应注入 version、commit 和 build date。

如果 `scripts/example-workspace-smoke.sh` 失败，优先检查复制后的 `examples/basic-workspace/workspace.yaml` 是否仍匹配当前 schema，以及 `adp env <workspace> --cd` 是否仍能生成带项目文件 symlink 的 kept runtime。

如果 `scripts/task-manager-smoke.sh` 失败，优先检查 task CLI 解析、workspace 解析、`planning/` 下的 task 存储、planning doctor diagnostics 和退出码 `2` 行为、next-work JSON selection、helper wiring、JSON report validation、next-work/plan-doctor/report read-only 检查，以及项目根目录污染防护。

如果 `scripts/plan-intake-smoke.sh` 失败，优先检查 plan input 解析、plan preview 只读行为、显式 apply 写入 `$ADP_HOME`、planning batch rollback 行为、JSON 输出结构，以及 project-root/runtime/Git 副作用检查。

如果 phase-gate smoke 步骤失败，优先检查 phase record 存储、task owner 状态、claim lease parsing、owner-checked release、append-only progress events、acceptance 结果记录、commit hash 记录、push 结果记录和 lifecycle ordering。预期的 operator error 包括未通过 acceptance 就记录 commit evidence、未记录 commit evidence 就记录 push evidence，以及 earlier phase 没有 pushed evidence 就启动 later phase。修复路径是在真实 validation、commit 和 push 已经在 ADP 外部完成后，显式按 `adp phase accept`、`adp phase commit`、`adp phase push` 的顺序记录。预期状态必须继续保存在 `$ADP_HOME` 下，不能通过把 planning artifacts 写进项目根目录或让 ADP 自动执行 Git 来修复失败。

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
- Preview package 包含 `LICENSE`、`COMMERCIAL.md` 和 `COMMERCIAL.zh-CN.md`，保留必要声明和署名，并且不包含 `.envrc`、`mvp.md`、本地 ADP 状态、runtime overlay、日志、任务状态、凭据或机器特定 shell 配置。
- 发布前已记录 sorted package manifest，并按 required include/exclude list 检查。
- 至少一个 artifact 已从 package 安装到临时 `PATH` 并完成演练，且没有使用 source-tree binary。
- license 文件和 PolyForm Noncommercial/source-available 定位没有被意外修改，公开文档也没有暗示非商业可访问性会授予商业权利。
- packaged CLI artifact 使用 version、commit 和 build-date ldflags 构建，且 `adp version` 报告符合预期。
- README 和 focused docs 描述当前 CLI surface，且没有 Web、UI、SaaS、cloud sync、hosted tracker、hosted orchestration、automatic Git execution、automatic task closure、provider-native resume 或 project-root report export 偏移。
- 活跃开发阶段在下一阶段开始前，已有 acceptance、commit 和 successful push 的本地证据，并且 `adp phase status --workspace <name> --format json` 同意下一 planned phase 可以启动。
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
