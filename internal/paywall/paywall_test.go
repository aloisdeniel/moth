package paywall

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	storagev1 "github.com/aloisdeniel/moth/gen/moth/storage/v1"
)

func TestDefaultValidates(t *testing.T) {
	if err := Default().Validate(); err != nil {
		t.Fatalf("default config must validate: %v", err)
	}
}

func TestEncodeParseRoundTrip(t *testing.T) {
	in := Default()
	in.Offering = "promo"
	in.HighlightedIdentifier = "yearly"
	in.Legal = Legal{TermsURL: "https://example.com/terms", PrivacyURL: "https://example.com/privacy"}
	raw, err := Encode(in)
	if err != nil {
		t.Fatal(err)
	}
	out, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if out.Headline != in.Headline || out.Offering != in.Offering ||
		out.HighlightedIdentifier != in.HighlightedIdentifier ||
		len(out.Benefits) != len(in.Benefits) || out.Legal != in.Legal || out.Layout != in.Layout {
		t.Errorf("round trip mismatch: %+v vs %+v", out, in)
	}
}

func TestParseRejectsWrongVersion(t *testing.T) {
	future, err := proto.Marshal(&storagev1.StoredPaywall{Version: 99, Headline: "x", Layout: "tiles"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parse(future); err == nil {
		t.Fatal("want error for unsupported schema version")
	}
	// Empty bytes carry version 0: also rejected (callers treat empty as
	// "default config" before parsing).
	if _, err := Parse(nil); err == nil {
		t.Fatal("want error for empty document")
	}
}

func TestParseLegacyJSON(t *testing.T) {
	// The exact JSON shape the legacy Encode produced before the protobuf
	// storage migration; ParseLegacyJSON only exists for the backfill.
	raw := `{"version":1,"headline":"Unlock","subtitle":"Sub","benefits":["a","b"],"offering":"promo","highlightedIdentifier":"yearly","layout":"list","legal":{"termsUrl":"https://x.io/t"}}`
	c, err := ParseLegacyJSON([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if c.Headline != "Unlock" || c.Offering != "promo" || c.HighlightedIdentifier != "yearly" ||
		c.Layout != LayoutList || len(c.Benefits) != 2 || c.Legal.TermsURL != "https://x.io/t" {
		t.Errorf("unexpected legacy parse: %+v", c)
	}
	if _, err := ParseLegacyJSON([]byte(`{"version":99,"headline":"x","layout":"tiles"}`)); err == nil {
		t.Fatal("want error for unsupported legacy schema version")
	}
}

func tooManyBenefits() []string {
	b := make([]string, MaxBenefits+1)
	for i := range b {
		b[i] = "x"
	}
	return b
}

func TestValidate(t *testing.T) {
	valid := func() Config { return Default() }
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr bool
	}{
		{"default", func(*Config) {}, false},
		{"empty headline", func(c *Config) { c.Headline = "" }, true},
		{"blank headline", func(c *Config) { c.Headline = "   " }, true},
		{"headline too long", func(c *Config) { c.Headline = strings.Repeat("a", MaxHeadlineLen+1) }, true},
		{"subtitle too long", func(c *Config) { c.Subtitle = strings.Repeat("a", MaxSubtitleLen+1) }, true},
		{"too many benefits", func(c *Config) { c.Benefits = tooManyBenefits() }, true},
		{"empty benefit", func(c *Config) { c.Benefits = []string{"ok", ""} }, true},
		{"benefit too long", func(c *Config) { c.Benefits = []string{strings.Repeat("a", MaxBenefitLen+1)} }, true},
		{"unknown layout", func(c *Config) { c.Layout = "carousel" }, true},
		{"list layout", func(c *Config) { c.Layout = LayoutList }, false},
		{"compact layout", func(c *Config) { c.Layout = LayoutCompact }, false},
		{"bad terms url", func(c *Config) { c.Legal.TermsURL = "notaurl" }, true},
		{"relative terms url", func(c *Config) { c.Legal.TermsURL = "/terms" }, true},
		{"valid legal urls", func(c *Config) {
			c.Legal = Legal{TermsURL: "https://x.io/t", PrivacyURL: "http://x.io/p"}
		}, false},
		{"offering/highlight free-form ok", func(c *Config) {
			c.Offering = "anything"
			c.HighlightedIdentifier = "whatever"
		}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := valid()
			tc.mutate(&c)
			err := c.Validate()
			if tc.wantErr && err == nil {
				t.Errorf("want error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("want no error, got %v", err)
			}
		})
	}
}
