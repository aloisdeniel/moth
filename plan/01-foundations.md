# Milestone 01 — Foundations

## Goal

A runnable, testable skeleton: `moth serve` starts an HTTP server backed by SQLite, with migrations, configuration, a project model, admin bootstrap, and CI. Everything later builds on this.

## Deliverables

- Go module `github.com/aloisdeniel/moth` with a `moth` binary.
- CLI (stdlib `flag` or `spf13/cobra` — prefer cobra for subcommand ergonomics):
  - `moth serve` — start the server (`--addr`, `--data-dir`, `--base-url`).
  - `moth admin create --email ...` — bootstrap/reset an admin account from the shell.
  - `moth version`.
- Configuration precedence: flags > env vars (`MOTH_*`) > config file (`moth.toml`, optional) > defaults. Default data dir: `./data`.
- SQLite storage layer:
  - `modernc.org/sqlite`, WAL mode, foreign keys on, busy timeout.
  - Embedded SQL migrations (`go:embed`), applied automatically on startup, tracked in a `schema_migrations` table.
  - Initial schema: `admins`, `projects`, `project_keys`, `schema_migrations`.
- API server (gRPC, protobuf-first):
  - Proto workspace in `proto/` (`moth/admin/v1/`, `moth/auth/v1/`, `moth/server/v1/` from milestone 02, shared types in `moth/common/v1/`); `buf` for lint, breaking-change checks, and codegen (Go via `protoc-gen-connect-go`; Dart and TypeScript clients generated in later milestones from the same protos).
  - `connect-go` handlers mounted on a stdlib `net/http` mux with h2c — one port serves native gRPC, gRPC-Web, and the plain-HTTP surfaces below.
  - Interceptor chain (request ID, logging via `slog`, panic recovery) + CORS for gRPC-Web; standard `grpc.health.v1.Health` service and server reflection (dev builds) so `grpcurl` works out of the box.
  - Plain HTTP kept for what browsers/tools require: `GET /healthz`, static serving scaffold for the admin SPA under `/admin` (placeholder page embedded now).
- Project model & admin services (consumed by the SPA in milestone 03):
  - `moth.admin.v1.SessionService` — `Login`/`Logout` RPCs; the Login response sets an HttpOnly session cookie (SameSite=Lax; secure when base URL is https) — workable because connect-go rides on HTTP; an auth interceptor validates it on every admin RPC.
  - `moth.admin.v1.ProjectService` — `CreateProject`, `GetProject`, `ListProjects`, `UpdateProject`, `DeleteProject`. Creating a project generates a URL-safe slug, a **publishable key** (`pk_...`, identifies the project to the SDK) and a **secret key** (`sk_...`, stored hashed, for server-to-server calls).
- Signing keys, **one keypair per project**:
  - On first startup generate an instance **master key** (`data/keys/master.key`, overridable via env for KMS-style injection) used only to encrypt secrets at rest.
  - `CreateProject` generates a fresh ES256 keypair for the project, stored in `project_keys` with the private key encrypted under the master key and a `kid` derived from the public key thumbprint. Tokens for a project are only ever signed with its own key (used from milestone 02 on).
  - Expose `GET /p/{slug}/.well-known/jwks.json` — the project's active public keys (multiple entries once rotation exists in 10). Plain HTTP so any standard JWT library can consume it.
- Dev & CI:
  - `Makefile` (or `Taskfile`): `build`, `test`, `lint` (`golangci-lint`), `proto` (buf generate), `run`. Generated code committed so `go build` needs no proto toolchain.
  - GitHub Actions: test + lint + `buf lint`/`buf breaking` + stale-codegen check + cross-compile check (linux/amd64, linux/arm64, darwin/arm64, windows/amd64).
  - `CLAUDE.md` / `CONTRIBUTING.md` with repo layout and conventions.

## Repository layout

```
cmd/moth/            # main + CLI commands
proto/moth/          # .proto definitions (admin/v1, auth/v1, common/v1)
gen/                 # buf-generated Go code (committed)
internal/config/     # config resolution
internal/store/      # sqlite open, migrations, queries (per-domain files)
internal/store/migrations/*.sql
internal/server/     # mux assembly, interceptors, rpc/admin/, rpc/auth/, web/ (HTTP handlers)
internal/keys/       # master key, per-project keypair generation, JWKS
internal/version/
web/admin/           # SPA source (milestone 03); dist/ embedded
sdk/flutter/         # moth_auth package source (milestone 05)
plan/
```

## Key design points

- **Multi-project from the first migration** — moth's core promise is one server for a whole collection of independent apps, so projects are not an abstraction added later: the schema, key handling, and every API are project-scoped from day one, and serving the tenth app is structurally identical to serving the first.
- **Store as plain SQL** — hand-written queries behind small repository interfaces per domain (`AdminStore`, `ProjectStore`); no ORM. Interfaces keep handlers testable.
- **Everything embedded** — migrations, admin dist, email templates, SDK tarballs all via `go:embed` so the release artifact stays one file.
- **First-run experience** — if no admin exists, `/admin` shows a one-time setup screen to create the first admin account (token printed to the console guards it). This is the "run one binary and you're in" moment; treat it as a feature, not an afterthought.
- **IDs** — UUIDv7 primary keys as TEXT (sortable, no coordination).

## Acceptance criteria

- `go build ./...` produces a single static binary; `moth serve` on a clean machine creates `data/`, runs migrations, serves `/healthz`, and answers `grpcurl` (reflection) for the health service and `ProjectService`.
- First-run flow: console prints setup URL → create admin in browser → log in → create a project → keys returned once (secret shown only at creation) → the project's JWKS URL serves its public key.
- Restart keeps all state; deleting `data/` yields a fresh instance.
- CI green on the four target platforms' cross-compile.
- Unit tests for config precedence, migrations (fresh + idempotent re-run), project CRUD, admin session interceptor, per-project key generation (distinct keypairs, private key round-trips through master-key encryption, JWKS matches).

## Out of scope

End-user auth endpoints (02), real admin UI (03), remote CLI client mode (08 — milestone 01's CLI only manages the local instance), rate limiting and audit logging (10).
