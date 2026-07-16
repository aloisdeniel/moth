// Package pwimport verifies foreign password hashes so users migrated from
// another auth system (Firebase, Auth0, Supabase, a home-grown backend) can
// sign in with their existing password without a reset.
//
// A `moth project import` supplies, per user, an algorithm tag and an encoded
// hash. On that user's first sign-in moth calls Verify with the tag, the
// stored hash and the submitted password; on success the caller rehashes the
// password to moth's native argon2id (see NeedsRehash) and drops the foreign
// hash, so every account converges on one scheme after one login.
//
// Supported tags and their encoded forms:
//
//	bcrypt  standard modular-crypt string, e.g. "$2b$10$....".
//	scrypt  "$scrypt$ln=<log2N>,r=<r>,p=<p>$<saltB64>$<hashB64>"
//	        (RFC-style; ln is log2 of the cost parameter N).
//	pbkdf2  "$pbkdf2-<sha1|sha256|sha512>$<iter>$<saltB64>$<hashB64>"
//	        (passlib layout; the hash length selects the derived-key size).
//	argon2  PHC string "$argon2id|argon2i$v=19$m=..,t=..,p=..$<saltB64>$<hashB64>".
//
// Base64 fields are decoded as raw (unpadded) standard base64, falling back
// to padded standard base64, so hashes exported by tools using either form
// verify.
package pwimport

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"hash"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"
)

// ErrUnsupportedAlgorithm is returned for an algorithm tag pwimport does not
// know how to verify.
var ErrUnsupportedAlgorithm = errors.New("pwimport: unsupported algorithm")

// Verify reports whether password matches the encoded foreign hash under the
// named algorithm (bcrypt, scrypt, pbkdf2 or argon2, case-insensitive). A
// mismatching password returns (false, nil); a malformed hash or unknown
// algorithm returns a non-nil error so import bugs are not silently read as
// wrong passwords.
func Verify(algo, encoded, password string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(algo)) {
	case "bcrypt":
		return verifyBcrypt(encoded, password)
	case "scrypt":
		return verifyScrypt(encoded, password)
	case "pbkdf2":
		return verifyPBKDF2(encoded, password)
	case "argon2", "argon2id", "argon2i":
		return verifyArgon2(encoded, password)
	default:
		return false, fmt.Errorf("%w: %q", ErrUnsupportedAlgorithm, algo)
	}
}

// NeedsRehash reports whether a successfully-verified hash should be replaced
// with a fresh moth-native argon2id hash. It is true for every foreign scheme
// and for argon2 variants that are not already argon2id in PHC form, and
// false only when the stored hash is already an argon2id PHC string. It does
// not judge argon2id cost parameters — moth rehashes imported users on first
// login regardless, so a coarse format check suffices.
func NeedsRehash(algo, encoded string) bool {
	return !strings.HasPrefix(encoded, "$argon2id$")
}

func verifyBcrypt(encoded, password string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(encoded), []byte(password))
	if err == nil {
		return true, nil
	}
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return false, nil
	}
	return false, fmt.Errorf("pwimport: bcrypt: %w", err)
}

func verifyScrypt(encoded, password string) (bool, error) {
	// $scrypt$ln=..,r=..,p=..$salt$hash
	parts := strings.Split(encoded, "$")
	if len(parts) != 5 || parts[0] != "" || parts[1] != "scrypt" {
		return false, errors.New("pwimport: malformed scrypt hash")
	}
	var ln, r, p int
	if _, err := fmt.Sscanf(parts[2], "ln=%d,r=%d,p=%d", &ln, &r, &p); err != nil {
		return false, fmt.Errorf("pwimport: scrypt params: %w", err)
	}
	if ln <= 0 || ln > 31 || r <= 0 || p <= 0 {
		return false, errors.New("pwimport: scrypt params out of range")
	}
	salt, err := decodeB64(parts[3])
	if err != nil {
		return false, fmt.Errorf("pwimport: scrypt salt: %w", err)
	}
	want, err := decodeB64(parts[4])
	if err != nil {
		return false, fmt.Errorf("pwimport: scrypt hash: %w", err)
	}
	got, err := scrypt.Key([]byte(password), salt, 1<<ln, r, p, len(want))
	if err != nil {
		return false, fmt.Errorf("pwimport: scrypt: %w", err)
	}
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

func verifyPBKDF2(encoded, password string) (bool, error) {
	// $pbkdf2-<hash>$<iter>$salt$hash
	parts := strings.Split(encoded, "$")
	if len(parts) != 5 || parts[0] != "" {
		return false, errors.New("pwimport: malformed pbkdf2 hash")
	}
	name, ok := strings.CutPrefix(parts[1], "pbkdf2-")
	if !ok {
		return false, errors.New("pwimport: pbkdf2 scheme must be pbkdf2-<hash>")
	}
	var newHash func() hash.Hash
	switch name {
	case "sha1":
		newHash = sha1.New
	case "sha256":
		newHash = sha256.New
	case "sha512":
		newHash = sha512.New
	default:
		return false, fmt.Errorf("pwimport: unsupported pbkdf2 hash %q", name)
	}
	iter, err := strconv.Atoi(parts[2])
	if err != nil || iter <= 0 {
		return false, errors.New("pwimport: pbkdf2 iteration count")
	}
	salt, err := decodeB64(parts[3])
	if err != nil {
		return false, fmt.Errorf("pwimport: pbkdf2 salt: %w", err)
	}
	want, err := decodeB64(parts[4])
	if err != nil {
		return false, fmt.Errorf("pwimport: pbkdf2 hash: %w", err)
	}
	got := pbkdf2.Key([]byte(password), salt, iter, len(want), newHash)
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

func verifyArgon2(encoded, password string) (bool, error) {
	// $argon2id$v=19$m=..,t=..,p=..$salt$hash
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" {
		return false, errors.New("pwimport: malformed argon2 hash")
	}
	variant := parts[1]
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, fmt.Errorf("pwimport: argon2 version: %w", err)
	}
	if version != argon2.Version {
		return false, fmt.Errorf("pwimport: unsupported argon2 version %d", version)
	}
	var m, t uint32
	var par uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &par); err != nil {
		return false, fmt.Errorf("pwimport: argon2 params: %w", err)
	}
	if m == 0 || t == 0 || par == 0 {
		return false, errors.New("pwimport: argon2 params out of range")
	}
	salt, err := decodeB64(parts[4])
	if err != nil {
		return false, fmt.Errorf("pwimport: argon2 salt: %w", err)
	}
	want, err := decodeB64(parts[5])
	if err != nil {
		return false, fmt.Errorf("pwimport: argon2 hash: %w", err)
	}
	var got []byte
	switch variant {
	case "argon2id":
		got = argon2.IDKey([]byte(password), salt, t, m, par, uint32(len(want)))
	case "argon2i":
		got = argon2.Key([]byte(password), salt, t, m, par, uint32(len(want)))
	default:
		// argon2d is not implemented by x/crypto.
		return false, fmt.Errorf("%w: %s", ErrUnsupportedAlgorithm, variant)
	}
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

// decodeB64 accepts raw (unpadded) or padded standard base64.
func decodeB64(s string) ([]byte, error) {
	if b, err := base64.RawStdEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	return base64.StdEncoding.DecodeString(s)
}
