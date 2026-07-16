package setup

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// fingerprintRE matches one certificate fingerprint line of
// `keytool -list -v` output.
var fingerprintRE = regexp.MustCompile(`(?m)^\s*(SHA1|SHA256):\s*([0-9A-Fa-f:]+)\s*$`)

// NormalizeFingerprint validates a certificate fingerprint of size bytes
// (20 for SHA-1, 32 for SHA-256), accepting hex with or without colons,
// and returns the canonical upper-case colon-separated form.
func NormalizeFingerprint(s string, size int) (string, error) {
	hexed := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(s), ":", ""))
	if len(hexed) != size*2 {
		return "", fmt.Errorf("expected a %d-byte fingerprint (%d hex characters), got %d", size, size*2, len(hexed))
	}
	pairs := make([]string, 0, size)
	for i := 0; i < len(hexed); i += 2 {
		pair := hexed[i : i+2]
		for _, r := range pair {
			if (r < '0' || r > '9') && (r < 'A' || r > 'F') {
				return "", fmt.Errorf("invalid hex character %q in fingerprint", r)
			}
		}
		pairs = append(pairs, pair)
	}
	return strings.Join(pairs, ":"), nil
}

// ParseKeytoolOutput extracts the SHA-1 and SHA-256 certificate
// fingerprints from `keytool -list -v` output.
func ParseKeytoolOutput(out []byte) (sha1, sha256 string, err error) {
	for _, m := range fingerprintRE.FindAllStringSubmatch(string(out), -1) {
		switch m[1] {
		case "SHA1":
			if sha1 == "" {
				if sha1, err = NormalizeFingerprint(m[2], 20); err != nil {
					return "", "", fmt.Errorf("keytool SHA1: %w", err)
				}
			}
		case "SHA256":
			if sha256 == "" {
				if sha256, err = NormalizeFingerprint(m[2], 32); err != nil {
					return "", "", fmt.Errorf("keytool SHA256: %w", err)
				}
			}
		}
	}
	if sha1 == "" || sha256 == "" {
		return "", "", errors.New("no certificate fingerprints in keytool output")
	}
	return sha1, sha256, nil
}

// keystorePassEnvVar carries the keystore password to keytool via
// `-storepass:env`, keeping it off the world-readable argv.
const keystorePassEnvVar = "MOTH_KEYSTORE_PASS"

// KeystoreFingerprints computes the signing certificate fingerprints of a
// keystore with keytool. keytool cannot prompt through the captured-output
// runner (its stdin and stdout are pipes), so the caller must resolve the
// password first; storepass may be empty for keystores that list their
// certificate chain without one (JKS — PKCS12 keystores need the password).
func KeystoreFingerprints(ctx context.Context, r Runner, keystore, storepass string) (sha1, sha256 string, err error) {
	if _, err := r.LookPath("keytool"); err != nil {
		return "", "", fmt.Errorf("keytool not found on PATH (install a JDK, or paste the fingerprints instead): %w", err)
	}
	args := []string{"-list", "-v", "-keystore", keystore}
	var env []string
	if storepass != "" {
		// -storepass <value> would expose the password to every local user
		// via ps//proc for the lifetime of the keytool process; the :env
		// indirection exists precisely for this.
		args = append(args, "-storepass:env", keystorePassEnvVar)
		env = []string{keystorePassEnvVar + "=" + storepass}
	}
	out, err := r.Output(ctx, env, "keytool", args...)
	if err != nil {
		return "", "", err
	}
	return ParseKeytoolOutput(out)
}
