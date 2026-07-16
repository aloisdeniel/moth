// Single-sources the binary's embedded /docs from the published website
// content. It reads the Starlight markdown under
// website/src/content/docs/docs, strips the Astro frontmatter (keeping the
// title as a top-level heading), lowers Starlight-only syntax (`:::note`
// admonitions) to plain CommonMark, rewrites inter-doc links to the binary's
// /docs/<slug> routes, and writes the result under internal/docs/content so
// `go:embed` can ship it inside the binary.
//
// The generated CLI reference is single-sourced one level up from
// docs/cli/reference.md (see sync-generated.mjs); this script reads that
// canonical file directly so the embedded copy never depends on a prior
// website build.
//
// Run via `make docs-embed`; the output is committed (the binary embeds it).
import {
  mkdirSync,
  readFileSync,
  writeFileSync,
  rmSync,
  existsSync,
} from 'node:fs';
import { dirname, join } from 'node:path';
import path from 'node:path/posix';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const repo = join(here, '..', '..');
const contentRoot = join(repo, 'website', 'src', 'content', 'docs', 'docs');
const cliReference = join(repo, 'docs', 'cli', 'reference.md');
const outRoot = join(repo, 'internal', 'docs', 'content');

// slug (without the leading "docs/") → source markdown file. The index lives
// at content/index.md; every other page keeps its relative path. Order here
// is informational only — the binary owns the nav order (internal/docs).
const pages = [
  ['index', join(contentRoot, 'index.md')],
  ['quick-start', join(contentRoot, 'quick-start.md')],
  ['installation', join(contentRoot, 'installation.md')],
  ['guides/google', join(contentRoot, 'guides', 'google.md')],
  ['guides/apple', join(contentRoot, 'guides', 'apple.md')],
  ['guides/theming', join(contentRoot, 'guides', 'theming.md')],
  ['guides/analytics', join(contentRoot, 'guides', 'analytics.md')],
  ['guides/backups', join(contentRoot, 'guides', 'backups.md')],
  ['guides/migration', join(contentRoot, 'guides', 'migration.md')],
  ['sdk', join(contentRoot, 'sdk.md')],
  ['cli', join(contentRoot, 'cli.md')],
  ['cli/reference', cliReference],
  ['agents', join(contentRoot, 'agents.md')],
  ['api', join(contentRoot, 'api.md')],
  ['security', join(contentRoot, 'security.md')],
  ['changelog', join(contentRoot, 'changelog.md')],
];

const known = new Set(pages.map(([slug]) => slug));

function parseFrontmatter(raw) {
  const m = raw.match(/^---\n([\s\S]*?)\n---\n?/u);
  if (!m) return { title: '', description: '', body: raw };
  const fm = m[1];
  const title = (fm.match(/^title:\s*(.+)$/mu)?.[1] ?? '').trim();
  const description = (fm.match(/^description:\s*(.+)$/mu)?.[1] ?? '').trim();
  return { title, description, body: raw.slice(m[0].length) };
}

// `:::note[Title]\nbody\n:::` → a blockquote titled by the admonition kind.
function lowerAdmonitions(body) {
  return body.replace(
    /^:::(note|tip|caution|danger|warning)(?:\[([^\]]*)\])?\n([\s\S]*?)\n:::\s*$/gmu,
    (_all, kind, title, inner) => {
      const heading =
        (title || kind.charAt(0).toUpperCase() + kind.slice(1)).trim();
      const quoted = inner
        .trimEnd()
        .split('\n')
        .map((l) => (l ? `> ${l}` : '>'))
        .join('\n');
      return `> **${heading}**\n>\n${quoted}\n`;
    },
  );
}

// Rewrite doc-relative Starlight links (e.g. `../guides/backups/`,
// `quick-start/`) to the binary's `/docs/<slug>` routes. External URLs,
// fragments, mailto and asset paths are left untouched.
function rewriteLinks(body, slug) {
  const dir = path.dirname(slug); // "" for top-level, "guides" for nested
  return body.replace(/\]\(([^)]+)\)/gu, (all, target) => {
    const [pathPart, hash = ''] = target.split('#');
    if (
      pathPart === '' ||
      /^(?:[a-z]+:|\/\/|\/|#)/iu.test(target) ||
      /\.(png|jpg|jpeg|svg|gif|webp|tar\.gz|proto)$/iu.test(pathPart)
    ) {
      return all;
    }
    // Resolve the relative link against the current page's directory.
    const base = dir === '.' ? '' : dir;
    let resolved = path.normalize(path.join(base, pathPart));
    resolved = resolved.replace(/\/+$/u, '').replace(/^\.$/u, 'index');
    if (!known.has(resolved)) return all; // unknown → leave as authored
    const anchor = hash ? `#${hash}` : '';
    return `](/docs/${resolved === 'index' ? '' : resolved}${anchor})`;
  });
}

if (existsSync(outRoot)) rmSync(outRoot, { recursive: true, force: true });

let written = 0;
for (const [slug, source] of pages) {
  if (!existsSync(source)) {
    console.warn(`sync-embedded-docs: missing source for ${slug}: ${source}`);
    continue;
  }
  const raw = readFileSync(source, 'utf8');
  const { title, body } = parseFrontmatter(raw);
  // Drop the generator HTML comment the CLI reference ships with, and any
  // leading H1 so the frontmatter title is the single page heading.
  let cleaned = body.replace(/^<!--[\s\S]*?-->\s*/u, '').replace(/^#\s.*\n+/u, '');
  cleaned = lowerAdmonitions(cleaned);
  cleaned = rewriteLinks(cleaned, slug);
  const heading = title || slug;
  const page = `# ${heading}\n\n${cleaned.trimStart()}`;
  const target = join(outRoot, `${slug}.md`);
  mkdirSync(dirname(target), { recursive: true });
  writeFileSync(target, page.trimEnd() + '\n');
  written += 1;
}

console.log(`sync-embedded-docs: wrote ${written} pages to ${outRoot}`);
