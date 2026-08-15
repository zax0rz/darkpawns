import { readFileSync, writeFileSync } from 'node:fs';
import { basename, resolve } from 'node:path';

const records = {
  p492: {
    title: 'Object Descriptions Needed',
    description: 'Players collaborate on descriptions for unfinished Dark Pawns equipment and potions.',
    date: '2004-04-28',
    dateLabel: 'April 28–May 6, 2004',
  },
  p501: {
    title: 'advertising dark pawns',
    description: 'Players trade stories about how friends drew them into Dark Pawns—and how to bring in somebody new.',
    date: '2004-05-03',
    dateLabel: 'May 3–8, 2004',
  },
  p513: {
    title: 'Ninjas',
    description: 'Players debate ninja combat mechanics while an immortal declines to reveal the code behind them.',
    date: '2004-05-05',
    dateLabel: 'May 5–9, 2004',
  },
};

const [inputArg, outputArg] = process.argv.slice(2);
if (!inputArg || !outputArg) {
  throw new Error('usage: node scripts/import-phpbb-text.mjs <pNNN.txt> <output.md>');
}

const input = resolve(inputArg);
const output = resolve(outputArg);
const id = basename(input, '.txt');
const record = records[id];
if (!record) throw new Error(`no reviewed metadata for ${id}`);

const lines = readFileSync(input, 'utf8').replaceAll('\u00a0', ' ').split(/\r?\n/);
const authorLine = /^(.+?) (site administrator|immortal|moderator|dark pawn|pawn|newbie) Joined:/i;
const postedLine = /^Posted: (.+?)(?:\s+Post subject:.*)?$/;
const posts = [];

for (let i = 0; i < lines.length; i += 1) {
  const author = lines[i].match(authorLine);
  if (!author || !lines[i + 1]?.startsWith('Posted: ')) continue;
  const posted = lines[i + 1].match(postedLine);
  const body = [];
  let inSignature = false;
  i += 2;
  while (i < lines.length && lines[i] !== 'Back to top') {
    const signatureAt = lines[i].search(/\s+_{5,}/);
    if (signatureAt >= 0) {
      const beforeSignature = lines[i].slice(0, signatureAt).trimEnd();
      if (beforeSignature) body.push(beforeSignature);
      inSignature = true;
    } else if (!inSignature) {
      body.push(lines[i].trimEnd());
    }
    i += 1;
  }
  while (body.at(-1) === '') body.pop();
  posts.push({ author: author[1], role: author[2], posted: posted?.[1] ?? 'date unknown', body });
}

if (posts.length === 0) throw new Error(`no posts parsed from ${input}`);

const quote = (value) => JSON.stringify(value);
const frontmatter = [
  '---',
  `title: ${quote(record.title)}`,
  `description: ${quote(record.description)}`,
  'kind: "forum-thread"',
  `sortDate: ${record.date}`,
  `dateLabel: ${quote(record.dateLabel)}`,
  `publishedAt: ${record.date}`,
  'sourceSite: "dp-players.com"',
  `sourceUrl: "http://www.dp-players.com/forum/viewtopic.php?${id}"`,
  `captureUrl: "https://web.archive.org/web/20040803020622/http://www.dp-players.com/forum/viewtopic.php?${id}"`,
  'recoveredAt: 2026-08-14',
  'draft: false',
  '---',
  '',
];
const renderBodyLine = (line) => {
  if (line === '') return '';
  if (line.startsWith('Quote: ')) return `> ${line.slice(7)}`;
  return `${line}\\`;
};
const markdown = posts.flatMap((post) => [
  `### ${post.author} — ${post.posted}`,
  '',
  ...post.body.map(renderBodyLine),
  '',
  '---',
  '',
]);

writeFileSync(output, [...frontmatter, ...markdown].join('\n'));
console.log(`${id}: wrote ${posts.length} posts to ${output}`);
