// Audits how the colorizer classifies room contents: objects are green, people
// yellow (act.informative.c:824-835). The 2000-era world differs from
// lib/world, so some lines are classified from the fallback patterns rather
// than from the world's own long descriptions. This prints those cases.
//
//   node --experimental-strip-types scripts/colorize-room-audit.ts

import { readdirSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { loadWorldText } from '../src/lib/mudlog/colorize.ts';

const world = loadWorldText();
const dir = resolve(process.cwd(), 'src/content/logs');

const PROMPT = /^(?:i\d+ )?(?:(?:\d+)H )?(?:(?:\d+)M )?(?:(?:\d+)V )?(?:[^>]*?:\([^)]*\) )*(?:(?:AFK|INACTIVE) )?>/;
const CHAR_IN_ROOM = /\b(?:is|are) (?:standing|sitting|resting|sleeping|lying|floating|flying|here|meditating|fighting)\b|^\.\.\.(?:he|she|it) /;
const OBJ_IN_ROOM = /\bhere\.\s*$|\bground\.\s*$|\bfloor\.\s*$/;

let content = 0;
let matchedMob = 0;
let matchedObj = 0;
let fallback = 0;
let personCount = 0;
let objectCount = 0;
const ambiguousHere: string[] = [];

for (const name of readdirSync(dir).filter((f) => f.endsWith('.log')).sort()) {
  const lines = readFileSync(resolve(dir, name), 'utf8').split('\n');
  for (let e = 0; e < lines.length; e++) {
    if (!/^\s*\[ Exits: .*\]\s*$/.test(lines[e])) continue;
    // Title: first non-blank line after the prompt or blank above the block.
    let s = e - 1;
    while (s >= 0 && lines[s].trim() !== '' && !PROMPT.test(lines[s])) s--;
    let title = -1;
    for (let j = s + 1; j < e; j++) if (lines[j].trim()) { title = j; break; }
    if (title < 0) continue;
    // do_look prints the contents AFTER the exits line (act.informative.c:818-835).
    let seenChar = false;
    for (let i = e + 1; i < lines.length; i++) {
      const bare = lines[i].trim();
      if (!bare) break;
      if (PROMPT.test(lines[i])) break;
      const isMob = world.mobLongs.has(bare);
      const isObj = world.objLongs.has(bare);
      const person = isMob || CHAR_IN_ROOM.test(bare);
      const object = !seenChar && (isObj || OBJ_IN_ROOM.test(bare));
      if (!person && !object) break;
      content += 1;
      if (isMob) matchedMob += 1;
      else if (isObj) matchedObj += 1;
      else fallback += 1;
      if (person) {
        personCount += 1;
        seenChar = true;
        // A bare "<thing> is here." line: an object or a mob, and the world
        // data does not settle which. Every one of these is coloured yellow.
        if (/\b(?:is|are) here\.\s*$/.test(bare)) ambiguousHere.push(`${name}: ${bare}`);
      } else {
        objectCount += 1;
      }
    }
  }
}

console.log(`room content lines: ${content}`);
console.log(`  matched a mob long description: ${matchedMob}`);
console.log(`  matched an object long description: ${matchedObj}`);
console.log(`  classified by fallback pattern only: ${fallback}`);
console.log(`  coloured yellow (person): ${personCount}`);
console.log(`  coloured green (object): ${objectCount}`);
console.log(`\n"<thing> is here." lines coloured yellow (${ambiguousHere.length}); the world`);
console.log(`data does not say whether each is a mob or an object:`);
for (const line of ambiguousHere.slice(0, 15)) console.log('  ' + line);

