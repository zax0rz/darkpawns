// Measures how much of the published logs the colorizer actually colours.
//
// It runs the real Colorizer over every scrubbed excerpt and tallies the lines
// that come back plain, collapsed to a shape (names and numbers folded away)
// so the count is of message kinds, not of one busy line. The top shapes are
// the candidates for a new rule; a shape only earns a rule if the C source
// colours it, which is why each one is checked by hand before it is added.
//
//   node --experimental-strip-types scripts/colorize-coverage.ts [--top 50]

import { readdirSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { Colorizer, loadWorldText } from '../src/lib/mudlog/colorize.ts';

const args = process.argv.slice(2);
const topIndex = args.indexOf('--top');
const TOP = topIndex >= 0 ? Number(args[topIndex + 1]) : 50;

// Fold a line to its shape: drop leading prompt/stat noise, replace runs of
// digits and capitalised words so "Aiko hits the troll" and "Lucy hits the
// goblin" count as one kind.
function shape(line: string): string {
  let s = line.trim();
  s = s.replace(/\b\d+\b/g, 'N');
  s = s.replace(/\b[A-Z][a-z]+\b/g, 'X');
  s = s.replace(/\b[A-Z]{2,}\b/g, 'X');
  s = s.replace(/'[^']*'/g, "'…'");
  return s.slice(0, 100);
}

const world = loadWorldText();
const colorizer = new Colorizer(world);
const dir = resolve(process.cwd(), 'src/content/logs');

let total = 0;
let plain = 0;
const shapes = new Map<string, { count: number; sample: string; file: string }>();

for (const name of readdirSync(dir).filter((f) => f.endsWith('.log')).sort()) {
  const lines = colorizer.colorize(readFileSync(resolve(dir, name), 'utf8'));
  for (const runs of lines) {
    const text = runs.map((r) => r.text).join('');
    if (!text.trim()) continue;
    total += 1;
    if (runs.some((r) => r.color)) continue;
    plain += 1;
    const key = shape(text);
    const row = shapes.get(key) ?? { count: 0, sample: text.trim(), file: name };
    row.count += 1;
    shapes.set(key, row);
  }
}

const ranked = [...shapes.entries()].sort((a, b) => b[1].count - a[1].count).slice(0, TOP);
console.log(`lines ${total}, plain ${plain} (${((1 - plain / total) * 100).toFixed(1)}% coloured)`);
console.log(`distinct plain shapes ${shapes.size}\n`);
for (const [key, row] of ranked) {
  console.log(`${String(row.count).padStart(7)}  ${key}`);
}
