# Game Development Example

> Learn agent orchestration patterns for game development with specialized agents for gameplay and graphics engineering.

## What You'll Learn

- **Agent Specialization**: Assign domain-specific tasks to specialized agents
- **Task Dependencies**: Coordinate work between agents using task dependencies
- **Phase-Based Development**: Organize projects into logical development phases
- **Collaboration Patterns**: How agents work together on complex features

## Prerequisites

- **ADP installed**: Run `adp version` to verify
- **Go 1.21+**: Required for building the example game engine
- **5 minutes**: Time budget from setup to running agents

## Quick Start

```bash
# Clone the repository (if not already)
cd examples/game-development

# One-command setup
./setup.sh

# Verify configuration
adp workspace show game-dev

# Start an agent
adp run codex --workspace game-dev
```

**That's it!** No configuration edits required.

## Project Structure

```
game-development/
├── README.md                    # This file
├── setup.sh                     # One-command setup script
├── workspace.yaml               # Workspace configuration
├── AGENTS.md                    # Agent collaboration patterns
├── tasks.yaml                   # Example task definitions
├── phases.yaml                  # Development phase structure
│
├── profiles/                    # Agent profiles
│   ├── gameplay-dev.yaml        # Gameplay engineer
│   └── graphics-dev.yaml        # Graphics engineer
│
├── prompts/                     # Agent instructions
│   ├── gameplay-engineer.md     # Gameplay engineering guidelines
│   └── graphics-engineer.md     # Graphics engineering guidelines
│
├── memory/                      # Shared context
│   └── game-context.md          # Game constants and conventions
│
├── mcp/                         # MCP server config
│   └── config.yaml
│
└── project/                     # Minimal game engine
    ├── main.go                  # Entry point
    ├── go.mod
    └── game/
        ├── engine.go            # Core game loop
        ├── physics.go           # Physics simulation
        ├── renderer.go          # Rendering system
        └── engine_test.go       # Tests
```

## Agent Orchestration

This example demonstrates **domain specialization**:

### gameplay-dev (Gameplay Engineer)
- **Focus**: Game logic, physics, AI
- **Skills**: Algorithm design, numerical methods, systems programming
- **Assigned Tasks**: T1 (gravity), T3 (collision detection)

### graphics-dev (Graphics Engineer)
- **Focus**: Rendering, shaders, performance
- **Skills**: Graphics APIs, optimization, profiling
- **Assigned Tasks**: T2 (render optimization), T4 (sprite rendering)

### Collaboration
For features requiring both domains (e.g., particle systems):
- Both agents coordinate through `AGENTS.md` patterns
- Task dependencies ensure proper sequencing
- Shared memory maintains common conventions

## Try It Out

### 1. Explore the Game Engine

```bash
cd project

# Run tests
go test ./...

# Test mode
./game-engine --test
# Output: Physics: OK, Renderer: OK

# Run demo (5 seconds at 60 FPS)
./game-engine
```

### 2. Review Agent Configuration

```bash
# View agent profiles
cat profiles/gameplay-dev.yaml
cat profiles/graphics-dev.yaml

# Review collaboration patterns
cat AGENTS.md

# Check task assignments
cat tasks.yaml
```

### 3. Launch an Agent

```bash
# Start gameplay engineer
adp run codex --workspace game-dev --profile gameplay-dev

# Or start graphics engineer
adp run codex --workspace game-dev --profile graphics-dev
```

### 4. Assign a Task

Once an agent is running:

```
User: "Work on task T1 - implement gravity physics"

Agent: [reads tasks.yaml, implements gravity, writes tests]
```

## Task Flow Example

From `tasks.yaml`:

```yaml
- id: T1
  title: "Implement gravity physics"
  assignee: gameplay-dev
  priority: high

- id: T3
  title: "Add collision detection"
  assignee: gameplay-dev
  depends_on: [T1]  # Waits for T1
```

This creates a dependency chain: T3 starts only after T1 completes.

## Development Phases

From `phases.yaml`:

- **Phase 1 (Core Engine)**: Physics + Rendering foundation
- **Phase 2 (Gameplay Features)**: Collision + Sprites  
- **Phase 3 (Polish)**: Effects + Visual feedback

Each phase builds on the previous, with clear milestone criteria.

## Performance Targets

Defined in `memory/game-context.md`:

- **Frame Rate**: 60 FPS (16.67ms per frame)
- **Physics Timestep**: Fixed 16.67ms
- **Max Objects**: 100+ without frame drops

Agents consult this shared memory for consistency.

## Next Steps

- **Modify Tasks**: Edit `tasks.yaml` to add your own tasks
- **Customize Agents**: Adjust agent profiles and prompts
- **Extend the Engine**: Add features like collision detection, sprites
- **Try Other Examples**: 
  - `examples/web-application` - Full-stack web development
  - `examples/data-pipeline` - ETL pipeline with quality checks

## Validation

Run the workspace doctor to verify configuration:

```bash
adp workspace doctor game-dev
```

All checks should pass ✓

## Time Budget Verification

- **Setup**: < 2 minutes (`./setup.sh`)
- **First Agent Run**: < 5 minutes (from clone to running)
- **Total**: Meets the "5-minute rule" ✓

## Learn More

- [ADP Documentation](../../docs/)
- [Workspace Configuration Guide](../../docs/workspace.md)
- [Agent Orchestration Patterns](../../docs/agent-patterns.md)
- [Task Management](../../docs/tasks.md)

## Troubleshooting

**Setup fails?**
- Verify Go 1.21+ installed: `go version`
- Check ADP installed: `adp version`
- Ensure project builds: `cd project && go build`

**Agent doesn't see tasks?**
- Verify workspace registered: `adp workspace list`
- Check tasks.yaml exists: `cat tasks.yaml`
- Review agent profile: `cat profiles/gameplay-dev.yaml`

**Tests fail?**
- Check Go dependencies: `cd project && go mod tidy`
- Run tests with output: `go test -v ./...`
