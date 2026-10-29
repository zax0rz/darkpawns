import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs';
import { join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import assert from 'node:assert/strict';
import { parseHTML } from 'linkedom';
import { docsRedirects } from './docs-navigation.mjs';

const root = fileURLToPath(new URL('../../', import.meta.url));
const dist = join(root, 'website-astro/dist');
const walk = (path) => readdirSync(path, { withFileTypes: true }).flatMap((entry) => entry.isDirectory() ? walk(join(path, entry.name)) : [join(path, entry.name)]);
let pages = 0;
for (const file of walk(join(dist, 'docs')).filter((path) => path.endsWith('/index.html'))) {
  const html = readFileSync(file, 'utf8');
  if (html.includes('http-equiv="refresh"')) continue;
  const { document } = parseHTML(html);
  assert(document.querySelector('h1')?.textContent?.trim(), `${file}: missing title`);
  const description = document.querySelector('meta[name="description"]')?.getAttribute('content');
  assert(description?.length >= 120 && description.length <= 160, `${file}: missing or invalid description`);
  const twin = file.replace(/\/index\.html$/, '.md');
  assert(existsSync(twin), `${file}: missing Markdown twin`);
  const markdown = readFileSync(twin, 'utf8');
  for (const field of ['title', 'description', 'type', 'audience', 'verified', 'canonical']) assert(new RegExp(`^${field}:`, 'm').test(markdown), `${twin}: missing ${field}`);
  assert(document.querySelector('.last-verified'), `${file}: missing verification line`);
  const edit = [...document.querySelectorAll('a[href]')].find((a) => a.getAttribute('href').includes('/edit/main/'))?.getAttribute('href');
  assert(edit, `${file}: missing edit link`);
  const source = decodeURIComponent(edit.split('/edit/main/')[1]);
  assert(existsSync(resolve(root, source)) && statSync(resolve(root, source)).isFile(), `${file}: edit link points to absent source ${source}`);
  for (const anchor of document.querySelectorAll('main a[href^="/docs/"]')) {
    const url = new URL(anchor.getAttribute('href'), 'https://darkpawns.org');
    const target = join(dist, url.pathname);
    assert(existsSync(target) && (statSync(target).isFile() || existsSync(join(target, 'index.html'))), `${file}: broken docs link ${url.pathname}`);
  }
  pages++;
}
for (const [from, to] of Object.entries(docsRedirects)) {
  const redirectFile = existsSync(join(dist, from, 'index.html')) ? join(dist, from, 'index.html') : join(dist, from);
  const html = readFileSync(redirectFile, 'utf8');
  assert(html.includes(`url=${to}`), `${from}: missing redirect to ${to}`);
  assert(existsSync(to.endsWith('.md') ? join(dist, to) : join(dist, to, 'index.html')), `${from}: redirect target missing`);
}
const update = readFileSync(join(dist, 'docs/host/update-server/index.html'), 'utf8');
assert(update.includes('class="mermaid-diagram"') && update.includes('<svg'), 'Update guide: missing build-time diagram');
assert(!/<script[^>]+src=["'][^"']*mermaid/.test(update), 'Mermaid must not ship as a browser runtime');
assert(existsSync(join(dist, 'pagefind/pagefind.js')), 'Missing Pagefind index');
assert(readFileSync(join(dist, '404.html'), 'utf8').includes('data-search-trigger'), '404 needs a search action');
console.log(`docs-check: ${pages} pages have titles, descriptions, Markdown twins, verification lines, valid source edit links and docs links; ${Object.keys(docsRedirects).length} redirects, search and diagram present`);
