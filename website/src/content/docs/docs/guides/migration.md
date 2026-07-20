---
title: Migration import & export
description: Move users in and out of a project with a portable JSON document.
---

Two separate concerns travel separately in moth: a project's
**configuration** (settings, providers, theme) and its **users**. This
guide covers users; for config-as-code see
[`moth project apply`](../../cli/#declarative-config).

Export and import operate on a portable JSON document, so they work
between projects, between instances, and as a plain migration off another
auth system.

## Exporting users

```sh
moth project export bird-spotter -o users.json
```

The document carries each user's email, display name, email-verification
state, disabled state, custom claims, provider identities, **and the
encoded password hash** — everything needed to move an account without a
reset. See [`moth project export`](../../cli/reference/#moth-project-export).

:::caution[An export contains credentials — handle it like one]
Password hashes travel with the users (argon2id, tagged `argon2id`), so
the JSON is as sensitive as your password database. Store and transfer it
accordingly — encrypted at rest, never in a shared bucket or a ticket
attachment — and delete it once the import is confirmed. Publishable
project config is a separate document; see
[`moth project dump`](../../cli/#declarative-config).
:::

## Importing users

```sh
moth project import bird-spotter -f users.json
```

Import creates the document's users in the target project, restoring
display name, verification, disabled state, claims, provider identities,
and the password hash. It is **safe to re-run**: a user whose email
already exists is skipped, so a partial or repeated import converges
rather than duplicating (`--yes` skips the confirmation prompt for
scripting).
[`moth project import`](../../cli/reference/#moth-project-import).

Because the hash round-trips, **users keep their existing password** — a
moth-to-moth migration signs everyone back in with no action on their
part. Social identities re-link automatically on the user's next
provider-verified sign-in.

## Migrating from another auth system

Import also ingests **foreign password hashes** — bcrypt, scrypt, argon2,
and pbkdf2 — tagged per user in the document's `password_algorithm` field.
Each foreign hash is verified with its original algorithm on the user's
first sign-in and then transparently rehashed to argon2id, so a team can
move off Firebase, Auth0, or Supabase **without forcing a password reset**.

Produce a JSON array matching the export shape (`moth project export` on a
throwaway project shows the exact fields), set each user's
`password_algorithm`, and import. Users whose source system you can't
export hashes from can still recover through "forgot password" or by
linking Google/Apple, which re-links to the imported account on the first
provider-verified sign-in.

## Restoring a whole instance

Import/export moves *users*. To restore an entire instance — every
project, its keys, its uploads — you want a
[data-directory backup](../backups/), not this JSON.
