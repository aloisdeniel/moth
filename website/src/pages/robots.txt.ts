import type { APIRoute } from 'astro';

// Static robots.txt pointing at the sitemap Starlight/@astrojs/sitemap
// emits; site and base come from astro.config.mjs so any deploy layout
// (custom domain or project pages) gets the right absolute URL.
export const GET: APIRoute = ({ site }) => {
  const base = import.meta.env.BASE_URL.replace(/\/$/, '');
  const sitemap = new URL(`${base}/sitemap-index.xml`, site).href;
  return new Response(`User-agent: *\nAllow: /\n\nSitemap: ${sitemap}\n`, {
    headers: { 'Content-Type': 'text/plain; charset=utf-8' },
  });
};
