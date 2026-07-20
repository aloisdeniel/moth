// Package token generates and hashes the random secrets moth hands out:
// publishable/secret API keys, admin session tokens, setup tokens.
package token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"strings"
)

// Prefixes for project API keys.
const (
	PublishableKeyPrefix = "pk_"
	SecretKeyPrefix      = "sk_"
)

// PATPrefix is the prefix of admin personal access tokens, the credential
// the moth CLI presents as `authorization: Bearer` metadata.
const PATPrefix = "moth_pat_"

// lowercase base32 without padding: URL-safe, case-insensitive-proof, and
// double-click selectable in a terminal.
var encoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// New returns prefix + 26 random characters (130 bits of entropy).
func New(prefix string) string {
	return prefix + Random(16)
}

// Random returns n random bytes encoded as lowercase base32.
func Random(n int) string {
	raw := make([]byte, n)
	if _, err := rand.Read(raw); err != nil {
		// crypto/rand never fails on supported platforms; if it does the
		// process must not continue handing out secrets.
		panic(fmt.Sprintf("crypto/rand: %v", err))
	}
	return strings.ToLower(encoding.EncodeToString(raw))
}

// Hash returns the hex SHA-256 of a token. High-entropy random tokens need
// no salt or slow KDF.
func Hash(tok string) string {
	sum := sha256.Sum256([]byte(tok))
	return hex.EncodeToString(sum[:])
}
