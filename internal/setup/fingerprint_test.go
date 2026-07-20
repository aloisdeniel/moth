package setup

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestNormalizeFingerprint(t *testing.T) {
	tests := []struct {
		name string
		in   string
		size int
		want string
		ok   bool
	}{
		{
			name: "sha1 with colons",
			in:   "bb:0d:ac:74:d3:3e:f9:04:eb:3f:ab:fb:0c:a8:1f:2f:20:70:7d:f4",
			size: 20,
			want: "BB:0D:AC:74:D3:3E:F9:04:EB:3F:AB:FB:0C:A8:1F:2F:20:70:7D:F4",
			ok:   true,
		},
		{
			name: "sha1 without colons",
			in:   "BB0DAC74D33EF904EB3FABFB0CA81F2F20707DF4",
			size: 20,
			want: "BB:0D:AC:74:D3:3E:F9:04:EB:3F:AB:FB:0C:A8:1F:2F:20:70:7D:F4",
			ok:   true,
		},
		{
			name: "sha256",
			in:   strings.Repeat("ab:", 31) + "ab",
			size: 32,
			want: strings.TrimSuffix(strings.Repeat("AB:", 32), ":"),
			ok:   true,
		},
		{name: "wrong length", in: "AB:CD", size: 20},
		{name: "not hex", in: strings.Repeat("ZZ", 20), size: 20},
		{name: "empty", in: "", size: 20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeFingerprint(tt.in, tt.size)
			if tt.ok != (err == nil) {
				t.Fatalf("NormalizeFingerprint(%q) error = %v, want ok=%v", tt.in, err, tt.ok)
			}
			if tt.ok && got != tt.want {
				t.Fatalf("NormalizeFingerprint(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

const keytoolOutput = `
Alias name: upload
Creation date: Jan 1, 2026
Entry type: PrivateKeyEntry
Certificate fingerprints:
	 SHA1: BB:0D:AC:74:D3:3E:F9:04:EB:3F:AB:FB:0C:A8:1F:2F:20:70:7D:F4
	 SHA256: 09:83:F8:24:99:32:8B:11:8D:6B:02:C3:D8:1D:7F:CA:C1:5A:C6:76:0F:D4:C0:64:22:AF:68:9A:6B:BC:D6:AD
Signature algorithm name: SHA256withRSA
`

func TestParseKeytoolOutput(t *testing.T) {
	sha1, sha256, err := ParseKeytoolOutput([]byte(keytoolOutput))
	if err != nil {
		t.Fatal(err)
	}
	if want := "BB:0D:AC:74:D3:3E:F9:04:EB:3F:AB:FB:0C:A8:1F:2F:20:70:7D:F4"; sha1 != want {
		t.Fatalf("sha1 = %q, want %q", sha1, want)
	}
	if !strings.HasPrefix(sha256, "09:83:F8:24") {
		t.Fatalf("sha256 = %q", sha256)
	}

	if _, _, err := ParseKeytoolOutput([]byte("no fingerprints here")); err == nil {
		t.Fatal("expected an error for output without fingerprints")
	}
}

// fakeRunner scripts LookPath and Output for gcloud/keytool.
type fakeRunner struct {
	missing map[string]bool
	output  map[string][]byte
	err     error
	calls   []string
	envs    [][]string
}

func (f *fakeRunner) LookPath(name string) (string, error) {
	if f.missing[name] {
		return "", errors.New(name + " not found")
	}
	return "/usr/bin/" + name, nil
}

func (f *fakeRunner) Output(_ context.Context, env []string, name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, name+" "+strings.Join(args, " "))
	f.envs = append(f.envs, env)
	if f.err != nil {
		return nil, f.err
	}
	return f.output[name], nil
}

func TestKeystoreFingerprints(t *testing.T) {
	r := &fakeRunner{output: map[string][]byte{"keytool": []byte(keytoolOutput)}}
	sha1, _, err := KeystoreFingerprints(context.Background(), r, "/tmp/upload.jks", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(sha1, "BB:0D") {
		t.Fatalf("sha1 = %q", sha1)
	}
	// The password travels through the environment, never the argv (argv is
	// world-readable via ps//proc for the process lifetime).
	if len(r.calls) != 1 || !strings.Contains(r.calls[0], "-storepass:env "+keystorePassEnvVar) {
		t.Fatalf("keytool call = %v", r.calls)
	}
	if strings.Contains(r.calls[0], "secret") {
		t.Fatalf("password leaked into the argv: %v", r.calls)
	}
	if len(r.envs) != 1 || len(r.envs[0]) != 1 || r.envs[0][0] != keystorePassEnvVar+"=secret" {
		t.Fatalf("keytool env = %v", r.envs)
	}

	// An empty password omits -storepass entirely (JKS keystores list their
	// fingerprints without one).
	r = &fakeRunner{output: map[string][]byte{"keytool": []byte(keytoolOutput)}}
	if _, _, err := KeystoreFingerprints(context.Background(), r, "/tmp/upload.jks", ""); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(r.calls[0], "storepass") || r.envs[0] != nil {
		t.Fatalf("empty password should omit -storepass: %v env %v", r.calls, r.envs)
	}

	missing := &fakeRunner{missing: map[string]bool{"keytool": true}}
	if _, _, err := KeystoreFingerprints(context.Background(), missing, "/tmp/upload.jks", ""); err == nil {
		t.Fatal("expected an error when keytool is missing")
	}
}
