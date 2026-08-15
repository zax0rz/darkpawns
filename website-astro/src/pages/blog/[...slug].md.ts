import type { APIRoute } from 'astro';
import { getCollection } from 'astro:content';
import { markdownResponse } from '../../lib/markdown';

export async function getStaticPaths() {
  const entries = await getCollection('blog', ({ data }) => !data.draft);
  return entries.map((entry) => ({ params: { slug: entry.id }, props: { entry } }));
}

export const GET: APIRoute = ({ props }) => {
  const { entry } = props;
  return markdownResponse(entry.body, {
    title: entry.data.title,
    description: entry.data.description,
    date: entry.data.date,
    textKind: entry.data.textKind,
    source: entry.data.source,
  }, `/blog/${entry.id}/`);
};
