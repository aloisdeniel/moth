# Migration import & export

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
state, disabled state, and custom claims — everything except credentials.
See [`moth project export`](../../cli/reference/#moth-project-export).

> **Credentials never leave the server**
>
> Password hashes are **not** exported, and social identities are not
> carried in the document. This is deliberate: hashes are argon2id and
> account-bound, and re-exporting them widens their exposure for no benefit.
> See below for how users regain access after an import.

## Importing users

```sh
moth project import bird-spotter -f users.json --invite
```

Import creates the document's users in the target project, restoring
display name, verification, disabled state, and claims. It is **safe to
re-run**: a user whose email already exists is skipped, so a partial or
repeated import converges rather than duplicating.
[`moth project import`](../../cli/reference/#moth-project-import).

Because passwords don't round-trip, each imported user needs a way back in:

- **`--invite`** — emails every newly created user a set-password link.
  The clean choice when you have working SMTP.
- **Without `--invite`** — each user gets an unusable random password and
  recovers through "forgot password" (a password identity) or simply
  signs in with Google/Apple, which **re-links automatically** to the
  imported account on the first provider-verified sign-in.

## Migrating from another auth system

Produce a JSON array matching the export shape (`moth project export` on a
throwaway project shows the exact fields), then `import --invite`. Users
set a fresh password on first sign-in, or link a social provider — moth
never ingests foreign password hashes in v1.

> **Foreign hashes: coming in v1.0**
>
> The v1.0 hardening milestone adds a migration format that can carry
> foreign password hashes (bcrypt/scrypt/argon2 from another system) and
> verify against them once, upgrading to argon2id on first successful
> sign-in — so users migrate without a password reset. Until then, use the
> invite/social-relink path above.

## Restoring a whole instance

Import/export moves *users*. To restore an entire instance — every
project, its keys, its uploads — you want a
[data-directory backup](../backups/), not this JSON.
