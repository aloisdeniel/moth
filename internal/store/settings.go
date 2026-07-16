package store

import (
	"encoding/json"
	"time"
)

// ProjectSettings is the per-project auth policy, stored as a JSON column
// on the project row and editable through the admin projects API.
type ProjectSettings struct {
	// PasswordMinLength is the minimum accepted password length.
	PasswordMinLength int `json:"password_min_length"`
	// RequireEmailVerification blocks SignIn until the email is verified.
	RequireEmailVerification bool `json:"require_email_verification"`
	// AllowPublicSignup gates the SignUp RPC; invite-only projects set it
	// to false and create users through the server API instead.
	AllowPublicSignup bool `json:"allow_public_signup"`
	// EnumerationSafeSignup makes SignUp with an already-registered email
	// return OK (mailing the owner) instead of failing, so responses never
	// reveal whether an account exists.
	EnumerationSafeSignup bool `json:"enumeration_safe_signup"`
	// AccessTokenTTLSeconds is the JWT lifetime.
	AccessTokenTTLSeconds int `json:"access_token_ttl_seconds"`
	// RefreshTokenTTLDays is the sliding window extended on each rotation.
	RefreshTokenTTLDays int `json:"refresh_token_ttl_days"`
	// Google is the Sign in with Google configuration (public part; the
	// web client secret lives encrypted in project_provider_secrets).
	Google GoogleProviderSettings `json:"google"`
	// Apple is the Sign in with Apple configuration (public part; the .p8
	// private key lives encrypted in project_provider_secrets).
	Apple AppleProviderSettings `json:"apple"`
	// AutoLinkVerifiedEmail links a social identity to an existing account
	// when the provider asserts the same, verified email (default true).
	// Pointer so that a stored JSON without the key keeps the default while
	// an explicit false survives the parse-over-defaults round trip.
	AutoLinkVerifiedEmail *bool `json:"auto_link_verified_email,omitempty"`
	// RedirectSchemes are the custom URL schemes the web-redirect OAuth
	// fallback may redirect back to (open-redirect protection).
	RedirectSchemes []string `json:"redirect_schemes,omitempty"`
	// AnalyticsRetentionDays is how long raw analytics events are kept
	// before the rollup job prunes them (default 90).
	AnalyticsRetentionDays int `json:"analytics_retention_days"`
	// RollupTimezone is the IANA timezone name (e.g. "Europe/Paris") the
	// daily analytics rollup buckets days in (default "UTC").
	RollupTimezone string `json:"rollup_timezone"`
}

// RollupLocation resolves RollupTimezone, falling back to UTC when the
// stored name is empty or no longer resolves.
func (ps ProjectSettings) RollupLocation() *time.Location {
	if ps.RollupTimezone == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(ps.RollupTimezone)
	if err != nil {
		return time.UTC
	}
	return loc
}

// AutoLinkEnabled reports the effective auto_link_verified_email policy.
func (ps ProjectSettings) AutoLinkEnabled() bool {
	return ps.AutoLinkVerifiedEmail == nil || *ps.AutoLinkVerifiedEmail
}

// GoogleProviderSettings configures Sign in with Google for one project.
// The client IDs are the allowed `aud` values of Google ID tokens.
type GoogleProviderSettings struct {
	Enabled         bool   `json:"enabled"`
	WebClientID     string `json:"web_client_id"`
	IOSClientID     string `json:"ios_client_id"`
	AndroidClientID string `json:"android_client_id"`
}

// AppleProviderSettings configures Sign in with Apple for one project.
type AppleProviderSettings struct {
	Enabled    bool   `json:"enabled"`
	ServicesID string `json:"services_id"`
	TeamID     string `json:"team_id"`
	KeyID      string `json:"key_id"`
	// BundleIDs are accepted as `aud` on native Apple ID tokens.
	BundleIDs []string `json:"bundle_ids,omitempty"`
}

// DefaultProjectSettings returns the policy applied to new projects and to
// settings fields left at their zero value.
func DefaultProjectSettings() ProjectSettings {
	return ProjectSettings{
		PasswordMinLength:      8,
		AllowPublicSignup:      true,
		AccessTokenTTLSeconds:  15 * 60,
		RefreshTokenTTLDays:    30,
		AnalyticsRetentionDays: 90,
		RollupTimezone:         "UTC",
	}
}

// parseProjectSettings decodes the stored JSON over the defaults, then
// re-applies defaults to numeric fields left at zero.
func parseProjectSettings(raw string) (ProjectSettings, error) {
	ps := DefaultProjectSettings()
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &ps); err != nil {
			return ProjectSettings{}, err
		}
	}
	def := DefaultProjectSettings()
	if ps.PasswordMinLength <= 0 {
		ps.PasswordMinLength = def.PasswordMinLength
	}
	if ps.AccessTokenTTLSeconds <= 0 {
		ps.AccessTokenTTLSeconds = def.AccessTokenTTLSeconds
	}
	if ps.RefreshTokenTTLDays <= 0 {
		ps.RefreshTokenTTLDays = def.RefreshTokenTTLDays
	}
	if ps.AnalyticsRetentionDays <= 0 {
		ps.AnalyticsRetentionDays = def.AnalyticsRetentionDays
	}
	if ps.RollupTimezone == "" {
		ps.RollupTimezone = def.RollupTimezone
	}
	return ps, nil
}

func encodeProjectSettings(ps ProjectSettings) (string, error) {
	raw, err := json.Marshal(ps)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
