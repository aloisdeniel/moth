#!/usr/bin/env node
/**
 * Generates the raster favicon/social assets from the moth mark
 * (docs/logo.svg) into website/public/. Committed outputs; re-run when the
 * logo or the card copy changes:
 *
 *   og.png                 1200x630 OpenGraph/social card
 *   apple-touch-icon.png   180x180
 *   favicon-32.png         32x32 PNG fallback next to favicon.svg
 *
 * Everything is drawn from DESIGN.md tokens (near-monochrome, mono type
 * for the install line). Uses system fonts at render time, which is fine:
 * these are static images. At runtime the site self-hosts Cascadia Code and
 * loads Satoshi from Fontshare's CDN (with a system-font fallback).
 */

import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import sharp from 'sharp';

const here = dirname(fileURLToPath(import.meta.url));
const pub = resolve(here, '../public');

// The moth mark's two paths, verbatim from docs/logo.svg (256-canvas
// coordinates; content bounds ≈ x 52–204, y 56–184).
const body =
  'M102.724 107C104.359 107 105.83 107.996 106.438 109.514L128.094 163.605L128.254 164.01L128.413 163.605L150.069 109.514C150.677 107.996 152.148 107 153.783 107H196.503C200.576 107 203.466 110.973 202.211 114.849L197.376 129.773C194.861 137.537 188.076 143.14 179.977 144.143L167.38 145.702L181.08 154.237C183.64 155.832 185.087 158.731 184.822 161.735C184.349 167.103 184.773 172.513 186.076 177.741L186.234 178.378C186.804 180.667 185.073 182.884 182.713 182.884H155.281C153.072 182.884 151.281 181.093 151.281 178.884V156.837C151.281 154.171 151.315 151.949 151.383 150.172C151.452 148.394 151.52 146.856 151.588 145.557C151.646 144.983 150.779 144.764 150.57 145.302L136.072 182.701C136.03 182.811 135.924 182.884 135.806 182.884H120.701C120.583 182.884 120.477 182.811 120.435 182.701L105.937 145.302C105.728 144.764 104.861 144.983 104.919 145.557C104.987 146.856 105.055 148.394 105.124 150.172C105.192 151.949 105.226 154.171 105.226 156.837V178.884C105.226 181.093 103.435 182.884 101.226 182.884H73.7936C71.4341 182.884 69.7025 180.667 70.273 178.378L70.4312 177.741C71.7343 172.513 72.1582 167.103 71.6852 161.735C71.4204 158.731 72.8671 155.832 75.4273 154.237L89.1266 145.702L76.5299 144.143C68.4309 143.14 61.6454 137.537 59.1305 129.773L54.2965 114.849C53.0412 110.973 55.9306 107 60.0045 107H102.724Z';
const antennae =
  'M103.317 58.6336C106.281 57.1518 109.885 58.3528 111.367 61.3163L122.367 83.3163C123.849 86.28 122.648 89.884 119.685 91.3661C116.721 92.8479 113.117 91.647 111.635 88.6835L100.635 66.6835C99.153 63.7197 100.354 60.1157 103.317 58.6336ZM145.635 61.3163C147.117 58.3528 150.721 57.1519 153.685 58.6336C156.648 60.1157 157.849 63.7197 156.367 66.6835L145.367 88.6835C143.885 91.647 140.281 92.8479 137.317 91.3661C134.354 89.884 133.153 86.28 134.635 83.3163L145.635 61.3163Z';

// Opaque light-scheme colors (rasters can't follow prefers-color-scheme);
// square viewBox centered on the mark.
const mark = (fg, size) => `
  <svg xmlns="http://www.w3.org/2000/svg" width="${size}" height="${size}" viewBox="44 36 168 168">
    <path fill="${fg}" d="${body}"/>
    <path fill="${fg}" d="${antennae}"/>
  </svg>`;

const og = `
  <svg xmlns="http://www.w3.org/2000/svg" width="1200" height="630">
    <rect width="1200" height="630" fill="#FFFFFF"/>
    <rect x="0.5" y="0.5" width="1199" height="629" fill="none" stroke="#EBEBEB"/>
    <g transform="translate(48 45) scale(0.79)">
      <path fill="#0A0A0A" d="${body}"/>
      <path fill="#0A0A0A" d="${antennae}"/>
    </g>
    <text x="96" y="316" font-family="-apple-system, 'Segoe UI', Roboto, sans-serif" font-weight="700" font-size="64" letter-spacing="-1" fill="#0A0A0A">Authentication for all your</text>
    <text x="96" y="392" font-family="-apple-system, 'Segoe UI', Roboto, sans-serif" font-weight="700" font-size="64" letter-spacing="-1" fill="#0A0A0A">mobile apps. One small binary.</text>
    <text x="96" y="470" font-family="-apple-system, 'Segoe UI', Roboto, sans-serif" font-size="30" fill="#666666">One server. Every app gets its own users, keys, and branded login.</text>
    <rect x="90" y="512" width="330" height="56" rx="8" fill="#FAFAFA" stroke="#EBEBEB"/>
    <text x="114" y="548" font-family="Menlo, Consolas, monospace" font-size="24" fill="#0A0A0A">$ moth serve</text>
  </svg>`;

await sharp(Buffer.from(og)).png().toFile(join(pub, 'og.png'));
await sharp(Buffer.from(mark('#0A0A0A', 180)))
  .flatten({ background: '#FFFFFF' })
  .png()
  .toFile(join(pub, 'apple-touch-icon.png'));
await sharp(Buffer.from(mark('#0A0A0A', 32))).png().toFile(join(pub, 'favicon-32.png'));

console.log('assets: wrote og.png (1200x630), apple-touch-icon.png (180x180), favicon-32.png');
