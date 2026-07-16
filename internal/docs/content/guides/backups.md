# Backups

moth keeps its entire state in one place. Backing it up is copying a
directory; the only subtlety is the master key.

> **Finalized in v1.0**
>
> First-class backup tooling (`moth backup` / snapshot verification) lands
> in the v1.0 hardening milestone. Everything below works today with
> standard tools — that tooling will automate exactly this.

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

> **Store the master key separately**
>
> Back up `master.key` to a *different* location than the database — a
> secrets manager, not the same disk snapshot. Keeping them together defeats
> encryption at rest: whoever has the backup has both halves. If you inject
> `MOTH_MASTER_KEY` from a KMS, the key file may not exist on disk at all;
> back up the KMS entry instead.

## Taking a backup

SQLite is a live database, so don't copy `moth.db` byte-for-byte while the
server runs (WAL mode means an in-flight write can leave the plain copy
inconsistent). Use SQLite's online backup, which is safe against a running
instance:

```sh
sqlite3 /var/lib/moth/data/moth.db ".backup '/backups/moth-$(date +%F).db'"
```

Then archive `uploads/` (ordinary files, safe to copy live) and confirm
your `master.key` is in your secrets store. For a cold backup, stop the
service first and copy the whole `data/` directory:

```sh
sudo systemctl stop moth
sudo tar czf /backups/moth-$(date +%F).tar.gz -C /var/lib/moth data
sudo systemctl start moth
```

Automate the hot path from cron/systemd-timer; keep the master key out of
that archive.

## Restoring

1. Install the same (or newer) moth version — migrations only ever move
   the schema forward, so a newer binary opens an older database; the
   reverse is not supported.
2. Put the data directory back at `--data-dir`.
3. Restore `master.key` to `data/keys/` (or set `MOTH_MASTER_KEY`) — the
   *same* key that encrypted this database.
4. Start moth and run [`moth doctor`](../../cli/reference/#moth-doctor) —
   it confirms the JWKS is served and provider configs still verify.

If you've restored to a new host with a different public URL, update
`base_url` before creating or issuing anything: it's baked into token
`iss`, JWKS URLs, email links, and OAuth redirects
([configuration](../../installation/#configuration)).

## Moving users, not the instance

To copy *users* between projects or instances — rather than restore a
whole instance — use the [migration import/export](../migration/) flow,
which is a portable JSON document rather than a database snapshot.
