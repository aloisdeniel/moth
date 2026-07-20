// Demo backend for the monetization services: the product/entitlement
// catalog, the paywall offering, per-user subscriptions and operator grants,
// store API credentials and the store-catalog reconcile. The seeded world is
// "Aurora Pro": one entitlement unlocked by monthly/annual/lifetime tiers,
// a handful of paying subscribers, and Apple + Google + Stripe wired up.

import { BillingCredentialsService } from "../../gen/moth/admin/v1/billing_credentials_pb";
import { EntitlementService } from "../../gen/moth/admin/v1/entitlement_pb";
import {
  MonetizationService,
  ProductSyncStatus,
  SyncAction,
} from "../../gen/moth/admin/v1/monetization_pb";
import { ProductService } from "../../gen/moth/admin/v1/product_pb";
import {
  Store,
  SubscriptionService,
  SubscriptionStatus,
} from "../../gen/moth/admin/v1/subscription_pb";
import {
  ADMIN,
  CHURNED_ID,
  demoId,
  ENTITLEMENT_PRO_ID,
  LIFETIME_ID,
  PEOPLE,
  PRODUCT_ANNUAL_ID,
  PRODUCT_LIFETIME_ID,
  PRODUCT_MONTHLY_ID,
  PROJECT_MAIN,
  SUBSCRIBER_IDS,
  TRIAL_ID,
} from "../ids";
import { handle } from "../transport";
import type { Millis } from "../util";
import {
  daysAgo,
  failedPrecondition,
  invalidArgument,
  minutesAgo,
  notFound,
  now,
  randomId,
  ts,
} from "../util";

// ---------- Slice ----------

export interface DemoProduct {
  id: string;
  projectId: string;
  identifier: string;
  displayName: string;
  appleProductId: string;
  googleProductId: string;
  stripePriceId: string;
  stripeProductId: string;
  billingPeriod: string;
  priceAmountMicros: number;
  currency: string;
  trialPeriod: string;
  introPriceAmountMicros: number;
  introPeriod: string;
  offering: string;
  sortOrder: number;
  entitlementIds: string[];
  createTime: Millis;
  updateTime: Millis;
}

export interface DemoEntitlement {
  id: string;
  projectId: string;
  identifier: string;
  displayName: string;
  createTime: Millis;
  updateTime: Millis;
}

export interface DemoSubscription {
  id: string;
  projectId: string;
  userId: string;
  store: Store;
  productId: string;
  status: SubscriptionStatus;
  // Unset for one-time (lifetime) purchases.
  currentPeriodEnd?: Millis;
  autoRenew: boolean;
  environment: string;
  storeTransactionId: string;
  createTime: Millis;
  updateTime: Millis;
}

export interface DemoGrant {
  id: string;
  projectId: string;
  userId: string;
  entitlementId: string;
  expireTime?: Millis;
  reason: string;
  grantedBy: string;
  createTime: Millis;
  revokeTime?: Millis;
}

// Store credentials keep only the non-secret fields plus the has_* presence
// flags — like the real server, the demo never keeps (or returns) a secret.
export interface DemoAppleBilling {
  iapKeyId: string;
  iapIssuerId: string;
  hasIapKey: boolean;
  bundleId: string;
  appAppleId: string;
  hasNotificationSecret: boolean;
  notificationUrl: string;
}

export interface DemoGoogleBilling {
  hasServiceAccount: boolean;
  packageName: string;
  pubsubTopic: string;
  hasRtdnSecret: boolean;
}

export interface DemoStripeBilling {
  hasSecretKey: boolean;
  hasWebhookSecret: boolean;
  webhookEndpointId: string;
}

export interface DemoProjectBilling {
  apple: DemoAppleBilling;
  google: DemoGoogleBilling;
  stripe: DemoStripeBilling;
}

// What the store held for a product at the last sync; the diff against the
// current catalog drives drift/UPDATE plans.
export interface DemoStoreProductSync {
  syncTime: Millis;
  revision: string;
  displayName: string;
  priceAmountMicros: number;
  billingPeriod: string;
}

export interface DemoStoreSync {
  lastSyncTime: Millis;
  products: Record<string, DemoStoreProductSync | undefined>;
}

type StoreKey = "apple" | "google" | "stripe";

export type DemoStoreCatalog = Partial<Record<StoreKey, DemoStoreSync>>;

export interface MonetizationSlice {
  products: DemoProduct[];
  entitlements: DemoEntitlement[];
  subscriptions: DemoSubscription[];
  entitlementGrants: DemoGrant[];
  billingCredentials: Record<string, DemoProjectBilling | undefined>;
  storeCatalog: Record<string, DemoStoreCatalog | undefined>;
}

// ---------- Seed ----------

const inDays = (d: number): Millis => daysAgo(-d);

export function seedMonetization(): MonetizationSlice {
  const pid = PROJECT_MAIN.id;
  const catalogBorn = daysAgo(178);

  const products: DemoProduct[] = [
    {
      id: PRODUCT_ANNUAL_ID,
      projectId: pid,
      identifier: "annual",
      displayName: "Aurora Pro Annual",
      appleProductId: "app.aurora.pro.annual",
      googleProductId: "aurora_pro_annual",
      stripePriceId: "price_1PDemoAuroraAnnual1",
      stripeProductId: "prod_DemoAuroraPro",
      billingPeriod: "yearly",
      priceAmountMicros: 39_990_000,
      currency: "USD",
      trialPeriod: "7-day free trial",
      introPriceAmountMicros: 0,
      introPeriod: "",
      offering: "default",
      sortOrder: 0,
      entitlementIds: [ENTITLEMENT_PRO_ID],
      createTime: catalogBorn,
      updateTime: daysAgo(9),
    },
    {
      id: PRODUCT_MONTHLY_ID,
      projectId: pid,
      identifier: "monthly",
      displayName: "Aurora Pro Monthly",
      appleProductId: "app.aurora.pro.monthly",
      googleProductId: "aurora_pro_monthly",
      stripePriceId: "price_1PDemoAuroraMonthly",
      stripeProductId: "prod_DemoAuroraPro",
      billingPeriod: "monthly",
      priceAmountMicros: 4_990_000,
      currency: "USD",
      trialPeriod: "7-day free trial",
      introPriceAmountMicros: 0,
      introPeriod: "",
      offering: "default",
      sortOrder: 1,
      entitlementIds: [ENTITLEMENT_PRO_ID],
      createTime: catalogBorn,
      updateTime: daysAgo(9),
    },
    {
      id: PRODUCT_LIFETIME_ID,
      projectId: pid,
      identifier: "lifetime",
      displayName: "Aurora Pro Lifetime",
      appleProductId: "app.aurora.pro.lifetime",
      googleProductId: "aurora_pro_lifetime",
      // Sold on the app stores only; a Stripe push would provision it.
      stripePriceId: "",
      stripeProductId: "",
      billingPeriod: "",
      priceAmountMicros: 99_990_000,
      currency: "USD",
      trialPeriod: "",
      introPriceAmountMicros: 0,
      introPeriod: "",
      offering: "",
      sortOrder: 2,
      entitlementIds: [ENTITLEMENT_PRO_ID],
      createTime: daysAgo(120),
      updateTime: daysAgo(9),
    },
  ];

  const entitlements: DemoEntitlement[] = [
    {
      id: ENTITLEMENT_PRO_ID,
      projectId: pid,
      identifier: "pro",
      displayName: "Aurora Pro",
      createTime: catalogBorn,
      updateTime: catalogBorn,
    },
  ];

  let subSeq = 0;
  const sub = (
    userId: string,
    productId: string,
    store: Store,
    status: SubscriptionStatus,
    start: Millis,
    end: Millis | undefined,
    autoRenew: boolean,
    txn: string,
    updated?: Millis,
  ): DemoSubscription => ({
    id: demoId("subs", ++subSeq),
    projectId: pid,
    userId,
    store,
    productId,
    status,
    currentPeriodEnd: end,
    autoRenew,
    environment: "production",
    storeTransactionId: txn,
    createTime: start,
    updateTime: updated ?? start,
  });

  const A = SubscriptionStatus.ACTIVE;
  const subscriptions: DemoSubscription[] = [
    // Active auto-renewing subscribers (started after each person's signup).
    sub(SUBSCRIBER_IDS[0], PRODUCT_ANNUAL_ID, Store.APPLE, A, daysAgo(170), inDays(195), true, "200000086543201"), // Maya
    sub(SUBSCRIBER_IDS[1], PRODUCT_ANNUAL_ID, Store.GOOGLE, A, daysAgo(130), inDays(235), true, "gpa.3312-7789-2045-51160"), // Yuki
    sub(SUBSCRIBER_IDS[2], PRODUCT_MONTHLY_ID, Store.GOOGLE, A, daysAgo(98), inDays(22), true, "gpa.5527-1044-9821-70413"), // Priya
    sub(SUBSCRIBER_IDS[3], PRODUCT_MONTHLY_ID, Store.APPLE, A, daysAgo(58), inDays(2), true, "200000091237408"), // Liam
    sub(SUBSCRIBER_IDS[4], PRODUCT_MONTHLY_ID, Store.GOOGLE, A, daysAgo(21), inDays(9), true, "gpa.8804-3317-6650-28794"), // Ingrid
    // Omar — free trial ending in a few days.
    sub(TRIAL_ID, PRODUCT_MONTHLY_ID, Store.GOOGLE, SubscriptionStatus.TRIALING, daysAgo(2), inDays(5), true, "gpa.1108-4472-9917-30256"),
    // Claire — canceled, lapsed weeks ago.
    sub(CHURNED_ID, PRODUCT_MONTHLY_ID, Store.APPLE, SubscriptionStatus.EXPIRED, daysAgo(120), daysAgo(25), false, "200000078804113", daysAgo(25)),
    // Amara — one-time lifetime purchase; no renewal date.
    sub(LIFETIME_ID, PRODUCT_LIFETIME_ID, Store.APPLE, A, daysAgo(150), undefined, false, "200000082291554"),
  ];

  const entitlementGrants: DemoGrant[] = [
    {
      id: demoId("grnt", 1),
      projectId: pid,
      userId: PEOPLE[1].id, // Jonas
      entitlementId: ENTITLEMENT_PRO_ID,
      reason: "Founding beta tester comp",
      grantedBy: ADMIN.email,
      createTime: daysAgo(140),
    },
    {
      id: demoId("grnt", 2),
      projectId: pid,
      userId: PEOPLE[16].id, // Marco
      entitlementId: ENTITLEMENT_PRO_ID,
      reason: "Support goodwill after a double charge",
      grantedBy: ADMIN.email,
      createTime: daysAgo(50),
      revokeTime: daysAgo(36),
    },
    {
      id: demoId("grnt", 3),
      projectId: pid,
      userId: PEOPLE[18].id, // Sven
      entitlementId: ENTITLEMENT_PRO_ID,
      expireTime: inDays(21),
      reason: "App Store reviewer comp",
      grantedBy: ADMIN.email,
      createTime: daysAgo(7),
    },
  ];

  const billingCredentials: MonetizationSlice["billingCredentials"] = {
    [pid]: {
      apple: {
        iapKeyId: "8KQ2R4HXF3",
        iapIssuerId: "69a6de70-03db-47e3-e053-5b8c7c11a4d1",
        hasIapKey: true,
        bundleId: "app.aurora.journal",
        appAppleId: "6449120347",
        hasNotificationSecret: true,
        notificationUrl: `https://auth.aurora.example/billing/apple/notifications/${PROJECT_MAIN.slug}`,
      },
      google: {
        hasServiceAccount: true,
        packageName: "app.aurora.journal",
        pubsubTopic: "projects/aurora-journal-demo/topics/play-rtdn",
        hasRtdnSecret: true,
      },
      stripe: {
        hasSecretKey: true,
        hasWebhookSecret: true,
        webhookEndpointId: "we_1PxDemoAurora7kQ2",
      },
    },
  };

  // Everything synced a few hours ago; snapshots match the live catalog, so
  // every store reports in-sync until a product is edited.
  const snap = (p: DemoProduct, syncTime: Millis, revision: string): DemoStoreProductSync => ({
    syncTime,
    revision,
    displayName: p.displayName,
    priceAmountMicros: p.priceAmountMicros,
    billingPeriod: p.billingPeriod,
  });
  const appleSync = minutesAgo(185);
  const googleSync = minutesAgo(178);
  const stripeSync = minutesAgo(174);
  const [annual, monthly, lifetime] = products;
  const storeCatalog: MonetizationSlice["storeCatalog"] = {
    [pid]: {
      apple: {
        lastSyncTime: appleSync,
        products: {
          [annual.id]: snap(annual, appleSync, "rev-4c19a2"),
          [monthly.id]: snap(monthly, appleSync, "rev-77b0e4"),
          [lifetime.id]: snap(lifetime, appleSync, "rev-b3d901"),
        },
      },
      google: {
        lastSyncTime: googleSync,
        products: {
          [annual.id]: snap(annual, googleSync, "2025071802"),
          [monthly.id]: snap(monthly, googleSync, "2025071802"),
          [lifetime.id]: snap(lifetime, googleSync, "2025071801"),
        },
      },
      stripe: {
        lastSyncTime: stripeSync,
        products: {
          [annual.id]: snap(annual, stripeSync, "price_1PDemoAuroraAnnual1"),
          [monthly.id]: snap(monthly, stripeSync, "price_1PDemoAuroraMonthly"),
        },
      },
    },
  };

  // PROJECT_SIDE seeds nothing: empty catalog, no credentials, never synced.
  return {
    products,
    entitlements,
    subscriptions,
    entitlementGrants,
    billingCredentials,
    storeCatalog,
  };
}

// ---------- Shared helpers ----------

function projectProducts(state: MonetizationSlice, projectId: string): DemoProduct[] {
  return state.products
    .filter((p) => p.projectId === projectId)
    .sort((a, b) => a.sortOrder - b.sortOrder || a.createTime - b.createTime);
}

function toProduct(p: DemoProduct) {
  return {
    id: p.id,
    identifier: p.identifier,
    displayName: p.displayName,
    appleProductId: p.appleProductId,
    googleProductId: p.googleProductId,
    stripePriceId: p.stripePriceId,
    stripeProductId: p.stripeProductId,
    billingPeriod: p.billingPeriod,
    priceAmountMicros: BigInt(p.priceAmountMicros),
    currency: p.currency,
    trialPeriod: p.trialPeriod,
    introPriceAmountMicros: BigInt(p.introPriceAmountMicros),
    introPeriod: p.introPeriod,
    offering: p.offering,
    sortOrder: p.sortOrder,
    entitlementIds: [...p.entitlementIds],
    createTime: ts(p.createTime),
    updateTime: ts(p.updateTime),
  };
}

function toEntitlement(e: DemoEntitlement) {
  return {
    id: e.id,
    identifier: e.identifier,
    displayName: e.displayName,
    createTime: ts(e.createTime),
    updateTime: ts(e.updateTime),
  };
}

function toSubscription(s: DemoSubscription) {
  return {
    id: s.id,
    userId: s.userId,
    store: s.store,
    productId: s.productId,
    status: s.status,
    currentPeriodEnd: s.currentPeriodEnd === undefined ? undefined : ts(s.currentPeriodEnd),
    autoRenew: s.autoRenew,
    environment: s.environment,
    storeTransactionId: s.storeTransactionId,
    createTime: ts(s.createTime),
    updateTime: ts(s.updateTime),
  };
}

function toGrant(g: DemoGrant) {
  return {
    id: g.id,
    entitlementId: g.entitlementId,
    expireTime: g.expireTime === undefined ? undefined : ts(g.expireTime),
    reason: g.reason,
    grantedBy: g.grantedBy,
    createTime: ts(g.createTime),
    revokeTime: g.revokeTime === undefined ? undefined : ts(g.revokeTime),
  };
}

function millisFromTimestamp(
  t: { seconds: bigint; nanos: number } | undefined,
): Millis | undefined {
  if (t === undefined) return undefined;
  return Number(t.seconds) * 1000 + Math.floor(t.nanos / 1e6);
}

function emptyBilling(): DemoProjectBilling {
  return {
    apple: {
      iapKeyId: "",
      iapIssuerId: "",
      hasIapKey: false,
      bundleId: "",
      appAppleId: "",
      hasNotificationSecret: false,
      notificationUrl: "",
    },
    google: {
      hasServiceAccount: false,
      packageName: "",
      pubsubTopic: "",
      hasRtdnSecret: false,
    },
    stripe: {
      hasSecretKey: false,
      hasWebhookSecret: false,
      webhookEndpointId: "",
    },
  };
}

function storeKeyOf(store: Store): StoreKey {
  switch (store) {
    case Store.APPLE:
      return "apple";
    case Store.GOOGLE:
      return "google";
    case Store.STRIPE:
      return "stripe";
    default:
      throw invalidArgument("store must be specified");
  }
}

function storeNameOf(key: StoreKey): string {
  return key === "apple" ? "Apple" : key === "google" ? "Google" : "Stripe";
}

// The store SKU a product maps to for one store ("" when unmapped).
function storeSku(p: DemoProduct, key: StoreKey): string {
  return key === "apple" ? p.appleProductId : key === "google" ? p.googleProductId : p.stripePriceId;
}

function credentialsPresent(billing: DemoProjectBilling | undefined, key: StoreKey): boolean {
  if (!billing) return false;
  if (key === "apple") return billing.apple.hasIapKey;
  if (key === "google") return billing.google.hasServiceAccount;
  return billing.stripe.hasSecretKey;
}

function notificationsWired(billing: DemoProjectBilling | undefined, key: StoreKey): boolean {
  if (!billing) return false;
  if (key === "apple") {
    return billing.apple.hasNotificationSecret && billing.apple.notificationUrl !== "";
  }
  if (key === "google") {
    return billing.google.hasRtdnSecret && billing.google.pubsubTopic !== "";
  }
  return billing.stripe.hasWebhookSecret && billing.stripe.webhookEndpointId !== "";
}

function priceLabel(micros: number, currency: string): string {
  if (micros === 0) return "0.00" + (currency ? ` ${currency}` : "");
  return `${(micros / 1_000_000).toFixed(2)}${currency ? ` ${currency}` : ""}`;
}

interface FieldChange {
  field: string;
  current: string;
  desired: string;
}

// The store-side diff for one product: what a reconcile would push.
function productChanges(p: DemoProduct, rec: DemoStoreProductSync | undefined): FieldChange[] {
  const changes: FieldChange[] = [];
  const curName = rec?.displayName ?? "";
  const curPrice = rec?.priceAmountMicros ?? -1;
  const curPeriod = rec?.billingPeriod ?? "";
  if (curName !== p.displayName && p.displayName !== "") {
    changes.push({ field: "localization.en-US.name", current: curName, desired: p.displayName });
  }
  if (curPrice !== p.priceAmountMicros) {
    changes.push({
      field: "price",
      current: rec ? priceLabel(curPrice, p.currency) : "",
      desired: priceLabel(p.priceAmountMicros, p.currency),
    });
  }
  if (curPeriod !== p.billingPeriod && p.billingPeriod !== "") {
    changes.push({ field: "billing_period", current: curPeriod, desired: p.billingPeriod });
  }
  return changes;
}

function productSyncStatusOf(
  p: DemoProduct,
  rec: DemoStoreProductSync | undefined,
): ProductSyncStatus {
  if (!rec) return ProductSyncStatus.PENDING;
  return productChanges(p, rec).length === 0
    ? ProductSyncStatus.IN_SYNC
    : ProductSyncStatus.DRIFT;
}

// ---------- Registration ----------

export function registerMonetization(): void {
  registerProducts();
  registerEntitlements();
  registerOfferingAndCatalog();
  registerSubscriptions();
  registerBillingCredentials();
}

// ---------- ProductService ----------

function registerProducts(): void {
  handle(ProductService.method.listProducts, (state: MonetizationSlice, req) => {
    return { products: projectProducts(state, req.projectId).map(toProduct) };
  });

  handle(ProductService.method.getProduct, (state: MonetizationSlice, req) => {
    const p = state.products.find((x) => x.projectId === req.projectId && x.id === req.id);
    if (!p) throw notFound("product");
    return { product: toProduct(p) };
  });

  handle(ProductService.method.createProduct, (state: MonetizationSlice, req) => {
    const input = req.product;
    if (!input || input.identifier.trim() === "") {
      throw invalidArgument("product identifier is required");
    }
    const identifier = input.identifier.trim();
    if (state.products.some((x) => x.projectId === req.projectId && x.identifier === identifier)) {
      throw invalidArgument(`a product with identifier "${identifier}" already exists`);
    }
    const t = now();
    const p: DemoProduct = {
      id: randomId(),
      projectId: req.projectId,
      identifier,
      displayName: input.displayName,
      appleProductId: input.appleProductId,
      googleProductId: input.googleProductId,
      stripePriceId: input.stripePriceId,
      stripeProductId: input.stripeProductId,
      billingPeriod: input.billingPeriod,
      priceAmountMicros: Number(input.priceAmountMicros),
      currency: input.currency,
      trialPeriod: input.trialPeriod,
      introPriceAmountMicros: Number(input.introPriceAmountMicros),
      introPeriod: input.introPeriod,
      offering: input.offering,
      sortOrder: input.sortOrder,
      entitlementIds: [...input.entitlementIds],
      createTime: t,
      updateTime: t,
    };
    state.products.push(p);
    return { product: toProduct(p) };
  });

  handle(ProductService.method.updateProduct, (state: MonetizationSlice, req) => {
    const p = state.products.find((x) => x.projectId === req.projectId && x.id === req.id);
    if (!p) throw notFound("product");
    const input = req.product;
    if (!input) throw invalidArgument("product is required");
    // The identifier is immutable in the UI; keep the stored one when the
    // request leaves it blank.
    if (input.identifier.trim() !== "") p.identifier = input.identifier.trim();
    p.displayName = input.displayName;
    p.appleProductId = input.appleProductId;
    p.googleProductId = input.googleProductId;
    p.stripePriceId = input.stripePriceId;
    p.stripeProductId = input.stripeProductId;
    p.billingPeriod = input.billingPeriod;
    p.priceAmountMicros = Number(input.priceAmountMicros);
    p.currency = input.currency;
    p.trialPeriod = input.trialPeriod;
    p.introPriceAmountMicros = Number(input.introPriceAmountMicros);
    p.introPeriod = input.introPeriod;
    p.offering = input.offering;
    p.sortOrder = input.sortOrder;
    p.entitlementIds = [...input.entitlementIds];
    p.updateTime = now();
    return { product: toProduct(p) };
  });

  handle(ProductService.method.deleteProduct, (state: MonetizationSlice, req) => {
    const i = state.products.findIndex((x) => x.projectId === req.projectId && x.id === req.id);
    if (i === -1) throw notFound("product");
    state.products.splice(i, 1);
    // Drop the product's store sync records too.
    const catalog = state.storeCatalog[req.projectId];
    if (catalog) {
      for (const key of Object.keys(catalog) as StoreKey[]) {
        const record = catalog[key];
        if (record) delete record.products[req.id];
      }
    }
    return {};
  });
}

// ---------- EntitlementService ----------

function registerEntitlements(): void {
  handle(EntitlementService.method.listEntitlements, (state: MonetizationSlice, req) => {
    const entitlements = state.entitlements
      .filter((e) => e.projectId === req.projectId)
      .sort((a, b) => a.createTime - b.createTime)
      .map(toEntitlement);
    return { entitlements };
  });

  handle(EntitlementService.method.createEntitlement, (state: MonetizationSlice, req) => {
    const identifier = req.identifier.trim();
    if (identifier === "") throw invalidArgument("entitlement identifier is required");
    if (state.entitlements.some((e) => e.projectId === req.projectId && e.identifier === identifier)) {
      throw invalidArgument(`an entitlement with identifier "${identifier}" already exists`);
    }
    const t = now();
    const e: DemoEntitlement = {
      id: randomId(),
      projectId: req.projectId,
      identifier,
      displayName: req.displayName,
      createTime: t,
      updateTime: t,
    };
    state.entitlements.push(e);
    return { entitlement: toEntitlement(e) };
  });

  handle(EntitlementService.method.updateEntitlement, (state: MonetizationSlice, req) => {
    const e = state.entitlements.find((x) => x.projectId === req.projectId && x.id === req.id);
    if (!e) throw notFound("entitlement");
    e.displayName = req.displayName;
    e.updateTime = now();
    return { entitlement: toEntitlement(e) };
  });

  handle(EntitlementService.method.deleteEntitlement, (state: MonetizationSlice, req) => {
    const i = state.entitlements.findIndex(
      (x) => x.projectId === req.projectId && x.id === req.id,
    );
    if (i === -1) throw notFound("entitlement");
    state.entitlements.splice(i, 1);
    // Cascade: drop the entitlement from product grants and operator grants.
    for (const p of state.products) {
      if (p.projectId === req.projectId) {
        p.entitlementIds = p.entitlementIds.filter((id) => id !== req.id);
      }
    }
    state.entitlementGrants = state.entitlementGrants.filter(
      (g) => !(g.projectId === req.projectId && g.entitlementId === req.id),
    );
    return {};
  });
}

// ---------- MonetizationService (offering + store catalog) ----------

function offeringResponse(state: MonetizationSlice, projectId: string, tag: string) {
  const inOffering = projectProducts(state, projectId).filter((p) => p.offering === tag);
  return {
    identifier: tag,
    isDefault: tag === "default",
    productIds: inOffering.map((p) => p.id),
    products: inOffering.map(toProduct),
  };
}

function registerOfferingAndCatalog(): void {
  handle(MonetizationService.method.getOffering, (state: MonetizationSlice, req) => {
    const tag = req.offering === "" ? "default" : req.offering;
    return { offering: offeringResponse(state, req.projectId, tag) };
  });

  handle(MonetizationService.method.reorderOffering, (state: MonetizationSlice, req) => {
    const tag = req.offering === "" ? "default" : req.offering;
    const current = projectProducts(state, req.projectId).filter((p) => p.offering === tag);
    const currentIds = current.map((p) => p.id);
    const requested = [...req.productIds];
    if (
      requested.length !== currentIds.length ||
      new Set(requested).size !== requested.length ||
      !currentIds.every((id) => requested.includes(id))
    ) {
      throw invalidArgument(
        "product_ids must contain every product in the offering exactly once",
      );
    }
    for (const p of current) {
      p.sortOrder = requested.indexOf(p.id);
    }
    return { offering: offeringResponse(state, req.projectId, tag) };
  });

  handle(MonetizationService.method.getStoreCatalogStatus, (state: MonetizationSlice, req) => {
    const products = projectProducts(state, req.projectId);
    const billing = state.billingCredentials[req.projectId];
    const catalog = state.storeCatalog[req.projectId];
    const stores = ([Store.APPLE, Store.GOOGLE, Store.STRIPE] as const).map((store) => {
      const key = storeKeyOf(store);
      const record = catalog?.[key];
      let inSync = 0;
      let drift = 0;
      let unmapped = 0;
      const states = [];
      for (const p of products) {
        const sku = storeSku(p, key);
        if (sku === "") {
          unmapped++;
          continue;
        }
        const rec = record?.products[p.id];
        const status = productSyncStatusOf(p, rec);
        if (status === ProductSyncStatus.IN_SYNC) inSync++;
        if (status === ProductSyncStatus.DRIFT) drift++;
        states.push({
          productId: p.id,
          identifier: p.identifier,
          storeProductId: sku,
          status,
          revision: rec?.revision ?? "",
          error: "",
          lastSyncTime: rec ? ts(rec.syncTime) : undefined,
        });
      }
      return {
        store,
        credentialsPresent: credentialsPresent(billing, key),
        notificationsWired: notificationsWired(billing, key),
        productsTotal: products.length,
        productsInSync: inSync,
        productsDrift: drift,
        productsError: 0,
        productsUnmapped: unmapped,
        lastSyncTime: record ? ts(record.lastSyncTime) : undefined,
        products: states,
      };
    });
    return { stores };
  });

  handle(MonetizationService.method.syncStoreCatalog, (state: MonetizationSlice, req) => {
    const key = storeKeyOf(req.store);
    const billing = state.billingCredentials[req.projectId];
    if (!credentialsPresent(billing, key)) {
      throw failedPrecondition(
        `${storeNameOf(key)} API credentials are not configured — add them on the Monetization tab first`,
      );
    }
    const products = projectProducts(state, req.projectId);
    // Stripe provisions unmapped tiers itself (it creates the product +
    // price); the app stores only reconcile products with an SKU set.
    const candidates =
      key === "stripe" ? products : products.filter((p) => storeSku(p, key) !== "");
    const record = state.storeCatalog[req.projectId]?.[key];

    const planned = candidates.map((p) => {
      const rec = record?.products[p.id];
      const sku = storeSku(p, key);
      const creating = key === "stripe" ? p.stripePriceId === "" : rec === undefined;
      const changes = productChanges(p, creating ? undefined : rec);
      const action = creating
        ? SyncAction.CREATE
        : changes.length === 0
          ? SyncAction.NOOP
          : SyncAction.UPDATE;
      return { p, sku, action, changes };
    });
    const inSync = planned.every((x) => x.action === SyncAction.NOOP);

    const globalSteps = notificationsWired(billing, key)
      ? []
      : [
          key === "apple"
            ? {
                title: "Register the App Store Server Notifications URL",
                detail:
                  "Paste the notification URL from the Billing credentials card into App Store Connect → your app → App Information (V2, Production and Sandbox).",
                url: "https://appstoreconnect.apple.com/apps",
                values: [],
              }
            : key === "google"
              ? {
                  title: "Wire Real-time developer notifications",
                  detail:
                    "Create a Cloud Pub/Sub topic, register it in Play Console → Monetization setup, and add a push subscription pointing at the RTDN URL from the Billing credentials card.",
                  url: "https://play.google.com/console",
                  values: [],
                }
              : {
                  title: "Register the Stripe webhook endpoint",
                  detail:
                    "Add an endpoint with the webhook URL from the Billing credentials card (events: checkout.session.completed, customer.subscription.*) and store its signing secret.",
                  url: "https://dashboard.stripe.com/webhooks",
                  values: [],
                },
        ];

    if (req.dryRun) {
      return {
        store: req.store,
        inSync,
        items: planned.map(({ p, sku, action, changes }) => ({
          productId: p.id,
          identifier: p.identifier,
          storeProductId: sku,
          action,
          summary:
            action === SyncAction.CREATE
              ? `create ${p.identifier} at ${priceLabel(p.priceAmountMicros, p.currency)}${p.billingPeriod ? ` / ${p.billingPeriod}` : ""}`
              : action === SyncAction.UPDATE
                ? `update ${sku || p.identifier}: ${changes.map((c) => c.field).join(", ")}`
                : "",
          changes,
          guidedSteps:
            key === "apple" && action === SyncAction.CREATE
              ? [
                  {
                    title: `Submit ${p.identifier} for review`,
                    detail:
                      "The App Store Connect API cannot submit a new subscription for review — submit it with your next app version.",
                    url: "https://appstoreconnect.apple.com/apps",
                    values: [],
                  },
                ]
              : [],
          error: "",
        })),
        guidedSteps: globalSteps,
      };
    }

    // Apply: write the snapshots so a second push reports in-sync/all-noop.
    const t = now();
    const catalog = (state.storeCatalog[req.projectId] ??= {});
    const rec = (catalog[key] ??= { lastSyncTime: t, products: {} });
    rec.lastSyncTime = t;
    const items = planned.map(({ p, action, changes }) => {
      if (key === "stripe" && p.stripePriceId === "") {
        p.stripePriceId = `price_demo_${randomId().slice(0, 8)}`;
        if (p.stripeProductId === "") {
          p.stripeProductId = `prod_demo_${randomId().slice(0, 8)}`;
        }
      }
      const sku = storeSku(p, key);
      rec.products[p.id] = {
        syncTime: t,
        revision: key === "stripe" ? sku : `rev-${randomId().slice(0, 6)}`,
        displayName: p.displayName,
        priceAmountMicros: p.priceAmountMicros,
        billingPeriod: p.billingPeriod,
      };
      return {
        productId: p.id,
        identifier: p.identifier,
        storeProductId: sku,
        action,
        summary:
          action === SyncAction.CREATE
            ? `created ${sku} at ${priceLabel(p.priceAmountMicros, p.currency)}${p.billingPeriod ? ` / ${p.billingPeriod}` : ""}`
            : action === SyncAction.UPDATE
              ? `updated ${sku}: ${changes.map((c) => c.field).join(", ")}`
              : "",
        changes,
        guidedSteps:
          key === "apple" && action === SyncAction.CREATE
            ? [
                {
                  title: `Submit ${p.identifier} for review`,
                  detail:
                    "The App Store Connect API cannot submit a new subscription for review — submit it with your next app version.",
                  url: "https://appstoreconnect.apple.com/apps",
                  values: [],
                },
              ]
            : [],
        status: ProductSyncStatus.IN_SYNC,
        error: "",
      };
    });
    return { store: req.store, inSync: true, items, guidedSteps: globalSteps };
  });
}

// ---------- SubscriptionService ----------

function registerSubscriptions(): void {
  handle(SubscriptionService.method.listUserSubscriptions, (state: MonetizationSlice, req) => {
    const subscriptions = state.subscriptions
      .filter((s) => s.projectId === req.projectId && s.userId === req.userId)
      .sort((a, b) => b.createTime - a.createTime)
      .map(toSubscription);
    const grants = state.entitlementGrants
      .filter((g) => g.projectId === req.projectId && g.userId === req.userId)
      .sort((a, b) => b.createTime - a.createTime)
      .map(toGrant);
    return { subscriptions, grants };
  });

  handle(SubscriptionService.method.getUserSubscription, (state: MonetizationSlice, req) => {
    const s = state.subscriptions.find((x) => x.projectId === req.projectId && x.id === req.id);
    if (!s) throw notFound("subscription");
    return { subscription: toSubscription(s) };
  });

  handle(SubscriptionService.method.grantEntitlement, (state: MonetizationSlice, req) => {
    if (req.userId === "") throw invalidArgument("user_id is required");
    const ent = state.entitlements.find(
      (e) => e.projectId === req.projectId && e.id === req.entitlementId,
    );
    if (!ent) throw notFound("entitlement");
    const g: DemoGrant = {
      id: randomId(),
      projectId: req.projectId,
      userId: req.userId,
      entitlementId: ent.id,
      expireTime: millisFromTimestamp(req.expireTime),
      reason: req.reason,
      grantedBy: ADMIN.email,
      createTime: now(),
    };
    state.entitlementGrants.push(g);
    return { grant: toGrant(g) };
  });

  handle(SubscriptionService.method.revokeGrant, (state: MonetizationSlice, req) => {
    const g = state.entitlementGrants.find(
      (x) => x.projectId === req.projectId && x.id === req.grantId,
    );
    if (!g) throw notFound("grant");
    if (g.revokeTime !== undefined) throw failedPrecondition("grant is already revoked");
    g.revokeTime = now();
    return { grant: toGrant(g) };
  });
}

// ---------- BillingCredentialsService ----------

function billingResponse(billing: DemoProjectBilling) {
  // Secrets are write-only: reads carry the has_* flags and empty secret
  // fields, exactly like the real server.
  return {
    apple: {
      iapKeyId: billing.apple.iapKeyId,
      iapIssuerId: billing.apple.iapIssuerId,
      iapKeyP8: "",
      hasIapKey: billing.apple.hasIapKey,
      bundleId: billing.apple.bundleId,
      appAppleId: billing.apple.appAppleId,
      notificationSecret: "",
      hasNotificationSecret: billing.apple.hasNotificationSecret,
      notificationUrl: billing.apple.notificationUrl,
    },
    google: {
      serviceAccountJson: "",
      hasServiceAccount: billing.google.hasServiceAccount,
      packageName: billing.google.packageName,
      pubsubTopic: billing.google.pubsubTopic,
      rtdnSecret: "",
      hasRtdnSecret: billing.google.hasRtdnSecret,
    },
    stripe: {
      secretKey: "",
      hasSecretKey: billing.stripe.hasSecretKey,
      webhookSecret: "",
      hasWebhookSecret: billing.stripe.hasWebhookSecret,
      webhookEndpointId: billing.stripe.webhookEndpointId,
    },
  };
}

function registerBillingCredentials(): void {
  handle(BillingCredentialsService.method.getBillingCredentials, (state: MonetizationSlice, req) => {
    return billingResponse(state.billingCredentials[req.projectId] ?? emptyBilling());
  });

  handle(
    BillingCredentialsService.method.updateBillingCredentials,
    (state: MonetizationSlice, req) => {
      const billing = state.billingCredentials[req.projectId] ?? emptyBilling();
      state.billingCredentials[req.projectId] = billing;
      if (req.apple) {
        billing.apple.iapKeyId = req.apple.iapKeyId;
        billing.apple.iapIssuerId = req.apple.iapIssuerId;
        billing.apple.bundleId = req.apple.bundleId;
        billing.apple.appAppleId = req.apple.appAppleId;
        // Write-only secrets: a non-empty value replaces the stored one (we
        // keep only the presence flag), empty keeps it.
        if (req.apple.iapKeyP8 !== "") billing.apple.hasIapKey = true;
        if (req.apple.notificationSecret !== "") billing.apple.hasNotificationSecret = true;
        if (req.apple.notificationUrl !== "") {
          billing.apple.notificationUrl = req.apple.notificationUrl;
        }
      }
      if (req.google) {
        billing.google.packageName = req.google.packageName;
        billing.google.pubsubTopic = req.google.pubsubTopic;
        if (req.google.serviceAccountJson !== "") billing.google.hasServiceAccount = true;
        if (req.google.rtdnSecret !== "") billing.google.hasRtdnSecret = true;
      }
      if (req.stripe) {
        if (req.stripe.secretKey !== "") billing.stripe.hasSecretKey = true;
        if (req.stripe.webhookSecret !== "") billing.stripe.hasWebhookSecret = true;
        if (req.stripe.webhookEndpointId !== "") {
          billing.stripe.webhookEndpointId = req.stripe.webhookEndpointId;
        }
      }
      return billingResponse(billing);
    },
  );
}
