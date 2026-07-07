# Local Replay Proposal

English: [local-replay-proposal.md](local-replay-proposal.md)

状态：P77 proposal only。本文档没有实现 local replay 命令。

当前 baseline：ADP 已有 `adp sessions list`、`adp sessions show`、`adp sessions restore-plan` 和 `adp sessions resume-plan`。当前代码树中没有 `adp sessions replay` 命令。

## 建议

ADP 可以考虑未来增加一个显式的 local replay 命令，但它只能作为 `adp sessions resume-plan` 的窄范围执行型配套能力。价值是明确存在的：operator 已经会使用 ADP session evidence 复制建议的 `adp run ...` 命令、检查 task ownership，并启动新的 worker。一个一等的 local replay 命令可以减少复制粘贴错误，并让新运行明确关联到之前的 ADP session。

这个能力不能实现成 automatic provider resume。它只能基于 ADP-owned local evidence 启动新的 ADP runtime 和新的 provider process。现有 `resume-plan` 命令必须继续保持只读。

建议方向：

- 保留 `adp sessions resume-plan` 作为 inspection 和 proposal command。
- 只有在后续已验收阶段中，才增加单独的执行型命令，候选位置是 `adp sessions replay`。
- Task ownership 行为必须显式。命令不能静默 claim、renew、release、complete 或 block 任务。
- 第一版 MVP 不做 task mutation。需要 ownership 变化时，operator 必须先显式运行 `adp tasks claim` 或 `adp tasks renew`。
- Provider-native conversation state 不在范围内。
- Redacted 或不完整 invocation data 应作为 replay blocker，而不是靠猜测补齐。

## 问题

`resume-plan` 当前回答“下一步应该运行什么？”它有意保持只读，并且可以输出带 `inspect`、`task_mutation`、`runtime_creation` 等 side-effect labels 的建议命令。

剩余摩擦在于从 plan review 到 execution 的交接：

- Operator 必须从 text 或 JSON output 中复制命令。
- Operator 必须单独判断是否需要 task claim、renew 或 stale reclaim。
- 新运行除了人工 note 外，不会明确关联到 source session。
- Same-tool rerun 可以复用安全的 invocation shape，但 cross-tool rerun 会有意省略 provider-specific profile 和 agent arguments。

显式 local replay 命令可以降低这个转换过程的出错概率，同时保留现有 local-first 和 explicit-mutation 契约。

## 现有契约

当前实现已经具备所需基础：

- `adp run` 会在启动 adapter 前记录 `run_started` event，并在结束后记录 `run_finished` event。
- `run_started` 包含非敏感 `fields.invocation` snapshot，其中有 schema version、redacted agent args、`keep_runtime`、workspace resolution、profile source、task binding source，以及可用时的 task snapshot。
- `adp sessions restore-plan` 可以从一个 session 重建只读 rerun command。
- `adp sessions resume-plan` 会把 session evidence 与当前 task、lease、phase、owner 和 target-agent context 组合起来。
- `resume-plan` 会用 `side_effect` 标记建议命令，让调用方区分 inspection、task mutation 和 runtime creation。

这些组件足以把 replay 构建为“plan、validate，然后执行一次新的本地运行”。它们不足以恢复 provider-private conversations、恢复未记录的 environment variables、重放隐藏 shell state，或重建被有意 redacted 的 secrets。

## Proposed MVP

候选命令形态：

```bash
adp sessions replay <session-id> \
  --dry-run \
  [--workspace <name>] \
  [--agent <agent>] \
  --owner <owner> \
  [--lease <duration>] \
  [--format text|json]

adp sessions replay <session-id> \
  --execute \
  [--workspace <name>] \
  [--agent <agent>] \
  --owner <owner> \
  [--lease <duration>] \
  [--format text|json]
```

这是候选 API，不是已经接受的实现契约。裸的 `adp sessions replay <session-id>` 不应默认执行。后续实现阶段应通过 command metadata、help examples、completion、tests 和 smoke coverage 确认最终 flag names。

MVP 行为：

- 构建与 `sessions resume-plan` 相同的内部 plan。
- 除非 plan 是 `ready`，否则拒绝执行。
- 如果 plan 没有 `runtime_creation` command，则拒绝执行。
- 如果 invocation data 包含 redaction placeholders 或缺少 launch fields，则拒绝执行。
- 默认拒绝 workspace-only replay，除非 operator 显式允许。
- Source session 绑定 task 时必须提供 `--owner`。
- Task-bound replay 只有在当前 task 已由 `--owner` 持有且 lease 未 stale 时才启动。
- 如果 task unowned、stale、由其他 owner 持有、closed 或 blocked，则停止并输出所需的显式 ADP task command，而不是修改 task state。
- MVP 不 claim、renew、release、complete、block 或 update tasks。
- 通过与 `adp run` 相同的路径启动新的本地 ADP runtime。
- 为新 session 生成正常的 `run_started` 和 `run_finished` events。
- 在新 run evidence 中增加 replay source metadata，例如 `replay_source_session_id`，但不存储 provider-native state。

Dry-run 行为：

- `--dry-run` 必须保持只读。
- 它应打印将要执行的 task preflight decision 和 launch command。
- JSON dry-run output 应包含 `read_only: true`、`would_mutate_task: false`、`would_create_runtime` 和 source session ID。
- 它不得追加 events、创建 runtime、修改 tasks 或 phases、运行 Git，或写入 project root。

可能的后续扩展：

- Post-MVP design 可以考虑显式 `--renew` 或 `--claim` replay flags。
- 这类 flags 会把 task mutation 与 runtime creation 组合到一起，因此应作为单独 phase 评审。
- Dry-run JSON 需要把这些步骤与 runtime creation 分开分类，并复用 `resume-plan` 的 `task_mutation` 和 `runtime_creation` side-effect vocabulary。

## Non-MVP

第一版 replay 实现不应包含：

- Provider-native Codex 或 Claude conversation resume。
- Provider-private session handle transfer。
- Provider transcript scraping 或 replay。
- 自动把 native task 或 plan panels 当作 recovery evidence。
- Automatic task completion、phase acceptance、commit evidence、push evidence 或 Git execution。
- 第一版 MVP 中的 task claim、renew、release、complete、block 或 update shortcuts。
- Automatic stale reclaim。
- 复制 provider-specific profile 或 agent arguments 的 cross-workspace replay。
- 重放完整 environment variables、shell history、generated adapter instructions、project file contents 或 secrets。
- Batch replay、daemon replay、scheduled replay 或 hosted orchestration。
- Web UI、dashboard、SaaS tracker、cloud sync 或 remote issue-service integration。

## Safety Rules

Local replay 必须保持这些不变量：

- 权威 task 和 phase ledger 留在 `$ADP_HOME`。
- Runtime artifacts 留在 `$ADP_RUNTIME_DIR`。
- 真实 project roots 保持干净，除非启动后的 Agent 有意修改工作文件。
- `resume-plan` 永远保持只读。
- Replay 是执行型命令，必须按执行型命令记录和说明。
- 如果未来 replay 扩展允许 task mutation，每一个 mutation 都必须通过 flags 和 output 显式呈现。
- 不允许 phase mutation。
- 不允许运行 Git command。
- 不暗示 provider-native resume。
- Redaction 是硬边界。如果 ADP 记录了 `***REDACTED***`，replay 必须停止，并要求 operator 用替换值显式运行 `adp run ...` command。
- Partial session data 应输出清晰错误和 `resume-plan` 建议，而不是 best-effort launch。

## Suggested Output

Text output 应短小并可审计：

```text
source_session: session-20260707-0001
status: ready
mode: dry_run
task_preflight: task is owned by reviewer and lease is valid
runtime: will create a new ADP runtime
provider_native_resume: false
git_side_effects: false
project_root_writes_by_adp: false
launch: adp run codex --workspace game-a --task task-20260707-0003 -- --example-smoke
```

JSON output 应尽量复用 `resume-plan` structure，并增加 execution-specific fields：

- `source_session_id`
- `plan_status`
- `mode`
- `task_preflight`
- `executed_commands`
- `new_session_id`，在 launch 启动后出现
- `side_effects`
- `guarantees`

## Future Implementation Validation

后续实现阶段应在验收前增加 focused unit 和 smoke coverage：

- `sessions replay` flags、invalid combinations 和 required `--owner` behavior 的 parser tests。
- Ready、partial、blocked、stale、unowned、same-owner、different-owner 和 closed-task cases 的 resume planner 或 replay preflight tests。
- 证明 `--dry-run` 只读的 tests：不修改 task、不修改 phase、不追加 event、不创建 runtime、不写 project-root，并且没有 Git side effects。
- 证明 MVP replay 拒绝修改 task ownership，并提示 operator 显式运行 `adp tasks claim` 或 `adp tasks renew` 的 tests。
- 证明 default replay 会拒绝 redacted agent args 和 incomplete invocation snapshots 的 tests。
- 证明 replay 会创建新 session，而不是 attach 到旧 provider conversation 的 tests。
- 使用 fake Codex 和 fake Claude 的 runtime smoke coverage。
- 覆盖 help text、JSON output、side-effect fields 和 read-only dry-run behavior 的 runtime audit smoke。
- 双语 docs 和 command metadata examples。
- Phase acceptance 前运行完整 `scripts/check-all.sh`。

## Open Questions

- 命令应命名为 `adp sessions replay`，还是暴露 `run --from-session <session-id>` 形态？
- 第一版实现是否应要求 interactive confirmation，除非传入 `--yes`？
- Workspace-only replay 是否应纳入 MVP，还是等 task-bound replay 证明稳定后再考虑？
- Replay source metadata 应作为 `run_started` field，还是使用专用 `replay_started` event？
- 未来命令是否允许 operator 为 redacted values 提供 replacement arguments，还是这种情况始终要求 operator 手动运行 `adp run`？
- Post-MVP replay 是否应增加 `--renew` 或 `--claim` 等显式 task-mutation shortcuts，还是 ownership 始终留在 replay 外部？
