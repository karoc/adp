package adapters

import (
	"context"
	"io/fs"

	"github.com/karoc/adp/internal/paths"
	"github.com/karoc/adp/internal/schema"
)

type Adapter interface {
	Name() string
	Validate(context.Context, Context) error
	Render(context.Context, Context) (*RenderResult, error)
	Launch(context.Context, Context, RuntimeHandle, []string) (*LaunchSpec, error)
}

type Context struct {
	Layout       paths.Layout
	WorkspaceDir string
	Config       schema.Config
	Agent        schema.AgentConfig
	Profile      string
}

type RenderResult struct {
	Files []GeneratedFile
	Env   map[string]string
}

type GeneratedFile struct {
	Path string
	Mode fs.FileMode
	Data []byte
}

type RuntimeHandle struct {
	SessionID     string
	WorkspaceName string
	ProjectRoot   string
	Root          string
	Env           map[string]string
	Keep          bool
	Warnings      []string
}

type LaunchSpec struct {
	Command string
	Args    []string
	Env     map[string]string
	Dir     string
}
