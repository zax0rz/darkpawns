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
  }),
});

export const collections = { blog, world, help };
