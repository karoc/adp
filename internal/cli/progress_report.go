package cli

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/karoc/adp/internal/sessions"
	taskstore "github.com/karoc/adp/internal/tasks"
)

const (
	reportLanguageEnglish = "en"
	reportLanguageChinese = "zh-CN"
	reportFormatMarkdown  = "markdown"
	reportFormatJSON      = "json"
)

type progressReportOptions struct {
	workspace string
	language  string
	format    string
}

func (a *App) progressReport(ctx context.Context, args []string) error {
	opts, err := parseProgressReportArgs(args)
	if err != nil {
		return err
	}
	store, workspaceName, err := a.loadTaskStore(ctx, opts.workspace)
	if err != nil {
		return err
	}
	tasks, err := store.List(ctx)
	if err != nil {
		return err
	}
	progress, err := store.Progress(ctx)
	if err != nil {
		return err
	}
	phases, err := store.ListPhases(ctx)
	if err != nil {
		return err
	}
	runtimeSessions, err := a.progressReportSessions(ctx, workspaceName)
	if err != nil {
		return err
	}
	data := progressReportData{
		Workspace:       workspaceName,
		Tasks:           tasks,
		Progress:        progress,
		Phases:          phases,
		RuntimeSessions: runtimeSessions,
		Language:        opts.language,
	}
	if opts.format == reportFormatJSON {
		return writePlanningJSON(a.stdout, progressReportOutput(data))
	}
	writeProgressReport(a.stdout, data)
	return nil
}

func (a *App) progressReportSessions(ctx context.Context, workspaceName string) ([]sessions.Summary, error) {
	if a.deps.ListSessions == nil {
		return nil, nil
	}
	return a.deps.ListSessions(ctx, a.deps.Layout, sessions.Query{
		Workspace: workspaceName,
		Limit:     5,
	})
}

func parseProgressReportArgs(args []string) (progressReportOptions, error) {
	opts := progressReportOptions{language: reportLanguageEnglish, format: reportFormatMarkdown}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--workspace", "-w":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return progressReportOptions{}, err
			}
			opts.workspace, i = value, next
		case "--language":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return progressReportOptions{}, err
			}
			language, err := parseProgressReportLanguage(value)
			if err != nil {
				return progressReportOptions{}, err
			}
			opts.language, i = language, next
		case "--format":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return progressReportOptions{}, err
			}
			format, err := parseProgressReportFormat(value)
			if err != nil {
				return progressReportOptions{}, err
			}
			opts.format, i = format, next
		default:
			return progressReportOptions{}, fmt.Errorf("unknown progress report option %q", arg)
		}
	}
	return opts, nil
}

func parseProgressReportLanguage(value string) (string, error) {
	switch strings.TrimSpace(value) {
	case "", reportLanguageEnglish:
		return reportLanguageEnglish, nil
	case reportLanguageChinese:
		return reportLanguageChinese, nil
	default:
		return "", fmt.Errorf("unknown progress report language %q", value)
	}
}

func parseProgressReportFormat(value string) (string, error) {
	switch strings.TrimSpace(value) {
	case "", reportFormatMarkdown, string(outputFormatText):
		return reportFormatMarkdown, nil
	case reportFormatJSON:
		return reportFormatJSON, nil
	default:
		return "", fmt.Errorf("unknown progress report format %q", value)
	}
}

type progressReportData struct {
	Workspace       string
	Tasks           []taskstore.Task
	Progress        taskstore.Progress
	Phases          []taskstore.Phase
	RuntimeSessions []sessions.Summary
	Language        string
}

func writeProgressReport(w io.Writer, data progressReportData) {
	if data.Language == reportLanguageChinese {
		writeProgressReportChinese(w, data)
		return
	}
	writeProgressReportEnglish(w, data)
}

func writeProgressReportEnglish(w io.Writer, data progressReportData) {
	fmt.Fprintln(w, "# ADP Progress Report")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Workspace: %s\n", data.Workspace)
	fmt.Fprintf(w, "Total Tasks: %d\n", data.Progress.Total)
	fmt.Fprintln(w)
	writePhaseReportEnglish(w, data.Phases)
	writeTaskCountReportEnglish(w, data.Progress)
	writeTaskReportEnglish(w, data.Tasks)
	writeNextWorkReportEnglish(w, data.Tasks)
	writeEvidenceReportEnglish(w, data.Phases)
	writeRuntimeSessionReportEnglish(w, data.RuntimeSessions)
}

func writeProgressReportChinese(w io.Writer, data progressReportData) {
	fmt.Fprintln(w, "# ADP 执行进度报告")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "工作区：%s\n", data.Workspace)
	fmt.Fprintf(w, "任务总数：%d\n", data.Progress.Total)
	fmt.Fprintln(w)
	writePhaseReportChinese(w, data.Phases)
	writeTaskCountReportChinese(w, data.Progress)
	writeTaskReportChinese(w, data.Tasks)
	writeNextWorkReportChinese(w, data.Tasks)
	writeEvidenceReportChinese(w, data.Phases)
	writeRuntimeSessionReportChinese(w, data.RuntimeSessions)
}

func writePhaseReportEnglish(w io.Writer, phases []taskstore.Phase) {
	fmt.Fprintln(w, "## Phases")
	if len(phases) == 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "No phases recorded.")
		fmt.Fprintln(w)
		return
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| ID | Status | Title | Goal |")
	fmt.Fprintln(w, "| --- | --- | --- | --- |")
	for _, phase := range phases {
		fmt.Fprintf(w, "| %s | %s | %s | %s |\n", markdownCell(phase.ID), phase.Status, markdownCell(phase.Title), markdownCell(valueOrDash(phase.Goal)))
	}
	fmt.Fprintln(w)
}

func writePhaseReportChinese(w io.Writer, phases []taskstore.Phase) {
	fmt.Fprintln(w, "## 阶段")
	if len(phases) == 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "暂无阶段记录。")
		fmt.Fprintln(w)
		return
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| ID | 状态 | 标题 | 目标 |")
	fmt.Fprintln(w, "| --- | --- | --- | --- |")
	for _, phase := range phases {
		fmt.Fprintf(w, "| %s | %s | %s | %s |\n", markdownCell(phase.ID), phase.Status, markdownCell(phase.Title), markdownCell(valueOrDash(phase.Goal)))
	}
	fmt.Fprintln(w)
}

func writeTaskCountReportEnglish(w io.Writer, progress taskstore.Progress) {
	fmt.Fprintln(w, "## Task Counts")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| Status | Count |")
	fmt.Fprintln(w, "| --- | ---: |")
	for _, status := range taskstore.Statuses() {
		fmt.Fprintf(w, "| %s | %d |\n", status, progress.Counts[status])
	}
	fmt.Fprintln(w)
}

func writeTaskCountReportChinese(w io.Writer, progress taskstore.Progress) {
	fmt.Fprintln(w, "## 任务统计")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| 状态 | 数量 |")
	fmt.Fprintln(w, "| --- | ---: |")
	for _, status := range taskstore.Statuses() {
		fmt.Fprintf(w, "| %s | %d |\n", status, progress.Counts[status])
	}
	fmt.Fprintln(w)
}

func writeTaskReportEnglish(w io.Writer, tasks []taskstore.Task) {
	fmt.Fprintln(w, "## Tasks")
	if len(tasks) == 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "No tasks recorded.")
		fmt.Fprintln(w)
		return
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| ID | Status | Priority | Phase | Owner | Title |")
	fmt.Fprintln(w, "| --- | --- | --- | --- | --- | --- |")
	for _, task := range tasks {
		fmt.Fprintf(w, "| %s | %s | %s | %s | %s | %s |\n",
			markdownCell(task.ID),
			task.Status,
			markdownCell(valueOrDash(task.Priority)),
			markdownCell(valueOrDash(task.Phase)),
			markdownCell(valueOrDash(task.Owner)),
			markdownCell(task.Title),
		)
	}
	fmt.Fprintln(w)
}

func writeTaskReportChinese(w io.Writer, tasks []taskstore.Task) {
	fmt.Fprintln(w, "## 任务")
	if len(tasks) == 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "暂无任务记录。")
		fmt.Fprintln(w)
		return
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| ID | 状态 | 优先级 | 阶段 | 负责人 | 标题 |")
	fmt.Fprintln(w, "| --- | --- | --- | --- | --- | --- |")
	for _, task := range tasks {
		fmt.Fprintf(w, "| %s | %s | %s | %s | %s | %s |\n",
			markdownCell(task.ID),
			task.Status,
			markdownCell(valueOrDash(task.Priority)),
			markdownCell(valueOrDash(task.Phase)),
			markdownCell(valueOrDash(task.Owner)),
			markdownCell(task.Title),
		)
	}
	fmt.Fprintln(w)
}

func writeNextWorkReportEnglish(w io.Writer, tasks []taskstore.Task) {
	fmt.Fprintln(w, "## Next Work")
	open := reportableTasks(tasks)
	if len(open) == 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "No ready, in-progress, or review tasks.")
		fmt.Fprintln(w)
		return
	}
	fmt.Fprintln(w)
	for _, task := range open {
		fmt.Fprintf(w, "- `%s` [%s] %s (priority: %s, phase: %s, owner: %s)\n",
			task.ID,
			task.Status,
			task.Title,
			valueOrDash(task.Priority),
			valueOrDash(task.Phase),
			valueOrDash(task.Owner),
		)
	}
	fmt.Fprintln(w)
}

func writeNextWorkReportChinese(w io.Writer, tasks []taskstore.Task) {
	fmt.Fprintln(w, "## 下一步工作")
	open := reportableTasks(tasks)
	if len(open) == 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "暂无 ready、in_progress 或 review 任务。")
		fmt.Fprintln(w)
		return
	}
	fmt.Fprintln(w)
	for _, task := range open {
		fmt.Fprintf(w, "- `%s` [%s] %s（优先级：%s，阶段：%s，负责人：%s）\n",
			task.ID,
			task.Status,
			task.Title,
			valueOrDash(task.Priority),
			valueOrDash(task.Phase),
			valueOrDash(task.Owner),
		)
	}
	fmt.Fprintln(w)
}

func writeEvidenceReportEnglish(w io.Writer, phases []taskstore.Phase) {
	fmt.Fprintln(w, "## Phase Evidence")
	if len(phases) == 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "No phase evidence recorded.")
		return
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| Phase | Acceptance | Commit | Push |")
	fmt.Fprintln(w, "| --- | --- | --- | --- |")
	for _, phase := range phases {
		fmt.Fprintf(w, "| %s | %s | %s | %s |\n",
			markdownCell(phase.ID),
			markdownCell(acceptanceSummary(phase.Acceptance)),
			markdownCell(commitSummary(phase.Commit)),
			markdownCell(pushSummary(phase.Push)),
		)
	}
}

func writeEvidenceReportChinese(w io.Writer, phases []taskstore.Phase) {
	fmt.Fprintln(w, "## 阶段证据")
	if len(phases) == 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "暂无阶段证据记录。")
		return
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| 阶段 | 验收 | 提交 | 推送 |")
	fmt.Fprintln(w, "| --- | --- | --- | --- |")
	for _, phase := range phases {
		fmt.Fprintf(w, "| %s | %s | %s | %s |\n",
			markdownCell(phase.ID),
			markdownCell(acceptanceSummary(phase.Acceptance)),
			markdownCell(commitSummary(phase.Commit)),
			markdownCell(pushSummary(phase.Push)),
		)
	}
}

func writeRuntimeSessionReportEnglish(w io.Writer, summaries []sessions.Summary) {
	fmt.Fprintln(w, "## Runtime Sessions")
	if len(summaries) == 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "No runtime sessions recorded.")
		return
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| Session | Agent | Task | Started | Finished | Exit | Duration | Events | Runtime |")
	fmt.Fprintln(w, "| --- | --- | --- | --- | --- | --- | --- | ---: | --- |")
	for _, summary := range summaries {
		fmt.Fprintf(w, "| %s | %s | %s | %s | %s | %s | %s | %d | %s |\n",
			markdownCell(summary.SessionID),
			markdownCell(valueOrDash(summary.Agent)),
			markdownCell(valueOrDash(summary.TaskID)),
			formatEventTime(summary.StartedAt),
			formatEventTime(summary.FinishedAt),
			formatExitCode(summary.ExitCode),
			formatDurationMillis(summary.DurationMillis),
			summary.EventCount,
			markdownCell(valueOrDash(summary.RuntimePath)),
		)
	}
}

func writeRuntimeSessionReportChinese(w io.Writer, summaries []sessions.Summary) {
	fmt.Fprintln(w, "## Runtime 会话")
	if len(summaries) == 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "暂无 runtime 会话记录。")
		return
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| Session | Agent | Task | Started | Finished | Exit | Duration | Events | Runtime |")
	fmt.Fprintln(w, "| --- | --- | --- | --- | --- | --- | --- | ---: | --- |")
	for _, summary := range summaries {
		fmt.Fprintf(w, "| %s | %s | %s | %s | %s | %s | %s | %d | %s |\n",
			markdownCell(summary.SessionID),
			markdownCell(valueOrDash(summary.Agent)),
			markdownCell(valueOrDash(summary.TaskID)),
			formatEventTime(summary.StartedAt),
			formatEventTime(summary.FinishedAt),
			formatExitCode(summary.ExitCode),
			formatDurationMillis(summary.DurationMillis),
			summary.EventCount,
			markdownCell(valueOrDash(summary.RuntimePath)),
		)
	}
}

func reportableTasks(tasks []taskstore.Task) []taskstore.Task {
	open := make([]taskstore.Task, 0, len(tasks))
	for _, task := range tasks {
		if task.Status == taskstore.StatusReady || task.Status == taskstore.StatusInProgress || task.Status == taskstore.StatusReview {
			open = append(open, task)
		}
	}
	sort.SliceStable(open, func(i, j int) bool {
		left := priorityRank(open[i].Priority)
		right := priorityRank(open[j].Priority)
		if left != right {
			return left < right
		}
		if open[i].CreatedAt.Equal(open[j].CreatedAt) {
			return open[i].ID < open[j].ID
		}
		return open[i].CreatedAt.Before(open[j].CreatedAt)
	})
	return open
}

func priorityRank(priority string) int {
	switch strings.ToLower(strings.TrimSpace(priority)) {
	case "critical", "urgent", "p0":
		return 0
	case "high", "p1":
		return 1
	case "normal", "medium", "p2", "":
		return 2
	case "low", "p3":
		return 3
	default:
		return 2
	}
}

func acceptanceSummary(record taskstore.AcceptanceRecord) string {
	if record.Result == "" {
		return "-"
	}
	if len(record.Commands) == 0 {
		return record.Result
	}
	return record.Result + ": " + strings.Join(record.Commands, "; ")
}

func commitSummary(record taskstore.CommitRecord) string {
	if record.Hash == "" {
		return "-"
	}
	if record.Message == "" {
		return record.Hash
	}
	return record.Hash + ": " + record.Message
}

func pushSummary(record taskstore.PushRecord) string {
	if record.Result == "" && record.Remote == "" && record.Branch == "" {
		return "-"
	}
	target := strings.Trim(record.Remote+"/"+record.Branch, "/")
	if record.Result == "" {
		return target
	}
	if target == "" {
		return record.Result
	}
	return record.Result + ": " + target
}

func markdownCell(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	return strings.ReplaceAll(value, "|", "\\|")
}
