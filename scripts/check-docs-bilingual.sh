#!/usr/bin/env bash
set -euo pipefail

status=0
tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/adp-docs-bilingual.XXXXXX")

cleanup() {
  rm -rf "$tmp_dir"
}
trap cleanup EXIT INT TERM

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

should_compare_command_refs() {
  case "$1" in
    README.md|docs/*.md|examples/*.md|examples/*/*.md|examples/*/*/*.md)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

extract_command_refs() {
  local file="$1"

  awk '
    function is_command_ref(value) {
      return value ~ /^(adp([[:space:]]|$)|go run \.\/cmd\/adp([[:space:]]|$)|\.\/bin\/adp([[:space:]]|$)|scripts\/[A-Za-z0-9_.-]+|go test|go vet|git diff --check|ADP_SMOKE_|GOOS=|go build|go install|dist\/adp([[:space:]]|$)|export ADP_HOME|export ADP_RUNTIME_DIR|mkdir -p|cp -R|TASK_ID=|cd .+adp)/
    }

    /^[[:space:]]*```/ {
      in_code = !in_code
      next
    }

    in_code {
      line = $0
      sub(/^[[:space:]]+/, "", line)
      if (is_command_ref(line)) {
        print line
      }
      next
    }

    {
      text = $0
      while (match(text, /`[^`]+`/)) {
        span = substr(text, RSTART + 1, RLENGTH - 2)
        if (is_command_ref(span)) {
          print span
        }
        text = substr(text, RSTART + RLENGTH)
      }
    }
  ' "$file" | sort -u
}

compare_command_refs() {
  local english="$1"
  local chinese="$2"
  local safe_name
  local english_refs
  local chinese_refs
  local diff_output

  safe_name=$(printf '%s' "$english" | tr '/ ' '__')
  english_refs="$tmp_dir/${safe_name}.en"
  chinese_refs="$tmp_dir/${safe_name}.zh"
  diff_output="$tmp_dir/${safe_name}.diff"

  extract_command_refs "$english" > "$english_refs"
  extract_command_refs "$chinese" > "$chinese_refs"

  if ! diff -u "$english_refs" "$chinese_refs" > "$diff_output"; then
    printf 'command references differ between %s and %s\n' "$english" "$chinese" >&2
    cat "$diff_output" >&2
    status=1
  fi
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
      elif should_compare_command_refs "$file"; then
        compare_command_refs "$file" "$chinese"
      fi
      ;;
  esac
done < <(git ls-files --cached --others --exclude-standard '*.md')

exit "$status"
