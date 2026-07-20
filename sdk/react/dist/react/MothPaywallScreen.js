import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { useCallback, useEffect, useState } from 'react';
import { productHasTrial, } from '../core/offering.js';
import { loadPaywall } from '../core/paywallLoader.js';
import { useMothContext, useMothCopy, useMothTheme } from './context.js';
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
export function MothPaywallScreen(props) {
    const { client, customerInfo } = useMothContext();
    const theme = useMothTheme();
    const copy = useMothCopy();
    const [paywall, setPaywall] = useState(props.paywall ?? null);
    const [offering, setOffering] = useState(null);
    const [failed, setFailed] = useState(false);
    const [selectedId, setSelectedId] = useState(null);
    const [busy, setBusy] = useState(false);
    const [message, setMessage] = useState(null);
    const t = (key, vars) => copy.value(key, { app: client.config.appName ?? '', ...vars });
    const load = useCallback(async () => {
        setFailed(false);
        setOffering(null);
        try {
            const loadedPaywall = props.paywall ?? (await loadPaywall(client));
            const loadedOffering = await client.getOfferings({
                offering: loadedPaywall.offering,
            });
            setPaywall(loadedPaywall);
            setOffering(loadedOffering);
            setSelectedId(defaultSelection(loadedPaywall, loadedOffering));
        }
        catch {
            setFailed(true);
        }
    }, [client, props.paywall]);
    useEffect(() => {
        void load();
    }, [load]);
    const purchase = async () => {
        if (offering === null || selectedId === null)
            return;
        const product = offering.productById(selectedId);
        if (product === undefined)
            return;
        setBusy(true);
        setMessage(null);
        const result = await client.purchase(product);
        switch (result.status) {
            case 'redirect':
                props.onRedirect?.();
                break; // the page is navigating away; keep busy
            case 'error':
                setBusy(false);
                setMessage(result.message);
                break;
            default:
                setBusy(false);
        }
    };
    const manageBilling = async () => {
        setBusy(true);
        setMessage(null);
        try {
            await client.manageBilling();
            // Redirecting — leave busy on.
        }
        catch (err) {
            setBusy(false);
            setMessage(err instanceof Error ? err.message : String(err));
        }
    };
    let content;
    if (failed) {
        content = (_jsxs("div", { className: "moth-error-state", "data-moth": "paywall-error", children: [_jsx("h2", { className: "moth-title", children: t('paywall.error_title') }), _jsx("p", { className: "moth-subtitle", children: t('paywall.error_body') }), _jsx("button", { type: "button", className: "moth-btn", "data-moth": "retry", onClick: () => void load(), children: t('paywall.retry') })] }));
    }
    else if (paywall === null || offering === null) {
        content = _jsx("div", { className: "moth-spinner", role: "progressbar", "aria-label": "Loading" });
    }
    else {
        // The server delivers the headline/subtitle already localized; the
        // bundled paywall.title/subtitle are the floor when the config has not
        // arrived (offline first launch, no cache).
        const headline = paywall.headline !== '' ? paywall.headline : t('paywall.title');
        const subtitle = paywall.subtitle !== '' ? paywall.subtitle : t('paywall.subtitle');
        const hasSubscription = customerInfo.subscriptions.length > 0;
        content = (_jsxs(_Fragment, { children: [props.onClose !== undefined && (_jsx("div", { className: "moth-row moth-row--end", children: _jsx("button", { type: "button", className: "moth-btn-text", "aria-label": "Close", "data-moth": "close", onClick: props.onClose, children: "\u2715" }) })), _jsx(MothPaywallHeader, { headline: headline, subtitle: subtitle, logoLightUrl: theme.logoLightUrl, logoDarkUrl: theme.logoDarkUrl }), paywall.benefits.length > 0 && (_jsx("ul", { className: "moth-benefits", children: paywall.benefits.map((benefit) => (_jsx("li", { children: benefit }, benefit))) })), offering.isEmpty ? (_jsxs("div", { className: "moth-empty", "data-moth": "paywall-empty", children: [_jsx("h2", { className: "moth-title", children: t('paywall.empty_title') }), _jsx("p", { className: "moth-subtitle", children: t('paywall.empty_body') })] })) : (_jsxs(_Fragment, { children: [_jsx(TierSection, { paywall: paywall, offering: offering, selectedId: selectedId, busy: busy, copy: copy, onSelect: setSelectedId }), message !== null && (_jsx("div", { className: "moth-banner moth-banner--info", role: "status", "data-moth": "banner", children: message })), _jsx(MothPurchaseButton, { product: selectedId !== null ? offering.productById(selectedId) : undefined, busy: busy, label: t('paywall.cta'), onPress: () => void purchase() })] })), _jsxs("div", { className: "moth-footer", children: [hasSubscription && (_jsx("button", { type: "button", className: "moth-btn-text", disabled: busy, "data-moth": "manage-billing", onClick: () => void manageBilling(), children: t('paywall.manage_subscription') })), _jsxs("div", { className: "moth-row", children: [paywall.termsUrl !== undefined && (_jsx("a", { className: "moth-link-muted", href: paywall.termsUrl, target: "_blank", rel: "noreferrer", children: t('paywall.terms_link') })), paywall.termsUrl !== undefined && paywall.privacyUrl !== undefined && _jsx("span", { children: "\u00B7" }), paywall.privacyUrl !== undefined && (_jsx("a", { className: "moth-link-muted", href: paywall.privacyUrl, target: "_blank", rel: "noreferrer", children: t('paywall.privacy_link') }))] })] })] }));
    }
    return (_jsx("div", { className: "moth-screen", children: _jsx("div", { className: "moth-content moth-content--wide", children: content }) }));
}
function defaultSelection(paywall, offering) {
    const purchasable = offering.products.filter((p) => p.stripePriceId !== '');
    const pool = purchasable.length > 0 ? purchasable : [...offering.products];
    if (pool.length === 0)
        return null;
    const highlighted = paywall.highlightedProductIdentifier;
    if (highlighted !== '' && pool.some((p) => p.identifier === highlighted)) {
        return highlighted;
    }
    return (pool.find((p) => p.highlighted) ?? pool[0]).identifier;
}
function TierSection(props) {
    const { paywall, offering, selectedId, busy, copy } = props;
    // tiles = cards side by side; list = stacked rows; compact = a period
    // toggle plus the single selected card.
    if (paywall.layout === 'compact' && offering.products.length > 1) {
        const selected = (selectedId !== null ? offering.productById(selectedId) : undefined) ??
            offering.products[0];
        return (_jsxs("div", { className: "moth-tiers", children: [_jsx("div", { className: "moth-segments", role: "group", children: offering.products.map((product) => (_jsx("button", { type: "button", "aria-pressed": product.identifier === selected.identifier, disabled: busy, onClick: () => props.onSelect(product.identifier), children: compactSegmentLabel(product, copy) }, product.identifier))) }), _jsx(MothTierCard, { product: selected, selected: true, copy: copy })] }));
    }
    return (_jsx("div", { className: `moth-tiers${paywall.layout === 'tiles' ? ' moth-tiers--tiles' : ''}`, role: "radiogroup", children: offering.products.map((product) => (_jsx(MothTierCard, { product: product, selected: product.identifier === selectedId, copy: copy, onSelect: busy ? undefined : () => props.onSelect(product.identifier) }, product.identifier))) }));
}
/**
 * The paywall header: logo (when the theme sets one), headline and
 * subtitle. Exported for custom paywalls.
 */
export function MothPaywallHeader(props) {
    const light = props.logoLightUrl ?? props.logoDarkUrl;
    const dark = props.logoDarkUrl ?? props.logoLightUrl;
    return (_jsxs("div", { className: "moth-content moth-center", children: [light !== undefined && (_jsx("img", { className: "moth-logo moth-logo--light", src: light, alt: "" })), dark !== undefined && (_jsx("img", { className: "moth-logo moth-logo--dark", src: dark, alt: "" })), _jsx("h1", { className: "moth-title", "data-moth": "headline", children: props.headline }), props.subtitle !== undefined && props.subtitle !== '' && (_jsx("p", { className: "moth-subtitle", children: props.subtitle }))] }));
}
/**
 * One selectable subscription tier card: name, price/period, a trial badge
 * and the "most popular" highlight. Tiers without a Stripe price render
 * disabled with an unavailable-on-web note. Exported for custom paywalls.
 */
export function MothTierCard(props) {
    const { product, copy } = props;
    const unavailable = product.stripePriceId === '';
    const selected = props.selected === true && !unavailable;
    const classes = ['moth-tier'];
    if (product.highlighted)
        classes.push('moth-tier--highlighted');
    if (unavailable)
        classes.push('moth-tier--unavailable');
    return (_jsx("button", { type: "button", className: classes.join(' '), role: "radio", "aria-checked": selected, "aria-disabled": unavailable, "data-moth": `tier-${product.identifier}`, onClick: unavailable ? undefined : props.onSelect, children: _jsxs("div", { className: "moth-tier-body", children: [_jsxs("div", { className: "moth-tier-line", children: [_jsx("span", { className: "moth-tier-name", children: product.displayName !== '' ? product.displayName : product.identifier }), _jsx("span", { className: "moth-tier-price", children: priceLabel(product, copy) })] }), (product.highlighted || productHasTrial(product) || unavailable) && (_jsxs("div", { className: "moth-row", style: { justifyContent: 'flex-start' }, children: [product.highlighted && (_jsx("span", { className: "moth-badge moth-badge--primary", children: copy.value('paywall.most_popular') })), productHasTrial(product) && (_jsx("span", { className: "moth-badge moth-badge--soft", children: trialLabel(product.trialPeriod, copy) })), unavailable && (_jsx("span", { className: "moth-tier-note", "data-moth": "unavailable-web", children: copy.value('paywall.unavailable_web') }))] }))] }) }));
}
/**
 * The primary purchase button; label defaults to the localized
 * `paywall.cta`. Exported for custom paywalls.
 */
export function MothPurchaseButton(props) {
    const copy = useMothCopy();
    const busy = props.busy === true;
    const unavailable = props.product === undefined || props.product.stripePriceId === '';
    return (_jsx("button", { type: "button", className: "moth-btn", disabled: busy || unavailable, "data-moth": "purchase", onClick: props.onPress, children: busy ? '…' : (props.label ?? copy.value('paywall.cta')) }));
}
/**
 * Formats a tier's price with the billing period suffix (e.g. `$9.99 /
 * month`) from the catalog micros + currency.
 */
export function priceLabel(product, copy) {
    if (product.priceAmountMicros <= 0)
        return '—';
    const amount = product.priceAmountMicros / 1000000;
    const symbol = currencySymbol(product.currency);
    const formatted = Number.isInteger(amount)
        ? String(amount)
        : amount.toFixed(2);
    const price = symbol === ''
        ? `${formatted} ${product.currency}`.trim()
        : `${symbol}${formatted}`;
    const period = periodSuffix(product.billingPeriod, copy);
    return period === '' ? price : `${price} / ${period}`;
}
function compactSegmentLabel(product, copy) {
    const period = periodSuffix(product.billingPeriod, copy);
    if (period !== '')
        return period.charAt(0).toUpperCase() + period.slice(1);
    return product.displayName !== '' ? product.displayName : product.identifier;
}
function currencySymbol(currency) {
    switch (currency.toUpperCase()) {
        case 'USD':
        case 'AUD':
        case 'CAD':
        case 'NZD':
            return '$';
        case 'EUR':
            return '€';
        case 'GBP':
            return '£';
        case 'JPY':
        case 'CNY':
            return '¥';
        default:
            return '';
    }
}
/** ISO-8601 recurrence (`P1M`, `P1Y`, ...) → localized period suffix. */
function periodSuffix(period, copy) {
    switch (period.toUpperCase()) {
        case 'P1W':
            return copy.value('paywall.period_week');
        case 'P1M':
            return copy.value('paywall.period_month');
        case 'P3M':
            return copy.value('paywall.period_quarter');
        case 'P6M':
            return copy.value('paywall.period_6_month');
        case 'P1Y':
            return copy.value('paywall.period_year');
        default:
            return '';
    }
}
/** Human-readable, localized trial badge (e.g. `P1W` → `1-week free trial`). */
function trialLabel(period, copy) {
    switch (period.toUpperCase()) {
        case 'P3D':
            return copy.value('paywall.trial_3_day');
        case 'P1W':
        case 'P7D':
            return copy.value('paywall.trial_1_week');
        case 'P2W':
        case 'P14D':
            return copy.value('paywall.trial_2_week');
        case 'P1M':
            return copy.value('paywall.trial_1_month');
        default:
            return copy.value('paywall.trial_generic');
    }
}
