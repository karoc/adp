package tasks

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestFindByPrefix_ExactMatch(t *testing.T) {
	store := testStore(t)

	// Add multiple tasks
	task1, err := store.Add(context.Background(), AddRequest{
		Title:    "First task",
		Priority: "high",
	})
	if err != nil {
		t.Fatalf("Add task1 failed: %v", err)
	}

	task2, err := store.Add(context.Background(), AddRequest{
		Title:    "Second task",
		Priority: "normal",
	})
	if err != nil {
		t.Fatalf("Add task2 failed: %v", err)
	}

	// Test exact match with full ID
	matches, err := store.FindByPrefix(context.Background(), task1.ID)
	if err != nil {
		t.Fatalf("FindByPrefix with exact ID failed: %v", err)
	}

	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}

	if matches[0].ID != task1.ID {
		t.Fatalf("expected task ID %q, got %q", task1.ID, matches[0].ID)
	}

	// Test exact match for second task
	matches, err = store.FindByPrefix(context.Background(), task2.ID)
	if err != nil {
		t.Fatalf("FindByPrefix with exact ID for task2 failed: %v", err)
	}

	if len(matches) != 1 || matches[0].ID != task2.ID {
		t.Fatalf("expected exact match for task2, got %+v", matches)
	}
}

func TestFindByPrefix_SinglePrefixMatch(t *testing.T) {
	store := testStore(t)

	task, err := store.Add(context.Background(), AddRequest{
		Title: "Unique task",
	})
	if err != nil {
		t.Fatalf("Add task failed: %v", err)
	}

	// Test with shorter prefix that uniquely identifies the task
	// Assuming task ID is like "task-20260608-0001"
	prefix := task.ID[:10] // e.g., "task-20260"

	matches, err := store.FindByPrefix(context.Background(), prefix)
	if err != nil {
		t.Fatalf("FindByPrefix with unique prefix failed: %v", err)
	}

	if len(matches) != 1 {
		t.Fatalf("expected 1 match for prefix %q, got %d", prefix, len(matches))
	}

	if matches[0].ID != task.ID {
		t.Fatalf("expected task ID %q, got %q", task.ID, matches[0].ID)
	}
}

func TestFindByPrefix_AmbiguousMatch(t *testing.T) {
	store := testStore(t)

	// Add multiple tasks (they will share the same date prefix)
	task1, err := store.Add(context.Background(), AddRequest{
		Title: "First task",
	})
	if err != nil {
		t.Fatalf("Add task1 failed: %v", err)
	}

	task2, err := store.Add(context.Background(), AddRequest{
		Title: "Second task",
	})
	if err != nil {
		t.Fatalf("Add task2 failed: %v", err)
	}

	// Use a prefix that matches both tasks
	// e.g., "task-" or "task-20260608" should match both
	prefix := "task-"

	_, err = store.FindByPrefix(context.Background(), prefix)

	// Should return an ambiguous error
	if err == nil {
		t.Fatalf("expected ambiguous error, got nil")
	}

	if !errors.Is(err, ErrAmbiguousTaskID) {
		t.Fatalf("expected ErrAmbiguousTaskID, got: %v", err)
	}

	// Error message should contain both task IDs
	errMsg := err.Error()
	if !strings.Contains(errMsg, task1.ID) || !strings.Contains(errMsg, task2.ID) {
		t.Fatalf("error message should list both task IDs, got: %s", errMsg)
	}
}

func TestFindByPrefix_NotFound(t *testing.T) {
	store := testStore(t)

	// Add a task
	_, err := store.Add(context.Background(), AddRequest{
		Title: "Existing task",
	})
	if err != nil {
		t.Fatalf("Add task failed: %v", err)
	}

	// Try to find a non-existent prefix
	matches, err := store.FindByPrefix(context.Background(), "nonexistent-")

	if err == nil {
		t.Fatalf("expected not found error, got nil with matches: %+v", matches)
	}

	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("expected ErrTaskNotFound, got: %v", err)
	}

	// Error message should mention the prefix
	if !strings.Contains(err.Error(), "nonexistent-") {
		t.Fatalf("error message should mention the prefix, got: %s", err.Error())
	}
}

func TestFindByPrefix_EmptyPrefix(t *testing.T) {
	store := testStore(t)

	// Add a task
	_, err := store.Add(context.Background(), AddRequest{
		Title: "Test task",
	})
	if err != nil {
		t.Fatalf("Add task failed: %v", err)
	}

	// Try with empty prefix
	_, err = store.FindByPrefix(context.Background(), "")
	if err == nil {
		t.Fatalf("expected error for empty prefix, got nil")
	}

	if !strings.Contains(err.Error(), "empty") {
		t.Fatalf("expected 'empty' in error message, got: %s", err.Error())
	}

	// Try with whitespace-only prefix
	_, err = store.FindByPrefix(context.Background(), "   ")
	if err == nil {
		t.Fatalf("expected error for whitespace prefix, got nil")
	}
}

func TestFindByPrefix_ContextCanceled(t *testing.T) {
	store := testStore(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := store.FindByPrefix(ctx, "task-")
	if err == nil {
		t.Fatalf("expected context canceled error, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
}

func TestFindByPrefix_PrefersExactMatch(t *testing.T) {
	store := testStore(t)

	// Add first task
	task1, err := store.Add(context.Background(), AddRequest{
		Title: "First task",
	})
	if err != nil {
		t.Fatalf("Add task1 failed: %v", err)
	}

	// Add second task
	_, err = store.Add(context.Background(), AddRequest{
		Title: "Second task",
	})
	if err != nil {
		t.Fatalf("Add task2 failed: %v", err)
	}

	// Use task1's full ID, which is also a prefix for searching
	// Even though task2 might share some prefix, exact match should win
	matches, err := store.FindByPrefix(context.Background(), task1.ID)
	if err != nil {
		t.Fatalf("FindByPrefix failed: %v", err)
	}

	if len(matches) != 1 {
		t.Fatalf("expected exactly 1 match for exact ID, got %d", len(matches))
	}

	if matches[0].ID != task1.ID {
		t.Fatalf("expected exact match to return task1, got task ID %q", matches[0].ID)
	}
}
