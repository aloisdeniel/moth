package keys

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func noEnv(string) string { return "" }

func TestLoadOrCreateMasterKeyPersists(t *testing.T) {
	dir := t.TempDir()
	mk1, err := LoadOrCreateMasterKey(dir, noEnv)
	if err != nil {
		t.Fatal(err)
	}
	mk2, err := LoadOrCreateMasterKey(dir, noEnv)
	if err != nil {
		t.Fatal(err)
	}
	if mk1.key != mk2.key {
		t.Fatal("master key changed between loads")
	}
	info, err := os.Stat(filepath.Join(dir, "keys", "master.key"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("master key file should be 0600, got %v", info.Mode().Perm())
	}
}

func TestMasterKeyFromEnv(t *testing.T) {
	raw := make([]byte, 32)
	for i := range raw {
		raw[i] = byte(i)
	}
	env := func(k string) string {
		if k == MasterKeyEnv {
			return hex.EncodeToString(raw)
		}
		return ""
	}
	dir := t.TempDir()
	mk, err := LoadOrCreateMasterKey(dir, env)
	if err != nil {
		t.Fatal(err)
	}
	if mk.key[0] != 0 || mk.key[31] != 31 {
		t.Fatal("env master key not used")
	}
	// Env injection must not write anything to disk.
	if _, err := os.Stat(filepath.Join(dir, "keys")); !os.IsNotExist(err) {
		t.Fatal("keys dir should not exist when key comes from env")
	}

	if _, err := LoadOrCreateMasterKey(dir, func(string) string { return "zz" }); err == nil {
		t.Fatal("invalid env key should error")
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	mk, err := LoadOrCreateMasterKey(t.TempDir(), noEnv)
	if err != nil {
		t.Fatal(err)
	}
	plaintext := []byte("secret key material")
	ct, err := mk.Encrypt(plaintext)
	if err != nil {
		t.Fatal(err)
	}
	pt, err := mk.Decrypt(ct)
	if err != nil {
		t.Fatal(err)
	}
	if string(pt) != string(plaintext) {
		t.Fatal("round trip mismatch")
	}
	// Tampering must be detected.
	ct[len(ct)-1] ^= 1
	if _, err := mk.Decrypt(ct); err == nil {
		t.Fatal("tampered ciphertext should fail to decrypt")
	}
}

func TestGenerateSigningKeyDistinctAndRecoverable(t *testing.T) {
	mk, err := LoadOrCreateMasterKey(t.TempDir(), noEnv)
	if err != nil {
		t.Fatal(err)
	}
	k1, err := GenerateSigningKey(mk)
	if err != nil {
		t.Fatal(err)
	}
	k2, err := GenerateSigningKey(mk)
	if err != nil {
		t.Fatal(err)
	}
	if k1.Kid == k2.Kid || k1.PublicKeyPEM == k2.PublicKeyPEM {
		t.Fatal("two projects must get distinct keypairs")
	}
	if k1.Algorithm != "ES256" {
		t.Fatalf("algorithm: %s", k1.Algorithm)
	}

	// Private key round-trips through master-key encryption and matches
	// the public part.
	priv, err := DecryptPrivateKey(mk, k1.PrivateKeyEnc)
	if err != nil {
		t.Fatal(err)
	}
	pub, err := ParsePublicKeyPEM(k1.PublicKeyPEM)
	if err != nil {
		t.Fatal(err)
	}
	if !priv.PublicKey.Equal(pub) {
		t.Fatal("decrypted private key does not match public key")
	}

	// kid is the RFC 7638 thumbprint of the public key.
	kid, err := Thumbprint(pub)
	if err != nil {
		t.Fatal(err)
	}
	if kid != k1.Kid {
		t.Fatalf("kid mismatch: %s vs %s", kid, k1.Kid)
	}

	// A different master key must not decrypt it.
	other, err := LoadOrCreateMasterKey(t.TempDir(), noEnv)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecryptPrivateKey(other, k1.PrivateKeyEnc); err == nil {
		t.Fatal("wrong master key should fail to decrypt")
	}
}

func TestBuildJWKSMatchesKey(t *testing.T) {
	mk, err := LoadOrCreateMasterKey(t.TempDir(), noEnv)
	if err != nil {
		t.Fatal(err)
	}
	k, err := GenerateSigningKey(mk)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := BuildJWKS(map[string]string{k.Kid: k.PublicKeyPEM})
	if err != nil {
		t.Fatal(err)
	}
	var jwks JWKS
	if err := json.Unmarshal(doc, &jwks); err != nil {
		t.Fatal(err)
	}
	if len(jwks.Keys) != 1 {
		t.Fatalf("want 1 key, got %d", len(jwks.Keys))
	}
	jwk := jwks.Keys[0]
	if jwk.Kty != "EC" || jwk.Crv != "P-256" || jwk.Alg != "ES256" || jwk.Use != "sig" || jwk.Kid != k.Kid {
		t.Fatalf("jwk metadata: %+v", jwk)
	}

	pub, err := ParsePublicKeyPEM(k.PublicKeyPEM)
	if err != nil {
		t.Fatal(err)
	}
	x, _ := base64.RawURLEncoding.DecodeString(jwk.X)
	y, _ := base64.RawURLEncoding.DecodeString(jwk.Y)
	want := pub.X.FillBytes(make([]byte, 32))
	if string(x) != string(want) {
		t.Fatal("JWKS x coordinate does not match public key")
	}
	wantY := pub.Y.FillBytes(make([]byte, 32))
	if string(y) != string(wantY) {
		t.Fatal("JWKS y coordinate does not match public key")
	}
}
