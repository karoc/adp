# License Policy

Simplified Chinese: [license-policy.zh-CN.md](license-policy.zh-CN.md)

This document explains ADP's licensing and contribution policy for operators and contributors. It is operational guidance, not a replacement for the authoritative license text in [../LICENSE](../LICENSE).

## Public License

ADP uses the [PolyForm Noncommercial License 1.0.0](../LICENSE) for public noncommercial use. The repository is source-available for learning, research, evaluation, and noncommercial open collaboration.

Commercial use is not granted by the public license. Any commercial use requires separate paid authorization from the copyright holder. Public source availability, public forks, preview packages, and noncommercial redistribution do not grant commercial rights.

ADP is not published under an OSI-approved open-source license because the public license restricts commercial use. Use `source-available` or `noncommercial source-available` when describing the public licensing model.

## Required Notices

Redistributed noncommercial copies, forks, source archives, and release packages must preserve:

- `LICENSE`;
- the `Required Notice:` lines in `LICENSE`;
- `COMMERCIAL.md` and `COMMERCIAL.zh-CN.md` when packaging public docs;
- attribution to ADP and the copyright holder;
- this licensing boundary in public documentation when relevant.

The English `LICENSE` file is the authoritative legal text. Translations and summaries are explanatory only.

## Commercial Authorization

Commercial use includes using ADP for paid products, services, consulting, hosting, automation, integration, support, internal business operations, customer delivery, managed services, proprietary integrations, revenue-generating systems, or commercial redistribution.

Commercial authorization is handled separately from the public repository. Contact the repository owner through the GitHub repository for commercial licensing.

## Contributions

Contributions must be compatible with ADP's existing licensing model. By submitting a contribution, contributors confirm that they have the right to contribute it and that it can be distributed as part of ADP under the repository's current public license and commercial authorization model.

Do not contribute code, documentation, generated output, datasets, examples, or dependencies that require ADP to adopt an incompatible license, remove noncommercial restrictions, or grant commercial rights through the public license.

Dependency changes must be checked for compatibility with a noncommercial source-available distribution model before they are accepted.

## Release Packages

Release packages must keep the public license and commercial notice intact and must not imply that a preview binary grants commercial rights. Packages must not include local `$ADP_HOME` state, `$ADP_RUNTIME_DIR` contents, runtime overlays, logs, task state, credentials, `.envrc`, `mvp.md`, or machine-specific shell configuration.

For package details, see [release-packaging.md](release-packaging.md).
