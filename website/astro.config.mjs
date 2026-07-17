// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

/**
 * GitHub Pages configuration.
 *
 * The deploy targets a custom domain (CNAME), so the defaults are
 * `base: '/'` and a placeholder `site`. Until the domain is registered,
 * GitHub *project* pages can be targeted instead by setting, at build time:
 *
 *   WEBSITE_SITE=https://aloisdeniel.github.io WEBSITE_BASE=/moth npm run build
 *
 * Once the custom domain exists, the deploy workflow (.github/workflows/
 * pages.yml) handles it from repository variables — set WEBSITE_DOMAIN=<domain>
 * (writes dist/CNAME), WEBSITE_SITE=https://<domain>, and WEBSITE_BASE=/. No
 * CNAME file is committed to the repo.
 */
const site = process.env.WEBSITE_SITE ?? 'https://aloisdeniel.github.io';
const base = process.env.WEBSITE_BASE ?? '/';

// Absolute OG-image URL for the docs pages (Starlight sets og:title /
// og:description itself; the landing layout carries its own tags).
const ogImage = new URL(`${base.replace(/\/$/, '')}/og.png`, site).href;

export default defineConfig({
  site,
  base,
  output: 'static',
  integrations: [
    starlight({
      title: 'moth',
      description:
        'Authentication for all your mobile apps. One small binary.',
      social: [
        {
          icon: 'github',
          label: 'GitHub',
          href: 'https://github.com/aloisdeniel/moth',
        },
      ],
      customCss: ['./src/styles/tokens.css', './src/styles/starlight.css'],
      head: [
        { tag: 'meta', attrs: { property: 'og:image', content: ogImage } },
        { tag: 'meta', attrs: { property: 'og:image:width', content: '1200' } },
        { tag: 'meta', attrs: { property: 'og:image:height', content: '630' } },
        { tag: 'meta', attrs: { name: 'twitter:card', content: 'summary_large_image' } },
        { tag: 'meta', attrs: { name: 'twitter:image', content: ogImage } },
      ],
      sidebar: [
        { label: 'Quick start', slug: 'docs/quick-start' },
        { label: 'Installation & deployment', slug: 'docs/installation' },
        {
          label: 'Guides',
          items: [
            { label: 'Sign in with Google', slug: 'docs/guides/google' },
            { label: 'Sign in with Apple', slug: 'docs/guides/apple' },
            { label: 'Theming the login screen', slug: 'docs/guides/theming' },
            { label: 'Subscriptions & paywall', slug: 'docs/guides/monetization' },
            { label: 'Analytics', slug: 'docs/guides/analytics' },
            { label: 'Backups', slug: 'docs/guides/backups' },
            { label: 'Migration import & export', slug: 'docs/guides/migration' },
          ],
        },
        { label: 'Flutter SDK reference', slug: 'docs/sdk' },
        {
          label: 'CLI reference',
          items: [
            { label: 'Overview', slug: 'docs/cli' },
            { label: 'Commands', slug: 'docs/cli/reference' },
          ],
        },
        { label: 'Agents & automation', slug: 'docs/agents' },
        { label: 'API reference', slug: 'docs/api' },
        { label: 'Security & threat model', slug: 'docs/security' },
        { label: 'Changelog', slug: 'docs/changelog' },
      ],
    }),
  ],
});
