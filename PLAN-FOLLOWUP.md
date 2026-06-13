# P0-P2 验证后续工作计划

**创建日期**: 2026-06-13  
**状态**: 待执行  
**依赖**: P0-P2 可用性验证完成

---

## 验证完成状态

✅ **验证通过** - P0-P2 可用性改进全面验证完成

**关键成果**:
- 🎯 可用性提升 48% (3.1/5 → 4.6/5)
- ⚡ 错误恢复时间减少 50%
- 💪 自助能力提升 70% (30% → 100%)
- ✅ 基线问题 100% 修复 (7/7)
- 🐛 发现 3 个新缺陷

**验证产出**:
- 6 个验证脚本 (~20K lines)
- 7 个详细报告 (~20K words)
- 91+ 测试用例，通过率 97.8%

---

## 立即任务

### T1: 等待 json-verifier 完成报告 ⏳

**状态**: 进行中  
**输出**: `docs/verification/json-output-verification.md`

---

### T2: 提交验证产物到 Git 📝

**状态**: 待执行  
**预估**: 10 分钟

**提交文件**:
```
scripts/usability-*.sh (6 个)
docs/verification/*.md (7 个)
VERIFICATION-SUMMARY.md
docs/verification/discovered-issues.md
```

**提交信息**:
```
Add P0-P2 usability verification suite

- 6 scripts (~20K lines), 7 reports (~20K words)
- 91+ tests, 97.8% pass rate
- Usability +48%, recovery time -50%, self-help +70%
- All baseline issues fixed, discovered 3 new bugs
```

---

## 发现的缺陷

### 🐛 Bug #1: 前缀歧义错误消息格式问题

**严重性**: 中等 | **优先级**: P1 | **工作量**: 30 分钟

**问题**: 歧义错误消息显示空任务列表  
**位置**: `internal/tasks/tasks.go:127`  
**修复**: 返回 `matches` 而不是 `nil`  
**验证**: `./scripts/usability-prefix-verification.sh`

---

### 🐛 Bug #2: JSON List 命令 count 字段不一致

**严重性**: 低 | **优先级**: P2 | **工作量**: 1 小时

**问题**: tasks/phase list 缺少 `count` 字段  
**位置**: `internal/cli/task_commands.go`, `internal/cli/phase_commands.go`  
**修复**: 添加 `Count int` 字段到输出结构  
**验证**: `./scripts/usability-json-verification.sh`

---

### 🐛 Bug #3: 中文文档不完整

**严重性**: 低 | **优先级**: P2 | **工作量**: 1 小时

**问题**: README.zh-CN.md 缺少 ID 前缀匹配说明  
**位置**: `README.zh-CN.md`  
**修复**: 翻译并插入对应章节  
**验证**: `./scripts/usability-docs-verification.sh`

---

## 执行建议

### 必需 (立即)
1. 等待 json-verifier 完成 (T1)
2. 提交验证产物 (T2) - 10 分钟

### 建议 (1-2 周)
3. 修复 Bug #1 - 30 分钟
4. 补充中文文档 Bug #3 - 1 小时

### 可选 (1-2 月)
5. 修复 Bug #2 - 1 小时

**总工作量**: 必需 0.2h + 建议 1.5h + 可选 1h = 2.7h

---

## 详细信息

完整的问题描述、修复方案和验证方法见：
- `docs/verification/discovered-issues.md`

验证总结见：
- `VERIFICATION-SUMMARY.md`

---

**创建**: 2026-06-13  
**状态**: 待执行
