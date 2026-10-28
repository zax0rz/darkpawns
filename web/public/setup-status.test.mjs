import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { showSetupStatus } from './setup-status.js';

const document = () => {
  const nodes = Object.fromEntries(['setup-notice', 'setup-heading', 'setup-message', 'setup-next'].map(id => [id, { hidden: false, textContent: '' }]));
  return { nodes, getElementById: id => nodes[id] };
};
test('fresh worlds warn about level40 ownership, then show restart, then running state', () => {
  const doc = document();
  showSetupStatus({ first_character_pending: true, started_empty: true }, doc);
  assert.match(doc.nodes['setup-message'].textContent, /first character.*level 40/);
  assert.match(doc.nodes['setup-message'].textContent, /before you let anyone else in/);
  assert.equal(doc.nodes['setup-notice'].hidden, false);
  showSetupStatus({ first_character_pending: false, started_empty: true }, doc);
  assert.match(doc.nodes['setup-next'].textContent, /Restart/);
  assert.equal(doc.nodes['setup-notice'].hidden, false);
  showSetupStatus({ first_character_pending: false, started_empty: false }, doc);
  assert.equal(doc.nodes['setup-notice'].hidden, true);
});
test('unavailable or malformed status cannot hide the initial conditional warning', async () => {
  const doc = document();
  for (const data of [null, {}, { first_character_pending: false }, { first_character_pending: 'false', started_empty: false }]) showSetupStatus(data, doc);
  assert.equal(doc.nodes['setup-notice'].hidden, false);
  const html = await readFile(new URL('./index.html', import.meta.url), 'utf8');
  assert.match(html, /On a fresh world, the first character/);
  assert.match(html, /href="\/admin\/"/);
  assert.ok(html.indexOf('Play in this browser') < html.indexOf('setup-notice'));
});
