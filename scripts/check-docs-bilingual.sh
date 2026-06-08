#!/usr/bin/env bash
set -euo pipefail

status=0

is_exempt_markdown() {
  case "$1" in
    LICENSE.md|LICENSES/*|vendor/*|third_party/*|node_modules/*)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

while IFS= read -r file; do
  if is_exempt_markdown "$file"; then
    continue
  fi

  case "$file" in
    *.zh-CN.md)
      english="${file%.zh-CN.md}.md"
      if [ ! -f "$english" ]; then
        printf 'missing English default document for %s: expected %s\n' "$file" "$english" >&2
        status=1
      fi
      ;;
    *.md)
      chinese="${file%.md}.zh-CN.md"
      if [ ! -f "$chinese" ]; then
        printf 'missing Simplified Chinese document for %s: expected %s\n' "$file" "$chinese" >&2
        status=1
      fi
      ;;
  esac
done < <(git ls-files '*.md')

exit "$status"
