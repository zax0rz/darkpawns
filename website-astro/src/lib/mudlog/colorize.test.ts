// Tests for the log colorizer. One case per rule, each asserting the `color`
// level the C gates the rule on (screen.h: clr() is level >= gate).
//
//   node --experimental-strip-types --test src/lib/mudlog/colorize.test.ts

import { test } from 'node:test';
import assert from 'node:assert/strict';
import { resolve } from 'node:path';
import { Colorizer, type Run } from './colorize.ts';

// An empty world: these cases are all message shapes, none is a room listing.
const colorizer = new Colorizer({ mobLongs: new Set(), objLongs: new Set() }, resolve('..', 'lib/misc/messages'));

// Colour a block and return the runs of the line that contains `needle`.
function runsFor(block: string, needle: string): Run[] {
  const lines = colorizer.colorize(block);
  const line = lines.find((runs) => runs.map((r) => r.text).join('').includes(needle));
  assert.ok(line, `no line contains ${JSON.stringify(needle)}`);
  return line;
}

function coloured(runs: Run[]): Run[] {
  return runs.filter((r) => r.color);
}

test('your own gossip is the channel colour at C_CMP', () => {
  const runs = runsFor("You gossip, 'hello'\n", 'hello');
  assert.deepEqual(coloured(runs).map((r) => [r.color, r.level]), [['yellow', 3]]);
});

test('someone else\u2019s gossip is the channel colour at C_NRM', () => {
  const runs = runsFor("Aiko gossips, 'hello'\n", 'hello');
  assert.deepEqual(coloured(runs).map((r) => [r.color, r.level]), [['yellow', 2]]);
});

test('an auction is magenta, not yellow', () => {
  const runs = runsFor("You auction, 'wts sword'\n", 'sword');
  assert.deepEqual(coloured(runs).map((r) => [r.color, r.level]), [['magenta', 3]]);
});

test('a group tell you send is white at C_SPR', () => {
  const runs = runsFor("You tell the group, 'sleep'\n", 'sleep');
  assert.deepEqual(coloured(runs).map((r) => [r.color, r.level]), [['white', 1]]);
});

test('a group tell you hear is white at C_CMP', () => {
  const runs = runsFor("Xungon tells the group, 'sleep'\n", 'sleep');
  assert.deepEqual(coloured(runs).map((r) => [r.color, r.level]), [['white', 3]]);
});

test('a clan tell is cyan at C_CMP', () => {
  const runs = runsFor("Aiko tells your clan, 'on my way'\n", 'way');
  assert.deepEqual(coloured(runs).map((r) => [r.color, r.level]), [['cyan', 3]]);
});

test('a private tell you hear is red at C_NRM', () => {
  const runs = runsFor("Aiko tells you, 'psst'\n", 'psst');
  assert.deepEqual(coloured(runs).map((r) => [r.color, r.level]), [['red', 2]]);
});

test('a private tell you send is red at C_CMP', () => {
  const runs = runsFor("You tell Aiko, 'psst'\n", 'psst');
  assert.deepEqual(coloured(runs).map((r) => [r.color, r.level]), [['red', 3]]);
});

test('your weapon damage is yellow at C_CMP', () => {
  const runs = runsFor('You slash the troll hard.\n', 'slash');
  assert.deepEqual(coloured(runs).map((r) => [r.color, r.level]), [['yellow', 3]]);
});

test('weapon damage you take is red at C_CMP', () => {
  const runs = runsFor('The troll slashes you hard.\n', 'slashes');
  assert.deepEqual(coloured(runs).map((r) => [r.color, r.level]), [['red', 3]]);
});

test('a room title is cyan at C_NRM', () => {
  const runs = runsFor('A Narrow Path\nYou are on a path.\n[ Exits: north south ]\n', 'Narrow');
  assert.deepEqual(coloured(runs).map((r) => [r.color, r.level]), [['cyan', 2]]);
});

test('the exits line is cyan at C_NRM', () => {
  const runs = runsFor('A Narrow Path\nYou are on a path.\n[ Exits: north south ]\n', 'Exits');
  assert.deepEqual(coloured(runs).map((r) => [r.color, r.level]), [['cyan', 2]]);
});

test('prompt vitals are coloured by share at C_CMP', () => {
  const block = '100H 100M 100V > \nlook\n50H 80M 20V > \n';
  const runs = runsFor(block, '50H');
  const colors = coloured(runs).map((r) => [r.text, r.color, r.level]);
  assert.deepEqual(colors, [['50', 'yellow', 3], ['80', 'green', 3], ['20', 'red', 3]]);
});

test('score vitals colour the current green and the maximum red at C_CMP', () => {
  const runs = runsFor('Hit points: 100(200)  Mana points: 50(100)  Movement points: 10(80)\n', 'Hit points');
  const colors = coloured(runs).map((r) => [r.text, r.color, r.level]);
  assert.deepEqual(colors, [
    ['100', 'green', 3], ['200', 'red', 3],
    ['50', 'green', 3], ['100', 'red', 3],
    ['10', 'green', 3], ['80', 'red', 3],
  ]);
});

test('score experience is green at C_CMP', () => {
  const runs = runsFor('Experience:    950016 points\n', '950016');
  assert.deepEqual(coloured(runs).map((r) => [r.color, r.level]), [['green', 3]]);
});

test('score gold is green at C_CMP', () => {
  const runs = runsFor('Coins carried: 725 gold coins    Coins in bank: 0 gold coins\n', '725');
  assert.deepEqual(coloured(runs).map((r) => [r.color, r.level]), [['green', 3], ['green', 3]]);
});

test('experience still needed is green at C_CMP', () => {
  const runs = runsFor('You need 11409983 exp to reach your next level.\n', '11409983');
  assert.deepEqual(coloured(runs).map((r) => [r.color, r.level]), [['green', 3]]);
});

test('the score name is green at C_CMP', () => {
  const runs = runsFor('Jyan                           Age: 25 years\n', 'Jyan');
  assert.deepEqual(coloured(runs).map((r) => [r.color, r.level]), [['green', 3]]);
});

test('the spell list under score is cyan at C_NRM', () => {
  const block = 'Spells affecting you:\n     infravision              detect invisibility \n';
  const runs = runsFor(block, 'infravision');
  const colors = coloured(runs);
  assert.ok(colors.length >= 1);
  assert.deepEqual([...new Set(colors.map((r) => `${r.color}:${r.level}`))], ['cyan:2']);
});

test('a plain say is left uncoloured (do_say passes no colour to act)', () => {
  const runs = runsFor("Aiko says, 'hello'\n", 'hello');
  assert.equal(coloured(runs).length, 0);
});
