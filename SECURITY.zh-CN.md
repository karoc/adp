# 安全

English: [SECURITY.md](SECURITY.md)

ADP 是本地 CLI 和 runtime overlay 工具。安全报告应聚焦本地执行边界、project-root cleanliness、凭据处理、runtime isolation、package integrity，以及与外部 Agent CLI 的不安全交互。

## 报告方式

请通过 GitHub 仓库所有者联系方式私下报告安全问题。不要在公开 issue 中包含 exploit 细节、secret、私有日志、provider token、API key 或专有项目内容。

请提供足够的非敏感复现信息：

- ADP commit 或 version。
- 操作系统和 shell。
- 涉及的 ADP 命令。
- 已脱敏的 workspace 配置。
- 是否涉及真实 Codex 或 Claude CLI。
- 预期行为和实际行为。
- 确认报告中已经移除 credential、token、私有 prompt 和专有代码。

## 支持范围

main 分支和当前 preview artifact 会获得安全关注。较旧的 preview artifact 可能很快被替代；可行时请在当前 main 分支上复现。

范围内：

- ADP 意外把 runtime 或 planning 文件写入真实 project root。
- `$ADP_HOME`、`$ADP_RUNTIME_DIR`、runtime overlay、event log、session、task ledger 或 release package 的不安全处理。
- 意外捕获或展示 credential、token、API key、完整环境变量、provider-private conversation identifier 或专有项目内容。
- 默认测试或 release gate 在没有显式 opt-in 时访问真实 provider。
- Package 内容包含本地状态、日志、credential、机器特定 shell 配置或 runtime overlay。

不属于 ADP 负责范围：

- Provider 账号被攻破。
- Provider model 行为、quota、网络可用性或计费。
- Codex、Claude、shell、Git、Go、操作系统或第三方工具中的漏洞；除非 ADP 集成方式制造了暴露面。
- 用户主动启动的本地 Agent 读取公开项目文件。

## Operator Safety

默认 smoke tests 是 provider-free，并使用 fake local agents。真实外部 CLI 检查必须通过文档化的 opt-in 环境变量显式启用。ADP 可以验证本地 launch wiring 和 evidence collection，但不能保证 provider credentials、model access、quota、network behavior 或 interactive provider behavior。

不要把 secret 粘贴到 issue、task description、progress report、prompt、生成的 runtime file 或公开 evidence note 中。
