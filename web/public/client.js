// A self-hoster is not on darkpawns.org, so the landing page names the host
// they actually reached. Inline script is blocked by the server's CSP, so this
// lives here, where script-src 'self' allows it.
(function nameThisHost() {
  var el = document.getElementById('telnet-host');
  if (el) el.textContent = window.location.hostname + ' port 7777';
})();

// Liveness for the page the setup wizard sends people to on success
// (docs/specs/tui-setup-wizard.md). /health is unauthenticated on this server
// and answers the one question a self-hoster has just opened the page to ask.
//
// Player and world counts are deliberately absent. /metrics exposes them, but
// every gauge reads zero: the collectors are registered and cmd/server wires
// only the handler, never a writer, so the numbers would be honest-looking
// zeros rather than readings. Nothing here is invented.
(function serverStatus() {
  var section = document.querySelector('.status');
  var text = document.getElementById('status-text');
  if (!section || !text) return;

  function setState(state, message) {
    section.classList.remove('is-up', 'is-down');
    if (state) section.classList.add(state);
    text.textContent = message;
  }

  function refresh() {
    fetch('/health', { cache: 'no-store' })
      .then(function (res) {
        if (!res.ok) throw new Error('health ' + res.status);
        setState('is-up', 'This server is running');
      })
      .catch(function () {
        // A failed request proves the page could not reach the server, not
        // that the server is down; say the smaller thing.
        setState('is-down', 'Cannot reach this server');
      });
  }

  refresh();
  setInterval(refresh, 15000);
})();


// The terminal. Until 2026-09-17 this file carried its own copy of the client,
// which had drifted from the one on /play and rendered `state` messages into
// the terminal as invented prose — so a self-hoster saw every room twice, once
// as the game wrote it and once paraphrased. The protocol now lives in
// mud-client.js, shared verbatim with /play and the admin console.
//
// xterm arrives as a UMD global from the two <script> tags above rather than as
// a bare import, because this page is served straight off disk with no build
// step. The shared module takes the constructed terminal, so that difference
// stops at this file.
import { createMudClient } from './mud-client.js';

(function bootTerminal() {
  'use strict';

  const mount = document.getElementById('terminal');
  if (!mount || typeof Terminal === 'undefined') return;

  const term = new Terminal({
    cursorBlink: true,
    // Game text (MOTD, room/help files) uses bare "\n" line endings. Without
    // convertEol, xterm treats a lone LF as line-feed-only — the cursor drops a
    // row but keeps its column — producing runaway "staircase" indentation.
    // convertEol makes every "\n" behave as "\r\n" so lines return to column 0.
    convertEol: true,
    fontSize: 15,
    fontFamily: '"IM Fell English", "Courier New", monospace',
    theme: {
      background: '#0a0908',
      foreground: '#c8b896',
      cursor: '#8b0000',
      selectionBackground: '#3a2a1a',
    },
  });

  const fit = new FitAddon.FitAddon();
  term.loadAddon(fit);
  term.open(mount);
  fit.fit();
  window.addEventListener('resize', () => fit.fit());

  // No sidebar panels on this page, so the shared client renders the terminal
  // and the status bar and skips the rest. Nothing to configure for that.
  createMudClient({ terminal: term });
})();
