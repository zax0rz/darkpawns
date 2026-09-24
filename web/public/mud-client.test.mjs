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

test('the dock reveals supplied data and clears stale panels on disconnect', async () => {
  const previous = {
    WebSocket: globalThis.WebSocket,
    fetch: globalThis.fetch,
    location: globalThis.location,
  };
  class FakeWebSocket {
    static OPEN = 1;
    constructor() {
      this.readyState = FakeWebSocket.OPEN;
      FakeWebSocket.latest = this;
    }
    send() {}
    close() {}
  }
  const element = () => {
    const classes = new Set(['hidden']);
    return {
      classList: {
        add: name => classes.add(name),
        remove: name => classes.delete(name),
        contains: name => classes.has(name),
      },
      contains: () => false,
      style: {},
      innerHTML: '',
      textContent: '',
    };
  };
  const ids = Object.fromEntries([
    'status-bar', 'character-identity', 'hp-bar', 'hp-text', 'mana-bar', 'mana-text', 'move-bar',
    'move-text', 'level-info', 'gold-info', 'sidebar-connect-panel',
    'minimap-container', 'target-container', 'room-contents-container',
    'inventory-equipment-container',
  ].map(id => [id, element()]));
  const panels = ['minimap-container', 'target-container', 'room-contents-container', 'inventory-equipment-container'].map(id => ids[id]);
  globalThis.WebSocket = FakeWebSocket;
  globalThis.fetch = async () => ({
    ok: true,
    json: async () => ({ rooms: [{ id: 8004, name: 'At the Temple Altar', zone_id: 1, x: 0, y: 0, sector: 0 }], links: [] }),
  });
  globalThis.location = { search: '', protocol: 'https:', host: 'darkpawns.org' };
  const doc = {
    activeElement: null,
    querySelector: () => null,
    querySelectorAll: () => panels,
    getElementById: id => ids[id] || null,
  };
  const terminal = { write() {}, writeln() {}, onData() {}, cols: 80 };

  try {
    createMudClient({ terminal, doc });
    const socket = FakeWebSocket.latest;
    await new Promise(resolve => setImmediate(resolve));
    socket.onmessage({ data: JSON.stringify({ type: 'state', data: {
      player: { name: 'Tester', race: 'Kenderkin', class: 'Thief', health: 23, max_health: 23 },
      room: { vnum: 8004 },
    } }) });
    socket.onmessage({ data: JSON.stringify({ type: 'vars', data: {
      HEALTH: 23, MAX_HEALTH: 23, ROOM_VNUM: 8004,
    } }) });
    socket.onmessage({ data: JSON.stringify({ type: 'gmcp', data: {
      package: 'Room.Info', json: JSON.stringify({ num: 8004, name: 'At the Temple Altar' }),
    } }) });
    assert.equal(ids['character-identity'].textContent, 'Tester · Kenderkin · Thief');
    assert.equal(ids['hp-text'].textContent, '23/23');
    assert.match(ids['minimap-container'].innerHTML, /Current room: At the Temple Altar/);
    assert.match(ids['minimap-container'].innerHTML, /aria-hidden="true"/);
    assert.equal(ids['minimap-container'].classList.contains('hidden'), false);
    assert.equal(ids['inventory-equipment-container'].classList.contains('hidden'), true);
    socket.onclose();
    assert.equal(ids['status-bar'].classList.contains('hidden'), true);
    assert.equal(ids['sidebar-connect-panel'].classList.contains('hidden'), false);
    assert.ok(panels.every(panel => panel.classList.contains('hidden')));
  } finally {
    globalThis.WebSocket = previous.WebSocket;
    globalThis.fetch = previous.fetch;
    globalThis.location = previous.location;
  }
});

test('structured channel messages fill chat without changing terminal output', () => {
  const previous = {
    WebSocket: globalThis.WebSocket,
    fetch: globalThis.fetch,
    location: globalThis.location,
  };
  class FakeWebSocket {
    static OPEN = 1;
    constructor() { this.readyState = FakeWebSocket.OPEN; FakeWebSocket.latest = this; }
    send() {}
    close() {}
  }
  const children = [];
  const chatMessages = {
    append: node => children.push(node),
    replaceChildren: () => { children.length = 0; },
    get childElementCount() { return children.length; },
    get firstElementChild() { return { remove: () => children.shift() }; },
  };
  const classes = new Set();
  const chatEmpty = { classList: {
    add: name => classes.add(name),
    remove: name => classes.delete(name),
  } };
  const nodes = { 'chat-messages': chatMessages, 'chat-empty': chatEmpty };
  const doc = {
    getElementById: id => nodes[id] || null,
    querySelector: () => null,
    querySelectorAll: () => [],
    createElement: tag => ({ tag, append(...parts) { this.parts = parts; } }),
    createTextNode: text => ({ text }),
  };
  const writes = [];
  const terminal = { write: text => writes.push(text), writeln: text => writes.push(text), onData() {}, cols: 80 };
  globalThis.WebSocket = FakeWebSocket;
  globalThis.fetch = async () => ({ ok: true, json: async () => ({}) });
  globalThis.location = { search: '', protocol: 'https:', host: 'darkpawns.org' };
  try {
    const client = createMudClient({ terminal, doc });
    const socket = FakeWebSocket.latest;
    const before = writes.join('');
    socket.onmessage({ data: JSON.stringify({ type: 'gmcp', data: {
      package: 'Comm.Channel.Text',
      json: JSON.stringify({ channel: 'say', talker: 'Walker', text: 'Walker says hi' }),
    } }) });
    assert.equal(children.length, 1);
    assert.equal(children[0].parts[0].textContent, 'say');
    assert.equal(children[0].parts[1].text, ' Walker says hi');
    assert.equal(writes.join(''), before);
    assert.equal(classes.has('hidden'), true);
    socket.onclose();
    assert.equal(children.length, 0);
    assert.equal(classes.has('hidden'), false);
    client.disconnect();
  } finally {
    globalThis.WebSocket = previous.WebSocket;
    globalThis.fetch = previous.fetch;
    globalThis.location = previous.location;
  }
});

test('a failing dock handler never writes the protocol envelope to the terminal', async () => {
  const previous = {
    WebSocket: globalThis.WebSocket,
    fetch: globalThis.fetch,
    location: globalThis.location,
    consoleError: console.error,
  };
  class FakeWebSocket {
    static OPEN = 1;
    constructor() { this.readyState = FakeWebSocket.OPEN; FakeWebSocket.latest = this; }
    send() {}
    close() {}
  }
  globalThis.WebSocket = FakeWebSocket;
  globalThis.fetch = async () => ({ json: async () => ({}) });
  globalThis.location = { search: '', protocol: 'https:', host: 'darkpawns.org' };
  console.error = () => {};

  const writes = [];
  const terminal = {
    write: text => writes.push(text),
    writeln: text => writes.push(text + '\n'),
    onData: () => {},
    cols: 80,
  };
  // A chat panel whose DOM throws, standing in for any handler bug.
  const broken = { replaceChildren() {}, append() { throw new Error('boom'); } };
  const doc = {
    querySelector: () => null,
    querySelectorAll: () => [],
    getElementById: id => (id === 'chat-messages' ? broken : null),
    createElement: () => ({ append() {} }),
    createTextNode: text => text,
  };

  try {
    const client = createMudClient({ terminal, doc });
    const socket = FakeWebSocket.latest;
    socket.onopen();
    const frame = JSON.stringify({ type: 'gmcp', data: {
      package: 'Comm.Channel.Text',
      json: JSON.stringify({ channel: 'say', talker: 'Aiko', text: "Aiko says, 'hi'" }),
    } });
    socket.onmessage({ data: frame });
    assert.equal(writes.join('').includes('"type"'), false);
    assert.equal(writes.join('').includes('Comm.Channel.Text'), false);
    client.disconnect();
  } finally {
    globalThis.WebSocket = previous.WebSocket;
    globalThis.fetch = previous.fetch;
    globalThis.location = previous.location;
    console.error = previous.consoleError;
  }
});
