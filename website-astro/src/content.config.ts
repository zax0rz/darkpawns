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

export const collections = { blog };
