# ADP Workshop: Hands-On Learning Guide

Simplified Chinese: [workshop.zh-CN.md](workshop.zh-CN.md)

**⏱️ Total Time: 30 minutes**

This hands-on workshop teaches core ADP workflows through progressive, task-driven exercises. By the end, you'll understand workspace management, task coordination, agent execution, and debugging techniques.

**What you will learn:**
- ✅ Set up and validate ADP workspaces
- ✅ Create and manage tasks through their lifecycle
- ✅ Run agents with atomic task pickup
- ✅ Inspect execution history via sessions and events
- ✅ Debug issues using diagnostic commands
- ✅ Coordinate multi-agent workflows

**Prerequisites:**
- ADP installed (see [install.md](install.md))
- Go installed (for sample project)
- Basic command-line familiarity
- 30 minutes of focused time

**Workshop vs Onboarding:**
- **Operator Onboarding** ([operator-onboarding.md](operator-onboarding.md)): Proves "ADP works on your machine" (15-20 min)
- **This Workshop**: Teaches "how to use ADP effectively" through practice (30 min)

## Quick Start

Run the automated setup script:

```bash
cd examples/workshop
./setup.sh
```

This creates:
- Sample Go project at `~/adp-workshop-project`
- ADP workspace named `workshop`
- Fake `workshop-agent` command for learning

Then follow the modules below.

---

## Module 1: Workspace Setup & Validation

**⏱️ Time: 5 minutes**

### Learning Objectives

- Create and inspect workspaces
- Validate configuration before running agents
- Understand workspace diagnostics

### Scenario

You're joining a project using ADP. Set up your local workspace and verify it's ready for agent runs.

### Hands-On Steps

**Step 1.1: Verify your workshop environment**

```bash
# Check workspace exists (setup.sh created it)
adp workspace list

# View workspace details
adp workspace show workshop
```

**✓ You should see**: Workspace `workshop` pointing to `~/adp-workshop-project`.

**Step 1.2: Run diagnostics**

```bash
# Check workspace health
adp workspace doctor workshop

# Run comprehensive diagnostics
adp doctor workshop --verbose
```

**✓ You should see**: Status report showing workspace validation. Some warnings about agent commands are expected if codex/claude CLI isn't installed (that's fine—we're using `workshop-agent`).

**Step 1.3: Explore the sample project**

```bash
# Navigate to project
cd ~/adp-workshop-project

# Build the sample CLI
go build -o task-cli main.go

# Try it out
./task-cli add "Test task"
./task-cli list
```

**✓ You should see**: Simple task manager CLI working. Note: this project has an intentional bug you'll discover later!

### What You Learned

- **Workspace**: Maps a name (`workshop`) to project root + configuration
- **Doctor commands**: Validate setup before running agents
- **Configuration visibility**: `show` and `doctor` expose ADP's view of your environment

### Troubleshooting

**If workspace doesn't exist:**
- Run setup script: `./examples/workshop/setup.sh`
- Or manually: `adp workspace add workshop ~/adp-workshop-project`

**If doctor shows errors:**
- Check project directory exists: `ls -ld ~/adp-workshop-project`
- Warnings about missing codex/claude are OK for this workshop
- Use `--verbose` for detailed diagnostic output

---

## Module 2: Task Lifecycle Management

**⏱️ Time: 10 minutes**

### Learning Objectives

- Create tasks with descriptions and priorities
- Understand task state transitions
- Run agents with atomic `--take` pickup
- Manage task ownership and leases
- Complete and track progress

### Scenario

Create a task, assign it to an agent, monitor execution, and mark it complete.

### Hands-On Steps

**Step 2.1: Create your first task**

```bash
# Add a task
TASK_ID=$(adp tasks add \
  --workspace workshop \
  --priority high \
  "Fix bounds checking in CompleteTask function" | \
  sed -n 's/^task \(task-[^ ]*\) added$/\1/p')

echo "Created task: $TASK_ID"

# View task board
adp tasks list --workspace workshop
```

**✓ You should see**: New task with status `pending` and priority `high`.

**Step 2.2: Preview available work**

```bash
# See eligible tasks for pickup
adp tasks next --workspace workshop

# Machine-readable format
adp tasks next --workspace workshop --format json | jq '.'
```

**✓ You should see**: Your task listed as available work.

**Step 2.3: Run agent with atomic task pickup**

```bash
# Atomically claim task and launch agent
adp run workshop-agent \
  --workspace workshop \
  --take \
  --owner alice \
  --lease 30m
```

**✓ You should see**:
- Workshop agent banner and environment display
- Runtime overlay validation
- Simulated work steps
- Task claimed by `alice`

**Step 2.4: Inspect task status**

```bash
# Show task details
adp tasks show --workspace workshop "$TASK_ID"

# Check progress
adp progress --workspace workshop
```

**✓ You should see**: Task now has status `in_progress`, owner `alice`, and lease expiration time.

**Step 2.5: Manage leases**

```bash
# Renew lease (simulate ongoing work)
adp tasks renew \
  --workspace workshop \
  "$TASK_ID" \
  --owner alice \
  --lease 1h

# Check for stale tasks (none yet)
adp tasks stale --workspace workshop
```

**✓ You should see**: Updated lease expiration time.

**Step 2.6: Complete the task**

```bash
# Mark task done
adp tasks done --workspace workshop "$TASK_ID"

# View final progress
adp progress report --workspace workshop
```

**✓ You should see**: Task status `completed`, progress shows 1 completed task.

### What You Learned

- **Task states**: `pending` → `in_progress` → `completed`
- **Atomic pickup**: `run --take` claims + launches in one operation
- **Ownership**: Tasks have owners and time-limited leases
- **Lease management**: Renew for long-running work
- **Progress tracking**: Monitor workspace-wide completion

### Troubleshooting

**If `run --take` fails with "no eligible tasks":**
- Check tasks exist: `adp tasks list --workspace workshop`
- Ensure task is `pending` (not already claimed)
- Try explicit task: `adp run workshop-agent --workspace workshop --task $TASK_ID`

**If workshop-agent not found:**
- Check PATH: `which workshop-agent`
- Re-run setup: `./examples/workshop/setup.sh`
- Manually add: `export PATH="$HOME/.local/bin:$PATH"`

**If task stays `in_progress`:**
- Agent exit doesn't auto-complete tasks (by design)
- Manual completion allows inspection before marking done
- Complete with: `adp tasks done --workspace workshop $TASK_ID`

---

## Module 3: Runtime Inspection & Debugging

**⏱️ Time: 10 minutes**

### Learning Objectives

- Inspect agent execution history
- Understand runtime overlay structure
- Use diagnostic commands for debugging
- Examine task-to-session relationships
- Generate execution reports

### Scenario

Investigate a completed agent run to understand what happened and debug potential issues.

### Hands-On Steps

**Step 3.1: Create a task and keep runtime**

```bash
# Add a new task
DEBUG_TASK=$(adp tasks add \
  --workspace workshop \
  --priority normal \
  "Investigate task-cli memory usage" | \
  sed -n 's/^task \(task-[^ ]*\) added$/\1/p')

# Run agent with kept runtime
adp run workshop-agent \
  --workspace workshop \
  --task "$DEBUG_TASK" \
  --keep-runtime
```

**✓ You should see**: Agent runs and runtime directory is preserved after exit.

**Step 3.2: Inspect sessions**

```bash
# List recent sessions
adp sessions list --workspace workshop --limit 5

# Get latest session ID
LATEST_SESSION=$(adp sessions list --workspace workshop | sed -n '2s/ .*//p')
echo "Latest session: $LATEST_SESSION"

# Show session details
adp sessions show "$LATEST_SESSION"
```

**✓ You should see**: Session details including task ID, runtime path, duration, agent command, and exit code.

**Step 3.3: Examine events**

```bash
# List all events for this task
adp events list \
  --workspace workshop \
  --task "$DEBUG_TASK" \
  --limit 10

# Filter by event type
adp events list \
  --workspace workshop \
  --type task_claimed

# Session-specific events
adp events list \
  --workspace workshop \
  --session "$LATEST_SESSION"
```

**✓ You should see**: Event timeline showing `task_claimed`, `runtime_created`, `agent_started`, `agent_exited`, `runtime_kept`.

**Step 3.4: Explore runtime overlay**

```bash
# Find runtime directory (from session show output)
RUNTIME_DIR=$(adp sessions show "$LATEST_SESSION" | grep -oP 'runtime_path: \K.*')

if [ -d "$RUNTIME_DIR" ]; then
  echo "Runtime directory exists: $RUNTIME_DIR"
  
  # Inspect structure
  ls -la "$RUNTIME_DIR"
  
  # Check AGENTS.md
  cat "$RUNTIME_DIR/AGENTS.md"
  
  # Check runtime metadata
  cat "$RUNTIME_DIR/.adp-runtime.yaml"
else
  echo "Runtime already cleaned up. Run with --keep-runtime next time."
fi
```

**✓ You should see**: Runtime directory containing AGENTS.md, .adp-runtime.yaml, and symlinks to project files.

**Step 3.5: Generate restore plan**

```bash
# Get restore guidance
adp sessions restore-plan "$LATEST_SESSION"

# Machine-readable format
adp sessions restore-plan "$LATEST_SESSION" --format json | jq '.'

# Cross-agent handoff plan
adp sessions resume-plan "$LATEST_SESSION" \
  --agent claude \
  --owner bob \
  --lease 2h
```

**✓ You should see**: Suggested `adp run` commands to reproduce or resume the session, with context for handoffs.

**Step 3.6: Generate progress report**

```bash
# Human-readable report
adp progress report --workspace workshop

# Export as JSON
adp progress report --workspace workshop --format json > /tmp/workshop-progress.json
cat /tmp/workshop-progress.json | jq '.summary'
```

**✓ You should see**: Comprehensive report with task counts, recent activity, and session evidence.

**Step 3.7: Clean up runtimes**

```bash
# Preview cleanup (dry-run)
adp runtime prune --older-than 24h --dry-run

# Actually clean up workshop runtimes
adp runtime prune --older-than 5m
```

**✓ You should see**: List of runtime directories pruned.

### What You Learned

- **Sessions**: Record each agent run (task + runtime + timing)
- **Events**: Detailed log of all ADP operations
- **Runtime overlay**: Temporary directory with generated files
- **Restore plans**: Reproduce or resume sessions
- **Progress reports**: Generate status documentation
- **Runtime cleanup**: Manage disk space

### Troubleshooting

**If sessions list is empty:**
- Verify agents ran: `adp events list --workspace workshop`
- Check workspace name is correct

**If runtime directory missing:**
- Runtime already cleaned up
- Use `--keep-runtime` flag on next run
- Check: `ls /tmp/adp-runtime/workshop-*`

**If restore-plan shows "insufficient data":**
- Try with a fresh session
- Use `sessions show` to see available data

---

## Module 4: Cross-Session Workflow

**⏱️ Time: 5 minutes**

### Learning Objectives

- Coordinate multi-agent workflows
- Hand off work between agents
- Use task dependencies
- Track cross-agent progress

### Scenario

Coordinate multiple agents working on dependent tasks.

### Hands-On Steps

**Step 4.1: Create dependent tasks**

```bash
# Create foundation task
FOUNDATION=$(adp tasks add \
  --workspace workshop \
  --priority high \
  "Add unit tests for TaskManager" | \
  sed -n 's/^task \(task-[^ ]*\) added$/\1/p')

# Create dependent task
DEPENDENT=$(adp tasks add \
  --workspace workshop \
  --priority normal \
  "Add integration tests using TaskManager" | \
  sed -n 's/^task \(task-[^ ]*\) added$/\1/p')

# Set up dependency
adp tasks block \
  --workspace workshop \
  --task "$DEPENDENT" \
  --blocked-by "$FOUNDATION"

echo "Foundation task: $FOUNDATION"
echo "Dependent task:  $DEPENDENT (blocked)"
```

**✓ You should see**: Two tasks created with blocking relationship.

**Step 4.2: Observe task visibility**

```bash
# Check available work
adp tasks next --workspace workshop

# Show dependent task details
adp tasks show --workspace workshop "$DEPENDENT"
```

**✓ You should see**: Foundation task appears in `next`, dependent task shows `blocked_by` and doesn't appear in `next`.

**Step 4.3: Complete foundation and unblock**

```bash
# Work on foundation
adp run workshop-agent \
  --workspace workshop \
  --task "$FOUNDATION" \
  --owner alice \
  --lease 30m

# Complete it
adp tasks done --workspace workshop "$FOUNDATION"

# Check board again
adp tasks next --workspace workshop
```

**✓ You should see**: Dependent task now appears in `next` output (unblocked).

**Step 4.4: Hand off to different agent**

```bash
# Get foundation session
FOUNDATION_SESSION=$(adp sessions list \
  --workspace workshop \
  --task "$FOUNDATION" | \
  sed -n '2s/ .*//p')

# Generate cross-agent handoff
adp sessions resume-plan "$FOUNDATION_SESSION" \
  --agent claude \
  --owner bob \
  --lease 1h

# Bob picks up dependent task
adp tasks claim "$DEPENDENT" \
  --workspace workshop \
  --owner bob \
  --lease 1h

# Check ownership
adp tasks list --workspace workshop
```

**✓ You should see**: Resume plan with context, task ownership showing alice (completed) and bob (in_progress).

**Step 4.5: Generate final report**

```bash
# Complete dependent task
adp tasks done --workspace workshop "$DEPENDENT"

# Final progress report
adp progress report --workspace workshop

# Export for tools
adp progress report --workspace workshop --format json > /tmp/workshop-final.json
cat /tmp/workshop-final.json | jq '.summary'
```

**✓ You should see**: Report showing multiple completed tasks with session evidence.

### What You Learned

- **Task dependencies**: Enforce completion order with `block`
- **Task visibility**: Blocked tasks hidden from `next` until unblocked
- **Cross-agent handoffs**: Use `resume-plan` for context
- **Ownership tracking**: See who worked on what
- **Progress reporting**: Generate reports at any time

### Troubleshooting

**If blocked task appears in `next`:**
- Verify blocking: `adp tasks show $TASK_ID | grep blocked_by`
- Check blocker is completed
- Blocking affects visibility, not claiming

**If `tasks claim` fails:**
- Task may be already claimed
- Check: `adp tasks list --workspace workshop`
- Use `tasks take` for expired leases

---

## Workshop Completion

### Summary

✅ **Congratulations!** You've completed the ADP workshop.

You now know how to:
- Set up and validate workspaces
- Create and manage tasks through their lifecycle
- Run agents with atomic pickup
- Inspect execution history
- Debug with diagnostic commands
- Coordinate multi-agent workflows
- Generate progress reports

### Next Steps

**For daily use:**
1. Install real agent CLI (codex/claude)
2. Set up your actual project workspace
3. Configure profiles (see `examples/basic-workspace`)
4. Start with simple tasks

**For advanced workflows:**
1. Explore phase management: `adp phase --help`
2. Set up MCP integrations
3. Use workspace profiles
4. Automate with plan intake

**Documentation:**
- [Installation](install.md)
- [Operator Onboarding](operator-onboarding.md)
- [Task Management](task-management.md)
- [Session Restore](session-restore.md)
- [Real Agent Setup](real-agent-compatibility.md)

### Cleanup

Remove workshop artifacts:

```bash
# Remove fake agent
rm ~/.local/bin/workshop-agent

# Remove workspace (optional)
adp workspace remove workshop

# Remove project (optional)
rm -rf ~/adp-workshop-project

# Clean up runtimes
adp runtime prune --older-than 1h
```

---

## FAQ

**Q: Why use a fake agent instead of real codex/claude?**  
A: The fake agent eliminates setup friction (installation, authentication, network, quota) so you can focus on learning ADP concepts. For real work, install the actual agent CLI.

**Q: Is the bug in task-cli intentional?**  
A: Yes! The off-by-one error in `CompleteTask` is deliberate. It demonstrates discovering issues during development, which you'd coordinate with ADP tasks.

**Q: Can I use this workshop for team training?**  
A: Absolutely! The setup script makes it reproducible. Each person runs `./setup.sh` and follows the modules.

**Q: How is this different from operator-onboarding.md?**  
A: Onboarding verifies installation (15-20 min, validation-focused). Workshop teaches usage (30 min, hands-on skill-building).

**Q: What if I want to try with real agents?**  
A: After completing the workshop, see [real-agent-compatibility.md](real-agent-compatibility.md) for codex/claude setup, then repeat modules with real agents.

---

**Version**: 1.0  
**Last Updated**: 2026-06-14  
**Feedback**: Open an issue or PR with suggestions
