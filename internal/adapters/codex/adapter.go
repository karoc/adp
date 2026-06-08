package codex

import (
	"context"

	"github.com/karoc/adp/internal/adapters/api"
	"github.com/karoc/adp/internal/adapters/shared"
)

const Name = "codex"

type Adapter struct{}

func New() *Adapter {
	return &Adapter{}
}

func (a *Adapter) Name() string {
	return Name
}

func (a *Adapter) Validate(ctx context.Context, _ api.Context) error {
	return ctx.Err()
}

func (a *Adapter) Render(ctx context.Context, adapterCtx api.Context) (*api.RenderResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return &api.RenderResult{
		Files: []api.GeneratedFile{
			{
				Path: "AGENTS.md",
				Mode: 0o644,
				Data: shared.Instructions(Name, adapterCtx),
			},
			{
				Path: ".codex/config.toml",
				Mode: 0o644,
				Data: shared.MetadataTOML(Name, adapterCtx),
			},
		},
		Env: shared.RenderEnv(Name, adapterCtx),
	}, nil
}

func (a *Adapter) Launch(ctx context.Context, adapterCtx api.Context, runtime api.RuntimeHandle, extraArgs []string) (*api.LaunchSpec, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return shared.Launch(Name, adapterCtx, runtime, Name, extraArgs), nil
}
