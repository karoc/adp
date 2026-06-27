# Shared helpers for smoke scripts that need cross-platform path handling.
# On Windows (Git Bash/MSYS2), mktemp returns MSYS2-style paths (/tmp/...)
# while the Go binary outputs Windows-native paths (C:\...\Temp\...).
# These helpers bridge that gap without affecting POSIX behavior.

# smoke_normalize_path converts backslashes to forward slashes.
# This is a no-op on POSIX where paths already use forward slashes.
smoke_normalize_path() {
  printf '%s\n' "$1" | tr '\\' '/'
}

# smoke_native_path converts an MSYS2/POSIX path to a native Windows path
# when cygpath is available. Uses mixed mode (-m) to produce forward-slash
# paths with drive letters (e.g. C:/Users/...) that both bash and the Go
# binary accept. On POSIX it returns the path unchanged.
smoke_native_path() {
  if command -v cygpath >/dev/null 2>&1; then
    cygpath -m "$1"
  else
    printf '%s\n' "$1"
  fi
}

# smoke_native_tmpdir creates a temp directory and returns its native path.
smoke_native_tmpdir() {
  local template="${1:-${TMPDIR:-/tmp}/adp-smoke.XXXXXX}"
  local dir
  dir=$(mktemp -d "$template")
  smoke_native_path "$dir"
}

# assert_contains checks that $needle appears in $output, normalizing
# backslashes to forward slashes in both arguments so that path format
# differences across platforms do not cause false failures.
assert_contains() {
  local output
  local needle
  output=$(smoke_normalize_path "$1")
  needle=$(smoke_normalize_path "$2")
  local label="$3"

  case "$output" in
    *"$needle"*) ;;
    *)
      printf '%s\n' "$1" >&2
      fail "$label missing expected text: $2"
      ;;
  esac
}
