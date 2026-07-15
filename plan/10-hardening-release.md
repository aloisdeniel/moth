# Milestone 10 — Hardening & Release (v1.0)

## Goal

Turn a feature-complete moth into something you'd confidently point real apps at: security hardening, operability, backups, documentation, and a repeatable release pipeline producing signed binaries for all platforms.

## Deliverables

### Security hardening

- **Rate limiting, persistent & complete**: replace the milestone-02 in-memory buckets with SQLite-backed limits surviving restarts; one shared limiter behind both a gRPC interceptor (SignIn, SignUp, RequestPasswordReset, ExchangeOAuthCode, ...) and HTTP middleware (OAuth redirects, hosted pages, pub API); per-IP + per-account + per-project tiers; `RESOURCE_EXHAUSTED` with a `google.rpc.RetryInfo` detail on RPCs, `Retry-After` headers on HTTP.
- **Audit log**: append-only `audit_log` for every admin action — SPA session or CLI personal access token, attributed to the specific credential (who, what, target, before/after summary, coarse IP) and security-relevant user events (family revocation on refresh reuse, provider config change). Viewer tab in admin with filters; export.
- **Headers & transport**: strict security headers on admin + hosted pages (CSP without `unsafe-inline`, HSTS when https); optional built-in ACME/Let's Encrypt (`--acme-domain`) so a bare VPS gets TLS from the single binary; documented reverse-proxy mode (`--trusted-proxies` for correct client IPs; guide covers HTTP/2 end-to-end configuration for Caddy/nginx/Traefik, which native gRPC requires — gRPC-Web and Connect fall back to HTTP/1.1 fine).
- **Secrets at rest**: the milestone-01 instance master key extended to encrypt Apple private keys and SMTP passwords (project signing keys already encrypted under it since 01); secret-key hashing already in place from 01.
- **Graceful signing-key rotation**: alongside the milestone-03 hard reset (immediate invalidation), an admin-triggered *rotate* — new key signs from now on, the old public key stays in the project JWKS for a configurable grace period (default: access-token TTL + clock skew) so in-flight tokens expire naturally and no user is signed out; expired keys pruned automatically, both actions audit-logged.
- **Dependency & code audit**: `govulncheck` + `gosec` in CI; run the repo `/security-review` flow on the full codebase; fix findings.
- **Abuse controls**: signup email-domain allow/block lists per project; optional CAPTCHA hook (pluggable verification URL) left documented but off by default.

### Operability

- `moth backup [--to path]` — online SQLite backup (VACUUM INTO) + uploads + keys into one archive; `moth restore`; documented cron example. Scheduled automatic backups to a local path as a config option.
- Structured `slog` JSON logs (opt-in), `/metrics` Prometheus endpoint fed by a metrics interceptor (per-RPC counts/latencies/status codes, auth attempts, event-buffer drops, rollup runs), `GET /healthz` and the gRPC health service extended with db/disk checks. Server reflection off by default in production (`--reflection` to enable).
- Graceful shutdown, connection draining, and a startup self-check (writable data dir, clock sanity, key integrity) with actionable error messages.
- Data lifecycle: `moth project export` (users JSON for migration off moth — no lock-in) and `moth project import` — users JSON *with foreign password hashes* (bcrypt/scrypt/argon2, algorithm declared per user, verified with the original algorithm then transparently rehashed to argon2id on first login) so teams can migrate from Firebase/Auth0/Supabase without forcing a password reset. GDPR-style single-user delete already covered in 02/03, cascades verified by tests.

### Release engineering

- GoReleaser: darwin/linux/windows × amd64/arm64 archives, Homebrew tap, Docker image (scratch-based, ~15 MB), checksums + cosign signatures.
- Version-stamped builds; `moth self-update` deliberately **not** included (predictability) — update instructions in docs instead.
- SDK release coupling: pub tarball version pinned to binary version in the release pipeline; `CHANGELOG.md` generated from conventional commits.
- Load sanity check: scripted `ghz` run (gRPC load tool) demonstrating a stated baseline (e.g. 200 SignIn/s on a small VPS) documented honestly in the README.

### Documentation

- Complete the docs content tree started in milestone 09 (single source: public website + embedded `/docs` in the binary, version-matched): finish the deployment guide (systemd unit, Docker compose, reverse proxy, ACME), provider setup guides (moved from admin inline help, kept in sync), API reference generated from the `.proto` files (`protoc-gen-doc` / `buf` docs) with published proto files for third-party client generation, CLI reference from the milestone-08 command definitions, SDK reference, backup/restore, threat-model summary. The `/docs` embedding pipeline (proven on a subset in 09) ships here.
- README rewrite: what/why, 4-step quick start, screenshot of admin + themed login, link to the website.
- Launch checklist: v1.0 tag triggers the website's versioned deploy (09) so site, docs, binary, and SDK version land together.

## Acceptance criteria

- Clean-machine drill: VPS with nothing installed → download binary → `moth serve --acme-domain auth.example.com` → full product tour (admin, project, Flutter example login) using only the docs.
- Backup taken under write load restores to a working instance (test).
- `govulncheck`/`gosec`/security review: zero unaddressed high findings.
- `ghz` baseline met; no goroutine/file-descriptor leaks over a 24 h soak with synthetic traffic.
- Tagged `v1.0.0` release with signed artifacts for all six platform targets; Docker image runs with a mounted volume.

## Post-v1 backlog (parking lot)

Magic links & passkeys (WebAuthn), MFA (TOTP), anonymous/guest accounts with later linking, more OAuth providers, webhooks (`user.created`...), Terraform/Pulumi provider over `moth.admin.v1`, breached-password checks (HIBP k-anonymity), per-project email template/copy customization, organizations/teams within a project, native Swift/Kotlin SDKs, S3 backup target, multi-admin roles/permissions, i18n for login screens and emails.
