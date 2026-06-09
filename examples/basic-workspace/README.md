# Basic Workspace Example

This directory is a copyable local workspace configuration for ADP. Copy it into
`$ADP_HOME/workspaces/<name>`, point `project.root` at a real local project, and
validate it before launching any agent.

The checked-in `workspace.yaml` uses:

```yaml
workspace:
  name: game-a

project:
  root: /srv/game-a
```

Before using the example, replace `workspace.name` with your workspace name and
replace `project.root: /srv/game-a` with an absolute path on your machine.

## Contents

- `prompts/`: reusable prompt files referenced by the workspace.
- `memory/`: shared local memory files for agent context.
- `mcp/`: MCP configuration used by the workspace.
- `profiles/`: agent profile files, including Codex and Claude examples.
- `workspace.yaml`: the workspace manifest ADP reads from `$ADP_HOME/workspaces/<name>`.

## Quick Local Rehearsal

Run these commands from the ADP repository root after installing or building
`adp`. The rehearsal uses temporary directories so it does not depend on existing
operator state.

Prepare isolated ADP state and a tiny local project:

```bash
export ADP_HOME="$(mktemp -d)"
export ADP_RUNTIME_DIR="$(mktemp -d)"
project_root="$(mktemp -d)"
printf 'module example.com/adp-basic-workspace\n' > "${project_root}/go.mod"
printf 'package main\n' > "${project_root}/main.go"
adp init
```

Copy the example into `$ADP_HOME`:

```bash
mkdir -p "${ADP_HOME}/workspaces"
cp -R examples/basic-workspace "${ADP_HOME}/workspaces/my-workspace"
```

Edit the copied manifest:

```bash
$EDITOR "${ADP_HOME}/workspaces/my-workspace/workspace.yaml"
```

Set the workspace name and project root:

```yaml
workspace:
  name: my-workspace

project:
  root: /absolute/path/from/project_root
```

Use the absolute path stored in the `project_root` shell variable. Do not point
`project.root` at `$ADP_HOME`, `$ADP_RUNTIME_DIR`, or the copied workspace
directory.

Validate the copied workspace before any run:

```bash
adp workspace doctor my-workspace
adp workspace show my-workspace
adp env my-workspace --cd
```

`adp env my-workspace --cd` creates or locates a runtime overlay under
`$ADP_RUNTIME_DIR` and prints shell exports plus a `cd` command for that overlay.
The real project root should stay limited to the files you created or already
own.

## Provider-Free Run

Use a fake local `codex` command for onboarding. This proves ADP can launch an
agent through the copied workspace without requiring a real provider CLI,
account, network access, or hosted service.

```bash
fake_bin="$(mktemp -d)"
cat > "${fake_bin}/codex" <<'SH'
#!/usr/bin/env sh
printf 'fake codex received: %s\n' "$*"
SH
chmod +x "${fake_bin}/codex"
export PATH="${fake_bin}:${PATH}"
adp run codex --workspace my-workspace -- --example-smoke
```

Inspect local runtime evidence:

```bash
adp events list --workspace my-workspace
adp sessions list --workspace my-workspace
adp sessions show <session-id>
adp sessions restore-plan <session-id>
```

`restore-plan` prints a suggested `adp run ...` command when enough
non-sensitive invocation data is available. It does not execute the command,
create a runtime workspace, mutate task state, append new events, write to the
real project root, or resume a provider-native conversation. See
[../../docs/session-restore.md](../../docs/session-restore.md).

After the run, the real project root should not contain ADP-generated files such
as `AGENTS.md`, `CLAUDE.md`, `.codex`, `.claude`, `.adp-runtime.yaml`,
`planning`, `tasks.yaml`, `phases.yaml`, or `progress.jsonl`. Runtime overlays
belong under `$ADP_RUNTIME_DIR`; workspace configuration and local planning
state belong under `$ADP_HOME`.

## Optional Real Agent Runs

Run real agents through ADP only after the corresponding external CLI is
installed and authenticated:

```bash
adp run codex --workspace my-workspace
adp run claude --workspace my-workspace
```

Missing real Codex or Claude CLIs are not an onboarding failure. Use the
provider-free run above for deterministic local validation.

ADP keeps runtime state local. This example does not require a web service,
hosted control plane, SaaS account, cloud sync, or automatic Git behavior.
