import { documentationSections } from '../lib/agent-docs';
import { getCollection } from 'astro:content';
import { describeLog } from '../lib/mudlog/entry';
import guidance from '../data/agent-guidance.md?raw';
export const prerender = true;
export async function GET() {
  const sections = await documentationSections();
  const line = (text: string) => text.replace(/\s+/g, ' ').trim();
  const index = sections.map(section => `## ${section.title}\n\n` + section.pages.map(page => `- [${line(page.title).replaceAll(']','\\]')}](https://darkpawns.org${page.markdown}): ${line(page.description)}`).join('\n')).join('\n\n');

  // Game logs and the characters in them. Every entry has a Markdown twin that
  // carries the plain scrubbed capture, so the whole log set is readable as
  // text.
  const logs = (await getCollection('logs'))
    .sort((a, b) => a.data.title.localeCompare(b.data.title, 'en'));
  const characters = new Set<string>();
  for (const entry of logs) for (const name of entry.data.cast) characters.add(name.toLowerCase());
  const logSection = [
    '## Game logs',
    '',
    `Sessions captured on the original Dark Pawns server by the players who were there, donated by Vargus: ${logs.length} capture files, one page each. Private tells, mail, contact details, slurs and sexual violence are removed; each page counts what it removed.`,
    '',
    `- [Characters in the logs](https://darkpawns.org/archive/logs/characters/): all ${characters.size} characters named in a capture, with the logs each appears in.`,
    ...logs.map(entry => `- [${line(entry.data.title).replaceAll(']','\\]')}](https://darkpawns.org/archive/logs/${entry.id}.md): ${line(describeLog(entry.data))}`),
  ].join('\n');

  return new Response(guidance.trim()+'\n\n## Documentation\n\nGenerated from the documentation collections in sidebar order. Every link below opens the page’s Markdown copy. [All documentation in one file](https://darkpawns.org/llms-full.txt).\n\n'+index+'\n\n'+logSection+'\n', {headers:{'Content-Type':'text/plain; charset=utf-8'}});
}
