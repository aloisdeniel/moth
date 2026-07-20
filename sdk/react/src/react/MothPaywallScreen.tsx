import { useCallback, useEffect, useState, type ReactNode } from 'react'
import type { MothCopy } from '../core/copy.js'
import {
  productHasTrial,
  type MothOffering,
  type MothOfferingProduct,
  type MothPaywall,
} from '../core/offering.js'
import { loadPaywall } from '../core/paywallLoader.js'
import { useMothContext, useMothCopy, useMothTheme } from './context.js'

export interface MothPaywallScreenProps {
  /** Paywall copy/layout override; when unset it is fetched (revision-cached). */
  paywall?: MothPaywall
  /** Called after a checkout redirect begins (e.g. to show a note). */
  onRedirect?: () => void
  /** Called when the user dismisses the paywall; shows a close button when set. */
  onClose?: () => void
}

/**
 * Batteries-included paywall screen — the purchasing counterpart to
 * {@link MothLoginScreen}, driven by the same admin-configured paywall
 * config as the mobile SDKs: header (logo + headline + subtitle), benefit
 * bullets, one selectable card per tier (price, trial badge, "most popular"
 * highlight), the purchase button (Stripe-hosted Checkout redirect — no
 * card fields, no Stripe.js), a manage-billing link when subscriptions
 * exist, and terms/privacy links. Tiers without a `stripe_price_id` render
 * as unavailable-on-web rather than disappearing silently.
 *
 * The building blocks ({@link MothPaywallHeader}, {@link MothTierCard},
 * {@link MothPurchaseButton}) are exported for custom paywalls.
 */
export function MothPaywallScreen(props: MothPaywallScreenProps) {
  const { client, customerInfo } = useMothContext()
  const theme = useMothTheme()
  const copy = useMothCopy()

  const [paywall, setPaywall] = useState<MothPaywall | null>(
    props.paywall ?? null,
  )
  const [offering, setOffering] = useState<MothOffering | null>(null)
  const [failed, setFailed] = useState(false)
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [message, setMessage] = useState<string | null>(null)

  const t = (key: string, vars?: Record<string, string>) =>
    copy.value(key, { app: client.config.appName ?? '', ...vars })

  const load = useCallback(async () => {
    setFailed(false)
    setOffering(null)
    try {
      const loadedPaywall = props.paywall ?? (await loadPaywall(client))
      const loadedOffering = await client.getOfferings({
        offering: loadedPaywall.offering,
      })
      setPaywall(loadedPaywall)
      setOffering(loadedOffering)
      setSelectedId(defaultSelection(loadedPaywall, loadedOffering))
    } catch {
      setFailed(true)
    }
  }, [client, props.paywall])

  useEffect(() => {
    void load()
  }, [load])

  const purchase = async () => {
    if (offering === null || selectedId === null) return
    const product = offering.productById(selectedId)
    if (product === undefined) return
    setBusy(true)
    setMessage(null)
    const result = await client.purchase(product)
    switch (result.status) {
      case 'redirect':
        props.onRedirect?.()
        break // the page is navigating away; keep busy
      case 'error':
        setBusy(false)
        setMessage(result.message)
        break
      default:
        setBusy(false)
    }
  }

  const manageBilling = async () => {
    setBusy(true)
    setMessage(null)
    try {
      await client.manageBilling()
      // Redirecting — leave busy on.
    } catch (err) {
      setBusy(false)
      setMessage(err instanceof Error ? err.message : String(err))
    }
  }

  let content: ReactNode
  if (failed) {
    content = (
      <div className="moth-error-state" data-moth="paywall-error">
        <h2 className="moth-title">{t('paywall.error_title')}</h2>
        <p className="moth-subtitle">{t('paywall.error_body')}</p>
        <button type="button" className="moth-btn" data-moth="retry" onClick={() => void load()}>
          {t('paywall.retry')}
        </button>
      </div>
    )
  } else if (paywall === null || offering === null) {
    content = <div className="moth-spinner" role="progressbar" aria-label="Loading" />
  } else {
    // The server delivers the headline/subtitle already localized; the
    // bundled paywall.title/subtitle are the floor when the config has not
    // arrived (offline first launch, no cache).
    const headline = paywall.headline !== '' ? paywall.headline : t('paywall.title')
    const subtitle = paywall.subtitle !== '' ? paywall.subtitle : t('paywall.subtitle')
    const hasSubscription = customerInfo.subscriptions.length > 0
    content = (
      <>
        {props.onClose !== undefined && (
          <div className="moth-row moth-row--end">
            <button
              type="button"
              className="moth-btn-text"
              aria-label="Close"
              data-moth="close"
              onClick={props.onClose}
            >
              ✕
            </button>
          </div>
        )}
        <MothPaywallHeader
          headline={headline}
          subtitle={subtitle}
          logoLightUrl={theme.logoLightUrl}
          logoDarkUrl={theme.logoDarkUrl}
        />
        {paywall.benefits.length > 0 && (
          <ul className="moth-benefits">
            {paywall.benefits.map((benefit) => (
              <li key={benefit}>{benefit}</li>
            ))}
          </ul>
        )}
        {offering.isEmpty ? (
          <div className="moth-empty" data-moth="paywall-empty">
            <h2 className="moth-title">{t('paywall.empty_title')}</h2>
            <p className="moth-subtitle">{t('paywall.empty_body')}</p>
          </div>
        ) : (
          <>
            <TierSection
              paywall={paywall}
              offering={offering}
              selectedId={selectedId}
              busy={busy}
              copy={copy}
              onSelect={setSelectedId}
            />
            {message !== null && (
              <div className="moth-banner moth-banner--info" role="status" data-moth="banner">
                {message}
              </div>
            )}
            <MothPurchaseButton
              product={
                selectedId !== null ? offering.productById(selectedId) : undefined
              }
              busy={busy}
              label={t('paywall.cta')}
              onPress={() => void purchase()}
            />
          </>
        )}
        <div className="moth-footer">
          {hasSubscription && (
            <button
              type="button"
              className="moth-btn-text"
              disabled={busy}
              data-moth="manage-billing"
              onClick={() => void manageBilling()}
            >
              {t('paywall.manage_subscription')}
            </button>
          )}
          <div className="moth-row">
            {paywall.termsUrl !== undefined && (
              <a className="moth-link-muted" href={paywall.termsUrl} target="_blank" rel="noreferrer">
                {t('paywall.terms_link')}
              </a>
            )}
            {paywall.termsUrl !== undefined && paywall.privacyUrl !== undefined && <span>·</span>}
            {paywall.privacyUrl !== undefined && (
              <a className="moth-link-muted" href={paywall.privacyUrl} target="_blank" rel="noreferrer">
                {t('paywall.privacy_link')}
              </a>
            )}
          </div>
        </div>
      </>
    )
  }

  return (
    <div className="moth-screen">
      <div className="moth-content moth-content--wide">{content}</div>
    </div>
  )
}

function defaultSelection(
  paywall: MothPaywall,
  offering: MothOffering,
): string | null {
  const purchasable = offering.products.filter((p) => p.stripePriceId !== '')
  const pool = purchasable.length > 0 ? purchasable : [...offering.products]
  if (pool.length === 0) return null
  const highlighted = paywall.highlightedProductIdentifier
  if (highlighted !== '' && pool.some((p) => p.identifier === highlighted)) {
    return highlighted
  }
  return (pool.find((p) => p.highlighted) ?? pool[0]!).identifier
}

function TierSection(props: {
  paywall: MothPaywall
  offering: MothOffering
  selectedId: string | null
  busy: boolean
  copy: MothCopy
  onSelect: (id: string) => void
}) {
  const { paywall, offering, selectedId, busy, copy } = props
  // tiles = cards side by side; list = stacked rows; compact = a period
  // toggle plus the single selected card.
  if (paywall.layout === 'compact' && offering.products.length > 1) {
    const selected =
      (selectedId !== null ? offering.productById(selectedId) : undefined) ??
      offering.products[0]!
    return (
      <div className="moth-tiers">
        <div className="moth-segments" role="group">
          {offering.products.map((product) => (
            <button
              key={product.identifier}
              type="button"
              aria-pressed={product.identifier === selected.identifier}
              disabled={busy}
              onClick={() => props.onSelect(product.identifier)}
            >
              {compactSegmentLabel(product, copy)}
            </button>
          ))}
        </div>
        <MothTierCard product={selected} selected copy={copy} />
      </div>
    )
  }
  return (
    <div
      className={`moth-tiers${paywall.layout === 'tiles' ? ' moth-tiers--tiles' : ''}`}
      role="radiogroup"
    >
      {offering.products.map((product) => (
        <MothTierCard
          key={product.identifier}
          product={product}
          selected={product.identifier === selectedId}
          copy={copy}
          onSelect={busy ? undefined : () => props.onSelect(product.identifier)}
        />
      ))}
    </div>
  )
}

/**
 * The paywall header: logo (when the theme sets one), headline and
 * subtitle. Exported for custom paywalls.
 */
export function MothPaywallHeader(props: {
  headline: string
  subtitle?: string
  logoLightUrl?: string
  logoDarkUrl?: string
}) {
  const light = props.logoLightUrl ?? props.logoDarkUrl
  const dark = props.logoDarkUrl ?? props.logoLightUrl
  return (
    <div className="moth-content moth-center">
      {light !== undefined && (
        <img className="moth-logo moth-logo--light" src={light} alt="" />
      )}
      {dark !== undefined && (
        <img className="moth-logo moth-logo--dark" src={dark} alt="" />
      )}
      <h1 className="moth-title" data-moth="headline">
        {props.headline}
      </h1>
      {props.subtitle !== undefined && props.subtitle !== '' && (
        <p className="moth-subtitle">{props.subtitle}</p>
      )}
    </div>
  )
}

/**
 * One selectable subscription tier card: name, price/period, a trial badge
 * and the "most popular" highlight. Tiers without a Stripe price render
 * disabled with an unavailable-on-web note. Exported for custom paywalls.
 */
export function MothTierCard(props: {
  product: MothOfferingProduct
  selected?: boolean
  copy: MothCopy
  onSelect?: () => void
}) {
  const { product, copy } = props
  const unavailable = product.stripePriceId === ''
  const selected = props.selected === true && !unavailable
  const classes = ['moth-tier']
  if (product.highlighted) classes.push('moth-tier--highlighted')
  if (unavailable) classes.push('moth-tier--unavailable')
  return (
    <button
      type="button"
      className={classes.join(' ')}
      role="radio"
      aria-checked={selected}
      aria-disabled={unavailable}
      data-moth={`tier-${product.identifier}`}
      onClick={unavailable ? undefined : props.onSelect}
    >
      <div className="moth-tier-body">
        <div className="moth-tier-line">
          <span className="moth-tier-name">
            {product.displayName !== '' ? product.displayName : product.identifier}
          </span>
          <span className="moth-tier-price">{priceLabel(product, copy)}</span>
        </div>
        {(product.highlighted || productHasTrial(product) || unavailable) && (
          <div className="moth-row" style={{ justifyContent: 'flex-start' }}>
            {product.highlighted && (
              <span className="moth-badge moth-badge--primary">
                {copy.value('paywall.most_popular')}
              </span>
            )}
            {productHasTrial(product) && (
              <span className="moth-badge moth-badge--soft">
                {trialLabel(product.trialPeriod, copy)}
              </span>
            )}
            {unavailable && (
              <span className="moth-tier-note" data-moth="unavailable-web">
                {copy.value('paywall.unavailable_web')}
              </span>
            )}
          </div>
        )}
      </div>
    </button>
  )
}

/**
 * The primary purchase button; label defaults to the localized
 * `paywall.cta`. Exported for custom paywalls.
 */
export function MothPurchaseButton(props: {
  product?: MothOfferingProduct | undefined
  busy?: boolean
  label?: string
  onPress?: () => void
}) {
  const copy = useMothCopy()
  const busy = props.busy === true
  const unavailable =
    props.product === undefined || props.product.stripePriceId === ''
  return (
    <button
      type="button"
      className="moth-btn"
      disabled={busy || unavailable}
      data-moth="purchase"
      onClick={props.onPress}
    >
      {busy ? '…' : (props.label ?? copy.value('paywall.cta'))}
    </button>
  )
}

/**
 * Formats a tier's price with the billing period suffix (e.g. `$9.99 /
 * month`) from the catalog micros + currency.
 */
export function priceLabel(
  product: MothOfferingProduct,
  copy: MothCopy,
): string {
  if (product.priceAmountMicros <= 0) return '—'
  const amount = product.priceAmountMicros / 1_000_000
  const symbol = currencySymbol(product.currency)
  const formatted = Number.isInteger(amount)
    ? String(amount)
    : amount.toFixed(2)
  const price =
    symbol === ''
      ? `${formatted} ${product.currency}`.trim()
      : `${symbol}${formatted}`
  const period = periodSuffix(product.billingPeriod, copy)
  return period === '' ? price : `${price} / ${period}`
}

function compactSegmentLabel(
  product: MothOfferingProduct,
  copy: MothCopy,
): string {
  const period = periodSuffix(product.billingPeriod, copy)
  if (period !== '') return period.charAt(0).toUpperCase() + period.slice(1)
  return product.displayName !== '' ? product.displayName : product.identifier
}

function currencySymbol(currency: string): string {
  switch (currency.toUpperCase()) {
    case 'USD':
    case 'AUD':
    case 'CAD':
    case 'NZD':
      return '$'
    case 'EUR':
      return '€'
    case 'GBP':
      return '£'
    case 'JPY':
    case 'CNY':
      return '¥'
    default:
      return ''
  }
}

/** ISO-8601 recurrence (`P1M`, `P1Y`, ...) → localized period suffix. */
function periodSuffix(period: string, copy: MothCopy): string {
  switch (period.toUpperCase()) {
    case 'P1W':
      return copy.value('paywall.period_week')
    case 'P1M':
      return copy.value('paywall.period_month')
    case 'P3M':
      return copy.value('paywall.period_quarter')
    case 'P6M':
      return copy.value('paywall.period_6_month')
    case 'P1Y':
      return copy.value('paywall.period_year')
    default:
      return ''
  }
}

/** Human-readable, localized trial badge (e.g. `P1W` → `1-week free trial`). */
function trialLabel(period: string, copy: MothCopy): string {
  switch (period.toUpperCase()) {
    case 'P3D':
      return copy.value('paywall.trial_3_day')
    case 'P1W':
    case 'P7D':
      return copy.value('paywall.trial_1_week')
    case 'P2W':
    case 'P14D':
      return copy.value('paywall.trial_2_week')
    case 'P1M':
      return copy.value('paywall.trial_1_month')
    default:
      return copy.value('paywall.trial_generic')
  }
}
