#!/usr/bin/env node
/**
 * Internal link/asset integrity check over the built site (dist/).
 *
 * Every href/src/srcset in every built HTML page must resolve to a file in
 * dist (respecting the configured base path and trailing-slash → index.html
 * routing). External http(s) links are collected and printed but not
 * fetched here — lychee covers them in CI (network flakes should not make
 * local builds red). Exits non-zero on any broken internal reference.
 *
 * Usage: node scripts/check-links.mjs  (after `npm run build`; reads the
 * same WEBSITE_BASE the build used)
 */

import { readdirSync, readFileSync, existsSync, statSync } from 'node:fs';
import { dirname, join, resolve, sep } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const dist = resolve(here, '../dist');
const base = (process.env.WEBSITE_BASE ?? '/').replace(/\/$/, '');

if (!existsSync(dist)) {
  console.error('check-links: dist/ not found — run `npm run build` first');
  process.exit(1);
}

function* htmlFiles(dir) {
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const p = join(dir, entry.name);
    if (entry.isDirectory()) yield* htmlFiles(p);
    else if (entry.name.endsWith('.html')) yield p;
  }
}

// Does an internal absolute path resolve to a file in dist?
function resolves(urlPath) {
  let p = urlPath;
  if (base && p.startsWith(base + '/')) p = p.slice(base.length);
  else if (base && p === base) p = '/';
  p = decodeURIComponent(p.split('#')[0].split('?')[0]);
  const onDisk = join(dist, ...p.split('/').filter(Boolean));
  if (existsSync(onDisk)) {
    if (statSync(onDisk).isDirectory()) return existsSync(join(onDisk, 'index.html'));
    return true;
  }
  // /foo (no trailing slash) may still be /foo/index.html or /foo.html
  return existsSync(join(onDisk, 'index.html')) || existsSync(onDisk + '.html');
}

const broken = [];
const external = new Set();
let checked = 0;

const attr = /(?:href|src)=["']([^"']+)["']|srcset=["']([^"']+)["']/g;

for (const file of htmlFiles(dist)) {
  const html = readFileSync(file, 'utf-8');
  const page = file.slice(dist.length).split(sep).join('/');
  const urls = [];
  for (const m of html.matchAll(attr)) {
    if (m[1]) urls.push(m[1]);
    if (m[2]) for (const part of m[2].split(',')) urls.push(part.trim().split(/\s+/)[0]);
  }
  for (const url of urls) {
    if (
      url.startsWith('#') ||
      url.startsWith('mailto:') ||
      url.startsWith('data:') ||
      url === ''
    ) {
      continue;
    }
    if (/^https?:\/\//.test(url)) {
      // Self-referential absolute URLs (canonical, og:url) → check locally.
      const site = new URL(url);
      const self = process.env.WEBSITE_SITE ?? 'https://aloisdeniel.github.io';
      if (url.startsWith(self) && new URL(self).host === site.host) {
        checked++;
        if (!resolves(site.pathname)) broken.push({ page, url });
      } else {
        external.add(url);
      }
      continue;
    }
    checked++;
    const target = url.startsWith('/')
      ? url
      : new URL(url, `http://x${page}`).pathname; // relative → resolve against the page
    if (!resolves(target)) broken.push({ page, url });
  }
}

console.log(
  `check-links: ${checked} internal references checked, ${external.size} distinct external URLs (verified by lychee in CI)`,
);
for (const url of [...external].sort()) console.log(`  external: ${url}`);

if (broken.length > 0) {
  console.error(`check-links: ${broken.length} broken internal reference(s):`);
  for (const b of broken) console.error(`  ${b.page} -> ${b.url}`);
  process.exit(1);
}
console.log('check-links: OK');
