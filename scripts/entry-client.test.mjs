import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import vm from 'node:vm';

// Run the actual shipped client event handler with transport/terminal doubles.
// This covers browser-owned dialogue that the telnet oracle cannot observe.
function client() {
  let input;
  let socket;
  let output = '';
  const sent = [];
  const element = {
    classList: { add() {}, remove() {}, toggle() {} },
    addEventListener() {},
    querySelector() { return { textContent: '' }; },
  };
  class Terminal {
    loadAddon() {}
    open() {}
    onData(handler) { input = handler; }
    write(text) { output += text; }
    writeln(text) { output += `${text ?? ''}\r\n`; }
  }
  class WebSocket {
    static OPEN = 1;
    readyState = 1;
    constructor() { socket = this; }
    send(message) { sent.push(JSON.parse(message)); }
  }
  const source = readFileSync(new URL('../website-astro/src/scripts/client.js', import.meta.url), 'utf8')
    .replace(/^import .*;$/gm, ''); // Dependencies are replaced by terminal/transport doubles.
  vm.runInNewContext(source, {
    Terminal, WebSocket, URLSearchParams, console,
    FitAddon: class { fit() {} },
    location: { search: '', protocol: 'http:', host: 'localhost' },
    window: { addEventListener() {} },
    document: { readyState: 'complete', getElementById() { return element; }, querySelector() { return element; }, querySelectorAll() { return []; } },
    fetch: async () => ({ json: async () => ({}) }),
  }, { filename: 'client.js' });
  socket.onopen();
  output = '';
  return {
    input, sent,
    output: () => output,
    clearOutput: () => { output = ''; },
    receive: (type, data) => socket.onmessage({ data: JSON.stringify({ type, data }) }),
  };
}

test('entering a name contacts the server before asking for a password', () => {
  const c = client();
  c.input('aiko\r');
  assert.equal(c.sent.length, 1, `name was not sent to server; browser printed ${JSON.stringify(c.output())}`);
  assert.equal(c.sent[0].data.player_name, 'aiko');
  assert.ok(!c.output().includes('Password:'), 'browser must wait for the server name lookup');
});

test('pasted password waits for the server prompt and stays secret', () => {
  const c = client();
  c.input('aiko\roraclepass\r');
  assert.equal(c.sent.length, 1);
  assert.ok(!c.output().includes('oraclepass'));
  c.clearOutput();
  c.receive('char_create', { stage: 'login_password', prompt: 'Password: ', secret: true });
  assert.equal(c.output(), 'Password: ');
  assert.equal(c.sent.length, 2);
  assert.equal(c.sent[1].type, 'char_input');
  assert.equal(c.sent[1].data.choice, 'oraclepass');
});

test('creation prompts render exact server text without added menus or chevrons', () => {
  const c = client();
  const prompt = "Press 'Y' to keep these stats, and 'N' to reroll:";
  c.receive('char_create', { stage: 'stats_roll', prompt, options: [{ key: 'Y', label: 'Keep' }] });
  assert.equal(c.output(), prompt);
});

test('secret backspace changes submitted password without terminal echo', () => {
  const c = client();
  c.receive('char_create', { stage: 'create_password', prompt: 'Password: ', secret: true });
  c.clearOutput();
  c.input(' secretX\x7f \r');
  assert.equal(c.output(), '');
  assert.equal(c.sent[0].data.choice, ' secret ');
});
