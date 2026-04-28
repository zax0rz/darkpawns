(function () {
  'use strict';

  const params = new URLSearchParams(location.search);
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
  const wsUrl = params.get('host') || `${proto}//${location.host}/ws`;

  const term = new Terminal({
    cursorBlink: true,
    fontSize: 15,
    fontFamily: '"IM Fell English", "Courier New", monospace',
    theme: {
      background: '#0a0908',
      foreground: '#c8b896',
      cursor: '#8b0000',
      selectionBackground: '#3a2a1a',
    },
  });
  const fitAddon = new FitAddon.FitAddon();
  term.loadAddon(fitAddon);
  term.open(document.getElementById('terminal'));
  fitAddon.fit();

  window.addEventListener('resize', () => fitAddon.fit());

  const statusEl = document.querySelector('.conn-status');
  const reconnectBtn = document.getElementById('reconnect-btn');
  let inputBuffer = '';
  let ws;

  function setStatus(state) {
    statusEl.className = 'conn-status ' + state;
    const label = state === 'connected' ? 'Connected' : 'Disconnected';
    statusEl.querySelector('span').textContent = label;
    reconnectBtn.classList.toggle('visible', state === 'disconnected');
  }

  function connect() {
    setStatus('disconnected');
    term.writeln('\x1b[2mConnecting to ' + wsUrl + '...\x1b[0m');
    try {
      ws = new WebSocket(wsUrl);
    } catch (e) {
      term.writeln('\x1b[31mConnection failed: ' + e.message + '\x1b[0m');
      return;
    }

    ws.onopen = function () {
      setStatus('connected');
      term.writeln('\x1b[32mConnected.\x1b[0m');
      term.writeln('Enter your character name:');
    };

    ws.onmessage = function (evt) {
      let text;
      try {
        const msg = JSON.parse(evt.data);
        text = msg.text || evt.data;
      } catch {
        text = evt.data;
      }
      term.writeln(text);
    };

    ws.onclose = function () {
      setStatus('disconnected');
      term.writeln('\x1b[31m--- Connection lost ---\x1b[0m');
    };

    ws.onerror = function () {
      term.writeln('\x1b[31mConnection error.\x1b[0m');
    };
  }

  let loggedIn = false;

  term.onData(function (data) {
    if (!ws || ws.readyState !== WebSocket.OPEN) return;

    if (data === '\r' || data === '\n') {
      term.writeln('');
      if (!loggedIn) {
        const name = inputBuffer.trim();
        if (name) {
          ws.send(JSON.stringify({ type: 'login', data: { player_name: name } }));
          loggedIn = true;
        }
        inputBuffer = '';
        return;
      }
      if (inputBuffer.trim()) {
        ws.send(JSON.stringify({ type: 'command', data: { command: inputBuffer } }));
      }
      inputBuffer = '';
    } else if (data === '\x7f' || data === '\b') {
      if (inputBuffer.length > 0) {
        inputBuffer = inputBuffer.slice(0, -1);
        term.write('\b \b');
      }
    } else if (data >= ' ') {
      inputBuffer += data;
      term.write(data);
    }
  });

  reconnectBtn.addEventListener('click', function () {
    loggedIn = false;
    inputBuffer = '';
    connect();
  });

  connect();
})();
