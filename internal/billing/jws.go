package billing

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"
)

var b64url = base64.RawURLEncoding

// appStoreReceiptSigningOID (1.2.840.113635.100.6.11.1) is the marker extension
// Apple stamps on the leaf certificate that signs StoreKit transactions, signed
// renewal info and App Store Server Notifications V2. Apple Root CA - G3 anchors
// much of Apple's PKI (ordinary WWDR developer/distribution certs chain to it
// too), so a chain-to-root check alone is not enough: without pinning this
// marker, any holder of any Apple-issued leaf could mint a JWS with an
// attacker-chosen bundleId / originalTransactionId. Apple's own App Store Server
// Library requires this check for exactly that reason.
var appStoreReceiptSigningOID = asn1.ObjectIdentifier{1, 2, 840, 113635, 100, 6, 11, 1}

// jwsHeader is the header of an Apple StoreKit / App Store Server JWS. Apple
// signs these with ES256 and attaches the signing certificate chain in the
// x5c header (leaf first), rather than publishing a JWKS — so verification is
// an x509 chain build to Apple's root CA, not a kid lookup.
type jwsHeader struct {
	Alg string   `json:"alg"`
	X5C []string `json:"x5c"`
}

// verifyAppleJWS verifies a compact ES256 JWS whose header carries an x5c
// certificate chain, and returns the decoded (still-JSON) payload bytes.
//
// Verification is threefold and any failure rejects the token:
//  1. alg must be ES256 (no "none", no alg confusion) and x5c must be present.
//  2. The x5c chain (leaf, intermediates...) must build to a certificate in
//     roots, valid at now — this is what proves Apple, not an attacker, minted
//     the token. The root embedded in x5c is never trusted on its own; only
//     roots (Apple's real root CA, or a test CA in tests) is a trust anchor.
//  3. The leaf's public key must actually verify the JWS signature, so a valid
//     chain stapled onto a tampered body is rejected.
//
// Apple's x5c entries are standard-base64 DER, while the JWS segments are
// base64url — a detail that trips up naive implementations.
func verifyAppleJWS(token string, roots *x509.CertPool, now time.Time) ([]byte, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("%w: expected 3 JWS segments", ErrMalformed)
	}
	headRaw, err := b64url.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("%w: header: %v", ErrMalformed, err)
	}
	var head jwsHeader
	if err := json.Unmarshal(headRaw, &head); err != nil {
		return nil, fmt.Errorf("%w: header: %v", ErrMalformed, err)
	}
	if head.Alg != "ES256" {
		return nil, fmt.Errorf("%w: unexpected alg %q", ErrMalformed, head.Alg)
	}
	if len(head.X5C) == 0 {
		return nil, fmt.Errorf("%w: missing x5c chain", ErrMalformed)
	}

	leaf, err := verifyChain(head.X5C, roots, now)
	if err != nil {
		return nil, err
	}
	pub, ok := leaf.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("%w: leaf key is not ECDSA", ErrUntrustedChain)
	}

	sig, err := b64url.DecodeString(parts[2])
	if err != nil || len(sig) != 64 {
		return nil, fmt.Errorf("%w: signature", ErrMalformed)
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	r := new(big.Int).SetBytes(sig[:32])
	s := new(big.Int).SetBytes(sig[32:])
	if !ecdsa.Verify(pub, digest[:], r, s) {
		return nil, ErrInvalidSignature
	}

	payload, err := b64url.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("%w: payload: %v", ErrMalformed, err)
	}
	return payload, nil
}

// verifyChain parses a base64-DER x5c chain (leaf first) and verifies it to a
// trust anchor in roots, valid at now, AND requires the leaf to carry Apple's
// App Store receipt-signing marker OID so that only Apple's dedicated signing
// leaf — not any developer certificate that happens to chain to Apple Root CA -
// G3 — is accepted. It returns the leaf certificate. The leaf's ExtKeyUsage is
// not otherwise constrained (the marker + signature-over-JWS binding is what
// matters), so KeyUsages accepts any.
func verifyChain(x5c []string, roots *x509.CertPool, now time.Time) (*x509.Certificate, error) {
	if roots == nil {
		return nil, fmt.Errorf("%w: no trust anchors", ErrUntrustedChain)
	}
	certs := make([]*x509.Certificate, 0, len(x5c))
	for i, enc := range x5c {
		der, err := base64.StdEncoding.DecodeString(enc)
		if err != nil {
			return nil, fmt.Errorf("%w: x5c[%d] base64: %v", ErrMalformed, i, err)
		}
		cert, err := x509.ParseCertificate(der)
		if err != nil {
			return nil, fmt.Errorf("%w: x5c[%d]: %v", ErrMalformed, i, err)
		}
		certs = append(certs, cert)
	}
	intermediates := x509.NewCertPool()
	for _, c := range certs[1:] {
		intermediates.AddCert(c)
	}
	if _, err := certs[0].Verify(x509.VerifyOptions{
		Roots:         roots,
		Intermediates: intermediates,
		CurrentTime:   now,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUntrustedChain, err)
	}
	if !hasExtension(certs[0], appStoreReceiptSigningOID) {
		return nil, fmt.Errorf("%w: leaf is not an App Store signing certificate", ErrUntrustedChain)
	}
	return certs[0], nil
}

// hasExtension reports whether cert carries an extension with the given OID.
func hasExtension(cert *x509.Certificate, oid asn1.ObjectIdentifier) bool {
	for _, ext := range cert.Extensions {
		if ext.Id.Equal(oid) {
			return true
		}
	}
	return false
}
