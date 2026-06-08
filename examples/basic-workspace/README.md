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

Ask for a read-only restore plan for a historical session:

```bash
adp sessions restore-plan <session-id>
```

`restore-plan` prints a suggested `adp run ...` command when enough non-sensitive invocation data is available. It does not execute the command, create a runtime workspace, mutate task state, append new events, write to the real project root, or resume a provider-native conversation. See [../../docs/session-restore.md](../../docs/session-restore.md).

## Task-Bound Fake Agent Workflow

Use a fake local agent when you want to validate runtime history and restore-plan guidance without depending on a real provider CLI:

```bash
fake_bin="$(mktemp -d)"
cat > "${fake_bin}/codex" <<'SH'
#!/usr/bin/env sh
printf 'fake codex received: %s\n' "$*"
SH
chmod +x "${fake_bin}/codex"
PATH="${fake_bin}:${PATH}"
```

Create a task, bind one runtime session to it, and pass local agent arguments after `--`:

```bash
TASK_ID=$(adp tasks add --workspace my-workspace --priority high --phase p4-session-restore "Exercise restore-plan guidance" | sed -n 's/^task \(task-[^ ]*\) added$/\1/p')
adp run codex --workspace my-workspace --task "$TASK_ID" -- --example-smoke
```

Inspect the task-bound evidence:

```bash
adp events list --workspace my-workspace --task "$TASK_ID"
adp sessions list --workspace my-workspace --agent codex --task "$TASK_ID"
adp sessions show <session-id>
adp sessions restore-plan <session-id>
```

If you manually run the suggested command, ADP starts a new local run with a new session ID. The command is guidance for a new run, not automatic replay and not provider-native conversation resume.

ADP keeps runtime state local. This example does not require a web service, hosted control plane, or SaaS account.
