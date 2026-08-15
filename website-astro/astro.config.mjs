import { defineConfig } from 'astro/config';

// Dark Pawns — Astro migration (replaces the Hugo site in ../website).
export default defineConfig({
  site: 'https://darkpawns.labz0rz.com',
  redirects: {
    '/connect': '/play/',
    '/connect/client-downloads': '/play/#desktop-client',
    '/connect/connection-instructions': '/play/#desktop-client',
    '/connect/contact-info': '/play/',
    '/connect/external-links': '/archive/',
    '/changelog': '/',
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
