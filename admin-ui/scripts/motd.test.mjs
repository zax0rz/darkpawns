import { test } from 'node:test';
import assert from 'node:assert/strict';
import { motdTerminalText, motdBytes, discordSnippet, gossipNotice } from '../src/lib/motd.ts';

test('game colours use the C mappings, not CSS word wrapping', () => {
  assert.equal(motdTerminalText('&rred&n && &x unknown &'), '\x1b[0;31mred\x1b[0m & &x unknown &');
  assert.equal(motdTerminalText('&d&b&g&c&r&m&y&w&D&B&G&C&R&M&Y&W'),
    '\x1b[0;30m\x1b[0;34m\x1b[0;32m\x1b[0;36m\x1b[0;31m\x1b[0;35m\x1b[0;33m\x1b[0;37m\x1b[1;30m\x1b[1;34m\x1b[1;32m\x1b[1;36m\x1b[1;31m\x1b[1;35m\x1b[1;33m\x1b[1;37m');
  assert.equal(motdTerminalText('&cRaw&n', false), '&cRaw&n');
  assert.equal(motdTerminalText('a\nb\r\nc\rd\n\r'), 'a\r\nb\r\nc\r\nd\n\r');
});
test('byte count includes CRLF and Unicode, as the save API does', () => {
  assert.equal(motdBytes('é\n雪\r\n'), 9);
  assert.equal(motdBytes('x'.repeat(2398) + '\n'), 2400);
  assert.equal(motdBytes('x'.repeat(2399) + '\n'), 2401);
});
test('Discord insertion needs an operator-supplied HTTPS invite', () => {
  assert.equal(discordSnippet(' https://discord.gg/test-world '), '&CJoin us on Discord&n\nhttps://discord.gg/test-world\n');
  assert.ok(discordSnippet('https://discord.com/invite/test-world'));
  for (const url of ['', 'http://discord.gg/world', 'https://example.com/world', 'https://discord.gg.evil.test/world', 'https://discord.gg/world?x=1', 'https://name@discord.gg/world', 'https://discord.com/channels/1/2']) assert.equal(discordSnippet(url), null, url);
  assert.equal(gossipNotice, "What's said on gossip echoes to our Discord.");
});
