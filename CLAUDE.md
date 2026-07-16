# moth

One-binary authentication server for mobile apps: one moth instance hosts a
whole portfolio of independent apps ("projects"), each a sealed tenant with
its own users, ES256 signing keypair and configuration. The plan lives in
`plan/` (start with `plan/00-overview.md`); milestones are implemented in
order.

## Commands

- `make build` — build `bin/moth` (set `VERSION=` for releases); embeds the
  committed SPA build in `internal/server/web/dist`
- `make test` — `go test ./...`
- `make lint` — `golangci-lint run` + `buf lint`
- `make proto` — `buf generate` (regenerates `gen/` Go + `web/admin/src/gen`
  TypeScript; needs `npm ci` in `web/admin` once; commit the results)
- `make web` — rebuild the embedded admin SPA into `internal/server/web/dist`
  (commit the result; CI fails when it is stale)
- `make dev` — Go server on :8080 + Vite dev server on :5173 with an RPC
  proxy (open `http://localhost:5173/admin/`); frontend edits hot-reload
- `make run` — build and start `moth serve`
- `make cross` — check the four release targets compile
- `npx playwright test` (in `web/admin`, after `make build`) — browser smoke
  test against the real binary

## Layout

```
cmd/moth/            main + cobra commands (serve, admin create, version)
proto/moth/          protobuf definitions (moth.admin.v1, moth.auth.v1,
                     moth.server.v1); embed.go serves the sources at /protos/
gen/                 buf-generated Go code — committed, never hand-edited
internal/config/     flags > MOTH_* env > moth.toml > defaults resolution
                     (incl. [smtp] section)
internal/store/      SQLite (modernc.org/sqlite), embedded migrations,
                     hand-written SQL behind per-domain interfaces (no ORM)
internal/keys/       master key (encrypts at rest), per-project ES256
                     keypairs, JWKS building
internal/jwt/        minimal ES256 JWS sign/verify for moth access tokens
                     (third-party JOSE libs verify them via the JWKS)
internal/mail/       Mailer interface: SMTP + console transports, embedded
                     email templates
internal/password/   argon2id hashing
internal/token/      random API keys / session tokens + SHA-256 hashing
internal/ratelimit/  in-memory token buckets for credential-facing RPCs
internal/server/     handler assembly, interceptors, plain-HTTP surfaces,
                     hosted verify/reset/confirm-email pages
internal/server/rpc/admin/     moth.admin.v1 handlers + session interceptor
internal/server/rpc/auth/      moth.auth.v1 handlers; publishable-key (pk_)
                               interceptor + rate limiting; ErrorInfo reasons
internal/server/rpc/serverapi/ moth.server.v1 handlers; secret-key (sk_)
                               interceptor
internal/server/web/ hosted-page template + dist/ (committed Vite build of
                     the admin SPA, embedded and served at /admin)
web/admin/           admin SPA source: React + Vite + TS, connect-web +
                     connect-query client generated from the protos
                     (src/gen), DESIGN.md tokens in src/styles, Playwright
                     smoke test in e2e/
scripts/             e2e_grpcurl.sh — manual auth-lifecycle pass with grpcurl
```

## Conventions

- **Protobuf-first**: change `proto/`, run `make proto`, commit `gen/`.
  `buf breaking` runs against the PR base in CI. Codegen plugins are pinned
  as `tool` directives in go.mod (`go tool protoc-gen-go`).
- **Store**: plain SQL only; every new table comes from a new numbered file
  in `internal/store/migrations/` (`NNNN_description.sql`, applied in order,
  recorded in `schema_migrations`). Timestamps stored as RFC3339Nano UTC
  TEXT; IDs are UUIDv7 TEXT.
- **Everything project-scoped**: users, keys, credentials and tokens always
  hang off a project; nothing is shared across projects.
- **Everything embedded**: migrations, web assets, (later) email templates
  and SDK tarballs ship inside the binary via `go:embed`.
- **Secrets**: secret keys and session tokens are stored as SHA-256 hashes;
  project private keys are AES-GCM-encrypted under the master key
  (`data/keys/master.key`, or `MOTH_MASTER_KEY` env); passwords use
  argon2id. Plaintext secrets are returned to the caller exactly once.
- **Errors**: `store` returns `store.ErrNotFound`; RPC handlers map it to
  `connect.CodeNotFound` and validation failures to `CodeInvalidArgument`.
- gRPC server reflection is enabled only in dev builds
  (`version.Version == "dev"`, i.e. built without release ldflags).
