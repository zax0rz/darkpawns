import type { APIRoute } from 'astro';
import { readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { getCollection } from 'astro:content';
import { markdownResponse } from '../../../lib/markdown';
import { describeLog, recorders, redactionSummary } from '../../../lib/mudlog/entry';

export async function getStaticPaths() {
  const entries = await getCollection('logs');
  return entries.map((entry) => ({ params: { slug: entry.id }, props: { entry } }));
}

// The Markdown twin carries the plain scrubbed capture, so an agent or a
// reader gets the same text the page shows without the rebuilt color.
export const GET: APIRoute = ({ props }) => {
  const { entry } = props;
  const data = entry.data;
  const capture = readFileSync(resolve(dirname(resolve(entry.filePath!)), data.log), 'utf8').trimEnd();
  const body = [describeLog(data, Infinity), '', '```', capture, '```'].join('\n');

  return markdownResponse(body, {
    title: data.title,
    description: describeLog(data),
    recordedBy: recorders(data),
    lines: String(data.lines),
    collection: 'The Vargus logs, donated by Vargus',
    cast: data.cast.join(', '),
    removed: redactionSummary(data.redactions) || 'nothing',
  }, `/archive/logs/${entry.id}/`);
};
