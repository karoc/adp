package runtime

import (
	"context"
	"errors"

	"github.com/karoc/adp/internal/adapters"
	"github.com/karoc/adp/internal/overlay"
	"github.com/karoc/adp/internal/paths"
	"github.com/karoc/adp/internal/schema"
)

var ErrNotImplemented = errors.New("runtime scaffold is not implemented")

type Handle = adapters.RuntimeHandle

type BuildRequest struct {
	Layout       paths.Layout
	Config       schema.Config
	WorkspaceDir string
	Files        []adapters.GeneratedFile
	Backend      overlay.Backend
	Keep         bool
	SessionID    string
}

func Build(_ context.Context, _ BuildRequest) (*Handle, error) {
	return nil, ErrNotImplemented
}
