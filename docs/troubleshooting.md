# Troubleshooting Guide

简体中文：[troubleshooting.zh-CN.md](troubleshooting.zh-CN.md)

This guide helps you diagnose and resolve common ADP issues. Issues are organized by error message or symptom for easy search.

---

## Table of Contents

- [Installation & Setup](#installation--setup)
- [Workspace Issues](#workspace-issues)
- [Agent Execution Issues](#agent-execution-issues)
- [Runtime Issues](#runtime-issues)
- [Task Management Issues](#task-management-issues)
- [Phase Management Issues](#phase-management-issues)
- [Session Issues](#session-issues)
- [Interactive & Confirmation Issues](#interactive--confirmation-issues)
- [Parameter Validation Issues](#parameter-validation-issues)
- [Environment Variables](#environment-variables)
- [Permission Issues](#permission-issues)
- [Diagnostic Commands](#diagnostic-commands)

---

## Installation & Setup

### "command not found: adp"

**Cause:**
- ADP binary is not in your `$PATH`
- Binary was not installed correctly
- Shell hasn't reloaded `$PATH`

**Diagnosis:**
```bash
# Check if binary exists
ls -la ./bin/adp
which adp

# Check PATH
echo $PATH
```

**Solution:**
1. Add ADP binary directory to `$PATH`:
   ```bash
   export PATH="$HOME/.local/bin:$PATH"
   ```
2. Or use absolute path:
   ```bash
   /path/to/adp --help
   ```
3. Reload shell configuration:
   ```bash
   source ~/.bashrc  # or ~/.zshrc
   ```

---

### "ADP_HOME not set or invalid"

**Cause:**
- `$ADP_HOME` environment variable is not set
- Directory does not exist or is not writable

**Diagnosis:**
```bash
# Check environment variable
echo $ADP_HOME

# Check directory exists
ls -ld $ADP_HOME
```

**Solution:**
1. Set `ADP_HOME` (defaults to `~/.adp`):
   ```bash
   export ADP_HOME="$HOME/.adp"
   ```
2. Initialize ADP:
   ```bash
   adp init
   ```

---

## Workspace Issues

### "workspace not found"

**Cause:**
- Workspace name is misspelled
- Workspace has not been created
- `$ADP_HOME` points to wrong directory

**Diagnosis:**
```bash
# List all workspaces
adp workspace list

# Check ADP_HOME
echo $ADP_HOME
ls -la $ADP_HOME/workspaces/
```

**Solution:**
1. Verify workspace name spelling
2. Create workspace if needed:
   ```bash
   adp workspace add my-project /path/to/project
   ```
3. Check `$ADP_HOME` is correct

---

### "project root does not exist"

**Cause:**
- Project path is incorrect
- Project directory was moved or deleted
- Symlink is broken

**Diagnosis:**
```bash
# Check project path from workspace config
adp workspace show my-workspace

# Verify directory exists
ls -ld /path/to/project
```

**Solution:**
1. If project moved, update workspace:
   ```bash
   adp workspace remove old-name
   adp workspace add new-name /new/path/to/project
   ```
2. Or recreate workspace with correct path

---

### "workspace doctor reports errors"

**Cause:**
- Configuration files are missing or invalid
- Referenced files (prompts, memory, MCP) don't exist
- Runtime parent directory is unsafe

**Diagnosis:**
```bash
# Run detailed diagnostics
adp workspace doctor my-workspace --verbose

# JSON output for machine parsing
adp workspace doctor my-workspace --format json
```

**Solution:**
- Follow specific recommendations in doctor output
- Check all file paths referenced in workspace config
- Verify `$ADP_RUNTIME_DIR` is not inside project root

---

## Agent Execution Issues

### "agent command not found: <command>"

**Cause:**
- The agent CLI (e.g. `codex`, `claude`) is not installed
- The agent binary is not in your `$PATH`
- The workspace is configured with a command path that does not exist

**Diagnosis:**
```bash
# Check if the agent command is resolvable
which codex
which claude

# Inspect the workspace command configuration
adp workspace show my-workspace

# Run workspace diagnostics
adp workspace doctor my-workspace
```

**Solution:**
1. Install the missing agent CLI, then verify:
   ```bash
   codex --version
   ```
2. If installed but not found, add its location to `$PATH`:
   ```bash
   export PATH="/path/to/agent/bin:$PATH"
   ```
3. Configure an explicit command path in the workspace so ADP does not rely on `$PATH` lookup:
   ```bash
   # Edit the workspace config and set the agent command path explicitly
   adp workspace doctor my-workspace
   ```

---

### "unknown command" with "Did you mean ...?"

**Cause:**
- A top-level command or subcommand name is misspelled
- ADP could not match the input and suggests similar commands

**Diagnosis:**
```bash
# Example: typo in a command name
adp wrkspace list
# Error: unknown command "wrkspace"
# Did you mean one of these?
#   workspace
```

**Solution:**
1. Use the suggested command from the "Did you mean" output.
2. List all top-level commands to confirm the correct spelling:
   ```bash
   adp --help
   ```
3. Command aliases are also available for common shortcuts:
   - `ws` → `workspace`
   - `t` → `tasks`
   - `s` → `sessions`
   - `e` → `events`
   - `rt` → `runtime`
   - `p` → `phase`

---

## Runtime Issues

### "failed to build runtime"

**Cause:**
- `$ADP_RUNTIME_DIR` is not writable
- Disk space exhausted
- Symlink creation failed

**Diagnosis:**
```bash
# Check runtime directory
echo $ADP_RUNTIME_DIR
ls -ld $ADP_RUNTIME_DIR

# Check disk space
df -h $ADP_RUNTIME_DIR

# Check permissions
ls -ld $(dirname $ADP_RUNTIME_DIR)
```

**Solution:**
1. Set writable runtime directory:
   ```bash
   export ADP_RUNTIME_DIR="/tmp/adp-runtime"
   ```
2. Clean up old runtimes:
   ```bash
   adp runtime prune --older-than 24h
   ```
3. Check file system permissions

---

### "runtime directory not cleaned up"

**Cause:**
- Runtime was created with `--keep-runtime`
- Agent crashed before cleanup
- Manual inspection needed

**Diagnosis:**
```bash
# List kept runtimes
adp runtime prune --dry-run --include-kept
```

**Solution:**
1. Remove old runtimes:
   ```bash
   # Without kept runtimes
   adp runtime prune --older-than 1h

   # Including kept runtimes
   adp runtime prune --older-than 1h --include-kept
   ```

---

### "symlink conflicts in runtime"

**Cause:**
- Project files conflict with generated files
- Runtime was not cleaned properly

**Diagnosis:**
```bash
# Check runtime structure
ls -la $ADP_RUNTIME_ROOT

# Check workspace doctor
adp workspace doctor --verbose
```

**Solution:**
- Avoid files like `AGENTS.md`, `CLAUDE.md` in project root
- Clean runtime and try again
- Check workspace doctor recommendations

---

### "unsafe runtime parent"

**Cause:**
- `$ADP_RUNTIME_DIR` resolves to the filesystem root
- The runtime parent is the same as the project root
- The runtime parent is inside the project root (or vice versa)

ADP refuses these configurations to prevent accidental deletion of project files or the entire filesystem during runtime pruning.

**Diagnosis:**
```bash
# Check runtime parent and project root relationship
echo $ADP_RUNTIME_DIR
echo $ADP_HOME

# Inspect workspace project root
adp workspace show my-workspace

# Run doctor for safety checks
adp workspace doctor my-workspace --verbose
```

**Solution:**
1. Point `$ADP_RUNTIME_DIR` at a dedicated directory outside the project:
   ```bash
   export ADP_RUNTIME_DIR="/tmp/adp-runtime"
   ```
2. Never set the runtime parent to `/`, the project root, or any directory containing the project.
3. Re-run after fixing to confirm the safety check passes:
   ```bash
   adp workspace doctor my-workspace
   ```

---

### ".adp-runtime.yaml is reserved"

**Cause:**
- A project file or adapter-generated file uses the reserved name `.adp-runtime.yaml`
- This filename is owned by the ADP runtime manifest and cannot be overridden

**Diagnosis:**
```bash
# Search for the reserved file in the project
find . -name '.adp-runtime.yaml' -not -path '*/.adp-runtime/*'

# Inspect workspace adapters that generate files
adp workspace show my-workspace
```

**Solution:**
1. Rename or remove the conflicting project file:
   ```bash
   git mv .adp-runtime.yaml runtime-config.yaml
   ```
2. If an adapter generates it, update the adapter configuration to use a different output path.
3. Rebuild the runtime:
   ```bash
   adp run codex --workspace my-workspace --keep-runtime
   ```

---

## Task Management Issues

### "task not found"

**Cause:**
- Task ID is incorrect or ambiguous
- Task belongs to different workspace
- Task was deleted

**Diagnosis:**
```bash
# List all tasks
adp tasks list --workspace my-workspace

# Check task with prefix
adp tasks show task-2026
```

**Solution:**
1. Use correct task ID or unique prefix
2. Verify workspace name
3. Check task exists in task list

---

### "ambiguous task ID"

**Cause:**
- Prefix matches multiple tasks

**Diagnosis:**
```bash
# The error message lists all matches
adp tasks show task-20
# Error: ambiguous task ID "task-20", matches:
#   - task-20260611-0001
#   - task-20260612-0002
```

**Solution:**
- Use longer prefix to make it unique:
  ```bash
  adp tasks show task-20260611
  ```
- Or use full task ID

---

### "task already claimed"

**Cause:**
- Task is currently owned by another agent
- Lease has not expired yet

**Diagnosis:**
```bash
# Check task status
adp tasks show task-123

# Check stale tasks
adp tasks stale --workspace my-workspace
```

**Solution:**
- Wait for lease to expire
- Or release task if you own it:
  ```bash
  adp tasks release task-123 --owner current-owner
  ```

---

### "task owner mismatch"

**Cause:**
- You are trying to release, renew, or update a task owned by a different operator
- The `--owner` value does not match the task's current owner

**Diagnosis:**
```bash
# Check who currently owns the task
adp tasks show task-123
```

**Solution:**
1. Use the correct `--owner` matching the current owner.
2. If the previous owner is unavailable, wait for the lease to expire so the task becomes claimable again:
   ```bash
   adp tasks stale --workspace my-workspace
   ```
3. Coordinate ownership transfer explicitly rather than overriding another operator's task.

---

### "no claimable task"

**Cause:**
- `adp run --take` found no unclaimed, unblocked task in the workspace
- All tasks are either claimed, blocked, or done

**Diagnosis:**
```bash
# List tasks and their statuses
adp tasks list --workspace my-workspace

# Find the next claimable task explicitly
adp tasks next --workspace my-workspace

# Inspect blocked tasks individually
adp tasks show task-123
```

**Solution:**
1. Add a new task to claim:
   ```bash
   adp tasks add --workspace my-workspace "Next piece of work"
   ```
2. Unblock tasks that are waiting on prerequisites.
3. Wait for existing leases to expire if tasks are temporarily claimed.

---

## Phase Management Issues

### "phase not found"

**Cause:**
- Phase ID is misspelled or does not exist in the workspace
- The phase belongs to a different workspace

**Diagnosis:**
```bash
# List all phases
adp phase list --workspace my-workspace

# Show a specific phase
adp phase show --workspace my-workspace phase1
```

**Solution:**
1. Verify the phase ID against `adp phase list`.
2. Create the phase if it does not exist:
   ```bash
   adp phase add --workspace my-workspace phase1 "Phase 1 title"
   ```

---

### "invalid phase transition"

**Cause:**
- Attempting to start a phase that is not in an allowed status (e.g. already started)
- A blocking phase has not been accepted/pushed yet
- Recording acceptance, commit, or push evidence out of order

Phase gates enforce a strict lifecycle: a phase must be started before acceptance, accepted before commit evidence, and have commit evidence before push evidence.

**Diagnosis:**
```bash
# Inspect current phase status and gate requirements
adp phase status --workspace my-workspace phase1

adp phase show --workspace my-workspace phase1
```

**Solution:**
1. Follow the documented phase lifecycle in order:
   ```bash
   adp phase start --workspace my-workspace phase1
   adp phase accept --workspace my-workspace phase1
   adp phase commit --workspace my-workspace phase1 --hash <commit-hash>
   adp phase push --workspace my-workspace phase1 --remote origin --branch main
   ```
2. Resolve any blocking phase first — the error message names the blocking phase.

---

## Session Issues

### "session not found"

**Cause:**
- Session ID is incorrect or belongs to a different workspace
- Session prefix is too short

**Diagnosis:**
```bash
# List recent sessions
adp sessions list --workspace my-workspace --limit 20
```

**Solution:**
1. Use the full session ID from `adp sessions list`.
2. Verify the workspace name with `--workspace`.

---

### "ambiguous session ID"

**Cause:**
- The session ID prefix matches multiple sessions

**Diagnosis:**
```bash
# The error lists every matching session
adp sessions show session-2026
# Error: ambiguous session ID "session-2026", matches:
#   - session-20260611T120000-aaa
#   - session-20260612T140000-bbb
```

**Solution:**
- Provide a longer prefix or the full session ID:
  ```bash
  adp sessions show session-20260611T120000
  ```

---

## Interactive & Confirmation Issues

### "operation requires confirmation; use --yes to proceed in non-interactive mode"

**Cause:**
- A destructive operation was invoked in a non-interactive context (pipe, CI, script) without the `--yes` flag
- ADP protects `workspace remove` and `runtime prune --include-kept` from accidental deletion

**Diagnosis:**
```bash
# Reproduce in a non-TTY context
echo n | adp workspace remove my-workspace
```

**Solution:**
1. Pass `--yes` (or `-y`) to confirm non-interactively in scripts:
   ```bash
   adp workspace remove my-workspace --yes
   adp runtime prune --include-kept --older-than 1h --yes
   ```
2. `runtime prune --dry-run` never requires confirmation — preview first:
   ```bash
   adp runtime prune --include-kept --dry-run
   ```
3. In an interactive TTY, answer the `[y/N]` prompt directly.

---

## Parameter Validation Issues

### "--take cannot be combined with --task"

**Cause:**
- `adp run` was given both `--task <id>` (run a specific task) and `--take` (claim the next available task)

**Solution:**
- Choose one mode:
  ```bash
  # Run a specific task
  adp run codex --workspace my-workspace --task task-20260614-0001

  # Or claim and run the next available task
  adp run codex --workspace my-workspace --take --owner alice --lease 2h
  ```

---

### "--owner is required with --take"

**Cause:**
- `--take` claims a task and therefore requires an `--owner` to record who claimed it

**Solution:**
```bash
adp run codex --workspace my-workspace --take --owner alice --lease 2h
```

---

### "lease must not be negative" / "parse lease duration"

**Cause:**
- The `--lease` value is negative, zero, or not a valid Go duration string

**Diagnosis:**
```bash
# Valid duration formats
echo "Examples: 30m, 2h, 1h30m, 480m"
```

**Solution:**
- Use a positive Go duration:
  ```bash
  adp run codex --workspace my-workspace --take --owner alice --lease 2h
  adp tasks claim task-123 --owner alice --lease 90m
  ```

---

### "unknown option / unknown command"

**Cause:**
- A flag or subcommand name is misspelled or unsupported for that command

**Diagnosis:**
```bash
# Show accepted options for a command
adp run --help
adp tasks --help
```

**Solution:**
1. Check the command usage line printed with the error — it lists accepted arguments.
2. Use `--help` on the parent command to enumerate valid subcommands and flags.

---

## Environment Variables

### Environment Variables Not Working

**Cause:**
- Variables not exported
- Typo in variable name
- Shell not reloaded

**Diagnosis:**
```bash
# Check all ADP environment variables
env | grep ADP

# Check specific variables
echo $ADP_HOME
echo $ADP_RUNTIME_DIR
echo $ADP_WORKSPACE
```

**Solution:**
1. Export variables:
   ```bash
   export ADP_HOME="$HOME/.adp"
   export ADP_RUNTIME_DIR="/tmp/adp-runtime"
   ```
2. Add to shell profile for persistence:
   ```bash
   echo 'export ADP_HOME="$HOME/.adp"' >> ~/.bashrc
   source ~/.bashrc
   ```

---

### "dangerous Git environment variables"

**Cause:**
- Git-specific variables interfering with runtime

**Diagnosis:**
```bash
# Check Git environment
env | grep GIT_
```

**Solution:**
- ADP automatically neutralizes these during runtime
- If issues persist, unset manually:
  ```bash
  unset GIT_DIR GIT_WORK_TREE GIT_INDEX_FILE
  ```

---

## Permission Issues

### "permission denied" errors

**Cause:**
- Binary not executable
- Directory not writable
- File ownership issues

**Diagnosis:**
```bash
# Check binary permissions
ls -la $(which adp)

# Check ADP_HOME permissions
ls -ld $ADP_HOME

# Check runtime directory
ls -ld $ADP_RUNTIME_DIR
```

**Solution:**
1. Make binary executable:
   ```bash
   chmod +x /path/to/adp
   ```
2. Fix directory permissions:
   ```bash
   chmod 755 $ADP_HOME
   ```
3. Check file ownership:
   ```bash
   ls -la $ADP_HOME/workspaces/
   ```

---

## Diagnostic Commands

### Quick Health Check

```bash
# Check ADP installation
adp version

# Check environment
echo $ADP_HOME
echo $ADP_RUNTIME_DIR

# List workspaces
adp workspace list

# Run diagnostics on all workspaces
adp doctor

# Check specific workspace
adp workspace doctor my-workspace --verbose
```

---

### Debugging Task Issues

```bash
# List all tasks
adp tasks list --workspace my-workspace

# Show task details
adp tasks show task-123 --format json

# Check stale tasks
adp tasks stale --workspace my-workspace

# View task progress
adp progress --workspace my-workspace
```

---

### Debugging Runtime Issues

```bash
# Check runtime directory
ls -la $ADP_RUNTIME_DIR

# List runtimes (dry-run)
adp runtime prune --dry-run

# Clean old runtimes
adp runtime prune --older-than 1h

# Check events
adp events list --workspace my-workspace --limit 20
```

---

### Debugging Session Issues

```bash
# List recent sessions
adp sessions list --workspace my-workspace --limit 10

# Show session details
adp sessions show session-123

# Restore session plan
adp sessions restore-plan session-123
```

---

## Getting Help

If none of the above solutions work:

1. **Run diagnostics:**
   ```bash
   adp doctor --verbose --format json > diagnostics.json
   ```

2. **Check logs:**
   ```bash
   # Event logs
   cat $ADP_HOME/logs/events.jsonl

   # Recent events
   adp events list --limit 50
   ```

3. **Verify installation:**
   ```bash
   adp version
   go version
   ```

4. **Clean slate test:**
   ```bash
   # Use temporary ADP_HOME
   ADP_HOME=$(mktemp -d) adp init
   ```

5. **Report issue:**
   - Include `adp version` output
   - Include `adp doctor --verbose` output
   - Include relevant error messages
   - Describe steps to reproduce

---

## Common Patterns

### Fresh Start

```bash
# Backup existing ADP_HOME if needed
mv $ADP_HOME $ADP_HOME.backup

# Initialize fresh
adp init

# Re-add workspaces
adp workspace add my-project /path/to/project

# Verify
adp workspace doctor my-project
```

---

### Workspace Migration

```bash
# Export workspace config
adp workspace show old-workspace --format json > workspace.json

# Create new workspace with updated settings
adp workspace add new-workspace /new/path

# Migrate tasks if needed (manual process)
```

---

### Runtime Cleanup

```bash
# See what would be deleted
adp runtime prune --dry-run --older-than 0s

# Delete old runtimes
adp runtime prune --older-than 24h

# Include kept runtimes
adp runtime prune --older-than 24h --include-kept
```

---

For additional documentation:
- [Installation Guide](install.md)
- [Operator Onboarding](operator-onboarding.md)
- [Task Management](task-management.md)
- [Session Restore](session-restore.md)
