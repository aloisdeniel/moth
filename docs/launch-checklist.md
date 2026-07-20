<!-- Human-maintained runbook (not generated). The generated CLI reference
     lives next to it at docs/cli/reference.md. -->

# v1.0 launch checklist

The goal: **site, docs, binary, and SDK all land on the same version.** A
`vX.Y.Z` tag is the single trigger; everything downstream keys off it.

## Before tagging

- [ ] `main` is green: `test`, `lint`, `proto`, `web`, `flutter`, `e2e`,
      `cross-compile`, plus the new `govulncheck` and `gosec` jobs.
- [ ] `/security-review` run over the full codebase; zero unaddressed **high**
      findings (`govulncheck ./...` and `gosec` clean locally too).
- [ ] Committed codegen and embedded assets are current — CI fails on stale
      `gen/`, `web/admin/src/gen`, `internal/server/web/dist`, the Dart stubs,
      and the embedded docs (`make docs-embed` produces no diff).
- [ ] `CHANGELOG.md` regenerated for the release: `git cliff --tag vX.Y.Z`,
      reviewed, committed.
- [ ] `make cross` passes (all six targets compile CGO-free).
- [ ] `goreleaser check` is clean and `goreleaser release --snapshot --clean`
      builds locally without pushing.
- [ ] Clean-machine drill done: a bare VPS → download binary →
      `moth serve --acme-domain auth.example.com` → full product tour (admin,
      project, Flutter example login) using only the published docs.
- [ ] Backup/restore drill: a backup taken under write load restores to a
      working instance.

## Version coupling (why one tag is enough)

- The GoReleaser ldflags stamp the tag into
  `internal/version.Version` (`-X …=v{{ .Version }}`).
- That same constant drives the pub tarball version served at `/pub`
  (`internal/server/pub.go`) **and** the embedded `/docs` content is built
  from the tree at that tag — so a client running the binary gets an SDK and
  docs that match the binary exactly.
- Nothing hard-codes a version string that could drift.

## Cutting the release

1. [ ] `git tag -a vX.Y.Z -m "moth vX.Y.Z"` on the reviewed commit.
2. [ ] `git push origin vX.Y.Z` → `release.yml` runs GoReleaser:
       six signed archives, cosign-signed `checksums.txt`, the Homebrew cask,
       and the scratch multi-arch Docker images pushed to GHCR.
3. [ ] Verify the release: `cosign verify-blob` on the checksums (command in
       the GitHub release footer), `docker run` the image with a mounted
       volume, `dart pub add moth_auth --hosted-url https://<instance>/pub`
       resolves the new version.

## Website / docs deploy (versioned)

The website (milestone 09) currently deploys on `main`. To make the tag also
publish a **versioned** docs snapshot (parking-lot TODO already noted in
`.github/workflows/pages.yml`):

1. [ ] Add `tags: ['v*']` under `push:` in `pages.yml`.
2. [ ] Branch the base path per version so old snapshots stay reachable.
3. [ ] Confirm the site's CLI reference and embedded-docs source were synced
       from the tagged tree (`make docs-embed`, `sync-generated.mjs`).

## After launch

- [ ] Announce with the honest performance note (measured `ghz` numbers from
      `scripts/loadtest/RESULTS.md`, not an invented baseline).
- [ ] Open the next milestone from the post-v1 backlog in `plan/`.
