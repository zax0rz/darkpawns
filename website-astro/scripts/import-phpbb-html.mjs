/**
 * Import a raw phpBB (subSilver) topic capture into an archive collection entry.
 *
 * The text-only importer (import-phpbb-text.mjs) works from Wayback text
 * renderings, which lose quote structure: a quoted passage and the reply that
 * follows it become two separate blocks with no author. This importer reads the
 * original HTML, so quotes stay attached to the post that made them and keep
 * their attribution.
 *
 * Deliberately dropped: the author sidebar (rank image, join date, post count,
 * location, avatar), profile links, private-message links, e-mail and messenger
 * links. That sidebar is where personal information lives. See ARCHIVE-POLICY.md.
 *
 * usage: node scripts/import-phpbb-html.mjs <capture.html> <output.md> --record <id>
 */
import { readFileSync, writeFileSync } from 'node:fs';
import { basename, resolve } from 'node:path';

/* ------------------------------------------------------------------ metadata */

// Reviewed, human-checked metadata per source record. Nothing is published
// without an entry here: the title, description and date label are editorial
// copy, and every capture needs a person to have looked at it.
const RECORDS = {
  p731: {
    slug: 'topic-731',
    title: 'random observation',
    description: 'Aidan wonders aloud whether Dark Pawns should be saved or put down, and the regulars argue about whose job it is.',
    dateLabel: 'July 1-17, 2004',
    sourceUrl: 'http://www.dp-players.com/forum/viewtopic.php?p=731#731',
    captureTimestamp: '20040724090156',
    captureOriginal: 'http://www.dp-players.com:80/forum/viewtopic.php?p=731&amp',
    recoveredAt: '2026-09-04',
    voiceLayer: 'frontline',
  },
  p737: {
    slug: 'topic-737',
    title: 'The Unforeseen Occultesque Following',
    description: 'A jargon-file definition of "mudhead" sets off a thread about how much of their lives players had given to the game.',
    dateLabel: 'July 14-18, 2004',
    sourceUrl: 'http://www.dp-players.com/forum/viewtopic.php?p=737#737',
    captureTimestamp: '20040724090847',
    captureOriginal: 'http://www.dp-players.com:80/forum/viewtopic.php?p=737&amp',
    recoveredAt: '2026-09-04',
    voiceLayer: 'frontline',
  },
  t10: {
    slug: "topic-t10",
    title: "those were the days...",
    description: "Aidan posts an old log of a crowded game and the regulars count how many of those names are still around.",
    dateLabel: "April 12-18, 2004",
    sourceUrl: "http://www.dp-players.com/forum/viewtopic.php?t=10",
    captureTimestamp: "20040724091202",
    captureOriginal: "http://www.dp-players.com:80/forum/viewtopic.php?t=10&amp",
    recoveredAt: "2026-09-02",
    voiceLayer: "frontline",
  },
  t20: {
    slug: "topic-t20",
    title: "The BG",
    description: "Morpheus posts the log of a group death, and the thread turns into gallows humour about being flattened.",
    dateLabel: "April 19-22, 2004",
    sourceUrl: "http://www.dp-players.com/forum/viewtopic.php?t=20",
    captureTimestamp: "20040724091848",
    captureOriginal: "http://www.dp-players.com:80/forum/viewtopic.php?t=20&amp",
    recoveredAt: "2026-09-04",
    voiceLayer: "frontline",
    contentWarning: "Contains crude locker-room joking of the kind common on 2004 game forums.",
  },
  t38: {
    slug: "topic-t38",
    title: "little known DP facts",
    description: "Players trade the secrets, dead portals and cut features they know about, and Orodreth explains where some of the world came from.",
    dateLabel: "April 25-May 30, 2004",
    sourceUrl: "http://www.dp-players.com/forum/viewtopic.php?t=38",
    captureTimestamp: "20040724092630",
    captureOriginal: "http://www.dp-players.com:80/forum/viewtopic.php?t=38&amp",
    recoveredAt: "2026-09-02",
    voiceLayer: "frontline",
  },
  t39: {
    slug: "topic-t39",
    title: "forum updates",
    description: "Aidan announces new smilies, rank images and a quick-reply hack, and Vargus finds the statistics module broken.",
    dateLabel: "April 26-29, 2004",
    sourceUrl: "http://www.dp-players.com/forum/viewtopic.php?t=39",
    captureTimestamp: "20040509170227",
    captureOriginal: "http://www.dp-players.com:80/forum/viewtopic.php?t=39&amp",
    recoveredAt: "2026-09-04",
    voiceLayer: "frontline",
  },
  t51: {
    slug: "topic-t51",
    title: "more forum updates",
    description: "A new statistics module, more emoticons, and a broken link caught within minutes of the announcement.",
    dateLabel: "April 30, 2004",
    sourceUrl: "http://www.dp-players.com/forum/viewtopic.php?t=51",
    captureTimestamp: "20040803054421",
    captureOriginal: "http://www.dp-players.com:80/forum/viewtopic.php?t=51&amp",
    recoveredAt: "2026-09-02",
    voiceLayer: "frontline",
  },
  t60: {
    slug: "topic-t60",
    title: "Frontline and Paying for the Mud Humor(?)",
    description: "A log of an exasperated administrator, posted at three in the morning, about paying fifteen dollars a month for the game he keeps getting killed in.",
    dateLabel: "May 4-10, 2004",
    sourceUrl: "http://www.dp-players.com/forum/viewtopic.php?t=60",
    captureTimestamp: "20040724093747",
    captureOriginal: "http://www.dp-players.com:80/forum/viewtopic.php?t=60&amp",
    recoveredAt: "2026-09-02",
    voiceLayer: "frontline",
  },
  t67: {
    slug: "topic-t67",
    title: "more logs",
    description: "Two new game logs go up on the site and the regulars argue about which section they belong in.",
    dateLabel: "May 9-13, 2004",
    sourceUrl: "http://www.dp-players.com/forum/viewtopic.php?t=67",
    captureTimestamp: "20040803060042",
    captureOriginal: "http://www.dp-players.com:80/forum/viewtopic.php?t=67&amp",
    recoveredAt: "2026-08-24",
    voiceLayer: "frontline",
  },
  t79: {
    slug: "topic-t79",
    title: "how old are your charcters?",
    description: "A thread about character age that turns into a roll call of the oldest players in the game, and how the ageing penalties drove people to reroll.",
    dateLabel: "May 21-24, 2004",
    sourceUrl: "http://www.dp-players.com/forum/viewtopic.php?p=625",
    captureTimestamp: "20040724082830",
    captureOriginal: "http://www.dp-players.com:80/forum/viewtopic.php?p=625&amp",
    recoveredAt: "2026-08-25",
    voiceLayer: "frontline",
  },
  t82: {
    slug: "topic-t82",
    title: "Fun with funeraries....",
    description: "A failed steal, an accidental kill, and four posts of unsympathetic commentary.",
    dateLabel: "May 22-23, 2004",
    sourceUrl: "http://www.dp-players.com/forum/viewtopic.php?p=612",
    captureTimestamp: "20040724082150",
    captureOriginal: "http://www.dp-players.com:80/forum/viewtopic.php?p=612&amp",
    recoveredAt: "2026-08-25",
    voiceLayer: "frontline",
  },
  t83: {
    slug: "topic-t83",
    title: "To pk or not topk!?",
    description: "The long-running argument about player killing: whether the restrictions ruined it, and why the staff say they cannot police it.",
    dateLabel: "May 24-25, 2004",
    sourceUrl: "http://www.dp-players.com/forum/viewtopic.php?p=627",
    captureTimestamp: "20040724083546",
    captureOriginal: "http://www.dp-players.com:80/forum/viewtopic.php?p=627&amp",
    recoveredAt: "2026-09-01",
    voiceLayer: "frontline",
  },
  t88: {
    slug: "topic-t88",
    title: "Infobar, ZMud, GMud and others...",
    description: "Doragar asks whether anyone uses the in-game infobar, and the answers turn into a survey of which clients people played with.",
    dateLabel: "May 30-31, 2004",
    sourceUrl: "http://www.dp-players.com/forum/viewtopic.php?p=654",
    captureTimestamp: "20040724084550",
    captureOriginal: "http://www.dp-players.com:80/forum/viewtopic.php?p=654&amp",
    recoveredAt: "2026-09-02",
    voiceLayer: "frontline",
  },
  t32: {
    slug: "topic-t32",
    title: "Greatest players",
    description: "The final page of a six-page argument about who the best players in the game ever were.",
    dateLabel: "May 6-8, 2004",
    sourceUrl: "http://www.dp-players.com/forum/viewtopic.php?t=32",
    captureTimestamp: "20040803020622",
    captureOriginal: "http://www.dp-players.com/forum/viewtopic.php?p=514",
    recoveredAt: "2026-08-14",
    voiceLayer: "frontline",
  },
};

/* -------------------------------------------------------------------- parsing */

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

const collapse = (html) => html.replace(/\s+/g, ' ');

/** Find the matching close for a tag that can nest, starting after `from`. */
function matchDepth(html, from, openRe, closeRe) {
  let depth = 1;
  let i = from;
  while (i < html.length && depth > 0) {
    openRe.lastIndex = i;
    closeRe.lastIndex = i;
    const open = openRe.exec(html);
    const close = closeRe.exec(html);
    if (!close) return html.length;
    if (open && open.index < close.index) {
      depth += 1;
      i = open.index + open[0].length;
    } else {
      depth -= 1;
      i = close.index + close[0].length;
      if (depth === 0) return close.index;
    }
  }
  return i;
}

const QUOTE_HEAD =
  /<table[^>]*>\s*<tr>\s*<td><span class="genmed"><b>([^<]*)<\/b><\/span><\/td>\s*<\/tr>\s*<tr>\s*<td class="quote">/i;
const POSTBODY_OPEN = /<span class="postbody">/i;

/**
 * Walk a post's message cell in document order, returning body text and quote
 * blocks in the sequence the author wrote them. phpBB closes and reopens the
 * postbody span around every quote table, which is why a naive parser splits
 * one post into several.
 */
function parseSegments(region) {
  const segments = [];
  let i = 0;
  while (i < region.length) {
    const rest = region.slice(i);
    const quote = rest.match(QUOTE_HEAD);
    const body = rest.match(POSTBODY_OPEN);
    if (!quote && !body) break;
    const quoteAt = quote ? quote.index : Infinity;
    const bodyAt = body ? body.index : Infinity;

    if (quoteAt < bodyAt) {
      const start = i + quoteAt;
      const innerStart = start + quote[0].length;
      const innerEnd = matchDepth(region, innerStart, /<td\b/gi, /<\/td>/gi);
      segments.push({
        type: 'quote',
        label: quote[1].trim(),
        inner: region.slice(innerStart, innerEnd),
      });
      const tableEnd = region.indexOf('</table>', innerEnd);
      i = tableEnd === -1 ? innerEnd : tableEnd + '</table>'.length;
    } else {
      const start = i + bodyAt;
      const innerStart = start + body[0].length;
      const innerEnd = matchDepth(region, innerStart, /<span\b/gi, /<\/span>/gi);
      segments.push({ type: 'text', html: region.slice(innerStart, innerEnd) });
      i = innerEnd;
    }
  }
  return segments;
}

/* ------------------------------------------------------------------- text out */

const ENTITIES = {
  '&amp;': '&', '&lt;': '<', '&gt;': '>', '&quot;': '"', '&#39;': "'",
  '&apos;': "'", '&nbsp;': ' ', '&hellip;': '…', '&mdash;': '—', '&ndash;': '–',
};

// Links that exist only to reach a person are dropped; their text is kept.
const PRIVATE_LINK = /profile\.php|privmsg\.php|mailto:|aim:|ymsgr|msnim|posting\.php|login\.php|search\.php|groupcp\.php|memberlist\.php/i;

function toText(html) {
  let text = html;
  text = text.replace(/<br\s*\/?>/gi, '\n');
  // Smilies carry meaning; keep the alt text the way the earlier transcripts did.
  text = text.replace(/<img[^>]*src="[^"]*smiles\/[^"]*"[^>]*alt="([^"]*)"[^>]*>/gi,
    (_, alt) => (alt ? `[${alt}]` : ''));
  text = text.replace(/<img[^>]*>/gi, '');
  text = text.replace(/<a[^>]*href="([^"]*)"[^>]*>(.*?)<\/a>/gi, (_, href, label) => {
    const inner = label.replace(/<[^>]+>/g, '').trim();
    if (!href || PRIVATE_LINK.test(href) || !/^https?:/i.test(href)) return inner;
    return inner && inner !== href ? `[${inner}](${href})` : href;
  });
  text = text.replace(/<[^>]+>/g, '');
  text = text.replace(/&#(\d+);/g, (_, code) => String.fromCharCode(Number(code)));
  for (const [entity, character] of Object.entries(ENTITIES)) {
    text = text.split(entity).join(character);
  }
  return text.replace(/[ \t]+\n/g, '\n').replace(/\n{3,}/g, '\n\n').trim();
}

/**
 * Escape the characters that would otherwise turn an author's own punctuation
 * into Markdown formatting. Policy is to preserve the words as typed, so an
 * asterisk the author typed has to survive as an asterisk.
 */
function escapeMarkdown(text) {
  return text
    .split('\n')
    .map((line) =>
      line
        .replace(/([\\`*_[\]<])/g, '\\$1')
        .replace(/^(\s*)([#>+-])/, '$1\\$2')
        .replace(/^(\s*\d+)\.(\s)/, '$1\\.$2'),
    )
    .join('\n');
}

/** Markdown paragraphs, with hard breaks for the single newlines inside one. */
function toMarkdown(text) {
  return text
    .split(/\n{2,}/)
    .map((block) => escapeMarkdown(block).split('\n').join('\\\n'))
    .filter((block) => block.trim())
    .join('\n\n');
}

/**
 * A signature is the author's own sign-off, kept but set apart. Each line stays
 * a line: these are usually short quotes or ASCII flourishes where the breaks
 * are the point.
 */
function renderSignature(text) {
  return text
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => `*${escapeMarkdown(line)}*`)
    .join('\\\n');
}

const blockquote = (markdown) =>
  markdown.split('\n').map((line) => (line ? `> ${line}` : '>')).join('\n');

/* ------------------------------------------------------------------ the topic */

function parseTopic(rawHtml) {
  const html = collapse(rawHtml);

  const title = (html.match(/<title>(.*?)<\/title>/i)?.[1] ?? '')
    .replace(/^DP-Players\.com :: View topic - /, '').trim();
  const topicId = html.match(/viewtopic\.php\?t=(\d+)/i)?.[1] ?? null;
  const breadcrumb = html.match(
    /<a href="viewforum\.php\?f=(\d+)[^"]*" class="nav">(.*?)<\/a>/i);
  const board = breadcrumb ? toText(breadcrumb[2]) : null;
  const boardId = breadcrumb ? breadcrumb[1] : null;
  const pageMatch = html.match(/Page <b>(\d+)<\/b> of <b>(\d+)<\/b>/i);
  const page = pageMatch ? Number(pageMatch[1]) : 1;
  const pages = pageMatch ? Number(pageMatch[2]) : 1;

  const postRe = /<span class="name"><a name="(\d+)"><\/a><b>(.*?)<\/b><\/span>/gi;
  const heads = [...html.matchAll(postRe)];
  const posts = heads.map((head, index) => {
    const start = head.index;
    const end = index + 1 < heads.length ? heads[index + 1].index : html.length;
    const chunk = html.slice(start, end);
    // Stop before the row of profile / private-message / e-mail icons.
    const region = chunk.slice(0, chunk.indexOf('Back to top') + 1 || chunk.length);

    const role = toText(region.match(/<span class="postdetails">([^<]+)<br/i)?.[1] ?? '') || null;
    const posted = toText(region.match(/Posted: (.*?)<span class="gen">/i)?.[1] ?? '') || 'date unknown';
    const subject = toText(region.match(/Post subject: ([^<]*)<\/span>/i)?.[1] ?? '') || null;

    const segments = parseSegments(region.slice(region.indexOf('Posted:')));
    return {
      id: head[1],
      author: toText(head[2]),
      role,
      posted,
      subject,
      segments,
    };
  });

  return { title, topicId, board, boardId, page, pages, posts };
}

/* ------------------------------------------- quote attribution (deterministic) */

const normalize = (text) => text.toLowerCase().replace(/\s+/g, ' ').trim();

/** Plain body text of a post, quotes excluded — what an later post might quote. */
function ownWords(post) {
  return post.segments
    .filter((segment) => segment.type === 'text')
    .map((segment) => toText(segment.html).split('_________________')[0])
    .join('\n');
}

/**
 * phpBB records "Name wrote:" only when the quoting author used the reply-with-
 * quote button. A bare "Quote:" keeps no attribution at all, so the only honest
 * way to name the speaker is to find the words verbatim in an earlier post on
 * this page. A unique match names them; anything else stays unattributed. Never
 * guess: a wrong name in an archive is worse than no name.
 */
function attribute(quotedText, priorPosts) {
  const needle = normalize(quotedText).slice(0, 80);
  if (needle.length < 25) return null;
  const hits = priorPosts.filter((post) => normalize(ownWords(post)).includes(needle));
  const names = [...new Set(hits.map((post) => post.author))];
  return names.length === 1 ? names[0] : null;
}

/* ------------------------------------------------------------------ rendering */

function renderSegments(post, priorPosts) {
  const parts = [];
  let signature = null;

  for (const segment of post.segments) {
    if (segment.type === 'quote') {
      const nested = parseSegments(segment.inner);
      const innerText = nested.length
        ? renderSegments({ segments: nested }, priorPosts).body
        : toMarkdown(toText(segment.inner));
      const explicit = segment.label.match(/^(.*?) wrote:$/i)?.[1]?.trim();
      let attribution;
      if (explicit) {
        attribution = `**${explicit} wrote:**`;
      } else {
        const guess = attribute(toText(segment.inner), priorPosts);
        attribution = guess
          ? `**${guess}, earlier in this thread:**`
          : '**Quoted:**';
      }
      parts.push(blockquote(`${attribution}\n\n${innerText}`.trim()));
      continue;
    }
    const raw = toText(segment.html);
    if (!raw) continue;
    const [body, ...rest] = raw.split('_________________');
    if (body.trim()) parts.push(toMarkdown(body.trim()));
    if (rest.length) {
      const tail = rest.join('_________________').trim();
      if (tail) signature = tail;
    }
  }
  return { body: parts.filter(Boolean).join('\n\n'), signature };
}

/** Calendar date as the poster saw it, with no timezone re-interpretation. */
const isoDay = (date) =>
  `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`;

/** "1", "1 and 2", "1 to 5", "1, 2 and 4" - readable, and honest about gaps. */
function listNumbers(numbers) {
  if (numbers.length === 1) return `${numbers[0]}`;
  const contiguous = numbers.every((value, index) => index === 0 || value === numbers[index - 1] + 1);
  if (contiguous && numbers.length > 2) return `${numbers[0]} to ${numbers.at(-1)}`;
  return `${numbers.slice(0, -1).join(', ')} and ${numbers.at(-1)}`;
}

function frontmatter(record, topic, participants, completeness, note) {
  const captureUrl =
    `https://web.archive.org/web/${record.captureTimestamp}/${record.captureOriginal}`;
  // phpBB prints a wall-clock date with no zone. Reading it back through
  // toISOString() would re-interpret it as UTC and push any evening post to the
  // following day, so the date is taken from the parsed local parts instead.
  const parsed = new Date(topic.posts[0]?.posted ?? '');
  const sortDate = Number.isNaN(parsed.getTime()) ? null : parsed;
  const lines = [
    '---',
    `title: ${JSON.stringify(topic.title || record.title)}`,
    `description: ${JSON.stringify(record.description)}`,
    'kind: "forum-thread"',
    `sortDate: ${isoDay(sortDate ?? new Date())}`,
    `dateLabel: ${JSON.stringify(record.dateLabel)}`,
    `publishedAt: ${isoDay(sortDate ?? new Date())}`,
    'sourceSite: "dp-players.com"',
    `sourceUrl: ${JSON.stringify(record.sourceUrl)}`,
    `captureUrl: ${JSON.stringify(captureUrl)}`,
    `recoveredAt: ${record.recoveredAt}`,
    'textKind: "verbatim"',
    'source: "Wayback capture identified by captureUrl"',
    `voiceLayer: ${JSON.stringify(record.voiceLayer)}`,
    topic.board ? `board: ${JSON.stringify(topic.board)}` : null,
    `postCount: ${topic.posts.length}`,
    `completeness: ${JSON.stringify(completeness)}`,
    note ? `completenessNote: ${JSON.stringify(note)}` : null,
    record.contentWarning ? `contentWarning: ${JSON.stringify(record.contentWarning)}` : null,
    'participants:',
    ...participants.map((person) =>
      `  - name: ${JSON.stringify(person.name)}\n    role: ${JSON.stringify(person.role ?? 'unknown')}\n    posts: ${person.posts}`),
    'draft: false',
    '---',
  ];
  return lines.filter((line) => line !== null).join('\n');
}

/* ----------------------------------------------------------------------- main */

function main() {
  const args = process.argv.slice(2);
  const recordFlag = args.indexOf('--record');
  const recordId = recordFlag === -1 ? null : args[recordFlag + 1];
  const positional = args.filter((_, index) => index !== recordFlag && index !== recordFlag + 1);
  const outputArg = positional.pop();
  const inputArgs = positional;
  if (!inputArgs.length || !outputArg || !recordId) {
    throw new Error('usage: node scripts/import-phpbb-html.mjs <capture.html>... <output.md> --record <id>');
  }
  const record = RECORDS[recordId];
  if (!record) throw new Error(`no reviewed metadata for record ${recordId}`);

  // A long thread can survive as separate captured pages with gaps between
  // them. Each page is parsed on its own and then read in page order, so the
  // gaps stay visible instead of being silently closed up.
  const pages = inputArgs
    .map((input) => ({ file: basename(input), ...parseTopic(readCapture(resolve(input))) }))
    .sort((a, b) => a.page - b.page);
  if (!pages.length || !pages[0].posts.length) {
    throw new Error('no posts found; is this a phpBB topic capture?');
  }

  const topic = pages[0];
  const posts = pages.flatMap((entry) => entry.posts);
  const havePages = pages.map((entry) => entry.page);
  const totalPages = Math.max(...pages.map((entry) => entry.pages));
  const complete = totalPages === 1 && havePages.length === 1;
  const missingPages = Array.from({ length: totalPages }, (_, index) => index + 1)
    .filter((number) => !havePages.includes(number));
  const note = complete
    ? null
    : missingPages.length === 0
      ? null
      : `${havePages.length === 1 ? 'Page' : 'Pages'} ${listNumbers(havePages)} of ${totalPages}. ` +
        `${missingPages.length === 1 ? 'Page' : 'Pages'} ${listNumbers(missingPages)} ` +
        `${missingPages.length === 1 ? 'was' : 'were'} never captured.`;

  const participants = [];
  for (const post of posts) {
    const existing = participants.find((person) => person.name === post.author);
    if (existing) existing.posts += 1;
    else participants.push({ name: post.author, role: post.role, posts: 1 });
  }

  const blocks = [];
  let cursor = 0;
  pages.forEach((entry, pageIndex) => {
    if (pageIndex > 0) {
      const previous = pages[pageIndex - 1].page;
      const gap = entry.page - previous - 1;
      blocks.push(
        gap > 0
          ? `*${gap === 1 ? 'Page' : 'Pages'} ${listNumbers(Array.from({ length: gap }, (_, i) => previous + 1 + i))} ` +
            `of this thread ${gap === 1 ? 'was' : 'were'} never captured. The thread continues on page ${entry.page}.*`
          : `*Page ${entry.page}.*`,
      );
    }
    entry.posts.forEach((post) => {
      const { body, signature } = renderSegments(post, posts.slice(0, cursor));
      cursor += 1;
      blocks.push(`### ${post.author} — ${post.posted}`);
      if (post.subject && post.subject !== topic.title) {
        blocks.push(`*Post subject: ${escapeMarkdown(post.subject)}*`);
      }
      if (body) blocks.push(body);
      if (signature) blocks.push(renderSignature(signature));
    });
  });

  const sources = pages.map((entry) => `\`${entry.file}\``).join(', ');
  const transcriptNote =
    `*Transcript note: generated from the raw phpBB capture${pages.length > 1 ? 's' : ''} ${sources}` +
    `${complete ? '' : ` (page${havePages.length > 1 ? 's' : ''} ${listNumbers(havePages)} of ${totalPages})`}. ` +
    'Author sidebars, profile links and private-message links are omitted.*';

  const output = [
    frontmatter(record, { ...topic, posts, board: topic.board }, participants,
      complete ? 'complete' : 'partial', note),
    '',
    transcriptNote,
    '',
    blocks.join('\n\n'),
    '',
  ].join('\n');

  writeFileSync(resolve(outputArg), output, 'utf8');
  console.log(
    `${record.slug}: ${posts.length} posts, ${participants.length} participants, ` +
    `board ${topic.board ?? 'unknown'}, page${havePages.length > 1 ? 's' : ''} ${havePages.join('+')}/${totalPages}`);
}

main();
