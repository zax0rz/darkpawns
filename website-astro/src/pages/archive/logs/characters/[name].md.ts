import type { APIRoute } from 'astro';
import { getCollection } from 'astro:content';
import { markdownResponse } from '../../../../lib/markdown';

export async function getStaticPaths() {
  const entries = await getCollection('logs', ({ data }) => !data.draft);
  const keys = new Set<string>();
  for (const entry of entries) for (const name of entry.data.cast) keys.add(name.toLowerCase());
  return [...keys].map((name) => ({ params: { name }, props: { name } }));
}

// The Markdown twin lists the same sessions the page does, with the line each
// one first shows the character.
export const GET: APIRoute = async ({ props }) => {
  const key = props.name as string;
  const entries = await getCollection('logs', ({ data }) => !data.draft);
  const appears = entries
    .filter((entry) => entry.data.cast.some((name) => name.toLowerCase() === key))
    .map((entry) => {
      const name = entry.data.cast.find((n) => n.toLowerCase() === key)!;
      return { entry, line: entry.data.appearances[name] ?? entry.data.sourceStart };
    })
    .sort((a, b) => a.entry.data.title.localeCompare(b.entry.data.title));

  const display = appears.length
    ? appears[0].entry.data.cast.find((n) => n.toLowerCase() === key)!
    : key;
  const body = [
    `${display} appears in ${appears.length} ${appears.length === 1 ? 'session' : 'sessions'} of the Dark Pawns logs.`,
    '',
    ...appears.map(({ entry, line }) =>
      `- [${entry.data.title}](https://darkpawns.org/archive/logs/${entry.id}/#L${line}) · ${entry.data.dateLabel}, first at line ${line}`),
  ].join('\n');

  return markdownResponse(body, {
    title: `${display} in the logs`,
    description: `${display} appears in the Dark Pawns logs captured on the original server.`,
  }, `/archive/logs/characters/${key}/`);
};
