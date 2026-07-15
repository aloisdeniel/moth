package password

import (
	"strings"
	"testing"
)

func TestHashAndVerify(t *testing.T) {
	h, err := Hash("hunter22")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(h, "$argon2id$") {
		t.Fatalf("unexpected format: %s", h)
	}
	if !Verify("hunter22", h) {
		t.Fatal("correct password rejected")
	}
	if Verify("hunter23", h) {
		t.Fatal("wrong password accepted")
	}
	if Verify("hunter22", "garbage") {
		t.Fatal("garbage hash accepted")
	}

	h2, err := Hash("hunter22")
	if err != nil {
		t.Fatal(err)
	}
	if h == h2 {
		t.Fatal("hashes must be salted (identical output for same input)")
	}
}
