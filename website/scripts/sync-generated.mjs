// Copies the generated CLI reference (docs/cli/reference.md, produced by
// `moth docs gen` in the main repo) into the Starlight content tree so the
// website can never drift from the binary's real command surface.
//
// Runs automatically before `npm run dev` / `npm run build` (pre-scripts).
// The output file is gitignored — never hand-edit it, never commit it.
import { mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const source = join(here, '..', '..', 'docs', 'cli', 'reference.md');
const target = join(
  here,
  '..',
  'src',
  'content',
  'docs',
  'docs',
  'cli',
  'reference.md',
);

let body = readFileSync(source, 'utf8');

// Strip the generator comment and the H1 (Starlight renders the frontmatter
// title as the page heading); keep everything else verbatim.
body = body
  .replace(/^<!--[\s\S]*?-->\s*/u, '')
  .replace(/^# .*\n+/u, '');

const page = `---
title: Commands
description: Generated reference for every moth command and flag — always matching the binary.
---

<!-- GENERATED at build time from docs/cli/reference.md by
     website/scripts/sync-generated.mjs. Do not edit, do not commit. -->

${body}`;

mkdirSync(dirname(target), { recursive: true });
writeFileSync(target, page);
console.log(`sync-generated: wrote ${target}`);
