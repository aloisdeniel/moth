// In-process fake moth server for unit and component tests: the real
// connect router transport running the three services over an in-memory
// protocol stack, so errors, details and metadata behave like the wire.

import { clone, create } from '@bufbuild/protobuf'
import { timestampFromDate } from '@bufbuild/protobuf/wkt'
import {
  Code,
  ConnectError,
  createRouterTransport,
  type Transport,
} from '@connectrpc/connect'
import {
  AuthService,
  TokenPairSchema,
  UserSchema,
  type TokenPair,
  type User,
} from '../gen/moth/auth/v1/auth_pb.js'
import {
  ConfigService,
  type GetProjectConfigResponse,
} from '../gen/moth/auth/v1/config_pb.js'
import {
  BillingService,
  CustomerInfoSchema,
  EntitlementSource,
  OfferingSchema,
  PaywallLayout,
  PaywallSchema,
  type CustomerInfo,
  type Offering,
  type Paywall,
} from '../gen/moth/billing/v1/billing_pb.js'
import {
  PushService,
  type RegisterDeviceRequest,
} from '../gen/moth/push/v1/push_pb.js'
import { MothClient, type MothClientOptions } from '../core/client.js'
import type { MothConfig } from '../core/config.js'
import { encodeErrorInfo, mothErrorDomain } from '../core/errors.js'

/** A ConnectError carrying a moth ErrorInfo reason, like the real server. */
export function mothConnectError(
  code: Code,
  reason: string,
  message = 'nope',
): ConnectError {
  const err = new ConnectError(message, code)
  err.details.push({
    type: 'google.rpc.ErrorInfo',
    value: encodeErrorInfo(reason, mothErrorDomain),
  })
  return err
}

/** A minimal unsigned JWT whose payload carries moth custom claims. */
export function fakeJwt(claims: Record<string, unknown> = {}, seq = 0): string {
  const b64 = (obj: unknown) =>
    btoa(JSON.stringify(obj)).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
  return `${b64({ alg: 'ES256' })}.${b64({ sub: 'user-1', seq, claims })}.sig`
}

/** A CustomerInfo carrying one active store entitlement. */
export function proCustomerInfo(identifier = 'pro'): CustomerInfo {
  return create(CustomerInfoSchema, {
    activeEntitlements: [
      {
        identifier,
        source: EntitlementSource.STORE,
        productIdentifier: 'monthly',
      },
    ],
  })
}

export interface FakeMothOptions {
  /** Access-token lifetime handed out with every token pair (seconds). */
  expiresInSeconds?: number
  /** Sign-up policy: session, verify (user only) or nothing (enumeration-safe). */
  signUpPolicy?: 'session' | 'verify' | 'silent'
  /** Custom claims embedded in issued access tokens. */
  claims?: Record<string, unknown>
}

export interface FakeMoth {
  transport: Transport
  calls: Record<string, number>
  /** Last request headers seen, by RPC name. */
  headers: Record<string, Headers>
  /** Overridable per-RPC error queues: shift()ed before the handler runs. */
  failNext: Record<string, ConnectError[]>
  /** Gates the refresh RPC: when set, refreshes await it. */
  refreshGate: { promise: Promise<void>; resolve: () => void } | null
  gateRefresh(): void
  releaseRefresh(): void
  /** Generic per-RPC gates: the handler awaits the gate before responding. */
  gates: Record<string, Promise<void>>
  gate(name: string): () => void
  /** Mutable server-side state. */
  state: {
    user: User
    tokenCounter: number
    expiresInSeconds: number
    customerInfo: CustomerInfo
    offering: Offering
    paywall: Paywall | null
    projectConfig: GetProjectConfigResponse
    checkoutUrl: string
    portalUrl: string
    lastCheckoutRequest: { productIdentifier: string; successUrl: string; cancelUrl: string } | null
    /** Refresh tokens that are still valid. */
    validRefreshTokens: Set<string>
    /** The last RegisterDevice request, verbatim. */
    lastRegisterDevice: RegisterDeviceRequest | null
    /** The device id of the last UnregisterDevice request. */
    lastUnregisterDeviceId: string | null
  }
  issueTokens(): TokenPair
}

export function fakeUser(
  overrides: {
    id?: string
    email?: string
    emailVerified?: boolean
    displayName?: string
    avatarUrl?: string
  } = {},
): User {
  return create(UserSchema, {
    id: overrides.id ?? 'user-1',
    email: overrides.email ?? 'ada@example.com',
    emailVerified: overrides.emailVerified ?? true,
    displayName: overrides.displayName ?? 'Ada',
    avatarUrl: overrides.avatarUrl ?? '',
    createTime: timestampFromDate(new Date('2026-01-01T00:00:00Z')),
  })
}

export function fakeMoth(options: FakeMothOptions = {}): FakeMoth {
  const calls: Record<string, number> = {}
  const headers: Record<string, Headers> = {}
  const failNext: Record<string, ConnectError[]> = {}
  const claims = options.claims ?? {}

  const fake: FakeMoth = {
    transport: undefined as unknown as Transport,
    calls,
    headers,
    failNext,
    refreshGate: null,
    gateRefresh() {
      let resolve!: () => void
      const promise = new Promise<void>((r) => (resolve = r))
      fake.refreshGate = { promise, resolve }
    },
    releaseRefresh() {
      fake.refreshGate?.resolve()
      fake.refreshGate = null
    },
    gates: {},
    gate(name: string) {
      let release!: () => void
      fake.gates[name] = new Promise<void>((r) => (release = r))
      return () => {
        release()
        delete fake.gates[name]
      }
    },
    state: {
      user: fakeUser(),
      tokenCounter: 0,
      expiresInSeconds: options.expiresInSeconds ?? 3600,
      customerInfo: create(CustomerInfoSchema, {}),
      offering: create(OfferingSchema, {
        identifier: 'default',
        isDefault: true,
      }),
      paywall: null,
      projectConfig: create(ConfigService.method.getProjectConfig.output, {
        google: { enabled: false },
        apple: { enabled: false },
        passwordMinLength: 8,
        signUpOpen: true,
      }),
      checkoutUrl: 'https://checkout.stripe.test/session',
      portalUrl: 'https://billing.stripe.test/portal',
      lastCheckoutRequest: null,
      validRefreshTokens: new Set(),
      lastRegisterDevice: null,
      lastUnregisterDeviceId: null,
    },
    issueTokens,
  }

  function issueTokens(): TokenPair {
    fake.state.tokenCounter++
    const refreshToken = `rt-${fake.state.tokenCounter}`
    fake.state.validRefreshTokens.add(refreshToken)
    return create(TokenPairSchema, {
      accessToken: fakeJwt(claims, fake.state.tokenCounter),
      refreshToken,
      expiresIn: BigInt(fake.state.expiresInSeconds),
    })
  }

  function track(name: string, header: Headers): void {
    calls[name] = (calls[name] ?? 0) + 1
    headers[name] = header
    const queued = failNext[name]
    if (queued !== undefined && queued.length > 0) {
      throw queued.shift()
    }
  }

  fake.transport = createRouterTransport(({ service }) => {
    service(AuthService, {
      signUp(_req, ctx) {
        track('signUp', ctx.requestHeader)
        const policy = options.signUpPolicy ?? 'session'
        if (policy === 'silent') return {}
        if (policy === 'verify') return { user: fake.state.user }
        return { user: fake.state.user, tokens: issueTokens() }
      },
      signIn(req, ctx) {
        track('signIn', ctx.requestHeader)
        if (req.password === 'wrong') {
          throw mothConnectError(
            Code.Unauthenticated,
            'INVALID_CREDENTIALS',
            'invalid email or password',
          )
        }
        return { user: fake.state.user, tokens: issueTokens() }
      },
      async refreshToken(req, ctx) {
        track('refreshToken', ctx.requestHeader)
        if (fake.refreshGate !== null) await fake.refreshGate.promise
        if (!fake.state.validRefreshTokens.has(req.refreshToken)) {
          throw mothConnectError(
            Code.Unauthenticated,
            'INVALID_REFRESH_TOKEN',
            'refresh token revoked',
          )
        }
        fake.state.validRefreshTokens.delete(req.refreshToken)
        return { user: fake.state.user, tokens: issueTokens() }
      },
      async signOut(_req, ctx) {
        track('signOut', ctx.requestHeader)
        const gate = fake.gates['signOut']
        if (gate !== undefined) await gate
        return {}
      },
      getMe(_req, ctx) {
        track('getMe', ctx.requestHeader)
        return { user: fake.state.user }
      },
      updateMe(req, ctx) {
        track('updateMe', ctx.requestHeader)
        const user = clone(UserSchema, fake.state.user)
        if (req.displayName !== undefined) user.displayName = req.displayName
        if (req.avatarUrl !== undefined) user.avatarUrl = req.avatarUrl
        fake.state.user = user
        return { user }
      },
      changePassword(_req, ctx) {
        track('changePassword', ctx.requestHeader)
        return { tokens: issueTokens() }
      },
      requestEmailVerification(_req, ctx) {
        track('requestEmailVerification', ctx.requestHeader)
        return {}
      },
      confirmEmailVerification(_req, ctx) {
        track('confirmEmailVerification', ctx.requestHeader)
        return {}
      },
      requestPasswordReset(_req, ctx) {
        track('requestPasswordReset', ctx.requestHeader)
        return {}
      },
      confirmPasswordReset(_req, ctx) {
        track('confirmPasswordReset', ctx.requestHeader)
        return {}
      },
      requestEmailChange(_req, ctx) {
        track('requestEmailChange', ctx.requestHeader)
        return {}
      },
      confirmEmailChange(_req, ctx) {
        track('confirmEmailChange', ctx.requestHeader)
        return {}
      },
      signInWithOAuth(_req, ctx) {
        track('signInWithOAuth', ctx.requestHeader)
        return { user: fake.state.user, tokens: issueTokens() }
      },
      exchangeOAuthCode(_req, ctx) {
        track('exchangeOAuthCode', ctx.requestHeader)
        return { user: fake.state.user, tokens: issueTokens() }
      },
      unlinkIdentity(_req, ctx) {
        track('unlinkIdentity', ctx.requestHeader)
        return {}
      },
      deleteAccount(_req, ctx) {
        track('deleteAccount', ctx.requestHeader)
        return {}
      },
    })
    service(ConfigService, {
      getProjectConfig(req, ctx) {
        track('getProjectConfig', ctx.requestHeader)
        const resp = clone(
          ConfigService.method.getProjectConfig.output,
          fake.state.projectConfig,
        )
        // Revision-matched bodies are omitted, like the real server.
        if (
          resp.theme !== undefined &&
          req.knownThemeRevision !== '' &&
          req.knownThemeRevision === resp.theme.revisionId
        ) {
          resp.theme = undefined
        }
        if (
          resp.copy !== undefined &&
          req.knownCopyRevision !== '' &&
          req.knownCopyRevision === resp.copy.copyRevision
        ) {
          resp.copy.messages = {}
        }
        return resp
      },
    })
    service(BillingService, {
      getCustomerInfo(_req, ctx) {
        track('getCustomerInfo', ctx.requestHeader)
        return { customerInfo: fake.state.customerInfo }
      },
      submitPurchase(_req, ctx) {
        track('submitPurchase', ctx.requestHeader)
        return { customerInfo: fake.state.customerInfo }
      },
      restorePurchases(_req, ctx) {
        track('restorePurchases', ctx.requestHeader)
        return { customerInfo: fake.state.customerInfo }
      },
      getOfferings(_req, ctx) {
        track('getOfferings', ctx.requestHeader)
        return { offering: fake.state.offering }
      },
      getPaywall(req, ctx) {
        track('getPaywall', ctx.requestHeader)
        const paywall = fake.state.paywall
        if (paywall === null) return {}
        if (
          req.knownPaywallRevision !== '' &&
          req.knownPaywallRevision === paywall.revisionId
        ) {
          return {}
        }
        return { paywall }
      },
      createCheckoutSession(req, ctx) {
        track('createCheckoutSession', ctx.requestHeader)
        fake.state.lastCheckoutRequest = {
          productIdentifier: req.productIdentifier,
          successUrl: req.successUrl,
          cancelUrl: req.cancelUrl,
        }
        return { url: fake.state.checkoutUrl }
      },
      createBillingPortalSession(_req, ctx) {
        track('createBillingPortalSession', ctx.requestHeader)
        return { url: fake.state.portalUrl }
      },
    })
    service(PushService, {
      registerDevice(req, ctx) {
        track('registerDevice', ctx.requestHeader)
        fake.state.lastRegisterDevice = req
        return {
          device: {
            id: 'pd-1',
            target: req.target,
            deviceId: req.deviceId,
            permission: req.permission,
            metadata: req.metadata,
          },
        }
      },
      unregisterDevice(req, ctx) {
        track('unregisterDevice', ctx.requestHeader)
        fake.state.lastUnregisterDeviceId = req.deviceId
        return {}
      },
    })
  })
  return fake
}

export { CustomerInfoSchema, OfferingSchema, PaywallLayout, PaywallSchema }

export const testConfig: MothConfig = {
  endpoint: 'https://moth.test',
  publishableKey: 'pk_test',
  appName: 'TestApp',
  storage: 'memory',
}

/** A MothClient wired to a fresh fake server. */
export function fakeClient(
  options: FakeMothOptions & {
    config?: Partial<MothConfig>
    client?: Partial<MothClientOptions>
  } = {},
): { client: MothClient; fake: FakeMoth } {
  const fake = fakeMoth(options)
  const client = new MothClient(
    { ...testConfig, ...options.config },
    {
      transport: fake.transport,
      checkoutPollIntervalMs: 1,
      navigate: () => undefined,
      ...options.client,
    },
  )
  return { client, fake }
}
