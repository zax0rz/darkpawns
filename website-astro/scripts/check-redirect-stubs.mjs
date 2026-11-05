// Internal links must point at the page, not at a refresh-redirect stub.
//
// Astro emits a 200 HTML page carrying <meta http-equiv="refresh"> for every
// entry in astro.config.mjs `redirects` (the retired /help and old docs URLs).
// A link that lands on one costs the reader a second navigation and search
// engines an extra hop, so the site links the destination directly. This
// scans the built HTML for any internal <a> that still resolves to a stub
// (specification.website/spec/seo/ — canonical URLs, no redirect chains).
//
// The site builds ~13k pages, so this reads with regexes rather than a DOM
// parser; only hrefs and the refresh meta matter here.
import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';

const dist = new URL('../dist/', import.meta.url).pathname;
const origin = 'https://darkpawns.org';
const walk = (dir) => readdirSync(dir, { withFileTypes: true }).flatMap((entry) => entry.isDirectory() ? walk(join(dir, entry.name)) : [join(dir, entry.name)]);
const toPath = (file) => '/' + file.slice(dist.length).replace(/index\.html$/, '');
const decode = (value) => { try { return decodeURIComponent(value); } catch { return value; } };

const htmlFiles = walk(dist).filter((file) => file.endsWith('.html'));

// Every refresh stub, keyed by the site path a link would use to reach it.
const stubs = new Map();
for (const file of htmlFiles) {
  const html = readFileSync(file, 'utf8');
  const refresh = html.match(/http-equiv=["']refresh["'][^>]*content=["'][^"']*url=([^"'\s>]+)/i);
  if (!refresh) continue;
  stubs.set(toPath(file).replace(/\/$/, '') || '/', decode(refresh[1]));
}

const problems = [];

// A stub must point at a real page, not at another stub: redirect chains cost
// a second hop (specification.website/spec/seo/ — one redirect, to the target).
const stubTargets = new Map([...stubs].map(([path, target]) => [path, decode(target.replace(/[#?].*$/, '').replace(/\/$/, '')) || '/']));
for (const [path, target] of stubTargets) {
  if (stubs.has(target)) problems.push(`${path} redirects to ${target}, which is itself a stub (real page ${stubs.get(target)})`);
}

let links = 0;
for (const file of htmlFiles) {
  const html = readFileSync(file, 'utf8');
  const from = toPath(file);
  for (const match of html.matchAll(/\shref=["']([^"']+)["']/g)) {
    const href = match[1];
    if (!href || /^(#|mailto:|tel:|javascript:|data:)/.test(href)) continue;
    let url;
    try { url = new URL(href, origin + from); } catch { continue; }
    if (url.origin !== origin) continue;
    links++;
    const path = decode(url.pathname).replace(/\/$/, '') || '/';
    if (stubs.has(path)) problems.push(`${from} links ${url.pathname} (a stub; the page is ${stubs.get(path)})`);
  }
}

if (problems.length) {
  console.error([...new Set(problems)].slice(0, 40).join('\n'));
  console.error(`check-redirect-stubs: ${problems.length} problems across ${stubs.size} stubs and ${links} internal links`);
  process.exit(1);
}
console.log(`check-redirect-stubs: ${stubs.size} stubs, ${links} internal links, no chains and no links to a stub`);
