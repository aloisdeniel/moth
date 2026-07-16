package authrpc

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"connectrpc.com/connect"

	authv1 "github.com/aloisdeniel/moth/gen/moth/auth/v1"
	"github.com/aloisdeniel/moth/internal/events"
	"github.com/aloisdeniel/moth/internal/oidc"
	"github.com/aloisdeniel/moth/internal/store"
	"github.com/aloisdeniel/moth/internal/token"
)

// Lifetimes of the web-redirect flow's single-use artifacts.
const (
	oauthStateTTL = 10 * time.Minute
	oauthCodeTTL  = 5 * time.Minute
)

func (h *Handler) SignInWithOAuth(ctx context.Context, req *connect.Request[authv1.SignInWithOAuthRequest]) (*connect.Response[authv1.SignInWithOAuthResponse], error) {
	project, err := h.project(ctx)
	if err != nil {
		return nil, err
	}
	provider, err := providerName(req.Msg.Provider)
	if err != nil {
		return nil, err
	}
	audiences, err := nativeAudiences(project.Settings, provider)
	if err != nil {
		h.emit(events.LoginFailed(ctx, project.ID, provider, events.ReasonProviderDisabled))
		return nil, err
	}
	if req.Msg.IdToken == "" {
		return nil, newError(connect.CodeInvalidArgument, ReasonInvalidProviderToken,
			"id_token is required")
	}
	// The nonce binds the ID token to this sign-in attempt; without it a
	// stolen token could be replayed, so the native flow always requires
	// one (the web-redirect flow binds attempts through state instead).
	if req.Msg.Nonce == "" {
		return nil, newError(connect.CodeInvalidArgument, ReasonInvalidProviderToken,
			"nonce is required")
	}
	ident, err := h.verifier(provider).Verify(ctx, req.Msg.IdToken, audiences, req.Msg.Nonce)
	if err != nil {
		h.log.InfoContext(ctx, "provider token rejected",
			"provider", provider, "error", err.Error())
		h.emit(events.LoginFailed(ctx, project.ID, provider, events.ReasonInvalidCredentials))
		return nil, errInvalidProviderToken()
	}
	// A native ID token is single-use: its hash is recorded until the token
	// expires, so a captured token (whose payload — including a NonceRaw
	// nonce — is readable by anyone holding it) cannot be replayed to mint
	// further sessions. The insert is atomic, so a replay racing the first
	// use fails too.
	err = h.store.CreateOAuthToken(ctx, store.OAuthToken{
		ID:        NewID(),
		ProjectID: project.ID,
		Purpose:   store.OAuthTokenPurposeIDToken,
		TokenHash: hashToken(req.Msg.IdToken),
		Provider:  provider,
		// One extra minute covers the verifier's clock-skew leeway.
		ExpiresAt: ident.ExpiresAt.Add(time.Minute),
		CreatedAt: h.now(),
	})
	if errors.Is(err, store.ErrConflict) {
		h.log.WarnContext(ctx, "replayed provider token rejected", "provider", provider)
		h.emit(events.LoginFailed(ctx, project.ID, provider, events.ReasonInvalidCredentials))
		return nil, errInvalidProviderToken()
	}
	if err != nil {
		return nil, errInternal(err)
	}

	user, identity, created, err := h.resolveOAuthUser(ctx, project, provider, ident,
		req.Msg.GivenName, req.Msg.FamilyName)
	if err != nil {
		return nil, err
	}

	// Apple only: trade the native authorization code for the refresh
	// token that account deletion later revokes (App Store requirement).
	// Best effort — the ID token already authenticated the user.
	if provider == store.IdentityProviderApple && req.Msg.AuthorizationCode != "" {
		clientID := tokenAudience(req.Msg.IdToken, audiences)
		h.exchangeAppleCode(ctx, project, identity, clientID, req.Msg.AuthorizationCode, "")
	}

	tokens, err := h.issueSession(ctx, project, user, req.Msg.DeviceInfo)
	if err != nil {
		return nil, errInternal(err)
	}
	if err := h.store.SetUserLastLogin(ctx, project.ID, user.ID, h.now()); err != nil {
		h.log.ErrorContext(ctx, "set last login", "error", err.Error())
	}
	if !created {
		h.emit(events.Login(ctx, project.ID, user.ID, provider))
	}
	return connect.NewResponse(&authv1.SignInWithOAuthResponse{
		User:   userProto(user),
		Tokens: tokens,
	}), nil
}

func (h *Handler) ExchangeOAuthCode(ctx context.Context, req *connect.Request[authv1.ExchangeOAuthCodeRequest]) (*connect.Response[authv1.ExchangeOAuthCodeResponse], error) {
	project, err := h.project(ctx)
	if err != nil {
		return nil, err
	}
	invalid := newError(connect.CodeUnauthenticated, ReasonInvalidOAuthCode,
		"invalid or expired code")
	if req.Msg.Code == "" {
		return nil, invalid
	}
	// Single-use: the claim is atomic, so a replayed code fails even when
	// racing the first exchange.
	ot, err := h.store.ConsumeOAuthToken(ctx, project.ID, store.OAuthTokenPurposeCode,
		hashToken(req.Msg.Code), h.now())
	if errors.Is(err, store.ErrNotFound) {
		// The provider of an unknown code is itself unknown.
		h.emit(events.LoginFailed(ctx, project.ID, "", events.ReasonInvalidCredentials))
		return nil, invalid
	}
	if err != nil {
		return nil, errInternal(err)
	}
	// An admin disable takes effect immediately, even for codes minted
	// while the provider was still enabled.
	if !providerEnabled(project.Settings, ot.Provider) {
		h.emit(events.LoginFailed(ctx, project.ID, ot.Provider, events.ReasonProviderDisabled))
		return nil, errProviderDisabled(ot.Provider)
	}
	user, err := h.store.GetUser(ctx, project.ID, ot.UserID)
	if errors.Is(err, store.ErrNotFound) {
		// The user vanished between callback and exchange.
		h.emit(events.LoginFailed(ctx, project.ID, ot.Provider, events.ReasonInvalidCredentials))
		return nil, invalid
	}
	if err != nil {
		return nil, errInternal(err)
	}
	if user.Disabled() {
		h.emit(events.LoginFailed(ctx, project.ID, ot.Provider, events.ReasonDisabled))
		return nil, errUserDisabled()
	}
	tokens, err := h.issueSession(ctx, project, user, req.Msg.DeviceInfo)
	if err != nil {
		return nil, errInternal(err)
	}
	if err := h.store.SetUserLastLogin(ctx, project.ID, user.ID, h.now()); err != nil {
		h.log.ErrorContext(ctx, "set last login", "error", err.Error())
	}
	// Same event semantics as the native flow: a code minted for a user the
	// callback just created already produced user.signup, not user.login.
	var payload oauthCodePayload
	if ot.Payload != "" {
		if err := json.Unmarshal([]byte(ot.Payload), &payload); err != nil {
			h.log.ErrorContext(ctx, "decode oauth code payload", "error", err.Error())
		}
	}
	if !payload.Signup {
		h.emit(events.Login(ctx, project.ID, user.ID, ot.Provider))
	}
	return connect.NewResponse(&authv1.ExchangeOAuthCodeResponse{
		User:   userProto(user),
		Tokens: tokens,
	}), nil
}

func (h *Handler) UnlinkIdentity(ctx context.Context, req *connect.Request[authv1.UnlinkIdentityRequest]) (*connect.Response[authv1.UnlinkIdentityResponse], error) {
	project, user, err := h.requireUser(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	provider, err := providerName(req.Msg.Provider)
	if err != nil {
		return nil, err
	}
	identities, err := h.store.ListUserIdentities(ctx, project.ID, user.ID)
	if err != nil {
		return nil, errInternal(err)
	}
	var removing []store.Identity
	remaining := 0
	if user.PasswordHash != "" {
		remaining++ // the password counts as a login method
	}
	for _, id := range identities {
		switch {
		case id.Provider == provider:
			removing = append(removing, id)
		case id.Provider != store.IdentityProviderPassword:
			remaining++
		}
	}
	if len(removing) == 0 {
		return nil, connect.NewError(connect.CodeNotFound,
			fmt.Errorf("no linked %s identity", provider))
	}
	if remaining == 0 {
		return nil, newError(connect.CodeFailedPrecondition, ReasonLastLoginMethod,
			"unlinking the last sign-in method would lock the account; set a password or link another provider first")
	}
	// Revoke stored Apple refresh tokens at Apple before dropping them
	// (best effort; the unlink proceeds regardless).
	for _, id := range removing {
		h.revokeAppleRefreshToken(ctx, project, id)
	}
	if err := h.store.DeleteUserIdentities(ctx, project.ID, user.ID, provider); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			// Lost a race with a concurrent unlink of the same provider;
			// same outcome as the sequential already-unlinked path above.
			return nil, connect.NewError(connect.CodeNotFound,
				fmt.Errorf("no linked %s identity", provider))
		}
		return nil, errInternal(err)
	}
	return connect.NewResponse(&authv1.UnlinkIdentityResponse{}), nil
}

// OAuthStart begins the web-redirect fallback flow: it validates that the
// provider is configured for web sign-in and that redirectURI targets a
// registered scheme (open-redirect protection), persists a hashed
// single-use state and returns the provider consent URL to send the
// browser to. The returned state embeds the project slug so the
// provider-console callback URL can stay project-agnostic.
func (h *Handler) OAuthStart(ctx context.Context, project store.Project, provider, redirectURI string) (string, error) {
	if provider != store.IdentityProviderGoogle && provider != store.IdentityProviderApple {
		return "", connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("unknown provider %q", provider))
	}
	if err := h.checkWebProvider(ctx, project, provider); err != nil {
		return "", err
	}
	if redirectURI != "" {
		if err := checkRedirectURI(redirectURI, project.Settings.RedirectSchemes); err != nil {
			return "", err
		}
	}

	// The per-attempt nonce rides in the state payload; the callback
	// requires the exchanged ID token to carry it.
	nonce := token.Random(16)
	payload, err := json.Marshal(oauthStatePayload{Nonce: nonce})
	if err != nil {
		return "", errInternal(err)
	}
	state := project.Slug + "." + token.Random(32)
	now := h.now()
	err = h.store.CreateOAuthToken(ctx, store.OAuthToken{
		ID:          NewID(),
		ProjectID:   project.ID,
		Purpose:     store.OAuthTokenPurposeState,
		TokenHash:   hashToken(state),
		Provider:    provider,
		RedirectURI: redirectURI,
		Payload:     string(payload),
		ExpiresAt:   now.Add(oauthStateTTL),
		CreatedAt:   now,
	})
	if err != nil {
		return "", errInternal(err)
	}

	q := url.Values{
		"response_type": {"code"},
		"redirect_uri":  {h.oauthCallbackURL(provider)},
		"state":         {state},
	}
	switch provider {
	case store.IdentityProviderGoogle:
		q.Set("client_id", project.Settings.Google.WebClientID)
		q.Set("scope", "openid email profile")
		q.Set("nonce", nonce)
		return h.googleAuthURL + "?" + q.Encode(), nil
	default: // apple
		q.Set("client_id", project.Settings.Apple.ServicesID)
		q.Set("scope", "name email")
		q.Set("response_mode", "form_post")
		// Apple echoes the nonce parameter into the ID token verbatim; it
		// is sent SHA-256-hashed per Apple's scheme so the verifier's
		// NonceSHA256Hex comparison against the raw nonce matches.
		sum := sha256.Sum256([]byte(nonce))
		q.Set("nonce", hex.EncodeToString(sum[:]))
		return h.appleAuthURL + "?" + q.Encode(), nil
	}
}

// OAuthCallback completes the web-redirect flow after the provider consent:
// it claims the single-use state (tampered, expired and replayed states all
// fail identically), exchanges the authorization code server-side, verifies
// the resulting ID token, resolves the user through the same matrix as
// SignInWithOAuth, and mints the one-time app code that ExchangeOAuthCode
// trades for tokens. It returns the app code and the redirect URI
// registered at start ("" means show the hosted success page). userJSON is
// Apple's optional first-launch `user` form field.
func (h *Handler) OAuthCallback(ctx context.Context, project store.Project, provider, state, code, userJSON string) (string, string, error) {
	invalidState := newError(connect.CodeInvalidArgument, ReasonInvalidToken,
		"invalid, expired or already-used state")
	if state == "" || code == "" {
		return "", "", invalidState
	}
	ot, err := h.store.ConsumeOAuthToken(ctx, project.ID, store.OAuthTokenPurposeState,
		hashToken(state), h.now())
	if errors.Is(err, store.ErrNotFound) {
		return "", "", invalidState
	}
	if err != nil {
		return "", "", errInternal(err)
	}
	if ot.Provider != provider {
		return "", "", invalidState
	}
	var payload oauthStatePayload
	if err := json.Unmarshal([]byte(ot.Payload), &payload); err != nil {
		return "", "", errInternal(fmt.Errorf("decode oauth state payload: %w", err))
	}
	// Every state row minted by OAuthStart carries a nonce; an empty one
	// would silently skip the ID-token nonce binding in Verify below.
	if payload.Nonce == "" {
		return "", "", errInternal(fmt.Errorf("oauth state payload misses the nonce"))
	}

	var (
		tokens   oidc.TokenResponse
		audience string
	)
	switch provider {
	case store.IdentityProviderGoogle:
		secret, err := h.googleWebClientSecret(ctx, project)
		if err != nil {
			return "", "", err
		}
		audience = project.Settings.Google.WebClientID
		client := oidc.NewGoogleClient(h.googleTokenURL, audience, secret, h.httpc)
		tokens, err = client.ExchangeCode(ctx, code, h.oauthCallbackURL(provider))
		if err != nil {
			h.log.InfoContext(ctx, "google code exchange failed", "error", err.Error())
			return "", "", errInvalidProviderToken()
		}
	default: // apple
		audience = project.Settings.Apple.ServicesID
		client, err := h.appleClient(ctx, project, audience)
		if err != nil {
			return "", "", errProviderDisabled(provider)
		}
		tokens, err = client.ExchangeCode(ctx, code, h.oauthCallbackURL(provider))
		if err != nil {
			h.log.InfoContext(ctx, "apple code exchange failed", "error", err.Error())
			return "", "", errInvalidProviderToken()
		}
	}

	ident, err := h.verifier(provider).Verify(ctx, tokens.IDToken, []string{audience}, payload.Nonce)
	if err != nil {
		h.log.InfoContext(ctx, "provider token rejected",
			"provider", provider, "error", err.Error())
		return "", "", errInvalidProviderToken()
	}
	givenName, familyName := appleUserName(userJSON)
	user, identity, created, err := h.resolveOAuthUser(ctx, project, provider, ident, givenName, familyName)
	if err != nil {
		return "", "", err
	}
	if provider == store.IdentityProviderApple && tokens.RefreshToken != "" {
		h.storeAppleRefreshToken(ctx, project, identity, audience, tokens.RefreshToken)
	}

	// The exchange emits the user.login analytics event; a code for a
	// just-created user carries a signup marker so it does not (the native
	// flow's semantics: a first sign-in produces only user.signup).
	var codePayload string
	if created {
		raw, err := json.Marshal(oauthCodePayload{Signup: true})
		if err != nil {
			return "", "", errInternal(fmt.Errorf("encode oauth code payload: %w", err))
		}
		codePayload = string(raw)
	}

	appCode := token.Random(32)
	now := h.now()
	err = h.store.CreateOAuthToken(ctx, store.OAuthToken{
		ID:          NewID(),
		ProjectID:   project.ID,
		Purpose:     store.OAuthTokenPurposeCode,
		TokenHash:   hashToken(appCode),
		Provider:    provider,
		UserID:      user.ID,
		RedirectURI: ot.RedirectURI,
		Payload:     codePayload,
		ExpiresAt:   now.Add(oauthCodeTTL),
		CreatedAt:   now,
	})
	if err != nil {
		return "", "", errInternal(err)
	}
	return appCode, ot.RedirectURI, nil
}

// oauthStatePayload rides JSON-encoded in the state row between the two
// legs of the web-redirect flow.
type oauthStatePayload struct {
	Nonce string `json:"nonce"`
}

// oauthCodePayload rides JSON-encoded in the one-time code row between the
// callback and ExchangeOAuthCode.
type oauthCodePayload struct {
	// Signup marks a code minted for a user the callback just created.
	Signup bool `json:"signup,omitempty"`
}

// resolveOAuthUser turns a verified provider identity into a moth user,
// implementing the account-resolution matrix (security-sensitive; every
// input comes from the *verified* token, never from client-asserted request
// fields — the Apple first-launch name is the one documented exception and
// is only ever used as the initial display name of a brand-new user):
//
//	(a) a (provider, subject) identity exists → sign that user in;
//	(b) else the token asserts a VERIFIED email that matches an existing
//	    user whose email is itself VERIFIED and the project auto-links
//	    verified emails (default on) → link a new identity to that user;
//	(c) else → create user + identity, email_verified taken from the
//	    verified token claim.
//
// Case (b) requires the existing account to be verified: otherwise anyone
// could pre-register the victim's email with a password and silently
// capture the victim's provider identity — and keep password access —
// when the victim later signs in socially (account pre-hijacking).
//
// An unverified or auto-link-refused email that collides with an existing
// account never links; it fails with EMAIL_ALREADY_EXISTS (emails are
// unique per project, so a separate account cannot be created either).
// Returns the user, its matched-or-new identity, and whether a user was
// created (the caller emits the signed-in event for the other cases).
func (h *Handler) resolveOAuthUser(ctx context.Context, project store.Project, provider string, ident oidc.Identity, givenName, familyName string) (store.User, store.Identity, bool, error) {
	// (a) Existing identity → login.
	identity, err := h.store.GetIdentity(ctx, project.ID, provider, ident.Subject)
	if err == nil {
		user, err := h.store.GetUser(ctx, project.ID, identity.UserID)
		if err != nil {
			return store.User{}, store.Identity{}, false, errInternal(err)
		}
		if user.Disabled() {
			h.emit(events.LoginFailed(ctx, project.ID, provider, events.ReasonDisabled))
			return store.User{}, store.Identity{}, false, errUserDisabled()
		}
		if email := normalizeEmail(ident.Email); email != "" && email != identity.ProviderEmail {
			if err := h.store.SetIdentityProviderEmail(ctx, project.ID, identity.ID, email); err != nil {
				h.log.ErrorContext(ctx, "update identity provider email", "error", err.Error())
			}
		}
		return user, identity, false, nil
	}
	if !errors.Is(err, store.ErrNotFound) {
		return store.User{}, store.Identity{}, false, errInternal(err)
	}

	email := normalizeEmail(ident.Email)
	if email == "" {
		// Without an email there is nothing to link or create an account
		// around (Apple always sends one on the first authorization).
		return store.User{}, store.Identity{}, false, newError(connect.CodeFailedPrecondition,
			ReasonInvalidProviderToken, "the provider token carries no email address")
	}

	// (b) Verified email matching an existing, itself-verified user → link.
	existing, err := h.store.GetUserByEmail(ctx, project.ID, email)
	if err == nil {
		if !ident.EmailVerified || !existing.Verified() || !project.Settings.AutoLinkEnabled() {
			return store.User{}, store.Identity{}, false, newError(connect.CodeAlreadyExists,
				ReasonEmailAlreadyExists,
				"an account with this email already exists; sign in with it to link this provider")
		}
		if existing.Disabled() {
			h.emit(events.LoginFailed(ctx, project.ID, provider, events.ReasonDisabled))
			return store.User{}, store.Identity{}, false, errUserDisabled()
		}
		identity := store.Identity{
			ID:              NewID(),
			ProjectID:       project.ID,
			UserID:          existing.ID,
			Provider:        provider,
			ProviderSubject: ident.Subject,
			ProviderEmail:   email,
			CreatedAt:       h.now(),
		}
		if err := h.store.CreateIdentity(ctx, identity); err != nil {
			if errors.Is(err, store.ErrConflict) {
				// Lost a race with a concurrent sign-in linking the same
				// (provider, subject); reuse the winner's identity — it
				// resolves to the same user by construction.
				identity, err = h.store.GetIdentity(ctx, project.ID, provider, ident.Subject)
				if err != nil {
					return store.User{}, store.Identity{}, false, errInternal(err)
				}
				return existing, identity, false, nil
			}
			return store.User{}, store.Identity{}, false, errInternal(err)
		}
		h.emit(events.IdentityLinked(ctx, project.ID, existing.ID, provider))
		return existing, identity, false, nil
	}
	if !errors.Is(err, store.ErrNotFound) {
		return store.User{}, store.Identity{}, false, errInternal(err)
	}

	// (c) New user + identity. Social sign-up respects the same
	// public-signup gate as SignUp.
	if !project.Settings.AllowPublicSignup {
		return store.User{}, store.Identity{}, false, newError(connect.CodePermissionDenied,
			ReasonSignupClosed, "public signup is closed for this project")
	}
	now := h.now()
	user := store.User{
		ID:           NewID(),
		ProjectID:    project.ID,
		Email:        email,
		DisplayName:  oauthDisplayName(ident, givenName, familyName),
		CustomClaims: "{}",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if ident.EmailVerified {
		user.EmailVerifiedAt = &now
	}
	identity = store.Identity{
		ID:              NewID(),
		ProjectID:       project.ID,
		UserID:          user.ID,
		Provider:        provider,
		ProviderSubject: ident.Subject,
		ProviderEmail:   email,
		CreatedAt:       now,
	}
	if err := h.store.CreateUser(ctx, user, identity); err != nil {
		if errors.Is(err, store.ErrConflict) {
			// Lost a race with a concurrent signup on the same email.
			return store.User{}, store.Identity{}, false, newError(connect.CodeAlreadyExists,
				ReasonEmailAlreadyExists, "an account with this email already exists")
		}
		return store.User{}, store.Identity{}, false, errInternal(err)
	}
	h.emit(events.Signup(ctx, project.ID, user.ID, provider))
	return user, identity, true, nil
}

// oauthDisplayName picks the initial display name of a created user: the
// verified token's name when present, else the client-asserted Apple
// first-launch name.
func oauthDisplayName(ident oidc.Identity, givenName, familyName string) string {
	if ident.Name != "" {
		return ident.Name
	}
	if name := strings.TrimSpace(ident.GivenName + " " + ident.FamilyName); name != "" {
		return name
	}
	return strings.TrimSpace(strings.TrimSpace(givenName) + " " + strings.TrimSpace(familyName))
}

// providerName maps the proto enum to the identity-provider constant.
func providerName(p authv1.OAuthProvider) (string, error) {
	switch p {
	case authv1.OAuthProvider_OAUTH_PROVIDER_GOOGLE:
		return store.IdentityProviderGoogle, nil
	case authv1.OAuthProvider_OAUTH_PROVIDER_APPLE:
		return store.IdentityProviderApple, nil
	default:
		return "", connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("unknown provider %v", p))
	}
}

// verifier returns the ID-token verifier of a provider.
func (h *Handler) verifier(provider string) *oidc.Verifier {
	if provider == store.IdentityProviderApple {
		return h.appleVerifier
	}
	return h.googleVerifier
}

// nativeAudiences returns the allowed aud values of a native ID token, or
// PROVIDER_DISABLED when the provider is off or has nothing configured.
func nativeAudiences(s store.ProjectSettings, provider string) ([]string, error) {
	var audiences []string
	switch provider {
	case store.IdentityProviderGoogle:
		if !s.Google.Enabled {
			return nil, errProviderDisabled(provider)
		}
		for _, id := range []string{s.Google.WebClientID, s.Google.IOSClientID, s.Google.AndroidClientID} {
			if id != "" {
				audiences = append(audiences, id)
			}
		}
	case store.IdentityProviderApple:
		if !s.Apple.Enabled {
			return nil, errProviderDisabled(provider)
		}
		audiences = append(audiences, s.Apple.BundleIDs...)
		if s.Apple.ServicesID != "" {
			audiences = append(audiences, s.Apple.ServicesID)
		}
	}
	if len(audiences) == 0 {
		return nil, errProviderDisabled(provider)
	}
	return audiences, nil
}

// checkWebProvider verifies a provider is enabled and fully configured for
// the web-redirect flow (which, unlike the native flow, needs server-held
// credentials for the code exchange).
func (h *Handler) checkWebProvider(ctx context.Context, project store.Project, provider string) error {
	s := project.Settings
	switch provider {
	case store.IdentityProviderGoogle:
		if !s.Google.Enabled || s.Google.WebClientID == "" {
			return errProviderDisabled(provider)
		}
		if _, err := h.googleWebClientSecret(ctx, project); err != nil {
			return err
		}
	case store.IdentityProviderApple:
		if !s.Apple.Enabled || s.Apple.ServicesID == "" {
			return errProviderDisabled(provider)
		}
		if _, err := h.appleClient(ctx, project, s.Apple.ServicesID); err != nil {
			return errProviderDisabled(provider)
		}
	}
	return nil
}

// checkRedirectURI enforces that the callback only ever redirects to a
// scheme the project registered (open-redirect protection).
func checkRedirectURI(redirectURI string, schemes []string) error {
	invalid := newError(connect.CodeInvalidArgument, ReasonInvalidRedirect,
		"redirect does not target a scheme registered for this project")
	u, err := url.Parse(redirectURI)
	if err != nil || u.Scheme == "" {
		return invalid
	}
	// Scheme-only matching cannot constrain the host, so http(s) redirects
	// would be open redirects leaking the one-time code to any site; they
	// are refused even if a settings row somehow registered them (the
	// admin API rejects them at write time too).
	if strings.EqualFold(u.Scheme, "http") || strings.EqualFold(u.Scheme, "https") {
		return invalid
	}
	for _, s := range schemes {
		if strings.EqualFold(u.Scheme, s) {
			return nil
		}
	}
	return invalid
}

// providerEnabled reports whether a provider's toggle is on for the
// project.
func providerEnabled(s store.ProjectSettings, provider string) bool {
	switch provider {
	case store.IdentityProviderGoogle:
		return s.Google.Enabled
	case store.IdentityProviderApple:
		return s.Apple.Enabled
	}
	return false
}

// oauthCallbackURL is the redirect URI registered in the provider console;
// it is project-agnostic (the state carries the project).
func (h *Handler) oauthCallbackURL(provider string) string {
	return h.baseURL + "/oauth/" + provider + "/callback"
}

// googleWebClientSecret loads and decrypts the project's Google web client
// secret.
func (h *Handler) googleWebClientSecret(ctx context.Context, project store.Project) (string, error) {
	enc, err := h.store.GetProviderSecret(ctx, project.ID, store.ProviderSecretGoogleWebClientSecret)
	if errors.Is(err, store.ErrNotFound) {
		return "", errProviderDisabled(store.IdentityProviderGoogle)
	}
	if err != nil {
		return "", errInternal(err)
	}
	secret, err := h.master.Decrypt(enc)
	if err != nil {
		return "", errInternal(fmt.Errorf("decrypt google web client secret: %w", err))
	}
	return string(secret), nil
}

// appleClient builds an Apple token-endpoint client authenticating as
// clientID (the ID-token aud: a bundle ID natively, the Services ID on the
// web). It errors when the Apple key configuration is incomplete.
//
// The client-secret generator is cached per (project, clientID) so its
// signed-secret cache actually amortizes the .p8 decrypt/parse and the
// ES256 signature across requests; a fingerprint of the key configuration
// invalidates the entry when the admin rotates the .p8 or its IDs, without
// decrypting on the cached path.
func (h *Handler) appleClient(ctx context.Context, project store.Project, clientID string) (*oidc.AppleClient, error) {
	s := project.Settings.Apple
	if s.TeamID == "" || s.KeyID == "" || clientID == "" {
		return nil, errors.New("apple key configuration is incomplete")
	}
	enc, err := h.store.GetProviderSecret(ctx, project.ID, store.ProviderSecretApplePrivateKey)
	if err != nil {
		return nil, fmt.Errorf("load apple private key: %w", err)
	}
	sum := sha256.Sum256(enc)
	fingerprint := s.TeamID + "|" + s.KeyID + "|" + hex.EncodeToString(sum[:])
	cacheKey := project.ID + "|" + clientID

	h.appleSecretsMu.Lock()
	entry, ok := h.appleSecrets[cacheKey]
	h.appleSecretsMu.Unlock()
	if !ok || entry.fingerprint != fingerprint {
		raw, err := h.master.Decrypt(enc)
		if err != nil {
			return nil, fmt.Errorf("decrypt apple private key: %w", err)
		}
		key, err := oidc.ParseP8(raw)
		if err != nil {
			return nil, err
		}
		entry = appleSecretsEntry{
			fingerprint: fingerprint,
			secrets: oidc.NewAppleSecrets(oidc.AppleSecretConfig{
				TeamID:   s.TeamID,
				KeyID:    s.KeyID,
				ClientID: clientID,
				Key:      key,
			}, h.now),
		}
		h.appleSecretsMu.Lock()
		h.appleSecrets[cacheKey] = entry
		h.appleSecretsMu.Unlock()
	}
	return oidc.NewAppleClient(h.appleBaseURL, clientID, entry.secrets, h.httpc), nil
}

// appleRefreshBlob is the plaintext stored (AES-GCM-encrypted under the
// master key) in identities.apple_refresh_token_enc. The client ID rides
// along because Apple's revocation endpoint requires the same client the
// token was issued to.
type appleRefreshBlob struct {
	ClientID     string `json:"client_id"`
	RefreshToken string `json:"refresh_token"`
}

// exchangeAppleCode trades a native authorization code for a refresh token
// and stores it on the identity. Best effort: failures are logged, never
// surfaced — the verified ID token already authenticated the sign-in.
func (h *Handler) exchangeAppleCode(ctx context.Context, project store.Project, identity store.Identity, clientID, code, redirectURI string) {
	client, err := h.appleClient(ctx, project, clientID)
	if err != nil {
		h.log.InfoContext(ctx, "skipping apple code exchange", "error", err.Error())
		return
	}
	tokens, err := client.ExchangeCode(ctx, code, redirectURI)
	if err != nil {
		h.log.WarnContext(ctx, "apple code exchange failed", "error", err.Error())
		return
	}
	if tokens.RefreshToken == "" {
		return
	}
	h.storeAppleRefreshToken(ctx, project, identity, clientID, tokens.RefreshToken)
}

// storeAppleRefreshToken encrypts and persists an Apple refresh token on
// the identity (best effort, logged).
func (h *Handler) storeAppleRefreshToken(ctx context.Context, project store.Project, identity store.Identity, clientID, refreshToken string) {
	blob, err := json.Marshal(appleRefreshBlob{ClientID: clientID, RefreshToken: refreshToken})
	if err != nil {
		h.log.ErrorContext(ctx, "encode apple refresh token", "error", err.Error())
		return
	}
	enc, err := h.master.Encrypt(blob)
	if err != nil {
		h.log.ErrorContext(ctx, "encrypt apple refresh token", "error", err.Error())
		return
	}
	if err := h.store.SetIdentityAppleRefreshToken(ctx, project.ID, identity.ID, enc); err != nil {
		h.log.ErrorContext(ctx, "store apple refresh token", "error", err.Error())
	}
}

// revokeAppleRefreshToken revokes the identity's stored Apple refresh token
// at Apple (App Store review requirement on account deletion and unlink).
// Best effort: failures are logged, never surfaced.
func (h *Handler) revokeAppleRefreshToken(ctx context.Context, project store.Project, identity store.Identity) {
	if identity.Provider != store.IdentityProviderApple || len(identity.AppleRefreshTokenEnc) == 0 {
		return
	}
	raw, err := h.master.Decrypt(identity.AppleRefreshTokenEnc)
	if err != nil {
		h.log.ErrorContext(ctx, "decrypt apple refresh token", "error", err.Error())
		return
	}
	var blob appleRefreshBlob
	if err := json.Unmarshal(raw, &blob); err != nil {
		h.log.ErrorContext(ctx, "decode apple refresh token", "error", err.Error())
		return
	}
	client, err := h.appleClient(ctx, project, blob.ClientID)
	if err != nil {
		h.log.WarnContext(ctx, "apple token revocation skipped", "error", err.Error())
		return
	}
	if err := client.Revoke(ctx, blob.RefreshToken); err != nil {
		h.log.WarnContext(ctx, "apple token revocation failed", "error", err.Error())
	}
}

// tokenAudience returns the first aud of the (already verified) ID token
// that is in allowed; used to pick the client_id of Apple's code exchange.
func tokenAudience(idToken string, allowed []string) string {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return ""
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims struct {
		Aud json.RawMessage `json:"aud"`
	}
	if err := json.Unmarshal(body, &claims); err != nil {
		return ""
	}
	var auds []string
	var one string
	if err := json.Unmarshal(claims.Aud, &one); err == nil {
		auds = []string{one}
	} else if err := json.Unmarshal(claims.Aud, &auds); err != nil {
		return ""
	}
	for _, aud := range auds {
		for _, ok := range allowed {
			if aud == ok {
				return aud
			}
		}
	}
	return ""
}

// appleUserName extracts the first-launch name from Apple's form_post
// `user` field, e.g. {"name":{"firstName":"Jane","lastName":"Doe"}}.
func appleUserName(userJSON string) (string, string) {
	if userJSON == "" {
		return "", ""
	}
	var u struct {
		Name struct {
			FirstName string `json:"firstName"`
			LastName  string `json:"lastName"`
		} `json:"name"`
	}
	if err := json.Unmarshal([]byte(userJSON), &u); err != nil {
		return "", ""
	}
	return u.Name.FirstName, u.Name.LastName
}
