# ADP Workshop Design

**Target Audience**: Developers and operators new to ADP who want hands-on experience  
**Total Duration**: 30 minutes  
**Format**: Terminal-first, interactive, progressive modules  
**Prerequisites**: Basic command-line familiarity, Go or project source code

---

## Design Philosophy

### Workshop vs Operator Onboarding

**Operator Onboarding** (`docs/operator-onboarding.md`):
- Comprehensive first-time setup guide (15-20 minutes)
- Covers installation, initialization, workspace creation
- Verification-focused with checkpoint guidance
- Reference documentation for troubleshooting
- Goal: Get ADP working on your machine

**Workshop** (this document):
- Hands-on skill-building exercises (30 minutes)
- Assumes ADP is already installed
- Task-driven learning with real scenarios
- Progressive complexity building on prior modules
- Goal: Learn core ADP workflows through practice

**Key Distinction**: Onboarding proves "it works"; workshop teaches "how to use it".

---

## Learning Objectives

By completing this workshop, participants will:

1. Understand ADP's workspace and runtime overlay model
2. Create and manage tasks across project phases
3. Run agents with task pickup and lease management
4. Inspect execution history through events and sessions
5. Debug runtime issues using diagnostics tools
6. Understand when to use different ADP commands

---

## Workshop Structure

The workshop follows Docker and Kubernetes workshop best practices:
- **Hands-on first**: Every concept paired with immediate practice
- **Progressive building**: Each module builds on previous skills
- **Verification checkpoints**: Clear "you should see" confirmations
- **Troubleshooting guidance**: "If this fails" recovery steps
- **Real scenarios**: Task-driven exercises matching actual workflows

### Module Overview

| Module | Topic | Duration | Complexity |
|--------|-------|----------|------------|
| 1 | Workspace Setup & Validation | 5 min | Foundation |
| 2 | Task Lifecycle Management | 10 min | Core |
| 3 | Runtime Inspection & Debugging | 10 min | Intermediate |
| 4 | Cross-Session Workflow | 5 min | Advanced |

**Total**: 30 minutes of active learning

---

## Module 1: Workspace Setup & Validation (5 minutes)

### Learning Objectives

- Create a workspace pointing to a real project
- Understand workspace configuration structure
- Validate setup before running agents
- Recognize common configuration issues

### Scenario

You're joining a project that uses ADP for agent coordination. Your first task is to set up your local ADP workspace and verify it's ready for agent runs.

### Hands-On Steps

**Step 1.1: Initialize ADP and create workspace**

```bash
# Initialize ADP home directory (if not already done)
export ADP_HOME="${HOME}/.adp"
adp init

# Add your project as a workspace
# Replace /path/to/your/project with an actual project directory
adp workspace add my-project /path/to/your/project

# List all workspaces to confirm
adp workspace list
```

**✓ You should see**: `my-project` workspace listed with your project path.

**Step 1.2: Inspect workspace configuration**

```bash
# Show detailed workspace info
adp workspace show my-project

# Check configuration health
adp workspace doctor my-project
```

**✓ You should see**: Workspace details including project root, and "ok - no issues" from doctor (or specific warnings to address).

**Step 1.3: Verify runtime environment**

```bash
# Check global diagnostics
adp doctor my-project --verbose

# Test environment variable export (dry-run, doesn't change shell)
adp env my-project
```

**✓ You should see**: Diagnostic report showing agent commands found, runtime paths validated, and shell commands that would set up the runtime environment.

### What You Learned

- **Workspace**: Maps a name to a project root + configuration
- **Doctor commands**: Validate setup before running agents
- **Runtime safety**: ADP checks runtime paths to prevent conflicts
- **Configuration visibility**: `show` and `doctor` expose what ADP will do

### Troubleshooting

**If workspace add fails with "not a directory"**:
- Verify the path exists: `ls -ld /path/to/your/project`
- Use absolute paths, not relative ones
- Ensure you have read permissions

**If doctor reports "agent command not found"**:
- This is expected if codex/claude CLI isn't installed
- For this workshop, we'll use a fake agent (see Module 2)
- For real work, install and authenticate the agent CLI

**If doctor warns about runtime directory**:
- Check `$ADP_RUNTIME_DIR` isn't inside your project
- Default `/tmp/adp-runtime` is usually safe
- See `adp doctor --verbose` for specific issues

---

## Module 2: Task Lifecycle Management (10 minutes)

### Learning Objectives

- Create tasks with priorities and descriptions
- Understand task states: pending → in_progress → completed
- Use atomic task pickup with `--take`
- Manage task ownership with leases
- Complete and track task progress

### Scenario

Your team uses ADP to coordinate agent work. You need to create a task, assign it to an agent, monitor progress, and mark it complete when done.

### Hands-On Steps

**Step 2.1: Create your first task**

```bash
# Add a high-priority task
TASK_ID=$(adp tasks add \
  --workspace my-project \
  --priority high \
  --description "Add authentication to the user service" \
  "Implement JWT-based auth" | \
  sed -n 's/^task \(task-[^ ]*\) added$/\1/p')

echo "Created task: $TASK_ID"

# View the task board
adp tasks list --workspace my-project
```

**✓ You should see**: A new task with status `pending` and priority `high`.

**Step 2.2: Preview available work**

```bash
# See what's ready to be picked up
adp tasks next --workspace my-project --format json | jq '.tasks[] | {id, title, priority, status}'
```

**✓ You should see**: Your task listed as available work in JSON format.

**Step 2.3: Set up a fake agent (workshop only)**

For this workshop, we'll create a simple fake agent to demonstrate task pickup without needing real codex/claude CLI:

```bash
# Create a workshop bin directory
mkdir -p ~/adp-workshop-bin

# Create fake agent that simulates work
cat > ~/adp-workshop-bin/workshop-agent <<'EOF'
#!/usr/bin/env bash
echo "Workshop agent started"
echo "Working directory: $(pwd)"
echo "Task ID: ${ADP_TASK_ID:-none}"
echo "Session ID: ${ADP_SESSION_ID:-none}"

# Simulate some work
echo "Processing task..."
sleep 2
echo "Task simulation complete"

# Verify runtime environment
test -f "$ADP_RUNTIME_ROOT/AGENTS.md" && echo "✓ Runtime overlay verified"
test -f "$ADP_RUNTIME_ROOT/.adp-runtime.yaml" && echo "✓ Runtime metadata present"
EOF

chmod +x ~/adp-workshop-bin/workshop-agent
export PATH="$HOME/adp-workshop-bin:$PATH"
```

**Step 2.4: Run agent with atomic task pickup**

```bash
# Atomically claim task and launch agent
adp run workshop-agent \
  --workspace my-project \
  --take \
  --owner alice \
  --lease 30m

# Check task status after run
adp tasks show --workspace my-project "$TASK_ID"
```

**✓ You should see**: 
- Agent output showing runtime environment variables
- Task status changed to `in_progress`
- Owner set to `alice`
- Lease expiration time set

**Step 2.5: Manage task ownership**

```bash
# Show tasks owned by alice
adp tasks list --workspace my-project | grep alice

# Renew the lease (simulate ongoing work)
adp tasks renew \
  --workspace my-project \
  "$TASK_ID" \
  --owner alice \
  --lease 1h

# Check for stale tasks (none yet, lease is fresh)
adp tasks stale --workspace my-project
```

**✓ You should see**: Lease expiration time extended by the renew command.

**Step 2.6: Complete the task**

```bash
# Mark task as done
adp tasks done --workspace my-project "$TASK_ID"

# View updated task board
adp tasks list --workspace my-project

# Check progress summary
adp progress --workspace my-project
```

**✓ You should see**: 
- Task status changed to `completed`
- Progress shows 1 completed task
- Task no longer appears in `tasks next` output

### What You Learned

- **Task states**: `pending` (available) → `in_progress` (claimed) → `completed` (done)
- **Atomic pickup**: `run --take` claims task and launches agent in one command
- **Ownership model**: Tasks have owners and time-limited leases
- **Lease management**: Renew leases for long-running work
- **Progress tracking**: View task completion across the workspace

### Troubleshooting

**If `run --take` fails with "no eligible tasks"**:
- Check tasks exist: `adp tasks list --workspace my-project`
- Verify task is `pending`: already claimed tasks aren't eligible
- Try explicit task ID: `adp run workshop-agent --workspace my-project --task $TASK_ID`

**If task stays `in_progress` after agent exits**:
- This is expected! Agent exit doesn't auto-complete tasks
- Manually complete: `adp tasks done --workspace my-project $TASK_ID`
- This allows inspection before marking done

**If lease expires**:
- View expired leases: `adp tasks stale --workspace my-project`
- Reclaim with: `adp tasks take --workspace my-project --owner alice`
- Or create new claim: `adp tasks claim $TASK_ID --owner bob --lease 2h`

---

## Module 3: Runtime Inspection & Debugging (10 minutes)

### Learning Objectives

- Inspect agent execution history via sessions and events
- Understand runtime overlay structure
- Use diagnostic commands to debug issues
- Examine task-to-session relationships
- Generate execution reports for handoffs

### Scenario

An agent ran earlier and encountered issues. You need to investigate what happened, inspect the runtime environment, and generate a report for your team.

### Hands-On Steps

**Step 3.1: Create a task with intentional complexity**

```bash
# Add a multi-step task
COMPLEX_TASK_ID=$(adp tasks add \
  --workspace my-project \
  --priority normal \
  --description "Refactor database layer with connection pooling, add retry logic, and update all service callsites" \
  "Refactor database connection handling" | \
  sed -n 's/^task \(task-[^ ]*\) added$/\1/p')

echo "Created task: $COMPLEX_TASK_ID"
```

**Step 3.2: Run agent with runtime inspection**

```bash
# Run agent and keep runtime for inspection
adp run workshop-agent \
  --workspace my-project \
  --task "$COMPLEX_TASK_ID" \
  --keep-runtime

# The runtime directory stays after agent exits
echo "Runtime kept at: $ADP_RUNTIME_ROOT"
```

**✓ You should see**: Agent output plus a message about kept runtime directory.

**Step 3.3: Inspect the runtime overlay**

```bash
# List recent sessions
adp sessions list --workspace my-project --limit 5

# Get the latest session ID
LATEST_SESSION=$(adp sessions list --workspace my-project | sed -n '2s/ .*//p')
echo "Latest session: $LATEST_SESSION"

# Show detailed session info
adp sessions show "$LATEST_SESSION"
```

**✓ You should see**: Session details including:
- Session ID and workspace
- Task ID that was being worked on
- Runtime path (if kept)
- Start/end times and duration
- Agent command and exit code

**Step 3.4: Examine session events**

```bash
# List all events for this task
adp events list \
  --workspace my-project \
  --task "$COMPLEX_TASK_ID" \
  --limit 10

# Filter by event type
adp events list \
  --workspace my-project \
  --type task_claimed

# Get events for specific session
adp events list \
  --workspace my-project \
  --session "$LATEST_SESSION"
```

**✓ You should see**: Chronological event log showing:
- `task_claimed` when agent picked up work
- `runtime_created` when overlay was built
- `agent_started` when agent launched
- `agent_exited` with exit code
- `runtime_kept` if `--keep-runtime` was used

**Step 3.5: Generate restore plan**

```bash
# Get restore guidance for the session
adp sessions restore-plan "$LATEST_SESSION"

# Get machine-readable plan
adp sessions restore-plan "$LATEST_SESSION" --format json | jq '.'

# Get cross-tool resume guidance
adp sessions resume-plan "$LATEST_SESSION" \
  --owner bob \
  --lease 2h \
  --agent claude
```

**✓ You should see**: 
- Suggested `adp run` command to rerun the session
- Owner/lease context for handoffs
- Cross-tool guidance when specifying different agent

**Step 3.6: Generate progress report**

```bash
# Create handoff report
adp progress report --workspace my-project

# Get machine-readable format
adp progress report --workspace my-project --format json | jq '.tasks[] | {id, title, status, owner}'
```

**✓ You should see**: Summary report with:
- Task counts by status
- Recent task activity
- Session evidence for completed work
- Ready-to-share progress snapshot

**Step 3.7: Clean up old runtimes**

```bash
# Preview cleanup (dry-run)
adp runtime prune --older-than 24h --dry-run

# Actually clean up (workshop: use short duration to see effect)
adp runtime prune --older-than 1m
```

**✓ You should see**: List of runtime directories that would be/were removed.

### What You Learned

- **Sessions**: Record of each agent run (task + runtime + timing)
- **Events**: Low-level log of all ADP operations
- **Runtime overlay**: Temporary directory with generated files + project symlinks
- **Restore plans**: Reproduce or resume previous sessions
- **Progress reports**: Generate handoff documentation from local state
- **Runtime cleanup**: Manage disk space with targeted pruning

### Troubleshooting

**If sessions list is empty**:
- Verify agents have run: `adp events list --workspace my-project`
- Check you're querying the right workspace
- Events exist but may not have session boundaries yet

**If runtime directory doesn't exist after `--keep-runtime`**:
- Runtime was already cleaned up
- Check: `ls $ADP_RUNTIME_DIR` for workspace-prefixed directories
- Use `--keep-runtime` on next run and inspect immediately

**If restore-plan shows "insufficient data"**:
- Session may be from older ADP version
- Try with a fresh session
- Use `sessions show` to see what data is available

---

## Module 4: Cross-Session Workflow (5 minutes)

### Learning Objectives

- Coordinate multi-agent workflows
- Hand off work between agents
- Use phases for project stages
- Understand task blocking relationships

### Scenario

Your project has multiple agents working on dependent tasks. You need to coordinate work handoffs and ensure tasks complete in the right order.

### Hands-On Steps

**Step 4.1: Create dependent tasks**

```bash
# Create a foundational task
FOUNDATION_TASK=$(adp tasks add \
  --workspace my-project \
  --priority high \
  --description "Set up database migrations framework" \
  "Database migration infrastructure" | \
  sed -n 's/^task \(task-[^ ]*\) added$/\1/p')

# Create a dependent task
DEPENDENT_TASK=$(adp tasks add \
  --workspace my-project \
  --priority normal \
  --description "Add user table migration" \
  "Create user schema migration" | \
  sed -n 's/^task \(task-[^ ]*\) added$/\1/p')

# Set up blocking relationship
adp tasks block \
  --workspace my-project \
  --task "$DEPENDENT_TASK" \
  --blocked-by "$FOUNDATION_TASK"

echo "Created dependent tasks:"
echo "  Foundation: $FOUNDATION_TASK"
echo "  Dependent:  $DEPENDENT_TASK"
```

**✓ You should see**: Two tasks created, with blocking relationship established.

**Step 4.2: Observe task visibility**

```bash
# Check what's available on the board
adp tasks next --workspace my-project

# Show the dependent task (won't be pickable yet)
adp tasks show --workspace my-project "$DEPENDENT_TASK"
```

**✓ You should see**: 
- Foundation task appears in `next` (available)
- Dependent task shows `blocked_by` field
- Dependent task NOT in `next` output (blocked)

**Step 4.3: Complete foundation and unblock dependent**

```bash
# Work on foundation task
adp run workshop-agent \
  --workspace my-project \
  --task "$FOUNDATION_TASK" \
  --owner alice \
  --lease 30m

# Complete foundation
adp tasks done --workspace my-project "$FOUNDATION_TASK"

# Check board again
adp tasks next --workspace my-project
```

**✓ You should see**: Dependent task now appears in `next` output (unblocked).

**Step 4.4: Hand off to different agent**

```bash
# Get resume guidance for handoff
FOUNDATION_SESSION=$(adp sessions list \
  --workspace my-project \
  --task "$FOUNDATION_TASK" | \
  sed -n '2s/ .*//p')

# Generate cross-agent handoff
adp sessions resume-plan "$FOUNDATION_SESSION" \
  --agent claude \
  --owner bob \
  --lease 1h

# Bob picks up dependent task
adp tasks claim "$DEPENDENT_TASK" \
  --workspace my-project \
  --owner bob \
  --lease 1h

# Check ownership
adp tasks list --workspace my-project | grep -E 'alice|bob|ID'
```

**✓ You should see**: 
- Resume plan with context for different agent
- Task ownership showing alice (foundation, completed) and bob (dependent, in_progress)

**Step 4.5: Generate final progress report**

```bash
# Complete dependent task
adp tasks done --workspace my-project "$DEPENDENT_TASK"

# Generate comprehensive report
adp progress report --workspace my-project

# Export as JSON for tools
adp progress report \
  --workspace my-project \
  --format json > /tmp/workshop-progress.json

echo "Report exported to: /tmp/workshop-progress.json"
cat /tmp/workshop-progress.json | jq '.summary'
```

**✓ You should see**: 
- Progress report showing both completed tasks
- Task counts and recent activity
- JSON export with machine-readable structure

### What You Learned

- **Task dependencies**: Use `block` to enforce completion order
- **Task visibility**: Blocked tasks don't appear in `next` until unblocked
- **Cross-agent handoffs**: Use `resume-plan` to provide context
- **Ownership tracking**: See who worked on what via task ownership
- **Progress reporting**: Generate reports at any point for status updates

### Troubleshooting

**If blocked task appears in `next` output**:
- Verify blocking relationship: `adp tasks show $TASK_ID`
- Check blocker is completed: `adp tasks list`
- Blocking only affects visibility, not claiming

**If `tasks claim` fails**:
- Task may already be claimed: check `tasks list` for owner
- Task may have expired lease: use `tasks take` instead
- Use `tasks stale` to see expired claims

---

## Workshop Completion

### Summary

You've completed the ADP workshop! You now know how to:

✅ Set up and validate ADP workspaces  
✅ Create and manage tasks through their lifecycle  
✅ Run agents with atomic task pickup  
✅ Inspect execution history via sessions and events  
✅ Debug issues using diagnostic commands  
✅ Coordinate multi-agent workflows with dependencies  
✅ Generate progress reports for team handoffs  

### Next Steps

**For daily use**:
1. Install real agent CLI: codex and/or claude
2. Set up your actual project workspace
3. Configure profiles in workspace config (see `examples/basic-workspace`)
4. Start with simple tasks and build confidence

**For advanced workflows**:
1. Explore phase management: `adp phase --help`
2. Set up MCP integrations for agents
3. Use workspace profiles for different agent configurations
4. Automate with plan intake: `adp plan preview/apply`

**Documentation resources**:
- Installation guide: [docs/install.md](../install.md)
- Operator onboarding: [docs/operator-onboarding.md](../operator-onboarding.md)
- Task management deep-dive: [docs/task-management.md](../task-management.md)
- Session restore: [docs/session-restore.md](../session-restore.md)
- Real agent setup: [docs/real-agent-compatibility.md](../real-agent-compatibility.md)

### Cleanup

Remove workshop artifacts:

```bash
# Remove fake agent
rm -rf ~/adp-workshop-bin

# Remove workshop workspace (optional)
adp workspace remove my-project

# Clean up runtimes
adp runtime prune --older-than 1h

# Keep $ADP_HOME for future use, or remove completely:
# rm -rf ~/.adp
```

---

## Appendix: Workshop Design Decisions

### Why These Modules?

**Module 1: Setup** - Foundation for everything else. Must validate before proceeding.  
**Module 2: Tasks** - Core workflow. Most users spend 80% of time here.  
**Module 3: Debugging** - Critical for self-service problem resolution.  
**Module 4: Coordination** - Shows real multi-agent value proposition.

### Time Allocation Rationale

- **5 min (Module 1)**: Quick wins build confidence
- **10 min (Module 2)**: Core skills deserve most time
- **10 min (Module 3)**: Deep inspection once basics are solid
- **5 min (Module 4)**: Advanced pattern as capstone

### Fake Agent Strategy

Real Codex/Claude CLI adds variables outside ADP's control:
- Installation and authentication friction
- Network dependencies and quota limits
- Model availability and response times
- Cost considerations for workshop participants

The fake `workshop-agent` proves ADP's task pickup, runtime overlay, and event tracking without external dependencies. Participants can focus on ADP concepts, not provider troubleshooting.

### Verification Checkpoint Pattern

Every step includes:
- **"✓ You should see"**: Clear success criteria
- **"Troubleshooting"**: Common failure modes and fixes
- **"What You Learned"**: Explicit knowledge capture

This pattern comes from Kubernetes and Docker workshops where clear success signals dramatically improve completion rates.

### Progressive Complexity

```
Module 1: Single workspace, read-only commands
         ↓
Module 2: Single task, single agent, basic state changes
         ↓
Module 3: Multiple sessions, inspection tools, cleanup
         ↓
Module 4: Multiple tasks, multiple agents, dependencies
```

Each module assumes only prior module knowledge. Participants who stop early still gain usable skills.

### Why Not Include?

**Excluded topics** (not missing, intentionally deferred):
- Phase management - more complex than 30min allows
- Plan intake (YAML/JSON) - power-user feature
- MCP integration - external dependency
- Workspace profiles - configuration detail overload
- Git integration - separate concern from ADP basics
- Real provider auth - environment-specific

These topics belong in follow-up workshops or documentation deep-dives, not foundational hands-on training.

---

## Design Research Sources

Workshop structure informed by:
- [Kubernetes Official Tutorials](https://kubernetes.io/docs/tutorials/kubernetes-basics/) - Progressive module design
- [Docker Official Workshop](https://docs.docker.com/guides/workshop/) - Build-run-share pattern
- [Educates Lab Framework](https://github.com/educates/lab-k8s-fundamentals) - Interactive checkpoint structure
- [KodeKloud Kubernetes Tutorials](https://kodekloud.com/blog/kubernetes-tutorial-for-beginners-2025/) - Zero-setup approach
- [Docker Best Practices Workshop](https://github.com/aabouzaid/docker-best-practices-workshop) - Hands-on ✅/🚫 format
- [SFEIR Kubernetes Practical Guides](https://institute.sfeir.com/en/kubernetes-training/kubernetes-tutorials-practical-guides/) - 30-minute module timing

Key insight from research: Most workshop failures happen during environment setup (40+ minutes of troubleshooting). The fake agent strategy eliminates this bottleneck while preserving learning objectives.

---

**Version**: 1.0  
**Last Updated**: 2026-06-14  
**Target ADP Version**: P3+ (task management, sessions, events)
