# Local Replay Proposal

English: [local-replay-proposal.md](local-replay-proposal.md)

状态：P77 proposal 已由 P78 更新。P78 只实现只读 dry-run preflight；execute mode 仍是未来工作。

当前 baseline：ADP 已有 `adp sessions list`、`adp sessions show`、`adp sessions restore-plan`、`adp sessions resume-plan` 和 `adp sessions replay <session-id> --dry-run [--workspace <name>] [--owner <owner>] [--lease <duration>] [--agent <agent>] [--format <text|json>]`。当前代码树中没有已实现的 replay execute mode。

## 建议

ADP 应继续把 local replay 构建为 `adp sessions resume-plan` 的窄范围配套能力。价值是明确存在的：operator 已经会使用 ADP session evidence 复制建议的 `adp run ...` 命令、检查 task ownership，并启动新的 worker。P78 通过在 `adp sessions replay` 下增加一等 dry-run preflight，先减少一部分摩擦；后续 execute mode 仍可进一步减少复制粘贴错误，并让新运行明确关联到之前的 ADP session。

这个能力不能实现成 automatic provider resume。它只能基于 ADP-owned local evidence 启动新的 ADP runtime 和新的 provider process。现有 `resume-plan` 命令必须继续保持只读。

建议方向：

- 保留 `adp sessions resume-plan` 作为 inspection 和 proposal command。
- 保留 `adp sessions replay <session-id> --dry-run` 作为 inspection-only local replay preflight。
- 只有在后续已验收阶段中，才增加 replay execution。
- Task ownership 行为必须显式。命令不能静默 claim、renew、release、complete 或 block 任务。
- 第一版 MVP 不做 task mutation。需要 ownership 变化时，operator 必须先显式运行 `adp tasks claim` 或 `adp tasks renew`。
- Provider-native conversation state 不在范围内。
- Redacted 或不完整 invocation data 应作为 replay blocker，而不是靠猜测补齐。

## 问题

`resume-plan` 当前回答“下一步应该运行什么？”它有意保持只读，并且可以输出带 `inspect`、`task_mutation`、`runtime_creation` 等 side-effect labels 的建议命令。

剩余摩擦在于从 plan review 和 dry-run preflight 到 execution 的交接：

- Operator 必须从 text 或 JSON output 中复制命令。
- Operator 必须单独判断是否需要 task claim、renew 或 stale reclaim。
- 新运行除了人工 note 外，不会明确关联到 source session。
- Same-tool rerun 可以复用安全的 invocation shape，但 cross-tool rerun 会有意省略 provider-specific profile 和 agent arguments。
- P78 dry-run 可以证明 local replay preflight 是否 ready，但它仍然拒绝启动 runtime。

显式 local replay 命令可以降低这个转换过程的出错概率，同时保留现有 local-first 和 explicit-mutation 契约。

## 现有契约

当前实现已经具备所需基础：

- `adp run` 会在启动 adapter 前记录 `run_started` event，并在结束后记录 `run_finished` event。
- `run_started` 包含非敏感 `fields.invocation` snapshot，其中有 schema version、redacted agent args、`keep_runtime`、workspace resolution、profile source、task binding source，以及可用时的 task snapshot。
- `adp sessions restore-plan` 可以从一个 session 重建只读 rerun command。
- `adp sessions resume-plan` 会把 session evidence 与当前 task、lease、phase、owner 和 target-agent context 组合起来。
- `resume-plan` 会用 `side_effect` 标记建议命令，让调用方区分 inspection、task mutation 和 runtime creation。
- `adp sessions replay <session-id> --dry-run` 会从同一个 resume plan 构建，并在不启动 Agent、不修改本地状态的前提下报告 replay readiness。

这些组件足以把 replay 构建为“plan、validate，然后执行一次新的本地运行”。它们不足以恢复 provider-private conversations、恢复未记录的 environment variables、重放隐藏 shell state，或重建被有意 redacted 的 secrets。

## Implemented Dry-Run MVP And Future Execute

已实现 dry-run 命令形态：

```bash
adp sessions replay <session-id> \
  --dry-run \
  [--workspace <name>] \
  [--agent <agent>] \
  --owner <owner> \
  [--lease <duration>] \
  [--format text|json]
```

未来 execute 候选形态：

```bash
adp sessions replay <session-id> \
  --execute \
  [--workspace <name>] \
  [--agent <agent>] \
  --owner <owner> \
  [--lease <duration>] \
  [--format text|json]
```

Dry-run 现在是已接受的 P78 contract。裸的 `adp sessions replay <session-id>` 不会执行；它会失败并要求 `--dry-run`。`adp sessions replay <session-id> --execute` 也会失败，因为 execute mode 在本阶段有意不实现。后续实现阶段必须先通过 command metadata、help examples、completion、tests 和 smoke coverage 重新确认 execute flags，才可以启动任何东西。

Dry-run 行为：

- 构建与 `sessions resume-plan` 相同的内部 plan。
- 拒绝 redaction placeholders 和缺失 launch fields。
- 拒绝 workspace-only replay 和 cross-workspace replay。
- 拒绝 stale、unowned、blocked、closed 或其他不可运行的 task state。
- 需要 ownership action 时，停止并输出所需的显式 ADP task command，而不是修改 task state。
- Plan ready 时，打印未来 execute mode 会使用的 exact task preflight decision 和 launch command。
- 包含 `source_session_id`、`mode`、`status`、`plan_status`、`task_preflight`、`launch_command`、`required_commands`、`blockers`、`executed_commands`、`read_only`、`would_mutate_task`、`would_create_runtime`、`provider_native_resume`、`git_side_effects` 和 `project_root_writes_by_adp` 等 JSON fields。
- 永远不追加 events、不创建 runtimes、不修改 tasks 或 phases、不运行 Git，也不写入 project root。

可能的后续扩展：

- Execute mode 可以拒绝启动，除非 plan 为 `ready` 且包含 runtime-creation command。
- Execute mode 可以通过与 `adp run` 相同的路径启动新的本地 ADP runtime，为新 session 生成正常的 `run_started` 和 `run_finished` events，并增加 `replay_source_session_id` 等 replay source metadata。
- Post-MVP design 可以考虑显式 `--renew` 或 `--claim` replay flags。
- 这类 flags 会把 task mutation 与 runtime creation 组合到一起，因此应作为单独 phase 评审。
- Dry-run JSON 需要把这些步骤与 runtime creation 分开分类，并复用 `resume-plan` 的 `task_mutation` 和 `runtime_creation` side-effect vocabulary。

## Non-MVP

第一版 replay execute 实现不应包含：

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
- `replay --dry-run` 永远保持只读。
- Replay execute 一旦实现，就是执行型命令，必须按执行型命令记录和说明。
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

## Validation

P78 dry-run validation 覆盖只读 preflight。Future execute work 在验收前应增加 focused unit 和 smoke coverage：

- `sessions replay` flags 和 invalid combinations 的 parser tests。
- Ready、partial、blocked、stale、unowned、same-owner、different-owner 和 closed-task cases 的 resume planner 或 replay preflight tests。
- 证明 `--dry-run` 只读的 tests：不修改 task、不修改 phase、不追加 event、不创建 runtime、不写 project-root，并且没有 Git side effects。
- 证明 MVP replay 拒绝修改 task ownership，并提示 operator 显式运行 `adp tasks claim` 或 `adp tasks renew` 的 tests。
- 证明 default replay 会拒绝 redacted agent args 和 incomplete invocation snapshots 的 tests。
- Future execute tests 证明 replay 会创建新 session，而不是 attach 到旧 provider conversation。
- 使用 fake Codex 和 fake Claude 的 runtime smoke coverage。
- 覆盖 help text、JSON output、side-effect fields 和 read-only dry-run behavior 的 runtime audit smoke。
- 双语 docs 和 command metadata examples。
- Phase acceptance 前运行完整 `scripts/check-all.sh`。

## Open Questions

- Dry-run 命令名已确定：`adp sessions replay`。
- Execute mode 是否应要求 interactive confirmation，除非传入 `--yes`？
- Workspace-only replay 是否继续推迟，直到 task-bound replay 证明稳定？
- Replay source metadata 应作为 `run_started` field，还是使用专用 `replay_started` event？
- 未来命令是否允许 operator 为 redacted values 提供 replacement arguments，还是这种情况始终要求 operator 手动运行 `adp run`？
- Post-MVP replay 是否应增加 `--renew` 或 `--claim` 等显式 task-mutation shortcuts，还是 ownership 始终留在 replay 外部？
