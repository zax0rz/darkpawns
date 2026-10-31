import { readdirSync, readFileSync } from 'node:fs';
import { join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = fileURLToPath(new URL('../../', import.meta.url));
const helpRoot = join(root, 'website-astro/src/content/help');
const helpCategories = ['commands', 'info', 'socials', 'spells', 'wizhelp'];
const rawHelpRoot = join(root, 'lib/text/help');

// Ordered from specific command families to broad player tasks. Keep this as
// the single taxonomy source for generated pages, legacy redirects and audit.
const groups = [
  { slug: 'combat', title: 'Combat', pattern: /^(ambush|backstab|bash|bearhug|behead|berserk|bite|carve|charge|circle|cutthroat|diagnose|disarm|disembowel|dragon|evasion|escape|flee|groinrip|headbutt|kick|kill|murder|neckbreak|parry|retreat|serpent|shoot|slug|spike|strike|subdue|tiger|trip|yank|rescue|assist|consider|scout|aid|flesh|kuji|kuji-kiri|kabuki|palm|peek|sleeper|smackheads|steal|shadow|stealth|hide)$/ },
  { slug: 'movement', title: 'Movement', pattern: /^(dismount|enter|exits|follow|leave|north|ride|search|sneak|track|turn|visible|wake|sleep|sit|stand|rest)$/ },
  { slug: 'communication', title: 'Communication', pattern: /^(auction|channel|ctell|emote|gossip|gsay|gtell|holler|noshout|notell|qsay|reply|rsay|say|shout|tell|think|whisper|write|check|mail|bug|typo|idea|point|tag)$/ },
  { slug: 'objects', title: 'Objects', pattern: /^(appraise|bank|balance|buy|compare|donate|drop|eat|drink|sip|taste|equipment|examine|fill|get|take|give|gold|grab|hold|inventory|junk|list|offer|rent|open|close|lock|unlock|pick|pour|put|quaff|potion|read|recite|scroll|recharge|remove|retrieve|sell|scrounge|sharpen|split|use|wand|staff|value|wear|wield)$/ },
  { slug: 'information', title: 'Information', pattern: /^(abilities|commands|finger|whois|help|info|news|policy|version|future|levels|look|glance|motd|newbie|score|time|weather|who|whoami|wizlist|immlist|report|review|quest|house|age|clear|cls)$/ },
  { slug: 'groups', title: 'Groups', pattern: /^(clan|group|multi|multiplay|order|ungroup|follow|assist|rescue|gsay|gtell)$/ },
  { slug: 'character', title: 'Character', pattern: /^(practice|train|title|transform|reallyquit|quit|save|recall)$/ },
  { slug: 'settings', title: 'Settings', pattern: /^(brief|compact|display|prompt|infobar|lines|norepeat|toggle|wimpy|auto|loot|inactive|ansi|color|alias|aliases|afk|nosummon)$/ },
  { slug: 'system', title: 'System', pattern: /^(bang|caret|!|\^|qui|shutdow|quit|reallyquit|save|time|commands|help)$/ },
  { slug: 'spells', title: 'Spells', category: 'spells' },
  { slug: 'socials', title: 'Socials', category: 'socials' },
  { slug: 'immortal-commands', title: 'Immortal commands', category: 'wizhelp' },
  { slug: 'information', title: 'Information', category: 'info' },
];
const fallback = { slug: 'other', title: 'Other commands', pattern: /.*/ };

function frontmatterValue(raw, key) {
  const match = raw.match(new RegExp(`^${key}:\\s*(.*)$`, 'm'));
  if (!match) return '';
  let value = match[1].trim();
  if (value.startsWith('"') && value.endsWith('"')) value = value.slice(1, -1);
  return value;
}

function frontmatterAliases(raw) {
  const value = frontmatterValue(raw, 'aliases');
  return [...value.matchAll(/['"]([^'"]+)['"]/g)].map((match) => match[1]);
}

function firstWord(title, slug) {
  const cleaned = title.replaceAll("'", '').toLowerCase();
  return cleaned.split(/\s+/).find((word) => /^[a-z!^ -]+$/.test(word)) ?? slug.split('-')[0];
}

export function helpTopics() {
  const topics = [];
  for (const category of helpCategories) {
    const dir = join(helpRoot, category);
    for (const file of readdirSync(dir).filter((name) => name.endsWith('.md') && name !== '_index.md').sort()) {
      const raw = readFileSync(join(dir, file), 'utf8');
      const slug = file.slice(0, -3);
      const title = frontmatterValue(raw, 'title') || slug;
      const command = category === 'commands' ? firstWord(title, slug) : '';
      const group = category === 'commands'
        ? groups.find(({ category: groupCategory, pattern }) => !groupCategory && pattern.test(command)) ?? fallback
        : groups.find((candidate) => candidate.category === category);
      topics.push({ slug, anchor: `help-${category}-${slug}`, title, sourcePath: `website-astro/src/content/help/${category}/${file}`, category, aliases: frontmatterAliases(raw), command, groupSlug: group.slug, groupTitle: group.title });
    }
  }

  // The Markdown archive is useful for titles, aliases, and classification,
  // but its bodies contain links and HTML entities. Public reference blocks
  // must come from the original game help files byte-for-byte.
  const byIdentity = new Map(topics.map((topic) => [`${topic.category}:${normalizeHelpKey(topic.title)}`, topic]));
  const sourceTopics = new Set();
  const unmapped = [];
  for (const category of helpCategories) {
    for (const record of parseRawHelpFile(join(rawHelpRoot, `${category}.hlp`))) {
      const identity = `${category}:${normalizeHelpKey(record.keyLine)}`;
      let topic = byIdentity.get(identity);
      if (!topic) {
        const candidates = [...new Set(record.keywords
          .map((keyword) => byIdentity.get(`${category}:${normalizeHelpKey(keyword)}`))
          .filter(Boolean))];
        if (candidates.length === 1) [topic] = candidates;
      }
      if (!topic || sourceTopics.has(topic)) {
        unmapped.push(`${category}: ${JSON.stringify(record.keyLine)}${topic ? ` (duplicate mapping to ${topic.slug})` : ''}`);
        continue;
      }
      sourceTopics.add(topic);
      Object.assign(topic, { body: record.body, sourceKeyLine: record.keyLine, sourcePath: `lib/text/help/${category}.hlp` });
    }
  }
  const missing = topics.filter((topic) => !sourceTopics.has(topic)).map((topic) => `${topic.category}: ${topic.slug} (${topic.title})`);
  if (unmapped.length || missing.length) {
    throw new Error(`Could not map raw in-game help records. Unmapped records: ${unmapped.join('; ') || 'none'}. Missing museum topics: ${missing.join('; ') || 'none'}`);
  }
  return topics;
}

function normalizeHelpKey(value) {
  return value.toLowerCase().replace(/["']/g, '').replace(/[^a-z0-9!^]+/g, ' ').trim().replace(/\s+/g, ' ');
}

function helpKeywords(keyLine) {
  const keywords = [];
  for (let i = 0; i < keyLine.length;) {
    while (i < keyLine.length && /\s/.test(keyLine[i])) i++;
    if (i >= keyLine.length) break;
    let word = '';
    if (keyLine[i] === '"') {
      i++;
      while (i < keyLine.length && keyLine[i] !== '"') word += keyLine[i++];
      if (i < keyLine.length) i++;
    } else {
      while (i < keyLine.length && !/\s/.test(keyLine[i])) word += keyLine[i++];
    }
    if (word) keywords.push(word);
  }
  return keywords;
}

// Match C/Go load_help: each keyword line starts a record, and a line whose
// first byte is # ends it. The body slice retains every original byte,
// including CRLFs, blank lines, indentation, and the newline before #.
export function parseRawHelpFile(path) {
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
      const line = text.slice(pos, end).replace(/\r$/, '');
      if (line.startsWith('#')) {
        records.push({ keyLine, keywords: helpKeywords(keyLine), body: text.slice(bodyStart, pos) });
        pos = end < text.length ? end + 1 : end;
        terminated = true;
        break;
      }
      pos = end < text.length ? end + 1 : end;
    }
    if (!terminated) throw new Error(`Unterminated raw help record in ${path}: ${keyLine}`);
  }
  return records;
}

export function commandGroups() {
  const grouped = new Map();
  for (const topic of helpTopics()) {
    if (!grouped.has(topic.groupSlug)) grouped.set(topic.groupSlug, { slug: topic.groupSlug, title: topic.groupTitle, topics: [] });
    grouped.get(topic.groupSlug).topics.push(topic);
  }
  return [...grouped.values()].sort((a, b) => groups.findIndex((candidate) => candidate.slug === a.slug) - groups.findIndex((candidate) => candidate.slug === b.slug));
}

export function docsHelpRedirects() {
  const redirects = {};
  const groupByCategory = { commands: '', info: 'information', socials: 'socials', spells: 'spells', wizhelp: 'immortal-commands' };
  for (const topic of helpTopics()) {
    const htmlTarget = `/docs/reference/commands/${topic.groupSlug}/#${topic.anchor}`;
    const markdownTarget = `/docs/reference/commands/${topic.groupSlug}.md#${topic.anchor}`;
    const oldTopic = `/help/${topic.category}/${topic.slug}`;
    redirects[`${oldTopic}/`] = htmlTarget;
    redirects[oldTopic] = htmlTarget;
    redirects[`${oldTopic}.md`] = markdownTarget;
    redirects[`/help/${topic.slug}`] = htmlTarget;
    redirects[`/help/${topic.slug}.md`] = markdownTarget;
    for (const alias of topic.aliases) {
      const aliasPath = alias.startsWith('/') ? alias : `/help/${alias}`;
      const normalizedAlias = aliasPath.replace(/\/$/, '');
      redirects[normalizedAlias] ??= htmlTarget;
      redirects[`${normalizedAlias}/`] ??= htmlTarget;
      redirects[`${normalizedAlias}.md`] ??= markdownTarget;
    }
    if (topic.category === 'commands') {
      const oldDocsTopic = `/docs/reference/commands/${topic.slug}`;
      redirects[`${oldDocsTopic}/`] = htmlTarget;
      redirects[oldDocsTopic] = htmlTarget;
      redirects[`${oldDocsTopic}.md`] = markdownTarget;
    }
  }
  for (const [category, group] of Object.entries(groupByCategory)) {
    const target = group ? `/docs/reference/commands/${group}/` : '/docs/reference/commands/';
    redirects[`/help/${category}/`] = target;
    redirects[`/help/${category}`] = target;
    redirects[`/help/${category}.md`] = target.replace(/\/$/, '') + '.md';
  }
  redirects['/help/'] = '/docs/reference/commands/';
  return redirects;
}

// content_inventory.py uses the same source of truth, avoiding a second
// taxonomy or redirect list in Python.
if (process.argv[1] && fileURLToPath(import.meta.url) === resolve(process.argv[1])) {
  process.stdout.write(JSON.stringify({ groups: commandGroups(), redirects: docsHelpRedirects() }));
}
