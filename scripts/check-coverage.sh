#!/usr/bin/env bash
set -euo pipefail

# check-coverage.sh runs the Go test suite with coverage instrumentation and
# fails if total statement coverage falls below COVERAGE_MIN. It is a
# regression guard: the floor is set a little below the current measured
# coverage so ordinary churn does not trip it, but a meaningful drop (e.g. a
# new untested package or a deleted test) does.
#
# Override the floor with COVERAGE_MIN=NN (integer or decimal percent). Set
# COVERAGE_PROFILE to keep the raw profile for inspection.

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

coverage_min="${COVERAGE_MIN:-75}"
profile="${COVERAGE_PROFILE:-$(mktemp "${TMPDIR:-/tmp}/adp-coverage.XXXXXX")}"
cleanup_profile=1
if [ -n "${COVERAGE_PROFILE:-}" ]; then
  cleanup_profile=0
fi
trap '[ "$cleanup_profile" = 1 ] && rm -f "$profile"' EXIT

# Cover every package the test binary can instrument. Packages with no
# statements (e.g. cmd/adp, generated shims) contribute 0 and are reported by
# `go tool cover` but do not distort the weighted total meaningfully. Test
# output is left on stdout so failures stay visible when this replaces the
# plain `go test` step in check-all.sh.
go test -count=1 -covermode=atomic -coverprofile="$profile" ./...

total_line="$(go tool cover -func="$profile" | tail -1)"
# Format: "total:\t(statements)\t92.2%"
total_pct="${total_line##*$'\t'}"
total_pct="${total_pct%\%}"

printf 'total coverage: %s%% (floor: %s%%)\n' "$total_pct" "$coverage_min"

# Compare as floats without relying on bc: awk is always present.
if awk "BEGIN { exit !($total_pct < $coverage_min) }"; then
  printf 'coverage %s%% is below the required floor of %s%%\n' "$total_pct" "$coverage_min" >&2
  printf 'run: go tool cover -func=<profile> to see which packages regressed\n' >&2
  exit 1
fi

printf 'coverage check passed\n'
