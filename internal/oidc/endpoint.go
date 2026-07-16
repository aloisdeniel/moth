package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// TokenResponse is the successful body of an OAuth token-endpoint call.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
}

// TokenError is a non-2xx token-endpoint response, carrying the OAuth error
// code (e.g. "invalid_grant") when the provider sent one.
type TokenError struct {
	StatusCode  int
	Code        string
	Description string
}

func (e *TokenError) Error() string {
	msg := fmt.Sprintf("token endpoint: status %d", e.StatusCode)
	if e.Code != "" {
		msg += ": " + e.Code
	}
	if e.Description != "" {
		msg += ": " + e.Description
	}
	return msg
}

// postTokenForm posts a form and decodes the TokenResponse.
func postTokenForm(ctx context.Context, httpc Doer, endpoint string, form url.Values) (TokenResponse, error) {
	body, err := postForm(ctx, httpc, endpoint, form)
	if err != nil {
		return TokenResponse{}, err
	}
	var tokens TokenResponse
	if err := json.Unmarshal(body, &tokens); err != nil {
		return TokenResponse{}, fmt.Errorf("token endpoint: decode response: %w", err)
	}
	return tokens, nil
}

// postForm posts application/x-www-form-urlencoded values and returns the
// raw 2xx body, or a *TokenError for OAuth-style error responses.
func postForm(ctx context.Context, httpc Doer, endpoint string, form url.Values) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("token endpoint: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := httpc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token endpoint: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("token endpoint: read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		tokErr := &TokenError{StatusCode: resp.StatusCode}
		var oauthErr struct {
			Code        string `json:"error"`
			Description string `json:"error_description"`
		}
		if json.Unmarshal(body, &oauthErr) == nil {
			tokErr.Code = oauthErr.Code
			tokErr.Description = oauthErr.Description
		}
		return nil, tokErr
	}
	return body, nil
}
