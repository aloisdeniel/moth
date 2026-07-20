// A minimal-but-real Stripe API test double for the React SDK e2e suite —
// the local stand-in milestone 17 drives in its Go tests, here as a
// standalone process the spawned moth binary reaches via the env-only
// MOTH_STRIPE_API_URL testing hook. It implements exactly the surface the
// billing path exercises:
//
//   POST /v1/customers                  -> mints cus_N
//   POST /v1/checkout/sessions          -> mints cs_N with a hosted-checkout
//                                          URL pointing back at this double
//   GET  /checkout/{cs_id}              -> "completes" the checkout: creates
//                                          the subscription, POSTs a properly
//                                          HMAC-signed
//                                          checkout.session.completed event
//                                          to the moth webhook, then 302s to
//                                          the session's success_url
//   GET  /v1/subscriptions/{sub_id}     -> authoritative re-read (what moth
//                                          trusts; the webhook is a nudge)
//   POST /v1/billing_portal/sessions    -> mints a portal URL at /portal
//   GET  /portal                        -> stub Billing Portal landing page
//
// Webhook signing follows Stripe's scheme verbatim:
//   Stripe-Signature: t=<unix>,v1=hex(HMAC-SHA256(secret, `${t}.${body}`))
//
// Config via env: PORT, STRIPE_SECRET_KEY (required Bearer token),
// MOTH_WEBHOOK_URL (the project's /billing/stripe/webhook/{slug} endpoint),
// STRIPE_WEBHOOK_SECRET (whsec_..., shared with the project's credentials).

import { createHmac } from 'node:crypto'
import { createServer } from 'node:http'

const port = Number(process.env.PORT ?? 8993)
const secretKey = process.env.STRIPE_SECRET_KEY ?? ''
const webhookUrl = process.env.MOTH_WEBHOOK_URL ?? ''
const webhookSecret = process.env.STRIPE_WEBHOOK_SECRET ?? ''

let seq = 0
const nextId = (prefix) => `${prefix}_e2e_${++seq}`

/** @type {Map<string, any>} checkout sessions by id */
const sessions = new Map()
/** @type {Map<string, any>} subscriptions by id (Stripe wire shape) */
const subscriptions = new Map()

function readBody(req) {
  return new Promise((resolve, reject) => {
    const chunks = []
    req.on('data', (c) => chunks.push(c))
    req.on('end', () => resolve(Buffer.concat(chunks).toString()))
    req.on('error', reject)
  })
}

function json(res, status, body) {
  const payload = JSON.stringify(body)
  res.writeHead(status, { 'content-type': 'application/json' })
  res.end(payload)
}

function stripeError(res, status, message, code = '') {
  json(res, status, { error: { message, type: 'invalid_request_error', code } })
}

function signWebhook(body) {
  const t = Math.floor(Date.now() / 1000)
  const mac = createHmac('sha256', webhookSecret)
  mac.update(`${t}.${body}`)
  return `t=${t},v1=${mac.digest('hex')}`
}

/** POSTs a signed event to the moth webhook, retrying transient failures. */
async function deliverWebhook(event) {
  const body = JSON.stringify(event)
  const signature = signWebhook(body)
  for (let attempt = 0; attempt < 5; attempt++) {
    try {
      const resp = await fetch(webhookUrl, {
        method: 'POST',
        headers: {
          'content-type': 'application/json',
          'stripe-signature': signature,
        },
        body,
      })
      if (resp.ok) return
      // 4xx is permanent (would never verify differently); 5xx retries.
      if (resp.status < 500) {
        console.error(`stripe-double: webhook rejected: ${resp.status} ${await resp.text()}`)
        return
      }
    } catch (err) {
      console.error(`stripe-double: webhook delivery failed: ${err}`)
    }
    await new Promise((r) => setTimeout(r, 200))
  }
}

const server = createServer((req, res) => {
  handle(req, res).catch((err) => {
    console.error(`stripe-double: ${err}`)
    stripeError(res, 500, String(err))
  })
})

async function handle(req, res) {
  const url = new URL(req.url ?? '/', `http://127.0.0.1:${port}`)
  const path = url.pathname

  if (req.method === 'GET' && path === '/healthz') {
    res.writeHead(200)
    res.end('ok')
    return
  }

  // Browser-facing pages (no API auth).
  if (req.method === 'GET' && path.startsWith('/checkout/')) {
    await completeCheckout(req, res, path.slice('/checkout/'.length))
    return
  }
  if (req.method === 'GET' && path === '/portal') {
    const returnUrl = url.searchParams.get('return_url') ?? ''
    res.writeHead(200, { 'content-type': 'text/html' })
    res.end(`<!doctype html><title>Billing Portal</title>
<h1>Stripe billing portal (test double)</h1>
<p>Manage your subscription here.</p>
${returnUrl !== '' ? `<a href="${returnUrl}">Return to app</a>` : ''}`)
    return
  }

  // API endpoints: Bearer-authenticated like the real thing.
  const auth = req.headers.authorization ?? ''
  if (auth !== `Bearer ${secretKey}`) {
    stripeError(res, 401, 'Invalid API key provided')
    return
  }

  if (req.method === 'POST' && path === '/v1/customers') {
    const form = new URLSearchParams(await readBody(req))
    const customer = { id: nextId('cus'), object: 'customer', email: form.get('email') ?? '' }
    json(res, 200, customer)
    return
  }

  if (req.method === 'POST' && path === '/v1/checkout/sessions') {
    const form = new URLSearchParams(await readBody(req))
    const id = nextId('cs')
    const metadata = collectMetadata(form, 'metadata')
    const session = {
      id,
      mode: form.get('mode'),
      price: form.get('line_items[0][price]') ?? '',
      customer: form.get('customer') ?? '',
      success_url: form.get('success_url') ?? '',
      cancel_url: form.get('cancel_url') ?? '',
      client_reference_id: form.get('client_reference_id') ?? '',
      metadata,
      subscription_metadata: collectMetadata(form, 'subscription_data[metadata]'),
    }
    sessions.set(id, session)
    json(res, 200, { id, object: 'checkout.session', url: `http://127.0.0.1:${port}/checkout/${id}` })
    return
  }

  if (req.method === 'GET' && path.startsWith('/v1/subscriptions/')) {
    const id = decodeURIComponent(path.slice('/v1/subscriptions/'.length))
    const sub = subscriptions.get(id)
    if (sub === undefined) {
      stripeError(res, 404, `No such subscription: '${id}'`, 'resource_missing')
      return
    }
    json(res, 200, sub)
    return
  }

  if (req.method === 'POST' && path === '/v1/billing_portal/sessions') {
    const form = new URLSearchParams(await readBody(req))
    const returnUrl = form.get('return_url') ?? ''
    json(res, 200, {
      id: nextId('bps'),
      object: 'billing_portal.session',
      url: `http://127.0.0.1:${port}/portal?return_url=${encodeURIComponent(returnUrl)}`,
    })
    return
  }

  stripeError(res, 404, `Unrecognized request URL (${req.method}: ${path})`)
}

/**
 * The hosted-checkout page: "completing" a session creates the subscription,
 * fires the signed checkout.session.completed webhook at moth (awaited, so
 * the entitlement is normally live before the browser lands back), then
 * redirects to the session's success_url.
 */
async function completeCheckout(req, res, sessionId) {
  const session = sessions.get(sessionId)
  if (session === undefined) {
    res.writeHead(404)
    res.end('unknown checkout session')
    return
  }
  const subId = nextId('sub')
  const periodEnd = Math.floor(Date.now() / 1000) + 30 * 24 * 3600
  const subscription = {
    id: subId,
    object: 'subscription',
    status: 'active',
    cancel_at_period_end: false,
    customer: session.customer,
    livemode: false,
    trial_end: null,
    metadata: session.subscription_metadata,
    items: {
      data: [
        {
          current_period_end: periodEnd,
          price: {
            id: session.price,
            product: 'prod_e2e_1',
            unit_amount: 999,
            currency: 'usd',
            recurring: { interval: 'month', interval_count: 1 },
          },
        },
      ],
    },
  }
  subscriptions.set(subId, subscription)
  session.subscription = subId

  await deliverWebhook({
    id: nextId('evt'),
    object: 'event',
    type: 'checkout.session.completed',
    livemode: false,
    data: {
      object: {
        id: session.id,
        object: 'checkout.session',
        subscription: subId,
        customer: session.customer,
        client_reference_id: session.client_reference_id,
        metadata: session.metadata,
      },
    },
  })

  res.writeHead(302, { location: session.success_url })
  res.end()
}

function collectMetadata(form, prefix) {
  const metadata = {}
  for (const [key, value] of form) {
    if (key.startsWith(`${prefix}[`) && key.endsWith(']')) {
      metadata[key.slice(prefix.length + 1, -1)] = value
    }
  }
  return metadata
}

server.listen(port, '127.0.0.1', () => {
  console.log(`stripe-double listening on 127.0.0.1:${port}`)
})
