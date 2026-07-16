#!/usr/bin/env bash
# Load test for moth's hot path: AuthService.SignIn (argon2id verify +
# ES256 sign + refresh-token issue). Drives it with ghz over gRPC.
#
# Requirements:
#   - ghz            https://ghz.sh (`brew install ghz`)
#   - a moth instance running in DEV build (gRPC server reflection is on only
#     in dev builds; a release binary needs the .proto passed with --proto)
#   - a project with a known publishable key and one seeded user
#
# Set up a target quickly:
#   make build
#   ./bin/moth serve &                      # dev build → reflection on
#   # create an admin + project in the admin UI (http://localhost:8080/admin),
#   # note the publishable key (pk_...), then sign up one user from your app
#   # or the SDK e2e helper.
#
# Then, with the env vars below:
#   MOTH_PK=pk_xxx EMAIL=load@example.com PASSWORD=secret123 ./signin.sh
#
# ghz reuses HTTP/2 connections, so this measures moth's server-side cost,
# not connection setup. Record real numbers in RESULTS.md — do not quote a
# baseline you have not measured on the hardware you are documenting.
set -euo pipefail

HOST="${HOST:-localhost:8080}"
MOTH_PK="${MOTH_PK:?set MOTH_PK to a project publishable key (pk_...)}"
EMAIL="${EMAIL:?set EMAIL to a seeded user}"
PASSWORD="${PASSWORD:?set PASSWORD for that user}"
CONCURRENCY="${CONCURRENCY:-50}"
TOTAL="${TOTAL:-5000}"
CALL="moth.auth.v1.AuthService.SignIn"

if ! command -v ghz >/dev/null 2>&1; then
  echo "ghz not found — install from https://ghz.sh (brew install ghz)" >&2
  exit 127
fi

# --insecure: plain-HTTP h2c (dev). Behind TLS, drop it and add -cacert / the
# real scheme. --reflection is implied when no --proto is given; a release
# binary has reflection off, so pass:  --proto ../../proto/moth/auth/v1/auth.proto
exec ghz \
  --insecure \
  --call "$CALL" \
  --metadata "{\"x-moth-key\":\"$MOTH_PK\"}" \
  --data "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\",\"device_info\":\"ghz-loadtest\"}" \
  --concurrency "$CONCURRENCY" \
  --total "$TOTAL" \
  "$HOST"
