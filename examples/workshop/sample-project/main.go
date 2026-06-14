package main

import (
	"fmt"
	"os"
	"strings"
)

// TaskManager simulates a simple task tracking CLI
type TaskManager struct {
	tasks []string
}

func NewTaskManager() *TaskManager {
	return &TaskManager{
		tasks: []string{},
	}
}

func (tm *TaskManager) AddTask(task string) {
	tm.tasks = append(tm.tasks, task)
	fmt.Printf("✓ Added task: %s\n", task)
}

func (tm *TaskManager) ListTasks() {
	if len(tm.tasks) == 0 {
		fmt.Println("No tasks yet. Add one with: ./task-cli add <task>")
		return
	}

	fmt.Println("Current Tasks:")
	for i, task := range tm.tasks {
		fmt.Printf("  %d. %s\n", i+1, task)
	}
}

// BUG: This function has an intentional off-by-one error
// Workshop participants will discover this during Module 2
func (tm *TaskManager) CompleteTask(index int) error {
	// BUG: Should check (index < 0 || index >= len(tm.tasks))
	// Currently allows invalid index len(tm.tasks) to pass
	if index < 0 || index > len(tm.tasks) {
		return fmt.Errorf("invalid task index: %d", index)
	}

	completed := tm.tasks[index]
	tm.tasks = append(tm.tasks[:index], tm.tasks[index+1:]...)
	fmt.Printf("✓ Completed task: %s\n", completed)
	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Task CLI - Simple task manager for ADP workshop")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  ./task-cli add <task>       Add a new task")
		fmt.Println("  ./task-cli list             List all tasks")
		fmt.Println("  ./task-cli complete <num>   Mark task as complete")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  ./task-cli add 'Review pull request'")
		fmt.Println("  ./task-cli list")
		fmt.Println("  ./task-cli complete 1")
		os.Exit(0)
	}

	manager := NewTaskManager()
	command := os.Args[1]

	switch command {
	case "add":
		if len(os.Args) < 3 {
			fmt.Println("Error: missing task description")
			os.Exit(1)
		}
		task := strings.Join(os.Args[2:], " ")
		manager.AddTask(task)

	case "list":
		manager.ListTasks()

	case "complete":
		if len(os.Args) < 3 {
			fmt.Println("Error: missing task index")
			os.Exit(1)
		}
		var index int
		_, err := fmt.Sscanf(os.Args[2], "%d", &index)
		if err != nil {
			fmt.Printf("Error: invalid index '%s'\n", os.Args[2])
			os.Exit(1)
		}
		// Convert from 1-based to 0-based index
		if err := manager.CompleteTask(index - 1); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Run './task-cli' for usage help")
		os.Exit(1)
	}
}
