# 贡献指南

English: [CONTRIBUTING.md](CONTRIBUTING.md)

感谢你投入时间改进 ADP。本项目保持 terminal-first、local-first，并以 source-available 形式提供非商业使用。贡献应保留这一产品边界，以及仓库当前的 PolyForm Noncommercial 授权模型。

## 范围

合适的贡献包括改进本地 CLI workflow、runtime overlay、workspace registry、adapter、shell integration、本地 event/session evidence、diagnostics、task 和 phase ledger、release gate、测试、示例或双语文档。

避免把 ADP 推向 Web dashboard、SaaS tracker、cloud sync service、hosted orchestration platform、图形化多 Agent 产品、自动 Git 执行器、provider-private state scraper，或 provider-native conversation resume 实现。

## 许可证边界

ADP 在 [PolyForm Noncommercial License 1.0.0](LICENSE) 下提供。商业使用必须取得版权持有人的单独付费授权。公开可访问不代表已经授予商业权利。

提交贡献即表示你确认自己有权贡献该内容，并且它可以作为 ADP 的一部分按仓库现有授权模型分发。不要提交与非商业 source-available 分发模式冲突的代码、文档、生成输出或依赖变更。

详情见 [docs/license-policy.zh-CN.md](docs/license-policy.zh-CN.md) 和 [COMMERCIAL.zh-CN.md](COMMERCIAL.zh-CN.md)。

## 开发规则

提交或交接变更前：

- 代码文件保持在 700 个物理行以内。
- 英文文档作为默认版本；维护中的 Markdown 文件必须增加等价的简体中文 `*.zh-CN.md` counterpart。
- `.envrc` 和 `mvp.md` 保持 ignored，不提交。
- ADP runtime 和 planning state 必须留在真实项目根目录之外。
- 不配置仓库本地 Git `user.name` 或 `user.email`。
- 保持默认测试 provider-free；真实 Codex 和 Claude 检查必须显式 opt-in。

运行标准门禁：

```bash
scripts/check-all.sh
```

开发窄范围变更时可以先运行聚焦本地检查，但提交或交接前仍必须运行完整门禁。

## 规划与交接

ADP 自身开发 dogfood ADP 的本地 task 和 phase ledger。规划中的实现切片应登记为 ADP phase 和 task，执行状态保存在 `$ADP_HOME` 下，并在启动下一阶段前记录 acceptance、commit evidence 和 push evidence。

使用多 Agent 时，应分配互不重叠的写入范围，让 ADP 成为权威任务看板，审阅每个返回 diff，并在集成后运行全仓门禁。

完整贡献者和 Agent 操作契约见 [AGENTS.zh-CN.md](AGENTS.zh-CN.md)。
