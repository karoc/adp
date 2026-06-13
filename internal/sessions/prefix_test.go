package sessions

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/karoc/adp/internal/events"
	"github.com/karoc/adp/internal/paths"
)

func TestFindByPrefix(t *testing.T) {
	ctx := context.Background()
	layout := setupTestLayout(t)
	logger := events.NewLogger(layout)

	// Create test sessions with predictable IDs
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	sessions := []struct {
		id        string
		workspace string
		agent     string
	}{
		{"abc123def456", "workspace1", "agent1"},
		{"abc456ghi789", "workspace1", "agent2"},
		{"xyz789jkl012", "workspace2", "agent1"},
		{"def123mno345", "workspace2", "agent2"},
	}

	// Write events for each session
	var eventsToLog []events.Event
	for i, session := range sessions {
		eventsToLog = append(eventsToLog, events.Event{
			SessionID:   session.id,
			Workspace:   session.workspace,
			Agent:       session.agent,
			Type:        eventTypeRunStarted,
			Timestamp:   now.Add(time.Duration(i) * time.Minute),
			ProjectRoot: "/test/root",
		})
	}
	logEvents(t, logger, eventsToLog)

	tests := []struct {
		name        string
		prefix      string
		wantCount   int
		wantErr     error
		wantIDs     []string
		description string
	}{
		{
			name:        "exact match",
			prefix:      "abc123def456",
			wantCount:   1,
			wantIDs:     []string{"abc123def456"},
			description: "Exact match should return single session",
		},
		{
			name:        "unique prefix match",
			prefix:      "xyz",
			wantCount:   1,
			wantIDs:     []string{"xyz789jkl012"},
			description: "Unique prefix should return single session",
		},
		{
			name:        "ambiguous prefix",
			prefix:      "abc",
			wantCount:   2,
			wantErr:     ErrAmbiguousSessionID,
			wantIDs:     []string{"abc123def456", "abc456ghi789"},
			description: "Ambiguous prefix should return error with all matches",
		},
		{
			name:        "no match",
			prefix:      "zzz",
			wantCount:   0,
			wantErr:     ErrSessionNotFound,
			description: "Non-matching prefix should return not found error",
		},
		{
			name:        "empty prefix",
			prefix:      "",
			wantCount:   0,
			wantErr:     ErrSessionNotFound,
			description: "Empty prefix should return not found error",
		},
		{
			name:        "longer unique prefix",
			prefix:      "abc123",
			wantCount:   1,
			wantIDs:     []string{"abc123def456"},
			description: "Longer unique prefix should match",
		},
		{
			name:        "partial match with multiple candidates",
			prefix:      "def",
			wantCount:   1,
			wantIDs:     []string{"def123mno345"},
			description: "Prefix matching one session should succeed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FindByPrefix(ctx, layout, tt.prefix)

			// Check error
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("FindByPrefix() error = nil, wantErr %v", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("FindByPrefix() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("FindByPrefix() unexpected error = %v", err)
			}

			// Check count
			if len(got) != tt.wantCount {
				t.Errorf("FindByPrefix() returned %d sessions, want %d", len(got), tt.wantCount)
			}

			// Check IDs if specified
			if len(tt.wantIDs) > 0 {
				gotIDs := make([]string, len(got))
				for i, s := range got {
					gotIDs[i] = s.SessionID
				}
				if !equalStringSlices(gotIDs, tt.wantIDs) {
					t.Errorf("FindByPrefix() IDs = %v, want %v", gotIDs, tt.wantIDs)
				}
			}
		})
	}
}

func TestFindByPrefix_ContextCancellation(t *testing.T) {
	layout := setupTestLayout(t)

	// Create a canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := FindByPrefix(ctx, layout, "abc")
	if err == nil {
		t.Error("FindByPrefix() with canceled context should return error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("FindByPrefix() error = %v, want context.Canceled", err)
	}
}

func TestFindByPrefix_ExactMatchPriority(t *testing.T) {
	ctx := context.Background()
	layout := setupTestLayout(t)
	logger := events.NewLogger(layout)

	now := time.Now()

	// Create sessions where one ID is a prefix of another
	sessions := []string{
		"abc",
		"abc123",
		"abc456",
	}

	var eventsToLog []events.Event
	for i, id := range sessions {
		eventsToLog = append(eventsToLog, events.Event{
			SessionID: id,
			Workspace: "test",
			Type:      eventTypeRunStarted,
			Timestamp: now.Add(time.Duration(i) * time.Minute),
		})
	}
	logEvents(t, logger, eventsToLog)

	// Search for "abc" - should match exactly, not as prefix
	got, err := FindByPrefix(ctx, layout, "abc")
	if err != nil {
		t.Fatalf("FindByPrefix() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("FindByPrefix() returned %d sessions, want 1", len(got))
	}
	if got[0].SessionID != "abc" {
		t.Errorf("FindByPrefix() SessionID = %q, want %q", got[0].SessionID, "abc")
	}
}

// Helper functions

func setupTestLayout(t *testing.T) paths.Layout {
	t.Helper()
	tmpDir := t.TempDir()
	return paths.New(tmpDir, tmpDir)
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	counts := make(map[string]int)
	for _, s := range a {
		counts[s]++
	}
	for _, s := range b {
		counts[s]--
		if counts[s] < 0 {
			return false
		}
	}
	for _, count := range counts {
		if count != 0 {
			return false
		}
	}
	return true
}
