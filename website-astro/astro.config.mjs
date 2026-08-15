import { defineConfig } from 'astro/config';
import sitemap from '@astrojs/sitemap';
import { readdirSync } from 'node:fs';

const helpRoot = new URL('./src/content/help/', import.meta.url);
const helpRedirects = Object.fromEntries(
  readdirSync(helpRoot, { withFileTypes: true })
    .filter((entry) => entry.isDirectory())
    .flatMap((category) =>
      readdirSync(new URL(`./${category.name}/`, helpRoot), { withFileTypes: true })
        .filter((entry) => entry.isFile() && entry.name.endsWith('.md'))
        .map((entry) => {
          const slug = entry.name.slice(0, -3);
          return [`/help/${slug}`, `/help/${category.name}/${slug}/`];
        }),
    ),
);

const redirects = {
  ...helpRedirects,
  '/help/!': '/help/commands/bang-caret/',
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
  '/research/agent-protocols': '/docs/agents/protocol/',
  '/research/narrative-memory': '/docs/agents/memory/',
  '/research/port-fidelity': '/docs/research/port-fidelity/',
  '/docs/game': '/world/',
  '/docs/game/commands': '/world/commands/',
  '/docs/game/mechanics': '/world/mechanics/',
  '/docs/game/progression': '/world/progression/',
  '/docs/game/zones': '/world/zones/',
  '/lore': '/world/lore/',
  '/lore/world-creation': '/world/world-creation/',
  '/world/classes/classes': '/world/classes/',
  '/world/lore/world-creation': '/world/world-creation/',
  '/world/races/races': '/world/races/',
  '/world/skills/skills': '/world/skills/',
  '/news': '/blog/',
  '/news/historical-posts': '/blog/historical-posts/',
  '/news/resurrection': '/blog/resurrection/',
  '/news/website-launch': '/blog/website-launch/',
  '/community': '/archive/',
  '/community/equipment': '/database/',
  '/community/equipment/equipment-database': '/database/',
  '/community/forums': '/archive/',
  '/community/forums/forum-statistics': '/archive/',
  '/community/forums/topic-705': '/archive/topic-705/',
  '/community/forums/topic-724': '/archive/topic-724/',
  '/community/forums/topic-731': '/archive/topic-731/',
  '/community/forums/topic-737': '/archive/topic-737/',
  '/community/guides': '/help/',
  '/community/guides/articles': '/help/',
  '/community/history': '/archive/history/',
  '/community/history/game-logs': '/archive/',
  '/community/history/timeline': '/archive/history/',
  '/community/quotes': '/archive/player-quotes/',
  '/community/quotes/player-photos': '/archive/',
  '/community/quotes/player-quotes': '/archive/player-quotes/',
  '/categories': '/blog/',
  '/categories/game': '/blog/',
  '/tags': '/blog/',
  '/tags/game': '/blog/',
  '/tags/interactive': '/map/',
  '/tags/map': '/map/',
  '/tags/world': '/world/',
  '/pawn-lab.html': '/docs/research/',
  '/status': '/about/project/',
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
