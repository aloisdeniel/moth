package setup

import adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"

// This file holds the shared "what does configured mean?" predicates. They
// are the single source of that knowledge: `moth doctor` runs them forward
// as health checks, and the admin GetProjectSetupStatus RPC runs them
// backward to derive the milestone-22 setup checklist — same probes,
// factored, never duplicated. All of them are pure functions over the admin
// config messages both callers already have.

// GoogleProviderHasClientID reports whether any Google client ID (web, iOS,
// Android) is configured — the minimum for Sign in with Google to work on at
// least one platform.
func GoogleProviderHasClientID(g *adminv1.GoogleProviderConfig) bool {
	return g.GetWebClientId() != "" || g.GetIosClientId() != "" || g.GetAndroidClientId() != ""
}

// GoogleProviderConfigured reports whether Sign in with Google is usable:
// enabled with at least one client ID.
func GoogleProviderConfigured(g *adminv1.GoogleProviderConfig) bool {
	return g.GetEnabled() && GoogleProviderHasClientID(g)
}

// AppleProviderMissing lists the required Sign in with Apple pieces that are
// absent (team ID, key ID, private key, services/bundle ID); empty when the
// configuration is complete. Enablement is the caller's concern.
func AppleProviderMissing(a *adminv1.AppleProviderConfig) []string {
	var missing []string
	if a.GetTeamId() == "" {
		missing = append(missing, "team ID")
	}
	if a.GetKeyId() == "" {
		missing = append(missing, "key ID")
	}
	if !a.GetHasPrivateKey() {
		missing = append(missing, "private key")
	}
	if a.GetServicesId() == "" && len(a.GetBundleIds()) == 0 {
		missing = append(missing, "services ID or bundle ID")
	}
	return missing
}

// AppleProviderConfigured reports whether Sign in with Apple is usable:
// enabled with nothing missing.
func AppleProviderConfigured(a *adminv1.AppleProviderConfig) bool {
	return a.GetEnabled() && len(AppleProviderMissing(a)) == 0
}

// AppleProviderMissingNativeOnly is the platform-aware variant of
// AppleProviderMissing for a project that ships on neither web nor Android:
// a bundle ID alone is sufficient there, because the native flow verifies
// Apple ID tokens against the bundle-ID audiences
// (internal/server/rpc/auth/oauth.go) and the Services ID / team ID / key ID
// / .p8 trio only signs the web-redirect flow and the best-effort code
// exchange. Used by the profile-aware setup checklist; `moth doctor` keeps
// AppleProviderMissing (it has no profile context). Enablement is the
// caller's concern.
func AppleProviderMissingNativeOnly(a *adminv1.AppleProviderConfig) []string {
	if len(a.GetBundleIds()) == 0 {
		return []string{"bundle ID"}
	}
	return nil
}

// AppleBillingMissing lists the required App Store Server API credential
// pieces that are absent; empty when the configuration is complete.
func AppleBillingMissing(a *adminv1.AppleBillingConfig) []string {
	var missing []string
	if !a.GetHasIapKey() {
		missing = append(missing, "In-App-Purchase .p8")
	}
	if a.GetIapKeyId() == "" {
		missing = append(missing, "key id")
	}
	if a.GetIapIssuerId() == "" {
		missing = append(missing, "issuer id")
	}
	if a.GetBundleId() == "" {
		missing = append(missing, "bundle id")
	}
	return missing
}

// GoogleBillingMissing lists the required Play Developer API credential
// pieces that are absent; empty when the configuration is complete.
func GoogleBillingMissing(g *adminv1.GoogleBillingConfig) []string {
	var missing []string
	if !g.GetHasServiceAccount() {
		missing = append(missing, "service-account JSON")
	}
	if g.GetPackageName() == "" {
		missing = append(missing, "package name")
	}
	return missing
}

// StripeBillingMissing lists the required Stripe credential pieces that are
// absent; empty when the configuration is complete.
func StripeBillingMissing(s *adminv1.StripeBillingConfig) []string {
	var missing []string
	if !s.GetHasSecretKey() {
		missing = append(missing, "secret key")
	}
	return missing
}
