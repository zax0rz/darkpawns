import { readFileSync, writeFileSync } from 'node:fs';
import { resolve } from 'node:path';

const [inputArg, outputArg] = process.argv.slice(2);
if (!inputArg || !outputArg) {
  throw new Error('usage: node scripts/import-history.mjs <timeline.md> <output.md>');
}

const lines = readFileSync(resolve(inputArg), 'utf8').split(/\r?\n/);
const firstRow = lines.findIndex((line) => line.startsWith('| September 1994 |'));
if (firstRow < 0) throw new Error('timeline start not found');

const rows = [];
let current;
const flush = () => {
  if (!current) return;
  while (current.body.at(-1) === '') current.body.pop();
  rows.push(current);
};

for (const rawLine of lines.slice(firstRow)) {
  if (rawLine === '---' || rawLine.startsWith('[<- back]')) break;
  if (rawLine === '| --- | --- |') continue;
  const start = rawLine.match(/^\| ([^|]+) \|\s?(.*)$/);
  if (start) {
    flush();
    current = { date: start[1].trim(), body: [start[2].replace(/ \|$/, '').trimEnd()] };
    continue;
  }
  if (current) current.body.push(rawLine.replace(/ \|$/, '').trimEnd());
}
flush();

const frontmatter = `---
title: "A Brief History of Dark Pawns"
sortDate: 2004-07-20
dateLabel: "Recorded through 2001; archived in 2004"
draft: false
description: "Frontline's record of Dark Pawns from its 1994 founding through the loss and recovery of the game in 2001."
kind: "history"
sourceSite: "dp-players.com"
sourceUrl: "http://www.dp-players.com/go.php?dp=history.mud"
captureUrl: "https://web.archive.org/web/20040720230810/http://www.dp-players.com/go.php?dp=history.mud"
recoveredAt: 2026-08-14
---

**Recorded by: Frontline**

This is the unabridged history as best anyone can remember. There is likely some stuff missing. If you know what it is, send us a mudmail and we'll get it corrected. This is an updated version of the history of DP originally written by Orodreth circa 1999.
`;
const sections = rows.map((row) => `## ${row.date}\n\n${row.body.join('\n').trim()}\n`).join('\n');

writeFileSync(resolve(outputArg), `${frontmatter}\n${sections}`);
console.log(`wrote ${rows.length} timeline entries to ${resolve(outputArg)}`);
