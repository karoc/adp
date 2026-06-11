write_fake_agent() {
  local path="$1"
  local agent="$2"
  local profile="$3"
  local instructions="$4"
  local config="$5"
  local config_kind="$6"
  local payload="$7"

  cat > "$path" <<EOF
#!/usr/bin/env sh
set -eu

printf 'fake-$agent cwd=%s args=%s\n' "\$(pwd)" "\$*"

require_runtime_text() {
  file=\$1
  needle=\$2
  label=\$3
  if ! grep -F -q "\$needle" "\$file"; then
    printf 'fake-$agent missing %s in %s: %s\n' "\$label" "\$file" "\$needle" >&2
    exit 97
  fi
}

assert_git_env_unset() {
  name=\$1
  if env | grep -q "^\$name="; then
    value=\$(env | sed -n "s/^\$name=//p" | head -n 1)
    printf '%s leaked into fake-$agent environment: %s\n' "\$name" "\$value" >&2
    exit 96
  fi
}

assert_git_env_unset GIT_ALTERNATE_OBJECT_DIRECTORIES
assert_git_env_unset GIT_COMMON_DIR
assert_git_env_unset GIT_DIR
assert_git_env_unset GIT_INDEX_FILE
assert_git_env_unset GIT_NAMESPACE
assert_git_env_unset GIT_OBJECT_DIRECTORY
assert_git_env_unset GIT_WORK_TREE

test "\${ADP_AGENT:-}" = "$agent"
test "\${ADP_WORKSPACE:-}" = "context-a"
test "\${ADP_HOME:-}" = "\$ADP_EXPECT_ADP_HOME"
test "\${ADP_PROJECT_ROOT:-}" = "\$ADP_EXPECT_PROJECT_ROOT"
test "\${ADP_GIT_ROOT:-}" = "\$ADP_EXPECT_GIT_ROOT"
test "\${ADP_PROFILE:-}" = "$profile"
test "\${ADP_TASK_ID:-}" = "\$ADP_EXPECT_TASK_ID"
test "\${ADP_TASK_TITLE:-}" = "\$ADP_EXPECT_TASK_TITLE"
test "\${ADP_TASK_STATUS:-}" = "\$ADP_EXPECT_TASK_STATUS"
test "\${ADP_TASK_PRIORITY:-}" = "\$ADP_EXPECT_TASK_PRIORITY"
test "\${ADP_TASK_PHASE:-}" = "\$ADP_EXPECT_TASK_PHASE"
if [ -n "\${ADP_CLI:-}" ]; then
  test "\$ADP_CLI" = "\$ADP_EXPECT_ADP_CLI"
  test -x "\$ADP_CLI"
fi
test -n "\${ADP_SESSION_ID:-}"
test -n "\${ADP_RUNTIME_ROOT:-}"
test "\$(pwd)" = "\$ADP_RUNTIME_ROOT"
case ":\${GIT_CEILING_DIRECTORIES:-}:" in
  *":\$ADP_RUNTIME_ROOT:"*) ;;
  *)
    printf 'GIT_CEILING_DIRECTORIES missing runtime root: %s\n' "\${GIT_CEILING_DIRECTORIES:-}" >&2
    exit 96
    ;;
esac

test -f "\$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
grep -F -q "version: 1" "\$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
grep -F -q "session_id: \$ADP_SESSION_ID" "\$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
grep -F -q "workspace: context-a" "\$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
grep -F -q "task_id: \$ADP_EXPECT_TASK_ID" "\$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
grep -F -q "task_title: \$ADP_EXPECT_TASK_TITLE" "\$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
grep -F -q "project_root: \$ADP_EXPECT_PROJECT_ROOT" "\$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
grep -F -q "git_root: \$ADP_EXPECT_GIT_ROOT" "\$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
grep -F -q "git_metadata_skipped: true" "\$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
grep -F -q "runtime_root: \$ADP_RUNTIME_ROOT" "\$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
grep -F -q "keep: false" "\$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
grep -F -q "generated_by: adp" "\$ADP_RUNTIME_ROOT/.adp-runtime.yaml"
test ! -e "\$ADP_RUNTIME_ROOT/.git"
if git -C "\$ADP_RUNTIME_ROOT" rev-parse --show-toplevel >/dev/null 2>&1; then
  printf 'Git root unexpectedly resolved from ADP runtime root\n' >&2
  exit 96
fi
if git -C "\$ADP_RUNTIME_ROOT" status --short --branch >/dev/null 2>&1; then
  printf 'git status unexpectedly succeeded inside ADP runtime root\n' >&2
  exit 96
fi
project_git_root=\$(git -C "\$ADP_PROJECT_ROOT" rev-parse --show-toplevel)
test "\$project_git_root" = "\$ADP_EXPECT_GIT_ROOT"
git -C "\$ADP_PROJECT_ROOT" status --short --branch >/dev/null
test -L "\$ADP_RUNTIME_ROOT/pkg"
runtime_subpath_git_root=\$(git -C "\$ADP_RUNTIME_ROOT/pkg" rev-parse --show-toplevel)
test "\$runtime_subpath_git_root" = "\$ADP_EXPECT_GIT_ROOT"

test -f "$instructions"
grep -F -q "# ADP Runtime Instructions for" "$instructions"
grep -F -q -- "- Name: context-a" "$instructions"
grep -F -q -- "- Agent: $agent" "$instructions"
grep -F -q -- "- Profile: $profile" "$instructions"
grep -F -q -- "- ID: \$ADP_EXPECT_TASK_ID" "$instructions"
grep -F -q -- "- Title: \$ADP_EXPECT_TASK_TITLE" "$instructions"
grep -F -q -- "- Status: \$ADP_EXPECT_TASK_STATUS" "$instructions"
grep -F -q -- "- Priority: \$ADP_EXPECT_TASK_PRIORITY" "$instructions"
grep -F -q -- "- Phase: \$ADP_EXPECT_TASK_PHASE" "$instructions"
grep -F -q -- "- Description: \$ADP_EXPECT_TASK_DESCRIPTION" "$instructions"
require_runtime_text "$instructions" "## ADP Planning Contract" "planning contract heading"
require_runtime_text "$instructions" "ADP is the authoritative local planning and progress ledger" "planning source-of-truth contract"
require_runtime_text "$instructions" "tasks take --workspace" "atomic task take command"
require_runtime_text "$instructions" "Inspect stale claims" "stale claim inspection guidance"
require_runtime_text "$instructions" "tasks stale --workspace" "stale claim inspection command"
require_runtime_text "$instructions" "Claim selected work" "selected task claim guidance"
require_runtime_text "$instructions" "tasks claim --workspace" "selected task claim command"
require_runtime_text "$instructions" "Renew this task" "task-bound lease renewal guidance"
require_runtime_text "$instructions" "tasks renew --workspace" "task-bound lease renewal command"
require_runtime_text "$instructions" "ADP_WORKSPACE" "task-bound lease renewal workspace"
require_runtime_text "$instructions" "ADP_TASK_ID" "task-bound lease renewal task id"
require_runtime_text "$instructions" "scratch space only" "provider taskbox boundary"
require_runtime_text "$instructions" "## Tool Taskbox Bridge" "taskbox bridge heading"
require_runtime_text "$instructions" "mirror the active ADP task" "taskbox mirror guidance"
require_runtime_text "$instructions" "do not treat provider-native task state as authoritative" "taskbox authority boundary"
require_runtime_text "$instructions" "Do not sync provider-private todo state back into ADP automatically" "taskbox automatic sync guard"
require_runtime_text "$instructions" "## Tool Plan Mode Bridge" "plan mode bridge heading"
require_runtime_text "$instructions" "proposal view" "plan mode proposal boundary"
require_runtime_text "$instructions" "do not edit project files, complete tasks, accept phases, commit, or push" "plan mode execution guard"
require_runtime_text "$instructions" "plan preview --workspace" "plan mode preview command"
require_runtime_text "$instructions" "plan apply --workspace" "plan mode apply command"
require_runtime_text "$instructions" "not ADP phase acceptance" "plan mode phase boundary"
require_runtime_text "$instructions" "Provider-native plan approval is not ADP phase acceptance" "plan mode phase acceptance guard"
require_runtime_text "$instructions" "## Git Boundary" "git boundary heading"
require_runtime_text "$instructions" "not the authoritative Git working tree" "git worktree boundary"
require_runtime_text "$instructions" "Detected Git worktree root: \$ADP_EXPECT_GIT_ROOT" "detected git root guidance"
require_runtime_text "$instructions" "differ. This usually means" "project/git root distinction"
require_runtime_text "$instructions" "configured project root is a subdirectory inside a larger Git worktree" "nested project guidance"
require_runtime_text "$instructions" "Staging and committing still use the repository index for the whole worktree" "whole-worktree index guidance"
require_runtime_text "$instructions" 'git -C "\$ADP_PROJECT_ROOT" status --short --branch' "project-root git status guidance"
require_runtime_text "$instructions" 'git -C "\$ADP_PROJECT_ROOT" diff' "project-root git diff guidance"
require_runtime_text "$instructions" 'git -C "\$ADP_PROJECT_ROOT" diff --cached' "project-root git staged diff guidance"
require_runtime_text "$instructions" 'git -C "\$ADP_GIT_ROOT" status --short --branch' "git-root status guidance"
require_runtime_text "$instructions" 'git -C "\$ADP_PROJECT_ROOT" ...' "project-root mutation guidance"
require_runtime_text "$instructions" "Real project root: \$ADP_EXPECT_PROJECT_ROOT" "real project root guidance"
if [ -n "\${ADP_CLI:-}" ]; then
  require_runtime_text "$instructions" "ADP_CLI" "ADP CLI hint"
fi
grep -F -q "P35 base prompt marker" "$instructions"
grep -F -q "P35 shared memory marker" "$instructions"
grep -F -q "review_depth: context-audit" "$instructions"
grep -F -q "Servers:" "$instructions"
grep -F -q -- "- github" "$instructions"
grep -F -q -- "- local-tools" "$instructions"
grep -F -q "p35-mcp-config-marker" "$instructions"
grep -F -q "Name: $profile" "$instructions"
grep -F -q "Agent enabled: true" "$instructions"
grep -F -q "Agent command: $agent" "$instructions"
grep -F -q "P35 $profile profile marker" "$instructions"

test -f "$config"
case "$config_kind" in
  toml)
    grep -F -q 'adapter = "$agent"' "$config"
    grep -F -q 'workspace = "context-a"' "$config"
    grep -F -q "project_root = \"\$ADP_EXPECT_PROJECT_ROOT\"" "$config"
    grep -F -q "git_root = \"\$ADP_EXPECT_GIT_ROOT\"" "$config"
    grep -F -q 'profile = "$profile"' "$config"
    grep -F -q 'memory_enabled = true' "$config"
    grep -F -q 'mcp_enabled = true' "$config"
    grep -F -q "task_id = \"\$ADP_EXPECT_TASK_ID\"" "$config"
    grep -F -q "task_title = \"\$ADP_EXPECT_TASK_TITLE\"" "$config"
    grep -F -q "task_status = \"\$ADP_EXPECT_TASK_STATUS\"" "$config"
    grep -F -q "task_priority = \"\$ADP_EXPECT_TASK_PRIORITY\"" "$config"
    grep -F -q "task_phase = \"\$ADP_EXPECT_TASK_PHASE\"" "$config"
    ;;
  json)
    grep -F -q '"adapter": "$agent"' "$config"
    grep -F -q '"workspace": "context-a"' "$config"
    grep -F -q "\"projectRoot\": \"\$ADP_EXPECT_PROJECT_ROOT\"" "$config"
    grep -F -q "\"gitRoot\": \"\$ADP_EXPECT_GIT_ROOT\"" "$config"
    grep -F -q '"profile": "$profile"' "$config"
    grep -F -q '"memoryEnabled": true' "$config"
    grep -F -q '"mcpEnabled": true' "$config"
    grep -F -q "\"id\": \"\$ADP_EXPECT_TASK_ID\"" "$config"
    grep -F -q "\"title\": \"\$ADP_EXPECT_TASK_TITLE\"" "$config"
    grep -F -q "\"status\": \"\$ADP_EXPECT_TASK_STATUS\"" "$config"
    grep -F -q "\"priority\": \"\$ADP_EXPECT_TASK_PRIORITY\"" "$config"
    grep -F -q "\"phase\": \"\$ADP_EXPECT_TASK_PHASE\"" "$config"
    ;;
  *)
    printf 'unknown config kind: %s\n' "$config_kind" >&2
    exit 98
    ;;
esac

test -L go.mod
test -f go.mod
test -L main.go
test -f main.go
test -L pkg
test -f pkg/pkg.go
test "\$#" -eq 1
test "\$1" = "$payload"
EOF
  chmod 755 "$path"
}
