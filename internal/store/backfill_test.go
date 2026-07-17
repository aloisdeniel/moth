package store

import (
	"bytes"
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/aloisdeniel/moth/internal/paywall"
	"github.com/aloisdeniel/moth/internal/theme"
)

// The legacy JSON documents below are shaped exactly as the pre-0019 Encode
// produced them (encoding/json with the tags the domain structs carried), so
// the backfill tests exercise the real on-disk format.
const (
	legacyThemeJSON = `{"version":1,"colors":{"primary":"#0B57D0","onPrimary":"#FFFFFF","background":"#FFFBFE","onBackground":"#1C1B1F","surface":"#FFFBFE","onSurface":"#1C1B1F","error":"#B3261E","onError":"#FFFFFF"},"darkColors":{"primary":"#D0BCFF"},"typography":{"fontFamily":"Inter","scale":1},"spacing":{"unit":8},"shape":{"cornerRadius":12},"logo":{"light":"/assets/p1/logo-light.png"},"legal":{"termsUrl":"https://example.com/terms"}}`

	legacyPaywallJSON = `{"version":1,"headline":"Unlock Premium","subtitle":"The full experience.","benefits":["Unlimited access","Priority support"],"offering":"promo","highlightedIdentifier":"yearly","layout":"list","legal":{"termsUrl":"https://example.com/terms","privacyUrl":"https://example.com/privacy"}}`

	legacyCopyJSON = `{"de":{"sign_in.title":"Anmelden"},"fr":{"sign_in.title":"Connexion","sign_up.title":"Inscription"}}`
)

func wantBackfilledTheme() theme.Theme {
	return theme.Theme{
		Version: theme.SchemaVersion,
		Colors: theme.Colors{
			Primary: "#0B57D0", OnPrimary: "#FFFFFF",
			Background: "#FFFBFE", OnBackground: "#1C1B1F",
			Surface: "#FFFBFE", OnSurface: "#1C1B1F",
			Error: "#B3261E", OnError: "#FFFFFF",
		},
		DarkColors: &theme.ColorOverrides{Primary: "#D0BCFF"},
		Typography: theme.Typography{FontFamily: "Inter", Scale: 1.0},
		Spacing:    theme.Spacing{Unit: 8},
		Shape:      theme.Shape{CornerRadius: 12},
		Logo:       theme.Logo{Light: "/assets/p1/logo-light.png"},
		Legal:      theme.Legal{TermsURL: "https://example.com/terms"},
	}
}

func wantBackfilledPaywall() paywall.Config {
	return paywall.Config{
		Version:               paywall.SchemaVersion,
		Headline:              "Unlock Premium",
		Subtitle:              "The full experience.",
		Benefits:              []string{"Unlimited access", "Priority support"},
		Offering:              "promo",
		HighlightedIdentifier: "yearly",
		Layout:                paywall.LayoutList,
		Legal:                 paywall.Legal{TermsURL: "https://example.com/terms", PrivacyURL: "https://example.com/privacy"},
	}
}

// seedLegacyConfig writes a pre-0019 state for one config type with raw SQL:
// the legacy JSON in the frozen TEXT column of the project row and one
// revision row, with the *_pb BLOBs left empty as migration 0019 creates
// them.
func seedLegacyConfig(t *testing.T, s *Store, table, col, projectID, revisionID, doc string) {
	t.Helper()
	if _, err := s.db.Exec(
		`UPDATE projects SET `+col+` = ?, `+col+`_revision = ? WHERE id = ?`,
		doc, revisionID, projectID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(
		`INSERT INTO `+table+` (id, project_id, `+col+`, created_at) VALUES (?, ?, ?, ?)`,
		revisionID, projectID, doc, formatTime(time.Now().UTC())); err != nil {
		t.Fatal(err)
	}
}

// legacyColumn reads the frozen TEXT column back so tests can assert it was
// cleared (project row) or kept (unparseable rows).
func legacyColumn(t *testing.T, s *Store, table, col, where, id string) string {
	t.Helper()
	var v string
	if err := s.db.QueryRow(
		`SELECT `+col+` FROM `+table+` WHERE `+where+` = ?`, id).Scan(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestBackfillTheme(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	p, k := testProject("p1", "app-one")
	if err := s.CreateProject(ctx, p, k); err != nil {
		t.Fatal(err)
	}
	seedLegacyConfig(t, s, "theme_revisions", "theme", "p1", "rev-001", legacyThemeJSON)

	if err := s.backfillProtoConfigs(ctx); err != nil {
		t.Fatal(err)
	}

	got, err := s.GetProject(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := theme.Parse(got.Theme)
	if err != nil {
		t.Fatalf("backfilled project theme does not parse as proto: %v", err)
	}
	if want := wantBackfilledTheme(); !reflect.DeepEqual(parsed, want) {
		t.Errorf("backfilled theme mismatch: got %+v, want %+v", parsed, want)
	}
	if got.ThemeRevisionID != "rev-001" {
		t.Errorf("revision id changed by backfill: %q", got.ThemeRevisionID)
	}
	if v := legacyColumn(t, s, "projects", "theme", "id", "p1"); v != "" {
		t.Errorf("legacy projects.theme not cleared: %q", v)
	}

	// The revision survives and round-trips through the proto column too.
	rev, err := s.GetThemeRevision(ctx, "p1", "rev-001")
	if err != nil {
		t.Fatal(err)
	}
	revParsed, err := theme.Parse(rev.Theme)
	if err != nil {
		t.Fatalf("backfilled theme revision does not parse: %v", err)
	}
	if want := wantBackfilledTheme(); !reflect.DeepEqual(revParsed, want) {
		t.Errorf("backfilled revision mismatch: got %+v, want %+v", revParsed, want)
	}
	if v := legacyColumn(t, s, "theme_revisions", "theme", "id", "rev-001"); v != "" {
		t.Errorf("legacy theme_revisions.theme not cleared: %q", v)
	}

	// A second run is a no-op: the stored bytes do not change.
	if err := s.backfillProtoConfigs(ctx); err != nil {
		t.Fatal(err)
	}
	again, err := s.GetProject(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(again.Theme, got.Theme) {
		t.Error("second backfill run changed the stored theme bytes")
	}
}

func TestBackfillPaywall(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	p, k := testProject("p1", "app-one")
	if err := s.CreateProject(ctx, p, k); err != nil {
		t.Fatal(err)
	}
	seedLegacyConfig(t, s, "paywall_revisions", "paywall", "p1", "rev-001", legacyPaywallJSON)

	if err := s.backfillProtoConfigs(ctx); err != nil {
		t.Fatal(err)
	}

	got, err := s.GetProject(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := paywall.Parse(got.Paywall)
	if err != nil {
		t.Fatalf("backfilled project paywall does not parse as proto: %v", err)
	}
	if want := wantBackfilledPaywall(); !reflect.DeepEqual(parsed, want) {
		t.Errorf("backfilled paywall mismatch: got %+v, want %+v", parsed, want)
	}
	if v := legacyColumn(t, s, "projects", "paywall", "id", "p1"); v != "" {
		t.Errorf("legacy projects.paywall not cleared: %q", v)
	}

	rev, err := s.GetPaywallRevision(ctx, "p1", "rev-001")
	if err != nil {
		t.Fatal(err)
	}
	revParsed, err := paywall.Parse(rev.Paywall)
	if err != nil {
		t.Fatalf("backfilled paywall revision does not parse: %v", err)
	}
	if want := wantBackfilledPaywall(); !reflect.DeepEqual(revParsed, want) {
		t.Errorf("backfilled revision mismatch: got %+v, want %+v", revParsed, want)
	}
	if v := legacyColumn(t, s, "paywall_revisions", "paywall", "id", "rev-001"); v != "" {
		t.Errorf("legacy paywall_revisions.paywall not cleared: %q", v)
	}

	if err := s.backfillProtoConfigs(ctx); err != nil {
		t.Fatal(err)
	}
	again, err := s.GetProject(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(again.Paywall, got.Paywall) {
		t.Error("second backfill run changed the stored paywall bytes")
	}
}

func TestBackfillCopy(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	p, k := testProject("p1", "app-one")
	if err := s.CreateProject(ctx, p, k); err != nil {
		t.Fatal(err)
	}
	seedLegacyConfig(t, s, "copy_revisions", "copy", "p1", "rev-001", legacyCopyJSON)

	if err := s.backfillProtoConfigs(ctx); err != nil {
		t.Fatal(err)
	}

	want := CopyOverrides{
		"de": {"sign_in.title": "Anmelden"},
		"fr": {"sign_in.title": "Connexion", "sign_up.title": "Inscription"},
	}
	o, revID, err := s.GetProjectCopy(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(o, want) {
		t.Errorf("backfilled copy mismatch: got %v, want %v", o, want)
	}
	if revID != "rev-001" {
		t.Errorf("revision id changed by backfill: %q", revID)
	}
	if v := legacyColumn(t, s, "projects", "copy", "id", "p1"); v != "" {
		t.Errorf("legacy projects.copy not cleared: %q", v)
	}

	rev, err := s.GetCopyRevision(ctx, "p1", "rev-001")
	if err != nil {
		t.Fatal(err)
	}
	revParsed, err := parseCopyOverrides(rev.Copy)
	if err != nil {
		t.Fatalf("backfilled copy revision does not parse: %v", err)
	}
	if !reflect.DeepEqual(revParsed, want) {
		t.Errorf("backfilled revision mismatch: got %v, want %v", revParsed, want)
	}
	if v := legacyColumn(t, s, "copy_revisions", "copy", "id", "rev-001"); v != "" {
		t.Errorf("legacy copy_revisions.copy not cleared: %q", v)
	}

	before, err := s.GetProject(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.backfillProtoConfigs(ctx); err != nil {
		t.Fatal(err)
	}
	again, err := s.GetProject(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(again.Copy, before.Copy) {
		t.Error("second backfill run changed the stored copy bytes")
	}
}

func TestBackfillLeavesDefaultsAlone(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	p, k := testProject("p1", "app-one")
	if err := s.CreateProject(ctx, p, k); err != nil {
		t.Fatal(err)
	}

	// A never-customized project: every legacy column is '' and every proto
	// column empty; the backfill must not invent documents.
	if err := s.backfillProtoConfigs(ctx); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetProject(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Theme) != 0 || len(got.Paywall) != 0 || len(got.Copy) != 0 {
		t.Errorf("backfill must keep empty configs empty: theme=%d paywall=%d copy=%d bytes",
			len(got.Theme), len(got.Paywall), len(got.Copy))
	}
	if got.ThemeRevisionID != "" || got.PaywallRevisionID != "" || got.CopyRevisionID != "" {
		t.Errorf("backfill must not mint revisions: %q/%q/%q",
			got.ThemeRevisionID, got.PaywallRevisionID, got.CopyRevisionID)
	}
}

func TestBackfillKeepsUnparseableLegacyDocument(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	p, k := testProject("p1", "app-one")
	if err := s.CreateProject(ctx, p, k); err != nil {
		t.Fatal(err)
	}
	// A schema version this binary does not know: the backfill must neither
	// fail startup nor drop the document.
	future := `{"version":2,"colors":{"primary":"#123456"}}`
	if _, err := s.db.Exec(
		`UPDATE projects SET theme = ?, theme_revision = 'rev-f' WHERE id = 'p1'`, future); err != nil {
		t.Fatal(err)
	}

	if err := s.backfillProtoConfigs(ctx); err != nil {
		t.Fatal(err)
	}
	if v := legacyColumn(t, s, "projects", "theme", "id", "p1"); v != future {
		t.Errorf("unparseable legacy document was modified: %q", v)
	}
	got, err := s.GetProject(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Theme) != 0 {
		t.Errorf("proto column must stay empty for a skipped row, got %d bytes", len(got.Theme))
	}
}
