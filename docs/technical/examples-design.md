# Examples Directory Design

## Executive Summary

This document defines the organization structure, content specifications, and quality standards for the `examples/` directory. The goal is to provide domain-specific, production-ready examples that enable users to go from clone to running agent in under 5 minutes.

## Audit of Current State

### Existing: `examples/basic-workspace/`

**Strengths:**
- ✅ Complete workspace structure with all essential directories (prompts, memory, mcp, profiles)
- ✅ Bilingual documentation (English + Chinese)
- ✅ Detailed setup instructions with copy-paste commands
- ✅ Provider-free onboarding path using fake CLI
- ✅ Clear separation of concerns (workspace config, profiles, prompts, memory)

**Limitations:**
- ❌ Generic example without domain-specific context
- ❌ No actual project code - user must supply their own
- ❌ Requires multiple manual edits before use (workspace name, project root)
- ❌ No task/phase examples despite ADP supporting tasks.yaml
- ❌ No AGENTS.md example showing agent orchestration patterns

**Time to Value:** ~10-15 minutes (requires creating project, editing paths, validation)

## Research: Industry Best Practices

### Pattern Analysis

Based on research of Docker, Vercel, and Kubernetes examples repositories:

**Common Success Patterns:**
1. **Category-based organization** - Examples grouped by use case or domain
2. **Self-contained directories** - Each example is a complete, runnable unit
3. **Consistent structure** - Predictable file layout across all examples
4. **README-first approach** - Every example starts with clear setup instructions
5. **Copy-paste commands** - No manual editing required to get started
6. **Verification steps** - Clear success criteria and expected outputs
7. **Progressive complexity** - Basic → intermediate → advanced paths

**Key Files Per Example:**
- `README.md` - Setup guide with prerequisites, quick start, architecture notes
- Configuration files - Ready-to-use, no placeholders requiring edits
- Sample project - Minimal but functional codebase
- Validation script - Automated verification of successful setup

**Sources:**
- [Docker examples best practices](https://docs.docker.com/articles/dockerfile_best-practices/)
- [Vercel examples repository](https://github.com/vercel/examples)
- [Kubernetes configuration best practices](https://kubernetes.io/blog/2025/11/25/configuration-good-practices/)

## Proposed Directory Structure

```
examples/
├── README.md                          # Examples index and navigation
├── README.zh-CN.md
├── _templates/                        # Shared reusable components
│   ├── workspace-base.yaml
│   ├── profiles/
│   │   ├── codex.yaml
│   │   ├── claude.yaml
│   │   └── architect.yaml
│   ├── prompts/
│   │   ├── coding-style.md
│   │   └── testing-requirements.md
│   └── mcp/
│       └── config.yaml
│
├── game-development/                  # Domain: Game Development
│   ├── README.md
│   ├── README.zh-CN.md
│   ├── workspace.yaml
│   ├── AGENTS.md                     # Agent orchestration pattern
│   ├── tasks.yaml                    # Example task definitions
│   ├── phases.yaml                   # Example phase structure
│   ├── prompts/
│   │   ├── gameplay-engineer.md
│   │   └── graphics-engineer.md
│   ├── profiles/
│   │   ├── gameplay-dev.yaml
│   │   └── graphics-dev.yaml
│   ├── memory/
│   │   └── game-context.md
│   ├── mcp/
│   │   └── config.yaml
│   └── project/                      # Minimal runnable game
│       ├── main.go
│       ├── go.mod
│       ├── game/
│       │   ├── engine.go
│       │   └── physics.go
│       └── README.md
│
├── web-application/                   # Domain: Web Development
│   ├── README.md
│   ├── README.zh-CN.md
│   ├── workspace.yaml
│   ├── AGENTS.md
│   ├── tasks.yaml
│   ├── phases.yaml
│   ├── prompts/
│   │   ├── frontend-engineer.md
│   │   └── backend-engineer.md
│   ├── profiles/
│   │   ├── frontend-dev.yaml
│   │   └── backend-dev.yaml
│   ├── memory/
│   │   └── api-contracts.md
│   ├── mcp/
│   │   └── config.yaml
│   └── project/                      # Minimal API + frontend
│       ├── backend/
│       │   ├── main.go
│       │   ├── go.mod
│       │   └── api/
│       ├── frontend/
│       │   ├── package.json
│       │   ├── src/
│       │   └── public/
│       └── README.md
│
└── data-pipeline/                     # Domain: Data Engineering
    ├── README.md
    ├── README.zh-CN.md
    ├── workspace.yaml
    ├── AGENTS.md
    ├── tasks.yaml
    ├── phases.yaml
    ├── prompts/
    │   ├── etl-engineer.md
    │   └── data-quality.md
    ├── profiles/
    │   ├── etl-dev.yaml
    │   └── qa-dev.yaml
    ├── memory/
    │   └── pipeline-schema.md
    ├── mcp/
    │   └── config.yaml
    └── project/                       # Minimal ETL pipeline
        ├── main.go
        ├── go.mod
        ├── pipeline/
        │   ├── extract.go
        │   ├── transform.go
        │   └── load.go
        └── README.md
```

## Design Principles

### 1. Zero-Edit Quick Start

**Problem:** Current basic-workspace requires editing workspace.yaml before use.

**Solution:** Each example includes a self-contained project directory with absolute paths that work out-of-the-box.

```bash
# User workflow - no edits required
cd examples/game-development
./setup.sh                           # One-command setup
adp workspace show game-dev          # Verify
adp run codex --workspace game-dev   # Launch agent
```

### 2. Domain-Specific Context

**Problem:** Generic examples don't demonstrate real-world agent orchestration patterns.

**Solution:** Each example represents a complete domain with:
- Realistic project structure
- Domain-specific agent profiles (gameplay-dev, graphics-dev)
- Relevant task definitions (implement physics, optimize rendering)
- Contextual memory (game state, physics constants)

### 3. Progressive Learning Path

Examples are ordered by complexity:

1. **game-development** - Single domain, 2 agents, simple tasks
2. **web-application** - Multi-component (frontend/backend), 2+ agents, API contracts
3. **data-pipeline** - Complex orchestration, data quality checks, multi-phase workflows

### 4. Copy-Paste Commands

Every README includes:
```bash
# Complete command sequences with no placeholders
export ADP_HOME="$(pwd)/.adp-state"
./setup.sh
adp workspace doctor game-dev
adp run codex --workspace game-dev
```

### 5. Verification Built-In

Each example includes:
- Expected output samples in README
- Automated validation script
- Success criteria checklist

## Example Specifications

### Example 1: Game Development

**Domain:** Game development with gameplay and graphics specialization

**Use Case:** A small game engine project where agents collaborate on gameplay logic and rendering optimization.

**Project Structure:**
```
project/
├── main.go              # Entry point, ~50 lines
├── go.mod
├── game/
│   ├── engine.go        # Core game loop
│   ├── physics.go       # Physics simulation
│   └── renderer.go      # Rendering stub
└── README.md
```

**Agents:**
- `gameplay-dev` - Focuses on game logic, physics, AI
- `graphics-dev` - Focuses on rendering, shaders, optimization

**Sample Tasks (tasks.yaml):**
```yaml
tasks:
  - id: T1
    title: "Implement gravity physics"
    description: "Add gravity acceleration to physics engine"
    assignee: gameplay-dev
    
  - id: T2
    title: "Optimize render loop"
    description: "Reduce draw calls in renderer"
    assignee: graphics-dev
```

**Memory Context:**
```markdown
# game-context.md
- Game runs at 60 FPS target
- Physics timestep: 16.67ms
- Coordinate system: Y-up, right-handed
```

**Setup Time:** < 3 minutes

**Validation:**
```bash
cd project
go build && ./game --test
# Expected: "Physics: OK, Renderer: OK"
```

### Example 2: Web Application

**Domain:** Full-stack web development with API backend and React frontend

**Use Case:** A REST API service with a web interface, demonstrating frontend/backend agent collaboration.

**Project Structure:**
```
project/
├── backend/
│   ├── main.go          # HTTP server, ~80 lines
│   ├── go.mod
│   └── api/
│       ├── handlers.go
│       └── models.go
├── frontend/
│   ├── package.json
│   ├── src/
│   │   ├── App.tsx
│   │   └── api.ts       # API client
│   └── public/
└── README.md
```

**Agents:**
- `frontend-dev` - React, TypeScript, UI components
- `backend-dev` - Go, REST API, data models

**Sample Tasks:**
```yaml
tasks:
  - id: T1
    title: "Add user authentication endpoint"
    description: "POST /api/auth/login with JWT"
    assignee: backend-dev
    
  - id: T2
    title: "Create login form component"
    description: "React form calling /api/auth/login"
    assignee: frontend-dev
    depends_on: [T1]
```

**Memory Context:**
```markdown
# api-contracts.md
## Authentication
- POST /api/auth/login
- Request: { username, password }
- Response: { token, expires_at }
```

**Setup Time:** < 4 minutes

**Validation:**
```bash
# Terminal 1
cd project/backend && go run main.go

# Terminal 2
cd project/frontend && npm install && npm start

# Check: http://localhost:3000 loads, API responds at :8080
```

### Example 3: Data Pipeline

**Domain:** ETL data pipeline with quality checks

**Use Case:** A data processing pipeline that extracts, transforms, and loads data with quality validation.

**Project Structure:**
```
project/
├── main.go              # Pipeline orchestrator
├── go.mod
├── pipeline/
│   ├── extract.go       # Data extraction
│   ├── transform.go     # Data transformation
│   ├── load.go          # Data loading
│   └── validate.go      # Quality checks
├── data/
│   ├── input/           # Sample input data
│   └── output/          # Expected output
└── README.md
```

**Agents:**
- `etl-dev` - Pipeline logic, data transformations
- `qa-dev` - Quality checks, validation rules

**Sample Tasks:**
```yaml
tasks:
  - id: T1
    title: "Add null value handling"
    description: "Handle missing data in transform step"
    assignee: etl-dev
    
  - id: T2
    title: "Add data completeness check"
    description: "Validate all required fields present"
    assignee: qa-dev
    
  - id: T3
    title: "Integration test"
    description: "End-to-end pipeline test"
    depends_on: [T1, T2]
```

**Phases:**
```yaml
phases:
  - id: P1
    name: "Development"
    tasks: [T1, T2]
    
  - id: P2
    name: "Testing"
    tasks: [T3]
    depends_on: [P1]
```

**Setup Time:** < 5 minutes

**Validation:**
```bash
cd project
go build && ./pipeline --input data/input --output /tmp/output
diff -r /tmp/output data/output/
# Expected: No differences
```

## Reusable Components (_templates/)

### Purpose

Avoid duplication by providing shared components that examples extend:

```yaml
# game-development/workspace.yaml includes template
extends: ../_templates/workspace-base.yaml

workspace:
  name: game-dev

project:
  root: ./project  # Relative path to bundled project
```

### Template Contents

**workspace-base.yaml:**
```yaml
version: 1

memory:
  enabled: true

rules:
  coding_style: strict

mcp:
  enabled: true
  config: mcp/config.yaml
```

**profiles/codex.yaml:**
```yaml
profile: default
command: codex
notes:
  - Use AGENTS.md as primary runtime instruction
  - Consult tasks.yaml for current work items
```

**prompts/coding-style.md:**
```markdown
# Coding Style Guidelines
- Prefer readability over cleverness
- Write tests for new features
- Keep functions under 50 lines
```

## Setup Scripts

Each example includes a `setup.sh` script:

```bash
#!/usr/bin/env bash
set -euo pipefail

EXAMPLE_NAME="game-development"
EXAMPLE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Create isolated ADP state
export ADP_HOME="${EXAMPLE_DIR}/.adp-state"
export ADP_RUNTIME_DIR="${EXAMPLE_DIR}/.adp-runtime"

echo "Setting up ${EXAMPLE_NAME} example..."

# Initialize ADP
adp init

# Copy workspace configuration
mkdir -p "${ADP_HOME}/workspaces"
cp -R "${EXAMPLE_DIR}/workspace.yaml" "${ADP_HOME}/workspaces/${EXAMPLE_NAME}/"
cp -R "${EXAMPLE_DIR}/prompts" "${ADP_HOME}/workspaces/${EXAMPLE_NAME}/"
cp -R "${EXAMPLE_DIR}/profiles" "${ADP_HOME}/workspaces/${EXAMPLE_NAME}/"
cp -R "${EXAMPLE_DIR}/memory" "${ADP_HOME}/workspaces/${EXAMPLE_NAME}/"
cp -R "${EXAMPLE_DIR}/mcp" "${ADP_HOME}/workspaces/${EXAMPLE_NAME}/"

# Validate workspace
adp workspace doctor "${EXAMPLE_NAME}"

echo "✓ Setup complete!"
echo ""
echo "Next steps:"
echo "  adp workspace show ${EXAMPLE_NAME}"
echo "  adp run codex --workspace ${EXAMPLE_NAME}"
```

## README Template

Every example README follows this structure:

```markdown
# [Example Name]

> [One-sentence description of what this example demonstrates]

## What You'll Learn

- [Key concept 1]
- [Key concept 2]
- [Key concept 3]

## Prerequisites

- Go 1.21+ (for this example)
- ADP installed (`adp version`)

## Quick Start

[Copy-paste command block - no edits required]

## Project Structure

[Tree view of project/ directory with explanations]

## Agent Orchestration

[Explanation of how agents collaborate in this domain]

## Try It Out

[Suggested commands and expected outputs]

## Next Steps

- [Link to related example]
- [Link to relevant documentation]
```

## Validation Standards

Each example must pass:

### 1. Time Budget
- Setup script completes in < 2 minutes
- First agent run starts in < 5 minutes from clone

### 2. Zero-Edit Requirement
```bash
# Must work without any file editing
git clone <repo>
cd examples/game-development
./setup.sh
adp run codex --workspace game-dev
```

### 3. Project Validity
- Project code compiles/runs without errors
- Includes at least one test that passes
- README verification steps succeed

### 4. Documentation Completeness
- README includes prerequisites
- README includes copy-paste setup commands
- README includes expected output samples
- Bilingual (English + Chinese)

### 5. Workspace Validation
```bash
adp workspace doctor <name>  # Must exit 0
adp workspace show <name>    # Must print valid YAML
```

## Migration Plan

### Phase 1: Foundation (Week 1)
1. Create `_templates/` with reusable components
2. Update `examples/README.md` with navigation
3. Create `setup.sh` template script

### Phase 2: First Example (Week 1-2)
1. Implement `game-development/` example
2. Create minimal game project
3. Write and test README
4. Validate < 5 minute time budget

### Phase 3: Additional Examples (Week 2-3)
1. Implement `web-application/` example
2. Implement `data-pipeline/` example
3. Cross-link READMEs
4. Final validation pass

### Phase 4: Polish (Week 3)
1. Bilingual documentation complete
2. Automated CI validation
3. Update main documentation to reference examples

## Success Metrics

- **Time to First Run:** < 5 minutes for any example
- **Setup Success Rate:** 100% without manual edits
- **Documentation Clarity:** User can understand agent orchestration pattern from README
- **Reusability:** Users copy example as starting point for real projects

## Future Extensions

After initial 3 examples:

1. **CLI Tool Development** - Example for building CLI apps
2. **Microservices** - Multi-service architecture with service mesh
3. **Machine Learning** - Model training pipeline with experiment tracking
4. **Infrastructure** - Terraform/Kubernetes deployment automation

## References

- Current basic-workspace: `/srv/agent-development-platform/examples/basic-workspace/`
- Docker compose examples: https://github.com/docker/awesome-compose
- Vercel examples: https://github.com/vercel/examples
- Kubernetes patterns: https://kubernetes.io/examples/
