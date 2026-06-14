package game

import (
	"time"
)

// Engine is the core game engine managing the game loop
type Engine struct {
	Physics  *PhysicsEngine
	Renderer *Renderer
	fps      int
}

// NewEngine creates a new game engine instance
func NewEngine(fps int) *Engine {
	return &Engine{
		Physics:  NewPhysicsEngine(),
		Renderer: NewRenderer(),
		fps:      fps,
	}
}

// Update runs one frame of the game loop
func (e *Engine) Update() {
	// Update physics simulation
	e.Physics.Step(e.FrameDuration())

	// Render the current state
	e.Renderer.Render()
}

// FrameDuration returns the duration of one frame
func (e *Engine) FrameDuration() time.Duration {
	return time.Second / time.Duration(e.fps)
}
