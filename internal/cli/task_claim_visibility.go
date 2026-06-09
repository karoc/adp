package cli

import (
	"strings"
	"time"

	taskstore "github.com/karoc/adp/internal/tasks"
)

const (
	taskClaimStateUnclaimed = "unclaimed"
	taskClaimStateClaimed   = "claimed"
	taskClaimStateLeased    = "leased"
	taskClaimStateStale     = "stale"
)

func taskClaimState(task taskstore.Task, now time.Time) string {
	if strings.TrimSpace(task.Owner) == "" {
		return taskClaimStateUnclaimed
	}
	if task.LeaseExpiresAt.IsZero() {
		return taskClaimStateClaimed
	}
	if task.LeaseExpiresAt.After(now.UTC()) {
		return taskClaimStateLeased
	}
	return taskClaimStateStale
}

func taskClaimDetail(task taskstore.Task, now time.Time) string {
	switch taskClaimState(task, now) {
	case taskClaimStateUnclaimed:
		return taskClaimStateUnclaimed
	case taskClaimStateClaimed:
		return taskClaimStateClaimed
	case taskClaimStateLeased:
		return "leased until " + formatEventTime(task.LeaseExpiresAt)
	default:
		return "stale since " + formatEventTime(task.LeaseExpiresAt)
	}
}

func taskClaimDetailChinese(task taskstore.Task, now time.Time) string {
	switch taskClaimState(task, now) {
	case taskClaimStateUnclaimed:
		return "未领取"
	case taskClaimStateClaimed:
		return "已领取"
	case taskClaimStateLeased:
		return "租约至 " + formatEventTime(task.LeaseExpiresAt)
	default:
		return "已过期 " + formatEventTime(task.LeaseExpiresAt)
	}
}

func taskClaimHandoff(task taskstore.Task, now time.Time) string {
	owner := strings.TrimSpace(task.Owner)
	switch taskClaimState(task, now) {
	case taskClaimStateUnclaimed:
		return taskClaimStateUnclaimed
	case taskClaimStateClaimed:
		return "claimed by " + owner
	case taskClaimStateLeased:
		return "leased to " + owner + " until " + formatEventTime(task.LeaseExpiresAt)
	default:
		return "stale claim by " + owner + " since " + formatEventTime(task.LeaseExpiresAt)
	}
}

func taskClaimHandoffChinese(task taskstore.Task, now time.Time) string {
	owner := strings.TrimSpace(task.Owner)
	switch taskClaimState(task, now) {
	case taskClaimStateUnclaimed:
		return "未领取"
	case taskClaimStateClaimed:
		return owner + " 已领取"
	case taskClaimStateLeased:
		return owner + " 租约至 " + formatEventTime(task.LeaseExpiresAt)
	default:
		return owner + " 的领取已过期：" + formatEventTime(task.LeaseExpiresAt)
	}
}
