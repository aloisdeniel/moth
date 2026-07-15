package store

import "encoding/json"

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
}

// DefaultProjectSettings returns the policy applied to new projects and to
// settings fields left at their zero value.
func DefaultProjectSettings() ProjectSettings {
	return ProjectSettings{
		PasswordMinLength:     8,
		AllowPublicSignup:     true,
		AccessTokenTTLSeconds: 15 * 60,
		RefreshTokenTTLDays:   30,
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
	return ps, nil
}

func encodeProjectSettings(ps ProjectSettings) (string, error) {
	raw, err := json.Marshal(ps)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
