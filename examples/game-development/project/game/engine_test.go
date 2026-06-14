package game

import (
	"testing"
	"time"
)

func TestNewEngine(t *testing.T) {
	engine := NewEngine(60)

	if engine == nil {
		t.Fatal("NewEngine returned nil")
	}

	if engine.Physics == nil {
		t.Error("Physics engine not initialized")
	}

	if engine.Renderer == nil {
		t.Error("Renderer not initialized")
	}
}

func TestEngineUpdate(t *testing.T) {
	engine := NewEngine(60)

	initialFrameCount := engine.Renderer.FrameCount()

	engine.Update()

	if engine.Renderer.FrameCount() != initialFrameCount+1 {
		t.Errorf("Expected frame count %d, got %d", initialFrameCount+1, engine.Renderer.FrameCount())
	}
}

func TestPhysicsGravity(t *testing.T) {
	physics := NewPhysicsEngine()

	// Create an object at rest
	obj := physics.AddObject(1.0, Vector2D{X: 0, Y: 100}, Vector2D{X: 0, Y: 0})

	initialY := obj.Position.Y

	// Step physics for 1 second
	for i := 0; i < 60; i++ {
		physics.Step(time.Millisecond * 16667 / 1000)
	}

	// Object should have fallen due to gravity
	if obj.Position.Y >= initialY {
		t.Errorf("Expected object to fall, but Y position went from %f to %f", initialY, obj.Position.Y)
	}
}

func TestPhysicsAddObject(t *testing.T) {
	physics := NewPhysicsEngine()

	initialCount := physics.ObjectCount()

	physics.AddObject(1.0, Vector2D{}, Vector2D{})

	if physics.ObjectCount() != initialCount+1 {
		t.Errorf("Expected object count %d, got %d", initialCount+1, physics.ObjectCount())
	}
}

func TestRendererIsReady(t *testing.T) {
	renderer := NewRenderer()

	if !renderer.IsReady() {
		t.Error("Renderer should be ready after creation")
	}
}

func TestRendererFrameCount(t *testing.T) {
	renderer := NewRenderer()

	if renderer.FrameCount() != 0 {
		t.Errorf("Expected initial frame count 0, got %d", renderer.FrameCount())
	}

	renderer.Render()

	if renderer.FrameCount() != 1 {
		t.Errorf("Expected frame count 1 after render, got %d", renderer.FrameCount())
	}
}

func TestRendererReset(t *testing.T) {
	renderer := NewRenderer()

	renderer.Render()
	renderer.Render()

	if renderer.FrameCount() != 2 {
		t.Fatalf("Setup failed: expected 2 frames, got %d", renderer.FrameCount())
	}

	renderer.Reset()

	if renderer.FrameCount() != 0 {
		t.Errorf("Expected frame count 0 after reset, got %d", renderer.FrameCount())
	}
}
