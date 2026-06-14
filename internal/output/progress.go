package output

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Spinner provides a simple progress indicator for long-running operations.
type Spinner struct {
	writer  io.Writer
	message string
	frames  []string
	delay   time.Duration
	active  bool
	done    chan struct{}
	mu      sync.Mutex
	tty     bool
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// NewSpinner creates a new spinner with the given message.
// If output is not a TTY, it just prints the message once without animation.
func NewSpinner(w io.Writer, message string) *Spinner {
	if w == nil {
		w = os.Stderr
	}

	tty := isTTY(w)
	s := &Spinner{
		writer:  w,
		message: message,
		frames:  spinnerFrames,
		delay:   80 * time.Millisecond,
		done:    make(chan struct{}),
		tty:     tty,
	}

	if !tty {
		// Non-TTY: just print the message once
		fmt.Fprintf(w, "%s\n", message)
	}

	return s
}

// Start begins the spinner animation.
func (s *Spinner) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.active || !s.tty {
		return
	}

	s.active = true
	go s.animate()
}

// Stop stops the spinner and clears the line.
func (s *Spinner) Stop() {
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return
	}
	s.active = false
	s.mu.Unlock()

	close(s.done)

	if s.tty {
		// Clear the line
		fmt.Fprintf(s.writer, "\r\033[K")
	}
}

// Success stops the spinner and prints a success message.
func (s *Spinner) Success(message string) {
	s.Stop()
	if message != "" {
		fmt.Fprintf(s.writer, "%s %s\n", Success("✓"), message)
	}
}

// Fail stops the spinner and prints an error message.
func (s *Spinner) Fail(message string) {
	s.Stop()
	if message != "" {
		fmt.Fprintf(s.writer, "%s %s\n", Error("✗"), message)
	}
}

func (s *Spinner) animate() {
	ticker := time.NewTicker(s.delay)
	defer ticker.Stop()

	frame := 0
	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.mu.Lock()
			if !s.active {
				s.mu.Unlock()
				return
			}
			fmt.Fprintf(s.writer, "\r%s %s", Colorize(ColorCyan, s.frames[frame]), s.message)
			frame = (frame + 1) % len(s.frames)
			s.mu.Unlock()
		}
	}
}

// StepProgress tracks progress through multiple steps.
type StepProgress struct {
	writer io.Writer
	steps  []step
	tty    bool
	mu     sync.Mutex
}

type step struct {
	name   string
	status stepStatus
}

type stepStatus int

const (
	stepPending stepStatus = iota
	stepRunning
	stepDone
	stepFailed
)

// NewStepProgress creates a step-based progress tracker.
func NewStepProgress(w io.Writer, steps []string) *StepProgress {
	if w == nil {
		w = os.Stderr
	}

	sp := &StepProgress{
		writer: w,
		steps:  make([]step, len(steps)),
		tty:    isTTY(w),
	}

	for i, name := range steps {
		sp.steps[i] = step{name: name, status: stepPending}
	}

	return sp
}

// StartStep marks a step as running and displays it.
func (sp *StepProgress) StartStep(index int) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if index < 0 || index >= len(sp.steps) {
		return
	}

	sp.steps[index].status = stepRunning
	if sp.tty {
		sp.render()
	} else {
		fmt.Fprintf(sp.writer, "  %s\n", sp.steps[index].name)
	}
}

// CompleteStep marks a step as done.
func (sp *StepProgress) CompleteStep(index int) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if index < 0 || index >= len(sp.steps) {
		return
	}

	sp.steps[index].status = stepDone
	if sp.tty {
		sp.render()
	} else {
		fmt.Fprintf(sp.writer, "  %s ✓\n", sp.steps[index].name)
	}
}

// FailStep marks a step as failed.
func (sp *StepProgress) FailStep(index int) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if index < 0 || index >= len(sp.steps) {
		return
	}

	sp.steps[index].status = stepFailed
	if sp.tty {
		sp.render()
	}
}

// Finish clears the progress display.
func (sp *StepProgress) Finish() {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if sp.tty {
		// Move cursor up by number of steps and clear
		fmt.Fprintf(sp.writer, "\033[%dA\033[J", len(sp.steps))
	}
}

func (sp *StepProgress) render() {
	if !sp.tty {
		return
	}

	// Save cursor position
	fmt.Fprint(sp.writer, "\033[s")

	var output strings.Builder
	for i, s := range sp.steps {
		var prefix string
		switch s.status {
		case stepPending:
			prefix = Colorize(ColorReset, "  ")
		case stepRunning:
			prefix = Colorize(ColorCyan, "⠋ ")
		case stepDone:
			prefix = Success("✓ ")
		case stepFailed:
			prefix = Error("✗ ")
		}
		output.WriteString(fmt.Sprintf("%s%s", prefix, s.name))
		if i < len(sp.steps)-1 {
			output.WriteString("\n")
		}
	}

	// Restore cursor and print
	fmt.Fprintf(sp.writer, "\033[u%s", output.String())
}

func isTTY(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		fileInfo, err := f.Stat()
		if err != nil {
			return false
		}
		return (fileInfo.Mode() & os.ModeCharDevice) != 0
	}
	return false
}
