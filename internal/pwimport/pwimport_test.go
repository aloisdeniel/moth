package pwimport

import (
	"errors"
	"testing"
)

// All vectors below encode the same password, generated with x/crypto.
const vecPassword = "correct horse battery staple"

var vectors = []struct {
	algo    string
	encoded string
}{
	{"bcrypt", "$2a$10$pJKK0Jmzk4Ew6uargL6udea8oA9QKlVVL.cZ74CMO5az2xxMLJHKK"},
	{"scrypt", "$scrypt$ln=15,r=8,p=1$MDEyMzQ1Njc4OWFiY2RlZg$9rcVF+DZ8uU77qz3H/v29+n2g8c877AOCRXSQvC/fs0"},
	{"pbkdf2", "$pbkdf2-sha256$100000$MDEyMzQ1Njc4OWFiY2RlZg$CQEF03iMraucElCfobodRqkaFY16F3mxFDIvP9WoJcs"},
	{"argon2", "$argon2id$v=19$m=19456,t=2,p=1$MDEyMzQ1Njc4OWFiY2RlZg$gy5SuVm5Z7Vw7keB9se9p87QGcomaseB/S2U1OhTsM0"},
	{"argon2", "$argon2i$v=19$m=19456,t=2,p=1$MDEyMzQ1Njc4OWFiY2RlZg$a4Vmb8BztzJQAjar4pBJac3RHJRwOM41W0GNxOaaUsU"},
}

func TestVerifyKnownVectors(t *testing.T) {
	for _, v := range vectors {
		t.Run(v.encoded[:12], func(t *testing.T) {
			ok, err := Verify(v.algo, v.encoded, vecPassword)
			if err != nil {
				t.Fatalf("Verify error: %v", err)
			}
			if !ok {
				t.Fatal("correct password rejected")
			}
			bad, err := Verify(v.algo, v.encoded, "wrong password")
			if err != nil {
				t.Fatalf("Verify(wrong) error: %v", err)
			}
			if bad {
				t.Fatal("wrong password accepted")
			}
		})
	}
}

func TestVerifyCaseInsensitiveAlgo(t *testing.T) {
	ok, err := Verify("BCRYPT", vectors[0].encoded, vecPassword)
	if err != nil || !ok {
		t.Fatalf("case-insensitive algo failed: ok=%v err=%v", ok, err)
	}
}

func TestVerifyUnknownAlgorithm(t *testing.T) {
	_, err := Verify("md5", "x", "y")
	if !errors.Is(err, ErrUnsupportedAlgorithm) {
		t.Fatalf("expected ErrUnsupportedAlgorithm, got %v", err)
	}
}

func TestVerifyMalformed(t *testing.T) {
	cases := []struct{ algo, encoded string }{
		{"scrypt", "$scrypt$bad"},
		{"scrypt", "$scrypt$ln=15,r=8,p=1$@@@$@@@"},
		{"pbkdf2", "$pbkdf2-md5$1000$c2FsdA$aGFzaA"},
		{"pbkdf2", "not-a-hash"},
		{"argon2", "$argon2id$v=19$m=1$onlythree$x"},
		{"argon2", "$argon2d$v=19$m=16,t=1,p=1$c2FsdA$aGFzaA"},
	}
	for _, c := range cases {
		if _, err := Verify(c.algo, c.encoded, "pw"); err == nil {
			t.Fatalf("expected error for %s %q", c.algo, c.encoded)
		}
	}
}

func TestNeedsRehash(t *testing.T) {
	// Foreign schemes always want a rehash to argon2id.
	for _, v := range vectors {
		want := v.encoded[:9] != "$argon2id"
		if got := NeedsRehash(v.algo, v.encoded); got != want {
			t.Fatalf("NeedsRehash(%q) = %v, want %v", v.encoded[:12], got, want)
		}
	}
	if NeedsRehash("argon2", "$argon2id$v=19$m=19456,t=2,p=1$c2FsdA$aGFzaA") {
		t.Fatal("argon2id hash should not need a rehash")
	}
	if !NeedsRehash("bcrypt", "$2a$10$abc") {
		t.Fatal("bcrypt hash should need a rehash")
	}
}

func TestPBKDF2PaddedBase64(t *testing.T) {
	// Same pbkdf2 vector but with padded base64 must still verify.
	padded := "$pbkdf2-sha256$100000$MDEyMzQ1Njc4OWFiY2RlZg==$CQEF03iMraucElCfobodRqkaFY16F3mxFDIvP9WoJcs="
	ok, err := Verify("pbkdf2", padded, vecPassword)
	if err != nil || !ok {
		t.Fatalf("padded base64 failed: ok=%v err=%v", ok, err)
	}
}
