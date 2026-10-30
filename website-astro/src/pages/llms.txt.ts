import { documentationSections } from '../lib/agent-docs';
import guidance from '../data/agent-guidance.md?raw';
export const prerender = true;
export async function GET() {
  const sections = await documentationSections();
  const line = (text: string) => text.replace(/\s+/g, ' ').trim();
  const index = sections.map(section => `## ${section.title}\n\n` + section.pages.map(page => `- [${line(page.title).replaceAll(']','\\]')}](https://darkpawns.org${page.markdown}): ${line(page.description)}`).join('\n')).join('\n\n');
  return new Response(guidance.trim()+'\n\n## Documentation\n\nGenerated from the documentation collections in sidebar order. Every link below opens the page’s Markdown copy. [All documentation in one file](https://darkpawns.org/llms-full.txt).\n\n'+index+'\n', {headers:{'Content-Type':'text/plain; charset=utf-8'}});
}
