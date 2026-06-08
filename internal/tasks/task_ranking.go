package tasks

import (
	"sort"
	"strings"
)

func sortTasks(tasks []Task) {
	sort.SliceStable(tasks, func(i, j int) bool {
		if tasks[i].CreatedAt.Equal(tasks[j].CreatedAt) {
			return tasks[i].ID < tasks[j].ID
		}
		return tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
	})
}

func NextTasks(tasks []Task, limit int) []Task {
	open := make([]Task, 0, len(tasks))
	for _, task := range tasks {
		if isNextStatus(task.Status) {
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
	if limit > 0 && len(open) > limit {
		open = open[:limit]
	}
	return open
}

func isNextStatus(status Status) bool {
	return status == StatusReady || status == StatusInProgress || status == StatusReview
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
