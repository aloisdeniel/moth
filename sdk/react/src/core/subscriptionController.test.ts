import { create } from '@bufbuild/protobuf'
import { describe, expect, it } from 'vitest'
import { CustomerInfoSchema } from '../gen/moth/billing/v1/billing_pb.js'
import {
  fakeClient,
  fakeMoth,
  fakeUser,
  proCustomerInfo,
  testConfig,
} from '../test/fake.js'
import { MothClient } from './client.js'
import {
  cacheNamespace,
  customerInfoCacheKey,
  MemoryStorage,
} from './cache.js'
import { MothCustomerInfo } from './customerInfo.js'
import { MothSubscriptionController } from './subscriptionController.js'

const namespace = cacheNamespace('pk_test')

function flush(): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, 0))
}

describe('MothSubscriptionController', () => {
  it('persists billing results per user and primes from the cache on sign-in', async () => {
    const storage = new MemoryStorage()
    const { client, fake } = fakeClient()
    fake.state.customerInfo = proCustomerInfo()
    const controller = new MothSubscriptionController(client, { storage })
    controller.start()

    await client.signIn({ email: 'a@b.co', password: 'pw' })
    await flush() // background getCustomerInfo lands
    expect(client.currentCustomerInfo.hasEntitlement('pro')).toBe(true)
    expect(
      storage.getItem(customerInfoCacheKey(namespace, 'user-1')),
    ).not.toBeNull()

    // Sign out: reset to free — and the auth event lands before the free
    // reset, so the outgoing user's cached snapshot survives.
    await client.signOut()
    expect(client.currentCustomerInfo.equals(MothCustomerInfo.free())).toBe(true)
    expect(
      storage.getItem(customerInfoCacheKey(namespace, 'user-1')),
    ).not.toBeNull()

    // Sign back in with the server now unreachable for billing: the cached
    // snapshot primes instantly.
    const { Code, ConnectError } = await import('@connectrpc/connect')
    fake.failNext['getCustomerInfo'] = Array.from(
      { length: 5 },
      () => new ConnectError('down', Code.Unavailable),
    )
    const emissions: boolean[] = []
    client.onEntitlementsChanged((info) =>
      emissions.push(info.hasEntitlement('pro')),
    )
    await client.signIn({ email: 'a@b.co', password: 'pw' })
    await flush()
    expect(emissions).toContain(true) // primed from the per-user cache
    controller.dispose()
  })

  it('drops a stale GetCustomerInfo issued under a previous user (no cross-user leak)', async () => {
    const fake = fakeMoth()
    // Hold the FIRST getCustomerInfo response (user A's) in flight.
    let release!: () => void
    const gate = new Promise<void>((resolve) => (release = resolve))
    let gated = false
    const delayed: typeof fake.transport = {
      async unary(method, ...rest) {
        const response = fake.transport.unary(method, ...rest)
        const name = (method as { name?: string }).name ?? ''
        if (/getCustomerInfo/i.test(name) && !gated) {
          gated = true
          const resolved = await response
          await gate
          return resolved
        }
        return response
      },
      stream: fake.transport.stream.bind(fake.transport),
    }
    const client = new MothClient(testConfig, {
      transport: delayed,
      navigate: () => undefined,
    })
    const storage = new MemoryStorage()
    const controller = new MothSubscriptionController(client, { storage })
    controller.start()

    // User A holds 'pro'; the controller's background fetch is now gated.
    fake.state.user = fakeUser({ id: 'user-a', email: 'a@example.com' })
    fake.state.customerInfo = proCustomerInfo()
    await client.signIn({ email: 'a@example.com', password: 'pw' })
    await flush()

    // A signs out; B signs in quickly. B is free — and B's own background
    // fetch fails (offline right after sign-in), so nothing overwrites the
    // leak if it happens.
    await client.signOut()
    fake.state.user = fakeUser({ id: 'user-b', email: 'b@example.com' })
    fake.state.customerInfo = create(CustomerInfoSchema, {})
    const { Code, ConnectError } = await import('@connectrpc/connect')
    fake.failNext['getCustomerInfo'] = [
      new ConnectError('down', Code.Unavailable),
    ]
    await client.signIn({ email: 'b@example.com', password: 'pw' })
    await flush()
    expect(client.currentUser?.id).toBe('user-b')

    // A's slow response finally lands — it must be dropped, not published
    // as B's entitlements nor persisted under B's cache key.
    release()
    await flush()
    await flush()
    expect(client.currentCustomerInfo.hasEntitlement('pro')).toBe(false)
    expect(
      storage.getItem(customerInfoCacheKey(namespace, 'user-b')),
    ).toBeNull()
    controller.dispose()
  })

  it('never leaks entitlements across users', async () => {
    const storage = new MemoryStorage()
    const { client, fake } = fakeClient()
    fake.state.customerInfo = proCustomerInfo()
    const controller = new MothSubscriptionController(client, { storage })
    controller.start()
    await client.signIn({ email: 'a@b.co', password: 'pw' })
    await flush()
    await client.signOut()

    // A different user signs in; the server reports no entitlements.
    fake.state.user = fakeUser({ id: 'user-2', email: 'bob@example.com' })
    fake.state.customerInfo = create(CustomerInfoSchema, {})
    await client.signIn({ email: 'bob@example.com', password: 'pw' })
    // user-2 has no cache entry: nothing primes, and the fresh fetch says free.
    await flush()
    expect(client.currentCustomerInfo.hasEntitlement('pro')).toBe(false)
    // user-1's cache is untouched.
    expect(
      storage.getItem(customerInfoCacheKey(namespace, 'user-1')),
    ).not.toBeNull()
    expect(storage.getItem(customerInfoCacheKey(namespace, 'user-2'))).not.toBeNull()
    controller.dispose()
  })
})
