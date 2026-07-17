// Package paywall defines the per-project paywall configuration: the copy
// and layout knobs that the SDK's batteries-included paywall screen renders
// from (milestone 13). It deliberately owns no token space of its own —
// colors, typography, spacing, radius and logo all come from the design
// system (internal/theme). A paywall config only carries what the theme
// cannot: the headline/subtitle copy, the benefit bullets, which offering to
// present, which tier to highlight, the layout variant and the legal links.
//
// Like a theme, a config is a small versioned document stored per project
// with a revision id (as a moth.projectconfig.v1.StoredPaywall protobuf message),
// delivered to clients through the public billing API and cached client-side
// by revision (stale-while-revalidate).
package paywall

import (
	"fmt"
	"net/url"
	"slices"
	"strings"

	"google.golang.org/protobuf/proto"

	projectconfigv1 "github.com/aloisdeniel/moth/gen/moth/projectconfig/v1"
)

// SchemaVersion is stamped on every encoded paywall document. Parse rejects
// documents from a different version; future schema changes bump it and add
// an explicit upgrade path.
const SchemaVersion = 1

// Layout variants a paywall can render in. The SDK maps each to a widget
// arrangement; the token space (colors/spacing/radius) is the theme's.
const (
	// LayoutTiles renders one card per tier side by side (the default).
	LayoutTiles = "tiles"
	// LayoutList stacks the tiers as full-width rows.
	LayoutList = "list"
	// LayoutCompact shows a single selected tier with a period toggle.
	LayoutCompact = "compact"
)

// Layouts is the accepted set; Validate rejects anything else.
var Layouts = []string{LayoutTiles, LayoutList, LayoutCompact}

// Bounds accepted by Validate. Copy is bounded so a paywall always renders
// legibly on a phone; the store price/period come from the catalog, never
// from this config.
const (
	MaxHeadlineLen = 80
	MaxSubtitleLen = 240
	MaxBenefits    = 8
	MaxBenefitLen  = 120
)

// Config is one project's complete paywall configuration.
type Config struct {
	// Version is the schema version of the document (SchemaVersion).
	Version int
	// Headline is the paywall's primary title (required).
	Headline string
	// Subtitle is the supporting line under the headline (optional).
	Subtitle string
	// Benefits are the feature/benefit bullets, in display order.
	Benefits []string
	// Offering is the offering tag whose products the paywall lists; empty
	// means the project's default offering.
	Offering string
	// HighlightedIdentifier is the product identifier rendered as "most
	// popular"; empty highlights nothing. A stable catalog identifier (e.g.
	// "yearly"), never a store SKU — it survives store re-provisioning.
	HighlightedIdentifier string
	// Layout is one of Layouts.
	Layout string
	// Legal holds the optional terms/privacy links rendered in the footer.
	Legal Legal
}

// Legal holds the optional legal links rendered in the paywall footer.
type Legal struct {
	TermsURL   string
	PrivacyURL string
}

// Default returns the paywall config applied to projects that never
// customized anything: a generic premium headline, three benefit bullets,
// the default offering and the tiles layout.
func Default() Config {
	return Config{
		Version:  SchemaVersion,
		Headline: "Unlock Premium",
		Subtitle: "Get the full experience with a subscription.",
		Benefits: []string{
			"Unlimited access to every feature",
			"Priority support",
			"New features first",
		},
		Layout: LayoutTiles,
	}
}

// Encode serializes the config as its canonical storage document (a
// moth.projectconfig.v1.StoredPaywall protobuf message), stamping the current
// schema version. An encoded document is never empty: empty stored bytes
// keep meaning "the built-in default paywall".
func Encode(c Config) ([]byte, error) {
	c.Version = SchemaVersion
	raw, err := proto.Marshal(ToProto(c))
	if err != nil {
		return nil, fmt.Errorf("encode paywall: %w", err)
	}
	return raw, nil
}

// Parse decodes a stored paywall document (moth.projectconfig.v1.StoredPaywall).
// It rejects documents from a different schema version — including empty
// input, which callers treat as "default paywall" before parsing; it does
// not validate values (Validate does).
func Parse(raw []byte) (Config, error) {
	var msg projectconfigv1.StoredPaywall
	if err := proto.Unmarshal(raw, &msg); err != nil {
		return Config{}, fmt.Errorf("parse paywall: %w", err)
	}
	if msg.Version != SchemaVersion {
		return Config{}, fmt.Errorf("parse paywall: unsupported schema version %d (want %d)", msg.Version, SchemaVersion)
	}
	return FromProto(&msg), nil
}

// ToProto converts the domain config into its storage message.
func ToProto(c Config) *projectconfigv1.StoredPaywall {
	return &projectconfigv1.StoredPaywall{
		Version:               int32(c.Version),
		Headline:              c.Headline,
		Subtitle:              c.Subtitle,
		Benefits:              c.Benefits,
		Offering:              c.Offering,
		HighlightedIdentifier: c.HighlightedIdentifier,
		Layout:                c.Layout,
		Legal:                 &projectconfigv1.LegalLinks{TermsUrl: c.Legal.TermsURL, PrivacyUrl: c.Legal.PrivacyURL},
	}
}

// FromProto converts a storage message into the domain config (a nil legal
// sub-message becomes empty links).
func FromProto(msg *projectconfigv1.StoredPaywall) Config {
	c := Config{
		Version:               int(msg.GetVersion()),
		Headline:              msg.GetHeadline(),
		Subtitle:              msg.GetSubtitle(),
		Benefits:              msg.GetBenefits(),
		Offering:              msg.GetOffering(),
		HighlightedIdentifier: msg.GetHighlightedIdentifier(),
		Layout:                msg.GetLayout(),
	}
	if l := msg.GetLegal(); l != nil {
		c.Legal = Legal{TermsURL: l.TermsUrl, PrivacyURL: l.PrivacyUrl}
	}
	return c
}

// Validate checks every field and returns the first violation. Offering and
// HighlightedIdentifier are catalog references validated for shape only, not
// existence: a paywall renders gracefully when a referenced product is gone
// (optional by construction), so the store never has to stay in lockstep
// with the config.
func (c Config) Validate() error {
	if c.Version != SchemaVersion {
		return fmt.Errorf("unsupported schema version %d (want %d)", c.Version, SchemaVersion)
	}
	if strings.TrimSpace(c.Headline) == "" {
		return fmt.Errorf("headline is required")
	}
	if len(c.Headline) > MaxHeadlineLen {
		return fmt.Errorf("headline: %d chars exceeds the %d maximum", len(c.Headline), MaxHeadlineLen)
	}
	if len(c.Subtitle) > MaxSubtitleLen {
		return fmt.Errorf("subtitle: %d chars exceeds the %d maximum", len(c.Subtitle), MaxSubtitleLen)
	}
	if len(c.Benefits) > MaxBenefits {
		return fmt.Errorf("benefits: %d exceeds the %d maximum", len(c.Benefits), MaxBenefits)
	}
	for i, b := range c.Benefits {
		if strings.TrimSpace(b) == "" {
			return fmt.Errorf("benefits[%d]: must not be empty", i)
		}
		if len(b) > MaxBenefitLen {
			return fmt.Errorf("benefits[%d]: %d chars exceeds the %d maximum", i, len(b), MaxBenefitLen)
		}
	}
	if !slices.Contains(Layouts, c.Layout) {
		return fmt.Errorf("layout: unknown variant %q (available: %s)", c.Layout, strings.Join(Layouts, ", "))
	}
	for name, u := range map[string]string{"termsUrl": c.Legal.TermsURL, "privacyUrl": c.Legal.PrivacyURL} {
		if u == "" {
			continue
		}
		if err := validHTTPURL(u); err != nil {
			return fmt.Errorf("legal.%s: %w", name, err)
		}
	}
	return nil
}

// validHTTPURL accepts only absolute http(s) URLs.
func validHTTPURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid URL %q: %w", raw, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("URL %q must be http(s)", raw)
	}
	if u.Host == "" {
		return fmt.Errorf("URL %q must be absolute", raw)
	}
	return nil
}
