package billingrpc

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	_ "modernc.org/sqlite"

	"github.com/aloisdeniel/moth/internal/jwt"
	"github.com/aloisdeniel/moth/internal/keys"
	"github.com/aloisdeniel/moth/internal/mail"
	authrpc "github.com/aloisdeniel/moth/internal/server/rpc/auth"
	"github.com/aloisdeniel/moth/internal/store"
)

var b64url = base64.RawURLEncoding

const testBundleID = "com.demo.app"

// fixture is a project with catalog, one user, an access token and a billing
// handler whose clock and store endpoints the test controls.
type fixture struct {
	t       *testing.T
	h       *Handler
	st      *store.Store
	master  keys.MasterKey
	project store.Project
	user    store.User
	access  string
	now     time.Time
	ca      *testCA
	entPro  store.Entitlement
	prodID  string // moth product id granting "pro"
	dbPath  string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "moth.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	ctx := context.Background()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	master, err := keys.LoadOrCreateMasterKey(t.TempDir(), os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)

	settings := store.DefaultProjectSettings()
	settings.AccessTokenTTLSeconds = 100000
	project := store.Project{
		ID: authrpc.NewID(), Name: "Demo", Slug: "demo",
		PublishableKey: "pk_" + authrpc.NewID(), SecretKeyHash: authrpc.NewID(),
		Settings: settings, CreatedAt: now, UpdatedAt: now,
	}
	sk, err := keys.GenerateSigningKey(master)
	if err != nil {
		t.Fatal(err)
	}
	pk := store.ProjectKey{
		ID: authrpc.NewID(), ProjectID: project.ID, Kid: sk.Kid, Algorithm: sk.Algorithm,
		PublicKeyPEM: sk.PublicKeyPEM, PrivateKeyEnc: sk.PrivateKeyEnc,
		Status: store.ProjectKeyStatusActive, CreatedAt: now,
	}
	if err := st.CreateProject(ctx, project, pk); err != nil {
		t.Fatal(err)
	}
	project, err = st.GetProject(ctx, project.ID)
	if err != nil {
		t.Fatal(err)
	}

	user := store.User{ID: authrpc.NewID(), ProjectID: project.ID, Email: "u@demo.test",
		CustomClaims: "{}", CreatedAt: now, UpdatedAt: now}
	verified := now
	user.EmailVerifiedAt = &verified
	if err := st.CreateUser(ctx, user); err != nil {
		t.Fatal(err)
	}

	// Catalog: entitlement "pro" granted by product with the store SKUs below.
	entPro := store.Entitlement{ID: authrpc.NewID(), ProjectID: project.ID, Identifier: "pro",
		DisplayName: "Pro", CreatedAt: now, UpdatedAt: now}
	if err := st.CreateEntitlement(ctx, entPro); err != nil {
		t.Fatal(err)
	}
	prod := store.Product{ID: authrpc.NewID(), ProjectID: project.ID, Identifier: "monthly",
		DisplayName: "Monthly", AppleProductID: "com.demo.monthly", GoogleProductID: "monthly",
		BillingPeriod: "monthly", PriceAmountMicros: 4990000, Currency: "USD",
		EntitlementIDs: []string{entPro.ID}, CreatedAt: now, UpdatedAt: now}
	if err := st.CreateProduct(ctx, prod); err != nil {
		t.Fatal(err)
	}

	priv, err := keys.DecryptPrivateKey(master, sk.PrivateKeyEnc)
	if err != nil {
		t.Fatal(err)
	}
	access, err := jwt.Sign(priv, sk.Kid, jwt.Claims{
		Subject: user.ID, Audience: project.Slug,
		IssuedAt: now.Unix(), ExpiresAt: now.Add(time.Hour).Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}

	authH := authrpc.New(authrpc.Options{Store: st, Master: master, Mailer: mail.Console{},
		BaseURL: "http://localhost", Now: func() time.Time { return now }})

	ca := newTestCA(t, now.Add(-time.Hour), now.Add(24*time.Hour))
	h := New(Options{Store: st, Master: master, Auth: authH,
		Now: func() time.Time { return now }, AppleRoots: ca.rootPool()})

	return &fixture{t: t, h: h, st: st, master: master, project: project, user: user,
		access: access, now: now, ca: ca, entPro: entPro, prodID: prod.ID, dbPath: dbPath}
}

// storeRawEvents opens a second read-only connection to the fixture database and
// returns the type of every subscription_events row (the store exposes no list
// method — this is analytics data owned by milestone 14).
func storeRawEvents(f *fixture) ([]string, error) {
	db, err := sql.Open("sqlite", "file:"+f.dbPath+"?_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.Query(`SELECT type FROM subscription_events ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var ty string
		if err := rows.Scan(&ty); err != nil {
			return nil, err
		}
		out = append(out, ty)
	}
	return out, rows.Err()
}

// subEventRow is one raw subscription_events row for the emission tests.
type subEventRow struct {
	Type              string
	PriceAmountMicros int64
	Currency          string
}

// storeRawEventRows returns every subscription_events row's type/price/currency
// in insertion order, for asserting store-reported revenue and conversion.
func storeRawEventRows(f *fixture) ([]subEventRow, error) {
	db, err := sql.Open("sqlite", "file:"+f.dbPath+"?_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.Query(`SELECT type, price_amount_micros, currency FROM subscription_events ORDER BY created_at, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []subEventRow
	for rows.Next() {
		var r subEventRow
		if err := rows.Scan(&r.Type, &r.PriceAmountMicros, &r.Currency); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ctx returns a context scoped to the fixture project (as the pk_ interceptor
// would set it) with the Bearer access token attached to the returned request
// via authReq.
func (f *fixture) ctx() context.Context {
	return authrpc.WithProject(context.Background(), f.project)
}

func authReq[T any](f *fixture, msg *T) *connect.Request[T] {
	req := connect.NewRequest(msg)
	req.Header().Set("Authorization", "Bearer "+f.access)
	return req
}

// setAppleCreds stores encrypted Apple billing credentials pointing the App
// Store Server API client at baseURL.
func (f *fixture) setAppleCreds(baseURL string) {
	f.t.Helper()
	p8 := testP8Key(f.t)
	enc, err := f.master.Encrypt(p8)
	if err != nil {
		f.t.Fatal(err)
	}
	if err := f.st.UpsertBillingCredentials(f.ctx(), store.BillingCredentials{
		ProjectID: f.project.ID, AppleIAPKeyID: "KEY123", AppleIAPIssuerID: "ISS123",
		AppleIAPKeyEnc: enc, AppleBundleID: testBundleID, AppleAppAppleID: "123456",
		CreatedAt: f.now, UpdatedAt: f.now,
	}); err != nil {
		f.t.Fatal(err)
	}
	f.h.appleBaseURL = baseURL
	f.h.appleSandboxURL = "" // disable sandbox fallback in tests
}

// setGoogleCreds stores encrypted Google billing credentials pointing the Play
// Developer API + token endpoints at the doubles.
func (f *fixture) setGoogleCreds(baseURL, tokenURL, rtdnSecret string) {
	f.t.Helper()
	sa := testServiceAccountJSON(f.t, tokenURL)
	enc, err := f.master.Encrypt(sa)
	if err != nil {
		f.t.Fatal(err)
	}
	cred := store.BillingCredentials{ProjectID: f.project.ID, GoogleServiceAccountEnc: enc,
		GooglePackageName: testBundleID, CreatedAt: f.now, UpdatedAt: f.now}
	if rtdnSecret != "" {
		secEnc, err := f.master.Encrypt([]byte(rtdnSecret))
		if err != nil {
			f.t.Fatal(err)
		}
		cred.GoogleRTDNSecretEnc = secEnc
	}
	if err := f.st.UpsertBillingCredentials(f.ctx(), cred); err != nil {
		f.t.Fatal(err)
	}
	f.h.googleBaseURL = baseURL
	f.h.googleTokenURL = tokenURL
}

// --- Apple test CA + JWS signer (mirrors internal/billing test helpers) ----

type testCA struct {
	rootCert  *x509.Certificate
	rootKey   *ecdsa.PrivateKey
	interCert *x509.Certificate
	interKey  *ecdsa.PrivateKey
	leafCert  *x509.Certificate
	leafKey   *ecdsa.PrivateKey
}

func newTestCA(t *testing.T, notBefore, notAfter time.Time) *testCA {
	t.Helper()
	ca := &testCA{rootKey: mustECKey(t), interKey: mustECKey(t), leafKey: mustECKey(t)}
	rootTmpl := &x509.Certificate{SerialNumber: big.NewInt(1),
		Subject: pkix.Name{CommonName: "Test Apple Root"}, NotBefore: notBefore, NotAfter: notAfter,
		IsCA: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign, BasicConstraintsValid: true}
	ca.rootCert = mustCert(t, rootTmpl, rootTmpl, &ca.rootKey.PublicKey, ca.rootKey)
	interTmpl := &x509.Certificate{SerialNumber: big.NewInt(2),
		Subject: pkix.Name{CommonName: "Test WWDR"}, NotBefore: notBefore, NotAfter: notAfter,
		IsCA: true, KeyUsage: x509.KeyUsageCertSign, BasicConstraintsValid: true}
	ca.interCert = mustCert(t, interTmpl, ca.rootCert, &ca.interKey.PublicKey, ca.rootKey)
	leafTmpl := &x509.Certificate{SerialNumber: big.NewInt(3),
		Subject: pkix.Name{CommonName: "Test Leaf"}, NotBefore: notBefore, NotAfter: notAfter,
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
		// Apple's App Store receipt-signing marker OID (1.2.840.113635.100.6.11.1),
		// which verifyChain pins the leaf to.
		ExtraExtensions: []pkix.Extension{{
			Id:    asn1.ObjectIdentifier{1, 2, 840, 113635, 100, 6, 11, 1},
			Value: []byte{0x05, 0x00},
		}}}
	ca.leafCert = mustCert(t, leafTmpl, ca.interCert, &ca.leafKey.PublicKey, ca.interKey)
	return ca
}

func (ca *testCA) rootPool() *x509.CertPool {
	p := x509.NewCertPool()
	p.AddCert(ca.rootCert)
	return p
}

func (ca *testCA) x5c() []string {
	return []string{
		base64.StdEncoding.EncodeToString(ca.leafCert.Raw),
		base64.StdEncoding.EncodeToString(ca.interCert.Raw),
		base64.StdEncoding.EncodeToString(ca.rootCert.Raw),
	}
}

func (ca *testCA) signJWS(t *testing.T, payload any) string {
	t.Helper()
	head, _ := json.Marshal(map[string]any{"alg": "ES256", "x5c": ca.x5c()})
	pb, _ := json.Marshal(payload)
	signingInput := b64url.EncodeToString(head) + "." + b64url.EncodeToString(pb)
	digest := sha256.Sum256([]byte(signingInput))
	r, s, err := ecdsa.Sign(rand.Reader, ca.leafKey, digest[:])
	if err != nil {
		t.Fatal(err)
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
		t.Fatal(err)
	}
	return k
}

func mustCert(t *testing.T, tmpl, parent *x509.Certificate, pub *ecdsa.PublicKey, signer *ecdsa.PrivateKey) *x509.Certificate {
	t.Helper()
	der, err := x509.CreateCertificate(rand.Reader, tmpl, parent, pub, signer)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return cert
}

// testP8Key returns a PEM-encoded PKCS#8 EC P-256 private key standing in for an
// Apple In-App-Purchase .p8.
func testP8Key(t *testing.T) []byte {
	t.Helper()
	key := mustECKey(t)
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
}

// testServiceAccountJSON builds a minimal Google service-account JSON key
// backed by a throwaway RSA key (ParseServiceAccount requires a real RSA key).
func testServiceAccountJSON(t *testing.T, tokenURI string) []byte {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der := x509.MarshalPKCS1PrivateKey(key)
	pemBytes := string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: der}))
	doc := map[string]string{
		"type": "service_account", "client_email": "moth@proj.iam.gserviceaccount.com",
		"private_key_id": "kid-1", "private_key": pemBytes, "token_uri": tokenURI,
	}
	raw, _ := json.Marshal(doc)
	return raw
}

// appleTxn is a signed-transaction payload builder.
func appleTxn(origTxnID, productID string, expires time.Time, offerType, revocation int) map[string]any {
	return map[string]any{
		"transactionId": origTxnID, "originalTransactionId": origTxnID,
		"bundleId": testBundleID, "productId": productID,
		"subscriptionGroupIdentifier": "grp1", "expiresDate": expires.UnixMilli(),
		"type": "Auto-Renewable Subscription", "environment": "Production",
		"offerType": offerType, "revocationDate": revocation,
	}
}

// appleStatusDouble serves the App Store Server API Get All Subscription
// Statuses response for a single transaction with the given status code and
// signed transaction payload.
func (f *fixture) appleStatusDouble(statusCode int, txn map[string]any) *httptest.Server {
	signed := f.ca.signJWS(f.t, txn)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"environment": "Production", "bundleId": testBundleID,
			"data": []map[string]any{{
				"subscriptionGroupIdentifier": "grp1",
				"lastTransactions": []map[string]any{{
					"originalTransactionId": txn["originalTransactionId"],
					"status":                statusCode,
					"signedTransactionInfo": signed,
				}},
			}},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	f.t.Cleanup(srv.Close)
	return srv
}

// googleDoubles serves the token endpoint and a subscriptionsv2.get response
// with the given state and expiry.
func (f *fixture) googleDoubles(state string, expiry time.Time) *httptest.Server {
	return f.googleDoublesOffer(state, expiry, "")
}

// googleDoublesOffer is googleDoubles with a base-plan offer applied to the
// line item when offerID is non-empty (a free-trial / intro offer), so the
// normalized status reads as trialing until offerID is dropped.
func (f *fixture) googleDoublesOffer(state string, expiry time.Time, offerID string) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "expires_in": 3600})
			return
		}
		li := map[string]any{
			"productId": "monthly", "expiryTime": expiry.UTC().Format(time.RFC3339),
			"autoRenewingPlan": map[string]any{"autoRenewEnabled": true},
		}
		if offerID != "" {
			li["offerDetails"] = map[string]any{"offerId": offerID, "basePlanId": "monthly"}
		}
		resp := map[string]any{
			"subscriptionState":    state,
			"lineItems":            []map[string]any{li},
			"acknowledgementState": "acknowledgementStateAcknowledged",
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	f.t.Cleanup(srv.Close)
	return srv
}
