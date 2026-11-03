import type { APIRoute } from 'astro';
import { readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { getCollection } from 'astro:content';
import { markdownResponse } from '../../../lib/markdown';

export async function getStaticPaths() {
  const entries = await getCollection('logs', ({ data }) => !data.draft);
  return entries.map((entry) => ({ params: { slug: entry.id }, props: { entry } }));
}

// The Markdown twin carries the entry's own words and the plain scrubbed
// capture, so an agent or a reader gets the same text the page shows without
// the rebuilt color.
export const GET: APIRoute = ({ props }) => {
  const { entry } = props;
  const data = entry.data;
  const capture = readFileSync(resolve(dirname(resolve(entry.filePath!)), data.log), 'latin1').replace(/\r\n?/g, '\n').trimEnd();
  const body = [
    entry.body.trim(),
    '',
    '## The capture',
    '',
    `\`\`\``,
    capture,
    `\`\`\``,
  ].join('\n');

  return markdownResponse(body, {
    title: data.title,
    description: data.description,
    recordedBy: data.recordedBy || 'not identified',
    dateLabel: data.dateLabel,
    dateSource: data.dateSource,
    collection: data.collection,
    sourceFile: data.sourceFile,
    sourceLines: data.sourceLines,
    cast: data.cast.join(', '),
    redactions: Object.entries(data.redactions).filter(([, n]) => n > 0).map(([k, n]) => `${n} ${k}`).join(', ') || 'none',
  }, `/archive/logs/${entry.id}/`);
};
