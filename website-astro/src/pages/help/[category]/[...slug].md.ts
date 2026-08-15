import type { APIRoute } from 'astro';
import { getCollection } from 'astro:content';
import { markdownResponse } from '../../../lib/markdown';

export async function getStaticPaths() {
  const entries = await getCollection('help');
  return entries.map((entry) => {
    const [category, ...slug] = entry.id.split('/');
    return { params: { category, slug: slug.join('/') }, props: { entry } };
  });
}

export const GET: APIRoute = ({ props }) => {
  const { entry } = props;
  return markdownResponse(entry.body, {
    title: entry.data.title,
    description: entry.data.description,
    textKind: entry.data.textKind,
  }, `/help/${entry.id}/`);
};
