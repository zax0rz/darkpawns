// driver.mjs runs the real browser client, web/public/mud-client.js, headless
// for the oracle harness. What a /play player sees is whatever that client
// writes to its xterm terminal after rendering the server's JSON messages, so
// the harness drives the client itself rather than a reimplementation of it.
//
//   node driver.mjs <mud-client.js path> <ws url>
//
// Each stdin line is typed into the client followed by Enter, as xterm's
// onData would deliver it; everything the client writes to the terminal goes
// to stdout, byte for byte (xterm's writeln appends \r\n).
import readline from 'node:readline';
import { pathToFileURL } from 'node:url';

const [clientPath, wsUrl] = process.argv.slice(2);
if (!clientPath || !wsUrl) {
  process.stderr.write('usage: node driver.mjs <mud-client.js> <ws url>\n');
  process.exit(2);
}

// The client reads location for its default URL and query parameters. On
// /play every sidebar element it asks for exists; the driver answers with an
// inert element that accepts any property, method call or child, so the
// client's DOM work (status bar, minimap, panels) runs as it does on the page
// and never throws. The terminal is the only output that matters here.
globalThis.location = { search: '', protocol: 'http:', host: '127.0.0.1' };
function inertElement() {
  const target = function () {};
  const handler = {
    get: (_obj, prop) => {
      if (prop === Symbol.toPrimitive) return () => '';
      if (prop === 'length') return 0;
      if (prop === Symbol.iterator) return function* () {};
      return new Proxy(function () {}, handler);
    },
    set: () => true,
    apply: () => new Proxy(function () {}, handler),
    construct: () => new Proxy(function () {}, handler),
  };
  return new Proxy(target, handler);
}
const doc = {
  querySelector: () => inertElement(),
  querySelectorAll: () => [],
  getElementById: () => inertElement(),
  createElement: () => inertElement(),
};
globalThis.document = doc;
// The minimap fetch has nowhere to go; the client already tolerates failure.
globalThis.fetch = async () => {
  throw new Error('no fetch in the harness driver');
};
console.warn = () => {};
console.debug = () => {};

let onData = null;
const terminal = {
  write: (data) => process.stdout.write(data),
  writeln: (data) => process.stdout.write(data + '\r\n'),
  onData: (callback) => {
    onData = callback;
  },
};

const { createMudClient } = await import(pathToFileURL(clientPath).href);
createMudClient({ terminal, doc, wsUrl });

const lines = readline.createInterface({ input: process.stdin });
lines.on('line', (line) => {
  if (onData) onData(line + '\r');
});
lines.on('close', () => process.exit(0));
