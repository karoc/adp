# 发布采用证据

English: [release-adoption-evidence.md](release-adoption-evidence.md)

本文档记录已发布 ADP artifacts 的 post-publish adoption evidence。它是 operator evidence note，不是 hosted release ledger、SaaS workflow、provider credential check，也不能替代必跑 release gate。ADP 的默认验证仍然保持 provider-free。

## v1.0.1 Fresh Post-Publish Adoption Smoke

证据日期：

- UTC：2026-07-06T22:56:28Z
- 本地 operator 日期：2026-07-07

范围：

- Release：`v1.0.1`
- Artifact：`adp-1.0.1-linux-amd64`
- Artifact 来源：`https://github.com/karoc/adp/releases/download/v1.0.1/adp-1.0.1-linux-amd64`
- Checksum 来源：`https://github.com/karoc/adp/releases/download/v1.0.1/SHA256SUMS`
- 执行环境：全新的临时 `HOME`、`ADP_HOME`、`ADP_RUNTIME_DIR`、`PATH` 和 project root
- Binary 来源：下载的 release artifact，安装到临时目录后运行
- Provider 模式：只使用 fake local Codex command；没有真实 provider credentials、model invocation、quota 或联网 provider session

Fresh-environment 证明：

- 使用 `mktemp -d` 创建了新的临时根目录。
- Release artifact 和 `SHA256SUMS` 下载到了该临时根目录中。
- 下载得到的 artifact 以 `adp` 名称安装到临时 install directory。
- `PATH` 被设置为临时 install directory 位于系统路径之前。
- 第一次执行 `adp` 命令前，`HOME`、`ADP_HOME` 和 `ADP_RUNTIME_DIR` 都指向临时根目录内部。
- `command -v adp` 解析到临时 install directory，而不是 repository checkout、`dist/` 或 source-built binary。

Artifact 验证：

```text
adp-1.0.1-linux-amd64: OK
sha256: 9fba1e473e5997124f92e1e43377d7132bc2766a1c71788a937e8bae0bf2c1a4
```

Installed binary identity：

```text
/tmp/adp-p72-adoption.<redacted>/install-bin/adp
adp version 1.0.1
commit: c02033a45c7440bb116538dac6a00bb6c4ca864d
built: 2026-07-06T20:10:40Z
go: go1.25.7
platform: linux/amd64
```

Sanitized command sequence：

```bash
TMP_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/adp-p72-adoption.XXXXXX")
curl -fsSL https://github.com/karoc/adp/releases/download/v1.0.1/adp-1.0.1-linux-amd64 -o "$TMP_ROOT/download/adp-1.0.1-linux-amd64"
curl -fsSL https://github.com/karoc/adp/releases/download/v1.0.1/SHA256SUMS -o "$TMP_ROOT/download/SHA256SUMS"
(cd "$TMP_ROOT/download" && grep 'adp-1.0.1-linux-amd64$' SHA256SUMS | sha256sum -c -)
install -m 0755 "$TMP_ROOT/download/adp-1.0.1-linux-amd64" "$TMP_ROOT/install-bin/adp"
export HOME="$TMP_ROOT/home"
export ADP_HOME="$TMP_ROOT/adp-home"
export ADP_RUNTIME_DIR="$TMP_ROOT/runtime"
export PATH="$TMP_ROOT/install-bin:$TMP_ROOT/fake-bin:/usr/bin:/bin"
command -v adp
adp version
adp init
adp workspace add p72-adoption "$TMP_ROOT/project"
adp workspace doctor p72-adoption
adp phase add --workspace p72-adoption p72 "P72 adoption phase"
adp phase start --workspace p72-adoption p72
adp tasks add --workspace p72-adoption --phase p72 --priority high "Validate published v1.0.1 adoption"
adp run codex --workspace p72-adoption --task task-20260706-0001 -- --p72-adoption
adp events list --workspace p72-adoption --task task-20260706-0001 --limit 5
adp sessions list --workspace p72-adoption --agent codex --task task-20260706-0001
adp progress report --workspace p72-adoption --format markdown
adp plan doctor --workspace p72-adoption --format text
find "$TMP_ROOT/project" -maxdepth 2 \( -name AGENTS.md -o -name CLAUDE.md -o -name .codex -o -name .claude -o -name .adp-runtime.yaml -o -name planning -o -name tasks.yaml -o -name phases.yaml -o -name progress.jsonl \) -print
```

Provider-free workflow evidence：

- `adp init` 创建了新的临时 ADP home。
- `adp workspace add p72-adoption <temporary-project>` 注册了全新的临时 project root。
- `adp workspace doctor p72-adoption` 完成，并出现一个预期 warning，因为临时 project root 不是 Git worktree。该 warning 不阻塞 ADP 的 local-first 运行。
- `adp phase add` 和 `adp phase start` 在临时 workspace 中创建并启动了一个本地 phase。
- `adp tasks add` 在临时 planning ledger 中创建了 `task-20260706-0001`。
- `adp run codex --workspace p72-adoption --task task-20260706-0001 -- --p72-adoption` 通过 fake local Codex command 执行并以 `0` 退出。
- `adp events list` 显示该 task-bound fake Codex session 的 `run_started` 和 `run_finished`。
- `adp sessions list` 显示已完成的 task-bound Codex session。
- `adp progress report --format markdown` 在临时 workspace 上完成。
- `adp plan doctor --workspace p72-adoption --format text` 对临时 planning ledger 报告 `status: ok`、`error_count: 0`、`warning_count: 0`。
- 临时 project root 保持干净：没有 `AGENTS.md`、`CLAUDE.md`、`.codex/`、`.claude/`、`.adp-runtime.yaml`、`planning/`、`tasks.yaml`、`phases.yaml` 或 `progress.jsonl` 写入其中。

P72 范围说明：

P69 已经验证了广义的 release publication、downloaded assets、checksums、package boundaries、package install、fake provider handoff、events、sessions、progress 和 project-root cleanliness。P72 不替代也不扩展这类 package verification。本 note 记录的是一次从 public release URLs 开始、单独计时的 fresh post-publish adoption smoke，并隔离了 `HOME`、`ADP_HOME`、`ADP_RUNTIME_DIR`、`PATH`、install directory 和 project root。它仍然是 provider-free 的，不声明真实 Codex 或 Claude 模型可用性。

可选 real-agent evidence：

- Command availability：本 adoption note 未运行。
- 非交互真实模型 invocation：本 adoption note 未运行。
- 手工交互式 provider acceptance：本 adoption note 未运行。

剩余风险：

- 该 evidence 只在一个全新本地 operator environment 中验证了 `linux/amd64` artifact。它不验证其他平台、package managers、真实 provider credentials、billing、quota、model access、联网模型行为或交互式 provider session 质量。
