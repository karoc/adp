package workspace

import (
	"context"
	"errors"

	"github.com/karoc/adp/internal/paths"
	"github.com/karoc/adp/internal/schema"
)

var ErrNotImplemented = errors.New("workspace registry scaffold is not implemented")

type Registry struct {
	Layout paths.Layout
}

func NewRegistry(layout paths.Layout) *Registry {
	return &Registry{Layout: layout}
}

func (r *Registry) Init(_ context.Context) error {
	return ErrNotImplemented
}

func (r *Registry) Add(_ context.Context, _ string, _ string) (*schema.Config, error) {
	return nil, ErrNotImplemented
}

func (r *Registry) Get(_ context.Context, _ string) (*schema.Config, string, error) {
	return nil, "", ErrNotImplemented
}
