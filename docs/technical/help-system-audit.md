# Help System Audit and "See Also" Feature Design

**Date:** 2026-06-14  
**Status:** Design Document  
**Author:** ADP Team

## Executive Summary

This document provides a comprehensive audit of ADP's command help system and proposes a technical design for implementing cross-referenced "See Also" sections to improve command discoverability and reduce user confusion.

**Key Findings:**
- ADP uses a custom command metadata system (no Cobra dependency)
- Help text is centrally defined in `internal/commandmeta/metadata.go`
- Current "See Also" only exists in subcommand help (pointing back to parent)
- 18 root commands with varying complexity (2-12 subcommands each)
- Strong need for cross-command references in task workflow, session recovery, and diagnostics

**Proposed Solution:**
- Extend `commandmeta.Command` struct with `SeeAlso []string` field
- Centralize command relationship mapping in metadata layer
- Implement bilingual support (English/Chinese) for see-also text
- Phased rollout prioritizing high-confusion command pairs

---

## 1. Current Help System Architecture

### 1.1 Core Components

**Location:** `/srv/agent-development-platform/internal/commandmeta/`

**Key Files:**
- `metadata.go` - Command metadata definitions and help rendering
- `examples.go` - Usage examples for commands/subcommands
- `cli.go` - Command dispatcher and error handling

**Data Structures:**

```go
// internal/commandmeta/metadata.go
type Command struct {
    Name        string
    Description string
    Usage       []string      // Multi-line usage examples
    Subcommands []Value       // Nested subcommands
    Options     []Value       // Available flags/options
}

type Value struct {
    Name        string
    Description string
}
```

### 1.2 Help Generation Flow

```
User runs: adp tasks take --help
    ↓
cli.Execute() checks for --help flag
    ↓
commandHelp() called with ["tasks", "take"]
    ↓
commandmeta.SubcommandHelp("tasks", "take")
    ↓
Renders: usage + examples + "See also: adp tasks --help"
```

**Current "See Also" Implementation:**
- **Location:** `metadata.go:450-452`
- **Scope:** Only in subcommand help
- **Format:** Always points back to parent command
- **Example:** `adp tasks take --help` shows "See also: adp tasks --help"

**Limitations:**
1. No cross-command references (e.g., `tasks` ↔ `run`, `sessions` ↔ `events`)
2. No workflow guidance (e.g., `tasks next` → `tasks take` → `run`)
3. No diagnostic command suggestions (e.g., `doctor` when errors occur)

### 1.3 Command Inventory

ADP currently has **18 root commands** across 6 functional areas:

| Category | Commands | Subcommands | Complexity |
|----------|----------|-------------|------------|
| **Setup** | init, quickstart, doctor, version | 0, 0, 0, 0 | Low |
| **Workspace** | workspace | 6 (add, list, show, remove, rename, doctor) | Medium |
| **Runtime** | enter, env, run, runtime | 0, 0, 0, 1 (prune) | Medium |
| **Task Management** | tasks, plan, phase, progress | 12, 3, 8, 1 | High |
| **Observability** | events, sessions | 1 (list), 4 (list, show, restore-plan, resume-plan) | Medium |
| **Shell Integration** | shell-hook, completion | 0, 1 (values) | Low |

**Total:** 18 root commands, 36 subcommands

---

## 2. Command Relationship Analysis

### 2.1 Workflow-Based Relationships

#### **Workflow 1: Initial Setup**
```
init → workspace add → doctor → quickstart
```
- **Confusion point:** Users don't know `doctor` exists after `workspace add`
- **Need:** `workspace add` should suggest `doctor` to verify setup

#### **Workflow 2: Task Pickup**
```
tasks next → tasks take → run --take → tasks renew → tasks done
         ↘ tasks claim ↗
```
- **Confusion point:** Difference between `take` (atomic next) vs `claim` (specific task)
- **Need:** Cross-reference between `tasks next`, `tasks take`, `tasks claim`, and `run --take`

#### **Workflow 3: Planning**
```
plan preview → plan apply → tasks list → phase status
```
- **Confusion point:** Relationship between plan import and task creation
- **Need:** `plan apply` should reference `tasks list` and `phase list`

#### **Workflow 4: Session Recovery**
```
sessions list → sessions show → sessions restore-plan → run
                            ↘ sessions resume-plan ↗
```
- **Confusion point:** When to use `restore-plan` vs `resume-plan`
- **Need:** Clear distinction and cross-reference between recovery commands

#### **Workflow 5: Diagnostics**
```
doctor → workspace doctor → plan doctor → runtime prune
```
- **Confusion point:** Multiple `doctor` variants for different scopes
- **Need:** Root `doctor` should reference specialized diagnostic commands

### 2.2 Confusion Matrix

High-confusion command pairs that need "See Also" references:

| Command Pair | Confusion Type | Priority |
|--------------|----------------|----------|
| `tasks take` ↔ `tasks claim` | Semantic overlap | **P0** |
| `run --take` ↔ `tasks take` | Atomic vs manual | **P0** |
| `doctor` ↔ `workspace doctor` | Scope difference | **P0** |
| `sessions restore-plan` ↔ `sessions resume-plan` | Similar names | **P0** |
| `tasks next` ↔ `tasks take` | Preview vs action | **P1** |
| `plan apply` ↔ `tasks list` | Cause and effect | **P1** |
| `workspace add` ↔ `doctor` | Setup sequence | **P1** |
| `tasks done` ↔ `phase accept` | Task vs phase completion | **P1** |
| `events list` ↔ `sessions list` | Different views | **P2** |
| `runtime prune` ↔ `enter` | Cleanup context | **P2** |

### 2.3 Command Alias Relationships

From `cli.go:204-218`:
```go
aliases := map[string]string{
    "ws": "workspace",
    "t":  "tasks",
    "s":  "sessions",
    "e":  "events",
    "rt": "runtime",
    "p":  "phase",
}
```

**Finding:** Aliases are functional but not documented in help text.  
**Recommendation:** Include alias information in "See Also" or command description.

---

## 3. Design Proposal: "See Also" Feature

### 3.1 Design Goals

1. **Discoverability:** Help users find related commands in their workflow
2. **Clarity:** Reduce confusion between similar commands
3. **Minimal friction:** Don't overwhelm users with too many references
4. **Maintainability:** Centralize relationship mapping for easy updates
5. **Internationalization:** Support bilingual (EN/中文) help text

### 3.2 Data Structure Extension

**File:** `internal/commandmeta/metadata.go`

```go
// Extend Command struct
type Command struct {
    Name        string
    Description string
    Usage       []string
    Subcommands []Value
    Options     []Value
    SeeAlso     []string      // NEW: Related command references
}

// Extend Value struct for subcommands
type Value struct {
    Name        string
    Description string
    SeeAlso     []string      // NEW: Related subcommand references (optional)
}
```

**Alternative Design (More Explicit):**

```go
// More structured approach with relationship types
type RelatedCommand struct {
    Command     string
    Subcommand  string        // Empty for root commands
    Relation    RelationType  // "workflow-next", "alternative", "diagnostic"
    Description string        // Optional context
}

type RelationType string
const (
    RelationWorkflowNext   RelationType = "workflow-next"    // Natural next step
    RelationAlternative    RelationType = "alternative"      // Similar functionality
    RelationDiagnostic     RelationType = "diagnostic"       // Troubleshooting
    RelationParent         RelationType = "parent"           // Parent command
    RelationChild          RelationType = "child"            // Sub-functionality
)
```

**Recommendation:** Start with simple `[]string` approach for MVP, migrate to structured approach if relationship types become valuable.

### 3.3 Command Relationship Mapping

**Central mapping location:** `internal/commandmeta/metadata.go` after `rootCommands` definition

```go
// Command relationship mapping for "See Also" sections
var commandRelationships = map[string][]string{
    // Setup and diagnostics
    "init":         {"workspace", "quickstart"},
    "quickstart":   {"doctor", "workspace add"},
    "doctor":       {"workspace doctor", "plan doctor"},
    
    // Workspace management
    "workspace":    {"doctor", "tasks", "run"},
    
    // Task workflows
    "tasks":        {"run", "phase", "progress"},
    "run":          {"tasks", "events", "sessions"},
    
    // Planning
    "plan":         {"tasks list", "phase list", "plan doctor"},
    "phase":        {"tasks", "progress", "plan"},
    "progress":     {"tasks", "phase", "sessions"},
    
    // Observability
    "events":       {"sessions", "tasks show"},
    "sessions":     {"events", "run", "tasks"},
    
    // Runtime management
    "runtime":      {"enter", "env", "sessions"},
    "enter":        {"env", "run", "runtime prune"},
}

// Subcommand relationships (command.subcommand format)
var subcommandRelationships = map[string][]string{
    // Workspace subcommands
    "workspace.add":    {"doctor", "tasks add"},
    "workspace.doctor": {"doctor", "plan doctor"},
    
    // Task subcommands
    "tasks.next":       {"tasks take", "tasks claim"},
    "tasks.take":       {"run --take", "tasks next", "tasks renew"},
    "tasks.claim":      {"tasks take", "tasks renew"},
    "tasks.done":       {"phase accept", "tasks list"},
    "tasks.stale":      {"tasks renew", "tasks release"},
    
    // Session subcommands
    "sessions.restore-plan": {"sessions resume-plan", "run"},
    "sessions.resume-plan":  {"sessions restore-plan", "run --take"},
    
    // Plan subcommands
    "plan.preview":  {"plan apply"},
    "plan.apply":    {"tasks list", "phase list"},
    "plan.doctor":   {"tasks list", "phase status"},
}
```

### 3.4 Help Rendering Functions

**Modify `CommandHelp()` function:**

```go
// File: internal/commandmeta/metadata.go

func CommandHelp(name string) (string, bool) {
    command, ok := Lookup(name)
    if !ok {
        return "", false
    }

    var out strings.Builder
    out.WriteString("adp ")
    out.WriteString(command.Name)
    if command.Description != "" {
        out.WriteString(" - ")
        out.WriteString(command.Description)
    }
    out.WriteString("\n\nUsage:\n")
    writeUsageLines(&out, command.Usage)
    writeValuesSection(&out, "Subcommands", command.Subcommands)
    writeValuesSection(&out, "Options", command.Options)
    writeExamplesSection(&out, examplesForCommand(command.Name))
    
    // NEW: Add "See Also" section
    writeSeeAlsoSection(&out, name, "")
    
    return out.String(), true
}

func SubcommandHelp(commandName, subcommand string) (string, bool) {
    command, ok := Lookup(commandName)
    if !ok || !hasValue(command.Subcommands, subcommand) {
        return "", false
    }

    usage := usageLinesForSubcommand(command, subcommand)
    if len(usage) == 0 {
        return "", false
    }

    var out strings.Builder
    out.WriteString("adp ")
    out.WriteString(command.Name)
    out.WriteByte(' ')
    out.WriteString(subcommand)
    if description := valueDescription(command.Subcommands, subcommand); description != "" {
        out.WriteString(" - ")
        out.WriteString(description)
    }
    out.WriteString("\n\nUsage:\n")
    writeUsageLines(&out, usage)
    writeExamplesSection(&out, examplesForSubcommand(command.Name, subcommand))
    
    // NEW: Enhanced "See Also" section
    writeSeeAlsoSection(&out, command.Name, subcommand)
    
    return out.String(), true
}
```

**New helper function:**

```go
// writeSeeAlsoSection renders related command references
func writeSeeAlsoSection(out *strings.Builder, commandName, subcommand string) {
    var related []string
    
    if subcommand != "" {
        // Subcommand help: check subcommand relationships first
        key := commandName + "." + subcommand
        if refs, ok := subcommandRelationships[key]; ok {
            related = append(related, refs...)
        }
        // Always include parent command
        related = append(related, commandName+" --help")
    } else {
        // Root command help: check command relationships
        if refs, ok := commandRelationships[commandName]; ok {
            related = append(related, refs...)
        }
    }
    
    if len(related) == 0 {
        return
    }
    
    out.WriteString("\nSee also:\n")
    for _, ref := range related {
        out.WriteString("  adp ")
        // Format reference (add --help if it's just a command name)
        if !strings.Contains(ref, " ") {
            out.WriteString(ref)
            out.WriteString(" --help")
        } else {
            out.WriteString(ref)
        }
        out.WriteByte('\n')
    }
}
```

### 3.5 Internationalization Considerations

For bilingual support (EN/中文), consider:

**Option 1: Separate mapping (simpler)**
```go
var commandRelationshipsZhCN = map[string][]string{
    "init":       {"workspace", "quickstart"},
    // Same keys, localized if needed
}
```

**Option 2: Inline localization (more flexible)**
```go
type LocalizedRef struct {
    Command string
    DescEN  string
    DescZH  string
}
```

**Recommendation:** Start with Option 1 (separate mapping) since command names are not localized. Add descriptions later if needed.

---

## 4. Implementation Plan

### 4.1 Phased Rollout

#### **Phase 1: High-Priority Commands (P0)**
**Estimated effort:** 2-3 days

Target commands with highest confusion:
- `tasks take` ↔ `tasks claim` ↔ `run --take`
- `doctor` ↔ `workspace doctor`
- `sessions restore-plan` ↔ `sessions resume-plan`

**Deliverables:**
1. Add relationship mappings for P0 commands
2. Implement `writeSeeAlsoSection()` function
3. Update `CommandHelp()` and `SubcommandHelp()`
4. Add unit tests for see-also rendering

#### **Phase 2: Workflow Sequences (P1)**
**Estimated effort:** 3-4 days

Target workflow-based relationships:
- `tasks next` → `tasks take` → `tasks renew` → `tasks done`
- `workspace add` → `doctor` → `tasks add`
- `plan preview` → `plan apply` → `tasks list`

**Deliverables:**
1. Expand relationship mappings for P1 commands
2. Add workflow-oriented references
3. Update documentation examples

#### **Phase 3: Comprehensive Coverage (P2)**
**Estimated effort:** 2-3 days

Complete remaining commands:
- Observability commands (`events`, `sessions`)
- Runtime management (`runtime prune`, `enter`, `env`)
- Utility commands (`completion`, `shell-hook`)

**Deliverables:**
1. Full relationship mapping
2. Edge case handling
3. Comprehensive test coverage

### 4.2 Testing Strategy

**Unit Tests:** `internal/commandmeta/metadata_test.go`

```go
func TestSeeAlsoSection(t *testing.T) {
    tests := []struct {
        name       string
        command    string
        subcommand string
        wantRefs   []string
    }{
        {
            name:     "tasks take subcommand",
            command:  "tasks",
            subcommand: "take",
            wantRefs: []string{"run --take", "tasks next", "tasks renew"},
        },
        {
            name:     "doctor root command",
            command:  "doctor",
            subcommand: "",
            wantRefs: []string{"workspace doctor", "plan doctor"},
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            var buf strings.Builder
            writeSeeAlsoSection(&buf, tt.command, tt.subcommand)
            output := buf.String()
            
            for _, ref := range tt.wantRefs {
                if !strings.Contains(output, ref) {
                    t.Errorf("See Also section missing reference: %q", ref)
                }
            }
        })
    }
}
```

**Integration Tests:**

```bash
# Test actual help output
go test ./internal/cli/... -run TestCommandHelp

# Manual verification
adp tasks take --help | grep -A 5 "See also:"
adp doctor --help | grep -A 5 "See also:"
```

### 4.3 Backward Compatibility

**Consideration:** Existing help output is part of the CLI contract.

**Approach:**
1. Additive only - no breaking changes to existing help format
2. "See Also" section always appears at the end
3. Maintain existing "See also: adp <parent> --help" in subcommand help
4. No changes to usage lines or option descriptions

**Example - Before:**
```
adp tasks take - atomically claim next work

Usage:
  adp tasks take [--workspace <name>] --owner <owner> [--lease <duration>]

See also:
  adp tasks --help
```

**Example - After:**
```
adp tasks take - atomically claim next work

Usage:
  adp tasks take [--workspace <name>] --owner <owner> [--lease <duration>]

See also:
  adp run --take
  adp tasks next --help
  adp tasks renew --help
  adp tasks --help
```

---

## 5. Code Examples

### 5.1 Complete Implementation Snippet

```go
// File: internal/commandmeta/metadata.go

// Add after rootCommands definition (line ~296)
var commandRelationships = map[string][]string{
    "init":       {"workspace", "quickstart"},
    "quickstart": {"doctor", "workspace add"},
    "doctor":     {"workspace doctor", "plan doctor"},
    "workspace":  {"doctor", "tasks", "run"},
    "tasks":      {"run", "phase", "progress"},
    "run":        {"tasks", "events", "sessions"},
    "plan":       {"tasks list", "phase list"},
    "phase":      {"tasks", "progress"},
    "events":     {"sessions", "tasks show"},
    "sessions":   {"events", "run"},
    "runtime":    {"enter", "env"},
}

var subcommandRelationships = map[string][]string{
    "workspace.add":           {"doctor", "tasks add"},
    "tasks.next":              {"tasks take", "tasks claim"},
    "tasks.take":              {"run --take", "tasks next", "tasks renew"},
    "tasks.claim":             {"tasks take", "tasks renew"},
    "tasks.done":              {"phase accept"},
    "tasks.stale":             {"tasks renew", "tasks release"},
    "sessions.restore-plan":   {"sessions resume-plan", "run"},
    "sessions.resume-plan":    {"sessions restore-plan", "run --take"},
    "plan.preview":            {"plan apply"},
    "plan.apply":              {"tasks list", "phase list"},
}

func writeSeeAlsoSection(out *strings.Builder, commandName, subcommand string) {
    var related []string
    
    if subcommand != "" {
        // Subcommand-specific relationships
        key := commandName + "." + subcommand
        if refs, ok := subcommandRelationships[key]; ok {
            related = append(related, refs...)
        }
        // Always include parent command reference
        related = append(related, commandName+" --help")
    } else {
        // Root command relationships
        if refs, ok := commandRelationships[commandName]; ok {
            related = append(related, refs...)
        }
    }
    
    if len(related) == 0 {
        return
    }
    
    out.WriteString("\nSee also:\n")
    for _, ref := range related {
        out.WriteString("  adp ")
        // Add --help suffix if reference is just a command name
        if !strings.Contains(ref, " ") && !strings.HasSuffix(ref, "--help") {
            out.WriteString(ref)
            out.WriteString(" --help")
        } else {
            out.WriteString(ref)
        }
        out.WriteByte('\n')
    }
}

// Update CommandHelp() - add after writeExamplesSection call (line ~423)
func CommandHelp(name string) (string, bool) {
    // ... existing code ...
    writeExamplesSection(&out, examplesForCommand(command.Name))
    writeSeeAlsoSection(&out, name, "")  // NEW
    return out.String(), true
}

// Update SubcommandHelp() - replace existing "See also" logic (line ~449-452)
func SubcommandHelp(commandName, subcommand string) (string, bool) {
    // ... existing code ...
    writeExamplesSection(&out, examplesForSubcommand(command.Name, subcommand))
    writeSeeAlsoSection(&out, command.Name, subcommand)  // REPLACE old logic
    return out.String(), true
}
```

### 5.2 Usage Examples After Implementation

```bash
# Example 1: tasks take
$ adp tasks take --help
adp tasks take - atomically claim next work

Usage:
  adp tasks take [--workspace <name>] --owner <owner> [--lease <duration>]

Examples:
  adp tasks take --workspace game-a --owner codex-main --lease 4h

See also:
  adp run --take
  adp tasks next --help
  adp tasks renew --help
  adp tasks --help

# Example 2: doctor
$ adp doctor --help
adp doctor - diagnose registered workspaces

Usage:
  adp doctor [workspace] [--verbose] [--format <text|json>]

See also:
  adp workspace doctor --help
  adp plan doctor --help
```

---

## 6. Alternative Approaches Considered

### 6.1 Cobra Framework Migration

**Approach:** Migrate from custom command system to Cobra framework.

**Pros:**
- Industry standard with mature ecosystem
- Built-in help generation and command groups
- Better command aliasing support

**Cons:**
- Major breaking change requiring full rewrite
- Current system is working and well-understood
- Migration effort: 2-3 weeks
- Risk of regressions in existing workflows

**Decision:** **Rejected** - Too high risk/effort for the benefit. Current system is adequate.

### 6.2 Dynamic Help Generation

**Approach:** Generate see-also references dynamically based on command usage patterns.

**Pros:**
- Automatically adapts to user behavior
- No manual relationship maintenance

**Cons:**
- Requires telemetry/analytics infrastructure
- Privacy concerns with usage tracking
- Complex implementation
- Unpredictable help output

**Decision:** **Rejected** - Violates local-first principle, adds complexity.

### 6.3 Inline Help Hints in Error Messages

**Approach:** Add see-also hints directly in error messages instead of help text.

**Example:**
```
Error: task not found
Hint: Use 'adp tasks list' to see available tasks
```

**Pros:**
- Contextual help exactly when needed
- Already partially implemented in error handling

**Cons:**
- Help text should be comprehensive on its own
- Error messages should focus on the error

**Decision:** **Partial adoption** - Keep both. Error hints complement help text but don't replace it.

---

## 7. Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| **Too many references overwhelm users** | Medium | Medium | Limit to 3-5 most relevant per command |
| **Relationship mapping becomes stale** | Low | Medium | Add validation tests that check referenced commands exist |
| **Circular references in help** | Low | Low | Document relationship patterns, add cycle detection in tests |
| **Breaking existing scripts parsing help** | High | Low | Make changes additive only, preserve existing format |
| **Inconsistent relationship definitions** | Medium | Medium | Centralize all mappings, use code review checklist |

### 7.1 Validation Tests

```go
func TestRelationshipIntegrity(t *testing.T) {
    // Verify all referenced commands exist
    allCommands := map[string]bool{}
    for _, cmd := range rootCommands {
        allCommands[cmd.Name] = true
        for _, sub := range cmd.Subcommands {
            allCommands[cmd.Name+"."+sub.Name] = true
        }
    }
    
    for cmd, refs := range commandRelationships {
        for _, ref := range refs {
            parts := strings.Fields(ref)
            refCmd := parts[0]
            if !allCommands[refCmd] {
                t.Errorf("Command %q references non-existent command %q", cmd, refCmd)
            }
        }
    }
}
```

---

## 8. Success Metrics

### 8.1 Qualitative Metrics

- User feedback: "I found the related command easily"
- Reduced confusion between similar commands
- Faster onboarding for new operators

### 8.2 Quantitative Metrics (Future)

If telemetry is added (opt-in):
- Reduced help command repetition (e.g., fewer `--help` calls for same command)
- Command discovery time (time between related command invocations)
- Error rate reduction after seeing help

**Note:** Current ADP is local-first with no telemetry. Metrics would need explicit opt-in.

---

## 9. Recommendations

### 9.1 Immediate Actions (Week 1)

1. **Implement Phase 1 (P0 commands)**
   - Focus on highest confusion pairs
   - Add relationship mappings for critical workflows
   - Implement `writeSeeAlsoSection()` function
   - Update help rendering functions

2. **Add Validation Tests**
   - Reference integrity checks
   - Help output format verification
   - Ensure no circular dependencies

3. **Document Pattern for Future Commands**
   - Update contribution guide
   - Add checklist for new command additions
   - Document relationship types and when to use each

### 9.2 Follow-up Actions (Weeks 2-3)

1. **Phase 2 Implementation**
   - Workflow-based relationships
   - Expand test coverage
   - Update operator documentation

2. **User Testing**
   - Get feedback from real operators
   - Adjust relationships based on confusion patterns
   - Iterate on reference count (too many vs too few)

3. **Documentation Updates**
   - Update README with see-also examples
   - Add to operator onboarding guide
   - Update Chinese documentation

### 9.3 Long-term Considerations

1. **Consider Structured Relationships**
   - If relationship types become valuable
   - Migrate from `[]string` to `[]RelatedCommand`
   - Add relationship descriptions

2. **Interactive Help**
   - Consider TUI for command exploration
   - Add command tree visualization
   - Provide workflow-based guided tours

3. **Documentation Generation**
   - Auto-generate command reference docs
   - Include see-also in markdown output
   - Maintain single source of truth

---

## 10. Appendix

### 10.1 Full Command Matrix

Complete command inventory with all subcommands:

```
adp
├── init (0 subcommands)
├── quickstart (0 subcommands)
├── doctor (0 subcommands)
├── version (0 subcommands)
├── workspace (6 subcommands)
│   ├── add
│   ├── list
│   ├── show
│   ├── remove
│   ├── rename
│   └── doctor
├── enter (0 subcommands)
├── env (0 subcommands)
├── shell-hook (0 subcommands)
├── completion (1 subcommand)
│   └── values
├── events (1 subcommand)
│   └── list
├── sessions (4 subcommands)
│   ├── list
│   ├── show
│   ├── restore-plan
│   └── resume-plan
├── runtime (1 subcommand)
│   └── prune
├── tasks (12 subcommands)
│   ├── add
│   ├── list
│   ├── next
│   ├── take
│   ├── stale
│   ├── show
│   ├── update
│   ├── claim
│   ├── renew
│   ├── release
│   ├── done
│   └── block
├── plan (3 subcommands)
│   ├── preview
│   ├── apply
│   └── doctor
├── phase (8 subcommands)
│   ├── add
│   ├── list
│   ├── show
│   ├── status
│   ├── start
│   ├── accept
│   ├── commit
│   └── push
├── progress (1 subcommand)
│   └── report
└── run (0 subcommands)
```

### 10.2 Reference CLI Implementations

Studied implementations for comparison:

1. **kubectl** - Kubernetes CLI
   - Uses "See Also" extensively
   - Groups by workflow and resource type
   - Example: `kubectl get --help` references `describe`, `logs`, `exec`

2. **gh** - GitHub CLI
   - Minimal see-also (only parent references)
   - Relies more on command grouping
   - Example: `gh pr` groups all PR-related commands

3. **git** - Git CLI
   - Heavy use of "See Also" in man pages
   - References related porcelain and plumbing commands
   - Example: `git commit` references `add`, `reset`, `amend`

4. **aws** - AWS CLI
   - Structured service grouping
   - Cross-service references are rare
   - Relies on documentation website

**ADP Position:** Closer to git/kubectl model with workflow-oriented references.

### 10.3 Glossary

- **Root command:** Top-level command (e.g., `adp tasks`)
- **Subcommand:** Nested command (e.g., `take` in `adp tasks take`)
- **Command relationship:** Logical connection between commands (workflow, alternative, diagnostic)
- **Help surface:** The complete help text shown for a command
- **See Also section:** The reference list at the end of help text

---

## 11. Conclusion

The proposed "See Also" feature is a low-risk, high-value improvement to ADP's command discoverability. By implementing a centralized relationship mapping and extending the existing help system, we can significantly reduce user confusion without major architectural changes.

**Key Takeaways:**
1. Current help system is well-structured and maintainable
2. "See Also" can be added incrementally with minimal risk
3. Focus on P0 high-confusion commands first
4. Maintain backward compatibility throughout
5. Success depends on careful curation of relationships (quality over quantity)

**Next Steps:**
1. Review this design document with team
2. Get approval for Phase 1 implementation
3. Create implementation tasks in ADP task board
4. Begin coding with P0 command relationships

---

**Document Version:** 1.0  
**Last Updated:** 2026-06-14  
**Review Status:** Ready for team review
