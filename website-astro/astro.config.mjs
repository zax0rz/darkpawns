import { defineConfig } from 'astro/config';
import sitemap from '@astrojs/sitemap';

const redirects = {
  '/connect': '/play/',
  '/connect/client-downloads': '/play/#desktop-client',
  '/connect/connection-instructions': '/play/#desktop-client',
  '/connect/contact-info': '/play/',
  '/connect/external-links': '/archive/',
  '/changelog': '/',
  '/about/features': '/world/',
  '/about/site-history': '/archive/',
  '/about/archive-index': '/archive/',
  '/credits/wizlist': '/credits/#archived-staff-list',
  '/credits/player-roster': '/archive/',
  '/docs/getting-started/quick-start': '/docs/getting-started/local-server',
  '/docs/getting-started/installation': '/docs/getting-started/local-server',
  '/docs/development': '/docs/server',
  '/docs/development/architecture': '/docs/server/architecture',
  '/docs/agents/memory-system': '/docs/agents/memory',
  '/docs/research/agent-protocols': '/docs/agents/protocol',
  '/docs/research/narrative-memory': '/docs/agents/memory',
  '/docs/api': '/docs/agents/protocol',
  '/research': '/docs/research',
};

// Dark Pawns — Astro migration (replaces the Hugo site in ../website).
export default defineConfig({
  site: 'https://darkpawns.labz0rz.com',
  integrations: [
    sitemap({
      filter: (page) => {
        const path = new URL(page).pathname;
        return !page.endsWith('.json') && !page.endsWith('.md') && path !== '/404/' && !(path.replace(/\/$/, '') in redirects);
      },
    }),
  ],
  // Share the generated world/database assets with the Hugo site while the
  // migration is in progress. The parsers continue to have one output tree.
  publicDir: '../website/static',
  redirects,
});
