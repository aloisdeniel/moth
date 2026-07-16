package oidc

import (
	"context"
	"net/url"
)

// GoogleClient exchanges web-redirect-flow authorization codes at Google's
// token endpoint.
type GoogleClient struct {
	tokenURL     string
	clientID     string
	clientSecret string
	httpc        Doer
}

// NewGoogleClient returns a client for Google's token endpoint at tokenURL
// (defaults to GoogleTokenURL when empty; overridable for tests). httpc
// defaults to a timeout-bounded client when nil.
func NewGoogleClient(tokenURL, clientID, clientSecret string, httpc Doer) *GoogleClient {
	if tokenURL == "" {
		tokenURL = GoogleTokenURL
	}
	if httpc == nil {
		httpc = defaultDoer()
	}
	return &GoogleClient{tokenURL: tokenURL, clientID: clientID, clientSecret: clientSecret, httpc: httpc}
}

// ExchangeCode trades an authorization code for tokens (including the
// id_token the caller then verifies). redirectURI must match the one used in
// the authorization request.
func (c *GoogleClient) ExchangeCode(ctx context.Context, code, redirectURI string) (TokenResponse, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
		"redirect_uri":  {redirectURI},
	}
	return postTokenForm(ctx, c.httpc, c.tokenURL, form)
}
