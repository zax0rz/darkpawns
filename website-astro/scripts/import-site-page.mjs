/**
 * Import a recovered dp-players.com / darkpawns.com page into the archive.
 *
 * These captures are not forum topics, so they need different handling from
 * import-phpbb-html.mjs. Both sites wrap their content in navigation chrome:
 * dp-players.com repeats its title, a "Latest Discussion" sidebar and the menu
 * before every page, ending with a line like "logs*" that names the page.
 * Everything before that marker is chrome. darkpawns.com is plainer and only
 * needs its title line removed.
 *
 * usage: node scripts/import-site-page.mjs <capture> <output.md> --record <id>
 */
import { readFileSync, writeFileSync } from 'node:fs';
import { basename, resolve } from 'node:path';

/* ------------------------------------------------------------------ metadata */

const D = 'dp-players.com';
const F = 'darkpawns.com';

const RECORDS = {
  'player-page-directions': {
    title: 'Directions',
    description: 'Speedwalks from Market Square to the newbie, midbie and high-level zones, plus the gates and the dark mage tower.',
    kind: 'guide', mode: 'dp-html', site: D, shape: 'plain',
    dateLabel: 'Published 2004', sortDate: '2004-08-03',
    sourceUrl: 'http://www.dp-players.com/go.php?dp=directions.mud',
    captureUrl: 'https://web.archive.org/web/20040803063652/http://www.dp-players.com/go.php?dp=directions.mud',
    recoveredAt: '2026-08-15', voiceLayer: 'frontline',
  },
  'darkpawns-faq': {
    title: 'Dark Pawns FAQ',
    description: 'The official answers on races, classes, character creation, remorts, policy and etiquette.',
    kind: 'site-page', mode: 'dpcom-html', site: F, startAt: 'zMUD mud client',
    dateLabel: 'Captured August 2004', sortDate: '2004-08-03',
    sourceUrl: 'http://darkpawns.com/faq.html',
    captureUrl: 'https://web.archive.org/web/20040803042109/http://darkpawns.com/faq.html',
    recoveredAt: '2026-08-15', voiceLayer: 'mythic-admin',
  },
  'player-page-news': {
    title: 'Site News',
    description: "Aidan's running log of what he changed on the player page, from the front page as it stood in July 2004.",
    kind: 'site-page', mode: 'dp-html', site: D, shape: 'player-news',
    dateLabel: 'March to May 2004', sortDate: '2004-05-14',
    sourceUrl: 'http://www.dp-players.com/',
    captureUrl: 'https://web.archive.org/web/20040728202301/http://www.dp-players.com/',
    recoveredAt: '2026-08-15', voiceLayer: 'frontline',
  },
  'player-page-archives': {
    title: 'News Archive',
    description: 'The older half of the news log, back to the launch of the third version of the site in April 2004.',
    kind: 'site-page', mode: 'dp-html', site: D, shape: 'player-news',
    dateLabel: 'April 2004', sortDate: '2004-04-21',
    sourceUrl: 'http://www.dp-players.com/archives.php',
    captureUrl: 'https://web.archive.org/web/20040610043934/http://www.dp-players.com/archives.php',
    recoveredAt: '2026-08-15', voiceLayer: 'frontline',
  },
  'board-index-general': {
    title: 'General Discussion, July 2004',
    description: 'The board index as it stood, with the reply count on every topic. It is the clearest measure of how much of the forum is gone.',
    kind: 'board-index', mode: 'raw', site: D, shape: 'board-index',
    dateLabel: 'July 23, 2004', sortDate: '2004-07-23',
    sourceUrl: 'http://www.dp-players.com/forum/viewforum.php?f=1',
    captureUrl: 'https://web.archive.org/web/20040723032431/http://www.dp-players.com:80/forum/viewforum.php?f=1&amp',
    recoveredAt: '2026-08-23', voiceLayer: 'frontline',
  },
  'map-kir-draxin': {
    title: "ASCII Map of Kir Drax'in",
    description: "Frontline's text map of the starting city, with the legend that names every building on it.",
    kind: 'map', mode: 'pre', site: F,
    dateLabel: 'Published by 2002', sortDate: '2002-04-20',
    sourceUrl: 'http://www.darkpawns.com/map_text.html',
    captureUrl: 'https://web.archive.org/web/20020420044503/http://www.darkpawns.com:80/map_text.html',
    recoveredAt: '2026-08-14', voiceLayer: 'mythic-admin',
  },
  'darkpawns-news-2002': {
    title: 'Frontline’s news posts, 2002',
    description: 'The news page from darkpawns.com as it stood in 2002, including the OUTLAW system revision.',
    kind: 'site-page', mode: 'dpcom-html', site: F,
    dateLabel: 'January to June 2002', sortDate: '2002-06-05',
    sourceUrl: 'http://www.darkpawns.com/main.html',
    captureUrl: 'https://web.archive.org/web/20020605035215/http://www.darkpawns.com:80/main.html',
    recoveredAt: '2026-08-14', voiceLayer: 'mythic-admin',
    shape: 'news',
  },
  'darkpawns-news-2005': {
    title: 'Dark Pawns 3.0 goes to beta',
    description: 'The 2005 front page, announcing the 3.0 beta the forum had spent the previous year waiting for.',
    kind: 'site-page', mode: 'dpcom-html', site: F,
    dateLabel: 'October 2005', sortDate: '2005-10-17',
    sourceUrl: 'http://www.darkpawns.com/',
    captureUrl: 'https://web.archive.org/web/20051017081230/http://www.darkpawns.com:80/',
    recoveredAt: '2026-08-14', voiceLayer: 'mythic-admin',
    shape: 'news',
    startAt: 'News',
    dropLines: ['Dark Pawns', '// home', '::home:: ::faq::', 'DP-specific Links', 'Forums', 'MUD Links', 'Mudconnector', 'zMUD mud client'],
  },
  'background': {
    title: 'Background Story',
    description: 'The founding myth of the world: gods, free will, and a board of bishops, kings and dark pawns.',
    kind: 'site-page', mode: 'dpcom-text', site: F,
    dateLabel: 'Date not recorded', sortDate: '2004-08-03',
    sourceUrl: 'http://darkpawns.com/background.html',
    captureUrl: 'https://web.archive.org/web/*/http://darkpawns.com/background.html',
    recoveredAt: '2026-08-14', voiceLayer: 'mythic-admin',
    usedIn: [{ label: 'World › World Creation', href: '/world/world-creation/' }],
  },
  'classes': {
    title: 'Class Listing',
    description: 'The six base classes and their remort counterparts, as darkpawns.com described them.',
    kind: 'site-page', mode: 'dpcom-text', site: F,
    dateLabel: 'Date not recorded', sortDate: '2004-08-03',
    sourceUrl: 'http://darkpawns.com/class.html',
    captureUrl: 'https://web.archive.org/web/*/http://darkpawns.com/class.html',
    recoveredAt: '2026-08-14', voiceLayer: 'mythic-admin',
    usedIn: [{ label: 'World › Classes', href: '/world/classes/' }],
  },
  'features': {
    title: 'Features',
    description: 'What the game advertised about itself: rent-free play, remorts, vampirism, and a world built on real terrain.',
    kind: 'site-page', mode: 'dpcom-text', site: F,
    dateLabel: 'Date not recorded', sortDate: '2004-08-03',
    sourceUrl: 'http://darkpawns.com/features.html',
    captureUrl: 'https://web.archive.org/web/*/http://darkpawns.com/features.html',
    recoveredAt: '2026-08-14', voiceLayer: 'mythic-admin',
  },
  'wizlist': {
    title: 'Staff List',
    description: 'The wizards credited with keeping the game playable, as listed on darkpawns.com.',
    kind: 'roster', mode: 'dpcom-text', site: F,
    dateLabel: 'Date not recorded', sortDate: '2004-08-03',
    sourceUrl: 'http://darkpawns.com/wizlist.html',
    captureUrl: 'https://web.archive.org/web/*/http://darkpawns.com/wizlist.html',
    recoveredAt: '2026-08-14', voiceLayer: 'mythic-admin',
    shape: 'wizlist',
  },
  'player-page-about': {
    title: 'About the Player Page',
    description: 'Aidan explains what the third version of dp-players.com was for, and how it was built.',
    kind: 'site-page', mode: 'dp-text', site: D,
    dateLabel: 'Date not recorded', sortDate: '2004-04-01',
    sourceUrl: 'http://www.dp-players.com/go.php?dp=about_site.mud',
    captureUrl: 'https://web.archive.org/web/*/http://www.dp-players.com/go.php?dp=about_site.mud',
    recoveredAt: '2026-08-14', voiceLayer: 'frontline',
    shape: 'player-page-about',
  },
  'player-page-articles': {
    title: 'Articles Index',
    description: 'The index of the strategy guides the player page hosted. The index survived; neither article did.',
    kind: 'site-page', mode: 'dp-text', site: D,
    dateLabel: 'Date not recorded', sortDate: '2004-08-03',
    sourceUrl: 'http://www.dp-players.com/go.php?dp=articles.mud',
    captureUrl: 'https://web.archive.org/web/*/http://www.dp-players.com/go.php?dp=articles.mud',
    recoveredAt: '2026-08-14', voiceLayer: 'frontline',
    shape: 'player-page-articles',
  },
  'player-page-logs': {
    title: 'Game Logs Index',
    description: 'The index of 201 battle logs players uploaded. Every category count survived; not one log did.',
    kind: 'site-page', mode: 'dp-text', site: D,
    dateLabel: 'Date not recorded', sortDate: '2004-08-03',
    sourceUrl: 'http://www.dp-players.com/go.php?dp=logs.mud',
    captureUrl: 'https://web.archive.org/web/*/http://www.dp-players.com/go.php?dp=logs.mud',
    recoveredAt: '2026-08-14', voiceLayer: 'frontline',
    shape: 'player-page-logs',
  },
  'player-page-equipment': {
    title: 'Equipment Guide',
    description: 'The equipment reference Aidan identified item by item, organised by where each piece is worn.',
    kind: 'guide', mode: 'dp-text', site: D,
    dateLabel: 'Date not recorded', sortDate: '2004-08-03',
    sourceUrl: 'http://www.dp-players.com/go.php?dp=equipment.mud',
    captureUrl: 'https://web.archive.org/web/*/http://www.dp-players.com/go.php?dp=equipment.mud',
    recoveredAt: '2026-08-14', voiceLayer: 'frontline',
  },
};

/* ------------------------------------------------------------------ stripping */

/**
 * Some 2004 captures are Windows-1252, not UTF-8: their curly quotes and dashes
 * are single high bytes. Reading those as UTF-8 turns an apostrophe into a
 * replacement character, so the file is decoded strictly first and only falls
 * back when that proves it is not UTF-8.
 */
function readCapture(path) {
  const bytes = readFileSync(path);
  try {
    return new TextDecoder('utf-8', { fatal: true }).decode(bytes);
  } catch {
    return new TextDecoder('windows-1252').decode(bytes);
  }
}

const ENTITIES = {
  '&amp;': '&', '&lt;': '<', '&gt;': '>', '&quot;': '"', '&#39;': "'",
  '&apos;': "'", '&nbsp;': ' ', '&hellip;': '…',
};

function htmlToText(html) {
  let text = html.replace(/<(script|style)[^>]*>[\s\S]*?<\/\1>/gi, ' ');
  text = text.replace(/<br\s*\/?>|<\/p>|<\/tr>|<\/div>|<\/li>|<\/h[1-6]>/gi, '\n');
  text = text.replace(/<[^>]+>/g, ' ');
  text = text.replace(/&#(\d+);/g, (_, code) => String.fromCharCode(Number(code)));
  for (const [entity, character] of Object.entries(ENTITIES)) text = text.split(entity).join(character);
  return text;
}

/** dp-players.com repeats its chrome above every page; the "name*" line ends it. */
function stripPlayerPageChrome(text) {
  const lines = text.split('\n').map((line) => line.replace(/[ \t]+/g, ' ').trimEnd());
  const marker = lines.findIndex((line) => /^\s*[a-z][a-z '?-]*\*\s*$/i.test(line));
  const body = marker === -1 ? lines : lines.slice(marker + 1);
  const tail = body.findIndex((line) => /^\s*::\s*$|^\s*<-\s*back|^\s*next\s*->|^\s*::\s*home\s*::/.test(line));
  return (tail === -1 ? body : body.slice(0, tail)).join('\n');
}

const clean = (text) =>
  text.split('\n').map((line) => line.replace(/[ \t]+/g, ' ').trim())
    .join('\n').replace(/\n{3,}/g, '\n\n').trim();

function escapeMarkdown(text) {
  return text
    .replace(/([\\`*_[\]<])/g, '\\$1')
    .replace(/^(\s*)([#>+-])/, '$1\\$2')
    .replace(/^(\s*\d+)\.(\s)/, '$1\\.$2');
}

/* ------------------------------------------------------------------ rendering */


/**
 * These captures are hard-wrapped plain text: the original paragraph breaks
 * were lost when the page was rendered to text. A wrapped line runs close to
 * the wrap width, so a line that stops noticeably short and ends a sentence is
 * where a paragraph ended. Reflowing on that rule restores the shape of the
 * page without inventing or moving a single word.
 */
function reflow(lines) {
  const widths = lines.filter((line) => line.length > 20).map((line) => line.length);
  if (widths.length < 3) return lines.map((line) => [line]);
  const wrap = Math.max(...widths);
  const paragraphs = [];
  let current = [];
  for (const line of lines) {
    if (!line) { if (current.length) paragraphs.push(current); current = []; continue; }
    current.push(line);
    const short = line.length < wrap * 0.8;
    if (short && /[.!?:"]$/.test(line)) { paragraphs.push(current); current = []; }
  }
  if (current.length) paragraphs.push(current);
  return paragraphs;
}

/** Per-page shaping. Reorders nothing; only marks up structure already there. */
const SHAPERS = {
  'player-news': (lines) => {
    const out = [];
    for (let i = 0; i < lines.length; i += 1) {
      if (lines[i] === 'Posted by' && lines[i + 2]?.startsWith('on ')) {
        const title = out.pop() ?? 'Untitled';
        out.push(`### ${title.replace(/^\\?/, '')}`);
        out.push(`*${escapeMarkdown(lines[i + 1])} — ${escapeMarkdown(lines[i + 2].replace(/^on /, ''))}*`);
        i += 2;
        continue;
      }
      if (/^\(\d+\)$/.test(lines[i]) && lines[i + 1] === 'Comments') { i += 1; continue; }
      out.push(escapeMarkdown(lines[i]));
    }
    return out.join('\n\n');
  },
  'board-index': (lines, raw) => {
    // Rendered by the entry template as a ledger, not as prose, so the shaper
    // emits nothing here and the rows travel in frontmatter instead.
    return '';
  },
  plain: (lines) => lines.map(escapeMarkdown).join('\n\n'),
  'player-page-articles': (lines) => {
    const rows = [];
    for (let i = 1; i < lines.length; i += 2) {
      const title = lines[i];
      const rest = (lines[i + 1] ?? '').match(/^(\S+)\s+(.*)$/);
      if (title && rest) rows.push(`| ${escapeMarkdown(title)} | ${rest[1]} | ${rest[2]} |`);
    }
    return ['| Article | Author | Date |', '| --- | --- | --- |', ...rows].join('\n');
  },
  'player-page-logs': (lines) => {
    const out = [];
    const joined = [];
    for (const line of lines) {
      if (/^\(\d+\)$/.test(line) && joined.length) joined[joined.length - 1] += ` ${line}`;
      else joined.push(line);
    }
    for (const line of joined) {
      const match = line.match(/^(.*?)\s*\((\d+)\)$/);
      out.push(match ? `- ${escapeMarkdown(match[1])} — ${match[2]} logs` : escapeMarkdown(line));
    }
    return out.join('\n');
  },
  wizlist: (lines) => {
    const split = lines.findIndex((line) => /^[A-Z][a-z]+$/.test(line));
    const prose = lines.slice(0, split).map(escapeMarkdown).join('\n');
    const names = lines.slice(split).map((line) => `- ${escapeMarkdown(line)}`).join('\n');
    return `${prose}\n\n${names}`;
  },
  'player-page-about': (lines) => {
    const out = [];
    for (const line of lines) {
      if (/^(the site|printer-friendly|in conclusion)$/i.test(line)) out.push(`\n### ${line}\n`);
      else out.push(escapeMarkdown(line));
    }
    return out.join('\n').replace(/\n{3,}/g, '\n\n').trim();
  },
  news: (lines) => {
    // Frontline's news pages repeat "Posted by / Date Posted / Subject" per item.
    const out = [];
    let index = 0;
    while (index < lines.length) {
      const line = lines[index];
      if (/^Posted by:/.test(line)) {
        const who = line.replace(/^Posted by:\s*/, '');
        let when = '';
        let subject = '';
        let cursor = index + 1;
        while (cursor < lines.length && !/^Subject:/.test(lines[cursor]) && !/^Posted by:/.test(lines[cursor])) {
          when += ` ${lines[cursor].replace(/^Date Posted:\s*/, '')}`;
          cursor += 1;
        }
        if (/^Subject:/.test(lines[cursor] ?? '')) { subject = lines[cursor].replace(/^Subject:\s*/, ''); cursor += 1; }
        out.push(`### ${escapeMarkdown(subject || 'Untitled')}`);
        out.push(`*${escapeMarkdown(who)} — ${escapeMarkdown(when.trim())}*`);
        index = cursor;
        continue;
      }
      out.push(escapeMarkdown(line));
      index += 1;
    }
    return out.join('\n\n');
  },
};

function bodyFor(record, raw) {
  if (record.mode === 'pre') {
    // The map is art made of characters. Column alignment is the content, so it
    // is preserved exactly inside a code block rather than reflowed as prose.
    const text = htmlToText(raw);
    const start = text.indexOf('Kir Drax');
    const legend = text.search(/^\s*1 = /m);
    const art = text.slice(start, legend === -1 ? undefined : legend);
    const rest = legend === -1 ? '' : text.slice(legend);
    return [
      '```',
      art.replace(/^\s*Kir Drax'?in\s*/, '').replace(/\n{3,}/g, '\n\n').replace(/\s+$/, ''),
      '```',
      '',
      ...clean(rest).split('\n').filter(Boolean).map((line) => `- ${escapeMarkdown(line)}`),
    ].join('\n');
  }
  let text;
  if (record.mode === 'dp-text') text = stripPlayerPageChrome(raw);
  else if (record.mode === 'dp-html') text = stripPlayerPageChrome(htmlToText(raw));
  else if (record.mode === 'dpcom-text') text = raw.split('\n').slice(1).join('\n');
  else if (record.mode === 'raw') text = raw;
  else text = htmlToText(raw);

  let lines = clean(text).split('\n');
  if (record.startAt) {
    const from = lines.findIndex((line) => line.trim() === record.startAt);
    if (from !== -1) lines = lines.slice(from + 1);
  }
  if (record.dropLines) lines = lines.filter((line) => !record.dropLines.includes(line.trim()));
  lines = lines.filter((line, index) => !(index === 0 && /^Dark Pawns$/.test(line)));

  const shaper = SHAPERS[record.shape ?? ''];
  if (shaper) return shaper(lines.filter(Boolean), raw);

  return reflow(lines)
    .map((paragraph) => paragraph.map(escapeMarkdown).join('\n'))
    .filter((paragraph) => paragraph.trim())
    .join('\n\n');
}

/**
 * The policy keeps contact details out of the archive. A capture can still
 * carry one in the middle of an otherwise publishable page, so addresses are
 * removed here and the entry is downgraded from verbatim to edited-excerpt.
 * Silently publishing the address would be worse; silently dropping the page
 * would lose the history. Marking the edit keeps both honest.
 */
const EMAIL = /[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}/g;

function redactContacts(body) {
  const found = body.match(EMAIL);
  if (!found) return { body, redacted: 0 };
  return { body: body.replace(EMAIL, '\\[address removed\\]'), redacted: found.length };
}

/**
 * Every topic row the captured board index listed, in its original order, with
 * the reply count phpBB printed beside it. Most of those topics were never
 * archived, so these rows are the measure of what is gone.
 */
function boardTopics(raw) {
  const flat = raw.replace(/\s+/g, ' ');
  const strip = (value) => value.replace(/<[^>]+>/g, '').replace(/&amp;/g, '&').trim();
  return [...flat.matchAll(/viewtopic\.php\?t=(\d+)[^"]*" class="topictitle">(.*?)<\/a>[\s\S]*?<td[^>]*class="row2"[^>]*>\s*<span class="postdetails">(\d+)<\/span>[\s\S]*?<span class="name"><a[^>]*>(.*?)<\/a>/g)]
    .map((row) => ({
      id: Number(row[1]),
      title: strip(row[2]),
      author: strip(row[4]),
      replies: Number(row[3]),
    }));
}

function main() {
  const args = process.argv.slice(2);
  const flag = args.indexOf('--record');
  const recordId = flag === -1 ? null : args[flag + 1];
  const [input, output] = args.filter((_, i) => i !== flag && i !== flag + 1);
  if (!input || !output || !recordId) {
    throw new Error('usage: node scripts/import-site-page.mjs <capture> <output.md> --record <id>');
  }
  const record = RECORDS[recordId];
  if (!record) throw new Error(`no reviewed metadata for record ${recordId}`);

  const capture = readCapture(resolve(input));
  const topics = record.shape === 'board-index' ? boardTopics(capture) : [];
  const raw = bodyFor(record, capture);
  // The board index carries no prose: its rows travel in frontmatter and the
  // entry template renders them as a ledger.
  if (!raw.trim() && !topics.length) {
    throw new Error('no content extracted; check the chrome-stripping mode');
  }
  const { body, redacted } = redactContacts(raw);

  const frontmatter = [
    '---',
    `title: ${JSON.stringify(record.title)}`,
    `description: ${JSON.stringify(record.description)}`,
    `kind: ${JSON.stringify(record.kind)}`,
    `sortDate: ${record.sortDate}`,
    `dateLabel: ${JSON.stringify(record.dateLabel)}`,
    `sourceSite: ${JSON.stringify(record.site)}`,
    `sourceUrl: ${JSON.stringify(record.sourceUrl)}`,
    `captureUrl: ${JSON.stringify(record.captureUrl)}`,
    `recoveredAt: ${record.recoveredAt}`,
    redacted ? 'textKind: "edited-excerpt"' : 'textKind: "verbatim"',
    'source: "Wayback capture identified by captureUrl"',
    `voiceLayer: ${JSON.stringify(record.voiceLayer)}`,
    'completeness: "complete"',
    record.usedIn
      ? ['usedIn:', ...record.usedIn.map((u) => `  - label: ${JSON.stringify(u.label)}\n    href: ${JSON.stringify(u.href)}`)].join('\n')
      : null,
    topics.length
      ? ['topics:', ...topics.map((t) =>
          `  - id: ${t.id}\n    title: ${JSON.stringify(t.title)}\n    author: ${JSON.stringify(t.author)}\n    replies: ${t.replies}`)].join('\n')
      : null,
    'draft: false',
    '---',
  ].filter((line) => line !== null).join('\n');

  const note = `*Transcript note: generated from the capture \`${basename(input)}\`. ` +
    'Site navigation, the recent-topics sidebar and advertising are omitted.' +
    (redacted
      ? ` ${redacted === 1 ? 'One e-mail address was' : `${redacted} e-mail addresses were`} removed from the text, ` +
        'as the archive does not publish contact details.'
      : '') +
    '*';

  writeFileSync(resolve(output), `${frontmatter}\n\n${note}\n\n${body}\n`, 'utf8');
  console.log(`${recordId}: ${body.split('\n').length} lines${redacted ? `, ${redacted} contact detail(s) removed` : ''}`);
}

main();
