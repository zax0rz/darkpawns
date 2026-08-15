import { defineConfig } from 'astro/config';

// Dark Pawns — Astro migration (replaces the Hugo site in ../website).
export default defineConfig({
  site: 'https://darkpawns.labz0rz.com',
  // Share the generated world/database assets with the Hugo site while the
  // migration is in progress. The parsers continue to have one output tree.
  publicDir: '../website/static',
  redirects: {
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
  },
});
