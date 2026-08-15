import type { APIRoute } from 'astro';
import { getCollection } from 'astro:content';
import { markdownResponse } from '../../lib/markdown';

export async function getStaticPaths() {
  const entries = await getCollection('archive', ({ data }) => !data.draft);
  return entries.map((entry) => ({ params: { slug: entry.id }, props: { entry } }));
}

export const GET: APIRoute = ({ props }) => {
  const { entry } = props;
  return markdownResponse(entry.body, {
    title: entry.data.title,
    description: entry.data.description,
    dateLabel: entry.data.dateLabel,
    sourceSite: entry.data.sourceSite,
    sourceUrl: entry.data.sourceUrl,
    captureUrl: entry.data.captureUrl,
    textKind: entry.data.textKind,
  }, `/archive/${entry.id}/`);
};
