import assert from 'node:assert/strict';
import { test } from 'node:test';
import { markdownToHtml } from 'satteri';
import { noLiteralAutolinks } from './no-literal-autolinks.mjs';

const render = (source) => markdownToHtml(source, { mdastPlugins: [noLiteralAutolinks] }).html;

test('bare hostnames, URLs and emails stay plain text', () => {
  const html = render('Host www.augusta.net 4000, see http://pawns.guru.org/pawns or mail me@example.com.');
  assert.doesNotMatch(html, /<a /);
  assert.match(html, /www\.augusta\.net 4000/);
  assert.match(html, /http:\/\/pawns\.guru\.org\/pawns/);
});

test('links an author wrote keep working', () => {
  const html = render('Read [the archive](https://darkpawns.org/archive/) or <https://darkpawns.org/>.');
  assert.match(html, /<a href="https:\/\/darkpawns\.org\/archive\/">the archive<\/a>/);
  assert.match(html, /<a href="https:\/\/darkpawns\.org\/">/);
});

test('byte offsets hold after non-ASCII text', () => {
  // Sätteri reports offsets in UTF-8 bytes; a string index check misses this.
  const html = render('Fäncy “quotes” first.\n\nThen www.topmudsites.com/vote and [kept](https://darkpawns.org/).');
  assert.doesNotMatch(html, /topmudsites\.com\/vote<\/a>/);
  assert.match(html, /<a href="https:\/\/darkpawns\.org\/">kept<\/a>/);
});
