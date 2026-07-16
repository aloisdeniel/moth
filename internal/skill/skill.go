// Package skill renders the embedded moth agent skill: a SKILL.md +
// references/ directory (the Agent Skills format) teaching a coding agent
// both halves of moth — integrating the Flutter SDK into an app and
// administering an instance through the CLI. `moth skill export` writes it
// either with placeholders or interpolated with one project's real values,
// the agent equivalent of the admin SPA's setup-instructions page.
package skill

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

//go:embed templates
var templatesFS embed.FS

// Format selects the flavor of the exported skill directory.
type Format string

const (
	// FormatClaude follows Claude Code conventions: SKILL.md with YAML
	// name/description frontmatter, references linked relatively.
	FormatClaude Format = "claude"
	// FormatGeneric is the plain-markdown fallback for other agent
	// frameworks: README.md with a heading instead of frontmatter.
	FormatGeneric Format = "generic"
)

// ParseFormat validates a --format flag value.
func ParseFormat(s string) (Format, error) {
	switch Format(s) {
	case FormatClaude, FormatGeneric:
		return Format(s), nil
	default:
		return "", fmt.Errorf("unknown format %q (want %q or %q)", s, FormatClaude, FormatGeneric)
	}
}

// GoogleValues is the public part of a project's Google provider config.
type GoogleValues struct {
	Enabled         bool
	WebClientID     string
	IOSClientID     string
	AndroidClientID string
}

// AppleValues is the public part of a project's Apple provider config.
type AppleValues struct {
	Enabled    bool
	ServicesID string
	BundleIDs  []string
}

// Values is everything the templates interpolate. Placeholders() returns
// the un-interpolated defaults; `moth skill export --project` fills the
// struct from the admin RPCs instead.
type Values struct {
	// Endpoint is the instance base URL, no trailing slash.
	Endpoint       string
	ProjectName    string
	Slug           string
	PublishableKey string
	JWKSURL        string
	Issuer         string
	Audience       string
	// SDKVersion is the pubspec version constraint for the served
	// moth_auth package (pre-releases pinned exactly, releases careted).
	SDKVersion string
	Google     GoogleValues
	Apple      AppleValues
	// Interpolated says whether the values above are a real project's
	// (true) or the documented placeholders (false).
	Interpolated bool
}

// Placeholders returns the un-interpolated template values. The strings
// are deliberately loud, greppable placeholders; SKILL.md documents each
// one and where its real value comes from.
func Placeholders() Values {
	return Values{
		Endpoint:       "https://MOTH_BASE_URL",
		ProjectName:    "your project",
		Slug:           "PROJECT_SLUG",
		PublishableKey: "pk_YOUR_PUBLISHABLE_KEY",
		JWKSURL:        "https://MOTH_BASE_URL/p/PROJECT_SLUG/.well-known/jwks.json",
		Issuer:         "https://MOTH_BASE_URL/p/PROJECT_SLUG",
		Audience:       "PROJECT_SLUG",
		SDKVersion:     "MOTH_SDK_VERSION",
	}
}

// renderData is what the templates see: the values plus the format switch.
type renderData struct {
	Values
	// Claude toggles the Claude Code-specific bits (frontmatter).
	Claude bool
}

// outputs maps each embedded template to the file it renders to. The main
// document is SKILL.md in the Claude format and README.md in the generic
// one; the references are shared.
var outputs = []struct{ src, dst string }{
	{"SKILL.md.tmpl", "SKILL.md"},
	{"flutter-integration.md.tmpl", "references/flutter-integration.md"},
	{"backend-verification.md.tmpl", "references/backend-verification.md"},
	{"provider-setup.md.tmpl", "references/provider-setup.md"},
	{"cli-administration.md.tmpl", "references/cli-administration.md"},
}

// Export renders the skill into dir (created if needed, existing files
// overwritten — exports are idempotent) and returns the dir-relative paths
// written, in a fixed order.
func Export(dir string, format Format, v Values) ([]string, error) {
	if _, err := ParseFormat(string(format)); err != nil {
		return nil, err
	}
	tmpl, err := template.ParseFS(templatesFS, "templates/*.tmpl", "templates/references/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parse skill templates: %w", err)
	}
	data := renderData{Values: v, Claude: format == FormatClaude}

	written := make([]string, 0, len(outputs))
	for _, o := range outputs {
		dst := o.dst
		if dst == "SKILL.md" && format == FormatGeneric {
			dst = "README.md"
		}
		var buf bytes.Buffer
		if err := tmpl.ExecuteTemplate(&buf, o.src, data); err != nil {
			return nil, fmt.Errorf("render %s: %w", o.src, err)
		}
		path := filepath.Join(dir, filepath.FromSlash(dst))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
			return nil, err
		}
		written = append(written, dst)
	}
	return written, nil
}
