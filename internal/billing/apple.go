package billing

import (
	"crypto/x509"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// appleRootG3PEM is Apple Root CA - G3, the trust anchor for StoreKit 2 signed
// transactions, signed renewal info, and App Store Server Notifications V2.
// Embedded so verification needs no network and no host trust store.
//
//go:embed roots/AppleRootCA-G3.pem
var appleRootG3PEM []byte

var (
	appleRootsOnce sync.Once
	appleRoots     *x509.CertPool
)

// AppleRoots returns the CertPool moth verifies Apple JWS chains against — the
// embedded Apple Root CA - G3. Tests inject their own pool instead.
func AppleRoots() *x509.CertPool {
	appleRootsOnce.Do(func() {
		appleRoots = x509.NewCertPool()
		if !appleRoots.AppendCertsFromPEM(appleRootPEM()) {
			panic("billing: embedded Apple root CA failed to parse")
		}
	})
	return appleRoots
}

func appleRootPEM() []byte { return appleRootG3PEM }

// JWSTransaction is the JWSTransactionDecodedPayload subset moth reads from a
// StoreKit 2 signed transaction. Dates are Apple's milliseconds-since-epoch.
type JWSTransaction struct {
	TransactionID               string `json:"transactionId"`
	OriginalTransactionID       string `json:"originalTransactionId"`
	BundleID                    string `json:"bundleId"`
	ProductID                   string `json:"productId"`
	SubscriptionGroupIdentifier string `json:"subscriptionGroupIdentifier"`
	PurchaseDate                int64  `json:"purchaseDate"`
	ExpiresDate                 int64  `json:"expiresDate"`
	// Type is e.g. "Auto-Renewable Subscription".
	Type string `json:"type"`
	// Environment is "Sandbox" or "Production".
	Environment string `json:"environment"`
	// OfferType is non-zero for introductory / promotional / trial offers;
	// used to distinguish trialing from active.
	OfferType int `json:"offerType"`
	// RevocationDate is set when Apple revoked the purchase (refund, family
	// removal). Non-zero => revoked.
	RevocationDate int64 `json:"revocationDate"`
}

// JWSRenewalInfo is the JWSRenewalInfoDecodedPayload subset moth reads from a
// signed renewal-info blob.
type JWSRenewalInfo struct {
	OriginalTransactionID  string `json:"originalTransactionId"`
	ProductID              string `json:"productId"`
	AutoRenewProductID     string `json:"autoRenewProductId"`
	AutoRenewStatus        int    `json:"autoRenewStatus"` // 1 on, 0 off
	ExpirationIntent       int    `json:"expirationIntent"`
	GracePeriodExpiresDate int64  `json:"gracePeriodExpiresDate"`
	Environment            string `json:"environment"`
}

// AppleVerifier verifies Apple JWS blobs (transactions, renewal info,
// notifications) against a trust anchor. Zero value is not usable; use
// NewAppleVerifier.
type AppleVerifier struct {
	roots        *x509.CertPool
	now          func() time.Time
	expectBundle string
}

// NewAppleVerifier returns a verifier over roots (defaults to AppleRoots when
// nil — tests pass a test CA pool). now defaults to time.Now. expectBundle,
// when non-empty, is required to match every verified transaction's bundleId,
// rejecting a receipt minted for another app.
func NewAppleVerifier(roots *x509.CertPool, expectBundle string, now func() time.Time) *AppleVerifier {
	if roots == nil {
		roots = AppleRoots()
	}
	if now == nil {
		now = time.Now
	}
	return &AppleVerifier{roots: roots, now: now, expectBundle: expectBundle}
}

// VerifyTransaction verifies a signed transaction JWS and returns its decoded
// payload. It rejects a tampered signature, an untrusted or expired chain, and
// (when expectBundle is set) a mismatched bundle id.
func (v *AppleVerifier) VerifyTransaction(jws string) (*JWSTransaction, error) {
	payload, err := verifyAppleJWS(jws, v.roots, v.now())
	if err != nil {
		return nil, err
	}
	var txn JWSTransaction
	if err := json.Unmarshal(payload, &txn); err != nil {
		return nil, fmt.Errorf("%w: transaction payload: %v", ErrMalformed, err)
	}
	if v.expectBundle != "" && txn.BundleID != v.expectBundle {
		return nil, fmt.Errorf("%w: got %q want %q", ErrBundleMismatch, txn.BundleID, v.expectBundle)
	}
	return &txn, nil
}

// VerifyRenewalInfo verifies a signed renewal-info JWS and returns its decoded
// payload.
func (v *AppleVerifier) VerifyRenewalInfo(jws string) (*JWSRenewalInfo, error) {
	payload, err := verifyAppleJWS(jws, v.roots, v.now())
	if err != nil {
		return nil, err
	}
	var ri JWSRenewalInfo
	if err := json.Unmarshal(payload, &ri); err != nil {
		return nil, fmt.Errorf("%w: renewal payload: %v", ErrMalformed, err)
	}
	return &ri, nil
}

// appleNotificationPayload is the ResponseBodyV2DecodedPayload of an App Store
// Server Notification V2.
type appleNotificationPayload struct {
	NotificationType string `json:"notificationType"`
	Subtype          string `json:"subtype"`
	NotificationUUID string `json:"notificationUUID"`
	Version          string `json:"version"`
	SignedDate       int64  `json:"signedDate"`
	Data             struct {
		BundleID              string `json:"bundleId"`
		Environment           string `json:"environment"`
		SignedTransactionInfo string `json:"signedTransactionInfo"`
		SignedRenewalInfo     string `json:"signedRenewalInfo"`
	} `json:"data"`
}

// VerifyNotification verifies an App Store Server Notification V2 signedPayload
// (same x5c chain as a transaction), then verifies the nested transaction /
// renewal-info JWS it carries, and returns a NormalizedNotification. The
// notification's Subscription is best-effort: the caller must still re-read
// authoritative state via the App Store Server API before granting.
func (v *AppleVerifier) VerifyNotification(signedPayload string) (*NormalizedNotification, error) {
	payload, err := verifyAppleJWS(signedPayload, v.roots, v.now())
	if err != nil {
		return nil, err
	}
	var np appleNotificationPayload
	if err := json.Unmarshal(payload, &np); err != nil {
		return nil, fmt.Errorf("%w: notification payload: %v", ErrMalformed, err)
	}
	if v.expectBundle != "" && np.Data.BundleID != "" && np.Data.BundleID != v.expectBundle {
		return nil, fmt.Errorf("%w: got %q want %q", ErrBundleMismatch, np.Data.BundleID, v.expectBundle)
	}

	var txn *JWSTransaction
	if np.Data.SignedTransactionInfo != "" {
		if txn, err = v.VerifyTransaction(np.Data.SignedTransactionInfo); err != nil {
			return nil, err
		}
	}
	var ri *JWSRenewalInfo
	if np.Data.SignedRenewalInfo != "" {
		if ri, err = v.VerifyRenewalInfo(np.Data.SignedRenewalInfo); err != nil {
			return nil, err
		}
	}

	status := appleNotificationStatus(np.NotificationType, np.Subtype, txn)
	sub := normalizeAppleSubscription(txn, ri, status)
	return &NormalizedNotification{
		Store:          StoreApple,
		Type:           np.NotificationType,
		Subtype:        np.Subtype,
		NotificationID: np.NotificationUUID,
		Subscription:   sub,
		Raw:            payload,
	}, nil
}

// normalizeAppleEnv maps Apple's "Sandbox"/"Production" to moth's lowercase
// environment constants.
func normalizeAppleEnv(env string) string {
	if strings.EqualFold(env, "Sandbox") {
		return EnvSandbox
	}
	return EnvProduction
}

// normalizeAppleSubscription builds a NormalizedSubscription from a verified
// transaction, optional renewal info, and a resolved status.
func normalizeAppleSubscription(txn *JWSTransaction, ri *JWSRenewalInfo, status string) NormalizedSubscription {
	sub := NormalizedSubscription{Store: StoreApple, Status: status}
	if txn != nil {
		sub.ProductID = txn.ProductID
		sub.StoreTransactionID = txn.OriginalTransactionID
		sub.SubscriptionID = txn.SubscriptionGroupIdentifier
		sub.CurrentPeriodEnd = appleMillisToTime(txn.ExpiresDate)
		sub.Environment = normalizeAppleEnv(txn.Environment)
		raw, _ := json.Marshal(txn)
		sub.RawState = raw
	}
	if ri != nil {
		sub.AutoRenew = ri.AutoRenewStatus == 1
		if sub.Environment == "" {
			sub.Environment = normalizeAppleEnv(ri.Environment)
		}
		// During a billing grace period the last paid transaction's ExpiresDate
		// is already in the past (that is why renewal is being retried); access
		// is retained until gracePeriodExpiresDate. Surface that as the period
		// end so a consumer gating on `now < expireTime` does not wrongly treat
		// a still-entitled grace-period user as lapsed.
		if (status == StatusInGracePeriod || status == StatusInBillingRetry) && ri.GracePeriodExpiresDate != 0 {
			sub.CurrentPeriodEnd = appleMillisToTime(ri.GracePeriodExpiresDate)
		}
	}
	return sub
}

// appleStatusFromCode maps an App Store Server API subscription status code
// (Get All Subscription Statuses: data[].lastTransactions[].status) plus the
// transaction to a moth status. Codes: 1 active, 2 expired, 3 billing retry,
// 4 grace period, 5 revoked.
func appleStatusFromCode(code int, txn *JWSTransaction) string {
	switch code {
	case 1:
		if txn != nil && txn.OfferType != 0 {
			return StatusTrialing
		}
		return StatusActive
	case 2:
		return StatusExpired
	case 3:
		return StatusInBillingRetry
	case 4:
		return StatusInGracePeriod
	case 5:
		return StatusRevoked
	default:
		return StatusExpired
	}
}

// appleNotificationStatus maps an ASSN V2 notificationType/subtype (and the
// transaction) to a moth status for the best-effort notification subscription.
func appleNotificationStatus(nType, subtype string, txn *JWSTransaction) string {
	switch nType {
	case "SUBSCRIBED", "DID_RENEW", "OFFER_REDEEMED", "DID_CHANGE_RENEWAL_PREF", "DID_CHANGE_RENEWAL_STATUS", "RENEWAL_EXTENDED", "PRICE_INCREASE":
		if txn != nil && txn.OfferType != 0 && nType == "SUBSCRIBED" {
			return StatusTrialing
		}
		return StatusActive
	case "DID_FAIL_TO_RENEW":
		if subtype == "GRACE_PERIOD" {
			return StatusInGracePeriod
		}
		return StatusInBillingRetry
	case "GRACE_PERIOD_EXPIRED":
		return StatusInBillingRetry
	case "EXPIRED":
		return StatusExpired
	case "REFUND", "REVOKE", "CONSUMPTION_REQUEST":
		return StatusRevoked
	default:
		return StatusExpired
	}
}
