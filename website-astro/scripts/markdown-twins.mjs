import { readdirSync, readFileSync, writeFileSync, existsSync } from 'node:fs';
import { join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';
import { parseHTML } from 'linkedom';
import TurndownService from 'turndown';
import { latestRelease } from './docs-navigation.mjs';
// All Starlight content, including plugin-generated API pages, gets a text twin.
export function markdownTwins() {
  return { name: 'darkpawns:markdown-twins', hooks: { 'astro:build:done': ({ dir }) => {
    const root = fileURLToPath(dir);
    const walk = (path) => readdirSync(path, { withFileTypes: true }).flatMap((f) => f.isDirectory() ? walk(join(path,f.name)) : [join(path,f.name)]);
    const markdown = new TurndownService({ headingStyle: 'atx', codeBlockStyle: 'fenced' });
    for(const file of existsSync(join(root,'docs')) ? walk(join(root,'docs')).filter((f)=>f.endsWith('index.html')) : []) {
      const html = readFileSync(file,'utf8');
      if(html.includes('http-equiv="refresh"')) continue;
      const route = relative(root,file).replace(/\/index\.html$/, '');
      const slug = route.replace(/^docs\/?/, '');
      const sourceRoot = fileURLToPath(new URL('../', import.meta.url));
      const candidates = [
        join(sourceRoot, 'src/content/docs', slug+'.md'),
        join(sourceRoot, 'src/content/docs', slug, 'index.md'),
        join(sourceRoot, '.generated/docs', slug+'.md'),
      ];
      const source = candidates.find((p) => existsSync(p));
      if (source) {
        const raw = readFileSync(source,'utf8');
        writeFileSync(file.replace(/\/index\.html$/,'.md'),raw.replace(/^---\n/,`---\ncanonical: https://darkpawns.org/${route}/\n`));
        continue;
      }
      const { document } = parseHTML(html);
      const main = document.querySelector('main');
      if(!main) continue;
      main.querySelectorAll('script,style,footer,.sr-only,.pagination-links,.sl-anchor-link,.meta').forEach((el)=>el.remove());
      const path = '/'+relative(root,file).replace(/index\.html$/,'');
      const title = document.querySelector('h1')?.textContent ?? 'Dark Pawns';
      const description = document.querySelector('meta[name="description"]')?.getAttribute('content') ?? '';
      const mdxSource = join(sourceRoot, 'src/content/docs', slug+'.mdx');
      const mdxFields = existsSync(mdxSource) ? readFileSync(mdxSource, 'utf8').match(/^---\n([\s\S]*?)\n---/)?.[1] : undefined;
      const body = markdown.turndown(main.innerHTML);
      if (mdxFields) {
        writeFileSync(file.replace(/\/index\.html$/, '.md'), `---\n${mdxFields}\ncanonical: https://darkpawns.org${path}\n---\n\n${body}\n`);
        continue;
      }
      writeFileSync(file.replace(/\/index\.html$/,'.md'),`---\ntitle: ${JSON.stringify(title)}\ndescription: ${JSON.stringify(description)}\ntype: reference\naudience: ${path.includes("/api/admin/") ? "operators" : "developers"}\nverified: ${latestRelease}\ncanonical: https://darkpawns.org${path}\n---\n\n${markdown.turndown(main.innerHTML)}\n`);
    }
    // llms.txt is rendered from Astro collections and the resolved sidebar.
    // Build the full bundle only after every authored/generated/API twin exists.
    const index = readFileSync(join(root, 'llms.txt'), 'utf8');
    const routes = [...new Set([...index.matchAll(/https:\/\/darkpawns\.org(\/docs(?:\/[^\s)#]*)?\.md)/g)].map(match => match[1]))];
    const chapters = routes.map(route => {
      const twin = join(root, route);
      if (!existsSync(twin)) throw new Error(`Agent index points at missing Markdown: ${route}`);
      return `<!-- Source: https://darkpawns.org${route} -->\n\n${readFileSync(twin, 'utf8').trim()}\n`;
    });
    writeFileSync(join(root, 'llms-full.txt'), '# Dark Pawns documentation\n\nGenerated in the same section and page order as /llms.txt. Source URLs identify each document.\n\n'+chapters.join('\n---\n\n'));
  } } };
}
