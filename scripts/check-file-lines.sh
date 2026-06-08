#!/usr/bin/env bash
set -euo pipefail

limit="${MAX_FILE_LINES:-700}"
status=0

is_code_file() {
  case "$1" in
    *.go|*.rs|*.py|*.js|*.jsx|*.ts|*.tsx|*.java|*.kt|*.kts|*.c|*.h|*.cc|*.cpp|*.hpp|*.cs|*.sh|*.bash|*.zsh|*.ps1|*.sql|*.toml|*.yaml|*.yml|*.json)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

is_exempt_path() {
  case "$1" in
    vendor/*|third_party/*|node_modules/*|dist/*|build/*|coverage/*)
      return 0
      ;;
    *.gen.*|*.generated.*|*.lock|package-lock.json|pnpm-lock.yaml|yarn.lock)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

while IFS= read -r file; do
  if is_exempt_path "$file" || ! is_code_file "$file"; then
    continue
  fi

  lines="$(wc -l < "$file" | tr -d ' ')"
  if [ "$lines" -gt "$limit" ]; then
    printf '%s has %s lines; limit is %s\n' "$file" "$lines" "$limit" >&2
    status=1
  fi
done < <(git ls-files --cached --others --exclude-standard)

exit "$status"
