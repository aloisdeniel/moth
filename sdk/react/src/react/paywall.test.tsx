import { create } from '@bufbuild/protobuf'
import { act, fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import {
  OfferingSchema,
  PaywallLayout,
  PaywallSchema,
} from '../gen/moth/billing/v1/billing_pb.js'
import { fakeClient, proCustomerInfo } from '../test/fake.js'
import { MothCustomerInfo } from '../core/customerInfo.js'
import { MothProvider } from './context.js'
import { MothPaywallScreen } from './MothPaywallScreen.js'

function offeringWithTiers() {
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
        trialPeriod: 'P1W',
        entitlements: ['pro'],
        sortOrder: 1,
      },
      {
        identifier: 'yearly',
        displayName: 'Yearly',
        stripePriceId: 'price_year',
        billingPeriod: 'P1Y',
        priceAmountMicros: 79_000_000n,
        currency: 'USD',
        entitlements: ['pro'],
        sortOrder: 2,
        highlighted: true,
      },
      {
        identifier: 'mobile_only',
        displayName: 'Mobile Only',
        billingPeriod: 'P1M',
        priceAmountMicros: 4_990_000n,
        currency: 'EUR',
        entitlements: ['pro'],
        sortOrder: 3,
      },
    ],
  })
}

async function renderPaywall(
  mutate?: (fake: ReturnType<typeof fakeClient>['fake']) => void,
  navigated: string[] = [],
) {
  const { client, fake } = fakeClient({
    client: { navigate: (url) => navigated.push(url) },
  })
  fake.state.offering = offeringWithTiers()
  fake.state.paywall = create(PaywallSchema, {
    revisionId: 'pw-1',
    headline: 'Go beyond',
    subtitle: 'Everything, unlimited.',
    benefits: ['Unlimited moths', 'Priority lamp'],
    highlightedProductIdentifier: 'yearly',
    layout: PaywallLayout.LIST,
    termsUrl: 'https://example.com/terms',
  })
  mutate?.(fake)
  await client.signIn({ email: 'ada@example.com', password: 'pw' })
  render(
    <MothProvider client={client}>
      <MothPaywallScreen />
    </MothProvider>,
  )
  return { client, fake, navigated }
}

describe('MothPaywallScreen', () => {
  it('renders the config: headline, benefits, tiers, prices and badges', async () => {
    await renderPaywall()
    expect(await screen.findByText('Go beyond')).toBeInTheDocument()
    expect(screen.getByText('Everything, unlimited.')).toBeInTheDocument()
    expect(screen.getByText('Unlimited moths')).toBeInTheDocument()
    expect(screen.getByText('Priority lamp')).toBeInTheDocument()
    // Tiers with formatted prices + period suffixes from copy keys.
    expect(screen.getByText('Monthly')).toBeInTheDocument()
    expect(screen.getByText('$9.99 / month')).toBeInTheDocument()
    expect(screen.getByText('Yearly')).toBeInTheDocument()
    expect(screen.getByText('$79 / year')).toBeInTheDocument()
    // Badges.
    expect(screen.getByText('Most popular')).toBeInTheDocument()
    expect(screen.getByText('1-week free trial')).toBeInTheDocument()
    // Legal link from the paywall config.
    expect(screen.getByText('Terms of Service')).toHaveAttribute(
      'href',
      'https://example.com/terms',
    )
  })

  it('marks tiers without a Stripe price unavailable on the web', async () => {
    await renderPaywall()
    await screen.findByText('Go beyond')
    const unavailable = screen.getByText('Mobile Only').closest('button')!
    expect(unavailable).toHaveAttribute('aria-disabled', 'true')
    expect(
      screen.getByText('Not available for purchase on the web'),
    ).toBeInTheDocument()
    // Clicking it never selects it.
    fireEvent.click(unavailable)
    expect(unavailable).toHaveAttribute('aria-checked', 'false')
  })

  it('defaults the selection to the highlighted tier and purchases it', async () => {
    const { fake, navigated } = await renderPaywall()
    await screen.findByText('Go beyond')
    const yearly = screen.getByText('Yearly').closest('button')!
    expect(yearly).toHaveAttribute('aria-checked', 'true')
    await act(async () => {
      fireEvent.click(screen.getByText('Continue'))
    })
    expect(fake.calls['createCheckoutSession']).toBe(1)
    expect(fake.state.lastCheckoutRequest?.productIdentifier).toBe('yearly')
    expect(navigated).toEqual(['https://checkout.stripe.test/session'])
  })

  it('selecting another tier purchases that one', async () => {
    const { fake } = await renderPaywall()
    await screen.findByText('Go beyond')
    fireEvent.click(screen.getByText('Monthly').closest('button')!)
    await act(async () => {
      fireEvent.click(screen.getByText('Continue'))
    })
    expect(fake.state.lastCheckoutRequest?.productIdentifier).toBe('monthly')
  })

  it('surfaces checkout errors in the banner without throwing', async () => {
    const { mothConnectError } = await import('../test/fake.js')
    const { Code } = await import('@connectrpc/connect')
    const { fake, navigated } = await renderPaywall()
    await screen.findByText('Go beyond')
    fake.failNext['createCheckoutSession'] = [
      mothConnectError(
        Code.FailedPrecondition,
        'BILLING_NOT_CONFIGURED',
        'stripe is not configured',
      ),
    ]
    await act(async () => {
      fireEvent.click(screen.getByText('Continue'))
    })
    expect(
      await screen.findByText('stripe is not configured'),
    ).toBeInTheDocument()
    expect(navigated).toEqual([])
  })

  it('renders the empty state when there is nothing to sell', async () => {
    await renderPaywall((fake) => {
      fake.state.offering = create(OfferingSchema, {
        identifier: 'default',
        isDefault: true,
      })
    })
    expect(
      await screen.findByText('Nothing to purchase yet'),
    ).toBeInTheDocument()
    expect(screen.queryByText('Continue')).not.toBeInTheDocument()
  })

  it('renders the error state with retry when the catalog fails to load', async () => {
    const { ConnectError, Code } = await import('@connectrpc/connect')
    const { fake } = await renderPaywall((f) => {
      f.failNext['getOfferings'] = [new ConnectError('down', Code.Unavailable)]
    })
    expect(
      await screen.findByText('Cannot reach the store'),
    ).toBeInTheDocument()
    // Retry: the next attempt succeeds.
    await act(async () => {
      fireEvent.click(screen.getByText('Try again'))
    })
    expect(await screen.findByText('Go beyond')).toBeInTheDocument()
    expect(fake.calls['getOfferings']).toBe(2)
  })

  it('shows the manage-billing link when subscriptions exist and opens the portal', async () => {
    const { client, navigated } = await renderPaywall((fake) => {
      fake.state.customerInfo = proCustomerInfo()
    })
    await screen.findByText('Go beyond')
    // The provider's subscription state carries an active subscription.
    await act(async () => {
      client.primeCustomerInfo(
        new MothCustomerInfo(
          [],
          [
            {
              productIdentifier: 'monthly',
              store: 'stripe',
              status: 'active',
              autoRenew: true,
              isSandbox: true,
            },
          ],
        ),
      )
    })
    const manage = await screen.findByText('Manage subscription')
    await act(async () => {
      fireEvent.click(manage)
    })
    expect(navigated).toContain('https://billing.stripe.test/portal')
  })

  it('renders the compact layout as a period toggle plus one card', async () => {
    await renderPaywall((fake) => {
      fake.state.paywall = create(PaywallSchema, {
        revisionId: 'pw-2',
        headline: 'Compact',
        layout: PaywallLayout.COMPACT,
      })
    })
    await screen.findByText('Compact')
    // Segments named by period (monthly + mobile_only are both P1M).
    expect(screen.getAllByText('Month')).toHaveLength(2)
    expect(screen.getByText('Year')).toBeInTheDocument()
    fireEvent.click(screen.getAllByText('Month')[0]!)
    expect(screen.getByText('$9.99 / month')).toBeInTheDocument()
  })
})
