---
title: Installation & deployment
description: Install the binary and run it for real — configuration, systemd, Docker, reverse proxy, TLS.
---

moth is a single static binary with a single data directory. A deployment
is: the binary, a config file (optional), and something that gives it TLS.

:::note[What "coming in v1.0" means]
moth is pre-1.0. Some deployment conveniences are finalized in the last
milestone before release and are marked **coming in v1.0** below. Nothing
marked that way is required to run moth today.
:::

## Get the binary

**Coming in v1.0:** signed release binaries for darwin/linux/windows
(amd64/arm64), a Homebrew tap, and a ~15 MB scratch-based Docker image.

Today, build from source with Go 1.25+:

```sh
git clone https://github.com/aloisdeniel/moth.git
cd moth
make build            # → bin/moth (dev build)
make build VERSION=…  # release-style build
```

The binary is fully self-contained — the admin SPA, migrations, email
templates, and the Flutter SDK tarball are embedded. Copy `bin/moth` to
the target machine and you're done installing.

## Configuration

Resolution order, highest first: **command-line flags** → **`MOTH_*`
environment variables** → **config file** (`moth.toml`, loaded from the
working directory when present, or `--config <path>` / `MOTH_CONFIG`) →
built-in defaults.

| Setting | Flag | Env | `moth.toml` | Default |
|---|---|---|---|---|
| Listen address | `--addr` | `MOTH_ADDR` | `addr` | `:8080` |
| Data directory | `--data-dir` | `MOTH_DATA_DIR` | `data_dir` | `./data` |
| Public base URL | `--base-url` | `MOTH_BASE_URL` | `base_url` | `http://localhost:8080` |
| SMTP host | — | `MOTH_SMTP_HOST` | `[smtp] host` | *(empty → console transport)* |
| SMTP port | — | `MOTH_SMTP_PORT` | `[smtp] port` | `587` |
| SMTP username | — | `MOTH_SMTP_USERNAME` | `[smtp] username` | — |
| SMTP password | — | `MOTH_SMTP_PASSWORD` | `[smtp] password` | — |
| SMTP from | — | `MOTH_SMTP_FROM` | `[smtp] from` | — |

`base_url` matters: it is baked into email links, the token `iss` claim,
JWKS URLs, and OAuth redirect URIs, and it decides whether admin session
cookies are marked `Secure`. Set it to the exact public URL
(`https://auth.example.com`) before creating projects.

A production `moth.toml`:

```toml title="/etc/moth/moth.toml"
addr     = ":8080"
data_dir = "/var/lib/moth/data"
base_url = "https://auth.example.com"

[smtp]
host     = "smtp.eu.example.com"
port     = 587
username = "moth"
password = "…"
from     = "auth@example.com"
```

SMTP can also be configured at runtime — stored in the database, taking
precedence over the file — via the admin console or
[`moth instance smtp set`](../cli/reference/#moth-instance-smtp-set), and
tested with `moth instance smtp test --to you@example.com`. With no SMTP
configured anywhere, emails are printed to the server log (fine for dev,
a visible warning in the admin for production).

## The data directory

Everything stateful lives under one directory:

```
data/
  moth.db      SQLite database (users, projects, tokens, events)
  keys/        master.key + nothing else you should touch
  uploads/     project logo assets
```

- `keys/master.key` encrypts project signing keys and provider secrets at
  rest. Alternatively supply it as the `MOTH_MASTER_KEY` environment
  variable and keep no key file on disk. **Losing the master key means
  losing every project's signing keys** — see [Backups](../guides/backups/).
- Back up the whole directory, not just the database.

## systemd

```ini title="/etc/systemd/system/moth.service"
[Unit]
Description=moth authentication server
After=network-online.target
Wants=network-online.target

[Service]
User=moth
Group=moth
ExecStart=/usr/local/bin/moth serve --config /etc/moth/moth.toml
Restart=on-failure
RestartSec=2

# Hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
ReadWritePaths=/var/lib/moth

[Install]
WantedBy=multi-user.target
```

```sh
sudo useradd --system --home /var/lib/moth --shell /usr/sbin/nologin moth
sudo mkdir -p /var/lib/moth && sudo chown moth:moth /var/lib/moth
sudo systemctl enable --now moth
```

Create the first admin on the host (it talks to the same database):

```sh
sudo -u moth moth admin create --config /etc/moth/moth.toml --email you@example.com
```

— or just open `https://auth.example.com/admin` and use the first-run
setup screen.

## Docker

**Coming in v1.0:** an official scratch-based image. Until then a
build-your-own image is a few lines:

```dockerfile title="Dockerfile"
FROM golang:1.25 AS build
WORKDIR /src
RUN git clone --depth 1 https://github.com/aloisdeniel/moth.git . \
 && CGO_ENABLED=0 make build

FROM gcr.io/distroless/static
COPY --from=build /src/bin/moth /moth
VOLUME /data
EXPOSE 8080
ENTRYPOINT ["/moth", "serve", "--data-dir", "/data"]
```

```sh
docker build -t moth .
docker run -d -p 8080:8080 -v moth-data:/data \
  -e MOTH_BASE_URL=https://auth.example.com \
  -e MOTH_MASTER_KEY=… \
  moth
```

moth is CGO-free (pure-Go SQLite driver), so the static/distroless base
works without glibc. Mount `/data` — it is the entire state of the
instance.

### Docker Compose

A minimal stack with a named volume for the data directory and Caddy in
front for automatic TLS:

```yaml title="docker-compose.yml"
services:
  moth:
    image: ghcr.io/aloisdeniel/moth:latest   # or `build: .`
    restart: unless-stopped
    environment:
      MOTH_BASE_URL: https://auth.example.com
      MOTH_MASTER_KEY: ${MOTH_MASTER_KEY}     # from a .env file / secret
    volumes:
      - moth-data:/data
    expose:
      - "8080"

  caddy:
    image: caddy:2
    restart: unless-stopped
    ports: ["80:80", "443:443"]
    command: caddy reverse-proxy --from auth.example.com --to h2c://moth:8080
    volumes:
      - caddy-data:/data

volumes:
  moth-data:
  caddy-data:
```

`caddy:2`'s `reverse-proxy --to h2c://…` keeps the upstream on HTTP/2 so
native gRPC from the mobile SDK works end to end. Generate a master key once
(`openssl rand -base64 32`) and keep it in the `.env` — losing it means
losing every project's signing keys.

## Reverse proxy & TLS

moth listens on plain HTTP and speaks **HTTP/2 without TLS (h2c)** on the
same port, so a proxy can sit in front of everything. The one requirement
that trips people up:

:::caution[Native gRPC needs HTTP/2 end to end]
The Flutter SDK on iOS/Android uses native gRPC, which requires HTTP/2 on
the whole path — browser → proxy **and** proxy → moth. The admin SPA,
gRPC-Web, Connect, the pub repository, and the hosted pages are all fine
over HTTP/1.1. If everything works except the mobile app, your proxy is
downgrading the upstream connection to HTTP/1.1.
:::

### Caddy (recommended)

Caddy terminates TLS with automatic Let's Encrypt certificates and can
proxy h2c upstream — the entire config:

```text title="Caddyfile"
auth.example.com {
    reverse_proxy h2c://127.0.0.1:8080
}
```

The `h2c://` scheme is the important part: it keeps proxy → moth on
HTTP/2 so native gRPC round-trips.

### Traefik

```yaml
# dynamic configuration
http:
  routers:
    moth:
      rule: Host(`auth.example.com`)
      service: moth
      tls:
        certResolver: letsencrypt
  services:
    moth:
      loadBalancer:
        servers:
          - url: h2c://127.0.0.1:8080
```

### nginx

nginx cannot proxy generic HTTP/2 upstream: `proxy_pass` is HTTP/1.1-only
and `grpc_pass` handles only gRPC framing — but every protocol moth
serves shares the same `/moth.*` paths, so a clean path-based split is
not possible. `proxy_pass http://127.0.0.1:8080` keeps everything working
*except* native gRPC from the mobile SDK. A hardened, tested nginx recipe
(routing on `Content-Type: application/grpc`) ships with the v1.0
deployment guide; until then, prefer Caddy or Traefik in front of moth.

### Built-in ACME

**Coming in v1.0:** `moth serve --acme-domain auth.example.com` — TLS
directly from the binary on a bare VPS, no proxy at all. Also coming in
v1.0: `--trusted-proxies`, so per-IP rate limits see real client
addresses behind a proxy.

## Health & diagnostics

- `GET /healthz` — plain-HTTP liveness (also: the standard gRPC health
  service).
- [`moth doctor`](../cli/reference/#moth-doctor) — the support checklist for
  "login stopped working": base-URL/TLS sanity, health and pub endpoints,
  SMTP (with a real test send), and per-project JWKS + provider
  verification against Google's and Apple's live endpoints.
- gRPC server reflection is enabled only in dev builds — release builds
  don't advertise their surface.

**Coming in v1.0:** a `/metrics` Prometheus endpoint, structured JSON
logs, and an append-only audit log for every admin action.
