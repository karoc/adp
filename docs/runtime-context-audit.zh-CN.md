# Runtime Context 审计

English: [runtime-context-audit.md](runtime-context-audit.md)

本文档审计 ADP 在启动 Agent 时会让 Agent 看到的上下文。它只覆盖现有 terminal-first、local-first runtime 行为：ADP 准备本地 runtime overlay，启动本地外部命令，记录本地证据，并把权威 workspace 和 planning 状态保存在真实项目根目录之外。

它不会引入 Web UI、dashboard、SaaS tracker、cloud sync、hosted orchestration、automatic Git workflow、provider-native conversation resume，或 project-root report export path。

## 启动边界

完整启动形态是 `adp run <agent> --workspace <name> --profile <profile> --task <task-id> -- <agent-args>`。如果需要启动时原子领取任务，则用 `--take --owner <owner> [--lease 4h]` 替代 `--task <task-id>`。当普通 workspace 解析或 adapter 默认值已经足够时，workspace、profile 和 task 参数可以省略；但这个形态展示了所有会影响 Agent 可见上下文的输入。

启动时，ADP 会：

- 从 `--workspace`、`ADP_WORKSPACE`，或当前目录是否位于已注册 project root 内部来解析 workspace。
- 从 `$ADP_HOME/workspaces/<workspace>/workspace.yaml` 读取 workspace config。
- 从显式 `--profile`、workspace agent profile，或 `default` fallback 解析 adapter 和 selected profile。
- 当传入 `--task <task-id>` 时，从 `$ADP_HOME/workspaces/<workspace>/planning` 读取 task metadata。
- 当传入 `--take --owner <owner>` 时，在 runtime 创建前先在 planning lock 下原子领取下一个可领取 task，然后把该 task 作为 runtime-bound task 加载。
- 在 `$ADP_RUNTIME_DIR` 下构建临时 runtime root。
- 把 ADP-generated files 写入 runtime root。
- 把真实项目文件 symlink 到 runtime root，且 generated paths 优先。
- 以 runtime root 作为进程工作目录启动 agent command，并转发 `--` 之后的参数。

`adp env <workspace> --cd` 和 `adp enter <workspace>` 对 shell workflow 使用同一 runtime overlay 边界，但除非通过 `adp run` 启动 agent adapter，否则它们不会渲染 adapter-specific instruction files。

## Runtime 视图

一次 Codex 启动会看到类似这样的 runtime root：

```txt
$ADP_RUNTIME_DIR/<workspace>-<session>/
├── AGENTS.md
├── .adp-runtime.yaml
├── .codex/
│   └── config.toml
├── go.mod -> <project-root>/go.mod
└── internal -> <project-root>/internal
```

Claude 启动使用相同 runtime 模型，但看到的是 `CLAUDE.md` 和 `.claude/settings.json`，而不是 `AGENTS.md` 和 `.codex/config.toml`。

如果真实项目已经存在 `.codex/` 或 `.claude/` 等 provider-local configuration directories，ADP 会把非冲突子文件合并进 runtime overlay。例如，项目拥有的 `.claude/settings.local.json` 在 Claude runtime 中仍然可见，而 ADP 生成的 `.claude/settings.json` 会在这个精确路径上优先于项目文件。冲突会作为 runtime evidence 记录；真实项目目录不会被修改。

生成的 `.adp-runtime.yaml` manifest 会记录 ADP ownership 和 cleanup metadata：manifest version、session ID、workspace name、可选 task ID 和 title、project root、runtime root、创建时间、keep flag，以及 `generated_by: adp`。Runtime pruning 会先把该 manifest 作为 compatibility evidence，再删除 ADP-owned runtime 目录。

## Git 边界

Runtime overlay 会为 Agent workflow 暴露项目内容，但不会暴露仓库 Git metadata。`.gitignore`、`.gitattributes`、`.gitmodules` 等普通项目 Git 文件仍然是 project files，可以像其他非生成文件一样链接进 overlay；`.git` metadata 会从 runtime overlay 中排除。Runtime 中被链接的子路径仍可能是 symlink 背后的真实 project file 或 directory，因此在这些子路径中编辑文件仍可能影响真实项目；这不代表 runtime root 是 Git worktree。

ADP 还会在启动 Agent 前 neutralize 指向仓库的 Git environment variables，包括 `GIT_DIR`、`GIT_WORK_TREE`、`GIT_INDEX_FILE`、`GIT_OBJECT_DIRECTORY`、`GIT_ALTERNATE_OBJECT_DIRECTORIES`、`GIT_COMMON_DIR` 和 `GIT_NAMESPACE`。这些值可能把 Git 重定向到其他 worktree、index、object store、common directory 或 namespace，因此会在 runtime 边界被移除。普通 shell environment 和 auth-related variables 会被保留。

Runtime handle 和 `.adp-runtime.yaml` 会在 ADP 能从配置的 project root 发现 repository root 时记录 `ADP_GIT_ROOT`/`git_root`，并通过 `git_metadata_skipped: true` 明确说明 Git metadata 被省略。Runtime environment 还会把 runtime root 加入 `GIT_CEILING_DIRECTORIES`，避免 Git discovery 从 `$ADP_RUNTIME_ROOT` 向父目录继续查找并误把 overlay 当作权威 worktree。当配置的 project root 是更大 worktree 的子目录时，`ADP_GIT_ROOT` 可能不同于 `ADP_PROJECT_ROOT`。

`$ADP_RUNTIME_ROOT` 不是权威 Git worktree。需要 Git inspection 或 mutation 的 Agent 和 operator 应从真实 project root 执行 Git，可以使用 `git -C "$ADP_PROJECT_ROOT" ...`，也可以先 `cd "$ADP_PROJECT_ROOT"`。`adp env <workspace> --cd` 和 shell-hook 输出可能会在 export ADP runtime environment 前，为危险 Git variables 生成 `unset` 命令。Workspace diagnostics 可能针对真实 project root 运行只读 Git inspection，包括 topology discovery、`.git` metadata 形态，以及 `git status --porcelain=v2 --branch` 等 status 检查。这些 diagnostics 可以报告 nested project roots、linked worktree 或 submodule 使用的 gitfile metadata、status 不可用，或 dirty status，但不会执行 commit、push、stage、checkout、cleanup，或任何其他 Git mutation。ADP 可以在本地 planning ledger 中记录显式的 phase commit 和 push evidence，但不会 wrap 或自动运行 Git。

## 指令文件

生成的 instruction files 是主要的人类可读上下文表面：

- Codex 会收到 `AGENTS.md`。
- Claude 会收到 `CLAUDE.md`。

两个文件都由同一个 ADP renderer 生成。可见 section 包括：

- Workspace metadata：workspace name、真实 project root、adapter name 和 effective profile。
- Current task：当绑定 task 时，包含 task ID、title、status、priority、phase、description 和 blocked reason。
- ADP 规划契约：ADP 仍然是权威本地 planning ledger，持久 task state 变更必须使用 ADP task 和 phase commands。
- Task lease 维护：长时间运行的 owner 通过 ADP 续租，过期的 in-progress claims 通过只读 stale-task commands 检查。
- 工具任务框桥接：provider 原生 todo 或 task panel 可以为了本地可见性镜像当前 ADP task，但不能成为事实源。
- 工具 Plan Mode 桥接：provider 原生 plan mode 可以组织 proposal，但只读 ADP plan preview 和明确批准后的 plan apply 才是持久 planning 路径。
- Base prompt：配置的 `prompts.base` 文件；如果没有可读文件，则使用本地 fallback message。
- Shared memory：memory 启用时读取配置的 `memory.shared` 文件；否则使用本地 disabled/missing fallback。
- Rules：来自 `workspace.yaml` 的按键排序 workspace rules。
- MCP：enabled server names 和配置的 MCP config file content；否则使用本地 disabled/missing fallback。
- Profile：effective profile name、agent enabled state、command summary、options，以及第一个匹配的 profile file。

Profile file lookup 会先使用非 `default` 的 effective profile，然后回退到 adapter profile file，例如 `profiles/codex.yaml` 或 `profiles/claude.yaml`。支持的 profile file 后缀是 `.md`、`.yaml`、`.yml` 和 `.json`。

这些 instruction files 是 runtime artifacts。正常 ADP workflow 不会把它们复制到真实项目根目录。

## Adapter Config 文件

Adapter config files 会在 instruction file 旁边生成，用于把 ADP runtime metadata 暴露给被启动的工具：

- Codex 会收到 `.codex/config.toml`。
- Claude 会收到 `.claude/settings.json`。

Codex metadata 包含一个 `[adp]` table，其中有 adapter name、workspace name、project root、effective profile、memory enabled state、MCP enabled state，以及绑定 task 时的 task fields。

Claude metadata 包含一个 `adp` JSON object，其中有 adapter name、workspace name、project root、effective profile、memory enabled state、MCP enabled state，以及绑定 task 时的 task object。

这些文件是 ADP metadata，不是任何外部 provider CLI 当前原生配置 schema 的完整声明。外部 CLI authentication、model selection、network behavior、tool permissions 和 prompt interpretation 仍由外部命令和本地 operator 负责。

不与这些精确生成路径冲突的项目 provider-local files 会被链接进 runtime overlay。这样可以保留既有本地 provider 配置，同时不允许项目文件覆盖 ADP runtime metadata。

## Profile、Prompt、Memory 与 MCP

Selected profile 会影响四个启动表面：

- 生成的 instruction file 中的 effective profile 行。
- 生成的 instruction file 中包含的 profile file content。
- 生成的 adapter metadata。
- `ADP_PROFILE` runtime environment variable。

Base prompt、shared memory、rules 和 MCP references 都从 `$ADP_HOME` 下的本地 ADP workspace 目录读取。ADP 把这些文件视为本地 context inputs。缺失、空文件、不可读文件或路径逃逸文件，会在生成的 instructions 中产生本地 fallback text，而不是导致 ADP 把状态复制进 project root。

生成的 instructions 中的 MCP 内容是给被启动 Agent 使用的 reference 和 configuration summary。Runtime context audit 不能把 MCP support 描述为 hosted orchestration 或 cloud sync。

## Task Metadata

当传入 `--task <task-id>` 时，ADP 会在启动 Agent 前从 workspace planning ledger 读取 task。缺失 task 会在 agent command 启动前失败。当传入 `--take --owner <owner>` 时，ADP 会先领取下一个可领取 task，并把该 task 绑定到 runtime；没有可领取 task 时不会构建 runtime，也不会启动 agent command。`--take` 与 `--task` 互斥。

Task metadata 会出现在：

- 生成的 instruction file 的 `Current Task` section。
- 生成的 adapter config file。
- Runtime environment variables。
- `.adp-runtime.yaml` manifest 中的 task ID 和 task title。
- 本地 `run_started` 和 `run_finished` events。
- 从本地 events 派生的 session history。

使用 `--task` 把 task 绑定到 runtime 不会自动 claim、complete、block、accept、commit 或 push 该 task 或 phase。使用 `--take` 启动时，会在 runtime 创建前记录 ownership 并把被选中的 task 推进到 `in_progress`，但仍然不会完成 task、accept phase、记录 commit 或 push evidence、运行 Git，或根据 provider exit code 推断成功。

## 规划契约与任务框桥接

生成的 instructions 会把 ADP planning contract 带入每个被启动的工具。Agent 应该把 ADP 视为 workspace planning 和 progress 的持久事实源，并使用 ADP 命令完成持久状态变更：

```bash
adp tasks next --workspace <workspace> --format json
adp tasks take --workspace <workspace> --owner <owner> --lease <duration> --format json
adp run <agent> --workspace <workspace> --take --owner <owner> --lease <duration> -- <agent-args>
adp tasks add --workspace <workspace> --phase <phase-id> --priority <priority> "<title>"
adp tasks claim --workspace <workspace> <task-id> --owner <owner> --lease <duration>
adp tasks renew --workspace <workspace> <task-id> --owner <owner> --lease <duration>
adp tasks stale --workspace <workspace> --format json
adp tasks update --workspace <workspace> <task-id> --status in_progress
adp tasks block --workspace <workspace> <task-id> --reason "<reason>"
adp tasks release --workspace <workspace> <task-id> --owner <owner>
adp tasks done --workspace <workspace> <task-id>
adp phase status --workspace <workspace> --format json
adp progress report --workspace <workspace> --format json
```

如果外部工具提供原生 task 或 todo panel，Agent 应该为了可见性把当前 ADP task 镜像到那里。对于 `adp run --take`，启动时被选择并领取的 task 就是 active ADP task。镜像可以包含 task ID、title、status、phase、owner 或 lease，以及本地 subtasks，但它只是工作视图。持久状态仍然属于 `$ADP_HOME/workspaces/<workspace>/planning/`。

除非 provider 暴露稳定的本地 API，否则这个 bridge 当前是 instruction-level。ADP 不能抓取 provider-private todo state、把 provider task panel 视为权威状态、根据 agent exit code 推断 completion、自动 accept phase，或自动运行 Git。

对于通过 `adp run --take` 启动的长时间 session，owner 应在 task lease 过期前续租。如果 session 意外中断，`adp tasks stale` 会以只读 recovery evidence 的形式暴露过期的 `in_progress` claim。其他 worker 只能通过 `tasks take` 或显式 `tasks claim` 等 ADP task ownership commands 接管它。

## 工具 Plan Mode 桥接

当被启动的 provider 工具支持 plan mode 时，该模式只是 proposal surface。Agent 可以用它组织和展示候选工作，但 plan-mode items 在通过 ADP 验证并写入前都只是 scratch state。

Plan-mode Agent 不应编辑 implementation files、把 tasks 标记为 done、accept phases、commit、push，或产生其他 execution side effects；除非用户明确批准从 planning 进入 execution。结构化 plan proposal 应先用只读命令检查：

```bash
adp plan preview --workspace <workspace> --file - --format json
```

在用户或 operator 明确批准后，同一份 proposal 才能写入 ADP：

```bash
adp plan apply --workspace <workspace> --file - --format json
```

Task ownership、status changes、blocker records 和 phase evidence 继续使用 ADP planning contract 中的 task 和 phase commands。如果 provider 在 `adp run --take` 后进入 plan mode，ADP task 已归属该 session，但 native plan items 在通过 ADP-approved flow 显式 apply 或执行前仍然只是 proposals。Provider 原生 plan panels 不能被视为权威 planning storage 或 recovery evidence。

## Runtime 环境变量

被启动的 Agent 进程会继承 parent shell environment，并收到 ADP runtime variables：

- `ADP_HOME`：本地 ADP home。
- `ADP_WORKSPACE`：selected workspace。
- `ADP_PROJECT_ROOT`：真实 project root。
- `ADP_GIT_ROOT`：可用时为已发现的 Git worktree root；对于 nested workspace root，它可能不同于 `ADP_PROJECT_ROOT`。
- `ADP_RUNTIME_ROOT`：临时 runtime root，也是进程工作目录。
- `ADP_SESSION_ID`：ADP runtime session ID。
- `ADP_AGENT`：adapter name。
- `ADP_PROFILE`：解析出 effective profile 时存在。
- `ADP_TASK_ID`：绑定的 task ID。
- `ADP_TASK_TITLE`：绑定的 task title。
- `ADP_TASK_STATUS`：绑定的 task status。
- `ADP_TASK_PRIORITY`：绑定的 task priority。
- `ADP_TASK_PHASE`：绑定的 task phase。

Task variables 只在 task-bound runs 中存在。Event logger 会清理字段名类似完整环境变量集合的 event fields，因此本地 session evidence 不应保存完整 shell environments。

## Event 与 Session 证据

ADP 会向 `$ADP_HOME/logs/events.jsonl` 追加本地 JSONL runtime evidence：

- `run_started` 记录 workspace、agent、profile、runtime path、project root、session ID、存在时的 task ID，以及非敏感 invocation snapshot。
- `run_finished` 记录 workspace、agent、profile、runtime path、project root、session ID、存在时的 task ID、exit code 和 duration。

Invocation snapshot 可以包含 schema version、转发给 agent 的参数、keep-runtime 选择、workspace resolution source、profile source、original current directory，以及 task snapshot。它不能包含 credentials、tokens、完整 environment variables、generated instructions、provider conversation state 或 project file contents。

Session commands 会读取这些本地 events：

- `adp events list`
- `adp sessions list`
- `adp sessions show <session-id>`
- `adp sessions restore-plan <session-id>`

`adp sessions restore-plan <session-id>` 是只读命令。它会在非敏感数据足够时打印建议的新本地启动命令；它不会启动 Agent、创建 runtime、追加 events、修改 task 或 phase 状态、写入 project root，或恢复 provider-native conversation。

## Project Root 干净性

真实 project root 必须保持干净。Runtime context 属于 `$ADP_RUNTIME_DIR`；workspace config、prompts、shared memory、MCP config、profiles、planning ledgers、events 和 sessions 属于 `$ADP_HOME`。

正常 ADP runtime 和 reporting 路径不能在真实 project root 中创建这些文件或目录：

- `AGENTS.md`
- `CLAUDE.md`
- `.codex`
- `.claude`
- `.adp-runtime.yaml`
- `planning`
- `tasks.yaml`
- `phases.yaml`
- `progress.jsonl`
- Markdown 或 JSON report exports

如果真实项目已经包含 ADP 为 generated runtime context 保留的路径，ADP-generated content 会在 runtime overlay 内优先，workspace doctor 命令会报告本地 diagnostics。除非 operator 明确要求编辑这些 project files，否则 ADP 不应通过修改真实 project root 来修复该情况。

## 审计证据

当前 runtime context 的默认证据由本地 smokes 覆盖：

- `scripts/runtime-smoke.sh --fake` 验证 fake Codex 和 Claude runtime launch、generated files、task context、environment variables、本地 events、sessions、restore planning、pruning 和 project-root cleanliness。
- `scripts/runtime-audit-smoke.sh` 扩展覆盖 CLI discovery、runtime entry points、task/phase/plan/progress flows、session views、JSON outputs 和 local-first boundaries。
- `scripts/runtime-context-smoke.sh` 聚焦 launch-time context：generated instruction files、adapter metadata files、selected profiles、base prompt、shared memory、MCP references、task metadata、runtime environment variables、本地 event/session evidence、workspace diagnostics 和 project-root cleanliness。

`scripts/runtime-context-smoke.sh` 保持确定性和本地化：temporary `ADP_HOME`、temporary `ADP_RUNTIME_DIR`、temporary project root、fake agents only、无网络、无真实 provider CLI 要求、不执行 Git、不依赖 hosted service，并且不产生 project-root report 或 planning exports。
