# Engineering Standards

This document defines repository-wide engineering rules for ADP contributors and automation agents.

## File Size Limit

Project code files must stay at or below 700 physical lines.

When a code file would exceed 700 lines, split it before merging. Prefer splitting by stable responsibility boundaries:

- CLI command wiring vs command implementation.
- schema types vs validation logic.
- adapter registry vs concrete adapter implementation.
- runtime orchestration vs overlay materialization.
- runner process handling vs event logging.
- production code vs test helpers.

Allowed exceptions:

- generated files that are not edited by hand;
- vendored third-party code;
- lockfiles and machine-produced metadata;
- license files and long-form documentation.

Any exception for hand-written code needs a short justification in the pull request and should be treated as temporary.

Run the local check before handoff:

```bash
scripts/check-file-lines.sh
```

The check defaults to 700 lines. For local experiments, override it with:

```bash
MAX_FILE_LINES=700 scripts/check-file-lines.sh
```

## Licensing Boundary

ADP uses the PolyForm Noncommercial License 1.0.0 for public noncommercial use.

Do not introduce third-party dependencies whose licenses conflict with a noncommercial source-available distribution model. When adding a dependency, record the reason for the dependency and confirm that its license permits inclusion.

All source files should remain compatible with the repository license. New public-facing files should preserve copyright and required notices where appropriate.
