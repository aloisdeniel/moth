package theme

import (
	"encoding/json"
	"fmt"
)

// This file exists ONLY for the one-time JSON→protobuf storage backfill
// (store.backfillProtoConfigs, migration 0019): it freezes the legacy JSON
// document shape themes were stored in before moth.projectconfig.v1. Live code
// paths must never use it — Encode/Parse are the storage codec. Do not
// evolve these structs; the legacy shape is immutable by definition.

// legacyTheme mirrors the pre-0019 JSON document field-for-field.
type legacyTheme struct {
	Version    int                   `json:"version"`
	Colors     legacyColors          `json:"colors"`
	DarkColors *legacyColorOverrides `json:"darkColors,omitempty"`
	Typography legacyTypography      `json:"typography"`
	Spacing    legacySpacing         `json:"spacing"`
	Shape      legacyShape           `json:"shape"`
	Logo       legacyLogo            `json:"logo"`
	Legal      legacyLegal           `json:"legal"`
}

type legacyColors struct {
	Primary      string `json:"primary"`
	OnPrimary    string `json:"onPrimary"`
	Background   string `json:"background"`
	OnBackground string `json:"onBackground"`
	Surface      string `json:"surface"`
	OnSurface    string `json:"onSurface"`
	Error        string `json:"error"`
	OnError      string `json:"onError"`
}

type legacyColorOverrides struct {
	Primary      string `json:"primary,omitempty"`
	OnPrimary    string `json:"onPrimary,omitempty"`
	Background   string `json:"background,omitempty"`
	OnBackground string `json:"onBackground,omitempty"`
	Surface      string `json:"surface,omitempty"`
	OnSurface    string `json:"onSurface,omitempty"`
	Error        string `json:"error,omitempty"`
	OnError      string `json:"onError,omitempty"`
}

type legacyTypography struct {
	FontFamily string  `json:"fontFamily"`
	Scale      float64 `json:"scale"`
}

type legacySpacing struct {
	Unit int `json:"unit"`
}

type legacyShape struct {
	CornerRadius int `json:"cornerRadius"`
}

type legacyLogo struct {
	Light string `json:"light,omitempty"`
	Dark  string `json:"dark,omitempty"`
}

type legacyLegal struct {
	TermsURL   string `json:"termsUrl,omitempty"`
	PrivacyURL string `json:"privacyUrl,omitempty"`
}

// ParseLegacyJSON decodes a pre-0019 JSON theme document. BACKFILL ONLY:
// it is exported solely for the one-time store backfill and must not be
// called from any live read/write path.
func ParseLegacyJSON(raw []byte) (Theme, error) {
	var l legacyTheme
	if err := json.Unmarshal(raw, &l); err != nil {
		return Theme{}, fmt.Errorf("parse legacy theme: %w", err)
	}
	if l.Version != SchemaVersion {
		return Theme{}, fmt.Errorf("parse legacy theme: unsupported schema version %d (want %d)", l.Version, SchemaVersion)
	}
	t := Theme{
		Version:    l.Version,
		Colors:     Colors(l.Colors),
		Typography: Typography(l.Typography),
		Spacing:    Spacing(l.Spacing),
		Shape:      Shape(l.Shape),
		Logo:       Logo(l.Logo),
		Legal:      Legal(l.Legal),
	}
	if l.DarkColors != nil {
		ov := ColorOverrides(*l.DarkColors)
		t.DarkColors = &ov
	}
	return t, nil
}
