# Agent Orchestration: Game Development

This example demonstrates agent collaboration patterns for game development projects.

## Agents

### gameplay-dev (Gameplay Engineer)
- **Focus**: Game logic, physics, AI, and gameplay mechanics
- **Skills**: Algorithm design, mathematical modeling, systems programming
- **Responsibilities**:
  - Implement game rules and mechanics
  - Physics simulation and collision detection
  - AI behavior and pathfinding
  - Input handling and player controls

### graphics-dev (Graphics Engineer)
- **Focus**: Rendering, shaders, and performance optimization
- **Skills**: Graphics programming, shader languages, optimization
- **Responsibilities**:
  - Rendering pipeline implementation
  - Shader development and effects
  - Performance profiling and optimization
  - Asset loading and management

## Collaboration Patterns

### Pattern 1: Feature Development
When adding a new game feature:
1. **gameplay-dev** implements core logic and data structures
2. **graphics-dev** adds visual representation
3. Both agents review integration points

Example: Adding a particle system
- gameplay-dev: Particle physics and lifecycle
- graphics-dev: Rendering and visual effects

### Pattern 2: Performance Optimization
When optimizing performance:
1. **graphics-dev** profiles rendering bottlenecks
2. **gameplay-dev** optimizes physics calculations
3. Both coordinate to reduce memory allocations

### Pattern 3: Bug Fixing
When fixing bugs:
- Identify which subsystem is affected
- Assign to appropriate specialist
- Other agent reviews for side effects

## Task Assignment Guidelines

Assign to **gameplay-dev**:
- Physics and collision logic
- Game state management
- AI and behavior systems
- Input processing

Assign to **graphics-dev**:
- Rendering performance
- Visual effects and shaders
- Frame timing and vsync
- Asset pipeline

Assign to **both** (requires coordination):
- Architecture changes affecting both systems
- Integration of new libraries
- Major refactoring
- Performance optimization spanning both domains

## Communication

Agents communicate through:
- **Shared memory** (`memory/game-context.md`) - Key constants and conventions
- **Code comments** - Interface contracts and expectations
- **Task dependencies** - Explicit ordering in `tasks.yaml`

## Example Task Flow

```yaml
# Example from tasks.yaml
- id: T1
  title: "Add gravity to physics engine"
  assignee: gameplay-dev
  
- id: T2
  title: "Visualize physics debug info"
  assignee: graphics-dev
  depends_on: [T1]
```

This creates a dependency: T2 waits for T1 to complete.

## Detailed Collaboration Scenarios

### Scenario 1: Adding Collision Detection

**Step 1 - gameplay-dev**: Implement collision algorithm
```go
// gameplay-dev implements core collision logic
func (p *PhysicsEngine) CheckCollision(a, b *Object) bool {
    // AABB collision detection
    return a.Bounds.Intersects(b.Bounds)
}
```

**Step 2 - graphics-dev**: Add visual collision feedback
```go
// graphics-dev adds debug visualization
func (r *Renderer) DrawCollisionBox(obj *Object) {
    // Draw red box around colliding objects
    r.DrawBox(obj.Bounds, ColorRed)
}
```

**Coordination**: gameplay-dev exposes `GetCollidingObjects()` interface, graphics-dev consumes it for rendering.

### Scenario 2: Performance Optimization

**Issue**: Frame rate drops below 60 FPS with 100+ objects

**Step 1 - graphics-dev**: Profile rendering
- Identify bottleneck: draw calls per frame
- Propose: sprite batching

**Step 2 - gameplay-dev**: Profile physics
- Identify bottleneck: O(n²) collision checks
- Propose: spatial partitioning (grid-based)

**Step 3 - Joint implementation**:
- gameplay-dev: Implements spatial grid
- graphics-dev: Implements sprite batching
- Both: Validate combined 60 FPS target met

**Communication**: Document optimization results in `memory/game-context.md`

### Scenario 3: Particle System (Collaborative Feature)

This requires both agents working in parallel on different aspects:

**gameplay-dev responsibilities**:
```go
// Particle lifecycle and physics
type Particle struct {
    Position Vector2D
    Velocity Vector2D
    LifeTime float64
}

func (p *ParticleSystem) Update(dt float64) {
    // Update particle physics
    for _, particle := range p.particles {
        particle.Velocity.Y += p.gravity * dt
        particle.Position.Add(particle.Velocity.Scale(dt))
        particle.LifeTime -= dt
    }
    // Remove dead particles
    p.removeExpired()
}
```

**graphics-dev responsibilities**:
```go
// Particle rendering and visual effects
func (r *Renderer) RenderParticles(particles []Particle) {
    // Batch all particles into single draw call
    batch := r.BeginParticleBatch()
    for _, p := range particles {
        batch.Add(p.Position, p.Color, p.Size)
    }
    batch.Render()
}
```

**Interface contract** (documented in `memory/game-context.md`):
```go
// Agreed interface between agents
type ParticleData interface {
    GetPosition() Vector2D
    GetColor() Color
    GetSize() float64
}
```

**Workflow**:
1. Both agents review T5 task description
2. Agree on interface in memory/game-context.md
3. gameplay-dev implements physics in parallel with graphics-dev rendering
4. Integration test verifies both parts work together

### Scenario 4: Bug Fix Coordination

**Bug Report**: Objects fall through floor

**Step 1 - gameplay-dev investigates**:
- Check physics collision detection
- Finds issue: timestep too large causing tunneling

**Step 2 - gameplay-dev fixes**:
- Reduce physics timestep
- Add continuous collision detection

**Step 3 - graphics-dev verifies**:
- Run visual tests
- Confirm objects no longer fall through
- Update rendering to show collision contacts

**Documentation**: gameplay-dev updates `memory/game-context.md` with new timestep value
