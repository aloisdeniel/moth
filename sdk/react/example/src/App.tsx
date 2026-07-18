import { createMothFetch, MothGate, useMoth, useMothPush } from '@moth/react'
import { useMemo, useState } from 'react'

const backendUrl = import.meta.env.VITE_BACKEND_URL ?? 'http://localhost:8081'

/**
 * The signed-in app: user info + sign out, a call to the developer's own
 * backend (scripts/example_backend verifies the moth JWT against the
 * project JWKS), and a "pro" page gated on an entitlement — paywall →
 * Stripe test-mode Checkout → return → unlock.
 */
export function App() {
  const [page, setPage] = useState<'home' | 'pro'>('home')
  return (
    <div style={{ maxWidth: 560, margin: '0 auto', padding: 24 }}>
      <nav style={{ display: 'flex', gap: 12, marginBottom: 24 }}>
        <button onClick={() => setPage('home')}>Home</button>
        <button onClick={() => setPage('pro')}>Pro area</button>
      </nav>
      {page === 'home' ? <Home /> : <ProArea />}
    </div>
  )
}

function Home() {
  const { user, client, signOut, customerInfo, refreshCustomerInfo } = useMoth()
  const apiFetch = useMemo(() => createMothFetch(client), [client])
  const [backendReply, setBackendReply] = useState<string>('')
  const [accessStatus, setAccessStatus] = useState<string>('')

  const callBackend = async () => {
    // createMothFetch attaches `Authorization: Bearer <fresh access token>`;
    // the backend verifies it against the project JWKS (milestone 02).
    try {
      const resp = await apiFetch(`${backendUrl}/api/hello`)
      setBackendReply(await resp.text())
    } catch (err) {
      setBackendReply(String(err))
    }
  }

  const checkAccess = async () => {
    // An authenticated RPC: the SDK refreshes the access token under the
    // hood when it has expired, so this succeeds without a re-login.
    try {
      const info = await refreshCustomerInfo()
      setAccessStatus(
        info.activeEntitlements.length === 0
          ? 'free tier'
          : info.activeEntitlements.map((e) => e.identifier).join(', '),
      )
    } catch (err) {
      setAccessStatus(String(err))
    }
  }

  const manageBilling = async () => {
    // Redirects to the Stripe Billing Portal; throws when the user has no
    // web billing history yet.
    try {
      await client.manageBilling()
    } catch (err) {
      setAccessStatus(String(err))
    }
  }

  return (
    <main>
      <h1>Hello {user?.displayName ?? user?.email}</h1>
      <dl>
        <dt>User id</dt>
        <dd>
          <code>{user?.id}</code>
        </dd>
        <dt>Email</dt>
        <dd>
          {user?.email} {user?.emailVerified ? '(verified)' : '(unverified)'}
        </dd>
        <dt>Custom claims</dt>
        <dd>
          <code>{JSON.stringify(user?.claims ?? {})}</code>
        </dd>
        <dt>Entitlements</dt>
        <dd>
          {customerInfo.activeEntitlements.length === 0
            ? 'free tier'
            : customerInfo.activeEntitlements
                .map((e) => e.identifier)
                .join(', ')}
        </dd>
      </dl>
      <p>
        <button onClick={() => void callBackend()}>Call my backend</button>{' '}
        <button onClick={() => void checkAccess()}>Check access</button>{' '}
        <button onClick={() => void manageBilling()}>Manage billing</button>{' '}
        <button onClick={() => void signOut()}>Sign out</button>
      </p>
      <PushToggle />
      {accessStatus !== '' && <p data-testid="access-status">Access: {accessStatus}</p>}
      {backendReply !== '' && (
        <pre style={{ background: '#f5f5f5', padding: 12, overflowX: 'auto' }}>
          {backendReply}
        </pre>
      )}
    </main>
  )
}

/**
 * The Web Push settings row: `useMothPush()` as a toggle. The service worker
 * (public/sw.js, registered in main.tsx) owns display and click handling;
 * moth manages the subscription and the device-registry row. Environment
 * problems are states, never exceptions — a project without a VAPID key
 * reports `unavailable`, a browser without the Push API `unsupported`.
 */
function PushToggle() {
  const { status, permission, subscribe, unsubscribe } = useMothPush()
  if (status === 'unavailable' || status === 'unsupported') {
    return (
      <p data-testid="push-status">
        Push notifications: {status}
        {status === 'unavailable' &&
          ' — enable push (with a VAPID key) in the admin’s Settings tab.'}
      </p>
    )
  }
  return (
    <p data-testid="push-status">
      Push notifications:{' '}
      {status === 'denied' ? (
        // The browser remembers the denial; only the user can lift it.
        <>blocked — allow notifications for this site in the browser.</>
      ) : status === 'subscribed' ? (
        <button onClick={() => void unsubscribe()}>Disable</button>
      ) : (
        // Prompts for permission (an explicit user action, never an SDK
        // side effect), subscribes the service worker's PushManager and
        // registers this browser in the project's device registry.
        <button onClick={() => void subscribe()}>Enable</button>
      )}{' '}
      <small>
        (permission: {permission}, status: {status})
      </small>
    </p>
  )
}

function ProArea() {
  // A free user sees the themed paywall here; completing a test-mode Stripe
  // Checkout redirects back and the page unlocks without a manual refresh.
  return (
    <MothGate entitlement="pro">
      <main>
        <h1>Pro area</h1>
        <p>You hold the “pro” entitlement. Enjoy the shiny features.</p>
      </main>
    </MothGate>
  )
}
