# Simple Game Engine Example

This is a minimal game engine demonstration for the ADP game-development example.

## Features

- **Physics Engine**: Basic gravity simulation with fixed timestep
- **Renderer**: Frame rendering stub (extensible for real graphics)
- **Game Loop**: Fixed FPS game loop with update/render cycle

## Building

```bash
go build
```

## Running

```bash
# Run in test mode
./game --test

# Run demo (5 seconds at 60 FPS)
./game

# Custom FPS
./game --fps 120
```

## Testing

```bash
go test ./...
```

## Architecture

- `main.go` - Entry point and game loop
- `game/engine.go` - Core engine orchestration
- `game/physics.go` - Physics simulation
- `game/renderer.go` - Rendering system

## Extending

To add gameplay features:
1. Create new components in the `game/` package
2. Integrate them into the engine update loop
3. Add tests for new components

Common extensions:
- Input handling
- Entity/component system
- Collision detection
- Audio system
- Asset loading
