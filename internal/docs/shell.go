package docs

// shellHTML is the self-contained page chrome around a rendered docs page: a
// fixed sidebar plus the article, styled with system fonts and a light/dark
// palette. No external assets — everything ships in the binary, matching the
// project's "everything embedded" rule.
const shellHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="robots" content="noindex">
<title>{{.Title}} — moth docs</title>
<style>
:root {
  --bg: #ffffff; --fg: #1a1a1a; --muted: #666; --border: #e5e5e5;
  --surface: #f7f7f8; --link: #2563eb; --code-bg: #f2f2f4; --accent: #7c3aed;
}
@media (prefers-color-scheme: dark) {
  :root {
    --bg: #16161a; --fg: #e6e6e6; --muted: #9a9a9a; --border: #2a2a30;
    --surface: #1d1d22; --link: #7aa2ff; --code-bg: #24242b; --accent: #a78bfa;
  }
}
* { box-sizing: border-box; }
body {
  margin: 0; background: var(--bg); color: var(--fg);
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
  line-height: 1.65; font-size: 16px;
  display: grid; grid-template-columns: 280px minmax(0, 1fr);
}
nav {
  border-right: 1px solid var(--border); background: var(--surface);
  padding: 24px 16px; height: 100vh; position: sticky; top: 0; overflow-y: auto;
}
nav .brand { font-weight: 700; font-size: 18px; margin: 0 8px 20px; display: block; color: var(--fg); text-decoration: none; }
nav .brand span { color: var(--accent); }
nav .group { font-size: 12px; text-transform: uppercase; letter-spacing: .05em; color: var(--muted); margin: 18px 8px 6px; }
nav a { display: block; padding: 5px 8px; border-radius: 6px; color: var(--fg); text-decoration: none; font-size: 14px; }
nav a:hover { background: var(--code-bg); }
nav a.current { background: var(--accent); color: #fff; }
main { padding: 40px 48px; max-width: 860px; }
article { overflow-wrap: break-word; }
article h1 { font-size: 32px; margin: 0 0 24px; line-height: 1.2; }
article h2 { font-size: 24px; margin: 40px 0 12px; padding-top: 8px; border-top: 1px solid var(--border); }
article h3 { font-size: 19px; margin: 28px 0 8px; }
article a { color: var(--link); }
article code { background: var(--code-bg); padding: .15em .4em; border-radius: 4px; font-size: .88em;
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace; }
article pre { background: var(--code-bg); padding: 16px; border-radius: 8px; overflow-x: auto; }
article pre code { background: none; padding: 0; font-size: 13.5px; }
article blockquote { margin: 16px 0; padding: 12px 16px; border-left: 3px solid var(--accent);
  background: var(--surface); border-radius: 0 8px 8px 0; }
article blockquote p:first-child { margin-top: 0; }
article blockquote p:last-child { margin-bottom: 0; }
article table { border-collapse: collapse; width: 100%; margin: 16px 0; display: block; overflow-x: auto; }
article th, article td { border: 1px solid var(--border); padding: 8px 12px; text-align: left; }
article th { background: var(--surface); }
article img { max-width: 100%; }
footer { margin-top: 56px; padding-top: 20px; border-top: 1px solid var(--border); color: var(--muted); font-size: 13px; }
footer a { color: var(--link); }
@media (max-width: 720px) {
  body { grid-template-columns: 1fr; }
  nav { position: static; height: auto; border-right: none; border-bottom: 1px solid var(--border); }
  main { padding: 24px; }
}
</style>
</head>
<body>
<nav>
  <a class="brand" href="/docs"><span>◆</span> moth</a>
  {{- range .Nav}}
  {{- if .Group}}
  <div class="group">{{.Group}}</div>
  {{- else}}
  <a href="{{.Href}}"{{if .Current}} class="current"{{end}}>{{.Title}}</a>
  {{- end}}
  {{- end}}
</nav>
<main>
  <article>
    <h1>{{.Title}}</h1>
    {{.Body}}
  </article>
  <footer>
    These docs are embedded in your moth binary and match its exact version.
    The published copy lives at <a href="https://aloisdeniel.github.io/moth/">the moth website</a>.
  </footer>
</main>
</body>
</html>
`
