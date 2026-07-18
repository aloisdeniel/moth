# moth react example

A tiny Vite app against a local moth instance: login-gated home (user info,
sign out), a call to the sample backend (`scripts/example_backend` at the
repo root, which verifies the moth JWT against the project JWKS), and a
"Pro area" page behind `<MothGate entitlement="pro">` demonstrating
paywall → Stripe test-mode Checkout → unlock. The home page also carries a
Web Push toggle (`useMothPush()` + the minimal `public/sw.js` from the SDK
README) — enable push with a VAPID key in the admin's Settings tab and the
browser shows up under the user's push devices in the admin.

```sh
# 1. a moth instance with a project (see the repo README)
make run

# 2. the sample backend (verifies moth JWTs)
go run ./scripts/example_backend --issuer http://localhost:8080/p/<slug>

# 3. this app
cd sdk/react/example
npm install
VITE_MOTH_ENDPOINT=http://localhost:8080 \
VITE_MOTH_PUBLISHABLE_KEY=pk_... \
VITE_BACKEND_URL=http://localhost:8081 \
npm run dev
```

The SDK is aliased to `../src` in `vite.config.ts`, so edits to the SDK
source hot-reload straight into this app.
