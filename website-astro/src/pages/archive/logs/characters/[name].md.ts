import type { APIRoute } from 'astro';
import { getCollection } from 'astro:content';
import { markdownResponse } from '../../../../lib/markdown';
import { appearancesOf, characterKeys } from '../../../../lib/mudlog/entry';

export async function getStaticPaths() {
  return characterKeys(await getCollection('logs')).map((name) => ({ params: { name }, props: { name } }));
}

// The Markdown twin lists the same logs the page does, with the line each
// one first shows the character.
export const GET: APIRoute = async ({ props }) => {
  const key = props.name as string;
  const appears = appearancesOf(await getCollection('logs'), key);
  const display = appears[0]?.name ?? key;
  const summary = `${display} appears in ${appears.length} ${appears.length === 1 ? 'log' : 'logs'} from the original Dark Pawns server.`;
  const body = [
    summary,
    '',
    ...appears.map(({ entry, line, recorded }) =>
      `- [${entry.data.title}](https://darkpawns.org/archive/logs/${entry.id}/#L${line}): ${recorded ? `recorded by ${display}` : `first on screen at line ${line}`}`),
  ].join('\n');

  return markdownResponse(body, { title: `${display} in the logs`, description: summary }, `/archive/logs/characters/${key}/`);
};
