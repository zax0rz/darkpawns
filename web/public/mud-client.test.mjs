import assert from 'node:assert/strict';
import test from 'node:test';

import { createMudClient } from './mud-client.js';

test('a greeting cannot release a pasted password before the secret prompt', async () => {
  const previous = {
    WebSocket: globalThis.WebSocket,
    fetch: globalThis.fetch,
    location: globalThis.location,
  };
  class FakeWebSocket {
    static OPEN = 1;

    constructor() {
      this.readyState = FakeWebSocket.OPEN;
      this.sent = [];
      FakeWebSocket.latest = this;
    }

    send(message) { this.sent.push(JSON.parse(message)); }
    close() {}
  }
  globalThis.WebSocket = FakeWebSocket;
  globalThis.fetch = async () => ({ json: async () => ({}) });
  globalThis.location = { search: '', protocol: 'https:', host: 'darkpawns.org' };

  const writes = [];
  let input;
  const terminal = {
    write: text => writes.push(text),
    writeln: text => writes.push(text + '\n'),
    onData: callback => { input = callback; },
    cols: 80,
  };
  const doc = {
    querySelector: () => null,
    querySelectorAll: () => [],
    getElementById: () => null,
  };

  try {
    const client = createMudClient({ terminal, doc });
    const socket = FakeWebSocket.latest;
    socket.onopen();
    input('Aiko\rhunter2\r');
    assert.deepEqual(socket.sent.map(message => message.type), ['terminal', 'line']);

    // The name line can reach the server before its greeting reaches us.
    socket.onmessage({ data: JSON.stringify({ type: 'out', data: { text: 'By what name?' } }) });
    assert.equal(socket.sent.length, 2);
    assert.equal(writes.join('').includes('hunter2'), false);

    socket.onmessage({ data: JSON.stringify({ type: 'out', data: {
      text: 'Password: ', entry: true, secret: true,
    } }) });
    assert.equal(socket.sent.length, 3);
    assert.deepEqual(socket.sent[2], { type: 'line', data: { line: 'hunter2' } });
    assert.equal(writes.join('').includes('hunter2'), false);
    client.disconnect();
  } finally {
    globalThis.WebSocket = previous.WebSocket;
    globalThis.fetch = previous.fetch;
    globalThis.location = previous.location;
  }
});
