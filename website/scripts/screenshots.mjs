#!/usr/bin/env node
/**
 * Generates the landing-page screenshots from a seeded demo moth instance.
 *
 * What it does (deterministic, re-runnable — a fresh throwaway data dir
 * every run, fixed seed, fixed viewport, reduced motion, light scheme):
 *
 *   1. creates an admin account offline (`moth admin create`),
 *   2. starts `bin/moth serve` on 127.0.0.1:8991,
 *   3. creates three demo projects with distinct branding through the real
 *      admin SPA + connect JSON endpoints (same flows as web/admin/e2e),
 *   4. stops the server, seeds deterministic analytics for the featured
 *      project (`moth admin seed-analytics --seed 1`), restarts it,
 *   5. captures into website/public/screenshots/:
 *        admin-projects.png   projects list showing several apps
 *        login-light.png      themed login preview, light
 *        login-dark.png       themed login preview, dark
 *        analytics.png        analytics dashboard with 90 days of data
 *
 * Requirements: `make build` (bin/moth) and `npm ci` + Playwright chromium
 * in web/admin (`npx playwright install chromium`). When either is missing
 * the script degrades to clearly-labeled placeholder PNGs (sharp) so the
 * site still builds; set MOTH_SCREENSHOTS_STRICT=1 (CI does) to fail
 * instead.
 *
 * Run via `make website-screenshots` at the repo root, or directly:
 *   node scripts/screenshots.mjs
 */

import { spawn, spawnSync } from 'node:child_process';
import { createRequire } from 'node:module';
import { mkdtempSync, mkdirSync, rmSync, existsSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(here, '../..');
const outDir = resolve(here, '../public/screenshots');

const BIN = process.env.MOTH_BIN ?? join(repoRoot, 'bin/moth');
const ADDR = '127.0.0.1:8991';
const BASE_URL = `http://${ADDR}`;
const STRICT = process.env.MOTH_SCREENSHOTS_STRICT === '1';

const admin = { email: 'demo@moth.local', password: 'website-shot-1' };

// Three demo apps; the featured one gets the analytics seed and the themed
// login capture. Themes are the known-good reference themes shared with the
// SDK golden suite (sdk/flutter/test/golden) so contrast validation passes.
const projects = [
  {
    name: 'Birdwatch',
    slug: 'birdwatch',
    featured: true,
    theme: {
      colors: {
        primary: '#C8481F',
        onPrimary: '#FFFFFF',
        background: '#FFF8F2',
        onBackground: '#33201A',
        surface: '#FFFFFF',
        onSurface: '#33201A',
        error: '#8C1D18',
        onError: '#FFFFFF',
      },
      darkColors: {
        primary: '#FFB59D',
        onPrimary: '#3B0900',
        background: '#1F1410',
        onBackground: '#F5E4DD',
        surface: '#2A1B15',
        onSurface: '#F5E4DD',
        error: '#FFB4AB',
        onError: '#3B0900',
      },
      typography: { fontFamily: 'Inter', scale: 1.0 },
      spacing: { unit: 8 },
      shape: { cornerRadius: 16 },
      legal: { privacyUrl: 'https://example.com/privacy' },
    },
  },
  {
    name: 'Tidepool',
    slug: 'tidepool',
    theme: {
      colors: {
        primary: '#0B6E99',
        onPrimary: '#FFFFFF',
        background: '#F7FAFC',
        onBackground: '#102A43',
        surface: '#FFFFFF',
        onSurface: '#102A43',
        error: '#B00020',
        onError: '#FFFFFF',
      },
      typography: { fontFamily: 'Inter', scale: 0.9 },
      spacing: { unit: 6 },
      shape: { cornerRadius: 2 },
      legal: {},
    },
  },
  {
    name: 'Lumen Notes',
    slug: 'lumen-notes',
    theme: {
      colors: {
        primary: '#6750A4',
        onPrimary: '#FFFFFF',
        background: '#FFFBFE',
        onBackground: '#1C1B1F',
        surface: '#FFFBFE',
        onSurface: '#1C1B1F',
        error: '#B3261E',
        onError: '#FFFFFF',
      },
      typography: { fontFamily: 'Inter', scale: 1.0 },
      spacing: { unit: 8 },
      shape: { cornerRadius: 12 },
      legal: {},
    },
  },
];

// ---------------------------------------------------------------------------

function fail(msg) {
  if (STRICT) {
    console.error(`screenshots: ${msg}`);
    process.exit(1);
  }
  console.warn(`screenshots: ${msg}`);
  console.warn('screenshots: falling back to labeled placeholder images.');
  return placeholders();
}

// Placeholder PNGs at the same dimensions as the real captures, visibly
// labeled so they can never be mistaken for product screenshots.
async function placeholders() {
  const { default: sharp } = await import('sharp');
  mkdirSync(outDir, { recursive: true });
  const make = async (name, w, h, label) => {
    const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${w}" height="${h}">
      <rect width="100%" height="100%" fill="#FAFAFA"/>
      <rect x="1" y="1" width="${w - 2}" height="${h - 2}" fill="none" stroke="#D4D4D4" stroke-width="2" stroke-dasharray="12 10"/>
      <text x="50%" y="47%" text-anchor="middle" font-family="Menlo, monospace" font-size="${Math.round(w / 45)}" fill="#666666">${label}</text>
      <text x="50%" y="53%" text-anchor="middle" font-family="Menlo, monospace" font-size="${Math.round(w / 60)}" fill="#999999">placeholder — run \`make website-screenshots\`</text>
    </svg>`;
    await sharp(Buffer.from(svg)).png().toFile(join(outDir, name));
    console.log(`  placeholder ${name} (${w}x${h})`);
  };
  await make('admin-projects.png', 2400, 1500, 'admin console — projects list');
  await make('login-light.png', 760, 1520, 'themed login (light)');
  await make('login-dark.png', 760, 1520, 'themed login (dark)');
  await make('analytics.png', 2400, 1500, 'analytics dashboard');
  process.exit(0);
}

function run(args, opts = {}) {
  const res = spawnSync(BIN, args, { encoding: 'utf-8', ...opts });
  if (res.status !== 0) {
    throw new Error(`moth ${args.join(' ')} failed:\n${res.stdout}\n${res.stderr}`);
  }
  return res.stdout;
}

function startServer(dataDir) {
  const child = spawn(
    BIN,
    ['serve', '--addr', ADDR, '--base-url', BASE_URL, '--data-dir', dataDir],
    { stdio: ['ignore', 'pipe', 'pipe'] },
  );
  child.stdout.resume();
  child.stderr.resume();
  return child;
}

async function waitHealthy() {
  const deadline = Date.now() + 15_000;
  for (;;) {
    try {
      const resp = await fetch(`${BASE_URL}/healthz`);
      if (resp.ok) return;
    } catch {
      /* not up yet */
    }
    if (Date.now() > deadline) throw new Error('moth did not become healthy in 15s');
    await new Promise((r) => setTimeout(r, 100));
  }
}

async function stopServer(child) {
  if (!child || child.exitCode !== null) return;
  child.kill('SIGTERM');
  await new Promise((resolveExit) => {
    const t = setTimeout(() => {
      child.kill('SIGKILL');
      resolveExit();
    }, 5000);
    child.on('exit', () => {
      clearTimeout(t);
      resolveExit();
    });
  });
}

async function login(page) {
  await page.goto(`${BASE_URL}/admin/login`);
  await page.getByLabel('Email').fill(admin.email);
  await page.getByLabel('Password').fill(admin.password);
  await page.getByRole('button', { name: 'Sign in' }).click();
  await page.getByRole('heading', { name: 'Projects' }).waitFor();
}

// ---------------------------------------------------------------------------

async function main() {
  if (!existsSync(BIN)) {
    return fail(`moth binary not found at ${BIN} — run \`make build\` first (or set MOTH_BIN)`);
  }

  // Playwright is reused from the admin SPA's e2e setup — no second install.
  let chromium;
  try {
    const requireAdmin = createRequire(join(repoRoot, 'web/admin/package.json'));
    ({ chromium } = requireAdmin('@playwright/test'));
  } catch {
    return fail('Playwright not found — run `npm ci` in web/admin first');
  }

  const dataDir = mkdtempSync(join(tmpdir(), 'moth-website-shots-'));
  let server = null;
  let browser = null;

  try {
    // 1. Admin account, offline (same store the server will open).
    run(['admin', 'create', '--email', admin.email, '--password', admin.password, '--data-dir', dataDir]);

    // 2. First server run: create + brand the demo projects.
    server = startServer(dataDir);
    await waitHealthy();

    try {
      browser = await chromium.launch();
    } catch (err) {
      return fail(`Chromium not available (${err.message.split('\n')[0]}) — run \`npx playwright install chromium\` in web/admin`);
    }

    const context = await browser.newContext({
      viewport: { width: 1200, height: 750 },
      deviceScaleFactor: 2,
      colorScheme: 'light',
      reducedMotion: 'reduce',
    });
    const page = await context.newPage();
    await login(page);

    for (const p of projects) {
      await page.goto(`${BASE_URL}/admin`);
      await page.getByRole('button', { name: 'Create project' }).click();
      await page.getByLabel('Name', { exact: true }).fill(p.name);
      await page.getByRole('dialog').getByRole('button', { name: 'Create project' }).click();
      await page.getByText(`${p.name} is ready`).waitFor();
    }

    // Branding via the connect JSON endpoints (session cookie is shared
    // with page.request) — same pattern as web/admin/e2e/smoke.spec.ts.
    const list = await page.request.post(`${BASE_URL}/moth.admin.v1.ProjectService/ListProjects`, {
      data: {},
      headers: { 'Content-Type': 'application/json' },
    });
    if (!list.ok()) throw new Error('ListProjects failed');
    const listed = (await list.json()).projects;
    for (const p of projects) {
      const projectId = listed.find((x) => x.name === p.name)?.id;
      if (!projectId) throw new Error(`project ${p.name} missing from ListProjects`);
      const update = await page.request.post(`${BASE_URL}/moth.admin.v1.ThemeService/UpdateTheme`, {
        data: { projectId, theme: p.theme },
        headers: { 'Content-Type': 'application/json' },
      });
      if (!update.ok()) throw new Error(`UpdateTheme failed for ${p.name}: ${await update.text()}`);
    }

    // 3. Seed analytics offline (the CLI opens the SQLite file directly, so
    //    the server must not hold it), then restart.
    await stopServer(server);
    server = null;
    const featured = projects.find((p) => p.featured);
    run([
      'admin', 'seed-analytics',
      '--project', featured.slug,
      '--days', '90',
      '--seed', '1',
      '--data-dir', dataDir,
    ]);
    server = startServer(dataDir);
    await waitHealthy();

    // 4. Captures.
    mkdirSync(outDir, { recursive: true });

    // Projects list — several apps side by side on one instance.
    await page.goto(`${BASE_URL}/admin`);
    await page.getByRole('heading', { name: 'Projects' }).waitFor();
    await page.getByText('Lumen Notes').waitFor();
    await page.waitForLoadState('networkidle');
    // Crop the empty canvas below the single row of project cards.
    await page.screenshot({
      path: join(outDir, 'admin-projects.png'),
      clip: { x: 0, y: 0, width: 1200, height: 560 },
    });
    console.log('  admin-projects.png');

    // Themed login preview, light + dark (the Design tab's live phone
    // preview — the same rendering the SDK golden suite verifies).
    await page.getByText(featured.name).first().click();
    await page.getByRole('tab', { name: 'Design' }).click();
    const phone = page.locator('.phone');
    await phone.waitFor();
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: 'Light' }).click();
    await phone.screenshot({ path: join(outDir, 'login-light.png') });
    console.log('  login-light.png');
    await page.getByRole('button', { name: 'Dark' }).click();
    await phone.screenshot({ path: join(outDir, 'login-dark.png') });
    console.log('  login-dark.png');

    // Analytics dashboard with 90 seeded days.
    await page.getByRole('tab', { name: 'Analytics' }).click();
    await page.getByText('Login success rate').waitFor();
    await page.waitForLoadState('networkidle');
    // Align the tab bar with the top edge so nothing is cut mid-glyph.
    await page.evaluate(() => {
      const tabs = document.querySelector('[role="tablist"]');
      if (tabs) window.scrollTo(0, tabs.getBoundingClientRect().top + window.scrollY - 24);
    });
    await page.screenshot({ path: join(outDir, 'analytics.png') });
    console.log('  analytics.png');

    console.log(`screenshots: wrote 4 images to ${outDir}`);
  } finally {
    if (browser) await browser.close();
    await stopServer(server);
    rmSync(dataDir, { recursive: true, force: true });
  }
}

main().catch(async (err) => {
  console.error(err);
  if (STRICT) process.exit(1);
  await placeholders();
});
