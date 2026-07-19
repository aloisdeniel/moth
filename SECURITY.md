# Security Policy

moth is an authentication server, so we take security reports seriously and
appreciate responsible disclosure.

## Reporting a vulnerability

**Please do not open a public issue for a security vulnerability.**

Report it privately through GitHub's
[private vulnerability reporting](https://github.com/aloisdeniel/moth/security/advisories/new)
for this repository. If you cannot use that channel, email
**alois.deniel@gmail.com** with the details.

Please include:

- a description of the vulnerability and its impact,
- the moth version or commit affected,
- steps to reproduce (a proof of concept if you have one), and
- any suggested remediation.

## What to expect

- **Acknowledgement** within 3 business days.
- An initial assessment and a remediation plan within 10 business days.
- We will keep you updated on progress and coordinate a disclosure date. We
  aim to release a fix within 90 days and will credit you in the advisory
  unless you prefer to remain anonymous.

## Scope

In scope: the moth server and CLI (`cmd/moth`, `internal/`), the published
SDKs (`sdk/`), the admin console, and the hosted verify/reset/confirm-email
pages.

Out of scope: vulnerabilities in third-party dependencies (report those
upstream, though we welcome a heads-up), issues that require a
already-compromised host or master key, and findings against the
documentation website's hosting.

## Supported versions

moth is pre-1.0; security fixes land on `main` and in the next tagged
release. Once 1.0 ships, the latest minor release receives security
updates.
