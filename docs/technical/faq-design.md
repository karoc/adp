# FAQ Design Document

## 1. Executive Summary

This document defines the question classification system, content structure, and answer patterns for the ADP FAQ documentation. The FAQ serves as a conceptual bridge between README and technical guides, focusing on "why" and "when" questions rather than "how to fix" errors.

**Key Design Principles:**
- **Conceptual Focus**: Explain core concepts, design decisions, and usage scenarios
- **Decision Support**: Help users choose between alternatives (workspace vs direct run, task vs direct agent launch)
- **Complement, Not Duplicate**: FAQ covers concepts; Troubleshooting covers errors; guides cover procedures
- **Progressive Disclosure**: Short answer first, then details, then links to authoritative docs
- **Bilingual Consistency**: English and Simplified Chinese versions must stay synchronized

---

## 2. FAQ vs Troubleshooting Positioning

### Troubleshooting Guide Focus
- Error messages and symptoms ("workspace not found", "failed to build runtime")
- Diagnostic procedures ("Check if binary exists", "Verify directory permissions")
- Concrete fixes and recovery steps
- Command output examples and expected results

### FAQ Focus
- Conceptual understanding ("What is a workspace?", "Why use runtime overlays?")
- Usage decisions ("When should I use ADP vs running agent CLI directly?")
- Architectural rationale ("Why does ADP keep config outside project root?")
- Team workflows ("How do I share workspace configs?", "Should I commit .adp/ to Git?")
- Integration scenarios ("How to use ADP in CI/CD?", "Can I run ADP in Docker?")

**Clear Boundary Example:**
- Troubleshooting: "Error: workspace not found" → diagnostic steps → solution
- FAQ: "What's the difference between workspace and project?" → concept explanation → when to use each

---

## 3. Question Classification System

### 3.1 Category 1: Core Concepts (6 questions)

**Purpose**: Build mental model of ADP architecture

**Questions:**
1. **What is ADP and why would I use it?**
   - Core value proposition: clean project roots, reusable agent config
   - Terminal-first, local-first positioning
   - When NOT to use ADP (single-project, one-off runs)

2. **What is a workspace?**
   - Workspace = named agent configuration for a project
   - Contains: profiles, prompts, memory, MCP settings
   - Analogy: workspace is to agent configuration what Git remote is to repository URL

3. **What is a runtime overlay?**
   - Temporary symlink-based view of project + generated files
   - Why: keeps AGENTS.md, CLAUDE.md, .codex/, .claude/ out of real project
   - Lifecycle: created on `adp run`, cleaned on exit (unless `--keep-runtime`)

4. **What files does ADP create and where?**
   - `$ADP_HOME`: workspace configs, task state, session logs (durable)
   - `$ADP_RUNTIME_DIR`: temporary runtime overlays (ephemeral)
   - Real project root: NEVER modified by ADP
   - Table showing file types and locations

5. **How does ADP relate to Codex and Claude?**
   - ADP is a runtime environment manager, not a replacement
   - Codex/Claude are the agents; ADP provides consistent configuration
   - Analogy: ADP is to agent CLIs what Docker is to application processes

6. **What are tasks and phases?**
   - Tasks: individual work items with ownership and leases
   - Phases: stage gates for release discipline (planning → acceptance → commit → push)
   - Optional: can use `adp run` without task management
   - Cross-reference: [task-management.md](../task-management.md)

---

### 3.2 Category 2: Usage Decisions (5 questions)

**Purpose**: Help users choose the right workflow

**Questions:**
7. **When should I use ADP instead of running `codex` or `claude` directly?**
   - Use ADP when: multiple projects, reusable configs, team consistency, task tracking
   - Direct CLI when: one-off exploration, single project, interactive experimentation
   - Decision matrix with scenarios

8. **When should I use tasks vs direct `adp run`?**
   - Tasks when: multi-agent coordination, work handoff, lease management, audit trail
   - Direct run when: exploratory work, one-shot execution, no ownership tracking
   - Example scenarios for each

9. **How do I choose between `adp run --task` and `adp run --take`?**
   - `--task <id>`: pre-assigned task, explicit targeting
   - `--take --owner --lease`: atomic board pickup, first available work
   - `adp tasks take` alone: claim without launching agent
   - Flow chart showing decision path

10. **Should I use `--keep-runtime` or let ADP clean up?**
    - Keep when: debugging runtime issues, manual inspection needed
    - Clean when: normal workflow, CI/CD, automated runs
    - How to clean kept runtimes: `adp runtime prune`

11. **When should I use profiles vs workspace defaults?**
    - Profiles when: role-specific configs (architect, engineer, reviewer)
    - Defaults when: consistent team baseline, single workflow
    - Profile inheritance: profile overrides workspace overrides adapter defaults

---

### 3.3 Category 3: Team Collaboration (4 questions)

**Purpose**: Enable effective team workflows

**Questions:**
12. **How do I share workspace configurations with my team?**
    - Option 1: Commit example to project repo (e.g., `docs/adp-workspace-example/`)
    - Option 2: Shared template repository
    - Option 3: Copy from `examples/basic-workspace`
    - What to share: workspace.yaml, prompts, memory; NOT: credentials, $ADP_HOME state

13. **Should I commit `.adp/` or `$ADP_HOME` to Git?**
    - **NO**: $ADP_HOME contains local state, task ownership, session logs
    - **YES**: example workspace configs in `docs/` or `examples/`
    - Gitignore recommendations
    - What survives restart: workspace registry, task state, session history

14. **How do multiple operators coordinate on shared tasks?**
    - Lease-based ownership with configurable durations
    - `adp tasks next`: preview available work
    - `adp tasks stale`: detect expired leases
    - `adp tasks renew`: extend ownership for long-running work
    - Conflict resolution: last writer wins within lease boundaries

15. **How do I hand off work between agents or operators?**
    - Method 1: `adp tasks release` + another operator `adp tasks take`
    - Method 2: Let lease expire, next operator `adp tasks take`
    - Method 3: `adp sessions resume-plan` for cross-tool handoff
    - Evidence trail: events, sessions, progress reports

---

### 3.4 Category 4: Integration Scenarios (4 questions)

**Purpose**: Enable ADP in diverse environments

**Questions:**
16. **How do I use ADP in CI/CD pipelines?**
    - Set `ADP_HOME` to isolated temporary directory
    - Create workspace programmatically with `adp workspace add`
    - Use `adp run --take --owner ci-bot --lease 30m`
    - Always clean runtimes: `adp runtime prune --older-than 0s`
    - Example: GitHub Actions workflow snippet

17. **Can I run ADP in Docker containers?**
    - Yes: mount project as volume, set `ADP_HOME` and `ADP_RUNTIME_DIR` inside container
    - Ensure Codex/Claude CLI is available in container
    - Example Dockerfile snippet
    - Caveat: provider authentication must work in container environment

18. **How do I integrate ADP with existing tools (IDEs, task trackers)?**
    - JSON output modes: `--format json` for machine parsing
    - Shell integration: `adp shell-hook` for terminal workflows
    - Completion: `adp completion` for bash/zsh
    - API surface: all state under `$ADP_HOME`, parseable YAML/JSONL files
    - Example: VS Code task integration

19. **How does ADP work with Git workflows?**
    - ADP does NOT wrap Git commands or auto-commit/push
    - Runtime overlays exclude `.git` metadata
    - Agents see project files via symlinks, can run `git -C $ADP_PROJECT_ROOT`
    - Phase gates can record commit/push evidence, but don't execute Git
    - Recommendation: keep Git workflow explicit and observable

---

### 3.5 Category 5: Advanced Topics (3 questions)

**Purpose**: Address power-user scenarios

**Questions:**
20. **How does session restore and resume work?**
    - `adp sessions restore-plan`: rerun same session with same agent
    - `adp sessions resume-plan`: continue ADP work context, possibly with different agent
    - Cross-tool handoff: Codex → Claude with shared task/phase context
    - Limitation: does NOT resume provider-native conversations
    - Cross-reference: [session-restore.md](../session-restore.md)

21. **What are the performance implications of runtime overlays?**
    - Symlink overhead: negligible for modern filesystems
    - Build time: no impact (symlinks point to real files)
    - Disk space: minimal (only generated files are real, rest is symlinks)
    - Cleanup cost: proportional to number of kept runtimes
    - Benchmark: runtime build typically <100ms for projects with <10k files

22. **Can I customize ADP's behavior with hooks or plugins?**
    - Current: No plugin system or hooks
    - Extensibility: workspace configs, custom profiles, MCP server integration
    - Workaround: shell wrappers around `adp run` for pre/post logic
    - Future: Open to proposals, but maintaining terminal-first simplicity

---

## 4. Answer Structure Template

Each FAQ answer follows this structure:

### Short Answer (1-2 sentences)
Direct, actionable response to the question. Readers should get the core answer immediately.

### Detailed Explanation (2-4 paragraphs)
- **Context**: Why this matters
- **Mechanics**: How it works
- **Rationale**: Design decisions behind it

### Example or Diagram (when helpful)
```bash
# Code examples
```
or
```
[Simple ASCII diagram]
```

### Common Pitfalls (if applicable)
- ⚠️ Mistake 1: Why it fails
- ⚠️ Mistake 2: How to avoid

### Related Topics
- **See also**: [link to guide], [link to related FAQ]
- **Troubleshooting**: Link to relevant error resolution if applicable

---

## 5. Cross-Reference Strategy

### Internal Links
- **Authoritative Docs**: FAQ never duplicates; always links to canonical source
  - Core concepts → README.md
  - Setup procedures → install.md, operator-onboarding.md
  - Task workflows → task-management.md
  - Session concepts → session-restore.md
  - Error resolution → troubleshooting.md

- **FAQ ↔ Troubleshooting Bridge**:
  - FAQ Q6 (tasks/phases) → Troubleshooting "task not found"
  - FAQ Q3 (runtime overlay) → Troubleshooting "failed to build runtime"
  - FAQ Q19 (Git workflows) → Troubleshooting "dangerous Git environment variables"

- **FAQ ↔ Guide Bridge**:
  - FAQ Q7 (when to use ADP) → operator-onboarding.md (first-run walkthrough)
  - FAQ Q16 (CI/CD) → examples/ci-integration/ (if created)
  - FAQ Q12 (team sharing) → examples/basic-workspace/

### External Links
- None initially; ADP is self-contained
- If needed: link to Codex/Claude official docs for provider-specific questions

---

## 6. Bilingual Implementation Strategy

### Consistency Requirements
- Every English FAQ entry must have matching Simplified Chinese entry
- Question numbers and order must stay synchronized
- Cross-references must use language-appropriate links (faq.md vs faq.zh-CN.md)

### Translation Guidelines
- **Technical Terms**: Keep command names, file paths, environment variables untranslated
  - Keep: `adp run`, `$ADP_HOME`, `workspace.yaml`
- **Conceptual Terms**: Translate consistently
  - Workspace → 工作空间
  - Runtime overlay → 运行时覆盖层
  - Task → 任务
  - Phase → 阶段
  - Lease → 租约

### Review Process
1. Write English version first (source of truth for structure)
2. Translate to Simplified Chinese
3. Validate cross-references work in both languages
4. Check examples render correctly in both contexts

---

## 7. Maintenance Plan

### When to Update FAQ
- **Add**: New core concept introduced (e.g., P-series adds new abstraction)
- **Update**: Command behavior changes, new options added
- **Remove**: Deprecated feature questions (keep redirect to migration guide)
- **Clarify**: User reports confusion pattern (track in GitHub issues)

### Quality Checks
- [ ] Every question has 1-2 sentence short answer
- [ ] Detailed explanation <500 words per question
- [ ] At least one cross-reference per question
- [ ] Code examples use copyable commands
- [ ] No duplication with Troubleshooting or guides
- [ ] Bilingual versions synchronized

### Validation
- Run through with new operator (does FAQ answer their questions?)
- Check cross-reference links (no broken internal links)
- Verify examples execute successfully
- Confirm JSON/text output examples match current CLI behavior

---

## 8. Implementation Checklist

### Phase 1: Foundation (P13 - FAQ Design)
- [x] Analyze existing docs (troubleshooting, onboarding, README)
- [x] Research best practices (Homebrew, Docker, Git FAQs)
- [x] Define question classification (5 categories, 22 questions)
- [x] Design answer structure template
- [x] Plan cross-reference strategy
- [ ] Review and approval

### Phase 2: Content Creation (P14 - FAQ Implementation)
- [ ] Write English FAQ entries (faq.md)
- [ ] Create Simplified Chinese version (faq.zh-CN.md)
- [ ] Add navigation links from README
- [ ] Add cross-references to/from troubleshooting.md
- [ ] Validate all code examples
- [ ] Review for completeness and clarity

### Phase 3: Integration (P14 continued)
- [ ] Update README Table of Contents
- [ ] Add FAQ to docs/ index
- [ ] Create FAQ section in operator-onboarding.md
- [ ] Add CI check for bilingual FAQ consistency
- [ ] Test all internal links

---

## 9. Question List Summary

### Category 1: Core Concepts (6)
1. What is ADP and why would I use it?
2. What is a workspace?
3. What is a runtime overlay?
4. What files does ADP create and where?
5. How does ADP relate to Codex and Claude?
6. What are tasks and phases?

### Category 2: Usage Decisions (5)
7. When should I use ADP instead of running agent CLI directly?
8. When should I use tasks vs direct `adp run`?
9. How do I choose between `adp run --task` and `adp run --take`?
10. Should I use `--keep-runtime` or let ADP clean up?
11. When should I use profiles vs workspace defaults?

### Category 3: Team Collaboration (4)
12. How do I share workspace configurations with my team?
13. Should I commit `.adp/` or `$ADP_HOME` to Git?
14. How do multiple operators coordinate on shared tasks?
15. How do I hand off work between agents or operators?

### Category 4: Integration Scenarios (4)
16. How do I use ADP in CI/CD pipelines?
17. Can I run ADP in Docker containers?
18. How do I integrate ADP with existing tools?
19. How does ADP work with Git workflows?

### Category 5: Advanced Topics (3)
20. How does session restore and resume work?
21. What are the performance implications of runtime overlays?
22. Can I customize ADP's behavior with hooks or plugins?

**Total: 22 questions across 5 categories**

---

## 10. Sample Answer (Q3: Runtime Overlay)

### What is a runtime overlay?

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
2. Agent sees overlay as working directory
3. On exit, ADP removes overlay (unless `--keep-runtime` was used)
4. Use `adp runtime prune` to clean up kept or stale overlays

**Common Pitfalls:**
- ⚠️ Don't rely on overlay contents after agent exits (they're deleted)
- ⚠️ Don't point `$ADP_RUNTIME_DIR` inside your project (doctor will warn)
- ⚠️ Kept runtimes accumulate disk space; prune regularly

**See also:**
- [Runtime Model](../README.md#runtime-model) - detailed overlay mechanics
- [Troubleshooting: "failed to build runtime"](troubleshooting.md#failed-to-build-runtime) - common overlay creation errors
- [FAQ: What files does ADP create?](#q4-what-files-does-adp-create-and-where) - file location reference

---

## 11. Next Steps

After design approval:
1. Create `docs/faq.md` with full 22-question content
2. Create `docs/faq.zh-CN.md` with Simplified Chinese translations
3. Add cross-references from README, troubleshooting, and guides
4. Validate all examples against current CLI
5. Add to CI bilingual documentation check
6. Update task #5 status to completed

---

**Design Status:** Ready for review
**Target Docs:** `docs/faq.md`, `docs/faq.zh-CN.md`
**Dependencies:** None (uses existing docs as reference)
**Estimated Content Length:** ~150 lines per question × 22 questions = ~3,300 lines per language
