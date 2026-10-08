import assert from 'node:assert/strict';
import test from 'node:test';
import { reflowWorldText } from './world-text.mjs';

test('joins hard-wrapped lines into one paragraph', () => {
  const room8008 = "The city's citizens file in to this large open pavilion once a week for\nmandatory worship, but now it is quiet except for some battered adventurers\nand their healers.  The main temple and the altar are to your north.";
  assert.deepEqual(reflowWorldText(room8008), {
    drawing: false,
    paragraphs: ["The city's citizens file in to this large open pavilion once a week for mandatory worship, but now it is quiet except for some battered adventurers and their healers. The main temple and the altar are to your north."],
  });
});

test('an indented line starts a new paragraph', () => {
  const room1 = 'You are floating in a formless void, detached from all sensation of physical\nmatter.\n   There is a "No Tipping" notice pinned to the darkness.';
  assert.deepEqual(reflowWorldText(room1).paragraphs, [
    'You are floating in a formless void, detached from all sensation of physical matter.',
    'There is a "No Tipping" notice pinned to the darkness.',
  ]);
});

test('a blank line also starts a new paragraph', () => {
  assert.deepEqual(reflowWorldText('one\ntwo\n\nthree').paragraphs, ['one two', 'three']);
});

test('box drawings keep their line breaks', () => {
  const maze = '*--*--*--*--*\n   |  |  /  |  |\n   *--*-\\*/-*--*';
  assert.deepEqual(reflowWorldText(maze), { drawing: true, paragraphs: [maze] });
});

test('empty text has no paragraphs', () => {
  assert.deepEqual(reflowWorldText(undefined), { drawing: false, paragraphs: [] });
});
