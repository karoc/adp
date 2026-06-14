# ADP Examples Templates

This directory contains reusable components shared across all examples in the `examples/` directory.

## Purpose

Templates provide a consistent foundation for domain-specific examples, reducing duplication and ensuring best practices are applied uniformly.

## Contents

- `workspace-base.yaml` - Base workspace configuration with common settings
- `profiles/` - Standard agent profile definitions
- `prompts/` - Reusable prompt templates
- `mcp/` - MCP server configuration templates

## Usage

Examples extend these templates by including them in their workspace configuration:

```yaml
# examples/game-development/workspace.yaml
extends: ../_templates/workspace-base.yaml

workspace:
  name: game-dev

project:
  root: ./project
```

This inheritance model allows examples to:
- Focus on domain-specific configuration
- Share common settings automatically
- Maintain consistency across all examples

## Template Structure

### workspace-base.yaml

Provides foundation settings:
- Memory enablement
- Coding style rules
- MCP integration
- Basic project structure

### profiles/

Standard agent profiles:
- `codex.yaml` - Development agent focused on implementation
- `claude.yaml` - Architect agent for design and review
- `architect.yaml` - Senior architect for system design

### prompts/

Reusable prompt templates:
- `coding-style.md` - Language-agnostic coding guidelines
- `testing-requirements.md` - Test coverage expectations

### mcp/

MCP server configurations:
- `config.yaml` - Common MCP servers for development

## Customization

Examples can override template settings:

```yaml
extends: ../_templates/workspace-base.yaml

workspace:
  name: my-example

# Override memory settings
memory:
  enabled: true
  shared: memory/domain-specific.md

# Add domain-specific rules
rules:
  coding_style: strict
  domain_rules: prompts/domain-rules.md
```

## Maintenance

When updating templates:
1. Consider impact on all examples
2. Test with at least one example
3. Update this README if structure changes
4. Maintain backward compatibility when possible
