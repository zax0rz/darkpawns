import { docsHelpRedirects } from './scripts/command-taxonomy.mjs';
import { defineConfig } from 'astro/config';
import sitemap from '@astrojs/sitemap';
import starlight from '@astrojs/starlight';
import starlightOpenAPI, { openAPISidebarGroups } from 'starlight-openapi';
import mermaidBuild from './src/lib/mermaid-build.mjs';
import { sidebar, docsRedirects } from './scripts/docs-navigation.mjs';
import { markdownTwins } from './scripts/markdown-twins.mjs';
import { satteri } from '@astrojs/markdown-satteri';
import { noLiteralAutolinks } from './src/lib/no-literal-autolinks.mjs';
import { readdirSync } from 'node:fs';

const redirects = Object.fromEntries(Object.entries({
  ...docsRedirects,
  ...docsHelpRedirects(),
  '/mudlet': '/connect/#mudlet',
  '/connect/client-downloads': '/connect/',
  '/connect/connection-instructions': '/connect/',
  '/connect/contact-info': '/connect/',
  '/connect/external-links': '/archive/',
  '/changelog': '/',
  '/about/features': '/world/',
  '/about/site-history': '/archive/',
  '/about/archive-index': '/archive/',
  '/credits/wizlist': '/credits/#archived-staff-list',
  '/credits/player-roster': '/archive/',
  '/docs/getting-started/quick-start': '/docs/build/local-server/',
  '/docs/getting-started/installation': '/docs/host/install/',
  '/docs/development': '/docs/project/',
  '/docs/development/architecture': '/docs/project/architecture/',
  '/research': '/docs/project/',
  '/research/port-fidelity': '/docs/project/port-fidelity/',
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
  '/news/website-launch': '/blog/resurrection/',
  '/blog/website-launch': '/blog/resurrection/',
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
  '/pawn-lab.html': '/docs/project/',
  '/status': '/about/project/',
}).map(([from, to]) => [from.replace(/\/$/, '') || '/', to]));

// Dark Pawns — Astro migration (replaces the Hugo site in ../website).
export default defineConfig({
  site: 'https://darkpawns.org',
  markdown: {
    // Smart punctuation is off site-wide. Almost every Markdown file here is
    // verbatim text somebody else wrote: 430 help entries copied out of the
    // game, the recovered archive captures, the world handbook pages. Curling
    // an apostrophe that a player typed straight in 2004, or turning their
    // "..." into an ellipsis, edits the record. The three original blog posts
    // lose typographic quotes, which is the cheaper loss.
    //
    // Bare URLs stay plain text for the same reason: see
    // src/lib/no-literal-autolinks.mjs.
    processor: satteri({
      features: { smartPunctuation: false },
      mdastPlugins: [noLiteralAutolinks, mermaidBuild],
    }),
  },
  integrations: [
    starlight({
      title: 'Dark Pawns',
      description: 'The field guide for playing, hosting and building Dark Pawns.',
      disable404Route: true,
      routeMiddleware: './src/docs-route-data.ts',
      sidebar: sidebar(openAPISidebarGroups),
      customCss: ['./src/styles/docs.css'],
      components: {
        Header: './src/components/docs/Header.astro', MobileMenuToggle: './src/components/docs/MobileMenuToggle.astro',
        Head: './src/components/docs/Head.astro',
        EditLink: './src/components/docs/EditLink.astro',
        Footer: './src/components/docs/Footer.astro',
        ThemeProvider: './src/components/docs/Theme.astro',
        ThemeSelect: './src/components/docs/Empty.astro',
        PageTitle: './src/components/docs/PageTitle.astro',
        LastUpdated: './src/components/docs/LastVerified.astro',
      },
      editLink: { baseUrl: 'https://github.com/zax0rz/darkpawns/edit/main/website-astro/' },
      credits: false,
      expressiveCode: { themes: ['github-light'], styleOverrides: { borderRadius: '2px', codeBackground: '#E5DAC1', codeFontFamily: 'JetBrains Mono, monospace' } },
      plugins: [starlightOpenAPI([
        { base: 'docs/reference/api/public', schema: './.generated/api/public.json', sidebar: { label: 'Public API' } },
        { base: 'docs/reference/api/admin', schema: './.generated/api/admin.json', sidebar: { label: 'Admin API' } },
      ])],
    }),
    markdownTwins(),
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
