# 许可证策略

English: [license-policy.md](license-policy.md)

本文面向 operator 和 contributor 解释 ADP 的授权与贡献策略。它是操作说明，不能替代 [../LICENSE](../LICENSE) 中的权威许可证文本。

## 公共许可证

ADP 使用 [PolyForm Noncommercial License 1.0.0](../LICENSE) 提供公共非商业使用。仓库以 source-available 形式提供给学习、研究、评估和非商业开放协作。

公共许可证不授予商业使用权。任何商业使用都必须取得版权持有人的单独付费授权。公开源码、公开 fork、preview package 和非商业再分发都不授予商业权利。

ADP 没有使用 OSI-approved open-source license，因为公共许可证限制商业使用。描述公共授权模型时，请使用 `source-available` 或 `noncommercial source-available`。

## 必要声明

非商业再分发副本、fork、source archive 和 release package 必须保留：

- `LICENSE`；
- `LICENSE` 中的 `Required Notice:` 行；
- 打包公开文档时的 `COMMERCIAL.md` 和 `COMMERCIAL.zh-CN.md`；
- 对 ADP 和版权持有人的署名；
- 相关公开文档中的这一授权边界。

英文 `LICENSE` 文件是权威法律文本。翻译和摘要只作为说明。

## 商业授权

商业使用包括使用 ADP 提供付费产品、服务、咨询、托管、自动化、集成、支持、内部商业运营、客户交付、托管服务、专有集成、收入相关系统，或商业再分发。

商业授权独立于公共仓库处理。商业授权请通过 GitHub 仓库联系仓库所有者。

## 贡献

贡献必须兼容 ADP 现有授权模型。提交贡献即表示 contributor 确认自己有权贡献该内容，并且它可以作为 ADP 的一部分，按照仓库当前公共许可证和商业授权模型分发。

不要贡献会要求 ADP 采用不兼容许可证、移除非商业限制，或通过公共许可证授予商业权利的代码、文档、生成输出、数据集、示例或依赖。

依赖变更在接受前必须检查是否兼容非商业 source-available 分发模式。

## Release Packages

Release package 必须完整保留公共许可证和商业声明，且不得暗示 preview binary 已授予商业权利。Package 不得包含本地 `$ADP_HOME` 状态、`$ADP_RUNTIME_DIR` 内容、runtime overlay、日志、task state、credential、`.envrc`、`mvp.md` 或机器特定 shell 配置。

Package 细节见 [release-packaging.zh-CN.md](release-packaging.zh-CN.md)。
