# Contributing to moth

## Prerequisites

- Go 1.24+
- [buf](https://buf.build) (proto lint/codegen) — only needed when touching `proto/`
- [golangci-lint](https://golangci-lint.run) v2 — only needed for `make lint`

Codegen plugins (`protoc-gen-go`, `protoc-gen-connect-go`) are pinned as Go
tool directives in `go.mod`; `buf generate` invokes them via `go tool`, so no
global installs are required.

## Workflow

```sh
make build   # bin/moth
make test
make lint
make proto   # after editing proto/ — commit the regenerated gen/
```

`go build ./...` works without buf: generated code is committed.

Try it out:

```sh
./bin/moth serve
# open the printed setup URL, create the admin, create a project
```

State lives in `./data` (override with `--data-dir`); delete it for a fresh
instance.

## Guidelines

- Read `plan/00-overview.md` and the current milestone before larger changes;
  `CLAUDE.md` documents layout and conventions.
- Schema changes: add a new `internal/store/migrations/NNNN_*.sql` file —
  never edit an applied migration.
- Proto changes must pass `buf lint` and `buf breaking` (run in CI against
  the PR base branch).
- Add tests next to the package you change (`_test.go`); prefer exercising
  real SQLite in a `t.TempDir()` over mocks.
