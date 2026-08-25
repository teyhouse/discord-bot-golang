# Security Policy

## Supported versions

This project is a template intended to be forked; only the latest commit on
the default branch is supported. Please always run the most recent version.

## Reporting a vulnerability

Please do **not** open a public issue for security problems.

Report vulnerabilities privately via [GitHub Security Advisories]
("Security" tab → "Report a vulnerability"), or contact the maintainer
directly if you prefer.

You can expect an initial response within 7 days.

## Handling of secrets

- Bot tokens belong in `.env` (gitignored) or the process environment — never
  in code, never committed, never baked into container images.
- If you believe you have found a leaked credential in this repository,
  report it immediately as above; treat any token in your own forks as
  compromised and rotate it in the
  [Discord developer portal](https://discord.com/developers/applications).
