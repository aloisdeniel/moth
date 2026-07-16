package setup

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/aloisdeniel/moth/internal/oidc"
)

// ASCBaseURL is the official App Store Connect API host.
const ASCBaseURL = "https://api.appstoreconnect.apple.com"

// ascTokenLifetime keeps the request JWT well under Apple's 20-minute cap.
const ascTokenLifetime = 10 * time.Minute

// b64url is the JWS segment encoding.
var b64url = base64.RawURLEncoding

// ASC is a minimal App Store Connect API client covering exactly what
// `moth setup apple` needs: bundle IDs, their capabilities, and Sign in
// with Apple key creation. Requests are authenticated with a short-lived
// ES256 JWT minted from the operator's ASC API key — the key is used
// in-process and never persisted (plan: "store nothing platform-side").
type ASC struct {
	// BaseURL defaults to ASCBaseURL; tests point it at a double.
	BaseURL  string
	IssuerID string
	KeyID    string
	Key      *ecdsa.PrivateKey
	HTTPC    oidc.Doer
	// Now is the clock (test override); defaults to time.Now.
	Now func() time.Time
}

// ascError is a non-2xx ASC response.
type ascError struct {
	Status int
	Title  string
	Detail string
}

func (e *ascError) Error() string {
	msg := fmt.Sprintf("app store connect: status %d", e.Status)
	if e.Title != "" {
		msg += ": " + e.Title
	}
	if e.Detail != "" {
		msg += ": " + e.Detail
	}
	return msg
}

// isASCNotFound reports whether err is an ASC 404 (missing resource — or
// an endpoint Apple does not expose, which degrades to the guided flow).
func isASCNotFound(err error) bool {
	var ae *ascError
	return errors.As(err, &ae) && ae.Status == http.StatusNotFound
}

// Token mints the ES256 request JWT (header: alg/kid/typ; claims:
// iss/iat/exp/aud per Apple's "Generating Tokens for API Requests").
func (c *ASC) Token() (string, error) {
	now := time.Now
	if c.Now != nil {
		now = c.Now
	}
	head, err := json.Marshal(map[string]string{"alg": "ES256", "kid": c.KeyID, "typ": "JWT"})
	if err != nil {
		return "", err
	}
	claims, err := json.Marshal(map[string]any{
		"iss": c.IssuerID,
		"iat": now().Unix(),
		"exp": now().Add(ascTokenLifetime).Unix(),
		"aud": "appstoreconnect-v1",
	})
	if err != nil {
		return "", err
	}
	signingInput := b64url.EncodeToString(head) + "." + b64url.EncodeToString(claims)
	digest := sha256.Sum256([]byte(signingInput))
	r, s, err := ecdsa.Sign(rand.Reader, c.Key, digest[:])
	if err != nil {
		return "", fmt.Errorf("sign ASC token: %w", err)
	}
	sig := make([]byte, 64)
	r.FillBytes(sig[:32])
	s.FillBytes(sig[32:])
	return signingInput + "." + b64url.EncodeToString(sig), nil
}

func (c *ASC) do(ctx context.Context, method, path string, body, out any) error {
	base := c.BaseURL
	if base == "" {
		base = ASCBaseURL
	}
	httpc := c.HTTPC
	if httpc == nil {
		httpc = &http.Client{Timeout: 30 * time.Second}
	}
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, base+path, reader)
	if err != nil {
		return err
	}
	tok, err := c.Token()
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := httpc.Do(req)
	if err != nil {
		return fmt.Errorf("app store connect: %w", err)
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("app store connect: read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		ae := &ascError{Status: resp.StatusCode}
		var apiErr struct {
			Errors []struct {
				Title  string `json:"title"`
				Detail string `json:"detail"`
			} `json:"errors"`
		}
		if json.Unmarshal(payload, &apiErr) == nil && len(apiErr.Errors) > 0 {
			ae.Title = apiErr.Errors[0].Title
			ae.Detail = apiErr.Errors[0].Detail
		}
		return ae
	}
	if out != nil {
		if err := json.Unmarshal(payload, out); err != nil {
			return fmt.Errorf("app store connect: decode response: %w", err)
		}
	}
	return nil
}

// ASCBundleID is a registered bundle ID resource.
type ASCBundleID struct {
	// ResourceID is the opaque ASC resource id (not the identifier).
	ResourceID string
	Identifier string
	Name       string
}

type ascResource struct {
	Type       string `json:"type"`
	ID         string `json:"id"`
	Attributes struct {
		Identifier     string `json:"identifier,omitempty"`
		Name           string `json:"name,omitempty"`
		CapabilityType string `json:"capabilityType,omitempty"`
		KeyID          string `json:"keyId,omitempty"`
		PrivateKey     string `json:"privateKey,omitempty"`
	} `json:"attributes"`
}

// FindBundleID looks a bundle ID up by its identifier; store.ErrNotFound
// semantics are represented by a nil result.
func (c *ASC) FindBundleID(ctx context.Context, identifier string) (*ASCBundleID, error) {
	var out struct {
		Data []ascResource `json:"data"`
	}
	path := "/v1/bundleIds?filter%5Bidentifier%5D=" + url.QueryEscape(identifier)
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	for _, d := range out.Data {
		// The filter matches loosely; require the exact identifier.
		if d.Attributes.Identifier == identifier {
			return &ASCBundleID{ResourceID: d.ID, Identifier: identifier, Name: d.Attributes.Name}, nil
		}
	}
	return nil, nil
}

// CreateBundleID registers an iOS bundle ID.
func (c *ASC) CreateBundleID(ctx context.Context, identifier, name string) (*ASCBundleID, error) {
	body := map[string]any{"data": map[string]any{
		"type": "bundleIds",
		"attributes": map[string]any{
			"identifier": identifier,
			"name":       name,
			"platform":   "IOS",
		},
	}}
	var out struct {
		Data ascResource `json:"data"`
	}
	if err := c.do(ctx, http.MethodPost, "/v1/bundleIds", body, &out); err != nil {
		return nil, err
	}
	return &ASCBundleID{ResourceID: out.Data.ID, Identifier: identifier, Name: name}, nil
}

// HasSignInWithApple reports whether the bundle ID already carries the
// APPLE_ID_AUTH capability.
func (c *ASC) HasSignInWithApple(ctx context.Context, bundleResourceID string) (bool, error) {
	var out struct {
		Data []ascResource `json:"data"`
	}
	path := "/v1/bundleIds/" + url.PathEscape(bundleResourceID) + "/bundleIdCapabilities"
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return false, err
	}
	for _, d := range out.Data {
		if d.Attributes.CapabilityType == "APPLE_ID_AUTH" {
			return true, nil
		}
	}
	return false, nil
}

// EnableSignInWithApple adds the APPLE_ID_AUTH capability to a bundle ID.
func (c *ASC) EnableSignInWithApple(ctx context.Context, bundleResourceID string) error {
	body := map[string]any{"data": map[string]any{
		"type":       "bundleIdCapabilities",
		"attributes": map[string]any{"capabilityType": "APPLE_ID_AUTH"},
		"relationships": map[string]any{
			"bundleId": map[string]any{
				"data": map[string]any{"type": "bundleIds", "id": bundleResourceID},
			},
		},
	}}
	return c.do(ctx, http.MethodPost, "/v1/bundleIdCapabilities", body, nil)
}

// CreateSignInWithAppleKey creates a Sign in with Apple key and returns its
// key ID plus the .p8 contents — Apple serves the private key exactly once,
// so the caller must store it immediately. A 404 (isASCNotFound) means the
// endpoint is not available to this account/API surface; the caller then
// degrades to the guided portal flow (see the capability spike note).
func (c *ASC) CreateSignInWithAppleKey(ctx context.Context, name, primaryBundleResourceID string) (keyID string, p8 []byte, err error) {
	body := map[string]any{"data": map[string]any{
		"type": "keys",
		"attributes": map[string]any{
			"name":    name,
			"keyType": "SIGN_IN_WITH_APPLE",
		},
		"relationships": map[string]any{
			"primaryBundleId": map[string]any{
				"data": map[string]any{"type": "bundleIds", "id": primaryBundleResourceID},
			},
		},
	}}
	var out struct {
		Data ascResource `json:"data"`
	}
	if err := c.do(ctx, http.MethodPost, "/v1/keys", body, &out); err != nil {
		return "", nil, err
	}
	keyID = out.Data.Attributes.KeyID
	if keyID == "" {
		keyID = out.Data.ID
	}
	raw, err := base64.StdEncoding.DecodeString(out.Data.Attributes.PrivateKey)
	if err != nil {
		return "", nil, fmt.Errorf("app store connect: decode private key: %w", err)
	}
	return keyID, raw, nil
}
