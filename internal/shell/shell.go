package shell

import (
	"context"
	"errors"
	"io"

	"github.com/karoc/adp/internal/adapters"
)

var ErrNotImplemented = errors.New("shell scaffold is not implemented")

type Streams struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func Enter(_ context.Context, _ adapters.RuntimeHandle, _ Streams) error {
	return ErrNotImplemented
}
