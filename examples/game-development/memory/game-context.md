# Game Context Memory

This file contains shared conventions, constants, and context for the game development project.

## Performance Targets

- **Frame Rate**: 60 FPS (16.67ms per frame)
- **Physics Timestep**: 16.67ms (fixed timestep)
- **Max Objects**: 100+ simultaneous objects without frame drops

## Coordinate System

- **Origin**: Top-left corner (0, 0)
- **X-axis**: Increases to the right
- **Y-axis**: Increases downward
- **Units**: Pixels for position, meters for physics

## Physics Constants

- **Gravity**: 9.81 m/s² (downward)
- **Default Mass**: 1.0 kg
- **Default Friction**: 0.5 (coefficient)
- **Default Restitution**: 0.8 (bounciness)

## Rendering Conventions

- **Clear Color**: Black (0, 0, 0)
- **Coordinate Space**: Screen space (pixels)
- **Z-Order**: Higher values render on top
- **Alpha Blending**: Pre-multiplied alpha

## Code Organization

- `game/engine.go` - Core game loop and orchestration
- `game/physics.go` - Physics simulation
- `game/renderer.go` - Rendering system
- `main.go` - Entry point and configuration

## Testing Strategy

- **Unit Tests**: All physics calculations
- **Integration Tests**: Engine update loop
- **Performance Tests**: Frame time under load
- **Manual Testing**: Visual verification of behavior

## Development Workflow

1. Implement feature in appropriate module
2. Write unit tests
3. Verify integration with engine
4. Profile performance if needed
5. Document any new conventions here

## Known Limitations

- No collision detection (planned for T3)
- No sprite rendering (planned for T4)
- Renderer is a stub (displays nothing)
