package game

// Renderer handles rendering the game state
type Renderer struct {
	ready      bool
	frameCount int
}

// NewRenderer creates a new renderer
func NewRenderer() *Renderer {
	return &Renderer{
		ready: true,
	}
}

// Render draws the current frame
// In a real game, this would draw sprites, textures, etc.
// For this example, it's a stub that tracks frame count
func (r *Renderer) Render() {
	r.frameCount++
	// Stub: In a real implementation, this would:
	// - Clear the screen
	// - Draw all game objects
	// - Present the frame buffer
}

// IsReady returns whether the renderer is ready
func (r *Renderer) IsReady() bool {
	return r.ready
}

// FrameCount returns the total number of frames rendered
func (r *Renderer) FrameCount() int {
	return r.frameCount
}

// Reset resets the renderer state
func (r *Renderer) Reset() {
	r.frameCount = 0
}
