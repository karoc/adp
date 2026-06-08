# Basic Workspace Example

This directory is a copyable local workspace configuration for ADP. It is not a fixed project that can run unchanged on every machine.

The sample `workspace.yaml` uses:

```yaml
workspace:
  name: game-a

project:
  root: /srv/game-a
```

Before using it, replace `workspace.name` with your workspace name and replace `project.root: /srv/game-a` with the absolute path to a real project on your machine.

## Contents

- `prompts/`: reusable prompt files referenced by the workspace.
- `memory/`: shared local memory files for agent context.
- `mcp/`: MCP configuration used by the workspace.
- `profiles/`: agent profile files, including Codex and Claude examples.
- `workspace.yaml`: the workspace manifest ADP reads from `$ADP_HOME/workspaces/<name>`.

## Use It Locally

Set `ADP_HOME` if you do not already have one:

```bash
export ADP_HOME="${HOME}/.adp"
```

Copy the example into a workspace directory:

```bash
mkdir -p "${ADP_HOME}/workspaces"
cp -R examples/basic-workspace "${ADP_HOME}/workspaces/my-workspace"
```

Edit the copied manifest:

```bash
$EDITOR "${ADP_HOME}/workspaces/my-workspace/workspace.yaml"
```

Update these fields:

```yaml
workspace:
  name: my-workspace

project:
  root: /absolute/path/to/your/project
```

Run diagnostics before launching an agent:

```bash
adp workspace doctor my-workspace
```

Print shell environment hints for the workspace:

```bash
adp env my-workspace --cd
```

Run agents through ADP so they start inside the isolated runtime workspace:

```bash
adp run codex --workspace my-workspace
adp run claude --workspace my-workspace
```

Inspect local runtime history:

```bash
adp events list
adp sessions list --workspace my-workspace
adp sessions show <session-id>
```

ADP keeps runtime state local. This example does not require a web service, hosted control plane, or SaaS account.
