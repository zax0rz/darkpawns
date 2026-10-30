import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs';
import { join, resolve, relative } from 'node:path';
import { fileURLToPath } from 'node:url';
import assert from 'node:assert/strict';
import { parseHTML } from 'linkedom';
import { docsRedirects } from './docs-navigation.mjs';
import { docsHelpRedirects } from './command-taxonomy.mjs';

const root = fileURLToPath(new URL('../../', import.meta.url));
const dist = join(root, 'website-astro/dist');
const walk = (path) => readdirSync(path, { withFileTypes: true }).flatMap((entry) => entry.isDirectory() ? walk(join(path, entry.name)) : [join(path, entry.name)]);
const llms = readFileSync(join(dist, 'llms.txt'), 'utf8');
const full = readFileSync(join(dist, 'llms-full.txt'), 'utf8');
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
  const markdownPath = '/'+relative(dist, twin);
  const markdownUrl = 'https://darkpawns.org'+markdownPath;
  const alternate = document.querySelector('head link[rel="alternate"][type="text/markdown"]');
  assert(alternate, `${file}: missing Markdown alternate link`);
  assert.equal(new URL(alternate.getAttribute('href'), 'https://darkpawns.org').pathname, markdownPath, `${file}: Markdown alternate points at the wrong twin`);
  assert(llms.includes(']('+markdownUrl+'): '), `${file}: published docs page missing from llms.txt`);
  assert(llms.includes(']('+markdownUrl+'): '+description.replace(/\s+/g,' ').trim()), `${file}: agent summary differs from page description`);
  assert(full.includes('<!-- Source: '+markdownUrl+' -->'), `${file}: missing from llms-full.txt`);
  assert(document.querySelector(`a[href="${markdownPath}"][type="text/markdown"]`)?.textContent.includes('View as Markdown'), `${file}: missing visible Markdown link`);

  const markdown = readFileSync(twin, 'utf8');
  assert(full.includes(markdown.trim()), `${twin}: full bundle doesn't include the Markdown content`);
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
const redirects = { ...docsRedirects, ...docsHelpRedirects() };
const targetDocuments = new Map();
for (const [from, to] of Object.entries(redirects)) {
  const redirectFile = existsSync(join(dist, from, 'index.html')) ? join(dist, from, 'index.html') : join(dist, from);
  const html = readFileSync(redirectFile, 'utf8');
  assert(html.includes(`url=${to}`), `${from}: missing redirect to ${to}`);
  const url = new URL(to, 'https://darkpawns.org');
  if(url.origin !== 'https://darkpawns.org') continue;
  const target = url.pathname.endsWith('.md') ? join(dist,url.pathname) : join(dist,url.pathname,'index.html');
  assert(existsSync(target), `${from}: redirect target missing ${url.pathname}`);
  if(url.hash) {
    if(!targetDocuments.has(target)) targetDocuments.set(target, url.pathname.endsWith('.md') ? readFileSync(target,'utf8') : parseHTML(readFileSync(target,'utf8')).document);
    const doc=targetDocuments.get(target); const id=decodeURIComponent(url.hash.slice(1));
    assert(typeof doc==='string' ? doc.includes(`id="${id}"`) : doc.getElementById(id), `${from}: missing target anchor ${id}`);
  }
}
const update = readFileSync(join(dist, 'docs/host/update-server/index.html'), 'utf8');
assert(update.includes('class="mermaid-diagram"') && update.includes('<svg'), 'Update guide: missing build-time diagram');
assert(!/<script[^>]+src=["'][^"']*mermaid/.test(update), 'Mermaid must not ship as a browser runtime');
assert(existsSync(join(dist, 'pagefind/pagefind.js')), 'Missing Pagefind index');
assert(readFileSync(join(dist, '404.html'), 'utf8').includes('data-search-trigger'), '404 needs a search action');
console.log(`docs-check: ${pages} pages have titles, descriptions, Markdown twins, verification lines, valid source edit links and docs links; ${Object.keys(redirects).length} redirects; llms indexes, Markdown discovery, search and diagram present`);
