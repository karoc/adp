# ADP FAQ

> Frequently Asked Questions about Agent Development Platform

Simplified Chinese: [faq.zh-CN.md](faq.zh-CN.md)

This FAQ covers conceptual questions about ADP's architecture, usage decisions, and integration scenarios. For error resolution and diagnostic procedures, see the [Troubleshooting Guide](troubleshooting.md).

---

## Table of Contents

1. [Core Concepts](#core-concepts)
2. [Usage Decisions](#usage-decisions)
3. [Team Collaboration](#team-collaboration)
4. [Integration Scenarios](#integration-scenarios)
5. [Advanced Topics](#advanced-topics)

---

## Core Concepts

### Q1: What is ADP and why would I use it?

**Short Answer:**

ADP (Agent Development Platform) is a terminal-first workspace manager that keeps AI agent configuration files outside your project directory. Use it when you work with multiple projects, need reusable agent configurations, or want to keep your project root clean from agent-generated files.

**Details:**

Without ADP, running `codex` or `claude` directly creates configuration files like `AGENTS.md`, `CLAUDE.md`, `.codex/`, and `.claude/` in your project root. This causes:
- Git noise from untracked agent files in `git status`
- Configuration conflicts when switching between agents or projects
- Manual cleanup and .gitignore maintenance

ADP solves this by:
1. **Workspace Management**: Store agent configurations (profiles, prompts, memory, MCP settings) in `~/.adp` instead of project roots
2. **Runtime Overlays**: Create temporary directories that combine your project files (via symlinks) with generated agent files
3. **Reusability**: Define configurations once, reuse across multiple agent runs
4. **Team Consistency**: Share workspace configurations as templates without committing local state

**When to use ADP:**
- Multiple projects that need different agent configurations
- Team environments requiring consistent agent setups
- Projects where keeping the root clean is important
- Workflows needing task tracking and ownership management

**When NOT to use ADP:**
- One-off exploratory agent sessions
- Single project with no configuration reuse
- Quick experiments where setup overhead isn't justified

**Example:**

```bash
# Without ADP: agent files pollute project root
cd /srv/my-project
codex
# Creates: AGENTS.md, .codex/ in /srv/my-project

# With ADP: project root stays clean
adp workspace add my-project /srv/my-project
adp run codex --workspace my-project
# Config in ~/.adp, runtime in /tmp, project unchanged
```

**See also:**
- [README: Runtime Model](../README.md#runtime-model)
- [Operator Onboarding](operator-onboarding.md)
- [Q7: When should I use ADP vs direct CLI?](#q7-when-should-i-use-adp-instead-of-running-codex-or-claude-directly)

---

### Q2: What is a workspace?

**Short Answer:**

A workspace is a named configuration set for running agents in a specific project. It defines where your project lives, which agent profiles to use, what prompts to inject, and what MCP servers to enable.

**Details:**

A workspace is stored as a directory under `$ADP_HOME/workspaces/<workspace-name>/` and contains:

- **workspace.yaml**: Core configuration (project root path, adapter settings)
- **profiles/**: Agent-specific configurations (codex-architect.yaml, claude-reviewer.yaml)
- **prompts/**: Reusable instruction files (base-prompt.md, task-instructions.md)
- **memory/**: Shared context files (project-conventions.md, architecture-decisions.md)
- **mcp/**: MCP server configurations (servers.json)
- **planning/**: Task and phase state (tasks.yaml, phases.yaml, progress.jsonl)

**Analogy:** A workspace is to agent configuration what a Git remote is to a repository URL—a named reference that makes repeated operations convenient.

**Example:**

```bash
# Create workspace
adp workspace add game-a /srv/game-a

# Workspace structure
~/.adp/workspaces/game-a/
├── workspace.yaml
├── profiles/
│   ├── codex-architect.yaml
│   └── claude-engineer.yaml
├── prompts/
│   └── base-prompt.md
└── planning/
    └── tasks.yaml

# Use workspace
adp run codex --workspace game-a
adp run claude --workspace game-a --profile engineer
```

**Common Pitfalls:**
- ⚠️ Workspace names must be unique per `$ADP_HOME`
- ⚠️ Project root must be an absolute path
- ⚠️ Workspace configs are local; share as templates, not by committing `$ADP_HOME`

**See also:**
- [examples/basic-workspace](../examples/basic-workspace/) - copyable workspace template
- [Q12: How do I share workspace configurations?](#q12-how-do-i-share-workspace-configurations-with-my-team)

---

### Q3: What is the runtime overlay?

**Short Answer:**

A runtime overlay is a temporary directory that combines your real project files (via symlinks) with ADP-generated agent configuration files (AGENTS.md, CLAUDE.md, .codex/, .claude/). Agents work in this overlay, keeping your actual project directory clean.

**How It Works:**

When you run `adp run <agent>`, ADP creates a temporary directory under `$ADP_RUNTIME_DIR` and builds a view that looks like your project root:
- Generated files (AGENTS.md, CLAUDE.md, .codex/config.toml, .claude/settings.json) are real files written to the overlay
- Project files (source code, go.mod, package.json, etc.) are symlinked from your real project
- The agent's working directory is set to this overlay root

**Why This Matters:**

Without overlays, agents would write their configuration files directly into your project. This causes:
- Git noise: untracked AGENTS.md, CLAUDE.md polluting `git status`
- Conflicts: different agents overwriting each other's configs
- Cleanup burden: remembering to .gitignore agent-specific paths

Runtime overlays solve this by giving each agent run an isolated view with consistent configuration, then cleaning up automatically.

**Example:**

```bash
# Real project
/srv/my-project/
├── go.mod
├── main.go
└── internal/

# Runtime overlay (created during `adp run codex`)
/tmp/adp-runtime/my-project-20260614T120000-abc123/
├── AGENTS.md              # generated
├── .codex/                # generated
│   └── config.toml
├── go.mod -> /srv/my-project/go.mod        # symlink
├── main.go -> /srv/my-project/main.go      # symlink
└── internal -> /srv/my-project/internal    # symlink

# Agent works here ↑, real project unchanged ↓
```

**Lifecycle:**

1. `adp run` creates overlay before launching agent
2. Agent sees overlay as working directory (`$ADP_RUNTIME_ROOT`)
3. On exit, ADP removes overlay (unless `--keep-runtime` was used)
4. Use `adp runtime prune` to clean up kept or stale overlays

**Common Pitfalls:**
- ⚠️ Don't rely on overlay contents after agent exits (they're deleted by default)
- ⚠️ Don't point `$ADP_RUNTIME_DIR` inside your project (doctor will warn)
- ⚠️ Kept runtimes accumulate disk space; prune regularly with `adp runtime prune`

**See also:**
- [README: Runtime Model](../README.md#runtime-model) - detailed overlay mechanics
- [Q4: What files does ADP create and where?](#q4-what-files-does-adp-create-and-where)
- [Q10: Should I use --keep-runtime?](#q10-should-i-use---keep-runtime-or-let-adp-clean-up)

---

### Q4: What files does ADP create and where?

**Short Answer:**

ADP creates files in two places: durable configuration under `$ADP_HOME` (default `~/.adp`) and temporary runtime overlays under `$ADP_RUNTIME_DIR` (default `/tmp/adp-runtime`). Your real project root is **never** modified.

**File Locations:**

| File Type | Location | Lifetime | Purpose |
|-----------|----------|----------|---------|
| Workspace configs | `$ADP_HOME/workspaces/<name>/` | Durable | Profiles, prompts, memory, MCP settings |
| Task state | `$ADP_HOME/workspaces/<name>/planning/` | Durable | tasks.yaml, phases.yaml, progress.jsonl |
| Event logs | `$ADP_HOME/logs/events.jsonl` | Durable | Session history, runtime events |
| Runtime overlays | `$ADP_RUNTIME_DIR/<name>-<timestamp>/` | Temporary | AGENTS.md, CLAUDE.md, .codex/, .claude/, symlinks |
| Project files | Project root (unchanged) | N/A | Your source code, configs, etc. |

**Details:**

**$ADP_HOME (Durable State)**
- Default: `~/.adp`
- Set explicitly: `export ADP_HOME=/path/to/adp-home`
- Contains workspace registry, task ledger, session logs
- Survives restarts; safe to keep long-term
- Back up this directory to preserve workspace configurations

**$ADP_RUNTIME_DIR (Temporary Overlays)**
- Default: `$TMPDIR/adp-runtime` or `/tmp/adp-runtime`
- Set explicitly: `export ADP_RUNTIME_DIR=/path/to/runtime`
- Contains only active or kept runtime directories
- Cleaned automatically unless `--keep-runtime` used
- Prune with `adp runtime prune --older-than 24h`

**Project Root (Never Modified)**
- ADP never writes AGENTS.md, CLAUDE.md, .codex/, .claude/, or planning files to your project
- Agents see project files via symlinks in the runtime overlay
- `adp workspace doctor` verifies no ADP files leaked into project root

**Example Inspection:**

```bash
# Check ADP home structure
ls -la ~/.adp/
# workspaces/  logs/

# Check runtime overlays
ls -la /tmp/adp-runtime/
# game-a-20260614T120000-abc123/  (if kept)

# Verify project root is clean
cd /srv/my-project
ls -la
# No AGENTS.md, CLAUDE.md, .codex/, .claude/ here
```

**See also:**
- [Operator Onboarding: Isolated First Run](operator-onboarding.md#isolated-first-run)
- [Q3: What is the runtime overlay?](#q3-what-is-the-runtime-overlay)

---

### Q5: How does ADP relate to Codex and Claude?

**Short Answer:**

ADP is a runtime environment manager for agent CLIs, not a replacement. Codex and Claude are the agents that do the work; ADP provides consistent configuration, workspace isolation, and task coordination across multiple agent runs.

**Analogy:**

ADP is to agent CLIs what Docker is to application processes:
- Docker manages container environments; ADP manages runtime overlays
- Docker isolates process state; ADP isolates agent configuration
- Docker doesn't replace your app; ADP doesn't replace Codex or Claude

**What ADP Does:**
1. **Configuration Management**: Generate AGENTS.md, CLAUDE.md, .codex/config.toml, .claude/settings.json from workspace templates
2. **Runtime Isolation**: Build temporary overlays so agents see consistent config without polluting project roots
3. **Task Coordination**: Track work ownership, leases, and handoffs across agent runs
4. **Session History**: Record events and provide resume guidance for cross-agent workflows

**What Codex/Claude Do:**
- Process natural language instructions
- Read and modify project files
- Execute code, run tests, debug issues
- Provide interactive agent experiences

**Example Workflow:**

```bash
# ADP sets up the environment
adp run codex --workspace game-a --task task-001
# → ADP creates runtime overlay with generated configs
# → ADP launches: codex --config /tmp/.../codex/config.toml
# → Codex does the actual work in that environment

# Later, hand off to Claude
adp sessions resume-plan <session-id> --agent claude --owner reviewer
# → ADP suggests: adp run claude --workspace game-a --task task-001
# → Claude continues in fresh runtime with same task context
```

**See also:**
- [README: Core Value Proposition](../README.md)
- [Real Agent Compatibility](real-agent-compatibility.md)

---

### Q6: What are tasks and phases?

**Short Answer:**

Tasks are individual work items with ownership and leases. Phases are stage gates for release discipline (planning → acceptance → commit → push). Both are optional—you can use `adp run` without task management.

**Tasks:**

Tasks are work items tracked in `$ADP_HOME/workspaces/<name>/planning/tasks.yaml`. Each task has:
- **ID**: Unique identifier (task-20260614-0001)
- **Title**: Brief description
- **Status**: pending → in_progress → completed
- **Owner**: Who's working on it (optional)
- **Lease**: How long ownership lasts (optional, e.g., 2h, 30m)
- **Priority**: high, normal, low
- **Phase**: Which release phase it belongs to

**Phases:**

Phases enforce stage gates for structured release workflows. Common phases:
1. **planning**: Define and scope work
2. **implementation**: Build features, fix bugs
3. **acceptance**: Review and validate
4. **commit**: Record changes in Git
5. **push**: Publish to remote

Phase gates prevent skipping steps: you can't commit before acceptance, can't push before commit, etc.

**When to Use Tasks:**
- Multi-agent coordination (multiple workers picking from shared board)
- Work handoff (operator A starts, operator B continues)
- Audit trail (who worked on what, when)
- Lease management (prevent conflicts in concurrent workflows)

**When to Use Phases:**
- Structured release discipline
- Team workflows requiring review gates
- Compliance needs (evidence of acceptance before release)

**Example:**

```bash
# Create task
adp tasks add --workspace game-a --phase implementation "Fix auth bug"
# task-20260614-0001 added

# Claim and work on it
adp run codex --workspace game-a --take --owner alice --lease 2h

# Check progress
adp tasks show task-20260614-0001
# Status: in_progress, Owner: alice, Expires: 2026-06-14 14:00

# Complete task
adp tasks done task-20260614-0001
```

**See also:**
- [Task Management Guide](task-management.md)
- [Q8: When should I use tasks vs direct adp run?](#q8-when-should-i-use-tasks-vs-direct-adp-run)
- [Q14: How do multiple operators coordinate on shared tasks?](#q14-how-do-multiple-operators-coordinate-on-shared-tasks)

---

## Usage Decisions

### Q7: When should I use ADP instead of running `codex` or `claude` directly?

**Short Answer:**

Use ADP when you need reusable configurations, multiple project management, team consistency, or task tracking. Use direct CLI for one-off exploration and quick experiments.

**Decision Matrix:**

| Scenario | Use ADP | Use Direct CLI |
|----------|---------|----------------|
| Multiple projects with different configs | ✅ | ❌ |
| Reusable agent profiles (architect, reviewer) | ✅ | ❌ |
| Team needing consistent setup | ✅ | ❌ |
| Task tracking and ownership | ✅ | ❌ |
| Keep project root clean | ✅ | ❌ |
| One-off exploration | ❌ | ✅ |
| Quick experiment (no setup) | ❌ | ✅ |
| Interactive session with no reuse | ❌ | ✅ |

**Use ADP When:**
1. Working on 2+ projects that need different agent configurations
2. Team members need to share consistent agent setups
3. You want project roots free of AGENTS.md, CLAUDE.md, .codex/, .claude/
4. Multiple agents coordinate on shared tasks with ownership tracking
5. You need audit trails of who worked on what

**Use Direct CLI When:**
1. Exploring a new tool or project for the first time
2. Running a quick one-off command with no reuse
3. Interactive experimentation where setup overhead isn't worth it
4. Single project with no configuration complexity

**Example Comparison:**

```bash
# Direct CLI: quick but leaves files in project
cd /srv/my-project
codex
# Creates: /srv/my-project/AGENTS.md, /srv/my-project/.codex/

# ADP: more setup, but project stays clean
adp workspace add my-project /srv/my-project
adp run codex --workspace my-project
# Config in ~/.adp, runtime in /tmp, /srv/my-project unchanged
```

**See also:**
- [Q1: What is ADP and why would I use it?](#q1-what-is-adp-and-why-would-i-use-it)
- [Operator Onboarding](operator-onboarding.md)

---

### Q8: When should I use tasks vs direct `adp run`?

**Short Answer:**

Use tasks when you need work coordination, ownership tracking, or audit trails. Use direct `adp run` for exploratory work or one-shot execution where tracking isn't needed.

**With Tasks:**

Benefits:
- Ownership tracking (who's working on what)
- Lease management (prevent concurrent work conflicts)
- Work handoff (start task, hand to another operator)
- Progress visibility (`adp tasks list`, `adp progress report`)
- Session history linked to task context
- Priority ordering for board-based workflows

Workflow:
```bash
# Add task
adp tasks add --workspace game-a "Fix auth bug"

# Pick from board
adp run codex --workspace game-a --take --owner alice --lease 2h

# Check progress
adp tasks show task-20260614-0001

# Complete
adp tasks done task-20260614-0001
```

**Without Tasks:**

Benefits:
- Simpler workflow (no task management overhead)
- Faster for one-off work
- No ownership coordination needed

Workflow:
```bash
# Just run the agent
adp run codex --workspace game-a
```

**Decision Guide:**

Use tasks when:
- Multiple agents/operators coordinate on shared work
- You need "who worked on this?" audit trail
- Work spans multiple sessions or handoffs
- Preventing concurrent work conflicts matters

Skip tasks when:
- Exploratory work (no reuse or tracking needed)
- Single operator, no coordination needed
- Interactive session with no handoff
- Quick fixes where overhead isn't justified

**See also:**
- [Task Management Guide](task-management.md)
- [Q6: What are tasks and phases?](#q6-what-are-tasks-and-phases)
- [Q9: How do I choose between --task and --take?](#q9-how-do-i-choose-between-adp-run---task-and-adp-run---take)

---

### Q9: How do I choose between `adp run --task` and `adp run --take`?

**Short Answer:**

Use `--task <id>` when you know exactly which task to work on. Use `--take --owner <owner> --lease <duration>` to atomically claim the first available task from the board and launch the agent.

**Three Patterns:**

**1. Explicit Task Assignment (`--task <id>`)**

When to use:
- You or a previous step already identified the specific task
- Task was pre-assigned to you
- Re-running work on a known task

```bash
# Explicit task targeting
adp run codex --workspace game-a --task task-20260614-0001
```

**2. Atomic Board Pickup (`--take --owner --lease`)**

When to use:
- Worker should pick first available task from board
- Atomic claim + launch in one command
- Prevent race conditions in concurrent workflows

```bash
# Claim first available task and launch
adp run codex --workspace game-a --take --owner alice --lease 2h
```

**3. Claim Without Launch (`adp tasks take`)**

When to use:
- Claim a task but don't launch agent immediately
- Review task before starting work
- Batch claim for later execution

```bash
# Claim first, run later
TASK_ID=$(adp tasks take --workspace game-a --owner alice --lease 2h | grep -o 'task-[^ ]*')
# Review task details
adp tasks show $TASK_ID
# Then launch when ready
adp run codex --workspace game-a --task $TASK_ID
```

**Decision Flow:**

```
Do you know the specific task ID?
├─ Yes → Use --task <id>
└─ No → Do you need to review before starting?
    ├─ Yes → Use `adp tasks take` first, then --task
    └─ No → Use --take --owner --lease (atomic)
```

**Example Scenarios:**

```bash
# Scenario 1: Pre-assigned task
# Team lead: "Alice, work on task-20260614-0001"
adp run codex --workspace game-a --task task-20260614-0001

# Scenario 2: Worker picks from board
adp run codex --workspace game-a --take --owner alice --lease 4h

# Scenario 3: Review before starting
adp tasks take --workspace game-a --owner alice --lease 2h
# task-20260614-0002 taken
adp tasks show task-20260614-0002
# (review details)
adp run codex --workspace game-a --task task-20260614-0002
```

**See also:**
- [Task Management Guide: Workflow At A Glance](task-management.md#workflow-at-a-glance)
- [Q8: When should I use tasks vs direct adp run?](#q8-when-should-i-use-tasks-vs-direct-adp-run)

---

### Q10: Should I use `--keep-runtime` or let ADP clean up?

**Short Answer:**

Let ADP clean up by default (automatic cleanup on exit). Use `--keep-runtime` only when debugging runtime issues or when you need to manually inspect the generated files.

**When to Keep Runtime:**

Use `--keep-runtime` when:
- Debugging runtime overlay construction issues
- Manually inspecting generated AGENTS.md, CLAUDE.md, adapter configs
- Verifying symlink structure
- Troubleshooting agent launch failures

```bash
# Keep runtime for inspection
adp run codex --workspace game-a --keep-runtime
# Runtime preserved at: /tmp/adp-runtime/game-a-20260614T120000-abc123

# Inspect generated files
ls -la /tmp/adp-runtime/game-a-20260614T120000-abc123
cat /tmp/adp-runtime/game-a-20260614T120000-abc123/AGENTS.md
```

**When to Let ADP Clean Up:**

Default behavior (no `--keep-runtime`) when:
- Normal workflow (no debugging needed)
- CI/CD pipelines (clean state per run)
- Automated agent runs
- Disk space is constrained

```bash
# Automatic cleanup (default)
adp run codex --workspace game-a
# Runtime removed on exit
```

**Managing Kept Runtimes:**

Kept runtimes accumulate disk space. Clean them periodically:

```bash
# Preview what would be deleted
adp runtime prune --older-than 24h --dry-run

# Actually delete old runtimes
adp runtime prune --older-than 24h

# Delete all kept runtimes (careful!)
adp runtime prune --older-than 0s --include-kept
```

**Common Pitfalls:**
- ⚠️ Kept runtimes don't auto-expire; clean them manually
- ⚠️ Don't rely on kept runtime contents after fixing issues
- ⚠️ In CI/CD, always clean up: `adp runtime prune --older-than 0s`

**See also:**
- [Q3: What is the runtime overlay?](#q3-what-is-the-runtime-overlay)
- [README: Runtime Prune](../README.md#runtime-model)

---

### Q11: When should I use profiles vs workspace defaults?

**Short Answer:**

Use profiles for role-specific configurations (architect, engineer, reviewer). Use workspace defaults for consistent team baseline that applies to all runs.

**Profile Hierarchy:**

Configuration is layered: **Profile overrides Workspace overrides Adapter defaults**

```
Adapter defaults (built-in)
    ↓
Workspace defaults (workspace.yaml)
    ↓
Profile settings (profiles/architect.yaml)
```

**When to Use Workspace Defaults:**

Set in `workspace.yaml` for baseline configuration that applies to all agent runs:
- Shared base prompts for the entire project
- Common memory files all agents should see
- Shared MCP server configurations
- Default model settings

Example workspace.yaml:
```yaml
codex:
  base_prompt: prompts/base-prompt.md
  memory:
    - memory/project-conventions.md
  mcp:
    servers: mcp/servers.json
```

**When to Use Profiles:**

Create profiles in `profiles/` for role-specific variations:
- Different instruction sets (architecture vs implementation)
- Role-specific memory (senior engineer context vs code reviewer guidelines)
- Varying model settings (fast model for simple tasks, powerful model for architecture)

Example profiles:
```yaml
# profiles/architect.yaml
codex:
  base_prompt: prompts/architect-prompt.md
  memory:
    - memory/architecture-decisions.md
    - memory/design-patterns.md

# profiles/reviewer.yaml
codex:
  base_prompt: prompts/reviewer-prompt.md
  memory:
    - memory/code-review-checklist.md
```

**Usage:**

```bash
# Use workspace defaults
adp run codex --workspace game-a

# Use specific profile
adp run codex --workspace game-a --profile architect
adp run codex --workspace game-a --profile reviewer
```

**Decision Guide:**

Use workspace defaults when:
- Configuration applies to all agent runs
- Team needs consistent baseline
- Single workflow, no role variations

Use profiles when:
- Different roles need different configs
- Same agent, different contexts (review vs implementation)
- Experimenting with variations without changing baseline

**See also:**
- [examples/basic-workspace](../examples/basic-workspace/) - profile examples
- [Q2: What is a workspace?](#q2-what-is-a-workspace)

---

## Team Collaboration

### Q12: How do I share workspace configurations with my team?

**Short Answer:**

Commit example workspace configs to your project repo (e.g., `docs/adp-workspace-example/`) or use `examples/basic-workspace` as a template. Never commit `$ADP_HOME` itself—it contains local state and session logs.

**Three Sharing Methods:**

**Option 1: Example in Project Repo (Recommended)**

Commit workspace config template to your project:

```bash
# Project structure
my-project/
├── docs/
│   └── adp-workspace/
│       ├── workspace.yaml
│       ├── prompts/
│       │   └── base-prompt.md
│       ├── profiles/
│       │   ├── architect.yaml
│       │   └── engineer.yaml
│       └── memory/
│           └── conventions.md
└── README.md

# Team member setup
cd my-project
adp workspace add my-project $PWD
# Copy configs from docs/adp-workspace/ to ~/.adp/workspaces/my-project/
cp -r docs/adp-workspace/* ~/.adp/workspaces/my-project/
```

**Option 2: Shared Template Repository**

Maintain workspace templates in a separate repo:

```bash
# Template repo
workspace-templates/
├── golang-service/
│   ├── workspace.yaml
│   └── prompts/
└── react-app/
    ├── workspace.yaml
    └── prompts/

# Team member clones and copies
git clone https://git.example.com/team/workspace-templates
adp workspace add my-project /srv/my-project
cp -r workspace-templates/golang-service/* ~/.adp/workspaces/my-project/
```

**Option 3: Use ADP's Basic Workspace Example**

Copy from ADP's built-in example:

```bash
# Find ADP example workspace
ls -la /path/to/adp/examples/basic-workspace/

# Copy to your workspace
adp workspace add my-project /srv/my-project
cp -r /path/to/adp/examples/basic-workspace/* ~/.adp/workspaces/my-project/

# Edit project root
vim ~/.adp/workspaces/my-project/workspace.yaml
# Update: project.root: /srv/my-project
```

**What to Share vs Not Share:**

✅ **Safe to commit:**
- workspace.yaml (template with placeholder paths)
- prompts/ (instructions for agents)
- profiles/ (role-specific configs)
- memory/ (shared project knowledge)
- mcp/servers.json (MCP server configs)

❌ **Never commit:**
- `$ADP_HOME/` entire directory
- planning/ (tasks.yaml, phases.yaml - local state)
- logs/events.jsonl (session history)
- Credentials, API keys, tokens
- Machine-specific absolute paths

**See also:**
- [examples/basic-workspace](../examples/basic-workspace/)
- [Q13: Should I commit .adp/ to Git?](#q13-should-i-commit-adp-or-adp_home-to-git)

---

### Q13: Should I commit `.adp/` or `$ADP_HOME` to Git?

**Short Answer:**

**No**, never commit `$ADP_HOME` (default `~/.adp`)—it contains local state, task ownership, and session logs. **Yes**, commit example workspace configs in your project's `docs/` or `examples/` directory.

**What $ADP_HOME Contains:**

```
~/.adp/
├── workspaces/
│   ├── game-a/
│   │   ├── workspace.yaml       # ✅ Config (share as template)
│   │   ├── prompts/             # ✅ Instructions (shareable)
│   │   ├── profiles/            # ✅ Role configs (shareable)
│   │   └── planning/            # ❌ Local state (don't share)
│   │       ├── tasks.yaml       # Task ownership - machine-specific
│   │       ├── phases.yaml      # Phase state - local
│   │       └── progress.jsonl   # Execution log - local
└── logs/
    └── events.jsonl             # ❌ Session history (local only)
```

**Recommended .gitignore:**

```gitignore
# In project root .gitignore

# ADP local state - never commit
.adp/
$ADP_HOME/
**/planning/tasks.yaml
**/planning/phases.yaml
**/planning/progress.jsonl
**/logs/events.jsonl

# But DO commit example configs
!docs/adp-workspace/
!examples/adp-workspace/
```

**What Survives Restart:**

ADP state under `$ADP_HOME` is durable:
- ✅ Workspace registry (workspace list, configurations)
- ✅ Task state (ownership, status, leases)
- ✅ Phase state (current phase, acceptance records)
- ✅ Session history (events, runtime logs)

But these are **local to each machine**—share configs as templates, not as committed state.

**Recommended Team Workflow:**

1. Commit workspace config templates to `docs/adp-workspace/`
2. Add setup instructions to project README
3. Each team member copies template to their local `$ADP_HOME`
4. Task state stays local (coordinate through lease management)

**See also:**
- [Q12: How do I share workspace configurations?](#q12-how-do-i-share-workspace-configurations-with-my-team)
- [Q14: How do multiple operators coordinate on shared tasks?](#q14-how-do-multiple-operators-coordinate-on-shared-tasks)

---

### Q14: How do multiple operators coordinate on shared tasks?

**Short Answer:**

Operators coordinate through lease-based ownership: claim tasks with `adp tasks take --owner <owner> --lease <duration>`, preview available work with `adp tasks next`, detect expired leases with `adp tasks stale`, and extend ownership with `adp tasks renew`.

**Coordination Workflow:**

**1. Preview Available Work**

```bash
# See what's on the board
adp tasks next --workspace game-a

# JSON for machine parsing
adp tasks next --workspace game-a --format json
```

**2. Claim Task with Lease**

```bash
# Atomic claim
adp tasks take --workspace game-a --owner alice --lease 2h
# task-20260614-0001 taken

# Or claim + launch in one command
adp run codex --workspace game-a --take --owner alice --lease 2h
```

**3. Maintain Ownership for Long-Running Work**

```bash
# Extend lease before it expires
adp tasks renew --workspace game-a task-20260614-0001 --owner alice --lease 2h
```

**4. Detect Stale Ownership**

```bash
# Find expired leases
adp tasks stale --workspace game-a
# task-20260614-0002: owner bob, expired 30m ago

# Reclaim expired task
adp tasks take --workspace game-a --owner alice --lease 2h
```

**Ownership Rules:**

- **Pending tasks**: Anyone can claim (first come, first served)
- **In-progress with owner**: Only owner can renew or release
- **In-progress, lease expired**: Anyone can reclaim via `adp tasks take`
- **Completed tasks**: Immutable (use for audit trail)

**Conflict Resolution:**

ADP uses "last writer wins within lease boundaries":
- Owner has exclusive control during lease period
- After lease expires, task becomes available for reclaim
- No distributed locking (coordinate through local `$ADP_HOME` state)

**Example Multi-Operator Flow:**

```bash
# Operator Alice
adp tasks add --workspace game-a "Fix auth bug"
adp run codex --workspace game-a --take --owner alice --lease 4h
# Works for 2 hours, needs more time
adp tasks renew task-20260614-0001 --owner alice --lease 2h

# Operator Bob (different machine)
adp tasks next --workspace game-a
# Sees: No available tasks (Alice owns task-20260614-0001)

# 6 hours later, Alice's lease expired
adp tasks stale --workspace game-a
# task-20260614-0001: owner alice, expired 1h ago

# Bob claims the stale task
adp tasks take --workspace game-a --owner bob --lease 2h
# task-20260614-0001 taken by bob
```

**Common Pitfalls:**
- ⚠️ Leases don't auto-renew; set reminders or use longer durations
- ⚠️ No cross-machine state sync; coordinate through Git commits or external trackers
- ⚠️ `adp tasks next` shows only your local view; other operators' state may differ

**See also:**
- [Task Management Guide: Lease Management](task-management.md)
- [Q15: How do I hand off work between agents or operators?](#q15-how-do-i-hand-off-work-between-agents-or-operators)

---

### Q15: How do I hand off work between agents or operators?

**Short Answer:**

Hand off work by releasing ownership (`adp tasks release`), letting leases expire, or using `adp sessions resume-plan` for cross-tool handoffs. The next operator claims the task and reviews session history for context.

**Three Handoff Methods:**

**Method 1: Explicit Release (Immediate)**

```bash
# Operator Alice releases task
adp tasks release --workspace game-a task-20260614-0001 --owner alice

# Operator Bob claims it
adp tasks take --workspace game-a --owner bob --lease 2h
# OR
adp run claude --workspace game-a --take --owner bob --lease 2h
```

**Method 2: Lease Expiration (Delayed)**

```bash
# Operator Alice's lease expires naturally (no action needed)
# After 2 hours, task becomes stale

# Operator Bob checks stale tasks
adp tasks stale --workspace game-a
# task-20260614-0001: owner alice, expired 15m ago

# Bob claims expired task
adp tasks take --workspace game-a --owner bob --lease 2h
```

**Method 3: Cross-Tool Resume (Same Task, Different Agent)**

```bash
# Operator Alice worked with Codex
adp run codex --workspace game-a --task task-20260614-0001 --owner alice --lease 2h
# Session: session-20260614T120000-abc123

# Alice releases task
adp tasks release --workspace game-a task-20260614-0001 --owner alice

# Operator Bob wants to continue with Claude
adp sessions resume-plan session-20260614T120000-abc123 \
  --agent claude --owner bob --lease 2h

# Suggested command (copy and run):
adp run claude --workspace game-a --task task-20260614-0001 \
  --owner bob --lease 2h
```

**Handoff Evidence Trail:**

Before claiming, review context:

```bash
# Check task details
adp tasks show task-20260614-0001

# Review session history
adp sessions list --workspace game-a --task task-20260614-0001

# Get progress snapshot
adp progress report --workspace game-a --format json
```

**Example Complete Handoff:**

```bash
# === Operator Alice (implementation) ===
adp run codex --workspace game-a --take --owner alice --lease 2h
# Works on task-20260614-0001
# Session: session-20260614T120000-abc123

# Alice completes implementation, releases for review
adp tasks release --workspace game-a task-20260614-0001 --owner alice

# === Operator Bob (reviewer) ===
# Preview available work
adp tasks next --workspace game-a
# task-20260614-0001: "Fix auth bug", status: pending

# Review context
adp sessions show session-20260614T120000-abc123
adp tasks show task-20260614-0001

# Claim and review with Claude
adp run claude --workspace game-a --task task-20260614-0001 \
  --owner bob --lease 1h --profile reviewer

# Complete review
adp tasks done task-20260614-0001
```

**Cross-Agent Context Transfer:**

ADP provides handoff context through:
- Task description and status
- Session event history
- Progress reports
- Phase state (if using phase gates)

But does NOT transfer:
- Provider-native conversation history
- Codex/Claude internal state
- Interactive session handles

**See also:**
- [Session Resume Planning](session-restore.md)
- [Q20: How does session restore and resume work?](#q20-how-does-session-restore-and-resume-work)
- [Q14: How do multiple operators coordinate?](#q14-how-do-multiple-operators-coordinate-on-shared-tasks)

---

## Integration Scenarios

### Q16: How do I use ADP in CI/CD pipelines?

**Short Answer:**

Set isolated `$ADP_HOME`, create workspace programmatically, use `adp run --take` for task pickup, and always clean runtimes with `adp runtime prune`. Avoid persisting local state between CI runs.

**CI/CD Best Practices:**

**1. Use Isolated Temporary State**

```bash
# Don't persist between runs
export ADP_HOME="${RUNNER_TEMP}/adp-home"
export ADP_RUNTIME_DIR="${RUNNER_TEMP}/adp-runtime"
```

**2. Create Workspace Programmatically**

```bash
# Initialize fresh ADP state
adp init

# Create workspace
adp workspace add ci-workspace "${GITHUB_WORKSPACE}"

# Copy configs from repo
cp -r .github/adp-workspace/* "${ADP_HOME}/workspaces/ci-workspace/"
```

**3. Run Agent with Task Tracking**

```bash
# Create task
TASK_ID=$(adp tasks add --workspace ci-workspace \
  --priority high "CI: Automated code review" | \
  grep -o 'task-[^ ]*')

# Run agent
adp run codex --workspace ci-workspace \
  --task "${TASK_ID}" \
  --owner ci-bot \
  --lease 30m
```

**4. Always Clean Up**

```bash
# Clean all runtimes
adp runtime prune --older-than 0s

# Verify cleanup
find "${ADP_RUNTIME_DIR}" -type d -name "ci-workspace-*"
```

**Example GitHub Actions Workflow:**

```yaml
name: ADP Code Review

on: [pull_request]

jobs:
  review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup ADP
        run: |
          # Install ADP
          curl -L https://github.com/example/adp/releases/latest/adp -o /usr/local/bin/adp
          chmod +x /usr/local/bin/adp
          
          # Set isolated state
          echo "ADP_HOME=${RUNNER_TEMP}/adp-home" >> $GITHUB_ENV
          echo "ADP_RUNTIME_DIR=${RUNNER_TEMP}/adp-runtime" >> $GITHUB_ENV

      - name: Initialize Workspace
        run: |
          adp init
          adp workspace add ci-workspace "${GITHUB_WORKSPACE}"
          cp -r .github/adp-workspace/* "${ADP_HOME}/workspaces/ci-workspace/"

      - name: Run Review
        run: |
          adp run codex --workspace ci-workspace \
            --take --owner ci-bot --lease 30m \
            -- --review-pr ${{ github.event.pull_request.number }}

      - name: Cleanup
        if: always()
        run: adp runtime prune --older-than 0s
```

**Key Considerations:**

- **Authentication**: Ensure Codex/Claude CLI credentials are available in CI environment
- **Isolation**: Use separate `$ADP_HOME` per run (don't persist task state)
- **Cleanup**: Always prune runtimes, even on failure (use `if: always()`)
- **Timeouts**: Set realistic lease durations for CI environment
- **Audit**: Log session IDs and task IDs for troubleshooting

**See also:**
- [Q17: Can I run ADP in Docker containers?](#q17-can-i-run-adp-in-docker-containers)
- [Real Agent Compatibility](real-agent-compatibility.md)

---

### Q17: Can I run ADP in Docker containers?

**Short Answer:**

Yes. Mount your project as a volume, set `$ADP_HOME` and `$ADP_RUNTIME_DIR` inside the container, and ensure Codex/Claude CLI is available in the container image.

**Example Dockerfile:**

```dockerfile
FROM ubuntu:22.04

# Install dependencies
RUN apt-get update && apt-get install -y \
    curl \
    git \
    && rm -rf /var/lib/apt/lists/*

# Install ADP
RUN curl -L https://github.com/example/adp/releases/latest/adp -o /usr/local/bin/adp \
    && chmod +x /usr/local/bin/adp

# Install Codex CLI (example - adjust for your provider)
RUN curl -L https://codex-cli.example.com/install.sh | sh

# Set ADP paths
ENV ADP_HOME=/workspace/.adp
ENV ADP_RUNTIME_DIR=/tmp/adp-runtime

# Working directory
WORKDIR /workspace

CMD ["/bin/bash"]
```

**Run Container with Project Mount:**

```bash
# Build image
docker build -t adp-agent .

# Run with project mounted
docker run --rm -it \
  -v /srv/my-project:/workspace/project:ro \
  -v ~/.codex-credentials:/root/.codex-credentials:ro \
  -e ADP_HOME=/workspace/.adp \
  -e ADP_RUNTIME_DIR=/tmp/adp-runtime \
  adp-agent bash

# Inside container
adp init
adp workspace add my-project /workspace/project
adp run codex --workspace my-project
```

**Docker Compose Example:**

```yaml
version: '3.8'

services:
  adp-agent:
    build: .
    volumes:
      - ./project:/workspace/project:ro
      - ~/.codex-credentials:/root/.codex-credentials:ro
      - adp-home:/workspace/.adp
      - adp-runtime:/tmp/adp-runtime
    environment:
      - ADP_HOME=/workspace/.adp
      - ADP_RUNTIME_DIR=/tmp/adp-runtime
    command: |
      bash -c "
        adp init
        adp workspace add my-project /workspace/project
        adp run codex --workspace my-project --take --owner docker-bot --lease 1h
      "

volumes:
  adp-home:
  adp-runtime:
```

**Important Considerations:**

**Authentication:**
- Mount provider credentials as read-only volumes
- Or use environment variables (less secure)
- Ensure credentials work in container environment

**File Permissions:**
- Project mounted as `:ro` (read-only) prevents accidental modification
- ADP state volumes need write access
- Runtime directory needs write access

**Networking:**
- Ensure container can reach provider APIs (Codex/Claude endpoints)
- Configure proxy if needed

**Cleanup:**
- Prune runtimes: `docker exec <container> adp runtime prune --older-than 1h`
- Remove volumes: `docker volume rm adp-home adp-runtime`

**See also:**
- [Q16: How do I use ADP in CI/CD?](#q16-how-do-i-use-adp-in-cicd-pipelines)
- [Real Agent Compatibility](real-agent-compatibility.md)

---

### Q18: How do I integrate ADP with existing tools (IDEs, task trackers)?

**Short Answer:**

ADP provides JSON output modes (`--format json`) for machine parsing, shell integration (`adp shell-hook`), completion (`adp completion`), and file-based state under `$ADP_HOME` for external tool integration.

**Integration Points:**

**1. JSON Output for Parsing**

All inspection commands support `--format json`:

```bash
# Task list
adp tasks list --workspace game-a --format json | jq '.tasks[] | select(.status == "pending")'

# Progress report
adp progress report --workspace game-a --format json

# Session history
adp sessions list --workspace game-a --format json

# Diagnostics
adp doctor game-a --format json
```

**2. Shell Integration**

```bash
# Generate shell function for workspace switching
adp shell-hook --shell bash >> ~/.bashrc

# Usage: enter workspace environment
adp_env game-a

# Generates: export ADP_WORKSPACE=game-a; cd $ADP_RUNTIME_ROOT
```

**3. Shell Completion**

```bash
# Bash completion
adp completion --shell bash > /etc/bash_completion.d/adp

# Zsh completion
adp completion --shell zsh > /usr/share/zsh/site-functions/_adp

# Completes: workspaces, tasks, sessions, agents, profiles
```

**4. File-Based State Access**

External tools can read ADP state directly:

```bash
# Workspace config
cat ~/.adp/workspaces/game-a/workspace.yaml

# Task state
cat ~/.adp/workspaces/game-a/planning/tasks.yaml

# Event log
tail -f ~/.adp/logs/events.jsonl | jq .
```

**Example: VS Code Task Integration**

```json
// .vscode/tasks.json
{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "ADP: Run Codex",
      "type": "shell",
      "command": "adp run codex --workspace game-a --take --owner vscode --lease 2h",
      "problemMatcher": []
    },
    {
      "label": "ADP: Show Tasks",
      "type": "shell",
      "command": "adp tasks list --workspace game-a --format json | jq '.tasks'",
      "problemMatcher": []
    }
  ]
}
```

**Example: External Task Tracker Sync**

```bash
#!/bin/bash
# Sync ADP tasks to external tracker (e.g., Jira, Linear)

WORKSPACE="game-a"
TASKS=$(adp tasks list --workspace "$WORKSPACE" --format json)

echo "$TASKS" | jq -r '.tasks[] | select(.status == "completed")' | while read -r task; do
  TASK_ID=$(echo "$task" | jq -r '.id')
  TITLE=$(echo "$task" | jq -r '.title')
  
  # Sync to external tracker
  curl -X POST https://tracker.example.com/api/tasks \
    -H "Content-Type: application/json" \
    -d "{\"title\": \"$TITLE\", \"external_id\": \"$TASK_ID\"}"
done
```

**Example: IDE Extension (Conceptual)**

```typescript
// VS Code extension reading ADP state
import * as fs from 'fs';
import * as yaml from 'js-yaml';

function getWorkspaces(): string[] {
  const adpHome = process.env.ADP_HOME || `${process.env.HOME}/.adp`;
  const workspacesDir = `${adpHome}/workspaces`;
  return fs.readdirSync(workspacesDir);
}

function getTasks(workspace: string): Task[] {
  const tasksPath = `${process.env.ADP_HOME}/workspaces/${workspace}/planning/tasks.yaml`;
  const tasksYaml = fs.readFileSync(tasksPath, 'utf8');
  return yaml.load(tasksYaml).tasks;
}
```

**See also:**
- [README: Completion](../README.md#runtime-model)
- [Q19: How does ADP work with Git workflows?](#q19-how-does-adp-work-with-git-workflows)

---

### Q19: How does ADP work with Git workflows?

**Short Answer:**

ADP does NOT wrap Git commands or auto-commit/push. Agents see project files via symlinks and can run `git -C $ADP_PROJECT_ROOT` commands. Phase gates can record commit/push evidence but don't execute Git operations.

**Git Interaction Model:**

**What ADP Does:**
- Excludes `.git` metadata from runtime overlays
- Neutralizes Git environment variables (`GIT_DIR`, `GIT_WORK_TREE`, etc.)
- Provides `$ADP_PROJECT_ROOT` for agents to run Git commands
- Records commit/push evidence in phase gates (read-only tracking)
- Runs read-only Git diagnostics in `adp workspace doctor`

**What ADP Does NOT Do:**
- Wrap `git commit`, `git push`, `git checkout`
- Automatically commit or push on task completion
- Intercept or modify Git commands
- Resume provider-native conversations across Git branches

**Running Git from Agents:**

Agents working in runtime overlays can run Git against the real project:

```bash
# Inside agent runtime (e.g., AGENTS.md instructions)
# Run Git commands against real project root
git -C $ADP_PROJECT_ROOT status
git -C $ADP_PROJECT_ROOT diff
git -C $ADP_PROJECT_ROOT add .
git -C $ADP_PROJECT_ROOT commit -m "Fix auth bug"
git -C $ADP_PROJECT_ROOT push
```

**Phase Gate Git Evidence:**

Phase commands record Git operations without executing them:

```bash
# Phase workflow
adp phase accept --workspace game-a  # Mark phase as reviewed
adp phase commit --workspace game-a  # Record commit evidence (you run git commit separately)
adp phase push --workspace game-a    # Record push evidence (you run git push separately)

# ADP tracks that these steps happened, but doesn't run git itself
```

**Recommended Git Workflow with ADP:**

**Option 1: Manual Git (Full Control)**

```bash
# Work on task
adp run codex --workspace game-a --task task-001

# Agent modifies files in runtime overlay (symlinks point to real project)
# Changes appear in real project root

# Manually commit
cd /srv/my-project  # Real project root
git status
git add .
git commit -m "Fix: Auth bug in login flow"
git push

# Record evidence
adp phase commit --workspace game-a
adp phase push --workspace game-a
adp tasks done task-001
```

**Option 2: Agent-Driven Git (Agent Autonomy)**

```bash
# Give agent Git instructions in AGENTS.md
adp run codex --workspace game-a --task task-001 -- \
  "After fixing the bug, commit changes with message 'Fix: Auth bug'"

# Agent runs: git -C $ADP_PROJECT_ROOT commit -m "Fix: Auth bug"
# Agent runs: git -C $ADP_PROJECT_ROOT push

# You record evidence
adp phase commit --workspace game-a
adp phase push --workspace game-a
adp tasks done task-001
```

**Git Safety Considerations:**

- Runtime overlays see project files via symlinks, so file modifications affect real project
- Agents can run Git commands if given permission
- ADP doesn't prevent destructive Git operations (git reset --hard, git clean -f)
- Use `adp workspace doctor` to check Git status before agent runs

**See also:**
- [README: Runtime Model - Git Metadata](../README.md#runtime-model)
- [Task Management: Phase Gates](task-management.md)

---

## Advanced Topics

### Q20: How does session restore and resume work?

**Short Answer:**

`adp sessions restore-plan` suggests rerunning the same session with the same agent. `adp sessions resume-plan` suggests continuing ADP work context, possibly with a different agent. Neither resumes provider-native conversations.

**Two Resume Commands:**

**1. Restore Plan (Same-Tool Rerun)**

Suggests rerunning the same session with the same agent:

```bash
# View session
adp sessions show session-20260614T120000-abc123

# Get rerun suggestion
adp sessions restore-plan session-20260614T120000-abc123

# Output: Suggested command
adp run codex --workspace game-a --task task-001 \
  --profile architect --keep-runtime
```

**2. Resume Plan (Cross-Tool Handoff)**

Suggests continuing work with a different agent:

```bash
# Original session with Codex
adp run codex --workspace game-a --task task-001 --owner alice

# Get cross-tool resume plan
adp sessions resume-plan session-20260614T120000-abc123 \
  --agent claude --owner bob --lease 2h

# Output: Suggested command
adp run claude --workspace game-a --task task-001 \
  --owner bob --lease 2h
# Note: Profile and agent-specific args omitted (different tool)
```

**What Gets Transferred:**

✅ **ADP Work Context (Transfers)**
- Workspace and project root identity
- Task ID and task snapshot
- Phase state and gates
- Session event history
- Owner and lease guidance

❌ **Provider State (Does NOT Transfer)**
- Codex/Claude conversation history
- Provider-native task panels
- Interactive session handles
- Provider-specific internal state

**Resume is ADP Handoff, Not Provider Resume:**

```
Session A (Codex)  →  Session B (Claude)
     ↓                      ↓
Same ADP task context   Fresh provider conversation
Same workspace config   Different agent instructions
```

**Example Cross-Tool Handoff:**

```bash
# Step 1: Alice works with Codex
adp run codex --workspace game-a --task task-001 --owner alice --lease 2h
# Session: session-20260614T120000-abc123

# Step 2: Alice completes implementation, releases task
adp tasks release --workspace game-a task-001 --owner alice

# Step 3: Bob reviews context
adp sessions show session-20260614T120000-abc123
adp tasks show task-001
adp progress report --workspace game-a

# Step 4: Bob gets resume plan for Claude
adp sessions resume-plan session-20260614T120000-abc123 \
  --agent claude --owner bob --lease 1h

# Step 5: Bob runs suggested command
adp run claude --workspace game-a --task task-001 \
  --owner bob --lease 1h --profile reviewer

# Claude starts fresh conversation with ADP task context
```

**Common Pitfalls:**
- ⚠️ Resume does NOT continue Codex/Claude conversations
- ⚠️ Cross-tool resume starts fresh provider session
- ⚠️ Restore/resume commands are read-only; they don't execute suggested commands

**See also:**
- [Session Restore Documentation](session-restore.md)
- [Q15: How do I hand off work between agents?](#q15-how-do-i-hand-off-work-between-agents-or-operators)

---

### Q21: What are the performance implications of runtime overlays?

**Short Answer:**

Runtime overlays use symlinks, so overhead is negligible—typically <100ms to build for projects with <10k files. Symlinks don't copy data, so disk space and build times are unaffected.

**Performance Breakdown:**

**Runtime Overlay Creation:**
- **Time**: Typically <100ms for projects with <10k files
- **Mechanism**: Creates symlinks, not copies
- **Disk I/O**: Minimal (only writes generated config files)

**Build Time Impact:**
- **Zero impact**: Symlinks point to real files
- Tools (go build, npm install, cargo build) see original files
- No data duplication

**Disk Space:**
- **Generated files**: ~10-50KB (AGENTS.md, CLAUDE.md, adapter configs)
- **Symlinks**: ~1KB per symlink
- **Total**: Usually <1MB per runtime overlay
- **Kept runtimes**: Accumulate if not pruned

**Runtime Performance:**
- **File access**: No overhead (symlinks are transparent to programs)
- **Agent execution**: Same as running in real project
- **Cleanup**: ~10-50ms to remove overlay directory

**Benchmarks (Example Project):**

```
Project: 5,000 files, 500MB total size
Runtime overlay creation: 80ms
Generated files: 45KB
Symlink count: 5,003
Total overlay size: ~50KB (symlinks don't duplicate data)
Cleanup time: 30ms
```

**Large Project Considerations:**

For projects with >50k files:
- Overlay creation: ~500ms-1s
- Consider `--keep-runtime` for interactive debugging
- Symlink count doesn't affect runtime performance

**Cleanup Cost:**

```bash
# Check kept runtime disk usage
du -sh /tmp/adp-runtime/*

# Prune old runtimes
adp runtime prune --older-than 24h --dry-run
# Shows: 5 runtimes, 250MB total (mostly symlinks)

# Actual data: ~200KB generated files
```

**Optimization Tips:**
- Default cleanup (no `--keep-runtime`) prevents accumulation
- Prune kept runtimes periodically: `adp runtime prune --older-than 24h`
- No performance tuning needed for most projects

**See also:**
- [Q3: What is the runtime overlay?](#q3-what-is-the-runtime-overlay)
- [Q10: Should I use --keep-runtime?](#q10-should-i-use---keep-runtime-or-let-adp-clean-up)

---

### Q22: Can I customize ADP's behavior with hooks or plugins?

**Short Answer:**

Currently, no plugin system or hooks exist. Extensibility comes through workspace configs, custom profiles, MCP server integration, and shell wrappers around `adp run` for pre/post logic.

**Current Extensibility Options:**

**1. Workspace Configuration**

Customize per-project behavior through workspace.yaml:

```yaml
# workspace.yaml
project:
  root: /srv/my-project

codex:
  command: /custom/path/to/codex-wrapper.sh
  base_prompt: prompts/custom-prompt.md
  memory:
    - memory/custom-context.md
  mcp:
    servers: mcp/custom-servers.json
```

**2. Custom Profiles**

Create role-specific configurations:

```bash
# profiles/custom-role.yaml
codex:
  base_prompt: prompts/custom-role-prompt.md
  memory:
    - memory/role-specific-context.md
```

**3. MCP Server Integration**

Extend agent capabilities through Model Context Protocol:

```json
// mcp/servers.json
{
  "mcpServers": {
    "custom-tool": {
      "command": "node",
      "args": ["/path/to/custom-mcp-server.js"]
    }
  }
}
```

**4. Shell Wrapper Pattern**

Add pre/post logic around `adp run`:

```bash
#!/bin/bash
# custom-adp-run.sh

# Pre-run hook
echo "Starting agent run at $(date)"
adp tasks show "${TASK_ID}"

# Run ADP
adp run codex --workspace game-a --task "${TASK_ID}" "$@"
EXIT_CODE=$?

# Post-run hook
if [ $EXIT_CODE -eq 0 ]; then
  echo "Success! Sending notification..."
  curl -X POST https://slack.example.com/webhook \
    -d "{\"text\": \"Task ${TASK_ID} completed\"}"
fi

exit $EXIT_CODE
```

**What's NOT Available:**

❌ Plugin system for extending ADP core
❌ Git hooks integration (pre-commit, post-push)
❌ Event-driven triggers (on task complete, on phase accept)
❌ Runtime overlay customization hooks
❌ Custom task state machines

**Workarounds:**

For automation needs:
1. **Wrap `adp` commands** in scripts for pre/post logic
2. **Poll state** via `adp tasks list --format json` and react to changes
3. **Parse event logs** (`~/.adp/logs/events.jsonl`) for audit trails
4. **Extend agents** through MCP servers and custom prompts

**Future Considerations:**

ADP maintains terminal-first simplicity. Plugin systems add complexity. If you have extensibility needs, consider:
- Opening a GitHub issue describing the use case
- Contributing shell script patterns to the community
- Building external tools that read ADP state files

**Example External Tool:**

```python
#!/usr/bin/env python3
# adp-notifier.py - Watch for completed tasks and notify

import json
import time
import subprocess

def get_tasks(workspace):
    result = subprocess.run(
        ['adp', 'tasks', 'list', '--workspace', workspace, '--format', 'json'],
        capture_output=True, text=True
    )
    return json.loads(result.stdout)['tasks']

def notify(task):
    print(f"Task completed: {task['title']}")
    # Send to Slack, email, etc.

def main():
    workspace = 'game-a'
    seen_completed = set()
    
    while True:
        tasks = get_tasks(workspace)
        for task in tasks:
            if task['status'] == 'completed' and task['id'] not in seen_completed:
                notify(task)
                seen_completed.add(task['id'])
        time.sleep(30)

if __name__ == '__main__':
    main()
```

**See also:**
- [Q18: How do I integrate ADP with existing tools?](#q18-how-do-i-integrate-adp-with-existing-tools-ides-task-trackers)
- [examples/basic-workspace](../examples/basic-workspace/) - configuration patterns

---

## Back to Top

- [Core Concepts](#core-concepts)
- [Usage Decisions](#usage-decisions)
- [Team Collaboration](#team-collaboration)
- [Integration Scenarios](#integration-scenarios)
- [Advanced Topics](#advanced-topics)

---

**Questions or feedback?** See [Troubleshooting Guide](troubleshooting.md) for error resolution or open an issue on the project repository.

