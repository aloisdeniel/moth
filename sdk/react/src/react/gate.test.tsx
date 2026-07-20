import { create } from '@bufbuild/protobuf'
import { timestampFromDate } from '@bufbuild/protobuf/wkt'
import { act, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import {
  CustomerInfoSchema,
  EntitlementSource,
  OfferingSchema,
} from '../gen/moth/billing/v1/billing_pb.js'
import { fakeClient } from '../test/fake.js'
import { MothProvider } from './context.js'
import { MothGate } from './MothGate.js'

function grantingOffering(entitlement = 'pro') {
  return create(OfferingSchema, {
    identifier: 'default',
    isDefault: true,
    products: [
      {
        identifier: 'monthly',
        displayName: 'Monthly',
        stripePriceId: 'price_month',
        billingPeriod: 'P1M',
        priceAmountMicros: 9_990_000n,
        currency: 'USD',
        entitlements: [entitlement],
      },
    ],
  })
}

function entitlementInfo(expiresAt?: Date) {
  return create(CustomerInfoSchema, {
    activeEntitlements: [
      {
        identifier: 'pro',
        source: EntitlementSource.STORE,
        productIdentifier: 'monthly',
        ...(expiresAt ? { expireTime: timestampFromDate(expiresAt) } : {}),
      },
    ],
  })
}

async function renderGate(
  mutate?: (fake: ReturnType<typeof fakeClient>['fake']) => void,
) {
  const { client, fake } = fakeClient()
  fake.state.offering = grantingOffering()
  mutate?.(fake)
  await client.signIn({ email: 'ada@example.com', password: 'pw' })
  render(
    <MothProvider client={client}>
      <MothGate entitlement="pro">
        <div>pro content</div>
      </MothGate>
    </MothProvider>,
  )
  return { client, fake }
}

describe('MothGate', () => {
  it('shows the paywall while the entitlement is missing, flips instantly when it arrives', async () => {
    const { client, fake } = await renderGate()
    // The offering grants `pro`, so the gate blocks with the paywall.
    expect(await screen.findByText('Unlock TestApp Premium')).toBeInTheDocument()
    expect(screen.queryByText('pro content')).not.toBeInTheDocument()
    // The webhook lands: the entitlement arrives via a billing refresh.
    fake.state.customerInfo = entitlementInfo()
    await act(async () => {
      await client.getCustomerInfo()
    })
    expect(screen.getByText('pro content')).toBeInTheDocument()
    expect(screen.queryByText('Unlock TestApp Premium')).not.toBeInTheDocument()
  })

  it('falls through to children when no product grants the entitlement', async () => {
    await renderGate((fake) => {
      fake.state.offering = grantingOffering('other_entitlement')
    })
    // Nothing to sell for `pro` — never block.
    expect(await screen.findByText('pro content')).toBeInTheDocument()
  })

  it('shows the paywall when the catalog cannot be loaded (it has its own retry)', async () => {
    const { ConnectError, Code } = await import('@connectrpc/connect')
    await renderGate((fake) => {
      fake.failNext['getOfferings'] = [
        new ConnectError('down', Code.Unavailable),
        new ConnectError('down', Code.Unavailable),
      ]
    })
    expect(
      await screen.findByText('Cannot reach the store'),
    ).toBeInTheDocument()
  })

  it('renders a custom fallback bare', async () => {
    const { client, fake } = fakeClient()
    fake.state.offering = grantingOffering()
    await client.signIn({ email: 'ada@example.com', password: 'pw' })
    const { container } = render(
      <MothProvider client={client}>
        <MothGate entitlement="pro" fallback={<div>my own wall</div>}>
          <div>pro content</div>
        </MothGate>
      </MothProvider>,
    )
    expect(await screen.findByText('my own wall')).toBeInTheDocument()
    expect(container.querySelector('.moth-root')).toBeNull()
  })

  it('re-arms the capped expiry timer: an entitlement 30h out flips at 30h', async () => {
    vi.useFakeTimers()
    try {
      const { client, fake } = fakeClient()
      fake.state.offering = grantingOffering()
      await client.signIn({ email: 'ada@example.com', password: 'pw' })
      render(
        <MothProvider client={client}>
          <MothGate entitlement="pro" fallback={<div>the wall</div>}>
            <div>pro content</div>
          </MothGate>
        </MothProvider>,
      )
      await act(async () => {
        await vi.advanceTimersByTimeAsync(0)
      })
      fake.state.customerInfo = entitlementInfo(
        new Date(Date.now() + 30 * 3600_000),
      )
      await act(async () => {
        await client.getCustomerInfo()
      })
      expect(screen.getByText('pro content')).toBeInTheDocument()

      // 24h: the capped timer fires and must re-arm — not flip the gate.
      await act(async () => {
        await vi.advanceTimersByTimeAsync(24 * 3600_000)
      })
      expect(screen.getByText('pro content')).toBeInTheDocument()

      // 6h more: the expiry passed — the gate closes without any RPC.
      await act(async () => {
        await vi.advanceTimersByTimeAsync(6 * 3600_000 + 1_000)
      })
      expect(screen.getByText('the wall')).toBeInTheDocument()
      expect(screen.queryByText('pro content')).not.toBeInTheDocument()
    } finally {
      vi.useRealTimers()
    }
  })

  it('retries a failed resolution and falls through once the offering answers "nothing to sell"', async () => {
    vi.useFakeTimers()
    try {
      const { ConnectError, Code } = await import('@connectrpc/connect')
      const { client, fake } = fakeClient()
      // Nothing sells 'pro' — but the first catalog fetch fails.
      fake.state.offering = grantingOffering('other_entitlement')
      fake.failNext['getOfferings'] = [
        new ConnectError('down', Code.Unavailable),
      ]
      await client.signIn({ email: 'ada@example.com', password: 'pw' })
      render(
        <MothProvider client={client}>
          <MothGate entitlement="pro" fallback={<div>the wall</div>}>
            <div>pro content</div>
          </MothGate>
        </MothProvider>,
      )
      await act(async () => {
        await vi.advanceTimersByTimeAsync(0)
      })
      // Transient failure: the paywall fallback shows, but nothing latches.
      expect(screen.getByText('the wall')).toBeInTheDocument()
      // The backoff retry succeeds; no product grants 'pro' → fall through.
      await act(async () => {
        await vi.advanceTimersByTimeAsync(3_000)
      })
      expect(screen.getByText('pro content')).toBeInTheDocument()
    } finally {
      vi.useRealTimers()
    }
  })

  it('caches the offering verdict per client: a remount costs zero catalog RPCs', async () => {
    const { client, fake } = fakeClient()
    fake.state.offering = grantingOffering()
    await client.signIn({ email: 'ada@example.com', password: 'pw' })
    const view = render(
      <MothProvider client={client}>
        <MothGate entitlement="pro" fallback={<div>the wall</div>}>
          <div>pro content</div>
        </MothGate>
      </MothProvider>,
    )
    expect(await screen.findByText('the wall')).toBeInTheDocument()
    const offeringCalls = fake.calls['getOfferings']
    const paywallCalls = fake.calls['getPaywall']
    expect(offeringCalls).toBe(1)
    view.unmount()
    render(
      <MothProvider client={client}>
        <MothGate entitlement="pro" fallback={<div>the wall</div>}>
          <div>pro content</div>
        </MothGate>
      </MothProvider>,
    )
    expect(await screen.findByText('the wall')).toBeInTheDocument()
    expect(fake.calls['getOfferings']).toBe(offeringCalls)
    expect(fake.calls['getPaywall']).toBe(paywallCalls)
  })

  it('an entitled user triggers zero catalog RPCs', async () => {
    const { client, fake } = fakeClient()
    fake.state.offering = grantingOffering()
    fake.state.customerInfo = entitlementInfo()
    await client.signIn({ email: 'ada@example.com', password: 'pw' })
    await client.getCustomerInfo() // entitlement known before mount
    render(
      <MothProvider client={client}>
        <MothGate entitlement="pro">
          <div>pro content</div>
        </MothGate>
      </MothProvider>,
    )
    expect(await screen.findByText('pro content')).toBeInTheDocument()
    expect(fake.calls['getOfferings']).toBeUndefined()
    expect(fake.calls['getPaywall']).toBeUndefined()
  })

  it('none -> active -> expired transitions with a fake clock', async () => {
    vi.useFakeTimers()
    try {
      const { client, fake } = fakeClient()
      fake.state.offering = grantingOffering()
      await client.signIn({ email: 'ada@example.com', password: 'pw' })
      render(
        <MothProvider client={client}>
          <MothGate entitlement="pro" fallback={<div>the wall</div>}>
            <div>pro content</div>
          </MothGate>
        </MothProvider>,
      )
      // none: the gate resolves the offering and blocks.
      await act(async () => {
        await vi.advanceTimersByTimeAsync(0)
      })
      expect(screen.getByText('the wall')).toBeInTheDocument()

      // active: an entitlement expiring in 60s arrives.
      fake.state.customerInfo = entitlementInfo(
        new Date(Date.now() + 60_000),
      )
      await act(async () => {
        await client.getCustomerInfo()
      })
      expect(screen.getByText('pro content')).toBeInTheDocument()

      // expired: advancing the clock past the expiry flips the gate back
      // without any server round-trip.
      await act(async () => {
        await vi.advanceTimersByTimeAsync(61_000)
      })
      expect(screen.getByText('the wall')).toBeInTheDocument()
      expect(screen.queryByText('pro content')).not.toBeInTheDocument()
    } finally {
      vi.useRealTimers()
    }
  })
})
