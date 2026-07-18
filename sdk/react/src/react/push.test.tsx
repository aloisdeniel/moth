import { create } from '@bufbuild/protobuf'
import { Code } from '@connectrpc/connect'
import { act, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { PushConfigSchema } from '../gen/moth/auth/v1/config_pb.js'
import { PushPermission, PushTarget } from '../gen/moth/push/v1/push_pb.js'
import { cacheNamespace } from '../core/cache.js'
import type { MothClient } from '../core/client.js'
import { fakeClient, mothConnectError, type FakeMoth } from '../test/fake.js'
import { MothProvider } from './context.js'
import { useMothPush } from './hooks.js'

// ------------------------------------------------- browser environment stub

interface FakeSubscription {
  endpoint: string
  unsubscribeCalls: number
  toJSON(): unknown
  unsubscribe(): Promise<boolean>
}

function fakeSubscription(): FakeSubscription {
  const sub: FakeSubscription = {
    endpoint: 'https://push.example/sub-1',
    unsubscribeCalls: 0,
    toJSON: () => ({
      endpoint: sub.endpoint,
      keys: { p256dh: 'p256', auth: 'auth' },
    }),
    unsubscribe: async () => {
      sub.unsubscribeCalls++
      if (currentManager?.subscription === sub) {
        currentManager.subscription = null
      }
      return true
    },
  }
  return sub
}

interface FakeManager {
  subscription: FakeSubscription | null
  subscribeOptions: PushSubscriptionOptionsInit[]
  subscribe(options: PushSubscriptionOptionsInit): Promise<FakeSubscription>
  getSubscription(): Promise<FakeSubscription | null>
}

let currentManager: FakeManager | null = null

interface FakeNotification {
  permission: NotificationPermission
  requestPermission: ReturnType<typeof vi.fn>
}

/**
 * Installs a fake `PushManager`/`navigator.serviceWorker`/`Notification`
 * trio, the way the SDK feature-detects them.
 */
function stubPushEnvironment(
  options: {
    permission?: NotificationPermission
    /** What `Notification.requestPermission()` resolves (and flips) to. */
    requestResult?: NotificationPermission
  } = {},
): { manager: FakeManager; notification: FakeNotification } {
  const manager: FakeManager = {
    subscription: null,
    subscribeOptions: [],
    async subscribe(subscribeOptions) {
      manager.subscribeOptions.push(subscribeOptions)
      manager.subscription ??= fakeSubscription()
      return manager.subscription
    },
    async getSubscription() {
      return manager.subscription
    },
  }
  currentManager = manager
  const registration = { pushManager: manager }
  vi.stubGlobal('PushManager', class {})
  Object.defineProperty(navigator, 'serviceWorker', {
    configurable: true,
    value: {
      ready: Promise.resolve(registration),
      getRegistration: async () => registration,
    },
  })
  const notification: FakeNotification = {
    permission: options.permission ?? 'default',
    requestPermission: vi.fn(async () => {
      notification.permission = options.requestResult ?? 'granted'
      return notification.permission
    }),
  }
  vi.stubGlobal('Notification', notification)
  return { manager, notification }
}

afterEach(() => {
  vi.unstubAllGlobals()
  delete (navigator as { serviceWorker?: unknown }).serviceWorker
  currentManager = null
})

// ------------------------------------------------------------------ harness

function PushProbe() {
  const push = useMothPush()
  return (
    <div>
      <output data-testid="push-status">{push.status}</output>
      <output data-testid="push-permission">{push.permission}</output>
      <button onClick={() => void push.subscribe()}>subscribe</button>
      <button onClick={() => void push.unsubscribe()}>unsubscribe</button>
    </div>
  )
}

const vapidKey = 'AQID' // base64url of [1, 2, 3]
const deviceIdKey = `moth_${cacheNamespace('pk_test')}_push_device`

function renderPush(options: { vapid?: boolean } = {}): {
  client: MothClient
  fake: FakeMoth
} {
  const { client, fake } = fakeClient()
  if (options.vapid !== false) {
    fake.state.projectConfig.push = create(PushConfigSchema, {
      enabled: true,
      webpushVapidPublicKey: vapidKey,
    })
  }
  render(
    <MothProvider client={client} requireAuth={false}>
      <PushProbe />
    </MothProvider>,
  )
  return { client, fake }
}

async function signIn(client: MothClient): Promise<void> {
  await act(async () => {
    await client.signIn({ email: 'ada@example.com', password: 'pw' })
  })
}

function status(): string {
  return screen.getByTestId('push-status').textContent ?? ''
}

async function clickSubscribe(): Promise<void> {
  await act(async () => {
    screen.getByRole('button', { name: 'subscribe' }).click()
  })
}

// -------------------------------------------------------------------- tests

describe('useMothPush', () => {
  it('subscribes: permission, PushManager.subscribe with the VAPID key, RegisterDevice', async () => {
    const { manager, notification } = stubPushEnvironment()
    const { client, fake } = renderPush()
    await signIn(client)
    await waitFor(() => expect(status()).toBe('idle'))

    await clickSubscribe()
    await waitFor(() => expect(status()).toBe('subscribed'))
    expect(notification.requestPermission).toHaveBeenCalledOnce()
    expect(screen.getByTestId('push-permission')).toHaveTextContent('granted')

    // The browser subscription used the decoded base64url VAPID key.
    expect(manager.subscribeOptions).toHaveLength(1)
    const options = manager.subscribeOptions[0]!
    expect(options.userVisibleOnly).toBe(true)
    expect(
      new Uint8Array(options.applicationServerKey as ArrayBuffer),
    ).toEqual(new Uint8Array([1, 2, 3]))

    // The exact RegisterDevice payload.
    const deviceId = window.localStorage.getItem(deviceIdKey)
    expect(deviceId).toBeTruthy()
    expect(fake.calls['registerDevice']).toBe(1)
    const req = fake.state.lastRegisterDevice!
    expect(req.target).toBe(PushTarget.WEBPUSH)
    expect(req.token).toBe(JSON.stringify(manager.subscription))
    expect(JSON.parse(req.token)).toEqual({
      endpoint: 'https://push.example/sub-1',
      keys: { p256dh: 'p256', auth: 'auth' },
    })
    expect(req.deviceId).toBe(deviceId)
    expect(req.permission).toBe(PushPermission.GRANTED)
    expect(req.metadata?.platform).toBe('web')
    expect(req.metadata?.locale).toBe(navigator.language)
    // The RPC carried the session's Bearer token.
    expect(
      fake.headers['registerDevice']?.get('authorization'),
    ).toMatch(/^Bearer /)
  })

  it('reports denied (and never subscribes) when the permission is refused', async () => {
    const { manager, notification } = stubPushEnvironment({
      requestResult: 'denied',
    })
    const { client, fake } = renderPush()
    await signIn(client)
    await waitFor(() => expect(status()).toBe('idle'))

    await clickSubscribe()
    await waitFor(() => expect(status()).toBe('denied'))
    expect(notification.requestPermission).toHaveBeenCalledOnce()
    expect(manager.subscribeOptions).toHaveLength(0)
    expect(fake.calls['registerDevice']).toBeUndefined()
  })

  it('is unavailable (subscribe a no-op) when the project has no VAPID key', async () => {
    const { manager, notification } = stubPushEnvironment()
    const { client, fake } = renderPush({ vapid: false })
    await signIn(client)
    await waitFor(() => expect(status()).toBe('unavailable'))

    await clickSubscribe()
    expect(status()).toBe('unavailable')
    expect(notification.requestPermission).not.toHaveBeenCalled()
    expect(manager.subscribeOptions).toHaveLength(0)
    expect(fake.calls['registerDevice']).toBeUndefined()
  })

  it('is unsupported (subscribe a no-op) without the Push API', async () => {
    // No stubs: jsdom has no PushManager / serviceWorker / Notification.
    const { client, fake } = renderPush()
    await signIn(client)
    expect(status()).toBe('unsupported')

    await clickSubscribe()
    expect(status()).toBe('unsupported')
    expect(fake.calls['registerDevice']).toBeUndefined()
  })

  it('unsubscribes the PushManager subscription and calls UnregisterDevice', async () => {
    const { manager } = stubPushEnvironment()
    const { client, fake } = renderPush()
    await signIn(client)
    await waitFor(() => expect(status()).toBe('idle'))
    await clickSubscribe()
    await waitFor(() => expect(status()).toBe('subscribed'))
    const subscription = manager.subscription!
    const deviceId = window.localStorage.getItem(deviceIdKey)

    await act(async () => {
      screen.getByRole('button', { name: 'unsubscribe' }).click()
    })
    await waitFor(() => expect(status()).toBe('idle'))
    expect(subscription.unsubscribeCalls).toBe(1)
    expect(manager.subscription).toBeNull()
    expect(fake.calls['unregisterDevice']).toBe(1)
    expect(fake.state.lastUnregisterDeviceId).toBe(deviceId)
  })

  it('unregisters on sign-out, before the session drops', async () => {
    stubPushEnvironment()
    const { client, fake } = renderPush()
    await signIn(client)
    await waitFor(() => expect(status()).toBe('idle'))
    await clickSubscribe()
    await waitFor(() => expect(status()).toBe('subscribed'))

    await act(async () => {
      await client.signOut()
    })
    expect(fake.calls['unregisterDevice']).toBe(1)
    expect(fake.state.lastUnregisterDeviceId).toBe(
      window.localStorage.getItem(deviceIdKey),
    )
    // Revoked while the session was still live: the RPC carried the Bearer
    // token (UnregisterDevice requires it).
    expect(
      fake.headers['unregisterDevice']?.get('authorization'),
    ).toMatch(/^Bearer /)
    await waitFor(() => expect(status()).toBe('idle'))
  })

  it('unregisters on sign-out even when this launch never registered', async () => {
    // A previous session registered this installation; this launch's
    // re-registration fails, so nothing marks it registered in memory —
    // sign-out must still revoke the persisted device id.
    const { manager } = stubPushEnvironment({ permission: 'granted' })
    window.localStorage.setItem(deviceIdKey, 'device-from-last-launch')
    const { client, fake } = renderPush()
    manager.subscription = fakeSubscription()
    fake.failNext['registerDevice'] = [
      mothConnectError(Code.Internal, 'INTERNAL', 'boom'),
    ]
    await signIn(client)
    await waitFor(() => expect(fake.calls['registerDevice']).toBe(1))
    expect(status()).toBe('idle')

    await act(async () => {
      await client.signOut()
    })
    expect(fake.calls['unregisterDevice']).toBe(1)
    expect(fake.state.lastUnregisterDeviceId).toBe('device-from-last-launch')
    expect(
      fake.headers['unregisterDevice']?.get('authorization'),
    ).toMatch(/^Bearer /)
  })

  it('skips sign-out revocation when this installation never registered', async () => {
    // No persisted device id and no registration this launch: there is no
    // row to revoke, and sign-out must not mint an id just to unregister.
    stubPushEnvironment()
    window.localStorage.removeItem(deviceIdKey)
    const { client, fake } = renderPush()
    await signIn(client)
    await waitFor(() => expect(status()).toBe('idle'))

    await act(async () => {
      await client.signOut()
    })
    expect(fake.calls['unregisterDevice']).toBeUndefined()
    expect(window.localStorage.getItem(deviceIdKey)).toBeNull()
  })

  it('re-registers an existing subscription on launch while signed in', async () => {
    const { manager } = stubPushEnvironment({ permission: 'granted' })
    const { client, fake } = renderPush()
    // The subscription survived from a previous visit.
    manager.subscription = fakeSubscription()

    await signIn(client)
    await waitFor(() => expect(status()).toBe('subscribed'))
    expect(fake.calls['registerDevice']).toBe(1)
    const req = fake.state.lastRegisterDevice!
    expect(req.token).toBe(JSON.stringify(manager.subscription))
    expect(req.deviceId).toBe(window.localStorage.getItem(deviceIdKey))
    // No permission prompt, no fresh browser subscription: pure upsert.
    expect(manager.subscribeOptions).toHaveLength(0)
  })

  it('treats a RegisterDevice failure as non-fatal', async () => {
    const { manager } = stubPushEnvironment()
    const { client, fake } = renderPush()
    fake.failNext['registerDevice'] = [
      mothConnectError(Code.Internal, 'INTERNAL', 'boom'),
    ]
    await signIn(client)
    await waitFor(() => expect(status()).toBe('idle'))

    await clickSubscribe()
    await waitFor(() => expect(fake.calls['registerDevice']).toBe(1))
    // The browser subscription exists; the registration retries next
    // launch (upsert semantics), and nothing threw meanwhile.
    expect(manager.subscription).not.toBeNull()
    expect(status()).toBe('idle')

    // The next subscribe() attempt re-registers and succeeds.
    await clickSubscribe()
    await waitFor(() => expect(status()).toBe('subscribed'))
    expect(fake.calls['registerDevice']).toBe(2)
  })

  it('keeps sign-in working when the launch re-registration fails', async () => {
    const { manager } = stubPushEnvironment({ permission: 'granted' })
    const { client, fake } = renderPush()
    manager.subscription = fakeSubscription()
    fake.failNext['registerDevice'] = [
      mothConnectError(Code.Internal, 'INTERNAL', 'boom'),
    ]

    await signIn(client)
    await waitFor(() => expect(fake.calls['registerDevice']).toBe(1))
    expect(client.currentState.status).toBe('signedIn')
    expect(status()).toBe('idle')
  })
})
