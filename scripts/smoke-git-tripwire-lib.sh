#!/usr/bin/env bash

setup_git_tripwire() {
  local fake_bin="$1"
  local log_path="$2"
  local real_git

  real_git=$(command -v git || true)
  if [ -z "$real_git" ]; then
    fail "Git is required for smoke Git tripwire"
  fi

  export ADP_SMOKE_REAL_GIT="$real_git"
  export ADP_SMOKE_GIT_TRIPWIRE_LOG="$log_path"

  cat > "$fake_bin/git" <<'EOF'
#!/usr/bin/env sh
set -eu

: "${ADP_SMOKE_REAL_GIT:?}"
: "${ADP_SMOKE_GIT_TRIPWIRE_LOG:?}"

for arg do
  case "$arg" in
    add|am|apply|checkout|cherry-pick|clean|clone|commit|fetch|filter-branch|gc|init|maintenance|merge|mv|pull|push|rebase|reset|restore|revert|rm|stash|switch|update-index|update-ref|ls-remote)
      printf '%s\n' "$*" >> "$ADP_SMOKE_GIT_TRIPWIRE_LOG"
      printf 'fake git blocked smoke side-effect command: %s\n' "$*" >&2
      exit 97
      ;;
  esac
done

exec "$ADP_SMOKE_REAL_GIT" "$@"
EOF
  chmod 755 "$fake_bin/git"
  reset_git_tripwire
}

reset_git_tripwire() {
  : "${ADP_SMOKE_GIT_TRIPWIRE_LOG:?}"
  : > "$ADP_SMOKE_GIT_TRIPWIRE_LOG"
}

assert_no_git_side_effects() {
  local label="$1"

  : "${ADP_SMOKE_GIT_TRIPWIRE_LOG:?}"
  if [ -s "$ADP_SMOKE_GIT_TRIPWIRE_LOG" ]; then
    printf '%s\n' "Git side-effect command log:" >&2
    cat "$ADP_SMOKE_GIT_TRIPWIRE_LOG" >&2
    fail "$label invoked a Git side-effect command"
  fi
}
