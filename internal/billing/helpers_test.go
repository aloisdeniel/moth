package billing

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"testing"
	"time"
)

// testCA is a self-signed CA plus a leaf whose ECDSA key signs Apple-style
// JWS blobs, standing in for Apple's real root -> WWDR -> leaf chain so x5c
// verification is exercised without Apple's certificates.
type testCA struct {
	rootCert  *x509.Certificate
	rootKey   *ecdsa.PrivateKey
	interCert *x509.Certificate
	interKey  *ecdsa.PrivateKey
	leafCert  *x509.Certificate
	leafKey   *ecdsa.PrivateKey
}

// newTestCA builds root -> intermediate -> leaf, all valid across [notBefore,
// notAfter]. Passing a narrow window lets a test expire the chain.
func newTestCA(t *testing.T, notBefore, notAfter time.Time) *testCA {
	t.Helper()
	ca := &testCA{}
	ca.rootKey = mustECKey(t)
	ca.interKey = mustECKey(t)
	ca.leafKey = mustECKey(t)

	rootTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test Apple Root CA"},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}
	ca.rootCert = mustCert(t, rootTmpl, rootTmpl, &ca.rootKey.PublicKey, ca.rootKey)

	interTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(2),
		Subject:               pkix.Name{CommonName: "Test WWDR Intermediate"},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	ca.interCert = mustCert(t, interTmpl, ca.rootCert, &ca.interKey.PublicKey, ca.rootKey)

	leafTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject:      pkix.Name{CommonName: "Test StoreKit Leaf"},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
		// Stamp Apple's App Store receipt-signing marker so the leaf passes the
		// verifyChain marker-OID pin, mirroring a real StoreKit signing cert.
		ExtraExtensions: []pkix.Extension{{Id: appStoreReceiptSigningOID, Value: []byte{0x05, 0x00}}},
	}
	ca.leafCert = mustCert(t, leafTmpl, ca.interCert, &ca.leafKey.PublicKey, ca.interKey)
	return ca
}

// unmarkedLeaf mints an alternate leaf under the same trusted intermediate that
// lacks Apple's App Store signing marker OID — an ordinary Apple-PKI leaf. It
// stands in for the finding-2 attack: a cert that legitimately chains to the
// trusted root but is not Apple's dedicated receipt-signing leaf. Returns its
// key and the leaf-first x5c chain.
func (ca *testCA) unmarkedLeaf(t *testing.T) (*ecdsa.PrivateKey, []string) {
	t.Helper()
	key := mustECKey(t)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(99),
		Subject:      pkix.Name{CommonName: "Ordinary Apple Developer Leaf"},
		NotBefore:    ca.leafCert.NotBefore,
		NotAfter:     ca.leafCert.NotAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
	}
	cert := mustCert(t, tmpl, ca.interCert, &key.PublicKey, ca.interKey)
	x5c := []string{
		base64.StdEncoding.EncodeToString(cert.Raw),
		base64.StdEncoding.EncodeToString(ca.interCert.Raw),
		base64.StdEncoding.EncodeToString(ca.rootCert.Raw),
	}
	return key, x5c
}

func (ca *testCA) rootPool() *x509.CertPool {
	p := x509.NewCertPool()
	p.AddCert(ca.rootCert)
	return p
}

// x5c returns the leaf-first base64(std) DER chain for the JWS header.
func (ca *testCA) x5c() []string {
	return []string{
		base64.StdEncoding.EncodeToString(ca.leafCert.Raw),
		base64.StdEncoding.EncodeToString(ca.interCert.Raw),
		base64.StdEncoding.EncodeToString(ca.rootCert.Raw),
	}
}

// signJWS builds a compact ES256 JWS over payload with the given x5c chain,
// signed by leafKey. The default x5c/key come from ca; override to forge.
func (ca *testCA) signJWS(t *testing.T, payload any) string {
	t.Helper()
	return signES256JWS(t, ca.leafKey, ca.x5c(), payload)
}

func signES256JWS(t *testing.T, key *ecdsa.PrivateKey, x5c []string, payload any) string {
	t.Helper()
	head := map[string]any{"alg": "ES256", "x5c": x5c}
	hb, err := json.Marshal(head)
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}
	pb, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	signingInput := b64url.EncodeToString(hb) + "." + b64url.EncodeToString(pb)
	digest := sha256.Sum256([]byte(signingInput))
	r, s, err := ecdsa.Sign(rand.Reader, key, digest[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	sig := make([]byte, 64)
	r.FillBytes(sig[:32])
	s.FillBytes(sig[32:])
	return signingInput + "." + b64url.EncodeToString(sig)
}

func mustECKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate EC key: %v", err)
	}
	return k
}

func mustCert(t *testing.T, tmpl, parent *x509.Certificate, pub *ecdsa.PublicKey, signer *ecdsa.PrivateKey) *x509.Certificate {
	t.Helper()
	der, err := x509.CreateCertificate(rand.Reader, tmpl, parent, pub, signer)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse certificate: %v", err)
	}
	return cert
}

// testServiceAccountJSON builds a Google service-account JSON key backed by a
// throwaway RSA key, plus that key for assertion verification in tests.
func testServiceAccountJSON(t *testing.T, tokenURI string) ([]byte, *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	der := x509.MarshalPKCS1PrivateKey(key)
	pemBytes := "-----BEGIN RSA PRIVATE KEY-----\n" +
		base64Wrap(base64.StdEncoding.EncodeToString(der)) +
		"-----END RSA PRIVATE KEY-----\n"
	doc := map[string]string{
		"type":           "service_account",
		"client_email":   "moth@proj.iam.gserviceaccount.com",
		"private_key_id": "kid-123",
		"private_key":    pemBytes,
		"token_uri":      tokenURI,
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal service account: %v", err)
	}
	return raw, key
}

func base64Wrap(s string) string {
	const width = 64
	var out string
	for len(s) > width {
		out += s[:width] + "\n"
		s = s[width:]
	}
	return out + s + "\n"
}
