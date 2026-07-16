# Load testing moth

`signin.sh` drives `moth.auth.v1.AuthService.SignIn` — moth's most expensive
hot path: an argon2id password verify, an ES256 signature, and a
refresh-token insert per call. It is the number the README's baseline should
come from.

## Run it

```sh
make build
./bin/moth serve &                 # dev build: gRPC reflection is on
# In the admin (http://localhost:8080/admin): create an admin, a project,
# copy its publishable key, and sign up one user.

MOTH_PK=pk_xxx EMAIL=load@example.com PASSWORD=secret123 \
  CONCURRENCY=50 TOTAL=5000 ./scripts/loadtest/signin.sh
```

Against a **release** binary reflection is off, so pass the schema instead of
relying on it:

```sh
ghz --insecure --proto proto/moth/auth/v1/auth.proto \
    --call moth.auth.v1.AuthService.SignIn ...
```

## Reading the result

argon2id is deliberately slow (that is the point), so SignIn throughput is
dominated by the argon2 cost parameters, not by moth's own overhead. Expect
tens to low-hundreds of sign-ins per second per core. Tune with the argon2
parameters if you need more; do not lower them below the security floor.

`RefreshToken` and `GetJWKS` are far cheaper (no argon2) and will show much
higher numbers — worth measuring separately to characterise steady-state app
traffic, which is mostly token refreshes, not fresh sign-ins.

## RESULTS.md — record real numbers only

`RESULTS.md` is the place to paste **measured** `ghz` summaries with the
hardware they ran on. The project's policy is to never quote a load figure
that has not been measured on the machine it is attributed to. Until a figure
is recorded there and in the README, the README says so plainly rather than
inventing one.
