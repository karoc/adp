package claude

import (
	"context"

	"github.com/karoc/adp/internal/adapters/api"
	"github.com/karoc/adp/internal/adapters/shared"
)

const Name = "claude"

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

	settings, err := shared.MetadataJSON(Name, adapterCtx)
	if err != nil {
		return nil, err
	}

	return &api.RenderResult{
		Files: []api.GeneratedFile{
			{
				Path: "CLAUDE.md",
				Mode: 0o644,
				Data: shared.Instructions(Name, adapterCtx),
			},
			{
				Path: ".claude/settings.json",
				Mode: 0o644,
				Data: append(settings, '\n'),
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
