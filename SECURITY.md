# Security

Simplified Chinese: [SECURITY.zh-CN.md](SECURITY.zh-CN.md)

ADP is a local CLI and runtime-overlay tool. Security reports should focus on local execution boundaries, project-root cleanliness, credential handling, runtime isolation, package integrity, and unsafe interactions with external agent CLIs.

## Reporting

Please report security issues privately through the GitHub repository owner contact path. Do not open a public issue with exploit details, secrets, private logs, provider tokens, API keys, or proprietary project contents.

Include enough non-sensitive context to reproduce the issue:

- ADP commit or version.
- Operating system and shell.
- The ADP command involved.
- Sanitized workspace configuration.
- Whether real Codex or Claude CLIs were involved.
- Expected behavior and observed behavior.
- Confirmation that credentials, tokens, private prompts, and proprietary code were removed from the report.

## Supported Scope

The main branch and current preview artifacts receive security attention. Older preview artifacts may be superseded quickly; reproduce against the current main branch when practical.

In scope:

- ADP writing runtime or planning files into a real project root unexpectedly.
- Unsafe handling of `$ADP_HOME`, `$ADP_RUNTIME_DIR`, runtime overlays, event logs, sessions, task ledgers, or release packages.
- Accidental capture or display of credentials, tokens, API keys, full environments, provider-private conversation identifiers, or proprietary project contents.
- Default tests or release gates contacting real providers without explicit opt-in.
- Package contents that include local state, logs, credentials, machine-specific shell configuration, or runtime overlays.

Out of scope for ADP ownership:

- Provider account compromise.
- Provider model behavior, quota, network availability, or billing.
- Vulnerabilities in Codex, Claude, shells, Git, Go, the operating system, or third-party tools unless ADP's integration creates the exposure.
- Public project files intentionally read by a user-launched local agent.

## Operator Safety

Default smoke tests are provider-free and use fake local agents. Real external CLI checks must be explicitly enabled with documented opt-in environment variables. ADP can validate local launch wiring and evidence collection, but it cannot guarantee provider credentials, model access, quota, network behavior, or interactive provider behavior.

Do not paste secrets into issues, task descriptions, progress reports, prompts, generated runtime files, or public evidence notes.
