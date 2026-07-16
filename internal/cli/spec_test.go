package cli

import (
	"slices"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
)

func sampleSettings() *adminv1.ProjectSettings {
	autoLink := true
	return &adminv1.ProjectSettings{
		PasswordMinLength:      8,
		AllowPublicSignup:      true,
		AccessTokenTtlSeconds:  900,
		RefreshTokenTtlDays:    30,
		AnalyticsRetentionDays: 90,
		RollupTimezone:         "UTC",
		Google: &adminv1.GoogleProviderConfig{
			Enabled: true, WebClientId: "web.apps.example", HasWebClientSecret: true,
		},
		Apple:                 &adminv1.AppleProviderConfig{},
		AutoLinkVerifiedEmail: &autoLink,
		RedirectSchemes:       []string{"demoapp"},
	}
}

func sampleProject() *adminv1.Project {
	return &adminv1.Project{
		Id: "p1", Name: "Demo", Slug: "demo", Settings: sampleSettings(),
	}
}

func TestSpecYAMLRoundTrip(t *testing.T) {
	spec := &adminv1.ProjectSpec{
		Name:     "Demo",
		Slug:     "demo",
		Settings: sampleSettings(),
		Theme: &adminv1.Theme{
			Colors: &adminv1.ThemeColors{Primary: "#3355FF", OnPrimary: "#FFFFFF"},
			Shape:  &adminv1.ThemeShape{CornerRadius: 12},
		},
	}
	data, err := SpecToYAML(spec)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "slug: demo") {
		t.Fatalf("YAML should use proto field names:\n%s", data)
	}
	back, err := SpecFromYAML(data)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(spec, back) {
		t.Fatalf("round trip mismatch:\n in: %v\nout: %v", spec, back)
	}
}

func TestSpecFromYAMLRejectsUnknownFields(t *testing.T) {
	if _, err := SpecFromYAML([]byte("name: Demo\nslugg: demo\n")); err == nil {
		t.Fatal("typoed field should be rejected")
	}
}

func TestPlanApply(t *testing.T) {
	defaultTheme := &adminv1.GetThemeResponse{Theme: &adminv1.Theme{}, IsDefault: true}
	customTheme := &adminv1.GetThemeResponse{
		Theme: &adminv1.Theme{
			Colors: &adminv1.ThemeColors{Primary: "#112233"},
			Logo:   &adminv1.ThemeLogo{LightPath: "/assets/p1/logo-light.png"},
		},
	}
	// serverShapedTheme mirrors what GetTheme actually returns: every
	// sub-message present, the optional ones empty (themeProto builds them
	// unconditionally).
	serverShapedTheme := &adminv1.GetThemeResponse{
		Theme: &adminv1.Theme{
			Colors:     &adminv1.ThemeColors{Primary: "#112233"},
			Typography: &adminv1.ThemeTypography{},
			Spacing:    &adminv1.ThemeSpacing{},
			Shape:      &adminv1.ThemeShape{},
			Logo:       &adminv1.ThemeLogo{},
			Legal:      &adminv1.ThemeLegal{},
		},
	}

	cases := []struct {
		name    string
		spec    *adminv1.ProjectSpec
		current *adminv1.Project
		theme   *adminv1.GetThemeResponse
		want    ApplyPlan
		wantErr bool
	}{
		{
			name:    "missing slug",
			spec:    &adminv1.ProjectSpec{Name: "Demo"},
			wantErr: true,
		},
		{
			name:    "missing name",
			spec:    &adminv1.ProjectSpec{Slug: "demo"},
			wantErr: true,
		},
		{
			name: "create when slug is free",
			spec: &adminv1.ProjectSpec{Name: "Demo", Slug: "demo", Settings: sampleSettings()},
			want: ApplyPlan{Slug: "demo", Create: true, UpdateSettings: true},
		},
		{
			name:    "identical dump is a no-op",
			spec:    &adminv1.ProjectSpec{Name: "Demo", Slug: "demo", Settings: sampleSettings()},
			current: sampleProject(),
			theme:   defaultTheme,
			want:    ApplyPlan{Slug: "demo"},
		},
		{
			// The partial spec omits redirect_schemes entirely: the merge
			// must keep the registered OAuth redirect schemes, or every
			// mobile social sign-in breaks at the callback.
			name:    "partial spec keeps server values",
			spec:    &adminv1.ProjectSpec{Name: "Demo", Slug: "demo", Settings: &adminv1.ProjectSettings{AllowPublicSignup: true}},
			current: sampleProject(),
			theme:   defaultTheme,
			want:    ApplyPlan{Slug: "demo"},
		},
		{
			name:    "name change",
			spec:    &adminv1.ProjectSpec{Name: "Demo v2", Slug: "demo", Settings: sampleSettings()},
			current: sampleProject(),
			theme:   defaultTheme,
			want:    ApplyPlan{Slug: "demo", UpdateName: true},
		},
		{
			name: "settings change",
			spec: func() *adminv1.ProjectSpec {
				s := sampleSettings()
				s.RequireEmailVerification = true
				return &adminv1.ProjectSpec{Name: "Demo", Slug: "demo", Settings: s}
			}(),
			current: sampleProject(),
			theme:   defaultTheme,
			want: ApplyPlan{Slug: "demo", UpdateSettings: true,
				SettingsChanges: []string{"require_email_verification"}},
		},
		{
			name: "write-only secret always applies",
			spec: func() *adminv1.ProjectSpec {
				s := sampleSettings()
				s.Google.WebClientSecret = "shh"
				return &adminv1.ProjectSpec{Name: "Demo", Slug: "demo", Settings: s}
			}(),
			current: sampleProject(),
			theme:   defaultTheme,
			want: ApplyPlan{Slug: "demo", UpdateSettings: true,
				Notes: []string{"write Google web client secret (write-only: re-sent on every apply)"}},
		},
		{
			name: "theme install",
			spec: &adminv1.ProjectSpec{Name: "Demo", Slug: "demo", Settings: sampleSettings(),
				Theme: &adminv1.Theme{Colors: &adminv1.ThemeColors{Primary: "#112233"}}},
			current: sampleProject(),
			theme:   defaultTheme,
			want:    ApplyPlan{Slug: "demo", UpdateTheme: true},
		},
		{
			name: "same theme modulo logo is a no-op",
			spec: &adminv1.ProjectSpec{Name: "Demo", Slug: "demo", Settings: sampleSettings(),
				Theme: &adminv1.Theme{Colors: &adminv1.ThemeColors{Primary: "#112233"}}},
			current: sampleProject(),
			theme:   customTheme,
			want:    ApplyPlan{Slug: "demo"},
		},
		{
			name:    "absent theme resets a customized project",
			spec:    &adminv1.ProjectSpec{Name: "Demo", Slug: "demo", Settings: sampleSettings()},
			current: sampleProject(),
			theme:   customTheme,
			want:    ApplyPlan{Slug: "demo", ResetTheme: true},
		},
		{
			// GetTheme responses always carry non-nil (possibly empty)
			// sub-messages; a hand-written spec omits the optional ones
			// (legal at minimum). "Absent" must equal "empty" or apply
			// re-writes the theme — a new revision — on every run.
			name: "spec omitting empty theme sections is a no-op",
			spec: &adminv1.ProjectSpec{Name: "Demo", Slug: "demo", Settings: sampleSettings(),
				Theme: &adminv1.Theme{Colors: &adminv1.ThemeColors{Primary: "#112233"}}},
			current: sampleProject(),
			theme:   serverShapedTheme,
			want:    ApplyPlan{Slug: "demo"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan, _, err := PlanApply(tc.spec, tc.current, tc.theme)
			if tc.wantErr {
				if err == nil {
					t.Fatal("want error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if plan.Slug != tc.want.Slug || plan.Create != tc.want.Create ||
				plan.UpdateName != tc.want.UpdateName || plan.UpdateSettings != tc.want.UpdateSettings ||
				plan.UpdateTheme != tc.want.UpdateTheme || plan.ResetTheme != tc.want.ResetTheme {
				t.Fatalf("plan = %+v, want %+v", plan, tc.want)
			}
			if len(tc.want.Notes) > 0 && (len(plan.Notes) != len(tc.want.Notes) || plan.Notes[0] != tc.want.Notes[0]) {
				t.Fatalf("notes = %v, want %v", plan.Notes, tc.want.Notes)
			}
			if len(tc.want.SettingsChanges) > 0 && !slices.Equal(plan.SettingsChanges, tc.want.SettingsChanges) {
				t.Fatalf("settings changes = %v, want %v", plan.SettingsChanges, tc.want.SettingsChanges)
			}
		})
	}
}

func TestMergeSettingsSendsFullObject(t *testing.T) {
	current := sampleSettings()
	desired := &adminv1.ProjectSettings{RequireEmailVerification: true}
	merged := MergeSettings(current, desired)

	if !merged.RequireEmailVerification {
		t.Fatal("desired field lost")
	}
	if !slices.Equal(merged.RedirectSchemes, []string{"demoapp"}) {
		t.Fatalf("absent redirect_schemes should keep the server list, got %v", merged.RedirectSchemes)
	}
	// ... without aliasing the current message.
	merged.RedirectSchemes[0] = "changed"
	if current.RedirectSchemes[0] != "demoapp" {
		t.Fatal("merge aliased the current redirect schemes")
	}
	if merged.PasswordMinLength != 8 || merged.AccessTokenTtlSeconds != 900 ||
		merged.RefreshTokenTtlDays != 30 || merged.AnalyticsRetentionDays != 90 ||
		merged.RollupTimezone != "UTC" {
		t.Fatalf("unset numerics should keep server values: %+v", merged)
	}
	if merged.AutoLinkVerifiedEmail == nil || !*merged.AutoLinkVerifiedEmail {
		t.Fatal("unset optional bool should keep the server value")
	}
	if merged.Google.GetWebClientId() != "web.apps.example" {
		t.Fatal("absent google section should keep the server config")
	}
	// The merge must not alias the current message.
	merged.Google.WebClientId = "changed"
	if current.Google.WebClientId != "web.apps.example" {
		t.Fatal("merge aliased the current settings")
	}
}
