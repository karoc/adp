package runner

import (
	"context"
	"errors"
	"io"

	"github.com/karoc/adp/internal/adapters"
)

var ErrNotImplemented = errors.New("runner scaffold is not implemented")

type Streams struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

type Result struct {
	ExitCode int
}

func Run(_ context.Context, _ adapters.LaunchSpec, _ Streams) (*Result, error) {
	return nil, ErrNotImplemented
}
