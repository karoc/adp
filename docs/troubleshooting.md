# Troubleshooting Guide

Simplified Chinese: [troubleshooting.zh-CN.md](troubleshooting.zh-CN.md)

This guide helps you diagnose and resolve common ADP issues. Issues are organized by error message or symptom for easy search.

---

## Table of Contents

- [Installation & Setup](#installation--setup)
- [Workspace Issues](#workspace-issues)
- [Runtime Issues](#runtime-issues)
- [Task Management Issues](#task-management-issues)
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
