import type { GenEnum, GenFile, GenMessage, GenService } from "@bufbuild/protobuf/codegenv2";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import type { Message } from "@bufbuild/protobuf";
/**
 * Describes the file moth/billing/v1/billing.proto.
 */
export declare const file_moth_billing_v1_billing: GenFile;
/**
 * Offering is the ordered set of products a paywall presents — the products
 * sharing an `offering` tag, in sort order. Every project has a default
 * offering ("default").
 *
 * @generated from message moth.billing.v1.Offering
 */
export type Offering = Message<"moth.billing.v1.Offering"> & {
    /**
     * Offering tag; "default" for the project's default offering.
     *
     * @generated from field: string identifier = 1;
     */
    identifier: string;
    /**
     * @generated from field: bool is_default = 2;
     */
    isDefault: boolean;
    /**
     * The products to display, in paywall order.
     *
     * @generated from field: repeated moth.billing.v1.OfferingProduct products = 3;
     */
    products: OfferingProduct[];
};
/**
 * Describes the message moth.billing.v1.Offering.
 * Use `create(OfferingSchema)` to create a new message.
 */
export declare const OfferingSchema: GenMessage<Offering>;
/**
 * OfferingProduct is one purchasable tier as the paywall needs it: enough to
 * render a card and match the native store product. Price/period are display
 * + analytics metadata; the native store read stays authoritative for the
 * localized price actually charged.
 *
 * @generated from message moth.billing.v1.OfferingProduct
 */
export type OfferingProduct = Message<"moth.billing.v1.OfferingProduct"> & {
    /**
     * Stable moth catalog identifier (e.g. "monthly"); the app never gates on
     * this — it gates on entitlements — but the SDK uses it to drive purchases.
     *
     * @generated from field: string identifier = 1;
     */
    identifier: string;
    /**
     * @generated from field: string display_name = 2;
     */
    displayName: string;
    /**
     * Store SKUs so the SDK can pair this tier with the native store product;
     * either may be empty when the tier ships on one store only.
     *
     * @generated from field: string apple_product_id = 3;
     */
    appleProductId: string;
    /**
     * @generated from field: string google_product_id = 4;
     */
    googleProductId: string;
    /**
     * @generated from field: string billing_period = 5;
     */
    billingPeriod: string;
    /**
     * @generated from field: int64 price_amount_micros = 6;
     */
    priceAmountMicros: bigint;
    /**
     * @generated from field: string currency = 7;
     */
    currency: string;
    /**
     * Trial/intro descriptor (display + analytics only).
     *
     * @generated from field: string trial_period = 8;
     */
    trialPeriod: string;
    /**
     * @generated from field: int64 intro_price_amount_micros = 9;
     */
    introPriceAmountMicros: bigint;
    /**
     * @generated from field: string intro_period = 10;
     */
    introPeriod: string;
    /**
     * The stable entitlement identifiers this product grants while active (e.g.
     * "pro"), so the paywall can label what a tier unlocks.
     *
     * @generated from field: repeated string entitlements = 11;
     */
    entitlements: string[];
    /**
     * @generated from field: int32 sort_order = 12;
     */
    sortOrder: number;
    /**
     * Whether this tier is the paywall's highlighted "most popular" tier (from
     * the paywall config's highlighted_product_identifier).
     *
     * @generated from field: bool highlighted = 13;
     */
    highlighted: boolean;
    /**
     * Stripe recurring Price id ("price_..."); empty when the tier does not sell
     * on the web (the React paywall marks such tiers unavailable).
     *
     * @generated from field: string stripe_price_id = 14;
     */
    stripePriceId: string;
};
/**
 * Describes the message moth.billing.v1.OfferingProduct.
 * Use `create(OfferingProductSchema)` to create a new message.
 */
export declare const OfferingProductSchema: GenMessage<OfferingProduct>;
/**
 * Paywall is the public, render-ready paywall configuration. Copy and layout
 * only — colors/typography inherit from the theme.
 *
 * @generated from message moth.billing.v1.Paywall
 */
export type Paywall = Message<"moth.billing.v1.Paywall"> & {
    /**
     * Identifies this version of the paywall config; changes on every admin
     * edit. Cache the paywall keyed by this value and echo it as
     * GetPaywallRequest.known_paywall_revision.
     *
     * @generated from field: string revision_id = 1;
     */
    revisionId: string;
    /**
     * @generated from field: string headline = 2;
     */
    headline: string;
    /**
     * @generated from field: string subtitle = 3;
     */
    subtitle: string;
    /**
     * Feature/benefit bullets, in display order.
     *
     * @generated from field: repeated string benefits = 4;
     */
    benefits: string[];
    /**
     * The offering tag whose products this paywall lists; pass it to
     * GetOfferings.offering. Empty selects the default offering.
     *
     * @generated from field: string offering = 5;
     */
    offering: string;
    /**
     * The product identifier to render as "most popular"; empty for none.
     *
     * @generated from field: string highlighted_product_identifier = 6;
     */
    highlightedProductIdentifier: string;
    /**
     * @generated from field: moth.billing.v1.PaywallLayout layout = 7;
     */
    layout: PaywallLayout;
    /**
     * Optional legal links rendered in the paywall footer.
     *
     * @generated from field: string terms_url = 8;
     */
    termsUrl: string;
    /**
     * @generated from field: string privacy_url = 9;
     */
    privacyUrl: string;
};
/**
 * Describes the message moth.billing.v1.Paywall.
 * Use `create(PaywallSchema)` to create a new message.
 */
export declare const PaywallSchema: GenMessage<Paywall>;
/**
 * @generated from message moth.billing.v1.GetOfferingsRequest
 */
export type GetOfferingsRequest = Message<"moth.billing.v1.GetOfferingsRequest"> & {
    /**
     * Offering tag; empty selects the project's default offering.
     *
     * @generated from field: string offering = 1;
     */
    offering: string;
};
/**
 * Describes the message moth.billing.v1.GetOfferingsRequest.
 * Use `create(GetOfferingsRequestSchema)` to create a new message.
 */
export declare const GetOfferingsRequestSchema: GenMessage<GetOfferingsRequest>;
/**
 * @generated from message moth.billing.v1.GetOfferingsResponse
 */
export type GetOfferingsResponse = Message<"moth.billing.v1.GetOfferingsResponse"> & {
    /**
     * @generated from field: moth.billing.v1.Offering offering = 1;
     */
    offering?: Offering | undefined;
};
/**
 * Describes the message moth.billing.v1.GetOfferingsResponse.
 * Use `create(GetOfferingsResponseSchema)` to create a new message.
 */
export declare const GetOfferingsResponseSchema: GenMessage<GetOfferingsResponse>;
/**
 * Copy is the resolved, localized paywall copy for the negotiated locale: the
 * paywall.* message key → localized-string map (headline, subtitle, benefit
 * bullets, CTA, legal labels), merged bundled-default → project-override. The
 * paywall copy keys are part of the same catalog as the auth-screen copy
 * (moth.auth.v1). The locale is negotiated server-side from the request's
 * Accept-Language / x-moth-language metadata; the client never dictates raw
 * copy. The structural Paywall message above stays authoritative for
 * layout/offering/tier selection.
 *
 * @generated from message moth.billing.v1.Copy
 */
export type Copy = Message<"moth.billing.v1.Copy"> & {
    /**
     * Opaque cache token identifying this (locale, override-revision) pair.
     * Cache `messages` keyed by it and echo it as
     * GetPaywallRequest.known_copy_revision; the response omits `messages` when
     * it still matches.
     *
     * @generated from field: string copy_revision = 1;
     */
    copyRevision: string;
    /**
     * The negotiated BCP-47 locale this copy is for (e.g. "fr").
     *
     * @generated from field: string locale = 2;
     */
    locale: string;
    /**
     * Resolved paywall.* message key → localized string for the negotiated
     * locale.
     *
     * @generated from field: map<string, string> messages = 3;
     */
    messages: {
        [key: string]: string;
    };
};
/**
 * Describes the message moth.billing.v1.Copy.
 * Use `create(CopySchema)` to create a new message.
 */
export declare const CopySchema: GenMessage<Copy>;
/**
 * @generated from message moth.billing.v1.GetPaywallRequest
 */
export type GetPaywallRequest = Message<"moth.billing.v1.GetPaywallRequest"> & {
    /**
     * The revision_id of the paywall the client has cached (empty on first
     * call). When it still matches the current revision, the response omits
     * `paywall`; see the caching contract on GetPaywall.
     *
     * @generated from field: string known_paywall_revision = 1;
     */
    knownPaywallRevision: string;
    /**
     * The copy_revision the client has cached for the locale it is about to
     * render (empty on first call). When it still matches the token the server
     * computes for the negotiated locale, the response's `copy` carries the
     * locale + copy_revision but omits `messages`; when it differs (or was
     * empty), `messages` is present. The negotiated locale comes from
     * Accept-Language / x-moth-language metadata, never from this body.
     *
     * @generated from field: string known_copy_revision = 2;
     */
    knownCopyRevision: string;
};
/**
 * Describes the message moth.billing.v1.GetPaywallRequest.
 * Use `create(GetPaywallRequestSchema)` to create a new message.
 */
export declare const GetPaywallRequestSchema: GenMessage<GetPaywallRequest>;
/**
 * @generated from message moth.billing.v1.GetPaywallResponse
 */
export type GetPaywallResponse = Message<"moth.billing.v1.GetPaywallResponse"> & {
    /**
     * Omitted when GetPaywallRequest.known_paywall_revision matches the current
     * revision; present otherwise (including for projects on the built-in
     * default paywall config).
     *
     * @generated from field: moth.billing.v1.Paywall paywall = 1;
     */
    paywall?: Paywall | undefined;
    /**
     * The localized paywall copy for the negotiated locale. Always present (it
     * carries the negotiated locale + copy_revision); its `messages` map is
     * omitted when GetPaywallRequest.known_copy_revision matches, present
     * otherwise — including for projects with no copy overrides.
     *
     * @generated from field: moth.billing.v1.Copy copy = 2;
     */
    copy?: Copy | undefined;
};
/**
 * Describes the message moth.billing.v1.GetPaywallResponse.
 * Use `create(GetPaywallResponseSchema)` to create a new message.
 */
export declare const GetPaywallResponseSchema: GenMessage<GetPaywallResponse>;
/**
 * CustomerInfo is the complete subscription picture for one user. Apps gate
 * features on active_entitlements (never on a product id): check whether the
 * stable entitlement identifier (e.g. "pro") is present.
 *
 * @generated from message moth.billing.v1.CustomerInfo
 */
export type CustomerInfo = Message<"moth.billing.v1.CustomerInfo"> & {
    /**
     * The entitlements the user currently holds. Empty means the free `none`
     * tier — a valid, expected state, not an error.
     *
     * @generated from field: repeated moth.billing.v1.Entitlement active_entitlements = 1;
     */
    activeEntitlements: Entitlement[];
    /**
     * The user's known subscriptions across stores (may include inactive ones,
     * for history/paywall display).
     *
     * @generated from field: repeated moth.billing.v1.ActiveSubscription subscriptions = 2;
     */
    subscriptions: ActiveSubscription[];
};
/**
 * Describes the message moth.billing.v1.CustomerInfo.
 * Use `create(CustomerInfoSchema)` to create a new message.
 */
export declare const CustomerInfoSchema: GenMessage<CustomerInfo>;
/**
 * Entitlement is one active capability the user holds.
 *
 * @generated from message moth.billing.v1.Entitlement
 */
export type Entitlement = Message<"moth.billing.v1.Entitlement"> & {
    /**
     * Stable identifier the app checks (e.g. "pro"). Never changes across app
     * releases even when the granting product does.
     *
     * @generated from field: string identifier = 1;
     */
    identifier: string;
    /**
     * When the entitlement lapses; unset for a non-expiring grant.
     *
     * @generated from field: google.protobuf.Timestamp expire_time = 2;
     */
    expireTime?: Timestamp | undefined;
    /**
     * Why it is active (store subscription vs operator grant).
     *
     * @generated from field: moth.billing.v1.EntitlementSource source = 3;
     */
    source: EntitlementSource;
    /**
     * The moth product identifier that granted it, when source is STORE; empty
     * for grants.
     *
     * @generated from field: string product_identifier = 4;
     */
    productIdentifier: string;
};
/**
 * Describes the message moth.billing.v1.Entitlement.
 * Use `create(EntitlementSchema)` to create a new message.
 */
export declare const EntitlementSchema: GenMessage<Entitlement>;
/**
 * ActiveSubscription is one of the user's store subscriptions.
 *
 * @generated from message moth.billing.v1.ActiveSubscription
 */
export type ActiveSubscription = Message<"moth.billing.v1.ActiveSubscription"> & {
    /**
     * The moth product identifier, when the store SKU is mapped; empty otherwise.
     *
     * @generated from field: string product_identifier = 1;
     */
    productIdentifier: string;
    /**
     * @generated from field: moth.billing.v1.Store store = 2;
     */
    store: Store;
    /**
     * @generated from field: moth.billing.v1.SubscriptionStatus status = 3;
     */
    status: SubscriptionStatus;
    /**
     * End of the current paid (or trial) period; the renewal date when
     * auto_renew is true.
     *
     * @generated from field: google.protobuf.Timestamp current_period_end = 4;
     */
    currentPeriodEnd?: Timestamp | undefined;
    /**
     * @generated from field: bool auto_renew = 5;
     */
    autoRenew: boolean;
    /**
     * Whether this subscription is a sandbox/test purchase.
     *
     * @generated from field: bool is_sandbox = 6;
     */
    isSandbox: boolean;
};
/**
 * Describes the message moth.billing.v1.ActiveSubscription.
 * Use `create(ActiveSubscriptionSchema)` to create a new message.
 */
export declare const ActiveSubscriptionSchema: GenMessage<ActiveSubscription>;
/**
 * @generated from message moth.billing.v1.GetCustomerInfoRequest
 */
export type GetCustomerInfoRequest = Message<"moth.billing.v1.GetCustomerInfoRequest"> & {};
/**
 * Describes the message moth.billing.v1.GetCustomerInfoRequest.
 * Use `create(GetCustomerInfoRequestSchema)` to create a new message.
 */
export declare const GetCustomerInfoRequestSchema: GenMessage<GetCustomerInfoRequest>;
/**
 * @generated from message moth.billing.v1.GetCustomerInfoResponse
 */
export type GetCustomerInfoResponse = Message<"moth.billing.v1.GetCustomerInfoResponse"> & {
    /**
     * @generated from field: moth.billing.v1.CustomerInfo customer_info = 1;
     */
    customerInfo?: CustomerInfo | undefined;
};
/**
 * Describes the message moth.billing.v1.GetCustomerInfoResponse.
 * Use `create(GetCustomerInfoResponseSchema)` to create a new message.
 */
export declare const GetCustomerInfoResponseSchema: GenMessage<GetCustomerInfoResponse>;
/**
 * @generated from message moth.billing.v1.SubmitPurchaseRequest
 */
export type SubmitPurchaseRequest = Message<"moth.billing.v1.SubmitPurchaseRequest"> & {
    /**
     * @generated from field: moth.billing.v1.Store store = 1;
     */
    store: Store;
    /**
     * The moth product identifier the app is purchasing (its own catalog id, not
     * the store SKU). moth maps it to the store product for validation.
     *
     * @generated from field: string product_identifier = 2;
     */
    productIdentifier: string;
    /**
     * The store receipt. For Apple, a StoreKit 2 signed transaction (JWS). For
     * Google, the Play Billing purchase token — pair it with
     * google_subscription_id (Google needs both to resolve the purchase).
     *
     * @generated from oneof moth.billing.v1.SubmitPurchaseRequest.receipt
     */
    receipt: {
        /**
         * @generated from field: string apple_jws_transaction = 3;
         */
        value: string;
        case: "appleJwsTransaction";
    } | {
        /**
         * @generated from field: string google_purchase_token = 4;
         */
        value: string;
        case: "googlePurchaseToken";
    } | {
        case: undefined;
        value?: undefined;
    };
    /**
     * Google Play subscription id (the store product id); required alongside
     * google_purchase_token, ignored for Apple.
     *
     * @generated from field: string google_subscription_id = 5;
     */
    googleSubscriptionId: string;
};
/**
 * Describes the message moth.billing.v1.SubmitPurchaseRequest.
 * Use `create(SubmitPurchaseRequestSchema)` to create a new message.
 */
export declare const SubmitPurchaseRequestSchema: GenMessage<SubmitPurchaseRequest>;
/**
 * @generated from message moth.billing.v1.SubmitPurchaseResponse
 */
export type SubmitPurchaseResponse = Message<"moth.billing.v1.SubmitPurchaseResponse"> & {
    /**
     * @generated from field: moth.billing.v1.CustomerInfo customer_info = 1;
     */
    customerInfo?: CustomerInfo | undefined;
};
/**
 * Describes the message moth.billing.v1.SubmitPurchaseResponse.
 * Use `create(SubmitPurchaseResponseSchema)` to create a new message.
 */
export declare const SubmitPurchaseResponseSchema: GenMessage<SubmitPurchaseResponse>;
/**
 * @generated from message moth.billing.v1.RestorePurchasesRequest
 */
export type RestorePurchasesRequest = Message<"moth.billing.v1.RestorePurchasesRequest"> & {
    /**
     * @generated from field: moth.billing.v1.Store store = 1;
     */
    store: Store;
    /**
     * The receipts to re-link. For Apple, StoreKit 2 signed transactions (JWS);
     * for Google, Play Billing purchase tokens.
     *
     * @generated from field: repeated string receipts = 2;
     */
    receipts: string[];
};
/**
 * Describes the message moth.billing.v1.RestorePurchasesRequest.
 * Use `create(RestorePurchasesRequestSchema)` to create a new message.
 */
export declare const RestorePurchasesRequestSchema: GenMessage<RestorePurchasesRequest>;
/**
 * @generated from message moth.billing.v1.RestorePurchasesResponse
 */
export type RestorePurchasesResponse = Message<"moth.billing.v1.RestorePurchasesResponse"> & {
    /**
     * @generated from field: moth.billing.v1.CustomerInfo customer_info = 1;
     */
    customerInfo?: CustomerInfo | undefined;
};
/**
 * Describes the message moth.billing.v1.RestorePurchasesResponse.
 * Use `create(RestorePurchasesResponseSchema)` to create a new message.
 */
export declare const RestorePurchasesResponseSchema: GenMessage<RestorePurchasesResponse>;
/**
 * @generated from message moth.billing.v1.CreateCheckoutSessionRequest
 */
export type CreateCheckoutSessionRequest = Message<"moth.billing.v1.CreateCheckoutSessionRequest"> & {
    /**
     * The moth product identifier to subscribe to (its own catalog id, not the
     * Stripe price id). The tier must carry a stripe_price_id.
     *
     * @generated from field: string product_identifier = 1;
     */
    productIdentifier: string;
    /**
     * Where Stripe redirects the browser after a completed checkout.
     *
     * @generated from field: string success_url = 2;
     */
    successUrl: string;
    /**
     * Where Stripe redirects the browser when the user backs out.
     *
     * @generated from field: string cancel_url = 3;
     */
    cancelUrl: string;
};
/**
 * Describes the message moth.billing.v1.CreateCheckoutSessionRequest.
 * Use `create(CreateCheckoutSessionRequestSchema)` to create a new message.
 */
export declare const CreateCheckoutSessionRequestSchema: GenMessage<CreateCheckoutSessionRequest>;
/**
 * @generated from message moth.billing.v1.CreateCheckoutSessionResponse
 */
export type CreateCheckoutSessionResponse = Message<"moth.billing.v1.CreateCheckoutSessionResponse"> & {
    /**
     * The Stripe-hosted Checkout URL to redirect the browser to.
     *
     * @generated from field: string url = 1;
     */
    url: string;
};
/**
 * Describes the message moth.billing.v1.CreateCheckoutSessionResponse.
 * Use `create(CreateCheckoutSessionResponseSchema)` to create a new message.
 */
export declare const CreateCheckoutSessionResponseSchema: GenMessage<CreateCheckoutSessionResponse>;
/**
 * @generated from message moth.billing.v1.CreateBillingPortalSessionRequest
 */
export type CreateBillingPortalSessionRequest = Message<"moth.billing.v1.CreateBillingPortalSessionRequest"> & {
    /**
     * Where Stripe redirects the browser when the user leaves the portal.
     *
     * @generated from field: string return_url = 1;
     */
    returnUrl: string;
};
/**
 * Describes the message moth.billing.v1.CreateBillingPortalSessionRequest.
 * Use `create(CreateBillingPortalSessionRequestSchema)` to create a new message.
 */
export declare const CreateBillingPortalSessionRequestSchema: GenMessage<CreateBillingPortalSessionRequest>;
/**
 * @generated from message moth.billing.v1.CreateBillingPortalSessionResponse
 */
export type CreateBillingPortalSessionResponse = Message<"moth.billing.v1.CreateBillingPortalSessionResponse"> & {
    /**
     * The Stripe-hosted Billing Portal URL to redirect the browser to.
     *
     * @generated from field: string url = 1;
     */
    url: string;
};
/**
 * Describes the message moth.billing.v1.CreateBillingPortalSessionResponse.
 * Use `create(CreateBillingPortalSessionResponseSchema)` to create a new message.
 */
export declare const CreateBillingPortalSessionResponseSchema: GenMessage<CreateBillingPortalSessionResponse>;
/**
 * PaywallLayout is the rendering variant the paywall screen uses; the token
 * space (colors/spacing/radius) always comes from the theme.
 *
 * @generated from enum moth.billing.v1.PaywallLayout
 */
export declare enum PaywallLayout {
    /**
     * @generated from enum value: PAYWALL_LAYOUT_UNSPECIFIED = 0;
     */
    UNSPECIFIED = 0,
    /**
     * One card per tier, side by side (the default).
     *
     * @generated from enum value: PAYWALL_LAYOUT_TILES = 1;
     */
    TILES = 1,
    /**
     * Tiers stacked as full-width rows.
     *
     * @generated from enum value: PAYWALL_LAYOUT_LIST = 2;
     */
    LIST = 2,
    /**
     * A single selected tier with a period toggle.
     *
     * @generated from enum value: PAYWALL_LAYOUT_COMPACT = 3;
     */
    COMPACT = 3
}
/**
 * Describes the enum moth.billing.v1.PaywallLayout.
 */
export declare const PaywallLayoutSchema: GenEnum<PaywallLayout>;
/**
 * Store identifies which app store a purchase or subscription belongs to.
 *
 * @generated from enum moth.billing.v1.Store
 */
export declare enum Store {
    /**
     * @generated from enum value: STORE_UNSPECIFIED = 0;
     */
    UNSPECIFIED = 0,
    /**
     * @generated from enum value: STORE_APPLE = 1;
     */
    APPLE = 1,
    /**
     * @generated from enum value: STORE_GOOGLE = 2;
     */
    GOOGLE = 2,
    /**
     * @generated from enum value: STORE_STRIPE = 3;
     */
    STRIPE = 3
}
/**
 * Describes the enum moth.billing.v1.Store.
 */
export declare const StoreSchema: GenEnum<Store>;
/**
 * SubscriptionStatus mirrors the store's renewal state, mapped to a small set
 * common to Apple and Google. active/trialing/in_grace_period/in_billing_retry
 * all keep access; paused/expired/revoked do not.
 *
 * @generated from enum moth.billing.v1.SubscriptionStatus
 */
export declare enum SubscriptionStatus {
    /**
     * @generated from enum value: SUBSCRIPTION_STATUS_UNSPECIFIED = 0;
     */
    UNSPECIFIED = 0,
    /**
     * @generated from enum value: SUBSCRIPTION_STATUS_ACTIVE = 1;
     */
    ACTIVE = 1,
    /**
     * @generated from enum value: SUBSCRIPTION_STATUS_TRIALING = 2;
     */
    TRIALING = 2,
    /**
     * @generated from enum value: SUBSCRIPTION_STATUS_IN_GRACE_PERIOD = 3;
     */
    IN_GRACE_PERIOD = 3,
    /**
     * Google "on hold": the renewal is being retried after a payment failure.
     *
     * @generated from enum value: SUBSCRIPTION_STATUS_IN_BILLING_RETRY = 4;
     */
    IN_BILLING_RETRY = 4,
    /**
     * @generated from enum value: SUBSCRIPTION_STATUS_PAUSED = 5;
     */
    PAUSED = 5,
    /**
     * @generated from enum value: SUBSCRIPTION_STATUS_EXPIRED = 6;
     */
    EXPIRED = 6,
    /**
     * @generated from enum value: SUBSCRIPTION_STATUS_REVOKED = 7;
     */
    REVOKED = 7
}
/**
 * Describes the enum moth.billing.v1.SubscriptionStatus.
 */
export declare const SubscriptionStatusSchema: GenEnum<SubscriptionStatus>;
/**
 * EntitlementSource explains why an entitlement is active.
 *
 * @generated from enum moth.billing.v1.EntitlementSource
 */
export declare enum EntitlementSource {
    /**
     * @generated from enum value: ENTITLEMENT_SOURCE_UNSPECIFIED = 0;
     */
    UNSPECIFIED = 0,
    /**
     * Granted by an active store subscription.
     *
     * @generated from enum value: ENTITLEMENT_SOURCE_STORE = 1;
     */
    STORE = 1,
    /**
     * Granted by an operator (promo/comp), independent of store state.
     *
     * @generated from enum value: ENTITLEMENT_SOURCE_GRANT = 2;
     */
    GRANT = 2,
    /**
     * The built-in free tier (no active subscription or grant). Reserved; the
     * free tier is normally conveyed by an empty active_entitlements list.
     *
     * @generated from enum value: ENTITLEMENT_SOURCE_NONE = 3;
     */
    NONE = 3
}
/**
 * Describes the enum moth.billing.v1.EntitlementSource.
 */
export declare const EntitlementSourceSchema: GenEnum<EntitlementSource>;
/**
 * BillingService is the client-facing subscription API, consumed by the moth
 * Flutter SDK. Authenticated exactly like AuthService: every call carries the
 * project publishable key in `x-moth-key: pk_...` request metadata AND a user
 * access token in the `Authorization: Bearer <jwt>` header — every RPC is
 * scoped to the signed-in user.
 *
 * The core contract: **a user always has a valid subscription state.** A
 * never-paid user, a free-tier user, and a user in a project that has declared
 * no products all get a well-formed CustomerInfo with an empty
 * active_entitlements list (the built-in `none` tier) — never an error. moth is
 * a validating mirror of the store: it never marks a subscription active on the
 * client's say-so, only after verifying a signed transaction or reading the
 * store's authoritative state.
 *
 * @generated from service moth.billing.v1.BillingService
 */
export declare const BillingService: GenService<{
    /**
     * GetCustomerInfo returns the signed-in user's active entitlements and
     * subscriptions. Always succeeds with a valid object; `none` (empty
     * entitlements) for free users. Cheap and safe to call on every app launch.
     *
     * @generated from rpc moth.billing.v1.BillingService.GetCustomerInfo
     */
    getCustomerInfo: {
        methodKind: "unary";
        input: typeof GetCustomerInfoRequestSchema;
        output: typeof GetCustomerInfoResponseSchema;
    };
    /**
     * SubmitPurchase hands moth the receipt of a purchase the app just completed
     * natively. moth validates it against the store, links the subscription to
     * the current user, derives entitlements, and returns the fresh CustomerInfo.
     *
     * @generated from rpc moth.billing.v1.BillingService.SubmitPurchase
     */
    submitPurchase: {
        methodKind: "unary";
        input: typeof SubmitPurchaseRequestSchema;
        output: typeof SubmitPurchaseResponseSchema;
    };
    /**
     * RestorePurchases re-links a user's existing store purchases to the current
     * account (new device, reinstall, account change), applying the store's own
     * transfer rules, then returns the fresh CustomerInfo.
     *
     * @generated from rpc moth.billing.v1.BillingService.RestorePurchases
     */
    restorePurchases: {
        methodKind: "unary";
        input: typeof RestorePurchasesRequestSchema;
        output: typeof RestorePurchasesResponseSchema;
    };
    /**
     * GetOfferings returns an offering's products for the paywall to display:
     * per product the catalog identifier, display name, store SKUs (so the SDK
     * can match the native store products), price/period metadata, trial/intro
     * descriptor, the entitlements it grants, sort order and the "most popular"
     * highlight flag. Unlike the three RPCs above this is publishable-key only
     * (no Bearer): a paywall renders before the user signs in.
     *
     * @generated from rpc moth.billing.v1.BillingService.GetOfferings
     */
    getOfferings: {
        methodKind: "unary";
        input: typeof GetOfferingsRequestSchema;
        output: typeof GetOfferingsResponseSchema;
    };
    /**
     * GetPaywall returns the project's public paywall configuration (copy,
     * benefit bullets, offering ref, layout, highlighted tier, legal links) with
     * a revision id, for the SDK's batteries-included paywall screen. Colors and
     * typography are NOT here — the paywall inherits them from the theme
     * (GetProjectConfig, milestone 06).
     *
     * Caching contract (identical to GetProjectConfig + theme): the client caches
     * the paywall keyed by revision_id and echoes it as
     * GetPaywallRequest.known_paywall_revision. When it still matches, the
     * response omits `paywall` and the client keeps rendering its cache
     * (stale-while-revalidate); when it differs (or was empty on first call),
     * `paywall` is present and the client replaces its cache. Publishable-key
     * only, like GetOfferings.
     *
     * @generated from rpc moth.billing.v1.BillingService.GetPaywall
     */
    getPaywall: {
        methodKind: "unary";
        input: typeof GetPaywallRequestSchema;
        output: typeof GetPaywallResponseSchema;
    };
    /**
     * CreateCheckoutSession starts a Stripe-hosted Checkout for a subscription to
     * the tier's Stripe price, bound to the signed-in user's Stripe customer
     * (created on demand). moth never renders a card field: the response is a
     * redirect URL to Stripe's hosted Checkout, and the resulting subscription
     * lands through the webhook like any other store event. Requires Bearer
     * (like GetCustomerInfo); fails with a precondition error when the project
     * has no Stripe credentials, and an invalid-argument error when the tier has
     * no Stripe price.
     *
     * @generated from rpc moth.billing.v1.BillingService.CreateCheckoutSession
     */
    createCheckoutSession: {
        methodKind: "unary";
        input: typeof CreateCheckoutSessionRequestSchema;
        output: typeof CreateCheckoutSessionResponseSchema;
    };
    /**
     * CreateBillingPortalSession returns a Stripe Billing Portal URL for the
     * signed-in user — cancel, payment-method and invoice management stay
     * Stripe-hosted, the web analogue of deep-linking to the stores'
     * subscription-management UI. Requires Bearer.
     *
     * @generated from rpc moth.billing.v1.BillingService.CreateBillingPortalSession
     */
    createBillingPortalSession: {
        methodKind: "unary";
        input: typeof CreateBillingPortalSessionRequestSchema;
        output: typeof CreateBillingPortalSessionResponseSchema;
    };
}>;
