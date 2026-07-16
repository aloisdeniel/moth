package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
)

// SpecToYAML serializes a ProjectSpec as the YAML document `moth project
// dump` emits and `moth project apply -f` consumes. The field names are the
// proto ones (snake_case); the conversion goes through protojson so the
// document stays a faithful, versionable rendering of the proto message.
func SpecToYAML(spec *adminv1.ProjectSpec) ([]byte, error) {
	raw, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(spec)
	if err != nil {
		return nil, err
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	return yaml.Marshal(doc)
}

// SpecFromYAML parses a `moth project dump` document (unknown fields are
// rejected, so typos fail loudly instead of silently applying nothing).
func SpecFromYAML(data []byte) (*adminv1.ProjectSpec, error) {
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	spec := &adminv1.ProjectSpec{}
	if err := protojson.Unmarshal(raw, spec); err != nil {
		return nil, fmt.Errorf("invalid project spec: %w", err)
	}
	return spec, nil
}

// ApplyPlan lists what one `moth project apply` run will change. An empty
// plan is the idempotency signal: the live state already matches the spec.
type ApplyPlan struct {
	Slug           string `json:"slug"`
	Create         bool   `json:"create"`
	UpdateName     bool   `json:"update_name"`
	UpdateSettings bool   `json:"update_settings"`
	UpdateTheme    bool   `json:"update_theme"`
	ResetTheme     bool   `json:"reset_theme"`
	// SettingsChanges names the settings fields an update will change
	// (proto field names, in field order), so an accidental boolean reset
	// is visible in the plan before it is applied.
	SettingsChanges []string `json:"settings_changes,omitempty"`
	// Notes flags operations the diff cannot verify, e.g. write-only
	// provider secrets that are re-sent on every apply.
	Notes []string `json:"notes,omitempty"`
}

// Empty reports whether the apply is a no-op.
func (p ApplyPlan) Empty() bool {
	return !p.Create && !p.UpdateName && !p.UpdateSettings && !p.UpdateTheme && !p.ResetTheme
}

// Summary renders the plan as human lines ("create project", ...).
func (p ApplyPlan) Summary() []string {
	var lines []string
	if p.Create {
		lines = append(lines, "create project "+p.Slug)
	}
	if p.UpdateName {
		lines = append(lines, "update name")
	}
	if p.UpdateSettings {
		line := "update settings"
		if len(p.SettingsChanges) > 0 {
			line += " (" + strings.Join(p.SettingsChanges, ", ") + ")"
		}
		lines = append(lines, line)
	}
	if p.UpdateTheme {
		lines = append(lines, "update theme")
	}
	if p.ResetTheme {
		lines = append(lines, "reset theme to the built-in default")
	}
	lines = append(lines, p.Notes...)
	return lines
}

// PlanApply diffs a spec against the live state (current == nil when no
// project has the spec's slug; theme is the project's current theme, nil on
// create) and returns the plan plus the settings message an update must
// send. The sent settings are the spec's merged over the current ones —
// zero numeric fields, an empty timezone, an absent redirect_schemes list
// and absent sub-messages mean "keep what the server has" — so partial
// hand-written specs stay idempotent and never clobber unrelated fields.
// Plain proto3 booleans are the exception (omitted means false); the
// plan's SettingsChanges lists every field about to change.
func PlanApply(spec *adminv1.ProjectSpec, current *adminv1.Project, theme *adminv1.GetThemeResponse) (ApplyPlan, *adminv1.ProjectSettings, error) {
	if spec == nil {
		return ApplyPlan{}, nil, errors.New("empty spec")
	}
	if spec.Slug == "" {
		return ApplyPlan{}, nil, errors.New("spec is missing the slug (the identity apply keys on)")
	}
	if spec.Name == "" {
		return ApplyPlan{}, nil, errors.New("spec is missing the project name")
	}
	plan := ApplyPlan{Slug: spec.Slug}

	if current == nil {
		plan.Create = true
		plan.UpdateSettings = spec.Settings != nil
		plan.UpdateTheme = spec.Theme != nil
		if notes := secretNotes(spec.Settings); len(notes) > 0 {
			plan.Notes = notes
		}
		return plan, spec.Settings, nil
	}

	plan.UpdateName = spec.Name != current.Name

	var send *adminv1.ProjectSettings
	if spec.Settings != nil {
		send = MergeSettings(current.Settings, spec.Settings)
		if diff := settingsDiff(normalizeSettings(current.Settings), normalizeSettings(send)); len(diff) > 0 {
			plan.UpdateSettings = true
			plan.SettingsChanges = diff
		}
		if notes := secretNotes(send); len(notes) > 0 {
			// Write-only secrets never come back on reads, so the diff cannot
			// prove them unchanged; they are (re)written on every apply.
			plan.Notes = notes
			plan.UpdateSettings = true
		}
	}

	if spec.Theme == nil {
		// An absent theme in the spec means the built-in default.
		plan.ResetTheme = theme != nil && !theme.IsDefault
	} else if theme == nil || !proto.Equal(normalizeTheme(theme.Theme), normalizeTheme(spec.Theme)) {
		plan.UpdateTheme = true
	}
	return plan, send, nil
}

// MergeSettings returns desired with its "unset" fields (zero numerics,
// empty timezone, nil optional/sub-messages, an absent redirect_schemes
// list) filled from current, i.e. the full settings object an
// UpdateProject must send. proto3 booleans cannot express "unset", so
// false always means false — the plan's SettingsChanges makes an
// accidental reset visible.
func MergeSettings(current, desired *adminv1.ProjectSettings) *adminv1.ProjectSettings {
	merged, _ := proto.Clone(desired).(*adminv1.ProjectSettings)
	if merged == nil {
		merged = &adminv1.ProjectSettings{}
	}
	if current == nil {
		return merged
	}
	if merged.PasswordMinLength == 0 {
		merged.PasswordMinLength = current.PasswordMinLength
	}
	if merged.AccessTokenTtlSeconds == 0 {
		merged.AccessTokenTtlSeconds = current.AccessTokenTtlSeconds
	}
	if merged.RefreshTokenTtlDays == 0 {
		merged.RefreshTokenTtlDays = current.RefreshTokenTtlDays
	}
	if merged.AnalyticsRetentionDays == 0 {
		merged.AnalyticsRetentionDays = current.AnalyticsRetentionDays
	}
	if merged.RollupTimezone == "" {
		merged.RollupTimezone = current.RollupTimezone
	}
	if merged.AutoLinkVerifiedEmail == nil {
		merged.AutoLinkVerifiedEmail = current.AutoLinkVerifiedEmail
	}
	if merged.RedirectSchemes == nil {
		// An absent list keeps the registered OAuth redirect schemes; a
		// partial spec must never silently break mobile social sign-in.
		merged.RedirectSchemes = slices.Clone(current.RedirectSchemes)
	}
	if merged.Google == nil {
		merged.Google, _ = proto.Clone(current.Google).(*adminv1.GoogleProviderConfig)
	}
	if merged.Apple == nil {
		merged.Apple, _ = proto.Clone(current.Apple).(*adminv1.AppleProviderConfig)
	}
	return merged
}

// settingsDiff lists the proto field names on which a and b differ, in
// field order — the plan detail behind "update settings (...)".
func settingsDiff(a, b *adminv1.ProjectSettings) []string {
	ar, br := a.ProtoReflect(), b.ProtoReflect()
	fields := ar.Descriptor().Fields()
	var changed []string
	for i := range fields.Len() {
		fd := fields.Get(i)
		if !ar.Get(fd).Equal(br.Get(fd)) {
			changed = append(changed, string(fd.Name()))
		}
	}
	return changed
}

// normalizeSettings strips what a settings diff must ignore: output-only
// presence flags and write-only secrets (never returned by reads), and
// normalizes nil sub-messages to empty ones so "absent" equals "empty".
func normalizeSettings(s *adminv1.ProjectSettings) *adminv1.ProjectSettings {
	out, _ := proto.Clone(s).(*adminv1.ProjectSettings)
	if out == nil {
		out = &adminv1.ProjectSettings{}
	}
	if out.Google == nil {
		out.Google = &adminv1.GoogleProviderConfig{}
	}
	out.Google.WebClientSecret = ""
	out.Google.HasWebClientSecret = false
	if out.Apple == nil {
		out.Apple = &adminv1.AppleProviderConfig{}
	}
	out.Apple.PrivateKeyP8 = ""
	out.Apple.HasPrivateKey = false
	return out
}

// normalizeTheme strips the output-only logo asset paths (managed through
// UploadLogo/DeleteLogo, ignored by UpdateTheme) and normalizes nil
// sub-messages to empty ones: server reads always carry non-nil (possibly
// empty) sub-messages, while hand-written specs omit the optional
// sections — "absent" must equal "empty" or apply never converges.
func normalizeTheme(t *adminv1.Theme) *adminv1.Theme {
	out, _ := proto.Clone(t).(*adminv1.Theme)
	if out == nil {
		out = &adminv1.Theme{}
	}
	out.Logo = nil
	if out.Colors == nil {
		out.Colors = &adminv1.ThemeColors{}
	}
	if out.DarkColors == nil {
		out.DarkColors = &adminv1.ThemeColorOverrides{}
	}
	if out.Typography == nil {
		out.Typography = &adminv1.ThemeTypography{}
	}
	if out.Spacing == nil {
		out.Spacing = &adminv1.ThemeSpacing{}
	}
	if out.Shape == nil {
		out.Shape = &adminv1.ThemeShape{}
	}
	if out.Legal == nil {
		out.Legal = &adminv1.ThemeLegal{}
	}
	return out
}

// --- Monetization catalog -------------------------------------------------

// MonetizationPlan lists the catalog changes one apply will make, by stable
// identifier (entitlement/product identifiers, not server ids). An empty plan
// is the idempotency signal: the live catalog already matches the spec. Unlike
// settings (which merge partial specs), the catalog is full desired state —
// entitlements/products absent from the spec are deleted — so a dump re-applied
// converges to an empty plan.
type MonetizationPlan struct {
	CreateEntitlements []string `json:"create_entitlements,omitempty"`
	UpdateEntitlements []string `json:"update_entitlements,omitempty"`
	DeleteEntitlements []string `json:"delete_entitlements,omitempty"`
	CreateProducts     []string `json:"create_products,omitempty"`
	UpdateProducts     []string `json:"update_products,omitempty"`
	DeleteProducts     []string `json:"delete_products,omitempty"`
}

// Empty reports whether the monetization apply is a no-op.
func (p MonetizationPlan) Empty() bool {
	return len(p.CreateEntitlements) == 0 && len(p.UpdateEntitlements) == 0 &&
		len(p.DeleteEntitlements) == 0 && len(p.CreateProducts) == 0 &&
		len(p.UpdateProducts) == 0 && len(p.DeleteProducts) == 0
}

// Summary renders the plan as human lines.
func (p MonetizationPlan) Summary() []string {
	var lines []string
	add := func(verb, kind string, ids []string) {
		if len(ids) > 0 {
			lines = append(lines, fmt.Sprintf("%s %s: %s", verb, kind, strings.Join(ids, ", ")))
		}
	}
	add("create", "entitlements", p.CreateEntitlements)
	add("update", "entitlements", p.UpdateEntitlements)
	add("delete", "entitlements", p.DeleteEntitlements)
	add("create", "products", p.CreateProducts)
	add("update", "products", p.UpdateProducts)
	add("delete", "products", p.DeleteProducts)
	return lines
}

// PlanMonetization diffs a monetization spec against the live catalog (the
// project's current entitlements and products, as the admin services return
// them) and returns the create/update/delete plan, keyed on identifiers. A nil
// spec means the spec omitted the monetization block entirely and the catalog
// is left untouched (empty plan) — distinct from an explicit empty catalog,
// which deletes everything. Product entitlement grants are compared by
// entitlement identifier (resolved through currentEntitlements) so the diff is
// stable across the id/identifier boundary.
func PlanMonetization(spec *adminv1.MonetizationSpec, currentEntitlements []*adminv1.Entitlement, currentProducts []*adminv1.Product) MonetizationPlan {
	var plan MonetizationPlan
	if spec == nil {
		return plan
	}

	// Entitlements, matched by identifier.
	curEnt := map[string]*adminv1.Entitlement{}
	for _, e := range currentEntitlements {
		curEnt[e.GetIdentifier()] = e
	}
	specEnt := map[string]bool{}
	for _, e := range spec.GetEntitlements() {
		id := e.GetIdentifier()
		specEnt[id] = true
		cur, ok := curEnt[id]
		switch {
		case !ok:
			plan.CreateEntitlements = append(plan.CreateEntitlements, id)
		case cur.GetDisplayName() != e.GetDisplayName():
			plan.UpdateEntitlements = append(plan.UpdateEntitlements, id)
		}
	}
	for _, e := range currentEntitlements {
		if !specEnt[e.GetIdentifier()] {
			plan.DeleteEntitlements = append(plan.DeleteEntitlements, e.GetIdentifier())
		}
	}

	// entitlement id -> identifier, to compare product grants stably.
	entIDToIdent := map[string]string{}
	for _, e := range currentEntitlements {
		entIDToIdent[e.GetId()] = e.GetIdentifier()
	}

	// Products, matched by identifier.
	curProd := map[string]*adminv1.Product{}
	for _, p := range currentProducts {
		curProd[p.GetIdentifier()] = p
	}
	specProd := map[string]bool{}
	for _, p := range spec.GetProducts() {
		id := p.GetIdentifier()
		specProd[id] = true
		cur, ok := curProd[id]
		switch {
		case !ok:
			plan.CreateProducts = append(plan.CreateProducts, id)
		case productSpecDiffers(p, cur, entIDToIdent):
			plan.UpdateProducts = append(plan.UpdateProducts, id)
		}
	}
	for _, p := range currentProducts {
		if !specProd[p.GetIdentifier()] {
			plan.DeleteProducts = append(plan.DeleteProducts, p.GetIdentifier())
		}
	}
	return plan
}

// productSpecDiffers reports whether the desired product spec differs from the
// live product. Grants are compared as sorted sets of entitlement identifiers;
// an empty spec offering equals the "default" tag the store applies.
func productSpecDiffers(spec *adminv1.ProductSpec, cur *adminv1.Product, entIDToIdent map[string]string) bool {
	if spec.GetDisplayName() != cur.GetDisplayName() ||
		spec.GetAppleProductId() != cur.GetAppleProductId() ||
		spec.GetGoogleProductId() != cur.GetGoogleProductId() ||
		spec.GetBillingPeriod() != cur.GetBillingPeriod() ||
		spec.GetPriceAmountMicros() != cur.GetPriceAmountMicros() ||
		spec.GetCurrency() != cur.GetCurrency() ||
		spec.GetTrialPeriod() != cur.GetTrialPeriod() ||
		spec.GetIntroPriceAmountMicros() != cur.GetIntroPriceAmountMicros() ||
		spec.GetIntroPeriod() != cur.GetIntroPeriod() ||
		specOffering(spec.GetOffering()) != specOffering(cur.GetOffering()) ||
		spec.GetSortOrder() != cur.GetSortOrder() {
		return true
	}
	want := slices.Clone(spec.GetEntitlements())
	var have []string
	for _, eid := range cur.GetEntitlementIds() {
		if ident, ok := entIDToIdent[eid]; ok {
			have = append(have, ident)
		} else {
			have = append(have, eid)
		}
	}
	slices.Sort(want)
	slices.Sort(have)
	return !slices.Equal(want, have)
}

// specOffering normalizes an empty offering tag to the default, matching the
// store's behavior so a hand-written spec that omits `offering` stays in sync
// with a product the server tagged "default".
func specOffering(tag string) string {
	if tag == "" {
		return "default"
	}
	return tag
}

// MonetizationSpecFromCatalog renders the live entitlements and products as a
// MonetizationSpec — the block `moth project dump` emits so a re-applied dump
// converges to an empty plan. Product entitlement grants are emitted as stable
// entitlement identifiers (not server ids). A project with no catalog yields a
// non-nil empty spec so the round-trip stays lossless (absent == untouched;
// empty == an explicitly empty catalog, which for an already-empty project is a
// no-op).
func MonetizationSpecFromCatalog(ents []*adminv1.Entitlement, prods []*adminv1.Product) *adminv1.MonetizationSpec {
	spec := &adminv1.MonetizationSpec{}
	idToIdent := map[string]string{}
	for _, e := range ents {
		idToIdent[e.GetId()] = e.GetIdentifier()
		spec.Entitlements = append(spec.Entitlements, &adminv1.EntitlementSpec{
			Identifier:  e.GetIdentifier(),
			DisplayName: e.GetDisplayName(),
		})
	}
	for _, p := range prods {
		grants := make([]string, 0, len(p.GetEntitlementIds()))
		for _, eid := range p.GetEntitlementIds() {
			if ident, ok := idToIdent[eid]; ok {
				grants = append(grants, ident)
			} else {
				grants = append(grants, eid)
			}
		}
		slices.Sort(grants)
		spec.Products = append(spec.Products, &adminv1.ProductSpec{
			Identifier:             p.GetIdentifier(),
			DisplayName:            p.GetDisplayName(),
			AppleProductId:         p.GetAppleProductId(),
			GoogleProductId:        p.GetGoogleProductId(),
			BillingPeriod:          p.GetBillingPeriod(),
			PriceAmountMicros:      p.GetPriceAmountMicros(),
			Currency:               p.GetCurrency(),
			TrialPeriod:            p.GetTrialPeriod(),
			IntroPriceAmountMicros: p.GetIntroPriceAmountMicros(),
			IntroPeriod:            p.GetIntroPeriod(),
			Offering:               p.GetOffering(),
			SortOrder:              p.GetSortOrder(),
			Entitlements:           grants,
		})
	}
	return spec
}

// secretNotes describes the write-only secrets a settings message carries.
func secretNotes(s *adminv1.ProjectSettings) []string {
	if s == nil {
		return nil
	}
	var notes []string
	if s.Google.GetWebClientSecret() != "" {
		notes = append(notes, "write Google web client secret (write-only: re-sent on every apply)")
	}
	if s.Apple.GetPrivateKeyP8() != "" {
		notes = append(notes, "write Apple private key (write-only: re-sent on every apply)")
	}
	return notes
}
