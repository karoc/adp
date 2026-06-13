# 交互式输入库评估报告

**任务**: P2-1a - 为 `quickstart` 命令选择交互式输入库
**评估日期**: 2026-06-13  
**结论**: 推荐使用标准库（`bufio.Scanner`）

---

## 执行摘要

在评估了三个候选库（`promptui`、`survey/v2`、`huh`）后，最终选择使用 Go 标准库（`bufio.Scanner`）实现交互式输入。虽然 `huh` 库功能强大且现代化，但标准库方案更轻量、更可靠，完全满足 quickstart 命令的需求。

---

## 候选库评估

### 1. promptui

- **维护状态**: ⚠️ 停滞（最后更新 2021年）
- **优势**: API 简洁，轻量级
- **劣势**: 缺少原生非交互模式支持，维护停滞
- **评分**: 6.5/10

### 2. survey/v2

- **维护状态**: ⚠️ 已归档（2024年4月）
- **优势**: 功能丰富，跨平台支持好
- **劣势**: 原仓库已归档，依赖较重
- **评分**: 5.5/10

### 3. huh ⭐

- **维护状态**: ✅ 活跃（Charm 组织）
- **优势**: 
  - 零依赖核心
  - 原生非交互模式（`WithAccessible(true)`）
  - 现代化 API
  - 活跃维护
- **劣势**: 网络环境可能影响下载
- **评分**: 9/10

---

## 最终决策：标准库方案

考虑到实际开发环境的网络限制，最终采用 Go 标准库（`bufio.Scanner`）实现：

### 优势

1. **零外部依赖** - 无需下载第三方包
2. **可靠稳定** - 标准库长期维护
3. **代码简洁** - 实现清晰易懂
4. **完全满足需求** - quickstart 命令的交互需求不复杂

### 实现细节

```go
import (
    "bufio"
    "fmt"
    "os"
    "strings"
)

func promptForInput(prompt string, defaultValue string) (string, error) {
    if defaultValue != "" {
        fmt.Printf("%s [%s]: ", prompt, defaultValue)
    } else {
        fmt.Printf("%s: ", prompt)
    }
    
    scanner := bufio.NewScanner(os.Stdin)
    if !scanner.Scan() {
        return "", scanner.Err()
    }
    
    input := strings.TrimSpace(scanner.Text())
    if input == "" && defaultValue != "" {
        return defaultValue, nil
    }
    return input, nil
}

func promptYesNo(prompt string, defaultYes bool) (bool, error) {
    suffix := "(Y/n)"
    if !defaultYes {
        suffix = "(y/N)"
    }
    
    fmt.Printf("%s %s: ", prompt, suffix)
    scanner := bufio.NewScanner(os.Stdin)
    if !scanner.Scan() {
        return false, scanner.Err()
    }
    
    input := strings.ToLower(strings.TrimSpace(scanner.Text()))
    if input == "" {
        return defaultYes, nil
    }
    return input == "y" || input == "yes", nil
}
```

### 功能覆盖

✅ **文本输入** - 支持默认值  
✅ **是/否确认** - 支持默认选项  
✅ **路径验证** - 自定义验证函数  
✅ **非交互模式** - 通过命令行参数实现  

---

## 实施结果

在 `internal/cli/quickstart.go` 中使用标准库成功实现了：

1. **ADP home 初始化交互** (~100行)
   - 路径输入和验证
   - 已存在目录的确认
   - 错误处理和重试

2. **Workspace 创建交互** (~150行)
   - 工作区名称验证
   - 项目根路径验证
   - 可选配置选项（memory/MCP）
   - 自动诊断

3. **非交互模式** (~50行)
   - 完全参数化
   - 无用户交互
   - 适合脚本和 CI 环境

**总代码量**: ~500行（符合 <700行限制）  
**测试覆盖**: 单元测试 + 集成测试  
**文档**: 英文和中文完整文档

---

## 对比：标准库 vs huh

| 特性 | 标准库 | huh |
|------|--------|-----|
| 依赖数 | 0 | 0（核心） |
| 维护状态 | ✅ Go 团队 | ✅ Charm 组织 |
| 非交互模式 | 手动实现 | 原生支持 |
| 代码量 | ~500行 | ~300行（估计） |
| TUI 体验 | 基础 | 现代化 |
| 可靠性 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ |
| 学习曲线 | 平缓 | 中等 |

---

## 结论

标准库方案完全满足 `quickstart` 的需求：

✅ **功能完整** - 所有必需的交互已实现  
✅ **代码清晰** - 易于维护和扩展  
✅ **零依赖** - 无外部包引入  
✅ **测试充分** - 单元测试和集成测试覆盖  
✅ **文档完善** - 英文和中文文档齐全

如果未来需要更丰富的 TUI 体验（如表格选择、多步骤表单等），可以考虑迁移到 `huh` 库，但当前阶段标准库方案是最优选择。

---

**参考文档**:
- [Go bufio 包文档](https://pkg.go.dev/bufio)
- [ADP Quickstart 实现](../../internal/cli/quickstart.go)
- [ADP Quickstart 测试](../../internal/cli/quickstart_test.go)
