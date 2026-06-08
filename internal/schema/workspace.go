package schema

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"
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
	if err := ValidateWorkspaceName(c.Workspace.Name); err != nil {
		return err
	}
	if c.Project.Root == "" {
		return fmt.Errorf("project root is required")
	}
	if !filepath.IsAbs(c.Project.Root) {
		return fmt.Errorf("project root must be absolute: %s", c.Project.Root)
	}
	return nil
}

func ValidateWorkspaceName(name string) error {
	if !workspaceNamePattern.MatchString(name) {
		return fmt.Errorf("invalid workspace name %q", name)
	}
	return nil
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read workspace config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("decode workspace config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func SaveConfig(path string, cfg *Config) error {
	if cfg == nil {
		return errors.New("workspace config is nil")
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encode workspace config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write workspace config: %w", err)
	}
	return nil
}
