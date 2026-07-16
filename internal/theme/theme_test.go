package theme

import (
	"reflect"
	"strings"
	"testing"
)

func TestDefaultValidates(t *testing.T) {
	if err := Default().Validate(); err != nil {
		t.Fatalf("default theme must validate: %v", err)
	}
}

func TestJSONRoundTrip(t *testing.T) {
	in := Default()
	in.DarkColors = &ColorOverrides{Primary: "#D0BCFF", OnPrimary: "#381E72"}
	in.Logo = Logo{Light: "/assets/my-app/logo-light.png", Dark: "/assets/my-app/logo-dark.png"}
	in.Legal = Legal{TermsURL: "https://example.com/terms", PrivacyURL: "https://example.com/privacy"}

	raw, err := Encode(in)
	if err != nil {
		t.Fatal(err)
	}
	out, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Errorf("round trip mismatch:\n in: %+v\nout: %+v", in, out)
	}
}

func TestParsePlanExample(t *testing.T) {
	// The document shape promised in plan/06-design-system.md.
	raw := `{
	  "version": 1,
	  "colors": {
	    "primary": "#6750A4", "onPrimary": "#FFFFFF",
	    "background": "#FFFBFE", "onBackground": "#1C1B1F",
	    "surface": "#FFFBFE", "onSurface": "#1C1B1F",
	    "error": "#B3261E", "onError": "#FFFFFF"
	  },
	  "typography": { "fontFamily": "Inter", "scale": 1.0 },
	  "spacing": { "unit": 8 },
	  "shape": { "cornerRadius": 12 },
	  "logo": { "light": "/assets/my-app/logo-light.png" }
	}`
	th, err := Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if err := th.Validate(); err != nil {
		t.Fatal(err)
	}
	if th.Colors.Primary != "#6750A4" || th.Logo.Light != "/assets/my-app/logo-light.png" {
		t.Errorf("unexpected parse result: %+v", th)
	}
}

func TestParseRejectsVersions(t *testing.T) {
	for _, raw := range []string{
		`{"version": 2}`,
		`{"version": 0}`,
		`{}`,
		`not json`,
	} {
		if _, err := Parse([]byte(raw)); err == nil {
			t.Errorf("Parse(%q): want error", raw)
		}
	}
}

func TestValidateRejections(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Theme)
		wantErr string
	}{
		{
			name:    "wrong version",
			mutate:  func(th *Theme) { th.Version = 2 },
			wantErr: "schema version",
		},
		{
			name:    "invalid hex color",
			mutate:  func(th *Theme) { th.Colors.Primary = "purple" },
			wantErr: "colors.primary",
		},
		{
			name:    "shorthand hex rejected",
			mutate:  func(th *Theme) { th.Colors.Surface = "#FFF" },
			wantErr: "colors.surface",
		},
		{
			name:    "invalid dark override hex",
			mutate:  func(th *Theme) { th.DarkColors = &ColorOverrides{Background: "night"} },
			wantErr: "darkColors.background",
		},
		{
			name:    "light contrast too low",
			mutate:  func(th *Theme) { th.Colors.Primary = "#777777"; th.Colors.OnPrimary = "#FFFFFF" },
			wantErr: "primary/onPrimary contrast",
		},
		{
			name:    "background contrast too low",
			mutate:  func(th *Theme) { th.Colors.OnBackground = "#CCCCCC" },
			wantErr: "background/onBackground contrast",
		},
		{
			name: "dark override contrast too low",
			mutate: func(th *Theme) {
				th.DarkColors = &ColorOverrides{Surface: "#333333", OnSurface: "#555555"}
			},
			wantErr: "darkColors: surface/onSurface contrast",
		},
		{
			name:    "unknown font",
			mutate:  func(th *Theme) { th.Typography.FontFamily = "Comic Sans MS" },
			wantErr: "typography.fontFamily",
		},
		{
			name:    "scale too small",
			mutate:  func(th *Theme) { th.Typography.Scale = 0.5 },
			wantErr: "typography.scale",
		},
		{
			name:    "scale too large",
			mutate:  func(th *Theme) { th.Typography.Scale = 2.0 },
			wantErr: "typography.scale",
		},
		{
			name:    "spacing unit too small",
			mutate:  func(th *Theme) { th.Spacing.Unit = 1 },
			wantErr: "spacing.unit",
		},
		{
			name:    "spacing unit too large",
			mutate:  func(th *Theme) { th.Spacing.Unit = 64 },
			wantErr: "spacing.unit",
		},
		{
			name:    "negative corner radius",
			mutate:  func(th *Theme) { th.Shape.CornerRadius = -1 },
			wantErr: "shape.cornerRadius",
		},
		{
			name:    "corner radius too large",
			mutate:  func(th *Theme) { th.Shape.CornerRadius = 99 },
			wantErr: "shape.cornerRadius",
		},
		{
			name:    "unmanaged logo path",
			mutate:  func(th *Theme) { th.Logo.Light = "https://evil.example/logo.png" },
			wantErr: "logo.light",
		},
		{
			name:    "relative legal url",
			mutate:  func(th *Theme) { th.Legal.TermsURL = "/terms" },
			wantErr: "legal.termsUrl",
		},
		{
			name:    "non-http legal url",
			mutate:  func(th *Theme) { th.Legal.PrivacyURL = "javascript:alert(1)" },
			wantErr: "legal.privacyUrl",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			th := Default()
			tc.mutate(&th)
			err := th.Validate()
			if err == nil {
				t.Fatal("want error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error %q does not contain %q", err, tc.wantErr)
			}
		})
	}
}

func TestDeriveDarkDefaults(t *testing.T) {
	light := Default().Colors

	d1 := DeriveDark(light, nil)
	d2 := DeriveDark(light, &ColorOverrides{})
	if d1 != d2 {
		t.Errorf("nil and empty overrides must derive identically: %+v vs %+v", d1, d2)
	}
	// Derivation is deterministic.
	if d3 := DeriveDark(light, nil); d3 != d1 {
		t.Errorf("derivation not stable: %+v vs %+v", d3, d1)
	}
	// Surfaces land in the dark range, accents in the light range.
	for name, hex := range map[string]string{"background": d1.Background, "surface": d1.Surface} {
		if l := mustColor(t, hex).Luminance(); l > 0.1 {
			t.Errorf("derived dark %s %s luminance %.3f, want dark (<= 0.1)", name, hex, l)
		}
	}
	for name, hex := range map[string]string{"primary": d1.Primary, "error": d1.Error} {
		lightL := 0.15
		if l := mustColor(t, hex).Luminance(); l < lightL {
			t.Errorf("derived dark %s %s luminance %.3f, want lightened (>= %.2f)", name, hex, l, lightL)
		}
	}
	// Every derived pair meets AA (guaranteed by the black/white pick).
	if err := checkContrast("derived", d1); err != nil {
		t.Errorf("derived palette fails contrast: %v", err)
	}
}

func TestDeriveDarkDefaultConstants(t *testing.T) {
	// The exact palette derived from the default theme. The Flutter SDK
	// pins the same constants (sdk/flutter/test/theme_test.dart, 'matches
	// the server derivation byte-for-byte') against its own reimplementation
	// of the derivation, so any change to the algorithm must move both
	// files together — otherwise the SDK's offline fallback drifts from
	// what the server sends.
	got := DeriveDark(Default().Colors, nil)
	want := Colors{
		Primary:      "#A496C8",
		OnPrimary:    "#000000",
		Background:   "#1F1E1E",
		OnBackground: "#FFFFFF",
		Surface:      "#292829",
		OnSurface:    "#FFFFFF",
		Error:        "#D17D78",
		OnError:      "#000000",
	}
	if got != want {
		t.Errorf("DeriveDark(Default().Colors, nil):\n got %+v\nwant %+v\n(update sdk/flutter/test/theme_test.dart in the same change)", got, want)
	}
}

func TestDeriveDarkOverrides(t *testing.T) {
	light := Default().Colors
	ov := &ColorOverrides{Primary: "#D0BCFF", OnPrimary: "#381E72", Background: "#101014"}
	d := DeriveDark(light, ov)

	if d.Primary != "#D0BCFF" || d.OnPrimary != "#381E72" || d.Background != "#101014" {
		t.Errorf("explicit overrides must be used verbatim: %+v", d)
	}
	// Non-overridden fields still derive.
	derived := DeriveDark(light, nil)
	if d.Surface != derived.Surface || d.Error != derived.Error || d.OnError != derived.OnError {
		t.Errorf("non-overridden fields must match plain derivation: %+v vs %+v", d, derived)
	}
	// on* derivation follows an overridden counterpart.
	if d.OnBackground != bestOn("#101014") {
		t.Errorf("OnBackground = %s, want bestOn(#101014) = %s", d.OnBackground, bestOn("#101014"))
	}
}

func TestEffectiveDarkAlwaysAAWithoutOnOverrides(t *testing.T) {
	// Whatever the (valid) light palette, deriving without explicit on*
	// overrides must produce an AA-compliant dark palette.
	palettes := []Colors{
		Default().Colors,
		{
			Primary: "#0B57D0", OnPrimary: "#FFFFFF",
			Background: "#FFFFFF", OnBackground: "#000000",
			Surface: "#F2F6FC", OnSurface: "#001D35",
			Error: "#8C1D18", OnError: "#FFFFFF",
		},
		{
			Primary: "#004D40", OnPrimary: "#FFFFFF",
			Background: "#FAFDFB", OnBackground: "#191C1B",
			Surface: "#FAFDFB", OnSurface: "#191C1B",
			Error: "#BA1A1A", OnError: "#FFFFFF",
		},
	}
	for i, light := range palettes {
		if err := checkContrast("light", light); err != nil {
			t.Fatalf("palette %d: test input must be AA: %v", i, err)
		}
		if err := checkContrast("dark", DeriveDark(light, nil)); err != nil {
			t.Errorf("palette %d: derived dark palette fails AA: %v", i, err)
		}
	}
}
