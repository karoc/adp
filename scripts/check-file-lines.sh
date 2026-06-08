#!/usr/bin/env bash
set -euo pipefail

mode="enforce"
limit="${MAX_FILE_LINES:-700}"
warn_limit="${LINE_PRESSURE_WARN_LINES:-600}"
status=0

usage() {
  cat <<'USAGE'
Usage:
  scripts/check-file-lines.sh [--audit]

Modes:
  default   Fail when a hand-written code file exceeds MAX_FILE_LINES.
  --audit   Report files at or above LINE_PRESSURE_WARN_LINES and exit zero.

Environment:
  MAX_FILE_LINES              Hard limit for default mode. Default: 700.
  LINE_PRESSURE_WARN_LINES    Warning threshold for --audit. Default: 600.
USAGE
}

is_positive_integer() {
  case "$1" in
    ''|*[!0-9]*)
      return 1
      ;;
    0)
      return 1
      ;;
    *)
      return 0
      ;;
  esac
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --audit)
      mode="audit"
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      printf 'unknown argument: %s\n' "$1" >&2
      usage >&2
      exit 2
      ;;
  esac
  shift
done

if ! is_positive_integer "$limit"; then
  printf 'MAX_FILE_LINES must be a positive integer: %s\n' "$limit" >&2
  exit 2
fi

if ! is_positive_integer "$warn_limit"; then
  printf 'LINE_PRESSURE_WARN_LINES must be a positive integer: %s\n' "$warn_limit" >&2
  exit 2
fi

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
  if [ "$mode" = "audit" ]; then
    if [ "$lines" -ge "$warn_limit" ]; then
      printf '%s has %s lines; warning threshold is %s; hard limit is %s\n' "$file" "$lines" "$warn_limit" "$limit"
      status=1
    fi
    continue
  fi

  if [ "$lines" -gt "$limit" ]; then
    printf '%s has %s lines; limit is %s\n' "$file" "$lines" "$limit" >&2
    status=1
  fi
done < <(git ls-files --cached --others --exclude-standard)

if [ "$mode" = "audit" ]; then
  if [ "$status" -eq 0 ]; then
    printf 'no code files at or above warning threshold %s lines; hard limit is %s\n' "$warn_limit" "$limit"
  fi
  exit 0
fi

exit "$status"
