import { defineCollection, z } from 'astro:content';
import { glob } from 'astro/loaders';

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
  }),
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
    kind: z.enum(['forum-thread', 'history', 'guide', 'quote', 'roster', 'site-page', 'map', 'board-index']),
    sortDate: z.coerce.date(),
    dateLabel: z.string(),
    publishedAt: z.coerce.date().optional(),
    sourceSite: z.enum(['dp-players.com', 'darkpawns.com']),
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
const docs = defineCollection({
  loader: glob({ pattern: '**/*.md', base: './src/content/docs' }),
  schema: z.object({
    title: z.string(),
    description: z.string(),
    section: z.enum(['getting-started', 'server', 'agents', 'research']),
    audience: z.enum(['operator', 'developer', 'agent-author', 'researcher']),
    order: z.number().int().nonnegative(),
    sourcePath: z.string(),
    updated: z.coerce.date(),
    draft: z.boolean().default(false),
  }),
});

export const collections = { blog, world, help, archive, docs };
