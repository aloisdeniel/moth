package adminrpc

import (
	"context"
	"errors"
	"strings"
	"time"

	"connectrpc.com/connect"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/gen/moth/admin/v1/adminv1connect"
	"github.com/aloisdeniel/moth/internal/profile"
	"github.com/aloisdeniel/moth/internal/push"
	"github.com/aloisdeniel/moth/internal/setup"
	"github.com/aloisdeniel/moth/internal/store"
)

// ProfileHandler implements moth.admin.v1.ProfileService: the project setup
// profile (the milestone-22 wizard's answers — plain config, full
// replacement, no revisions, like the push settings) and the derived setup
// checklist. The checklist is recomputed from live configuration on every
// call using the same predicates `moth doctor` runs (internal/setup);
// nothing about completeness is ever stored, so it cannot go stale.
type ProfileHandler struct {
	store Store
	audit *Auditor
	// smtpConfigured reports whether the instance currently has a real SMTP
	// transport (the settings handler's view); nil is treated as not
	// configured.
	smtpConfigured func() bool
	now            func() time.Time
}

var _ adminv1connect.ProfileServiceHandler = (*ProfileHandler)(nil)

// NewProfileHandler builds the profile admin service. smtpConfigured is the
// instance settings handler's SMTPConfigured (nil: SMTP treated as not
// configured); now is injectable for tests (nil: time.Now).
func NewProfileHandler(st Store, auditor *Auditor, smtpConfigured func() bool, now func() time.Time) *ProfileHandler {
	if smtpConfigured == nil {
		smtpConfigured = func() bool { return false }
	}
	if now == nil {
		now = time.Now
	}
	return &ProfileHandler{store: st, audit: auditor, smtpConfigured: smtpConfigured, now: now}
}

// GetProfile returns the project's setup profile; a project created before
// the wizard has none (has_profile false, no profile message).
func (h *ProfileHandler) GetProfile(ctx context.Context, req *connect.Request[adminv1.GetProfileRequest]) (*connect.Response[adminv1.GetProfileResponse], error) {
	p, err := h.store.GetProject(ctx, req.Msg.ProjectId)
	if err != nil {
		return nil, projectErr(err)
	}
	c, ok := profile.FromStored(p.Profile)
	resp := &adminv1.GetProfileResponse{HasProfile: ok}
	if ok {
		resp.Profile = profileProto(c)
	}
	return connect.NewResponse(resp), nil
}

// UpdateProfile validates and installs a full replacement of the setup
// profile.
func (h *ProfileHandler) UpdateProfile(ctx context.Context, req *connect.Request[adminv1.UpdateProfileRequest]) (*connect.Response[adminv1.UpdateProfileResponse], error) {
	if req.Msg.Profile == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("profile is required"))
	}
	c := profileFromProto(req.Msg.Profile)
	if err := c.Validate(); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	raw, err := profile.Encode(c)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := h.store.SetProjectProfile(ctx, req.Msg.ProjectId, raw, h.now()); err != nil {
		return nil, projectErr(err)
	}
	h.audit.record(ctx, entry{
		Action: ActionProfileUpdate, TargetType: "profile", TargetID: req.Msg.ProjectId,
		ProjectID: req.Msg.ProjectId, Summary: "Updated the setup profile",
	})
	return connect.NewResponse(&adminv1.UpdateProfileResponse{
		Profile: profileProto(c),
	}), nil
}

// GetProjectSetupStatus derives the outstanding setup checklist from live
// configuration plus the profile's intent. Features the profile did not
// choose produce no item; each item disappears the moment the underlying
// configuration exists, however it got there (tabs, CLI, a teammate).
// Projects without a profile get an empty list and has_profile false.
func (h *ProfileHandler) GetProjectSetupStatus(ctx context.Context, req *connect.Request[adminv1.GetProjectSetupStatusRequest]) (*connect.Response[adminv1.GetProjectSetupStatusResponse], error) {
	p, err := h.store.GetProject(ctx, req.Msg.ProjectId)
	if err != nil {
		return nil, projectErr(err)
	}
	c, ok := profile.FromStored(p.Profile)
	if !ok {
		return connect.NewResponse(&adminv1.GetProjectSetupStatusResponse{HasProfile: false}), nil
	}
	resp := &adminv1.GetProjectSetupStatusResponse{
		HasProfile:         true,
		ChecklistDismissed: c.ChecklistDismissed,
	}

	if item, err := h.googleItem(ctx, p, c); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	} else if item != nil {
		resp.Items = append(resp.Items, item)
	}
	if item, err := h.appleItem(ctx, p, c); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	} else if item != nil {
		resp.Items = append(resp.Items, item)
	}
	billingItems, err := h.billingItems(ctx, p, c)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp.Items = append(resp.Items, billingItems...)
	if item := pushItem(p, c); item != nil {
		resp.Items = append(resp.Items, item)
	}
	if len(p.Theme) == 0 {
		resp.Items = append(resp.Items, &adminv1.SetupItem{
			Id:     "theme_default",
			Title:  "Customize the design",
			Detail: "The project still renders the built-in default theme.",
			Tab:    "design",
		})
	}
	if !h.smtpConfigured() {
		resp.Items = append(resp.Items, &adminv1.SetupItem{
			Id:    "smtp",
			Title: "Configure outgoing email (SMTP)",
			Detail: "No SMTP is configured at the instance level, so verification and " +
				"reset emails only appear in the server console. Configure it in the " +
				"instance Settings or moth.toml [smtp].",
			// Instance-level: no project tab finishes this.
		})
	}
	return connect.NewResponse(resp), nil
}

// googleItem returns the outstanding google_credentials item, or nil when
// the profile did not choose Google sign-in or the provider is usable.
func (h *ProfileHandler) googleItem(ctx context.Context, p store.Project, c profile.Config) (*adminv1.SetupItem, error) {
	if !c.GoogleSignIn {
		return nil, nil
	}
	hasSecret, err := providerSecretPresent(ctx, h.store, p.ID, store.ProviderSecretGoogleWebClientSecret)
	if err != nil {
		return nil, err
	}
	if setup.GoogleProviderConfigured(googleProviderProto(p.Settings.Google, hasSecret)) {
		return nil, nil
	}
	return &adminv1.SetupItem{
		Id:         "google_credentials",
		Title:      "Finish Google sign-in",
		Detail:     "Google sign-in was chosen, but the provider is not enabled with a client ID yet.",
		Tab:        "providers",
		CliCommand: "moth setup google",
	}, nil
}

// appleItem returns the outstanding apple_credentials item, or nil when the
// profile did not choose Apple sign-in or the provider is usable.
func (h *ProfileHandler) appleItem(ctx context.Context, p store.Project, c profile.Config) (*adminv1.SetupItem, error) {
	if !c.AppleSignIn {
		return nil, nil
	}
	hasKey, err := providerSecretPresent(ctx, h.store, p.ID, store.ProviderSecretApplePrivateKey)
	if err != nil {
		return nil, err
	}
	a := appleProviderProto(p.Settings.Apple, hasKey)
	// Platform-aware: a project shipping on neither web nor Android only
	// runs the native flow, which a bundle ID alone fully serves — the
	// team/key/.p8 trio must not nag as a permanent blocking item there.
	nativeOnly := !c.HasPlatform(profile.PlatformWeb) && !c.HasPlatform(profile.PlatformAndroid)
	missing := setup.AppleProviderMissing(a)
	if nativeOnly {
		missing = setup.AppleProviderMissingNativeOnly(a)
	}
	if a.GetEnabled() && len(missing) == 0 {
		return nil, nil
	}
	detail := "Sign in with Apple was chosen, but the provider is not enabled yet."
	if a.Enabled {
		detail = "Sign in with Apple is missing: " + strings.Join(missing, ", ") + "."
	}
	return &adminv1.SetupItem{
		Id:         "apple_credentials",
		Title:      "Finish Sign in with Apple",
		Detail:     detail,
		Tab:        "providers",
		CliCommand: "moth setup apple",
	}, nil
}

// billingStores pairs each store name with its display label, in the fixed
// apple/google/stripe checklist order.
var billingStores = []struct{ name, label string }{
	{store.SubscriptionStoreApple, "App Store"},
	{store.SubscriptionStoreGoogle, "Google Play"},
	{store.SubscriptionStoreStripe, "Stripe"},
}

// impliedStores maps the profile's platforms to the stores subscriptions
// must ship through: iOS → App Store, Android → Google Play, Web → Stripe.
func impliedStores(c profile.Config) map[string]bool {
	implied := map[string]bool{}
	if c.HasPlatform(profile.PlatformIOS) {
		implied[store.SubscriptionStoreApple] = true
	}
	if c.HasPlatform(profile.PlatformAndroid) {
		implied[store.SubscriptionStoreGoogle] = true
	}
	if c.HasPlatform(profile.PlatformWeb) {
		implied[store.SubscriptionStoreStripe] = true
	}
	return implied
}

// billingItems returns the outstanding billing_credentials and catalog_sync
// items for the platform-implied stores, or none when the profile does not
// sell subscriptions or everything is credentialed and in sync.
func (h *ProfileHandler) billingItems(ctx context.Context, p store.Project, c profile.Config) ([]*adminv1.SetupItem, error) {
	implied := impliedStores(c)
	if !c.SellsSubscriptions || len(implied) == 0 {
		return nil, nil
	}
	cred, err := h.store.GetBillingCredentials(ctx, p.ID)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return nil, err
	}
	// ErrNotFound leaves cred zero: every store then reads as missing.
	credMissing := map[string]bool{
		store.SubscriptionStoreApple:  len(setup.AppleBillingMissing(appleConfigProto(cred))) > 0,
		store.SubscriptionStoreGoogle: len(setup.GoogleBillingMissing(googleConfigProto(cred))) > 0,
		store.SubscriptionStoreStripe: len(setup.StripeBillingMissing(stripeConfigProto(cred))) > 0,
	}
	var items []*adminv1.SetupItem
	var missingLabels []string
	for _, s := range billingStores {
		if implied[s.name] && credMissing[s.name] {
			missingLabels = append(missingLabels, s.label)
		}
	}
	if len(missingLabels) > 0 {
		items = append(items, &adminv1.SetupItem{
			Id:         "billing_credentials",
			Title:      "Add store billing credentials",
			Detail:     "Store credentials are missing for: " + strings.Join(missingLabels, ", ") + ".",
			Tab:        "monetization",
			CliCommand: "moth setup billing",
		})
	}
	item, err := h.catalogSyncItem(ctx, p, implied)
	if err != nil {
		return nil, err
	}
	if item != nil {
		items = append(items, item)
	}
	return items, nil
}

// catalogSyncItem reports whether the product catalog is fully provisioned
// on every implied store: every product carries that store's SKU and its
// sync record says in_sync (a missing record is the implied pending state).
func (h *ProfileHandler) catalogSyncItem(ctx context.Context, p store.Project, implied map[string]bool) (*adminv1.SetupItem, error) {
	products, err := h.store.ListProducts(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	if len(products) == 0 {
		return &adminv1.SetupItem{
			Id:         "catalog_sync",
			Title:      "Define and sync the product catalog",
			Detail:     "Subscriptions were chosen, but no products are defined yet.",
			Tab:        "monetization",
			CliCommand: "moth setup billing",
		}, nil
	}
	syncs, err := h.store.ListProductStoreSyncs(ctx, p.ID, "")
	if err != nil {
		return nil, err
	}
	rows := map[string]store.ProductStoreSync{}
	for _, r := range syncs {
		rows[r.ProductID+"\x00"+r.Store] = r
	}
	// synced applies the same revision-drift demotion every other sync
	// surface runs (storeStatus, planResponse): a row recorded in_sync whose
	// product has since been edited in moth is drift, not parity.
	synced := func(prod store.Product, storeName string) bool {
		r, ok := rows[prod.ID+"\x00"+storeName]
		if !ok || r.Status != store.ProductSyncInSync {
			return false
		}
		return r.Revision == "" || r.Revision == productRevision(prod, storeName)
	}
	sku := func(prod store.Product, storeName string) string {
		switch storeName {
		case store.SubscriptionStoreApple:
			return prod.AppleProductID
		case store.SubscriptionStoreGoogle:
			return prod.GoogleProductID
		case store.SubscriptionStoreStripe:
			return prod.StripePriceID
		}
		return ""
	}
	var pendingLabels []string
	for _, s := range billingStores {
		if !implied[s.name] {
			continue
		}
		for _, prod := range products {
			if sku(prod, s.name) == "" || !synced(prod, s.name) {
				pendingLabels = append(pendingLabels, s.label)
				break
			}
		}
	}
	if len(pendingLabels) == 0 {
		return nil, nil
	}
	return &adminv1.SetupItem{
		Id:         "catalog_sync",
		Title:      "Sync the product catalog",
		Detail:     "The product catalog is not fully synced to: " + strings.Join(pendingLabels, ", ") + ".",
		Tab:        "monetization",
		CliCommand: "moth setup billing",
	}, nil
}

// pushItem returns the outstanding push item, or nil when the profile did
// not choose pushes or the configuration matches the intent. On a web
// platform readiness means enabled with a VAPID key (push_vapid); on a
// native-only project it means enabled (push_enable) — a disabled registry
// there (a failed or aborted wizard write) must still surface.
func pushItem(p store.Project, c profile.Config) *adminv1.SetupItem {
	if !c.SendsPushes {
		return nil
	}
	pc := push.FromStored(p.Push)
	if !c.HasPlatform(profile.PlatformWeb) {
		if pc.Enabled {
			return nil
		}
		return &adminv1.SetupItem{
			Id:     "push_enable",
			Title:  "Enable push registration",
			Detail: "Push notifications were chosen, but the push registry is disabled.",
			Tab:    "settings",
		}
	}
	if pc.Enabled && pc.WebPushVAPIDPublicKey != "" {
		return nil
	}
	detail := "The web platform needs a Web Push VAPID public key."
	if !pc.Enabled {
		detail = "Push notifications were chosen, but the push registry is disabled."
	}
	return &adminv1.SetupItem{
		Id:         "push_vapid",
		Title:      "Set up Web Push",
		Detail:     detail,
		Tab:        "settings",
		CliCommand: "npx web-push generate-vapid-keys",
	}
}

// --- proto mappers ---------------------------------------------------------

func profileProto(c profile.Config) *adminv1.Profile {
	msg := &adminv1.Profile{
		GoogleSignIn:       c.GoogleSignIn,
		AppleSignIn:        c.AppleSignIn,
		SellsSubscriptions: c.SellsSubscriptions,
		SendsPushes:        c.SendsPushes,
		ChecklistDismissed: c.ChecklistDismissed,
	}
	for _, p := range c.Platforms {
		msg.Platforms = append(msg.Platforms, profilePlatformProto(p))
	}
	return msg
}

// profileFromProto converts the admin message into the domain config,
// stamping the current schema version. Unknown platform values are dropped;
// Validate then rejects the profile when nothing usable remains.
func profileFromProto(msg *adminv1.Profile) profile.Config {
	c := profile.Config{
		Version:            profile.SchemaVersion,
		GoogleSignIn:       msg.GetGoogleSignIn(),
		AppleSignIn:        msg.GetAppleSignIn(),
		SellsSubscriptions: msg.GetSellsSubscriptions(),
		SendsPushes:        msg.GetSendsPushes(),
		ChecklistDismissed: msg.GetChecklistDismissed(),
	}
	for _, p := range msg.GetPlatforms() {
		if name := profilePlatformName(p); name != "" {
			c.Platforms = append(c.Platforms, name)
		}
	}
	return c
}

func profilePlatformProto(p string) adminv1.ProfilePlatform {
	switch p {
	case profile.PlatformIOS:
		return adminv1.ProfilePlatform_PROFILE_PLATFORM_IOS
	case profile.PlatformAndroid:
		return adminv1.ProfilePlatform_PROFILE_PLATFORM_ANDROID
	case profile.PlatformWeb:
		return adminv1.ProfilePlatform_PROFILE_PLATFORM_WEB
	default:
		return adminv1.ProfilePlatform_PROFILE_PLATFORM_UNSPECIFIED
	}
}

func profilePlatformName(p adminv1.ProfilePlatform) string {
	switch p {
	case adminv1.ProfilePlatform_PROFILE_PLATFORM_IOS:
		return profile.PlatformIOS
	case adminv1.ProfilePlatform_PROFILE_PLATFORM_ANDROID:
		return profile.PlatformAndroid
	case adminv1.ProfilePlatform_PROFILE_PLATFORM_WEB:
		return profile.PlatformWeb
	default:
		return ""
	}
}
