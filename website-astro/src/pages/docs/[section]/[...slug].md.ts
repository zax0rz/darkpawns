import type { APIRoute } from 'astro';
import { getCollection } from 'astro:content';
import { markdownResponse } from '../../../lib/markdown';

export async function getStaticPaths() {
  const entries = await getCollection('docs', ({ data }) => !data.draft);
  return entries.map((entry) => {
    const [section, ...slug] = entry.id.split('/');
    return { params: { section, slug: slug.join('/') }, props: { entry } };
  });
}

export const GET: APIRoute = ({ props }) => {
  const { entry } = props;
  return markdownResponse(entry.body, {
    title: entry.data.title,
    description: entry.data.description,
    audience: entry.data.audience,
    updated: entry.data.updated,
    sourcePath: entry.data.sourcePath,
  }, `/docs/${entry.id}/`);
};
