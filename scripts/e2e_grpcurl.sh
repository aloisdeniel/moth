#!/usr/bin/env bash
# End-to-end pass of the milestone-02 auth lifecycle with grpcurl against a
# dev instance (server reflection is enabled in dev builds).
#
# Prerequisites:
#   - a running dev server:        make run   (in another terminal)
#   - grpcurl and jq on the PATH
#   - an admin account (create via the printed /admin?setup=... link, or
#     `moth admin create`), exported as MOTH_ADMIN_EMAIL / MOTH_ADMIN_PASSWORD
#
# The script creates a throwaway project, walks signup → verify → sign-in →
# refresh → change-password → reset → sign-in, and prints PASS at the end.
set -euo pipefail

HOST=${MOTH_HOST:-localhost:8080}
BASE=${MOTH_BASE_URL:-http://$HOST}
ADMIN_EMAIL=${MOTH_ADMIN_EMAIL:?export MOTH_ADMIN_EMAIL}
ADMIN_PASSWORD=${MOTH_ADMIN_PASSWORD:?export MOTH_ADMIN_PASSWORD}
EMAIL="e2e-$(date +%s)@example.com"

say() { printf '\n\033[1m== %s ==\033[0m\n' "$*"; }
rpc() { # rpc <service/method> <json> [extra grpcurl args...]
  local method=$1 body=$2; shift 2
  grpcurl -plaintext "$@" -d "$body" "$HOST" "$method"
}

say "admin login (session cookie)"
COOKIE=$(curl -sf -D - -o /dev/null "$BASE/moth.admin.v1.SessionService/Login" \
  -H 'content-type: application/json' \
  -d "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}" \
  | awk -F': ' 'tolower($1)=="set-cookie"{print $2}' | cut -d';' -f1)
[ -n "$COOKIE" ] || { echo "admin login failed"; exit 1; }

say "create project"
PROJECT=$(curl -sf "$BASE/moth.admin.v1.ProjectService/CreateProject" \
  -H 'content-type: application/json' -H "cookie: $COOKIE" \
  -d '{"name":"grpcurl e2e"}')
PK=$(jq -r .project.publishableKey <<<"$PROJECT")
SK=$(jq -r .secretKey <<<"$PROJECT")
SLUG=$(jq -r .project.slug <<<"$PROJECT")
echo "project $SLUG ($PK)"

say "SignUp"
rpc moth.auth.v1.AuthService/SignUp \
  "{\"email\":\"$EMAIL\",\"password\":\"password-1\"}" -H "x-moth-key: $PK" | jq .

say "verify email (paste the token from the console-logged link)"
echo "The dev server just logged a verification email for $EMAIL."
read -rp "token= " VERIFY_TOKEN
rpc moth.auth.v1.AuthService/ConfirmEmailVerification \
  "{\"token\":\"$VERIFY_TOKEN\"}" -H "x-moth-key: $PK"

say "SignIn"
SIGNIN=$(rpc moth.auth.v1.AuthService/SignIn \
  "{\"email\":\"$EMAIL\",\"password\":\"password-1\"}" -H "x-moth-key: $PK")
ACCESS=$(jq -r .tokens.accessToken <<<"$SIGNIN")
REFRESH=$(jq -r .tokens.refreshToken <<<"$SIGNIN")

say "GetMe"
rpc moth.auth.v1.AuthService/GetMe '{}' \
  -H "x-moth-key: $PK" -H "authorization: Bearer $ACCESS" | jq .

say "RefreshToken (rotation)"
ROTATED=$(rpc moth.auth.v1.AuthService/RefreshToken \
  "{\"refreshToken\":\"$REFRESH\"}" -H "x-moth-key: $PK")
ACCESS=$(jq -r .tokens.accessToken <<<"$ROTATED")
REFRESH2=$(jq -r .tokens.refreshToken <<<"$ROTATED")

say "ChangePassword (revokes other sessions)"
CHANGED=$(rpc moth.auth.v1.AuthService/ChangePassword \
  '{"currentPassword":"password-1","newPassword":"password-2"}' \
  -H "x-moth-key: $PK" -H "authorization: Bearer $ACCESS")
if rpc moth.auth.v1.AuthService/RefreshToken \
  "{\"refreshToken\":\"$REFRESH2\"}" -H "x-moth-key: $PK" 2>/dev/null; then
  echo "FAIL: pre-change refresh token still works"; exit 1
fi
echo "pre-change refresh token correctly rejected"

say "password reset flow"
rpc moth.auth.v1.AuthService/RequestPasswordReset \
  "{\"email\":\"$EMAIL\"}" -H "x-moth-key: $PK"
echo "The dev server just logged a reset email for $EMAIL."
read -rp "token= " RESET_TOKEN
rpc moth.auth.v1.AuthService/ConfirmPasswordReset \
  "{\"token\":\"$RESET_TOKEN\",\"newPassword\":\"password-3\"}" -H "x-moth-key: $PK"

say "SignIn with the reset password"
rpc moth.auth.v1.AuthService/SignIn \
  "{\"email\":\"$EMAIL\",\"password\":\"password-3\"}" -H "x-moth-key: $PK" | jq .user

say "IntrospectToken with the secret key"
rpc moth.server.v1.TokenService/IntrospectToken \
  "{\"accessToken\":\"$ACCESS\"}" -H "x-moth-key: $SK" | jq .

say "PASS"
