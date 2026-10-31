import assert from 'node:assert/strict';
import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { commandGroups, docsHelpRedirects } from './command-taxonomy.mjs';

const site = fileURLToPath(new URL('../', import.meta.url));
const groups = commandGroups();
const topics = groups.flatMap((group) => group.topics);
const helpRoot = join(site, 'src/content/help');
const rawHelpRoot = fileURLToPath(new URL('../../lib/text/help/', import.meta.url));
const sourceCount = readdirSync(helpRoot, { withFileTypes: true })
  .filter((entry) => entry.isDirectory())
  .flatMap((entry) => readdirSync(join(helpRoot, entry.name)))
  .filter((file) => file.endsWith('.md') && file !== '_index.md').length;
const redirects = docsHelpRedirects();
const seen = new Map();

function fenceFor(body) {
  const longest = Math.max(0, ...[...body.matchAll(/`+/g)].map(([fence]) => fence.length));
  return '`'.repeat(Math.max(3, longest + 1));
}

function rawRecords(path) {
  const text = readFileSync(path, 'utf8');
  const records = [];
  let pos = 0;
  while (pos < text.length) {
    let end = text.indexOf('\n', pos);
    if (end < 0) end = text.length;
    const keyLine = text.slice(pos, end).replace(/\r$/, '');
    pos = end < text.length ? end + 1 : end;
    if (!keyLine || keyLine.startsWith('$')) break;
    const bodyStart = pos;
    let terminated = false;
    while (pos < text.length) {
      end = text.indexOf('\n', pos);
      if (end < 0) end = text.length;
      if (text.slice(pos, end).replace(/\r$/, '').startsWith('#')) {
        records.push({ keyLine, body: text.slice(bodyStart, pos) });
        pos = end < text.length ? end + 1 : end;
        terminated = true;
        break;
      }
      pos = end < text.length ? end + 1 : end;
    }
    assert(terminated, `${path} has an unterminated record: ${keyLine}`);
  }
  return records;
}

const rawByRecord = new Map();
for (const category of ['commands', 'info', 'socials', 'spells', 'wizhelp']) {
  for (const record of rawRecords(join(rawHelpRoot, `${category}.hlp`))) {
    const key = `${category}:${record.keyLine}`;
    assert(!rawByRecord.has(key), `${key} must be unique in raw help`);
    rawByRecord.set(key, record.body);
  }
}

for (const group of groups) {
  const path = join(site, '.generated/docs/reference/commands', `${group.slug}.md`);
  const page = readFileSync(path, 'utf8');
  const title = page.match(/^title: (.+)$/m)?.[1];
  const description = page.match(/^description: (.+)$/m)?.[1];
  const audience = page.match(/^audience: (.+)$/m)?.[1];
  assert(title && JSON.parse(title).length < 60, `${group.slug} title must stay under 60 characters`);
  assert(description && JSON.parse(description).length >= 120 && JSON.parse(description).length <= 160, `${group.slug} description must be 120 to 160 characters`);
  assert.equal(audience, group.slug === 'immortal-commands' ? 'operators' : 'players', `${group.slug} audience must match its command scope`);
  for (const topic of group.topics) {
    const anchor = `<a id="${topic.anchor}"></a>`;
    assert.equal(page.split(anchor).length - 1, 1, `${topic.slug} must appear exactly once in generated pages`);
    seen.set(topic.slug, (seen.get(topic.slug) ?? 0) + 1);

    const heading = `## ${topic.title}\n\n**Verbatim in-game help.**\n\n`;
    const start = page.indexOf(heading, page.indexOf(anchor) + anchor.length);
    assert.notEqual(start, -1, `${topic.slug} needs its heading and verbatim label`);
    const contentStart = start + heading.length;
    const fence = fenceFor(topic.body);
    assert.equal(page.slice(contentStart, contentStart + fence.length), fence, `${topic.slug} needs a fenced source block`);
    const sourceStart = contentStart + fence.length + 1;
    const sourceEnd = page.indexOf(`\n${fence}`, sourceStart);
    assert.notEqual(sourceEnd, -1, `${topic.slug} source block must close`);
    const rawBody = rawByRecord.get(`${topic.category}:${topic.sourceKeyLine}`);
    assert.notEqual(rawBody, undefined, `${topic.slug} must map to an actual raw .hlp record`);
    assert.equal(topic.body, rawBody, `${topic.slug} taxonomy body must be raw .hlp content`);
    assert.equal(page.slice(sourceStart, sourceEnd + 1), rawBody.endsWith('\n') ? rawBody : `${rawBody}\n`, `${topic.slug} generated block changed raw .hlp bytes`);

    const htmlTarget = `/docs/reference/commands/${group.slug}/#${topic.anchor}`;
    const markdownTarget = `/docs/reference/commands/${group.slug}.md#${topic.anchor}`;
    const legacy = `/help/${topic.category}/${topic.slug}`;
    assert.equal(redirects[`${legacy}/`], htmlTarget, `${legacy}/ must redirect to its anchor`);
    assert.equal(redirects[`${legacy}.md`], markdownTarget, `${legacy}.md must redirect to its Markdown anchor`);
    const reservedRoots = new Set(['commands','info','socials','spells','wizhelp'].map(category => '/help/'+category));
    const shortPath = '/help/'+topic.slug;
    if (!reservedRoots.has(shortPath)) {
      assert.equal(redirects[shortPath], htmlTarget, `${shortPath} must redirect to its anchor`);
      assert.equal(redirects[shortPath+'.md'], markdownTarget, `${shortPath}.md must redirect to its Markdown anchor`);
    }
    for (const alias of topic.aliases) {
      const aliasPath = (alias.startsWith('/') ? alias : `/help/${alias}`).replace(/\/$/, '');
      if (reservedRoots.has(aliasPath)) continue;
      assert.equal(redirects[aliasPath], htmlTarget, `${aliasPath} alias must redirect to its anchor`);
      assert.equal(redirects[`${aliasPath}.md`], markdownTarget, `${aliasPath}.md alias must redirect to its Markdown anchor`);
    }
    if (topic.category === 'commands') {
      assert.equal(redirects[`/docs/reference/commands/${topic.slug}/`], htmlTarget, `${topic.slug} old docs route must redirect to its anchor`);
      assert.equal(redirects[`/docs/reference/commands/${topic.slug}.md`], markdownTarget, `${topic.slug} old docs Markdown route must redirect to its anchor`);
    }
  }
}
assert.equal(topics.length, sourceCount, 'every source help file must be included in the generated groups');
assert.equal(rawByRecord.size, topics.length, 'every raw game help record must have exactly one museum topic');
assert.equal(seen.size, topics.length, 'each canonical topic must appear once');
assert([...seen.values()].every((count) => count === 1), 'canonical topics must not be duplicated');
for (const category of ['commands', 'info', 'socials', 'spells', 'wizhelp']) {
  assert(redirects[`/help/${category}/`]?.startsWith('/docs/reference/commands/'), `${category} museum index needs a valid docs redirect`);
}
assert.equal(redirects['/help/'], '/docs/reference/commands/', 'the Help museum index needs a redirect');
assert.equal(redirects['/help/commands/'], '/docs/reference/commands/', 'the command museum index needs the command group index');
const assist = topics.find((topic) => topic.slug === 'assist');
const assistPage = readFileSync(join(site, '.generated/docs/reference/commands', `${assist.groupSlug}.md`), 'utf8');
assert(assist.body.startsWith('\nUsage: ASSIST <person in need>\n'), 'ASSIST raw-source proof must retain source blank line and literal angle brackets');
assert(!assist.body.includes('[ASSIST]') && !assist.body.includes('&lt;'), 'ASSIST must not inherit archive links or escaped entities');
assert(assistPage.includes('\nUsage: ASSIST <person in need>\n'), 'generated ASSIST block must retain raw source punctuation');
console.log(`command-docs: ${topics.length} topics appear once; all ${rawByRecord.size} bodies match lib/text/help/*.hlp byte-for-byte; ASSIST link/entity negative example passes; ${Object.keys(redirects).length} HTML/Markdown redirects cover every topic`);
