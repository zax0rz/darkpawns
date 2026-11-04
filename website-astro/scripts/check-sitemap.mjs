// Every URL the sitemap lists must be a page a search engine should index:
// built, its own canonical, not noindex, not a refresh-redirect stub. Search
// Console reports each of those as an error against the sitemap, and the
// spec asks for canonical URLs only (specification.website/spec/seo/xml-sitemaps/).
import { existsSync, readFileSync } from 'node:fs';
import { join } from 'node:path';

const dist = new URL('../dist/', import.meta.url).pathname;
const site = 'https://darkpawns.org';
const locs = (file) => [...readFileSync(join(dist, file), 'utf8').matchAll(/<loc>([^<]+)<\/loc>/g)].map((m) => m[1]);

const problems = [];
let count = 0;
for (const sitemap of locs('sitemap-index.xml')) {
  const file = new URL(sitemap).pathname;
  if (!existsSync(join(dist, file))) {
    problems.push(`missing sitemap ${file}`);
    continue;
  }
  for (const url of locs(file)) {
    count++;
    const path = new URL(url).pathname;
    const html = join(dist, path, path.endsWith('/') ? 'index.html' : '');
    if (!url.startsWith(`${site}/`) || !path.endsWith('/')) problems.push(`not a canonical page URL: ${url}`);
    if (!existsSync(html)) { problems.push(`not built: ${url}`); continue; }
    const head = readFileSync(html, 'utf8').slice(0, 20000);
    const canonical = head.match(/<link rel="canonical" href="([^"]+)"/)?.[1];
    if (canonical !== url) problems.push(`canonical is ${canonical ?? 'missing'}: ${url}`);
    if (/<meta name="robots" content="[^"]*noindex/.test(head)) problems.push(`noindex: ${url}`);
    if (/http-equiv="refresh"/.test(head)) problems.push(`redirect stub: ${url}`);
  }
}

if (problems.length) {
  console.error(problems.slice(0, 40).join('\n'));
  console.error(`check-sitemap: ${problems.length} problems in ${count} URLs`);
  process.exit(1);
}
console.log(`check-sitemap: ${count} URLs, all canonical and indexable`);
