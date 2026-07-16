package billing

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aloisdeniel/moth/internal/oidc"
)

// App Store Server API hosts. The API mirrors data across environments: a
// production transaction is unknown to the sandbox host and vice-versa, so a
// production lookup that 404s is retried against sandbox (Apple's documented
// fallback rule).
const (
	AppleStoreKitProdURL    = "https://api.storekit.itunes.apple.com"
	AppleStoreKitSandboxURL = "https://api.storekit-sandbox.itunes.apple.com"
	// appleAPIAudience is the fixed aud claim of an App Store Server API JWT.
	appleAPIAudience = "appstoreconnect-v1"
	// appleTokenLifetime stays well under Apple's 1-hour cap.
	appleTokenLifetime = 30 * time.Minute
)

// ParseP8 re-exports oidc.ParseP8: an Apple .p8 In-App-Purchase key is the
// same PEM-wrapped PKCS#8 EC P-256 as a Sign-in-with-Apple key.
func ParseP8(data []byte) (*ecdsa.PrivateKey, error) { return oidc.ParseP8(data) }

// AppleClient calls the App Store Server API, authenticating each request with
// a short-lived ES256 JWT minted from the project's In-App-Purchase .p8 key
// (same mechanism as App Store Connect, distinct key type and audience). Every
// signed blob it receives is verified through Verifier before use.
type AppleClient struct {
	// BaseURL is the primary host (defaults to production).
	BaseURL string
	// SandboxURL is the fallback host tried on a production 404 (defaults to
	// the sandbox host; set "" to disable fallback, e.g. in tests).
	SandboxURL string
	IssuerID   string // ASC issuer id (iss)
	KeyID      string // In-App-Purchase key id (JWS kid)
	BundleID   string // bid claim + verifier bundle check
	Key        *ecdsa.PrivateKey
	HTTPC      Doer
	Now        func() time.Time
	// Verifier verifies the signed transaction/renewal JWS in responses.
	// Defaults to a verifier over Apple's real roots bound to BundleID.
	Verifier *AppleVerifier
}

func (c *AppleClient) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func (c *AppleClient) verifier() *AppleVerifier {
	if c.Verifier != nil {
		return c.Verifier
	}
	c.Verifier = NewAppleVerifier(nil, c.BundleID, c.Now)
	return c.Verifier
}

// Token mints the ES256 request JWT: header alg/kid/typ; claims
// iss/iat/exp/aud/bid (Apple's "Generating JSON Web Tokens for API requests").
func (c *AppleClient) Token() (string, error) {
	head, err := json.Marshal(map[string]string{"alg": "ES256", "kid": c.KeyID, "typ": "JWT"})
	if err != nil {
		return "", err
	}
	now := c.now()
	claims, err := json.Marshal(map[string]any{
		"iss": c.IssuerID,
		"iat": now.Unix(),
		"exp": now.Add(appleTokenLifetime).Unix(),
		"aud": appleAPIAudience,
		"bid": c.BundleID,
	})
	if err != nil {
		return "", err
	}
	signingInput := b64url.EncodeToString(head) + "." + b64url.EncodeToString(claims)
	digest := sha256.Sum256([]byte(signingInput))
	r, s, err := ecdsa.Sign(rand.Reader, c.Key, digest[:])
	if err != nil {
		return "", fmt.Errorf("billing: sign apple api token: %w", err)
	}
	sig := make([]byte, 64)
	r.FillBytes(sig[:32])
	s.FillBytes(sig[32:])
	return signingInput + "." + b64url.EncodeToString(sig), nil
}

// appleAPIError is a non-2xx App Store Server API response.
type appleAPIError struct {
	Status    int
	ErrorCode int64
	Message   string
}

func (e *appleAPIError) Error() string {
	return fmt.Sprintf("app store server api: status %d: [%d] %s", e.Status, e.ErrorCode, e.Message)
}

// get issues an authenticated GET against base+path and decodes out. It
// returns ErrNotFound (wrapped) on 404.
func (c *AppleClient) get(ctx context.Context, base, path string, out any) error {
	if base == "" {
		base = AppleStoreKitProdURL
	}
	httpc := c.HTTPC
	if httpc == nil {
		httpc = defaultDoer()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+path, nil)
	if err != nil {
		return err
	}
	tok, err := c.Token()
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Accept", "application/json")
	resp, err := httpc.Do(req)
	if err != nil {
		return fmt.Errorf("app store server api: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("app store server api: read response: %w", err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("%w: %s", ErrNotFound, apiErrMessage(body))
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		ae := &appleAPIError{Status: resp.StatusCode}
		var parsed struct {
			ErrorCode    int64  `json:"errorCode"`
			ErrorMessage string `json:"errorMessage"`
		}
		if json.Unmarshal(body, &parsed) == nil {
			ae.ErrorCode = parsed.ErrorCode
			ae.Message = parsed.ErrorMessage
		}
		return ae
	}
	if out != nil {
		if err := json.Unmarshal(body, out); err != nil {
			return fmt.Errorf("app store server api: decode response: %w", err)
		}
	}
	return nil
}

func apiErrMessage(body []byte) string {
	var parsed struct {
		ErrorMessage string `json:"errorMessage"`
	}
	_ = json.Unmarshal(body, &parsed)
	return parsed.ErrorMessage
}

// statusResponse is the Get All Subscription Statuses body.
type statusResponse struct {
	Environment string `json:"environment"`
	BundleID    string `json:"bundleId"`
	Data        []struct {
		SubscriptionGroupIdentifier string `json:"subscriptionGroupIdentifier"`
		LastTransactions            []struct {
			OriginalTransactionID string `json:"originalTransactionId"`
			Status                int    `json:"status"`
			SignedTransactionInfo string `json:"signedTransactionInfo"`
			SignedRenewalInfo     string `json:"signedRenewalInfo"`
		} `json:"lastTransactions"`
	} `json:"data"`
}

// GetAllSubscriptionStatuses resolves an originalTransactionId to authoritative
// state: it fetches the subscription-group statuses, verifies the signed
// transaction + renewal info of the matching last transaction, and returns a
// NormalizedSubscription. On a production 404 it retries sandbox (the
// documented fallback), so a sandbox transaction resolves without the caller
// guessing the environment.
func (c *AppleClient) GetAllSubscriptionStatuses(ctx context.Context, originalTransactionID string) (NormalizedSubscription, error) {
	path := "/inApps/v1/subscriptions/" + originalTransactionID
	var out statusResponse
	err := c.get(ctx, c.BaseURL, path, &out)
	if isNotFound(err) && c.shouldFallback() {
		err = c.get(ctx, c.sandboxURL(), path, &out)
	}
	if err != nil {
		return NormalizedSubscription{}, err
	}
	return c.normalizeStatus(originalTransactionID, &out)
}

func (c *AppleClient) normalizeStatus(originalTransactionID string, out *statusResponse) (NormalizedSubscription, error) {
	v := c.verifier()
	for _, group := range out.Data {
		for _, lt := range group.LastTransactions {
			if originalTransactionID != "" && lt.OriginalTransactionID != originalTransactionID {
				continue
			}
			txn, err := v.VerifyTransaction(lt.SignedTransactionInfo)
			if err != nil {
				return NormalizedSubscription{}, err
			}
			var ri *JWSRenewalInfo
			if lt.SignedRenewalInfo != "" {
				if ri, err = v.VerifyRenewalInfo(lt.SignedRenewalInfo); err != nil {
					return NormalizedSubscription{}, err
				}
			}
			status := appleStatusFromCode(lt.Status, txn)
			return normalizeAppleSubscription(txn, ri, status), nil
		}
	}
	return NormalizedSubscription{}, fmt.Errorf("%w: no transaction for %q", ErrNotFound, originalTransactionID)
}

// transactionResponse is the Get Transaction Info body.
type transactionResponse struct {
	SignedTransactionInfo string `json:"signedTransactionInfo"`
}

// GetTransactionInfo fetches and verifies a single transaction by
// transactionId, returning its decoded payload. Production 404 falls back to
// sandbox.
func (c *AppleClient) GetTransactionInfo(ctx context.Context, transactionID string) (*JWSTransaction, error) {
	path := "/inApps/v1/transactions/" + transactionID
	var out transactionResponse
	err := c.get(ctx, c.BaseURL, path, &out)
	if isNotFound(err) && c.shouldFallback() {
		err = c.get(ctx, c.sandboxURL(), path, &out)
	}
	if err != nil {
		return nil, err
	}
	return c.verifier().VerifyTransaction(out.SignedTransactionInfo)
}

func (c *AppleClient) shouldFallback() bool {
	return c.sandboxURL() != "" && c.sandboxURL() != c.BaseURL
}

func (c *AppleClient) sandboxURL() string {
	if c.SandboxURL == "" && c.BaseURL == "" {
		return AppleStoreKitSandboxURL
	}
	return c.SandboxURL
}

func isNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}
