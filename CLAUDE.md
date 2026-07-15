# moth

One-binary authentication server for mobile apps: one moth instance hosts a
whole portfolio of independent apps ("projects"), each a sealed tenant with
its own users, ES256 signing keypair and configuration. The plan lives in
`plan/` (start with `plan/00-overview.md`); milestones are implemented in
order.

## Commands

- `make build` — build `bin/moth` (set `VERSION=` for releases)
- `make test` — `go test ./...`
- `make lint` — `golangci-lint run` + `buf lint`
- `make proto` — `buf generate` (regenerates `gen/`; commit the result)
- `make run` — build and start `moth serve`
- `make cross` — check the four release targets compile

## Layout

```
cmd/moth/            main + cobra commands (serve, admin create, version)
proto/moth/          protobuf definitions (moth.admin.v1, later auth/server)
gen/                 buf-generated Go code — committed, never hand-edited
internal/config/     flags > MOTH_* env > moth.toml > defaults resolution
internal/store/      SQLite (modernc.org/sqlite), embedded migrations,
                     hand-written SQL behind per-domain interfaces (no ORM)
internal/keys/       master key (encrypts at rest), per-project ES256
                     keypairs, JWKS building
internal/password/   argon2id hashing
internal/token/      random API keys / session tokens + SHA-256 hashing
internal/server/     handler assembly, interceptors, plain-HTTP surfaces
internal/server/rpc/admin/  moth.admin.v1 connect handlers + auth interceptor
internal/server/web/ embedded placeholder admin page (real SPA: milestone 03)
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
