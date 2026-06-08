package schema

import (
	"fmt"
	"regexp"
)

const CurrentVersion = 1

var workspaceNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

type Config struct {
	Version   int                    `yaml:"version" json:"version"`
	Workspace Workspace              `yaml:"workspace" json:"workspace"`
	Project   Project                `yaml:"project" json:"project"`
	Memory    Memory                 `yaml:"memory,omitempty" json:"memory,omitempty"`
	Prompts   Prompts                `yaml:"prompts,omitempty" json:"prompts,omitempty"`
	Rules     map[string]string      `yaml:"rules,omitempty" json:"rules,omitempty"`
	MCP       MCP                    `yaml:"mcp,omitempty" json:"mcp,omitempty"`
	Agents    map[string]AgentConfig `yaml:"agents,omitempty" json:"agents,omitempty"`
}

type Workspace struct {
	Name string `yaml:"name" json:"name"`
}

type Project struct {
	Root string `yaml:"root" json:"root"`
}

type Memory struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	Shared  string `yaml:"shared,omitempty" json:"shared,omitempty"`
}

type Prompts struct {
	Base string `yaml:"base,omitempty" json:"base,omitempty"`
}

type MCP struct {
	Enabled bool     `yaml:"enabled" json:"enabled"`
	Config  string   `yaml:"config,omitempty" json:"config,omitempty"`
	Servers []string `yaml:"servers,omitempty" json:"servers,omitempty"`
}

type AgentConfig struct {
	Enabled bool              `yaml:"enabled" json:"enabled"`
	Profile string            `yaml:"profile,omitempty" json:"profile,omitempty"`
	Command string            `yaml:"command,omitempty" json:"command,omitempty"`
	Options map[string]string `yaml:"options,omitempty" json:"options,omitempty"`
}

func (c Config) Validate() error {
	if c.Version != CurrentVersion {
		return fmt.Errorf("unsupported workspace schema version %d", c.Version)
	}
	if !workspaceNamePattern.MatchString(c.Workspace.Name) {
		return fmt.Errorf("invalid workspace name %q", c.Workspace.Name)
	}
	if c.Project.Root == "" {
		return fmt.Errorf("project root is required")
	}
	return nil
}
