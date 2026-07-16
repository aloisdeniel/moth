# example_backend

The "your own API" half of the moth loop: a tiny HTTP server (standard
library only, ~200 lines) that authenticates requests by verifying the moth
access token against the project's public JWKS — what any real backend
does, in any language, with any standard JWT library.

```sh
go run ./scripts/example_backend --issuer http://localhost:8080/p/<slug>
```

`<slug>` is the project slug from the moth admin. The `--issuer` value is
exactly the `iss` claim moth mints (`<base URL>/p/<slug>`); the JWKS is
fetched from `<issuer>/.well-known/jwks.json` and cached, refetching when
an unknown `kid` shows up (key rotation).

`GET /api/hello` with `Authorization: Bearer <moth access token>` returns
the verified identity:

```json
{
  "message": "Hello jane@example.com, your JWT checks out.",
  "user_id": "0198...",
  "email": "jane@example.com",
  "email_verified": true,
  "claims": { "role": "admin" }
}
```

The Flutter example app (`sdk/flutter/example`) calls this endpoint from
its "Call my backend" button via the SDK's `authenticatedClient`, which
attaches an auto-refreshed token to every request.

What the verifier checks — the same list your production backend should:

1. `alg` is ES256 and the header `kid` resolves to a JWKS key.
2. The ECDSA signature over `<header>.<payload>` is valid for that key.
3. `exp` has not passed.
4. `iss` is exactly your instance's `<base URL>/p/<slug>`.

Verify with an off-the-shelf library instead by pointing it at the same
JWKS URL — moth's tokens are plain ES256 JWTs.
