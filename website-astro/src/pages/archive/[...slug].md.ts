import type { APIRoute } from 'astro';
import { getCollection } from 'astro:content';
import { markdownResponse } from '../../lib/markdown';

export async function getStaticPaths() {
  const entries = await getCollection('archive', ({ data }) => !data.draft);
  const recovered = new Map(
    entries.filter((entry) => entry.data.topicId).map((entry) => [entry.data.topicId, entry]),
  );
  return entries.map((entry) => ({ params: { slug: entry.id }, props: { entry, recovered } }));
}

export const GET: APIRoute = ({ props }) => {
  const { entry, recovered } = props;
  const data = entry.data;
  const sections = [entry.body.trim()];

  // Some records carry their substance as structured frontmatter rather than
  // prose: a board index is a table of topics, a thread knows who spoke in it.
  // Rendering those here keeps the Markdown twin a faithful representation of
  // the page rather than a stub.
  if (data.participants?.length) {
    sections.unshift([
      '## In this conversation',
      '',
      '| Name | Role | Posts |',
      '| --- | --- | --- |',
      ...data.participants.map((p) => `| ${p.name} | ${p.role} | ${p.posts} |`),
    ].join('\n'));
  }
  if (data.topics?.length) {
    const held = data.topics.filter((t) => recovered.has(t.id)).length;
    sections.unshift([
      '## The board, as captured',
      '',
      `Every topic the board was carrying when it was archived. ${held} of these`,
      `${data.topics.length} have a recovered page; the rest are listed here and`,
      'nowhere else.',
      '',
      '| Topic | Started by | Replies | Recovered |',
      '| --- | --- | ---: | --- |',
      ...data.topics.map((t) => {
        const record = recovered.get(t.id);
        const link = record ? `/archive/${record.id}/` : 'no';
        return `| ${t.title} | ${t.author} | ${t.replies} | ${link} |`;
      }),
    ].join('\n'));
  }

  return markdownResponse(sections.filter(Boolean).join('\n\n'), {
    title: data.title,
    description: data.description,
    kind: data.kind,
    dateLabel: data.dateLabel,
    board: data.board,
    postCount: data.postCount,
    completeness: data.completeness,
    completenessNote: data.completenessNote,
    sourceSite: data.sourceSite,
    sourceUrl: data.sourceUrl,
    captureUrl: data.captureUrl,
    recoveredAt: data.recoveredAt,
    textKind: data.textKind,
  }, `/archive/${entry.id}/`);
};
