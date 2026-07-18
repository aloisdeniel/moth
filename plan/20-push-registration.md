# Milestone 20 — Push Device Registry (server core)

## Goal

Make moth the place where a project's **push-notification device registrations** live —
without ever sending a push itself. Every signed-in user's devices register their push
identity (APNs device token, FCM registration token, or a Web Push subscription) with
moth; moth tracks each registration's target, permission state, and liveness, invalidates
it when it dies, and hands the current set to the **developer's own backend** through
`moth.server.v1` — which then talks to APNs/FCM/Web Push directly to deliver. The same
division of labor as auth (02) and billing (11): moth owns identity and state; the
external service owns delivery.

This milestone is the engine: model, RPCs, invalidation, admin visibility. The SDKs
learn to register in milestone 21.

## Model

- `push_devices` — per `(project_id, user_id)` device registration:
  - `target` — `apns` | `fcm` | `webpush`: which push service the credential belongs
    to, i.e. which API the developer's backend must call to reach it.
  - `token` — the push credential: APNs device token, FCM registration token, or the
    serialized Web Push subscription (endpoint + keys). Unique per `(project_id,
    target, token)`; re-registration **upserts** (tokens rotate; the newest owner
    wins, which also handles a device changing users on sign-in).
  - `permission` — `granted` | `provisional` | `denied` | `unknown`: the OS-level
    notification-permission state the client last reported. A registration with a
    token but `denied` permission is kept (data pushes may still work) but flagged, so
    senders can skip alert pushes.
  - `device_id` — a client-generated stable installation id, so one physical device
    re-registering replaces its own row instead of accumulating; plus display metadata
    (`platform`, `model`, `os_version`, `app_version`, `locale`) for the admin view and
    sender-side locale targeting.
  - `last_seen_at` — refreshed on every register/heartbeat; `revoked_at` + a
    `revoke_reason` (`signed_out`, `reported_invalid`, `stale`, `replaced`, `admin`)
    instead of hard deletes, so invalidation is auditable and idempotent.
- Per-project **push settings** on the project config (`moth.projectconfig.v1`): the
  Web Push **VAPID public key** (the SDK needs it to subscribe in the browser; the
  private key stays with the developer's sender and never touches moth) and an
  `enabled` switch. Delivered to clients via the existing public project config.

## Deliverables

### Client-facing RPCs (`moth.push.v1`, publishable key + Bearer)

- `RegisterDevice(target, token, device_id, permission, metadata)` — upsert the
  calling user's registration; returns the stored registration. Idempotent by design:
  the SDK calls it on every launch/token-rotation/permission-change without
  bookkeeping. Rate-limited like the other credential-facing RPCs (02).
- `UnregisterDevice(device_id)` — explicit revocation (`signed_out`); the milestone-21
  SDKs call it on sign-out. Unknown/already-revoked ids succeed (idempotent).
- Requires a signed-in user by construction — registrations always hang off
  `(project, user)`; there are no anonymous device registrations.

### Developer-backend RPCs (`moth.server.v1`, secret key)

- `ListUserPushDevices(user_id)` — the user's **active** registrations (target, token,
  permission, platform, locale, last-seen) — everything a sender needs to fan out to
  one user's devices.
- `ListPushDevices(page…)` — paginated active registrations across the project, for
  broadcast/segment sends; filterable by target.
- `RevokePushDevice(token | device_id)` — **feedback loop**: when the developer's
  sender gets `Unregistered`/`410 Gone` from APNs, `UNREGISTERED` from FCM, or `404/410`
  from a Web Push endpoint, it reports the dead credential back and moth revokes it
  (`reported_invalid`) so it never serves it again.

### Invalidation lifecycle

- **Sign-out** revokes (21 wires the SDK call; the auth handlers do not cascade —
  a device may host multiple accounts over time, and only the client knows the
  installation is ending its session).
- **Rotation/replacement**: an upsert on the same `(target, token)` or same
  `device_id` supersedes the old row (`replaced`).
- **Staleness sweep**: registrations not seen for a configurable window (default 90
  days) are revoked (`stale`) by the milestone-10 background sweep — dead installs
  decay without any feedback.
- **Admin revoke**: an operator can revoke a registration from the user view
  (`admin`, audit-logged per 10).

### Admin (`moth.admin.v1` + SPA)

- User detail gains a **Devices** panel: each registration's target, platform/model,
  permission, app version, last-seen, with a revoke action.
- Project **Settings** gains the push section: enable switch + VAPID public key field
  (plain config, not a secret — the private key never touches moth).
- Project overview/analytics: a registered-devices count per target (a gauge read off
  `push_devices`, not a new event stream).

## Key design points

- **moth registers; your server sends.** Delivery needs APNs/FCM credentials, retry
  logic, and payload semantics that belong to the developer's backend — putting them in
  moth would make it a push gateway and a liability. The registry + feedback RPC is the
  entire contract, mirroring how milestone 02 hands JWTs to the developer's backend
  instead of proxying its API calls.
- **Tokens are credentials.** They're stored plaintext (senders need them back — they
  cannot be hashed like sk_ keys), scoped per project, returned only over the
  secret-key surface, and never in admin RPCs or logs; the admin sees metadata, not
  tokens.
- **Idempotent, self-healing registry.** Upsert-on-register, tolerant unregister,
  feedback revocation, staleness decay: every path assumes clients repeat themselves
  and senders discover death lazily, because both are true in push systems.
- **`target` over `platform`.** The row answers "which API reaches this device", not
  "what OS is it" — an iOS app using FCM registers as `fcm`; platform is display
  metadata. This keeps the registry honest about what the sender must do with it.

## Acceptance criteria

- Register → `ListUserPushDevices` returns the registration; re-register with a
  rotated token supersedes the old row (asserted revoked `replaced`, not duplicated).
- Unregister, feedback `RevokePushDevice`, admin revoke, and the staleness sweep each
  flip a registration to revoked with the right reason; all are idempotent on replay;
  revoked registrations never appear in list RPCs.
- A second user signing in on the same device (same `device_id`) takes over the
  registration; the first user's list no longer shows it.
- Registrations are invisible across projects (secret key of project A cannot list
  B's devices); tokens appear in `moth.server.v1` responses only.
- Permission transitions (`granted` → `denied`) round-trip through `RegisterDevice`
  and show in the admin panel; the panel never renders the token.
- Table-driven store tests cover the upsert matrix (same device new token, same token
  new device, same token new user) and the sweep window.

## Out of scope

Sending pushes (APNs/FCM/Web Push clients, payload construction, scheduling,
campaigns), storing sender credentials (APNs keys, FCM service accounts, VAPID private
keys), notification analytics (delivery/open rates), topics/segments beyond the list
filters, and the SDK registration flows — milestone 21.
