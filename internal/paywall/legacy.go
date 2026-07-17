package paywall

import (
	"encoding/json"
	"fmt"
)

// This file exists ONLY for the one-time JSON→protobuf storage backfill
// (store.backfillProtoConfigs, migration 0019): it freezes the legacy JSON
// document shape paywall configs were stored in before moth.projectconfig.v1.
// Live code paths must never use it — Encode/Parse are the storage codec.
// Do not evolve these structs; the legacy shape is immutable by definition.

// legacyConfig mirrors the pre-0019 JSON document field-for-field.
type legacyConfig struct {
	Version               int         `json:"version"`
	Headline              string      `json:"headline"`
	Subtitle              string      `json:"subtitle,omitempty"`
	Benefits              []string    `json:"benefits,omitempty"`
	Offering              string      `json:"offering,omitempty"`
	HighlightedIdentifier string      `json:"highlightedIdentifier,omitempty"`
	Layout                string      `json:"layout"`
	Legal                 legacyLegal `json:"legal"`
}

type legacyLegal struct {
	TermsURL   string `json:"termsUrl,omitempty"`
	PrivacyURL string `json:"privacyUrl,omitempty"`
}

// ParseLegacyJSON decodes a pre-0019 JSON paywall document. BACKFILL ONLY:
// it is exported solely for the one-time store backfill and must not be
// called from any live read/write path.
func ParseLegacyJSON(raw []byte) (Config, error) {
	var l legacyConfig
	if err := json.Unmarshal(raw, &l); err != nil {
		return Config{}, fmt.Errorf("parse legacy paywall: %w", err)
	}
	if l.Version != SchemaVersion {
		return Config{}, fmt.Errorf("parse legacy paywall: unsupported schema version %d (want %d)", l.Version, SchemaVersion)
	}
	return Config{
		Version:               l.Version,
		Headline:              l.Headline,
		Subtitle:              l.Subtitle,
		Benefits:              l.Benefits,
		Offering:              l.Offering,
		HighlightedIdentifier: l.HighlightedIdentifier,
		Layout:                l.Layout,
		Legal:                 Legal(l.Legal),
	}, nil
}
