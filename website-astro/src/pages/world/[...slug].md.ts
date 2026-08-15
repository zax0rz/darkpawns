import type { APIRoute } from 'astro';
import { getCollection } from 'astro:content';
import { markdownResponse } from '../../lib/markdown';

export async function getStaticPaths() {
  const entries = await getCollection('world', ({ data }) => !data.draft);
  return entries.map((entry) => ({ params: { slug: entry.id }, props: { entry } }));
}

export const GET: APIRoute = ({ props }) => {
  const { entry } = props;
  return markdownResponse(entry.body, {
    title: entry.data.title,
    description: entry.data.description,
  }, `/world/${entry.id}/`);
};
