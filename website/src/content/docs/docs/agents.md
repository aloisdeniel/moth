---
title: Agents & automation
description: Drive moth from scripts and coding agents — the --json contracts and the exportable agent skill.
---

moth is built to be operated without a human at a keyboard. Two surfaces
make that reliable: **`--json` output** on every CLI command for scripts,
and **`moth skill export`** for coding agents that wire auth into apps.

## JSON output

Every remote CLI command accepts `--json` and prints a single
machine-readable JSON document — the same data the human view formats,
with no prompts, colors, or spinners. Combined with meaningful exit codes
and `--yes` for destructive operations, that is everything a script or CI
job needs:

```sh
# Create a project and capture its keys in CI
moth --context prod project create "Bird Spotter" --json --show-secret \
  | jq -r '.project.publishableKey'

# Fail the pipeline if any project's login success rate dips
moth --context prod stats get --project bird-spotter --json \
  | jq -e '.tiles.loginSuccessRate7d > 0.95'
```

Feed the token in from a secret so it never touches the config file on a
shared runner:

```sh
echo "$MOTH_PAT" | moth login https://auth.example.com --name ci --json
```

The exact fields per command are on the [Commands](../cli/reference/)
page; because that reference is generated from the binary, the `--json`
shapes it documents are the ones you get.

## Declarative project config

For anything beyond one-off calls, describe the desired state and apply
it. `moth project apply -f moth.yaml` diffs a `ProjectSpec` against the
live project and changes only what differs — idempotent, so running it
twice reports zero changes. `moth project dump` produces the document.
This gives teams reviewable, version-controlled auth config; see
[`moth project apply`](../cli/reference/#moth-project-apply).

## The agent skill

Coding agents are increasingly the ones adding auth to an app, so moth
ships them a first-class artifact instead of hoping they scrape docs:

```sh
moth skill export --project bird-spotter --dir .claude/skills/moth
```

This writes an [Agent Skills](../cli/reference/#moth-skill-export) package
— a `SKILL.md` with name/description frontmatter plus a `references/`
directory — teaching an agent **both halves** of moth:

- **Integrating moth into an app** — add the served `moth_auth`
  dependency, wrap `MothApp`, read `MothScope`, call the developer's
  backend with the token, verify JWTs server-side, and the platform setup
  steps for Google/Apple.
- **Administering an instance via the CLI** — the command groups, the
  `--json` contracts for parsing, `moth project apply` for declarative
  changes, `moth setup google|apple`, and `moth doctor` for diagnosis —
  written so an agent can operate an instance end to end without a
  browser.

Two properties make it trustworthy:

- **Interpolated with real values.** With `--project` and a configured
  [context](../cli/#connecting-to-an-instance), every snippet is filled in
  with that project's endpoint, publishable key, JWKS URL, and enabled
  providers — the agent equivalent of the project's Setup tab. Without
  `--project` it carries documented placeholders and contacts no server.
- **Can't drift.** The skill is assembled at release time from this same
  documentation tree and the generated CLI reference, then embedded in the
  binary — it can never fall out of sync with the docs or the real command
  surface.

`--format claude` (default) follows Claude Code conventions;
`--format generic` writes a plain-markdown `README.md` for other agent
frameworks. Exports are idempotent — regenerating after a config change
just overwrites the files in place.

## Generating your own clients

Not everything has to go through the CLI. moth is protobuf-first and
serves its `.proto` sources at [`/protos/`](../api/#the-protos), so you
can generate a typed gRPC client in any language and call the
[admin, auth, or server APIs](../api/) directly — the same surfaces the
CLI and SDK are built on.
