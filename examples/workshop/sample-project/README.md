# ADP Workshop Sample Project

A simple task management CLI tool used in the ADP workshop for hands-on learning.

## What This Project Is

This is a minimal Go application that demonstrates:
- Basic command-line argument parsing
- Simple in-memory data structures
- Intentional bug for debugging practice

**Important**: This project contains an intentional bug in the `CompleteTask` function (off-by-one error in bounds checking). You'll discover and fix this during the workshop.

## Building

```bash
go build -o task-cli main.go
```

## Usage

```bash
# Add tasks
./task-cli add "Review pull request"
./task-cli add "Write documentation"
./task-cli add "Fix bug #123"

# List tasks
./task-cli list

# Complete a task (1-based index)
./task-cli complete 1
```

## Workshop Context

This project is used in the **ADP Workshop** (`docs/workshop.md`) to:
- Practice workspace setup with a real Go project
- Create tasks for agent coordination
- Discover and fix the intentional bug
- Learn ADP's runtime overlay and inspection tools

The bug is intentional and documented in the workshop materials. Don't fix it before starting the workshop!

## Project Structure

```
.
├── main.go       # Main CLI application with intentional bug
├── go.mod        # Go module definition
└── README.md     # This file
```

## The Intentional Bug

Location: `main.go`, function `CompleteTask`, line ~36

The bounds check uses `>` instead of `>=`, allowing an invalid index equal to `len(tm.tasks)` to pass validation, which then causes a panic.

**Expected behavior**: `./task-cli complete 999` should show "invalid task index"  
**Actual behavior**: Panic when index equals array length

Fix will be discovered during Module 2 of the workshop.
