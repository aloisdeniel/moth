# CLI reference

The `moth` binary is two things at once: the server (`moth serve`) and a
remote admin client for any moth instance, kubectl-style. There is no
second binary to install. Everything the admin console can do is
scriptable from the terminal, because the CLI is a thin wrapper over the
same `moth.admin.v1` gRPC services the console uses — the two can never
diverge in capability.

The full, flag-by-flag list of commands lives on the
**[Commands](reference/)** page. It is **generated from the binary
itself** (`moth docs gen`) and synced into this site at build time, so it
always matches the released command surface — this overview is the only
hand-written part.

## Connecting to an instance

The CLI authenticates with a **personal access token** (`moth_pat_…`),
created in the admin console under **Account**, or on the server host with
[`moth admin token create`](reference/#moth-admin-token-create).
`moth login` validates the token and stores it as a named **context**
(server URL + credential) in `~/.config/moth/config.toml`:

```sh
moth login https://auth.example.com          # prompts for a PAT
moth login https://auth.example.com --name prod
```

Switch between instances with `--context <name>` or `MOTH_CONTEXT`. In
scripts, pipe the token in (`echo "$MOTH_PAT" | moth login … --name ci`).
Revoking the token in the console fails the CLI's next call immediately.

## Command groups

Each group mirrors an admin service — see the linked reference sections:

| Group | Does | Reference |
|---|---|---|
| [`moth project`](reference/#moth-project) | create / list / update / delete projects, `apply`/`dump` declarative config, `export`/`import` users | management |
| [`moth project keys`](reference/#moth-project-keys) | show signing key & JWKS, reset signing key, regenerate the secret key | keys |
| [`moth user`](reference/#moth-user) | list / get / create / invite / disable / delete users, set custom claims, revoke sessions | users |
| [`moth stats`](reference/#moth-stats) | pull a project's analytics tiles and breakdowns | analytics |
| [`moth instance`](reference/#moth-instance) | instance settings and SMTP (`set` / `test` / `clear`) | instance |
| [`moth setup`](reference/#moth-setup) | one-command [Google](../guides/google/) / [Apple](../guides/apple/) provider configuration | providers |
| [`moth doctor`](reference/#moth-doctor) | the "login stopped working" health check | diagnostics |
| [`moth skill`](reference/#moth-skill) | export the [agent skill](../agents/) | automation |

Local-only commands that run on the server host against the database
directly — `moth serve`, `moth admin create`,
[`moth admin token`](reference/#moth-admin-token) — need no context.

## Scripting ergonomics

- `--json` on every remote command emits machine-readable output for
  piping into `jq`; see [Agents & automation](../agents/) for the
  contract.
- Destructive commands prompt for confirmation; `--yes` skips it for
  non-interactive use.
- Secrets are printed only with an explicit `--show-secret` — a key you
  don't ask to see never lands in shell history or CI logs.
- Meaningful exit codes: non-zero on any failure.

## Declarative config

`moth project apply -f moth.yaml` is idempotent create-or-update of a
project's full desired state — settings, providers, theme, legal links.
`moth project dump` emits the current state as that document, so you can
check auth config into version control and review changes:

```sh
moth project dump bird-spotter > moth.yaml   # snapshot current state
moth project apply -f moth.yaml              # apply; re-running is a no-op
```

See [`moth project apply`](reference/#moth-project-apply) for the one
sharp edge (proto3 can't tell an omitted boolean from `false`, so start
from a `dump`).
