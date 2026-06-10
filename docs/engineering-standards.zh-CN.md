# 工程规范

English: [engineering-standards.md](engineering-standards.md)

本文档定义 ADP 贡献者和自动化 Agent 必须遵守的仓库级工程规则。

## 文件行数限制

项目代码文件必须控制在 700 个物理行以内。

如果代码文件将超过 700 行，合并前必须拆分。优先按稳定职责边界拆分：

- CLI command wiring vs command implementation。
- schema types vs validation logic。
- adapter registry vs concrete adapter implementation。
- runtime orchestration vs overlay materialization。
- runner process handling vs event logging。
- production code vs test helpers。

允许例外：

- 不手工编辑的生成文件；
- vendored third-party code；
- lockfiles 和机器生成 metadata；
- license files 和 long-form documentation。

手写代码的任何例外都需要在 pull request 中给出简短理由，并应视为临时状态。

交付前运行：

```bash
scripts/check-file-lines.sh
```

必跑检查默认限制为 700 行，当代码文件超过该硬限制时失败。本地实验可以覆盖：

```bash
MAX_FILE_LINES=700 scripts/check-file-lines.sh
```

规划拆分或 hardening 阶段前，可以运行非阻断 line pressure audit：

```bash
scripts/check-file-lines.sh --audit
```

Audit 会报告达到或超过 `LINE_PRESSURE_WARN_LINES` 的手写代码文件，默认阈值为 600，并且退出码保持为零。它只作为规划证据，不能替代必跑硬门禁。某个阶段需要更早规划拆分时，可以调整 warning threshold：

```bash
LINE_PRESSURE_WARN_LINES=550 scripts/check-file-lines.sh --audit
```

## 双语文档

默认文档语言是英文。

仓库维护的文档必须同时提供英文和简体中文：

- 英文默认文件使用基础文件名，例如 `README.md`、`docs/engineering-standards.md`。
- 简体中文对应文件使用 `*.zh-CN.md`，例如 `README.zh-CN.md`。
- 英文文档应链接到简体中文版本。
- 简体中文文档应链接到英文版本。

`LICENSE` 是英文权威法律文本。任何法律条款翻译只能作为说明，不能替代英文许可证。

交付前运行：

```bash
scripts/check-docs-bilingual.sh
```

## 许可证边界

ADP 使用 PolyForm Noncommercial License 1.0.0 公开提供非商业使用。

不要引入与非商业 source-available 分发模式冲突的第三方依赖。新增依赖时，需要记录引入原因，并确认其许可证允许被包含在本项目中。

所有源文件都应与仓库许可证兼容。新的 public-facing 文件应在合适位置保留 copyright 和 required notices。

贡献和依赖变更必须保留 [license-policy.zh-CN.md](license-policy.zh-CN.md) 中描述的授权边界。不要把 ADP 描述为不受限制的 open source，也不要暗示公开可访问会授予商业使用权。
