import { defineCollection, z } from 'astro:content';
import { glob, file } from 'astro/loaders';
import { docsSchema, i18nSchema } from '@astrojs/starlight/schema';
import { i18nLoader } from '@astrojs/starlight/loaders';
import release from '../.generated/release.json';

// The blog collection. This schema is enforced at build time: a post missing a
// title, a bad date, or a wrong type FAILS the build. That is the frontmatter
// slop across the old Hugo content getting killed structurally.
const blog = defineCollection({
  loader: glob({ pattern: '**/*.md', base: './src/content/blog' }),
  schema: z.object({
    title: z.string(),
    date: z.coerce.date(),
    description: z.string(),
    draft: z.boolean().default(false),
    textKind: z.enum(['original', 'summary', 'reconstruction', 'transcription', 'verbatim', 'edited-excerpt']),
    source: z.string(),
    voiceLayer: z.enum(['engine', 'mythic-admin', 'frontline']),
    // A 1200x630 share card under /images/blog/, with its alt text. Posts
    // without one share the site-wide /og-image.png.
    ogImage: z.string().startsWith('/images/blog/').optional(),
    ogImageAlt: z.string().optional(),
  }).refine((data) => !data.ogImage || data.ogImageAlt, { message: 'ogImage needs ogImageAlt', path: ['ogImageAlt'] }),
});

// The world handbook: classes, races, skills, systems, lore. Same schema
// discipline as the blog — every page needs a title and description.
const world = defineCollection({
  loader: glob({ pattern: '**/*.md', base: './src/content/world' }),
  schema: z.object({
    title: z.string(),
    description: z.string(),
    date: z.coerce.date().optional(),
    draft: z.boolean().default(false),
    // Set when this page still reproduces an archive entry verbatim. The archive
    // entry is the canonical copy until this page is rewritten as handbook text,
    // so the two do not compete as duplicates.
    canonicalPath: z.string().optional(),
  }),
});

// The in-game help archive (~425 files across commands, spells, info, socials,
// wizhelp). _index category pages are excluded; category pages are built in
// src/pages/help/. Same schema discipline.
const help = defineCollection({
  loader: glob({ pattern: ['**/*.md', '!**/_index.md'], base: './src/content/help' }),
  schema: z.object({
    title: z.string(),
    description: z.string(),
    date: z.coerce.date().optional(),
    draft: z.boolean().default(false),
    textKind: z.literal('verbatim').default('verbatim'),
  }),
});

// Curated primary-source material recovered from the original community sites.
// Every published item must identify what it is, when it originally appeared,
// and the archived capture used to verify it.
const archive = defineCollection({
  loader: glob({ pattern: '**/*.md', base: './src/content/archive' }),
  schema: z.object({
    title: z.string(),
    description: z.string(),
    kind: z.enum(['forum-thread', 'usenet-thread', 'history', 'guide', 'quote', 'roster', 'site-page', 'map', 'board-index']),
    sortDate: z.coerce.date(),
    dateLabel: z.string(),
    publishedAt: z.coerce.date().optional(),
    sourceSite: z.enum(['dp-players.com', 'darkpawns.com', 'Usenet']),
    sourceUrl: z.string().url(),
    captureUrl: z.string().url(),
    recoveredAt: z.coerce.date(),
    contentWarning: z.string().optional(),
    draft: z.boolean().default(false),
    // What survived. A capture is often one page of a longer thread, and the
    // archive says so rather than presenting a fragment as the whole record.
    completeness: z.enum(['complete', 'partial']).default('complete'),
    completenessNote: z.string().optional(),
    // Forum threads only: which board the conversation lived on, how many posts
    // the capture holds, and who spoke. Ranks are the public forum ranks.
    board: z.string().optional(),
    messageId: z.string().optional(),
    crossposts: z.array(z.object({
      newsgroup: z.string(),
      messageId: z.string(),
      captureUrl: z.string().url(),
    })).optional(),
    // phpBB topic number. It is what links a recovered thread back to the row
    // for it on the board index, so the index can show what survived.
    topicId: z.number().int().positive().optional(),
    postCount: z.number().int().positive().optional(),
    participants: z.array(z.object({
      name: z.string(),
      role: z.string().default('unknown'),
      posts: z.number().int().positive(),
    })).optional(),
    // Where this primary source is used elsewhere on the site, so an edited
    // page and the record behind it stay visibly connected.
    usedIn: z.array(z.object({ label: z.string(), href: z.string() })).optional(),
    // A board index lists every topic the board held on the day it was
    // captured, with the reply count for each. Most of those topics were never
    // archived, so this is the measure of what is gone.
    topics: z.array(z.object({
      id: z.number().int().positive(),
      title: z.string(),
      author: z.string(),
      replies: z.number().int().nonnegative(),
    })).optional(),
    textKind: z.enum(['verbatim', 'transcription', 'edited-excerpt']),
    source: z.string(),
    voiceLayer: z.enum(['engine', 'edgelord-dm', 'mythic-admin', 'frontline']),
  }),
});

// Technical manuals for operators, contributors, and agent authors. Player
// commands belong in /help; world material belongs in /world.
// Validate authored source before Starlight supplies defaults to plugin pages.
// OpenAPI's StarlightPage component passes only its generated title.
const docsFields = z.object({
  title: z.string().min(1).max(59),
  description: z.string().min(120).max(160),
  type: z.enum(['guide', 'explainer', 'reference', 'troubleshooting', 'release']),
  audience: z.enum(['players', 'operators', 'builders', 'developers']),
  verified: z.string().regex(/^v\d+\.\d+\.\d+$/),
  sourcePath: z.string().optional(),
});
const authoredDocs = glob({
  pattern: '**/*.{md,mdx}', base: './src/content/docs',
  generateId: ({ entry }) => ('docs/' + entry.replace(/\.(md|mdx)$/, '').replace(/(^|\/)index$/, '')).replace(/\/$/, ''),
});
const docs = defineCollection({
  loader: { ...authoredDocs, name: 'strict-authored-docs', async load(context) {
    return authoredDocs.load({ ...context, parseData: async (entry) => {
      docsFields.parse(entry.data);
      return context.parseData(entry);
    } });
  } },
  schema: docsSchema({ extend: docsFields.extend({
    description: docsFields.shape.description.default('Look up the Dark Pawns HTTP API request and response contract. This reference is generated from the same registered operations as the running server.'),
    type: docsFields.shape.type.default('reference'),
    audience: docsFields.shape.audience.default('developers'),
    verified: docsFields.shape.verified.default(release.version),
  }) }),
});
const generatedDocs = defineCollection({
  loader: glob({ pattern: '**/*.md', base: './.generated/docs' }),
  schema: z.object({ title: z.string().min(1).max(59), description: z.string().min(120).max(160), type: z.enum(['reference','release']), audience: z.enum(['players','operators','developers']), verified: z.string().regex(/^v\d+\.\d+\.\d+$/), sourcePath: z.string() }),
});

const i18n = defineCollection({ loader: i18nLoader(), schema: i18nSchema() });

const docsApiCatalogue = defineCollection({ loader: file('./.generated/api/catalogue.json'), schema: docsFields });

export const collections = { docsApiCatalogue, blog, world, help, archive, docs, generatedDocs, i18n };
