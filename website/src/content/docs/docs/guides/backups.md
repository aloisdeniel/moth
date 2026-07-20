---
title: Backups
description: Back up and restore a moth instance — the whole state is one directory plus one key.
---

moth keeps its entire state in one place, and ships first-class tooling to
snapshot and restore it: `moth backup` and `moth restore`. The only
subtlety is where the master key lives.

## What to back up

Two things, and they are backed up differently:

1. **The data directory** — `data/` by default (`--data-dir`):

   ```
   data/
     moth.db        SQLite database (users, projects, tokens, events)
     keys/          master.key
     uploads/       project logo assets
   ```

2. **The master key** — `data/keys/master.key`, or whatever you inject via
   `MOTH_MASTER_KEY`.

The master key encrypts every project's signing private key and all
provider secrets at rest. **A database backup without the master key is
unrecoverable** — you'd restore encrypted blobs you can't decrypt.

:::caution[Store the master key separately]
Back up `master.key` to a *different* location than the database — a
secrets manager, not the same disk snapshot. Keeping them together defeats
encryption at rest: whoever has the backup has both halves. If you inject
`MOTH_MASTER_KEY` from a KMS, the key file may not exist on disk at all;
back up the KMS entry instead.
:::

## Taking a backup

`moth backup` writes a single gzip-compressed tar archive with an
online-consistent snapshot of the database (taken with `VACUUM INTO`, so
it is **safe against a live server** — no need to stop it), plus `uploads/`
and the key material:

```sh
moth backup --data-dir /var/lib/moth --to /backups/moth-$(date +%F).tar.gz
```

Restore it with `moth restore` (below). The archive is self-contained, so
by default it **includes `master.key`** — which makes it as sensitive as
the master key itself. Store it encrypted and access-controlled, not in a
shared bucket. (If you inject the key via `MOTH_MASTER_KEY` instead of a
key file, there is no `master.key` on disk to bundle, and the archive
carries only the database and uploads — back the KMS entry up separately.)

Automate it from cron or a systemd timer:

```sh
# daily at 03:30, keeping the last 14 days
30 3 * * *  moth backup --data-dir /var/lib/moth --to /backups/moth-$(date +\%F).tar.gz \
              && find /backups -name 'moth-*.tar.gz' -mtime +14 -delete
```

Or let the server take them itself — set `backup_dir` (and optionally
`backup_interval`, default `24h`) in the config and `moth serve` writes
scheduled snapshots without any external scheduler.

### Manual alternative

If you'd rather not use `moth backup`, SQLite's online backup is also safe
against a running instance — don't copy `moth.db` byte-for-byte while the
server runs (WAL mode can leave a plain copy inconsistent):

```sh
sqlite3 /var/lib/moth/data/moth.db ".backup '/backups/moth-$(date +%F).db'"
```

Then archive `uploads/` and confirm your `master.key` is in your secrets
store.

## Restoring

From a `moth backup` archive:

```sh
moth restore /backups/moth-2026-07-18.tar.gz --data-dir /var/lib/moth
```

`moth restore` recreates the database, uploads, and keys in the data
directory. For safety it **refuses to write into a non-empty data
directory** unless you pass `--force`, so an accidental restore can't
clobber a running instance — stop `moth serve` before restoring over
existing data.

A few things to keep in mind:

1. Install the same (or newer) moth version — migrations only ever move
   the schema forward, so a newer binary opens an older database; the
   reverse is not supported.
2. If your backup did **not** include the master key (you inject
   `MOTH_MASTER_KEY`), make the *same* key available — it's the only thing
   that can decrypt the restored signing keys and provider secrets.
3. Start moth and run [`moth doctor`](../../cli/reference/#moth-doctor) —
   it confirms the JWKS is served and provider configs still verify.

If you've restored to a new host with a different public URL, update
`base_url` before creating or issuing anything: it's baked into token
`iss`, JWKS URLs, email links, and OAuth redirects
([configuration](../../installation/#configuration)).

## Moving users, not the instance

To copy *users* between projects or instances — rather than restore a
whole instance — use the [migration import/export](../migration/) flow,
which is a portable JSON document rather than a database snapshot.
