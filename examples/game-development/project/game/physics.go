package game

import (
	"time"
)

// PhysicsEngine handles physics simulation
type PhysicsEngine struct {
	gravity      float64
	objects      []*PhysicsObject
	active       bool
	timeAccum    time.Duration
}

// PhysicsObject represents an object in the physics simulation
type PhysicsObject struct {
	Mass     float64
	Position Vector2D
	Velocity Vector2D
}

// Vector2D represents a 2D vector
type Vector2D struct {
	X, Y float64
}

// NewPhysicsEngine creates a new physics engine
func NewPhysicsEngine() *PhysicsEngine {
	return &PhysicsEngine{
		gravity: 9.81, // m/s²
		objects: make([]*PhysicsObject, 0),
		active:  true,
	}
}

// Step advances the physics simulation by dt
func (p *PhysicsEngine) Step(dt time.Duration) {
	p.timeAccum += dt

	// Fixed timestep physics (16.67ms for 60 FPS)
	fixedDt := time.Millisecond * 16667 / 1000

	for p.timeAccum >= fixedDt {
		p.updatePhysics(fixedDt.Seconds())
		p.timeAccum -= fixedDt
	}
}

// updatePhysics performs the actual physics calculations
func (p *PhysicsEngine) updatePhysics(dt float64) {
	for _, obj := range p.objects {
		// Apply gravity
		obj.Velocity.Y -= p.gravity * dt

		// Update position
		obj.Position.X += obj.Velocity.X * dt
		obj.Position.Y += obj.Velocity.Y * dt
	}
}

// AddObject adds a new object to the physics simulation
func (p *PhysicsEngine) AddObject(mass float64, pos, vel Vector2D) *PhysicsObject {
	obj := &PhysicsObject{
		Mass:     mass,
		Position: pos,
		Velocity: vel,
	}
	p.objects = append(p.objects, obj)
	return obj
}

// IsActive returns whether the physics engine is active
func (p *PhysicsEngine) IsActive() bool {
	return p.active
}

// ObjectCount returns the number of objects in the simulation
func (p *PhysicsEngine) ObjectCount() int {
	return len(p.objects)
}
