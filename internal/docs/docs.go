// Package docs embeds the moth documentation and serves it, rendered to
// self-contained HTML, at /docs inside the binary. The markdown is
// single-sourced from the public website (website/src/content/docs) by
// website/scripts/sync-embedded-docs.mjs (`make docs-embed`); the committed
// content/ tree is what go:embed ships, so the embedded docs always match
// the binary's own release.
package docs

import (
	"bytes"
	"embed"
	"html/template"
	"net/http"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	ghtml "github.com/yuin/goldmark/renderer/html"
)

//go:embed content
var contentFS embed.FS

// md renders CommonMark plus the GitHub-flavoured extensions the docs use
// (pipe tables, strikethrough, autolinks). Heading IDs are enabled so the
// in-page anchors the docs link to (#moth-doctor, ...) resolve.
var md = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	goldmark.WithRendererOptions(ghtml.WithUnsafe()),
)

// NavEntry is one item in the docs sidebar. A Group with no Slug is a
// section heading rendered above its following pages.
type NavEntry struct {
	Slug  string // path under /docs (empty = the index)
	Title string
	Group string // non-empty marks a section header row
}

// nav is the ordered sidebar, mirroring the website's Starlight sidebar
// (website/astro.config.mjs). Slugs are content paths without the ".md".
var nav = []NavEntry{
	{Slug: "index", Title: "Overview"},
	{Slug: "quick-start", Title: "Quick start"},
	{Slug: "installation", Title: "Installation & deployment"},
	{Group: "Guides"},
	{Slug: "guides/google", Title: "Sign in with Google"},
	{Slug: "guides/apple", Title: "Sign in with Apple"},
	{Slug: "guides/theming", Title: "Theming the login screen"},
	{Slug: "guides/monetization", Title: "Subscriptions & paywall"},
	{Slug: "guides/analytics", Title: "Analytics"},
	{Slug: "guides/backups", Title: "Backups"},
	{Slug: "guides/migration", Title: "Migration import & export"},
	{Group: "Reference"},
	{Slug: "sdk", Title: "Flutter SDK reference"},
	{Slug: "cli", Title: "CLI overview"},
	{Slug: "cli/reference", Title: "CLI commands"},
	{Slug: "agents", Title: "Agents & automation"},
	{Slug: "api", Title: "API reference"},
	{Slug: "security", Title: "Security & threat model"},
	{Slug: "changelog", Title: "Changelog"},
}

// titleOf returns the sidebar title for a slug, or the slug itself.
func titleOf(slug string) string {
	for _, e := range nav {
		if e.Slug == slug {
			return e.Title
		}
	}
	return slug
}

// Slugs returns every rendered page slug (for tests and sanity checks).
func Slugs() []string {
	out := make([]string, 0, len(nav))
	for _, e := range nav {
		if e.Slug != "" {
			out = append(out, e.Slug)
		}
	}
	return out
}

// render reads content/<slug>.md and renders its body to HTML. The first
// line is the page's "# Title" heading; it is dropped here because the shell
// renders the title from the sidebar.
func render(slug string) (template.HTML, bool) {
	raw, err := contentFS.ReadFile("content/" + slug + ".md")
	if err != nil {
		return "", false
	}
	// Drop the leading H1 (the shell prints the title once).
	if i := bytes.IndexByte(raw, '\n'); i >= 0 && bytes.HasPrefix(raw, []byte("# ")) {
		raw = raw[i+1:]
	}
	var buf bytes.Buffer
	if err := md.Convert(raw, &buf); err != nil {
		return "", false
	}
	return template.HTML(buf.String()), true //nolint:gosec // trusted embedded docs
}

var shell = template.Must(template.New("docs").Parse(shellHTML))

type pageData struct {
	Title string
	Slug  string
	Body  template.HTML
	Nav   []navItem
}

type navItem struct {
	Group   string
	Title   string
	Href    string
	Current bool
}

func buildNav(current string) []navItem {
	items := make([]navItem, 0, len(nav))
	for _, e := range nav {
		if e.Group != "" {
			items = append(items, navItem{Group: e.Group})
			continue
		}
		href := "/docs/" + e.Slug
		if e.Slug == "index" {
			href = "/docs"
		}
		items = append(items, navItem{
			Title:   e.Title,
			Href:    href,
			Current: e.Slug == current,
		})
	}
	return items
}

// Handler serves the embedded docs. Mount it at /docs (the caller strips the
// /docs prefix so this handler sees the slug directly). Unknown slugs 404.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slug := strings.Trim(r.URL.Path, "/")
		if slug == "" {
			slug = "index"
		}
		body, ok := render(slug)
		if !ok {
			http.Error(w, "documentation page not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=300")
		data := pageData{
			Title: titleOf(slug),
			Slug:  slug,
			Body:  body,
			Nav:   buildNav(slug),
		}
		// The header is already committed; a template error here can only be
		// a truncated body, which the client observes directly.
		_ = shell.Execute(w, data)
	})
}
