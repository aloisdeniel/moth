package skill

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func read(t *testing.T, dir, rel string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(raw)
}

func export(t *testing.T, format Format, v Values) (string, []string) {
	t.Helper()
	dir := t.TempDir()
	files, err := Export(dir, format, v)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	return dir, files
}

// realValues is a fully interpolated fixture.
func realValues() Values {
	return Values{
		Endpoint:       "https://auth.acme.dev",
		ProjectName:    "Acme App",
		Slug:           "acme-app",
		PublishableKey: "pk_live_abc123",
		JWKSURL:        "https://auth.acme.dev/p/acme-app/.well-known/jwks.json",
		Issuer:         "https://auth.acme.dev/p/acme-app",
		Audience:       "acme-app",
		SDKVersion:     "^1.2.3",
		Google: GoogleValues{
			Enabled:         true,
			WebClientID:     "web-1.apps.googleusercontent.com",
			IOSClientID:     "ios-1.apps.googleusercontent.com",
			AndroidClientID: "android-1.apps.googleusercontent.com",
		},
		Apple: AppleValues{
			Enabled:    true,
			ServicesID: "dev.acme.app.signin",
			BundleIDs:  []string{"dev.acme.app"},
		},
		Interpolated: true,
	}
}

// frontmatter of a SKILL.md; yaml tags match the Agent Skills format.
type frontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// parseFrontmatter splits and decodes the leading YAML block.
func parseFrontmatter(t *testing.T, doc string) frontmatter {
	t.Helper()
	if !strings.HasPrefix(doc, "---\n") {
		t.Fatalf("document does not start with a frontmatter block:\n%.120s", doc)
	}
	rest := doc[len("---\n"):]
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		t.Fatal("frontmatter block is not terminated")
	}
	var fm frontmatter
	if err := yaml.Unmarshal([]byte(rest[:end]), &fm); err != nil {
		t.Fatalf("frontmatter is not valid YAML: %v", err)
	}
	return fm
}

func TestExportClaudeWritesSkillWithValidFrontmatter(t *testing.T) {
	dir, files := export(t, FormatClaude, Placeholders())

	want := []string{
		"SKILL.md",
		"references/flutter-integration.md",
		"references/backend-verification.md",
		"references/provider-setup.md",
		"references/cli-administration.md",
	}
	if len(files) != len(want) {
		t.Fatalf("written files = %v, want %v", files, want)
	}
	for i, w := range want {
		if files[i] != w {
			t.Errorf("files[%d] = %q, want %q", i, files[i], w)
		}
	}

	fm := parseFrontmatter(t, read(t, dir, "SKILL.md"))
	if fm.Name == "" || fm.Description == "" {
		t.Fatalf("frontmatter must set name and description, got %+v", fm)
	}
	// Agent Skills format: lowercase letters, digits and hyphens, <= 64
	// chars; description <= 1024 chars.
	if !regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`).MatchString(fm.Name) || len(fm.Name) > 64 {
		t.Errorf("invalid skill name %q", fm.Name)
	}
	if len(fm.Description) > 1024 {
		t.Errorf("description is %d chars, max 1024", len(fm.Description))
	}
}

func TestExportLeavesNoTemplateResidue(t *testing.T) {
	for _, tc := range []struct {
		name   string
		format Format
		values Values
	}{
		{"claude placeholders", FormatClaude, Placeholders()},
		{"claude interpolated", FormatClaude, realValues()},
		{"generic placeholders", FormatGeneric, Placeholders()},
		{"generic interpolated", FormatGeneric, realValues()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir, files := export(t, tc.format, tc.values)
			for _, f := range files {
				doc := read(t, dir, f)
				if doc == "" {
					t.Errorf("%s is empty", f)
				}
				if strings.Contains(doc, "{{") {
					t.Errorf("%s contains unrendered template syntax", f)
				}
			}
		})
	}
}

func TestExportInterpolatesRealValues(t *testing.T) {
	v := realValues()
	dir, files := export(t, FormatClaude, v)

	all := ""
	for _, f := range files {
		all += read(t, dir, f)
	}
	for _, want := range []string{
		v.Endpoint, v.PublishableKey, v.JWKSURL, v.Issuer, v.Slug,
		v.SDKVersion, v.Google.WebClientID, v.Apple.ServicesID,
	} {
		if !strings.Contains(all, want) {
			t.Errorf("exported skill misses interpolated value %q", want)
		}
	}
	for _, stale := range []string{"MOTH_BASE_URL", "pk_YOUR_PUBLISHABLE_KEY", "PROJECT_SLUG", "MOTH_SDK_VERSION"} {
		if strings.Contains(all, stale) {
			t.Errorf("interpolated skill still contains placeholder %q", stale)
		}
	}
	if !strings.Contains(read(t, dir, "SKILL.md"), "Sign in with Google | enabled") {
		t.Error("SKILL.md does not report Google as enabled")
	}
}

func TestExportKeepsPlaceholdersWithoutProject(t *testing.T) {
	dir, files := export(t, FormatClaude, Placeholders())
	all := ""
	for _, f := range files {
		all += read(t, dir, f)
	}
	for _, want := range []string{"MOTH_BASE_URL", "pk_YOUR_PUBLISHABLE_KEY", "PROJECT_SLUG", "MOTH_SDK_VERSION"} {
		if !strings.Contains(all, want) {
			t.Errorf("un-interpolated skill misses placeholder %q", want)
		}
	}
}

func TestExportGenericOmitsClaudeConventions(t *testing.T) {
	dir, files := export(t, FormatGeneric, Placeholders())

	if files[0] != "README.md" {
		t.Fatalf("generic main document = %q, want README.md", files[0])
	}
	if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); !os.IsNotExist(err) {
		t.Error("generic export must not write SKILL.md")
	}
	doc := read(t, dir, "README.md")
	if strings.HasPrefix(doc, "---") {
		t.Error("generic export must not emit YAML frontmatter")
	}
	if !strings.HasPrefix(doc, "# ") {
		t.Errorf("generic export should open with a markdown heading, got %.40q", doc)
	}
}

func TestExportRejectsUnknownFormat(t *testing.T) {
	if _, err := Export(t.TempDir(), Format("html"), Placeholders()); err == nil {
		t.Fatal("want error for unknown format")
	}
	if _, err := ParseFormat("html"); err == nil {
		t.Fatal("want error from ParseFormat")
	}
}
