package overlay

import (
	"context"

	"github.com/karoc/adp/internal/adapters"
)

type Backend interface {
	Materialize(context.Context, Request) (*Result, error)
	Cleanup(context.Context, Handle) error
}

type Request struct {
	WorkspaceName string
	ProjectRoot   string
	RuntimeRoot   string
	Files         []adapters.GeneratedFile
	ReservedPaths []string
	Keep          bool
}

type Handle struct {
	Root           string
	WorkspaceName  string
	ProjectRoot    string
	GeneratedPaths []string
	LinkedPaths    []string
	SkippedPaths   []string
	Conflicts      []Conflict
	Keep           bool
}

type Result = Handle

type Conflict struct {
	Path   string
	Reason string
}
