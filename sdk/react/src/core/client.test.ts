import { create } from '@bufbuild/protobuf'
import { Code, ConnectError } from '@connectrpc/connect'
import { describe, expect, it, vi } from 'vitest'
import { CustomerInfoSchema } from '../gen/moth/billing/v1/billing_pb.js'
import {
  fakeClient,
  fakeJwt,
  fakeMoth,
  fakeUser,
  mothConnectError,
  proCustomerInfo,
  testConfig,
} from '../test/fake.js'
import { MothClient } from './client.js'
import { MothCustomerInfo } from './customerInfo.js'
import {
  MothInvalidCredentialsError,
  MothInvalidRefreshTokenError,
} from './errors.js'
import { InMemoryTokenStore, type StoredSession, type TokenStore } from './tokenStore.js'
import type { MothAuthState } from './user.js'
import { checkoutReturnParam } from './purchase.js'

const user = {
  id: 'user-1',
  email: 'ada@example.com',
  emailVerified: true,
  claims: {},
}

function storedSession(overrides: Partial<StoredSession> = {}): StoredSession {
  return {
    accessToken: fakeJwt(),
    refreshToken: 'rt-stored',
    expiresAtMs: Date.now() + 3600_000,
    user,
    ...overrides,
  }
}

describe('MothClient auth state', () => {
  it('starts loading, restores to signedOut with no stored session', async () => {
    const { client } = fakeClient()
    expect(client.currentState).toEqual({ status: 'loading' })
    const state = await client.restore()
    expect(state.status).toBe('signedOut')
    expect(client.currentState.status).toBe('signedOut')
  })

  it('signIn transitions to signedIn and replays state to new subscribers', async () => {
    const { client } = fakeClient()
    await client.restore()
    const seen: MothAuthState[] = []
    const unsubscribe = client.onAuthStateChanged((s) => seen.push(s))
    expect(seen).toEqual([{ status: 'signedOut' }]) // replay
    const signedIn = await client.signIn({
      email: 'ada@example.com',
      password: 'pw',
    })
    expect(signedIn.id).toBe('user-1')
    expect(seen[seen.length - 1]!.status).toBe('signedIn')
    unsubscribe()
    await client.signOut()
    expect(seen[seen.length - 1]!.status).toBe('signedIn') // unsubscribed
  })

  it('signIn with wrong password maps to MothInvalidCredentialsError', async () => {
    const { client } = fakeClient()
    await client.restore()
    await expect(
      client.signIn({ email: 'ada@example.com', password: 'wrong' }),
    ).rejects.toBeInstanceOf(MothInvalidCredentialsError)
    expect(client.currentState.status).toBe('signedOut')
  })

  it('decodes custom claims from the access token', async () => {
    const { client } = fakeClient({ claims: { role: 'admin' } })
    const u = await client.signIn({ email: 'ada@example.com', password: 'pw' })
    expect(u.claims).toEqual({ role: 'admin' })
  })
})

describe('MothClient signUp policies', () => {
  it('opens a session when the server returns tokens', async () => {
    const { client } = fakeClient({ signUpPolicy: 'session' })
    const result = await client.signUp({ email: 'a@b.co', password: 'pw' })
    expect(result.signedIn).toBe(true)
    expect(result.user?.id).toBe('user-1')
    expect(client.currentState.status).toBe('signedIn')
  })

  it('returns the user without a session when verification is required', async () => {
    const { client } = fakeClient({ signUpPolicy: 'verify' })
    const result = await client.signUp({ email: 'a@b.co', password: 'pw' })
    expect(result.signedIn).toBe(false)
    expect(result.user?.id).toBe('user-1')
    expect(client.currentState.status).toBe('loading') // untouched
  })

  it('returns nothing for enumeration-safe projects', async () => {
    const { client } = fakeClient({ signUpPolicy: 'silent' })
    const result = await client.signUp({ email: 'a@b.co', password: 'pw' })
    expect(result.signedIn).toBe(false)
    expect(result.user).toBeUndefined()
  })
})

describe('MothClient refresh', () => {
  it('serves a fresh access token without any refresh RPC', async () => {
    const { client, fake } = fakeClient({ expiresInSeconds: 3600 })
    await client.signIn({ email: 'a@b.co', password: 'pw' })
    const token = await client.accessToken()
    expect(token).not.toBe('')
    expect(fake.calls['refreshToken']).toBeUndefined()
  })

  it('refreshes proactively when the token expires within the skew', async () => {
    const { client, fake } = fakeClient({ expiresInSeconds: 10 }) // < 30s skew
    await client.signIn({ email: 'a@b.co', password: 'pw' })
    await client.accessToken()
    expect(fake.calls['refreshToken']).toBe(1)
  })

  it('single-flights concurrent refreshes', async () => {
    const { client, fake } = fakeClient({ expiresInSeconds: 10 })
    await client.signIn({ email: 'a@b.co', password: 'pw' })
    fake.gateRefresh()
    const tokens = Promise.all(
      Array.from({ length: 5 }, () => client.accessToken()),
    )
    await Promise.resolve() // let the calls start
    fake.releaseRefresh()
    const resolved = await tokens
    expect(new Set(resolved).size).toBe(1)
    expect(fake.calls['refreshToken']).toBe(1)
  })

  it('retries an authed call exactly once after a server-side INVALID_ACCESS_TOKEN', async () => {
    const { client, fake } = fakeClient()
    await client.signIn({ email: 'a@b.co', password: 'pw' })
    fake.failNext['getMe'] = [
      mothConnectError(Code.Unauthenticated, 'INVALID_ACCESS_TOKEN', 'expired'),
    ]
    const me = await client.getMe()
    expect(me.id).toBe('user-1')
    expect(fake.calls['getMe']).toBe(2)
    expect(fake.calls['refreshToken']).toBe(1)
  })

  it('does not retry twice: a second INVALID_ACCESS_TOKEN surfaces', async () => {
    const { client, fake } = fakeClient()
    await client.signIn({ email: 'a@b.co', password: 'pw' })
    fake.failNext['getMe'] = [
      mothConnectError(Code.Unauthenticated, 'INVALID_ACCESS_TOKEN'),
      mothConnectError(Code.Unauthenticated, 'INVALID_ACCESS_TOKEN'),
    ]
    await expect(client.getMe()).rejects.toMatchObject({
      reason: 'INVALID_ACCESS_TOKEN',
    })
    expect(fake.calls['getMe']).toBe(2)
  })

  it('clears the session when the refresh token is rejected', async () => {
    const { client, fake } = fakeClient({ expiresInSeconds: 10 })
    await client.signIn({ email: 'a@b.co', password: 'pw' })
    fake.state.validRefreshTokens.clear() // server-side revocation
    await expect(client.accessToken()).rejects.toBeInstanceOf(
      MothInvalidRefreshTokenError,
    )
    expect(client.currentState.status).toBe('signedOut')
  })

  it('keeps the session on a network failure during refresh', async () => {
    const { client, fake } = fakeClient({ expiresInSeconds: 10 })
    await client.signIn({ email: 'a@b.co', password: 'pw' })
    fake.failNext['refreshToken'] = [
      new ConnectError('unreachable', Code.Unavailable),
    ]
    await expect(client.accessToken()).rejects.toMatchObject({
      name: 'MothNetworkError',
    })
    expect(client.currentState.status).toBe('signedIn')
    // The next call succeeds and refreshes normally.
    await client.accessToken()
    expect(fake.calls['refreshToken']).toBe(2)
  })

  it('a refresh in flight during signOut cannot resurrect the session', async () => {
    const { client, fake } = fakeClient({ expiresInSeconds: 10 })
    await client.signIn({ email: 'a@b.co', password: 'pw' })
    fake.gateRefresh()
    const pending = client.accessToken() // starts a gated refresh
    pending.catch(() => undefined)
    const signOutDone = client.signOut() // settles the refresh first
    fake.releaseRefresh()
    await signOutDone
    await pending.catch(() => undefined)
    expect(client.currentState.status).toBe('signedOut')
    await expect(client.accessToken()).rejects.toThrow('not signed in')
  })

  it('a refresh started while the signOut RPC is in flight cannot resurrect the session', async () => {
    const { client, fake } = fakeClient({ expiresInSeconds: 10 })
    await client.signIn({ email: 'a@b.co', password: 'pw' })
    const release = fake.gate('signOut')
    const signOutDone = client.signOut()
    await Promise.resolve() // the revocation RPC is now in flight
    // A background caller kicks off a refresh mid-sign-out; it succeeds and
    // briefly opens a session — signOut's final settle + clear must win.
    await client.accessToken().catch(() => undefined)
    release()
    await signOutDone
    expect(client.currentState.status).toBe('signedOut')
    await expect(client.accessToken()).rejects.toThrow('not signed in')
  })
})

describe('MothClient restore paths', () => {
  const config = { storage: 'memory' as const }

  function withStore(session: StoredSession | null): TokenStore {
    const store = new InMemoryTokenStore()
    if (session !== null) store.save(session)
    return store
  }

  function restoreClient(session: StoredSession | null, expiresInSeconds = 3600) {
    const { client, fake } = fakeClient({
      expiresInSeconds,
      config: { ...config, storage: withStore(session) },
    })
    return { client, fake }
  }

  it('signs in without network when the stored token is fresh', async () => {
    const { client, fake } = restoreClient(storedSession())
    const state = await client.restore()
    expect(state.status).toBe('signedIn')
    expect(Object.keys(fake.calls)).toHaveLength(0) // zero RPCs
  })

  it('refreshes an expired stored token', async () => {
    const { client, fake } = restoreClient(
      storedSession({ expiresAtMs: Date.now() - 1000, refreshToken: 'rt-old' }),
    )
    fake.state.validRefreshTokens.add('rt-old')
    const state = await client.restore()
    expect(state.status).toBe('signedIn')
    expect(fake.calls['refreshToken']).toBe(1)
  })

  it('signs out when the stored refresh token is rejected', async () => {
    const { client } = restoreClient(
      storedSession({ expiresAtMs: Date.now() - 1000, refreshToken: 'rt-bad' }),
    )
    const state = await client.restore()
    expect(state.status).toBe('signedOut')
  })

  it('stays signed in on the stored snapshot after a transient failure', async () => {
    const { client, fake } = restoreClient(
      storedSession({ expiresAtMs: Date.now() - 1000, refreshToken: 'rt-old' }),
    )
    fake.state.validRefreshTokens.add('rt-old')
    fake.failNext['refreshToken'] = [
      new ConnectError('unreachable', Code.Unavailable),
    ]
    const state = await client.restore()
    expect(state.status).toBe('signedIn')
  })

  it('signs out (never wedges in loading) when the store throws', async () => {
    const broken: TokenStore = {
      load: () => {
        throw new Error('storage exploded')
      },
      save: () => undefined,
      clear: () => undefined,
    }
    const { client } = fakeClient({ config: { storage: broken } })
    const state = await client.restore()
    expect(state.status).toBe('signedOut')
  })

  it('treats a corrupted localStorage entry as signed out and clears it', async () => {
    window.localStorage.setItem('moth_session_pk_test', '{not json')
    const { client } = fakeClient({ config: { storage: 'local' } })
    const state = await client.restore()
    expect(state.status).toBe('signedOut')
    expect(window.localStorage.getItem('moth_session_pk_test')).toBeNull()
  })

  it('continues in memory when localStorage.setItem throws', async () => {
    const setItem = vi
      .spyOn(window.localStorage, 'setItem')
      .mockImplementation(() => {
        throw new Error('quota exceeded')
      })
    try {
      const { client } = fakeClient({ config: { storage: 'local' } })
      await client.restore()
      const u = await client.signIn({ email: 'a@b.co', password: 'pw' })
      expect(u.id).toBe('user-1')
      expect(client.currentState.status).toBe('signedIn')
      expect(await client.accessToken()).not.toBe('')
    } finally {
      setItem.mockRestore()
    }
  })
})

describe('MothClient session-generation guards', () => {
  // Waits (in zero-delay ticks) until `cond` holds — for racing an
  // in-flight RPC against a concurrent state change deterministically.
  async function until(cond: () => boolean): Promise<void> {
    for (let i = 0; i < 200 && !cond(); i++) {
      await new Promise((resolve) => setTimeout(resolve, 0))
    }
    if (!cond()) throw new Error('condition never reached')
  }

  it('a stale refresh rejection from restore cannot clear a session opened meanwhile', async () => {
    const store = new InMemoryTokenStore()
    store.save(
      storedSession({
        expiresAtMs: Date.now() - 1000,
        refreshToken: 'rt-stale', // NOT valid server-side: will be rejected
        user: { ...user, id: 'user-old' },
      }),
    )
    const { client, fake } = fakeClient({ config: { storage: store } })
    fake.gateRefresh()
    const restoring = client.restore() // kicks off the gated refresh
    await until(() => (fake.calls['refreshToken'] ?? 0) === 1)
    // A fresh sign-in completes while the stale refresh is in flight.
    await client.signIn({ email: 'ada@example.com', password: 'pw' })
    expect(client.currentUser?.id).toBe('user-1')
    fake.releaseRefresh() // the stale refresh now rejects
    await restoring
    // The rejection belongs to the OLD session: the new one is intact.
    expect(client.currentState.status).toBe('signedIn')
    expect(client.currentUser?.id).toBe('user-1')
    expect(await client.accessToken()).not.toBe('')
    expect(store.load()).not.toBeNull() // storage not wiped
  })

  it('a stale refresh success from restore cannot overwrite a new session', async () => {
    const store = new InMemoryTokenStore()
    store.save(
      storedSession({ expiresAtMs: Date.now() - 1000, refreshToken: 'rt-old' }),
    )
    const { client, fake } = fakeClient({ config: { storage: store } })
    fake.state.validRefreshTokens.add('rt-old') // still valid server-side
    fake.gateRefresh()
    const restoring = client.restore()
    await until(() => (fake.calls['refreshToken'] ?? 0) === 1)
    await client.signIn({ email: 'ada@example.com', password: 'pw' })
    const tokenAfterSignIn = await client.accessToken()
    fake.releaseRefresh() // the stale refresh now SUCCEEDS server-side
    await restoring
    // Its tokens must not be committed over the new session's.
    expect(await client.accessToken()).toBe(tokenAfterSignIn)
    expect(client.currentState.status).toBe('signedIn')
    expect(store.load()?.accessToken).toBe(tokenAfterSignIn)
  })
})

describe('stale customer-info responses', () => {
  // A transport that holds the FIRST getCustomerInfo response in flight
  // until released, so a session change can be raced against it.
  function gatedCustomerInfo(fake: ReturnType<typeof fakeMoth>) {
    let release!: () => void
    const gate = new Promise<void>((resolve) => (release = resolve))
    let gated = false
    const transport: typeof fake.transport = {
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
    return { transport, release }
  }

  async function until(cond: () => boolean): Promise<void> {
    for (let i = 0; i < 200 && !cond(); i++) {
      await new Promise((resolve) => setTimeout(resolve, 0))
    }
    if (!cond()) throw new Error('condition never reached')
  }

  it('drops an in-flight GetCustomerInfo that lands after sign-out', async () => {
    const fake = fakeMoth()
    const { transport, release } = gatedCustomerInfo(fake)
    const client = new MothClient(testConfig, {
      transport,
      navigate: () => undefined,
    })
    fake.state.customerInfo = proCustomerInfo()
    await client.signIn({ email: 'a@b.co', password: 'pw' })
    const pending = client.getCustomerInfo()
    await until(() => (fake.calls['getCustomerInfo'] ?? 0) === 1)
    await client.signOut()
    let emissions = 0
    client.onEntitlementsChanged(() => emissions++)
    emissions = 0 // drop the replay
    release()
    const info = await pending
    // The stale response is dropped: the caller gets the (free) truth for
    // the current session and nothing is published.
    expect(info.hasEntitlement('pro')).toBe(false)
    expect(client.currentCustomerInfo.equals(MothCustomerInfo.free())).toBe(true)
    expect(emissions).toBe(0)
  })

  it("checkout-return polling never publishes a previous user's entitlements", async () => {
    window.history.replaceState(null, '', `/page?${checkoutReturnParam}=success`)
    const fake = fakeMoth()
    const { transport, release } = gatedCustomerInfo(fake)
    const client = new MothClient(testConfig, {
      transport,
      navigate: () => undefined,
      checkoutPollIntervalMs: 1,
    })
    // User A completed checkout; the first poll (A's, answering 'pro') is
    // held in flight.
    fake.state.customerInfo = proCustomerInfo()
    await client.signIn({ email: 'a@example.com', password: 'pw' })
    const returning = client.handleCheckoutReturn()
    await until(() => (fake.calls['getCustomerInfo'] ?? 0) === 1)
    // A signs out; B (free) signs in while A's poll response is in flight.
    await client.signOut()
    fake.state.user = fakeUser({ id: 'user-b', email: 'b@example.com' })
    fake.state.customerInfo = create(CustomerInfoSchema, {})
    await client.signIn({ email: 'b@example.com', password: 'pw' })
    release()
    const result = await returning
    // B's own polls answer free: no purchase is reported for B, and A's
    // stale poll response was never published as B's entitlements.
    expect(result).toEqual({ status: 'pending' })
    expect(client.currentUser?.id).toBe('user-b')
    expect(client.currentCustomerInfo.hasEntitlement('pro')).toBe(false)
  })
})

describe('token store namespacing', () => {
  it('persists under moth_session_<publishableKey>', async () => {
    const { client } = fakeClient({ config: { storage: 'local' } })
    await client.restore()
    await client.signIn({ email: 'a@b.co', password: 'pw' })
    expect(window.localStorage.getItem('moth_session_pk_test')).not.toBeNull()
    // A second project on the same origin uses its own slot.
    const { client: other } = fakeClient({
      config: { storage: 'local', publishableKey: 'pk_other' },
    })
    const state = await other.restore()
    expect(state.status).toBe('signedOut')
  })
})

describe('MothClient sign-out ordering', () => {
  it('emits signedOut before the free customer-info reset', async () => {
    const { client, fake } = fakeClient()
    fake.state.customerInfo = proCustomerInfo()
    await client.signIn({ email: 'a@b.co', password: 'pw' })
    await client.getCustomerInfo()
    const order: string[] = []
    client.onAuthStateChanged((s) => order.push(`auth:${s.status}`))
    client.onEntitlementsChanged((info) =>
      order.push(`info:${info.activeEntitlements.length}`),
    )
    order.length = 0 // drop the replays
    await client.signOut()
    expect(order).toEqual(['auth:signedOut', 'info:0'])
  })

  it('signOut revokes server-side best effort and always clears locally', async () => {
    const { client, fake } = fakeClient()
    await client.signIn({ email: 'a@b.co', password: 'pw' })
    fake.failNext['signOut'] = [new ConnectError('boom', Code.Internal)]
    await client.signOut()
    expect(client.currentState.status).toBe('signedOut')
    expect(fake.calls['signOut']).toBe(1)
  })
})

describe('MothClient metadata', () => {
  it('attaches the moth headers and Bearer token', async () => {
    const { client, fake } = fakeClient()
    await client.signIn({ email: 'a@b.co', password: 'pw' })
    await client.getMe()
    const headers = fake.headers['getMe']!
    expect(headers.get('x-moth-key')).toBe('pk_test')
    expect(headers.get('x-moth-platform')).toBe('web')
    expect(headers.get('x-moth-sdk-version')).toBe('0.0.0')
    expect(headers.get('x-moth-language')).not.toBeNull()
    expect(headers.get('authorization')).toMatch(/^Bearer /)
    // Publishable-key-only calls carry no Bearer while signed out.
    await client.signOut()
    await client.getOfferings()
    expect(fake.headers['getOfferings']!.get('authorization')).toBeNull()
  })
})

describe('checkout return', () => {
  function setUrl(query: string) {
    window.history.replaceState(null, '', `/page${query}`)
  }

  it('returns null without the marker', async () => {
    setUrl('')
    const { client } = fakeClient()
    expect(await client.handleCheckoutReturn()).toBeNull()
  })

  it('cancel marker resolves cancelled and strips the parameter', async () => {
    setUrl(`?${checkoutReturnParam}=cancel&keep=1`)
    const { client } = fakeClient()
    const result = await client.handleCheckoutReturn()
    expect(result).toEqual({ status: 'cancelled' })
    expect(window.location.search).toBe('?keep=1')
  })

  it('success marker polls customer info until the entitlement lands', async () => {
    setUrl(`?${checkoutReturnParam}=success`)
    const { client, fake } = fakeClient()
    await client.signIn({ email: 'a@b.co', password: 'pw' })
    // The webhook lands before the second poll.
    let polls = 0
    const free = fake.state.customerInfo
    const pro = proCustomerInfo()
    Object.defineProperty(fake.state, 'customerInfo', {
      configurable: true,
      get: () => (++polls >= 2 ? pro : free),
    })
    const result = await client.handleCheckoutReturn()
    expect(result).toEqual({ status: 'purchased' })
    expect(window.location.search).toBe('')
    expect(client.currentCustomerInfo.hasEntitlement('pro')).toBe(true)
  })

  it('resolves pending when the webhook never lands within the poll budget', async () => {
    setUrl(`?${checkoutReturnParam}=success`)
    const { client, fake } = fakeClient()
    await client.signIn({ email: 'a@b.co', password: 'pw' })
    const result = await client.handleCheckoutReturn()
    expect(result).toEqual({ status: 'pending' })
    expect(fake.calls['getCustomerInfo']).toBe(5)
  })
})

describe('purchase', () => {
  const product = {
    identifier: 'monthly',
    displayName: 'Monthly',
    appleProductId: '',
    googleProductId: '',
    stripePriceId: 'price_123',
    billingPeriod: 'P1M',
    priceAmountMicros: 9_990_000,
    currency: 'USD',
    trialPeriod: '',
    introPriceAmountMicros: 0,
    introPeriod: '',
    entitlements: ['pro'],
    sortOrder: 0,
    highlighted: false,
  }

  it('creates a checkout session with default return URLs and redirects', async () => {
    window.history.replaceState(null, '', '/pricing')
    const navigated: string[] = []
    const { client, fake } = fakeClient({
      client: { navigate: (url) => navigated.push(url) },
    })
    await client.signIn({ email: 'a@b.co', password: 'pw' })
    const result = await client.purchase(product)
    expect(result).toEqual({ status: 'redirect' })
    expect(navigated).toEqual(['https://checkout.stripe.test/session'])
    const req = fake.state.lastCheckoutRequest!
    expect(req.productIdentifier).toBe('monthly')
    expect(req.successUrl).toContain(`${checkoutReturnParam}=success`)
    expect(req.cancelUrl).toContain(`${checkoutReturnParam}=cancel`)
  })

  it('resolves an error result (never throws) for server failures', async () => {
    const { client, fake } = fakeClient()
    await client.signIn({ email: 'a@b.co', password: 'pw' })
    fake.failNext['createCheckoutSession'] = [
      mothConnectError(
        Code.FailedPrecondition,
        'BILLING_NOT_CONFIGURED',
        'no stripe credentials',
      ),
    ]
    const result = await client.purchase(product)
    expect(result).toMatchObject({
      status: 'error',
      reason: 'BILLING_NOT_CONFIGURED',
    })
  })

  it('refuses a tier without a Stripe price locally', async () => {
    const { client, fake } = fakeClient()
    await client.signIn({ email: 'a@b.co', password: 'pw' })
    const result = await client.purchase({ ...product, stripePriceId: '' })
    expect(result).toMatchObject({
      status: 'error',
      reason: 'PRODUCT_NOT_ON_STORE',
    })
    expect(fake.calls['createCheckoutSession']).toBeUndefined()
  })

  it('manageBilling redirects to the portal', async () => {
    const navigated: string[] = []
    const { client } = fakeClient({
      client: { navigate: (url) => navigated.push(url) },
    })
    await client.signIn({ email: 'a@b.co', password: 'pw' })
    await client.manageBilling()
    expect(navigated).toEqual(['https://billing.stripe.test/portal'])
  })
})

describe('customer info', () => {
  it('is always valid: free while signed out', () => {
    const { client } = fakeClient()
    expect(client.currentCustomerInfo.equals(MothCustomerInfo.free())).toBe(true)
    expect(client.currentCustomerInfo.hasEntitlement('pro')).toBe(false)
  })

  it('getCustomerInfo throws while signed out', async () => {
    const { client } = fakeClient()
    await client.restore()
    await expect(client.getCustomerInfo()).rejects.toThrow('not signed in')
  })

  it('always re-emits server results, even when unchanged', async () => {
    const { client } = fakeClient()
    await client.signIn({ email: 'a@b.co', password: 'pw' })
    let emissions = 0
    client.onEntitlementsChanged(() => emissions++)
    emissions = 0 // drop the replay
    await client.getCustomerInfo()
    await client.getCustomerInfo()
    expect(emissions).toBe(2)
  })
})

describe('MothClient misc', () => {
  it('oauthStartUrl requires the project slug and builds the start URL', () => {
    const { client } = fakeClient()
    expect(() => client.oauthStartUrl('google')).toThrow('projectSlug')
    const { client: withSlug } = fakeClient({
      config: { projectSlug: 'myapp' },
    })
    const url = new URL(
      withSlug.oauthStartUrl('google', 'https://app.example.com/login'),
    )
    expect(url.pathname).toBe('/oauth/google/start')
    expect(url.searchParams.get('project')).toBe('myapp')
    expect(url.searchParams.get('redirect')).toBe('https://app.example.com/login')
  })

  it('deleteAccount clears the session', async () => {
    const { client, fake } = fakeClient()
    await client.signIn({ email: 'a@b.co', password: 'pw' })
    await client.deleteAccount({ password: 'pw' })
    expect(fake.calls['deleteAccount']).toBe(1)
    expect(client.currentState.status).toBe('signedOut')
  })

  it('changePassword continues the session on a fresh token pair', async () => {
    const { client, fake } = fakeClient()
    await client.signIn({ email: 'a@b.co', password: 'pw' })
    const before = await client.accessToken()
    await client.changePassword({ currentPassword: 'pw', newPassword: 'pw2' })
    expect(client.currentState.status).toBe('signedIn')
    expect(fake.calls['changePassword']).toBe(1)
    expect(await client.accessToken()).not.toBe(before)
  })
})

describe('client construction', () => {
  it('exposes config and locale', () => {
    const client = new MothClient(
      { endpoint: 'https://m.test', publishableKey: 'pk_x', locale: 'fr' },
      { transport: undefined as never },
    )
    expect(client.currentLocale).toBe('fr')
  })
})
