package tasks

import (
	"sort"
	"strings"
	"time"
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
	sortByTaskPriority(open)
	if limit > 0 && len(open) > limit {
		open = open[:limit]
	}
	return open
}

func claimableTasks(tasks []Task, now time.Time, limit int) []Task {
	claimable := make([]Task, 0, len(tasks))
	for _, task := range tasks {
		if isTakeStatus(task, now) {
			claimable = append(claimable, task)
		}
	}
	sortByTaskPriority(claimable)
	if limit > 0 && len(claimable) > limit {
		claimable = claimable[:limit]
	}
	return claimable
}

func StaleTasks(tasks []Task, now time.Time) []Task {
	stale := make([]Task, 0, len(tasks))
	for _, task := range tasks {
		if isStaleTask(task, now) {
			stale = append(stale, task)
		}
	}
	sortByTaskPriority(stale)
	return stale
}

func isNextStatus(status Status) bool {
	return status == StatusReady || status == StatusInProgress || status == StatusReview
}

func isTakeStatus(task Task, now time.Time) bool {
	switch task.Status {
	case StatusReady:
		return task.Owner == "" || claimLeaseExpired(task, now)
	case StatusInProgress:
		return task.Owner != "" && claimLeaseExpired(task, now)
	default:
		return false
	}
}

func isStaleTask(task Task, now time.Time) bool {
	return task.Status == StatusInProgress && task.Owner != "" && claimLeaseExpired(task, now)
}

func sortByTaskPriority(tasks []Task) {
	sort.SliceStable(tasks, func(i, j int) bool {
		left := priorityRank(tasks[i].Priority)
		right := priorityRank(tasks[j].Priority)
		if left != right {
			return left < right
		}
		if tasks[i].CreatedAt.Equal(tasks[j].CreatedAt) {
			return tasks[i].ID < tasks[j].ID
		}
		return tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
	})
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
