// Package i18n is moth's self-contained message catalog and locale
// negotiation. It mirrors the design-system model (internal/theme): a small,
// curated, closed set of message keys — one per string an end user sees on
// the SDK screens, the hosted pages, and the emails — each with a bundled
// English default and a handful of bundled translations. A project customizes
// nothing and is still fully localized; per-project overrides (stored
// elsewhere, passed in) layer on top exactly like theme tokens.
//
// The package is deliberately dependency-free: no database, no proto, no
// network. Callers negotiate a locale from a request header, then Resolve the
// effective key→value map (bundled defaults merged with the project's
// overrides) and Interpolate the placeholders.
//
// # Placeholder contract
//
// Message values may contain {name} placeholders (e.g. {app}, {email}). The
// set of placeholders in a key's English default is that key's contract:
// every translation and every project override for that key must keep the
// same placeholders (see Validate). Interpolation is a literal {name}→value
// substitution — no pluralization or ICU-message syntax (out of scope, see
// plan/15).
package i18n

import (
	"regexp"
	"sort"
)

// Screen groups the keys a single end-user surface renders. The SDK screens
// (sign in / sign up / password reset / verify email / paywall) map one-to-one
// to a Flutter screen; ScreenHosted and ScreenEmail cover the server-rendered
// pages and the transactional emails.
type Screen string

const (
	ScreenSignIn        Screen = "sign_in"
	ScreenSignUp        Screen = "sign_up"
	ScreenPasswordReset Screen = "password_reset"
	ScreenVerifyEmail   Screen = "verify_email"
	ScreenPaywall       Screen = "paywall"
	ScreenHosted        Screen = "hosted"
	ScreenEmail         Screen = "email"
)

// Key is a catalog message key, formatted "<screen>.<name>". The set is
// closed: only the constants below are valid, and IsKey reports membership.
type Key string

// Sign-in screen keys.
const (
	SignInTitle          Key = "sign_in.title"
	SignInSubtitle       Key = "sign_in.subtitle"
	SignInEmailLabel     Key = "sign_in.email_label"
	SignInPasswordLabel  Key = "sign_in.password_label"
	SignInSubmit         Key = "sign_in.submit"
	SignInForgotPassword Key = "sign_in.forgot_password"
	SignInNoAccount      Key = "sign_in.no_account"
	SignInSwitchToSignUp Key = "sign_in.switch_to_sign_up"
	SignInErrorInvalid   Key = "sign_in.error_invalid"
)

// Sign-up screen keys.
const (
	SignUpTitle           Key = "sign_up.title"
	SignUpSubtitle        Key = "sign_up.subtitle"
	SignUpEmailLabel      Key = "sign_up.email_label"
	SignUpPasswordLabel   Key = "sign_up.password_label"
	SignUpSubmit          Key = "sign_up.submit"
	SignUpHaveAccount     Key = "sign_up.have_account"
	SignUpSwitchToSignIn  Key = "sign_up.switch_to_sign_in"
	SignUpLegal           Key = "sign_up.legal"
	SignUpErrorEmailTaken Key = "sign_up.error_email_taken"
)

// Password-reset screen keys (request form + set-new-password form).
const (
	PasswordResetTitle             Key = "password_reset.title"
	PasswordResetSubtitle          Key = "password_reset.subtitle"
	PasswordResetEmailLabel        Key = "password_reset.email_label"
	PasswordResetSubmit            Key = "password_reset.submit"
	PasswordResetBackToSignIn      Key = "password_reset.back_to_sign_in"
	PasswordResetSent              Key = "password_reset.sent"
	PasswordResetNewPasswordLabel  Key = "password_reset.new_password_label"
	PasswordResetNewPasswordSubmit Key = "password_reset.new_password_submit"
	PasswordResetSuccess           Key = "password_reset.success"
)

// Verify-email screen keys.
const (
	VerifyEmailTitle    Key = "verify_email.title"
	VerifyEmailSubtitle Key = "verify_email.subtitle"
	VerifyEmailResend   Key = "verify_email.resend"
	VerifyEmailSuccess  Key = "verify_email.success"
	VerifyEmailExpired  Key = "verify_email.expired"
)

// Paywall screen keys.
const (
	PaywallTitle    Key = "paywall.title"
	PaywallSubtitle Key = "paywall.subtitle"
	PaywallCTA      Key = "paywall.cta"
	PaywallRestore  Key = "paywall.restore"
	PaywallTerms    Key = "paywall.terms"
)

// Hosted-page keys (server-rendered verify / reset / email-change / OAuth
// success pages). The footer is shared across pages.
const (
	HostedResetTitle         Key = "hosted.reset_title"
	HostedResetPasswordLabel Key = "hosted.reset_password_label"
	HostedResetSubmit        Key = "hosted.reset_submit"
	HostedResetSuccess       Key = "hosted.reset_success"
	HostedVerifySuccess      Key = "hosted.verify_success"
	HostedVerifyFailed       Key = "hosted.verify_failed"
	HostedEmailChangeSuccess Key = "hosted.email_change_success"
	HostedOAuthSuccess       Key = "hosted.oauth_success"
	HostedFooter             Key = "hosted.footer"
	// Hosted reset inline validation.
	HostedResetTooShort Key = "hosted.reset_too_short"
	// Hosted OAuth callback success body + error pages.
	HostedOAuthCode                Key = "hosted.oauth_code"
	HostedOAuthIncompleteTitle     Key = "hosted.oauth_incomplete_title"
	HostedOAuthIncomplete          Key = "hosted.oauth_incomplete"
	HostedOAuthFailedTitle         Key = "hosted.oauth_failed_title"
	HostedOAuthErrProviderDisabled Key = "hosted.oauth_error_provider_disabled"
	HostedOAuthErrInvalidRedirect  Key = "hosted.oauth_error_invalid_redirect"
	HostedOAuthErrInvalidToken     Key = "hosted.oauth_error_invalid_token"
	HostedOAuthErrProviderToken    Key = "hosted.oauth_error_provider_token"
	HostedOAuthErrEmailExists      Key = "hosted.oauth_error_email_exists"
	HostedOAuthErrUserDisabled     Key = "hosted.oauth_error_user_disabled"
	HostedOAuthErrSignupClosed     Key = "hosted.oauth_error_signup_closed"
	HostedOAuthErrInvalid          Key = "hosted.oauth_error_invalid"
)

// Email keys (subject + body paragraph + button label per transactional mail,
// plus a shared ignore-notice line).
const (
	EmailVerifySubject Key = "email.verify_subject"
	EmailVerifyBody    Key = "email.verify_body"
	EmailVerifyButton  Key = "email.verify_button"
	EmailResetSubject  Key = "email.reset_subject"
	EmailResetBody     Key = "email.reset_body"
	EmailResetButton   Key = "email.reset_button"
	EmailIgnoreNotice  Key = "email.ignore_notice"
	// Email-change confirmation (to the new address) + changed notice (to the
	// old address, with a revert link).
	EmailChangeSubject  Key = "email.change_subject"
	EmailChangeBody     Key = "email.change_body"
	EmailChangeButton   Key = "email.change_button"
	EmailChangedSubject Key = "email.changed_subject"
	EmailChangedBody    Key = "email.changed_body"
	EmailChangedRevert  Key = "email.changed_revert"
	EmailChangedButton  Key = "email.changed_button"
)

// entry binds a key to its screen and its canonical English default. The
// English defaults live in Go (not a data file) so the key set, its grouping,
// and the placeholder contract are all reviewable in one typed place;
// non-English translations ship as embedded JSON (see bundle.go).
type entry struct {
	Key     Key
	Screen  Screen
	Default string
}

// entries is the closed catalog, ordered by screen. Every key an end user can
// see appears here exactly once.
var entries = []entry{
	// sign_in
	{SignInTitle, ScreenSignIn, "Sign in"},
	{SignInSubtitle, ScreenSignIn, "Welcome back to {app}."},
	{SignInEmailLabel, ScreenSignIn, "Email"},
	{SignInPasswordLabel, ScreenSignIn, "Password"},
	{SignInSubmit, ScreenSignIn, "Sign in"},
	{SignInForgotPassword, ScreenSignIn, "Forgot password?"},
	{SignInNoAccount, ScreenSignIn, "Don't have an account?"},
	{SignInSwitchToSignUp, ScreenSignIn, "Sign up"},
	{SignInErrorInvalid, ScreenSignIn, "Incorrect email or password."},

	// sign_up
	{SignUpTitle, ScreenSignUp, "Create account"},
	{SignUpSubtitle, ScreenSignUp, "Create your {app} account."},
	{SignUpEmailLabel, ScreenSignUp, "Email"},
	{SignUpPasswordLabel, ScreenSignUp, "Password"},
	{SignUpSubmit, ScreenSignUp, "Create account"},
	{SignUpHaveAccount, ScreenSignUp, "Already have an account?"},
	{SignUpSwitchToSignIn, ScreenSignUp, "Sign in"},
	{SignUpLegal, ScreenSignUp, "By continuing you agree to our Terms and Privacy Policy."},
	{SignUpErrorEmailTaken, ScreenSignUp, "An account with this email already exists."},

	// password_reset
	{PasswordResetTitle, ScreenPasswordReset, "Reset password"},
	{PasswordResetSubtitle, ScreenPasswordReset, "Enter your email and we'll send you a reset link."},
	{PasswordResetEmailLabel, ScreenPasswordReset, "Email"},
	{PasswordResetSubmit, ScreenPasswordReset, "Send reset link"},
	{PasswordResetBackToSignIn, ScreenPasswordReset, "Back to sign in"},
	{PasswordResetSent, ScreenPasswordReset, "If an account exists for {email}, a reset link is on its way."},
	{PasswordResetNewPasswordLabel, ScreenPasswordReset, "New password"},
	{PasswordResetNewPasswordSubmit, ScreenPasswordReset, "Set new password"},
	{PasswordResetSuccess, ScreenPasswordReset, "Your password has been reset."},

	// verify_email
	{VerifyEmailTitle, ScreenVerifyEmail, "Verify your email"},
	{VerifyEmailSubtitle, ScreenVerifyEmail, "We sent a verification link to {email}."},
	{VerifyEmailResend, ScreenVerifyEmail, "Resend email"},
	{VerifyEmailSuccess, ScreenVerifyEmail, "Your email address is verified."},
	{VerifyEmailExpired, ScreenVerifyEmail, "This verification link has expired."},

	// paywall
	{PaywallTitle, ScreenPaywall, "Unlock {app} Premium"},
	{PaywallSubtitle, ScreenPaywall, "Get unlimited access to every feature."},
	{PaywallCTA, ScreenPaywall, "Continue"},
	{PaywallRestore, ScreenPaywall, "Restore purchases"},
	{PaywallTerms, ScreenPaywall, "Payment is charged to your account. Cancel anytime."},

	// hosted
	{HostedResetTitle, ScreenHosted, "Reset your password"},
	{HostedResetPasswordLabel, ScreenHosted, "New password"},
	{HostedResetSubmit, ScreenHosted, "Reset password"},
	{HostedResetSuccess, ScreenHosted, "Your password has been reset. You can return to {app}."},
	{HostedVerifySuccess, ScreenHosted, "Your email address has been verified."},
	{HostedVerifyFailed, ScreenHosted, "This link is invalid or has expired."},
	{HostedEmailChangeSuccess, ScreenHosted, "Your email address has been updated."},
	{HostedOAuthSuccess, ScreenHosted, "You're signed in. You can return to {app}."},
	{HostedFooter, ScreenHosted, "Secured by moth"},
	{HostedResetTooShort, ScreenHosted, "That password is too short for this app. Try a longer one."},
	{HostedOAuthCode, ScreenHosted, "One-time code: {code}"},
	{HostedOAuthIncompleteTitle, ScreenHosted, "Sign-in not completed"},
	{HostedOAuthIncomplete, ScreenHosted, "The provider did not complete the sign-in. Return to the app and try again."},
	{HostedOAuthFailedTitle, ScreenHosted, "Sign-in failed"},
	{HostedOAuthErrProviderDisabled, ScreenHosted, "This sign-in method is not available for this app."},
	{HostedOAuthErrInvalidRedirect, ScreenHosted, "The requested redirect is not registered for this app."},
	{HostedOAuthErrInvalidToken, ScreenHosted, "This sign-in link is invalid, expired or was already used. Return to the app and try again."},
	{HostedOAuthErrProviderToken, ScreenHosted, "The provider sign-in could not be verified. Return to the app and try again."},
	{HostedOAuthErrEmailExists, ScreenHosted, "An account with this email already exists. Sign in with it to link this provider."},
	{HostedOAuthErrUserDisabled, ScreenHosted, "This account is disabled."},
	{HostedOAuthErrSignupClosed, ScreenHosted, "Signup is closed for this app."},
	{HostedOAuthErrInvalid, ScreenHosted, "Invalid sign-in request."},

	// email
	{EmailVerifySubject, ScreenEmail, "Verify your email for {app}"},
	{EmailVerifyBody, ScreenEmail, "Confirm this email address to finish setting up your {app} account."},
	{EmailVerifyButton, ScreenEmail, "Verify email"},
	{EmailResetSubject, ScreenEmail, "Reset your {app} password"},
	{EmailResetBody, ScreenEmail, "A password reset was requested for your {app} account."},
	{EmailResetButton, ScreenEmail, "Reset password"},
	{EmailIgnoreNotice, ScreenEmail, "If you didn't request this, you can safely ignore this email."},
	{EmailChangeSubject, ScreenEmail, "Confirm your new email for {app}"},
	{EmailChangeBody, ScreenEmail, "Confirm that you want to use this address for your {app} account."},
	{EmailChangeButton, ScreenEmail, "Confirm new email"},
	{EmailChangedSubject, ScreenEmail, "Your {app} email address was changed"},
	{EmailChangedBody, ScreenEmail, "The email address on your {app} account was changed to {email}."},
	{EmailChangedRevert, ScreenEmail, "If you didn't make this change, you can restore this address within 72 hours using the button below."},
	{EmailChangedButton, ScreenEmail, "Restore this email"},
}

var (
	allKeys      []Key
	englishByKey = map[Key]string{}
	screenByKey  = map[Key]Screen{}
	keysByScreen = map[Screen][]Key{}
	requiredPH   = map[Key][]string{}
)

// screenOrder is the canonical screen ordering for Screens().
var screenOrder = []Screen{
	ScreenSignIn, ScreenSignUp, ScreenPasswordReset,
	ScreenVerifyEmail, ScreenPaywall, ScreenHosted, ScreenEmail,
}

var placeholderRE = regexp.MustCompile(`\{([a-zA-Z][a-zA-Z0-9_]*)\}`)

func init() {
	for _, e := range entries {
		if _, dup := englishByKey[e.Key]; dup {
			panic("i18n: duplicate catalog key " + string(e.Key))
		}
		allKeys = append(allKeys, e.Key)
		englishByKey[e.Key] = e.Default
		screenByKey[e.Key] = e.Screen
		keysByScreen[e.Screen] = append(keysByScreen[e.Screen], e.Key)
		requiredPH[e.Key] = placeholdersIn(e.Default)
	}
	loadBundles() // bundle.go: parse embedded non-English locale files
}

// placeholdersIn returns the sorted, de-duplicated {name} placeholders in s.
func placeholdersIn(s string) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range placeholderRE.FindAllStringSubmatch(s, -1) {
		if !seen[m[1]] {
			seen[m[1]] = true
			out = append(out, m[1])
		}
	}
	sort.Strings(out)
	return out
}

// AllKeys returns every catalog key in canonical (by-screen) order. The
// returned slice is a copy and safe to retain.
func AllKeys() []Key {
	return append([]Key(nil), allKeys...)
}

// Screens returns the surfaces in canonical order.
func Screens() []Screen {
	return append([]Screen(nil), screenOrder...)
}

// ScreenKeys returns the keys of one screen in catalog order (nil for an
// unknown screen). The returned slice is a copy.
func ScreenKeys(s Screen) []Key {
	return append([]Key(nil), keysByScreen[s]...)
}

// IsKey reports whether k is a member of the closed catalog.
func IsKey(k Key) bool {
	_, ok := englishByKey[k]
	return ok
}

// ScreenOf returns the screen a key belongs to, and whether the key is known.
func ScreenOf(k Key) (Screen, bool) {
	s, ok := screenByKey[k]
	return s, ok
}

// English returns the canonical English default for a known key, and whether
// the key is known. English is never empty for a known key.
func English(k Key) (string, bool) {
	v, ok := englishByKey[k]
	return v, ok
}

// RequiredPlaceholders returns the placeholder names a key's value must
// contain (derived from its English default), sorted. The returned slice is a
// copy.
func RequiredPlaceholders(k Key) []string {
	return append([]string(nil), requiredPH[k]...)
}
