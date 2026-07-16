#!/usr/bin/env node
/**
 * Generates the raster favicon/social assets from the (placeholder) mark
 * into website/public/. Committed outputs; re-run after the real moth
 * logo/wordmark lands (a later stage of milestone 09):
 *
 *   og.png                 1200x630 OpenGraph/social card
 *   apple-touch-icon.png   180x180
 *   favicon-32.png         32x32 PNG fallback next to favicon.svg
 *
 * Everything is drawn from DESIGN.md tokens (near-monochrome, mono type
 * for the install line). Uses system fonts at render time, which is fine:
 * these are static images, the *site* still self-hosts Satoshi/Cascadia.
 */

import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import sharp from 'sharp';

const here = dirname(fileURLToPath(import.meta.url));
const pub = resolve(here, '../public');

// The placeholder "m" mark, opaque light-scheme colors (rasters can't
// follow prefers-color-scheme).
const mark = (fg, size) => `
  <svg xmlns="http://www.w3.org/2000/svg" width="${size}" height="${size}" viewBox="0 0 32 32">
    <path fill="${fg}" d="M6 25V11.5c0-.3.2-.5.5-.5h2.2c.2 0 .4.1.5.3L13 18l3.8-6.7c.1-.2.3-.3.5-.3h2.2c.3 0 .5.2.5.5V25h-3v-8l-3 5.2c-.2.4-.8.4-1 0l-3-5.2v8H6z" transform="translate(3.5 -2)"/>
  </svg>`;

const og = `
  <svg xmlns="http://www.w3.org/2000/svg" width="1200" height="630">
    <rect width="1200" height="630" fill="#FFFFFF"/>
    <rect x="0.5" y="0.5" width="1199" height="629" fill="none" stroke="#EBEBEB"/>
    <g transform="translate(96 120) scale(2.6)">
      <path fill="#0A0A0A" d="M6 25V11.5c0-.3.2-.5.5-.5h2.2c.2 0 .4.1.5.3L13 18l3.8-6.7c.1-.2.3-.3.5-.3h2.2c.3 0 .5.2.5.5V25h-3v-8l-3 5.2c-.2.4-.8.4-1 0l-3-5.2v8H6z" transform="translate(3.5 -2)"/>
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
