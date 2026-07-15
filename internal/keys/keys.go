// Package keys manages the instance master key and per-project ES256
// signing keypairs.
//
// The master key never signs anything: it only encrypts project private
// keys at rest (AES-256-GCM). Each project gets its own ES256 (ECDSA P-256)
// keypair so a token minted for one app can never validate for another.
package keys

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MasterKeyEnv is the environment variable that injects the master key
// (64 hex chars) instead of reading it from disk, for KMS-style setups.
const MasterKeyEnv = "MOTH_MASTER_KEY"

const masterKeyLen = 32

// MasterKey encrypts project private keys at rest.
type MasterKey struct {
	key [masterKeyLen]byte
}

// LoadOrCreateMasterKey returns the instance master key. Precedence: the
// MOTH_MASTER_KEY environment value (via getenv), then dataDir/keys/master.key,
// which is generated on first use.
func LoadOrCreateMasterKey(dataDir string, getenv func(string) string) (MasterKey, error) {
	if v := getenv(MasterKeyEnv); v != "" {
		return parseMasterKey(v)
	}
	path := filepath.Join(dataDir, "keys", "master.key")
	raw, err := os.ReadFile(path)
	if err == nil {
		return parseMasterKey(string(raw))
	}
	if !errors.Is(err, os.ErrNotExist) {
		return MasterKey{}, fmt.Errorf("read master key: %w", err)
	}

	var mk MasterKey
	if _, err := rand.Read(mk.key[:]); err != nil {
		return MasterKey{}, fmt.Errorf("generate master key: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return MasterKey{}, fmt.Errorf("create keys dir: %w", err)
	}
	if err := os.WriteFile(path, []byte(hex.EncodeToString(mk.key[:])+"\n"), 0o600); err != nil {
		return MasterKey{}, fmt.Errorf("write master key: %w", err)
	}
	return mk, nil
}

func parseMasterKey(s string) (MasterKey, error) {
	raw, err := hex.DecodeString(strings.TrimSpace(s))
	if err != nil || len(raw) != masterKeyLen {
		return MasterKey{}, fmt.Errorf("master key must be %d hex-encoded bytes", masterKeyLen)
	}
	var mk MasterKey
	copy(mk.key[:], raw)
	return mk, nil
}

// Encrypt seals plaintext with AES-256-GCM; the nonce is prepended.
func (mk MasterKey) Encrypt(plaintext []byte) ([]byte, error) {
	gcm, err := mk.aead()
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt opens a ciphertext produced by Encrypt.
func (mk MasterKey) Decrypt(ciphertext []byte) ([]byte, error) {
	gcm, err := mk.aead()
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}
	nonce, sealed := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	return plaintext, nil
}

func (mk MasterKey) aead() (cipher.AEAD, error) {
	block, err := aes.NewCipher(mk.key[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// SigningKey is a freshly generated per-project ES256 keypair in the forms
// the store needs: the kid, the public part as PEM, and the private part
// encrypted under the master key.
type SigningKey struct {
	Kid           string
	Algorithm     string
	PublicKeyPEM  string
	PrivateKeyEnc []byte
}

// GenerateSigningKey creates a new ES256 keypair for a project.
func GenerateSigningKey(mk MasterKey) (SigningKey, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return SigningKey{}, fmt.Errorf("generate keypair: %w", err)
	}
	kid, err := Thumbprint(&priv.PublicKey)
	if err != nil {
		return SigningKey{}, err
	}
	pubDER, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		return SigningKey{}, fmt.Errorf("marshal public key: %w", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})
	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return SigningKey{}, fmt.Errorf("marshal private key: %w", err)
	}
	enc, err := mk.Encrypt(privDER)
	if err != nil {
		return SigningKey{}, fmt.Errorf("encrypt private key: %w", err)
	}
	return SigningKey{
		Kid:           kid,
		Algorithm:     "ES256",
		PublicKeyPEM:  string(pubPEM),
		PrivateKeyEnc: enc,
	}, nil
}

// DecryptPrivateKey recovers a project's private key from its encrypted
// PKCS#8 form.
func DecryptPrivateKey(mk MasterKey, enc []byte) (*ecdsa.PrivateKey, error) {
	der, err := mk.Decrypt(enc)
	if err != nil {
		return nil, err
	}
	key, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	ec, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key is not ECDSA")
	}
	return ec, nil
}

// Thumbprint computes the RFC 7638 JWK thumbprint of a P-256 public key,
// base64url-encoded; moth uses it as the key's kid.
func Thumbprint(pub *ecdsa.PublicKey) (string, error) {
	x, y, err := coords(pub)
	if err != nil {
		return "", err
	}
	// Keys in lexicographic order per RFC 7638.
	canonical := fmt.Sprintf(`{"crv":"P-256","kty":"EC","x":%q,"y":%q}`, x, y)
	sum := sha256.Sum256([]byte(canonical))
	return base64.RawURLEncoding.EncodeToString(sum[:]), nil
}

// JWK is a JSON Web Key restricted to the fields moth serves.
type JWK struct {
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
}

// JWKS is the document served at /p/{slug}/.well-known/jwks.json.
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// BuildJWKS assembles the JWKS document for a project's public keys, given
// as (kid, public key PEM) pairs.
func BuildJWKS(pems map[string]string) ([]byte, error) {
	jwks := JWKS{Keys: []JWK{}}
	for kid, pubPEM := range pems {
		pub, err := ParsePublicKeyPEM(pubPEM)
		if err != nil {
			return nil, err
		}
		x, y, err := coords(pub)
		if err != nil {
			return nil, err
		}
		jwks.Keys = append(jwks.Keys, JWK{
			Kty: "EC", Crv: "P-256", X: x, Y: y,
			Kid: kid, Use: "sig", Alg: "ES256",
		})
	}
	return json.Marshal(jwks)
}

// ParsePublicKeyPEM parses a PEM-encoded ECDSA public key.
func ParsePublicKeyPEM(pubPEM string) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pubPEM))
	if block == nil {
		return nil, errors.New("invalid public key PEM")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}
	ec, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("public key is not ECDSA")
	}
	return ec, nil
}

func coords(pub *ecdsa.PublicKey) (x, y string, err error) {
	if pub.Curve != elliptic.P256() {
		return "", "", errors.New("public key is not P-256")
	}
	size := 32
	xb := pub.X.FillBytes(make([]byte, size))
	yb := pub.Y.FillBytes(make([]byte, size))
	return base64.RawURLEncoding.EncodeToString(xb), base64.RawURLEncoding.EncodeToString(yb), nil
}
