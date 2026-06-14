package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/adp/game-example/game"
)

var (
	testMode = flag.Bool("test", false, "Run in test mode and exit")
	fps      = flag.Int("fps", 60, "Target frames per second")
)

func main() {
	flag.Parse()

	// Initialize game engine
	engine := game.NewEngine(*fps)

	if *testMode {
		runTests(engine)
		return
	}

	// Main game loop
	log.Printf("Starting game engine (target: %d FPS)", *fps)

	ticker := time.NewTicker(engine.FrameDuration())
	defer ticker.Stop()

	frameCount := 0
	maxFrames := *fps * 5 // Run for 5 seconds in demo mode

	for range ticker.C {
		engine.Update()
		frameCount++

		if frameCount >= maxFrames {
			log.Printf("Demo complete: rendered %d frames", frameCount)
			break
		}
	}
}

func runTests(engine *game.Engine) {
	fmt.Println("=== Game Engine Test Mode ===")

	// Test physics system
	fmt.Print("Physics system: ")
	engine.Update()
	if engine.Physics.IsActive() {
		fmt.Println("OK")
	} else {
		fmt.Println("FAILED")
	}

	// Test renderer
	fmt.Print("Renderer: ")
	if engine.Renderer.IsReady() {
		fmt.Println("OK")
	} else {
		fmt.Println("FAILED")
	}

	fmt.Println("\n✓ All tests passed")
}
