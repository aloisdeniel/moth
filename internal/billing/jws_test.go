package billing

import (
	"errors"
	"strings"
	"testing"
	"time"
)

var testNow = time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)

func sampleTxn() map[string]any {
	return map[string]any{
		"transactionId":         "txn-1",
		"originalTransactionId": "orig-1",
		"bundleId":              "com.example.app",
		"productId":             "pro.monthly",
		"expiresDate":           testNow.Add(30 * 24 * time.Hour).UnixMilli(),
		"type":                  "Auto-Renewable Subscription",
		"environment":           "Production",
	}
}

func TestVerifyAppleJWSValidChain(t *testing.T) {
	ca := newTestCA(t, testNow.Add(-time.Hour), testNow.Add(time.Hour))
	tok := ca.signJWS(t, sampleTxn())

	payload, err := verifyAppleJWS(tok, ca.rootPool(), testNow)
	if err != nil {
		t.Fatalf("verifyAppleJWS() error = %v", err)
	}
	if !strings.Contains(string(payload), "com.example.app") {
		t.Fatalf("payload missing bundle id: %s", payload)
	}
}

func TestVerifyAppleJWSWrongRoot(t *testing.T) {
	ca := newTestCA(t, testNow.Add(-time.Hour), testNow.Add(time.Hour))
	other := newTestCA(t, testNow.Add(-time.Hour), testNow.Add(time.Hour))
	tok := ca.signJWS(t, sampleTxn())

	// Verify against a pool that trusts a different root: chain must fail.
	if _, err := verifyAppleJWS(tok, other.rootPool(), testNow); !errors.Is(err, ErrUntrustedChain) {
		t.Fatalf("error = %v, want ErrUntrustedChain", err)
	}
}

func TestVerifyAppleJWSExpiredChain(t *testing.T) {
	ca := newTestCA(t, testNow.Add(-2*time.Hour), testNow.Add(-time.Hour)) // already expired
	tok := ca.signJWS(t, sampleTxn())

	if _, err := verifyAppleJWS(tok, ca.rootPool(), testNow); !errors.Is(err, ErrUntrustedChain) {
		t.Fatalf("error = %v, want ErrUntrustedChain (expired)", err)
	}
}

func TestVerifyAppleJWSTampered(t *testing.T) {
	ca := newTestCA(t, testNow.Add(-time.Hour), testNow.Add(time.Hour))
	tok := ca.signJWS(t, sampleTxn())

	// Flip a byte of the payload segment; signature no longer matches.
	parts := strings.Split(tok, ".")
	body := []byte(parts[1])
	body[0] ^= 0x01
	tampered := parts[0] + "." + string(body) + "." + parts[2]

	if _, err := verifyAppleJWS(tampered, ca.rootPool(), testNow); err == nil {
		t.Fatal("verifyAppleJWS() accepted a tampered payload")
	}
}

func TestVerifyAppleJWSForgedLeaf(t *testing.T) {
	ca := newTestCA(t, testNow.Add(-time.Hour), testNow.Add(time.Hour))
	// Attacker keeps the trusted chain in x5c but signs with their own key.
	attacker := mustECKey(t)
	tok := signES256JWS(t, attacker, ca.x5c(), sampleTxn())

	if _, err := verifyAppleJWS(tok, ca.rootPool(), testNow); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("error = %v, want ErrInvalidSignature", err)
	}
}

func TestVerifyAppleJWSRejectsLeafWithoutMarker(t *testing.T) {
	ca := newTestCA(t, testNow.Add(-time.Hour), testNow.Add(time.Hour))
	// Attacker holds an ordinary Apple-PKI leaf: it validly chains to the
	// trusted root but lacks Apple's App Store signing marker OID. They sign a
	// well-formed JWS with their own key and staple the real chain.
	key, x5c := ca.unmarkedLeaf(t)
	tok := signES256JWS(t, key, x5c, sampleTxn())

	if _, err := verifyAppleJWS(tok, ca.rootPool(), testNow); !errors.Is(err, ErrUntrustedChain) {
		t.Fatalf("error = %v, want ErrUntrustedChain (leaf missing App Store marker OID)", err)
	}
}

func TestVerifyAppleJWSRejectsNonES256(t *testing.T) {
	ca := newTestCA(t, testNow.Add(-time.Hour), testNow.Add(time.Hour))
	tok := ca.signJWS(t, sampleTxn())
	// Rebuild header with alg=none.
	parts := strings.Split(tok, ".")
	noneHead := b64url.EncodeToString([]byte(`{"alg":"none","x5c":["x"]}`))
	if _, err := verifyAppleJWS(noneHead+"."+parts[1]+"."+parts[2], ca.rootPool(), testNow); !errors.Is(err, ErrMalformed) {
		t.Fatalf("error = %v, want ErrMalformed", err)
	}
}

func TestEmbeddedAppleRootParses(t *testing.T) {
	// The embedded Apple Root CA - G3 must load into a usable pool.
	if AppleRoots() == nil {
		t.Fatal("AppleRoots() returned nil")
	}
}
