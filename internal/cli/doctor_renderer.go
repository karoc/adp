package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/karoc/adp/internal/output"
	"github.com/karoc/adp/internal/workspace"
)

// DoctorTextRenderer 渲染 doctor 诊断结果为人类可读格式
type DoctorTextRenderer struct {
	writer  io.Writer
	verbose bool
}

// Render 渲染所有诊断报告
func (r *DoctorTextRenderer) Render(reports []workspace.DiagnosticReport) error {
	for i, report := range reports {
		if i > 0 {
			fmt.Fprintln(r.writer)
		}
		if err := r.renderReport(report); err != nil {
			return err
		}
	}
	return nil
}

// renderReport 渲染单个工作区的诊断报告
func (r *DoctorTextRenderer) renderReport(report workspace.DiagnosticReport) error {
	// 计算统计
	errorCount := countLevel(report.Diagnostics, workspace.DiagnosticLevelError)
	warningCount := countLevel(report.Diagnostics, workspace.DiagnosticLevelWarning)

	// 过滤诊断
	filteredDiagnostics := r.filterDiagnostics(report.Diagnostics)

	// 渲染标题
	if errorCount > 0 || warningCount > 0 {
		fmt.Fprintf(r.writer, "✗ 工作区 '%s'", report.Workspace)
		parts := []string{}
		if errorCount > 0 {
			parts = append(parts, fmt.Sprintf("%d 个错误", errorCount))
		}
		if warningCount > 0 {
			parts = append(parts, fmt.Sprintf("%d 个警告", warningCount))
		}
		if len(parts) > 0 {
			fmt.Fprintf(r.writer, " 有 %s", strings.Join(parts, "、"))
		}
		fmt.Fprintln(r.writer)
		fmt.Fprintln(r.writer)

		// 渲染诊断
		for _, diag := range filteredDiagnostics {
			if err := r.renderDiagnostic(diag, report); err != nil {
				return err
			}
		}
	} else if len(filteredDiagnostics) > 0 {
		// 只有 info 级别诊断（仅在 verbose 模式下）
		fmt.Fprintf(r.writer, "✓ 工作区 '%s' 健康 (ok)\n", report.Workspace)
		fmt.Fprintln(r.writer)
		for _, diag := range filteredDiagnostics {
			if err := r.renderDiagnostic(diag, report); err != nil {
				return err
			}
		}
	} else {
		fmt.Fprintf(r.writer, "✓ 工作区 '%s' 健康 (ok) - no issues\n", report.Workspace)
	}

	return nil
}

// renderDiagnostic 渲染单个诊断项
func (r *DoctorTextRenderer) renderDiagnostic(diag workspace.Diagnostic, report workspace.DiagnosticReport) error {
	// 标题行
	levelLabel := r.getLevelLabel(diag.Level)
	title := r.formatTitle(diag.Message)

	fmt.Fprintf(r.writer, "  %s %s\n", levelLabel, title)

	// 路径信息
	if diag.Path != "" {
		fmt.Fprintf(r.writer, "    路径: %s\n", diag.Path)
	}

	// 诊断代码（始终显示以保持向后兼容）
	fmt.Fprintf(r.writer, "    代码: %s\n", diag.Code)

	// 建议信息
	if diag.Suggestion != nil {
		fmt.Fprintln(r.writer)
		r.renderSuggestion(diag.Suggestion)
	}

	fmt.Fprintln(r.writer)
	return nil
}

// renderSuggestion 渲染建议信息
func (r *DoctorTextRenderer) renderSuggestion(suggestion *workspace.DiagnosticSuggestion) {
	// 原因
	if suggestion.Reason != "" {
		fmt.Fprintf(r.writer, "    原因: %s\n", suggestion.Reason)
		fmt.Fprintln(r.writer)
	}

	// 下一步命令
	if len(suggestion.Commands) > 0 {
		fmt.Fprintln(r.writer, "    下一步:")
		for i, cmd := range suggestion.Commands {
			if len(suggestion.Commands) == 1 {
				fmt.Fprintf(r.writer, "      %s\n", output.Command(cmd))
			} else {
				fmt.Fprintf(r.writer, "      %d. %s\n", i+1, output.Command(cmd))
			}
		}
	}

	// 额外说明
	for _, note := range suggestion.Notes {
		fmt.Fprintf(r.writer, "      提示: %s\n", note)
	}

	// 文档链接
	if suggestion.DocLink != "" {
		fmt.Fprintf(r.writer, "      文档: docs/%s\n", suggestion.DocLink)
	}
}

// filterDiagnostics 根据 verbose 模式过滤诊断
func (r *DoctorTextRenderer) filterDiagnostics(diagnostics []workspace.Diagnostic) []workspace.Diagnostic {
	if r.verbose {
		return diagnostics
	}
	filtered := make([]workspace.Diagnostic, 0, len(diagnostics))
	for _, diag := range diagnostics {
		if diag.Level != workspace.DiagnosticLevelInfo {
			filtered = append(filtered, diag)
		}
	}
	return filtered
}

// getLevelLabel 获取级别的标签
func (r *DoctorTextRenderer) getLevelLabel(level workspace.DiagnosticLevel) string {
	switch level {
	case workspace.DiagnosticLevelError:
		return output.Error("[错误 error]")
	case workspace.DiagnosticLevelWarning:
		return output.Warning("[警告 warning]")
	case workspace.DiagnosticLevelInfo:
		return output.Colorize(output.ColorBlue, "[信息 info]")
	default:
		return "[未知]"
	}
}

// formatTitle 格式化标题（首字母大写）
func (r *DoctorTextRenderer) formatTitle(message string) string {
	if len(message) == 0 {
		return message
	}
	// 简单的首字母大写处理
	return strings.ToUpper(message[:1]) + message[1:]
}

// countLevel 统计指定级别的诊断数量
func countLevel(diagnostics []workspace.Diagnostic, level workspace.DiagnosticLevel) int {
	count := 0
	for _, diag := range diagnostics {
		if diag.Level == level {
			count++
		}
	}
	return count
}
